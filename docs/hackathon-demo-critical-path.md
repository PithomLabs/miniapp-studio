# Miniapp Studio — Hackathon Demo Critical Path

**Status:** planning document (prompt15 §5). No Phase C/D/E code is
implemented by this document; no live operations are authorized by
it. Phase B Part 2 requires a separate explicit GO.

**Baseline (confirmed from repo evidence; see prompt15_plan U0):**
Phase A accepted; Phase B Part 1 accepted with bounded residuals
(N1/N4/R-DB/R-NET closed, P1–P4 green, N2–N10 tracked,
`TestAdapterStatus` pre-existing test-only flake); Part 2, Phase
C/D/E live proofs not executed.

**Source limitation (owner-confirmed):**
`miniapp-studio-roadmap-refresh-2026-11-25.md` does not exist in
the tree and is not cited as a decision source. Baseline here is
prompt15 §1–3 + `plans/roadmap4.md` (original Nov 25 status) +
accepted reports/evidence, newest accepted decision winning on
conflict. This fallback must not be treated as the authoritative
roadmap for a GO decision without explicit owner confirmation.

**Golden path:** Invoice Review Desk, single demonstration. Synthetic
or fixture invoice data only; simulated approval effects (G-01
unresolved — no production accounting systems, no consequential
business writes).

**Status taxonomy** (every step below carries one label per
surface; backend and Focus Console tracked separately):

- `implemented` — in-tree, tested, accepted.
- `fake-backed` — in-tree and tested, but only against fakes,
  doubles, or scripted transcripts. Never live evidence.
- `legacy-ui` — exists only in the P4 server-rendered shell, to
  be superseded by the Focus Console.
- `live-unverified` — implemented but never exercised against
  real infrastructure.
- `missing` — not implemented anywhere.

## A. Demo narrative (13 steps)

1. **Owner describes the Invoice Review Desk requirement.**
   Backend: `implemented` (`conversation.create`,
   `message.send`, tools + tests). Console Line: `missing`.
2. **Studio creates the conversation and build.** `implemented`
   (`conversation.create`, `build.create`, tenant isolation).
3. **Gateway routes work to the correct Builder.**
   `fake-backed` + `live-unverified` (`build.provision`,
   per-application builders, signed commands; Part 2 proves it).
4. **OpenCode performs real supervised work.** `fake-backed` +
   `live-unverified` (supervisor, adapter, translator tested
   vs doubles; no live agent run yet).
5. **Agent requests a clarification.** `fake-backed` +
   `live-unverified` (file-drop consume → `build.ingest_event`
   → question records + turn fencing; native tool denied at
   v1.19.6, live P-G3 pending).
6. **Studio persists and renders the question in the original
   conversation.** Persist: `implemented`. Render:
   `legacy-ui` (`conversation.html`) / Console Pulse: `missing`.
7. **Owner answers.** `implemented` (`question.answer` with
   conflict handling).
8. **The same OpenCode session consumes the answer and
   continues.** `missing` (no Studio→Gateway continue-dispatch
   sender loop exists — verified absent) + `live-unverified`
   (gateway resume path). **This is the Phase C crux.**
9. **Agent produces a Go/Miniapp artifact and MiniappSpec.**
   Spec validation: `implemented` (`spec.validate`,
   runner admission). Agent-produced artifact pipeline:
   `missing`.
10. **Studio verifies the artifact and tests.** Spec/admission
    checks: `implemented`. Manifest/digest verification at
    retrieval, Go build/test execution, machine-readable test
    report: `missing`.
11. **Owner reviews changes and approves or requests rework.**
    Decision primitives: `implemented` (Miniapp runtime
    approval support, P0-tested). Artifact review/approval
    experience: `missing` (conversation-level approval
    rendering is `legacy-ui` only).
12. **Approved artifact deploys to a separate Runtime Sprite.**
    [iff Phase E is selected for the judging demo — owner decision]
    `fake-backed` (`deploy.provision` P3-era app deploy) +
    `missing` Phase E requirements (approved-digest pinning,
    no-rebuild transfer, health-before-live, rollback).
13. **Studio records deployment and verifies health.**
    [iff Phase E is selected for the judging demo — owner decision]
    `fake-backed` (`deploy.status`) + `missing` (real health
    verification and Runtime recording).

## B. Phase B Part 2 — owner-run live gate

Uses `docs/gateway-phase-b-live-runbook.md` unchanged. Nothing
below executes without a separate explicit owner GO.

