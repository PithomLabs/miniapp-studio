# Card 001: focused Neon regression gate (seven tests)

- Status: `in-progress` (owner GO granted for focused run pending this amendment's review; no live execution
  until owner confirms)
- Roadmap: A (quality gate for all B-series work)
- Owner decision required: GO to execute one focused live run + fix any defects at root cause (repo-local
  fixes only; no infra changes; no commits without separate authorization). Full-suite-green is explicitly
  DEFERRED to the pre-demo gate — not claimed by this card.

## Outcome

The seven-test focused gate passes live against the Neon dev branch: the three previously failing store
tests (privilege parity ×2, set-repair cleanup), the two pool-reset tests covering the hook change, the
hook-timeout mechanism proof (unit-only, runs anywhere), and the down/up/down-up cycle test. Last measured
state (2026-11-27): 145 pass / 0 fail / 11 skip with three aggregate timeouts at 10m.

## Acceptance criteria

- [ ] Focused run completes with the exact command in §7; evidence log preserved.
- [ ] Seven tests pass live: TestRoleAttributes, TestPrivilegeParity, TestStudioLookupSetRepair,
  TestMigrationDownUpBounded, TestPoolResetsTenantSetting, TestPoolDiscardsUnresettable,
  TestTenantResetHookTimesOut.
- [ ] Zero `--- FAIL`, zero panics in the focused run; per-test durations recorded.
- [ ] Any red item triaged at root cause.
- [ ] Full-suite verification recorded as DEFERRED to the pre-demo gate, never as green.

## Dependencies

- Neon dev branch + binding files (exist, verified in prior runs).
- Owner-terminal execution (agent shell has no Neon path).

## Decision brief

1. Execution location: agent sandbox (analysis, fixes) + owner terminal (live suite run).
2. Application target: `miniapp-studio` test suite, no product behavior change expected.
3. Database target: Neon dev branch, run-pattern databases only; bootstrap database untouched.
4. Infrastructure actions: none (test-run DB provisioning under the existing authorized dev workflow only).
5. Authorization class: repo-local implementation + tests; live test execution owner-terminal; publish
   of this card + `store/pool.go` + `store/pool_test.go` explicitly granted; no other commits, pushes,
   provisioning, or destructive ops.
6. Verification and evidence: `gofmt`, `go vet ./...`, `go build ./...`, focused tests, then the full
   owner-terminal run log with counts; phase trails reviewed for anomalies.
7. Recovery and rollback: run DBs are disposable (fresh per run, reaper covers orphans); code fixes revert
   by checkout since nothing is committed without authorization.

## Report (§7)

**Agent-side prep complete (2026-11-27).** Tree verified clean (no tracked modifications); binding file
valid (schema + host count checked, values withheld); repo-local gates green: `gofmt` clean, `go vet ./...`
clean, `go build ./...` OK, unit suite 425 pass / 306 skip / 0 fail.
**Code changes in this card:** `store/pool_test.go` gains `TestTenantResetHookTimesOut` (in-memory pgwire
blackhole peer; asserts the hook returns false at its deadline); `store/pool.go` changes `tenantResetTimeout`
from const to test-overridable var (production value unchanged at 10s). Proof command + result:
`go test -count=1 -run TestTenantResetHookTimesOut ./store/` → PASS in 0.2s. The test also passed 3× under
`-count=3` alongside the migratedb timing suite.
**Unresolved pre-demo follow-up (not closed):** one transient `--- FAIL` appeared in a single full-suite run
and never reproduced across two full reruns plus targeted stress (migratedb timing tests 3×, hook test 3×,
gateway package 3×). Prime suspect is the known pre-existing gateway race; if it recurs, capture the verbose
log before rerunning. It must not block this card, but it must not be forgotten either.

**Blocker (boundary, not defect):** the approved Neon connection is unavailable in the agent shell — all four
Neon/DSN environment variables are unset, and sandbox policy forbids seeking credentials. Per card
instructions, live execution stops here. No credentials requested, probed, or exposed.

**Minimum owner action:** run the focused gate owner-terminal (binding + DSN files already in place):
```
set -o pipefail
time task dev:test -- -- -run 'TestRoleAttributes|TestPrivilegeParity|TestStudioLookupSetRepair|TestMigrationDownUpBounded|TestPoolResetsTenantSetting|TestPoolDiscardsUnresettable|TestTenantResetHookTimesOut' -timeout 15m -v -count=1 2>&1 | tee test-run-card001.log
```
then reply "done". The double `--` is load-bearing (task separator + dev-test passthrough terminator);
`-count=1` defeats result caching. Agent will verify from the log, triage every red item at root cause, and
either close the card green or file named defect cards. No commits, pushes, or Card 002 work in this card.

**Focused gate result: RED (2026-11-27, ~22:55 +08:00, store package 297.7s).** Three confirmed failures
(owner-terminal output pasted; the tee log was clobbered by a follow-up split-paste shell error, so only
pasted fragments survive — the re-run must preserve one intact log):
- `TestRoleAttributes` (43.1s): `no PUBLIC execute = false`; `app cannot execute operator_upsert = false`;
  `test cannot execute operator_upsert = false`.
- `TestPrivilegeParity` (89.3s): `function studio.operator_upsert(text,text,boolean) EXECUTE: live=true
  test=true, want false`. (The `live CONNECT` line is informational, not a failure.)
