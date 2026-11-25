# Prompt20 Plan — C6 Focus Console (Boundary B)

## Goal

Implement the Focus Console per the locked spec (`plans/plan_ui.html`,
`plans/plan_ui_z.md`), preceded by the five mandatory backend
touch-ups (F2–F6). Every displayed state derives from persisted Studio
state. Stop at Boundary B with the §11-style report (files,
invariants, tests, browser evidence, residuals).

## Success Criteria

- F2–F6 corrections landed, each with a red-before/green-after test.
- Console serves the 4-zone shell: rail, spine, scene, Pulse, Line.
- Spine pills (Describe→Evidence) reflect persisted state only; green
  means proven, never completed-by-claim.
- Pulse streams persisted events/receipts/alerts/clarifications/
  approvals/terminal dispatch outcomes with replay + reconnect.
- Clarification reply binds (tenant, build, turn, question); answering
  never authorizes; stale answers 409 honestly.
- Approvals are human-only (operator session), bound to
  tenant/build/question (+ artifact when present; full digest-bound
  artifact approval is Phase D), inline approve/reject with durable
  outcomes.
- OPS/DEV are density modes over identical state; DEV exposes no
  secrets.
- Hold-to-confirm stop is tenant-bounded, counted, persisted
  (per-build cancel rows + system messages stream in Pulse).
- Desktop + mobile layouts, keyboard + a11y pass, all eight UI states
  demonstrable against real state.
- Full gates green on PG19; no commits without authorization.

## Approach

Two workstreams, strictly ordered: backend touch-ups first (Console
must not display untrustworthy state), then Console shell → live
wiring → scenes → browser matrix. Server-rendered full pages per
scene (progressive enhancement, works without JS); vanilla JS adds
only SSE append, density toggle, kill-hold timing, and reply focus.
No framework, no SPA-JSON architecture, no new abstraction layers.

## Key Decisions

1. **F2 prompt freeze (no migration):** `claimAndPlan` writes the
   resolved prompt into the row payload (`payload.frozen_prompt`,
   `payload.frozen_prompt_src`) on first claim and reuses it on
   retry. Safe: assign wire bytes are built from prompt+model+agent,
   never from the row payload, so the extra keys cannot perturb the
   gateway digest. Alternative (new column) rejected: payload JSON is
   the row's existing extension point and keeps this a code-only fix.
2. **F3 terminal guard placement:** in `claimAndPlan`, same txn as the
   claim. Terminal build + cancel kind → ack ("already X; cancel goal
   holds"); terminal + other kinds → terminal fail ("build already
   X; not dispatched"); missing gateway URL + cancel → ack
   immediately (goal holds vacuously). No green path changes.
3. **F4 expiry (migration 000019):** add
   `questions.expired_reason TEXT NOT NULL DEFAULT ''`. On
   terminal-while-`waiting_user`: expire the turn's open questions
   with cause (`terminal <type> arrived while awaiting answer;
   event <id>`), transition turn → `expired`, build →
   succeeded/failed, `transitioned=true`. Later answers hit the
   existing "no longer open" path → 409 page (reuse
   `renderAnswerConflict`). Trust assumption recorded as a comment
   at the assigned-join in `applyYield`/`applyEvent`.
4. **F5 logging:** one `logger.Warn` in the callback 400 branch with
   tenant/build/event_id/question_id (qid extracted defensively
   from the verified payload; never body/options).
5. **F6 docs:** new `docs/c1-command-row-upgrade-note.md` (fresh DBs
   required; pre-C1 continue rows strand) + pointer comment at the
   `frozenTurn` gate. No backfill code.
6. **Pulse dispatch outcomes (migration 000020 + F9):** extend
   `conversation_stream.kind` CHECK with `'dispatch'`; sender
   `settleTx` appends a stream row (ref = command id) on terminal
   outcomes (failed/exhausted) only. `conversation.stream` gains a
   `LEFT JOIN studio.commands` for dispatch rows, exposing a
   shaped subset (state, attempts, capped first-line reason —
   never raw transport text). `build.status` output gains a
   bounded `commands` list (latest 20) with the six display
   states mapped as: queued+0 attempts → queued;
   queued+attempts → retrying; sending; acked; failed split by
   `exhausted` detail prefix.
7. **Console architecture:** `GET /console` (+ `?conversation=`,
   `?scene=`) renders full pages from `server/templates/focus/`
   (own layout — the 4-zone grid replaces app chrome; extend the
   embed pattern, do not touch the shared layout). One embedded
   vanilla JS file served at `GET /static/focus/console.js`
   (session-gated, `no-cache`, new route — no static serving
   exists today). POSTs (Line, reply, approve/reject, kill) reuse
   session + Origin protection; no new auth scheme.
8. **Scene scope for C6:** shell + Describe/Gallery + Run + Evidence
   (receipts ledger from persisted approvals/commands/events/turns)
   fully live. Spec/Validate/Connect/Deploy pills render gray
   "untouched" and their scenes render honest "lands in Phase
   D/E" empty states with working navigation — wiring them now
   would be Phase D/E implementation, which §6 forbids.
9. **Kill switch:** new `build.cancel_all` tool cancelling every
   non-terminal build of the caller's tenant in one txn (reuses
   `build.cancel` semantics: withdraw questions, close turns, log
   cancel rows, system messages), returns count + ids. Tenant
   isolation + idempotency tests per standing rules. No "void"
   approval state — decided rows are immutable; pending questions
   withdraw via cancel.
