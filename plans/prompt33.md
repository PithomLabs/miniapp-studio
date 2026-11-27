I recommend making the target the **complete Roadmap A — Engineering Studio (Focus Console), B0–B5**, delivered as one coherent UI experience.

This does not include building the separate Roadmap B Automation Operations Studio. The ultimate roadmap explicitly separates those experiences while sharing their underlying platform. It also distinguishes UI completeness from backend functional maturity: planned capabilities must be visible, but never falsely represented as operational.

Here is the implementation prompt.

# Mission: Deliver the Complete Miniapp Studio Engineering UI (B0–B5)

You are the senior product engineer, UI architect, and implementation owner of `PithomLabs/miniapp-studio`.

Your mission is to deliver the **complete, polished, coherent Miniapp Studio Engineering UI — Roadmap A, B0 through B5 — in one execution cycle.**

This is an implementation assignment, not a planning assignment. Investigate the repository, determine the necessary work, make the implementation decisions, implement the UI, test it, conduct an adversarial review, correct genuine defects, and deliver the finished result.

Do not stop after producing a plan or design document. Do not request intermediate approval.

## 1. Governing authority

Use this precedence:

1. Latest explicit Project Owner decisions.
2. `plans/miniapp-studio-ultimate-roadmap.md` — canonical overarching roadmap.
3. Approved architectural decisions, ADRs, and engineering contracts.
4. Existing implementation and task-specific documentation.

Read the roadmap, especially its Roadmap A / Roadmap B separation, B0–B5 milestones, Focus Console architecture, environment definitions, UI maturity matrix, security model, and governance rules.

Inspect the actual current implementation before making assumptions. The roadmap's status descriptions are a starting point, not proof of current repository behavior.

## 2. Target product experience

Deliver the complete Roadmap A Engineering Studio experience using the existing Focus Console architecture and established design language.

The UI should feel like one integrated professional software engineering control plane, not a collection of disconnected CRUD pages.

Preserve the existing architectural conventions and UI technology. Do not introduce a competing frontend framework or rewrite the application architecture merely for stylistic preference.

Implement and integrate the following experiences:

**B0 — Development task management**

- Conversations and development tasks.
- Agent questions, user answers, and approval interactions.
- Task status, activity, history, and evidence.
- Clear distinction between human authorization and agent execution.

**B1 — Application development**

- Application and MiniappSpec management.
- Specification editing, validation, and relevant test results.
- Local development workflow and Git awareness.
- Development and deployment configuration.
- Honest representation of supported development operations.

**B2 — Remote Builder and agent experience**

- Agent status, attempts, progress, checkpoints, handoffs, and recovery.
- Kill, resume, and continuation controls where genuinely supported.
- Connector discovery, management, configuration, and custom connector wizard.
- Connector testing and authorization state.
- Clear separation of Builder Workspace Preview from Application Preview.
- Surface real handoff and lifecycle evidence where available.

**B3 — Application Preview**

- Preview environments and their lifecycle.
- Per-PR preview identity, commit/SHA, application URL, health, expiration, and status where supported.
- Deployment and retirement status.
- Engineering PR Preview and customer Application Preview must not be conflated.
- Clearly indicate unsupported or not-yet-connected preview capabilities.

**B4 — Troubleshooting and observability**

- Structured logs and meaningful filtering.
- Environment, deployment, commit, and digest context.
- Incident and failure investigation experience.
- Relevant events, diagnostic evidence, recovery status, and next actions.
- Do not represent an activity feed as a complete incident-management system.

**B5 — Staging and Production**

- Environment overview and release status.
- Candidate artifact identity and digest.
- Migration and health status where available.
- Approval and authorization boundaries.
- Release history, promotion, and rollback evidence.
- Visibly distinguish planned release controls from executable ones.

## 3. Cohesive UI architecture and quality

The complete experience should follow the Focus Console design principles:

- Consistent navigation and information architecture.
- Clear development lifecycle and stage progression.
- Contextual work area appropriate to the selected task or application.
- Useful activity, evidence, and status presentation.
- Consistent interactions, terminology, visual hierarchy, and responsive layout.

