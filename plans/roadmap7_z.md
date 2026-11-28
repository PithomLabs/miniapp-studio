# Adversarial Review — Grand Vision & Plan (roadmap7.md)

| Field | Value |
|---|---|
| Reviewed material | `roadmap7.md`: the status brief (F0 increments 1–3, Mom-n-Pop sequence, W1 Lead-to-Payment), the grand-vision exchange (five layers + P1–P14 primitives → corrected platform abstractions → webhooks added), the two-experience roadmap (F0 / Roadmap A B0–B5 / Roadmap B R0–R8), and the recommended sequencing (§6 of the answer) |
| Stance | Per owner instruction: **the vision is not challenged** — process-agnostic, API-first, capability-driven, multi-tenant, agent-native, minimal core, webhooks first-class all stand. This review targets precision, contracts, and sequencing — the places where the vision would fail in execution |
| Method | Against the vision's own doctrine (anti-bloat, ADR-001 authority, reuse-first) and the §14 failure-mode list of the governing prompt. No invented findings; design-stage |
| Verdict | **APPROVED WITH AMENDMENTS** — the conceptual model is sound; 3 blockers and 4 high-severity contract/sequencing gaps must be closed before implementation planning |

---

## 1. What survives adversarial pressure

Stated plainly, because it constrains what amendments are allowed to touch:

- **The platform-versus-business-capability correction is the pivotal improvement.** The owner's objection (contact upsert, invoice issue, intake poll are business capabilities, not primitives) caught a real categorization error that would have bloated the core into a catalog of thousands of operations. The corrected boundary — the platform understands how to *execute and govern* a process, not what an invoice *means* — is the single most valuable property of the architecture and must be preserved verbatim.
- **Webhooks as a first-class trigger abstraction** with the six-way distinction (endpoint, delivery, verification, event, routing, trigger invocation) is correct and correctly separated from polling, cron, and internal events.
- **The anti-bloat doctrine** (five questions per proposed abstraction, composition over addition) is enforceable as written.
- **ADR-001 remains the authority for Class D; no assumed DBOS adoption; no second orchestration engine** — the guardrails are the right ones and are retained.
- **Reuse-first is real, not rhetorical:** approvals, effects, intent-hash, `CapabilityRef`, manifest contracts, and the credential broker (F0 increment 2, Neon-verified) already exist and the vision reuses them rather than duplicating.
- **The two-experiences-one-backend rule** and the honest status discipline carried from the Ultimate Roadmap.

---

## 2. Findings

Severity scale: **Blocker** (must resolve before implementation planning) · **High** (contract gap that will produce rework or a security hole) · **Medium** (ambiguity that will produce divergent interpretations) · **Low** (hygiene).

### R1 — Blocker: two contradictory orderings of the same work

**Where:** status brief ("The immediate objective is to complete the minimum RPA primitives and deliver an actual working business workflow", followed by the Mom-n-Pop sequence 1–5 *then* W1) versus §6 of the recommended roadmap (R0 freeze → R1–R2 → R3 → *then* first vertical slice).
**Failure scenario:** one agent builds R1/R2/R3 platform phases first and delays W1 by months (the framework-first diversion this project has already forbidden once); another builds W1 against primitives the R0 contract never froze, then retrofits. Both can cite this document.
**Why existing controls miss it:** §6 says "Roadmap A continues in parallel" but never reconciles the Mom-n-Pop increments with the R-series; the increments and the R-phases describe the same machinery (intake = triggers; scheduling = R3; receipts = evidence) under two names and two orders.
**Correction:** one dependency-ordered plan in which R0 is a *thin* freeze (vocabulary, invariants, lifecycle states) and the Mom-n-Pop increments become R3 increments *pulled by W1*, each built at the scale W1 demands. Platform-first and slice-first are merged, not chosen between.

### R2 — Blocker: control flow is load-bearing but undefined — the one place a second workflow engine can grow

**Where:** the corrected abstraction list includes "Control flow: sequence, branch, repeat, wait, and transition between steps" — with no semantics, no bounds, and no statement of what implementation is forbidden.
**Failure scenario:** an executing agent, needing W1's branching and waits, implements a general expression evaluator or an interpreter over Automation definitions. The platform now contains a second orchestration engine outside the Miniapp execution categories and ADR-001 — the exact violation the guardrails name, grown quietly from a legitimate need.
**Why existing controls miss it:** ADR-001 governs *Class D substrate adoption*; it does not govern what the composition layer itself may do. Nothing in the vision bounds the composition runtime.
**Correction:** freeze minimal control-flow semantics as a contract: sequence, typed conditional branch over validated step outputs (small predicate subset, no side effects), bounded iteration with declared maximums, durable wait-for-event with timeout, terminal outcomes. Everything beyond (parallel fan-out, loops without bounds, richer expressions) requires an ADR under the same five-question anti-bloat test. The composition runtime is a thin state machine over receipts and dispatched classified steps — never an interpreter of business logic.

### R3 — Blocker: W1 is a multi-day durable process, and the durable-wait contract it needs is never named

**Where:** W1 stages 4→8: owner approval wait, estimate send, customer acceptance capture, job scheduling, invoice/payment, review request — spanning days, with human decisions and external events across restarts.
**Failure scenario:** the first slice gets built with in-process timers, in-memory waits, or fire-and-forget jobs; the first deployment loses the customer-acceptance wait or double-sends a payment request. The guidelines' "it's just async" trap, reproduced in the flagship demo.
**Why existing controls miss it:** the vision correctly says "a workflow is not automatically Class D" and defers to ADR-001 — but deferral is not a plan. No increment owns the durable-wait mechanism W1 cannot exist without (approval wait at minimum).
**Correction:** make the durable-wait contract an explicit W1 gate: waits are persisted process state + event/signal resumption + bounded timeout, exercised through the existing approval machinery; the substrate question (state-row + queue vs DBOS) is decided by ADR-001 *at the W1 gate*, before stage 4 is coded. Note this is the *bounded* case — the decision should be cheap, but it must be made in writing.

### R4 — High: no capability-version binding contract

**Where:** the vision defines versioned Automations and versioned Miniapp capabilities but never states their relationship at runtime.
**Failure scenario:** a Miniapp ships capability v2 with changed output shape; every active Automation referencing it either breaks at runtime or silently changes behavior; an in-flight run resumes against a different contract than it started with. Duplicate sends and corrupted runs follow.
**Why existing controls miss it:** manifests declare per-capability contracts, but nothing binds an Automation version to a pinned set, and nothing forbids in-place mutation of a referenced contract.
**Correction:** Automation versions pin capability contract references *and version constraints*; activation validates availability and attestations; a run binds to the pinned set for its whole life; incompatible capability change ⇒ new Automation version via normal promotion, never silent mutation. Enforced by a CI drift test.

### R5 — High: the agent interaction layer has no contract — agent-native is currently a slogan

