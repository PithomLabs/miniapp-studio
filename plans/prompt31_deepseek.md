## prompt

read attached prompts, when I set out to start Miniapp Studio I envision it to be a robust RPA (robotic process automation) multi-tenant framework in Go (e.g. a single company with multiple departments or a single agency with multiple clients is considered multi-tenant), the goal is that there is an organized way to code all those tasks regardless of its app category (class A to D) that can be visualized and organized via the Studio UI in a robust software engineering methodology (ready for per PR preview environment, staging and prod environments), is that covered in our previous prompts already? the goal is to provide the ultimate roadmap for this project




# Coverage Check: Is the RPA Vision in the Prompts?

**Short answer:** The *substrate* for a robust multi-tenant RPA framework is well covered — the framework, the Studio control plane, the Neon-first data layer, the per-PR preview environments, the dual (local/Sprite) dev contract. The *RPA product layer* — the things that make it an RPA platform rather than a governed app builder — is **not** covered. Three specific gaps. The ultimate roadmap needs two axes, not one.

---

## 1. Coverage matrix

Mapping your vision to what the attached prompts actually specify:

| Vision element | Covered? | Where |
|---|---|---|
| **Go as the application language** | ✅ Covered | roadmap4.md §1 (binding invariant) |
| **Class A–D execution taxonomy** | ✅ Covered | roadmap4.md §6, writeup2.md, five-phase model |
| **Multi-tenant at the Sprite level** | ✅ Covered | roadmap4.md §2 ("one dedicated Builder per application workspace") |
| **Durable execution substrate** (DBOS, River, effect ledger) | ✅ Covered | Miniapp framework, writeup2.md |
| **Authorization / approval gates** | ✅ Covered | roadmap4.md §3, writeup2.md |
| **Three-environment parity** (local, Sprite, CI) | ✅ Covered | prompt28.1 |
| **Neon-first Postgres** | ✅ Covered | prompt28.3 |
| **Per-PR preview environments** | ✅ Covered | prompt28.2 |
| **Background coding agents** | ✅ Covered | prompt28.2, prompt28 |
| **Staging environment** | ⚠️ Implied, not designed | Between PR preview and prod in prompt28.2; no explicit spec |
| **Production promotion gates** | ⚠️ Partially covered | Phase E in roadmap4.md, but no per-task promotion |
| **Studio UI as conversation workspace** | ⚠️ Designed in prior conversation, not in prompts | Prior UI reimagining, not in prompt27-30 |
| **Sub-tenant model** (department / client) | ❌ **Missing** | Not addressed anywhere |
| **RPA task catalog** (organized coding of 100s of tasks) | ❌ **Missing** | Not addressed |
| **RPA operations dashboard** (fleet health, SLA, backlog) | ❌ **Missing** | Not addressed |
| **Scheduling / triggers at scale** (per-task cron, event, dependency) | ❌ **Missing** | DBOS/River cover per-execution; no catalog-level scheduler |
| **Cross-task orchestration** (task B after task A; alert on C) | ❌ **Missing** | Not addressed |
| **Per-task promotion** (each task versioned and promoted independently) | ❌ **Missing** | Phase E promotes *apps*, not *tasks* within a catalog |
| **RPA test strategy for scheduled tasks** (deterministic clock, replay) | ❌ **Missing** | Fixtures covered; scheduled-task regression not covered |
| **Tenant onboarding** | ❌ **Missing** | Not addressed |
| **Billing / metering per department/client** | ⚠️ Deferred | roadmap4.md §5.2 explicitly defers |

**Verdict:** ~60% covered at the substrate level, ~15% covered at the RPA product level, ~25% of your vision has no explicit design.

---

## 2. The three gaps that matter most

### Gap 1 — The sub-tenant model (department / client)

Your vision says: *"a single company with multiple departments or a single agency with multiple clients is considered multi-tenant."*

That is **sub-tenancy**. A tenant (the company or agency) contains sub-tenants (departments or clients). Each sub-tenant likely needs:

- Its own Sprite(s) — or a shared Sprite with isolation
- Its own RPA task catalog
- Its own operational dashboard
- Its own billing line
- Its own audit trail scoped to the sub-tenant
- Its own approval chain (department manager ≠ company admin)
- Its own role model (a client's admin ≠ agency's admin)

