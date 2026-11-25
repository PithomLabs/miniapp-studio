Use this as the next adversarial review prompt. It is deliberately scoped to **verifying the completed Phase B Part 1 correction pass**, not authorizing Part 2 or reopening the architecture.

# Adversarial Code Review — Miniapp Studio Agent Gateway, Phase B Part 1 Corrections

## Role

Act as an independent principal security engineer, senior Go systems architect, and adversarial code reviewer.

You did not implement this correction pass. Your responsibility is to challenge its claims, inspect the actual implementation, reproduce relevant failure modes, and determine whether the corrections are genuinely secure, durable, and correct.

Do not trust implementation summaries, test counts, comments, design explanations, or the previous review's conclusions without independently checking the underlying evidence.

Your objective is not to find cosmetic imperfections or generate another broad architecture review. Your objective is to determine whether the Phase B Part 1 correction pass has actually closed its assigned findings without introducing new security, reliability, concurrency, or lifecycle defects.

## 1. Governing material

Read and cross-reference these documents before reviewing the implementation:

1. `plans/adv_review_prompt10.md` — original adversarial findings.
2. `plans/prompt11.md` — governing correction directive.
3. `plans/prompt11_plan_review.md` — A1/A2/A3 amendments, especially the indeterminate-claim recovery rules.
4. `plans/prompt11_plan.md` — implementation plan and acceptance criteria.
5. `docs/prompt11-correction-report.md` — implementation report and evidence.
6. `docs/gateway-phase-b-evidence.md` — capability matrix and phase evidence.
7. `docs/gateway-phase-b-live-runbook.md` — owner-run Part 2 verification procedure.
8. `docs/opencode-run-v1.19.6.md` — pinned OpenCode source audit record.
9. The governing Miniapp Studio Implementation Agent Guidelines.

Also inspect the actual source code and tests. Relevant areas include `gateway/`, `tools/`, `server/`, `fly/`, `cmd/gateway/`, `cmd/studio/`, and their associated tests and provisioning assets.

If a referenced governing document is missing, report that fact. Do not invent its contents or silently substitute another source.

The correction report claims:

- B1 and F2–F9 corrected at root cause.
- M1–M16 all fixed, with none carried.
- G1–G8 evidence corrected.
- 349 test functions, including 99 gateway tests.
- Full suite and race tests green on PostgreSQL 19 only.
- P1–P4 adversarial probes reproduced red before correction and green afterward.
- Static Linux binary, Darwin cross-compilation, and shell syntax verified.
- No commits or pushes.
- Part 2 was not executed.

Treat all these as claims to verify, not established facts.

## 2. Review rules and engineering methodology

The governing engineering guidelines apply throughout this review.

### Root-cause investigation

For every suspected defect:

1. Identify the violated invariant and the actual trust boundary.
2. Trace the complete execution path, including the caller, validation, state mutation, persistence, process interaction, callbacks, retries, and recovery.
3. Reproduce the underlying failure where safe and practical.
4. Identify the root cause and the component that owns the invariant.
5. Check whether the correction fixes the failure at that ownership boundary.
6. Verify a regression test that would fail if the underlying defect returned.

Do not recommend a superficial workaround that merely suppresses the observed symptom.

A mutex is not automatically a durable concurrency solution. A permission bit is not proof of privilege separation. A successful fake-backed test is not proof of live infrastructure behavior. A passing happy path is not proof of crash safety.

### Threat-model discipline

For security-sensitive paths, explicitly identify:

- Trusted and untrusted actors.
- Protected assets and credentials.
- Actor × resource × operation permission boundaries.
- Attacker-controlled paths, payloads, filenames, process state, and timing.
- Expected denial behavior, including kernel-enforced `EACCES` where applicable.
- Failure behavior when a required security mechanism cannot be applied.

Assume that an agent process may be compromised or malicious. Do not rely on agent cooperation to preserve Gateway secrets or state.

### Evidence discipline

Distinguish these evidence classes:

- **Code-verified:** established by source inspection.
- **Test-verified:** exercised by an identified automated test.
- **Local-process-verified:** exercised using actual local OS processes.
- **Database-verified:** exercised against the required PostgreSQL 19 Docker database.
- **Live-verified:** demonstrated on the actual Sprite/Fly/OpenCode environment.
- **Unverified:** still an assumption or future proof requirement.

Never promote one evidence class into another.

Every finding must have concrete evidence. Do not report speculative risks as confirmed vulnerabilities. Label hypotheses clearly and explain what experiment would prove or disprove them.

## 3. Scope and safety boundaries

This is a read-only adversarial review.

**Do not modify source code, tests, plans, documentation, configuration, or provisioning files. Do not apply fixes.**

Do not:

- Create, delete, provision, or modify Sprites.
- Perform Fly API mutations.
- Start OAuth ceremonies or install anything on live infrastructure.
- Execute the Part 2 live runbook.
- Use production credentials or request credentials from the owner.
- Modify Miniapp or its repository.
- Change F1, G-01, P5, or any frozen plan.
- Add an adapter, workflow engine, framework, or unrelated feature.
- Commit, push, stage, or otherwise change Git history.
- Expand the review into Phases C, D, or E.

Part 2 remains unauthorized. Its live-only items must be evaluated as explicitly deferred proof obligations, not falsely reported as completed or automatically treated as Part 1 implementation defects.

Preserve the existing all-untracked, zero-commit working-tree boundary. Record `git status` before and after the review.

## 4. Critical review areas

### A. B1 — OS identity separation and least privilege

Inspect the complete path from service installation to agent process creation.

Verify:

- `SysProcAttr.Credential` is applied on every agent spawn.
- There is no execution path that launches an agent without the required identity drop.
- Missing UID/GID/group configuration, unsupported platforms, and insufficient privileges fail closed.
- There is no production same-UID fallback or accidental `AllowSameUID` activation.
- Supplementary group behavior is correct for the actual Go/platform implementation.
- The Gateway's effective identity and privilege requirements match the setup and service configuration.
- The agent cannot read or modify Gateway signing keys or Gateway state.
- Staging directories and temporary key copies are protected throughout installation, not just after cleanup.
- PATH and executable resolution cannot be redirected through agent-writable locations.
- Umask, group ownership, sticky-bit behavior, and directory modes work together rather than merely looking correct individually.

Independently reconstruct the permission matrix. Check both intended permissions and denied operations.

Pay particular attention to the report's residual: co-located builds share the same agent UID. Determine exactly what an agent with that identity can read, write, replace, or delete across sibling build workdirs and inboxes.

Decide whether the residual is correctly bounded to one tenant/application, whether any claimed build-level isolation is misleading, and whether it needs an explicit owner risk acceptance before live operation.

Do not require a live EACCES demonstration from this Part 1 review. The actual different-UID/key-read-denied proof remains a Part 2 live gate. Verify that the code and runbook honestly distinguish fake/local evidence from that missing proof.

### B. F2/F3 — Safe file-drop operations and confused-deputy prevention

Inspect the complete question and answer file lifecycle.

Challenge:

- `os.OpenRoot` confinement and all directory-relative operations.
- `O_NOFOLLOW`, `O_EXCL`, ownership, file type, and group checks.
- Open-then-`fstat` and descriptor-bound reading.
- Symlinks, hard links, path substitution, rename races, and concurrent creation.
- Atomic replacement and temporary-file cleanup.
- Ownership-aware `EEXIST` handling.
- Byte limits, malformed inputs, and scanner/decoder failures.
- Behavior during process restart or a crash between file creation, consumption, and deletion.
- Whether stale question or answer files can affect a later turn.
- Whether an agent can overwrite, delete, impersonate, or forge Gateway-owned answers.

Reproduce the original P2 and P3 exploits or their equivalent safe regression probes:

1. A question symlink targeting a Gateway key or other protected file must not disclose its contents through an event or callback.
2. An answer symlink targeting a victim file must not truncate or modify that victim.

Also inspect the artifact collector's descriptor-pinned implementation. Check whether it eliminates the original check-then-open TOCTOU problem under concurrent path replacement.

### C. F4 — Durable claims, concurrency, and crash recovery

