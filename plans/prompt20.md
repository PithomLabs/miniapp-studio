APPROVED — BOUNDARY A RATIFIED

I accept the adversarial review verdict: **ACCEPTED WITH BOUNDED RESIDUALS.**

Phase C backend C1–C4 is ratified. There are no structural or foundational blockers. The evidence is sufficient: 13/13 packages green, targeted race tests green, formatting/build/static analysis clean, and a genuine browser pass against PostgreSQL 19. The reviewer correctly distinguishes simulated Gateway/agent execution from actual live execution.&#x20;

adv_review_prompt19(1).md

adv_review_prompt19(1).md



### **My decisions on the findings**

| Finding                               | Architectural decision                                                                                 |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| F1 — concurrent senders               | Accept for single-instance local development. Enforce singleton before multi-instance operation.       |
| F2 — unstable assign payload          | Fix before exposing dispatch status in Console.                                                        |
| F3 — terminal-build dispatch/cancel   | Fix before Console integration.                                                                        |
| F4 — terminal while waiting           | Ratify yield-on-assigned with condition: expire outstanding questions and record the trust assumption. |
| F5 — conflicting duplicate visibility | Add safe structured warning logging.                                                                   |
| F6 — legacy continue rows             | Document upgrade/backfill restriction; never silently strand answered work.                            |
| F7 — SSE lifetime/cap                 | Defer; mandatory before shared or multi-user deployment.                                               |
| F8 — mid-run messages                 | Make behavior explicit in Console; no silent implication that the agent received them.                 |
| F9 — terminal dispatch visibility     | Address in C6 through authoritative Pulse events.                                                      |

The yield-on-assigned change is specifically **RATIFIED WITH A BOUNDED CONDITION**, not rejected. The old behavior demonstrably wedged assigned builds.&#x20;

adv_review_prompt19(1).md

IMPLEMENT THIS

## **Next: C6 Focus Console**

My instruction is to begin C6. Do not commission another broad review.

# Owner Authorization — Begin Phase C6 Focus Console

**GO — C1–C4 RATIFIED. C6 IMPLEMENTATION AUTHORIZED.**

## 1. Objective

Implement the Miniapp Studio Focus Console using the locked UI specification:

- `plans/plan_ui.html`
- `plans/plan_ui_z.md`

C6 is the next implementation milestone. The seven-stage lifecycle and Line/Pulse interaction model are binding.

## 2. Mandatory early backend touch-ups

Before exposing the affected backend state as trustworthy Console behavior, address:

- F2: freeze resolved assign prompt bytes in the durable row on first claim. Add red-before/green-after retry-after-new-message regression test.
- F3: prevent dispatch for terminal builds and settle cancel for an unprovisioned build. Add both regression tests.
- F4: expire outstanding questions with a durable cause when a terminal event arrives while waiting for an answer. A subsequent answer must conflict honestly. Record the authenticated-yield trust assumption next to the owning code.
- F5: add structured warning logging for conflicting question IDs without logging payloads or secrets.
- F6: document the legacy continue-row migration restriction and safe backfill/fresh-database requirements.

F1 remains a strict single- Studio-sender-instance operational constraint. Do not claim multi-instance safety. Add database-backed singleton enforcement before any future multi-instance deployment.

F7 remains deferred until shared or multi-user deployment.

These bounded corrections do not require reopening C1–C4 acceptance.

## 3. Focus Console implementation

Implement:

- Seven-stage spine: Describe, Spec, Validate, Connect, Deploy, Run, Evidence.
- Four-zone layout: rail, spine, focused scene, Pulse.
- Single Line composer.
- Chronological Pulse with events, receipts, alerts, clarifications, approvals, and terminal dispatch outcomes.
- First-class clarification reply bound to exact tenant/build/turn/question.
- Human-only approval and rejection.
- OPS/DEV density modes over identical state.
- Tenant-bounded hold-to-confirm stop behavior.
- Responsive layout and keyboard/accessibility behavior.

Use the approved implementation choice: embedded Go templates under `server/templates/focus/` and vanilla JavaScript served from the existing server.

Do not introduce a frontend framework or another application stack.

## 4. Authoritative behavior

Every displayed state must be derived from persisted Studio state.

In particular:

- Show command queued, sending, acknowledged, retrying, failed, and exhausted states truthfully.
- Surface terminal dispatch failures through Pulse using durable events.
- Never imply an agent received a message that was only appended to conversation history.
- A clarification answer must not authorize an application approval.
- Navigation must not bypass lifecycle gates.
- No hardcoded green states, scripted success timers, fabricated URLs, receipts, or evidence.

All approvals remain human-authorized and digest-bound.

## 5. Acceptance

Test:

- Desktop and mobile layouts.
- All seven stages.
- All loading, empty, pending, refused, stale, failed, offline, and reconnect states.
- Clarification and answer concurrency.
- Approval/rejection.
- SSE replay and reconnect.
- OPS/DEV density.
- Keyboard and accessibility.
- Tenant isolation and authorization.
- Accurate dispatch failure visibility.

Use PostgreSQL 19 at `127.0.0.1:5434`, isolated test databases only.

Do not run full suites concurrently with a local Studio server because the test harness can rotate the cluster-global application password.

## 6. Boundaries

- No Sprite CLI invocations, including reads.
- No Fly operations.
- No live OpenCode.
- No Part 2 or C5 live proof.
- No Phase D or E implementation yet.
- No Miniapp/F1/G-01/P5 changes.
- No commits or pushes without explicit authorization.

## 7. Reporting

Complete C6 as a bounded implementation phase. Report exact files, invariants, tests, browser evidence, and remaining fake-backed/live-unverified capabilities.

Stop at Boundary B and request acceptance before starting Phase D.

**Begin C6 implementation. Do not produce another broad architecture review or ask for another GO.**

**Strategic direction:** finish the Focus Console and make Miniapp Studio feel like a real product. Then build the artifact pipeline. We are making progress toward the business-owner demonstration, without paying for cloud infrastructure prematurely.