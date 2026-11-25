# Miniapp Studio — Refreshed Product and Engineering Roadmap

**Revision date:** 25 November 2026  
**Status:** Phase A accepted; Phase B Part 1 authorized; Phase B live execution requires a separate explicit GO.  
**Governing architecture:** `plans/agent-gateway-implementation-plan.md`, incorporating the OpenCode amendment and the decisions consolidated through 25 November 2026.

---

## 1. Product vision

Miniapp Studio is a **multi-tenant operating platform for governed AI-assisted business work**.

A business owner describes a business process in ordinary language. Studio helps turn that requirement into a working application, asks for clarification instead of silently guessing, presents verifiable implementation evidence, obtains human decisions where required, and eventually deploys an approved application. The owner interacts through one durable Studio conversation rather than managing terminals, coding-agent sessions, or cloud infrastructure.

The intended customer value is not simply code generation. It is the ability to create, inspect, govern, operate, and evolve software that fits the customer's business process without having to become a software engineer or surrender control to an opaque agent.

### Binding application invariant

> **Miniapp is the only application framework. Go is the only implementation language for Miniapp applications. MiniappSpec is the canonical application specification, and the Miniapp framework/runtime is the only application execution model.**

This is a hard product and engineering constraint, not a coding-agent preference.

Every AI-generated Miniapp must:

- Be represented by a valid MiniappSpec.
- Be implemented in Go using the Miniapp framework and its supported contracts.
- Pass the existing Miniapp validation and admission boundary.
- Be built and tested through the supported Go/Miniapp toolchain.
- Execute through the Miniapp framework/runtime.
- Be deployed only as an approved, verified Miniapp-compatible artifact.

Python, JavaScript, TypeScript, Node.js, React, Django, or another language, framework, or runtime must not become an alternative way to implement or execute a Miniapp. A coding agent cannot choose a different application stack. Infrastructure, gateway, provisioning, and development tooling may use other technologies where appropriate, but those tools must not become part of or bypass the Miniapp application runtime.

This invariant must be enforced programmatically at artifact validation, admission, build, test, and deployment boundaries. Agent instructions alone are insufficient.

## 2. Architectural model and ownership

Miniapp Studio serves multiple business owners from one central control plane. Each tenant may have multiple application workspaces. The intended isolation model is **one dedicated, long-lived Builder Sprite per application workspace**, reused across builds; Builder Sprites are not shared across tenants. Each deployed application receives a separate Runtime Sprite.

| Component | Owns | Must not own |
|---|---|---|
| **Miniapp Studio** | Tenant-scoped conversations, requirements, builds, turns, questions, answers, commands, events, approvals, artifacts, deployment records, user interface and audit correlation | The coding agent's process-local truth |
| **Miniapp framework** | Application framework, MiniappSpec interpretation, tool invocation, policy, tenant context, consequential capability and audit boundaries | Studio's business conversation and Builder lifecycle |
| **Builder Sprite** | Assigned application checkout, development dependencies, OpenCode session files, build/test work, Agent Gateway and temporary artifacts | Authoritative business history or production credentials |
| **Agent Gateway** | Signed command handling, agent supervision, session bookkeeping, event translation, callback outbox, artifact collection | Human authorization or independent business authority |
| **Coding agent (initially OpenCode)** | Producing Go/Miniapp implementation work inside the assigned Builder workspace | Choosing an alternative application framework/runtime or granting itself permissions |
| **Runtime Sprite** | Running the approved Miniapp application artifact | Coding agent, Builder checkout, Gateway command surface or unrestricted development tools |
| **Fly Connectors** | Controlled provider-call routing and credential custody | Implicit authorization for an agent or Builder |

There are two conversations:

1. **Business conversation:** user ↔ Studio. Permanent, authoritative, replayable and tenant-owned.
2. **Execution conversation:** Studio ↔ Gateway ↔ coding agent. Internal protocol and agent-specific state.

The user never connects directly to the Sprite terminal. The agent's native transcript is not the customer's permanent business record.

### Communication paths

- Browser → Studio: authenticated HTTP.
- Studio → Browser: Server-Sent Events (SSE) with cursor replay.
- Studio → Builder Gateway: authenticated HTTPS commands.
- Builder Gateway → Studio: authenticated HTTPS callbacks.

