I consolidated the Gemini and DeepSeek reviews, removing duplicated arguments, resolving their disagreements, and checking their recommendations against the current architecture and the completed Phase A closure.

The important distinction: **some criticisms remain valid, some have been overtaken by completed work, and some recommendations would weaken or unnecessarily redesign the approved architecture.**

The consolidated position is below.

# Consolidated Adversarial Review

## Miniapp Studio — Multi-Tenant Architecture, Agent Gateway, and Fly.io Sprites

**Date:** November 25, 2026\
**Review basis:** `writeup_review.md` (Gemini and DeepSeek reviews), the governing Agent Gateway implementation plan, the OpenCode amendment, and the completed Phase A evidence.

## 1. Executive verdict

The two reviews converge on an important conclusion: **the Miniapp Studio control-plane / execution-plane architecture is appropriate for a multi-tenant SaaS platform.**

The separation between Studio, Builder Sprites, and Runtime Sprites is not unnecessary complexity. It establishes distinct ownership and trust boundaries for business data, agent execution, and production applications.

However, the architecture needs a stronger articulation of its multi-tenant operating model and several operational controls before it is ready for broad production use.

The consolidated verdict is:

| Area                                               | Assessment                                           |
| -------------------------------------------------- | ---------------------------------------------------- |
| Multi-tenant control-plane architecture            | Sound                                                |
| Studio as authoritative business record            | Sound                                                |
| Dedicated Builder Sprite per application workspace | Correct direction; lifecycle policy needs definition |
| Separate Runtime Sprite                            | Sound and necessary                                  |
| Phase A security and recovery                      | Completed and accepted                               |
| HMAC protocol                                      | Retain; explain its specific security purpose        |
| Durable callback outbox                            | Retain                                               |
| OpenCode question/answer integration               | Critical live proof still pending                    |
| Gateway/agent Unix identity separation             | Correct design; live provisioning proof pending      |
| Event vocabulary                                   | Retain v1; no redesign justified                     |
| SSE resource limits                                | Valid outstanding operational concern                |
| Tenant fleet lifecycle                             | Missing operational specification                    |
| Cost and resource metering                         | Missing operational specification                    |
| Full end-to-end agent execution                    | Not yet demonstrated                                 |

The architecture does not need another redesign. It needs disciplined execution of the approved phases and a small number of explicit operational requirements.

---

## 2. Findings that are valid and should be incorporated

### 2.1 State the multi-tenant model explicitly

Both reviews correctly identify that the original technical writeup does not explain the relationship between tenants, applications, Builder Sprites, and Runtime Sprites clearly enough.

This should be corrected.

The governing architecture already describes Builder Sprites as per-application and long-lived across builds. The writeup should make that relationship explicit:

> Miniapp Studio serves multiple business owners through one central control plane. Each tenant may have multiple application workspaces. Each application workspace receives its own dedicated, long-lived Builder Sprite, reused across builds. Runtime environments are separate from Builder environments and are provisioned for approved applications. No Builder Sprite is shared across tenants.

This establishes the intended isolation model:

- One Studio control plane.
- Multiple tenant workspaces.
- Application-scoped Builder environments.
- Separately controlled Runtime environments.
- Tenant-scoped database records and permissions.

The exact provisioning, retention, and decommissioning policies still need to be defined, but the architecture should not leave the Builder ownership model ambiguous.

**Disposition: Accept. Add to the governing architecture documentation.**

### 2.2 Treat every agent-generated output as untrusted

The reviews correctly point out that secret-prefix scrubbing cannot guarantee that arbitrary secrets will never appear in agent output.

Phase A now has write-path sanitization for recognized credential families, including embedded spans, and read-path defense-in-depth. The evidence explicitly describes the detection patterns as non-exhaustive.

That is the correct claim.

The system should not present the scrubber as a comprehensive Data Loss Prevention system.

The broader security rule should be:

- Agent-generated text is untrusted.
- Agent event payloads are untrusted.
- Artifact manifests are untrusted.
- Agent-generated source code is untrusted.
- Test reports are untrusted until independently validated.
- Logs and diagnostic excerpts must be treated as potentially sensitive.

Phase D must validate artifact manifests, paths, sizes, and digest relationships. It must also prevent secrets from being exposed in review pages, operational logs, or other tenant-visible output.

There is an important distinction between artifact integrity and content sanitization: the original artifact bytes used for digest verification must not be silently modified by a redaction pass. Store and verify the artifact securely, then apply appropriate controls to what is displayed or logged.

The `app.invocations` framework persistence residual also remains explicitly disclosed. Miniapp is not being modified as part of this work.

**Disposition: Accept. Carry into Phase B logging and Phase D artifact review.**

### 2.3 Validate the operating-system boundary between Gateway and agent

The separate `studio-gw` and `studio-agent` Unix identities are a sound design decision.

The Gemini review correctly identifies that this boundary can fail through implementation mistakes even if the intended file permissions look correct.

Phase B provisioning should verify:

- The gateway process actually runs as `studio-gw`.
- The agent process actually runs as `studio-agent`.
- The agent cannot read Gateway HMAC keys and receives `EACCES`.
- Gateway key directories are not writable by the agent.
- The shared `studio-io` directory is narrowly scoped.
- The agent cannot escape its assigned checkout through unintended directory permissions.
- The gateway does not accidentally pass privileged file descriptors or capabilities to the agent.
- File handoff is atomic and validated rather than trusting arbitrary filenames or partial writes.

For the shared inbox, use restrictive directory permissions and strict schemas. File creation should use temporary files and atomic rename, with symlink/path traversal defenses.

This is a provisioning and security-acceptance requirement, not a reason to abandon the two-user design.

**Disposition: Accept. Include as mandatory Phase B security proofs.**

### 2.4 Strengthen stale-event and turn ownership guarantees

The distributed-state criticism is valid.

The system has multiple persistent state locations:

1. Studio PostgreSQL.
2. Builder filesystem.
3. Gateway callback outbox.
4. OpenCode session files.

These are not four equal sources of truth.

Studio PostgreSQL is authoritative. The Sprite filesystem contains execution state that must be reconciled against Studio records.

The protocol already has build, turn, message, and event identifiers, per-build ordering, deduplication, and state-guarded transitions. The implementation should now prove that these mechanisms also protect against delayed callbacks from an obsolete turn.

For example:

1. Turn A is active.
2. The agent asks a question.
3. The user answers.
4. Turn A is superseded or closed.
5. A delayed callback from Turn A arrives after Turn B has started.

That event must never mutate Turn B's state.

The acceptance requirements should include:

- Every state-changing callback is bound to its tenant, build, and immutable turn identity.
- Commands are bound to their command identity and intended turn.
- Duplicate events remain idempotent.
- Late events from a superseded or cancelled turn cannot reopen or mutate current state.
- Out-of-order events may be retained for historical evidence but cannot bypass transition guards.
- Gateway sequence numbers are interpreted with gap-tolerant semantics.

An explicit numeric epoch or generation counter should be introduced only if immutable turn identities and current command bindings are insufficient. Do not add a new schema concept without demonstrating the need.

**Disposition: Accept. Strengthen Phase B/C regression and concurrency tests without prematurely redesigning the protocol.**

### 2.5 SSE connection limits are a real multi-tenant concern

The original Phase A review identified that SSE streams have no maximum lifetime or concurrent-stream budget.

This remains valid.

One-second polling and cursor replay provide useful correctness properties, but they do not establish that the design is safe under hundreds or thousands of concurrent connections.

The service should eventually implement:

- Per-tenant and per-operator connection limits.
- Maximum stream lifetime.
- Bounded database polling.
- Reconnect backoff.
- Protection against reconnect storms.
- Metrics for active connections, poll load, disconnects, and replay volume.

