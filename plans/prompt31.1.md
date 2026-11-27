**Yes. I agree with this direction, and I think it resolves an important product-design ambiguity.** I would make it a binding architectural decision for Miniapp Studio.

We should build two distinct user experiences, preserve the existing developer-oriented UI, and build both experiences on one common platform foundation.

More importantly, I agree with your demo requirement:

**Roadmap A must present the complete B0–B5 developer experience in the UI, even where the corresponding functionality is still planned or incomplete.**

We should not confuse completeness of the product's visual and navigational structure with completeness of its implementation.

This is a legitimate software-engineering approach, provided every feature is represented honestly.

# **1. Consolidated product architecture**

I recommend keeping the names Roadmap A and Roadmap B rather than renaming B0–B5 to E0–E6 as suggested by the attached analysis.

# **Miniapp Studio**

One platform · One common foundation · Two specialized workspaces · Two coordinated roadmaps

Roadmap A

**Engineering Studio**

# **B0–B5**

Developers, coding agents and software engineering teams.

Build · Code · Test · Debug · Preview · Release

Roadmap B

**Automation Operations Studio**

# **R0–R8**

Operations, QA, department managers and agency/client administrators.

Organize · Monitor · Approve · Operate · Optimize

**Common Platform Foundation**

Go · Hierarchical tenancy · Identity · Authorization · Automation contracts · Execution · Connectors · Neon · Fly · Audit · Environments · Observability

This is the key architectural decision:

- Roadmap A is the developer's engineering workbench.
- Roadmap B is the operator's automation management and operations workbench.
- They share domain models, services, permissions, data contracts and infrastructure.
- They must not become two independently implemented products with duplicated business logic.

The attached analysis correctly identifies that the engineering workspace and the RPA operations dashboard serve fundamentally different daily workflows.&#x20;

prompt31_deepseek.md



## **2. Roadmap A — preserve B0–B5 and complete the UI**

I would retain the existing roadmap without renaming or reordering it.

| Milestone | Purpose                                           | UI requirement                                                         |
| --------- | ------------------------------------------------- | ---------------------------------------------------------------------- |
| B0        | Studio manages one development task               | Task creation, organization, status and detail                         |
| B1        | Studio supports its local development loop        | Project workspace, development state, tests and diagnostics            |
| B2        | Remote development and resident engineering agent | Sprite, agent activity, connectors, approvals, logs and evidence       |
| B3        | Formal Application Preview                        | Preview environments, deployment status, URLs and health               |
| B4        | Studio troubleshoots itself                       | Diagnostics, incidents, explanations and recovery workflows            |
| B5        | Release management                                | Version history, staging, production promotion, approvals and rollback |

These remain the established B0–B5 definitions.&#x20;

prompt28.md





My recommendation

&#x20;Complete the UI architecture and screen coverage for all six milestones, even though their implementation maturity differs.



### **The UI completeness contract**

I would introduce two independent acceptance dimensions.

| Acceptance dimension    | Meaning                                                                                |
| ----------------------- | -------------------------------------------------------------------------------------- |
| UI completeness         | All required screens, navigation, layouts, and representative states exist             |
| Functional completeness | Actual operations work and have corresponding implementation and verification evidence |

For example:

- B0 and B1 can show their established functional capabilities.
- B2 shows the actual working resident-agent experience, including evidence and authorization.
- B3 shows the complete Application Preview UI, with unimplemented deployment capabilities clearly marked.
- B4 shows the intended troubleshooting and recovery interface, without pretending automated recovery works.
- B5 shows the release-management interface, without pretending production promotion or rollback is operational.

The demo should be visually coherent across B0–B5. It should not misrepresent the implementation status.

I would require explicit UI states such as `Available`, `Partially available`, `Planned`, `Not connected` and `Unavailable`.

Use illustrative fixtures only when clearly labeled as examples. Never present fabricated execution history or deployment status as live evidence. Unimplemented buttons should be visibly disabled or explicitly identified as planned.

This gives you a complete product experience to present to judges, developers and future customers without making misleading claims.

## **3. The common foundation must be designed before either roadmap expands**

This is where I would be particularly careful. Two UIs must not evolve into two systems with different ideas of ownership, permissions, automation state or deployments.

I recommend a shared foundation with eight architectural pillars.

