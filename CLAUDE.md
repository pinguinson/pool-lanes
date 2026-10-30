# pool-lanes

Shows how booked-up swimming is at London Aquatics Centre.

- [GLOSSARY.md](GLOSSARY.md): the domain language. Use its terms in code, UI copy and commits: Swim Session, Pool Busyness, Typical Busyness and so on.
- [docs/design.md](docs/design.md): product decisions, the UI spec, deployment and current status. Read it before working on the UI, queries over the stored data, or deployment.
- [docs/adr/](docs/adr/): why the hard-to-reverse decisions went the way they did. Read it before changing the data source.

## Gotchas

- Swim Session kinds, Speeds and Pools are decoded from the feed's session series IDs (`feed.go`). That layout was inferred from the data and isn't documented by Everyone Active. A series letter not in the `kinds` map is stored as it is, as a raw kind.
- The live database is on the homelab server, not in the repo: `/mnt/user/appdata/pool-lanes/data/pool-lanes.db` on Tower, in the `pool-lanes-pool-lanes-1` container. `pool-lanes.db` in the repo root is a stale snapshot from 30 Oct. The schema is only ever created, never migrated, so a schema change needs a migration for the live database.
