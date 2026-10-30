package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const feedURL = "https://opendata.leisurecloud.live/api/feeds/EveryoneActive-live-scheduled-sessions"

// The feed is CC BY 4.0, published by Everyone Active through Gladstone's
// leisurecloud. It rejects some default user agents.
const userAgent = "pool-lanes (+https://github.com/pinguinson/pool-lanes)"

// A session series ID such as 3304WS4MC140096 reads: centre 330, weekday 4,
// swimming (WS4), kind M, pool C, start 14:00. Inferred from the feed, not
// documented.
var swimSeries = regexp.MustCompile(`/session-series/330\dWS4([A-Z])([CT])\d+$`)

const laneSwim = "Lane Swim"

type kind struct{ name, speed string }

// Kinds missing from here are stored under their letter, so new ones show up
// in the data.
var kinds = map[string]kind{
	"F": {laneSwim, "Fast"},
	"M": {laneSwim, "Medium"},
	"S": {laneSwim, "Slow"},
	"A": {"Public Swim", ""},
	"C": {"Disabled Swim", ""},
}

var pools = map[string]string{"C": "Competition", "T": "Training"}

// errPageGone is the feed's answer to a page past its end. That happens when
// its timestamps go backwards, as they did after an outage on 30 Oct 2026.
var errPageGone = errors.New("feed has no such page")

type page struct {
	Next  string `json:"next"`
	Items []item `json:"items"`
}

type item struct {
	ID    string          `json:"id"`
	State string          `json:"state"`
	Data  json.RawMessage `json:"data"`
}

type scheduledSession struct {
	Start      time.Time `json:"startDate"`
	End        time.Time `json:"endDate"`
	SuperEvent string    `json:"superEvent"`
	Capacity   int       `json:"maximumAttendeeCapacity"`
	Remaining  int       `json:"remainingAttendeeCapacity"`
	Locations  []struct {
		// Either one lane's name or a list of them.
		Name json.RawMessage `json:"name"`
	} `json:"beta:sportsActivityLocation"`
}

// A swim session at London Aquatics Centre, as last seen in the feed.
type session struct {
	ID        string
	Series    string
	Pool      string
	Kind      string
	Speed     string // Lane swims only.
	Lanes     string
	Start     time.Time
	End       time.Time
	Capacity  int
	Remaining int
}

// parseSession returns false for anything that isn't a London Aquatics Centre
// swim session.
func parseSession(it item) (session, bool, error) {
	var d scheduledSession
	if err := json.Unmarshal(it.Data, &d); err != nil {
		return session{}, false, fmt.Errorf("item %s: %w", it.ID, err)
	}
	m := swimSeries.FindStringSubmatch(d.SuperEvent)
	if m == nil {
		return session{}, false, nil
	}
	k, ok := kinds[m[1]]
	if !ok {
		k = kind{name: m[1]}
	}
	var lanes []string
	for _, l := range d.Locations {
		var one string
		var many []string
		if json.Unmarshal(l.Name, &one) == nil {
			lanes = append(lanes, one)
		} else if json.Unmarshal(l.Name, &many) == nil {
			lanes = append(lanes, many...)
		}
	}
	return session{
		ID:        it.ID,
		Series:    d.SuperEvent[strings.LastIndex(d.SuperEvent, "/")+1:],
		Pool:      pools[m[2]],
		Kind:      k.name,
		Speed:     k.speed,
		Lanes:     strings.Join(lanes, ", "),
		Start:     d.Start,
		End:       d.End,
		Capacity:  d.Capacity,
		Remaining: d.Remaining,
	}, true, nil
}

func fetchPage(ctx context.Context, client *http.Client, url string) (page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return page{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return page{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return page{}, fmt.Errorf("GET %s: %w", url, errPageGone)
	}
	if resp.StatusCode != http.StatusOK {
		return page{}, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	var p page
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return page{}, fmt.Errorf("GET %s: %w", url, err)
	}
	return p, nil
}