**Where:** "Agents … interpret requests, gather information, propose actions, and explain outcomes. They do not own authorization, credentials, or execution state." No defined step contract, no output validation, no fabrication guard.
**Failure scenario:** an agent translating a Skill into an Automation references a capability that does not exist, invents an input field, or executes an unregistered operation via the API surface. Skill translation becomes the platform's largest fabrication surface precisely where it is least supervised (§14 failure modes 10 and 11).
**Why existing controls miss it:** approval machinery governs consequential *effects*; it does not govern *authoring*. Capability discovery exists; capability *validation of agent proposals* is not scheduled anywhere.
**Correction:** a constrained agent-step contract: declared input/output schemas, validated structured outputs, explicit proposing principal, zero ambient credential or permission access; agent proposals validated against the registered capability inventory and connector attestations before owner review; unsupported requirements surfaced explicitly, never silently filled; deterministic domain logic and probabilistic agent reasoning structurally separated (agent outputs are evidence-typed as agent-produced).

### R6 — High: webhooks are declared first-class but unscheduled and under-contracted

**Where:** the webhook section of the vision (six concepts, "not synonymous with an internal event or an Automation") versus the §6 roadmap, which schedules triggers only inside R3 *after* R1–R2, and the Mom-n-Pop sequence, where "event intake" (increment 3) appears with none of the webhook contracts attached.
**Failure scenario:** event intake gets built as a polling-only increment; webhook ingress is later bolted on with payload-claimed tenant binding, no delivery identity, and ack-before-persist — the spoofing/replay/duplicate-processing failure set (§14 failure mode 8) becomes the platform's front door.
**Why existing controls miss it:** the effect ledger protects *outbound* effects; inbound acceptance (dedup, replay, ordering, tenant binding) is a distinct contract nothing owns yet.
**Correction:** webhook ingress = the generalization of Mom-n-Pop increment 3, scheduled as the first R3 increment pulled by W1 (stages 1, 6, 8 all need it), with the full contract: tenant binding derived from endpoint registration (never payload claims), delivery identity + replay window, persist-then-ack with async dispatch, normalized event schemas, declared out-of-order policy, secret rotation, quarantined deliveries as evidence. Outbound delivery stays a capability/effect — no new generic delivery machinery (as the prompt itself suggests).

### R7 — High: W1 says "a real tenant" while hierarchical tenancy has no scheduled relation to W1

**Where:** §6 step 5 ("demonstrate the entire workflow using a real tenant") versus R1 (hierarchical multi-tenancy, positioned after W1-enabling work and never mapped to the Mom-n-Pop sequence, which has no tenancy increment at all) — while the Ultimate Roadmap's R0 gate forbids single-tenant shortcuts.
**Failure scenario:** either W1 is built single-tenant (violating the gate) or R1's full org/workspace/delegation lifecycle blocks W1 behind months of administration machinery W1 does not need.
**Why existing controls miss it:** tenant *mechanics* (RLS, tenant context, broker leases) are Done (F0 increment 2); tenant *hierarchy as product* is R1. The boundary between "enough tenancy for W1" and "R1 complete" is undefined.
**Correction:** define W1's minimum tenancy explicitly: existing RLS tenant context, one organization, one tenant scope, tenant-scoped broker leases end-to-end (this validates increment 2 in anger), org/workspace/delegation *contracts designed* at R0 but *implemented incrementally after W1*. Full hierarchy remains R1; nothing in W1 hard-codes its absence.

### R8 — Medium: the receipts/evidence increment is misclassifiable

**Where:** Mom-n-Pop increment 1 ("Receipts and evidence: record decisions, execution outcomes, and external effects") sits in a list otherwise composed of business capabilities.
**Failure scenario:** an agent implements "receipts" as a business Miniapp, creating a parallel evidence system beside the framework's — the duplicate-backend defect the two-experience rule forbids, recreated inside one backend.
**Why existing controls miss it:** the brief says approvals/effects/intent-hash already exist and are reused, but never states that increment 1 is framework wiring + validation, not a business capability.
**Correction:** declare increment 1 as F0/framework scope: append-only receipt store already implied by existing machinery; the increment delivers the write path, retention policy, and query API — no business semantics.

### R9 — Medium: vocabulary drift — three abstraction sets and three names for the foundation

**Where:** the first answer (five layers + P1–P14 primitives), the corrected answer (seven abstractions), and the final framing (four central concepts) all remain in the document; "Process" vs "Automation", "Evidence" vs "Receipts", "Platform runtime" vs "RPA framework layer" vs "F0" vs "shared platform foundation".
**Failure scenario:** downstream agents pick different sets and build overlapping concepts — the exact "independently inventing incompatible interpretations" outcome the governing prompt exists to prevent. The superseded P1–P14 list is especially dangerous because it is concrete and appears authoritative.
**Correction:** one glossary in one place; the P1–P14 list explicitly marked superseded (its items are business capabilities and F0 services); one name per concept (Automation, not Process; Evidence, with receipts as the records Evidence comprises; F0 as the only name for shared foundation).

### R10 — Medium: W1 stage 4 needs an approvals surface that Roadmap B does not yet have

**Where:** W1 stage 4 ("Studio presents the exact proposed estimate. Approval is bound to its material fingerprint") versus Roadmap B all-PLANNED and R4 deferred to after W1.
**Failure scenario:** the flagship demonstration has no owner-facing approval UI, so stage 4 is faked through a database mutation or an API call — undermining the "human authorization" guardrail in the very demo meant to prove it.
**Why existing controls miss it:** approval *machinery* exists (P0/P2-accepted handlers per the capability matrix); the plan never connects W1's gates to it.
**Correction:** W1's approval gates render through the existing B0 approval handlers (conversation/question/approval surfaces are live); the Ops-Studio approval inbox arrives with R4. State the dependency explicitly.

### R11 — Medium: honest capability differences have no declared-unavailable handling

**Where:** "The connector contract must describe what a provider supports and what it does not… never grants execution permission" — but nothing defines what an Automation *does* when a bound connector lacks a capability.
**Failure scenario:** runtime failure mid-process with no policy, or silent lowest-common-denominator degradation (§14 failure mode 14).
**Correction:** attestation checked at activation (admission-time failure for hard requirements); runtime behavior per declared policy: fail the step, take a declared fallback branch, or escalate to a human decision — never silent substitution.

### R12 — Low: quota enforcement (in scope) has no usage-event contract, while metering is deferred to R7

**Where:** non-goals defer billing; tenancy lifecycle includes quotas.
**Failure scenario:** R7 retrofits usage counting onto tables that never recorded it.
**Correction:** an append-only usage-event contract at F0 (schema-compatible with future billing, no billing logic) — the same pattern already accepted for metering in the Ultimate Roadmap.

---

## 3. Required amendments (summary)

