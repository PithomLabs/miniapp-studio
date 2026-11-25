The review should be particularly rigorous about the new file-drop Q&A mechanism, durable outbox recovery, command deduplication, and the distinction between fake-backed evidence and actual OpenCode/Sprite proofs.

Here is the review prompt.

# Adversarial Code Review — Miniapp Studio Agent Gateway, Phase B Part 1

## 1. Role and objective

Act as a principal software architect and adversarial security/reliability engineer reviewing the Phase B Part 1 implementation of Miniapp Studio.

Your responsibility is to determine whether the fake-backed implementation is architecturally sound, secure, durable, and genuinely ready for the separately authorized live installation phase.

Do not act as an implementation assistant. Do not modify code.

This is a read-only adversarial review. Inspect the actual source code, tests, configuration, scripts, evidence, and runbook. Do not accept the implementer's summary as proof.

**Primary question: Can this implementation safely become the foundation for a real Agent Gateway running inside a tenant's Builder Sprite, supervising OpenCode and communicating reliably with Miniapp Studio?**

## 2. Project context and locked architectural invariants

Miniapp Studio is a multi-tenant SaaS platform.

Its architectural ownership boundaries are:

- **Miniapp Studio:** authoritative control plane and permanent business conversation.
- **Miniapp framework:** framework execution, policy, tenant context, tool invocation, and audit boundaries.
- **Builder Sprite:** isolated, persistent development environment containing the checkout, coding agent, and Agent Gateway.
- **Agent Gateway:** supervised execution and protocol translation between Studio and the coding agent.
- **OpenCode:** initial coding-agent implementation, replaceable through the adapter contract.
- **Runtime Sprite:** separate environment containing only the approved running application.
- **Fly Connectors:** controlled provider-credential and integration access.

The following invariants are binding:

1. Studio PostgreSQL is the authoritative record of conversations, builds, commands, events, questions, approvals, artifacts, and deployment history.
2. Builder-local files and agent session files are execution state, not authoritative business records.
3. Builder and Runtime environments are separate.
4. The user communicates with Studio, never directly with a Sprite terminal.
5. Studio and the Gateway communicate using authenticated HTTPS commands and callbacks. No permanent Studio-to-Sprite socket.
6. The Gateway runs as a persistent Sprite service. Agent processes must be recoverable after sleep and restart.
7. Gateway HMAC signing keys must be inaccessible to the agent through an operating-system-enforced identity boundary.
8. Clarification, proposal, and authorization are distinct. An agent cannot authorize its own actions.
9. F1 remains deferred. G-01 remains open. Consequential business writes remain disabled.
10. No production connector credentials are provided to Builder Sprites.
11. No Miniapp framework changes are permitted without the established approval process.
12. No commits or pushes without explicit authorization.

### Critical application-framework invariant

The coding agent's eventual application output must obey an additional binding architectural invariant:

**Miniapp is the exclusive application framework. Go is the exclusive implementation language for Miniapp applications. MiniappSpec is their canonical application specification.**

Generated Miniapps must be represented, validated, built, tested, and executed through the Miniapp framework.

Python, JavaScript, TypeScript, Node.js, or another language/framework must never become an alternative way to implement or execute a Miniapp.

The Agent Gateway and infrastructure tooling are separate components and are not subject to the Miniapp-application-language restriction.

Phase B does not yet implement the complete artifact-admission pipeline. Nevertheless, review the current instructions, adapter, and provisioning configuration for architectural contradictions that could encourage the coding agent to generate arbitrary-language applications.

## 3. Review scope

Inspect the complete Phase B Part 1 working tree, including all untracked files.

The reported implementation includes:

- `gateway/`: protocol, Studio client, command server, persisted command deduplication, durable outbox, callback delivery, supervisor, session manager, OpenCode adapter, NDJSON translator, file-drop Q&A, HMAC key boundary, and boot recovery.
- `tools/build_provision`: Builder provisioning and embedded-bundle integration.
- `cmd/gateway`: static Sprite Gateway binary.
- Studio gateway operator CLI: provisioning and command operations.
- `docs/gateway-phase-b-evidence.md`: capability matrix and implementation evidence.
- `docs/gateway-phase-b-live-runbook.md`: owner-executable live installation and recovery procedure.

