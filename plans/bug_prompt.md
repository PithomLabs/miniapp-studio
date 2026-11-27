# Architect diagnostic + plan request: migration hang (TestRLSEnable000016, Neon PG18)

## 1. Incident

Live owner-terminal run (`task dev:bootstrap` → `dev test --db-only`) panicked on Go's default 10m test timeout.
`TestRLSEnable000016` hung 2m40s+ inside `store.Migrate` → `migrate.Up()`. Separately, the owner reports DBeaver
shows "no tables" — UNSCOPED: the inspected database is not yet established (see §4, item 0).

No credentials, DSNs, hostnames, or IDs may appear in any reply. Evidence moves through files and booleans only.

## 2. Verified facts (read from the panic stack + module sources, not inferred)

- **Hang point:** `pgx/v5.(*Postgres).SetVersion(0x11=17, dirty=false)` → `BeginTx` → TLS `conn.Read` blocks forever.
  Per migrate v4.20.1 `runMigrations` source (dirty-mark → run body → clean-mark), v17's body RETURNED; the hang is
  the clean-mark's fresh `BEGIN` receiving no server reply.
- **v17 body is trivial:** two `ALTER TABLE studio.builds ADD COLUMN` (TEXT DEFAULT '', INTEGER) — metadata-only.
- **Test flow** (`store/migrate_down_test.go:90`, the SECOND `Migrate` call): full Up v1→v25 ✓ → Down to v15 →
  flag checks → Up replay v16→25 → hang at v17's clean-mark. Fresh scratch run-pattern database; branch roles persist.
- **`SetVersion` opens a NEW transaction** via `database/sql` stdlib over `context.Background()` — no deadline.
  The whole migrate path (Lock, Run, SetVersion) uses `Background`; `dev test` passes no `-timeout`, so the 10m
  go-test default was the ONLY backstop in the system.
- **Advisory locking exists** (`SELECT pg_advisory_lock($1)`, "will wait indefinitely", keyed by db+schema+table,
  held for the whole run) — but the hang stack is NOT in `Lock()`; the lock was acquired. Ruled out as the direct
  cause, subject to your confirmation.
- **Direct endpoint, not pooler** (binding metadata inspected structurally: no `pooler` hostname). PgBouncer
  transaction-pooling hazards do not apply.
- **Prefetch pile-up is secondary:** ~16 goroutines parked in migrate `readUp`/`Buffer` pipe-writes are the
  producer side backing up behind the stalled consumer. Two `readUp` roots exist (one ~5 min old); read as
  backpressure (+ possible benign prefetcher leak from the earlier completed Up), not causal — challenge if wrong.
- **A plain `BEGIN` waits on no locks.** A live connection receiving no reply to it means transport/server-event-loop
  stall, backend stuck pre-response, or client-side connection-state desync — not a lock conflict (which would bite
  at `TRUNCATE`, inside the txn, and TRUNCATE was never reached).

## 3. Ruled out (one line each)

- Advisory-lock contention: hang is post-acquisition, in SetVersion's BeginTx.
- Pooler semantics: direct endpoint verified.
- v17 body content: two instant DDL statements that already returned.
-- `dev bind`/TLS/endpoint policy: live fingerprint + full v1→v25 Up succeeded minutes earlier on the same branch.
- Slow query needing patience: 2m40s+ on a bare BEGIN is not slowness.

## 4. Ranked hypotheses + discriminating evidence (all secret-free)

- **H1 — dead/half-open connection to Neon (plausibly gateway/NAT silently dropping):** reads block forever with
  zero timeouts configured anywhere. Discriminator: during a reproduced hang, `SELECT count(*), state FROM
  pg_stat_activity WHERE backend_type='client backend' GROUP BY state` (counts only) + whether a fresh trivial
  query on a NEW connection answers while the old one hangs.
- **H2 — backend stuck from the preceding body (protocol/desync aftermath):** v17's body returned to the client
  but left the backend unresponsive. Discriminator: `pg_stat_activity` row for the stuck backend (state, wait_event,
  backend_xid/xmin) at hang time; exact migrate version/dirty from the version table (integers only).
- **H3 — a second migrator holds the database:** a leaked earlier `Migrate` instance (test `defer m.Close()` never
  reached?) keeping a transaction open. Discriminator: count of backends + open transactions + the advisory-lock
  holders (`pg_locks` locktype-only listing, no values beyond lock types/counts) during the hang.
- **H0 (scoping, before all else):** which database did DBeaver inspect — the default/bootstrap one (expected EMPTY:
  schema lives only in run-pattern databases) or the `studio_test_run_*` run database? Answerable as "default" or
  "run-pattern" with zero pasted values. If the run DB is truly empty despite versions 1–16 clean-marking, that
  reframes everything toward persistence/visibility rather than a v17-local stall.

## 5. What we need from you

1. A diagnosis: which hypothesis (or new one) best fits, with the mechanism spelled out to the level of "which
   component waits for what, and why no error ever surfaces".
2. An implementation plan for (a) the minimal fix, (b) mandatory deadline/timeout instrumentation on the migrate
   path (exact layer: our `store.Migrate` wrapper vs DSN `statement_timeout`/`connect_timeout` vs migrate config —
   migrate's API takes no context, so say where the backstop lives), and (c) a regression that reproduces the
   stall mechanism without depending on a 10-minute live hang.
3. The single cheapest next owner-terminal command (secret-free output only) that would confirm or kill your
   leading hypothesis, with the exact expected-vs-actual decision rule.

## 6. Standing constraints (binding on any proposal)

- No self-hosted PostgreSQL anywhere (no local/Docker/Testcontainers PG, even as a test fixture).
- No credentials, DSNs, hostnames, or IDs in chat, logs, or pasted output — evidence returns through
  gitignored JSON/boolean files the agent reads from the repo.
- No weakening: no skipped failure modes, no timeout that masks the stall without diagnosing it, no
  advisory-lock removal without a replacement mutual-exclusion story.
- F1/F2/F2b history (driver scheme, OWNER TO membership + CREATE) is settled and live-proven; do not relitigate.