This is a critical correctness boundary. Do not accept the phrase "at-most-once" without tracing all crash windows.

Verify:

- A durable claim is persisted before dispatch.
- The claim includes the correct message ID, payload digest, build/session/turn identity, and invocation identity.
- The same message ID with a different payload digest is rejected.
- Concurrent identical requests share one logical execution.
- Distinct commands for a build are serialized correctly.
- Per-message call coordination and per-build serialization have a valid lock order.
- The behavior is correct across process restarts, not merely within one process.
- `Drop` is permitted only when the implementation can prove that no child was started.
- A possibly-started child produces an indeterminate claim, never an automatic relaunch.
- Completion-persistence failure cannot silently convert a possibly executed command into a retryable one.
- Orphan reaping does not kill unrelated processes or misidentify a reused PID.
- Stable event identities and downstream idempotency prevent duplicated logical events.

Specifically challenge the hardest failure window:

1. The agent starts.
2. It emits an event.
3. The event is persisted to the outbox or otherwise becomes externally observable.
4. The Gateway crashes before recording command completion.
5. The same command is delivered again after restart.

Prove that recovery does not dispatch the same logical command a second time and does not duplicate the logical event.

Also test or inspect crashes between claim persistence, process creation, PID/session recording, event persistence, and completion persistence.

Verify the meaning of F6 recovery: deliberate continuation must be a new authorized, fenced turn. It must not silently retry or relaunch the indeterminate original command.

The correct guarantee is at-most-once dispatch with explicit recovery for ambiguous outcomes. Do not upgrade that to exactly-once execution.

### D. F5 — Terminality and process outcomes

Verify that intermediate NDJSON error messages are diagnostics/progress and do not themselves terminalize the session.

Check that terminal success/failure is decided by the actual process outcome and the recognized question-yield behavior, including:

- Error followed by successful exit.
- Error followed by nonzero exit.
- Question yield with exit zero.
- Malformed or foreign-session records.
- Overlong lines and scanner errors.
- Process termination, cancellation, and missing terminal events.
- Stderr buffering and bounded memory behavior.

Inspect the vendored OpenCode source record and determine exactly which claims are established by the excerpt and hash evidence, and which require the actual live P-G0/P-G1 proof.

Do not assume that OpenCode v1.19.6 behavior establishes the behavior of a different binary or version.

### E. F6 — Honest recovery and turn fencing

Inspect the complete failed-session continuation path.

Verify:

- Failed sessions permit only the intended authorized continuation.
- Done and cancelled sessions remain terminal.
- Continuation uses the preserved workdir and the intended session lineage.
- The adapter verifies the resumed session identity rather than accepting any process output.
- `agent.resumed` is emitted only after the resumed process is genuinely verified.
- Launch failures restore the prior failed state rather than fabricating a yielded or resumed state.
- Absent, stale, duplicate, and next-turn values are handled correctly.
- Recovery does not auto-restart an agent.
- Old commands and stale turns cannot trigger new work.

Challenge whether the claimed recovery tests prove the full server → adapter → supervisor → process → event path, rather than only a mocked state transition.

### F. F7 — Safe provisioning and explicit upgrades

Inspect provisioning and upgrade behavior under failures and concurrent operators.

Verify:

- Absent service leads to creation.
- Healthy existing service is preserved without restart.
- Existing unhealthy service and service-inspection errors fail closed without destructive replacement.
- Key refresh does not interrupt active sessions.
- Binary digest drift requires an explicit upgrade.
- Upgrade checks all builds sharing the Sprite, not just the selected build.
- Upgrade refuses while an agent is live unless the owner explicitly uses the documented force path.
- The cross-process PostgreSQL advisory lock is correctly keyed, acquired, held, and released.
- Separate CLI processes cannot race through the provision or upgrade sequence.
- Service arguments, state directory, required `studio_url`, keys directory, and binary path are complete and correctly validated.
- Provision retries converge without recreating healthy shared infrastructure.
- Digest information is actually recorded and returned as claimed.

Pay particular attention to the boot-argument defect found during implementation. Verify that the correction closes the real startup path and not merely a unit-test fixture.

