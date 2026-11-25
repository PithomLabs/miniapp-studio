I’ll drive Miniapp Studio development as its **Senior Architect**, owning architectural direction, phase sequencing, scope control, coding-agent instructions, and acceptance decisions. We will avoid unnecessary review cycles and move forward whenever there are no foundational or structural blockers.

**My decision: proceed with Phase A — Conversation Infrastructure. Not Phase B provisioning yet.** Phase A must establish the durable Studio-side foundation before we introduce OpenCode into a Builder Sprite.&#x20;

agent-gateway-implementation-plan.md



## **Next coding-agent assignment**

Give the coding agent this prompt:

# IMPLEMENTATION DIRECTIVE — PHASE A: MINIAPP STUDIO CONVERSATION INFRASTRUCTURE

**Project:** Miniapp Studio\
**Role:** You are the Senior Implementation Engineer working under the direction of the project's Senior Architect.\
**Authorization:** Implement Phase A only.\
**Governing architecture:** `plans/agent-gateway-implementation-plan.md`, including the approved and merged OpenCode amendment M1–M11.

## 1. Objective

Implement Phase A — Conversation Infrastructure.

Establish Miniapp Studio as the authoritative, persistent conversation and build-management system through which a business user communicates with a coding agent.

Studio must support:

- Persistent business-user conversations.
- Persistent messages and agent events.
- Application build assignments.
- Agent turns and sessions.
- Agent questions and user answers.
- Human decisions and authorization references.
- Build lifecycle and status transitions.
- Authenticated gateway callback ingestion.
- Real-time browser updates through Server-Sent Events (SSE).
- Browser disconnection and reliable event replay.
- Durable operation across Studio restarts.

**Phase A must work without a live Sprite or coding-agent process.** Use a deterministic fake Gateway to prove the complete Studio-side interaction.

The exit condition is a working, durable conversation infrastructure that Phase B can use without redesigning its data model or communication protocol.

## 2. Read the governing sources first

Before modifying code, inspect the current working checkout.

Read:

1. Repository `AGENTS.md` instructions and applicable user-level instructions.
2. `plans/miniapp-studio-implementation-plan-consolidated.md`, Revision 2.
3. The complete merged `plans/agent-gateway-implementation-plan.md`, including the OpenCode amendment merge markers and changelog.
4. `plans/agent-gateway-opencode-amendment.md` as the separate review record.
5. Relevant architecture decisions, especially ADR-001, ADR-002, ADR-003.
6. P0–P4 evidence, especially `docs/p4-evidence.md`.
7. Existing implementation in `server/`, `store/`, `tools/`, `runner/`, `apps/`, `fly/`, and `cmd/studio`.
8. Existing Miniapp runtime invocation, approval, audit, and tenant-isolation implementations.

Verify the actual checkout. Do not rely on historical conversation summaries as substitutes for source inspection.

Inspect current Git status, preserve existing uncommitted work, and establish the baseline test results before making changes.

## 3. Locked architectural boundaries

Preserve these decisions:

**Miniapp Studio owns:**

- Conversations and permanent history.
- Messages and agent events.
- Build assignments and agent-session references.
- Questions, user answers, decisions, and approvals.
- Build status and artifact metadata.
- Business-user-facing conversation interface.

**Miniapp owns:**

- Runtime invocation and static tool registration.
- Trusted execution and policy boundaries.
- Tenant context and governed capabilities.
- Audit and execution boundaries.

**Future Builder Sprite owns:**

- OpenCode execution.
- Gateway execution.
- Repository and development workspace.
- Agent session files and generated artifacts.

OpenCode is the designated first coding agent, pinned as specified by the merged plan. Do not implement the OpenCode adapter or provision Sprites during Phase A.

Maintain separation between Builder and Application Runtime environments.

The business user must see one continuous Studio conversation. Never connect the browser directly to a Sprite.

## 4. Implementation scope

### A. Durable data model

Implement the Phase A data model from §7 of the governing gateway plan, using fix-forward migrations after the existing migration sequence.

Implement the required conversation, message, build, turn, question, command, event, artifact, and approval-related records, excluding `deployments_gateway`, which belongs to Phase E.

Follow existing repository conventions.

Mandatory requirements:

- Tenant-scoped records.
- Composite tenant foreign keys where appropriate.
- Forced Row-Level Security.
- Explicit application-role grants.
- Consistent indexes and uniqueness constraints.
- Startup schema-version validation through the existing `LatestVersion()` mechanism.
- Migration up/down discipline.
- No application handler bypassing the trusted data-access and tool boundaries.

Treat Studio's database as the authoritative record. Sprite-local state must never become the only surviving copy of a business decision or conversation message.

### B. Conversation and build tools

Implement the appropriate `build.*` tools using the existing static registration and `runtime.Service.Invoke` architecture.

At minimum:

- Create a build.
- Append a user message.
- Answer an outstanding question.
- Cancel a build.
- Read build and conversation status.

All consequential state transitions belong in the tool/service layer, not HTTP handlers.

Handlers should authenticate, validate requests, and invoke the appropriate service operations.

Preserve server-derived tenant identity. Never accept browser-supplied tenant identity as authoritative.

Use the existing idempotency conventions for user submissions, answers, and other retriable mutations.

Implement the state machines defined in the governing plan. Reject invalid transitions deterministically.

Clarification, proposal, and authorization request must remain distinct interaction types.

An answer to a question must never silently grant authorization for a consequential operation.

### C. Versioned Gateway protocol

Implement the Studio-side protocol foundation from §8.

Use the versioned JSON envelope and event vocabulary defined in the governing plan.

Implement:

- Envelope parsing and validation.
- Canonical signing/verification interfaces.
- Timestamp and replay-window checks.
- Event identity and deduplication.
- Build and turn correlation.
- Unknown event handling.
- Callback acknowledgement semantics.
- Protocol-version validation.

Create a narrow key-resolution boundary.

For Phase A, use deterministic test-only key providers and fake signed callbacks. Do not distribute production HMAC keys or configure a real Builder Sprite.

No agent-provided text may be treated as instructions to Studio, SQL, or shell commands.

### D. Authenticated callback receiver

Implement the callback endpoint specified in the plan.

It must:

- Verify signatures before accepting events.
- Validate build and tenant binding.
- Reject malformed and replayed requests.
- Deduplicate events.
- Persist accepted events durably.
- Enforce allowed state transitions.
- Return appropriate acknowledgements and error statuses.
- Preserve unknown valid event types as opaque stored events rather than silently discarding them.

Test forged signatures, timestamp violations, replayed event IDs, duplicate delivery, cross-tenant references, and malformed payloads.

The fake Gateway must exercise the same callback endpoint and verification path that Phase B will use.

### E. Server-Sent Events

Implement authenticated SSE delivery from Studio to the browser.

Required functionality:

- Initial conversation state.
- New messages.
- Agent progress and messages.
- Questions and proposals.
- Build status changes.
- Artifact-ready notifications.
- Reconnection and durable replay.
- Browser disconnect handling.
- Studio restart recovery.

**Important architectural verification:**

The gateway plan describes message sequences per conversation and event sequences per build. Verify that these can support a single, globally ordered conversation stream.

A single `Last-Event-ID` must not silently lose or misorder events because two independent sequence spaces collide or advance differently.

Establish a monotonic, durable conversation-stream cursor covering both user messages and agent events. Prefer the smallest correct extension of the planned schema rather than introducing an unnecessary parallel event subsystem.

Test replay across mixed message and agent-event traffic.

SSE must render only persisted records. No important event may exist only in an in-memory notification.

### F. Conversation UI

Implement a functional, minimal conversation experience in the existing server-rendered UI architecture.

The user must be able to:

- Open a conversation.
- Submit a message.
- See the persisted response/history.
- Receive agent progress.
- See an agent clarification question.
- Answer the question.
- See the resulting continuation status.
- Reconnect and recover the same conversation.

Use the existing template and server-rendering approach. Do not introduce an unrelated frontend framework.

Keep the user experience focused. Do not build a workflow canvas or generalized task-management interface.

### G. Audit and observability

Extend the existing audit architecture.

Record relevant actions with:

- Tenant.
- Actor.
- Conversation/build identifiers.
- Event and correlation identifiers.
- Action and result.
- Timestamp.

Redact secrets and sensitive material.

Do not log HMAC keys, OAuth tokens, raw credentials, or unfiltered sensitive agent output.

## 5. Mandatory architectural discipline

Investigate the complete execution path before changing code.

Do not merely make tests green by bypassing the actual invariant.

Preserve ownership boundaries:

- HTTP handlers do not independently implement business state transitions.
- Tenant identity is server-derived.
- RLS remains an additional database security boundary.
- All tool invocations preserve tenant context.
- Audit records remain durable.
- Idempotency protects retries.
- Agent events never authorize their own actions.

Add regression tests that fail before and pass after each substantive correction.

Cover duplicate delivery, replay, concurrency, race conditions, process restarts, and failure handling wherever applicable.

If an existing architecture boundary cannot safely support Phase A, identify the root cause and the smallest correct correction at its owning layer.

Do not modify Miniapp merely to simplify Studio implementation. Any demonstrated general framework requirement must be documented and handled through the existing §14 approval process.

## 6. Explicit exclusions

Do not:

- Provision or destroy Fly.io Sprites.
- Install OpenCode.
- Install or configure the Fly Sprites OpenCode plugin.
- Perform OAuth authentication.
- Implement the Agent Gateway service.
- Implement `opencode_adapter.go`.
- Perform P-E1–P-E3 or P-G0–P-G4 live proofs.
- Enable live GHL, monday.com, WhatsApp, SMS, or calendar integrations.
- Enable consequential external business writes.
- Resolve F1.
- Resolve or bypass G-01.
- Modify the P5 scope or Invoice Review Desk golden path.
- Implement Lead-to-Booking.
- Introduce DBOS or another workflow engine.
- Modify frozen historical plans or evidence.
- Commit, push, publish, or destroy any existing user resources.

OpenCode-specific live proofs belong to their corresponding later phases. Phase A must prepare the Studio-side interfaces that those proofs will exercise.

## 7. Implementation sequence

Execute in this dependency order:

1. Inspect the current architecture and run baseline gates.
2. Finalize the Phase A schema and protocol decisions.
3. Implement migrations and tenant/RLS protections.
4. Implement build and conversation service/tool operations.
5. Implement callback verification and durable ingestion.
6. Implement conversation-wide event ordering and SSE replay.
7. Implement conversation and question interfaces.
8. Complete audit and redaction wiring.
9. Run the complete acceptance and regression suite.
10. Produce the Phase A evidence report.

Do not begin Phase B in this assignment.

Do not pause for intermediate section-by-section approval. Complete the authorized Phase A scope in one continuous implementation pass unless a genuine foundational or structural blocker prevents safe progress.

## 8. Acceptance criteria — Phase A exit gate

Phase A is accepted only when the following are demonstrated.

### Functional proof

Using the fake Gateway:

1. A business user creates a conversation.
2. The user submits a business requirement.
3. Studio persists the message and creates a build assignment.
4. The fake Gateway sends a signed progress event.
5. Studio validates and persists the event.
6. The browser receives the event through SSE.
7. The fake Gateway sends a signed clarification question.
8. Studio persists the question and parks the relevant turn.
9. The browser displays the question in the original conversation.
10. The user answers the question.
11. Studio validates and persists the answer.
12. The fake Gateway receives the continuation command.
13. The fake Gateway returns a completion event and artifact metadata.
14. Studio persists the result.
15. The browser displays the completed history and artifact record.

### Recovery proof

- Restart Studio during an open question.
- Reconnect the browser with its previous SSE cursor.
- Replay all required messages and events.
- Confirm that the conversation renders consistently.
- Confirm that the answer remains applicable.
- Confirm that duplicate callbacks are acknowledged without duplicate effects.

### Security proof

- Cross-tenant reads and writes rejected.
- RLS enforcement tested.
- Forged callback rejected.
- Replayed callback rejected or idempotently acknowledged without reapplication.
- Unauthorized answer rejected.
- Expired/superseded question answer rejected with the current state.
- Untrusted event text rendered inert.
- No secrets in logs, SSE, or persisted agent messages.
- No unauthorized transition bypassing service/tool boundaries.

### Engineering gates

Run and report:

- `gofmt`.
- `go vet ./...`.
- `go test -p 1 ./...`.
- Applicable race tests.
- Migration up/down tests.
- PostgreSQL integration tests.
- Tenant isolation and callback security tests.

Use the repository's actual test infrastructure and established P4 discipline. Distinguish executed tests from inspected code.

### Definition of Phase A done

The scripted-fake conversation must render identically after Studio restart and browser reconnection.

All newly introduced state-transition edges must have explicit tests, with full transition-edge coverage.

The protocol envelope must be stable and documented before Phase B begins.

The Phase A evidence report must be committed as a project artifact in the working tree, but **do not create a Git commit**.

No live Sprite must have been touched.

## 9. Final report

At completion, report:

1. Implemented capabilities.
2. Exact files and migrations changed.
3. Data model and protocol decisions.
4. Tests actually executed and their results.
5. Security and tenant-isolation evidence.
6. Recovery and replay evidence.
7. Any deviations from the governing plan and the concrete reasons.
8. Remaining risks or unverified assumptions.
9. Whether the Phase A exit gate is fully satisfied.
10. Whether Phase B can safely begin.

If a genuine blocker remains, provide exact evidence, root cause, invariant breached, durable correction, and required regression test.

Do not manufacture a blocker from optional improvements or unverified assumptions.

**The objective is a durable Studio-owned conversation foundation ready for the OpenCode Builder Sprite integration — not a premature implementation of the Sprite integration itself.**

## **How we will drive the project from here**

I will use this development cadence:

1. **Implementation:** coding agent executes one authorized phase.
2. **Evidence review:** I inspect the deliverable against that phase's acceptance gates.
3. **Focused correction:** only structural or demonstrated invariant failures return for correction.
4. **Phase decision:** accept and authorize the next phase, or identify a specific blocker.

After Phase A, I will review its evidence and decide whether to authorize Phase B. We will not reopen the overall architecture or conduct another general planning exercise.