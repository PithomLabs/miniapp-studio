## Goal

Implement `plans/prompt30.md`: restore deterministic, repeatable Neon migration execution by replacing the
unbounded stock migrate driver path with a small maintained internal adaptation — universal bounded execution,
preserved dirty/advisory-lock semantics, secret-free instrumentation, fault-injection regression without
self-hosted PostgreSQL, and repeatable live Neon proof. No credentials, DSNs, hostnames, or IDs in chat, logs,
or pasted output at any stage; evidence returns through gitignored files.

## Success Criteria

- Every migration database operation (connect, ping, lock, version read, dirty mark, body, clean mark,
  commit/rollback, unlock, close) executes under an explicit deadline; nothing on the migrate path can block
  indefinitely — verified by fault-injection tests, not by code inspection alone.
- The original `SetVersion`-hang failure mode is reproduced deterministically in-repo and then proven bounded.
- Advisory mutual exclusion still holds (second migrator blocks-then-proceeds or times out per configuration,
  never runs concurrently); dirty-marker protocol unchanged (ambiguous failure stays visibly dirty, never
  auto-cleaned).
- Migration traffic carries the same verify-full TLS pinning as the rest of the contract (no DSN-string-only trust).
- RLS, tenant-isolation, privilege-parity, and no-membership suites stay green; clean-bootstrap Up/Down/Up
  repeats on the same Neon branch; fingerprint distinguishes local fault simulation from live Neon verification.
- Incident scope established first: the DBeaver observation attributed to a specific database ("default" vs
  "run-pattern", zero pasted values), hang-phase timings captured.

## Approach

The stock driver's `database.Driver` interface (`Open/Close/Lock/Unlock/Run/SetVersion/Version/Drop`) takes no
context anywhere, and its implementation uses `context.Background()` for every database call except
statement-timeout-wrapped `Run` bodies (verified in migrate v4.20.1 source). Timeouts therefore cannot flow
through the interface — they must live inside an adapted driver, configured at construction.

Key decisions:

- **D1 — Adapter package, injected, not forked.** New package `migratedb` implementing `database.Driver`,
  constructed in Go with deadlines, event sink, and pinned TLS config, injected via
  `migrate.NewWithInstance` (verified to exist). Reuses migrate's exported `multistmt.Parse` for statement
  splitting. Rejected: forking pgx.go wholesale (maintenance burden); URL-param configuration (cannot carry
  sinks or TLS configs); goroutine-abandon wrapper (explicitly forbidden — hides stalls instead of bounding them).
- **D2 — Deadline model.** Per-operation child timeouts derived from `min(overall ctx deadline if present,
  configured op caps)`; hardened module defaults when the caller passes a deadline-less context. Cancellation
  cooperates at operation boundaries; a timed-out or suspect connection is invalidated and never reused.
  Explicitly: this bounds every operation; it does not promise instant cancel of in-flight server work.
- **D3 — Dedicated connection.** The adapter reserves one `*sql.Conn`, health-checks it with bounded
  `PingContext`, runs the whole migration lifecycle on it (advisory lock is session-held by design), and
  invalidates on protocol/timeout failure. `application_name` identifies migration sessions in
  `pg_stat_activity`. Server-side `statement_timeout`/`lock_timeout` via connection options plus
  connect-timeout/keepalives are complementary, never the primary backstop.
- **D4 — TLS unification.** The adapter builds `database/sql` over `stdlib.GetConnector` with the pinned
  `pgx.ConnConfig` from `dbenv.PoolConfig` (verified to exist) — migrate traffic gets identical verify-full
  pinning (system roots, ServerName, TLS 1.2+). DSN strings stay policy-constrained at call sites as today.
- **D5 — All three migrate entry points move together.** `store.migrateTo`, `dbenv.ApplyFrameworkBaseline`,
  and the six `testkit.MigrateUp` test call sites across five areas (server, down/round-trip ×2, f3, testdb, e2e provision) switch
  to the injected adapter; testdb/tests move from `testkit.MigrateUp` to `dbenv.ApplyFrameworkBaseline`
  (which already mirrors it). `testkit.NewDB` stays untouched; the framework itself is unchanged (no §14 event).
  `withMigrationsTable`/`pgx5URL` scheme mapping retires where the adapter takes over (verify each survivor).
- **D6 — Signature change, mechanically applied.** `store.Migrate`/`migrateTo`/`ApplyFrameworkBaseline` take the
  `dbenv.EndpointPolicy` (cycle-clean: store has zero dbenv imports today and dbenv deliberately never imports
  store). Test call sites thread deadlines where the test owns them; adapter defaults cover the rest.
- **D7 — Honest fault-injection split.** Fixture-provable (stdlib-only pgwire-scripting stub: handshake, then
  blackhole/scripted responses — no PG server): stalled-op bounds, `SetVersion`-BeginTx bound, post-command
  silence, conn invalidation, no-clean-on-failure, no-goroutine-leak, all-phases-bounded. Neon-proven:
  advisory mutual exclusion under contention, healthy-migration success, full Up/Down/Up. Fake-driver tests
  cover propagation only and are labeled as such — never presented as network proof.
