# Part 1 — Adversarial Review of `roadmap7.md`

## What the roadmap gets right

Before the critique: several things in `roadmap7.md` are genuinely correct and should survive into any plan.

1. **The three-workstream structure** (F0 foundations, minimum RPA primitives + W1 slice, larger A/B product roadmap) is the right shape. It separates shared infrastructure from product layers from engineering discipline.
2. **F0's progress is stated honestly** (increments 1–2 done, 3 in progress). This is unusual and welcome; most roadmaps pad status.
3. **"Don't build a giant automation platform before proving one useful automation"** is the correct prioritization. It is also, as I argue below, incompletely applied.
4. **The guardrails** (Neon-only, Sprites vs Machines, A/B/C/D categories, ADR-001 for D, human approval for consequential actions, no UI-RPA) are the correct set of non-negotiables.
5. **The A/B product split sharing one backend** is the correct framing. The roadmap does not confuse two information architectures with two backends.

None of these should be reopened. The plan I propose preserves them.

## Structural issues

### S1 — The "minimum RPA primitives" list mixes platform primitives with business capabilities (HIGH)

The roadmap's own section 2 lists:

> 1. Receipts and evidence
> 2. Contact management
> 3. Event intake
> 4. Scheduling
> 5. Business capabilities

Items 1, 3, and 4 are plausibly platform primitives. **Item 2 (contact management) is a business capability.** So is everything in item 5. The roadmap then lists W1's steps (contact upsert, estimate creation, invoice issuance, payment collection, message sending) as if these were the natural "business capabilities" layer — but the roadmap does not distinguish which of them are business capabilities (all of them) from what the *platform* provides (the mechanics to compose and execute them).

**Why this matters:** this is the same categorization error the user explicitly rejected in the prior turn. The plan cannot simultaneously (a) commit to process agnosticism and (b) list "contact management" as a platform primitive. If the plan ships with this list intact, either the platform bloats to include contact management, or the W1 demonstration has a conceptual gap (where does contact upsert live if not in the platform?).

**Correction:** rename the list to distinguish the two layers. "Contact upsert" is a *Miniapp capability* required by W1. "Event intake" and "scheduling" are *platform primitives*. "Receipts and evidence" is a *platform primitive already partially implemented*.

### S2 — W1 is sequenced *after* primitives, but the primitives' correctness is supposed to be validated *by* W1 (HIGH)

The roadmap says:

> "Complete the minimum RPA primitives and deliver an actual working business workflow"

and then lists primitives 1–5, followed by:

> "Then comes the first major end-to-end demonstration. W1 — Lead-to-Payment"

This sequencing is the classic "build infrastructure, then validate it" pattern — the one the prior roadmap critique flagged as the source of the 18-month timeline. The correct sequencing is the opposite: **W1 forces the primitives to emerge**, not the other way around. Each primitive gets built because W1 requires it, tested in W1's context, and validated by W1 running end-to-end. The plan should state this explicitly.

**Correction:** reorder as "W1 drives the minimum primitives." The primitive list becomes a *forecast* of what W1 will likely require, not a *prerequisite* to be built before W1 starts. Some primitives will turn out to be unnecessary (cut them); some will turn out to be under-specified (extend them); and at least one will surprise you (accept that).

### S3 — Skills are named in the vision but absent from the roadmap (HIGH)

The user's vision (and the prior turn's restated vision) makes Skills a central concept:

> **Skills → Automation → Miniapp → Capability → Connector**

`roadmap7.md` does not mention Skills. The roadmap proceeds as if the authoring path is "founder writes prompts → agent writes Go" — the M1-only mode from the prior turn. But the vision depends on Skills as a first-class authoring input.

**Why this matters:** if Skills are not in the roadmap at all, the plan will build Miniapps and Automations but not the mechanism that lets agents *reuse* business knowledge across processes. The "process-agnostic" claim weakens because each new Automation will re-derive its business logic from prompts, not from reusable Skills.

**Correction:** introduce Skills as a roadmap item. The minimum viable Skills are markdown files that agents consume when proposing Automations. This is not a runtime; it is an authoring input. It is cheap to add and expensive to retrofit.

### S4 — Webhooks are named in the vision but absent from the roadmap (HIGH)

The user's prompt adds webhooks as first-class platform abstractions. `roadmap7.md` mentions "event intake" as a primitive but does not specify webhooks. W1's first step is "Customer sends an inquiry" — that is an inbound webhook (or a poll). The roadmap's "event intake" primitive is under-specified: does it mean webhook ingress, polling, or both? Are they the same primitive or two different ones?

