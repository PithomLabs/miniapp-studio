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