# Prompt30 implementation evidence (S0–S6, 2026-11-27)

**Scope delivered:** bounded migrate driver adapter, wiring, instrumentation, fault-injection suite, gates.
**Not yet run:** S0 owner evidence (DBeaver attribution + hang-phase capture), S7 live Neon proof (separately gated).

## Files changed

| File(s) | Change |
|---|---|
| `migratedb/` (new: config, driver, ops, events + pgwire stub, 12 tests) | Bounded `database.Driver`: dedicated conn, per-op child deadlines, try-loop advisory lock, pinned TLS via `stdlib.GetConnector`, poisoning, slog instrumentation, capture summaries |
| `store/migrate.go` | `Migrate`/`migrateTo` take ctx + policy; construct adapter; inject via `NewWithInstance`; phase trail on errors; `withMigrationsTable` retired |
| `dbenv/provision.go` | `ApplyFrameworkBaseline` takes policy; adapter + `NewWithDatabaseInstance`; pgx-scheme stock path retired |
| `testdb/testdb.go` | `Policy(t)` + `FrameworkBaseline(t, dsn)` helpers; provision path off `testkit.MigrateUp` |
| 6 test call sites | `testkit.MigrateUp` → `testdb.FrameworkBaseline`; `Migrate`/`migrateTo` signature updates |
| `cmd/studio/{dev,migrate}.go` | Policy threading (`--metadata-file` on `migrate` verb too) |
| `store/provision_guard_test.go` | Guard carve-out for `store/migrate.go` + symbol restriction (no test-credential surface); role-literal twin pins (cycle fix) |
| `dbenv/dbenv_test.go` | Role literal pinned locally (test-binary import cycle fix) |
| `store/migrate_wiring_test.go` | Policy-before-migrate wiring proof (replaces retired scheme test) |

## Approach refinement (disclosed, within approved architecture)

