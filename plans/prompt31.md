## prompt

read attached prompts, when I set out to start Miniapp Studio I envision it to be a robust RPA (robotic process automation) multi-tenant framework in Go (e.g. a single company with multiple departments or a single agency with multiple clients is considered multi-tenant), the goal is that there is an organized way to code all those tasks regardless of its app category (class A to D) that can be visualized and organized via the Studio UI in a robust software engineering methodology (ready for per PR preview environment, staging and prod environments), is that covered in our previous prompts already? the goal is to provide the ultimate roadmap for this project

## deepseek

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


## chatgpt

# **My assessment: Miniapp Studio's original vision is only partially covered**

Having read all seven prompts (27.1 through 30) together, I think there is an important distinction we need to recognize before committing to the ultimate roadmap.

**The previous prompts establish a strong foundation for an AI-native software engineering platform. They do not yet fully specify the multi-tenant RPA platform you originally envisioned.**

These are related, but they are not the same product capability.

Your original vision is substantially broader:

> Miniapp Studio is a robust, multi-tenant RPA and business automation engineering platform, built in Go, where organizations can develop, organize, execute, visualize, govern and continuously improve automations of different application categories through one coherent software engineering methodology.

An organization might be a company with multiple departments, or an agency managing multiple client businesses. Each needs independently governed automation assets, data, credentials, executions and environments.

This distinction should become foundational to the next roadmap.

## **1. What the previous prompts already cover**

There is considerably valuable architecture already established. I would preserve it rather than start over.

| Capability                          | Existing coverage                                           | My assessment                         |
| ----------------------------------- | ----------------------------------------------------------- | ------------------------------------- |
| Go-based backend                    | Go owns application behavior, APIs and canonical interfaces | Strong foundation                     |
| AI coding agents                    | Resident agent, Fly Sprites, background execution           | Strong                                |
| Engineering task management         | GitHub Discussions, Issues, Project Task Cards, PRs         | Strong                                |
| Authorization                       | Explicit owner approval, refusal, scoped credentials        | Strong                                |
| Database isolation                  | Neon, RLS, tenant isolation, branch strategy                | Partial                               |
| CI                                  | Isolated execution, commit-specific verification            | Designed                              |
| Per-PR previews                     | Fly apps and Neon branches                                  | Designed                              |
| Staging and production              | Application targets and authorization distinctions          | Partial                               |
| Release management                  | B5 future milestone                                         | Not fully specified                   |
| RPA task framework                  | General business automation definitions and execution       | Not specified                         |
| Organization/dept/client hierarchy  | General tenant isolation only                               | Not specified                         |
| Categories A–D                      | Classification axis identified                              | Not fully specified                   |
| Visual automation management        | Focus Console and Studio UI foundations                     | Partial                               |
| Durable business workflow execution | Some engineering execution lifecycle work                   | Not specified as a general RPA engine |

The distinction is supported by the original product description.

Prompt27.1 defines Miniapp Studio's north star as **The Self-Developing Software Engineer**, with a resident coding agent in a persistent Sprite, GitHub Connector, checkpoint recovery and evidence returning to Studio.&#x20;

prompt27.1.md



Prompt28 then locks B0–B5 around Studio managing development tasks, remote engineering, application preview, self-troubleshooting and release management. That is a development-platform roadmap, not a complete RPA product roadmap.&#x20;

prompt28.md



This is an important difference.

## **2. The most important distinction: two kinds of tasks**

I think we've been using the word *task* for two fundamentally different things.

Engineering work

**Software Engineering Task**

Examples:

- Fix a failing integration test.
- Implement a connector.
- Add a new automation capability.
- Refactor a Go package.
- Prepare a pull request.

Managed through Discussions, Project Task Cards, Git branches, PRs, CI and coding agents.

Business automation

**RPA Task / Automation**

Examples:

- Synchronize customer records between CRM systems.
- Process an incoming invoice.
- Reconcile data between business applications.
- Generate and distribute reports.
- Execute a multi-step approval process.

Managed through tenant-owned automation definitions, workflows, triggers, connectors, execution policies and run histories.

The first kind builds and maintains the platform.

The second kind is what customers actually buy and use.

Our current GitHub Project model explicitly makes Project Task Cards implementation work packages, while Issues are for bugs and Discussions are for initiatives. That is a good engineering governance model, but it is not the customer-facing RPA task model.&#x20;

prompt27.1.md



**We need both, with separate domain models and lifecycles.**

## **3. The missing foundation: real hierarchical multi-tenancy**

This is the most important architectural gap I see.