No permanent Studio–Sprite socket or browser–Sprite connection is required. Commands may wake a sleeping Builder. The Gateway runs as a Sprite **service** because exec/TTY processes do not survive Sprite sleep; the coding agent is relaunched and its session reconstructed from persistent files.

## 3. Security and governance invariants

These apply across all phases.

### 3.1 Tenant and workspace isolation

- Every Studio record is tenant-scoped; RLS must be both **ENABLED and FORCED** on every tenant-owned table.
- Catalog tests must cover every new tenant-owned table, and behavioral tests must use the restricted application role.
- Commands and callbacks are bound to the tenant, application workspace, build and turn. Untrusted identity hints are revalidated against Studio records.
- One Builder belongs to one application workspace and must not access another tenant's Builder or Runtime.
- Builder compromise must not provide lateral access to other tenants or production environments.

### 3.2 Gateway authentication and key isolation

Retain the v1 HMAC-SHA256 envelope and per-build key design. HTTPS protects transport; HMAC provides application-level authenticity and integrity so a process without the relevant build key cannot forge a valid command or callback merely by knowing a build identifier. Per-build key scope limits the potential forgery domain, but does not protect a Gateway whose own signing identity or key has been compromised. Timestamp freshness and deduplication address replay and duplicate processing; they are distinct controls, not benefits supplied by HMAC alone.

The Gateway and coding agent must run under separate OS identities (for example, `studio-gw` and `studio-agent`). The agent must receive `EACCES` when attempting to read Gateway signing keys. A narrowly scoped `studio-io` handoff may exchange validated artifacts or question/answer files; it must not expose key material. Provisioning fails closed if process separation cannot be enforced.

### 3.3 Builder privilege and connector boundaries

- The OpenCode Fly MCP grant is scoped to the assigned Builder, with zero Sprite-creation allowance and explicit denials for destroy, network-policy mutation and checkpoint restore.
- Production connector bindings never enter Builder Sprites.
- Runtime Sprites receive only the approved production connections required by that application.
- Prompts and policy instructions are not security boundaries; grants, OS permissions, Studio authorization checks and runtime admission enforce the boundary.

### 3.4 Human control

Clarification, proposal and authorization remain distinct concepts. An agent can ask, recommend, or request permission; it cannot grant itself permission. Consequential business-effect writes remain disabled while G-01 is unresolved. F1 remains deferred and must not be bypassed by the Gateway.

### 3.5 Untrusted content and secret handling

Agent output, event payloads, manifests, source code, transcripts and test reports are untrusted. Phase A write-path sanitization covers recognized secret patterns, but is deliberately non-exhaustive. Do not claim it is complete DLP. Keep read-path sanitization, redact operational logs, and apply content-sensitive handling to artifacts in Phase D.

The Miniapp framework's `app.invocations` input/result persistence is a documented residual; Miniapp is unchanged. No Studio tool may select invocation inputs/results into user-visible responses.

---

## 4. Current status

### Phase A — Conversation Infrastructure: ACCEPTED

Phase A is closed as of 25 November 2026. The closure evidence records:

- Migrations `000011`–`000016`; RLS enabled and forced on all ten Phase A tables.
- Catalog and application-role tests proving RLS enforcement, including unscoped reads and cross-tenant write rejection.
- Write-path secret sanitization before event persistence and question/artifact derivation, with embedded-canary tests through database, tools, page rendering and SSE.
- Genuine ordered shutdown → fresh boot recovery, including parked-question answer, history/SSE replay and duplicate callback idempotency.
- M1–M4 completed: consistent lock order and deadlock/error-classification correction, unused code removal, strict trailing-data parsing, and corrected sequence documentation.
- Formatting, vet, build, full tests on PostgreSQL 19.0 and 17.12, race tests, and migration round-trip through v16 reported green.

M5–M8 remain carried as recorded in `docs/gateway-phase-a-evidence.md`:

- M5: SSE lifetime and concurrency budgets — Phase B operational hardening.
- M6: `app.invocations` raw input/result residual — disclose; never expose invocation inputs in responses.
- M7: `artifacts.manifest` sensitivity — Phase D must scrub/validate before display or logging.
- M8: evidence wording/count discipline — keep reports exact and auditable.