The plan encoded the overall deadline as a timestamp; implementation passes the caller's context into
`migratedb.Open` as the BASE context instead. Every operation derives child deadlines from it, so an overall
deadline (dev init's 5m) genuinely bounds migration work and cancellation cooperates at operation boundaries —
strictly stronger than encoding, same structure, no new surface. Close uses its own short deadline (never the
possibly-cancelled base), per safeguard 2.

## Directive P7 ten-item mapping (honest split)

Fixture-proven (pgwire stub, stdlib only, race-clean): 1 blocked-op bounds, 2 SetVersion-BeginTx bound
(deterministic hang repro), 3 post-command silence, 4 deadline return, 5 conn invalidation + no-reuse,
8 goroutine/resource hygiene, 10 all-phases-bounded (Open/Ping/Lock/Version/Run/SetVersion/Drop/Close each
covered). Neon-proven (require real server semantics): 6 no-clean-on-failure under real dirty protocol,
7 advisory mutual exclusion under contention, 9 subsequent healthy migration. Fake-driver tests: none shipped —
propagation is proven through the real adapter against the stub instead (stronger than fakes).

## Counts (observed)

`go build ./...` OK · `go vet ./...` OK · `gofmt` clean · `migratedb` race-clean.
Full suite: **423 pass, 305 skip, 0 fail**. DB-backed tests skip honestly without binding.

## Incidental findings (fixed, with tests)

- `db.Close()` stalls on connections still checked out: `Open` failure paths now release the reservation first.
- pgx close-drain runs its 15s timer against never-responding servers: stub tracks and drops server conns on
  close (production servers FIN instantly; tests stay subsecond instead of 78s).
- `sql.ErrNoRows` explicitly exempted from connection poisoning (normal control flow, e.g. fresh version table).
- pgx returns `driver.ErrBadConn` (not DeadlineExceeded) on cancelled queries: accepted as a bounded outcome;
  classification distinguishes deadline/canceled/connfailed/error.

## Open items for owner-terminal

- **S0 evidence:** (a) DBeaver attribution — ANSWERED 2026-11-27: `default` (`neondb`); empty tables expected,
  not a persistence failure. Run-pattern DBs remain the authoritative migration target. (b) if the hang
  reproduces: activity/lock counts + version/dirty during the stall.
- **S7 live proof** (separate authorization): bootstrap → full Up → suite → Down to 15 → RLS checks → Up replay
  to 25 → repeat; record versions/dirty/counts/timings; review phase-trail output for full coverage.

## operator_upsert privilege hole + cleanup fix (2026-11-27)

Sized run surfaced 3 real store failures: two tests (RoleAttributes, PrivilegeParity) forbid app EXECUTE on
`studio.operator_upsert`, but 000002 grants it — a live privilege-escalation hole (SECURITY DEFINER upserting
operator rows incl. admin flag, zero app callers; sole caller is the owner-side provisioner). Fix: forward
migration 000026 revokes app EXECUTE and grants the migrating owner instead (dynamic session_user; history
immutable since 000002 applied live). My set_repair cleanup missed `DROP OWNED` (probe's schema grant blocked
DROP ROLE) — fixed. Suite: 425/306/0. Tools `TestAnswerAnswerConcurrency` (9m6s mid-DB-read, no sleeps in path)
remains the one unadjudicated stall suspect: targeted re-run with Neon-console lock capture discriminates
lock-wait cycle from extreme slowness.

## Full-suite timeouts: aggregate slowness, not hangs (owner-terminal 2026-11-27)

First full live run: 94 pass / 0 fail / 11 skip; three packages (store, server, tools) hit the 10m go-test
default; five packages green (33–252s). Panic-dump forensics on all three: every alarm fired on healthy
mid-flight work (4s into startup; 27s into pool acquire under load; slow heavy test), with steady completions
before each (slowest singles: 161s, 155s, 176s — full provision+migrate cycles over cloud latency per test).
Zero AfterRelease/Close/Acquire-stall signatures anywhere: the pool fix holds. Conclusion: per-package wall
budgets, sized for local runs, undersize live cloud runs. Remedy: `-timeout 30m` (measured: slowest single
test 176s; slowest package pace projects 15–25m). A timeout at 30m would constitute stall evidence; until
then it is capacity planning. Package parallelism (`-p`) deliberately unchanged: serial discipline stays
while attributing results; revisit only for wall-time optimization afterward.

## Pool-cleanup hang: diagnosed + fixed (owner-terminal 2026-11-27)

Full-suite run produced two 10m panics with a NEW signature, after all migrate legs passed: the test function
completed (`v25→v15→v25` asserted) but teardown hung — `admin.Close()` waiting on a connection parked in
`TenantResetHook`'s unbounded `RESET`, plus pool `Acquire` starvation in the server package (same family:
connections trapped in never-completing releases drain pools). Root cause class matches the incident theme:
unbounded network waits, this time in pool lifecycle rather than migrate. Fix: 10s deadline on the hook with
destroy-on-failure (the hook's own documented contract) + secret-free warn log; 60s bound on scratch DROP;
missing `owner.Close()` added to the decisive test (registered to run before the drop). Other tests already
close pools correctly. Suite: 425 pass / 306 skip / 0 fail. Live re-run adjudicates.

## Decisive test PASS (owner-terminal 2026-11-27, live Neon)

`TestMigrationDownUpBounded` (plans/bug_plan.md) PASSED in 195.76s: full Up → v25 clean → Down to v15 clean →
Up replay → v25 clean, zero failures/panics, version/dirty asserted at each leg. The exact incident sequence
that stalled at SetVersion(17) now completes with every operation bounded and phase-trailed.
**Incident mechanism CLOSED.** The earlier 10-minute hang is attributable to the unbounded stock driver path
(now fully retired: all entry points use the adapter); no hang signature has appeared in any bounded run.

## S7 round 1 findings (owner-terminal 2026-11-27, two runs, identical)

The instrumentation worked as designed — the phase trail
`open✓→lock✓→ensure_table✓→unlock✓→lock✓→version✓→set_version✓→run✗(error)→unlock✗(error)` pinned the failure
to the framework v1 body (42601 `syntax error at or near "mandatory"`), not to any timeout or stall.

- **F3 — client-side statement splitting diverged from stock.** The adapter used migrate's `multistmt` splitter,
  but the stock default (`x-multi-statement` off) sends the whole body as one Exec for server-side parsing. The
  splitter mangled a real framework migration. Fix: `Run` sends bytes untouched (server splits); pinned by
  `TestReadWholeBodyPinsNoSplitting`. Deviation from the plan's S2 line disclosed here: multistmt reuse dropped
  in favor of exact stock-default parity.
- **F4 — poisoning saw through nothing.** `database.Error` lacks `Unwrap`, so wrapped SQL errors hit the
  conservative-unknown branch and poisoned the connection (the spurious `unlock✗`). Fix: explicit `OrigErr`
  recursion; wrapped 42601/ErrNoRows now classify clean, 08xxx still poisons. Covered by extended table test.
- **F5 — dirty/clean marks now distinct phases** (`mark_dirty`/`mark_clean`) so trails disambiguate version writes.
- Abandoned dirty run DBs from both failed runs need no cleanup (fresh DB per run; reaper covers orphans).
- Suite after fixes: 424 pass / 305 skip / 0 fail; adapter race-clean.