This should be measured rather than addressed by introducing unnecessary fan-out infrastructure immediately.

**Disposition: Accept. Carry M5 into Phase B operational hardening and before broad multi-tenant availability.**

### 2.6 Define Builder fleet lifecycle management

This is one of the strongest new findings in the reviews.

The architecture explains how an assigned Builder Sprite executes work, but it does not fully specify the lifecycle of all tenant Builders as the SaaS platform grows.

The design should eventually answer:

| Lifecycle concern     | Required policy                                                  |
| --------------------- | ---------------------------------------------------------------- |
| First Builder         | When is a tenant's application workspace provisioned?            |
| Reuse                 | How is its Builder reused across subsequent builds?              |
| Multiple applications | How are multiple Builders assigned and identified?               |
| Idle state            | When does a Builder sleep?                                       |
| Retirement            | What happens when an application is deleted?                     |
| Tenant departure      | How are all Builders and Runtime environments deprovisioned?     |
| Compromise            | How is a Builder isolated and its access revoked?                |
| Audit                 | How are tenant, application, Sprite, build, and time correlated? |

A reasonable initial operating recommendation is lazy provisioning: create the Builder when the tenant first requests an application build rather than allocating one at signup.

That is a recommendation, not yet a locked provisioning decision.

A Builder retirement policy must also account for pending callbacks, agent session retention, uncollected artifacts, and the need to preserve Studio's authoritative history.

For a suspected compromise, the control plane must be able to quarantine the Builder, revoke its credentials and grants, remove integration access, and preserve the required forensic evidence without allowing the compromised environment to affect other tenants.

**Disposition: Accept. Add a tenant fleet lifecycle section before scaling the platform. Basic resource cleanup and Builder identity binding remain Phase B concerns.**

### 2.7 Introduce resource metering and cost accountability

The second major new operational finding is cost.

A multi-tenant platform cannot operate an expanding fleet of Sprites without measuring who consumes the resources.

The system should eventually record resource usage by tenant, application workspace, and build.

Relevant measures include:

- Sprite provisioning and runtime hours.
- Awake versus sleeping time.
- Storage consumption.
- Number of concurrent builds.
- Agent execution and model usage where available.
- Callback volume and retry activity.
- Artifact transfer volume.
- Connector/API usage where measurable.

The control plane should be able to enforce resource quotas and concurrency limits and identify tenants that are consuming disproportionate resources.

This does not mean implementing a full billing system in Phase B. It means designing the operational data so that a cost model can be built without retrofitting the entire fleet architecture.

**Disposition: Accept. Introduce instrumentation progressively; billing and pricing remain separate product decisions.**

---

## 3. Findings that are valid concerns but do not justify the proposed changes

### 3.1 File-drop should not automatically replace native OpenCode Q&A

DeepSeek argues that the file-drop handoff should become the primary question mechanism rather than a fallback.

The underlying concern is valid: OpenCode's question behavior under noninteractive `opencode run` is not yet proven.

However, the recommendation to reverse the priority is not established by evidence.

A file-drop mechanism has its own failure modes:

- Partial writes.
- Duplicate question files.
- Stale answers.
- Incorrect turn association.
- Agent failure to follow the file-writing instruction.
- Resume attempts that read an obsolete answer.

It is not automatically more reliable than a native structured event.

The existing OpenCode amendment has a better evidence-driven approach:

1. Test native question-event capture and same-session answer consumption.
2. If it works, use that mechanism.
3. If it cannot satisfy the contract, test the structured file-drop fallback.
4. If neither works, stop and return the unresolved issue rather than inventing another interface.

Only one Q&A mechanism should become the selected production path for the initial adapter. The system should not maintain two equally active protocols merely because both were considered.

**Disposition: Keep the existing primary/fallback order. Make the decision from P-G3 evidence.**

### 3.2 HMAC needs a better explanation, not immediate removal