- **D8 — Instrumentation via stdlib slog** (repo convention), secret-free event schema per the directive
  (version, phase, elapsed, deadline, outcome, classification, reusability), synchronous to stderr like the
  server — no networked sink, no blocking path beyond local stderr.
- **D9 — Reconciliation reuses fingerprint** (`Applied`/`Dirty` already reported): ambiguous failure →
  re-read state on a fresh verified connection → operator-only explicit force step, documented. No new
  surface, no auto-force, no auto-drop outside run-pattern authorization.

## Steps

- **S0 — Incident scoping (P1, owner-terminal first).** Establish which database DBeaver inspected
  ("default" = expected-empty, tables live only in run DBs; "run-pattern" = reframes the incident).
  During a reproduced hang capture (secret-free): `pg_stat_activity` state/wait counts, `pg_locks` lock-type
  listing, version/dirty via fingerprint, fresh-connection trivial query, phase timings. Do not modify
  migration SQL for a transport-level hang; do not treat H1 as proven until this lands.
- **S1 — Adapter skeleton + construction.** `migratedb` package: config (deadlines, sink), `Open` acquiring +
  `PingContext`-checking the dedicated conn, `Close` with bounded unlock attempt and invalidation semantics.
  Unit tests with the pgwire stub: connect/hang/close paths.
- **S2 — Operation deadlines.** `Lock`/`Unlock`/`Run`/`SetVersion`/`Version`/`Drop` each on explicit child
  contexts; multistmt reuse for `Run`; `x-multi-statement` behavior preserved; advisory-lock SQL unchanged.
  Verify migrate's `LockTimeout` default interplay (read, don't assume) and keep framework semantics.
- **S3 — Wiring + TLS.** Switch the three entry points to `NewWithInstance` with the constructed adapter;
  `stdlib.GetConnector(pinnedConnConfig)` inside; thread `EndpointPolicy` through the signature changes;
  move test call sites off `testkit.MigrateUp`. Retire/verify each `pgx5://` mapping survivor.
- **S4 — Instrumentation + diagnostics.** slog event emission per phase per the directive schema; keep the
  existing `diagnoseMigrateErr` (version/hash/SQLSTATE/position) and feed it adapter phase context so every
  timeout names its exact operation.
- **S5 — Fault-injection suite.** pgwire stub (handshake → scripted silence/slowness), the ten directive items
  split per D7, goroutine-leak assertions, propagation unit tests labeled as such. No new dependencies.
- **S6 — Repo gates + docs.** `gofmt`/`vet`/full unit suite with counts; dev-contract + evidence updates;
  reconciliation procedure documented (fingerprint read → operator decision, no auto-force).
- **S7 — Live Neon proof (owner-terminal, per-op authorization).** Clean bootstrap → full Up → DB suite →
  Down to 15 → RLS checks → Up replay to 25 → repeat whole sequence on the same branch with fresh run DBs.
  Record versions, dirty states, counts, timings, lifecycle evidence. Instrumentation output reviewed for
  phase coverage (every listed operation emitted events).

## Validation Plan

- S0: scoping answers recorded (database attribution + hang-phase capture or documented inability); plan
  proceeds on evidence, not on H1-by-default.
- S1–S2: `go test ./migratedb/` — blackhole-connect timeout, per-op deadline tests with tight bounds,
  unlock-on-close, invalidation (poisoned conn never reused); race detector on the package.
- S3: full `go build ./...`, `go vet ./...`; grep-proof that no call site constructs the stock driver path
  for Studio/framework chains; TLS-pinning test asserting the adapter's connector config equals
  `dbenv` pinning.
- S4: simulated-failure tests asserting event completeness (one event per phase incl. failure classification
  and reusability flag) and secret-free output scan (no DSN/host/credential substrings in any event).
- S5: the ten items each mapped to fixture or Neon proof in the test file header; suite run with counts;
  leak assertions green.
- S7: two consecutive full sequences green on Neon; RLS/parity/no-membership suites green; evidence file
  separates simulated vs live proofs. Highest-risk validation: the live Up/Down/Up replay — the only place
  real advisory, dirty-state, and timing behavior exists.

## Risks / Open Questions

- If S0 shows the run database genuinely empty despite clean-marked versions, the incident reframes from
  "v17-local stall" to persistence/visibility — S1+ still applies (bounded execution is required regardless),
  but the fix target widens; stop and re-scope before S3.
- `migrate` internals (`NewWithInstance` behavior with our source, `LockTimeout` default, prefetch
  backpressure under slow consumers) are read-from-source assumptions — each gets a pinning test or a
  documented deviation during S1.
- Test-harness `context.Background()` ubiquity means most deadlines come from adapter defaults initially;
  threading real deadlines into test setup is follow-up work, explicitly not a gate.
- Non-goals: `testdb` provisioning-path timeouts (adjacent, separate slice); framework (`miniapp`) code
  changes; `go test -timeout` increases; blind retries; any self-hosted PostgreSQL including as fixtures.
- Open questions: none blocking. The DBeaver attribution and hang-phase capture are S0 evidence steps with
  fail-closed handling, not plan uncertainties.