**Why this matters:** W1 cannot be demonstrated without an event-intake mechanism, and if webhooks are the mechanism, their security, idempotency, and routing must be specified before W1 begins. Retrofitting webhook security after W1 works is a known source of production incidents.

**Correction:** state explicitly that event intake has two distinct sub-mechanisms (inbound webhook delivery and polling/scheduled pull), that inbound webhooks are first-class (not just "events"), and that W1 requires at least one of them to be functional.

### S5 — The prerequisite chain from Roadmap A to W1 is unstated (MEDIUM)

The roadmap says Roadmap A "must continue progressing without waiting for Operations Studio" and lists B0–B5. But it does not say which of B0–B5 W1 actually requires. Reading the roadmap as written, an agent could reasonably start B3 (per-PR previews) before W1 because "Roadmap A must continue."

**Why this matters:** W1 requires B1 (local dev loop) and possibly B2 (remote agent). It does **not** require B3, B4, or B5. B3–B5 are valuable but are not on W1's critical path. If the roadmap does not say this, resources will be split.

**Correction:** add an explicit "W1 critical path" table: what W1 needs from F0, from A, and from what does not yet exist. Everything else is deferred until W1 ships.

### S6 — The "process-agnostic" claim has no acceptance test (MEDIUM)

The roadmap states the goal ("process-agnostic") but does not define a test that would prove the goal was achieved. "Process-agnostic" is unfalsifiable as stated.

**Why this matters:** Without a test, every future feature debate becomes an argument about whether it "feels process-agnostic." With a test, the conversation is concrete.

**Proposed test:** *"After W1 (Lead-to-Payment) is deployed, build W2 (Missed-call-to-booked-job) using only: new Miniapp implementations, new connector adapters, new Skills, and new Automation definitions — **zero changes to the platform runtime, zero changes to the platform abstraction set.**"* If W2 requires a platform change, the abstraction set was incomplete. If W2 requires zero platform changes, the abstraction set is validated for at least two processes. This is the test that makes process-agnosticism provable.

### S7 — F0 Increment 3 (invitations) is in progress, but its necessity for W1 is unstated (LOW)

Invitations are for tenant onboarding. W1 is one tenant, one demonstration. Does W1 need invitations? If W1 is a demo run by the founder for a design partner, invitations are unnecessary overhead. If W1 requires the design partner to have their own tenant, invitations are required.

The roadmap does not resolve this. It proceeds with Increment 3 as if it is on the critical path without saying so.

**Correction:** either state "Increment 3 is required for W1 because [reason]" or "Increment 3 is required for R1 (tenancy), not for W1; it is currently being built in parallel and W1 does not block on it."

## Summary of recommendations for the roadmap

1. **Fix the primitive/capability confusion in the minimum RPA list.** Rename "Contact management" and "business capabilities" as Miniapp capabilities, not platform primitives.
2. **Reorder so W1 forces the primitives**, not the reverse.
3. **Add Skills to the roadmap.** Skills are the authoring input that makes the vision coherent. Even minimal markdown-based Skills are worth building now.
4. **Add webhooks to the roadmap as first-class ingress.** Specify that "event intake" has two mechanisms (webhook, poll) and that W1 requires one.
5. **State the W1 critical path** through Roadmap A explicitly. W1 needs B1 and probably B2; it does not need B3–B5.
6. **State the process-agnosticism acceptance test** (W2 built with zero platform changes).
7. **Clarify Increment 3's relationship to W1.**

With these seven corrections, the roadmap is buildable against. Without them, the plan risks bloating the platform to include business operations — the exact failure the vision is trying to avoid.

---

# Part 2 — The Plan

Per the prompt in `prompt37.md`, the plan produces two artifacts: the architecture specification and an adversarial review of that specification. The specification preserves the vision — process-agnostic, minimal primitives, Skills as authoring input, webhooks first-class — and addresses the structural issues identified above.

---

## Artifact A — `plans/rpa-platform-grand-vision.md`

### 1. Executive vision

Miniapp Studio is a **process-agnostic, API-first, multi-tenant RPA platform**. A business owner describes a business process in natural language. An AI agent, guided by reusable Skills, proposes a declarative Automation composed of typed capabilities. Studio validates the Automation against registered Miniapps and connector attestations, executes it under framework-owned authorization, effect management, and evidence recording, and returns outcomes to the owner.

The platform understands **how** to execute a business process. It does not understand **what** any particular business process means. An invoice, a contact, a campaign, and a construction job are all just capabilities invoked by Automations that know what those capabilities mean; the platform does not.

