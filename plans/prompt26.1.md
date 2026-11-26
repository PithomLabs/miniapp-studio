# **B1 adversarial review — architectural decision**

CONDITIONAL ACCEPTANCE — REMEDIATION REQUIRED

I agree with the review verdict. The three Medium findings affect the central promise of B1: Studio must know that the workspace it resumes is actually the workspace represented by its checkpoint.

The review demonstrated three real weaknesses: a filesystem inspection race, invisible changes under Git's assume-unchanged/skip-worktree flags, and submodule contents not bound to the manifest.&#x20;

adv_review_prompt26(1).md



**My decision is to remediate these, not accept them as permanent risks.** They are directly relevant to our future Sprite handoff, where filesystem and workspace identity become even more important.

## **1. Disposition of all eleven findings**

| Finding                                              | Decision                                                     | Priority                 |
| ---------------------------------------------------- | ------------------------------------------------------------ | ------------------------ |
| F-01 — Symlink-swap TOCTOU and FIFO hang             | Fix                                                          | Required                 |
| F-02 — Assume-unchanged / skip-worktree              | Fix                                                          | Required                 |
| F-03 — Submodule state not bound                     | Fix or refuse unsupported state                              | Required                 |
| F-04 — Idempotency replay identity                   | Fix                                                          | Before B2                |
| F-05 — Ignored files invisible                       | Document explicit limitation; define reconciliation behavior | Accepted with conditions |
| F-06 — Git-derived strings not scrubbed              | Fix                                                          | Before B2                |
| F-07 — Incorrect `waiting_user` message              | Fix                                                          | Before B2                |
| F-08 — `repo_path` not compared                      | Document as provenance-only; use workspace identity in B2    | Accepted with conditions |
| F-09 — Ambiguous manifest serialization              | Fix using unambiguous canonical encoding                     | Before B2                |
| F-10 — Git rename configuration sensitivity          | Pin behavior                                                 | Before B2                |
| F-11 — Unreadable workbench entries silently skipped | Surface skipped/error state                                  | Before B2                |

The two accepted limitations must be documented honestly. They must not become assumptions that the manifest covers all local configuration or that an absolute filesystem path is a durable workspace identity.&#x20;

adv_review_prompt26(1).md

adv_review_prompt26(1).md



## **2. Additional verification required**

The adversarial reviewer could not independently execute the database-backed suites because PostgreSQL was unreachable. The implementation's reported green results are encouraging, but we need an independently reproduced run before B2.&#x20;

adv_review_prompt26(1).md

adv_review_prompt26(1).md



Also required:

- A concurrent-resume test and resume-versus-checkpoint race test.
- Symlink, traversal and FIFO regression tests.
- Authentication/Origin coverage for every B1 mutating route.
- Independent database migration, RLS, privilege-parity and full-suite verification.

## **3. Authorization decision**

I authorize **a bounded B1 remediation continuation**, Local Desktop only. This is not authorization for B2.

## **4. Directive to the coding agent**

# B1-R1 — Adversarial Review Remediation and Final Acceptance

## Role

Act as the Senior Go Engineer responsible for closing the findings from the independent B1 adversarial review.

Read root `SKILLS.md`, the approved lifecycle plan, the B1 implementation evidence, and:

`plans/adv_review_prompt26.md`

Treat the review findings as verified defects or explicitly classified limitations, not suggestions to be dismissed.

The objective is to close B1's conditional acceptance and produce independently verifiable evidence.

## Decision brief

| Field                  | Decision                                                                                                                      |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| Execution location     | Local Desktop only                                                                                                            |
| Application target     | Local Miniapp Studio only                                                                                                     |
| Database target        | Local PostgreSQL; uniquely created test databases and roles                                                                   |
| Infrastructure actions | None; no Fly, Sprite, Machine, Neon, or GitHub operations                                                                     |
| Authorization          | B1 remediation continuation authorized; no B2, commits, or pushes                                                             |
| Verification           | Targeted regression tests, DB-backed tests, race tests, full suite and focused adversarial verification                       |
| Recovery               | Preserve all user changes; never reset or clean the worktree; temporary resources created by this task only may be cleaned up |

## 1. Required remediation

### F-01: Filesystem TOCTOU and FIFO hang

Implement secure file inspection:

- Open without following symlinks.
- Perform `fstat` against the opened descriptor.
- Verify regular-file type and size using descriptor metadata.
- Defend against intermediate path-component symlink substitution through secure directory-relative traversal.
- Ensure FIFO and other special-file attempts cannot block indefinitely.
- Thread cancellation/deadlines through hashing.
- Fail closed on any inconsistency.

Add deterministic tests for final-component symlink refusal, parent-component substitution, FIFO handling, and size validation. Assert the actual secure-opening mechanism, not successful exploitation of a timing race.

### F-02: Assume-unchanged and skip-worktree

Inspect Git index flags using appropriate machine-readable Git commands.

Detect both assume-unchanged and skip-worktree states.

Refuse checkpoint capture or require explicit reconciliation whenever tracked content may be hidden by these flags.

Test both flags, including modified file contents that ordinary `git status` would omit.

### F-03: Submodules

Bind the superproject's submodule references to actual checked-out submodule state.

For B1, prefer the smaller safe implementation:

- Support initialized, clean submodules whose checked-out HEAD matches the expected gitlink.
- Refuse dirty, uninitialized, mismatched, or unsupported nested submodule states with an explicit reconciliation requirement.
- Do not silently treat a submodule directory as an ordinary non-regular file.