| Shared pillar          | What both roadmaps use                                               |
| ---------------------- | -------------------------------------------------------------------- |
| Identity and tenancy   | Organizations, tenants, users, memberships and delegated access      |
| Authorization          | Permissions, capabilities, approvals and audit                       |
| Automation domain      | Automation definitions, ownership, classification and versions       |
| Execution contracts    | Activities, workflows, jobs, runs, status and recovery               |
| Integration platform   | Connectors, capability discovery and tenant-scoped credentials       |
| Environment management | Development, preview, staging and production                         |
| Persistence            | Neon, migrations, RLS, transactional consistency and durable records |
| Observability          | Logs, events, evidence, metrics, execution history and diagnostics   |

### **The tenancy model deserves special attention**

The DeepSeek analysis correctly identifies the department/client problem, but I would refine its proposed terminology before adopting it.

Don't simply assume every department, client or application workspace must become a child of one universal tenant object.

I recommend explicitly modeling:

- **Organization:** A company, agency or other business entity.
- **Tenant scope:** A security and data-ownership boundary.
- **Workspace:** A collaborative environment within an authorized scope.
- **Membership:** A user's relationship to an organization or tenant.
- **Delegation:** Explicit authority granted to an external organization or service provider.

This handles both patterns:

**Company:** One organization with Finance, HR and Operations tenant scopes.

**Agency:** One agency organization with separately owned client organizations and explicit delegated access.

The distinction matters because an agency employee may legitimately operate across several client environments without owning their data.

Do not equate an application workspace or a Fly Sprite with a tenant. The previously established Sprite-per-app and one-active-writer principles should remain infrastructure decisions, not the definition of business tenancy.

### **The automation model must be shared**

I recommend making `Automation` a first-class business domain object.

A conceptual model:

**Organization / Tenant**

**Automation**

Owner · Business purpose · Category A–D · Lifecycle

**Automation versions**

Source · Contract · Tests · Dependencies

**Workflow definitions**

Triggers · Activities · Dependencies · Rules

**Execution runs**

Status · Evidence · Results · Recovery

**Deployment targets**

Development · Preview · Staging · Production

An automation might contain several workflows, activities, integrations and deployable application components. It should not automatically be equated with one Miniapp, one Fly application or one Sprite.

This distinction will be important when a department manages hundreds of automations.

### **Preserve the category model**

The existing three dimensions remain independent:

1. Execution category: A, B, C or D.
2. Compute placement: Sprite, Machine or Hybrid.
3. Authorization: dry-run, confirm or autonomous.

These must not be collapsed into one category or deployment setting.&#x20;

prompt28.md



The actual Class A–D definitions must come from the canonical `app_category_guidelines.md`. The coding agent should inspect that source and translate its definitions into enforceable contracts rather than inventing new category semantics.

## **4. Roadmap B — separate Operations, QA and Management Studio**

I would use the attached R0–R8 proposal as a starting point, but revise its sequencing to avoid building a single-tenant catalog that later needs to be redesigned.

| Phase | Scope                   | Major outcome                                                           |
| ----- | ----------------------- | ----------------------------------------------------------------------- |
| R0    | RPA domain architecture | Tenant model, Automation contract, category rules and operating model   |
| R1    | Multi-tenancy           | Organization, department/client, membership, delegation and isolation   |
| R2    | Automation catalog      | Inventory, ownership, categorization, discovery and versioning          |
| R3    | Automation runtime      | Triggers, scheduling, dependencies, durable execution, retries and SLAs |
| R4    | Operations workspace    | Fleet health, run history, QA, approvals and exception management       |
| R5    | Automation delivery     | Per-automation preview, staging, production promotion and rollback      |
| R6    | Tenant onboarding       | Company departments, agency clients, provisioning and lifecycle         |
| R7    | Commercial operations   | Metering, quotas, cost attribution and billing                          |
| R8    | Enterprise scale        | Capacity, resilience, governance and large automation portfolios        |

An important change: R0 should establish the shared domain contracts before R1 and R2 implementation. There should be no temporary single-tenant catalog that becomes the permanent architectural foundation.

The R-series should have its own UI information architecture from the beginning, even though its complete implementation follows the relevant shared-foundation gates.

### **What the Operations Studio should eventually look like**

## **Automation Operations Workspace**

**Organization / Tenant context**

