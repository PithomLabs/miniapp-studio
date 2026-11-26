GO — B1 IMPLEMENTATION AUTHORIZED

B1 is the local development loop: checkpoints, progress tracking, interruption recovery, and minimal Git awareness. Local-only execution; no cloud infrastructure.

Give the coding agent this directive.

# B1 — Implement Miniapp Studio's Local Development Loop

## Role

Act as the Senior Go Engineer implementing Bootstrap 1 of Miniapp Studio under the direction of the Technical Director.

Read the repository-root `SKILLS.md` before starting. It is the governing engineering constitution.

Follow the latest approved, renumbered implementation plan:

`docs/architecture/unified-lifecycle-ui-implementation-plan.md`

Bootstrap numbering is binding:

- B0: Studio manages one development task — ACCEPTED.
- B1: Studio supports its own local development loop — CURRENT AUTHORIZED TASK.
- B2: Remote development and Fly Sprite handoff — NOT AUTHORIZED.
- B3: Application Preview — NOT AUTHORIZED.
- B4: Troubleshooting and real logs — NOT AUTHORIZED.
- B5: Staging and Production releases — NOT AUTHORIZED.

Do not use obsolete B1–B6 numbering from earlier plan revisions.

## 1. Decision brief — binding execution instructions

| Field                  | Decision                                                                                                                                                                                |
| ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Objective              | Make Studio capable of coordinating and recovering its own local development work.                                                                                                      |
| Scope                  | Durable checkpoints, progress tracking, interruption recovery, minimal Git awareness, and the narrowly scoped local workbench/context-bar functionality specified for B1.               |
| Execution location     | Local Desktop only. Use the existing local coding-agent workflow (Muse).                                                                                                                |
| Application target     | Local Miniapp Studio only. No Preview, Staging, or Production runtime.                                                                                                                  |
| Database target        | Local PostgreSQL only. Use isolated, run-unique test databases and test roles.                                                                                                          |
| Infrastructure actions | None. Do not invoke Fly or Neon mutation commands. Do not provision Sprites, Machines, or Neon branches.                                                                                |
| Authorization          | Local implementation, local migrations, and local tests are authorized. No commits, pushes, cloud changes, or destructive operations.                                                   |
| Dependencies           | B0 accepted; root SKILLS.md; approved lifecycle plan; existing Phase A–D implementation and B0 evidence.                                                                                |
| Verification           | Checkpoint round-trip, interruption/resume, Git manifest accuracy, tenant isolation, security tests, and complete Go verification.                                                      |
| Recovery               | Preserve dirty working-tree state, use the latest complete checkpoint, explicitly identify interrupted attempts, and refuse unsafe resume when checkpoint and workspace state disagree. |

This is an implementation authorization for B1 only. It does not authorize B2 or any later milestone.

## 2. Goal

B0 established that Studio can manage one development task:

Describe → execution packet → manual Muse execution → recorded attempt → evidence → human decision.

B1 must turn that initial capability into a durable local development loop.

Studio must now be able to:

1. Track meaningful progress during an execution attempt.
2. Persist recoverable checkpoints.
3. Resume interrupted work from the most recent complete checkpoint.
4. Reconstruct the necessary agent context from Studio's durable records.
5. Record the Git state associated with checkpoints.
6. Show the user what happened, what remains unfinished, and what can safely be resumed.

The core invariant:

**An interrupted development attempt must never silently disappear, and Studio must never claim that work has been resumed from state it cannot actually recover.**

## 3. Investigate before implementing

Inspect the actual repository before making changes.

Read:

- `SKILLS.md`
- `docs/architecture/unified-lifecycle-ui-implementation-plan.md`
- `docs/bootstrap-evidence.md`
- `docs/local-agent-bridge.md`
- Existing Focus Console references in `plans/plan_ui.html` and `plans/plan_ui_z.md`
- The B0 implementation and migration 000023
- Existing conversation, build, turn, message, event, stream, artifact, and approval models
- Gateway sender, command delivery, reconciliation, and existing held-assignment behavior
- Current PostgreSQL migration and test-role conventions

