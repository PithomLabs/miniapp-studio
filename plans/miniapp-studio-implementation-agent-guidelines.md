# Miniapp Studio — Implementation Agent Guidelines

**Purpose:** A reusable operating standard for coding agents implementing Miniapp Studio across Phases B–E. These guidelines are intended to produce changes that are secure, reliable, recoverable, reviewable, and straightforward for both agents and human operators to understand.

**Authority:** Follow the current governing roadmap, architecture decisions, phase plan, and task-specific implementation directive. This guide supplies cross-cutting engineering and proof standards; it does not authorize scope expansion or override a phase gate. Where a task-specific directive is stricter, follow the stricter rule. Where a conflict exists, stop and report it rather than silently choosing a new architecture.

---

## 1. Non-negotiable engineering contract

### 1.1 Solve the underlying problem

Do not patch the immediate symptom and declare the task complete. For every defect or requested change:

1. Reproduce the failure on the unmodified baseline when practical.
2. Identify the violated invariant and the actual failure mechanism.
3. Trace the complete execution path, including callers, persistence, process boundaries, retries, recovery, and user-visible effects.
4. Fix the problem at the component that owns the invariant.
5. Add regression tests that fail before the fix and pass afterward.
6. Exercise relevant concurrency, crash, restart, malformed-input, and unauthorized-access paths.
7. Explain residual risks and any behavior that remains unverified.

A workaround is not a durable fix when the underlying race, privilege gap, unsafe ownership boundary, or invalid state transition remains possible.

### 1.2 Preserve phase discipline

- Implement only the authorized phase and task scope.
- Respect the phase sequence and its exit gates. Do not implement later-phase functionality early because it appears convenient.
- Keep deferred items deferred unless an authorized directive explicitly changes their status.
- Do not refactor unrelated code or redesign accepted architecture without concrete evidence of a foundational defect and explicit authorization.
- Do not modify Miniapp framework behavior to compensate for a Studio defect unless a demonstrated framework-level problem and the approved process justify it.
- Do not add a new workflow engine, adapter, framework, or persistence mechanism merely to avoid understanding the current design.
- Do not commit, push, deploy, provision infrastructure, or perform live provider operations unless explicitly authorized.

### 1.3 Be precise about evidence

Never describe an assumption, mock, fake, unit test, or source-code inspection as a live production verification. Label evidence accurately:

- **Code-verified:** implementation and static inspection establish the behavior, but it has not been exercised end to end.
- **Unit-tested:** isolated test exercises the behavior, often with fakes or controlled dependencies.
- **Integration-tested:** real collaborating components are exercised together in a controlled environment.
- **Live-tested:** the actual external service, operating-system boundary, provider, or deployed environment was exercised.
- **Specified / unverified:** behavior is based on documentation, an unpinned source, an assumption, or a test that could not be run.

For each acceptance claim, record the evidence category, exact command or procedure, result, and remaining limitation. A documentation statement is not proof that the runtime enforces it.

---

## 2. Locked Miniapp Studio architecture and product constraints

These invariants apply to all implementation work unless a later, explicit architecture decision changes them.

### 2.1 Product and tenant boundaries

- Studio is the authoritative multi-tenant control plane and permanent record of business conversations, approvals, builds, artifacts, and deployments.
- One Studio control plane serves multiple tenants; each tenant may have multiple application workspaces.
- Each application workspace has a dedicated, long-lived Builder Sprite. Builds reuse that Builder; they do not receive a shared Builder across tenants.
- An approved application runs in a separate Runtime Sprite. The Runtime Sprite is not a coding environment and must not run the coding agent or Agent Gateway.
- Users interact with Studio, not directly with Sprite terminals.
- Studio-to-Sprite communication uses authenticated HTTPS commands and callbacks. Do not introduce a permanent browser-to-Sprite socket.
- Studio Postgres rows are the source of truth. Sprite files, agent sessions, deduplication records, and outbox entries are execution state that must be reconciled with Studio.

### 2.2 Trust and credential boundaries

- Builder Sprites are execution environments, not trusted holders of production connector credentials.
- Provider credentials belong to the connector boundary and must be narrowly scoped.
- An agent must not grant itself permissions, approve its own proposals, or bypass Studio authorization.
- Clarification, proposal, and authorization are separate operations.
- Agent Gateway is agent-independent through its adapter contract; OpenCode is the initial adapter. Do not silently bind Studio architecture to OpenCode-specific behavior.
- The Builder may hold only the minimum scoped credentials required for its authorized Gateway function. It must not receive production connector secrets.