The prompts currently use **"application workspace"** as the isolation unit, which conflates "one app" with "one tenant." That doesn't scale to your model. A department might run 20 RPA tasks; a client might have 3 sub-clients. The isolation unit needs to be **the sub-tenant**, with apps nested underneath.

**Why this matters now, not later:** RLS schema, Sprite provisioning policy, audit-log scoping, billing, and Studio UI grouping all depend on this decision. Deferring it forces a schema migration and a UI rebuild later.

### Gap 2 — The RPA task catalog

The framework lets you build **a** miniapp. It says nothing about organizing **hundreds** of miniapps into a coherent automation catalog that a department head or client can browse, understand, and manage.

An RPA catalog needs:

- **Categorization** — by domain (finance, HR, ops), by app category (Class A–D), by trigger type, by business process
- **Discovery** — search, filter, "which tasks touch the invoice system?", "which tasks have failed in the last 24h?"
- **Ownership** — which task belongs to which sub-tenant, which team, which role
- **Lifecycle state** — draft, preview, staging, production, deprecated, retired
- **Dependency graph** — "task B depends on task A's output"
- **Version pinning** — "production is running v1.4 of this task; staging is on v1.5"
- **Scheduling** — when does this run, on what calendar, with what SLA

None of this is in the prompts. All of it is RPA-specific and all of it is what makes the difference between "a durable app builder" and "an RPA platform."

### Gap 3 — The RPA operations dashboard

The conversation-as-workspace UI (from the prior UI reimagining) is the **build** experience. It's what a developer or business owner uses while describing and reviewing an app.

But the **operate** experience for RPA is different:

- "All my scheduled tasks running now, across all departments"
- "Which tasks are behind SLA, which are ahead?"
- "What approvals are pending across the catalog?"
- "What broke in the last 24h and who's been notified?"
- "Cost per task per day"
- "Success rate trend per task per department"

This is the RPA operations dashboard. It's not the chat/describe interface. It's the fleet-monitoring interface — the one an ops person lives in daily. The prior UI reimagining touched on it briefly ("Operate mode") but didn't design it for RPA scale or department/client grouping.

---

## 3. The ultimate roadmap needs two axes

The prompts (27–30) are **engineering Studio itself**. That's necessary — you can't build an RPA platform on a shaky foundation. But it's not sufficient. The RPA product layer needs its own explicit workstream.

I'd split the roadmap into two axes that progress in parallel:

### Axis 1 — E-series (Engineering Studio's own discipline)

This is essentially the current B0–B5, plus the per-PR/CI work from prompt28.2:

| Phase | Deliverable | Status |
|---|---|---|
| **E0** | Studio manages one dev task | ≈ B0, ACCEPTED |
| **E1** | Studio supports its local dev loop | ≈ B1, ACCEPTED |
| **E2** | Remote dev + resident engineering agent | ≈ B2, CURRENT |
| **E3** | Per-PR previews + CI environment | From prompt28.2, NOT STARTED |
| **E4** | Staging environment and promotion gates | Partially in B3, needs design |
| **E5** | Production release management | ≈ B5 |
| **E6** | Studio troubleshoots itself | ≈ B4 |

### Axis 2 — R-series (RPA product capabilities)

This is what's currently missing:

| Phase | Deliverable | Depends on |
|---|---|---|
| **R0** | Single-tenant miniapp catalog (the substrate for R1+) | E2 |
| **R1** | Sub-tenant model (department/client) with RLS, Sprite policy, audit scoping | R0, E3 |
| **R2** | RPA task catalog (categorization, discovery, ownership, lifecycle state) | R1 |
| **R3** | RPA scheduling layer (per-task cron, event triggers, dependency graph, SLA) | R2 |
| **R4** | RPA operations dashboard (fleet health, SLA, backlog, exceptions) | R3 |
| **R5** | Per-task promotion pipeline (preview → staging → prod, per task version) | R2, E4 |
| **R6** | Tenant onboarding (agency provisions a client, company provisions a department) | R5 |
| **R7** | Billing, metering, quotas per sub-tenant | R6 |
| **R8** | Multi-tenant RPA at scale (100+ tasks per department, 10+ departments per tenant) | E5, R7 |

### Intersection points

Some E-series work unlocks R-series work; some R-series work depends on E-series discipline being real:

- **R0 needs E2** because the resident agent is how tasks get built.
- **R1 can start designing during E3**, but implementation needs per-PR previews so sub-tenant changes can be tested independently.
- **R5 needs E4** (staging promotion must work for a single app before it works per-task).
- **R8 needs E5** (production release management must exist before it can be per-task).
- **R7 needs R6** (billing needs onboarding as its data source).

