The roadmap's direction is sound. The important distinction is that we should not confuse a solid foundation with production completeness. We need enough of the foundation to demonstrate a trustworthy end-to-end product, without building every feature needed for a large commercial SaaS before the hackathon.
1. Where we are on the roadmap
The roadmap has five phases. The latest acceptance results update the original November 25 roadmap's status. 
miniapp-studio-roadmap-refresh-2026-11-25.md

Phase	Purpose	Current position
A — Conversation Infrastructure	Durable conversations, tenant isolation, approvals, events, SSE	Accepted
B — Sprite Agent Gateway	Secure Builder, coding-agent supervision, signed commands and callbacks	Part 1 accepted with bounded residuals; Part 2 awaits explicit GO
C — Genuine Two-Way Interaction	User ↔ Studio ↔ Gateway ↔ OpenCode, including clarification and same-session continuation	Not yet demonstrated live
D — Miniapp Build and Artifact Verification	Build, validate, test, inspect and approve a real Go/Miniapp artifact	Not yet completed
E — Approved Deployment	Deploy the approved artifact to a separate Runtime Sprite, verify health and rollback	Not yet completed

The architectural foundation is already substantial:
- Studio owns permanent business history and user decisions.
- Builder and Runtime are separate execution environments.
- Gateway handles authenticated, recoverable agent execution.
- Tenant isolation, key boundaries and human authorization are established.
- Go + Miniapp + MiniappSpec is the binding application contract.
These are not features we should casually compromise to save demo time. 
miniapp-studio-roadmap-refresh-2026-11-25.md

2. What actually matters for winning the hackathon
I recommend making the Invoice Review Desk the single golden path, rather than expanding the product into multiple workflows.
The intended judge experience
1. Business owner describes a problem
   
   "Build me an Invoice Review Desk so my team can review incoming invoices, flag discrepancies and track approval decisions."
2. AI asks a meaningful clarification
   
   The agent asks how discrepancies should be flagged or what information is required before review.
3. Owner answers inside Studio
   
   The same conversation continues. The agent demonstrably consumes the answer in its existing session.
4. Studio produces a working application
   
   A Go/Miniapp implementation with a valid MiniappSpec, passing tests and inspectable evidence.
5. Owner reviews and approves
   
   Studio presents the specification, changes, test results and artifact provenance.
6. Application runs independently
   
   A separate Runtime Sprite runs the approved application, with health verification.
One complete business outcome, not a collection of disconnected features.
This follows the roadmap's Phase C, D and E exits. 
miniapp-studio-roadmap-refresh-2026-11-25.md
 miniapp-studio-roadmap-refresh-2026-11-25.md
 miniapp-studio-roadmap-refresh-2026-11-25.md

One important safety constraint: while G-01 remains unresolved, use synthetic or fixture invoice data and simulated approval effects. Do not connect the demo to production accounting systems or enable consequential business writes.
3. The next sequence I recommend
First: close three small residuals. N8's re-provision restriction must appear at the point where an operator would re-provision; N5 needs an explicit post-recovery PID-absence check; report counts need exact-set wording. These are bounded documentation tasks, not another architecture review. 
adv_review_prompt14(1).md

Second: Phase B Part 2. Execute the existing owner-run live runbook after a separate explicit GO. Prove real Sprite provisioning, Gateway service identity, signed status, OpenCode execution, key-read denial, callbacks and sleep/wake recovery. Do not mistake fake-backed success for live evidence.

Third: Phase C. Demonstrate the actual two-way conversation. For the pinned OpenCode v1.19.6 noninteractive run, the native question tool is denied, so the documented file-drop Q&A mechanism must receive particular attention in the live proof. 
opencode-run-v1.19.6.md

Fourth: Phase D. Implement the minimum real artifact pipeline for the Invoice Review Desk: manifest and digest verification, MiniappSpec validation, Go/Miniapp build and tests, review presentation, and explicit approval. Do not attempt a generalized application marketplace or arbitrary framework support.