### 2.3 Exclusive Miniapp application contract

**Miniapp is the exclusive application framework. Go is the exclusive implementation language for Miniapp applications. MiniappSpec is the canonical application specification. Miniapp framework/runtime is the exclusive execution model.**

Every AI-generated Miniapp must:

- be implemented in Go;
- use the Miniapp framework;
- conform to MiniappSpec; and
- execute through the Miniapp runtime.

Python, JavaScript/TypeScript/Node, or another framework/runtime must not become an alternative implementation or execution path for a Miniapp. Other languages may be used for infrastructure, Gateway, coding-agent software, and development tooling where appropriate; that does not relax the Miniapp application contract.

Agent instructions alone are not enforcement. Where the phase scope includes the relevant layer, reinforce the contract in assignment construction, MiniappSpec validation/admission, build/test tooling, Runtime provisioning, deployment, and negative regression tests. Do not claim the full artifact-admission gate exists before its authorized phase.

### 2.4 Current restrictions to preserve

- F1 operator session issuance and G-01 stale `PENDING` invocation ownership remain deferred/open as documented. Consequential Miniapp business writes remain disabled until G-01 is fixed and accepted.
- P5 golden-path scope remains unchanged; do not expand the current roadmap into unrelated application examples or generalized task-management functionality.
- Do not introduce DBOS or another workflow engine without a demonstrated need and authorization.
- Do not change frozen plans or accepted architecture documents unless explicitly instructed.

---

## 3. Security pre-flight: boundaries, identities, and permissions

Complete this analysis before changing security-sensitive code.

### 3.1 Write the threat and trust model first

State explicitly:

- which components, processes, identities, and users are trusted;
- which components or inputs are untrusted or less privileged;
- which assets must be protected (signing keys, provider tokens, tenant data, build artifacts, session files, callbacks, and host files);
- what each actor can read, write, execute, create, delete, or invoke;
- where data crosses a process, filesystem, network, tenant, or privilege boundary; and
- how an attacker or compromised agent could exploit each boundary.

Do not use vague labels such as “secure directory” or “private key.” Identify the concrete OS identity, path, mode, ownership, API, and enforcement mechanism.

### 3.2 Build an actor × resource × operation permission matrix

For every identity and shared resource, document the intended permissions. Include at least:

| Actor | Resource | Read | Write/create | Delete/replace | Required denial |
|---|---|---:|---:|---:|---|
| Gateway process | Signing key material | Yes | As required for rotation | As authorized | Other tenants' keys |
| Agent process | Signing key material | No | No | No | All key reads and writes |
| Agent process | Question handoff | Only its authorized answer path | Question submission only | No gateway-owned content | Keys, other builds, gateway state |
| Gateway process | Question handoff | Yes | Yes | Controlled cleanup | Other tenants' handoff data |
| Agent process | Answer handoff | Yes, within its build | No gateway impersonation | No gateway-owned files | Other builds and protected paths |

This table is illustrative. Replace it with the exact identities, UID/GID values, paths, groups, file types, and modes for the task. Verify both allowed and denied operations under the real identities. A setup script that creates users without applying those identities to the running processes does not establish isolation.

### 3.3 Enforce least privilege at the actual execution boundary

- Apply OS identity separation at process creation and service provisioning, not only in setup scripts or documentation.
- Do not invent unsupported service API fields or assume a platform supports a privilege mechanism.
- If the required identity drop or privilege boundary cannot be established, fail closed. Never fall back to running the agent with Gateway privileges.
- Set supplementary groups deliberately and verify effective UID, GID, and groups from the spawned process.
- Verify actual denied access (for example, `EACCES` when the agent attempts to read Gateway keys) as well as permitted handoff operations.
- Treat a fake-backed refusal/plumbing test as useful unit evidence, not proof of a real privilege drop. Record the live proof as a separate gate when the environment is available.

### 3.4 Verify permissions across the entire lifecycle

For every file and directory, specify and test:

- owner and group at creation;
- mode at creation and after installation;
- parent directory permissions and sticky/setgid behavior;
- permissions after atomic replacement or rotation;
- permissions on temporary, staged, backup, and copied artifacts;
- cleanup and deletion behavior; and
- what happens after restart, upgrade, interrupted setup, or partial failure.