No live Sprite, OpenCode, plugin, or OAuth integration was exercised in Phase A. The evidence record is `docs/gateway-phase-a-evidence.md`.

### Phase B — Sprite Agent Gateway: NEXT, PART 1 AUTHORIZED

The approved two-part execution plan is `plans/prompt9_imp_plan.md`. **The live/fake gate is explicit:** Part 1 is authorized; Part 2 is not.

#### Part 1 — fake-backed implementation (authorized)

1. Record the clean baseline and reverify pinned OpenCode/plugin facts against current sources. Do not float versions silently. If the ecosystem has moved, report the verified facts and apply the amendment's pinning rules.
2. Implement the Studio gateway client and Gateway command-handler skeleton. Prove signed command round trips against an in-process fake. Implement wake-on-command as retry behavior over the existing Fly boundary.
3. Implement per-build command deduplication and a durable on-disk callback outbox with fsync, acknowledgment tracking, retry/backoff and jitter. Test duplicates, loss, ordering and replay.
4. Implement process supervision, session bookkeeping, the agent-neutral adapter interface, and the initial OpenCode adapter against scripted doubles. Translate structured NDJSON events; do not scrape terminal output or invent CLI behavior.
5. Implement the static Linux Gateway build/install path and security/configuration artifacts: dual-user layout, HMAC key provisioning drop, pinned OpenCode configuration, `guidance: false`, Builder-specific instructions, exact permission rules, and Fly grant parameter sheet. Exercise local provisioning behavior through `FakeSpriteManager` and scripted service/sleep/kill behavior.
6. Publish the Phase B Part 1 report, capability matrix (executed versus unknown), full gates, and exact owner-run live runbook. Then STOP.

Part 1 must not provision or mutate a live Sprite, perform OAuth, or use live Fly credentials.

#### Part 2 — live installation and recovery (owner-run; separate explicit GO required)