Fifth: Phase E. Deploy that approved artifact to its separate Runtime Sprite. Verify health and rollback. Make this part of the demo if the available hackathon scope and time permit, without bypassing approval.

Keep fleet billing, broad quotas, additional agent adapters, workflow engines, marketplace features and generalized administration outside the demo critical path. The roadmap explicitly defers these broader concerns. 
miniapp-studio-roadmap-refresh-2026-11-25.md


4. Prompt for the next implementation-agent task
I recommend giving the agent the following directive. It first closes the small residuals and produces an executable demo plan, then stops at the existing live-authorization boundary.
Implementation Directive — Miniapp Studio Hackathon Demo Critical Path
1. Mission
Act as the Senior Architect and implementation agent for:

github.com/PithomLabs/miniapp-studio

Our objective is to deliver a compelling, reliable, business-owner-friendly hackathon demonstration while preserving the foundational architecture established by the accepted roadmap.

The product must demonstrate that a business owner can describe a real business process, have Studio orchestrate an AI coding agent, clarify requirements through the original conversation, obtain a working Go/Miniapp application, review evidence, approve it, and run the approved artifact independently.

Our golden path is the Invoice Review Desk.

We are optimizing for a complete, credible business outcome, not maximum feature count.
2. Governing documents and decisions
Read and follow:
- miniapp-studio-roadmap-refresh-2026-11-25.md
- plans/agent-gateway-implementation-plan.md
- plans/prompt11_plan.md
- plans/adv_review_prompt12.md
- plans/adv_review_prompt14.md
- docs/prompt12-residual-closure-report.md
- docs/prompt11-correction-report.md
- docs/gateway-phase-b-evidence.md
- docs/gateway-phase-b-live-runbook.md
- docs/opencode-run-v1.19.6.md
- plans/miniapp-studio-implementation-agent-guidelines.md
Reconcile their contents with the actual current source tree. The most recent accepted decisions and reports supersede outdated roadmap status statements.

Do not assume that a file is committed merely because it exists in the working tree.
3. Current baseline
Treat the following as the current accepted baseline, subject to confirming the actual repository state:
- Phase A: accepted.
- Phase B Part 1: accepted with bounded residuals.
- N1: independently verified closed.
- N4: independently verified closed.
- R-DB: PostgreSQL 19 verification independently reproduced.
- R-NET: OpenCode provenance independently verified.
- P1–P4: corrected probes pass.
- N2–N10: tracked with the restrictions recorded in the accepted review.
- TestAdapterStatus: pre-existing, test-only cleanup flake; do not silently classify it as fixed.
- Phase B Part 2: not executed and not authorized.
- Phase C: live two-way interaction not yet demonstrated.
- Phase D: complete artifact-admission pipeline not yet delivered.
- Phase E: approved Runtime deployment not yet demonstrated.
Confirm these facts from repository evidence before planning work.
4. Immediate task — bounded residual closure
Close only the three remaining low-priority documentation/evidence findings from plans/adv_review_prompt14.md.
F-R1 / N8
Place the staging permission-window restriction directly beside the re-provision operation in docs/gateway-phase-b-live-runbook.md.

The operator must be told to:
- Verify staging is mode 0700.
- Verify there are no live untrusted sibling agent processes that can exploit staging recreation.
- Drain relevant agent activity first when the condition cannot be established.
Do not claim this race is fixed in provisioning code.
F-R2 / N5
Update the cold-recovery portion of runbook §11:
- Record the agent PIDs before the sleep/restart observation.
- After cold recovery, verify that the pre-restart agent processes are absent.
- Report any remaining process or ambiguous identity.
- Do not claim orphan cleanup succeeded solely because the Gateway emitted agent.failed.
F-R3
Correct the test-count language in docs/prompt12-residual-closure-report.md.

Enumerate the exact test sets or explicitly distinguish DB-backed subsets from all tests in the relevant files.

