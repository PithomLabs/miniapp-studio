# Plan to implement plans/prompt13.md — CI PostgreSQL migration failure

## Goal

Establish the root cause of the GitHub Actions step-8 failure
(`policy "tenant_isolation" ... already exists`, SQLSTATE 42710, then
`Dirty database version 1`), fix the violated migration/test-lifecycle
invariant at the correct ownership boundary, and prove the correction
with the adversarial regression matrix — without weakening RLS,
reordering unrelated tests, or committing anything.

## Success Criteria

- The exact replay mechanism is identified with source + catalog evidence
  (which runner, which database/schema/role, why tracking failed).
- `go test -p 1 -count=1 ./...` passes repeatedly on PG19 without
  order- or residue-dependence; the regression matrix from prompt13 §7
  is implemented and green.
- RLS posture byte-identical in effect: same policies, still enforced,
  cross-tenant denial and fail-closed missing-context behavior proven.
- The `public.schema_migrations`-missing and `role "root"` log lines
  are each explained (root cause vs noise) with evidence.
- CI workflow corrected (pinned PG19 image, real readiness gate,
  version check, correct DSNs, no secret emission) and the actual
  workflow passes.
- `docs/ci-postgres-migration-failure-report.md` exists with all 14
  prompt13 §12 items. No commits or pushes.

## Approach

Investigate before changing anything, in this order: reproduce →
catalog forensics → single-root-cause fix at the owning boundary →
regression matrix → CI workflow → evidence report. The plan assumes
nothing about the cause; it sequences the proofs that discriminate
the hypotheses.

### Grounded starting facts (verified this session, not assumed)

- CI (`.github/workflows/ci.yml`): one job, one PG service
  (`POSTGRES_USER: studio`, DB `studio_test`, port 5432), single
  shared `TEST_DATABASE_URL`, `go test -p 1 ./...`. Formatting, vet,
  and build pass; only the test step fails.
- The colliding policy `tenant_isolation ON app.invocations` is
  created by **framework** migration `000001_framework_init.up.sql`
  (pinned miniapp module). Studio migrations create same-named
  policies only on `studio.*` tables. "Dirty version 1" therefore
  implicates the framework migrator replaying v1, not Studio DDL.
- Studio tracks in `studio_schema_migrations`
  (`store/migrate.go:21`, via `x-migrations-table`); the framework
  `testkit.MigrateUp` uses the golang-migrate default table name,
  unqualified. Every DB test helper calls `testkit.MigrateUp` then
  `store.Migrate` against the one shared database
  (`cmd/studio/studio_test.go:31-44`, `tools/tools_test.go:55-68`,
  `server/server_test.go:63-76`); no test uses `t.Parallel`.
- Structural collision candidate (prime hypothesis, to be proven):
  Studio migration `000002` runs `CREATE SCHEMA IF NOT EXISTS
  studio`, while the CI database role is also named `studio`. Under
  PostgreSQL's default `search_path ("$user", public)`, an
  unqualified `schema_migrations` resolves to `public` before Studio
  migrations run and to `studio` after — so a later `MigrateUp`
  could see "no version table" and replay framework v1. Nothing in
  the repo pins `search_path` for migration connections.
- `FATAL: role "root" does not exist` is most plausibly healthcheck
  noise (`pg_isready` with no user inside the container), since no
  repo code connects as root — but the implementer must confirm from
  timestamps, not assume.
- The literal string "migration failed" and the cited
  `gateway_test.go:153` do not match this tree — CI ran a nearby but
  non-identical revision. Reconcile line numbers against the CI
  commit before citing them.
- Local PG19 (`127.0.0.1:5434`) is currently unreachable from this
  session (connection refused). DB-backed steps are gated on it per
  prompt13 §8; no substitute version is acceptable.

## Steps

### Phase 0 — Reproduce and capture the forensic baseline (no code changes)

1. Start/access local PG19 Docker; verify endpoint + `SHOW
   server_version` / `SELECT version()` → 19.x. If unavailable: stop
   DB work, record the exact blocker, continue with static analysis
   only.
2. On a scratch database, run the smallest failing sequence first
   (single `cmd/studio` DB test, then the package), capturing PG logs.
3. Catalog forensics after the failure. Trace BOTH collisions
   independently — `tenant_isolation` on `studio.apps` (Studio DDL)
   and on `app.invocations` (framework DDL) — one shared mechanism
   may explain both, but that must be demonstrated, not assumed.
   Record: `current_user`, `session_user`, `current_schema()`;
   `SHOW search_path` and `current_schemas(true)`; ALL
   `schema_migrations` and `studio_schema_migrations` tables with
   actual schemas and version/dirty values; `to_regclass()`
   resolution for unqualified and schema-qualified migration-table
   names; and the resolved relation OIDs BEFORE and AFTER
   `CREATE SCHEMA studio` using the CI-equivalent `studio` role
   (the before/after comparison proves how the lookup changed; a
   post-failure snapshot alone does not). These checks confirm or
   kill the prime hypothesis per runner.
4. Pull the actual CI log lines around step 8 (timestamps!) to order:
   first 42710, dirty-version errors, `schema_migrations`-missing,
   `role "root"`. Establish the causal chain, including which test
   binary/package ran first.

### Phase 1 — Root-cause decision (the plan's main fork)

