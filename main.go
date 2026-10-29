// Command pool-lanes records how booked-up lane swimming is at London Aquatics
// Centre, from Everyone Active's OpenActive feed.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	dbPath := flag.String("db", "pool-lanes.db", "SQLite database path")
	interval := flag.Duration("interval", 5*time.Minute, "how often to check the feed for changes")
	once := flag.Bool("once", false, "catch up with the feed once, then exit")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := openStore(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer st.db.Close()
	client := &http.Client{Timeout: time.Minute}

	for {
		if err := catchUp(ctx, client, st); err != nil && ctx.Err() == nil {
			log.Print(err)
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

// catchUp reads every page after the stored cursor. The feed's last page is
// empty and links to itself, which is where the next catch-up starts.
func catchUp(ctx context.Context, client *http.Client, st *store) error {
	pages, changed := 0, 0
	defer func() { log.Printf("read %d pages, %d sessions changed", pages, changed) }()
	for {
		url, err := st.cursor(ctx)
		if err != nil {
			return err
		}
		p, err := fetchPage(ctx, client, url)
		if err != nil {
			return err
		}
		pages++
		if len(p.Items) == 0 {
			return nil
		}
		n, err := st.savePage(ctx, p, time.Now())
		if err != nil {
			return err
		}
		changed += n
		// Stay polite during the first catch-up, which reads the whole feed.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}