Do not change the tests to fit the numbers.

After these changes, verify only the affected documentation and cross-references. Do not launch another broad adversarial review.
5. Prepare the demo critical-path document
Create:

docs/hackathon-demo-critical-path.md

This document becomes the execution plan for the next delivery stage.

It must contain:
A. Demo narrative
Define the complete judge-facing story:
1. Business owner describes the Invoice Review Desk requirement.
2. Studio creates the conversation and build.
3. Gateway routes work to the correct Builder.
4. OpenCode performs real supervised work.
5. Agent requests a clarification.
6. Studio persists and renders that question in the original business conversation.
7. Owner answers.
8. The same OpenCode session consumes the answer and continues.
9. Agent produces a Go/Miniapp artifact and MiniappSpec.
10. Studio verifies the artifact and tests.
11. Owner reviews the changes and approves or requests rework.
12. The approved artifact is deployed to a separate Runtime Sprite.
13. Studio records the deployment and verifies health.
State exactly which steps are already implemented, fake-backed, locally tested, live-unverified, or not implemented.

Do not turn planned behavior into a capability claim.
B. Phase B Part 2 — owner-run live gate
Use the existing docs/gateway-phase-b-live-runbook.md.

Identify every owner decision that is still genuinely blocking.

Prepare the exact evidence checklist for:
- Builder provisioning.
- Gateway service boot and effective identity.
- Credential-drop self-test.
- Signed status round-trip.
- OpenCode no-op execution.
- Agent access denial to Gateway keys.
- Actual dual-user question/answer access.
- Studio callback delivery.
- Sleep/wake behavior.
- Resource cleanup and retention.
Do not execute these operations.

The owner must provide a separate explicit GO before any live step.
C. Phase C — minimum credible two-way interaction
Break Phase C into concrete implementation and verification tasks.

The primary acceptance requirement is not merely displaying a question. It is demonstrating that the owner's answer resolves the actual pending question in the same OpenCode session.

For pinned OpenCode v1.19.6, account for the documented noninteractive question-tool denial. The file-drop handshake is the currently documented mechanism and must be verified through live P-G3 evidence.

Do not invent CLI flags, parse ordinary prose as question metadata, or introduce an experimental second interaction mechanism.

Cover:
- Question identity and turn binding.
- Strict payload validation.
- Atomic handoff.
- Ownership and permissions.
- Stale-answer rejection.
- Same-session continuation.
- Honest failure when a session cannot resume.
- Studio history and SSE replay.
- Restart and reconnect behavior.
D. Phase D — minimum credible artifact pipeline
Plan a narrow vertical implementation that produces one actual Go/Miniapp Invoice Review Desk artifact.

Required foundation:
- Versioned artifact contract.
- Valid MiniappSpec.
- Go source/module and supported dependency metadata.
- Artifact manifest and SHA-256 digests.
- Studio-controlled artifact retrieval and verification.
- Path, archive, symlink and size validation.
- Existing MiniappSpec validation and admission.
- Explicit allowlist of supported Go/Miniapp application dependencies.
- Go/Miniapp build and test execution.
- Machine-readable test report.
- Unified diff and human-readable review.
- Explicit user approval or rework.
Include negative tests for:
- Non-Go implementations.
- Alternative frameworks/runtimes.
- Invalid MiniappSpec.
- Modified artifact bytes.
- Incorrect digests.
- Malicious paths and symlinks.
- Oversized artifacts.
- Failing tests.
- Secret-bearing human-visible output.
A prompt telling the coding agent to use Go is not sufficient. Enforce the application contract at the actual validation/admission/build boundary.

Do not build a generic marketplace or generalized plugin system.