Discover the actual file structure rather than assuming these descriptions are complete.

Inspect:

- Production code paths.
- Persistence and recovery mechanisms.
- Every relevant security boundary.
- Tests and their actual assertions.
- Embedded build artifacts and configuration.
- Install and provisioning scripts.
- Operator CLI behavior.
- Evidence classifications.
- Live runbook commands and expected outputs.

The tree is reported as entirely untracked with no commits. Do not rely on `git diff` alone. Explicitly inspect the untracked files.

## 4. Highest-priority adversarial review areas

### A. Multi-tenant isolation and authorization

Establish whether every command, session, event, callback, and persisted execution object is correctly bound to its tenant, application workspace, Builder Sprite, build, and turn.

Investigate:

- Cross-tenant confusion.
- Cross-build command replay.
- Incorrect Sprite-to-build association.
- Reuse of one HMAC key across builds.
- Forged callbacks and commands.
- Unauthorized continuation.
- Stale or superseded turn callbacks.
- Command identity collisions.
- Confused-deputy behavior in the Studio client and operator CLI.

Verify that the Gateway cannot select its own tenant or application identity merely from agent-supplied content.

The application workspace and Sprite binding must come from trusted provisioning records and be checked at enforcement boundaries.

### B. HMAC and secret isolation

Inspect the entire lifecycle of build-specific HMAC keys:

1. Generation and derivation.
2. Provisioning delivery.
3. Storage.
4. Runtime loading.
5. Signing and verification.
6. Rotation/re-provisioning.
7. Error handling and cleanup.

Determine whether the actual implementation preserves the intended boundary:

- Gateway executes as `studio-gw`.
- Agent executes as `studio-agent`.
- Agent cannot read Gateway signing keys.
- Key material does not enter prompts, agent-visible files, tool inputs, logs, callbacks, or artifacts.
- Missing or invalid key material fails closed.
- No test-only static key provider can accidentally become a production default.

Inspect process spawning, environment inheritance, file descriptors, group membership, directory ownership, symlinks, and writable paths.

Do not accept a permissions script as proof that the resulting live Sprite enforces the intended permissions. Distinguish scripted configuration from executed OS-level verification.

### C. Command processing and durable deduplication

Review the persisted command-processing mechanism.

Determine whether:

- Commands have stable identities and build bindings.
- Duplicate commands replay the original stored response.
- A crash between processing and recording completion cannot cause duplicate logical execution.
- Concurrent deliveries cannot execute the same command twice.
- Out-of-order commands are handled safely.
- A `continue` cannot resume a closed or superseded turn.
- Invalid commands cannot poison the deduplication store.
- Corrupt persistence fails safely rather than silently resetting deduplication state.

Pay particular attention to crash windows between recording command acceptance, starting a process, and recording its execution state.

Require evidence for the claimed replay semantics.

### D. Durable callback outbox

The outbox is a critical distributed-systems boundary.

Inspect:

- Atomic append and update behavior.
- `fsync` and durability assumptions.
- Partial writes.
- Process termination during writes.
- Disk-full handling.
- Corrupt or truncated records.
- Duplicate callback delivery.
- Acknowledgment persistence.
- Replay after gateway restart.
- Ordering and retry behavior.
- Backoff and jitter.
- Concurrent producers and consumers.
- Poison-event handling.

Determine whether an event can be lost after the agent produces it but before Studio acknowledges it.

Determine whether a callback can be removed from the outbox before Studio has durably accepted it.

Verify that retries preserve event identity and do not create duplicate logical effects in Studio.

Do not confuse HTTP response success with successful application-level acknowledgment.

### E. Wake-on-command and retry behavior

Inspect the Studio client and its Fly error classification.

Determine whether retry logic distinguishes:

- Sprite asleep or waking.
- Temporary network failures.
- Gateway unavailable.
- Authentication failures.
- Invalid commands.
- Permanent provisioning/configuration errors.

Retrying must not turn permanent errors into infinite loops.

A retry must preserve command identity so the Gateway's deduplication works.

Do not assume wake latency is sub-second. The implementation and runbook must remain correct under both warm and cold wake behavior.

The actual live wake behavior remains unverified.

### F. Process supervision and session recovery

Inspect the supervisor and session manager for:

- Correct child-process creation.
- Process-group management.
- Signal delivery.
- Graceful termination.
- SIGKILL escalation.
- Wait/reap behavior.
- Orphan prevention.
- Cancellation races.
- Process identity and privilege drop.
- Session persistence.
- Session-reference validation.
- Working-directory isolation.
- Restart reconstruction.

The supervisor must not silently auto-restart a dead agent and represent it as continuing successfully.

After restart, the Gateway must reconstruct what it can from persisted state and report `agent.resumed` or `agent.failed` honestly.

Inspect every recovery transition for a crash window.

### G. OpenCode adapter and NDJSON translator

The implementation has discovered that OpenCode 1.19.6's noninteractive `run` path hard-denies its native question tool and has no reply path in the run loop.

Inspect the claimed source basis and verify that the implementation does not still depend on the rejected native question mechanism.

The file-drop mechanism is now the required Q&A mechanism for this adapter.

Inspect the actual OpenCode adapter and translator for:

- Correct CLI invocation.
- Session-reference persistence.
- NDJSON parsing.
- Strict malformed-data handling.
- Partial lines and stream termination.
- Unknown event types.
- Exit-code interpretation.
- Tool invocation/results.
- Process cancellation.
- Session reuse.
- Honest completion/failure semantics.
- No terminal scraping or prose-based state inference.
- No invented CLI flags or undocumented interfaces.

Do not assume that a successful process exit means the build succeeded.

Do not accept a capability claim unless its test or source evidence establishes it.

### H. File-drop Q&A — highest-priority adapter-specific review

Review the question/answer handoff as an actual protocol, not merely two JSON files.

Inspect:

- Fixed paths.
- JSON schemas.
- Turn binding.
- Question identifiers.
- Answer identifiers.
- Atomic file creation.
- Partial-write protection.
- Duplicate questions.
- Duplicate answers.
- Stale answers.
- Superseded turns.
- Restart during file creation.
- Restart after question creation but before callback.
- Restart after answer creation but before agent resume.
- Agent failure to write a question.
- Agent failure to read an answer.
- Malformed and oversized files.
- Symlink and path-traversal attempts.
- File permissions and ownership.

**Challenge the directional trust boundary particularly hard.**

The agent is allowed to produce questions, but only the Gateway should be trusted to deliver the user's answer.

Determine whether the agent can pre-create, overwrite, replace, or forge an answer file. The shared `studio-io` group must not accidentally grant the agent the ability to impersonate Studio or alter a trusted answer.

The review must establish how the implementation distinguishes:

- Agent-authored question data.
- Gateway-authored user-answer data.

Verify that a response belongs to the correct tenant, build, and active turn.

Verify that a stale or fabricated answer cannot resume an unauthorized turn.

If the current permissions cannot guarantee directional integrity, report it as a security finding with the concrete path and root cause.

The fallback must be deterministic and must not depend on parsing arbitrary agent prose.

### I. Provisioning and embedded binary

Inspect `tools/build_provision`, `cmd/gateway`, embedded bundle generation, and all install scripts.

Verify:

- Static Linux build flags.
- Actual binary architecture.
- Embedded artifact identity and digest.
- No accidental host-binary deployment.
- No mutable, floating download during installation.
- No secret leakage through build arguments or logs.
- Correct Sprite/service configuration.
- Gateway service ownership of its HTTP port.
- Correct service startup ordering.
- Correct failure detection and cleanup.
- No false success based solely on HTTP 200.

The live service installation has not yet occurred. Clearly distinguish build-time and fake-backed verification from actual Sprite behavior.

### J. Operator CLI

Review provisioning and command operations for:

- Explicit tenant/build targeting.
- Safe defaults.
- Input validation.
- Idempotency.
- Clear error interpretation.
- Secret redaction.
- Prevention of accidental cross-tenant operations.
- No unapproved destructive actions.
- No ability to bypass F1 or G-01.
- Correct distinction between local/fake operations and live external operations.

### K. Miniapp compatibility and application-language exclusivity

Although artifact admission belongs principally to Phase D, inspect the current Phase B instructions and interfaces.

Determine whether the Gateway or OpenCode assignment builder could encourage creation of standalone Python, Node.js, or alternative-framework applications instead of Miniapp applications.

The intended application contract must remain:

- Go implementation.
- Miniapp framework.
- MiniappSpec.
- Miniapp runtime.

No alternative framework or language may become the default just because OpenCode can generate it.

Do not prematurely implement the Phase D artifact pipeline as part of this review. Identify architectural contradictions and ensure the current adapter boundary can enforce the requirement in later phases.

### L. Multi-tenant operational readiness

Assess whether the current implementation correctly supports application-scoped Builder identity and tenant association.

Identify operational requirements that are not yet implemented:

- Builder lifecycle and retirement.
- Compromise quarantine.
- Credential revocation.
- Resource accounting.
- Sprite quotas and concurrent-build limits.
- Callback and outbox observability.
- SSE lifetime and connection budgets.

Do not turn these into artificial Phase B blockers unless the current implementation violates a required invariant.

Record longer-term requirements for the phase where they belong.

## 5. Evidence and capability-claim audit

Review `docs/gateway-phase-b-evidence.md` line by line.

The capability matrix must distinguish:

1. Implemented and tested.
2. Implemented but not sufficiently tested.
3. Specified but not implemented.
4. Explicitly not built.
5. Unverified live behavior.

Pay special attention to the reported claims about OpenCode Q&A.

The implementation summary says that no live Sprite work has started, while also referring to a "live P-G3" proof. Determine exactly what that phrase means.

Was the actual OpenCode binary exercised? Was only a scripted double used? Was any test run inside a real Sprite?

Require precise terminology:

- Real OpenCode process in a local/fake-backed environment is not a live Sprite proof.
- Scripted agent-double compliance is not real OpenCode proof.
- Real Builder Sprite execution is a live infrastructure proof.

Do not silently downgrade or upgrade a claim.

Review the runbook for consistency with the evidence matrix.

Every expected output must have an observed basis or be labeled as an expected result.

Every unknown must be an explicit observe-and-decide checkpoint rather than a guessed command.

Verify that the runbook includes:

- Fly organization selection.
- Model provider and authentication decision.
- Studio callback reachability decision.
- Tunnel versus Fly-hosted Studio consideration.
- Builder retention decision.
- System package decision.
- Pin-drift policy.
- Resource cleanup.
- A verifiable empty-resource-list check, or explicit owner-approved Builder retention.

Do not perform any of these live operations.

## 6. Test and verification audit

The implementation reports:

- `gofmt` clean.
- `go vet ./...` clean.
- `go build ./...` successful.
- `go test -p 1 ./...` 13/13 on PostgreSQL 19 and PostgreSQL 17.
- Race tests clean on six touched packages.
- Static Linux ELF binary cross-compile verified.
- Thirty-six gateway tests plus other provision, CLI, and server tests.

Do not merely repeat these results.

Inspect whether the tests actually prove the invariants.

Pay particular attention to:

- Red-before/green-after evidence.
- Crash-recovery boundaries.
- Concurrent command and event handling.
- Deduplication under races.
- Outbox durability.
- File-drop directionality.
- Malformed input and path attacks.
- Process termination and reaping.
- Cross-build and cross-tenant isolation.
- Forged command and callback rejection.
- Stale-turn fencing.
- Static binary correctness.

Run appropriate local gates if the environment permits. Record exact commands and actual results.

Do not provision Sprites, authenticate with OAuth, mutate Fly resources, or initiate any live operation.

## 7. Scope and governance

Phase A is accepted and closed.

Phase B Part 1 is the subject of this review.

Part 2 — live Sprite installation, OAuth, Fly mutations, and live recovery — is explicitly on hold pending separate owner authorization.

Do not:

- Perform Part 2.
- Provision or delete Sprites.
- Request or use Fly credentials.
- Start OAuth authentication.
- Contact external infrastructure.
- Modify Miniapp.
- Resolve F1 or G-01.
- Change P5 scope or its golden path.
- Implement a second coding-agent adapter.
- Introduce DBOS or another workflow engine.
- Redesign the frozen v1 protocol without demonstrated evidence.
- Modify files.
- Commit or push anything.

Existing carried items:

- M5: SSE stream lifetime and resource budgets.
- M6: framework `app.invocations` data residual.
- M7: artifact-manifest sensitivity and Phase D read-path scrubbing.
- M8: evidence-report accuracy and wording.

Treat them according to their assigned future phases. Do not expand this review into a general redesign.