### 2. Goals, non-goals, governing principles

**Goals:**
- Enable a business owner to describe any API-centric business process and have it executed with framework-enforced reliability.
- Support thousands of business capabilities without requiring platform changes per capability.
- Provide multi-tenant isolation, human authorization for consequential actions, effect idempotency, and durable evidence.
- Make webhook-triggered, scheduled, manual, and API-triggered processes equally first-class.
- Allow agents to author, propose, and revise Automations without owning authorization or execution.

**Non-goals (for the current architecture):**
- Classic UI/screen-scraping RPA (deferred to a separate capability with its own security ADR).
- A visual workflow canvas or drag-and-drop editor.
- A proprietary agent runtime; existing coding and operational agents are consumed as-is.
- A large catalog of predefined business applications (CRM, ERP, accounting engine).
- A workflow engine that duplicates the Miniapp framework's A/B/C/D execution model.

**Governing principles:**
1. **Process-agnostic.** The platform must not enumerate business operations. If a proposed feature is a business operation, it is a Miniapp capability, not a platform primitive.
2. **Capability-driven.** Every executable operation is a typed, versioned capability.
3. **API-first.** External integrations are API/webhook/event-based. No UI scraping.
4. **Agent-native, not agent-trusting.** Agents author Automations; the framework authorizes and executes them.
5. **Reuse over invention.** Existing Miniapp framework contracts (A/B/C/D, effect ledger, approvals, idempotency, tenant isolation) are reused, not reimplemented.
6. **Compose, don't extend.** New processes are new compositions, not new platform primitives.

### 3. Glossary and conceptual model

**Skill** — Reusable business knowledge, procedural guidance, and decision rules. Consumed by agents when authoring Automations. Not executable. Not authority-bearing. Versioned and discoverable.

**Automation** — A versioned declarative definition of a business process: business objective, triggers, steps, control flow, gates, capability references, policies. The central business-facing concept.

**Miniapp** — A versioned executable application implemented in Go within the Miniapp framework. Exposes one or more capabilities. Contains domain logic. Does not know the whole process it participates in.

**Capability** — A typed, versioned, invokable operation with declared input/output, authorization, trust mode, execution class, idempotency, and provider provenance. The contract through which Automations invoke Miniapps.

**Connector** — The replaceable integration boundary between a capability and an external provider. Declares which capabilities it supports, which it does not, and under what constraints.

**Platform primitives** — The small set of generic abstractions (see §5) that describe and execute any process.

**Platform services** — Cross-cutting infrastructure (identity, tenancy, credentials, persistence, observability). Not business capabilities.

**Relationship:** A Skill guides an agent to author an Automation. The Automation composes Capabilities. Capabilities are implemented by Miniapps or connectors. The platform executes the Automation using framework primitives and services.

### 4. The Skill → Automation → Miniapp → Capability → Connector model

This is the authoring chain. It is not a runtime chain — each layer is a separate artifact with its own lifecycle.

**Skill (knowledge)**
- Authored by the platform team or the tenant's business owners.
- Contains: business objective, process knowledge, decision rules, examples, safety checklists, required capability references.
- Consumed by agents during authoring, never during execution.
- Versioned and discoverable via a Skill catalog.
- Does not grant execution authority.

**Automation (definition)**
- Authored by agents (interactively with the owner, or autonomously with owner review).
- Contains: business objective, trigger declarations, steps (capability references + control flow), gates (approvals), policies (trust mode, limits), evidence expectations.
- Versioned independently of Miniapps and Skills.
- Validated against registered capabilities before activation.
- Immutable once activated; changes require a new version.

**Miniapp (implementation)**
- Authored by coding agents or developers.
- Contains: Go code, tests, fixtures, manifest, MiniappSpec.
- Exposes one or more Capabilities.
- Classified by execution category (A/B/C/D per tool).
- Independent lifecycle from the Automations that use it.

**Capability (contract)**
- Declared by Miniapps and connectors.
- Contains: identity, version, input/output schema, authorization requirements, trust modes, execution class, idempotency, provider availability.
- Stable across Miniapp versions that preserve the contract.

**Connector (adapter)**
- Provides capabilities against a specific external provider.
- Declares: supported capabilities, unsupported capabilities, provider-specific restrictions, authentication method.
- Replaceable: an Automation using `party.contact.upsert` should not care whether it runs against HubSpot or Salesforce, subject to declared differences.

### 5. Minimal platform abstractions, including first-class webhooks

The platform's abstraction set is **eight concepts**. Each is generic; none is a business operation.