Inspect `git status` and the existing diffs before editing.

Preserve all pre-existing user modifications, concurrent agent work, and unrelated files.

Do not infer architecture from documentation alone. Trace the actual code paths and tests.

In particular, understand the B0 manual-build behavior and its held Gateway `assign` row. Determine whether B1 needs to change any execution or dispatch behavior. Do not burn attempts, dispatch commands, or send network requests merely to make the held row disappear.

Deliver a short implementation approach before making broad changes. If a structural conflict with the approved plan or constitution appears, stop and report it rather than silently redesigning the architecture.

## 4. Required implementation scope

### 4.1 Durable checkpoints

Introduce the smallest durable checkpoint capability needed for local development recovery.

The existing PostgreSQL database remains the authoritative journal.

Reuse existing builds, turns, messages, events, attempts, and artifact/evidence references wherever appropriate. Do not introduce a second task engine or duplicate conversation history.

The plan anticipates a `checkpoints` table. Inspect the actual migration chain after B0 before choosing the next migration number. Migration 000024 is the expected next number only if the repository confirms it remains available.

The checkpoint model must support, as appropriate:

- Tenant and project/application identity.
- Build and execution-attempt identity.
- Associated turn or task context.
- Checkpoint version and creation time.
- Git branch and commit references.
- Staged, unstaged, and untracked worktree manifest references or digest.
- Bounded, sanitized agent/execution context.
- References to relevant messages, events, test outputs, and evidence.
- Explicit complete/incomplete state.
- A recoverable ordering or sequence.

Do not store source-code blobs or entire repository snapshots in PostgreSQL.

Do not persist raw credentials, environment secrets, unrestricted terminal transcripts, or unlimited agent conversations.

Checkpoint payloads must be versioned, size-bounded, validated, and secret-scrubbed before persistence.

A checkpoint must represent a completed durable recovery point. An interrupted or partially written checkpoint must never be treated as the latest recoverable checkpoint.

Enforce tenant ownership and integrity through the database and server, not only in tool handlers.

Extend RLS, grants, test-role parity, and migration down/up verification as required by the existing repository conventions.

### 4.2 Checkpoint write/read APIs

Provide the minimal internal/tool/API interfaces necessary to:

- Create a checkpoint for an authorized active attempt.
- Retrieve the latest complete checkpoint for a build/attempt.
- Retrieve checkpoint history.
- Validate whether the current execution context can safely resume from a checkpoint.
- Record a resume operation and link it to its source checkpoint.

Checkpoint creation must be idempotent where retries can occur.

Concurrent writes must not produce ambiguous ordering or overwrite a newer checkpoint with stale state.

A checkpoint must be associated with the correct tenant, build, attempt, and task context.

Prefer atomic persistence of checkpoint state and its corresponding durable event/receipt.

### 4.3 Progress tracking

Make meaningful execution progress visible in Studio.

Reuse the existing event and stream/SSE architecture.

Do not introduce a second progress system or a separate task engine.

Progress should be associated with the correct build and attempt and should survive Studio server restarts.

At minimum, distinguish:

- Attempt started.
- Progress/checkpoint recorded.
- Verification or test activity recorded.
- Attempt interrupted.
- Resume requested.
- Resume completed or reconstructed.
- Attempt completed or awaiting human decision.

Keep progress event payloads structured, bounded, and secret-scrubbed.

Do not display fabricated percentages or infer completion from an agent's prose.

The UI must distinguish durable progress receipts from unverified agent-reported status.

### 4.4 Interruption and resume

Implement a credible local recovery workflow.

Simulate interruption during an active development attempt, then resume from the latest complete checkpoint.

The workflow must:

1. Identify the interrupted attempt.
2. Preserve its history and partial evidence.
3. Locate the latest complete checkpoint.
4. Validate its build, attempt, tenant, and Git/worktree references.
5. Detect whether the current local working tree is compatible with that checkpoint.
6. Construct a recoverable execution packet containing the previous task context and checkpoint references.
7. Allow the local operator to resume execution through the existing Muse bridge.
8. Record the resumed execution as a continuation linked to its previous attempt/checkpoint.

Do not promise that Muse's live process or in-memory agent session survives termination.

Resume means reconstructing the necessary working context from Studio's durable journal and the existing local checkout, and then re-invoking the agent where required.

The UI and records must honestly distinguish:

- Native session resumed, if genuinely supported and verified.
- Execution context reconstructed from a checkpoint.
- Resume refused because the state is incomplete, stale, or inconsistent.

A checkpoint is not a substitute for Git or a filesystem snapshot.

**Important B1 limitation:** the local checkout remains the source of actual source files. B1 does not implement workspace snapshots, remote transfer, or disaster recovery for a lost local filesystem. Those concerns belong to B2.

If the worktree changed after the checkpoint, do not silently overwrite or reset it. Report the discrepancy and require explicit reconciliation.

### 4.5 Minimal Git awareness

Add enough Git awareness to establish the source-code state associated with an attempt and checkpoint.

At minimum capture:

- Current branch.
- HEAD commit SHA.
- Whether the worktree is clean or dirty.
- Staged change state.
- Unstaged change state.
- Untracked file state.
- A deterministic manifest/digest associated with the checkpoint.

Git remains the authoritative source for source-code history.

Do not duplicate Git commit history in PostgreSQL or store file contents in the checkpoint table.

The manifest must accurately describe the Git state at checkpoint time. Do not generate a manifest from stale browser data or from the agent's unsupported claims.

Use safe, bounded command invocation and parse Git's machine-readable output rather than depending on human-formatted terminal output.

Handle ordinary staged, unstaged, untracked, deleted, and renamed-file states.

Never execute `git reset`, `git clean`, checkout-discard operations, or other destructive Git commands to manufacture a clean state.

Display enough information in the existing context bar and the planned minimal workbench slice for an operator to understand:

- Branch and commit.
- Dirty/clean status.
- Whether the displayed state is current or stale.
- Last checkpoint identity/time.
- Whether the workspace can safely resume.

### 4.6 Minimal UI integration

Extend the existing Focus Console. Do not redesign it.

The B1 scope is deliberately narrow:

- Show current branch, commit and dirty state in the existing project context area.
- Show checkpoint identity and last checkpoint time.
- Show progress and interruption/resume events in Pulse.
- Show relevant attempt/checkpoint history in the existing Run/Evidence surfaces.
- Provide the minimal read-only Git/workspace information needed for the local workflow.
- Provide checkpoint/resume actions only where the server can enforce their authorization and state preconditions.
- Show honest stale, unavailable, interrupted, and unsafe-resume states.

A small read-only file/worktree listing, Git panel, and test/build output view may be introduced if the approved plan requires them.

The full Developer Workbench is not part of B1.

Do not introduce an embedded terminal. Continue delegating terminal use to the owner's local terminal.

OPS/DEV density, the existing seven-stage spine, accessibility, and tenant-specific state must remain intact.

Navigation must not trigger checkpoint writes, resumes, or any other mutation.

## 5. Explicit non-goals

Do not implement:

- Fly Sprite provisioning or handoff.
- Remote Builder execution.
- Workspace snapshot transfer.
- Live agent process migration.
- Provider-native session migration guarantees.
- Fly Machines or application deployment.
- Preview, Staging, or Production.
- Neon branches or Neon CLI integration.
- GitHub repositories, pull requests, or Actions.
- Automatic coding-agent launch.
- Multi-agent/model registry.
- Full Developer Workbench.
- Embedded terminal.
- Metrics or distributed tracing.
- B2 or later roadmap functionality.

Do not modify the immutable artifact/build-to-runtime-image pipeline.