| # | Amendment | Resolves | Class |
|---|---|---|---|
| A1 | One dependency-ordered plan; R0 = thin freeze; Mom-n-Pop increments become R3 increments pulled by W1 | R1 | Blocker |
| A2 | Minimal control-flow semantics contract (sequence, branch, bounded repeat, durable wait, terminals); composition runtime = state machine over receipts; beyond-set requires ADR | R2 | Blocker |
| A3 | Durable-wait contract as explicit W1 gate; substrate decided by ADR-001 at the gate | R3 | Blocker |
| A4 | Automation-version → capability-version pinning; admission validation; run-bound sets; CI drift test | R4 | High |
| A5 | Constrained agent-step contract; validation of agent proposals; no fabrication of capabilities; determinism separation | R5 | High |
| A6 | Webhook ingress = generalized increment 3, first R3 increment pulled by W1, full contract per R6 | R6 | High |
| A7 | W1 minimum-tenancy definition (RLS + one org/tenant + broker path); hierarchy contracts designed, implemented post-W1 | R7 | High |
| A8 | Increment 1 declared F0/framework scope | R8 | Medium |
| A9 | Single glossary; P1–P14 marked superseded | R9 | Medium |
| A10 | W1 gates bound to existing B0 approval handlers | R10 | Medium |
| A11 | Declared-unavailable capability policy (activation check + runtime policy) | R11 | Medium |
| A12 | Append-only usage-event contract at F0 | R12 | Low |

All twelve are incorporated into the companion plan (`rpa-platform-grand-vision.md`). None alters the vision; each makes an existing vision statement executable.

---

## 4. Disposition note

This review was produced by an agent that did not author the reviewed material (the vision exchange and plan in `roadmap7.md` were authored in a prior owner/agent exchange), satisfying the independence requirement of the governing prompt. The companion plan is accompanied by a self-adversarial pass against the same §14 failure-mode list, reported in its final report. Findings here are structural or contract-level; no stylistic or duplicate-content findings are included, and none were invented to pad the list.


# Miniapp Studio — RPA Platform Grand Vision & Architecture Plan

| Field | Value |
|---|---|
| Document type | Design-stage architecture specification (Artifact A per Prompt 37: `plans/rpa-platform-grand-vision.md`) |
| Companion document | `rpa-platform-grand-vision-review.md` — adversarial review of the prior vision & plan (findings R1–R12, amendments A1–A12) |
| Status | PROPOSAL for Project Owner ratification. Design only: no code, migrations, schemas, endpoint paths, or infrastructure actions. Existing roadmap numbering (F0, Roadmap A B0–B5, Roadmap B R0–R8) is preserved and not modified |
| Date | 2026-11-27 |
| Vision constraint | Per owner instruction, the vision is preserved substantially: process-agnostic, API-first, capability-driven, agent-native, composable, provider-independent, multi-tenant, minimal at the core, webhooks first-class. This plan operationalizes it; it does not redirect it |

---

## 1. Executive vision

A business owner describes a workflow in plain language. An AI agent interprets it, discovers the tenant's available capabilities, and proposes a declarative **Automation**. Studio validates that every referenced capability is registered, attested, and authorized; the owner reviews and activates. At runtime the platform governs execution — authorization before dispatch, brokered credentials, idempotent effects, human gates bound to material hashes, durable waits across restarts, and append-only evidence for every decision and external outcome. The owner watches and manages the whole operation through Studio; agents build and revise under the same governance, never above it.

The platform provides a **small, frozen set of generic abstractions** — Automation, Step, Trigger (with first-class webhook ingress), bounded Control flow, Gate, Effect, Evidence — and lets everything domain-specific live in Miniapps and connectors, of which there may be thousands. Connectors make providers replaceable without rewriting processes. Tenancy makes departments and clients isolated and delegable. Two Studio experiences (Engineering, Operations) share one backend.

The governing principle, preserved verbatim from the vision:

> **Miniapp Studio should understand how to execute and govern a business process, not understand every business process that exists.**

The first proof is **W1 Lead-to-Payment** — one real business workflow, end-to-end, for one real tenant against controlled provider sandboxes — built from the same abstractions that any unrelated process would use.

## 2. Goals, non-goals, and governing principles

**Goals**

1. Deliver W1 (Lead-to-Payment) as the first vertical slice using only the frozen abstraction set — no runtime changes to add the process itself.
2. Freeze the minimal abstraction set and the four load-bearing contracts (control-flow semantics, capability-version binding, webhook ingress, agent-step boundary) as contracts-with-tests, before implementation planning.
3. Reuse what exists: approvals, effects, intent-hash, `CapabilityRef`, manifest, trust modes, credential broker (F0 increment 2, Neon-verified), RLS tenant mechanics.
4. Keep Roadmap A (B0–B5) progressing in parallel; Roadmap B (R0–R8) attaches at gates without renumbering or blocking it.
5. Maintain honest status labels (implemented / accepted / designed / planned / deferred) on every capability the plan touches.

**Non-goals (unless separately approved)**

- Visual workflow canvas, workflow programming language, plugin marketplace, agent registry, proprietary agent runtime.
- Giant CRM / ERP / accounting engine; predefined automation catalog.
- Classic UI-RPA and screen scraping (API-centric first; separate capability with its own security ADR if ever approved).
- Billing productization (append-only usage events only; invoicing deferred to R7).
- A new orchestration engine, or any composition machinery beyond the bounded semantics in §5/§8.
- Parallel fan-out semantics (deferred post-W1); generic outbound webhook delivery machinery (deferred; outbound notification is an ordinary capability/effect).

**Governing principles**

| # | Principle | Enforcement |
|---|---|---|
| P1 | Minimal core: business operations are never platform primitives | Anti-bloat gate (§6) on every proposed abstraction |
| P2 | Composition over abstraction | Five-question test before any new core concept |
| P3 | Lowest-sufficient execution class per invocation boundary | Existing A/B/C/D discipline; ADR-001 authority for Class D |
| P4 | The framework owns the hard parts once | Effects, idempotency, approvals, reconciliation, tenancy context, evidence — never per-process |
| P5 | Agent proposes → platform validates → owner authorizes | Agent-step contract (§7); activation is human |
| P6 | Evidence over claims | Green requires inspectable artifacts; agent claims never turn a stage green |
| P7 | Provider honesty | Attestations declare supported/unsupported; no silent degradation (§8) |
| P8 | Reuse first | Every design element names the existing contract it extends, or justifies the new one |
| P9 | One backend, two experiences | Shared domain services; separate information architectures (§12) |

## 3. Glossary and conceptual model

One glossary, one name per concept (review amendment A9). The P1–P14 primitive list from the prior exchange is **superseded**: its items are business capabilities (implemented by Miniapps/connectors) or F0 services — never platform primitives.

