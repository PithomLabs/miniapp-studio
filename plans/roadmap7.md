You're building Miniapp Studio into a multi-tenant, API-centric RPA platform: a business can automate its operations through reusable Go-based Miniapps, while Studio provides the engineering, governance, and operational control plane.

There are three connected workstreams. Here's where you are and where you're heading.
1. Current work: F0 shared foundations
F0 provides the common services that both Studio experiences and the RPA runtime depend on.
Increment	Scope	Status
1	Unified secrets management	Done
2	Tenant credential broker, encrypted storage, short-lived leases	Done — Neon verified
3	Tenant invitations and capability refresh	In progress

Importantly, approvals, effects, and intent-hash mechanisms already exist. We reuse them rather than build duplicates.
2. What comes after Increment 3?
The immediate objective is to complete the minimum RPA primitives and deliver an actual working business workflow.

The Mom-n-Pop plan identifies this sequence:
1. Receipts and evidence: record decisions, execution outcomes, and external effects.
2. Contact management: create/update contacts with reliable deduplication.
3. Event intake: receive or poll inbound messages, track watermarks, prevent duplicate processing.
4. Scheduling: recurring jobs, overlap prevention, and execution provenance.
5. Business capabilities: estimates, messaging, invoices, payments, and provider integrations.
Then comes the first major end-to-end demonstration.
W1 — Lead-to-Payment
1. Customer sends an inquiry.
2. Platform captures and qualifies the lead.
3. Creates an estimate.
4. Owner reviews and approves it.
5. Sends the estimate to the customer.
6. Issues an invoice and collects payment.
7. Requests a review and records execution evidence.
Target: first complete vertical slice

After W1, the plan expands to W2: missed-call-to-booked-job, then W3: campaign-to-cash.
3. The larger product roadmap
There are two separate Studio experiences sharing one backend.
Roadmap A — Engineering Studio

The developer experience: building, testing, deploying, and maintaining Miniapps.
Milestone	Purpose
B0	Manage development tasks
B1	Local development loop
B2	Remote Builder Sprite and agent handoff
B3	Per-PR Application Preview
B4	Logs and troubleshooting
B5	Staging and production releases


Roadmap B — Automation Operations Studio

The business/operator experience: managing automations and their outcomes.
Phase	Purpose
R0	RPA domain architecture
R1	Hierarchical multi-tenancy
R2	Automation catalog and versioning
R3	Scheduling and execution runtime
R4	Operations, QA, and management workspace
R5	Preview, staging, promotion, rollback
R6	Tenant onboarding and administration
R7	Metering, quotas, billing
R8	Enterprise scale and governance


These tracks are related but must not block each other. Shared infrastructure belongs in F0; the Engineering Studio must continue progressing without waiting for the Operations Studio.
4. Architectural guardrails
A few decisions remain especially important:
- Neon PostgreSQL everywhere; no alternative database backend.
- Fly Sprites for agent-generated/untrusted execution; Fly Machines for trusted application runtimes.
- Class A/B/C/D execution categories, using the simplest sufficient execution model.
- Class D durable workflows require ADR-001 evidence; DBOS is not automatically adopted.
- Human approval for consequential external actions, especially sending messages and handling money.
- No classic UI-RPA or screen scraping in the current scope; API-centric automation comes first.

Where I would focus
Don't build a giant automation platform before proving one useful automation.

Finish Increment 3, close any remaining F0 gaps, and concentrate on W1 lead-to-payment. That gives you a concrete business outcome against which to validate the broker, capabilities, approvals, effects, integrations, and tenant isolation.

The larger roadmap is here:
- Miniapp Studio — Ultimate Roadmap
- Mom-n-Pop RPA Platform Plan
The destination: a business owner describes a workflow in plain language, an AI agent helps construct it, the platform governs its execution, and the owner manages the entire operation through Studio.



## prompt

just to clarify about the roadmap and the target first vertical slice below, how exactly do you plan to do that using the primitives and abstractions of Miniapp Studio? The goal is to make the Studio a robust API-first RPA platform without the bloat by making it process-agnostic. How do declarative agent skills map to Studio? Give me the grand vision and high level plan only

The immediate objective is to complete the minimum RPA primitives and deliver an actual working business workflow.

The Mom-n-Pop plan identifies this sequence:
1. Receipts and evidence: record decisions, execution outcomes, and external effects.
2. Contact management: create/update contacts with reliable deduplication.
3. Event intake: receive or poll inbound messages, track watermarks, prevent duplicate processing.
4. Scheduling: recurring jobs, overlap prevention, and execution provenance.
5. Business capabilities: estimates, messaging, invoices, payments, and provider integrations.
Then comes the first major end-to-end demonstration.
W1 — Lead-to-Payment
1. Customer sends an inquiry.
2. Platform captures and qualifies the lead.
3. Creates an estimate.
4. Owner reviews and approves it.
5. Sends the estimate to the customer.
6. Issues an invoice and collects payment.
7. Requests a review and records execution evidence.


## answer

# **The grand vision: Miniapp Studio as a process-agnostic RPA platform**

My central architectural recommendation is this:

**Miniapp Studio should not become a collection of business-specific automations. It should become a platform that allows AI agents to assemble, execute, and govern business processes from reusable capabilities.**

The distinction is important.

A CRM, accounting system, email provider, payment processor, or project-management system is merely an external system that provides capabilities. A business process is a composition of those capabilities. Miniapp Studio provides the contracts, execution machinery, security, and governance that make those compositions reliable.

The Mom-n-Pop Skills repository gives us examples of useful business processes. It should not dictate the architecture of the platform.

## **1. How declarative agent skills map to Miniapp Studio**

A declarative skill is essentially a description of how to accomplish a task. It contains instructions, business rules, required integrations, safety constraints, and expected outcomes.

In Studio, I would translate those elements into existing Miniapp abstractions rather than create a separate skill execution framework.

## **Declarative Skill → Miniapp → Automation**

**Agent Skill**

Business instructions, process knowledge, rules, checklists

Translate into explicit, testable contracts

**Miniapp Definition**