Do not redesign the B0 packet or attempt workflow unless an actual compatibility defect requires a narrowly scoped, backward-compatible correction.

## 6. Security and database requirements

Follow the existing security architecture.

- Server-side tenant identity is authoritative.
- Enforce RLS on every new table.
- Include forced RLS, grants, and test-role parity.
- Validate build, attempt, checkpoint, and tenant relationships.
- Keep operator-only actions outside the agent tool registry unless explicitly required and approved.
- Do not let an agent authorize its own work.
- Scrub checkpoint and progress payloads before writing them.
- Bound payload size and reject malformed or unsupported checkpoint versions.
- Treat Git paths, agent context, and command output as untrusted data.
- Do not expose credentials in UI, logs, test output, or evidence.
- Do not rotate shared PostgreSQL roles or credentials.

Testing may mutate only test databases and roles created uniquely for that test run.

Never use an existing owner's database, shared role, or running server's credentials as a disposable test fixture.

A test must not truncate, reset, drop, or otherwise mutate a database it did not create.

Respect the established PostgreSQL endpoint allowlist and the existing role-separation protections.

## 7. Required tests and verification

Write regression tests that demonstrate the underlying invariants, not merely the happy-path implementation.

At minimum:

### Checkpoint persistence

- Checkpoint create/read round trip.
- Latest-complete-checkpoint selection.
- Version validation and malformed-payload refusal.
- Size-bound enforcement.
- Secret-scrubbing tests with recognizable secret canaries.
- Tenant isolation and cross-tenant refusal.
- Atomic checkpoint/event persistence.
- Retry/idempotency behavior.

### Interruption and recovery

- Create an attempt and checkpoint.
- Interrupt/kill the active execution or coordinating process.
- Restart/reconstruct from durable Studio records.
- Resume using the same local checkout.
- Verify no task context is lost.
- Verify the interrupted attempt remains visible.
- Verify the resumed attempt links to the correct checkpoint and prior attempt.
- Verify incomplete checkpoints are not selected.
- Verify stale or mismatched Git state is refused or explicitly reconciled.
- Verify retries do not duplicate progress events or corrupt checkpoint ordering.

Do not claim that an agent's live reasoning was resumed unless that behavior was actually demonstrated.

### Git manifest

Test clean, staged, unstaged, untracked, deleted, and renamed changes.

Verify branch, HEAD SHA, status and deterministic manifest/digest against actual Git state at checkpoint time.

Prove that the implementation does not modify the worktree while inspecting it.

### Progress and UI

- Progress survives Studio server restart.
- Attempt/checkpoint history renders from durable state.
- Pulse reflects actual persisted events.
- Stale and unavailable Git states are honest.
- Resume controls enforce server-side preconditions.
- Tenant isolation applies to all displayed information.
- Existing OPS/DEV and accessibility behavior remains intact.

### Full verification

Run and record exact outcomes for:

- `gofmt`
- Relevant focused package tests
- Migration up/down and schema checks
- RLS and privilege parity tests
- `go vet ./...`
- `go build ./...`
- `go test -p 1 -count=1 ./...`
- Relevant `-race` tests, especially for checkpoint sequencing, event persistence, and concurrent resume/write behavior

Follow the repository's safe local test-database conventions. Do not run the suite against a database being used by another process or agent.

Any unexplained red test is a failed gate. Do not suppress, skip, weaken, or relabel a failing test to obtain a green result.

## 8. Migration and schema discipline

B1 is expected to introduce a checkpoint table, but inspect the actual migration chain before choosing the migration number.

If no migration has superseded 000024, use that number. Otherwise select the next available migration and explain why.

The migration must:

- Follow the established naming convention.
- Include a paired down migration.
- Follow composite tenant foreign-key conventions.
- Enable and force RLS.
- Define least-privilege grants.
- Extend the test-role parity model.
- Include migration round-trip tests.
- Avoid unrelated schema changes.

Do not introduce speculative workspace, handoff, environment, release, telemetry, or agent-registry tables.

