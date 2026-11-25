This review should verify the **residual-closure work only**, not repeat the original Phase B adversarial review. Its most important checks are whether N1 and N4 are genuinely closed, whether the PostgreSQL 19 evidence is reproducible, whether the upstream hashes are correct, and whether the reported test flake is being characterized honestly.

# Scoped Adversarial Review — Prompt 12 Residual Closure

## 1. Role

Act as an independent Principal Security Engineer, Senior Go Systems Architect, and adversarial reviewer for:

`github.com/PithomLabs/miniapp-studio`

Your assignment is to independently verify the completed Prompt 12 residual-closure pass.

The implementation agent reports:

- N1, N4, R-DB, and R-NET independently closed.
- N2–N10 tracked with operational restrictions.
- Seven modified files and one new file, all uncommitted.
- Two pre-existing owner commits: `2d020a3` and `22ce9c4`.
- PostgreSQL 19 verification fully green.
- OpenCode v1.19.6 upstream hashes independently verified against tag commit `f4e363c`.
- One pre-existing `TestAdapterStatus` cleanup flake observed twice in approximately nine runs, attributed to untouched prompt11-era test code and left unchanged.
- Part 2 has not been executed or authorized.

Your responsibility is to challenge these claims against the actual working tree, source code, tests, Git history, database evidence, and documentation.

This is a **bounded verification exercise**, not another implementation pass or general architecture review.

## 2. Governing documents

Read the following local working-tree documents:

1. `docs/prompt12-residual-closure-report.md` — primary residual-closure evidence.
2. `plans/adv_review_prompt12.md` — independent review that established the residuals.
3. `docs/prompt11-correction-report.md` — preceding correction implementation and evidence.
4. `docs/gateway-phase-b-evidence.md` — current Phase B evidence.
5. `docs/gateway-phase-b-live-runbook.md` — owner-run live verification.
6. `docs/opencode-run-v1.19.6.md` — vendored upstream audit record.
7. `plans/miniapp-studio-implementation-agent-guidelines.md` — governing implementation and review methodology.

Inspect the actual changed source files, tests, and documentation. Do not accept the residual-closure report as proof of its own claims.

If a document is genuinely absent from the local working tree, report that explicitly. Do not confuse an uncommitted file with a missing file in GitHub's remote branch.

## 3. Review boundaries

This review is read-only.

Do not:

- Modify source, tests, documentation, plans, workflow files, or configuration.
- Stage, commit, push, reset, stash, or rewrite Git history.
- Delete or recreate databases.
- Provision, destroy, or modify Sprites.
- Perform Fly API mutations.
- Start OAuth or install OpenCode on live infrastructure.
- Execute the Part 2 live runbook.
- Request or expose production credentials.
- Modify Miniapp or its repository.
- Change F1, G-01, P5, or accepted architecture.
- Implement N2–N10.
- Reopen the original B1/F2–F9 correction bundle unless a residual-closure change demonstrably regresses one of its accepted invariants.

The Part 2 authorization boundary remains unchanged.

Record the initial Git state and verify that it is unchanged after the review, except for any temporary test artifacts outside the repository that can be safely cleaned up without modifying user data.

## 4. Git history and change attribution

The working tree is no longer zero-commit. Two owner commits exist:

- `2d020a3`
- `22ce9c4`

The residual-closure report claims the implementation agent performed zero Git writes and left exactly seven modified and one new files uncommitted.

Independently verify:

1. Resolve both short commit IDs to full SHAs.
2. Establish their parent/ancestry relationship and order.
3. Inspect both commits and identify their authorship, messages, and changed files.
4. Identify the current HEAD and branch.
5. Record `git status --short` and `git diff --stat`.
6. Enumerate every modified, deleted, renamed, and untracked file.
7. Compare the working tree against HEAD and identify exactly which changes are uncommitted.
8. Verify whether the claimed seven modified plus one new file is exact.
9. Identify whether any change is outside the authorized residual-closure scope.
10. Check that no source or documentation modification has been accidentally attributed to the two owner commits or vice versa.

Do not assume the two owner commits are necessarily the complete baseline for all residual changes. Prove the relationship from Git.

Do not stage or clean up untracked files.

The report must distinguish:

- Existing owner commits.
- Uncommitted residual-closure implementation changes.
- The new residual-closure report.
- Any pre-existing unrelated working-tree material, if found.

A changed-file count alone is not proof of scope compliance. Inspect the actual diff.

## 5. N1 — Gateway privilege contract and boot-time credential-drop self-test

This is the most important security verification in this review.

The previous defect was a mismatch between the claimed Gateway/agent privilege boundary and the actual operating-system process identity.

The residual pass claims to have added a boot-time credential-drop self-test in `serve()`.

### 5.1 Verify the actual startup path

Trace the complete production path:

```
Gateway binary
    → configuration loading
    → service identity resolution
    → Gateway/agent identity distinction
    → credential construction
    → self-test process spawn
    → successful child exit
    → server readiness / serve loop
```

Verify:

- The self-test executes before the Gateway becomes ready.
- It uses the exact credential-construction function used by real agent processes.
- It uses the configured agent UID, GID, and supplementary groups.
- It launches a harmless fixed executable.
- The child is actually waited for and its exit status checked.
- UID/GID resolution failures prevent readiness.
- Insufficient privilege, including `EPERM`, prevents readiness.
- Same-UID operation cannot silently bypass the production check.
- Test-only escape hatches cannot be enabled through normal service configuration.
- Alternate startup or serving paths cannot bypass the check.
- Errors are useful but do not disclose secrets.

Do not accept a test that checks only a constructed `SysProcAttr` value. Verify that the real startup path performs an actual child process launch using the production credential configuration.

### 5.2 Test coverage and failure semantics

Inspect all seven newly reported tests.

Determine whether they prove the relevant invariants or merely exercise helper functions.

Look for missing cases, including:

- Gateway and agent identities accidentally equal.
- Agent UID or GID cannot be resolved.
- Required supplementary group unavailable.
- Credential-drop spawn returns `EPERM`.
- Child exits nonzero.
- Child cannot be waited for or its execution status cannot be established.
- Service starts through another entry point.
- Self-test succeeds but actual agent spawn uses different credential settings.

Where practical, execute focused local-process tests safely.

### 5.3 Permission matrix and runbook

Verify the privilege contract is consistent across:

- `docs/prompt12-residual-closure-report.md`
- `docs/prompt11-correction-report.md`
- `docs/gateway-phase-b-evidence.md`
- `docs/gateway-phase-b-live-runbook.md`
- `tools/provisiondata/setup.sh`
- `cmd/gateway/main.go`
- `gateway/supervisor.go`
- Relevant filesystem implementation.

Verify that:

- Service EUID is derived from the actual running process.
- Numeric UID/GID comparisons are used where appropriate.
- Gateway-created files are not incorrectly assumed to be owned by `studio-gw`.
- `studio-io` group access remains correct.
- The exact permission bits, group ownership, and permitted/denied operations agree with the implementation.
- The runbook distinguishes kernel-observed denial from agent self-report.
- The service privilege requirement is operationally executable and not merely descriptive.

The actual live Sprite key-read-denied proof remains a Part 2 obligation. Do not require or execute it here.

**N1 passes only if the production service cannot become ready without proving that its actual agent credential drop works.**

## 6. N4 — M5 naming and superseded evidence matrix

The residual pass claims:

- `M5-STDERR` identifies the completed bounded stderr work.
- `M5-SSE` identifies the original deferred SSE lifetime/concurrency work.
- Matrix v1 is explicitly superseded.
- Matrix v2 is authoritative.
- The mapping is consistent across four documents.

Independently search all current relevant documentation and verify:

1. Every current M5 reference is unambiguous.
2. Historical references are explicitly annotated.
3. The mapping note accurately explains the identifier collision.
4. Matrix v1 carries an unmistakable superseded banner before its content.
5. Matrix v2 is identified as current authoritative evidence.
6. No historical present-tense statement can mislead a reader into treating old capabilities as current verification.
7. The SSE lifetime and concurrent-stream-budget work remain explicitly deferred.
8. The claim that prompt11 M1–M16 were fixed does not imply that the original M5-SSE backlog was completed.

Search for all occurrences, not only the four files listed by the implementation agent.

A clean audit must account for every remaining bare `M5` occurrence and explain why it is valid.

Do not change historical review documents merely to erase evidence of the original numbering collision.

## 7. R-DB — PostgreSQL 19 verification audit

The implementation agent reports:

- Full suite: 13/13 packages green.
- Six-package race suite green.
- Fourteen F7 tests.
- Three M15 tests.
- Fifteen migration/RLS tests.
- Zero skips in database-backed runs.

Do not accept these counts without checking the exact executions.

### 7.1 Version proof

Verify recorded output for:

```
SHOW server_version;
SELECT version();
```

Require PostgreSQL 19.x.

Verify that the tests used the user's intended local Docker PostgreSQL 19 environment:

`127.0.0.1:5434`

Do not substitute another version.

Confirm that database-backed commands used a safely isolated test database/schema and did not destroy or reset user data.

### 7.2 Full-suite and race evidence

Inspect exact command transcripts, exit codes, package output, and test skip summaries for:

```
go test -p 1 -count=1 ./...
```

and:

```
go test -race -p 1 -count=1 ./gateway ./store ./tools ./server ./cmd/studio ./cmd/gateway
```

Check that the six-package race command truly used the PG19-backed configuration where applicable.

Verify that the claimed 13/13 packages and zero database-test skips correspond to actual executed results.