- MiniappSpec and manifest
- Typed, versioned capabilities
- Input/output schemas
- Trust and approval requirements
- Idempotency and retry contracts
- Verification and capability attestations

Compose capabilities into a business process

**Automation**

A versioned business-process definition with triggers, stages, policies, and recovery behavior

**Miniapp Studio Runtime**

Admission, authorization, execution, approvals, effects, receipts, and recovery

A skill's different elements map to different platform responsibilities:

| Skill content                   | Studio responsibility                   |
| ------------------------------- | --------------------------------------- |
| Process instructions            | Agent-facing instructions               |
| Business rules                  | Pure Go domain logic                    |
| Actions                         | Typed capabilities                      |
| Provider-specific scripts       | Connector adapters                      |
| Safety checklists               | Admission tests and policies            |
| Human approval instructions     | Framework approval mechanism            |
| Retry and recovery instructions | Execution contracts and effect handling |
| Operational guidance            | Runbooks and evidence                   |

This means a skill does not automatically become one Miniapp or one executable workflow.

Some skills contribute domain logic. Some contribute provider adapters. Some become reusable process definitions. Others remain agent guidance or operational documentation.

**The skill is knowledge; the Miniapp is an executable, testable implementation of that knowledge.**

## **2. The platform should have only a few fundamental layers**

I would organize the architecture around five responsibilities.

**1. Agent interaction layer**

Agents such as Claude, Codex, OpenCode, or other compatible agents interpret requests, gather information, propose actions, and explain outcomes.

They do not own authorization, credentials, or execution state.

**2. Automation and Miniapp layer**

Contains versioned business-process definitions and reusable Miniapps.

Each declares what it does, what capabilities it needs, what inputs it accepts, how it handles failures, and where human authorization is required.

**3. RPA framework layer**

Owns the difficult operational machinery once:

- Execution categories A/B/C/D.
- Admission and policy enforcement.
- Idempotency and keyed serialization.
- Approval lifecycle.
- Effect ledger and reconciliation.
- Scheduling and durable execution where justified.
- Tenant isolation and audit receipts.

**4. Domain and connector layer**

Domain capabilities represent business operations. Connector adapters implement them against real providers.

Business logic does not know whether the customer uses Salesforce, HubSpot, GoHighLevel, QuickBooks, or another provider.

**5. Shared platform foundation (F0)**

Identity, tenancy, authorization, secrets, credential brokerage, persistence, evidence, and common infrastructure.

Shared once by the framework and both Studio experiences.

The domain model should remain deliberately small:

| Domain     | Responsibility                          |
| ---------- | --------------------------------------- |
| Party      | Contacts, organizations, consent        |
| Commercial | Estimates, invoices, payments           |
| Engagement | Messages, campaigns, reviews            |
| Workforce  | Jobs, schedules, field evidence         |
| Ledger     | Receipts, decisions, execution evidence |

The important anti-bloat rule: domains own their records; workflows compose them. We do not create another giant CRM that attempts to own every business object.

## **3. How the first vertical slice actually works**

Let's use the Mom-n-Pop **Lead-to-Payment** process.

The owner should be able to configure an automation without knowing which CRM or email provider it uses.

## **Lead-to-Payment**

One business process, several independently reusable capabilities

1. **Capture customer inquiry**

   Email, webform, or voice integration receives an event.

   Engagement / Intake
2. **Qualify and identify the customer**

   Agent interprets the inquiry and calls contact capabilities. Existing customer, new lead, duplicate, and junk are handled explicitly.

   Party / Contact
3. **Prepare an estimate**

   Deterministic pricing kernel calculates line items, assumptions, and deposit.

   Commercial / Estimate
4. **Owner reviews and approves**

   Studio presents the exact proposed estimate. Approval is bound to its material fingerprint.

   Human authorization
5. **Send estimate and capture customer acceptance**

   Messaging connector sends the approved proposal. Customer acceptance is captured as a separate, evidenced event.
6. **Schedule and manage the job**

   Calendar and workforce integrations coordinate the appointment and job records.

   Workforce
7. **Invoice and collect payment**

   Invoice issuance and payment collection use brokered credentials, effect tracking, and the required owner approvals.

   Financial authorization
8. **Request a review and report results**

   Follow-up is sent under the appropriate policy, with final receipts and operational reporting.

   Engagement / Ledger

Behind this apparent simplicity, each operation uses the same platform machinery.

| Platform primitive                 | Role in Lead-to-Payment               |
| ---------------------------------- | ------------------------------------- |
| P1 Contact upsert                  | Customer identity                     |
| P2 Estimate create                 | Deterministic pricing                 |
| P3 Invoice issue                   | Invoice creation and delivery         |
| P4 Payment collect                 | Payment operations                    |
| P6 Message send                    | Customer communications               |
| P7 Intake poll                     | Inbound customer events               |
| P8 Review request                  | Post-job follow-up                    |
| P9 Job sync                        | External job-management integration   |
| P11 Receipt append                 | Evidence across every stage           |
| P13 Credential lease               | Secure provider access                |
| P14 Cron management                | Scheduled work and overlap protection |
| Existing approval/effect machinery | Authorization, idempotency, recovery  |

Calendar appointment booking also needs an explicit calendar capability and adapter contract; we should not pretend that cron scheduling or job synchronization already provides that functionality.

The agent does not directly call Stripe or QuickBooks with ambient credentials. It invokes declared capabilities. The framework handles authorization, broker leases, idempotency, approvals, and external effects.

One important distinction: **a workflow is not automatically a Class D workflow.** Each tool receives its own A/B/C/D classification. If the business promise requires durable waits over days and human decisions across restarts, we evaluate Class D through ADR-001 rather than hand-building a fragile workflow engine.

## **4. How we avoid provider lock-in**

Consider a business using HubSpot and QuickBooks today, then switching to Salesforce and another accounting provider tomorrow.

The business process should not have to be rewritten.

**Business capability**

`party.contact.upsert`

**Provider A**

HubSpot adapter

**Provider B**

Salesforce adapter

**Provider C**

GoHighLevel adapter

**Provider D**

Custom REST adapter

Studio discovers actual provider capabilities, differences, connection status, and authorization. It does not assume every provider implements every operation.

