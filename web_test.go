package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testStore(t *testing.T) *store {
	t.Helper()
	st, err := openStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.db.Close() })
	// 3 Nov, Training Pool: a Medium Lane Swim booked twice, a Slow one, a
	// Public Swim, a Medium one cancelled before it started and one dropped
	// once it was over, and a Competition one.
	_, err = st.db.Exec(`
		INSERT INTO sessions (id, series, pool, kind, speed, lanes, start, end, deleted_at) VALUES
			('m',  'x', 'Training', 'Lane Swim', 'Medium', 'Training Pool Lane 4b', '2026-11-03T10:00:00Z', '2026-11-03T11:00:00Z', NULL),
			('s',  'x', 'Training', 'Lane Swim', 'Slow', 'Training Pool Lane 5b', '2026-11-03T10:00:00Z', '2026-11-03T11:00:00Z', NULL),
			('p',  'x', 'Training', 'Public Swim', '', 'Training Pool Lane 1b, Training Pool Lane 2b', '2026-11-03T10:00:00Z', '2026-11-03T11:00:00Z', NULL),
			('c',  'x', 'Training', 'Lane Swim', 'Medium', 'Training Pool Lane 3b', '2026-11-03T12:00:00Z', '2026-11-03T13:00:00Z', '2026-11-02T09:00:00Z'),
			('o',  'x', 'Training', 'Lane Swim', 'Medium', 'Training Pool Lane 3b', '2026-11-02T07:00:00Z', '2026-11-02T08:00:00Z', '2026-11-02T07:05:00Z'),
			('cm', 'x', 'Competition', 'Lane Swim', 'Medium', 'Competition Pool Lane 2', '2026-11-03T10:00:00Z', '2026-11-03T11:00:00Z', NULL);
		INSERT INTO observations (session_id, seen_at, capacity, remaining) VALUES
			('m', '2026-11-01T10:00:00Z', 8, 8), ('m', '2026-11-02T07:00:00Z', 8, 5),
			('s', '2026-11-01T10:00:00Z', 8, 1), ('p', '2026-11-01T10:00:00Z', 12, 9),
			('c', '2026-11-01T10:00:00Z', 8, 4), ('o', '2026-11-01T10:00:00Z', 8, 0),
			('cm', '2026-11-01T10:00:00Z', 12, 0);`)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestLoadSwims(t *testing.T) {
	st := testStore(t)
	got, err := st.loadSwims(context.Background(), "Training", "Medium", at(t, "2026-11-02T00:00"), at(t, "2026-11-09T00:00"))
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]swim{}
	for _, s := range got {
		byID[s.ID] = s
	}
	if len(byID) != 4 {
		t.Fatalf("got %d swims, want m, p, c and o: %+v", len(byID), got)
	}
	if m := byID["m"]; m.Remaining != 5 || m.Capacity != 8 || m.Cancelled {
		t.Errorf("m = %+v, want its latest places, 5 of 8 left", m)
	}
	if !byID["c"].Cancelled {
		t.Error("c was dropped before it started but isn't cancelled")
	}
	if byID["o"].Cancelled {
		t.Error("o was dropped once it had started but is cancelled")
	}
}

func testServer(t *testing.T) *server {
	srv := &server{st: testStore(t), loc: mustLoc(t), now: func() time.Time { return at(t, "2026-11-02T08:00") }}
	srv.checked(at(t, "2026-11-02T07:55"))
	return srv
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestPage(t *testing.T) {
	w := get(t, testServer(t).handler(), "/training/medium")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	body := w.Body.String()
	for _, want := range []string{
		"38%", "3/8", // Lane 4b at 10:00.
		"Public Swim · 2 lanes · 3/12",
		"Cancelled", // 12:00.
		"Last checked 07:55",
		"Everyone Active",
		`href="/competition/fast"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if strings.Contains(body, "7/8") {
		t.Error("page shows the Slow session")
	}
}

func TestStale(t *testing.T) {
	srv := testServer(t)
	srv.checked(at(t, "2026-11-02T07:40"))
	if body := get(t, srv.handler(), "/training/medium").Body.String(); !strings.Contains(body, `class="stale"`) {
		t.Error("a 20-minute-old check isn't marked stale")
	}
}

func TestRoutes(t *testing.T) {
	h := testServer(t).handler()
	for path, want := range map[string]string{"/": "/competition/medium", "/training": "/training/medium"} {
		w := get(t, h, path)
		if w.Code != http.StatusFound || w.Header().Get("Location") != want {
			t.Errorf("GET %s = %d to %q, want redirect to %s", path, w.Code, w.Header().Get("Location"), want)
		}
	}
	for _, path := range []string{"/diving", "/diving/fast", "/training/sprint"} {
		if w := get(t, h, path); w.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, w.Code)
		}
	}
	if w := get(t, h, "/healthz"); w.Code != http.StatusOK {
		t.Errorf("GET /healthz = %d", w.Code)
	}
}