**A. Process / Automation definition** — what should happen. Declarative. Versioned. Contains steps, control flow, gates, capability references.

**B. Step / Capability invocation** — a typed call to a registered capability. Carries input mapping, output mapping, error handling.

**C. Trigger** — how an Automation starts or resumes. Five sources:
- Manual (owner or agent initiates)
- API (external caller invokes)
- Scheduled (cron-style)
- Internal event (a platform-domain event, e.g., "Automation X completed")
- **Inbound webhook** (external provider sends HTTP)

**D. Webhook (first-class)** — inbound webhook ingress is its own primitive. Distinct sub-concepts:
- **Endpoint** — a tenant-registered URL owned by the tenant, not the Automation. Persistent across Automation versions.
- **Delivery** — one incoming HTTP request. Durable, deduplicated.
- **Verification** — signature validation against the tenant's stored secret.
- **Event** — the normalized, validated payload (post-verification).
- **Routing** — determining which tenant, which Automation, and which trigger invocation the event maps to. Derived from the endpoint's registration, not from the payload.
- **Invocation** — starting or resuming the mapped Automation.

**Outbound webhooks are not this primitive.** Outbound webhook delivery (notifying external systems of platform events) is an *effect* (see G below) and is delivered via capabilities. It does not require a new platform primitive.

**E. Control flow** — minimal generic semantics:
- Sequence
- Conditional branch (on deterministic capability output or agent output)
- Bounded iteration
- Wait for event / wait for time
- Timeout
- Completion / failure
- **Not** a workflow DSL. Not a canvas. The set is deliberately small and extensible only when a new use case cannot be expressed.

**F. Gate / Authorization** — human approval boundary. Reuses existing framework approval tables and material-hash binding. Applies per capability invocation according to the Miniapp's declared trust mode.

**G. Effect** — framework-owned handling of external side effects. Reuses existing effect ledger. Idempotency identity, attempt lifecycle, UNKNOWN quarantine, reconciliation.

**H. Evidence / Receipt** — append-only execution record. Records decisions, capability invocations, external effects, webhook deliveries, automation state transitions. Distinct from operational state and provider-authoritative state.

**Cross-cutting services (not abstractions, but required):**
- Identity and authentication
- Multi-tenancy and delegation
- Authorization and RBAC
- Credential broker and secrets management
- Persistence (Neon)
- Scheduling infrastructure
- Observability and audit
- Environment isolation

### 6. Platform versus business capability boundary (anti-bloat doctrine)

**This section is enforceable architectural law.** Every proposal for a new platform abstraction must answer:

1. Is this truly generic across unrelated business processes?
2. Can it be expressed using the existing eight abstractions?
3. Does it introduce business semantics into the platform core?
4. Does it solve a concrete cross-process reliability or security problem?
5. Is its absence demonstrably a limitation, not a preference?

If any answer is "no," the proposal is a business capability. It belongs in a Miniapp, not in the platform.

**Examples of business capabilities (never platform primitives):**
- Contact upsert
- Estimate creation
- Invoice issuance
- Payment collection
- Message sending
- Campaign creation
- Review request
- Job synchronization
- Appointment booking
- Financial analysis

There may eventually be thousands of these. The platform must never need a new primitive per business capability.

**Examples of platform primitives (the eight above):**
- Webhook ingress
- Effect idempotency
- Approval gates
- Capability invocation

The distinction is not "small vs large." It is "generic across unrelated processes vs specific to a business domain."

### 7. Process authoring lifecycle

1. Owner states a desired business outcome in natural language.
2. Agent (local coding agent or Studio-bound agent) retrieves applicable Skills from the tenant's Skill catalog.
3. Agent identifies required capabilities by querying the capability registry for the tenant's connected providers.
4. Agent proposes an Automation definition, including: steps, capability references, control flow, triggers, gates, trust modes, limits.
5. Platform validates the Automation:
   - Every capability reference resolves to a registered capability.
   - Every capability is attested as available for the tenant's connected providers.
   - Every gate is bound to a valid authorization class.
   - Every control flow construct is one of the eight abstractions.
   - Every capability's execution class is compatible with the Automation's declared constraints.
6. Agent produces a human-readable process explanation and an explicit list of unsupported requirements (capabilities the tenant has not connected, or features the platform does not provide).
7. Owner reviews and authorizes activation. Authorization is durable and bound to the specific Automation version.
8. Runtime executes the Automation per its trigger.
9. Studio presents run evidence, approvals, and exceptions.
10. Agent can propose a revision as a new Automation version; no in-place mutation.

### 8. Runtime and execution semantics