The prompts already require RLS, tenant isolation, branch-scoped database credentials and evaluation of project-per-miniapp versus project-per-tenant. Those are valuable database and infrastructure safeguards.&#x20;

prompt28.1.md

prompt28.3.md



But multi-tenancy is more than database isolation.

We need to define the business ownership hierarchy.

Consider these two examples.

**Example A — One company, multiple departments**

**Acme Corporation**

Organization / account

**Finance**

Tenant scope

**Human Resources**

Tenant scope

**Operations**

Tenant scope

Each department manages its own automation assets, connections, execution records and access policies. Authorized corporate administrators can have cross-department visibility.

**Example B — One agency, multiple clients**

**Automation Agency**

Agency / managing organization

**Client Alpha**

Client tenant

**Client Beta**

Client tenant

**Client Gamma**

Client tenant

The agency manages multiple independent client environments. Agency operators receive explicitly delegated access without automatically gaining unrestricted access to every client's records or credentials.

These two scenarios should be first-class use cases, not workarounds.

### **What the domain model needs**

I recommend evaluating a model along these lines:

| Domain object | Responsibility                                              |
| ------------- | ----------------------------------------------------------- |
| Organization  | Company, agency or account owner                            |
| Tenant        | Independent business/security scope                         |
| Workspace     | Collaborative working area within a tenant                  |
| Membership    | User or agent's relationship to an organization or tenant   |
| Role / Policy | What that member is permitted to do                         |
| Delegation    | Explicit access granted to a parent agency or administrator |
| Automation    | Business automation owned by a tenant                       |
| Connection    | External integration and its credentials                    |
| Environment   | Development, preview, staging or production scope           |

A tenant should have its own ownership, isolation, access rules and operational boundaries.

Important invariants:

- No implicit cross-tenant access.
- A user may belong to multiple tenants.
- An agency can manage multiple clients without becoming the owner of all client data.
- Department administrators can operate within their departments.
- Organization-level administrators can receive explicitly scoped oversight.
- Automations, run records, logs, files, secrets and connectors must carry tenant ownership.
- Shared automation templates must not imply shared customer data or credentials.

We must decide which resources inherit ownership, which may be shared, and which require explicit delegation.

Neon project-per-tenant versus project-per-client is an infrastructure decision. It must implement the logical tenancy model, not define it.

This distinction is especially important because a single shared Neon project with many branches does not automatically establish proper business-level authorization.

## **4. The second missing foundation: a unified automation model**

I would not build Miniapp Studio as four unrelated RPA products for categories A, B, C and D.

Instead, I would establish **one common automation engineering model with category-specific execution capabilities and policies.**

An automation should have a well-defined identity, lifecycle and contract.

At a conceptual level:

## **Automation definition**

**Identity and ownership**

Tenant · Workspace · Automation ID · Version

**Classification and execution policy**

Category A–D · Compute placement · Authorization policy

**Inputs and outputs**

Typed inputs · Outputs · Schemas · Validation

**Workflow definition**

Triggers · Steps · Dependencies · Branches · Conditions · Connectors

**Operational contracts**

Timeouts · Retries · Idempotency · Permissions · Compensation · Audit

**Delivery metadata**

Version · Tests · Environment configuration · Release history

These are proposed architecture elements, not claims that the current repository already implements them.

The central idea is that automation definitions should be version-controlled, testable and inspectable.

### **How I would approach Go**

The platform should have a reusable Go automation framework rather than having every new business automation reinvent its own infrastructure.

A good direction is to separate:

- Automation definitions and contracts.
- Workflow composition.
- Executable activities.
- Connector interfaces.
- Execution policies.
- Runtime scheduling and orchestration.
- Persistence and recovery.

The framework should make it straightforward to build a new automation using reusable Go components.

The Studio UI should visualize that same canonical definition. It should not maintain an independent workflow interpretation that can drift from the executable code.

Whether the authoring experience is entirely Go-code-first or eventually supports visual editing should be an explicit architectural decision. Either way, there should be one authoritative representation.

### **Categories A–D**

There is already a good starting principle in prompt28:

1. Execution category: A, B, C or D.
2. Compute placement: Sprite, Machine or Hybrid.
3. Authorization: dry-run, confirm or autonomous.

These are explicitly independent dimensions.&#x20;

prompt28.md



I strongly agree with that separation.

However, the attached prompts do not define the actual operational contracts for each class.

We must consult the canonical `app_category_guidelines.md` before deciding what each class permits.