Genuinely blocking owner decisions (from runbook §1/D1–D6):
Fly organization (D1); model provider + auth (D2); Studio
callback reachability — tunnel vs Fly-hosted Studio (D3, the
largest unknown: the sprite must POST callbacks to a reachable
Studio); pin re-verification policy on drift (D6). Non-blocking
but required before cleanup: Builder retention (D4), system
packages (D5, discoverable live).

Evidence checklist (observable pass criteria in the runbook):
Builder provisioning (sprite URL, attach convergence, §6 +
staging checks); Gateway service boot + effective identity
(§7a EUID derivation); credential-drop self-test (`boot
refused` absent from logs; agent uid ≠ service uid);
signed status round-trip (§9 latency recorded); OpenCode no-op
execution (§10 callbacks in order); key-read denial (§7b
EACCES, exit codes + stderr); dual-user Q/A access (§7b
matrix); Studio callback delivery (ordered, deduped);
sleep/wake behavior (§11 warm/cold + F-R2 PID absence);
cleanup/retention proof (§13 empty-list or retention record).

## C. Phase C — minimum credible two-way interaction

Acceptance is not displaying a question: the owner's answer
must resolve the actual pending question in the same OpenCode
session, observably. Mechanism is file-drop only (native tool
denied at v1.19.6; no invented flags, no prose-as-metadata,
no second mechanism).

Concrete tasks:

- **C1 — Continue-dispatch sender loop (the missing piece).**
  Studio-side durable dispatch of authorized continues to the
  Gateway: per-build ordering, retry with the same signed
  command id (gateway dedup replays), failure surfacing.
  Reuses: `question.answer` records, gateway `continue` +
  turn fencing + claim replay, outbox/inbox patterns.
  **Implemented (prompt18 U1, local-first):** `server/sender.go`
  + gateway stable-id/errors contract; 21 chain tests.
  Evidence: [Boundary A](prompt18-boundary-a.md).
- **C2 — Question/turn binding + stale-answer rejection.**
  Answer accepted only for the currently open
  (build, turn, question_id), authorized against the owning
  tenant (reuse `question.answer` ownership checks);
  superseded answers conflict honestly (`answer_conflict`
  precedent exists in legacy UI).
  **Implemented (prompt18 U2, local-first):** submit-time
  turn freeze + supersede/cancel/race rejection.
  Evidence: [Boundary A](prompt18-boundary-a.md).
- **C3 — Payload validation + atomic handoff.** Strict
  answer schema; exactly-once hand-to-gateway semantics
  (idempotency key per intentional answer, fingerprint-bound
  per prior GO).
  **Implemented (prompt18 U2, local-first):** assigned-join
  handoff (needs owner ratification — contract change),
  sender flip, terminal callback mapping.
  Evidence: [Boundary A](prompt18-boundary-a.md).
- **C4 — SSE replay + restart/reconnect.** Reuse
  `conversation.stream` + D1 `conversation_stream`; prove
  reconnect replays the question/answer/decision history.
  **Implemented (prompt18 U3, local-first):** kill-and-resume,
  backlog drain, concurrent no-gap proofs + a real-browser
  pass; fixed a send-vs-ingest deadlock at the boundary.
  Evidence: [Boundary A](prompt18-boundary-a.md).
- **C5 — Live P-G3 proof.** Real agent yields via file drop,
  owner answers in Studio, same native session continues and
  references the answer; honest failure when a session cannot
  resume (409/500 paths, no fabricated resume).
- **C6 — Focus Console shell + Line + Pulse + spine** wired
  to the above (see §UI items 2–4, 6); clarification reply
  binds Line to exact question/build/turn; answering never
  authorizes (see §UI item 6).

## D. Phase D — minimum credible artifact pipeline

Invoice-Desk-only vertical slice. Required foundation (each a
deliverable with tests): versioned artifact contract;
MiniappSpec valid + canonical; Go source/module +
dependency metadata; manifest + SHA-256 digests; fetching
only the pinned digest; sandboxed build, test, inspect, and
evidence pipeline.

- **D1 — Retrieval + verification.** Fetch via signed
  `fetch_artifact`; verify digests, paths (no traversal),
  archives (no zip-slip / nesting abuse), symlinks (refuse
  or confine), sizes, and the supported Go/Miniapp
  dependency allowlist before anything executes.
  Reuses: gateway collector patterns, `spec.validate`.
- **D2 — Build + test execution.** Recommended locus:
  Studio-side fetch-verify-build reusing `runner/` +
  `spec/`, sandboxed (rejected alternative: Builder-side
  execution, which couples build trust to the agent
  environment and needs toolchain changes on the sprite
  image). Machine-readable test report required.