Do not implement deployment inside Phase D.
E. Phase E — approved Runtime deployment
Plan the minimal separate Runtime Sprite path:
- Human approval and digest recorded.
- Runtime provisioned separately from Builder.
- Verified artifact transferred without rebuilding from mutable Builder state.
- Miniapp framework/runtime executes the application.
- No coding agent or Builder Gateway in Runtime.
- Health check before deployment is marked live.
- Rollback to a previously approved digest.
- Runtime identity, URL, digest and health status recorded in Studio.
Deployment must not be automatic.
F. Demo safety and governance
The demo must preserve:
- No production connector bindings in Builder Sprites.
- No consequential business writes while G-01 remains unresolved.
- Human authorization distinct from agent proposals.
- Tenant and application-workspace isolation.
- Studio as authoritative business history.
- Builder and Runtime separation.
- No direct browser-to-Sprite connection.
- No bypass of Miniapp validation or admission.
Use synthetic invoice data or safe fixture data. Approval screens may demonstrate human decisions without executing external financial effects.
6. Priority and scope discipline
Use three priority classes.

P0 — Demo blockers

Anything preventing the complete Invoice Review Desk golden path from working.

P1 — Foundation necessary for a trustworthy demo

Security, authorization, persistence, recovery, artifact validation and deployment health guarantees required by the path.

P2 — Post-demo production readiness

Fleet lifecycle, broad quotas, billing, usage metering, generalized administration and SSE scaling.

Do not implement P2 work as part of the hackathon critical path.

Do not add:
- A second coding-agent adapter.
- Another application framework.
- A new workflow engine or DBOS.
- Generic task management or workflow canvas.
- Marketplace or billing features.
- Lead-to-Booking Operations or unrelated applications.
- New architecture without a demonstrated requirement.
Preserve the existing accepted architecture.
7. Scheduling and dependency model
Build an explicit dependency graph.

The intended sequence is:
1. Close F-R1/F-R2/F-R3 documentation residuals.
2. Owner separately authorizes Phase B Part 2.
3. Execute the existing owner-run live verification.
4. Accept Phase B's live evidence or report concrete failures.
5. Complete the Phase C genuine two-way interaction.
6. Complete the minimum Phase D artifact verification and approval path.
7. Complete Phase E deployment and health verification if it is part of the judging demonstration.
8. Rehearse the judge-facing narrative from a clean starting state.
Do not skip a phase gate merely to shorten the schedule.

Do not start live operations just because the demo plan is complete.

Do not perform Phase C work that depends on unverified live capabilities while representing it as production-proven.
8. Deliverables and stop condition
Immediate deliverables:
1. Three bounded residual documentation corrections.
2. docs/hackathon-demo-critical-path.md.
3. A concise dependency graph and P0/P1/P2 task breakdown.
4. A list of genuinely blocking owner decisions.
5. Exact Phase B Part 2 GO prerequisites.
6. A demo acceptance checklist with observable evidence for every judge-facing claim.
Do not implement new Phase C/D/E features in this task.

Stop after producing the plan and verifying the documentation corrections.

The next execution step is the separately authorized Phase B Part 2 live gate. It is not implicitly authorized by this directive.

No commits or pushes without explicit owner authorization.
9. Final report
Report:
- Current phase status.
- Exact documentation files changed.
- Confirmation that N8, N5 and count wording are corrected.
- Current demo blockers.
- P0/P1/P2 roadmap.
- Owner decisions still required.
- What is already proven versus fake-backed or live-unverified.
- Explicit confirmation that no live operations were performed.
- Explicit confirmation that Phase B Part 2 remains unauthorized.
Do not produce another general architecture review. We have accepted the foundation. The purpose of this next step is to turn that foundation into a working demonstration.




5. My recommendation
The next milestone should be a real two-way Invoice Review Desk demonstration, not another round of architectural hardening.

The standard I would use is simple:
A business owner should be able to describe the application, answer an AI clarification, see the generated Go/Miniapp application, inspect its evidence, approve it, and watch it run independently—all without touching a terminal.

That is the story judges should remember. The foundation protects that experience; it should not become the demo itself.

