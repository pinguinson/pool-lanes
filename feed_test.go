package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSession(t *testing.T) {
	tests := []struct {
		name string
		data string
		want *session
	}{
		{
			name: "one lane",
			data: `{"startDate":"2026-10-29T14:00:00+00:00","endDate":"2026-10-29T15:00:00+00:00",
				"superEvent":"https://data.everyoneactive.com/OpenActive/api/session-series/3304WS4MC140096",
				"maximumAttendeeCapacity":12,"remainingAttendeeCapacity":9,
				"beta:sportsActivityLocation":[{"name":"Competition Pool Lane 2"}]}`,
			want: &session{Series: "3304WS4MC140096", Pool: "Competition", Kind: "Lane Swim", Speed: "Medium",
				Lanes: "Competition Pool Lane 2", Capacity: 12, Remaining: 9},
		},
		{
			name: "public swim across several lanes",
			data: `{"startDate":"2026-10-30T07:00:00+00:00","endDate":"2026-10-30T08:00:00+00:00",
				"superEvent":"https://data.everyoneactive.com/OpenActive/api/session-series/3305WS4AT070096",
				"maximumAttendeeCapacity":12,"remainingAttendeeCapacity":9,
				"beta:sportsActivityLocation":[{"name":["Training Pool Lane 1b","Training Pool Lane 2b"]}]}`,
			want: &session{Series: "3305WS4AT070096", Pool: "Training", Kind: "Public Swim",
				Lanes: "Training Pool Lane 1b, Training Pool Lane 2b", Capacity: 12, Remaining: 9},
		},
		{
			name: "disabled swim",
			data: `{"startDate":"2026-10-30T10:00:00+00:00","endDate":"2026-10-30T11:00:00+00:00",
				"superEvent":"https://data.everyoneactive.com/OpenActive/api/session-series/3305WS4CT100096",
				"maximumAttendeeCapacity":10,"remainingAttendeeCapacity":9,
				"beta:sportsActivityLocation":[{"name":"Training Pool Lane 8b"}]}`,
			want: &session{Series: "3305WS4CT100096", Pool: "Training", Kind: "Disabled Swim",
				Lanes: "Training Pool Lane 8b", Capacity: 10, Remaining: 9},
		},
		{
			name: "unknown kind",
			data: `{"startDate":"2026-10-30T10:00:00+00:00","endDate":"2026-10-30T11:00:00+00:00",
				"superEvent":"https://data.everyoneactive.com/OpenActive/api/session-series/3305WS4XT100096",
				"maximumAttendeeCapacity":10,"remainingAttendeeCapacity":10}`,
			want: &session{Series: "3305WS4XT100096", Pool: "Training", Kind: "X", Capacity: 10, Remaining: 10},
		},
		{
			name: "gym induction",
			data: `{"startDate":"2026-10-29T15:30:00+00:00","endDate":"2026-10-29T16:30:00+00:00",
				"superEvent":"https://data.everyoneactive.com/OpenActive/api/session-series/3303B1515310124"}`,
		},
		{
			name: "another centre's lane swimming",
			data: `{"startDate":"2026-10-29T15:30:00+00:00","endDate":"2026-10-29T16:30:00+00:00",
				"superEvent":"https://data.everyoneactive.com/OpenActive/api/session-series/1554WS4MC140096"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := parseSession(item{ID: "1", State: "updated", Data: json.RawMessage(tt.data)})
			if err != nil {
				t.Fatal(err)
			}
			if ok != (tt.want != nil) {
				t.Fatalf("ok = %v, want %v", ok, tt.want != nil)
			}
			if !ok {
				return
			}
			got.ID, got.Start, got.End = "", got.Start.UTC(), got.End.UTC()
			want := *tt.want
			want.Start, want.End = got.Start, got.End
			if got != want {
				t.Errorf("got %+v\nwant %+v", got, want)
			}
		})
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCatchUpStartsOverWhenCursorIsGone(t *testing.T) {
	st, err := openStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.db.Close()
	const old = feedURL + "?afterTimestamp=1211963832&afterId=00000000001950306579"
	if _, err := st.db.Exec(`INSERT INTO cursor (id, url) VALUES (1, ?)`, old); err != nil {
		t.Fatal(err)
	}
	var got []string
	client := &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		got = append(got, r.URL.String())
		if r.URL.String() == old {
			return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found",
				Body: io.NopCloser(strings.NewReader(`{"message":"feed not found"}`))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{"next":"` + feedURL + `","items":[]}`))}, nil
	})}

	if _, err := catchUp(context.Background(), client, st); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1] != feedURL {
		t.Errorf("requested %v, want the old cursor then the feed's start", got)
	}
}