Acme Corporation / Operations Department

Active automations

# **128**

Illustrative placeholder

Pending approvals

# **7**

Illustrative placeholder

**Operations navigation**

Automation portfolio

Schedules and triggers

Execution monitoring

QA and test results

Approvals and exceptions

Workflow dependencies

Business integrations

Environments and releases

Governance and audit

Cost and performance

This is a conceptual illustration, not live Studio telemetry or a claim of implemented features.

A finance manager should not need to understand Sprite lifecycle or Go build logs just to know whether invoice reconciliation ran successfully.

A developer should not have to navigate an SLA dashboard to find the relevant code, branch or test failure.

That separation is precisely why the two UI experiences matter.

## **5. Important corrections to the attached analysis**

I agree with its overall two-roadmap conclusion, but would not adopt every recommendation verbatim.

| DeepSeek proposal                                    | Consolidated decision                                                                              |
| ---------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| Engineering roadmap renamed E0–E6                    | Reject the rename. Retain B0–B5.                                                                   |
| R0 single-tenant catalog first                       | Revise. Establish tenant and Automation contracts before catalog implementation.                   |
| Sub-tenant must always be the isolation unit         | Refine. Model organization, tenant security scope and delegation explicitly.                       |
| Each sub-tenant potentially gets its own Sprite      | Do not assume. Compute placement is governed separately by execution category and security policy. |
| Durable execution through DBOS/River already covered | Not established as a binding technology decision. Evaluate against actual runtime requirements.    |
| RPA task catalog and operations dashboard            | Accept as essential missing product capabilities.                                                  |
| Per-task versioning and promotion                    | Accept. Distinguish automation releases from whole application releases.                           |
| Two roadmaps progressing in parallel                 | Accept, with shared-foundation dependencies and controlled implementation sequencing.              |
| Quantitative coverage estimates (60%, 15%)           | Discard as unsupported subjective measurements.                                                    |

In particular, prompt28.2 explicitly says not to introduce DBOS automatically and to evaluate whether PostgreSQL-backed state, leases and an outbox are sufficient. The earlier engineering mission also treats DBOS/River abstractions as non-goals until justified. We should preserve that discipline.&#x20;

prompt28.2.md

prompt28.md



## **6. One more important distinction: four different environment concepts**

We should reconcile several uses of the word *preview* in our existing plans.

| Environment / concept  | Purpose                                                  | Roadmap                                |
| ---------------------- | -------------------------------------------------------- | -------------------------------------- |
| Local Development      | Developer's workstation and development branch           | A: B1                                  |
| Workspace Preview      | Application running within the Builder Sprite            | A: B2                                  |
| Engineering PR Preview | Test a proposed change to Miniapp Studio itself          | A: engineering delivery infrastructure |
| Application Preview    | Preview a developed miniapp or automation before release | A: B3 and B: R5                        |
| Staging                | Validate an approved release candidate                   | A: B5 / B: R5                          |
| Production             | Operate approved customer automations                    | A: B5 / B: R5                          |

These are not interchangeable.

The existing prompts already distinguish Workspace Preview from formal Application Preview, while prompt28.2 establishes the independent CI and per-PR engineering lifecycle.&#x20;

prompt28.md

prompt28.2.md



The common environment and artifact contracts should eventually support both engineering changes and customer automation releases without sharing their mutable databases, secrets or execution state.

# **7. Proposed implementation and design sequence**

I would not start building Roadmap B functionality immediately. I would use this sequence:

1. **Common Foundation Architecture (F0)**

   Resolve and document tenancy, ownership, authorization, Automation artifacts, execution contracts, environment lifecycles and shared UI/API boundaries.
2. **Complete Roadmap A UI coverage (B0–B5)**

   Preserve the existing developer interface. Complete all planned screen structures, navigation and states. Make functionality maturity explicit.
3. **Finish Roadmap A fundamentals**

   Continue B2 and complete the B3, B4 and B5 architectural and UI foundations, with the underlying implementations sequenced according to their gates.
4. **Roadmap B architecture and UI design (R0)**

   Produce the Operations Studio information architecture and its contracts against the same foundation. Do not build a separate tenant, permission or execution system.
5. **Roadmap B implementation (R1–R8)**

   Progressively introduce the actual multi-tenant RPA catalog, scheduling, execution operations, QA, management and per-automation release capabilities.

