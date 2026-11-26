# B1 — Scoped Adversarial Security and Correctness Review

## 1. Role

Act as an independent Principal Security Engineer and Adversarial Go Code Reviewer for Miniapp Studio.

Your responsibility is to challenge the B1 implementation, find exploitable weaknesses and incorrect assumptions, and determine whether its checkpoint, Git manifest, progress, and recovery mechanisms are safe enough to accept.

You are not the implementation agent. Do not defend the implementation or assume the completion report proves correctness.

The goal is to establish whether the system preserves its architectural invariants under hostile inputs, concurrent operations, process interruption, and filesystem changes.

Read the repository-root `SKILLS.md` and the approved lifecycle plan before beginning.

## 2. Review scope

Review the completed B1 implementation and its interaction with the previously accepted B0 workflow.

Primary files and areas include, but are not limited to:

- `server/gitworktree.go`
- `server/build_checkpoint.go`
- `server/build_attempt.go`
- Checkpoint migration `000024` and its down migration
- Checkpoint and Git worktree test files
- Server configuration and workspace-root handling
- Console handlers, routes, templates and JavaScript
- Pulse, stream and Evidence integration
- Checkpoint and resume runbook documentation
- B0 `build.packet` and `RecordAttempt` compatibility
- PostgreSQL RLS, grants, tenant context and test-role parity
- Existing Gateway sender behavior and held assignment handling

Trace actual call paths from HTTP/tool entry points through authorization, validation, filesystem access, transaction boundaries, persistence and response rendering.

Inspect the complete relevant implementation, not just changed lines or test names.

The B1 completion report is a statement of claims to verify, not evidence by itself.

## 3. Binding execution decision brief

| Field | Decision |
|---|---|
| Objective | Independently evaluate B1 security, correctness, recovery safety and architectural compliance. |
| Execution location | Local Desktop only. |
| Application target | Local Studio only; no remote or deployed application. |
| Database target | Local PostgreSQL only, using uniquely created disposable test databases and roles if database tests are necessary. |
| Infrastructure actions | None. No Fly, Sprite, Machine, Neon, GitHub or other cloud operations. |
| Authorization class | Read-only inspection and safe local verification. No implementation, fixes, migrations against shared databases, commits or pushes. |
| Verification | Existing tests, targeted tests where safe, temporary adversarial fixtures, and reproducible evidence for every finding. |
| Recovery | Do not alter the existing working tree or user data. Clean up only temporary files and test resources created by this review. |

Do not change execution permissions or assume unrestricted shell access authorizes broader operations.

## 4. Review area A — Git filesystem security

This is a priority review area.

Inspect whether server-side Git inspection is safe when repository contents and filenames are adversarial.

Investigate:

### Path traversal

- Paths containing `../`, absolute paths, alternate separators and encoded traversal.
- Paths escaping the configured workspace root.
- Path normalization differences across operating systems.
- Git paths containing unusual bytes, control characters or newline characters.
- Paths that collide after normalization.
- Malicious filenames designed to confuse manifest parsing.

### Symlinks

- Symlinks pointing outside the workspace.
- Symlink chains and loops.
- Symlink replacement between discovery and hashing.
- Symlinked directories and files.
- Git symlink entries versus filesystem symlink targets.
- Whether the implementation hashes the intended object or follows a link into another location.

Determine whether workspace-root confinement is genuinely enforced or merely checked once before filesystem access.

### Filesystem race conditions

Investigate time-of-check/time-of-use gaps between:

1. Git status or file discovery.
2. Path validation.
3. File opening.
4. Content hashing.
5. Manifest persistence.
6. Resume validation.

Determine whether an attacker or concurrent local process could replace a file, directory, or symlink after validation but before hashing or use.

Do not accept a safe-looking initial path check as proof of race safety.

### Resource exhaustion

Evaluate:

- Maximum file size.
- Total hashed bytes.
- Maximum entry count.
- Deep directory structures.
- Huge filenames.
- Large numbers of untracked files.
- Unusual Git status output.
- Hashing cancellation and time limits.

Verify that limits fail closed and cannot be bypassed by directory collapsing, ignored-file behavior, malformed output or alternate Git states.

The implementation reports limits of 4 MiB per file, 64 MiB total and 5,000 entries. Verify these against actual code and tests.

## 5. Review area B — Git manifest integrity

The manifest must represent actual repository state, not a simplified approximation of `git status`.

Verify that it accurately handles:

- Clean worktrees.
- Staged modifications.
- Unstaged modifications.
- Staged and unstaged changes to the same file.
- Untracked files.
- Deleted files.
- Renamed files.
- Copies where relevant.
- File mode and executable-bit changes.
- Symlinks.
- Unmerged/conflicted states.
- Unborn HEAD.
- Detached HEAD.
- Branch changes.
- Content changes that leave Git status unchanged.

Challenge the content-sensitive digest.

In particular, independently demonstrate that changing file content while retaining the same general Git status produces a different manifest digest.

Verify that staged and unstaged content are represented accurately and cannot be confused.

