U5.12.1 ACCEPTED — CONDITION CLOSED

The endpoint-isolation evidence closes the final outstanding condition. The allowlist, PostgreSQL 19 version gate, and no-dial subprocess test address the problem at the correct boundary.

**U5.12 is fully accepted. The interim credential-mutation restriction is retired.**

The next milestone is **Phase D — Miniapp Build, Artifact Verification and Human Review**. I recommend planning it first, then reviewing the plan before implementation. This preserves the local-first strategy and avoids prematurely introducing build execution or approval semantics.

## **Prompt 22 — Phase D architecture and implementation plan**

# Prompt 22 — Phase D: Miniapp Artifact Pipeline and Verification

## Role

Act as the Senior Go Architect of Miniapp Studio.

You own the architecture, implementation sequencing, security boundaries, acceptance criteria, and technical decisions.

Your task is to investigate the current repository and produce an implementation-ready plan for Phase D.

You are not authorized to implement Phase D yet.

## 1. Current architectural status

The following milestones are accepted:

- Phase A: durable conversation infrastructure.
- Phase B Part 1: fake-backed Gateway implementation, accepted with bounded residuals.
- Phase C C1–C4: two-way conversation and backend interaction.
- Phase C C6: Focus Console, accepted at Boundary B.
- U5: ledger-driven browser matrix, two consecutive clean-slate acceptance passes.
- U5.12: permanent PostgreSQL role separation, accepted.
- U5.12.1: local PostgreSQL endpoint allowlist and PostgreSQL 19 version gate, accepted.

The repository contains the accepted, uncommitted Boundary B and U5.12 implementation. Preserve that work.

The current priority is to advance the product's core golden path without introducing cloud dependencies or broadening the architecture.

## 2. Phase D objective

Implement the first complete, verifiable Miniapp artifact lifecycle for the Invoice Review Desk.

The intended workflow:

1. The operator describes the application.
2. The agent proposes or updates a canonical MiniappSpec.
3. Studio validates the specification and artifact contract.
4. Studio obtains a versioned, immutable application artifact.
5. Studio verifies the artifact's identity, integrity, structure, and policy compliance.
6. Studio tests the artifact in an appropriately isolated environment.
7. Studio presents inspectable verification evidence.
8. A human operator explicitly approves or rejects the exact artifact.
9. Studio records the durable decision and its evidence.

Phase D ends at human review and approval.

**Do not implement application deployment or Runtime Sprite provisioning. Those belong to Phase E.**

## 3. Binding architectural invariants

All implementation decisions must preserve these requirements.

### Application implementation

- Miniapp is the exclusive application framework.
- Go is the exclusive application implementation language.
- MiniappSpec is the canonical application specification.
- Miniapp runtime is the exclusive application execution model.

These restrictions must be enforced at validation, artifact admission, verification, and build boundaries.

They must not exist merely as instructions in an agent prompt.

### Separation of responsibilities

- Studio owns durable state, conversations, questions, approvals, artifacts, builds, and evidence.
- Builder agents produce candidate artifacts but cannot authorize them.
- Artifact verification is a deterministic Studio-controlled operation.
- Artifact testing is an isolated execution operation.
- Human approval is an authenticated Studio control-plane operation.
- Runtime deployment is outside Phase D.

### Existing product boundaries

- Invoice Review Desk remains the golden path.
- G-01 remains in force: synthetic invoice data and simulated approval effects only.
- No consequential external business writes.
- No production connector credentials inside Builder environments.
- No agent can approve its own work.
- No broad multi-agent orchestration framework, marketplace, workflow canvas, or generic application builder expansion.

## 4. Repository investigation

Before designing the plan, inspect the actual repository.

Read and trace:

- `plans/agent-gateway-implementation-plan.md`
- `plans/miniapp-studio-implementation-agent-guidelines.md`
- `plans/roadmap4.md`
- `docs/hackathon-demo-critical-path.md`
- `docs/prompt21-boundary-b.md`
- `docs/u5.12-report.md`
- `docs/u5.12.1-endpoint-report.md`
- Existing Miniapp framework and runtime contracts.
- Existing artifact, build, app, and deployment models.
- Existing tools registry and tool manifest conventions.
- Existing tenant-scoped RLS and migration architecture.
- Existing Studio approval and authorization-request mechanisms.
- Focus Console scene handlers and templates.
- Existing test database package and endpoint restrictions.

Inspect actual code paths, migrations, tests, and manifest schemas.

Do not assume a previously discussed design has already been implemented. Verify every relevant capability in the repository.

Identify any discrepancies between accepted reports and current source code.

Do not reopen accepted Boundary B decisions without concrete evidence of an invariant violation.

## 5. Artifact contract

Design a versioned, deterministic artifact contract.

At minimum, investigate:

- Canonical MiniappSpec versioning.
- Application source layout.
- Go module structure and dependency declarations.
- Required manifests and metadata.
- Artifact packaging and archive format.
- Artifact content digest, including SHA-256.
- Immutable artifact identity.
- Build, tenant, conversation, and turn correlation.
- Provenance and source version.
- Dependency and toolchain requirements.
- Maximum artifact and individual-file sizes.
- Allowed file types and paths.
- Reproducibility and verification evidence.

The plan must make it possible to establish exactly which bytes were verified, tested, and approved.

An artifact cannot be modified after verification or approval without producing a new artifact identity and requiring verification and approval again.

Evaluate how the canonical specification and source artifact are related. Avoid competing sources of truth.

## 6. Artifact retrieval and admission security

Treat agent-produced artifacts as untrusted input.

Investigate safe artifact retrieval and extraction.

The design must address:

- Absolute paths and path traversal.
- Symlinks and hard links.
- Archive bombs and decompression limits.
- Oversized files and total extraction size.
- Duplicate paths and conflicting entries.
- Unexpected file types.
- Module replacement and dependency substitution.
- Unapproved external dependencies.
- Digest mismatches.
- Modified bytes after retrieval.
- Tenant and build ownership.
- Secret-bearing files and accidental credential inclusion.

Do not accept an artifact merely because its archive can be extracted.

The validation/admission process must prove that the artifact conforms to the exclusive Go/Miniapp architecture.

## 7. Required tools

Evaluate and design three Studio tools.

### `artifact.verify`

Deterministic verification without executing arbitrary application code.

It should establish:

- Artifact integrity and digest.
- Artifact contract version.
- MiniappSpec validity.
- Go implementation compliance.
- Miniapp framework/runtime compliance.
- Required file and manifest structure.
- Dependency policy compliance.
- Safe paths and archive properties.
- Tenant/build/artifact identity consistency.

It must produce structured, durable evidence explaining success or failure.

### `artifact.test`

Execute the artifact's build and tests in a secure, disposable environment.

This is the highest-risk part of Phase D.

A local subprocess with a temporary directory, `GOPROXY=off`, and a timeout is not security isolation for arbitrary untrusted Go code.

The design must explicitly address:

- No host credentials or inherited host environment.
- Restricted filesystem access.
- Network disabled.
- CPU and memory limits.
- Process-count limits.
- Execution deadlines.
- Process-tree termination.
- Temporary, disposable execution environment.
- Bounded output capture.
- Secret detection and redaction.
- Dependency availability and reproducibility.
- Cleanup after failures and process crashes.

Investigate which isolation mechanisms actually exist in the current local development environment.

Do not assume Docker, container tooling, or kernel sandboxing is available.

If adequate isolation is unavailable, define an honest capability boundary:

- Trusted repository fixtures may be tested locally.
- Arbitrary agent-produced code must not execute directly on the host.
- A missing sandbox must produce an explicit unavailable/blocked result, never fabricated success.

Plan the smallest durable solution rather than implementing a weak local workaround.

### `artifact.decide`

Human-only Studio control-plane operation.

It must record a durable decision tied to:

- Tenant.
- Build.
- Exact immutable artifact identity.
- Exact content digest.
- Approver identity.
- Decision (`approved` or `rejected`).
- Timestamp.
- Relevant verification and test evidence.

Only an authenticated, authorized human operator may invoke it.

It must not be registered as an agent-callable tool or exposed through Gateway/MCP agent capabilities.

An agent proposal, completed build, or successful test must never implicitly authorize an artifact.

Investigate idempotency, concurrent decisions, stale approvals, repeated requests, and rejection/rework behavior.

## 8. Persistence and migrations

Inspect the current schema before proposing database changes.

Migration numbers 000017–000020 are already occupied.

Do not assume which additional migration is required. Inspect the actual schema first.

If new persistence is necessary, propose the smallest migration set covering:

- Artifact identity and immutable metadata.
- Artifact digest and provenance.
- Verification results.
- Test executions and evidence references.
- Human decisions and approver identity.
- Tenant/build foreign-key relationships.
- Uniqueness and idempotency constraints.
- RLS and FORCE RLS.
- Safe deletion/retention semantics.

Do not duplicate durable conversation or approval functionality already provided by the existing schema.

Assess concurrency, transaction boundaries, crashes, retries, and partial completion.

## 9. Focus Console integration

Inspect the locked Focus Console design:

- `plans/plan_ui.html`
- `plans/plan_ui_z.md`

Phase D is expected to complete the relevant Spec, Validate, Connect, and Evidence experiences within the agreed scope.

Determine the minimal scene changes required for:

- Canonical MiniappSpec visibility.
- Validation results and failures.
- Artifact identity and digest.
- Test status and inspectable evidence.
- Approval/rejection actions.
- Rework after rejection.
- Honest blocked and unavailable states.