The broader multi-tenant lifecycle, metering, quotas, and SSE scaling work can remain on the roadmap for before production expansion, not before this hackathon. 
miniapp-studio-roadmap-refresh-2026-11-25.md

**Yes. We should incorporate the UI design into the roadmap and the implementation prompt now, not defer its architectural decisions until later.** However, we should defer building the complete UI until the appropriate implementation phases.

After studying both files, I see that your HTML is not simply a visual mockup. It defines the product's entire interaction model: how a business owner expresses intent, follows progress, sees decisions, grants approval, and inspects evidence.

I would treat it as the **canonical Miniapp Studio UX specification**.

## **1. What I understand you want**

The core philosophy from `plan_ui_z.md` is excellent:

> The UI is the pipeline. It is a state machine, not a conventional application shell.

Your HTML establishes four permanent interface zones.&#x20;

plan_ui_z.md



**1. Left rail — essentials only**

Gallery

Connectors

Evidence

Settings

Pulse

Bottom: OPS/DEV persona switch and hold-to-confirm emergency stop.

**2. Stage header — the lifecycle spine**

Describe

Spec

Validate

Connect

Deploy

Run

Evidence

Gray means untouched, amber means active, green means proven, red means refused.

**3. Middle pane — one object at a time**

The current application or business task determines what appears. No dashboard-of-dashboards, file-tree clutter, or conventional chat transcript.

**4. Right sidebar — Pulse**

Approvals

Receipts

Alerts

Events

Clarifications

One chronological activity stream, with important decisions actionable in place.

**5. Bottom Line — single-line interaction**

Describe what you want — the studio builds, validates and asks before it acts

User input becomes durable Studio events, not an accumulating wall of chat bubbles.

OPS and DEV are two levels of information density, not two applications or separate navigation systems. Mobile moves Pulse into a slide-over panel.&#x20;

plan_ui_z.md



## **2. How this changes the roadmap**

The UI's seven stages are the **user-facing application lifecycle**, not the engineering roadmap's five phases A–E. We must keep those distinct.

I would integrate the UI into the existing phases as follows:

| Roadmap phase             | UI responsibility                                                                                                    |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| B — Agent Gateway         | Establish real command/event contracts that the UI will consume.                                                     |
| C — Two-way interaction   | Implement the shell, stage spine, Line, Pulse, persistent conversation events, and clarification/answer interaction. |
| D — Artifact verification | Populate Spec, Validate, Connect, and Evidence with actual verified data.                                            |
| E — Deployment            | Complete Deploy and Run with a separate Runtime Sprite and real health checks.                                       |

This is important: the interface should not be an afterthought bolted onto a finished backend. The backend state, event vocabulary, API contracts, and authorization semantics must be designed to support this interaction model from the outset.

## **3. Important details I identified**

There are several things we should preserve or clarify before implementation.

**A. The Pulse needs a first-class clarification interaction.**

Your HTML already handles inline approvals, receipts, alerts and events. But the roadmap also requires an actual agent clarification question answered by the owner in the same session.

I propose a compact clarification event in Pulse with a reply action that focuses the existing bottom Line. It must be distinct from approval: answering a question never authorizes a consequential action.

**B. Green must mean proven, not merely completed.**

The stage spine is powerful precisely because it communicates trustworthy progress. A coding agent saying "done" cannot turn a stage green. The state must derive from authoritative Studio state and actual evidence.

**C. Builder provisioning and application deployment must be distinct.**

The Deploy screen should represent deployment of an explicitly approved application artifact to the separate Runtime Sprite. Builder provisioning is an internal infrastructure operation, not the same thing.

**D. The current HTML is a scripted prototype.**

It uses local state, hardcoded metrics, simulated delays, seeded connectors, fake deployment results and receipts. It explicitly describes itself as having no backend.&#x20;

plan_ui.html

plan_ui.html

plan_ui.html