### G. F8/F9 — Outbox durability, requeue, and freshness

Trace event generation through persistence, delivery, retry, acknowledgement, and operator recovery.

Verify:

- Enqueue/persist failures do not silently discard events.
- Degraded state is durably and visibly surfaced.
- Failure to persist degraded state is honestly handled and documented.
- Inspect and requeue operations are scoped to the correct tenant/build.
- Requeue checks ownership, preserves the logical message identity, and records the action.
- Acknowledged messages are not redelivered as new logical events.
- Outbox status and counts are truthful.
- The deliverer uses an explicit timeout.
- Every retry gets a fresh signed envelope timestamp while preserving stable logical identity and payload.
- The Studio freshness window is not weakened to accommodate retries.
- Forged stale envelopes remain rejected.
- HTTPS is enforced except for the explicitly allowed loopback development case.
- Error messages and logs do not leak keys, tokens, or staged secret contents.

Challenge disk-full, corrupted-store, process-crash, delayed delivery, HTTP timeout, 4xx, and retry exhaustion cases.

### H. M1–M16 — Verify every disposition

The report says all sixteen minor findings are fixed. Verify each one independently. Do not accept a collective "all fixed" assertion.

| Finding | Required verification                                                                                |
| ------- | ---------------------------------------------------------------------------------------------------- |
| M1      | Directory fsync and crash durability for every relevant atomic store.                                |
| M2      | Per-attempt delivery timeout, including a hanging endpoint.                                          |
| M3      | SIGTERM handling, process lifetime independent of request cancellation, and orderly shutdown.        |
| M4      | Process-group termination and grandchild/orphan cleanup.                                             |
| M5      | Bounded stderr ring and bounded diagnostic retention.                                                |
| M6      | Partial recovery listing, emit-before-mark ordering, deterministic IDs, and idempotent replay.       |
| M7      | Reaper identity checks using executable identity and process start time; PID reuse and decoy safety. |
| M8      | Descriptor-pinned collector behavior under symlink and path-swap attacks.                            |
| M9      | Scanner errors, malformed records, and overlong-line failures surfaced accurately.                   |
| M10     | File-drop size limits and oversized input rejection.                                                 |
| M11     | Turn fencing for absent, stale, duplicate, and next-turn inputs.                                     |
| M12     | HTTPS enforcement for both relevant transport directions, with loopback exception boundaries.        |
| M13     | CLI command allowlist, output truncation, and safe file-output handling.                             |
| M14     | Binary digest reporting and tool-result recording.                                                   |
| M15     | Actual KDF wiring and negative authentication test using the wrong secret.                           |
| M16     | Protocol header refresh, signed `turn_id` mapping, and safe short-ID handling.                       |

**Identifier ambiguity to resolve:** The correction report identifies M5 as bounded stderr handling, while `docs/gateway-phase-b-live-runbook.md` §15 still labels the SSE maximum stream lifetime backlog as "M5". Determine whether these refer to two different findings or whether the identifier has been reused incorrectly. Trace both to their original source. Do not let the claim "M1–M16 all fixed" obscure the still-listed SSE lifetime work. Recommend an unambiguous cross-reference or identifier correction if necessary.

Also verify that the runbook's SSE lifetime and concurrent-stream budget backlog is accurately classified as a remaining Phase B operational gate, not accidentally represented as implemented.

### I. G1–G8 — Evidence correction audit

Verify each correction against the report, source, and test evidence:

- G1: Test counts are exact and their counting method is clear.
- G2: File-drop directionality is proven only to the extent the evidence supports.
- G3: Live OpenCode execution is not falsely claimed.
- G4: The `run.ts` provenance and pinned-source claims are auditable.
- G5: Outbox terminal failure and recovery claims match the implementation.
- G6: Deduplication and concurrency claims match the durable-claim semantics.
- G7: Recovery and continuation claims match the actual state machine.
- G8: Fly error-output redaction claims are not overstated.

Check the evidence matrix's five states:

1. Implemented and tested.
2. Implemented, insufficiently tested.
3. Specified, not implemented.
4. Explicitly not built.
5. Live behavior not yet verified.

