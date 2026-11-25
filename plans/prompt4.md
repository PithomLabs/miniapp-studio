The next work order should combine **P0 gate completion followed by P2 implementation**, with a hard stop if P0 cannot pass. That avoids another planning cycle while preserving the approved dependency gate.

# Miniapp Studio — P0 Finalization and P2 Implementation Work Order

## Role

Act as the principal Go architect and implementation lead for Miniapp Studio (`github.com/PithomLabs/miniapp-studio`).

P1 is implemented, adversarially reviewed, and approved. Your task is to complete the outstanding P0 framework validation gate and, only after it passes, implement P2: the generic runner, interpreter, server renderer, and dry-run capability.

This is an implementation assignment, not another architecture-review exercise.

**Governing principle:** investigate mechanisms and solve root causes. Do not patch symptoms, introduce local bypasses, weaken invariants, or claim success without execution evidence.

## 1. Authoritative inputs and decisions

Read and follow:

- `plans/miniapp-studio-implementation-plan-consolidated.md` — current governing plan, Revision 2.
- `plans/plan4.md` — frozen historical snapshot.
- `plans/plan5.1.md` — frozen historical snapshot.
- `plans/plan6.md` — frozen superseded approach; do not modify.
- `plans/adv_review_prompt3.md` — accepted P1 adversarial review and mandatory P2 handoff.
- `docs/p0-evidence.md` — existing P0 evidence and exact outstanding commands.
- Existing P1 implementation under `spec/`.

The consolidated implementation plan governs scope, execution order, acceptance criteria, and architectural boundaries.

### Accepted state

P1 is approved. Its implementation includes:

- Versioned MiniappSpec types.
- Strict validator with a 13-code error taxonomy.
- Canonical JSON and SHA-256 content hash.
- Standard-library-only dependency closure.
- Positive and negative fixtures.
- Read-only v1 operation vocabulary.

The adversarial review found no foundational or structural blockers.

Four MAJOR findings are explicitly carried into P2:

- F1: invalid specs can be returned alongside validation errors.
- F2: fields irrelevant to a step/component type can be silently accepted.
- F3: step paths and component targets can contain `..` traversal.
- F4: connection values can contain credential-shaped strings; the secret scan is name-based.

The P2 handoff in `plans/adv_review_prompt3.md` is mandatory.

The provisional Miniapp baseline is `73351a801fc54e19ee7254e6fe3ea7fbed859378` (`73351a8`). It is not a final dependency pin until P0 passes.

## 2. Execution order — hard phase gate

Execute in this order:

1. P0 environment diagnosis and framework validation.
2. P0 Studio import and approval-resumption spike.
3. P0 evidence review and exact dependency SHA selection.
4. P2 generic runner and interpreter.
5. P2 server renderer and dry-run.
6. P2 integration, security, and failure-injection tests.

**Do not begin framework-dependent P2 implementation before P0 passes.**

P1's existing independent specification package may remain untouched or receive only the narrowly scoped F1–F4 corrections needed for P2. Do not use that allowance to bypass the P0 gate.

If Docker daemon access prevents P0 validation, stop at P0 and report the exact host-level blocker. Do not mark the gate green, skip tests, or start framework-dependent P2 work.

---

## 3. Phase A — diagnose Docker and complete P0

Docker Postgres is the designated path for PostgreSQL-backed validation.

The Docker installation is present, but a previous attempt encountered permission denial on the Docker socket. Diagnose the actual cause before testing.

### A1. Diagnose Docker access

Inspect:

- Docker client and server availability.
- Active Docker context.
- Daemon availability.
- Docker socket ownership and permissions.
- Current user and group membership.
- Whether the CLI points to the intended daemon.
- Whether the daemon is running and reachable.

Use the existing Docker installation and designated Docker Postgres path.

Do not:

- Change socket permissions to world-writable.
- Disable Docker security.
- Circumvent host permissions.
- Install an unrelated container runtime.
- Claim an image or database is running without verifying it.
- Use a local non-Postgres substitute for PostgreSQL tests.

If access requires an administrator or user-shell action, report the precise action required and stop. Preserve the exact failure output in the evidence record.

### A2. Run the PostgreSQL 19 framework validation gate

After Docker access is confirmed:

1. Start the project's PostgreSQL 19 test environment using its existing supported configuration.
2. Verify the actual server version and readiness.
3. Run the exact framework CI commands against the selected SHA:

```
gofmt -l .
go vet ./...
go test -p 1 ./...
go test -race ./contract/ ./internal/... ./spikes/
```

The required restart tests must actually execute:

- `TestSleepAcrossRestart`
- `TestProbeRestart`

A skipped restart test is not a passing gate. Investigate and correct the underlying test-environment or test-registration issue rather than weakening the test.

Record:

- Exact framework commit SHA.
- PostgreSQL server version.
- Docker image identity/digest where available.
- Exact commands.
- Exit codes.
- Test counts and relevant test output.
- Any skipped tests and their reasons.
- Any environmental limitations.

### A3. Run the Studio import spike

Using the exact candidate framework SHA:

1. Import Miniapp from the new Studio module.
2. Register and execute one minimal Class A operation.
3. Exercise one approval-gated operation.
4. Park it in `waiting_approval`.
5. Decide the approval.
6. Re-submit `Invoke` with the original idempotency key.
7. Prove the original invocation resumes through the verified `pollApproval` path and executes exactly as specified.
8. Verify `Status` remains observation-only.
9. Verify terminal replay and duplicate submission behavior.

Do not invent APIs or assume compatibility based on successful compilation alone.

### A4. P0 exit decision

P0 passes only when:

- The complete framework validation suite succeeds.
- Required restart tests execute rather than skip.
- The race subset succeeds.
- The Studio import and Class A invocation spike succeeds.
- Approval park/decide/same-key resumption succeeds.
- An exact immutable Miniapp SHA is recorded.

If any condition fails:

- Identify the actual root cause.
- Classify it under governing plan §14.
- Distinguish a framework defect from an environment failure.
- Do not silently patch Studio around the failure.
- Do not start framework-dependent P2 work.

Once P0 passes, proceed directly to Phase B without requesting another general architecture review.

---

## 4. Phase B — P2 generic runner and interpreter

Implement the P2 scope in the Studio repository.

P2 scope from the governing plan:

- Generic runner registration.
- Deterministic MiniappSpec interpreter.
- Class A execution through the supported Miniapp framework contract.
- Server-rendered UI.
- Fixture-backed dry-run.
- Deliberate-rejection test.

Do not expand into P3 Fly infrastructure or P5 builder-agent integration.

### B1. Correct and enforce the P1-to-P2 boundary

Before adding execution behavior, address the four P1 MAJOR findings.

**F1 — invalid specs must never be usable**

`spec.Validate` can return a populated specification alongside errors.

Implement a clear, reusable validation boundary. Every path that admits, stores, renders, previews, or executes a spec must reject it unless validation succeeds.

Requirements:

- Never ignore the returned error collection.
- Revalidate at execution time.
- Bind every run to the exact validated `ContentHash` it started with.
- Prevent an invalid or subsequently mutated spec from entering execution.
- Ensure the renderer cannot bypass validation by accepting a directly constructed struct.

Correct misleading comments and API documentation.

Add regression tests proving invalid specs cannot be admitted, rendered, or executed.

**F2 — reject irrelevant fields**

Define and enforce the per-step-type and per-component-type field matrix.

Requirements:

- Interpret only fields belonging to the declared discriminator.
- Reject contradictory or irrelevant declarations during validation where applicable.
- Do not silently interpret a generic field as an operation.
- Never use generic fallback dispatch that ignores step type.
- Resolve declared references consistently.

Add table-driven tests for all field-matrix violations.

**F3 — eliminate path traversal**

Use one shared path predicate for step request paths and component targets.

Requirements:

- Reject `..` path traversal.
- Reject whitespace and control characters.
- Define the treatment of `?` and `#`.
- Normalize paths before authorization.
- Recheck allowlist membership after normalization.
- Never authorize an unnormalized path and normalize it afterward.
- Preserve exact declared operation and endpoint constraints.

The runtime/gateway boundary must independently enforce the final normalized destination.

Add path-traversal and normalization regression tests.

**F4 — resolve connections through trusted server state**

Treat `connection` as an opaque logical identifier, never as a credential or trusted URL.

Requirements:

- Resolve every connection through a server-side registry.
- Reject unknown connection identifiers.
- Never accept inline API keys, tokens, credential-bearing URLs, or provider secrets.
- Never send user-controlled connection values directly to a gateway.
- Keep secrets outside MiniappSpec, Sprite processes, browser responses, logs, and execution results.