Check every copy of a secret, not just its canonical path. Remove staged key copies after installation and ensure they are not world-readable, included in logs, emitted in errors, or left in temporary directories.

---

## 4. Safe filesystem and untrusted-input handling

Treat every path writable by an agent, tenant, or lower-privileged process as adversarial.

### 4.1 Defend against symlinks and path races

- Use directory-relative operations rooted in a trusted directory where supported (for example, `os.OpenRoot`).
- Use `O_NOFOLLOW` when opening paths that must not follow symlinks.
- Use `O_EXCL` for creation when an existing path must never be overwritten.
- Check file type, owner, and mode on the opened file descriptor using `fstat`-style checks; do not rely on a separate `stat` followed by a path-based open.
- Read or write through the validated descriptor rather than resolving the path again.
- Use safe atomic replacement when replacement is intended. Define how existing files are handled based on ownership and type.
- Do not assume flags or filesystem semantics are honored identically by every platform. Test the actual target OS and filesystem behavior.

### 4.2 Bound and validate all data

- Set explicit limits for file size, line length, record count, request body size, decoded payload size, stderr capture, CLI output, and other unbounded input.
- Validate type, ownership, expected naming, schema, encoding, and required fields before trusting a handoff or artifact.
- Reject malformed, oversized, unexpected, or ambiguous content safely and observably.
- Do not print large binary payloads or secrets to terminal output. Truncate with an explicit notice or write to a controlled output file.
- For collectors, avoid check-then-open TOCTOU sequences. Open safely, validate the descriptor, then read from that descriptor.
- Surface scanner/read errors. Do not silently treat incomplete input as a valid empty result.

### 4.3 Make failures safe and diagnosable

- Fail closed on key, ownership, signature, schema, path, and tenant-boundary validation failures.
- Return enough diagnostic context for an operator to identify the failure without exposing secrets or cross-tenant information.
- Do not turn all database errors into “not found,” or all malformed records into silently ignored data.
- For record stores, isolate corrupt records where safe, report the corruption, and continue only when the contract explicitly permits it.

---

## 5. Concurrency, idempotency, retries, and state-machine integrity

Concurrency and failure recovery are security concerns whenever commands can cause external side effects.

### 5.1 Model the state machine before coding

For each command, session, build, callback, and outbox item, enumerate:

- valid states and transitions;
- which component owns each transition;
- the transaction or durable write that makes the transition authoritative;
- what happens if the process crashes immediately before or after that write;
- whether the operation is safe to repeat;
- how stale or unauthorized transitions are rejected; and
- the explicit next action for every terminal or degraded state.

Do not infer success from a progress message or a partially completed operation. Terminality must be based on authoritative outcomes, such as the actual supervised process result or a validated protocol event.

### 5.2 Claim work before dispatch

- Use durable atomic claims before dispatching work that must not execute twice.
- Use in-process mutexes or singleflight only as additional coordination; they do not replace durable claims across process restarts or multiple Gateway instances.
- Define lock ordering and prove that it cannot deadlock.
- Fence stale turns, commands, and session updates at the component that owns the state.
- Reject the same idempotency/message identifier when reused with a different payload or identity.

### 5.3 Treat ambiguous crash outcomes as indeterminate

**Never automatically drop a stale in-progress claim merely because the Gateway restarted.** A crash may occur after an agent performed work or emitted an event but before the Gateway persisted command completion.

Use this rule:

- Drop a claim only when the system can prove dispatch never began and no child process was started.
- If a process may have started, or its events may already have reached durable storage or an external recipient, mark the outcome **indeterminate**.
- Reconcile the process, persisted events, outbox, and Studio state before deciding the next action.
- Do not blindly relaunch the same logical command from an indeterminate state.
- Use an explicit authorized recovery operation, with a new fenced turn or invocation identity, when work must continue.
- Preserve a stable logical event/message identity so duplicate delivery can be recognized without pretending the side effect itself ran exactly once.

The honest target is at-most-once dispatch with explicit reconciliation and recovery for ambiguous outcomes. Do not claim exactly-once execution unless the complete end-to-end side-effect boundary actually provides and proves it.

### 5.4 Align retry semantics with freshness and replay protection