The HMAC critique is worth incorporating into the technical writeup.

The existing protocol uses per-build HMAC keys. The rationale should be stated more precisely.

HTTPS provides transport security. Timestamps and message deduplication address freshness and duplicate processing. HMAC adds application-level message authenticity and integrity using a build-scoped signing secret.

The important security property is that a process without the relevant signing key cannot forge a valid callback simply by knowing the build identifier or possessing an unrelated application credential.

The separate `studio-gw` and `studio-agent` identities are important here: the agent must not be able to read the Gateway's signing key.

Per-build key derivation also limits the set of messages that can be authenticated by a single key.

There is an equally important limitation: HMAC does not protect a build whose Gateway signing identity or key material has itself been compromised.

The correct conclusion is therefore:

- Retain per-build HMAC.
- Document the precise threat it mitigates.
- Keep keys inaccessible to agent processes.
- Keep TLS.
- Keep timestamp and replay checks.
- Preserve tenant/build binding and event deduplication.
- Do not claim HMAC solves full Builder compromise.

A bearer-token alternative would require a deliberate threat-model and key-management decision, not an untested simplification.

**Disposition: Accept the request for clearer justification. Retain the existing protocol.**

### 3.3 A full vertical slice is the correct destination, but not a reason to collapse the phases

Both reviews emphasize proving the entire system with one real tenant before scaling.

That is correct as an acceptance philosophy.

However, collapsing the five phases into one large implementation effort would remove useful failure boundaries.

The existing sequence deliberately isolates different unknowns:

- Phase B: Can the Gateway run and supervise an agent?
- Phase C: Can the actual agent participate in a genuine Studio conversation?
- Phase D: Can its output be verified and reviewed?
- Phase E: Can approved work be safely promoted?

Each phase should deliver a small executable vertical proof and close its own risks.

The eventual end-to-end slice is:

Prompt → Builder wake → OpenCode execution → question → answer → same-session resume → artifact verification → human approval → Runtime deployment.

This is the right destination, but it should be reached through the existing phase gates.

**Disposition: Accept the vertical-slice principle. Retain the five-phase sequence.**

---

## 4. Findings to reject or treat as obsolete

### 4.1 Phase B before Phase A closure — reject

DeepSeek's recommendation to implement Phase B before closing Phase A is no longer applicable.

More importantly, Phase A's closure was not dependent on a live Sprite.

The outstanding defects concerned Studio-owned data handling, PostgreSQL policy enforcement, and server lifecycle behavior. Those were directly tested and corrected with red-before/green-after evidence.

The closure is now complete:

- Migration 000016 enables RLS on all ten Phase A tables.
- Application-role behavioral tests prove tenant isolation.
- Write-path sanitization prevents recognized secret spans from entering Studio-owned agent-message surfaces.
- The genuine shutdown/restart lifecycle is tested.
- M1–M4 have been addressed.
- PostgreSQL 19 and 17 gates, race tests, and migration round-trip pass.

Phase A is accepted. The governing sequence now proceeds to Phase B.

**Disposition: Withdraw as an open finding. No phase reordering.**

### 4.2 RLS is still disabled — obsolete

This was a major valid finding in the earlier review.

It has been resolved by migration 000016 and the catalog plus application-role tests.

It must not be carried forward as an open defect. The continuing rule is that all future tenant-owned tables must enable and force RLS, with catalog coverage kept in lockstep.

**Disposition: Resolved.**

### 4.3 Collapse the event vocabulary from eleven events to three — reject for now

The event vocabulary is not merely a collection of UI messages.

`agent.question`, `agent.proposal`, and `agent.authorization_request` represent different business and security semantics.

In particular, an authorization request must never be mistaken for a normal clarification.

Collapsing them into one generic event with a subtype field may be possible internally, but there is no demonstrated benefit that justifies changing the frozen v1 protocol before the live integration exists.

Typed event names make the contract explicit and help prevent accidental state-machine conflation.

