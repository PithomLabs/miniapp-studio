# **Miniapp Studio roadmap — November 24, 2026**

P0 complete

P1 complete

P2 complete

P3 next

You've completed the foundation. The project now moves from building the execution engine to connecting it to real infrastructure and delivering the business-owner experience.

## **1. Progress so far**

**P0 — Framework foundation**

PASS

PostgreSQL 19 validation, restart tests, race tests, Class A invocation, and approval resumption. Framework pinned at `73351a8`.

**P1 — Specification**

ACCEPTED

MiniappSpec, deterministic validator, canonical hashing, fixtures, and adversarial review.

**P2 — Execution engine**

ACCEPTED

Generic runner, strict interpreter, server renderer, fixture dry-run, read-only enforcement, and tenant-aware execution.

## **2. Remaining roadmap**

**P3 — Fly and infrastructure adapters**

NEXT

Depends on P0 · Infrastructure foundation

Build the infrastructure layer that makes generated applications real.

- Owned Fly interfaces and `sprites-go` adapter.
- Typed Fly Connectors REST client.
- Fake implementations and contract tests.
- Compare Sprite-per-app versus Fly-App-per-app deployment.
- Decide MPG versus Neon through compatibility testing.
- Implement slug reservation, resource provisioning, and stable URL persistence.
- Reconcile failed provisioning and uncertain outcomes.

**Exit:** infrastructure adapters work, topology decisions are recorded, and per-app URL lifecycle is proven.

**P4 — Studio server and user interface**

Depends on P1 + P3 · Product experience

Build the actual application through which business owners operate Studio.

Six screens:

1. Home dashboard.
2. Create App — conversational builder and live preview.
3. Connectors — ecosystem explorer and Custom API wizard.
4. Validate & Deploy.
5. My Apps — application and URL management.
6. Activity — execution and deployment evidence.

Also implement authentication, workspace sessions, permissions, and run/evidence views.

Every consequential action must pass through `runtime.Service`.

**Exit:** users can manage applications and connectors through the real Studio server.

**P5 — Builder agent and golden path**

Depends on P2 + P3 + P4 · End-to-end product

This is where the separate engineering components become a usable product.

- Run the builder agent on a real Sprite.
- Implement draft → validate → fix.
- Translate natural-language requirements into MiniappSpec.
- Generate the Invoice Review Desk.
- Connect a real provider.
- Deploy the generated application.
- Verify its independent Fly-issued URL.
- Demonstrate editing, redeployment, version history, and failure recovery.

**Exit:** one complete, repeatable application-generation journey.

**P6 — Rehearsal and submission**

Depends on P5 · Delivery

- Clean-state demonstration rehearsal.
- Audit every README and product claim against actual passing tests.
- Verify limitations and failure behavior.
- Complete organizer eligibility filing.
- Submit the project.

**Exit:** a credible, repeatable demonstration with evidence supporting every claim.

miniapp-studio-implementation-plan-consolidated(2).md

## **3. The critical acceptance gates**

These are the important milestones to watch as implementation progresses.

| Gate | What it proves                                     |
| ---- | -------------------------------------------------- |
| G0   | Framework dependency and validation truth          |
| G1   | Real Sprite agent → governed connector call        |
| G2   | Accurate connector ecosystem discovery             |
| G3   | Custom API creation, testing, and scoping          |
| G4   | Specification generation and approval resumption   |
| G5   | Rendering from MiniappSpec                         |
| G6   | Safe, repeatable application execution             |
| G7   | Demonstration readiness                            |
| G8   | All consequential actions use the governed runtime |
| G9   | Studio deployment on Fly                           |
| G10  | Stable, independently issued per-app URLs          |
| G11  | Secure control-plane integration                   |
| G12  | Complete first-run-to-deployment rehearsal         |

miniapp-studio-implementation-plan-consolidated(2).md

## **4. Architectural rules that remain in force**

Three things must not get lost as the project gets closer to the demo.

**First: G-01 remains unresolved.** External writes stay disabled until cross-invocation effect ownership transfer is implemented and tested. Do not bypass the framework's refusal behavior.

**Second: multi-tenancy remains an architectural requirement.** Server-derived tenant identity and isolation must continue across application records, execution, approvals, connectors, and deployment resources.

**Third: keep the cut line narrow.** One excellent working application is more valuable than multiple incomplete verticals. No workflow canvas, marketplace, billing, multi-tenant admin, custom domains, or DBOS authoring UI in this release.



miniapp-studio-implementation-plan-consolidated(2).md

miniapp-studio-implementation-plan-consolidated(2).md



## **5. Where the project stands strategically**

# **Foundation → Infrastructure → Experience → Demonstration**

1. **P0–P2: Engine built.** The application specification can be validated, interpreted, rendered, and executed safely.
2. **P3: Infrastructure connected.** Applications gain real resources, connectors, deployments, and URLs.
3. **P4: Product experience built.** Business owners gain a usable Studio interface.
4. **P5: Product demonstrated.** A real user request becomes a deployed business application.
5. **P6: Product delivered.** A clean, repeatable demonstration backed by evidence.

**My recommendation: make P3 the immediate priority.** Its two major architectural decisions—deployment topology and database provider—should be settled through working spikes, not prolonged discussion. Once those are proven, P4 and P5 can build against real infrastructure rather than assumptions.

The destination remains unchanged: **one business owner describes a problem, Miniapp Studio generates an application, and that application becomes accessible at its own stable URL.**