We should preserve its design, but replace the scripted behavior with real Studio state and actual API results. In particular, never display a simulated approval receipt or a fabricated Fly URL as a real operational claim.

**E. Preserve safety boundaries.**

The current roadmap keeps G-01 unresolved and consequential business writes disabled. The Invoice Review Desk should use synthetic data and demonstrable approval decisions, not actually send financial reminders or perform external financial effects.

I would also explicitly define the kill-switch scope before wiring it to real runs. "Stop all runs" must not accidentally become an unbounded cross-tenant operation.

## **4. Add this to the previous implementation prompt**

This is the amendment I recommend. It makes the UI binding for planning while keeping implementation properly sequenced.

## Binding UX Specification — Miniapp Studio Focus Console

### 1. Design authority

The following files are the canonical UI design source:

- `plans/plan_ui.html`
- `plans/plan_ui_z.md`

The single-page HTML defines the target application layout, visual language, navigation, interaction pattern and lifecycle presentation.

The Markdown document explains the intended UX philosophy and behavior.

Treat these as binding product design specifications, not optional visual references.

Do not replace this design with a conventional SaaS dashboard, chat-first builder, multi-tab application, or generic admin interface.

Do not redesign the UI merely because an alternative pattern is more familiar.

The implementation may improve accessibility, responsiveness, reliability and interaction clarity, but must preserve the essential design philosophy and four-zone composition.

### 2. Locked interface structure

The Studio UI has four permanent zones and one interaction line:

**Left rail**

- Gallery
- Connectors
- Evidence
- Settings
- Pulse
- OPS/DEV persona switch
- Hold-to-confirm emergency stop

Keep the rail minimal and icon-based.

**Stage header**

- Describe
- Spec
- Validate
- Connect
- Deploy
- Run
- Evidence

Represent the current application lifecycle through live state pills.

Use the established meanings:

- Gray: untouched.
- Amber: active.
- Green: proven.
- Red: refused.

Stage navigation must not permit users to bypass authorization, validation, approval or deployment gates. A clickable stage is navigation, not permission to advance a state machine.

**Middle pane**\
Show one meaningful object or scene at a time:

- Gallery.
- Generated MiniappSpec.
- Validation results and refusal evidence.
- Connector catalog and scoped configuration.
- Deployment progress.
- Running application.
- Evidence ledger.

Avoid unnecessary nested navigation, file-tree clutter, and dashboard proliferation.

**Right Pulse**\
Use one chronological stream for:

- Events.
- Clarification questions.
- Approval requests.
- Receipts.
- Alerts.

Approval requests must support inline approve/reject, expiry and durable outcome reporting.

Clarification questions must have a reply action, clearly distinct from approval. Answering a clarification must not grant authorization.

The bottom Line remains the single interaction surface. Replying to a question should place the Line in a contextual reply state bound to the exact question, build and turn.

**Bottom Line**\
Retain the single-line composer as the primary way to describe business intent.

Do not introduce an accumulating conventional chat window. Preserve the complete durable business conversation in Studio and make it inspectable through the designed activity/history interaction.

### 3. OPS and DEV personas

OPS and DEV are two information-density modes over the same underlying data and workflow.

OPS:

- Plain-language descriptions.
- Business consequences.
- Simple status and decision explanations.
- Clear approval and refusal language.

DEV:

- MiniappSpec details.
- Artifact digests.
- Build and turn identifiers.
- Rule identifiers.
- Connector scope and policy information.
- Test and validation evidence.
- Technical event and receipt details.

DEV mode must never reveal secrets, signing keys, bearer tokens, or unrestricted sensitive payloads.

Do not create separate application logic or duplicate navigation for either persona.

### 4. Evidence-driven rendering

Studio is authoritative for lifecycle status.

The UI must not infer successful states from:

- Agent completion claims.
- Local browser state.
- Timed animations.
- Simulated command output.
- An HTTP request being dispatched.
- A deployment command returning before health verification.

