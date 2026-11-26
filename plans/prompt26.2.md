**No structural blockers apparent from the reported R1 changes.** The fixes address the original root causes rather than merely weakening the checks.

However, I am not granting final B1 acceptance yet. The next step is a **focused, independent adversarial re-review of all eleven findings**, especially the three previously Medium findings.

GO — INDEPENDENT RE-REVIEW

# B1-R1 — Independent Adversarial Remediation Verification

## Role

Act as an independent Principal Security Engineer and Adversarial Go Reviewer.

Review the completed B1-R1 remediation against the original adversarial findings in:

`plans/adv_review_prompt26.md`

Read root `SKILLS.md`, the approved lifecycle plan, `docs/bootstrap-evidence.md`, and the actual implementation.

Your mission is to independently establish whether every original finding has been genuinely resolved, whether the fixes introduce regressions, and whether B1 is now safe to accept.

Do not accept implementation-agent claims without inspecting the code and reproducing the relevant evidence.

## 1. Decision brief

| Field          | Direction                                                                          |
| -------------- | ---------------------------------------------------------------------------------- |
| Objective      | Independently verify closure of B1 adversarial findings F-01 through F-11          |
| Execution      | Local Desktop only                                                                 |
| Application    | Local Miniapp Studio                                                               |
| Database       | Local PostgreSQL, isolated run-unique test databases and roles                     |
| Infrastructure | None; no Fly, Sprites, Machines, Neon or GitHub operations                         |
| Authorization  | Read-only review and safe local testing; no remediation                            |
| Verification   | Targeted adversarial tests, DB-backed suites, race tests and full Go suite         |
| Recovery       | Preserve all existing modifications; clean only review-created temporary resources |

B2 is not authorized.

## 2. Re-review all eleven findings

Use the original report as the authoritative finding register.

### F-01 — Filesystem TOCTOU and FIFO safety

Verify that:

- File opening is descriptor-relative and uses O_NOFOLLOW appropriately.
- Parent-directory traversal cannot follow malicious symlinks.
- File type and size are verified using fstat on the opened descriptor.
- FIFO and special-file handling cannot hang or introduce unintended side effects.
- Hashing supports cancellation and bounded execution.
- No check-then-open vulnerability remains.
- Failures are explicit and fail closed.

Inspect the actual system calls and descriptor handling where practical. Run deterministic regression tests, not timing-dependent race lotteries.

### F-02 — Assume-unchanged / skip-worktree

Verify that both Git index flags are detected.

Demonstrate that changes hidden from ordinary `git status` cannot produce an accepted, apparently clean checkpoint.

Test both flag types and confirm the refusal or explicit reconciliation behavior.

### F-03 — Submodule state

Verify that:

- Submodule gitlinks are bound to actual checked-out HEADs.
- Initialized and clean state is required.
- Dirty, uninitialized, mismatched and unsupported nested submodules fail closed.
- Deinitialized submodules do not accidentally resolve through parent-directory traversal.
- Different submodule contents cannot produce identical accepted manifests.

### F-04 — Idempotency identity

Verify:

- Separate start and resume operation namespaces.
- Cross-build reuse returns a conflict.
- Cross-operation reuse returns a conflict.
- Checkpoint ownership is verified before returning a replay.
- Replays do not write duplicate records or events.

### F-05 — Ignored files

Verify that the exclusion is clearly documented in the runbook and visible to operators.

Confirm the system does not claim Git manifest validation covers ignored configuration or secrets.

### F-06 — Git-derived secrets

Test branch names and paths containing recognizable secret canaries.

Verify raw sensitive values cannot enter checkpoint records, events, Pulse, rendered Console content, or agent execution packets.

Confirm the refusal/scrubbing behavior does not corrupt manifest integrity or create ambiguous path identities.

### F-07 — waiting_user state

Verify the refusal message accurately identifies the gateway-owned state and that existing state-machine protections remain intact.

### F-08 — Repository path identity

Verify the documented B1 semantics: repo_path is provenance metadata, not a durable workspace identity.

Confirm the operator-facing warning is clear and that B2 requirements are explicitly documented.

### F-09 — Manifest serialization

Verify `mf2` uses unambiguous canonical framing.

Test legal filenames containing pipes, newlines and unusual characters.

Verify old `mf1` checkpoint rows remain readable without silently interpreting their digests as `mf2`.

### F-10 — Git configuration independence

Verify rename detection is explicitly pinned.

Test that different user/system Git settings do not produce different manifests for the same worktree.

### F-11 — Workbench listing honesty

Verify unreadable entries are surfaced as skipped/error information and that incomplete listings cannot be presented as complete.

## 3. Regression and integration review

In addition to the original findings, inspect whether R1 introduced regressions in:

- B0 packet and attempt compatibility.
- Checkpoint creation and retrieval.
- Resume state validation.
- Checkpoint/event atomicity.
- Tenant isolation and RLS.
- Secret scrubbing.
- Concurrency and operation-key semantics.
- Focus Console rendering.
- Held Gateway assignment behavior.

Verify the actual database migration state and test-role parity.

## 4. Independent verification requirements

Where the environment permits, independently execute:

- Targeted Git inspection tests.
- Checkpoint/resume tests.
- Submodule and Git flag fixtures.
- Symlink/FIFO safety tests.
- Concurrent checkpoint/resume tests.
- Migration up/down round-trip.
- RLS and privilege-parity tests.
- Targeted race-detector suites.
- `gofmt`, `go vet ./...`, `go build ./...`.
- `go test -p 1 -count=1 ./...`.

Use only databases and roles created specifically for the review.

Do not report DB-backed tests as green when they were skipped or unavailable.

## 5. Review method

For every finding:

1. Inspect the remediation implementation.
2. Trace the actual call path.
3. Reproduce the original failure scenario.
4. Verify the regression test.
5. Test the corrected behavior.
6. Inspect related failure and recovery paths.
7. Record exact file/line references and evidence.

Look for new weaknesses introduced by the remediation, not only whether the original reproduction now fails.

Distinguish confirmed fixes from code-inspected but unexecuted claims.

## 6. Required report

Write:

`plans/b1-r1-adversarial-re-review.md`

Include:

- Executive verdict.
- F-01 through F-11 closure matrix.
- Root-cause remediation verification.
- Regression findings.
- Database and concurrency verification results.
- Remaining risks and environmental limitations.
- Exact commands and test outcomes.
- B0 compatibility assessment.
- Final B1 acceptance recommendation.

Classify each finding as:

- Verified closed.
- Partially resolved.
- Open.
- Accepted limitation.

## 7. Final verdict

Choose exactly one:

**ACCEPT B1:** All mandatory findings are verified closed, no Critical or High findings remain, and DB-backed verification is independently green.

**CONDITIONAL ACCEPTANCE:** No Critical or High findings, but clearly identified bounded deficiencies or verification gaps remain.

**REJECT B1:** Any Critical or High finding, or any demonstrated failure of a fundamental integrity, recovery, filesystem safety, or tenant-isolation invariant.

Do not silently downgrade the original findings.

## 8. Strict boundaries

This is review-only.

- Do not modify source code or tests.
- Do not implement fixes.
- Do not modify migrations.
- Do not perform cloud operations.
- Do not start B2.
- Do not provision Sprites or Machines.
- Do not modify Neon or GitHub.
- Do not commit, push, reset, stash or change branches.
- Preserve all owner work.

Stop after delivering the report and recommendation.

**After a clean review:** I will issue final B1 acceptance, then prepare the B2 remote Builder/Sprite handoff directive as a separate authorized milestone. B2 will receive its own security and recovery review.