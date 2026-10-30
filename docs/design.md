# Design

Decisions made before building, in the terms of [GLOSSARY.md](../GLOSSARY.md). [docs/adr/](adr/) records why the harder ones went the way they did.

## The question it answers

"How busy is the Training Pool's Medium lane on Wednesday at 10:30?" A time resolves to the Lane Swim Sessions in progress then, in one Pool at one Speed, and the answer is their Speed Busyness right now.

v1 shows live numbers only. Typical Busyness and any other prediction wait until there's enough history to base them on.

## UI (v1)

- Six pages, one per Pool and Speed: `/competition/fast` … `/training/slow`, with a Pool × Speed nav at the top. `/` redirects to `/competition/medium`, and `/competition` or `/training` to that Pool's Medium page.
- Each page is a grid with one column per day, from today to today + 6, and one row per 15 minutes in Europe/London time. Swim Sessions last an hour and start on the quarter hour, staggered by Lane, so 15 minutes is the step at which Speed Busyness is exact. Only times at which that week has a Lane Swim Session at that Speed, or a Public Swim, get a row.
- A cell shows Speed Busyness as a percentage, shaded darker the fuller it is, with booked/capacity underneath. A cell is empty when no Lane Swim Session at that Speed is running. On desktop, hovering shows the cell's Lanes and each session's start–end and booked/capacity.
- Times today that have already passed are greyed out and show no numbers.
- Cancelled Sessions don't count towards Speed Busyness. A cell whose sessions are all cancelled says "Cancelled". A cell where only some are cancelled shows the rest, plus a note such as "1 lane cancelled".
- A Public Swim shows as a badge in the cells it overlaps on every Speed page of its Pool, e.g. "Public Swim · 2 lanes · 4/12", because it takes Lanes away from lane swimmers. A Public Swim over the whole Pool replaces the cell's contents. Disabled Swim is hidden.
- The page says when the feed was last checked successfully, in red once that's more than 15 minutes ago. It also warns, in red, when the feed has had nothing new for more than 15 minutes. The feed covers every Everyone Active centre, so it normally has something new at every check, and going without means it's frozen, as it was after an outage on 30 Oct 2026. The page reloads itself every 5 minutes.
- The page credits Everyone Active as the source; the feed's CC BY 4.0 licence requires it.
- No login: it's reachable only over the tailnet.
- Desktop-first. It should still work on a phone, but the layout is designed for a desktop screen.
- Served by the same binary as the poller, from the same SQLite database, when `-addr` is given: server-rendered HTML, no JavaScript build, and a `/healthz` endpoint for the health check.

## Data

- Read every 5 minutes from Everyone Active's OpenActive feed ([ADR 0001](adr/0001-openactive-feed-as-data-source.md)). Every change to a session's places is stored with its time, in SQLite, kept forever.
- Final Busyness comes from the last change recorded before a session's start.
- The feed publishes 14 days ahead.

## Deployment

The image is built by GitHub Actions and published to `ghcr.io/pinguinson/pool-lanes`. It runs as an App in the [homelab](https://github.com/pinguinson/homelab) repo: a stack with its own Tailscale Sidecar and health check, the image pinned by digest and kept up to date by Renovate, data in `/mnt/user/appdata/pool-lanes/`.

## Out of scope for v1

Alerts, booking, other Centres, and activities other than swimming. Later: an MCP server answering the same question from the same data.

## Status (29 Oct 2026)

- The reader has been running on the developer's Mac since 29 Oct, writing to `pool-lanes.db` in the repo root. Within its first five hours it recorded 109 changes to places, so the feed carries this Centre's bookings live, and building on it went ahead before the 48 hours were up.
- The feed drops a Swim Session once it has started: all 66 dropped in the first five hours had already started. So `deleted_at` means cancelled only when it's before `start`.
- The v1 web UI is built and served by the poller on the developer's Mac. The Docker image is built and published by GitHub Actions. Next: the homelab App.
