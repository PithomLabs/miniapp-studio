# Plan to implement plans/prompt18.md — local-first completion

## Goal

Execute the approved local-first sequence: implement and locally
accept Phase C backend + Focus Console shell, Phase D artifact
pipeline + scenes, and a full local Invoice Review Desk rehearsal —
with Part 2, C5-live, and Phase E explicitly gated off and zero
cloud Sprite operations.

## Success Criteria

- C1 sender loop dispatches authorized continues durably
  (ordered, idempotent, retry-safe) against a local Gateway
  harness; C2/C3 bind answers to exact turns and reject stale
  ones; C4 history/SSE/reconnect survives restart.
- D pipeline verifies pinned artifacts, sandboxed-builds/tests
  them, records machine-readable reports, and enforces
  approve/rework with all D4 negative classes refused.
- Focus Console implements all seven stages per the locked
  spec, every state derived from real Studio persistence,
  unverified capabilities honestly labeled.
- Rehearsal runs the golden path end to end locally with
  synthetic data only; evidence per §11 at each boundary.
- No sprite invocation of any kind, no live ops, no commits
  without authorization, Part 2/C5-live/E-live untouched.

## Approach

Reuse-first; the schema already anticipates this work
(`studio.approvals/artifacts/builds/questions/turns/runs/
commands/conversation_stream`) and the tool surface has no
`artifact.*`/approval tools yet — those are the deliberate
additions below, not discoveries.

1. **C1 sender loop** lives server-side (new `server/sender.go`
   worker using the real `gateway.Client` against a configured
   base URL), keyed idempotent on (build, turn, question_id),
   persisted via `studio.commands`, exercised against
   `httptest` + real `gateway.Server` (established pattern).
   No new adapter; no production local-execution path.
2. **Console delivery** extends the existing embedded-template
   pattern: `server/templates/focus/*.html` (added to the
   `ParseFS` set) + a small new `server/static/` vanilla-JS
   bundle (SSE client, timers, hold-to-confirm) served by a
   static handler, under a new `/console` route tree. Legacy
   routes/templates untouched. No JS framework.
3. **Sandboxed build/test** is a local subprocess runner with
   context timeout, fresh temp workdir per run, `GOPROXY=off`
   (stdlib + warm module cache; go.mod has no replace
   directive), capturing `go test -json` into the evidence
   record. No user namespaces; strength documented as
   local-dev grade, not hostile-tenant grade.
4. **New Class A tools** (exact contract): `artifact.verify`
   (fetch pinned manifest, verify digests/paths/archives/
   symlinks/sizes/allowlist), `artifact.test` (sandboxed
   build+test → report), `artifact.decide` (approve/rework →
   `studio.approvals`). Migration `000017` only if column
   inspection (first task of Phase D) proves a gap.
5. **No local OpenCode by default.** Canned transcripts +
   scripted doubles already pin the translator; a local-binary
   spike runs only on a demonstrated gap, time-boxed.
6. **Sprite CLI stays fully idle.** Verified present at
   `~/.local/bin/sprite`; per §9's "operate" ban, not even
   read-only invocations during this phase.
7. **Evidence honesty enforced mechanically:** demo-doc step
   labels updated with evidence links at each boundary; a
   final grep audit for fabricated-success patterns
   (hardcoded green, fixture URLs/receipts, simulated timers).

Non-goals: Part 2/C5-live/E-live, sprite invocation of any
kind, second adapter, workflow engines, marketplace/billing,
Lead-to-Booking, Miniapp/F1/G-01/P5 changes, commits without
authorization, broad re-reviews.

## Steps

### P0 — C backend (C1–C4; blocks everything)

- U1: sender loop + dispatch persistence + idempotent retry;
  red-before/green-after incl. crash-mid-dispatch, duplicate
  delivery, gateway-down retry. Depends on: nothing.
- U2: C2/C3 binding, validation, atomic handoff, stale
  rejection; concurrency + conflict tests. Depends on: U1.
- U3: C4 history/SSE/replay/reconnect/restart; kill-and-resume
  tests against real PG19. Depends on: U1.
- Boundary A report (§11 six items).

### P1 — Console shell (C6; needs P0)

- U4: `/console` routes + `templates/focus/` shell scenes
  (spine, Line, Pulse, rail) + static JS, wired to C1–C4
  state; loading/empty/error/offline/reconnect per scene.
- U5: clarification reply binding + approval/rejection
  outcomes against real records; OPS/DEV density; hold-to-
  confirm with documented tenant-bounded scope.
- Boundary B report + desktop/mobile + keyboard/a11y pass.

### P2 — D pipeline + scenes (needs P1)

- U6: column inspection → `artifact.verify` + `artifact.test`
  (+ `000017` iff gap proven); synthetic fixture path from
  `spec/testdata/valid_invoice_review.json`.
- U7: `artifact.decide` + diff/review/evidence views; all
  nine D4 negative classes as failing-first tests.
- U8: Spec/Validate/Connect/Evidence scenes on real D state.
- Boundary C report.

### P3 — Rehearsal + close (needs P2)

- U9: scripted local E2E golden path in a real browser
  against PG19; synthetic data only; local-only labeling
  audit; demo-doc status updates with evidence links.
- U10: full gates (fmt/vet/build/unit/race/DB/migrations/
  RLS), fabricated-capability + secret-exposure sweeps,
  Phase E recommendation with evidence, final §11 report. Stop.

## Validation Plan

- Per unit: red-before/green-after tests; concurrency, retry,
  crash, replay, recovery where relevant; new tools get
  tenant-isolation + idempotency tests.
- Per boundary: the six §11 items (files, invariants, exact
  test results, browser evidence, fake/live residuals, next
  dependency) — missing browser evidence fails the boundary.
- Gates: `gofmt`, `go vet`, `go build`, full suite + race on
  PG19 `127.0.0.1:5434` (isolated test DBs; stop + report if
  unreachable), migration + RLS suites, seven-scene browser
  matrix (states × density × viewport × keyboard), rehearsal
  green. Highest-risk check: the rehearsal — any fabricated
  success or hand-waved state fails the milestone regardless
  of unit-test green.

## Risks / Open Questions

None blocking. Defaults above (vanilla JS, `/console`,
subprocess sandbox, three new tools, no local OpenCode
unless gapped) are all reversible pre-acceptance. Two
watch items, not questions: `studio.approvals` column
sufficiency (decided by inspection in U6) and Sprite CLI
remaining untouched (verified by shell history at close).
