**Yes. I strongly recommend changing the execution order.** In fact, I think this is the better engineering and financial decision.

We should finish the locally testable Miniapp Studio product before spending money on cloud Builder Sprites.

There is no reason to pay for infrastructure while we are still implementing the application, its interaction model, artifact pipeline, and UI.

One important distinction: installing the `sprite` CLI locally is not the same as installing a local Sprite runtime. Fly describes Sprites as hosted Linux computers; the CLI creates, accesses, and operates those computers. `sprite create` creates a Sprite, while `sprite exec` and `sprite console` operate inside it.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Linux computers for agents+1



Your local installation is still useful. We no longer have the earlier CLI installation blocker. But we can keep it idle and avoid creating cloud resources.

## **My revised architecture and execution decision**

## **Local-first development sequence**

1. **Complete Phase C locally**

   Implement durable continue dispatch, clarification handling, turn binding, stale-answer rejection, atomic handoff, SSE replay, and the Focus Console shell.

   Test Studio against a local Gateway test harness and fake-backed Sprite interactions.
2. **Complete Phase D locally**

   Implement the Invoice Review Desk artifact pipeline: MiniappSpec, Go source, manifest, SHA-256 verification, sandboxed build/test, diff review, approval, and negative tests.

   No Builder Sprite is needed to develop or test these boundaries locally.
3. **Finish the Focus Console**

   Implement all seven stages using the locked `plan_ui.html` and `plan_ui_z.md` design.

   Test actual persisted Studio state, Line, Pulse, approvals, clarification, OPS/DEV density, responsiveness, and accessibility in a local browser.
4. **Local end-to-end rehearsal**

   Run the entire Invoice Review Desk experience using local services, PostgreSQL 19, local builds, and deterministic test doubles.

   No fabricated success states. Clearly label simulated infrastructure and local previews.
5. **Phase B Part 2 — one short-lived cloud Builder**

   Only after local acceptance, provision one Builder Sprite, verify real OpenCode execution, OpenRouter authentication, signed callbacks, identity isolation, and same-session recovery. Destroy it afterward.
6. **Phase E — only when justified by the demo**

   Deploy to a separate Runtime Sprite only when the local product and Builder proof are ready and deployment adds sufficient judging value.

### **What remains mandatory**

Local testing can prove a great deal, but it cannot prove:

- Actual Sprite provisioning and lifecycle.
- Kernel-enforced process identity separation in the cloud environment.
- Real OpenCode credential visibility under the deployed Gateway service.
- Signed callbacks from the real Builder.
- Actual Sprite sleep/wake and orphan recovery.
- Independent Runtime Sprite deployment.

Those remain live acceptance requirements. We are postponing them, not pretending they have passed.

For local development, retain the PostgreSQL 19 baseline at `127.0.0.1:5434`, with isolated test databases and no modification of user data.&#x20;

## **Instructions for the coding agent**

I would replace the previous immediate Part 2 execution priority with this directive.

# Strategic Directive — Local-First Completion of Miniapp Studio

**Architectural decision: APPROVED**

## 1. Change of execution order

The owner has decided to complete and locally validate Miniapp Studio before spending on live Fly.io Sprites.

This directive supersedes the immediate execution priority of the previously authorized Phase B Part 2 GO.

**Phase B Part 2 is now ON HOLD. Do not execute it until the owner explicitly reactivates its live authorization.**

This is a sequencing change, not cancellation of the approved architecture or runbook.

The owner has already installed the Sprites CLI locally. Do not confuse the local CLI installation with a local Sprite runtime. Do not create, provision, or operate cloud Sprites during this development phase.

## 2. Goal

Deliver a locally runnable, business-owner-demonstrable Miniapp Studio implementation, including the locked Focus Console UI and the locally testable backend capabilities.

The golden path remains:

**Invoice Review Desk**

The owner should be able to describe the process, answer an agent clarification, inspect the generated MiniappSpec and Go artifact, review evidence, and make a human approval decision through the Studio interface.

Use synthetic invoice and purchase-order data only.

## 3. Governing architecture — unchanged

Preserve every accepted architectural decision:

- Miniapp is the exclusive application framework.
- Go is the exclusive application implementation language.
- MiniappSpec is the canonical application specification.
- Miniapp framework/runtime is the exclusive Miniapp execution model.
- Studio DB is authoritative.
- Builder and Runtime remain separate execution environments.
- Builder remains dedicated per application workspace.
- OpenCode remains the initial agent adapter.
- Clarification, proposal, and authorization remain separate.
- No direct browser-to-Sprite communication.
- No production connector credentials inside Builder execution environments.
- G-01 remains enforced.
- F1 remains deferred.
- P5 remains Invoice Review Desk.
- No new workflow engine, DBOS, second adapter, alternative application stack, or architectural redesign.

A local test harness is permitted, but it must not become a second production execution architecture.

## 4. Phase C — implement and test locally

Implement the genuine two-way interaction path:

- C1: durable Studio-to-Gateway continue-dispatch sender loop.
- C2: exact question/build/turn binding and stale-answer rejection.
- C3: payload validation, idempotency, and atomic handoff.
- C4: durable conversation history, SSE replay, reconnect, and restart behavior.
- C6: Focus Console shell, Line, Pulse, and seven-stage spine.

C2 and C3 are P0 prerequisites for accepting C5.

Use the existing fake-backed Gateway and test infrastructure for local integration. Where useful, run OpenCode locally against a controlled test environment.

Do not introduce a production local-execution adapter merely to avoid Sprites.

C5, the real live P-G3 same-session proof, remains explicitly pending until the later cloud Builder verification.

