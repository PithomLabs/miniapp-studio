# **Architectural assessment: Yes, we're on the right track**

My conclusion is that these documents **validate our fundamental Miniapp Studio architecture, but reveal an important missing dimension that we should incorporate into the roadmap.**

I would not redesign B0–B5. I would refine how we define, build, validate and deploy miniapps within that roadmap.

The most important insight is:

**Execution durability and compute placement are two independent architectural decisions. Trust determines where code runs. Business requirements determine how it executes.**

This is a significant clarification.

## **1. The architectural model we should adopt**

There are actually three independent decisions for every miniapp tool.

Dimension 1

**Execution category — how does it execute?**

**A — Handler**

Synchronous operation, Postgres transaction

**B — Job**

One-shot retryable work, River

**C — Batch**

Bounded multi-step work, River by default

**D — Workflow**

Durable business process, DBOS

Dimension 2

**Compute placement — where does the code run?**

[Run a Static Website · Fly Docs](https://images.openai.com/static-rsc-4/2OPAFGeuD5xzX7Ii0MxyeGsJAIWQvi4W2R8mTQq7xFtJGR9xER32j5DLg5cmIIO0F999MRiDpfl8DuMzCgL8iL6ub2YVNrAABPL1oXfGscT0DUUbR1Ol2eDjT5ejzHMwOgsR8AOyRbNZc4bybshMuYNPwkBqCt1EbzdmjYZUXns?purpose=inline)

**Fly Sprites**

Agent-authored code, isolated workspaces, development and mutable previews.

[Per-User Dev Environments with Fly Machines · Fly Docs](https://images.openai.com/static-rsc-4/NKhNDagTFgnnbvtjKe3-7875w_EyN21bL3VFAvkhIAVZUtN87ZVQ8waSIrhxN9HbBn8PJWRWUXyRaNnDVbqaKv0IiS_t6vNQ3ZR08SC7gwWYbUmR9KURE1RUyDfohCt6crmisA9ndAkaFpzffEj5VXAfNVEFeYGoBmWGX2dKDc4?purpose=inline)

**Fly Machines**

Trusted, reviewed application code, production execution and deployment-parity testing.

**Hybrid execution**

A trusted DBOS workflow runs on a Machine and delegates a bounded, isolated task to a Sprite.

Dimension 3

**Authorization — what is this invocation allowed to do?**

Dry-run, confirm, autonomous; tenant/resource scope, explicit approvals, rate limits and blast-radius controls.

The category guidelines explicitly establish that execution class and placement must not be collapsed into one decision. They also distinguish caller identity from the trustworthiness of the code being executed.&#x20;

app_category_guidelines.md

app_category_guidelines.md



This separation is essential because an AI agent can invoke trusted production code without that code needing to run inside a Sprite.

Conversely, agent-authored code remains untrusted even if it looks like an ordinary small script.

## **2. The correct Fly placement matrix**

The first attached document's initial rule of thumb—Sprites for development/testing/staging and Machines for production—is too simplistic.

Its refined analysis is much more important: staging environments intended to validate production deployment, networking, container builds and routing should use Machines, because Sprites do not reproduce the same OCI deployment and Machine configuration path.&#x20;

machines_vs_sprites.md



I would adopt this placement matrix:

| Category     | Trusted, reviewed application | Agent-authored / untrusted code                                        |
| ------------ | ----------------------------- | ---------------------------------------------------------------------- |
| A — Handler  | Machine                       | Sprite for isolated development/prototyping                            |
| B — Job      | Machine                       | Sprite for isolated task testing                                       |
| C — Batch    | Machine                       | Sprite for isolated data-processing/testing                            |
| D — Workflow | Machine running DBOS          | Sprite for development/testing; Hybrid for bounded delegated execution |

There is an important distinction for Class D:

- DBOS runs on the trusted Machine.
- Neon holds authoritative business and workflow-related data.
- A Sprite can execute a specific bounded, untrusted task.
- Sprite checkpoints are filesystem snapshots, not DBOS workflow checkpoints.

Never make a Sprite the authoritative recovery mechanism for a multi-day business process.&#x20;

machines_vs_sprites.md



Also, promoting an agent-developed prototype requires a reviewed, immutable build artifact. Promoting a Sprite snapshot directly into production is not an acceptable release path.

### **The subtle but important distinction about Staging**

We should distinguish two things:

**Workspace Preview:** An agent wants to inspect its current mutable work. Use a Sprite.

**Application Preview / Staging:** We want to establish that the actual deployable artifact behaves correctly with production-like infrastructure. Use a Machine.

That means our previously defined distinction between Workspace Preview and Application Preview was exactly right.

## **3. What this means for our B0–B5 roadmap**

I would preserve the existing roadmap.

**B0 — Studio manages one task**

Keep

Local control-plane foundation. No architectural change.

**B1 — Studio manages its local development loop**

Keep

Local execution, evidence, checkpoints and recovery remain important. The local PostgreSQL test runner supports this foundation.

**B2 — Remote Builder Sprite**

Confirmed

This is where the resident coding agent lives. Persistent workspace, GitHub Connector, task handoff, checkpoints and Workspace Preview.

The Builder Sprite is a development computer, not the production runtime of trusted miniapps.

**B3 — Application Preview**

Strategically important

Build an immutable OCI artifact and deploy it to a temporary Fly Machine with an isolated Neon branch.

This is the first real proof that agent-developed software can move from workspace development into a trusted deployment pipeline.

I would keep it optional for the initial Sprite-centered demo, but mandatory before claiming release readiness.

**B4 — Studio troubleshoots itself**

Keep

The distinction becomes clearer:

- Diagnose deployed applications using Machine runtime telemetry.
- Reproduce and repair inside a Sprite.
- Rebuild and verify the corrected artifact before deployment.

**B5 — Release management**

Keep

Staging and Production use trusted, reviewed builds on Fly Machines, with separate Neon environments, approvals, health verification and rollback.

My conclusion: B0–B5 already reflect the right separation. These documents give us a much stronger theoretical and practical justification for it.

## **4. What is missing from our architecture?**

This is where I would make a meaningful roadmap adjustment.

The documents describe a framework that classifies each miniapp tool, enforces its category, selects its placement, and validates that the implementation complies with the declared contract.

Our existing Miniapp Studio design has the right building blocks, but we need to make this execution model explicit and enforceable.

I would introduce a cross-cutting capability:

**Miniapp Execution Classification and Placement Admission**

It should build on our existing specification model rather than create another competing manifest system.

For every exposed tool, the platform should eventually know:

| Contract                | What it establishes                               |
| ----------------------- | ------------------------------------------------- |
| Execution class         | A, B, C or D                                      |
| Placement profile       | Machine, Sprite or approved Hybrid                |
| Trust profile           | Trusted reviewed code or untrusted generated code |
| Trust mode              | Dry-run, confirm or autonomous                    |
| Idempotency identity    | How repeated requests collapse safely             |
| Entity key              | Which business entity requires serialized writes  |
| Capability requirements | Which provider operations are permitted           |
| Blast-radius policy     | Rate, volume, concurrency and kill-switch limits  |
| Recovery contract       | Retry, resume, reconcile or compensate            |

The document specifically proposes storing these decisions in the tool manifest and enforcing them through executor and placement admission checks. It also recommends content-addressed deployment-policy references so that changing a profile requires fresh validation.&#x20;

app_category_guidelines.md



That is important: **the platform should reject an invalid execution or placement decision before deployment, rather than discover the mistake after the application starts.**

### **The other critical platform responsibility: shared execution guarantees**

The framework should own common guarantees rather than asking every miniapp developer to reinvent them:

- Effect ledger with `PENDING`, `ATTEMPTING`, `CONFIRMED`, `FAILED`, `UNKNOWN`.
- Stable idempotency identities.
- Per-entity write serialization.
- One approval mechanism with expiry and material-change invalidation.
- Capability proxy and credential broker.
- Rate limits, volume limits and kill switches.
- Reconciliation before retrying an unknown external effect.
- Auditing and fail-closed authorization.

These are cross-cutting requirements, not additional execution categories.&#x20;

app_category_guidelines.md



## **5. How I would actually implement the categories**

This is another important lesson: do not implement everything immediately.

The guideline explicitly recommends choosing the lowest execution category that satisfies the business requirement.

| Category | Recommended adoption                                                                                      |
| -------- | --------------------------------------------------------------------------------------------------------- |
| A        | Establish first. Plain Go handlers and Postgres transactions.                                             |
| B        | Adopt when an independently retryable background task is required. River.                                 |
| C        | Pilot with real batch workloads. Compare River's progress/concurrency model against DBOS where justified. |
| D        | Adopt selectively after the workload passes ADR-001's durability and complexity gates.                    |

An important clarification: an approval does not automatically make a single-step tool Class D. A bounded approval wait can be handled by the shared approval mechanism and a one-shot operation. DBOS is for genuinely durable, multi-stage business processes.&#x20;

app_category_guidelines.md

app_category_guidelines.md



Also, keep the business rules in a pure domain core and put the framework-specific orchestration in thin shells. This lets the execution strategy change without rewriting the business logic.&#x20;

app_category_guidelines.md



This is precisely the kind of abstraction that should be built when a real miniapp requires it—not as a speculative framework project.

## **6. One architectural distinction I want us to preserve**

Our recent Host Execution Broker work and the capability proxy described in the category document are related, but they are not the same component.

| Component                | Responsibility                                                                                               |
| ------------------------ | ------------------------------------------------------------------------------------------------------------ |
| Host Execution Broker    | Safely grants the local coding agent narrowly scoped host operations, such as Docker test-runner execution   |
| Miniapp Capability Proxy | Safely grants deployed miniapps access to external systems such as CRMs, email providers and other SaaS APIs |
| Miniapp Studio           | Owns task orchestration, authorization, approvals, project state, evidence and lifecycle management          |
| Builder Sprite           | Provides the persistent development computer for agent-authored code                                         |
| Fly Machines             | Execute trusted, reviewed application workloads                                                              |

Do not turn the local Host Execution Broker into a universal privileged execution API for production miniapps.

Instead, preserve the shared security philosophy while giving each boundary its appropriate implementation.

For hybrid Class D execution, I would also keep the production-delegated Sprite task separate from the persistent Builder Sprite. A production workflow should not acquire accidental access to the agent's long-lived development environment.

## **7. My recommended roadmap adjustments**

I would make only three changes.

**First: preserve B0–B5.** There is no reason to restart or redesign the program.

**Second: add an execution-contract workstream.** Introduce per-tool classification, placement profiles, trust declarations and admission validation using our existing specification architecture.

**Third: make B3 a release-readiness prerequisite.** It need not delay the Sprite-centered demo, but a Sprite Workspace Preview must never be mistaken for proof that an application can be safely deployed.

I would not introduce DBOS for every operation, implement hybrid execution before we have a real workload requiring it, or build all four categories before delivering the first useful miniapp.

One document-quality issue: `app_category_guidelines.md` contains multiple model-generated drafts followed by a consolidated version. Its consolidated section explicitly gives plan4.1 and ADR-001 precedence where conflicts exist. We should eventually make one canonical, reconciled developer guide rather than treating the entire compilation as a single normative specification.&#x20;

app_category_guidelines.md

app_category_guidelines.md



## **Final verdict**

Architecture direction validated

I believe we are on the right track.

In fact, these documents strengthen three important architectural decisions we have already made:

1. **Sprites are the computers of our autonomous developers.**
2. **Machines are the execution environment for trusted, reviewed applications.**
3. **The execution engine is chosen by the business process's failure and recovery requirements, not by the compute environment or the AI agent calling it.**

The bigger opportunity is that Miniapp Studio can become more than an AI development environment. It can become a platform that understands the business operation, determines its execution contract, chooses a safe execution location, enforces authorization, and produces evidence that the promised outcome actually happened.

That is a coherent product architecture.

**My advice: don't pivot. Refine the contracts, keep building the working product, and let real miniapps demonstrate which platform capabilities deserve to be implemented next.**

That is exactly consistent with our reality-first engineering principle.&#x20;
