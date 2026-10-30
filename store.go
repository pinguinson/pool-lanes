package main

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS cursor (
	id  INTEGER PRIMARY KEY CHECK (id = 1),
	url TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
	id         TEXT PRIMARY KEY,
	series     TEXT NOT NULL,
	pool       TEXT NOT NULL,
	kind       TEXT NOT NULL,
	speed      TEXT NOT NULL, -- Empty unless kind is 'Lane Swim'.
	lanes      TEXT NOT NULL,
	start      TEXT NOT NULL,
	end        TEXT NOT NULL,
	-- When the feed dropped the session. That happens when it's cancelled,
	-- and possibly once it's over.
	deleted_at TEXT
);
-- One row per change to a session's places, so the latest row per session is
-- its current state.
CREATE TABLE IF NOT EXISTS observations (
	session_id TEXT NOT NULL REFERENCES sessions (id),
	seen_at    TEXT NOT NULL,
	capacity   INTEGER NOT NULL,
	remaining  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS observations_session ON observations (session_id, seen_at);
`

type store struct{ db *sql.DB }

func openStore(path string) (*store, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &store{db}, nil
}

func (s *store) cursor(ctx context.Context) (string, error) {
	var url string
	err := s.db.QueryRowContext(ctx, `SELECT url FROM cursor`).Scan(&url)
	if errors.Is(err, sql.ErrNoRows) {
		return feedURL, nil
	}
	return url, err
}

// resetCursor makes the next read start from the feed's first page.
func (s *store) resetCursor(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM cursor`)
	return err
}

// savePage records a page's sessions and moves the cursor past it, together,
// so a crash never skips or half-applies a page.
func (s *store) savePage(ctx context.Context, p page, now time.Time) (changed int, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	ts := now.UTC().Format(time.RFC3339)
	for _, it := range p.Items {
		if it.State == "deleted" {
			// Deleted items carry no data, so only sessions already seen can be marked.
			if _, err := tx.ExecContext(ctx,
				`UPDATE sessions SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`, ts, it.ID); err != nil {
				return 0, err
			}
			continue
		}
		sess, ok, err := parseSession(it)
		if err != nil {
			return 0, err
		}
		if !ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sessions (id, series, pool, kind, speed, lanes, start, end) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (id) DO UPDATE SET series = excluded.series, pool = excluded.pool,
				kind = excluded.kind, speed = excluded.speed, lanes = excluded.lanes, start = excluded.start,
				end = excluded.end, deleted_at = NULL`,
			sess.ID, sess.Series, sess.Pool, sess.Kind, sess.Speed, sess.Lanes,
			sess.Start.UTC().Format(time.RFC3339), sess.End.UTC().Format(time.RFC3339)); err != nil {
			return 0, err
		}
		res, err := tx.ExecContext(ctx, `
			INSERT INTO observations (session_id, seen_at, capacity, remaining)
			SELECT ?1, ?2, ?3, ?4
			WHERE NOT EXISTS (
				SELECT 1 FROM (
					SELECT capacity, remaining FROM observations
					WHERE session_id = ?1 ORDER BY seen_at DESC, rowid DESC LIMIT 1
				) WHERE capacity = ?3 AND remaining = ?4)`,
			sess.ID, ts, sess.Capacity, sess.Remaining)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		changed += int(n)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO cursor (id, url) VALUES (1, ?) ON CONFLICT (id) DO UPDATE SET url = excluded.url`, p.Next); err != nil {
		return 0, err
	}
	return changed, tx.Commit()
}

// loadSwims returns, with their latest places, a Pool's Lane Swim Sessions at
// one Speed and its Public Swims, starting in [from, to).
func (s *store) loadSwims(ctx context.Context, pool, speed string, from, to time.Time) ([]swim, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.kind, s.speed, s.lanes, s.start, s.end, s.deleted_at < s.start, o.capacity, o.remaining
		FROM sessions s JOIN observations o ON o.rowid = (
			SELECT rowid FROM observations WHERE session_id = s.id ORDER BY seen_at DESC, rowid DESC LIMIT 1)
		WHERE s.pool = ? AND s.start >= ? AND s.start < ?
			AND ((s.kind = ? AND s.speed = ?) OR s.kind = 'Public Swim')`,
		pool, from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339), laneSwim, speed)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var swims []swim
	for rows.Next() {
		var sw swim
		var start, end string
		var cancelled sql.NullBool
		if err := rows.Scan(&sw.ID, &sw.Kind, &sw.Speed, &sw.Lanes, &start, &end, &cancelled, &sw.Capacity, &sw.Remaining); err != nil {
			return nil, err
		}
		if sw.Start, err = time.Parse(time.RFC3339, start); err != nil {
			return nil, err
		}
		if sw.End, err = time.Parse(time.RFC3339, end); err != nil {
			return nil, err
		}
		sw.Pool, sw.Cancelled = pool, cancelled.Bool
		swims = append(swims, sw)
	}
	return swims, rows.Err()
}