A common internal handler can still reduce duplicate implementation without changing the external vocabulary.

**Disposition: Retain the v1 event names. No protocol redesign without evidence of actual implementation pain.**

### 4.4 Remove the durable callback outbox in favor of SSE — reject

The suggestion to replace the callback outbox with a long-lived SSE connection or ordinary polling confuses different communication responsibilities.

SSE is the browser-facing Studio-to-user delivery mechanism.

The Gateway callback outbox is the durable Builder-to-Studio delivery mechanism.

The Sprite may sleep, the Gateway may restart, and the Studio endpoint may temporarily be unavailable. A durable outbox allows the Gateway to retain an event until Studio acknowledges it.

A long-lived stream does not by itself provide that persistence and recovery guarantee.

Polling can be useful as a status reconciliation mechanism, but it is not a substitute for durable callback delivery.

**Disposition: Retain the outbox and retry/acknowledgment design.**

### 4.5 Estimated implementation savings and delivery times — omit

The reviews propose percentage savings and implementation-time estimates.

There is insufficient evidence to establish those figures. They should not influence architectural decisions.

**Disposition: Exclude from the consolidated technical assessment.**

---

## 5. Consolidated engineering requirements by phase

The reviews should produce actionable requirements rather than a second independent roadmap.

| Phase   | Consolidated requirement                                                                                                                                                   |
| ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Phase A | Accepted. Maintain RLS enforcement, write-path sanitization, durable conversation history, and true restart regression coverage.                                           |
| Phase B | Prove Gateway installation, command handling, durable outbox, process supervision, OpenCode bootstrap, OS identity separation, and real Sprite sleep/wake.                 |
| Phase C | Prove question/answer/same-session resume; stale-turn fencing; reconnect, duplicate, delayed callback, cancellation, and restart behavior.                                 |
| Phase D | Treat manifests, transcripts, code, and reports as untrusted; verify artifact digests; validate MiniappSpec; prevent automatic deployment; scrub displayed/logged content. |
| Phase E | Promote only approved immutable digests into separate Runtime Sprites; verify isolation, health, rollback, and resource cleanup.                                           |

### Phase B additions

Before Phase B acceptance, the following evidence should exist:

1. Gateway runs as a persistent Sprite service.
2. A real signed status request and response succeeds.
3. The assigned Builder identity is bound to the correct tenant and application workspace.
4. OpenCode runs as the unprivileged agent identity.
5. Agent-context attempts to read Gateway keys fail with `EACCES`.
6. The Fly MCP grant is limited to the assigned Builder.
7. The command processor deduplicates retries.
8. The callback outbox survives gateway restart and delivers after recovery.
9. Sprite sleep/wake behavior is measured.
10. All provisioning and test resources are accounted for and cleaned up.

### Phase C additions

The event and command paths must be tested against stale and delayed events.

A late event from a prior turn must not alter the active turn, reopen a completed build, or silently change an authorization outcome.

The same agent session must demonstrably consume the user's answer, rather than merely recording that the answer was sent.

### Phase D additions

Artifact handling should be content-sensitive.

The system needs to distinguish:

- Original artifact bytes, whose digest establishes integrity.
- Structured manifests, which require schema and path validation.
- Human-visible summaries and transcripts, which require safe rendering and redaction.
- Logs and diagnostics, which must not become a secret-exfiltration channel.

An agent's claim that a test passed is not evidence. Studio must inspect the collected test report and apply its own acceptance rules.

### Phase E additions

Builder and Runtime isolation must be enforced, not just described.

A production connector binding must never be available to the Builder merely because the Runtime belongs to the same tenant.

Promotion must be approval-gated and digest-pinned. Rollback must redeploy a previously approved digest, not reconstruct a previous application from an uncontrolled Builder filesystem.

---

## 6. Multi-tenant operations: requirements before broad availability

The reviews uncovered two omissions that deserve a dedicated architecture section.

