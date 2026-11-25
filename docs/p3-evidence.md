# P3 evidence record (2026-11-24)

All live calls ran from the authorized shell with org-revokable tokens held
in process memory or 600 files; no credential value was emitted anywhere.
Every created resource was deleted and deletion verified.

## Topology spike (Sprite per generated app)

Throwaway probe (`/tmp/p3spike`, uncommitted) driving the committed
`fly.SpriteAdapter` + `fly.ConnectorClient` — the spike live-verifies the
shipping adapter code, not just raw SDK calls.

- Token mint (`CreateToken`, org `personal`): 0.6–2.2s.
- `CreateSprite`: 1.4–1.5s; initial status `cold`; labels persisted.
- Fly-issued URL: `https://<name>-<region>.sprites.app` — the requested name
  (user slug) is embedded verbatim. Stored from the API response, never built.
- URL auth defaults: `auth="sprite"`, private `admins`. Anonymous GET returns
  the Fly login page (HTTP 200, ~4KB HTML). `Authorization: Bearer <token>`
  returns the service body — authorized URL→service routing proven live.
- Sprite machine: Linux x86_64, Ubuntu 26.04, non-root uid 1001. No
  python/node/busybox on a fresh sprite.
- Deploy vector proven: 5.8MB static Go binary via `WriteFile` + `chmod` +
  `CreateService` with `httpPort` reached `running` in ~6s and served HTTP 200
  through the Fly URL in ~1s from cold.
- Status transitions observed: `cold` → `running`.
- `ListConnections`: 2 org connections — **box:1, neon:1**. Box (the demo
  anchor) and Neon are both configured in this organization.
- Cleanup: service + sprite deleted; get-after-delete errors; final
  `studio-` prefix listing empty. Zero leftovers, zero orphan Fly resources.

Not measured (bounded out): natural sleep→wake latency distribution (needs
extended idle observation; P5 rehearsal item), org Sprite count/running
limits (read `OrgInfo` limits before fleet use in P4).

## Postgres compatibility (MPG Developer candidacy)

- Framework requirements (verified in code): pgx direct connections,
  LISTEN/NOTIFY (DBOS `sysdb/notifier.go`), advisory locks, RLS, zero
  extensions, golang-migrate migrations.
- MPG Developer facts (official docs, checked 2026-11-24): direct connections
  (no PgBouncer), ≤20 connections, PostgreSQL 16/17, $4/mo inside the free
  allowance, 2-node HA, daily backups kept 1 day, auto-pause after 15 idle
  minutes with 15–25s wake (client connect timeout ≥30s), 1–5GB storage,
  maintenance anytime (clients must reconnect), same-region traffic free.
- Empirical version check: full Miniapp suite (22 ok, exit 0), both restart
  tests executed and passing, forced (uncached) race subset, and the full
  Studio suite all green on PostgreSQL 17.12 (local digest-pinned container).
  The PG16/17-vs-19 skew is retired as a compatibility risk for PG17.
- MPG cluster provisioning was NOT performed (persistent paid infrastructure
  needs explicit owner action); deploy-time smoke test is specified in the
  decision record.