The eventual architecture must translate the category definitions into enforceable execution profiles, including appropriate capabilities, isolation, resource limits, permissions, observability and acceptance requirements.

A category must be more than a label shown in the UI.

## **5. The third missing foundation: a real RPA execution runtime**

The current prompts describe durable execution for coding agents, including Sprite lifecycle, webhooks, retries, cancellation, reconciliation, isolated workspaces and execution evidence.

Those are excellent engineering foundations.

But executing a coding agent is different from executing thousands of customer business automations.

The RPA runtime needs its own lifecycle.

I would expect a business automation run to support:

| Capability     | What it must guarantee                                           |
| -------------- | ---------------------------------------------------------------- |
| Triggers       | Schedule, webhook, event or manual execution                     |
| Scheduling     | Due-time and recurring execution                                 |
| Queueing       | Controlled concurrency and resource allocation                   |
| Idempotency    | Safe handling of duplicate triggers                              |
| State          | Durable execution and step progress                              |
| Retries        | Explicit, bounded retry policies                                 |
| Cancellation   | Safe interruption where supported                                |
| Human approval | Pause, approval and resume                                       |
| Error handling | Classification, escalation and recovery                          |
| Compensation   | Controlled reversal of completed side effects where possible     |
| Observability  | Run history, step logs, inputs/outputs and evidence              |
| Auditing       | Attribution of actions to tenant, user, automation and execution |
| Recovery       | Reconciliation after process, machine or network failures        |

For example, consider an invoice-processing automation:

1. Receive an invoice.
2. Extract information.
3. Validate the supplier.
4. Check the purchase order.
5. Request human approval if policy requires it.
6. Submit the approved invoice to the accounting system.
7. Record the result.
8. Notify the appropriate department.

If the accounting API times out after receiving the request, the runtime needs to distinguish an unknown outcome from a definite failure. It must not blindly submit the invoice twice.

This is where reliable execution, idempotency, durable state and compensation become essential.

The existing prompt28.2 controller requirements provide useful patterns, but they are directed at the agent, CI and preview lifecycles, not this general business-automation runtime.&#x20;

prompt28.2.md

prompt28.2.md



## **6. The fourth missing foundation: Studio as an automation engineering workbench**

This is where your original vision becomes a tangible product.

The Studio UI shouldn't merely display coding-agent sessions, GitHub issues and deployment states.

It should help users understand and manage their entire automation portfolio.

I envision the following major areas.

## **Proposed Miniapp Studio workspace**

**Organization / Tenant selector**

Acme Corporation / Finance Department

**Automation Catalog**

Discover, classify and organize automations

**Workflow Designer**

Understand steps, dependencies and execution paths

**Execution Monitor**

Runs, errors, retries and live progress

**Integrations**

Tenant-scoped connectors and capabilities

**Engineering**

Code, agents, reviews, CI and PRs

**Environments & Releases**

Preview, staging, production and rollback

**Governance**

Permissions, approvals and audit trails

**Usage & Performance**

Costs, capacity and automation effectiveness

Conceptual information architecture, not an existing UI implementation.

A particularly important feature would be the automation detail view.

An operator should be able to open one automation and see:

- Which organization and tenant own it.
- Its category and execution profile.
- Its source code and version.
- Its workflow or dependency graph.
- Its connected business applications.
- Its current deployment environment.
- Its latest executions and failure history.
- Its outstanding approvals.
- Its active schedule and triggers.
- Its test results and release history.

This is how Studio becomes the central control plane for business automation rather than just another developer dashboard.

## **7. The fifth missing foundation: software engineering lifecycle for customer automations**

This is where prompt28.2 is particularly valuable, but we need to extend its scope.

The existing prompt describes three distinct lifecycles:

- Coding agent lifecycle.
- CI lifecycle.
- PR preview lifecycle.

These are appropriate for developing Miniapp Studio itself.&#x20;

prompt28.2.md



However, a customer automation needs its own delivery lifecycle.

I recommend the following:

## **Automation software delivery lifecycle**

1. **Development**

   Author Go code, automation contracts, workflows and tests.
2. **Pull request**

   Version-controlled changes, review, CI and automated quality checks.
3. **PR preview**

   Isolated Fly application and Neon branch for the changed automation.
4. **Staging**

   Deploy the approved immutable artifact using staging-specific configuration and connectors.
5. **Production**

   Promote the accepted version with explicit authorization, audit and health checks.
6. **Rollback**

   Restore the previous approved version without losing execution history or corrupting in-flight runs.

There should be a critical distinction between three environments:

| Environment | Purpose                                                                  |
| ----------- | ------------------------------------------------------------------------ |
| Development | Authoring and developer testing                                          |
| PR preview  | Review an isolated, unmerged change                                      |
| Staging     | Validate a release candidate in a controlled production-like environment |
| Production  | Execute customer business automations                                    |

The staging and production distinction already appears in the Project Target fields and authorization rules. B5 is also explicitly described as release management. But the full release-promotion model remains future work.&#x20;

prompt27.1.md

prompt28.md



Prompt28.2 explicitly says its per-PR preview requirement is for the Miniapp Studio repository, not automatically customer-created miniapps. That is a crucial gap to resolve in the future product architecture.&#x20;

prompt28.2.md



My preferred rule:

**Build once, verify once, promote the same immutable automation artifact through environments.**

Environment-specific items should be configuration, credentials, resource allocation and policies—not silently different application code.

Also, a PR preview must never share mutable customer production state.

## **8. What I would preserve from the current architecture**

I don't recommend throwing away any of the important decisions made through prompt30.

The current architecture provides several foundational building blocks:

| Existing decision            | Long-term RPA value                                  |
| ---------------------------- | ---------------------------------------------------- |
| Go owns application behavior | Common execution framework and reusable capabilities |
| Taskfiles declare workflows  | Reproducible engineering and operations              |
| Fly Sprites                  | Isolated coding-agent execution                      |
| Fly Machines                 | Long-lived control-plane and automation compute      |
| Neon everywhere              | Database consistency and tenant-aware persistence    |
| GitHub governance            | Version control, reviews and change history          |
| Explicit authorization       | Governed business side effects                       |
| Structured evidence          | Auditable automation execution                       |
| Per-operation deadlines      | Reliable execution and bounded failure               |
| Dirty-state semantics        | Reliable migration and state management              |
| Independent lifecycles       | Isolation of development, testing and production     |

In particular, prompt30's bounded migration design is foundational operational engineering. Reliable automation requires reliable persistence, migration and transaction handling. However, prompt30 is a database-reliability deliverable, not a definition of the business automation engine.&#x20;

prompt30.md



Likewise, the Neon-first architecture is important, but its project and branch model must ultimately serve the logical tenant model, rather than dictate it.&#x20;

prompt28.3.md



# **9. My recommendation: establish two coordinated roadmaps**

I would not discard the existing B0–B5 roadmap. That would risk disrupting accepted decisions and the ongoing engineering work.

Instead, I recommend treating it as one of two complementary roadmaps.

Roadmap A

**Miniapp Studio — Engineering Platform**

How Studio develops, tests, maintains and releases itself.

- B0: Manage a development task.
- B1: Local development loop.
- B2: Remote engineering agent.
- B3: Formal Application Preview.
- B4: Self-troubleshooting.
- B5: Release management.

Roadmap B

**Miniapp Studio — Multi-Tenant RPA Platform**

What customers actually use to build, operate and govern their business automations.

- Organization and tenant management.
- Automation definition and development.
- Category-specific execution.
- Durable workflow runtime.
- Connector and integration ecosystem.
- Automation visualization and management.
- Testing, preview, staging and production.
- Governance, observability and usage management.

The engineering platform enables the RPA platform. Both should share the same Go foundations, security architecture, CI methodology, database contracts and deployment practices.

## **10. Proposed ultimate product roadmap**

This is the roadmap I would establish at the product-architecture level. These are recommended future workstreams, not claims of existing implementation.

| Phase | Workstream                       | Major outcomes                                                                                         |
| ----- | -------------------------------- | ------------------------------------------------------------------------------------------------------ |
| P0    | Product architecture             | Canonical domain model, tenancy model, execution categories A–D and capability contracts               |
| P1    | Multi-tenancy foundation         | Organization, department/client hierarchy, membership, delegated access, RLS and isolation             |
| P2    | Automation development framework | Go SDK, automation definitions, reusable packages, workflow contracts, versions and tests              |
| P3    | Execution engine                 | Triggers, scheduling, queues, durable runs, retries, idempotency, cancellation and recovery            |
| P4    | Category execution profiles      | Category-specific adapters, capabilities, permissions and execution policies                           |
| P5    | Connector ecosystem              | Tenant-scoped integrations, credential management, API discovery and reusable connector capabilities   |
| P6    | Studio automation workbench      | Catalog, visualization, task organization, workflow graphs, execution monitoring and approvals         |
| P7    | Engineering delivery pipeline    | PR validation, isolated previews, Neon branches, artifact versioning and automated testing             |
| P8    | Environment promotion            | Staging, production, release approvals, immutable promotion and rollback                               |
| P9    | Enterprise operations            | Audit, quotas, metering, cost attribution, service health, tenant lifecycle and operational governance |