**The critical path** is: E2 → E3 → R1 → R2 → R3 → R4 → E4 → R5 → R6 → R7 → E5 → R8.

You can parallelize E4/E5 design alongside R2/R3/R4, but the **critical path runs through the R-series**, because that's the RPA product.

---

## 4. What your RPA positioning actually implies

Two clarifications worth surfacing because they affect the roadmap:

**This is modern RPA, not legacy RPA.** Legacy RPA was UI-scraping (Blue Prism, UiPath). Your framework — durable workflows with effect ledger, approvals, capability proxy — is what the industry calls "orchestration-led RPA" or "API-first automation." That's the growing segment, not the declining one. Worth being explicit in positioning because it affects who the target buyer is (ops teams + IT, not just citizen developers).

**The unit of value is the task, not the app.** A tenant doesn't buy "an app." They buy "the invoice reconciliation task runs 24/7 with SLA guarantees." That framing changes what the Studio UI emphasizes: the catalog, the schedules, the SLAs, the fleet health — not "here's my app." The current prompts lean toward "here's my app." The RPA product layer shifts it to "here's my catalog of tasks."

---

## 5. What I would do next

Three concrete recommendations:

### 5.1 Before prompt31, add an R-series design cycle

Before continuing to implement E-series phases, spend one design cycle on the R-series fundamentals:

- **Sub-tenant data model** — how does `tenant_id` relate to `department_id` or `client_id`? One RLS schema, or two? What's the Sprite policy — one Sprite per department, or shared across a tenant?
- **RPA task catalog schema** — what fields, what lifecycle states, what ownership model?
- **Scheduling model** — per-task schedules, or a tenant-wide calendar? How are dependencies expressed? How are SLAs defined and enforced?
- **Operations dashboard spec** — what does the daily-driver view for an ops person look like, scoped to sub-tenant?

This cycle is one design document (ADR), not a code sprint. But its outputs feed everything in R1–R5 and prevent the "build it, then rebuild the schema" trap.

### 5.2 Add the two axes to `plans/stage2-tracks.md`

The current tracks file describes the RPA product implicitly. Making the two axes explicit:

- Removes ambiguity about what "complete" means
- Makes sub-tenant scoping a first-class concern instead of an emergent property
- Provides a place to record the R-series dependencies
- Makes the E-series feel like *infrastructure* rather than *the product* — which it is

### 5.3 Reword the B-series to be honest about scope

"B0 — Studio manages one development task" is about Studio's own engineering. "B2 — Remote development and resident engineering agent" is about Studio's own engineering. These are correct phases — they're the foundation. But calling them the roadmap implies they're the whole roadmap. Calling them **E0–E6** and introducing **R0–R8** makes it clear that the RPA product is the second axis, and that building Studio's engineering discipline is *preparation* for the RPA product, not the product itself.

---

## 6. The honest summary

You set out to build a robust RPA multi-tenant framework in Go. The prompts have built:

- ✅ A solid framework for Class A–D execution (the substrate)
- ✅ A control plane (Studio) that orchestrates agents building miniapps
- ✅ A modern data layer (Neon-first) with proper isolation
- ✅ A rigorous engineering methodology (per-PR preview, three-environment parity)

What they have **not** built:

- ❌ The sub-tenant model (department / client) that makes it multi-tenant RPA
- ❌ The RPA task catalog (organizing 100s of miniapps into a manageable inventory)
- ❌ The scheduling layer (per-task cron, dependencies, SLAs)
- ❌ The operations dashboard (fleet health, backlog, exceptions, cost)
- ❌ The per-task promotion pipeline (each task independently staged and promoted)
- ❌ The onboarding and billing flows that make it SaaS

**The current work is not misdirected — it's the necessary foundation.** But the roadmap is incomplete without the RPA product layer, and the two most consequential design decisions (sub-tenancy, task catalog) must be made *before* further implementation, not after, because they change the schema, the Studio UI, and the authorization model.

**The ultimate roadmap is not B0–B5 extended.** It is **E0–E6 parallel to R0–R8**, with three explicit design decisions (sub-tenancy, task catalog, scheduling model) front-loaded before any R-series code begins.

The prompts have been building a governed app builder. That's not the same product as a multi-tenant RPA platform. The difference is the R-series — and the R-series is what makes the vision real.