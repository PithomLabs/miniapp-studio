# Plan: implement `plans/prompt33.md` (Roadmap A Engineering UI, B0–B5)

Plan-only. No implementation, no live ops. Execution starts only on explicit owner GO.

## 1. Authority reconciliation (decided, not asked)

Prompt33 demands autonomous one-cycle delivery with no intermediate approval; standing rules withhold
live/infra/commit powers from any such grant. Both are satisfiable because prompt33 §9 withholds those
same powers itself. Reconciliation:

- Autonomy covers ALL implementation decisions, sequencing, and reversible assumptions (prompt33 §6).
- Three things still stop for the owner, exactly as prompt33 §6/§9 already require: consequential
  authorization, irreversible actions, genuine blockers — plus one forced exception below.
- FORCED EXCEPTION (sandbox reality, not preference): the agent shell has no Neon DSN and must never
  seek credentials, so live-DB verification cannot run agent-side. All live testing batches into at most
  two owner-terminal runs (exact commands prepared, logs preserved). This is the only planned interruption.

## 2. Current state (researched, not assumed)

- B0/B1: Implemented + Functional (accepted scope) — preserve, regression-guard, do not rebuild.
- B2 console + connectors: Implemented UI, Partial function (B2-01A accepted, live proofs pending Card 002).
- B2 Workspace Preview + B3: partial templates only, Planned function, no backend (T3 gated).
- B4/B5: no routes — design + render as Planned/Not Connected with disabled, explained actions.
- Two surfaces coexist: top-level screens (`server/templates/*.html`) + Focus Console scenes
  (`server/templates/focus/*.html` + `console.js`), 43 routes. Unifying these into one IA is a workstream,
  not a given.
- Tests: Go `httptest` pattern (`screens_test.go`, `conversation_ui_test.go`…) + Python `e2e/` harnesses.
  NO browser tooling installed (no chromium/playwright) — see §5.

## 3. Workstreams (dependency order; internal evidence, no owner gates between them)

- W0 recon: route × template × handler × backend-capability matrix; per-screen UI/functional maturity
  baseline from code (roadmap matrix is the starting claim, code is the proof).
- W1 shell: one IA — nav, layout, lifecycle stage progression, terminology; zero dead links/controls.
- W2 B0/B1: preserve accepted scope; complete state coverage (loading/empty/error/success); regression tests.
- W3 B2: console + connectors + Workspace Preview integration; wire kill/resume/continue/preview only where
  genuinely supported; honest Partial everywhere else (Card 002 pending stays visible).
- W4 B3: preview identity/SHA/URL/health/expiry/retirement scenes; wire what the backend has (expected:
  little — T3 gated), render the rest Planned with reasons.
- W5 B4/B5: new scenes per §2 of prompt33 — full IA and states, all actions disabled-explained except any
  genuinely readable data (e.g., activity-derived context explicitly labeled as such, never as incident
  management or release control).
- W6 polish: responsive, keyboard nav, focus states, typography/spacing/components, no-duplication pass,
  anti-accident confirmations on consequential controls.
- W7 tests + evidence: httptest per route × state × authz × tenant-split; honesty tests (no enabled
  control without backend; no fabricated statuses — assert the labels); regression for changed behavior;
  batched live runs (≤2).
- W8 adversarial review: separate-agent fresh-context pass (subagent/workflow) against prompt33 §8's ten
  challenges; fix substantiated P0/P1 + in-scope P2 with regression tests; disclose review-independence
  limits honestly.
- W9 delivery: uncommitted tree + final report per prompt33 §10 (summary, maturity matrix, screens, files,
  test evidence, findings/fixes, Planned/Partial/Blocked/Not-Connected list, remaining decisions,
  deliberately-unauthorized actions).

## 4. Backend integration rule

Wire only real capabilities. Additions only if minimal, reversible, justified by existing architecture —
and any surprise (a needed migration, a tenancy/topology decision, D-5) becomes an owner decision, never
an assumption. Default expectation: zero migrations, zero new contracts.

## 5. Visual/responsive verification (tooling fallback)

Attempt a time-boxed headless-browser spike first (install chromium/playwright; if the sandbox refuses,
stop — no escalation). On failure: DOM/state assertions in httptest + static checks + owner visual
acceptance of the running UI as the polish gate, recorded as a limitation. Screenshots only from genuine
runs.

## 6. Stop conditions

Prompt33 §6 four (consequential auth, architectural/material-product, irreversible, genuine blocker) +
sandbox-forced live runs + any N2/F1/G-01-class invariant threat. Per-capability continuation otherwise;
blocked capabilities rendered honestly, never faked or silently dropped.

## 7. Risks

- Single-cycle scope is large: contained by preserve-B0/B1 + render-Planned-B4/B5 (honest, not hollow).
- B2 backend proofs pending (Card 002) caps B2 at Partial — represented, not hidden.
- Owner-run latency on the two live batches — mitigated by exact pre-written commands.
- No-browser fallback weakens responsive/a11y proof — disclosed, owner-visual backstop.

## 8. Definition of done

Prompt33 §10's eight completion items + nine report items, verified against the tree; adversarial P0/P1
zero open; no commits/pushes/infra changes; everything uncommitted for owner review.