### **Suggested architectural dependency order**

**P0 — Domain and architecture contracts**

**P1 — Tenancy**

**P2 — Automation model**

**P3 + P4 — Runtime and execution profiles**

**P5 — Integrations**

**P6 — Studio UI**

**P7 + P8 — Engineering delivery and environment promotion**

**P9 — Enterprise operations**

Conceptual dependency model. P5 and P6 can develop in parallel once the domain and runtime contracts stabilize. Security, testing and observability are cross-cutting from P0 onward.

# **11. Five architectural questions that must be settled**

Before turning this into an authoritative roadmap, I would insist on explicit decisions about five subjects.

**1. What exactly is a tenant?**

Define how organizations, departments, client businesses, workspaces and delegated agency access relate to each other.

The same model must work for both company-with-departments and agency-with-clients scenarios.

**2. What is the canonical automation artifact?**

Define what a developer creates, versions, tests and deploys. Is it a Go package, a workflow definition, a versioned application containing both, or a combination?

There must be one clear source of truth.

**3. What do categories A–D mean operationally?**

Get the actual category definitions from `app_category_guidelines.md`. Define how each affects execution capabilities, isolation, placement, authorization and verification.

Do not invent the class semantics.

**4. What guarantees does the RPA runtime provide?**

Define durable execution, idempotency, failure recovery, human approvals, concurrency, scheduling and side-effect handling.

Do not assume that reliable coding-agent execution automatically provides reliable business workflow execution.

**5. How does an automation reach production?**

Define how a versioned automation moves from development to PR preview, staging and production, including approval, configuration, database migrations and rollback.

These five questions have much greater long-term architectural impact than another round of UI polishing.

# **12. The document I recommend creating next**

I recommend an overarching architecture initiative:

Recommended next initiative

# **Prompt31 — Miniapp Studio: Ultimate Product Architecture and Roadmap**

**A Multi-Tenant, Go-Native RPA and Automation Engineering Platform**

This should become the authoritative product-level document above the existing prompt27–30 implementation directives.

Its responsibility is to define the product, domain model, system boundaries, execution contracts, user experience and long-term roadmap.

Existing prompts remain specialized implementation workstreams. They should not be rewritten unnecessarily.

I would require Prompt31 to produce these concrete architectural deliverables:

1. Product vision and non-goals.
2. Organization, tenant and delegated-access domain model.
3. Unified automation and workflow domain model.
4. Class A–D execution architecture.
5. Go automation SDK and developer methodology.
6. Durable RPA execution runtime.
7. Connector and integration architecture.
8. Studio UI information architecture and visualization model.
9. Development, CI, PR preview, staging and production lifecycle.
10. Security, isolation, governance and observability architecture.
11. Dependency-ordered multi-phase roadmap.
12. Requirement traceability from the original vision through B0–B5 and future RPA capabilities.

It should also include an explicit gap matrix identifying what is already implemented, what is merely designed, what is underway, and what has not yet been addressed.

**The agent must not immediately start implementing this new architecture.** First it should inspect the governing Discussion #2, `app_category_guidelines.md`, existing ADRs, current repository, actual B0–B5 evidence and the seven prompts you supplied. Then it should produce a coherent product architecture and roadmap for your review.

## **My final perspective**

I think your original ambition is more interesting than simply building an AI coding-agent interface.

The coding agent is an important part of Miniapp Studio, but it should ultimately be one capability inside a much larger automation engineering platform.

The bigger opportunity is:

**A company or agency should be able to use Miniapp Studio as its central engineering workspace for creating, organizing, governing and operating all its business automations—regardless of application category, execution method or customer environment.**

An automation developer should write reusable Go code, define its execution contract, connect business applications, test it, visualize its workflow and deploy it through a professional software delivery process.

An operator should see what is running, where it is running, who authorized it, which tenant owns it, what happened when it failed and how to recover it.

An agency should manage dozens or hundreds of client environments without compromising their isolation.

And the system should make all of this understandable through one coherent Studio UI.

**That is the product I believe we should architect toward.**

The previous prompts have built important parts of its foundation, especially the engineering lifecycle, isolation, Neon, Fly and agent execution. But I would not yet call them the ultimate RPA roadmap.

The next architectural milestone should be defining that larger product explicitly, while preserving the ongoing Neon migration reliability work and B2 progress.