5. From Phase 0 evidence, select exactly one mechanism:
   - **(a) Search-path shadowing** (prime): version-table lookups
     resolve to different schemas before/after `CREATE SCHEMA
     studio`. Fix lives in studio-owned code/config only —
     `testkit.MigrateUp` is pinned-framework code and must not be
     edited; candidates are schema-qualified tracking tables,
     explicit `search_path` on migration DSNs/connections, or a
     role-level `SET search_path` — plus a regression test that
     fails on the old behavior.
   - **(b) Helper/lifecycle defect**: some helper replays framework
     migrations outside version tracking (prove with the execution
     map: every `MigrateUp`/`Migrate`/`migrateTo` call site per
     package run). Fix at the helper/ownership boundary.
   - **(c) Dirty-state reuse**: first failure is elsewhere; later
     tests reuse the dirty DB. Fix = fail-fast isolation (fresh
     scratch DB per package or explicit dirty detection that stops
     the run), never silent force/cleanup.
6. Explain the two secondary log lines from evidence: the
   `public.schema_migrations`-missing lookup (candidate: Studio's
   explicit `public.schema_migrations` read in
   `store/migrate.go:93-101` racing framework bootstrap, or a
   shadow-table artifact) and `role "root"` (candidate: container
   healthcheck without `PGUSER`). Each gets root-cause-or-noise with
   timestamps, not a guess.
7. If evidence forces a framework-side change, stop and route it
   through the established Miniapp approval process instead of
   working around it in Studio.

### Phase 2 — Minimal durable correction

8. Implement the Phase 1 fix at the owning boundary only:
   framework-owned version tracking vs Studio-owned helpers/config.
   Explicitly forbidden as the fix: conditional `CREATE POLICY`,
   dropping policies, touching RLS/force/predicates, force-setting
   versions, or deleting schema objects to clear dirty state.
9. Harden the immediate adjacents only if the root cause implicates
   them: version-table schema qualification for BOTH migrators,
   deterministic helper ordering (framework-then-Studio, exactly
   once per database), and dirty-state fail-fast before any test
   executes.

### Phase 3 — Regression matrix (prompt13 §7)

10. Implement each row as an automated test using isolated scratch
    databases (follow the existing `scratchDB` pattern in
    `store/f3_test.go`): fresh migrate; repeated invocation no-op;
    framework→Studio ordering; re-init; pre-initialized DB;
    concurrent migrators (two processes/binaries, not just
    goroutines, per the prompt's concurrency rule); mid-migration
    failure → explicit dirty + no silent reuse; exact-policy
    assertion; cross-tenant denial; missing-context fail-closed;
    repeated full-suite runs. Include one test reproducing the exact
    CI collision (unqualified tracking + studio schema present)
    that fails pre-fix and passes post-fix.
11. Do not delete, skip, weaken, or rename any failing test to get
    green. Pre-existing skips stay as-is unless the root cause owns
    them.

### Phase 4 — CI workflow correction

12. Pin/verify the PG image is PG19 (resolve the digest to a version;
    record it), add `PGUSER`/user-correct health check plus an
    explicit `SELECT version()` gate step before tests, verify
    `TEST_DATABASE_URL`/`TEST_DBOS_DATABASE_URL` wiring and that no
    step echoes credentials. No sleep-based readiness, no second
    database service.
13. Verify the workflow itself, not just local runs: the fix must be
    shown to address CI conditions (fresh container, role named
    `studio`, shared DB) — e.g., a local reproduction with the same
    role/DB names, plus reasoning recorded for what only CI can prove.

### Phase 5 — Evidence report

14. Write `docs/ci-postgres-migration-failure-report.md` with all 14
    prompt13 §12 items, including the migration execution map,
    red/green command transcripts, PG version proof, and remaining
    limitations. Update existing migration docs only where the
    correction changes stated behavior.

## Validation Plan

- Static: `gofmt -l .` (empty), `go vet ./...`, `go build ./...`.
- Migration tests standalone first (new + `store` package), then
  `go test -p 1 -count=1 ./...` twice in a row on PG19; both runs
  must pass on correctly isolated databases.
- `go test -race -p 1 -count=1 ./...` (or named affected packages
  with explicit scope justification — never claim full-race on a
  subset).
- RLS proof: exact-policy catalog assertions + cross-tenant denial +
  missing-context fail-closed tests, all green.
- Highest-risk validation: the ordered-replay reproduction (fresh DB
  → framework migrate → Studio migrate → framework migrate again
  must be a clean no-op, and the pre-fix tree must fail it). If PG19
  is unavailable, mark all DB gates blocked with the exact error and
  stop — do not substitute versions.
- No commits or pushes at any point; final tree diff reviewed for
  scope (migration/test-lifecycle + CI YAML + report only).

## Risks / Open Questions

- **Prime hypothesis could be wrong.** If catalog forensics show a
  single `public.schema_migrations` with correct versions, the
  replay comes from elsewhere (helper ordering, a second runner, or
  dirty reuse) — Phase 1 is designed to discriminate before any fix.
  Assumption, not fact, until Phase 0 proves it.
- **Framework-code boundary.** If the defect lives inside pinned
  `testkit`/migrator behavior, the fix needs the Miniapp approval
  process, which may exceed this task's scope — flagged early in
  Phase 1 rather than worked around.
- **CI-only factors.** Container-local DNS (`localhost:5432` inside
  the job), image digest contents, and healthcheck behavior can only
  be fully proven by a real workflow run; local reproduction uses
  the same role/DB names to close the gap as far as possible.
- **Open question (answerable only during implementation):** the
  exact first-failing statement and its transaction/dirty semantics
  in the CI log — needed to place the fix precisely; Phase 0, step 4.
- No user-owned product decisions are pending; all choices above are
  agent-owned technical calls constrained by the directive.