### 6.1 Tenant fleet lifecycle

The platform should eventually define:

- Lazy Builder provisioning.
- One Builder per application workspace.
- Reuse across builds.
- Idle sleep behavior.
- Resource retirement.
- Tenant deletion and deprovisioning.
- Artifact/session retention.
- Builder compromise quarantine.
- Credential and grant revocation.
- Cross-tenant fleet audit.

These are operational lifecycle policies. They do not require an immediate change to the Phase B protocol.

### 6.2 Cost, quotas, and fairness

The system should record usage by tenant and application workspace.

At minimum, the architecture should anticipate:

- Builder and Runtime Sprite hours.
- Storage usage.
- Concurrent build limits.
- Agent/model execution usage where available.
- Callback and retry volume.
- Artifact storage and transfer.
- Connector/API usage where measurable.

The control plane should eventually be able to impose tenant quotas and prevent one tenant's runaway build from consuming disproportionate fleet capacity.

This is necessary for multi-tenant operation, but it should not turn the Gateway implementation into a premature billing subsystem.

---

## 7. Final consolidated decisions

| Review subject                    | Final decision                                            |
| --------------------------------- | --------------------------------------------------------- |
| Multi-tenant framing              | Make explicit                                             |
| Studio as permanent control plane | Retain                                                    |
| Per-application Builder isolation | Retain and formalize                                      |
| Separate Runtime Sprite           | Retain                                                    |
| RLS finding                       | Resolved by Phase A closure                               |
| Secret scrubbing limitations      | Disclose and strengthen later artifact/log handling       |
| Separate Unix users               | Retain; prove during Phase B                              |
| Shared handoff folder             | Retain with strict filesystem and atomic-write controls   |
| HMAC                              | Retain; explain the build-scoped threat model             |
| Durable callback outbox           | Retain                                                    |
| Stale event/turn fencing          | Strengthen tests; no speculative schema redesign          |
| Native OpenCode Q&A               | Test first                                                |
| File-drop Q&A                     | Keep as tested fallback, not an assumed superior solution |
| Eleven event types                | Retain frozen v1 vocabulary                               |
| Phase B before Phase A closure    | Reject; Phase A is accepted                               |
| SSE limits                        | Carry as Phase B hardening                                |
| Builder fleet lifecycle           | Add operational policy                                    |
| Cost and metering                 | Add progressive instrumentation                           |
| Phase architecture                | Retain A → B → C → D → E                                  |

## 8. Final architectural conclusion

The two reviews do not reveal a need to abandon or simplify the core Miniapp Studio architecture.

They reveal that the architecture needs to be described and operated as a multi-tenant SaaS platform, rather than as a single business owner's private coding environment.

The control-plane / execution-plane split is justified. The Builder and Runtime separation is justified. The durable conversation and callback model is justified. The HMAC protocol, process isolation, and typed interaction model are defensible when their actual security purposes are made explicit.

The main remaining engineering challenge is now empirical: prove that the Gateway can supervise a real OpenCode process in a real Builder Sprite, survive sleep and restart, and support a genuine question-and-answer cycle without compromising the Studio trust boundary.

The next phase should not redesign the system. It should prove the system.

**Phase A is accepted. Phase B is the next authorized milestone.**

The wider multi-tenant lifecycle, cost, and operational controls should be incorporated before the platform is exposed to broad production usage, without expanding the current Phase B deliverable into a premature billing or fleet-management project.

The principal corrections are: Phase A's RLS/restart/secret findings are now resolved; the proposal to reorder phases is withdrawn; the durable outbox and typed v1 events remain; and tenant fleet lifecycle plus cost metering are added as genuine longer-term gaps. The source review's multi-tenant and distributed-state concerns remain useful.&#x20;

writeup_review\.md

writeup_review\.md

&#x20;The accepted Phase A closure is reflected in the current evidence record.&#x20;

gateway-phase-a-evidence(1).md