The connector contract must describe what a provider supports and what it does not. The capability inventory reports verified availability, but never grants execution permission.

This is also where the existing credential broker becomes important: each tenant supplies its own credentials, and execution receives narrowly scoped leases instead of exposing those credentials to agents.

## **5. What happens to the two Studio experiences?**

They serve two different audiences.

[Angy: Multi‑agent pipelines w/ AI‑driven scheduling + safety check | Product Hunt](https://images.openai.com/static-rsc-4/AS90UQcdtqs6G-C7lM1Gl_I8rOa2nDmEJTGvlJAa8kdttdeDHDZxfTp6b5pFN_ujDSxkrArcM6MQvPqo_XDskmxpXHAAfHiVqsxvt3lN64w0AhepVkHWTx2iDMjSYCZOHvy73YnzehUfqbYsMJK_Z9T3AUG9xfSifaJOUG6Ak0E?purpose=inline)

**Engineering Studio**

For developers and coding agents.

Build Miniapps, validate contracts, manage versions, run tests, deploy previews, diagnose failures, and release software.

[Yeeflow: A Complete Workflow & Business Process Automation Platform](https://images.openai.com/static-rsc-4/T3kPJnb5dCTcVRJ_sw41OzCRbKyDiYGihAwPSz0_rdmTm8mMSev4B8IAcmB3RZpR6hfvxJIyRerku6KfZrzGJNJSroPxOpI1Oli740RDZduH1KmQqktlEd0lZPOmTad41K2rLYenxI4V5JDDBXml-6MbLUPWB_uyxRFN1ozT00Y?purpose=inline)

**Automation Operations Studio**

For business owners, operators, and managers.

Discover automations, configure processes, monitor executions, resolve exceptions, approve actions, and inspect business outcomes.

They share identity, authorization, connectors, runtime, records, and audit services. They must not become two competing backends.

## **6. The high-level roadmap I recommend**

I would sequence the work as follows.

1. **Secure and complete F0**

   Finish Prompt36 remediation, the corrective migrations, regression testing, adversarial review, and owner-side Neon verification. Then finish the remaining shared receipts and attestation foundations.
2. **R0 — Freeze the Automation contract**

   Define what an Automation is, how it relates to Miniapps, how capabilities are composed, and how process versions, triggers, policies, and outcomes are represented.

   This is where declarative agent skills become first-class authoring inputs.
3. **R1–R2 — Tenancy and automation catalog**

   Organizations, tenants, delegation, lifecycle, quotas, automation definitions, capability requirements, and versioning.
4. **R3 — Execution runtime**

   Triggers, scheduling, staged execution, retry, idempotency, approvals, reconciliation, and durable workflows only when justified.
5. **First vertical slice: Lead-to-Payment**

   Build the necessary adapters and capabilities, then demonstrate the entire workflow using a real tenant and controlled provider sandboxes before enabling live financial operations.
6. **R4–R6 — Operations workspace and onboarding**

   Deliver the business-facing workspace, tenant administration, connector management, automation configuration, run monitoring, and exception handling.
7. **R7–R8 — Metering and enterprise readiness**

   Quotas, usage, billing infrastructure, scale, resilience, and governance.

Roadmap A's B0–B5 Engineering Studio work continues in parallel; the R-series must not block it.

## **7. The most important architectural decision**

I would resist building a visual workflow canvas, a massive collection of built-in business applications, a generic plugin marketplace, or a new proprietary agent runtime.

Instead, make the platform revolve around four concepts:

**Automation** — what business process needs to happen.

**Miniapp** — the versioned executable implementation.

**Capability** — the smallest typed, authorized business operation an agent can invoke.

**Connector** — the replaceable implementation that connects a capability to an external provider.

Everything else—execution, credentials, security, approvals, evidence, and recovery—is platform infrastructure.

The business owner describes the desired outcome. An AI coding agent helps translate the process into declarative definitions and Miniapps. Studio validates that the declared capabilities are actually implemented, tested, and authorized. At runtime, agents work within those contracts, while the framework guarantees the operational and security invariants.

**That is the grand vision: not an RPA product with hundreds of predefined automations, but an extensible RPA platform where new business processes can be assembled from small, reusable, provider-independent capabilities without rebuilding the machinery every time.**

The two architectural references that should govern this direction are the [Ultimate Roadmap](https://github.com/PithomLabs/miniapp-studio/blob/main/plans/miniapp-studio-ultimate-roadmap.md?utm_source=chatgpt.com) and the [Mom-n-Pop Skills → Miniapp Studio plan](https://github.com/PithomLabs/miniapp-studio/blob/main/plans/mom-n-pop-rpa-platform-plan.md?utm_source=chatgpt.com).


## prompt

I have objections to this platform primitives, invoice issue, payment collect, intake poll etc are clearly business processes, I do not want to bloat the platform to identify these platform primitives because they are literally thousands. The goal is to provide small layer of abstraction of business processes that we can call the platform primitives. Other than that, your vision is fine.

P1 Contact upsert

Customer identity

P2 Estimate create

Deterministic pricing

P3 Invoice issue

Invoice creation and delivery

P4 Payment collect

Payment operations

P6 Message send

Customer communications

P7 Intake poll

Inbound customer events

P8 Review request

Post-job follow-up

P9 Job sync

External job-management integration

P11 Receipt append

Evidence across every stage

P13 Credential lease

Secure provider access

P14 Cron management

Scheduled work and overlap protection

Existing approval/effect machinery

Authorization, idempotency, recovery

## answer

I agree. My previous proposal made a fundamental categorization mistake: **I confused reusable platform abstractions with business capabilities that happen to be used in the first vertical slice.**

That would gradually turn Miniapp Studio into a giant collection of predefined business operations. That's exactly what we want to avoid.

The correct distinction is:

## **1. Platform primitives versus business capabilities**

**Platform primitives are few, generic, and process-agnostic.**

They provide the machinery to describe and execute arbitrary business processes.

**Business capabilities are unlimited and domain-specific.**

They are implemented by Miniapps, exposed through connectors, or composed by agents. Studio should not need to understand their business semantics.

| Platform abstraction | Purpose                                                              |
| -------------------- | -------------------------------------------------------------------- |
| Process              | Describe a business process declaratively                            |
| Step                 | Invoke a capability through a typed contract                         |
| Trigger              | Start a process from an event, schedule, or explicit request         |
| Control flow         | Sequence, branch, repeat, wait, and transition between steps         |
| Gate                 | Apply policy and require human authorization                         |
| Effect               | Guarantee idempotency, retries, outcome tracking, and reconciliation |
| Evidence             | Record execution history, decisions, and results                     |

These are conceptual abstractions, not necessarily seven separate packages or database entities. We should reuse the existing Miniapp framework wherever possible.

Identity, tenancy, credential brokerage, and secrets management are foundational platform services, rather than business-process primitives.

## **2. Where business capabilities belong**

Consider your examples:

| Business capability | Where it belongs                    |
| ------------------- | ----------------------------------- |
| Contact upsert      | Business Miniapp                    |
| Estimate creation   | Business Miniapp                    |
| Invoice issuance    | Business Miniapp / provider adapter |
| Payment collection  | Business Miniapp / provider adapter |
| Email intake        | Connector / Miniapp                 |
| Review requests     | Business Miniapp                    |
| Job synchronization | Business Miniapp / provider adapter |
| Marketing campaigns | Business Miniapp                    |
| Financial analysis  | Business Miniapp                    |

We might eventually have thousands of these. That is perfectly fine.

The platform should not need a new core abstraction every time somebody invents another business capability.

A Miniapp declares a capability through the existing contract:

- Identity and version.
- Input/output schema.
- Authorization and trust requirements.
- Idempotency and retry policy.
- Execution category.

The framework knows how to invoke and govern it. It does not need to know what an invoice, customer, campaign, or construction job means.

## **3. How declarative skills fit**

This is where the architecture becomes particularly powerful.

A declarative agent skill describes *how to accomplish something*. It can become an authoring input for a process definition.

For example:

**Lead-to-Payment skill**

Declarative business knowledge, instructions, conditions, and policies

**Automation definition**

Composes capability references, triggers, decisions, transitions, and approval requirements

**Miniapp capabilities**

Concrete implementations supplied by Miniapps and provider connectors

**Miniapp Studio runtime**

Executes the process with authorization, reliability, evidence, and recovery

The agent can help create and modify these definitions. Studio validates them against registered capabilities and platform policies.

A skill is not necessarily executable code, and it does not automatically become a Miniapp. It is reusable knowledge that helps agents construct processes using available capabilities.

## **4. The corrected grand vision**

I would organize the product around just four central concepts:

1. **Skills** — reusable business knowledge and instructions.
2. **Automations** — declarative definitions of business processes.
3. **Miniapps** — executable implementations of capabilities.
4. **Platform runtime** — generic machinery that governs and executes everything.

Connectors provide replaceable integration implementations. Studio provides development, administration, observability, and governance.

The Lead-to-Payment workflow becomes our first proof that this architecture works. It is an example Automation, not a collection of primitives that the platform must permanently support.

**The architectural principle I want to preserve is:**

> Miniapp Studio should understand how to execute a business process, not understand every business process that exists.

That is how we build a genuinely process-agnostic RPA platform without accumulating thousands of domain-specific abstractions.


## prompt

I want to add webhooks to platform abstractions, give prompt for other coding agents to delineate the grand vision above for adversarial review, make it thorough and detailed as I want the specs to be described clearly on and clear while we are still at the design stage


## answer

I would have the agents produce a **platform architecture specification, not an implementation plan**. The objective is to settle the conceptual model, boundaries, contracts, and responsibilities before committing to schemas or code.

I've explicitly made inbound webhooks a first-class platform abstraction, while requiring the reviewers to distinguish webhook ingress from internal events, polling, and outbound webhook delivery.

# Prompt 37 — Miniapp Studio: Grand Architecture Vision and Adversarial Design Review

## Mission

Develop a comprehensive, technology-grounded architecture specification for the future of Miniapp Studio as a **process-agnostic, API-first, multi-tenant RPA platform**.

This is a design-stage assignment. Do not implement code, migrations, APIs, database schemas, provider integrations, or UI features.

The objective is to clearly articulate what the platform is, what it deliberately is not, what abstractions it provides, how declarative agent skills map into it, and how arbitrary business processes can be composed and executed without continually expanding the platform's core.

The specification must be detailed enough that separate engineering teams or coding agents can independently understand the architecture and derive consistent implementation plans without inventing their own interpretations.

Produce a rigorous design document, then subject it to an independent adversarial architecture review.

## 1. Governing architectural objective

Miniapp Studio is intended to become a general-purpose API-first RPA platform.

A business owner should be able to describe a business process in natural language. An AI agent should be able to interpret that process, discover available capabilities, construct or modify a declarative Automation, and use Miniapps to execute the necessary operations.

Studio provides the infrastructure to validate, authorize, execute, observe, and recover these processes.

The platform must be:

- **Process-agnostic:** it must not need to understand the business meaning of every workflow.
- **Capability-driven:** business operations are exposed through typed, versioned capability contracts.
- **API-first:** external systems are integrated through APIs, webhooks, events, and adapters; classic UI-RPA is deferred.
- **Agent-native:** coding and operational agents are first-class actors, but cannot bypass platform governance.
- **Composable:** new processes combine existing capabilities without requiring platform changes.
- **Provider-independent:** implementations can be replaced without rewriting business processes, subject to declared capability differences.
- **Multi-tenant:** organizations, departments, agencies, clients, and workspaces are isolated through platform-enforced identity, authorization, and data boundaries.
- **Minimal at the core:** the platform provides a small number of generic abstractions, not a catalog of predefined business operations.

The architectural principle to preserve throughout the document is:

> Miniapp Studio should understand how to execute and govern a business process, not understand every business process that exists.

## 2. Mandatory source investigation

Before proposing architecture, inspect the current repository and its governing documents.

At minimum, examine:

- `plans/miniapp-studio-ultimate-roadmap.md`
- `plans/mom-n-pop-rpa-platform-plan.md`
- `plans/app_category_guidelines.md`
- Existing Miniapp framework contracts, particularly `contract/manifest.go` and `contract/policy.go` (resolve their actual current paths).
- Existing trust, authority, approval, effect, and idempotency contracts.
- Existing connector and provider interfaces.
- Current identity, tenancy, session, authorization, and RLS implementation.
- Current F0 secrets, credential broker, invitation, and capability-refresh implementations.
- Current development, environment, execution, and Taskfile contracts.
- Relevant ADRs, particularly ADR-001 and the existing security decisions.

Read the actual source. Do not rely exclusively on README descriptions or prior architecture summaries.

### Precedence

Apply the established precedence:

1. Explicit Project Owner decisions.
2. Ultimate Roadmap.
3. Approved architectural decisions and ADRs.
4. Approved domain-architecture plans.
5. Task-specific prompts and proposals.

Do not silently alter a locked decision.

Identify conflicts explicitly and recommend resolutions. Distinguish implemented, accepted, designed, proposed, deferred, and unverified capabilities. Never represent a planned feature as implemented.

## 3. First establish the conceptual vocabulary

The architecture must clearly and consistently define the following concepts. Explain their purpose, ownership, boundaries, and relationships.

### 3.1 Skill

A Skill is reusable business knowledge and procedural guidance that an agent can consume.

It can contain:

- Business objectives and instructions.
- Decision rules and qualification logic.
- Process knowledge and sequencing guidance.
- Examples and reference materials.
- Operational procedures.
- Required capability references.
- Safety constraints and verification checklists.

A Skill is not automatically executable code, an Automation, or a Miniapp.

It does not independently grant credentials, permission, or execution authority.

Explain how Skills are authored, discovered, versioned, referenced by agents, and translated into validated Automation definitions.

Do not create a proprietary skill runtime unnecessarily. Evaluate how existing declarative agent-skill formats can be consumed through the agent interaction layer.

### 3.2 Automation

An Automation represents a business process the owner wants performed.

It is the central business-facing concept of the RPA platform.

Define its conceptual responsibilities:

- Business objective and expected outcome.
- Version and lifecycle.
- Trigger declarations.
- Process steps and capability references.
- Input/output mappings.
- Conditional decisions and control flow.
- Human authorization requirements.
- Trust and execution policies.
- Error and recovery behavior.
- Execution history and evidence.
- Tenant ownership and environment association.

An Automation is a declarative composition, not a hardcoded domain-specific program.

Determine the minimum information required to describe an Automation without prematurely freezing a schema.

Explain how Automations are validated, versioned, activated, suspended, tested, promoted, and retired.

### 3.3 Miniapp

A Miniapp is a versioned executable application implemented within the Miniapp framework.

It exposes one or more typed capabilities through declared contracts and tested handlers.

It contains domain logic and implementation details, but should not need to understand the entire business process that calls it.

A Miniapp may provide several related capabilities. An Automation may use multiple Miniapps. Neither relationship should be artificially constrained to one-to-one.

Preserve the existing MiniappSpec, manifest, execution-category, trust, and admission contracts.

### 3.4 Capability

A Capability is a typed, versioned, invokable operation.

It is the contract through which an Automation or agent invokes functionality provided by a Miniapp or connector-backed implementation.

Define:

- Identity and version.
- Input/output contracts.
- Invocation semantics.
- Authorization and trust requirements.
- Idempotency and retry declarations.
- Execution category.
- Provider and implementation provenance.
- Capability availability and attestation.

Capabilities can represent arbitrarily many business operations. The platform must not maintain a hardcoded list of business capabilities.

Existing `CapabilityRef` and Miniapp manifest contracts should be reused and extended only when justified.

### 3.5 Connector

A Connector is the replaceable integration boundary to an external provider or system.

It handles provider-specific authentication, transport, API semantics, rate limits, pagination, webhooks, errors, and capability differences.

Distinguish:

- The provider-neutral capability contract.
- The connector interface.
- The provider-specific adapter.
- The tenant's connection and credentials.
- The actual availability and restrictions of the provider.

A connector must explicitly describe unsupported or restricted capabilities. Never assume that all providers have identical functionality.

### 3.6 Process execution and run

Explain the distinction between:

- Automation definition.
- Automation version.
- Process execution/run.
- Individual step execution.
- Capability invocation.
- External effect.
- Execution receipt.

Clarify which objects are definitions and which represent runtime state.

An Automation must be independently versionable. A running execution must have a well-defined relationship to the version under which it started.

## 4. The small set of platform abstractions

This is the most important architectural section.

Define the smallest sufficient set of generic abstractions required to describe, trigger, govern, and execute arbitrary processes.

The platform should not evolve into a catalog of business-specific primitives.

Evaluate the following conceptual set. It is a starting hypothesis, not an instruction to introduce one package or table per item.

### A. Process / Automation

The declarative definition of what should happen.

It references capabilities and describes process behavior without embedding provider-specific implementation details.

### B. Capability invocation / Step

A typed invocation of an independently implemented capability.

A process step should describe what is invoked, the validated inputs, how outputs are mapped, and how the result affects subsequent process execution.

Do not confuse a step with a business operation. Invoice creation, CRM updates, and campaign sending are examples of capabilities, not built-in platform primitives.

### C. Trigger

A generic mechanism through which an Automation starts or resumes.

Evaluate at least:

- Manual invocation.
- API invocation.
- Internal domain/platform events.
- Scheduled execution.
- Inbound webhook delivery.

Define how a trigger identifies the target Automation/version, binds tenant context, supplies validated input, and prevents unauthorized execution.

### D. Webhooks — first-class platform abstraction

Webhooks must be explicitly included in the architecture.

Treat inbound webhook ingestion as a first-class way to trigger and resume Automations.

Distinguish the following concepts:

1. **Webhook endpoint:** a registered endpoint belonging to a tenant or controlled platform integration.
2. **Webhook delivery:** an individual incoming HTTP request from an external provider.
3. **Webhook verification:** authentication/signature verification and provider-specific trust validation.
4. **Webhook event:** a normalized, validated representation of the incoming event.
5. **Event routing:** mapping the authenticated event to its authorized tenant, subscription, and Automation.
6. **Trigger invocation:** starting or resuming the relevant process.

A webhook is not synonymous with an internal event or an Automation.

Specify the desired contracts for:

- Endpoint registration and lifecycle.
- Provider signature verification.
- Secret storage and rotation.
- Tenant binding derived from trusted registration, never untrusted payload claims.
- Payload size limits and schema validation.
- Replay protection and duplicate delivery.
- Idempotency and delivery identity.
- Acknowledgement behavior.
- Retry semantics and provider redelivery.
- Out-of-order events.
- Event correlation and process resumption.
- Failure handling, quarantine, and observability.
- Environment isolation.
- Security against spoofing, replay, SSRF, and cross-tenant routing.

Clarify how webhook ingress interacts with the existing effect ledger and receipt system.

**Inbound versus outbound webhooks:** evaluate them separately. Inbound webhooks are an explicit platform trigger abstraction. Outbound webhook delivery may be modeled as a capability/effect for notifying external consumers. Explain whether it needs additional generic delivery machinery or can reuse existing effect and connector contracts. Do not automatically expand the initial scope.

Also explain the relationship among webhooks, polling, cron, and internal events. They are different event sources, not competing implementations of one indistinguishable mechanism.

### E. Control flow

Describe the minimum generic process-control semantics.

Evaluate:

- Sequence.
- Conditional branching.
- Bounded iteration.
- Parallel execution, when safe.
- Waiting for an event.
- Timeouts.
- Process completion and failure.
- Retry and compensation boundaries.

Distinguish declarative process control from business logic.

Do not design an unnecessarily powerful workflow programming language or visual workflow canvas.

Explain which semantics can be implemented through the existing Miniapp execution categories and which require additional durable execution machinery.

### F. Gate / Authorization

Describe the reusable authorization and human-approval boundary.

The framework remains responsible for:

- Authority and principal identity.
- Tenant and capability scope.
- Trust mode.
- Authorization policy.
- Human approval where required.
- Material-hash binding.
- Approval expiry and decision receipts.
- Separation between the proposing agent and approving authority.

Approval is not a special business-process capability. It is a generic execution control.

### G. Effect

Describe the framework-owned handling of externally observable side effects.

Reuse existing effect and idempotency machinery.

Cover:

- Intent registration.
- Idempotency identity.
- Attempt lifecycle.
- Bounded retries.
- Confirmed success.
- Terminal failure.
- UNKNOWN outcomes.
- Quarantine.
- Reconciliation.
- Provider-side idempotency and authoritative reads.

A process must never blindly repeat an external effect whose outcome is unknown.

### H. Evidence / Receipt

Describe generic evidence for process executions, steps, decisions, webhook deliveries, capability invocations, and external effects.

Explain the distinction between:

- Operational state.
- Provider-authoritative state.
- Domain records.
- Append-only execution evidence.

Receipts are evidence, not the authoritative current state of a remote business object.

### I. Cross-cutting platform services

Clearly separate platform abstractions from supporting infrastructure.

Include identity, tenancy, authorization, credential brokerage, secrets management, persistence, event intake, scheduling, observability, and environment isolation.

Do not misclassify these as business capabilities.

Explain which existing implementations already satisfy these responsibilities and which remain planned.

## 5. Explicit anti-bloat doctrine

Create a section that makes the following rules enforceable architectural principles.

### Business operations are not platform primitives

The following are business capabilities or process-specific operations, not platform primitives:

- Contact creation or update.
- Customer qualification.
- Estimate creation.
- Invoice issuance.
- Payment collection.
- Email or SMS sending.
- Campaign creation.
- Review requests.
- Job synchronization.
- Appointment booking.
- Financial analysis.
- Payroll operations.
- Inventory management.

These examples are deliberately diverse. There may eventually be thousands of business capabilities.

The platform must never require a new core primitive merely because somebody introduces a new business operation.

For every proposed new platform abstraction, ask:

1. Is this truly generic across unrelated business processes?
2. Can it be represented using existing Automation, capability, trigger, control-flow, gate, effect, and evidence contracts?
3. Would it introduce domain-specific business semantics into the platform core?
4. Does it solve an actual cross-process reliability or security problem?
5. Can its absence be demonstrated as a concrete architectural limitation?

Prefer composition over additional abstractions.

Do not create a giant CRM, ERP, accounting engine, marketing platform, or predefined workflow catalog inside Studio.

The platform may provide reference Miniapps and example Automations, but these must remain independently deployable and replaceable.

## 6. How declarative Skills become executable processes

Describe the full conceptual authoring lifecycle:

1. Owner expresses a desired business outcome.
2. Agent discovers applicable Skills and available tenant capabilities.
3. Agent interprets the Skills and identifies required operations, decisions, inputs, and triggers.
4. Agent proposes an Automation definition.
5. Definition is validated against registered Miniapps, capability contracts, connector attestations, and tenant authorization.
6. Agent produces a human-readable process explanation and explicit list of unsupported requirements.
7. Owner reviews and authorizes activation.
8. Runtime executes the Automation through the framework.
9. Studio provides run evidence, approvals, failures, and recovery.
10. Agent can propose a process revision, but cannot silently modify an active version or bypass promotion controls.

Explain how to map skill instructions to:

- Agent-only guidance.
- Deterministic domain logic.
- Capability references.
- Process control flow.
- Trigger configuration.
- Human approval gates.
- Operational runbooks.
- Test and verification requirements.

Explain how an agent can reason about unknown or unavailable capabilities without fabricating support.

AI-generated decisions must be distinguished from deterministic business rules. Define a constrained agent-step contract: declared inputs, validated structured outputs, explicit authority, and no direct ambient access to provider credentials or unrestricted external effects.

## 7. Execution model and failure contracts

Preserve the existing A/B/C/D Miniapp execution classification.

Clarify that classification applies to tool or invocation boundaries as defined by the existing framework, not blindly to every Automation as a whole.

Describe:

- Class A: synchronous request/transaction.
- Class B: independently retryable background execution.
- Class C: bounded multi-step or batch execution with progress and recovery.
- Class D: durable workflows with long-lived waits, checkpoints, human or external events.

Explain how a multi-stage Automation can compose capabilities with different execution requirements without introducing a second orchestration model.

ADR-001 remains the authority for Class D substrate decisions. Do not assume DBOS adoption.

For inbound webhooks, explicitly explain how delivery acknowledgement relates to downstream execution. A provider should not be forced to wait for a long-running Automation to finish. Address durable intake, deduplication, event persistence, and subsequent dispatch without inventing unsupported implementation details.

## 8. API-first architecture

Describe the external and internal interaction boundaries.

At minimum distinguish:

- Automation management API.
- Automation validation and activation API.
- Execution/invocation API.
- Run status and evidence API.
- Approval/decision API.
- Capability discovery API.
- Connector and tenant-connection management API.
- Webhook registration and ingress API.
- Event and trigger dispatch contracts.

For each, explain its consumer, responsibility, tenant authorization requirements, and relationship to the core execution model.

Do not invent endpoint paths or finalize request/response JSON schemas prematurely. Define the conceptual contracts and identify where later ADRs or implementation specifications are required.

The API must expose the same underlying domain contracts used by the Studio interfaces and agents. Do not create separate business logic for different consumers.

## 9. Multi-tenancy, security, and governance

Explain how the architecture enforces:

- Organization → tenant → workspace boundaries.
- Tenant-scoped Automation definitions and versions.
- Explicit, revocable delegation.
- Tenant-specific provider connections and credentials.
- RLS and fail-closed tenant context.
- Separation of deployment secrets from tenant credentials.
- Short-lived, purpose-bound broker leases.
- Agent principal versus human authority principal.
- Approval separation and non-self-approval.
- Environment separation: development, preview, staging, production.
- Immutable/versioned artifacts and explicit promotion.
- Webhook endpoint ownership, verification secrets, replay prevention, and routing isolation.
- Audit and evidence retention.

Do not design a parallel authorization model.

Distinguish trust mode (dry-run, confirm, autonomous) from execution category and compute placement. These remain independent dimensions under existing decisions.

## 10. Studio product experience

Explain how the architecture supports the two distinct Studio experiences:

### Roadmap A — Engineering Studio

For developers and coding agents:

- Author Miniapps.
- Define and validate contracts.
- Develop and test.
- Manage versions.
- Build and deploy previews.
- Diagnose failures.
- Promote and release software.

### Roadmap B — Automation Operations Studio

For business owners, operators, QA, and managers:

- Discover Skills and available capabilities.
- Create and configure Automations.
- View process definitions and versions.
- Configure triggers, including webhooks.
- Manage tenant connections.
- Review authorization requirements.
- Monitor process executions.
- Inspect failed steps and UNKNOWN effects.
- Reconcile exceptions.
- Inspect evidence and operational health.

Both experiences share domain services, contracts, identity, authorization, runtime, connectors, and evidence. They must not develop competing backends or collapse into one confusing information architecture.

Do not design detailed screen mockups. Describe the capabilities and information architecture at a high level.

## 11. Lead-to-Payment as an architectural proof, not a platform catalog

Use Lead-to-Payment as the first illustrative vertical slice.

Show how it can be expressed entirely through the generic abstractions:

- Skill guidance.
- Automation definition.
- Manual, API, scheduled, and inbound webhook triggers as appropriate.
- Capability references implemented by Miniapps.
- Connector bindings.
- Conditional decisions.
- Agent reasoning where necessary.
- Human authorization.
- Effects and idempotency.
- Evidence and recovery.
- Customer acceptance and external event correlation.

Do NOT turn contact upsert, estimate creation, invoice issuance, payment collection, or message sending into platform primitives.

They are ordinary business capabilities implemented through Miniapps and connectors.

Also explain how the same platform architecture could support unrelated examples such as employee onboarding, order fulfillment, project coordination, marketing campaign execution, or financial reporting without changing the platform core.

The test of process agnosticism is not the number of workflows shipped. It is whether new workflows can be introduced by defining capabilities and composing existing contracts without modifying the platform runtime.

## 12. Implementation roadmap and sequencing

Provide a high-level, dependency-ordered roadmap.

Preserve existing roadmap numbering and distinctions:

- F0 shared foundations.
- Roadmap A Engineering Studio B0–B5.
- Roadmap B Automation Operations Studio R0–R8.
- Existing runtime/framework track and ADR-001 gates.

Do not create competing phase numbering or silently modify the Ultimate Roadmap.

Identify which design decisions must be frozen before implementation:

- Automation definition and version lifecycle.
- Relationship between Skill, Automation, Miniapp, and Capability.
- Trigger and event semantics.
- First-class webhook ingress contract.
- Minimal control-flow semantics.
- Execution and recovery contracts.
- Agent-step boundaries.
- Capability discovery and attestation.
- Tenant and environment scoping.
- API boundaries.
- Process-version promotion and rollback.

Separate:

- Already implemented infrastructure.
- Existing contracts that should be reused.
- New abstractions genuinely required.
- Deferred features.
- Owner decisions.

Give a practical path from the current F0 security remediation to a working Lead-to-Payment demonstration without building the entire platform up front.

Avoid speculative infrastructure and avoid allowing the first vertical slice to dictate the permanent core abstraction set.

## 13. Required architectural artifacts

Produce the following:

### Artifact A: `plans/rpa-platform-grand-vision.md`

The canonical architecture specification.

It must include:

1. Executive vision.
2. Goals, non-goals, and governing principles.
3. Consistent glossary and conceptual model.
4. Skill → Automation → Miniapp → Capability → Connector relationships.
5. Minimal platform abstractions, including first-class webhook ingress.
6. Explicit platform-versus-business-capability boundary.
7. Process authoring lifecycle.
8. Runtime and execution semantics.
9. Webhook, event, polling, and scheduler relationship.
10. API-first architecture.
11. Identity, tenancy, security, and governance.
12. Studio A/B product relationship.
13. Lead-to-Payment illustrative process.
14. Unrelated-process generality examples.
15. Dependency-ordered roadmap.
16. Open decisions and deferred topics.
17. Architecture acceptance criteria.

Include conceptual diagrams and tables where they improve precision. Make clear which diagrams are conceptual rather than finalized implementation topology.

### Artifact B: `plans/rpa-platform-grand-vision-review.md`

A separate adversarial architecture review.

The review should evaluate the resulting specification, not merely repeat its contents.

## 14. Adversarial review requirements

Conduct a genuinely adversarial pass, preferably using a second agent that did not author the design.

Challenge at least these failure modes:

1. **Primitive explosion:** does the design still smuggle business operations into the platform core?
2. **Over-abstraction:** are the proposed concepts too numerous or overlapping?
3. **Workflow-engine duplication:** does the design invent a second engine outside Miniapp categories and ADR-001?
4. **Skill confusion:** does it blur agent knowledge, executable code, and process definitions?
5. **Automation/Miniapp confusion:** are their lifecycles and responsibilities unambiguous?
6. **Capability leakage:** does the design make capabilities equivalent to provider-specific endpoints or hardcoded business functions?
7. **Webhook confusion:** are endpoints, deliveries, events, triggers, and executions distinct?
8. **Webhook security:** can spoofed, replayed, duplicated, out-of-order, oversized, or cross-tenant events cause unauthorized execution?
9. **Inbound/outbound confusion:** does outbound delivery unnecessarily expand the webhook ingress abstraction?
10. **Agent authority:** can an agent bypass approval, invent capabilities, or execute unregistered operations?
11. **Process determinism:** are deterministic logic and probabilistic agent reasoning adequately separated?
12. **External effects:** can a process duplicate payments, messages, or other writes after an ambiguous outcome?
13. **Long-lived execution:** are human waits and external events durable without introducing an unauthorized second execution engine?
14. **Provider portability:** are connector differences represented honestly without forcing false lowest-common-denominator APIs?
15. **Multi-tenancy:** are data, credentials, events, runs, and Automation versions properly scoped?
16. **Versioning:** what happens when a Miniapp capability changes while existing Automations or executions still reference old versions?
17. **Studio duplication:** do Roadmaps A and B share services while retaining separate information architectures?
18. **Premature complexity:** does the design introduce abstractions, schemas, or infrastructure that no current use case justifies?
19. **First-slice distortion:** has Lead-to-Payment unintentionally shaped the core around CRM, invoices, and payments?
20. **Implementation feasibility:** can the architecture be implemented incrementally within the existing Go, Miniapp, Neon, Fly, and security constraints?

Every material finding must include:

- Severity.
- Exact section or conceptual contract.
- Concrete failure scenario.
- Why existing controls do not address it.
- Recommended correction.
- Whether it is a blocker or future implementation risk.

Distinguish structural blockers from preferences. Do not invent findings to make the review look thorough.

## 15. Architecture acceptance criteria

The specification is successful only if a reader can answer these questions consistently:

1. What exactly is a Skill?
2. What exactly is an Automation?
3. What exactly is a Miniapp?
4. What exactly is a Capability?
5. What does a Connector own?
6. What is the smallest set of platform abstractions?
7. Where do webhooks enter the platform, and how do they become process triggers?
8. How are webhooks different from events, polling, and scheduled triggers?
9. Who decides what an agent can execute?
10. Where do deterministic business rules live?
11. Where does agent reasoning live?
12. How is process execution recovered after crashes?
13. How are external side effects made safe?
14. How are long-running processes handled without unnecessary workflow-engine adoption?
15. How can a new business capability be added without changing platform code?
16. How can a new Automation be created without changing the runtime?
17. How can a new provider be integrated without rewriting the Automation?
18. How are tenants and credentials isolated?
19. How do Engineering Studio and Automation Operations Studio relate?
20. Why will the architecture not become a bloated collection of business-specific primitives?

Add explicit architecture tests expressed as scenarios:

- Introduce a new business process using existing platform abstractions, with no runtime changes.
- Introduce a new business capability through a Miniapp, without creating a new platform primitive.
- Introduce another provider through a connector, without changing the process definition where capability equivalence holds.
- Introduce an inbound webhook trigger without embedding provider-specific webhook logic into the Automation definition.
- Demonstrate an unsupported provider capability honestly.
- Demonstrate that an agent cannot execute an unregistered or unauthorized capability.
- Demonstrate recovery from a duplicate webhook delivery and an ambiguous external effect.
- Demonstrate that a human approval survives process suspension and restart.

## 16. Scope and operating constraints

This is a design-only assignment.

Do not:

- Write application code.
- Add migrations or database tables.
- Change existing Miniapp contracts.
- Implement webhook handlers.
- Implement a process designer or workflow canvas.
- Add business-specific platform primitives.
- Add a new orchestration engine.
- Introduce a Python or Hermes runtime dependency.
- Modify the Ultimate Roadmap silently.
- Provision or mutate Fly or Neon resources.
- Perform provider OAuth or credential enrollment.
- Commit or push.

The agents may create and revise the two design documents only. Preserve other uncommitted work.

Make reasonable architectural decisions autonomously, but clearly label owner decisions that genuinely affect product scope, security, or irreversible architecture.

If the current repository contradicts an assumption in this prompt, report the discrepancy and recommend a resolution rather than inventing implementation facts.

## 17. Final report

Conclude with:

- The proposed minimum platform abstraction set.
- The precise role of webhooks.
- The distinction between business capabilities and platform primitives.
- The Skill → Automation → Miniapp → Capability → Connector model.
- The most important architectural decisions.
- Adversarial findings and their disposition.
- Unresolved owner decisions.
- Recommended next design or implementation assignment.

Final verdict:

- **APPROVED:** no structural blockers; the specification is coherent and sufficiently precise to guide implementation planning.
- **APPROVED WITH AMENDMENTS:** architecture is sound but named amendments must be incorporated before adoption.
- **NOT APPROVED:** important conceptual boundaries, security invariants, or execution contracts remain unresolved.

Do not start implementation. The deliverable is a clear, reviewable architecture that prevents future teams from independently inventing incompatible interpretations of Miniapp Studio.

**My recommendation:** Have one agent author the architecture and a different agent review it. The most important outcome is agreement on the minimal abstraction set—especially the boundary between Automation, Capability, Trigger, Webhook, and Miniapp—before anyone starts designing schemas or implementing another runtime.