Do not create fictitious green states.

The Console must display state derived from persisted backend records.

OPS/DEV modes remain density modes over the same underlying data.

Connect must not silently become a production integration project. Preserve G-01 and the no-production-credentials boundary.

## 10. Failure and recovery design

Trace complete execution paths.

Cover:

- Artifact upload interrupted.
- Artifact retrieval failure.
- Invalid or malicious archive.
- Digest mismatch.
- Artifact changed between verification and testing.
- Verification process crash.
- Test environment startup failure.
- Network isolation failure.
- Timeout and process termination.
- Output truncation.
- Cleanup failure.
- Duplicate tool invocation.
- Concurrent verification or testing.
- Approval racing with a new artifact version.
- Rejection followed by rework.
- Database transaction failure.
- Studio server restart during execution.

Every operation must have durable, honest lifecycle semantics.

Do not treat an indeterminate execution as successful.

## 11. Mandatory negative cases

Preserve and expand the Phase D negative acceptance cases:

- Non-Go application implementation.
- Alternative application framework or runtime.
- Invalid MiniappSpec.
- Modified bytes or incorrect digest.
- Path traversal or malicious symlink.
- Oversized artifact.
- Failing Go tests.
- Secret-bearing output or artifact content.

Identify any additional classes that follow from the threat model.

Every negative case needs an explicit expected result and regression test.

Prompt-only restrictions are insufficient.

## 12. Local-first operating boundaries

Preserve the local-first strategy.

For this planning task:

- Do not invoke Sprite CLI, including read-only operations.
- Do not perform Fly operations.
- Do not provision cloud resources.
- Do not run live OpenCode, Codex, Claude Code, Antigravity, or Muse sessions.
- Do not provision Runtime Sprites.
- Do not deploy applications.
- Do not access production credentials or business systems.
- Do not modify user data.
- Use PostgreSQL 19 only at `127.0.0.1:5434`, with the accepted `testdb` endpoint guard.
- Use isolated run-specific test databases.
- Do not run test-environment provisioning concurrently with a live Studio instance if the operation explicitly rotates `studio_app`.
- Ordinary test suite execution must preserve the accepted concurrent-live-server invariant.
- No commits or pushes.

Planning and repository inspection only.

## 13. Implementation sequencing

Produce a dependency-ordered implementation plan.

Evaluate a sequence along these lines, changing it only with evidence:

- D0: canonical contract and boundary decisions.
- D1: artifact identity, packaging, retrieval, and admission.
- D2: deterministic `artifact.verify`.
- D3: isolated `artifact.test`.
- D4: human-only `artifact.decide`.
- D5: Console integration and artifact evidence.
- D6: full Invoice Review Desk rehearsal and acceptance.

Identify which parts can be independently tested and which depend on secure execution infrastructure.

Do not let UI implementation precede trustworthy backend state.

No Phase E deployment work.

## 14. Required deliverable

Create:

`plans/phase-d-implementation-plan.md`

The document must include:

1. Executive architectural decision.
2. Current repository inventory with exact paths.
3. Existing schema and capability assessment.
4. End-to-end artifact lifecycle.
5. Versioned artifact contract.
6. Artifact retrieval and admission security design.
7. `artifact.verify` contract.
8. `artifact.test` isolation architecture.
9. `artifact.decide` authorization and persistence contract.
10. Migration and RLS impact.
11. Focus Console changes.
12. Threat model and failure/recovery analysis.
13. Negative test matrix.
14. Sequenced implementation units and dependencies.
15. Concrete acceptance criteria.
16. Risks, unresolved questions, and blockers.
17. Local-first validation and environment constraints.
18. Explicit Phase E boundary.

Where an implementation decision cannot be established from repository evidence, mark it as unresolved rather than guessing.

## 15. Architectural decision discipline

Solve the underlying problems rather than patching immediate symptoms.

Require:

- Complete call-path investigation.
- Invariants assigned to their owning components.
- Red-before/green-after regression tests.
- Concurrency and crash/recovery coverage.
- Technically enforced security boundaries.
- Accurate separation of unit, integration, local, and live evidence.
- No unsupported claims of sandbox or security guarantees.

The deliverable is an implementation-ready architectural plan, not code.

Stop after creating the plan. Report your findings and any structural blockers. Request architectural acceptance before beginning Phase D implementation.

## **What happens after this prompt**

The coding agent should return the plan and stop. I would then review the three critical architectural decisions before authorizing implementation:

1. **Artifact identity:** how exactly the bytes verified, tested, and approved remain identical.
2. **Execution isolation:** how arbitrary Go code is tested without access to the host or network.
3. **Human authorization:** how approval is bound to the exact artifact digest and cannot be produced by an agent.

Those are the foundations of Phase D. Getting them right matters more than quickly producing a green demo.&#x20;