Do not create a generalized checkpoint platform for hypothetical future use cases. Implement the smallest durable checkpoint contract that supports the B1 local workflow and can be extended later for B2.

## 9. Existing B0 behavior must remain compatible

Preserve the accepted B0 behavior:

- Versioned `build.packet` remains compatible.
- `RecordAttempt` remains idempotent.
- Attempt numbering remains server/database-derived.
- Parent linkage remains correct.
- Human accept/reject/rework semantics remain intact.
- Tenant isolation and secret scrubbing remain intact.
- The held Gateway `assign` row remains non-burning and must not cause network dispatch for manual local builds.

If B1 requires changes to execution-location-based dispatch, investigate the existing sender state machine and prove the intended behavior with regression tests. Do not add automatic agent launch or silently change the B0 manual bridge.

## 10. Operating sequence

Execute in this order:

1. Read `SKILLS.md` and the current approved plan.
2. Capture and preserve Git status and existing diffs.
3. Inspect B0 code, tests, schema, Gateway behavior, and current migration chain.
4. Produce the concise B1 implementation approach and decision brief.
5. Identify the smallest schema/API/UI changes.
6. Implement checkpoint persistence and security boundaries.
7. Implement interruption/resume and Git manifest behavior.
8. Integrate progress and minimal Focus Console UI.
9. Add regression, recovery, concurrency, and tenant tests.
10. Run the complete verification suite.
11. Inspect all diffs and database changes.
12. Produce the B1 evidence report and stop.

If work is interrupted, leave a clear report of the last verified state. Do not claim B1 completion until every acceptance criterion passes.

## 11. Acceptance criteria — all required

B1 is complete only when all of the following have evidence:

**A. Checkpoint durability:** A checkpoint can be written, read, validated, and retrieved as the latest complete checkpoint for the correct attempt and tenant.

**B. Interruption recovery:** An interrupted local attempt can be reconstructed and resumed from the latest complete checkpoint using its existing local checkout, without losing task context or silently dropping its attempt history.

**C. Git correctness:** The branch, commit, clean/dirty state, and staged/unstaged/untracked manifest match actual Git state at checkpoint time.

**D. Progress visibility:** Meaningful progress and checkpoint events survive process restart and appear in the existing Studio workflow.

**E. Security:** Cross-tenant access is refused; checkpoint and progress payloads are bounded and scrubbed; RLS, grants, and test-role parity pass.

**F. B0 compatibility:** The existing B0 task packet, attempt recording, human decision, and held-assignment behavior remain correct.

**G. Full verification:** Formatting, vet, build, migrations, relevant race tests, and the complete Go suite pass.

**H. UI integrity:** Focus Console stage structure, OPS/DEV parity, and honest loading/stale/error states remain intact.

**I. Repository integrity:** All pre-existing user work is preserved; only B1-scoped files are modified.

## 12. Required completion report

When finished, report:

1. What changed and why.
2. Files and migration numbers changed.
3. The implemented checkpoint and resume contract.
4. How progress and Git state are represented.
5. Exact tests/commands and outcomes, including anything not run.
6. Evidence for acceptance criteria A–I.
7. Any remaining residuals or unresolved decisions.
8. Current Git status and confirmation that unrelated changes were preserved.
9. Confirmation that no Fly, Sprite, Machine, Neon, GitHub, commit, or push operation occurred.

Distinguish implemented, tested, demonstrated-with-evidence, and not-yet-tested capabilities.

Do not present an agent-generated assertion as proof without the corresponding test, database record, or inspectable receipt.

## 13. Stop condition

After submitting the B1 implementation and evidence report:

**STOP. Do not begin B2.**

B2 requires a separate architectural review and explicit authorization for Remote Builder Sprite handoff. B1 completion does not automatically authorize cloud infrastructure operations.

**Execution boundary:** Local Desktop + local PostgreSQL only. B1 is authorized; B2 Sprite work remains gated on B1 review and a separate GO.