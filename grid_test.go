package main

import (
	"testing"
	"time"
)

func mustLoc(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

// at parses a UTC time such as "2026-11-02T10:00".
func at(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse("2006-01-02T15:04", s)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func laneSwimAt(t *testing.T, start, lane string, capacity, remaining int) swim {
	s := at(t, start)
	return swim{session: session{Kind: laneSwim, Speed: "Medium", Lanes: "Competition Pool " + lane,
		Start: s, End: s.Add(time.Hour), Capacity: capacity, Remaining: remaining}}
}

func findCell(t *testing.T, g grid, day int, clock string) cell {
	t.Helper()
	for _, r := range g.Rows {
		if r.Time == clock {
			return r.Cells[day]
		}
	}
	t.Fatalf("no row at %s", clock)
	return cell{}
}

func TestBuildGrid(t *testing.T) {
	loc := mustLoc(t)
	now := at(t, "2026-11-02T08:00") // Monday, GMT.

	t.Run("staggered sessions add up while they overlap", func(t *testing.T) {
		g := buildGrid([]swim{
			laneSwimAt(t, "2026-11-03T10:00", "Lane 2", 12, 6),
			laneSwimAt(t, "2026-11-03T10:30", "Lane 3", 12, 12),
		}, now, loc)
		if c := findCell(t, g, 1, "10:15"); c.Booked != 6 || c.Capacity != 12 || c.Percent != 50 {
			t.Errorf("10:15 = %d/%d (%d%%), want 6/12 (50%%)", c.Booked, c.Capacity, c.Percent)
		}
		if c := findCell(t, g, 1, "10:45"); c.Booked != 6 || c.Capacity != 24 || c.Percent != 25 {
			t.Errorf("10:45 = %d/%d (%d%%), want 6/24 (25%%)", c.Booked, c.Capacity, c.Percent)
		}
		if c := findCell(t, g, 1, "11:15"); c.Booked != 0 || c.Capacity != 12 {
			t.Errorf("11:15 = %d/%d, want 0/12", c.Booked, c.Capacity)
		}
	})

	t.Run("rows only where something runs, in London time", func(t *testing.T) {
		g := buildGrid([]swim{laneSwimAt(t, "2026-11-03T10:00", "Lane 2", 12, 6)}, now, loc)
		var times []string
		for _, r := range g.Rows {
			times = append(times, r.Time)
		}
		want := []string{"10:00", "10:15", "10:30", "10:45"}
		if len(times) != len(want) {
			t.Fatalf("rows = %v, want %v", times, want)
		}
		for i := range want {
			if times[i] != want[i] {
				t.Fatalf("rows = %v, want %v", times, want)
			}
		}
		if c := findCell(t, g, 0, "10:00"); !c.Empty {
			t.Error("Monday 10:00 has no session but isn't empty")
		}
	})

	t.Run("British Summer Time", func(t *testing.T) {
		g := buildGrid([]swim{laneSwimAt(t, "2026-07-01T09:00", "Lane 2", 12, 6)}, at(t, "2026-07-01T06:00"), loc)
		if c := findCell(t, g, 0, "10:00"); c.Empty {
			t.Error("09:00 UTC in July should show at 10:00")
		}
	})

	t.Run("seven days from today", func(t *testing.T) {
		g := buildGrid(nil, now, loc)
		if len(g.Days) != 7 || g.Days[0].Day() != 2 || g.Days[6].Day() != 8 {
			t.Errorf("days = %v", g.Days)
		}
	})

	t.Run("times already passed today", func(t *testing.T) {
		g := buildGrid([]swim{
			laneSwimAt(t, "2026-11-02T07:00", "Lane 2", 12, 6),
			laneSwimAt(t, "2026-11-03T07:00", "Lane 2", 12, 6),
		}, now, loc)
		if c := findCell(t, g, 0, "07:45"); !c.Past {
			t.Error("today 07:45 isn't past")
		}
		if c := findCell(t, g, 1, "07:45"); c.Past {
			t.Error("tomorrow 07:45 is past")
		}
	})

	t.Run("some sessions cancelled", func(t *testing.T) {
		cancelled := laneSwimAt(t, "2026-11-03T10:00", "Lane 3", 12, 2)
		cancelled.Cancelled = true
		g := buildGrid([]swim{laneSwimAt(t, "2026-11-03T10:00", "Lane 2", 12, 6), cancelled}, now, loc)
		c := findCell(t, g, 1, "10:00")
		if c.Booked != 6 || c.Capacity != 12 || c.Cancelled != 1 || c.AllCancelled {
			t.Errorf("cell = %+v, want 6/12 with 1 cancelled", c)
		}
	})

	t.Run("all sessions cancelled", func(t *testing.T) {
		cancelled := laneSwimAt(t, "2026-11-03T10:00", "Lane 3", 12, 2)
		cancelled.Cancelled = true
		g := buildGrid([]swim{cancelled}, now, loc)
		if c := findCell(t, g, 1, "10:00"); !c.AllCancelled || c.Empty {
			t.Errorf("cell = %+v, want all cancelled", c)
		}
	})

	t.Run("public swim", func(t *testing.T) {
		start := at(t, "2026-11-03T07:00")
		g := buildGrid([]swim{{session: session{Kind: "Public Swim", Lanes: "Training Pool Lane 1b, Training Pool Lane 2b",
			Start: start, End: start.Add(time.Hour), Capacity: 12, Remaining: 8}}}, now, loc)
		c := findCell(t, g, 1, "07:30")
		if !c.Empty || len(c.Public) != 1 || c.Public[0] != (publicSwim{Lanes: 2, Booked: 4, Capacity: 12}) {
			t.Errorf("cell = %+v, want only a 2-lane Public Swim at 4/12", c)
		}
	})
}