- **D3 — Review + explicit approval/rework.** Unified diff,
  human-readable summary, evidence links; approval records
  digest + approver; rework returns to the Builder with
  reasons. No auto-deploy from D state.
- **D4 — Negative tests (all required):** non-Go
  implementations, alternative frameworks/runtimes, invalid
  MiniappSpec, modified bytes, wrong digests, malicious
  paths/symlinks, oversized artifacts, failing tests,
  secret-bearing human-visible output.
- **D5 — Console Spec/Validate/Connect/Evidence scenes**
  populated from the above (see §UI item 8).

Contract enforcement lives at validation/admission/build
boundaries; agent instructions alone are insufficient. No
marketplace, no generalized plugins, no deployment inside D.

## E. Phase E — approved Runtime deployment

Minimal separate-Runtime path (include in the judging demo
only if scope/time permit — owner decision; never bypass
approval):

- **E1 — Approval + digest recorded** (from D3; the only
  deployable unit).
- **E2 — Runtime provisioned separately from Builder**
  (distinct sprite, distinct lifecycle; Builder never runs
  the app).
- **E3 — Verified artifact transferred without rebuilding
  from mutable Builder state** (pinned digest end to end).
- **E4 — Miniapp framework/runtime executes the app; no
  coding agent or Builder Gateway in Runtime.**
- **E5 — Health check before live; rollback to a previously
  approved digest; Runtime identity/URL/digest/health
  recorded in Studio.**
- **E6 — Console Deploy/Run scenes** wired to the approved
  digest, verified URL, health, and rollback evidence.

Gap work vs existing `deploy.provision` (P3-era, fake-backed):
approved-digest gating, no-rebuild transfer, real health
verification, rollback, Runtime recording. Deployment is
never automatic.

## F. Demo safety and governance

Preserved throughout: no production connector bindings in
Builder Sprites; no consequential business writes while G-01
is unresolved (synthetic/fixture invoice data; simulated
approval effects); human authorization distinct from agent
proposals (answering ≠ approving); tenant/application
isolation; Studio as authoritative history; Builder/Runtime
separation; no direct browser-to-Sprite connections; no
Miniapp validation/admission bypass. Approval screens
demonstrate human decisions without executing external
financial effects.

## G. Priorities (P0/P1/P2)

- **P0 — demo blockers and Phase B exit:** F-R1/F-R2/F-R3
  closure; Part 2 live gate green; Phase B exit gate green:
  M5-SSE bounded stream lifetime + explicit concurrent-stream
  budget, and serve-time `STUDIO_GATEWAY_BINARY` wiring; C1
  sender loop + C2 question/turn binding and stale-answer
  rejection + C3 payload validation and atomic handoff, all
  passing before C5 same-session proof is accepted; D1–D4
  artifact pipeline + review/approval (including every D4
  negative refusal class); Console shell/Line/Pulse/spine wired
  to real state; E1–E6 only iff Phase E is selected for the
  judging demo (owner decision).
- **P1 — post-P0 reliability and demo polish:** C4 SSE
  replay/reconnect, OPS/DEV density modes, kill-switch scoping.
- **P2 — post-demo (do not build):** fleet lifecycle, broad
  quotas, billing/metering, generalized admin, fleet-scale SSE
  scaling beyond the M5-SSE stream-lifetime/concurrency gate.

Excluded: second agent adapter, other frameworks, workflow
engines, task canvas, marketplace/billing, Lead-to-Booking,
new architecture without demonstrated need. Accepted
architecture preserved.

## H. Dependencies and sequence

```
U0 baseline → F-R1/F-R2/F-R3 → [separate GO] → Part 2 live gate
  → Phase B exit gate (M5-SSE: bounded stream lifetime +
    explicit concurrent-stream budget; serve-time
    STUDIO_GATEWAY_BINARY wiring)
  → Phase C (C1 + C2 + C3 → C5 live proof; C4 replay/reconnect;
    C6 console shell)
  → Phase D (D1→D2→D3→D4, including all negative refusals;
    D5 scenes)
  → Phase E (E1→E5; E6 scenes)
    [iff Phase E is selected for the judging demo — owner decision]
  → clean-state rehearsal → demo
```

Gates: Phase C cannot start until the Phase B exit gate passes;
C5 cannot be accepted until C1, C2, and C3 pass; D4 negative
refusal tests are part of the P0 Phase D gate. Phase E is
included only by explicit owner scope decision. No phase starts
on unverified capabilities represented as proven; no live ops
before the separate GO; no skipping gates to shorten schedule.