Add tests proving that different submodule states cannot yield the same accepted manifest.

### F-04: Idempotency identity

Ensure an operation-key replay verifies the logical operation identity, including build and operation type.

Separate start and resume operation-key namespaces.

Reject mismatched reuse with HTTP 409 or the established conflict response.

Add cross-build and cross-operation replay tests.

### F-05: Ignored-file limitation

Document clearly that the Git manifest does not bind ignored files.

A stable Git manifest must not be presented as proof that environment configuration or ignored runtime files are unchanged.

Do not hash or persist secret values to solve this.

Make the limitation visible in the checkpoint/resume runbook and relevant operator-facing state. Require explicit operator reconciliation where ignored configuration may affect the execution.

### F-06: Git-derived secret leakage

Apply appropriate secret handling to branch names, file paths, and other Git-derived strings before persistence, rendering, or agent-context construction.

Preserve manifest integrity: do not let redaction create path collisions or alter the raw content digest calculation.

Where a sensitive Git identifier cannot safely be represented after redaction, refuse checkpoint creation or require an explicit safe rename rather than persisting the raw value.

Add recognizable secret-canary tests for branch names and filenames across database, Pulse, Console and packet surfaces.

### F-07: `waiting_user` state

Return accurate state-specific refusal messages.

Document why gateway-owned `waiting_user` builds are not eligible for the local checkpoint/resume operations.

Do not weaken state-machine protections to make the operation succeed.

### F-08: Workspace path semantics

Explicitly document `repo_path` as provenance metadata in B1, not the authoritative workspace identity.

B2 must establish stable workspace identity and explicit ownership/location transitions; do not make an absolute local path a permanent cross-location identity.

Add documentation and a focused test for the chosen B1 semantics.

### F-09: Manifest serialization

Replace delimiter-only serialization with an unambiguous canonical encoding, such as length-prefixed fields or a rigorously defined canonical structured representation.

Add fixtures containing newline and pipe characters in legal filenames and prove distinct states produce distinct digests.

### F-10: Git configuration independence

Pin rename-detection behavior and audit other Git configuration influences on manifest serialization.

The same repository state must produce the same manifest regardless of the user's Git configuration.

Add a configuration-skew regression test.

### F-11: Workbench listing honesty

Do not silently skip unreadable entries.

Surface an explicit skipped/error count or fail the bounded listing operation according to its contract.

Add a test proving the UI never represents an incomplete listing as complete.

## 2. Additional mandatory coverage

Add tests for:

- Concurrent checkpoint/resume requests.
- Resume racing with checkpoint creation.
- Symlink and traversal refusal.
- FIFO refusal without hanging.
- Every B1 mutating POST route requiring session and Origin protection.
- RLS, grants and privilege parity.
- Checkpoint and resume integrity after server restart.

Do not rely solely on code inspection for these invariants.

## 3. Database and complete verification

Use the approved isolated local PostgreSQL testing environment.

All databases and test roles must be run-unique and created specifically for the verification run.

Re-run:

- Migration up/down and round-trip.
- Checkpoint CRUD and resume suites.
- Concurrent sequence and single-winner tests.
- Resume concurrency tests.
- RLS behavior and privilege parity.
- `go test -p 1 -count=1 ./...`
- Targeted `-race` suites.
- `gofmt`, `go vet ./...`, and `go build ./...`.

Any database test that cannot execute must be explicitly reported as blocked, not green or skipped without explanation.

Clean up only test resources created by this task.

## 4. Operating constraints

- Investigate root causes, not symptoms.
- Do not weaken tests or security boundaries.
- Preserve B0 compatibility.
- Do not change the held Gateway assignment behavior without a separately demonstrated requirement.
- Do not redesign the checkpoint architecture beyond what the findings require.
- Preserve all pre-existing owner modifications.
- Do not execute destructive Git commands.
- Do not provision infrastructure.
- Do not start B2 or implement Sprite handoff.
- Do not commit, push, stash, reset, or change branches.

## 5. Acceptance requirements

B1 remediation is complete only when:

1. F-01, F-02 and F-03 are fixed and independently regression-tested.
2. F-04, F-06, F-07, F-09, F-10 and F-11 are fixed.
3. F-05 and F-08 are explicitly documented with the accepted limitations and B2 requirements.
4. All additional coverage gaps listed above are addressed.
5. Database-backed suites and race tests execute successfully in the authorized local environment.
6. Full Go verification passes.
7. B0 compatibility remains intact.
8. The final diff contains only scoped remediation changes and pre-existing owner work remains untouched.

## 6. Required evidence report

Update `docs/bootstrap-evidence.md` with the B1-R1 remediation results.

Produce a report mapping each finding F-01 through F-11 to:

- Original finding.
- Root-cause correction.
- Changed files.
- Regression test.
- Exact verification result.
- Residual risk, if any.

Include the complete database and full-suite outputs or durable references to them.

Clearly distinguish implemented, tested, demonstrated, and not-yet-tested.

## 7. Stop condition

Submit the complete B1-R1 report and STOP.

Do not start B2.

Remote Sprite handoff requires a separate GO and a new adversarial review of the workspace-transfer and ownership model.

**Next gate:** after the agent returns, I will review the remediation evidence and request a focused independent re-review of the corrected findings. Only then will I issue B1 final acceptance and consider B2.