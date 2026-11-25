I've reviewed the uploaded `plan2(20261124-082438).md` against our Miniapp Studio objectives and the architectural decisions we've established.

**My verdict: strong direction, but I would request one revision before approving it as the implementation baseline.**

It is considerably closer to what we want. The agent has correctly recognized that this is a new product built on the existing Miniapp framework, not a collection of independently coded applications.&#x20;

plan2(20261124-082438).md



## **1. What I particularly like**

There are several excellent decisions I would preserve.

| Decision                                               | My assessment |
| ------------------------------------------------------ | ------------- |
| Separate `github.com/PithomLabs/miniapp-studio` module | Correct       |
| Existing framework as pinned dependency                | Correct       |
| Studio dogfoods Miniapp through `runtime.Service`      | Essential     |
| One generic, statically registered spec runner         | Excellent     |
| Declarative MiniappSpec instead of generated programs  | Essential     |
| Deterministic validation and fixture testing           | Essential     |
| Fly APIs and Sprites SDK behind adapters               | Correct       |
| Separate typed REST client for connectors              | Correct       |
| Invoice Review Desk as golden-path application         | Good choice   |
| Unique, persisted Fly-issued application URLs          | Essential     |

I particularly like the recognition that the existing Miniapp framework does not yet provide everything Studio needs.

The agent reports that `fly/`, `tokens/`, `capability/clients/` and `executor/river/` are empty, while `deploy/` contains interfaces rather than complete implementations. It also inspected `sprites-go` and found that connector management requires a separate REST client.&#x20;

plan2(20261124-082438).md



That is exactly the kind of repository-grounded assessment we need.

## **2. Five things I would require before approving the plan**

### **A. Resolve the execution semantics of the generic runner**

This is my biggest concern.

The agent proposes a generic runner using `runtime.Service`, while simultaneously declaring DBOS durable workflows out of scope for the Hacksgiving demo.

Those two decisions need an explicit execution contract.

I would require:

- Specify the Miniapp execution class used by the MVP runner.
- Identify which operations are synchronous and which are asynchronous.
- Define how approval requests are persisted and subsequently decided.
- Define behavior when the runner or application process crashes.
- Explicitly distinguish persisted application state from durable workflow recovery.
- Do not claim automatic workflow resumption, exactly-once effects or multi-day waits unless verified.

We should not accidentally implement half a durable workflow engine during the hackathon.

### **B. Make Fly.io integration a first-class architecture**

The plan includes useful Fly verification spikes, but I would make the operational responsibility explicit.

There should be three distinct deployment concerns:

1. **Miniapp Studio:** deployed as a Fly-hosted, serverless-style application using Fly's autostart/autostop capabilities.
2. **Builder agent:** runs on its designated Fly Sprite and uses actual Sprites connectors.
3. **Generated applications:** each receives its own stable, user-derived identity and Fly-issued URL.

The plan must decide how generated applications map to Fly Apps, Machines, Sprites or a shared routing layer.

Do not assume the cheapest topology is automatically the correct one. Test isolation, cold-start latency, cost, URL stability and operational complexity.

### **C. Explicitly test the entire generated application lifecycle**

The current success criteria correctly require one complete generated application with a persisted URL.&#x20;

plan2(20261124-082438).md



I would make the acceptance test:

**Golden-path acceptance test**

1. Describe an application using natural language.
2. Generate a valid MiniappSpec.
3. Reject an intentionally invalid specification.
4. Preview the generated application.
5. Connect an actual Fly Sprites connector.
6. Deploy the application to Fly.
7. Receive its actual Fly-issued URL.
8. Open the URL independently and use the application.
9. Return to Studio, modify the specification and redeploy.
10. Verify the application URL remains stable and its version history is recorded.

This is the experience that makes the hackathon MVP a genuine starting product rather than a prototype that must be discarded.

### **D. Make the connector catalog and Custom API non-negotiable**

The plan already includes the right foundation:

- Public ecosystem catalog.
- Organization-configured connectors.
- Sprite-level authorization.
- Custom API creation and testing.
- Permission inspection.

I would add an explicit test proving that a generated application cannot invoke an API path outside its authorized scope.

The Custom API credentials must remain inside Fly's connector gateway, not the Sprite, generated specification or application logs.

### **E. Require a polished product experience**

The current plan concentrates heavily on implementation and verification.

I would insist on explicit UI acceptance criteria as well.

The MVP should contain:

| Screen            | Required outcome                                       |
| ----------------- | ------------------------------------------------------ |
| Home              | Professional application dashboard                     |
| Create App        | Conversational builder with live preview and inspector |
| Connectors        | Ecosystem explorer and Custom API wizard               |
| Validate & Deploy | Clear validation, deployment progress and errors       |
| My Apps           | Applications, URLs, status and management              |
| Activity          | Deployment and execution evidence                      |

The builder and generated application must look like actual commercial software, including loading, empty, failure and success states.

## **3. My decisions on the five outstanding questions**

The agent explicitly lists five owner decisions.&#x20;

plan2(20261124-082438).md



Here is what I recommend:

| Decision                 | Recommendation                                                                                                                                         |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Framework pin            | Freeze one exact immutable commit that passes the required tests. No floating `or newer`.                                                              |
| MPG vs Neon              | Prefer Fly Managed Postgres if compatibility tests pass; otherwise retain Neon. Don't let future DBOS constraints block the MVP.                       |
| Application URL topology | Run the three competing topology spikes before selecting. Require a real URL round-trip and collision tests.                                           |
| Connector anchor         | Use the best working organization-native connector, preferably Box; Custom API is the fallback.                                                        |
| Git initialization       | Yes, create a separate Studio repository. Verify the repository root before initializing; never accidentally initialize the parent Miniapp repository. |

One additional environmental issue deserves attention: the agent reports that `flyctl` exists but cannot authenticate or operate in its current sandbox because of the read-only Fly configuration. Actual Fly integration tests therefore need an authorized execution environment, not mocked success.&#x20;

plan2(20261124-082438).md



## **4. Prompt I would send the coding agent**

I would give the agent this revision request before accepting its implementation plan.

# Revision Request: Adversarial Review and Completion of Miniapp Studio Implementation Plan

Revise the proposed implementation plan using the following mandatory architectural and product requirements.

Do not implement code. Produce a revised, evidence-backed implementation plan for owner review.

## 1. Execution semantics

Resolve the apparent ambiguity between the generic Spec Runner and the decision to defer DBOS durable workflow functionality.

Specify:

- The execution class and runtime path for the MVP Spec Runner.
- Synchronous versus asynchronous operation semantics.
- Approval persistence and decision handling.
- Crash, restart, duplicate submission and uncertain external-effect behavior.
- The precise boundary between MVP persistence/recovery and future DBOS durable execution.

Do not claim durable waits, automatic workflow resumption or exactly-once external effects without verified implementation and test evidence.

## 2. Fly.io deployment architecture

Explicitly design and compare:

- Miniapp Studio as a Fly-hosted, serverless-style application.
- The Sprite-hosted personal builder agent.
- Generated applications with their own stable, user-derived Fly-issued URLs.

Evaluate Fly Apps, Machines, Sprites, shared routing and per-application deployment as appropriate.

Record the trade-offs involving isolation, cost, cold starts, scale-to-zero/autostart, provisioning time, URL stability and operational complexity.

Do not invent Fly URLs. Persist authoritative deployment responses.

## 3. Full application lifecycle

Define automated acceptance tests for:

Describe → Generate → Validate → Preview → Connect → Deploy → Open by URL → Edit → Redeploy.

Verify that generated applications remain independent, their URL identities are stable, collisions are handled safely and deployment failures do not corrupt existing applications.

## 4. Connector ecosystem and extensibility

Retain the separate public ecosystem catalog, organization connection inventory and per-Sprite authorization state.

Make Custom API creation, testing, credential isolation and endpoint permissions first-class requirements.

Include actual Fly API contract tests and negative authorization tests. Do not treat a mocked connector as proof of real integration.

## 5. Miniapp dogfooding

Studio must itself be built using the Miniapp framework.

All consequential Studio business operations must use the canonical Miniapp runtime, registry, authorization and audit path. Identify exactly which components serve the UI and which invoke Miniapp operations.

Do not create an alternative backend that bypasses Miniapp.

## 6. Product-quality requirements

Specify implementation and acceptance criteria for Home, Create App, Connector Explorer, Validate & Deploy, My Apps and Activity.

Include professional visual design, responsive behavior, accessible controls and complete loading, empty, error and success states.

The generated Invoice Review Desk must be a functioning application, not a screenshot or static mockup.

## 7. Framework and dependency discipline

Pin an exact immutable Miniapp framework revision and record evidence for every reused capability.

Distinguish implemented functionality from interfaces, prototypes, empty packages and deferred features.

Do not copy the existing framework into the Studio repository.

## 8. Required deliverable

Produce the revised document:

`docs/architecture/miniapp-studio-implementation-plan.md`

Include:

- Executive summary.
- Repository inspection and evidence inventory.
- Target architecture and component responsibility matrix.
- MiniappSpec and interpreter contracts.
- Fly deployment and URL topology decisions.
- Connector integration architecture.
- UI and end-user workflows.
- Security and failure analysis.
- Sequenced implementation tasks.
- Dependency graph.
- Acceptance tests and measurable exit gates.
- Risks and mitigations.
- Explicit roadmap and deferred capabilities.
- Unresolved owner decisions.

Map every existing governing gate G0–G12 to concrete implementation tasks and test evidence.

Separate blockers from optional improvements.

Follow the principle: investigate underlying problems, prove invariants where failures can occur, and implement durable solutions rather than superficial workarounds.

Do not modify source code, dependencies, configuration, migrations, deployment resources or existing governing plans.

Stop after producing the implementation plan and report the evidence and unresolved decisions required for owner approval.

## **5. My overall assessment**

Directionally approved

I would preserve this agent's architecture rather than restart the planning exercise.

The important thing is not to let the implementation plan become an excuse to build infrastructure endlessly. The MVP must deliver a polished Studio that generates, deploys and serves a real application.

The product's governing philosophy should remain:

**The end user describes the business software. Miniapp generates it. Fly provides its infrastructure. The user owns the resulting application.**

DBOS-powered durable business workflows become the next major evolution—not a shortcut or an unverified promise in the first release.