- `TestStudioLookupSetRepair` (23.1s): cleanup `DROP ROLE` → `role "studio_probe_..." cannot be dropped
  because some objects depend on it (SQLSTATE 2BP01)`.
- `TestMigrationDownUpBounded`, `TestPoolResetsTenantSetting`, `TestPoolDiscardsUnresettable`,
  `TestTenantResetHookTimesOut`: UNADJUDICATED this run (no surviving output either way). The re-run
  settles all seven.
These are the same three signatures as the pre-000026 full-suite run: the 000026 forward REVOKE and the
DROP OWNED cleanup fix did not hold live — both prior fixes were made without observing the live
mechanism, and this card will not repeat that.

**Triage (static analysis exhausted, recorded honestly).** Audited every writer of `operator_upsert`'s ACL:
000002 creates it (revokes PUBLIC, grants migrating user); 000026 revokes `studio_app`; no other migration
touches it; `GrantTestPrivileges` grants only the three `session_*` functions to the test role; the cycle
test runs on a separate scratch DB; `CreateRunDB` is a plain `CREATE DATABASE` (no template); no
`ALTER DEFAULT PRIVILEGES` or `GRANT ALL` anywhere; roles are not superuser (all TRUNCATE negatives held).
On a fresh v26 DB the observed ACL (EXECUTE for app + test, EXECUTE for PUBLIC somewhere in `studio`) has
no producing code path — the mechanism lives in the live catalog, not the repo. Leading theories, both
UNPROVEN: (a) the SetRepair probe function survives cleanup with default PUBLIC EXECUTE (cleanup's
DROP OWNED error was swallowed by `_, _`, and the canonical repair's ADMIN FALSE may have removed the
cleanup's own permission — the test's step 4 breaking its own step 9); (b) a single PUBLIC grant on
`operator_upsert` explains all three privilege symptoms at once, which 000026 (revoking only `studio_app`)
would not remove. Neither is asserted as fact.

**Instrumentation slice (this commit).** `store/acl_dump_test.go` (new): `dumpFunctionACLs` renders every
studio/app function with owner + grantee=privilege/grantor triples, all default ACLs, and role superuser
flags — hooked into both privilege tests via `if t.Failed()`; `dumpRoleDeps` lists a scratch role's owned
objects, shared dependencies, memberships with admin/set/inherit options, and held grants. `set_repair_test.go`
cleanup rewritten: DROP OWNED errors now surface, object drops run AS the probe via SET ROLE on a dedicated
connection (SET TRUE survives the canonical repair; RESET guaranteed before release), schema grant revoked
explicitly, and any DROP ROLE failure carries the full dependency dump. Genuinely fixes the cleanup under
every realistic theory; worst case it fails with the mechanism attached. Repo-local gates green
(gofmt/vet/unit 426/306/0).
**Next:** owner pushes, reviews, and on GO re-runs the identical seven-test command with one intact log.
Expected: SetRepair green; privilege checks red-WITH-mechanism (ACL dump names the grantor) — then one final
targeted fix and one final green run. Two live runs to gate-green, each decisive; no full suite.

**Instrumented run (2026-11-27 ~23:18 +08:00, store 467s). Still red, now with mechanism. Three decisive facts:**
- SetRepair cleanup: `DROP OWNED BY probe → 42501 permission denied` (ADMIN-FALSE theory CONFIRMED live —
  DROP OWNED needs what the canonical repair took away); `SET ROLE probe` SUCCEEDED but the drop then failed
  `42501 permission denied for schema studio` (probe holds CREATE but not USAGE — CREATE does not imply
  USAGE). Deps dump errored on an uncast `"char"` (`d.deptype ||` — dump bug, fixed with `::text`).
- RoleAttributes dump: `probe_transfer_fn` survives with NULL-default ACL (PUBLIC EXECUTE) — the line-260
  "no PUBLIC execute" failure is SetRepair's leftover polluting the shared DB. CONFIRMED.
- Privilege paradox narrowed: the four migration functions show materialized owner-only ACLs yet
  `has_function_privilege` returns TRUE. pgx source read (v5.13.0 `MultiResultReader.receiveMessage`:
  every ErrorResponse overwrites `mrr.err`, drained by `Close`) proves mid-batch statement errors SURFACE
  as the last error — so the silent-42501 theory is DEAD and every GRANT/REVOKE statement SUCCEEDED,
  which promotes the dump itself to prime suspect (INNER JOINs silently drop PUBLIC grantee oid 0 and any
  unresolvable grantor).
**Second slice (this commit):** dump shows RAW `proacl::text` ground truth + LEFT-JOIN decode + regprocedure
OID vs catalog OID + privilege-by-text vs by-OID + raw schema/db ACLs; probe gets USAGE with CREATE;
cleanup revokes ALL schema grants. Predictions for the confirm run — P1: SetRepair GREEN; P2: line-260 GREEN
(probe fn gone), lines 264/276 still RED; P3: parity still RED with raw ACL ending it (PUBLIC `=X` present →
hunt the grantor; absent with by-OID TRUE → environment anomaly with proof); P4: cycle + pool×2 + hook GREEN
with `--- PASS` lines captured. If P3 shows PUBLIC, the repair slice is forward 000027 (SET ROLE
studio_lookup → REVOKE PUBLIC → restore intended grants → RESET ROLE) plus an adapter-hardening decision;
000027 is deliberately NOT in this slice — the confirm run must observe the hole, not the repair.