One qualification: existing B0/B1 functionality is already accepted, and B2 is the active milestone. We should not reopen them without evidence. Complete their UI consistency and shared-foundation alignment without destabilizing accepted work.

The shared foundation is a design and contract gate, not an excuse to postpone the working B2 demonstration.

# **8. Consolidated Prompt31 directive**

I recommend using the following as the governing instruction for the next architecture-planning pass. It incorporates the useful parts of `prompt31_deepseek.md`, your new UI direction, and the established decisions in prompts 27–30.

# PROMPT31.1 — Miniapp Studio: Unified Product Architecture, Dual Roadmaps and Shared Foundation

## Mission

Act as the Principal Software Architect and Product Architecture Lead for PithomLabs Miniapp Studio.

Produce the authoritative long-term product architecture and implementation roadmap for Miniapp Studio.

The original product vision is a robust, multi-tenant, Go-native RPA (Robotic Process Automation) and business automation engineering platform.

Organizations must be able to develop, organize, govern, execute, monitor and continuously improve their business automations through Miniapp Studio, regardless of application category (Class A–D).

Two distinct product experiences are required:

- **Roadmap A — Engineering Studio:** For developers, coding agents and software engineering teams.
- **Roadmap B — Automation Operations Studio:** For operations, QA, department managers, agency operators and business administrators.

Both roadmaps must be built on one coherent, shared platform foundation.

The architecture must avoid creating two independently implemented products with duplicated identity, tenancy, authorization, execution, data and infrastructure layers.

## 1. Governing product decisions

Preserve the following decisions:

1. Retain the existing B0–B5 roadmap. Do not rename it E0–E6.
2. Preserve the existing developer-oriented UI design as Roadmap A.
3. Introduce a separate Operations/QA/Management UI for Roadmap B.
4. Establish one common domain model, security model, execution foundation and infrastructure architecture.
5. Treat organizations, departments and client businesses as first-class multi-tenant use cases.
6. Treat business automations as first-class domain entities, not merely applications or coding tasks.
7. Preserve the independent classification dimensions:
   - Execution category: A, B, C or D.
   - Compute placement: Sprite, Machine or Hybrid.
   - Authorization: dry-run, confirm or autonomous.
8. Use the canonical `app_category_guidelines.md` definitions. Do not invent or silently modify category semantics.
9. Neon is the mandatory PostgreSQL provider.
10. Fly Machines and Fly Sprites remain the established compute and execution infrastructure.
11. Go owns application behavior and canonical contracts.
12. Existing Taskfiles declare repeatable workflows.
13. Preserve security, isolation, observability, durable-state and authorization requirements.
14. Do not introduce DBOS, River or another workflow engine as a predetermined decision. Evaluate execution requirements and alternatives through explicit ADRs.
15. Preserve existing accepted B0/B1 work and the current B2 engineering mission.

## 2. Roadmap A — Engineering Studio (B0–B5)

Retain the established milestones:

- B0 — Studio manages one development task.
- B1 — Studio supports its local development loop.
- B2 — Remote development and resident engineering agent.
- B3 — Formal Application Preview.
- B4 — Studio troubleshoots itself.
- B5 — Release management.

### Mandatory UI completeness requirement

The entire developer-facing UI for B0–B5 must be represented in the product, including:

- Navigation and information architecture.
- Major screens and subviews.
- Principal workflows.
- Loading, empty, success, error and permission states.
- Appropriate responsive layouts.
- Consistent design language and reusable components.
- Clear feature maturity indicators.

The UI does not have to be functionally complete for the demonstration.

However, it must present a coherent and sufficiently complete representation of the entire B0–B5 developer experience.

This distinction is mandatory:

**UI completeness does not imply functional completeness.**

Existing functionality must be connected to real implementations and evidence.

Future functionality must be clearly identified as planned, partially available or unavailable.

Never:

- Simulate a real deployment without performing one.
- Fabricate agent execution results.
- Present fixture data as live customer data.
- Enable controls that imply unsupported operations.
- Claim B3, B4 or B5 functionality is complete merely because their UI exists.

Build the UI as the intended long-term product interface, not a disposable mockup.

### Screen inventory

Produce a complete UI coverage matrix for every B0–B5 milestone.

At minimum, evaluate:

**B0 — Task Management**

- Development task list.
- Task creation and details.
- Task status and ownership.
- Execution history.

**B1 — Local Development**

- Project/workspace view.
- Local environment status.
- Development configuration.
- Database and migration status.
- Test results and diagnostics.

**B2 — Remote Engineering**

- Resident agent workspace.
- Sprite lifecycle.
- Agent sessions and execution.
- Genuine connectors.
- Authorization and approvals.
- Logs and evidence.
- Checkpoint and recovery.
- Workspace Preview.

**B3 — Application Preview**

- Application inventory.
- PR/Application Preview environments.
- Deployment details.
- Preview URLs.
- Database and environment association.
- Health and smoke-test results.
- Environment isolation.

**B4 — Self-Troubleshooting**

- Incident overview.
- Diagnostic findings.
- Root-cause investigation.
- Proposed corrective actions.
- Approval and recovery workflows.
- Historical evidence.

**B5 — Release Management**

- Application and artifact versions.
- Release history.
- Staging.
- Production.
- Promotion approvals.
- Deployment health.
- Rollback and recovery.
- Release audit.

Do not duplicate Workspace Preview, Engineering PR Preview and formal Application Preview as if they were the same capability.

Preserve their distinct responsibilities and lifecycles.

## 3. Roadmap B — Automation Operations Studio

Design a separate UI experience for users who operate and govern business automations rather than develop Studio itself.

Its core users include:

- Operations personnel.
- QA engineers.
- Department managers.
- Business process owners.
- Agency administrators.
- Client administrators.
- Organization administrators.

The Operations Studio must not be a simplified developer dashboard.

Design its information architecture around operating a portfolio of business automations.

Evaluate these principal areas:

1. Organization and tenant overview.
2. Automation catalog and discovery.
3. Business process and workflow visualization.
4. Automation details, versions and ownership.
5. Scheduling and trigger management.
6. Execution monitoring and run history.
7. QA, test results and release readiness.
8. Approval and exception queues.
9. SLA and operational health.
10. Connector and integration management.
11. Environment and release management.
12. Audit and governance.
13. Cost, usage and capacity.
14. Tenant and department/client administration.

Define role-specific views where genuinely required.

For example, a department manager should see pending approvals and departmental automation health without needing developer-level access to source code or unrestricted integration credentials.

Design the Roadmap B UI now, but implement it incrementally as its domain and runtime capabilities become available.

## 4. Common platform foundation — F0

This is the highest-priority architectural requirement.

Roadmaps A and B must share one foundational architecture.

Produce a complete domain model and responsibility matrix for the following.

### 4.1 Identity and hierarchical tenancy

Explicitly model:

- Organization/account.
- Tenant/security scope.
- Department.
- Client business.
- Workspace.
- User and agent identity.
- Membership.
- Role and permission.
- Delegated access.
- Ownership and resource boundaries.

Support both:

**Company model:** One company with multiple departments.

**Agency model:** One agency managing multiple independently governed clients.

Do not assume that all clients are child departments of the agency. Define explicit cross-organization delegation where appropriate.

Determine which entities constitute security, data ownership, billing and administrative boundaries.

A workspace is not automatically a tenant.

A Sprite or Fly application is not automatically a tenant.

Define how tenant identity propagates into every automation, workflow, database record, connector, credential, execution, log, approval and audit event.

Preserve Neon RLS and fail-closed tenant isolation.

Do not introduce a generic RBAC framework unless justified by actual requirements. Define the necessary permission and delegation contracts first.

### 4.2 Automation domain model

Define the canonical Automation domain entity.

It must distinguish:

- Business automation.
- Application or deployable artifact.
- Workflow.
- Activity or executable step.
- Trigger.
- External connection.
- Automation version.
- Execution run.
- Deployment environment.

An Automation may contain multiple workflows, activities, connectors and deployable components.

Do not equate one business automation with one Miniapp, one Fly app or one Sprite.

Define ownership, identity, versioning, classification, dependencies and lifecycle.

### 4.3 Category and execution contracts

Use the canonical Class A–D definitions from the repository.

Define how each category affects:

- Execution placement.
- Isolation.
- Permissions.
- External-system access.
- Timeouts and resource budgets.
- Retry and failure behavior.
- Observability.
- Testing and verification.
- Human authorization requirements.