- Understand the lifetime of timestamps, nonces, signatures, tokens, leases, and idempotency keys.
- If a signed envelope is retried after its freshness window, create a fresh authenticated attempt while preserving the stable logical message identity and payload.
- Do not weaken receiver-side freshness or signature validation simply to make retries pass.
- Verify that stale forged data is still rejected.
- Require HTTPS for non-loopback transport where the protocol depends on transport confidentiality/authenticity; document any intentional local exception.
- Test retry behavior across the actual configured skew, timeout, and retention windows.

### 5.5 Make recovery and requeue explicit

- Every failed/degraded terminal state must have a documented operator-visible status and an authorized recovery path, or be explicitly final.
- Requeue must preserve the logical message identity when required for deduplication, record the operator action, and validate build/tenant ownership.
- Recovery must not fabricate “resumed” or “completed” events. Emit them only after the corresponding operation is verified.
- Ensure a durable reconciliation/backstop exists for work that cannot be delivered or completed during the first attempt.

---

## 6. Process, service, and provisioning lifecycle

### 6.1 Supervise processes independently of request lifetimes

- Agent process lifetime must not accidentally inherit a short-lived HTTP request context.
- Define cancellation, shutdown, timeout, and restart semantics explicitly.
- Handle SIGTERM/graceful shutdown and ensure supervised children receive the intended signal.
- Manage process groups and descendants so grandchildren are not orphaned.
- Reap only processes whose identity is verified; protect against PID reuse by checking process start time and expected executable identity where applicable.
- Bound stderr and diagnostic buffers while retaining useful failure context.

### 6.2 Make provisioning non-disruptive by default

For shared, long-lived Builder services:

- Create the service only when absent.
- Preserve an existing healthy service and any active sessions.
- Treat service-inspection errors and unhealthy states as explicit failures requiring diagnosis; do not delete/recreate automatically.
- Make upgrades an explicit guarded operation, with binary identity/digest checks and an active-session policy.
- Coordinate concurrent provisioning across all actors that can provision the same Sprite. A process-local mutex is insufficient when multiple CLI or service processes can act concurrently.
- Verify the resulting service state after setup or upgrade.

### 6.3 Make binary and dependency identity auditable

- Pin tool, SDK, binary, and external source versions when the plan requires it.
- Record and verify digests for installed binaries and bundled source material.
- If a source cannot be fetched or pinned, mark claims that depend on it as unverified rather than silently claiming verification.
- Do not assume an upstream CLI option, exit code, event format, or session behavior from a comment or a mutable documentation page. Test the pinned version.

---

## 7. Agent-friendly implementation instructions

Every task prompt or implementation assignment should give the coding agent enough information to act without guessing.

### 7.1 Required task specification

Provide these sections, even if some are brief:

1. **Goal:** observable outcome and reason for the change.
2. **Scope:** exact packages, files, interfaces, and phase boundary.
3. **Current behavior:** relevant execution path and known failure evidence.
4. **Invariant:** behavior that must always hold, including security/tenant boundaries.
5. **Threat/boundary model:** trusted actors, untrusted inputs, protected assets, and authority.
6. **Required behavior:** precise inputs, outputs, state transitions, ownership, permissions, and errors.
7. **Failure cases:** concurrency, retries, crash windows, restart, malformed data, unauthorized access, and partial failure.
8. **Validation:** exact red/green tests, integration/live checks, database baseline, commands, and expected evidence.
9. **Forbidden changes:** explicitly list scope expansions, fallback paths, framework changes, credential use, live operations, and commits that are not authorized.
10. **Deliverable and stop condition:** required report/doc updates and the exact point at which the agent must stop.

### 7.2 Do not leave dangerous choices implicit

Specify exact paths, payload shapes, allowed commands, ownership and modes, terminal conditions, retry behavior, timeout rules, and error behavior. Require the agent to investigate platform/library capabilities rather than inventing unsupported APIs or silently substituting a different mechanism.

When a decision is reversible and grounded in the existing architecture, state the default and rationale. If an essential choice is genuinely unresolvable within the task's authority, stop and report the blocker. Do not ask the user to repeat information already available in the governing plan or repository.

### 7.3 Require root-cause evidence

For each finding, the implementation report must state:

- original reproduction and why it fails;
- violated invariant and root cause;
- ownership boundary where the fix belongs;
- durable change made;
- regression test that would have failed before the change;
- concurrency/crash/security cases exercised;
- evidence category (unit, integration, or live); and
- residual risk or deferred proof.

---

## 8. Testing and verification standard

### 8.1 Test the boundary, not just the happy path

Pair normal behavior tests with adversarial and negative tests appropriate to the change. Examples include:

- unauthorized identity attempts to read keys or modify Gateway-owned files;
- symlink reads and writes targeting secrets or unrelated files;
- path replacement between validation and use;
- oversized, truncated, malformed, or unexpected input;
- concurrent identical requests and concurrent distinct requests;
- reuse of an idempotency key with altered payload;
- crash after claim, after process start, after event persistence, and before completion;
- stale timestamps, signatures, turns, or session updates;
- duplicate callback delivery and retry after the freshness window;
- process exit after intermediate error output;
- SIGTERM, child/grandchild cleanup, and process recovery;
- service provisioning while another build is active; and
- outbox failure, requeue, and reconciliation.

For crash tests, place the crash at the boundary between two durable or externally visible actions. A generic “restart test” is not sufficient if it misses the dangerous window.

### 8.2 Reproduce red before green

- Run the smallest deterministic reproduction against the original code before changing it.
- Record the expected failure and actual result.
- Implement the fix and run the same reproduction.
- Add the regression test to the permanent suite; do not rely only on a one-off probe.
- Do not change the test to accept a weaker invariant merely to make the implementation pass.

### 8.3 PostgreSQL test baseline — mandatory

**Use the user's existing PostgreSQL 19 Docker instance only, at `127.0.0.1:5434`.** This is the uniform database baseline for this project.

Before database-backed verification:

1. Connect to the established Docker PostgreSQL endpoint using the approved local connection method.
2. Run `SELECT version();` and `SHOW server_version;`.
3. Confirm the server is PostgreSQL **19.x**.
4. Record the exact version and endpoint in the evidence report.

Rules:

- Use isolated disposable databases or schemas for tests. Never drop, reset, or overwrite user-owned data.
- Do not substitute PostgreSQL 16, 17, 18, SQLite, mocks, or another server and describe the result as equivalent PG19 verification.
- If PostgreSQL 19 at `127.0.0.1:5434` is unavailable or reports another major version, stop database-backed testing and report the exact connection/version failure. Continue only non-database work that remains valid, and clearly mark the relevant gates blocked.
- Do not hide failed database tests behind skips or alternate configurations.

### 8.4 Standard repository gates

Run the task-required gates, serially where specified, and record exact commands and exit codes. The standard Go gate set for this project is:

```sh
gofmt -l .
go vet ./...
go build ./...
go test -p 1 ./...
go test -race -p 1 ./gateway ./store ./tools ./server ./cmd/studio ./cmd/gateway
```

Run only packages/commands that exist in the repository and are relevant to the task; explain any adjustment. Re-verify static binary/build properties when the plan requires them. A green build does not replace database, concurrency, adversarial, or live verification.

### 8.5 Keep test doubles in their proper role

- Fakes and scripted runners are valuable for deterministic edge cases and unit coverage.
- They do not prove external CLI behavior, real UID separation, Sprite sleep/wake behavior, provider OAuth scopes, network policy, or production service semantics.
- Add live checks to the authorized runbook for facts that can only be established in the target environment.
- Never run live operations merely to strengthen evidence when the phase/task has not authorized them.

---

## 9. Documentation, runbooks, and operational truth

- Update evidence documents and runbooks in the same change as behavior changes.
- Remove statements that no longer match the code. Do not leave contradictory comments, fictitious future sections, stale test counts, or claims of live verification that did not occur.
- Include exact prerequisites, commands, expected outcomes, rollback/stop conditions, and evidence to capture in owner-run live procedures.
- Keep secrets, OAuth credentials, and production tokens owner-held. Do not request them in chat or place them in implementation logs.
- Make operational status expose degraded conditions, stuck/indeterminate work, failed outbox items, corrupt records, and required operator action without leaking sensitive data.
- Distinguish “not implemented,” “implemented but untested,” “unit-tested,” “integration-tested,” and “live-verified.”

---

## 10. Phase B-specific cautions carried forward

These points are especially important while completing the current Agent Gateway correction work. They do not authorize Phase B Part 2 live operations.

### 10.1 B1 — real identity isolation