## 5. Phase D — implement and test locally

Implement the Invoice Review Desk artifact pipeline:

- Versioned artifact contract.
- Canonical MiniappSpec.
- Go source/module and dependency metadata.
- Manifest and SHA-256 digests.
- Pinned artifact retrieval and integrity verification.
- Safe path, archive, symlink, and size validation.
- Sandboxed Go/Miniapp build and test.
- Machine-readable test report.
- Unified diff and human-readable review.
- Explicit approve/rework decisions.
- Negative test classes required by D4.

Enforce the Go/Miniapp invariant at validation, admission, and build boundaries.

Run build/test workflows locally where the existing architecture permits.

Use deterministic fake Gateway responses for remote boundaries, but test the real validation and build components.

Do not let agent instructions substitute for technical admission controls.

No automatic deployment in Phase D.

## 6. Focus Console — binding UI design

Implement the complete locked UI reference:

- `plans/plan_ui.html`
- `plans/plan_ui_z.md`

Exactly seven lifecycle stages:

**Describe → Spec → Validate → Connect → Deploy → Run → Evidence**

Preserve:

- Left rail.
- Seven-stage spine.
- Focused middle scene.
- Chronological Pulse.
- Single Line composer.
- OPS/DEV density modes.
- Hold-to-confirm stop control.
- Responsive Pulse slide-over.
- Accessibility and reduced-motion behavior.

No conventional central chat interface, dashboard-of-dashboards, file tree, or alternative UI architecture.

Every status must derive from authoritative Studio state and inspectable evidence.

Green means proven. Pending, unknown, refused, and failed states must remain distinguishable.

The HTML prototype is not production authority. Replace its scripted actions, fixtures, fabricated URLs, receipts, timers, and local kill behavior with truthful application behavior.

For capabilities that require a future live Sprite, show an honest not-yet-verified or unavailable state. Never manufacture success to make the interface appear complete.

## 7. Local development and verification

Use the established local environment.

PostgreSQL:

- PostgreSQL 19 only.
- Existing endpoint: `127.0.0.1:5434`.
- Use isolated project test databases/schemas.
- Never destroy, overwrite, or reset user data.
- If PG19 cannot be accessed, stop DB-dependent tests and report the exact blocker.

Run:

- Formatting and static analysis.
- Full Go build.
- Unit and integration tests.
- Race tests where applicable.
- Database migrations and RLS/tenant-isolation tests.
- Local Studio startup.
- Local browser UI testing.
- Full Invoice Review Desk rehearsal.

For UI verification cover:

- Desktop and mobile layouts.
- All seven lifecycle scenes.
- Loading, empty, stale, failure, offline, and reconnect states.
- Clarification answer binding.
- Approval/rejection outcomes.
- Evidence inspection.
- OPS/DEV mode.
- Accessibility and keyboard navigation.
- No fabricated capabilities or secret exposure.

Use Muse Spark and OpenCode as coding assistants where appropriate. Neither may bypass repository constraints or test gates.

## 8. Phase E and live deployment

Do not start Phase E implementation merely to make the interface look finished.

First achieve locally accepted Phase C and Phase D, and rehearse the complete business-owner journey.

Then present an evidence-based recommendation on whether Phase E is necessary for the hackathon demo.

The production Runtime must still be a separate Runtime Sprite. A local preview is not a Runtime deployment and must be labeled accordingly.

## 9. Cloud cost boundary

Until explicit reactivation by the owner:

- No Sprite creation.
- No Sprite provisioning.
- No Fly infrastructure mutation.
- No live OpenCode Builder runs.
- No OAuth ceremony on cloud infrastructure.
- No Runtime Sprite.
- No live external business effects.

The existing `sprite` CLI may remain installed locally, but do not invoke cloud-mutating operations.

## 10. Engineering discipline

Follow the locked Implementation Agent Guidelines.

For every implementation task:

- Trace the full execution path.
- Fix the invariant at the component that owns it.
- Require red-before/green-after regression tests.
- Cover concurrency, retry, crash, replay, and recovery where relevant.
- Fix root causes, not symptoms.
- Keep security enforced technically.
- Distinguish local, fake-backed, and live evidence.
- No commits or pushes without explicit owner authorization.

Do not reopen already accepted architecture without concrete evidence of an invariant violation.

## 11. Deliverables and stopping condition

First produce a revised, dependency-aware execution checklist reflecting this local-first decision.

Then execute the locally implementable Phase C, Phase D, and Focus Console work in bounded increments.

At each acceptance boundary provide:

- Files changed.
- Invariants implemented.
- Tests executed with exact results.
- Local browser evidence.
- Remaining fake-backed and live-unverified requirements.
- Next dependency.

The local completion milestone is reached when the Invoice Review Desk golden path can be rehearsed locally through the Focus Console with real Studio persistence, real artifact verification/build/test, real approval records, and honest evidence.

Stop before Part 2, C5 live P-G3, or Phase E live deployment until the owner separately reactivates those gates.

**This directive changes sequencing, not the product architecture. The immediate objective is a dependable local Miniapp Studio demonstration without cloud Sprite spending.**

## **My final decision**

Local-first strategy approved

This is the most sensible use of resources now.

We should spend engineering effort on the product that judges and business owners will actually see—not on repeatedly provisioning infrastructure while the product is unfinished.

The target is simple: **make Invoice Review Desk work locally from beginning to end, make the Focus Console genuinely functional, then use one short-lived Builder Sprite to prove the cloud boundary.**

That gives us a much stronger demonstration and avoids paying for cloud infrastructure during ordinary development.