### 7.3 Targeted regression evidence

Independently identify the actual tests included in:

- 14 F7 provisioning/upgrade/locking tests.
- 3 M15 KDF tests.
- 15 migration/RLS tests.

Verify that the counts are accurate, that no test was skipped, and that the tests assert the intended contracts.

For F7, inspect the cross-session advisory-lock test and verify that it genuinely uses separate database sessions/pools rather than two goroutines sharing one connection or transaction.

For M15, verify actual KDF wiring and rejection of an incorrect secret, not just a standalone derivation helper.

For migration/RLS, verify the relevant policies, tenant isolation, and fail-closed behavior.

### 7.4 Evidence quality

Distinguish:

- Tests executed against PG19.
- Test code inspected but not executed.
- Tests executed without a database.
- Historical test evidence.

Do not use the phrase "zero skips" if the test commands omitted database-dependent cases or ran with an absent environment variable.

**R-DB passes only when the reported PG19 evidence is independently substantiated.**

If the original run logs are unavailable, state precisely what can be independently confirmed and classify the rest as evidence-limited.

## 8. R-NET — OpenCode upstream hash verification

The implementation agent reports that both hashes were re-verified against upstream tag commit:

`f4e363c`

Source:

- Repository: `anomalyco/opencode`
- Tag: `v1.19.6`
- Path: `packages/opencode/src/cli/cmd/run.ts`

Expected hashes from the vendored audit record:

- Git blob SHA-1: `944aec17abd75cc3272688e89a140b56cd2bb2d3`
- Source SHA-256: `e040e90d4c81cdd59b9568ee90a53c0209cd9ddf1b8b207d1a8685b2e26d57d0`

Independently verify:

1. The tag resolves to the reported upstream commit.
2. The source path exists in the tag tree.
3. The fetched bytes match the expected file.
4. Git blob SHA-1 is calculated using Git's blob-object hashing semantics, including its header.
5. SHA-256 matches the vendored record.
6. The implementation report records the exact resolved commit and commands.

Do not rely solely on a hash printed in the report.

If independent network access is unavailable, mark the result evidence-limited instead of claiming verification.

No code from upstream should be copied or modified as part of this audit.

## 9. N2–N10 — Residual register and operational restrictions

The implementation agent claims all remaining findings are tracked.

Verify that the register accurately preserves the risks, required corrections, and restrictions.

| ID  | Residual                                     | Critical review question                                                                                                                    |
| --- | -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| N2  | Upgrade idle-check TOCTOU and empty-URL skip | Is live Gateway upgrade clearly prohibited until this is fixed? Is the restriction visible in the operator runbook and upgrade procedure?   |
| N3  | Recovery-guidance messages misdirect         | Does the register accurately distinguish retrying the same command from continuing as a new turn or creating a fresh assignment?            |
| N4  | M5 collision and matrix v1 supersession      | Is this actually closed as defined in §6?                                                                                                   |
| N5  | PID recorded after launch capture            | Is the orphan-recovery implication disclosed? Does the live runbook require recording pre-restart PIDs and verifying post-recovery absence? |
| N6  | Launch identity capture has no timeout       | Does the register accurately disclose the risk of a hung binary retaining the build semaphore?                                              |
| N7  | Deliverer transition failures silent         | Is the observability gap preserved rather than described as fixed?                                                                          |
| N8  | Staging re-creation permission window        | Are the exposure conditions and operational handling accurately documented?                                                                 |
| N9  | Adapter Continue check-then-act              | Is current server serialization explicitly the relied-upon boundary? Is this not misrepresented as adapter-level protection?                |
| N10 | Silent IOGID fallback                        | Is the fail-closed consequence and required future correction accurately recorded?                                                          |

For each residual, verify:

- Accurate technical description.
- Correct affected component.
- Realistic impact and blast radius.
- Exact required durable fix.
- Required regression test.
- Whether it blocks Part 2 or only a narrower operation.
- Owner or future phase assignment where recorded.

Specifically challenge N2: a documentation warning is insufficient if the operator can still accidentally perform a live upgrade without seeing the prohibition.

Do not implement these findings. This is a tracking and restriction verification.

## 10. TestAdapterStatus cleanup flake

The implementation report discloses:

- A pre-existing `TestAdapterStatus` cleanup flake.
- Observed twice in approximately nine runs.
- Root cause attributed to untouched prompt11-era test code.
- Left unchanged because it was outside the residual-closure scope.

Independently inspect this issue. Do not dismiss it merely because it predates the latest work.

### Required investigation