Keep category, compute placement and authorization independent.

### 4.4 Shared execution and workflow foundation

Evaluate a reusable Go automation framework supporting:

- Typed inputs and outputs.
- Reusable activities.
- Workflow composition.
- Dependency management.
- Trigger dispatch.
- Execution scheduling.
- Durable run state.
- Idempotency.
- Bounded retries.
- Cancellation.
- Human approval and resume.
- Compensation.
- Failure reconciliation.
- Execution audit and evidence.

Distinguish engineering-agent execution from customer business-automation execution.

Do not assume their lifecycles or resource requirements are identical.

Evaluate DBOS, River, PostgreSQL-backed orchestration and other alternatives only against concrete requirements.

### 4.5 Shared integrations

Define a common connector and capability architecture.

Support tenant-scoped external connections, capability discovery, permission controls, credential isolation, audit and provider-specific adapters.

The business automation runtime and developer engineering workspace must reuse the appropriate connector foundations without sharing privileged credentials.

### 4.6 Environment and release contracts

Define common contracts for:

- Development.
- Workspace Preview.
- Engineering PR Preview.
- Formal Application Preview.
- Staging.
- Production.

Distinguish Studio's own software delivery from customer automation delivery.

Support immutable, versioned artifacts and explicit environment-specific configuration.

The long-term target is to verify an artifact once and promote the same approved version through staging and production.

No preview environment may mutate production data or reuse production secrets without an explicitly defined and approved operational requirement.

### 4.7 Shared persistence and observability

Neon remains mandatory for PostgreSQL.

Define tenant-aware persistence, lifecycle records, execution history, events, logs, metrics, audit, evidence and resource ownership.

Preserve the reliability improvements already established in prompt30, especially bounded database execution, migration correctness, connection lifecycle and explicit failure handling.

## 5. UI architecture — two experiences, one platform

Define the relationship between the two UI experiences.

Roadmap A and Roadmap B must have:

- Separate information architectures.
- Persona-appropriate navigation.
- Distinct primary workflows.
- Shared visual design system.
- Shared identity and tenant context.
- Shared API and authorization contracts.
- Consistent terminology and state representation.
- Shared audit and evidence services where appropriate.

Do not force the Operations Studio into the conversation-oriented engineering workspace.

Do not duplicate the backend because the UIs serve different users.

Leave deployment topology (one frontend with separate workspaces versus distinct frontend applications) to an evidence-backed ADR.

## 6. Unified roadmap

Produce two coordinated roadmaps.

### Roadmap A — B0–B5

Preserve all existing milestone names, accepted status and dependencies.

Add the complete UI coverage requirement and distinguish design, UI implementation, backend implementation, functional acceptance and live verification.

B0/B1 remain accepted unless evidence reveals a material defect.

B2 remains the active engineering milestone.

B3–B5 require complete UI representation but their unimplemented functionality must remain honestly labeled.

### Roadmap B — R0–R8

Use the following as the initial proposed sequence:

- R0 — Shared product and RPA domain architecture.
- R1 — Hierarchical multi-tenancy and tenant lifecycle.
- R2 — Automation catalog, definitions and versioning.
- R3 — Scheduling, orchestration and execution runtime.
- R4 — Operations, QA and management workspace.
- R5 — Per-automation preview, staging, promotion and rollback.
- R6 — Tenant onboarding and administration.
- R7 — Metering, quotas and billing.
- R8 — Enterprise-scale automation operations.

Reconcile dependencies carefully.

Do not implement a single-tenant RPA catalog as an architectural shortcut before the multi-tenant ownership model is established.

Billing may remain deferred, but the domain model should not prevent future usage attribution.

### Cross-roadmap dependencies

Make explicit:

- Which B milestones establish capabilities R-series work depends on.
- Which common contracts must exist before both roadmaps proceed.
- Which design work can proceed in parallel.
- Which implementation work must remain sequential.
- Where one roadmap consumes another's services or artifacts.

The roadmaps must not duplicate execution engines, connector frameworks, release managers or authorization systems.

## 7. UI demonstration and functional honesty

Create a B0–B5 UI completion matrix with the following independent statuses:

- UI designed.
- UI implemented.
- Connected to actual functionality.
- Integration verified.
- Functionality accepted.

A screen can be complete while its operations remain planned.