Automations are composed of capabilities. Each capability has an execution class (A/B/C/D per the Miniapp framework). **The Automation's class is the maximum class of its constituent capabilities within each parallel branch**; capabilities of class D within an Automation inherit D-class durability, and an Automation that mixes A/B/C capabilities with at least one D capability executes the D portion durably and the rest per their declared class.

No second workflow engine. Class D capabilities use DBOS per ADR-001. Class A/B/C use existing substrates. The Automation layer orchestrates between capability invocations — it does not orchestrate inside them.

**Durable execution** is a property of the substrate, not the Automation. An Automation that composes class D capabilities can wait for days because the class D substrate keeps the execution alive; the Automation layer simply tracks which step it is on.

### 9. Webhook, event, polling, scheduler relationship

These are four distinct mechanisms serving four distinct purposes:

| Mechanism | Direction | Source | Use case |
|---|---|---|---|
| **Inbound webhook** | External → platform | Provider pushes | Real-time event from Stripe, GitHub, etc. |
| **Polling** | Platform → external | Provider has no push | Check a mailbox every 5 min |
| **Scheduled trigger** | Platform → platform | Cron | Run a report at 2 AM |
| **Internal event** | Platform → platform | Another Automation | Automation B runs when A completes |

They are not interchangeable. An Automation can be triggered by any combination. The choice depends on what the external provider offers, not on platform preference.

### 10. API-first architecture

