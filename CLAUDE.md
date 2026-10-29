# pool-lanes

Shows how booked-up swimming is at London Aquatics Centre.

- [GLOSSARY.md](GLOSSARY.md): the domain language. Use its terms in code, UI copy and commits: Swim Session, Pool Busyness, Typical Busyness and so on.
- [docs/design.md](docs/design.md): product decisions, the UI spec, deployment and current status. Read it before working on the UI, queries over the stored data, or deployment.
- [docs/adr/](docs/adr/): why the hard-to-reverse decisions went the way they did. Read it before changing the data source.

## Gotchas

- Swim Session kinds, Speeds and Pools are decoded from the feed's session series IDs (`feed.go`). That layout was inferred from the data and isn't documented by Everyone Active. A series letter not in the `kinds` map is stored as it is, as a raw kind.
- A poller may be running from the repo root against `pool-lanes.db`. Check with `pgrep -fl 'pool-lanes -db'` before touching the database or schema, and stop it first.
