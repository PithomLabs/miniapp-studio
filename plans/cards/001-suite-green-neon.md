# Card 001: suite green on Neon at measured timeouts

- Status: `draft`
- Roadmap: A (quality gate for all B-series work)
- Owner decision required: GO to execute one full live suite run + fix any defects at root cause (repo-local
  fixes only; no infra changes; no commits without separate authorization).

## Outcome

The full DB suite passes live against the Neon dev branch, establishing the verified quality baseline every
later card builds on. Last measured state (2026-11-27): 145 pass / 0 fail / 11 skip with three store FAILs
since fixed in-tree (000026 REVOKE, DROP OWNED cleanup) plus three aggregate timeouts at 10m.

## Acceptance criteria

- [ ] Full suite run completes at `-timeout 30m` streaming, evidence log preserved.
- [ ] Zero `--- FAIL`, zero panics; skips only pre-approved pending-live categories with counts recorded.
- [ ] The three previously failing store tests pass live (privilege parity ×2, set-repair cleanup).
- [ ] Per-package durations recorded as the new timeout-sizing baseline.
- [ ] Any new red item triaged to a named defect card with evidence attached (no silent carries).

## Dependencies

- Neon dev branch + binding files (exist, verified in prior runs).
- Owner-terminal execution (agent shell has no Neon path).

## Decision brief

1. Execution location: agent sandbox (analysis, fixes) + owner terminal (live suite run).
2. Application target: `miniapp-studio` test suite, no product behavior change expected.
3. Database target: Neon dev branch, run-pattern databases only; bootstrap database untouched.
4. Infrastructure actions: none (test-run DB provisioning under the existing authorized dev workflow only).
5. Authorization class: repo-local implementation + tests; live test execution owner-terminal; no commits,
   pushes, provisioning, or destructive ops.
6. Verification and evidence: `gofmt`, `go vet ./...`, `go build ./...`, focused tests, then the full
   owner-terminal run log with counts; phase trails reviewed for anomalies.
7. Recovery and rollback: run DBs are disposable (fresh per run, reaper covers orphans); code fixes revert
   by checkout since nothing is committed without authorization.

## Report (§7)

(Pending execution.)