`SysProcAttr.Credential` is acceptable only when the actual agent process is always spawned with the configured identity and there is no unprivileged fallback path. Missing UID/GID configuration or a failed credential drop must refuse the spawn. Unit tests can prove refusal and plumbing; the real different-UID drop and key-read denial remain live verification requirements for the authorized runbook.

### 10.2 F2/F3 — file-drop safety

Use directory-rooted, symlink-safe operations; verify ownership and regular-file type on opened descriptors; impose size limits; and define safe replacement behavior. Test both secret exfiltration and victim-file clobbering attempts, including races and restart windows.

### 10.3 F4 — ambiguous dispatch recovery

Persist a claim before dispatch. A stale claim is not automatically retryable. If process start or side effects may have occurred, classify it as indeterminate, reconcile events/outbox/process identity, and avoid relaunching the same logical command. Any authorized continuation must use the explicit recovery path and a new fenced turn/invocation identity. Test the exact crash window after an event reaches durable outbox storage but before command completion is recorded.

### 10.4 F5/F6 — honest terminality and continuation

Intermediate agent error output is diagnostic progress, not necessarily terminal failure. Determine terminality from the actual supervised process outcome and validated question-yield behavior. A resumed event must follow a verified launch; never fabricate recovery success. Reject stale turns.

### 10.5 F7 — preserve shared services

Provisioning a second build must not delete or restart a healthy shared Builder Gateway service or interrupt another build's active session. Create-if-absent is the default; unhealthy or uninspectable service state fails closed; upgrades are explicit and guarded. Verify coordination across independent provisioner processes, not only goroutines.

### 10.6 F8/F9 — outbox operability and freshness

Provide a tenant/build-scoped inspect and authorized requeue path; preserve logical message identity where required. Re-sign each delivery attempt with fresh freshness metadata while preserving the logical message and payload. Do not weaken Studio-side signature/skew validation. Surface enqueue failure and degraded state rather than silently losing events.

---

## 11. Final implementation-agent sign-off checklist

Before reporting completion, confirm every applicable item:

### Scope and architecture
- [ ] Work belongs to the authorized phase and task.
- [ ] Locked architecture and Miniapp application contract remain intact.
- [ ] No deferred item, live operation, framework change, or commit was performed without authorization.

### Security and boundaries
- [ ] Threat model and actor/resource/operation permission matrix are documented.
- [ ] OS identity and file permissions are enforced at runtime, not merely described.
- [ ] Secret copies, staging paths, logs, and error paths have been reviewed.
- [ ] Untrusted paths are symlink-safe, race-safe, type/ownership-checked, and size-bounded.
- [ ] Authentication, tenant/build ownership, turn fencing, and transport requirements are enforced.

### State, concurrency, and recovery
- [ ] Durable claims precede side-effecting dispatch.
- [ ] Concurrent duplicate and distinct requests are tested.
- [ ] Crash windows and ambiguous outcomes are handled without blind replay.
- [ ] Retry freshness, idempotency identity, and replay protection are tested together.
- [ ] Every failure/degraded state has an honest status and authorized next step.
- [ ] Provisioning and shutdown cannot disrupt unrelated active work or leak child processes.

### Verification and evidence
- [ ] Red reproduction exists for each reported defect where practical.
- [ ] Regression tests prove the invariant, including adversarial/negative cases.
- [ ] PostgreSQL 19 at `127.0.0.1:5434` was verified before DB-backed tests; no substitute was used.
- [ ] Required formatting, vet, build, unit, race, integration, and live gates are accurately reported.
- [ ] Test skips, failures, unavailability, and unverified claims are disclosed.
- [ ] Documentation and runbooks describe actual behavior and remaining live gates.

### Completion report
- [ ] Summary of changes by finding/invariant.
- [ ] Root cause and ownership boundary for each fix.
- [ ] Red/green test evidence and exact commands/exit codes.
- [ ] PostgreSQL 19 version evidence and isolation method.
- [ ] Evidence classification for each acceptance claim.
- [ ] Residual risks, deferred work, and live checks still required.
- [ ] Explicit statement of what was not done and the exact stop point.

**Completion rule:** Report completion only when the authorized acceptance criteria are met. If a required gate is blocked, state that the phase/task is not fully verified; do not convert a blocker into a pass by substituting a weaker test or changing the claim.
