package main

import (
	"bytes"
	_ "embed"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"
)

//go:embed page.html
var pageHTML string

var pageTmpl = template.Must(template.New("page").Parse(pageHTML))

// URL slugs for the Pools and Speeds, in nav order.
var (
	poolSlugs  = []slug{{"competition", "Competition"}, {"training", "Training"}}
	speedSlugs = []slug{{"fast", "Fast"}, {"medium", "Medium"}, {"slow", "Slow"}}
)

type slug struct{ Slug, Name string }

func lookup(slugs []slug, s string) (string, bool) {
	for _, sl := range slugs {
		if sl.Slug == s {
			return sl.Name, true
		}
	}
	return "", false
}

// A check of the feed older than this is shown as stale.
const staleAfter = 15 * time.Minute

type server struct {
	st  *store
	loc *time.Location
	now func() time.Time

	mu          sync.Mutex
	lastChecked time.Time
}

// checked records a successful check of the feed.
func (s *server) checked(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastChecked = t
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/competition/medium", http.StatusFound)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.st.db.PingContext(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
	})
	mux.HandleFunc("GET /{pool}", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := lookup(poolSlugs, r.PathValue("pool")); !ok {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/"+r.PathValue("pool")+"/medium", http.StatusFound)
	})
	mux.HandleFunc("GET /{pool}/{speed}", s.page)
	return mux
}

func (s *server) page(w http.ResponseWriter, r *http.Request) {
	pool, ok := lookup(poolSlugs, r.PathValue("pool"))
	speed, ok2 := lookup(speedSlugs, r.PathValue("speed"))
	if !ok || !ok2 {
		http.NotFound(w, r)
		return
	}
	now := s.now()
	local := now.In(s.loc)
	from := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.loc)
	swims, err := s.st.loadSwims(r.Context(), pool, speed, from, from.AddDate(0, 0, 7))
	if err != nil {
		log.Print(err)
		http.Error(w, "couldn't read the database", http.StatusInternalServerError)
		return
	}
	s.mu.Lock()
	checked := s.lastChecked
	s.mu.Unlock()

	data := struct {
		Pool, Speed         string
		PoolSlug, SpeedSlug string
		Pools, Speeds       []slug
		Grid                grid
		LastChecked         string
		Stale               bool
	}{
		Pool: pool, Speed: speed,
		PoolSlug: r.PathValue("pool"), SpeedSlug: r.PathValue("speed"),
		Pools: poolSlugs, Speeds: speedSlugs,
		Grid:  buildGrid(swims, now, s.loc),
		Stale: now.Sub(checked) > staleAfter,
	}
	if !checked.IsZero() {
		data.LastChecked = checked.In(s.loc).Format("15:04")
	}
	var buf bytes.Buffer
	if err := pageTmpl.Execute(&buf, data); err != nil {
		log.Print(err)
		http.Error(w, "couldn't render the page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}