The owner (or owner's authorized shell) executes the exact runbook after a separate GO. Credentials and OAuth consent remain owner-held; do not provide Fly credentials in chat or assume the assistant's sandbox can reach Fly.

Live sequence:

1. Provision the assigned Builder Sprite.
2. Install the Gateway as a persistent Sprite service.
3. Verify an authenticated signed `status` round trip.
4. Start a supervised OpenCode no-op turn.
5. Verify plugin configuration and effective Builder guidance.
6. Verify the per-Builder Fly grant and denial cases.
7. Verify agent-context access to HMAC key paths fails with `EACCES`.
8. Put the Sprite to sleep, wake it through a command, and report `agent.resumed` or an honest `agent.failed` result.
9. Verify Studio receives the signed callback.
10. Record timings, outputs, resource cleanup, and any explicitly owner-retained Builder.

No live step is authorized by Part 1 approval. No commits or pushes are authorized.

### Phase C — Genuine Two-Way Interaction

**Objective:** prove the complete real user ↔ Studio ↔ Gateway ↔ OpenCode interaction. No new architecture should be added merely to make the demo look complete.

Required proof:

1. User sends a requirement in Studio.
2. Studio assigns a build and wakes the correct Builder.
3. OpenCode starts in the workspace and emits structured progress.
4. OpenCode asks a real clarification question.
5. Studio persists and renders it in the original conversation.
6. User answers through Studio.
7. Gateway resumes the same OpenCode session.
8. OpenCode demonstrably consumes the answer as the question's resolution.
9. Work continues and completion/failure is honestly reported.
10. Studio history and SSE replay remain correct across disconnect/reconnect.

The critical OpenCode uncertainty is question behavior under noninteractive `opencode run`. Test native structured question-event capture first. If it cannot satisfy the contract, test the documented adapter-specific JSON file-drop handshake with strict schemas, atomic file writes, turn binding and stale-answer rejection. Select one working mechanism for the initial adapter; do not carry two production mechanisms without need. If neither proves out, stop and return the evidence for an owner decision. Do not parse prose, invent flags, or switch silently to an experimental API/PTy approach.

Also execute the recovery/failure matrix: agent kill, gateway restart with pending outbox, Sprite sleep/wake, Studio restart, duplicate/delayed/out-of-order callbacks, stale-turn events, cancellation, concurrent builds and unauthorized continuation. Extend tests for M5 stream lifetime/concurrency bounds where applicable.

Exit: documented live two-way interaction and recovery evidence, with no unresolved adapter capability misrepresented as working.

### Phase D — Miniapp Build, Artifact Verification and Review

**This phase enforces the locked Go/Miniapp invariant.**

The coding agent must produce an application in Go using the Miniapp framework and a valid MiniappSpec. It must not produce a Python/Node.js/React/Django or other alternative-framework application as the deliverable.

Required implementation and admission controls:

- Agent assignments explicitly instruct the agent to use Go and Miniapp exclusively for application code.
- Define a versioned artifact contract that identifies the MiniappSpec, Go source/module, supported Miniapp dependencies, build metadata, tests, and digests.
- Validate the MiniappSpec with the existing validator and admission boundary.
- Validate the implementation against an explicit allowlist of supported Go/Miniapp application dependencies and build/runtime requirements; reject unsupported frameworks, runtimes, or non-Go application implementations.
- Build and test the application using the supported Go/Miniapp toolchain and runner. A prompt or agent claim is not validation.
- Reject artifacts that bypass Miniapp admission or attempt to ship an independent application runtime.
- Preserve original artifact bytes for digest verification. Sanitize and safely render separate human-visible views and logs without altering the bytes whose digest is being verified.

Artifact pipeline:

1. Agent emits a manifest and SHA-256 digests; it does not claim that a manifest alone proves correctness.
2. Studio fetches pinned artifacts and verifies digests and size limits.
3. Studio validates the MiniappSpec and Go/Miniapp implementation.
4. Studio presents a unified diff, MiniappSpec, machine-readable test report, relevant transcript, provenance and validation results.
5. A claimed completion without a passing required test report is not accepted as success.
6. User can approve or request rework. There is no auto-deploy in Phase D.

Manifest paths, archives, symlinks, sizes, source content and logs are treated as untrusted. Add tests proving that non-Go or alternate-framework application artifacts are rejected, as well as tampered digests, malicious paths, oversize bundles, failing tests and secret-bearing display content.

Exit: a real small Go/Miniapp artifact is built, validated, reviewed, and explicitly approved or returned for rework. No deployment path is available from Phase D state.

### Phase E — Approved Deployment to Runtime Sprite

Promote only an explicitly approved, digest-pinned, verified Go/Miniapp artifact.

1. Record the human approval and artifact digest.
2. Provision a separate Runtime Sprite for the application.
3. Bind only the application's approved production connectors and network policy.
4. Transfer the verified artifact; do not rebuild from uncontrolled Builder state.
5. Install and run it through the Miniapp framework/runtime. Do not install Python/Node.js or another application runtime to execute the Miniapp.
6. Verify application health and record the Runtime identity, URL, digest and deployment status in Studio.
7. Test rollback to a previously approved digest.
8. Verify the Runtime contains no coding agent, Builder Gateway or development command surface.

A failed health check blocks marking the deployment live. A lost Runtime is recreated from the approved digest, not from a mutable Builder checkout. External business-effect tools remain subject to G-01 and their own acceptance gate.

---

## 5. Multi-tenant operational readiness (before broad production rollout)

These are real gaps identified by the consolidated reviews, but they do not expand the current Phase B deliverable into a billing or fleet-management project.

### 5.1 Builder and tenant lifecycle policy

Before broad tenant onboarding, define:

- Lazy Builder provisioning trigger and user-visible lifecycle.
- One Builder per application workspace and routing from Studio records.
- Idle/sleep policy and retention of checkout/session state.
- Application deletion, Builder retirement and artifact collection/drain sequence.
- Tenant offboarding and deprovisioning across Builders and Runtimes.
- Builder compromise quarantine, grant/key revocation, evidence retention and clean replacement.
- Audit correlation among tenant, application workspace, Sprite, build and timestamps.

A reasonable option to evaluate is lazy provisioning at the first build request, rather than allocating a Builder at signup. This remains a policy recommendation, not a locked behavior.

### 5.2 Usage, quotas and fairness

Instrument tenant/workspace-level usage before setting prices or hard quotas. Candidate measures include Builder and Runtime awake/sleep hours, storage, concurrent builds, agent/model usage where available, callback/retry volume, artifact transfer/storage and connector usage where measurable.

The control plane should eventually enforce configurable concurrency and resource ceilings and prevent one tenant's runaway workload from degrading service for others. Exact quotas, pricing and retention periods remain owner/product decisions.

### 5.3 SSE operational limits

Implement and measure per-tenant/operator stream budgets, maximum stream lifetime, bounded polling, reconnect backoff and reconnect-storm protection. Do not introduce fan-out infrastructure until measurements show it is needed.

---

## 6. Decisions retained from the consolidated review

| Subject | Decision |
|---|---|
| Multi-tenant control plane and execution-plane separation | Retain |
| One dedicated Builder per application workspace | Retain as the intended topology; formalize lifecycle policy before scale |
| Separate Runtime Sprite | Retain |
| Miniapp exclusivity + Go-only application implementation | Binding invariant |
| MiniappSpec as canonical specification | Retain |
| OpenCode as first adapter | Retain, subject to pin re-verification and live proofs |
| Native OpenCode Q&A | Test first; documented file-drop is fallback |
| HMAC v1 + per-build keys | Retain and document threat model/limits |
| Durable callback outbox | Retain; browser SSE is not a substitute |
| Typed v1 event vocabulary | Retain; no redesign without demonstrated need |
| Tenant/build/turn stale-event protection | Prove through Phase B/C tests; add epoch only if required by evidence |
| Phase A before B | Retain; Phase A is accepted |
| F1 and G-01 | Unchanged; no bypass |
| P5 golden path and scope | Unchanged |
| DBOS or another workflow engine | Not introduced without demonstrated requirement |
| Fleet cost/billing | Instrument progressively; separate product decision |

---

## 7. Scope boundaries and working discipline

- Phase B is the current priority; do not implement Phase C, D, or E early.
- Phase B Part 1 is fake-backed only. Part 2 is owner-run and requires a separate explicit GO.
- No production connector bindings in Builder Sprites.
- No consequential business writes while G-01 remains open.
- No F1 implementation or workaround.
- No P5 implementation or Lead-to-Booking implementation in this roadmap.
- No second coding-agent adapter or generalized multi-agent framework.
- No workflow canvas, marketplace, generic task manager, billing system, or new workflow engine as part of Phase B.
- No Miniapp framework changes unless a demonstrated framework-level defect is handled through the established approval process.
- No direct browser-to-Sprite channel or permanent Studio-to-Sprite socket.
- No automatic deployment before Phase E approval.
- No commits or pushes without explicit authorization.

Implementation prompts must require root-cause investigation, fixes at the correct ownership boundary, red-before/green-after regression tests, failure/retry/race coverage where relevant, and an honest distinction between inspected, simulated, and live-executed behavior. Never claim a fake-backed test proves a live Sprite capability.

## 8. Definition of done

The roadmap is complete when:

1. All five phase exits have accepted evidence records.
2. A real end-to-end conversation runs through Studio, the Builder Gateway and OpenCode, including a real question and same-session answer consumption.
3. Required sleep/wake, restart, callback, deduplication, cancellation and failure proofs have been executed.
4. Phase D demonstrably rejects non-Go and alternative-framework Miniapp implementations and admits only valid Go/Miniapp artifacts.
5. Artifact digests, MiniappSpec admission and test evidence are independently verified.
6. Only an explicitly approved artifact is promoted into a separate Runtime Sprite through the Miniapp runtime, with health verification and rollback.
7. Tenant isolation, Builder/Runtime boundaries and connector restrictions are demonstrated.
8. No unresolved F1/G-01 decision is bypassed.
9. Multi-tenant fleet lifecycle, resource metering, quota/fairness and SSE operating limits are defined before broad production rollout.
10. Live resource cleanup or explicitly owner-approved resource retention is recorded for each live phase.

**Immediate next action:** execute Phase B Part 1 only, then stop with the implementation report, capability matrix, full test evidence and owner-run live runbook. Seek a separate explicit GO before any live Part 2 action.