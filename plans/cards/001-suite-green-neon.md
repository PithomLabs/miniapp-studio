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