| Concept | Is | Is not | Owner |
|---|---|---|---|
| **Skill** | Declarative business knowledge: instructions, rules, checklists, sequencing guidance, required capability references | Executable code; an Automation; a Miniapp; a grant of authority or credentials | Authored content, consumed by agents |
| **Automation** | Tenant-owned, versioned, declarative business-process definition: objective, triggers, steps, gates, policies, environment binding | Hardcoded domain program; a workflow engine; a Miniapp | Platform (definition), tenant (ownership) |
| **Automation Version** | Immutable snapshot of an Automation; executions bind to it for life | Mutable definition; a running instance | Platform |
| **Miniapp** | Versioned executable Go application in the Miniapp framework, exposing capabilities via manifest | The whole process; a platform primitive | Platform (framework), developer/agent (code) |
| **Capability** | Typed, versioned, invokable operation contract: I/O schemas, idempotency identity, class, trust modes, limits | A business concept the platform understands; a provider endpoint | Platform (contract), Miniapp (implementation) |
| **Connector** | Replaceable provider integration boundary: adapter + attestation (supported / unsupported / restricted) | A capability; ambient credentials; a tenant's connection itself | Platform (interface), provider adapter (implementation) |
| **Trigger** | Binding that starts or resumes an Automation version from an event source | The event itself; the Automation | Platform |
| **Webhook Endpoint / Delivery / Verified Event / Route** | Ingress sub-model: registered endpoint → received request → verified, normalized event → tenant/subscription binding | Synonyms for "an event" or "a webhook" generally | Platform |
| **Step** | One capability invocation within a run, with validated input/output mapping | A business operation; a platform primitive | Platform (execution), Automation version (declaration) |
| **Run** | One execution of one Automation version, with environment binding and evidence trail | An Automation; a job | Platform |
| **Gate** | Generic execution control: policy check + human approval where required, bound to material hash | A business capability ("approval service"); agent self-approval | Platform |
| **Effect** | Framework-handled externally observable side effect: intent → attempts → confirmed/failed/UNKNOWN → quarantine/reconcile | A logged afterthought; a per-process concern | Platform (F0/framework machinery) |
| **Evidence / Receipt** | Append-only execution record: decisions, step outcomes, deliveries, approvals, effects. ("Receipts" are the records Evidence comprises) | Authoritative current state of a remote business object | Platform |
| **F0** | Shared foundation: identity, tenancy context, secrets, credential broker, persistence, evidence storage, environment isolation | A business capability; the RPA framework itself | Platform |
| **Business capability** | Any domain-specific operation (contact upsert, invoice issue, campaign send, …), implemented by Miniapps/connectors; unlimited in number | A platform primitive; something the platform core understands | Tenant's Miniapps and connectors |

Conceptual model (one line per relationship): **Skill → guides agent authoring → Automation (versioned definition) → composes Steps → each Step invokes a Capability → Capability is implemented by a Miniapp and, where external, bound to a Connector → runtime executes, governs, and records Evidence.**

## 4. Skill → Automation → Miniapp → Capability → Connector relationships

```
 Skill (knowledge)          ── authored, versioned, referenced by agents
        │  agent interprets; NOT executable; grants nothing
        ▼
 Automation vN  (declarative definition, tenant-owned, environment-bound)
   ├─ triggers: manual | API | schedule | internal event | inbound webhook
   ├─ steps: capability refs + validated I/O mapping + control flow
   ├─ gates: policy + human approval (material-hash bound)
   └─ pins: capability contract versions, connector interfaces, policies
        │  activation validated against registry + attestations
        ▼
 Capability contract (typed, versioned)  ◄── implemented by ──►  Miniapp
        │                                                          (Go, manifest,
        ▼                                                           classes A–D)
 Connector (per provider adapter + attestation)  ◄── tenant connection + brokered lease
        │
        ▼
 External provider (replaceable without rewriting the Automation)
```

Cardinalities and the four non-identity rules (preserved from the vision):

- A Skill contributes to 0..n Automation versions — some become domain logic, some connector guidance, some remain agent instructions or runbooks.
- An Automation version references 1..n capabilities; a Miniapp exposes 1..n capabilities. Neither relationship is one-to-one.
- **One Automation ≠ one Miniapp ≠ one Fly app ≠ one Sprite.** An Automation may compose several of each; ownership, identity, and versioning live on the Automation.
- A capability may have 0..n connector adapters (provider-neutral contract, provider-specific implementation); the tenant's connection and credentials are always separate from the contract.

Skill-content mapping (preserved, corrected):

| Skill content | Studio responsibility |
|---|---|
| Process instructions, sequencing | Agent-facing guidance only |
| Business rules, decision logic | Pure Go domain logic inside a Miniapp (deterministic) |
| Actions | Typed capability references |
| Provider-specific procedures | Connector adapter behavior + attestation |
| Safety checklists | Admission tests and policy declarations |
| Human-approval instructions | Gate declarations in the Automation |
| Retry/recovery guidance | Execution contracts (class, idempotency, effect policy) |
| Operational guidance | Runbooks linked from Evidence |

## 5. The minimal platform abstraction set (frozen)

Seven abstractions plus the webhook ingress sub-model. This set is **frozen at R0 as contracts-with-tests**; schemas remain evolvable under drift tests (review amendment A1 — thin freeze, not big-bang). "Owns / does not own" is the enforceable boundary:

| Abstraction | Owns | Explicitly does not own |
|---|---|---|
| **A. Automation (Process)** | Objective, version, lifecycle (draft → validated → active → suspended → deprecated → retired), trigger declarations, step composition, gate declarations, policies, environment binding, tenant ownership | Business semantics; execution state; provider details; scheduling machinery |
| **B. Step** | Typed capability invocation: what is invoked, validated inputs, output mapping, class assignment, failure policy | The capability's implementation; business meaning |
| **C. Trigger** | Event-source binding → target Automation version, tenant context from trusted source, validated input, authorization | Event persistence for external sources (ingress owns it); business logic |
| **D. Webhook ingress** | Endpoint registration + secrets + rotation; delivery identity + replay window; signature verification; normalized events; tenant binding **from registration, never payload claims**; persist-then-ack; quarantine | Provider API semantics (connector's job); outbound delivery (a capability/effect, deferred); long-running execution (dispatch is async) |
| **E. Control flow** | Bounded semantics only: sequence; typed conditional branch (small predicate subset over validated step outputs, no side effects); bounded iteration with declared maximums; durable wait-for-event with timeout; terminal outcomes (completed / failed / suspended / expired) | An expression language beyond the predicate subset; unbounded loops; parallel fan-out (deferred); compensation beyond declared compensating capabilities; any interpretation of business logic |
| **F. Gate** | Policy checks (tenant, capability scope, trust mode); human approval with material-hash binding, expiry, decision receipts; separation of proposing agent from approving authority | Business judgment; credential access |
| **G. Effect** | Intent registration, idempotency identity, attempt lifecycle, bounded retries, UNKNOWN quarantine, reconcile-by-read, provider-side idempotency | Retrying blindly after ambiguous outcomes; business semantics of the effect |
| **H. Evidence** | Append-only receipts for runs, steps, decisions, deliveries, invocations, effects, approvals; retention; query API | Mutating state; pretending to be provider-authoritative state |

F0 services (infrastructure, not abstractions — per the prompt's §4.I): identity and principals, tenancy context (RLS, fail-closed), secrets management, credential brokerage (short-lived purpose-bound leases), persistence (Neon), event storage, scheduling substrate, observability, environment isolation. Existing implementations are reused and named in §11.

**Addition gate (anti-bloat, binding):** every proposed new platform abstraction must pass all five questions — (1) truly generic across unrelated processes? (2) representable via the seven abstractions? (3) does it import domain semantics into the core? (4) does it solve a real cross-process reliability/security problem? (5) is its absence demonstrably a concrete architectural limitation? Fail any ⇒ compose, don't add.

## 6. Platform-versus-business-capability boundary

Deliberately diverse examples (the list is open-ended; there may be thousands):

| Business capability — belongs to | Platform treatment |
|---|---|
| Contact upsert, customer qualification | Business Miniapp (Party domain) |
| Estimate creation, deterministic pricing | Business Miniapp (Commercial domain) |
| Invoice issue, payment collect | Business Miniapp + provider adapter (money ⇒ Gate required) |
| Email/SMS send, review request, campaign send | Business Miniapp + messaging connector |
| Job sync, appointment booking | Business Miniapp + connector (calendar capability is an explicit adapter contract, not cron) |
| Email/webform/voice intake | Connector + **platform** ingress (the *mechanism* is platform: webhook/poll trigger; the *business event* is a capability) |
| Financial analysis, payroll, inventory | Business Miniapps — platform core never learns these words |

Superseded-list disposition (amendments A8/A9): P1 contact upsert, P2 estimate, P3 invoice, P4 payment, P6 message, P8 review, P9 job sync, P14-adjacent campaign ⇒ **business capabilities**. P7 intake poll ⇒ generalized into **platform ingress** (webhook/poll triggers). P11 receipt append ⇒ **F0 Evidence**. P13 credential lease ⇒ **F0 credential broker** (Done, increment 2). Cron/overlap protection ⇒ **F0 scheduling substrate + keyed serialization**.

**The test of process agnosticism is not the number of workflows shipped — it is whether a new workflow can be introduced by defining capabilities and composing existing contracts, with zero runtime changes.**

## 7. Process authoring lifecycle (Skill → executable process)

1. Owner expresses the desired business outcome in natural language.
2. Agent discovers applicable Skills, the tenant's registered capabilities, and connector attestations.
3. Agent interprets Skills: maps instructions to capability references, decisions, inputs, triggers, gates.
4. Agent proposes an **Automation definition** (declarative; every capability referenced must already exist).
5. Platform validates: capability availability + version constraints, connector attestations, trigger bindings, gate policies, control-flow bounds, schema conformance of step I/O mappings.
6. Agent produces a human-readable process explanation **and an explicit list of unsupported requirements** — never silent substitution, never fabricated capabilities (amendment A5).
7. Owner reviews and authorizes **activation** (activation is human; revisions re-enter at step 5).
8. Runtime executes the active version; Studio presents runs, approvals, failures, evidence.
9. Revision = a new Automation version through the same validation and promotion path; an active version is never silently modified.

**Constrained agent-step contract (frozen with R0):** an agent acting inside a run is itself a declared step type — declared input/output schemas, outputs validated before use, explicit proposing principal (never the approving authority), zero ambient credential or network access, all provider access via brokered capability invocation only. Agent outputs are evidence-typed as agent-produced (probabilistic), structurally separated from deterministic domain logic. Deterministic business rules live in pure Go domain logic in Miniapps; agent reasoning lives only in agent steps and authoring.

Skill-content mapping follows §4; the authoring lifecycle never grants an agent the ability to create capability *declarations* — only to reference registered ones.

## 8. Runtime and execution semantics

**Run state machine:** `created → running ⇄ suspended (waiting event/approval) → completed | failed | cancelled | expired`. Step states: `pending → running → succeeded | failed | skipped`. Effects reuse the existing ledger lifecycle (PENDING → ATTEMPTING → CONFIRMED/FAILED/UNKNOWN + quarantine + reconcile-by-read).

**Classification stays per invocation boundary** (existing discipline): each step invokes a capability whose class (A/B/C/D) was decided at Miniapp design time. The Automation composition layer assigns dispatch per step; mixed-class composition is normal. A multi-stage Automation is **not automatically Class D** — ADR-001 remains the authority whenever durable-wait semantics dominate.

**Composition runtime scope (frozen; the anti-second-engine boundary, amendment A2):** the runtime owns exactly — run records, step dispatch through the existing substrates (handler+tx / River / ADR-001-approved durable path), wait resumption via signals, gate checks against the approval machinery, receipt appends, declared-timeout enforcement via scheduled jobs. It never owns: an expression interpreter beyond the predicate subset, hidden retries beyond step contracts, business semantics, or its own timer/progress machinery for waits.

**Durable wait contract (frozen; amendment A3 — the W1 gate):** a wait is persisted process state + a correlation key + event/signal resumption + a declared timeout with expiry policy. Waits survive restarts and deploys because state lives in Neon, not in memory. The substrate for the wait mechanism (state-row + River + signals vs DBOS) is decided by ADR-001 at the W1 gate, before stage 4 of Lead-to-Payment is coded. A process never blindly repeats an external effect whose outcome is unknown.

**Capability-version binding (frozen; amendment A4):** an Automation version pins capability contract references **with version constraints** and connector interface versions. Activation validates availability + attestations against the registry. A run binds to the pinned set for its entire life, including across suspension/resume. Incompatible capability change ⇒ new Automation version via promotion — never silent mutation. A CI drift test proves manifest ↔ contract ↔ Automation pinning consistency.

**Declared-unavailable handling (frozen; amendment A11):** connector attestations are checked at activation (hard requirements missing ⇒ activation fails, honestly). At runtime, a step whose bound connector lacks a required capability follows its declared policy: fail the step, take a declared fallback branch, or escalate to a human decision — never silent substitution, never lowest-common-denominator fakery.

**Webhook delivery acknowledgement:** providers are acked after persist (durable intake), never after process completion; dispatch is asynchronous through the trigger machinery. See §9.

## 9. Webhook, event, polling, and scheduler relationship

Five distinct event sources — different ingestion, identity, dedup, ordering, and ack semantics; never one indistinguishable mechanism:

| Source | Ingestion | Delivery identity | Duplicate protection | Ordering | Ack semantics |
|---|---|---|---|---|---|
| **Inbound webhook** | Platform endpoint, tenant-owned registration | Provider message/event id + endpoint + payload hash | Replay window + idempotent dispatch on delivery identity | Declared per Automation: queue / latest-wins / reject | **Persist → ack → async dispatch** (provider never waits for the process) |
| **Poll** | Connector poller on a schedule | Source cursor + item id | Watermark advanced transactionally with intake | Source order via cursor | Poll success after persist |
| **Schedule (cron)** | F0 scheduling substrate | Schedule id + slot | Keyed serialization on the schedule entity (overlap prevention) | N/A | Job receipt |
| **Internal event** | Platform outbox / domain event | Event type + aggregate id | Outbox dedupe | Per-aggregate | Producer receipt |
| **Manual / API** | Invocation API | Invocation id | Idempotency key | N/A | Sync or run handle |

Webhook ingress lifecycle (frozen contract; amendment A6 — the generalization of Mom-n-Pop increment 3, and the first R3 increment pulled by W1):

1. **Endpoint registration** — tenant-owned (or controlled platform integration), scoped secret, rotation supported; environment-scoped (dev/preview/staging/prod never share endpoints).
2. **Delivery** — every received request recorded as Evidence, valid or not (including rejected ones, with reason).
3. **Verification** — provider-appropriate signature scheme + timestamp tolerance via the connector; verification secrets live in F0 secrets, rotated, never in payloads or logs.
4. **Normalized event** — schema-validated representation; oversize/malformed ⇒ quarantine with reason.
5. **Routing** — tenant, subscription, and target Automation version derived **from the registration and route table, never from payload claims** (anti-spoofing, anti-cross-tenant).
6. **Trigger invocation** — persist event, start/resume the pinned version via correlation key, dispatch async.

Security invariants: replay protection (window + seen-delivery set), duplicate delivery collapses on delivery identity, out-of-order handled by declared policy, size and schema limits at the door, quarantine observable, no SSRF surface (ingress never fetches remote resources), cross-tenant routing impossible by construction (registration-sourced binding). Relation to the ledger: deliveries, events, and trigger invocations are Evidence; the *process* then creates its own effects under the ledger. Outbound webhook delivery is deferred — modeled as an ordinary capability/effect when first needed.

## 10. API-first architecture (conceptual contracts only)

One domain model, every consumer the same contracts — no separate business logic per surface. Endpoint paths and JSON schemas are explicitly deferred to later ADRs/implementation specs.

| Conceptual API | Consumer | Responsibility | Authorization |
|---|---|---|---|
| Automation management | Owner/agent via Studio, API | CRUD draft definitions, versions, lifecycle transitions | Tenant-scoped; agent = proposing principal |
| Validation & activation | Owner, agent (propose only) | Run §7 steps 5–7: contract/attestation/policy validation; activation decision | Activation = human authority principal |
| Execution / invocation | Agents, webhooks, schedules, API | Start/resume a pinned version under a trigger | Trigger-sourced tenant binding; trust mode enforced |
| Run status & evidence | Both Studios, agents, API | Run/step state, receipts, deliveries, effects | Tenant-scoped; evidence is append-only |
| Approval / decision | Owner (authority principal) | Gate decisions bound to material hash, expiry | Out-of-band from the proposing agent |
| Capability discovery & attestation | Agents, validation | Registered capabilities + connector supported/unsupported inventory | Tenant-scoped; discovery never grants execution permission |
| Connector & connection management | Owner/ops, agent (propose) | Adapter registration, tenant connections, credential enrollment via broker | Credentials only through F0 broker |
| Webhook registration & ingress | External providers; owner config | §9 lifecycle end-to-end | Registration-sourced tenant binding |
| Trigger dispatch contracts | Platform-internal | Event → trigger → run, all sources | Internal, receipted |

## 11. Identity, tenancy, security, and governance

Boundary model: **Organization → Tenant scope → Workspace**. A department (company pattern) or client business (agency pattern) is a tenant scope; corporate oversight in the company pattern is explicit delegated read, not ownership; agency operators act under explicit, scoped, revocable per-client delegation — never ambient authority.

| Invariant | Status | Source |
|---|---|---|
| RLS + fail-closed tenant context, tenant-scoped idempotency (`tenant:entity:kind:op`) | **Accepted** (mechanics) | Existing tenant suites |
| Credential broker: encrypted storage, short-lived purpose-bound leases | **Done** (F0 inc 2, Neon-verified) | F0 |
| Unified secrets management | **Done** (F0 inc 1) | F0 |
| Tenant invitations & capability refresh | **In progress** (F0 inc 3) | F0 |
| Approvals, effects, intent-hash machinery | **Exists — reused, not rebuilt** | Framework |
| Delegation contracts (cross-org, scoped, revocable, audited) | **Designed** (contracts at R0; implementation post-W1) | This plan |
| Agent principal ≠ authority principal; non-self-approval; material-hash binding | **Accepted** | Constitution |
| Environment separation dev/preview/staging/prod; immutable versioned artifacts; digest-bound promotion | **Policy accepted; implementation = Roadmap A B3/B5** | Ultimate Roadmap |
| Webhook security set (registration-sourced binding, replay, size, rotation, quarantine) | **Designed** (§9) | This plan |
| Audit & evidence retention | **Designed** (Evidence abstraction; append-only) | This plan |
| Trust mode (dry-run/confirm/autonomous) × execution class × placement — independent dimensions | **Accepted** | Existing decisions |

**W1 minimum tenancy (frozen definition; amendment A7):** existing RLS tenant context end-to-end; one organization; one tenant scope; tenant-scoped broker leases exercised on every external call. This validates the broker in anger without implementing the full hierarchy. Org/workspace/delegation contracts are designed at R0 and implemented incrementally after W1 (R1). Nothing in W1 may hard-code single-tenancy — the R0 no-shortcut gate stands.

No parallel authorization model is introduced; trust mode, execution class, and compute placement (Machines for trusted runtimes, Sprites for untrusted/agent-generated execution) remain the existing independent dimensions.

## 12. Studio A/B product relationship

**Roadmap A — Engineering Studio** (developers and coding agents): author Miniapps, define and validate contracts, manage versions, build and deploy previews (B3), diagnose failures (B4), promote and release through staging/production (B5). The Go code implementing W1's capabilities rides this track like any other code.

**Roadmap B — Automation Operations Studio** (owners, operators, QA, managers): discover Skills and capabilities, create/configure Automations, configure triggers (including webhooks), manage tenant connections, review authorization requirements, monitor runs, inspect failed steps and UNKNOWN effects, reconcile exceptions, inspect evidence and operational health (R0–R8).

Both share identity, authorization, connectors, runtime, records, and audit services — one backend, two information architectures (D-5 deployment topology remains open and must not gate F0 contract design). **W1 dependency (amendment A10):** W1's approval gates (stage 4) render through the existing B0 approval handlers (live and accepted); the Ops-Studio approval inbox arrives with R4. No screen mockups in this document.

## 13. Lead-to-Payment — architectural proof, not a platform catalog

One real tenant, controlled provider sandboxes, live financial operations gated until the full path is proven. Every stage uses only the frozen abstraction set; none of these operations becomes a platform primitive:

| # | Stage | Abstractions used | Class | Gates / effects / evidence |
|---|---|---|---|---|
| 1 | Capture customer inquiry (email/webform/voice) | **Webhook/poll trigger** → Automation v1, intake capability | B (persist → ack → async dispatch) | Delivery + normalized event as Evidence; duplicate deliveries collapse on delivery identity |
| 2 | Qualify & identify customer | **Agent step** (proposing principal) + contact-upsert capability | A | Agent output evidence-typed; dedup explicit (new / existing / duplicate / junk branches) |
| 3 | Prepare estimate | Deterministic pricing capability (pure Go domain logic) | A | Step receipt with inputs/outputs; no agent reasoning inside the kernel |
| 4 | Owner reviews & approves | **Gate** + **durable wait** (material-hash bound approval via existing approval machinery, rendered through B0 handlers) | — (wait) | Decision receipt; approval survives restart (state in Neon) |
| 5 | Send estimate; capture customer acceptance | Messaging capability (Class B **effect**) → **durable wait** on acceptance webhook (correlation key) | B → wait | Send = ledgered effect (UNKNOWN ⇒ quarantine, never resend blindly); acceptance = verified inbound event |
| 6 | Schedule & manage the job | Calendar capability + job-sync capability | B/C | Explicit calendar adapter contract (not cron); job records receipted |
| 7 | Invoice & collect payment | Invoice + payment capabilities, brokered credentials | B | **Money ⇒ Gate required**; ledgered effects; reconcile-by-read on ambiguity |
| 8 | Request review; report results | Scheduled trigger + review-request capability; final Evidence roll-up | B | Follow-up under declared policy; operational report from receipts |

Notes: stages 4, 5, 7 are the durable-wait/money paths that trigger the ADR-001 substrate decision at the W1 gate (before stage 4 is coded — amendment A3). The process definition references trigger *types* and capability *contracts*; no provider-specific webhook parsing ever enters the Automation definition.

## 14. Unrelated-process generality (the agnosticism test)

The same frozen set, different capabilities and connectors, zero runtime changes:

- **Employee onboarding** — trigger: HR event/manual; steps: identity creation (Party), account provisioning (IT connector), equipment ordering (vendor connector), manager approval (Gate), orientation scheduling (calendar); evidence throughout. Nothing new in the core.
- **Order fulfillment** — trigger: commerce webhook; steps: order validation, inventory check (capability with declared availability), fulfillment dispatch, customer notification (messaging), exception branch on stock-out (Gate escalation). The declared-unavailable policy (§8) is exercised honestly.
- **Marketing campaign execution** — trigger: schedule; steps: audience selection (Party domain), content generation (agent step, human-reviewed via Gate), send through messaging connector with ledgered effects, response ingestion via webhook correlation.

If a new process needs a capability no Miniapp provides, the answer is a new Miniapp — never a new platform primitive. That is the whole point.

## 15. Dependency-ordered roadmap (numbering preserved: F0 / B0–B5 / R0–R8)

| Stage | Scope | Gate: needs |
|---|---|---|
| **S0 — Complete F0** | Finish increment 3 (invitations, capability refresh); Prompt36 remediation, corrective migrations, regression testing, adversarial review, owner-side Neon verification; receipts/evidence write-path + retention + query API (Mom-n-Pop increment 1 as **F0 scope**, amendment A8) | Existing F0 1–2 (Done) |
| **S1 — R0 thin contract freeze** | Freeze as contracts-with-tests: glossary & cardinalities (§3–4), Automation lifecycle states, control-flow semantics (§5.E), capability-version pinning (§8), webhook ingress contract (§9), agent-step contract (§7), declared-unavailable policy, W1 minimum tenancy (§11), API boundaries conceptually (§10). Schemas stay evolvable under drift tests (amendment A1) | S0; owner ratification of this document |
| **S2 — W1 pull path (R3 increments, pulled — not platform-first)** | Webhook/event intake first (generalized increment 3, amendment A6); then scheduling (increment 4, keyed-serialization overlap protection); then capability Miniapps (increment 5: contacts, estimates, messaging, invoices, payments, explicit calendar capability); durable-wait substrate ADR-001 decision at the gate; W1's Miniapps ride Roadmap A (B2 handoff, B3 preview) like any other code | S1 |
| **S3 — W1 demonstration** | Lead-to-Payment end-to-end: real tenant, sandbox providers, money gates live but amounts controlled; full evidence trail shown in Studio | S2 + owner authorization |
| **S4 — R1 full hierarchical tenancy + R2 catalog** | Org/workspace/delegation lifecycle (contracts already designed at R0); Automation catalog, definitions, versioning; W1 minimum-tenancy scaffolding generalizes, not rewrites | S3 |
| **S5 — R4 ops workspace + R5 per-automation promotion + R6 onboarding** | Operations/QA/management surface; per-Automation preview/staging/promotion/rollback (generalizing the Roadmap A B3/B5 release machinery — one pipeline, two consumers); tenant onboarding flows | S4; B5 staging/prod gates |
| **S6 — R7 metering/quotas + R8 enterprise** | Quota enforcement over the usage events recorded since S0 (amendment A12); billing infrastructure stays deferred as a model, not a blocker; scale, resilience, governance | S5 |

Roadmap A (B0–B5) continues in parallel throughout; the R-series attaches at gates and never renumbers or blocks it (cross-roadmap rule preserved). Mom-n-Pop increments 1–5 map onto S0/S2 — they are the pull-path, not a competing plan.

**Frozen before implementation vs evolvable:**

| Decision | Status |
|---|---|
| Automation definition & version lifecycle (states, immutability, promotion) | **Frozen at R0** (semantics); schema evolvable |
| Skill / Automation / Miniapp / Capability relationships | **Frozen at R0** (glossary + cardinalities) |
| Trigger & event-source semantics (five sources) | **Frozen at R0** |
| Webhook ingress contract | **Frozen at R0** |
| Minimal control-flow semantics + predicate subset | **Frozen at R0** (subset scope = OD-2) |
| Execution & recovery contracts (state machines, ledger reuse, wait contract) | **Frozen at R0** |
| Agent-step boundaries | **Frozen at R0** |
| Capability discovery & attestation | **Frozen at R0** (contract); schema evolvable |
| Tenant & environment scoping | **Frozen at R0** (boundary model + W1 minimum); hierarchy implementation scheduled S4 |
| API boundaries | **Frozen conceptually** (§10); paths/schemas via later ADRs |
| Process-version promotion & rollback | **Frozen at R0** (lifecycle); machinery at R5/B5 |

**Status decomposition:** already implemented = secrets (F0-1), broker (F0-2), approvals/effects/intent-hash, RLS tenant mechanics, Miniapp framework contracts (manifest, `CapabilityRef`, trust modes), B0–B1 Studio surfaces. Reused as-is = everything above. Genuinely new = webhook ingress (generalized intake), Automation version pinning, agent-step contract, attestation extension, Evidence query API. Deferred = parallel semantics, outbound delivery machinery, billing, UI-RPA, D-5 topology. Owner decisions = §16.

## 16. Open decisions and deferred topics

| # | Decision | Owner of the decision | Recommendation |
|---|---|---|---|
| OD-1 | Durable-wait substrate for W1 (persisted state + River + signals vs DBOS) | ADR-001 at the W1 gate | Decide on evidence; both paths keep state in Neon; no hand-built timers |
| OD-2 | Predicate subset for branch conditions | Owner at R0 | Minimal: equality/comparison/conjunction over validated step outputs; total default path required |
| OD-3 | Parallel fan-out semantics | Deferred post-W1 | Needs its own five-question pass; not required by W1 |
| OD-4 | Generic outbound webhook delivery machinery | Deferred | Ordinary capability/effect until a real consumer exists |
| OD-5 | Two-UI deployment topology (open D-5 in the Ultimate Roadmap) | Owner | Must not gate F0 contracts; decide by S5 with evidence |
| OD-6 | W1 sandbox provider set (messaging / payment / job management) | Owner | Choose providers with real sandbox + webhook support |
| OD-7 | Delegation scope for W1 (none vs one supervised delegation) | Owner | Recommend none for W1; delegation exercised in S4 with full R1 |
| OD-8 | Usage-event schema compatibility (billing-compatible events at S0) | ADR at S1 | Append-only events now; no billing logic |

## 17. Architecture acceptance criteria

A reader of this document can answer all twenty governing questions consistently:

1. **What exactly is a Skill?** Declarative business knowledge consumed by agents; never executable, never authoritative (§3).
2. **What exactly is an Automation?** Tenant-owned, versioned, declarative process definition with triggers, steps, gates, policies, environment binding (§3, §5.A).
3. **What exactly is a Miniapp?** Versioned executable Go application exposing typed capabilities via manifest (§3).
4. **What exactly is a Capability?** Typed, versioned, invokable operation contract — the only invocation surface (§3).
5. **What does a Connector own?** Provider-specific auth, transport, semantics, rate limits, webhooks-of-provider, errors, and honest supported/unsupported attestations (§3, §6).
6. **Smallest set of platform abstractions?** Automation, Step, Trigger (+ webhook ingress sub-model), bounded Control flow, Gate, Effect, Evidence; F0 is infrastructure, not abstraction (§5).
7. **Where do webhooks enter, and how do they become triggers?** Ingress lifecycle §9: registration → delivery → verification → normalized event → registration-sourced routing → trigger invocation on the pinned version.
8. **How are webhooks different from events, polling, and schedules?** Five sources with distinct ingestion, identity, dedup, ordering, ack semantics (§9 table) — never one mechanism.
9. **Who decides what an agent can execute?** The platform, against registered capabilities and trust modes; activation and approvals are human (§7, §11).
10. **Where do deterministic business rules live?** Pure Go domain logic in Miniapps (§7).
11. **Where does agent reasoning live?** Agent steps with declared schemas and validated outputs, plus authoring — never inside deterministic logic (§7).
12. **How is execution recovered after crashes?** Run/step state and waits are persisted in Neon; resumption re-enters at the pending step under the pinned version (§8).
13. **How are external effects made safe?** Existing ledger: intent, idempotency, attempts, UNKNOWN quarantine, reconcile-by-read — never blind repetition (§8).
14. **How are long-running processes handled without unnecessary engine adoption?** Durable-wait contract (persisted state + signals + timeout); substrate via ADR-001 only (§8).
15. **New business capability without platform code?** Register a Miniapp capability; core unchanged (§6, §17 tests).
16. **New Automation without runtime changes?** Compose existing capabilities; validation + activation only (§7).
17. **New provider without rewriting the Automation?** Connector adapter under the same capability contract, honestly attested (§6, §8).
18. **How are tenants and credentials isolated?** RLS fail-closed context, registration-sourced webhook binding, brokered short-lived leases, scoped principals (§9, §11).
19. **How do the two Studios relate?** One backend, two information architectures; shared services by rule (§12).
20. **Why will this not become a bloated primitive catalog?** The §5 addition gate + §6 boundary are binding; the P1–P14 precedent is marked superseded.

**Architecture acceptance tests (scenarios):**

1. Define a non-CRM process (employee onboarding) from existing abstractions — activation succeeds, zero runtime changes.
2. Register a new capability Miniapp — platform-core diff is registry data only; no new primitive.
3. Swap a provider adapter where capability equivalence holds — Automation version unchanged.
4. Add an inbound webhook trigger — no provider-specific parsing inside the Automation definition.
5. Demonstrate an unsupported provider capability honestly — attestation blocks activation or declared fallback policy runs.
6. Attempt execution of an unregistered/unauthorized capability via an agent — rejected at validation; ambient access impossible (broker-only).
7. Duplicate webhook delivery + crash-after-send — single run; UNKNOWN → quarantine → reconcile-by-read; never a duplicate message or payment.
8. Human approval across suspension and restart — same material hash, decision applies after restart.

---

## Final report (per Prompt 37 §17)

- **Minimum platform abstraction set:** Automation (Process), Step, Trigger — with webhook ingress as a first-class sub-model — bounded Control flow, Gate, Effect, Evidence. F0 services (identity, tenancy, secrets, broker, persistence, scheduling substrate, observability) are infrastructure, not abstractions. Addition gate: the five anti-bloat questions, binding.
- **Precise role of webhooks:** the platform's push-based ingress trigger — six distinct concepts (endpoint, delivery, verification, normalized event, routing, trigger invocation), persist-then-ack with async dispatch, tenant binding from registration never payload, replay/duplicate/oversize protection, quarantine as evidence. Not a synonym for internal events, polling, or cron; outbound delivery deferred as an ordinary capability/effect.
- **Business capabilities vs platform primitives:** capabilities are the unlimited, domain-specific operations the platform core never understands (invoice issue, contact upsert, campaign send); primitives are the seven generic mechanisms that execute and govern any of them. The prior P1–P14 list is superseded by this boundary.
- **The model:** Skill (knowledge) → guides agent authoring → Automation (versioned declarative definition) → composes Steps → each Step invokes a Capability → implemented by a Miniapp, bound per provider by a Connector → runtime governs execution and records Evidence.
- **Most important architectural decisions:** (1) R0 is a *thin* freeze — boundaries and contracts frozen, schemas evolvable under drift tests; (2) the composition runtime is a state machine over receipts and dispatched classified steps — no interpreter, no second engine; (3) durable waits are persisted state + signals + timeout, substrate decided by ADR-001 at the W1 gate; (4) Automation versions pin capability contracts for the life of every run; (5) agents propose, the platform validates, owners authorize — with a frozen agent-step contract; (6) webhooks persist-then-ack and bind tenants from registration; (7) the only near-term build target is W1, pulled increment by increment — no platform-first construction.
- **Adversarial findings and disposition:** the independent review of the prior vision/plan (`rpa-platform-grand-vision-review.md`) produced R1–R12; all twelve are incorporated as amendments A1–A12, each resolving its finding (§ cross-references above). A self-adversarial pass against the governing prompt's twenty failure modes found no structural residuals: primitive explosion (closed, §6 boundary + §5 gate), over-abstraction (closed — seven concepts, Process merged into Automation, Evidence/Receipt unified), workflow-engine duplication (closed — §8 boundary, ADR-001 retained), Skill/Automation/Miniapp confusion (closed — §3 is/is-not, §4 non-identity rules), capability leakage (closed — typed contracts, honest attestations), webhook confusion and security (closed — §9), inbound/outbound confusion (closed — outbound deferred), agent authority and determinism (closed — §7 contract, evidence-typed agent outputs), duplicate effects and long-lived execution (closed — §8), provider portability (closed — §8 declared-unavailable policy), multi-tenancy and versioning (closed — §11, §8 pinning), Studio duplication (closed — §12 one-backend rule), premature complexity and first-slice distortion (closed — deferred list, thin freeze, generality examples), implementation feasibility (closed — incremental within Go/Miniapp/Neon/Fly constraints, reusing accepted machinery).
- **Unresolved owner decisions:** OD-1 through OD-8 (§16) — durable-wait substrate (ADR-001), predicate subset scope, parallel semantics deferral, outbound delivery deferral, D-5 topology, W1 sandbox provider set, W1 delegation scope, usage-event schema ADR.
- **Recommended next assignment:** upon owner ratification, derive the implementation plan for S0+S1 (F0 completion; R0 contract freeze expressed as contracts-with-tests and drift tests), then the S2 W1 pull path beginning with webhook/event intake. No implementation begins before ratification.
- **Verdict: APPROVED WITH AMENDMENTS.** The vision stands as stated — process-agnostic, API-first, capability-driven, agent-native, multi-tenant, minimal at the core, webhooks first-class. Every amendment here makes an existing vision statement executable rather than altering it; the named amendments (A1–A12) are incorporated into this specification and must be carried into implementation planning.