Design and implement the required screens, dialogs, detail views, navigation paths, and empty/loading/error/success states.

The UI must be polished and usable, not merely a set of screens that happen to render.

Pay particular attention to:

- Responsive behavior and accessible keyboard navigation.
- Clear visual feedback and focus states.
- Consistent typography, spacing, components, and status semantics.
- Useful empty states and actionable error messages.
- Avoiding redundant navigation and information duplication.
- Preventing accidental consequential actions.

## 4. Functional integrity — non-negotiable

UI completeness and functional completeness are separate dimensions.

Connect controls to real existing backend capabilities wherever they are supported. Add the minimum necessary backend integration only when justified by the existing architecture and authorized scope.

For capabilities that are not implemented or not connected:

- Render the intended UI experience.
- Label it accurately as Planned, Partial, Blocked, or Not Connected.
- Explain what is missing where useful.
- Disable unsupported actions and explain why.
- Never fabricate activity, history, metrics, deployment status, approvals, or successful operations.
- Fixtures must be unmistakably labeled as examples.

A polished false representation is a defect.

The UI must never imply that an operation is authorized merely because a button is visible.

Server-side authorization remains authoritative. UI restrictions supplement but never replace backend authorization.

## 5. Architectural and security guardrails

Preserve the ultimate roadmap's decisions:

- Go remains the exclusive implementation language and Miniapp remains the exclusive application framework.
- Neon PostgreSQL is mandatory everywhere. No Docker PostgreSQL, self-hosted PostgreSQL, Testcontainers, or local PostgreSQL substitutes.
- Preserve verify-full TLS, RLS, tenant isolation, and fail-closed authorization.
- Never introduce ambient cross-tenant access.
- Tenant identity and workspace context must be explicit.
- Do not finalize unresolved tenancy or deployment-topology decisions by assumption.
- Keep Roadmap B Automation Operations Studio separate. Do not duplicate its IA or build its unapproved features as part of Roadmap A.
- Preserve the distinction between Fly Sprites and Fly Machines.
- Do not introduce DBOS, another workflow engine, a second application framework, or speculative infrastructure.
- No secrets in templates, HTML, logs, browser-visible payloads, or evidence.
- No fake functionality, bypassed authorization, or weakened security invariants to make the UI appear complete.

The existing backend and contracts remain the source of truth.

## 6. Autonomous execution

You own the implementation process.

Internally decompose the mission into the necessary workstreams and sequence them according to actual dependencies.

You are authorized to make ordinary, reversible implementation decisions within the existing architecture.

You should:

- Investigate existing functionality before changing it.
- Reuse working components and handlers.
- Fix root causes rather than symptoms.
- Avoid unnecessary migrations and backend rewrites.
- Build reusable components instead of duplicating implementations.
- Complete the UI across B0–B5 in a coherent pass.
- Run tests continuously and correct failures.
- Update relevant documentation only where necessary.

Do not produce a separate implementation-plan document for approval. Do not interrupt the owner with routine implementation questions.

Make reasonable reversible assumptions and document them.

Stop and request owner input only when a consequential authorization is required, an unresolved architectural decision materially changes the product, an irreversible action is proposed, or a genuine blocker cannot safely be resolved.

When a blocker affects only one capability, continue with the rest of the deliverable and represent the blocked capability honestly.

## 7. Testing and verification

Quality is part of the deliverable, not an optional follow-up.

Use the repository's established development and testing workflow.

At minimum verify:

- Formatting, build, and static analysis.
- Relevant Go package and server tests.
- All affected routes, handlers, templates, and interactions.
- Authorization and tenant-scoping behavior.
- Loading, empty, error, partial, and success states.
- Regression coverage for changed behavior.
- Navigation and cross-screen consistency.
- Responsive and accessibility behavior using available tooling.
- No broken links, dead controls, fabricated statuses, or console/server errors.
- No regression in existing accepted B0/B1 functionality.