Ensure every capability is placed in the right state. Fake-backed proof must not be described as live proof. A documented design must not be described as implemented behavior.

Check for contradictions between the historical evidence matrix v1 and corrected matrix v2. Historical material may remain for traceability, but it must be unmistakably marked as superseded so that readers do not mistake old claims for current behavior.

### J. Miniapp application contract

Verify that the Builder instructions and assignment construction reinforce the locked application contract:

- Miniapp is the exclusive application framework.
- Go is the exclusive implementation language for Miniapp applications.
- `MiniappSpec` is the canonical application specification.
- Miniapp framework/runtime is the exclusive execution model.

Python, JavaScript/TypeScript/Node, or another framework/runtime must not become an alternative way to implement or execute a Miniapp.

Check whether the instructions are present before artifact-bearing agent turns. Do not claim the Phase D admission/validation pipeline exists if it has not been implemented. Distinguish prompt-level reinforcement from actual runtime enforcement and negative tests.

## 5. Adversarial tests and verification

Re-examine the original P1–P4 probes and their corrected results:

- P1: concurrent duplicate Continue must execute once.
- P2: question symlink must not exfiltrate protected data.
- P3: answer symlink must not clobber a victim file.
- P4: intermediate error messages must not cause premature terminal failure.

Where possible, independently rerun the relevant existing regression tests and safe local probes. If `/tmp/b1probe` is missing or no longer reproducible, reconstruct the probes from the original review documentation without modifying the repository.

Do not accept a reported red-before/green-after history solely because the correction report says it happened. Establish what can be independently verified from the current tree and retained evidence. Clearly label any historical red result that cannot be reproduced or independently substantiated.

Verify the claimed 349 test functions independently. Distinguish test functions from subtests and actual test executions. Report per-package counts and actual command results.

### Mandatory database testing standard

PostgreSQL 19 through the existing Docker environment is the only permitted database baseline.

Before database-backed tests:

1. Verify the test connection is the intended local endpoint, `127.0.0.1:5434`.
2. Verify the actual server version using `SHOW server_version` and `SELECT version()`.
3. Confirm it is PostgreSQL 19.x.
4. Confirm the test database/schema is isolated and disposable.

Never substitute PostgreSQL 16, 17, or another version. Never run destructive setup, reset, truncation, or migration commands against user data or a shared database.

If PostgreSQL 19 is unavailable, inaccessible, or cannot be isolated safely, stop database-backed verification and report the exact blocker. Do not claim the database gates passed.

Where safe, independently verify the reported gates:

```
gofmt -l .
go vet ./...
go build ./...
go test -p 1 ./...
go test -race -p 1 ./gateway ./store ./tools ./server ./cmd/studio ./cmd/gateway
CGO_ENABLED=0 GOOS=linux go build -o /tmp/studio-gateway-review ./cmd/gateway
GOOS=darwin go build ./gateway/ ./cmd/gateway/
sh -n tools/provisiondata/setup.sh
```

Use the actual repository's required environment and record exact commands, exit codes, skips, and per-package results. Do not expose DSNs, credentials, secrets, or tokens in the report.

If a gate is not safe or possible to run, explain why and classify its evidence correctly. Do not silently omit it.

## 6. Residuals and live-only proof obligations

The correction report explicitly carries these matters forward:

- Real different-UID privilege drop and key-read-denied proof.
- Dual-user inbox permissions and actual answer-file access.
- Full `serve()` boot under real service identities.
- `setup.sh` execution on a real Builder Sprite.
- Real OpenCode no-op execution and file-convention compliance.
- P-G0 transcript/NDJSON behavior.
- P-G1 live exit-code semantics.
- P-G3 live file-drop compliance.
- Warm/cold sleep-wake behavior and wake latency.
- OAuth ceremony and restricted grant verification.
- Sprite execution privilege and service identity discovery.
- Plugin behavior, egress policy, and token handling.
- R1–R10 live unknowns.

These are not Part 1 proofs. Do not claim they are complete.

Verify that every residual has an explicit owner-run runbook checkpoint, a clear expected observation, a safe stop condition, and an evidence classification. Agent self-report must never be presented as equivalent to kernel-observed privilege denial.

