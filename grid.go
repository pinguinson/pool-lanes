package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// step is how far apart a page's rows are. Swim Sessions start on the quarter
// hour, so Speed Busyness only changes on a step.
const step = 15 * time.Minute

// A Swim Session with its latest places, as a page shows it.
type swim struct {
	session
	Cancelled bool
}

func (s swim) booked() int { return s.Capacity - s.Remaining }

// A week of one Pool at one Speed: a column per day and a row per step at
// which some Swim Session runs.
type grid struct {
	Days []time.Time
	Rows []row
}

type row struct {
	Time  string // Local clock time, e.g. "10:15".
	Cells []cell
}

type cell struct {
	Past bool
	// No Lane Swim Session runs then, cancelled or not.
	Empty bool
	// Speed Busyness, over the sessions that aren't cancelled.
	Booked, Capacity, Percent int
	Cancelled                 int
	AllCancelled              bool
	Public                    []publicSwim
	Detail                    string // One line per Lane Swim Session.
}

type publicSwim struct{ Lanes, Booked, Capacity int }

// buildGrid lays out swims, the Lane Swim Sessions of one Speed and the Public
// Swims of their Pool, over the 7 days from now's date in loc.
func buildGrid(swims []swim, now time.Time, loc *time.Location) grid {
	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	var g grid
	for i := range 7 {
		g.Days = append(g.Days, today.AddDate(0, 0, i))
	}

	clocks := map[int]bool{} // Minutes past midnight.
	for _, s := range swims {
		for t := s.Start.In(loc); t.Before(s.End); t = t.Add(step) {
			clocks[t.Hour()*60+t.Minute()] = true
		}
	}
	var sorted []int
	for m := range clocks {
		sorted = append(sorted, m)
	}
	sort.Ints(sorted)

	for _, m := range sorted {
		r := row{Time: fmt.Sprintf("%02d:%02d", m/60, m%60)}
		for _, d := range g.Days {
			t := time.Date(d.Year(), d.Month(), d.Day(), m/60, m%60, 0, 0, loc)
			r.Cells = append(r.Cells, buildCell(swims, t, t.Before(now)))
		}
		g.Rows = append(g.Rows, r)
	}
	return g
}

func buildCell(swims []swim, t time.Time, past bool) cell {
	c := cell{Past: past, Empty: true}
	var detail []string
	for _, s := range swims {
		if t.Before(s.Start) || !t.Before(s.End) {
			continue
		}
		if s.Kind != laneSwim {
			if !s.Cancelled {
				c.Public = append(c.Public, publicSwim{Lanes: len(strings.Split(s.Lanes, ", ")), Booked: s.booked(), Capacity: s.Capacity})
			}
			continue
		}
		c.Empty = false
		line := fmt.Sprintf("%s · %s–%s · ", s.Lanes, s.Start.In(t.Location()).Format("15:04"), s.End.In(t.Location()).Format("15:04"))
		if s.Cancelled {
			c.Cancelled++
			detail = append(detail, line+"cancelled")
			continue
		}
		c.Booked += s.booked()
		c.Capacity += s.Capacity
		detail = append(detail, line+fmt.Sprintf("%d/%d booked", s.booked(), s.Capacity))
	}
	c.AllCancelled = !c.Empty && c.Capacity == 0 && c.Cancelled > 0
	if c.Capacity > 0 {
		c.Percent = int(math.Round(100 * float64(c.Booked) / float64(c.Capacity)))
	}
	c.Detail = strings.Join(detail, "\n")
	return c
}