Define how this is presented in the demonstration so judges can understand the intended complete product without mistaking visual coverage for operational readiness.

Prioritize a coherent, navigable developer experience.

Do not delay the functioning B2 resident-agent and GitHub Connector demonstration to implement the full future RPA platform.

## 8. Architecture decisions and documentation

Before proposing implementation of the missing R-series capabilities, inspect:

- Existing ADRs.
- `app_category_guidelines.md`.
- `roadmap4.md`.
- `writeup2.md`.
- Current B0–B5 plans and evidence.
- Existing Studio UI and design conventions.
- Neon, Fly, connector and authorization contracts.
- Prompt27.1 through prompt30.

Produce:

1. Unified product architecture.
2. Roadmap A and Roadmap B specifications.
3. Common foundation architecture.
4. Organization/tenant/delegation domain model.
5. Automation domain model.
6. Class A–D capability and execution contract.
7. Shared Go runtime architecture and alternatives.
8. Two UI information architectures.
9. B0–B5 complete UI coverage matrix.
10. R0–R8 requirement and dependency matrix.
11. Environment and artifact promotion model.
12. Security and threat model.
13. Dependency-ordered implementation roadmap.
14. ADR backlog with unresolved decisions.
15. Proposed implementation Task Cards.

For every capability distinguish:

- IMPLEMENTED — verified in actual repository.
- ACCEPTED — formally accepted with evidence.
- DESIGNED — architecture documented.
- IN PROGRESS — actual implementation underway.
- PLANNED — not implemented.
- DEFERRED — deliberately postponed.
- UNVERIFIED — requires additional evidence.

## 9. Implementation governance

This assignment produces the authoritative architecture and roadmap, not immediate platform implementation.

Do not:

- Modify accepted B0–B5 behavior without evidence.
- Reopen prompt30 work unnecessarily.
- Introduce self-hosted PostgreSQL.
- Change Neon-first infrastructure decisions.
- Build a second automation runtime.
- Introduce a separate authorization system for Roadmap B.
- Implement DBOS/River merely because an RPA framework needs durable execution.
- Create misleading UI interactions or simulated success states.
- Delay the B2 demonstration for speculative future platform features.

The first implementation priorities following approval should be selected from actual architectural gaps, not from the desire to implement the entire roadmap simultaneously.

## 10. Final deliverable

Create or reconcile one authoritative Prompt31 product architecture and roadmap document.

It must answer:

1. What is Miniapp Studio?
2. Who uses it?
3. Why are there two UI experiences?
4. What do both experiences share?
5. How do organizations, departments and clients coexist securely?
6. What exactly is an Automation?
7. How do Class A–D automations differ operationally?
8. How are automations authored, executed, tested and recovered?
9. How are they previewed, staged, promoted and rolled back?
10. How do Roadmaps A and B depend on each other?
11. What is already real versus merely planned?
12. What must be built next and why?

Provide a prioritized, dependency-aware implementation roadmap with acceptance criteria.

Do not equate a complete UI with a complete platform.

**Governing product principle: One shared engineering and automation foundation, two specialized user experiences, and a consistent software engineering methodology from development through production.**

## **9. My final architectural verdict**

I would lock these four decisions now:

**Decision 1 — Two UI experiences**

Developers get Roadmap A. Operations, QA and management get Roadmap B. Both share a design system and backend contracts, not identical navigation or interaction models.

**Decision 2 — Complete Roadmap A UI coverage**

B0–B5 must all be represented in the developer UI. Functional maturity must be independently and honestly indicated.

**Decision 3 — Common foundation before RPA expansion**

Tenancy, Automation contracts, authorization, execution, connectors, persistence and environment lifecycles must be shared.

**Decision 4 — Protect B2 and avoid roadmap sprawl**

Finish the immediate engineering demonstration while designing the long-term RPA platform. Do not let the R-series derail the current critical path.

One final point: I would not require Roadmap A to finish every backend feature from B0–B5 before Roadmap B architecture begins. The shared foundation and R0 design should be settled early, while B2 continues. This avoids creating the very schema and permission problems we're trying to prevent.

**The ultimate product should feel like one platform to the organization, two purpose-built workspaces to its users, and one consistent engineering system to the developers who build and maintain its automations.**

That's the direction I recommend locking for Miniapp Studio.