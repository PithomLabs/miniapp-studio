# ADR-002: Platform database — MPG Developer (PG17)

**Status:** Decided 2026-11-24, conditional on deploy-time smoke test.
**Decision:** Fly Managed Postgres, Developer plan, PostgreSQL 17, same region
as the Studio app. **Fallback:** Neon, on the triggers below.
**Basis:** code-verified requirements + official MPG docs (checked 2026-11-24)
+ empirical PG17 suite runs; see `docs/p3-evidence.md`.

## Requirements (verified in code)

pgx direct connections; LISTEN/NOTIFY (DBOS `sysdb/notifier.go`); advisory
locks; RLS; zero extensions; golang-migrate migrations. No exotic features.

## Why MPG Developer fits

- Direct connections, no PgBouncer: LISTEN/NOTIFY and advisory locks work.
- PG 16/17: full Miniapp suite (22 ok), both restart tests executed, forced
  race subset, and the full Studio suite pass on PG17.12 — version skew retired.
- $4/mo inside the free allowance; 2-node HA; daily backups; 1–5GB storage is
  ample for MVP scale; same-region traffic free.
- Fly-native control plane (Machines API) matches the product direction.

## Operational consequences (binding)

- Connection budget: ≤20 across ALL processes (Studio + workers). Set explicit
  `MaxConns` values summing under the cap; document the budget in P4.
- Connect timeout ≥30s (auto-pause wake is 15–25s); pools must reconnect on
  drops (maintenance can occur anytime; autostop and auto-pause wake latencies
  stack — measure combined cold start in P5).
- Backups are daily/1-day retention: keep an independent `pg_dump` of anything
  irreplaceable (demo fixtures, golden-path data).
- Deploy-time smoke test (P4, owner-provisioned cluster): provision → connect
  → migrate → run the Studio DB suites → record SHA, region, timings.

## Neon fallback triggers

MPG region unavailable to the org; the 20-connection ceiling binds real
traffic; auto-pause latency unacceptable after measurement; any PG17
incompatibility found later. Neon keeps pooled+direct endpoints and stays a
compatible alternative; switching costs are one DSN + rerun of the gates.