Verify that the manifest is deterministic for an unchanged worktree.

Verify that hashing the worktree does not modify it, its index, Git metadata, timestamps, or user files.

Check whether the implementation can distinguish:

- Git index content.
- Worktree content.
- Untracked file content.
- Renamed-path relationships.

The reported use of staged blob SHAs, worktree SHA-256 digests, file modes and rename pairs must be verified in code.

Identify any state that cannot be represented reliably and determine whether the system refuses it safely.

## 6. Review area C — Checkpoint integrity and tampering

Review checkpoint creation, retrieval, validation and persistence.

Challenge the following invariants:

- Checkpoints are tenant-bound.
- Checkpoints are build- and attempt-bound.
- Checkpoints have validated versions.
- Incomplete checkpoints are never selected for resume.
- Latest-complete selection is deterministic.
- Sequence numbers are unique and correctly ordered.
- Idempotency keys cannot create duplicate logical checkpoints.
- Stale writes cannot overwrite newer recovery state.
- Checkpoint payloads are bounded and scrubbed.
- Checkpoint and event/receipt persistence is atomic where required.

Attempt to identify whether a caller could:

- Submit a checkpoint belonging to another tenant or build.
- Replay an old operation to alter newer state.
- Mark an incomplete or tampered checkpoint complete.
- Change the referenced Git state without changing the associated digest.
- Supply mismatched build, attempt, turn and workspace references.
- Resume from a superseded checkpoint.
- Cause a stale checkpoint to become the latest selected checkpoint.
- Exploit malformed JSON or unsupported versions.

Determine whether server-side validation is authoritative or whether the client can influence critical checkpoint state.

Inspect the database constraints and transactions, not just application-level checks.

## 7. Review area D — Resume safety and recovery

Adversarially test the resume state machine.

The central invariant is:

**Studio must never claim that it can safely resume from a workspace state it has not actually validated.**

Review:

- Interrupted attempt identification.
- Latest-complete checkpoint selection.
- Build and attempt relationship validation.
- Tenant and ownership validation.
- Branch and HEAD validation.
- Content-sensitive manifest comparison.
- Stale worktree refusal.
- Explicit reconciliation requirements and notes.
- Parent attempt/checkpoint linkage.
- Resume operation idempotency.
- Process restart and recovery behavior.

Try to demonstrate:

- Resume from an incomplete checkpoint.
- Resume from a superseded checkpoint.
- Resume after worktree content changes.
- Resume after branch or HEAD changes.
- Resume with untracked files modified or removed.
- Resume with staged versus unstaged state changed.
- Two concurrent resume requests.
- Resume racing with checkpoint creation.
- Resume retry after a partial response or process interruption.

Verify that the system never silently resets, cleans, overwrites, checks out, or discards user changes.

The permitted resume claims are:

- Reconstructed from durable context.
- Refused because the state is unsafe or inconsistent.

Do not accept claims of native agent-session resumption unless independently demonstrated.

## 8. Review area E — Concurrency and ordering

Inspect transaction boundaries and locking behavior.

Challenge:

- Concurrent checkpoint creation.
- Duplicate operation keys.
- Concurrent interrupt and resume.
- Concurrent resume attempts.
- Concurrent progress event persistence.
- Checkpoint sequence allocation.
- Stale checkpoint writes.
- Process crash between database operations.
- Retry after commit but before response delivery.
- Duplicate or missing Pulse events.

Verify that the build-row serialization strategy actually prevents inconsistent ordering.

Verify that a single logical operation produces at most one durable result when retried with the same operation key.

Run focused race tests where safe.

Do not rely solely on the race detector. Database-level concurrency and state-machine correctness require their own tests.

## 9. Review area F — Secret leakage and untrusted data

Inspect every path by which data enters checkpoint, event, log, response and UI records.

Investigate possible leakage through:

- Agent context.
- Task messages.
- Git remote URLs.
- Git configuration and commit metadata.
- File paths and filenames.
- Command output.
- Checkpoint JSON.
- Error messages.
- Resume packets.
- Pulse event text.
- DEV-mode raw JSON.
- HTTP responses.

Use recognizable synthetic secret canaries to test scrubbing.

Determine whether scrubbing occurs before persistence, not merely before rendering.

Check whether attackers can evade redaction using encoding, formatting, multiline content, unusual separators or nested JSON.

Do not assume Git metadata is safe simply because it is not source code.

Secrets must never enter a coding agent's context through a checkpoint or evidence bundle.

## 10. Review area G — Authorization and tenant isolation

Verify authorization at every entry point.

Inspect:

- Session checks.
- Origin validation.
- Operation-key handling.
- Server-side tenant resolution.
- Build and attempt ownership validation.
- Checkpoint ownership checks.
- Operator-only action separation.
- Agent tool registration boundaries.
- PostgreSQL RLS and forced RLS.
- Grants and test-role parity.

Attempt cross-tenant access using mismatched identifiers, nested references and direct request manipulation.

Verify that database constraints and RLS protect against accidental or malicious cross-tenant checkpoint references.

A UI-hidden action or client-side identifier check is not an authorization boundary.

