// Command pool-lanes records how booked-up lane swimming is at London Aquatics
// Centre, from Everyone Active's OpenActive feed.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // For Europe/London where the system has no zoneinfo.
)

func main() {
	dbPath := flag.String("db", "pool-lanes.db", "SQLite database path")
	interval := flag.Duration("interval", 5*time.Minute, "how often to check the feed for changes")
	once := flag.Bool("once", false, "catch up with the feed once, then exit")
	addr := flag.String("addr", "", "serve the web UI on this address, e.g. :8080")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := openStore(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer st.db.Close()
	client := &http.Client{Timeout: time.Minute}

	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		log.Fatal(err)
	}
	srv := &server{st: st, loc: loc, now: time.Now, started: time.Now()}
	if *addr != "" {
		hs := &http.Server{Addr: *addr, Handler: srv.handler()}
		go func() {
			if err := hs.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}()
		defer hs.Shutdown(context.Background())
		log.Printf("serving on %s", *addr)
	}

	for {
		if moved, err := catchUp(ctx, client, st); err != nil {
			if ctx.Err() == nil {
				log.Print(err)
			}
		} else {
			srv.checked(time.Now(), moved)
		}
		if *once {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(*interval):
		}
	}
}

// catchUp reads every page after the stored cursor, and says whether there
// were any. The feed's last page is empty and links to itself, which is where
// the next catch-up starts.
func catchUp(ctx context.Context, client *http.Client, st *store) (moved bool, err error) {
	pages, changed := 0, 0
	defer func() { log.Printf("read %d pages, %d sessions changed", pages, changed) }()
	for {
		url, err := st.cursor(ctx)
		if err != nil {
			return moved, err
		}
		p, err := fetchPage(ctx, client, url)
		if errors.Is(err, errPageGone) && url != feedURL {
			// Reading it all again only stores what changed.
			log.Printf("%v; starting over from the feed's first page", err)
			if err := st.resetCursor(ctx); err != nil {
				return moved, err
			}
			continue
		}
		if err != nil {
			return moved, err
		}
		pages++
		if len(p.Items) == 0 {
			return moved, nil
		}
		n, err := st.savePage(ctx, p, time.Now())
		if err != nil {
			return moved, err
		}
		moved = true
		changed += n
		// Stay polite during the first catch-up, which reads the whole feed.
		select {
		case <-ctx.Done():
			return moved, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}