Every green state must point to a real, inspectable proof record.

A refused state must identify the actual policy or validation reason.

Pending, unknown and indeterminate states must never be silently represented as success.

The Pulse must render actual persisted events and receipts with replay/reconnect support.

Do not hardcode metrics, URLs, provider status, approvals, receipts, evidence gates or deployment results into production UI.

### 5. Workflow and architecture boundaries

The seven UI lifecycle stages are not the five engineering roadmap phases.

Map them explicitly:

- Describe: durable Studio request and build creation.
- Spec: canonical MiniappSpec generation and version.
- Validate: real validation, fixtures and adversarial refusal evidence.
- Connect: actual connector catalog, organization configuration and scoped authorization.
- Deploy: approved artifact promotion into a separate Runtime Sprite.
- Run: actual Miniapp runtime and verified health.
- Evidence: authoritative evidence ledger and gate provenance.

Builder provisioning and Gateway lifecycle are internal execution infrastructure, not application deployment.

Use the current backend architecture and ownership boundaries. Do not introduce an alternate application framework or runtime.

### 6. UX details requiring explicit implementation design

Before coding, define:

- Clarification event presentation and contextual answer flow.
- Approval versus clarification semantics.
- Pulse ordering, unread/pending counts and reconnect behavior.
- Stage navigation behavior and prohibited transitions.
- Exact scope and authorization of the emergency stop.
- Mobile Pulse slide-over behavior and responsive layouts.
- Empty, loading, stale, error, offline and reconnect states.
- Evidence links from OPS summaries to DEV proof details.
- How users inspect historical decisions without introducing a conventional chat transcript.

Use the HTML prototype's responsive behavior and restrained dark visual language as the baseline.

### 7. Implementation sequence

Do not implement the entire UI in one large frontend project.

Phase C:

- Establish the responsive Focus Console shell.
- Wire the Line to durable Studio conversations.
- Wire the Pulse to persisted events, questions, approvals and receipts.
- Wire the lifecycle spine to real build/turn state.
- Implement OPS/DEV as information-density modes.
- Demonstrate the actual same-session clarification and answer workflow.

Phase D:

- Populate Spec, Validate, Connect and Evidence with real MiniappSpec, artifact and test evidence.
- Implement the minimum artifact review and approval experience.
- Enforce the Go/Miniapp/MiniappSpec contract independently of presentation logic.

Phase E:

- Wire Deploy to the approved digest and separate Runtime Sprite.
- Wire Run to the actual application.
- Display verified URL, health, deployment identity and rollback evidence.

Each scene must progressively become functional as its backend capability is accepted. No screen may fabricate capabilities that do not exist.

### 8. Immediate planning deliverable

Update `docs/hackathon-demo-critical-path.md` with:

1. The canonical UI design contract.
2. A zone-by-zone component map.
3. A scene-by-scene lifecycle map.
4. Backend source-of-truth and event/API dependencies.
5. OPS/DEV information-density rules.
6. Clarification, approval and emergency-stop interaction contracts.
7. Responsive and accessibility requirements.
8. Phase C/D/E implementation sequence.
9. The distinction between existing scripted prototype behavior and actual implemented capability.

Do not implement the full UI in the planning task.

The objective is to make UI design and backend architecture converge before implementation, while delivering the golden path incrementally.

The HTML prototype is the visual and interaction reference; its scripted business behavior is not production authority.

## **My final recommendation**

Lock this UI design now. Do not postpone it.

But don't spend the next several weeks polishing CSS. We should build the actual UI incrementally around the working backend:

- First, make the Focus Console shell communicate real execution state.
- Then make Pulse and Line support genuine two-way interaction.
- Then populate the Spec, Validate, Connect and Evidence scenes with real proof.
- Finally, connect Deploy and Run to the approved Runtime.

**The result should feel like one coherent operating console, not a backend with a pretty frontend attached afterward.**

That is the experience I understand you want to put in front of the hackathon judges.