Agents must not be able to authorize their own decisions or gain unrestricted filesystem access through checkpoint APIs.

## 11. Review area H — B0 compatibility and architectural compliance

Verify that B1 preserves accepted B0 behavior:

- `build.packet` version and contract.
- `RecordAttempt` operation-key idempotency.
- Database-derived attempt numbering.
- Parent linkage.
- Accept/reject/rework semantics.
- Secret scrubbing.
- Tenant isolation.
- Manual Muse bridge behavior.
- Held Gateway `assign` row remains non-burning.
- No unintended sender dispatch or network activity for manual local builds.

Check for accidental introduction of:

- A second task engine.
- A second progress system.
- A second event store.
- A duplicate conversation history.
- A second artifact or approval pipeline.
- An embedded unrestricted shell.

Confirm that Studio's durable database remains authoritative and that local workspace files remain execution state.

## 12. Review method

Use this sequence:

1. Read `SKILLS.md`, the approved lifecycle plan, the B1 implementation prompt, B0/B1 evidence and runbooks.
2. Inspect the actual B1 diff and complete relevant call paths.
3. Identify trust boundaries and state invariants.
4. Design adversarial scenarios for each review area.
5. Execute safe, reproducible local tests or temporary fixtures.
6. Inspect database constraints and actual stored records.
7. Verify existing test coverage and identify coverage gaps.
8. Record findings with precise file and line references.
9. Produce a verdict and recommended remediation priorities.

Do not modify source code, tests, migrations, configuration or documentation.

You may use temporary fixtures outside the repository and disposable databases/roles created specifically for this review. Do not modify shared databases or user-owned state.

If a safe test cannot be executed, identify it as not tested. Do not claim it passed.

## 13. Finding classification

Use these severity levels:

- **Critical:** Cross-tenant compromise, secret disclosure, arbitrary filesystem access, destructive workspace modification, or a bypass of a fundamental approval/security boundary.
- **High:** Unsafe resume, corrupted checkpoint history, exploitable filesystem race, incorrect manifest acceptance, or concurrency failure causing lost or misattributed work.
- **Medium:** Incorrect state representation, incomplete coverage of ordinary Git states, recoverability weaknesses, or meaningful reliability defects.
- **Low:** Limited usability, diagnostics, documentation, or non-security correctness issues.
- **Informational:** Observations or improvement opportunities without demonstrated defects.

Each finding must contain:

1. Finding ID and severity.
2. Affected invariant.
3. Exact file and line references.
4. Actual root cause.
5. Concrete attack or failure scenario.
6. Reproduction steps or test evidence.
7. Actual impact and affected boundary.
8. Recommended durable remediation.
9. Regression test required to prevent recurrence.

Separate confirmed defects from concerns that could not be demonstrated.

Do not inflate severity, report stylistic preferences as security findings, or count the same root cause multiple times.

## 14. Required deliverable

Create a review report at:

`docs/reviews/b1-adversarial-review.md`

This is the only permitted repository change, and only if the review findings need a durable report. Otherwise, provide the report in the agent's completion response without creating a file.

The report must contain:

1. Executive verdict.
2. Scope and files inspected.
3. Review method and tests executed.
4. Findings by severity.
5. Detailed Git/filesystem security findings.
6. Checkpoint and resume findings.
7. Concurrency and persistence findings.
8. Secret-scrubbing and tenant-isolation findings.
9. B0 compatibility assessment.
10. Coverage gaps and untested scenarios.
11. Prioritized remediation recommendations.
12. Final B1 acceptance recommendation.

Explicitly classify each review area as:

- Pass — inspected and tested, no defect found within scope.
- Findings — defects demonstrated.
- Not tested — insufficient evidence or unsafe to test.

A "Pass" must include the evidence supporting it.

## 15. Strict execution boundaries

This is an adversarial review, not a remediation assignment.

- Do not fix findings.
- Do not implement follow-up work.
- Do not change source or tests.
- Do not modify migrations or database schema.
- Do not perform cloud operations.
- Do not provision or mutate Fly Sprites or Machines.
- Do not access or mutate Neon branches.
- Do not use GitHub operations.
- Do not commit, push, stash, reset, clean or change branches.
- Do not overwrite or remove any existing user work.
- Do not start or stop unrelated processes.
- Do not run tests against shared databases.

The current B1 implementation is under review. B2 has not started and is not authorized.

## 16. Final decision

Conclude with exactly one recommendation:

- **ACCEPT B1:** No Critical or High findings; all required invariants have sufficient demonstrated evidence; Medium/Low residuals, if any, are explicitly bounded.
- **CONDITIONAL ACCEPTANCE:** No Critical or High findings, but specific bounded deficiencies must be resolved or explicitly accepted before B2.
- **REJECT B1:** Any Critical or High finding, or evidence that a fundamental security, integrity, tenant-isolation, or recovery invariant does not hold.

Do not grant B2 authorization. That decision belongs to the Project Owner after the Technical Director reviews the adversarial findings.

Stop after delivering the report and your verdict.