Do not silently skip a failed or unavailable test. Record actual commands, outcomes, and limitations.

Respect the existing Neon-only test contract and bounded-test rules. Do not provision cloud resources or create database environments without authorization.

## 8. Adversarial review — required before delivery

After implementation, perform a distinct adversarial review. Prefer a separate reviewer or agent with access to the resulting code and acceptance criteria, not just your own implementation rationale.

Review the complete change set as a skeptical senior architect and security reviewer.

Challenge:

1. Does the UI faithfully implement the Roadmap A B0–B5 experience?
2. Are any screens inconsistent, incomplete, disconnected, or misleading?
3. Are there fake controls, fabricated data, or unsupported actions?
4. Can tenant boundaries or authorization checks be bypassed?
5. Are secrets or sensitive information exposed?
6. Have existing workflows or accepted functionality regressed?
7. Are lifecycle and environment distinctions represented correctly?
8. Are tests proving real behavior rather than merely rendering templates?
9. Has unnecessary architecture or infrastructure been introduced?
10. Does the implementation remain consistent with the ultimate roadmap?

Classify findings:

- **P0/P1 — blocking:** security, data integrity, authorization, serious correctness, architectural violations, or critical acceptance failures.
- **P2 — material:** significant usability, reliability, or missing functionality.
- **P3 — non-blocking:** refinements and future improvements.

Fix all substantiated P0/P1 findings and material P2 findings within scope. Add regression tests. Do not spend cycles endlessly debating subjective P3 preferences.

If an independent reviewer is unavailable, disclose that limitation and perform a separate fresh-context adversarial pass. Never misrepresent self-review as independent review.

## 9. Owner authorization boundaries

This assignment authorizes local implementation, tests, UI integration, and documentation within the existing repository.

It does NOT authorize:

- Fly provisioning or cloud mutations.
- Sprite creation or destruction.
- OAuth grants or provider authentication.
- Neon project/branch provisioning or destructive database operations.
- Staging or Production deployment, migration, promotion, or rollback.
- Changes to production credentials or permissions.
- Consequential business-effect execution.
- Git commits, pushes, merges, or releases.

Stop for explicit owner authorization before any such action.

No permission is inferred from this assignment, the roadmap, or an implementation recommendation.

## 10. Definition of done and final delivery

Deliver one cohesive Engineering Studio UI implementation covering B0–B5.

Completion requires:

1. The complete Roadmap A navigation and screen experience exists.
2. Existing functional capabilities are genuinely integrated.
3. Future or unavailable functionality is accurately represented.
4. The UI is consistent, responsive, accessible, and professionally polished.
5. Security, tenancy, and authorization boundaries are preserved.
6. Relevant tests pass, with actual evidence.
7. The adversarial review is complete and all blocking findings are resolved.
8. No unauthorized infrastructure changes or repository publication occurred.

Final report must include:

- Executive summary of what was delivered.
- B0–B5 implementation and functional maturity matrix.
- Important screens and workflows completed.
- Files and architectural areas changed.
- Tests executed and actual results.
- Adversarial findings, fixes, and remaining non-blocking issues.
- Capabilities that remain Planned, Partial, Blocked, or Not Connected.
- Owner decisions or external dependencies that genuinely remain.
- Any actions deliberately not performed because authorization was required.

Provide screenshots or other genuine visual evidence where the existing tooling permits.

**The ultimate objective is a complete, coherent, useful Miniapp Studio Engineering UI — not a collection of prototypes, mockups, or disconnected pages. Work autonomously, preserve the architecture, prove the result, and deliver it in one execution cycle.**

**My intended outcome:** one substantial implementation delivery, not another planning/review loop. The agent can work through multiple internal implementation stages, but it should come back only with the completed UI, actual test evidence, and adversarial review findings—or a genuinely consequential owner decision.

One important distinction: this completes the **UI experience across B0–B5**, not automatically the underlying B0–B5 backend roadmap or the separate Roadmap B Operations Studio. That prevents a polished interface from being mistaken for capabilities the platform has not yet implemented.