## 8. Review methodology

Use a root-cause-first adversarial approach.

For every potential defect:

1. Identify the exact source location and execution path.
2. State the invariant that should hold.
3. Demonstrate how the implementation violates it.
4. Provide a reproducible test, exploit, or deterministic failure scenario where possible.
5. Explain the system-level consequence.
6. Identify the correct ownership boundary for a durable fix.
7. Specify the regression test required to prove the correction.

Do not report stylistic preferences as defects.

Do not invent missing behavior.

Do not infer that a test passes merely because its name describes the desired behavior.

Separate source inspection, executed tests, scripted-double evidence, real binary execution, and live Sprite evidence.

## 9. Severity classification

Use these categories:

**BLOCKER**

A reproducible defect that breaks a fundamental security, tenant-isolation, data-integrity, authorization, or recovery invariant and prevents Phase B Part 1 acceptance.

**MAJOR**

A demonstrated defect or missing guarantee with meaningful security, reliability, or architectural consequences. State whether it blocks Part 1 acceptance or can be carried to a later phase with explicit containment.

**MINOR**

A contained correctness, maintainability, observability, or consistency problem that does not undermine the current phase's core guarantees.

**RISK / UNVERIFIED**

A behavior whose correctness depends on live infrastructure or an external component that has not yet been exercised. Identify the exact proof needed.

**ACCEPTED / NO FINDING**

An inspected mechanism that satisfies its stated contract and has sufficient supporting evidence.

A finding is not a blocker merely because additional hardening is conceivable.

## 10. Required output

Produce one consolidated report at:

`plans/adv_review_phase_b_part1.md`

Use this structure:

### A. Executive verdict

Choose one:

- ACCEPT — Part 1 is sound; proceed to the separately authorized live-runbook review.
- ACCEPT WITH CONTAINED FINDINGS — Part 1 is structurally sound; list issues that can safely be carried.
- NOT ACCEPTED — specific blockers prevent acceptance.

Do not authorize Part 2. That decision remains with the owner.

### B. Architecture and invariant assessment

Evaluate each binding invariant and state:

- Holds.
- Holds with limitations.
- Violated.
- Not yet testable.

### C. Findings

Provide a concise table:

| ID | Severity | Component | Exact location | Reproduction/evidence | Violated invariant | Required durable correction |
| -- | -------- | --------- | -------------- | --------------------- | ------------------ | --------------------------- |

Order by security and correctness impact.

### D. File-drop Q&A assessment

Give this its own detailed assessment.

Explain whether the mechanism is trustworthy, correctly turn-bound, directionally protected, crash-safe, and recoverable.

Explicitly state whether real OpenCode behavior was demonstrated or only simulated.

### E. Durability and recovery assessment

Evaluate command deduplication, outbox recovery, supervisor lifecycle, session reconstruction, and restart behavior.

Identify every important untested crash window.

### F. Security assessment

Evaluate tenant isolation, HMAC, key separation, filesystem permissions, callback trust, Fly grant assumptions, and secret exposure.

### G. Evidence and capability-matrix corrections

Identify every overstatement, ambiguous claim, missing proof, or incorrect status in the evidence document and live runbook.

### H. Accepted design decisions

List components that are sound and should not be redesigned.

### I. Deferred requirements

List valid issues assigned to Phase C, D, E, or broader multi-tenant operations.

### J. Final acceptance conditions

If not accepted, specify the smallest bounded correction task with root causes, regression tests, and exact completion criteria.

If accepted, state precisely what Part 1 has proved and what remains unverified before the owner can authorize Part 2.

## 11. Review discipline

Be adversarial, but not performatively critical.

The purpose is to protect the architecture and establish whether the implementation is ready for its next gate, not to generate an endless review backlog.

Do not reopen settled architectural decisions without concrete evidence.

Do not ask for another general review cycle when a focused correction can close the issue.

Do not recommend new abstractions simply because another design is possible.

Do not modify the repository.

**The final report must enable the Senior Architect to make a clear acceptance decision and decide whether to authorize the owner-run live installation procedure.**

One particularly important point in this review is the **directional integrity of file-drop Q&A**: the agent must be able to submit a question, but must not be able to forge or overwrite the user's answer. That deserves an explicit security test before live execution.