A cheap known-credential-prefix blocklist may be used as a tripwire, but it is not the security boundary. Server-side registry resolution is the actual boundary.

Add tests for known credential prefixes and unknown connection IDs.

Do not build speculative entropy-based secret detection.

### B2. Establish the runner contract

Use the supported Miniapp public API, verified at the exact P0-approved SHA.

The runner must:

- Be statically registered.
- Execute through `runtime.Service.Invoke`.
- Use the framework's Class A execution contract.
- Accept a validated MiniappSpec and a fixed content hash.
- Interpret only declared v1 constructs.
- Execute synchronously within the approved MVP boundaries.
- Return deterministic, typed results and errors.
- Enforce bounded inputs, outputs, loops, and execution time.
- Preserve cancellation and authorization behavior supplied by the framework.
- Never execute arbitrary Go, JavaScript, shell, or generated source code.

Do not invent framework APIs or reach into private framework internals. If an essential general-purpose API is missing, classify it under §14 and follow the required change process.

### B3. Preserve read-only v1 at runtime

The v1 specification is read-only.

The runner must enforce:

- HTTP GET only.
- Declared capabilities only.
- Allowed methods and normalized paths only.
- Server-resolved connections only.
- No undeclared operations.
- No write step types.
- No write dispatch hidden behind another step.
- No renderer affordance that submits external writes.

Future write support requires a deliberate schema-version and architecture decision, plus the G-01 effect-lifecycle correction. Do not quietly extend v1.

Test hostile and directly constructed specifications at both admission and execution boundaries.

### B4. Tenant isolation

Multi-tenant readiness is a mandatory architecture constraint from the beginning, even though full tenant administration remains out of MVP scope.

Requirements:

- Derive tenant identity from authenticated server-side context.
- Never trust `app.owner` as proof of authorization.
- Compare or overwrite the submitted owner with the server-derived tenant.
- Scope application and specification identity by tenant.
- Ensure all reads and writes are explicitly tenant-scoped.
- Ensure execution, audit, approval, and result retrieval preserve the originating tenant.
- Deny and audit cross-tenant access attempts.
- Verify that one tenant cannot read, render, invoke, or mutate another tenant's resources.

Do not defer fundamental tenant isolation to a later multi-tenant UI project.

A single operator per workspace remains the MVP user experience. Do not introduce invitations, complex RBAC, or multi-tenant administration.

### B5. Server renderer

Implement the P2 server-rendered UI contract.

Requirements:

- Render only validated MiniappSpec data.
- Use a closed component catalog.
- Reject unknown components and unsupported bindings.
- Escape user-controlled content.
- Resolve bindings only against declared data.
- Render loading, empty, error, and success states.
- Render permissions and execution results honestly.
- Do not provide controls for unsupported or unapproved external writes.
- Keep business authorization out of templates and UI handlers.
- Route consequential operations through `runtime.Service`.

Do not introduce an SPA or client framework without repository evidence and an explicit need.

### B6. Dry-run

Implement fixture-backed dry-run.

Requirements:

- Use only declared fixtures.
- Execute deterministic transformations and presentation logic.
- Never call external providers or gateways.
- Never perform writes.
- Clearly distinguish dry-run output from a real invocation.
- Verify the input spec hash and fixture identity.
- Return actionable errors for invalid fixtures and unsupported constructs.

Add tests proving no network/provider call can occur during dry-run.

---

## 5. Test and acceptance requirements

Use the existing project CI and testing conventions.

Minimum P2 test coverage:

### Specification admission

- Invalid specs are rejected at every boundary.
- Directly constructed invalid structs cannot bypass validation.
- Content hash is verified and remains immutable during a run.
- Irrelevant fields are rejected.
- Unknown step and component types are rejected.
- Credential-shaped connections and unknown connection IDs are rejected.

### Interpreter

- Each supported v1 step has positive and negative tests.
- Dispatch is exhaustive and discriminator-based.
- Unsupported steps fail closed.
- Invalid bindings and outputs fail deterministically.
- Traversal and normalized-path allowlist tests pass.
- GET-only behavior is enforced at execution time.
- Bounded-loop and output-size limits are enforced.

### Renderer

- Only validated specs render.
- Unknown components and invalid bindings fail closed.
- User-controlled content is escaped.
- No write controls appear for v1.
- Cross-tenant rendering is denied.

### Runtime integration