**Public APIs (per tenant, authenticated):**
- Automation management: create, validate, activate, suspend, retire, list.
- Automation versions: create, list, promote.
- Execution: invoke manually, cancel, query status.
- Approvals: list pending, decide.
- Capabilities: discover (what's registered, what's attested for this tenant's providers).
- Connectors: connect provider, list connections, revoke.
- Webhook endpoints: register, list, rotate secret, delete.
- Evidence: query runs, receipts, effect outcomes.

**Internal APIs (platform-only):**
- Webhook delivery ingestion: verified, deduplicated, persisted, dispatched.
- Effect ledger: intent, attempt, settle, reconcile.
- Approval table: request, decide, expire.
- Credential broker: lease, revoke.

Every public API uses the same domain contracts as Studio's interfaces and agents. There is no "agent API" separate from the "Studio API." Same authorization model, same tenancy scoping, same evidence recording.

### 11. Identity, tenancy, security, governance

**Identity and principals:**
- Human principals (owners, operators, approvers)
- Agent principals (coding agents, operational agents)
- Service principals (the platform itself, connectors)

Authorization is against the human authority principal, not the agent. Agents act on behalf of humans; humans hold authority.

**Tenancy:**
- Organization → Tenant → Workspace
- RLS keyed on tenant, transaction-local context
- Tenant-scoped: Automations, capabilities, connector credentials, webhook endpoints, evidence
- Explicit, revocable delegation (agency pattern)

**Credentials:**
- Tenant-provided, stored in the credential broker, never exposed to agents or Automations
- Short-lived leases issued at execution time
- Webhook verification secrets stored per tenant endpoint, rotatable

**Environment isolation:**
- Development, Preview, Staging, Production
- Distinct Neon branches, distinct credentials, distinct webhook endpoints
- Promotion by immutable artifact digest

**Trust mode:**
- dry-run / confirm / autonomous
- Per capability, per tenant
- `confirm` gates execution through the approval table

**Three independent dimensions:**
- Execution category (A/B/C/D)
- Compute placement (Machine / Sprite / Hybrid / Local)
- Trust mode (dry-run / confirm / autonomous)

These are orthogonal. Never conflated.

### 12. Studio A/B product relationship

**Roadmap A — Engineering Studio**
- For developers and coding agents
- Author Miniapps, define contracts, develop, test, deploy previews
- Critical path for W1: B1 (local loop), possibly B2 (remote agent)

**Roadmap B — Automation Operations Studio**
- For business owners, operators, managers
- Discover Skills and capabilities, create Automations, configure triggers and webhooks, manage connections, monitor executions, resolve exceptions, approve actions
- Critical path for W1: a minimal "create Automation, monitor run, approve gate" experience

**Shared backend:** Both Studio experiences call the same public APIs, use the same tenancy model, the same credential broker, the same capability registry, the same evidence store. **No duplicated business logic.**

**Separate information architectures:** The two Studios have different primary users, different primary tasks, and different navigation. They share infrastructure, not UI.

### 13. Lead-to-Payment as architectural proof, not platform catalog

W1 must be expressible entirely through the eight platform abstractions plus Miniapp capabilities. **None of the following is a platform primitive:**

| W1 step | Layer | What it is |
|---|---|---|
| Customer sends inquiry | **Webhook** (platform primitive) + capability | Inbound webhook triggers an Automation |
| Qualify and identify customer | **Capability** (Miniapp) | `party.contact.upsert`, `party.contact.lookup` |
| Prepare estimate | **Capability** (Miniapp) | `commercial.estimate.create` |
| Owner reviews and approves | **Gate** (platform primitive) | Approval table with material hash |
| Send estimate | **Capability** (Miniapp) + **Effect** (platform primitive) | `engagement.message.send` via connector, with idempotent effect |
| Capture acceptance | **Webhook** (platform primitive) | Customer acceptance via another webhook |
| Schedule job | **Capability** (Miniapp) | `workforce.job.schedule` |
| Invoice | **Capability** (Miniapp) | `commercial.invoice.create` |
| Payment | **Capability** (Miniapp) + **Effect** (platform primitive) | `commercial.payment.collect` via connector, with idempotent effect |
| Request review | **Capability** (Miniapp) | `engagement.review.request` |
| Record evidence | **Evidence** (platform primitive) | Append-only receipt |

Every business operation is a capability. Every platform operation is one of the eight abstractions. No exceptions.

### 14. Unrelated-process generality examples

**Employee onboarding:** webhook (HR system sends new hire) → capability (identity create) → gate (manager approval) → capability (asset provision) → capability (welcome message) → evidence. Same eight abstractions.

**Order fulfillment:** webhook (order placed) → capability (inventory reserve) → capability (fulfillment schedule) → capability (shipment confirm) → evidence. Same eight abstractions.

**Marketing campaign:** scheduled trigger → capability (audience select) → capability (content generate) → gate (compliance approval) → capability (send across channels) → evidence. Same eight abstractions.

If any of these required a new platform primitive, the abstraction set would be incomplete. The roadmap's acceptance test (below) is designed to catch this.

### 15. Dependency-ordered roadmap

**Phase 0 — F0 foundations (largely complete)**
- Increment 1: secrets ✅
- Increment 2: credential broker ✅
- Increment 3: invitations — **deferred** for W1 (not on the critical path), required for R1

**Phase 1 — Minimum primitives, driven by W1**
Do not build a primitive list. Build W1. Add primitives as W1 demands them:
- Webhook ingress (required by step 1) — **first primitive to build**
- Capability registry query API (required by step 2)
- Effect ledger for message send and payment (already partially exists)
- Approval gate (already exists)
- Evidence / receipts (extend existing)
- Basic control flow — sequence, conditional (sufficient for W1)

**Phase 2 — W1 implementation**
- Author Miniapps: `party.contact`, `commercial.estimate`, `commercial.invoice`, `commercial.payment`, `engagement.message`, `engagement.review`, `workforce.job`
- Connect providers: email/CRM/payments/accounting sandbox
- Author W1 Automation: the lead-to-payment process definition
- Author W1 Skill: the process knowledge that guides an agent to author W1

**Phase 3 — W1 demo and evidence**
- W1 runs end-to-end in a sandbox environment
- Studio shows: run history, gate decisions, effect receipts, exceptions
- Owner approves through Studio

**Phase 4 — Process-agnosticism acceptance test (the load-bearing test)**
- Build W2 (Missed-call-to-booked-job) using only:
  - New Miniapp capabilities
  - New connector adapters
  - New Skills
  - New Automation definitions
- **Zero platform changes. Zero new platform primitives.**
- If W2 requires a platform change, Phase 1's primitive set was incomplete — extend it and re-run W2.

**Phase 5 — Operations Studio R-series**
Only after W1 and W2 validate the primitive set:
- R1 (tenancy lifecycle)
- R2 (Automation catalog and versioning UI)
- R3 (full execution runtime)
- R4 (operations workspace)
- ... and so on

Roadmap A's B3–B5 continue in parallel, not on W1's critical path.

### 16. Open decisions and deferred topics

**Open (require owner decisions):**
- D-1: Neon topology for multi-tenant data (project-per-tenant vs shared with RLS)
- D-2: UI-RPA scope (deferred, but stated explicitly)
- D-3: Skills storage and discovery mechanism (filesystem-based vs database-backed)
- D-4: Webhook endpoint naming and routing scheme
- D-5: W2's identity (which second process validates process-agnosticism)

**Deferred:**
- Visual workflow canvas
- Marketplace of pre-built Automations
- Multi-region deployment
- Billing and metering
- Metering per capability

### 17. Architecture acceptance criteria

The specification is successful if, after reading it, a new engineer can answer all of the following without ambiguity:

1. What is a Skill and what is it not?
2. What is an Automation and how does it relate to a Miniapp?
3. What is a Capability and how is it different from a Connector?
4. What are the eight platform primitives?
5. Where do webhooks enter, and what is the difference between a webhook and an event?
6. Where does agent reasoning live, and where does deterministic logic live?
7. How are long-running processes handled without introducing a second workflow engine?
8. How does a new business capability get added without a platform change?
9. How does a new Automation get added without a runtime change?
10. How does a new provider get added without rewriting the Automation?
11. How is tenant isolation enforced?
12. What happens if an agent tries to execute an unregistered capability?
13. What happens if a webhook is delivered twice?
14. What happens if an external effect times out with unknown outcome?
15. What is the process-agnosticism acceptance test?

### Architecture tests (scenario-based)

The specification is validated by these scenarios, each of which must be answerable as "yes, this works per the architecture":

- Introduce a new business process (e.g., employee onboarding) using existing primitives.
- Introduce a new capability via a Miniapp without new primitives.
- Introduce a new provider via a connector without changing the Automation.
- Introduce an inbound webhook trigger without embedding provider-specific logic in the Automation.
- Demonstrate an unsupported provider capability honestly (the Automation fails with a typed error, not a silent substitution).
- Demonstrate that an agent cannot execute an unregistered capability.
- Demonstrate recovery from a duplicate webhook delivery.
- Demonstrate recovery from an ambiguous external effect (UNKNOWN → reconcile → settle).
- Demonstrate that an approval survives process suspension and restart.

If any of these cannot be answered from the specification, the specification is incomplete.

---

## Artifact B — `plans/rpa-platform-grand-vision-review.md`

### Verdict

**APPROVED WITH AMENDMENTS.** The specification preserves the vision, defines the eight platform primitives, and includes the process-agnosticism acceptance test that turns "process-agnostic" from an aspiration into a checkable property. The specification has three substantive weaknesses and four smaller ones that should be addressed before implementation begins.

### Blockers

**B1 — "Execution class of an Automation" is underspecified for mixed-class compositions.**

Section 8 states that "the Automation's class is the maximum class of its constituent capabilities within each parallel branch." This is not sufficient. Consider an Automation that:
1. Reads a record (class A)
2. Sends a message (class B, effect)
3. Waits for customer response (class D — durable wait on webhook)
4. Sends a follow-up (class B)

The Automation as a whole must be durable through step 3. But steps 1, 2, and 4 are not themselves class D. What substrate does the Automation execute on? The current spec says "the D portion executes durably and the rest per their declared class," but this implies a compositional substrate that neither the Miniapp framework nor DBOS provides out of the box.

**Failure scenario:** the first implementation will run the whole Automation as class D (safe but wasteful) or as class B (unsafe because the wait is lost). The spec doesn't prevent either.

**Correction:** the spec must either (a) state that any Automation containing a class D capability is itself a class D workflow, or (b) specify how a class D orchestration layer composes class A/B/C capabilities. Option (a) is simpler and correct; the spec should adopt it explicitly.

**B2 — Agent step authority boundary is underspecified.**

Section 7 says agents "propose" Automations but doesn't define what an agent can do at execution time. Can an agent make a decision mid-Automation (e.g., "classify this lead as hot/warm/cold")? If yes, what authority does it have? If no, the Automation must be fully deterministic, which contradicts the "agent-native" claim.

The specification says agents author Automations but never resolves whether agents execute decisions inside Automations.

**Correction:** add a §7.1 "Agent steps within Automations." Define the agent-step contract:
- Declared inputs
- Validated structured outputs (JSON schema, not free text)
- Authority class (advisory / decision / action-request)
- If decision-class: the decision is logged as evidence and does not bypass subsequent gates
- If action-class: the action is a capability invocation subject to the same gates
- Never ambient access to credentials or unrestricted external effects

Without this, the spec either prohibits agent reasoning (losing the agent-native value) or allows unconstrained agent actions (losing the safety).

**B3 — Webhook idempotency identity is underspecified.**

Section 9 distinguishes inbound webhooks from events and polling. Section 5.D distinguishes endpoint/delivery/event/routing/invocation. But the spec doesn't state what the idempotency identity of a webhook delivery is — which is the crux of safe duplicate handling.

**Failure scenario:** a provider retries a webhook delivery. The platform must recognize it as the same delivery and not re-invoke the Automation. But what makes two deliveries "the same"? The provider's request ID header? A hash of the payload? The endpoint + timestamp? Different providers offer different identity signals.

**Correction:** add §5.D.4 specifying:
- Delivery identity is derived from a provider-declared idempotency header where available
- Fallback is (endpoint ID, provider signature, payload hash)
- The delivery is persisted with its identity before any dispatch
- Duplicate deliveries (same identity) are acknowledged but not re-dispatched
- The dispatched Automation invocation carries the delivery identity as its idempotency key

Without this, the platform has no defense against webhook duplicates except "hope the provider doesn't retry."

### Significant (should be addressed before implementation)

**S1 — Skill versioning has no specified mechanism.**

Section 3 says Skills are versioned and discoverable, but the spec doesn't say how. Filesystem? Database? Git-based? Different Skills may need different persistence. Without a decision, implementations will diverge.

**Correction:** add §3.1 specifying that Skills are stored in the tenant's evidence/service store with versioned records, discoverable via API, and referenced by version from Automation authoring sessions.

**S2 — Capability discovery doesn't specify tenant-connected provider resolution.**

Section 6's authoring lifecycle says "agent identifies required capabilities by querying the capability registry for the tenant's connected providers," but doesn't specify what happens when:
- A capability is registered by a Miniapp but no connector is connected to a provider
- A capability is registered but its connector doesn't support that capability
- A capability is connected but the tenant's credentials are expired

The spec should specify a typed "capability unavailable" outcome distinct from "capability not registered."

**Correction:** add §6.1 specifying capability availability states (available, connected-but-unauthorized, registered-no-connector, unsupported-by-connector) and that Automation validation fails with the appropriate state.

**S3 — Control flow semantics are listed but not composed.**

Section 5.E lists sequence, branch, loop, wait, timeout. But it doesn't specify how they compose (can a wait happen inside a loop? can a branch contain a wait? what happens if a branch's condition can't be evaluated?). Different implementations will interpret differently.

**Correction:** add §5.E.1 specifying composition rules: loops may contain waits but must be bounded; waits inside branches require the branch to be resolved before the wait begins; timeouts bound the entire composition, not just individual steps. This is a small addition but prevents a class of compositional ambiguity.

### Smaller (defer or address in ADRs)

**m1 — No mention of capability deprecation.** If a Miniapp deprecates a capability, what happens to Automations using it? The spec should state at least: deprecated capabilities fail validation for new Automations but continue to execute for existing Automations until retired.

**m2 — No mention of Automation testing.** The spec describes validation but not testing. A testing story ("run this Automation against synthetic data before activation") is needed; deferred to ADR.

**m3 — Webhook retry semantics are not specified.** If the platform acknowledges a webhook but dispatch fails, what happens? Does the platform retry dispatch? With what backoff? This should be specified before the first webhook is accepted.

**m4 — Agent-authored Automations have no provenance metadata.** An Automation authored by an agent should record which agent, which model, which Skill versions were used, and which capability attestations were checked. This is evidence, not a new primitive.

### Summary of amendments

The specification must incorporate:

1. **B1 fix:** state that any Automation containing class D capability is itself class D.
2. **B2 fix:** add §7.1 defining agent-step authority boundaries.
3. **B3 fix:** add §5.D.4 defining webhook idempotency identity.
4. **S1 fix:** add §3.1 specifying Skill storage.
5. **S2 fix:** add §6.1 specifying capability availability states.
6. **S3 fix:** add §5.E.1 specifying control flow composition rules.
7. **m1–m4:** defer to ADRs but note them in §16 (open decisions).

With these amendments, the specification is complete and ready to guide implementation. Without them, the implementation will diverge in predictable ways — mixed-class Automations, ambiguous agent authority, duplicate webhook processing, and inconsistent control flow.

### What the specification gets right and should not reopen

- The eight-primitive set is minimal and correct.
- Webhooks as first-class ingress, distinct from events and polling, is right.
- The Skill → Automation → Miniapp → Capability → Connector model is coherent and preserves the vision.
- The anti-bloat doctrine ("business operations are not platform primitives") is correct and enforceable.
- The process-agnosticism acceptance test (W2 with zero platform changes) is the load-bearing test that makes the vision provable.
- The A/B split sharing one backend is correct.
- W1 as architectural proof (not platform catalog) is the right framing.

### Final recommendation

Approve with the seven amendments above. Re-review the amended specification once. Then begin Phase 1 (webhook primitive + minimum capability registry) and Phase 2 (W1 Miniapps) as the plan proposes. Defer all R-series work until W1 ships and the process-agnosticism test (W2) has been run.

The vision is sound. The specification is close. The three blockers are fixable in one amendment cycle. The plan is ready for amendment.