## UI. Binding UX specification — Focus Console

`plans/plan_ui.html` + `plans/plan_ui_z.md` are the locked,
binding spec (approval clarification 2): seven-stage
lifecycle and Line/Pulse interaction model preserved; no
SaaS-dashboard/chat-builder/admin-interface substitution;
accessibility/responsiveness/clarity improvements allowed.

1. **Design contract:** four zones + Line, state-machine
   semantics, evidence-driven rendering (green = proven
   with inspectable proof; red names the policy/reason;
   pending/unknown/indeterminate never shown as success).
2. **Zone map:** rail (Gallery, Connectors, Evidence,
   Settings, Pulse, OPS/DEV, hold-to-confirm stop); spine
   (7 pills, navigation ≠ permission); middle (one scene);
   Pulse (chronological event/clarification/approval/
   receipt/alert stream); Line (sole composer; submits
   become events).
3. **Scene map:** gallery → spec → validate → connect →
   deploy → run → evidence (+refused/settings), one object
   at a time; Backend mapping in prompt15 §5 items 2–5.
4. **Source-of-truth deps:** Studio state + persisted
   events/SSE for Pulse/Line/spine (C); MiniappSpec/
   artifact/test evidence for Spec/Validate/Evidence (D);
   approved digest + Runtime health for Deploy/Run (E).
5. **OPS/DEV rules:** density modes over identical data and
   flow (DEV adds digests/ids/rules/evidence); DEV never
   reveals secrets/keys/tokens; no duplicated logic.
6. **Interaction contracts:** clarification = Pulse card +
   reply action focusing Line bound to exact
   question/build/turn, distinct from approval (answering
   grants no authorization); approval = inline
   approve/reject + expiry + durable outcome; kill switch =
   hold-to-confirm with explicitly scoped, tenant-bounded
   stop semantics defined before wiring.
7. **Responsive/a11y:** 900px Pulse slide-over, reduced
   motion, focus-visible, aria-live stage, empty/loading/
   stale/error/offline/reconnect states for every scene.
8. **Sequence:** C shell+Line+Pulse+spine → D scenes+
   review/approval → E Deploy/Run; each scene goes live
   only with its accepted backend.
9. **Prototype vs real:** the HTML is visual/interaction
   reference only — timed sleeps, RUNDEMO fixtures,
   fabricated URLs/receipts, seeded events, wizard, kill
   behavior, hardcoded gates are scripted, never
   production authority; no screen may fabricate
   capabilities.

## GO prerequisites (Part 2)

Separate explicit owner GO required; this document never
implies it. Prerequisites: F-R1/F-R2/F-R3 closed; D1/D2/D3
decided; pins re-verified per runbook §2; pre-live gates
green per §3; owner accepts the N2 upgrade prohibition and
N8 re-provision constraint for the live window.

## Acceptance checklist (claim → observable evidence)

- Judge describes need in Line → durable conversation +
  build records exist (queryable, replayable).
- Part 2 live gate passes → every owner-run observation in
  §B's evidence checklist is recorded, with cleanup/retention
  proof; separate GO is present.
- Phase B exit gate passes → M5-SSE maximum stream lifetime is
  bounded and its concurrent-stream budget is explicit and
  tested; serve-time `STUDIO_GATEWAY_BINARY` reaches the
  Gateway serve Options path and the serve-path test passes.
- Clarification asked → question persisted + rendered in
  original conversation (Pulse card bound to turn).
- Owner answers → C1 dispatches the authorized continue; C2
  binds the answer to the open build/turn/question and rejects
  stale answers; C3 validates the payload and proves atomic,
  idempotent handoff; only then does the same native session
  continue and reference the answer (session id + transcript
  evidence, live P-G3).
- Artifact produced → manifest + digests + valid
  MiniappSpec, all verified (D1 evidence).
- Tests pass → machine-readable report green (D2); every D4
  negative class demonstrably refused as part of the P0 gate.
- Owner approves → digest + approver + timestamp recorded;
  rework path demonstrated (D3).
- [iff Phase E is selected for the judging demo — owner decision]
  App runs independently → separate Runtime Sprite serves
  traffic; health verified before live (E5).
- [iff Phase E is selected for the judging demo — owner decision]
  Rollback works → previous approved digest restored and
  serving (E5).
- No terminal touched by the judge; no fabricated URL,
  receipt, metric, or green state anywhere in the demo.