- Class A invocation passes through the canonical `runtime.Service` path.
- Tenant context is preserved.
- Cancellation and authorization are enforced.
- Invocation and execution results are auditable.
- Idempotent submissions do not double-execute.
- Runs remain bound to their original spec version and content hash.

### Dry-run

- Fixture inputs produce deterministic results.
- External network access is never invoked.
- Unsupported constructs fail closed.

### Failure and security tests

- Malformed inputs.
- Oversized inputs.
- Spec mutation after admission.
- Cross-tenant attempts.
- Duplicate invocations.
- Cancellation.
- Unknown connections.
- Path normalization and traversal.
- Unsupported methods and step types.

Do not inflate the test suite with redundant tests. Every test must demonstrate an invariant or a relevant failure mode.

---

## 6. Architecture and framework change control

Follow governing plan §14.

When implementation exposes a Miniapp defect or missing framework capability:

1. Reproduce it.
2. Identify the actual failure path and invariant.
3. Decide whether the correct owner is Miniapp or Studio.
4. Record the smallest durable framework correction, if necessary.
5. Add a regression test proving the invariant.
6. Run the relevant Miniapp tests and Studio integration tests.
7. Preserve the exact dependency SHA.

Do not introduce a Studio-only workaround for a framework defect.

Do not silently change:

- Class A transaction semantics.
- Authorization guarantees.
- Effect locking or fencing.
- Recovery semantics.
- Tenant isolation.
- Frozen architectural contracts.

Changes that alter foundational guarantees require the approval process described in §14.4.

G-01 remains a framework-owned gap. P2 must remain read-only and must not bypass it. External writes remain disabled until G-01 has its own durable correction, regression tests, and acceptance evidence.

---

## 7. Scope restrictions

Do not:

- Modify frozen `plans/plan4.md`, `plans/plan5.1.md`, or `plans/plan6.md`.
- Reopen the approved architecture without concrete contradictory implementation evidence.
- Implement Fly/Sprite provisioning or Connector API integration (P3).
- Implement the builder agent or Invoice Review Desk end-to-end journey (P5).
- Add DBOS workflows or a background worker.
- Add external write support.
- Add managed-agent providers.
- Add marketplace, billing, or multi-tenant administration.
- Expand the Hacksgiving one-app cut line.
- Change the provisional dependency pin before the P0 gate passes.
- Commit, push, or publish changes without explicit authorization.

Keep Studio and Miniapp repository changes separate. Leave all work uncommitted.

---

## 8. Required deliverables

### P0 evidence

Update `docs/p0-evidence.md` with:

- Docker diagnosis.
- PostgreSQL 19 image and server evidence.
- Exact framework SHA.
- Full framework CI results.
- Restart test execution evidence.
- Race test evidence.
- Studio import and Class A invocation evidence.
- Approval park/decide/resumption results.
- Final P0 pass/fail decision.

### P2 implementation

Deliver:

- Corrected and hardened `spec/` validation boundary for F1–F4.
- Generic statically registered runner.
- Strict deterministic interpreter.
- Server-rendered MiniappSpec UI.
- Fixture-backed dry-run.
- Tests for the invariants in this work order.
- Updated README and architecture documentation only where implementation facts require it.
- Exact framework dependency SHA and build/test evidence.

### Final implementation report

Provide:

1. P0 result and evidence.
2. P2 files changed and their responsibilities.
3. Execution architecture and public framework APIs used.
4. F1–F4 disposition and regression tests.
5. Tenant-isolation evidence.
6. Test commands and actual results.
7. Known limitations and deferred work.
8. Any §14 framework correction records.
9. Remaining blockers against the governing plan.

Do not claim an unexecuted test passed.

## 9. Completion standard

P2 is complete when the generic runner and renderer execute a validated v1 spec through the approved Class A framework boundary, with deterministic dry-run and tests proving the read-only, validation, authorization, path, connector, tenant, and content-hash invariants.

If P0 cannot pass because Docker daemon permissions remain unavailable, stop after producing a precise P0 blocker report. Do not pretend P2 integration is verified.

**Final instruction:** P1 is accepted. Do not schedule another general architecture review. Implement the next approved work, surface only genuine foundational or structural blockers, and keep the one-app delivery cut line.

The prompt intentionally makes the four P1 findings implementation requirements, not new approval gates. **The immediate action is P0 with Docker Postgres 19; P2 begins automatically once that gate is genuinely green.**