Do not execute these checkpoints. They require separate explicit Part 2 authorization.

## 7. Review of documentation and operational consistency

Cross-check source behavior against:

- Correction report.
- Phase B evidence matrix.
- Phase B live runbook.
- OpenCode source audit record.
- Builder instructions and setup script.

Find stale, contradictory, ambiguous, or overconfident statements. Pay particular attention to:

- Historical v1 versus corrected v2 evidence.
- M5 stderr versus M5 SSE naming.
- What is implemented versus what is only specified.
- Which tests use fakes, scripted processes, local processes, or a real database.
- Which claims require Part 2.
- The exact recovery semantics for indeterminate commands.
- The shared agent UID residual.
- The distinction between Phase B Part 1 completion and overall Phase B exit.

Documentation must describe what the code actually does, not what the design intends it to do.

## 8. Findings standard

Report only findings supported by concrete evidence.

For every finding include:

1. Finding ID and severity: Critical, High, Medium, or Low.
2. Exact source file and line/function.
3. Violated security boundary, invariant, or acceptance criterion.
4. Root cause.
5. Reproduction or concrete failure scenario.
6. Impact and realistic blast radius.
7. Whether the issue is a confirmed defect, a documentation/evidence discrepancy, a live-only unknown, or a hypothesis.
8. Minimum durable correction required.
9. A regression test that would prove the correction.

Avoid style-only comments, speculative redesigns, and preference-based recommendations.

Do not reopen previously accepted architecture unless you identify a concrete implementation defect or violated invariant. Do not invent new scope to make the review appear more comprehensive.

If an original finding is genuinely closed, say so and cite the actual implementation/test evidence. Do not manufacture a replacement concern merely to keep the review going.

## 9. Required deliverable

Produce:

`docs/adv-review-prompt11-correction.md`

The report must contain:

1. Executive verdict.
2. Scope and exact source revisions/tree state reviewed.
3. Documents and source areas inspected.
4. Independent verification of test/build/database gates.
5. Finding-by-finding closure matrix for B1, F2–F9, M1–M16, and G1–G8.
6. P1–P4 reproduction and result matrix.
7. Miniapp contract assessment.
8. Documentation/evidence consistency findings, including the M5 identifier issue.
9. Residual and Part 2 live-proof register.
10. New findings, each with concrete evidence and reproduction.
11. Explicit list of checks not run and why.
12. Final acceptance recommendation.

The closure matrix must use these statuses:

- VERIFIED CLOSED
- CLOSED — EVIDENCE LIMITED
- DOCUMENTATION CORRECTION REQUIRED
- CARRIED TO PART 2
- NOT ACCEPTED

Do not mark a finding closed merely because a test with a similar name passes.

## 10. Acceptance decision

Conclude with exactly one of these recommendations:

**ACCEPTED** — All in-scope Part 1 corrections are independently substantiated; no blocking or major defect remains; documentation accurately represents behavior; remaining live-only matters are properly carried to Part 2.

**ACCEPTED WITH BOUNDED RESIDUALS** — No in-scope blocker remains, but clearly identified non-blocking residuals or documentation corrections require explicit tracking before the relevant later gate.

**NOT ACCEPTED** — At least one in-scope security, correctness, concurrency, recovery, or lifecycle defect remains, or a central acceptance claim lacks necessary evidence.

For every non-accepted finding, provide the minimum bounded correction required. Do not turn the review into an unlimited redesign.

A Part 1 acceptance recommendation is **not** authorization for Part 2. The final report must state explicitly:

- No Sprite/Fly/OAuth/live installation work was performed.
- Part 2 remains unauthorized.
- No code, docs, or configuration were modified by the reviewer.
- No commits or pushes were made.

Stop after producing the review report. Do not implement fixes or proceed to the next phase.

**My recommendation:** have the reviewer focus especially hard on F4's crash windows, the actual OS permission boundary in B1, cross-process provisioning locks in F7, and the M5 identifier inconsistency. Those are high-value places to challenge the completion claims without reopening the entire architecture.