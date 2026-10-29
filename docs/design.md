# Design

Decisions made before building, in the terms of [GLOSSARY.md](../GLOSSARY.md). [docs/adr/](adr/) records why the harder ones went the way they did.

## The question it answers

"How busy is the Training Pool on Wednesday at 10:30?" A time resolves to the Swim Sessions in progress then, in each Pool. If no Lane Swim Session covers that time, the answer says so and names the nearest sessions.

Every answer gives both:

- **Live**: the current Pool Busyness of the next matching date's sessions, broken down by Speed.
- **Typical**: Typical Busyness for that Pool, Speed, weekday and start time, or "no history yet" when the slot is new, e.g. after a timetable change. History is never borrowed from a nearby slot.

## UI (v1)

- A web page with a week grid covering the next 7 days and one row per Pool. Each cell shows live Pool Busyness, with Typical Busyness beside it.
- A cell's detail breaks Pool Busyness down by Speed.
- Public Swim gets its own row per Pool, because it takes Lanes away from lane swimmers. Disabled Swim is hidden.
- Cancelled Sessions show as cancelled.
- The page credits Everyone Active as the source; the feed's CC BY 4.0 licence requires it.
- No login: it's reachable only over the tailnet.
- Desktop-first. It should still work on a phone, but the layout is designed for a desktop screen.

## Data

- Read every 5 minutes from Everyone Active's OpenActive feed ([ADR 0001](adr/0001-openactive-feed-as-data-source.md)). Every change to a session's places is stored with its time, in SQLite, kept forever.
- Final Busyness comes from the last change recorded before a session's start.
- The feed publishes 14 days ahead.

## Deployment

The image is built by GitHub Actions and published to `ghcr.io/pinguinson/pool-lanes`. It runs as an App in the [homelab](https://github.com/pinguinson/homelab) repo: a stack with its own Tailscale Sidecar and health check, the image pinned by digest and kept up to date by Renovate, data in `/mnt/user/appdata/pool-lanes/`.

## Out of scope for v1

Alerts, booking, other Centres, and activities other than swimming. Later: an MCP server answering the same question from the same data.

## Status (29 Oct 2026)

- The reader is built and has been running on the developer's Mac since 29 Oct, writing to `pool-lanes.db` in the repo root, to prove over 48 hours that the feed's places change live.
- The first full download showed real bookings, but the first polls after it saw no changes. If counts stay flat, fall back to scraping the Connect booking portal, as ADR 0001 says.
- Unconfirmed: whether the feed drops a session only when it's cancelled, or also once it's over. Check with the poller's data before using the `deleted_at` column to mean cancelled.