10. **No approval expiry countdown:** `studio.approvals` has no
    expiry column and prompt20 authorizes no approval-semantics
    change; show pending age from `created_at`. Countdown is a
    recorded residual, not a faked timer.
11. **Density:** server renders OPS text + DEV detail nodes (in
    `<details>`/hidden spans); toggle flips a root class
    (localStorage-persisted). One fetch, identical state.
12. **F8 copy (no queuing):** when the focused build is
    running/waiting_user, the Line shows: "Saved to history — the
    agent only receives answers submitted to its open question."
    Queuing follow-ups is out of scope.
13. **F1/F7:** unchanged — singleton remains an operational
    constraint (doc line in Boundary B report), F7 stays deferred.

## Steps

### U1 — Backend touch-ups F2–F6 (blocks Console wiring)

- F2: payload freeze on first claim + retry-after-new-message test
  (script `FakeAdapter.Fail`, interleave M2, assert identical wire
  bytes + single execution).
- F3: claim-time terminal check + unprovisioned-cancel ack + three
  regression cases (terminal assign skipped; terminal cancel acked;
  unprovisioned cancel acked).
- Migration 000019 + F4 expiry path + 409-after-expiry tests (tool
  + HTTP) + trust-assumption comments.
- F5 warn log + assertion via injected test logger.
- F6 upgrade-note doc + gate comment.
- Gates: gofmt/vet/build, full serial suite on isolated PG19 DBs,
  migration round-trip; never concurrent with a live server.

### U2 — Console shell + static serving + spine

- Routes: `GET /console`, `GET /static/focus/console.js`;
  `server/templates/focus/` (console layout, rail, spine, scene
  shells, Pulse, Line) + `server/static/focus/console.js`
  (SSE client, density, kill hold, reply focus, offline banner).
- Spine mapping from `build.status`: Describe proven iff
  conversation+build exist; Run active/proven/refused from build
  status; Evidence lists persisted receipts; others gray + Phase
  note. Stage clicks navigate; never advance state.
- Handler tests: auth (login wall), tenant isolation (cross-tenant
  conv → 404, no oracle), scene params, empty states.

### U3 — Pulse + Line + clarification/approval wiring

- Migration 000020 (dispatch stream kind) + `conversation.stream`
  commands join + `build.status` commands list.
- Pulse: chronological render of stream items incl. dispatch
  terminals; filters (All/Approvals/Receipts/Alerts/Events);
  pending-approval badges; SSE live append + reconnect cursor.
- Line: submit → `message.send`; contextual reply state carrying
  question_id → `question.answer`; stale → 409 conflict page.
- Approvals: inline approve/reject POSTs; server enforces
  `authorization_request` kind + exact approved/rejected values;
  clarification answers assertably create no approval row.
- Kill: hold-to-confirm → `build.cancel_all` → counted
  confirmation; per-build "Build cancelled." system messages stream
  in Pulse; cross-tenant attempt impossible (tenant from session).
- F8 hint under Line for running/waiting builds.

### U4 — Density, responsive, keyboard/a11y

- OPS/DEV toggle (root class, persisted); DEV detail nodes
  server-rendered, secret-free by construction (assert in tests:
  no token/key/bearer substrings in DEV HTML).
- 900px breakpoint, Pulse slide-over + overlay + Esc, 56px rail
  (from prototype CSS, adapted).
- Keyboard: full Tab reachability, Enter/Space activation,
  keyboard hold-to-confirm (keydown/keyup 700ms, repeat-guarded),
  focus return on sheet close, `aria-live="polite"` Pulse,
  `prefers-reduced-motion` respected.

### U5 — Browser matrix + Boundary B report

- Headless-Chrome CDP pass (established Boundary A pattern) on an
  isolated DB + live server: desktop + 375px mobile; all seven
  scenes; loading/empty/pending/refused/stale/failed/offline
  (CDP offline emulation)/reconnect (offline toggle + server
  restart); clarification concurrency (double answer → one 409);
  approval + rejection; OPS/DEV; keyboard-only run; tenant
  isolation (second operator); dispatch-failure visibility via
  scripted gateway-down exhaustion.
- Update `docs/hackathon-demo-critical-path.md` C6 labels with
  evidence links; write Boundary B report; teardown (drop DB,
  kill processes). Stop; request acceptance.

## Validation Plan

- Per unit: red-before/green-after Go tests; new tool
  (`build.cancel_all`) gets tenant-isolation + idempotency tests;
  migrations get up/down round-trip in the suite.
- Gates each unit: `gofmt -l .`, `go vet ./...`, `go build ./...`,
  `go test -p 1 -count=1 ./...` with `TEST_DATABASE_URL` toward
  PG19 `127.0.0.1:5434` on isolated DBs; `-race` on concurrency
  tests (clarification race, kill vs answer race).
- Browser: CDP script asserting DOM + screenshots inspected via
  image read; SSE replay asserted by cursor-gap test (kill
  server mid-stream, resume, exact-set match).
- Secret sweep: `grep` DEV HTML + Pulse payloads for credential
  shapes; scrub assertions on dispatch reason text.
- Highest-risk check: the browser matrix — any fabricated green,
  scripted timer, or hardcoded URL/receipt fails the milestone
  regardless of unit-test green.

## Risks / Open Questions

- Risk: `build.status` + stream query growth from the commands
  join — bounded (20 rows, indexed); acceptable at demo scope.
- Risk: prototype CSS class reuse vs existing app CSS — mitigated
  by namespaced focus layout + separate template dir.
- Open questions: none — all material choices resolve from the
  locked spec, the ratified review, or inspected code above.