1. Locate the exact test and its cleanup path.
2. Trace the resource lifecycle: processes, sessions, temporary files, goroutines, channels, contexts, timers, and filesystem resources used by the test.
3. Determine the actual failure mode and whether it can contaminate another test or leave a process/resource behind.
4. Inspect the implementation agent's root-cause explanation and verify that it matches source behavior.
5. Re-run the isolated test a bounded number of times where safe and useful.
6. Inspect whether the flake is affected by race detection or test ordering.
7. Determine whether the issue is merely nondeterministic test cleanup or indicates an actual production lifecycle defect.
8. Verify that the residual-closure changes did not introduce or worsen it.

Do not modify the test or attempt to fix it.

Classify the result:

- Pre-existing, test-only, non-blocking residual.
- Test isolation/reliability defect requiring bounded correction.
- Possible production lifecycle defect.
- Not reproducible / evidence insufficient.

Explain the impact on confidence in the reported green suite.

If the flake could cause resource leakage, interference, false-green results, or conceal production process-lifecycle issues, it must not be dismissed as harmless.

Do not broaden this into a general audit of all tests.

## 11. Changed-file and implementation-scope review

Review the actual seven modified and one new files.

For each file, report:

- Why it changed.
- Which residual it closes.
- Relevant functions/sections.
- Whether the implementation matches the report.
- Whether there is unrelated scope.
- Whether tests prove the intended invariant.

Check that:

- No new production dependency or framework was introduced.
- Miniapp remains untouched.
- Accepted Agent Gateway architecture remains unchanged.
- No unrelated test or CI changes were slipped in.
- Owner commits are correctly attributed.
- No new Git writes were performed by the reviewer.

The review should be proportional to the narrow residual-closure diff. Do not reopen every previously accepted Phase B finding.

## 12. Adversarial methodology

Apply the governing Miniapp Studio Implementation Agent Guidelines.

For every suspected defect:

1. Identify the violated invariant.
2. Trace the full execution path.
3. Establish a concrete failure scenario.
4. Reproduce it safely where possible.
5. Identify the owning component.
6. Verify whether the residual correction closes the root cause.
7. Require evidence from the appropriate level.

Do not manufacture issues to make the review appear thorough.

Do not report preferences as defects. Do not treat documentation presence as proof of runtime behavior. Do not treat a test name as proof that the corresponding invariant is covered.

Use precise evidence classifications:

- Code-verified.
- Test-verified.
- Local-process-verified.
- PostgreSQL-19-verified.
- Upstream-source-verified.
- Live-verified.
- Unverified.

Live verification is outside this review's authorization.

## 13. Required deliverable

Create:

`docs/adv-review-prompt12-residual-closure.md`

The report must include:

1. Executive verdict.
2. Exact Git tree and commit state.
3. Changed-file inventory and scope assessment.
4. N1 security and privilege self-test assessment.
5. N4 documentation and M5 mapping assessment.
6. R-DB independent PG19 verification matrix.
7. R-NET provenance and hash verification.
8. N2–N10 residual/restriction matrix.
9. `TestAdapterStatus` flake investigation.
10. Confirmed defects and documentation discrepancies.
11. Tests and checks not run, with exact reasons.
12. Final acceptance recommendation.

For every finding, include severity, location, violated invariant, root cause, evidence, impact, and required correction or tracking action.

Use these closure statuses:

- VERIFIED CLOSED
- CLOSED — EVIDENCE LIMITED
- DOCUMENTATION CORRECTION REQUIRED
- TRACKED — OPERATIONAL RESTRICTION VERIFIED
- NOT ACCEPTED

Do not report N2–N10 as fixed merely because they are documented.

## 14. Final verdict

Choose exactly one:

**ACCEPTED**

Use only if N1, N4, R-DB, and R-NET are independently verified closed; N2–N10 are correctly tracked with their operational restrictions; and the test flake is demonstrated not to undermine production correctness or the verification result.

**ACCEPTED WITH BOUNDED RESIDUALS**

Use if the four closure prerequisites are established but the test flake or other clearly bounded, non-blocking issue remains, with an explicit owner and required action.

**NOT ACCEPTED**

Use if a claimed closure is unsupported, a security or correctness invariant remains violated, PG19 evidence is materially inaccurate, the upstream hash is wrong, the N2 upgrade restriction is ineffective, or the test flake reveals a material unresolved lifecycle/reliability defect.

For every non-accepted item, provide the smallest durable correction required. Do not request another broad implementation or review cycle.

A positive review is a recommendation only. It does not authorize Part 2.

Explicitly state in the final report:

- No live Sprite/Fly/OAuth operations were performed.
- Part 2 remains unauthorized.
- No source, tests, plans, or documentation were modified by the reviewer, other than the review report itself.
- No commits or pushes were made.

Stop after producing the report.

**Review emphasis:** I would give the most scrutiny to the actual boot-time credential self-test (not just its helper tests), the PG19 test transcripts, and the `TestAdapterStatus` cleanup flake. Those are the strongest checks against accepting a successful report without independently establishing the underlying guarantees.