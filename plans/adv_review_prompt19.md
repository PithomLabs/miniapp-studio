# Adversarial Code Review — Boundary A (Phase C Backend C1–C4)

Read-only, independent review of the working-tree implementation
against `docs/prompt18-boundary-a.md`. No fixes implemented, no
live operations, no commits. Scope separated via `git status`/`git
diff`: the runbook Part 2 sections (`docs/gateway-phase-b-live-runbook.md`),
`plans/plan_ui_z.md`, and the `docs/prompt12-residual-closure-report.md`
count-precision edit are other tracks and were read only to exclude;
all findings below are Boundary A files.

Verification performed: full serial suite reproduced green on PG19
(13/13, timings matching the report), targeted `-race` runs green,
`gofmt`/`vet`/`build` clean, new-test count verified at 39 by name,
every claimed invariant traced to implementation. No Sprite/Fly/
OAuth/OpenCode operations of any kind.

## A. Executive verdict

**ACCEPTED WITH BOUNDED RESIDUALS.**

C1–C4 may be ratified and the implementation may proceed to C6. No
structural or foundational blocker remains: dispatch is idempotent
per stable row ID with gateway-side exactly-once fencing, answers
bind atomically to (tenant, build, turn, question), tenant isolation
holds on every new path, SSE replay is cursor-correct across restart
and reconnect, and the yield-on-assigned contract change is sound
(adjudicated in §C). Three Medium findings (multi-instance sender
safety, assign wire-byte stability, terminal-build dispatch/cancel
handling) and six Lows require bounded corrections or explicit
tracking, but none invalidates the local-first backend: every
degraded path I could construct fails visibly and convergently,
never with silent corruption, duplication, or cross-tenant access.

## B. Findings

### F1 — Concurrent Studio senders can double-dispatch; sweep assumes singleton (Medium)

- **Invariant:** exactly-once dispatch per command row; a claimed row
  is dispatched by one worker; crash sweep only requeues genuinely
  orphaned rows. Owner: `server/sender.go` (`StartSender`,
  `sweepTenant`, `claimAndPlan`, `settleTx`).
- **Path:** `StartSender` guards one loop per process
  (`server/sender.go:146-148`) but nothing stops two processes
  (rolling overlap, accidental double-serve). The second instance's
  `sweepTenant` (`server/sender.go:223-230`) resets *all* `sending`
  rows to `queued` — including rows the first instance is actively
  dispatching — then claims and dispatches them concurrently.
- **Scenario:** A dispatches assign row R (prompt resolved M1);
  B boots, sweeps R to queued, claims R, dispatches same ID. With
  identical bytes the gateway dedup collapses (safe); with
  diverged assign bytes (see F2) one side eats `payload_mismatch`
  and terminally fails a launched turn; attempts double-burn; the
  loser's `settleTx` guard (`state='sending'`,
  `server/sender.go:870-880`) fails loudly. Separately, two live
  loops can settle `gateway_turn` out of order (unguarded UPDATEs
  at `server/sender.go:903-921`), regressing the tracked turn;
  the next answer then freezes stale and self-heals via
  reconcile-supersede (wasted cycle, no corruption).
- **Impact:** no duplicate execution (gateway ID fencing holds)
  and no silent corruption, but spurious terminal rows,
  misleading details, and turn regression are possible.
- **Demonstrated vs theoretical:** mechanism demonstrated by code
  trace; gateway collapse + settle guard verified in code and
  covered for the single-instance case by
  `TestSendWithIDCollapsesAcrossClients`,
  `TestServerConcurrentIdenticalContinueSingle` (Part 1),
  `TestSenderSweepRecoversOrphans`. No test runs two senders on
  one tenant (`TestSenderTenantIsolation` runs two senders on
  *different* tenants).
- **Correction:** enforce singleton per database: hold a session
  advisory lock for the loop's lifetime in `StartSender` and
  refuse a second starter fast, plus a one-line doc statement.
  Regression: two `StartSender` calls against one DB → second
  refuses; no row moves.
- **Blocks ratification:** no (local-first is single-server);
  blocks any multi-instance use. Carry to C6 ops.

### F2 — Assign wire bytes can differ across attempts of one row (Medium)

- **Invariant:** stable row ID ⟹ stable wire bytes on every
  attempt (asserted in `sendPlan` comment,
  `server/sender.go:288-290`; gateway `payload_mismatch` fencing
  relies on it). Owner: `server/sender.go` (`claimAndPlan` /
  `resolvePrompt`).
- **Path:** title-only assigns (`build.create`, and any pre-C1
  assign row) carry no `message_id`, so `resolvePrompt`
  (`server/sender.go:441-449`) resolves "latest user message" or
  title **at each claim**. Attempt 1 resolves M1 and the gateway
  launches; the response is lost; the user writes M2; attempt 2
  resolves M2 under the same row ID → gateway `payload_mismatch`
  → row terminally fails with "sender bug" on a successfully
  launched turn. (`message.send` assigns pin `message_id` and are
  immune — verified in `tools/conversation.go:202-203` and
  `TestMessageSendAssignCarriesMessageID`.)
- **Impact:** fabricated failure (row says failed; run is live)
  with a misleading detail. Converges without corruption: the
  live run's callbacks drive ingest (yield-on-assigned absorbs),
  and `gateway_turn` stays stale-safe via reconcile. Narrow
  window (message interleaved between attempts + lost response).
- **Demonstrated vs theoretical:** traced, not executed; no test
  covers redelivery-after-new-message (test gap below).
- **Correction:** freeze the resolved prompt into the row payload
  on first claim (UPDATE in the claim txn; reuse when present).
  Regression: resolve M1 → record interleave → redeliver → assert
  identical bytes sent (gateway-seen payloads equal) and one
  execution.
- **Blocks ratification:** no. Recommend before C6 (cheap; the
  Console will otherwise display "sender bug" terminals).

### F3 — Sender dispatches rows for terminal builds; cancel-for-unprovisioned never settles (Medium)

- **Invariant:** no work launches for dead builds; every command
  row reaches a truthful terminal state. Owner:
  `server/sender.go` (`claimAndPlan`).
- **Path 1:** `claimAndPlan` never checks build status. Cancel
  arrives while an assign is queued (gateway slow/down): the
  assign still dispatches and launches a real agent run against
  a cancelled build. Callbacks absorb as history
  (`terminalBuild` guard, `tools/ingest.go:108`) — wasted run,
  no corruption, but user-visible weirdness once C6 exposes
  dispatch states.
- **Path 2:** cancel of a never-provisioned build: `claimAndPlan`
  (`server/sender.go:359-363`) holds `queued` forever ("waiting
  for provisioning") although the cancel goal holds vacuously
  (compare the 404-cancel ack at `server/sender.go:668-673`).
  Build state is truthful (already `cancelled`); only the row
  never settles plus one hold-refresh per 30s poll.
- **Demonstrated vs theoretical:** traced; no test covers
  either (no terminal-build dispatch test; no unprovisioned
  cancel test).
- **Correction:** claim-time terminal check — skip with honest
  detail (`build already X; not dispatched`); ack cancel
  immediately when the build has no gateway URL. Two small
  tests. Does not change any green path (all current tests use
  live builds).
- **Blocks ratification:** no. Recommend before C6.

### F4 — Terminal-while-waiting + late answer strands build `running` (Low)

- **Invariant:** every build reaches an accurate terminal state;
  no unrecoverable non-terminal builds. Owner: `tools/ingest.go`
  (`applyEvent`) + `tools/question.go`.
- **Path:** terminal event arrives while `waiting_user` → absorbed
  with no state change (`tools/ingest.go:159-161`). User then
  answers the still-open question → build `waiting→running` +
  continue row → gateway session is done → `session_terminal` →
  row terminal. Build sits `running` with no live run and no
  future callbacks. Requires out-of-band completion (crash
  recovery emits `failed`, which F6 resumes — safe) or gateway
  misbehavior; in-band runs yield-XOR-complete so the race
  cannot arise (answer/callback txns serialize on the build
  lock; answer-first converges correctly).
- **Impact:** one visibly stale build; operator recovers by
  messaging (new build). No corruption, no duplication.
- **Correction:** on terminal-while-waiting, expire the open
  questions of the turn with cause instead of pure absorb.
  Regression: out-of-band completed → question expired, answer
  409s.
- **Blocks ratification:** no (out-of-band-only).

### F5 — Conflicting duplicate leaves no Studio trace and no log (Low)

- **Invariant:** conflicting gateway data fails visibly.
  Owner: `server/gateway_callback.go` (400 branch).
- **Path:** same `question_id` under a fresh `event_id` →
  question INSERT 23505 → whole ingest txn rolls back → HTTP
  400 via the `"already recorded"` substring match
  (`server/gateway_callback.go:82-86`). The 400 branch, unlike
  `default`, emits no log line, and the rolled-back event
  leaves no Studio row — evidence exists only gateway-side.
- **Impact:** the 400 itself is the *correct* protocol response
  (terminal, no retry poison — verified against gateway outbox
  4xx handling; adjudicated in analysis §7). The gap is purely
  forensic visibility for a gateway-bug signal. Exact
  duplicates still 200-ack via event-id dedup; identical-content
  re-emissions 400 harmlessly (already parked).
- **Demonstrated vs theoretical:** traced + covered by
  `TestCallbackDuplicateQuestionIDFailsTerminally` (conflicting
  bodies → 400); log absence verified by reading the branch.
- **Correction:** one `logger.Warn` with build/event/question
  identifiers (never bodies). No behavior change.
- **Blocks ratification:** no.

### F6 — Pre-C1 continue rows terminal-fail on upgrade; recorded answers never dispatch (Low)

- **Invariant:** upgrades don't silently strand authorized work.
  Owner: `server/sender.go` (`frozenTurn` gate) + migration notes.
- **Path:** verified (`git show HEAD:tools/question.go`) that
  pre-C1 `question.answer` logged continue rows *without* `turn`.
  After upgrade, first claim terminal-fails them ("lacks a frozen
  turn", pinned by `TestSenderContinueRequiresFrozenTurn`); the
  question is already `answered` so it can never be re-answered.
  (Pre-C1 assigns are fine — the message-less fallback was built
  for them.)
- **Impact:** nil for the demo (fresh DBs; browser pass used an
  isolated DB) but a real upgrade hazard with no documented
  fresh-DB requirement.
- **Correction:** document that C1 requires fresh command rows
  (or backfill frozen turns at claim for legacy rows).
  One-line doc or a small migration note; no code needed for
  ratification.

### F7 — SSE has no server-side lifetime/stream cap; polls cost a full invocation each (Low)

- **Invariant (M5-SSE gate):** bounded stream lifetime +
  explicit concurrent-stream budget. Owner: `server/sse.go`.
- **Path:** `handleConversationEvents` loops forever (ping on
  idle, never exits except error/client-close); no write
  timeout; no per-conversation stream cap. Each tick runs a full
  framework `Invoke` with a fresh op key → one
  `app.invocations` row per second per stream. A black-hole
  client (dead TCP the server can't detect) leaks one goroutine
  + ~86k invocation rows/day/stream; browsers close properly
  (verified: cursor is client-driven, reconnect replays from
  persisted rows — restart/reconnect correctness itself is
  solid and tested).
- **Impact:** bounded at local-demo scope (single user, few
  tabs); must gate any shared/multi-user use. The per-poll cost
  is correctly labeled a residual in the report.
- **Correction:** track under the existing M5-SSE gate; no code
  for Boundary A. Quantify in C6 planning (invocation-row
  growth rate).

### F8 — Mid-run user messages never reach the agent (Low)

- **Path:** `message.send` appends the message but only opens a
  build when none is non-terminal (`tools/conversation.go:188-207`);
  a message sent while a build runs is transcript-only — no
  continue, no agent visibility. Deliberate (frozen-prompt
  model), but silent.
- **Correction:** C6 UX must set the expectation (or queue
  follow-ups); one doc line. Not a defect.

### F9 — Exhausted/failed rows have no stream event (Low, admitted)

- **Path:** `settleTx` writes commands/builds rows only; terminal
  dispatch failures are DB-queryable, never streamed. A build
  whose continue exhausts sits in its last state (e.g.
  `waiting_user`) with failure visible only in `last_error`.
- **Impact:** operator confusion at demo scale if the gateway is
  down; states remain truthful (never falsely terminal).
- **Correction:** already deferred to C6 in the report — carried
  as an explicit C6 dependency, not a new ask.

### Test gaps (minor, recommended additions)

- **T1:** direct crash-accept-settle composition for continue:
  seed gateway accept under the row ID + row `sending`, sweep,
  redeliver → assert replayed 200, `acked`, exactly one
  execution. (Covered today only compositionally via
  `TestSendWithIDCollapsesAcrossClients` + sweep/adopt tests.)
- **T2:** delayed-original-after-supersede: original run emits
  terminal callbacks after its continue was superseded →
  assert no corruption, honest terminal row, converged build.
  (Traced convergent in all branches; unexecuted.)
- **T3:** F2 redelivery-after-new-message; F3 terminal-build
  skip + unprovisioned-cancel ack; F4 terminal-while-waiting
  expiry — one test each with the corrections above.

### Findings explicitly considered and NOT filed

- Same answer content delivered across two turns after genuine
  ambiguity: safe direction (re-presentation, never loss);
  same turn can never execute twice (gateway fencing + id
  retirement). Verified, not a defect.
- `next_attempt_at` app-clock vs DB-clock basis: same-host
  local scope, harmless; multi-host note only.
- Mid-run `s.tenants` staleness and operator-removal stalls:
  config-coherence edge, documented restart requirement.
- `ackPlan` recording the frozen (not gateway-echoed) turn:
  consistent with "advances only from gateway-acknowledged
  accepts" since fencing enforces the value; fencing tested.
- Two-query `latest_seq` vs items snapshot: same txn, consistent.
- `TestAdapterStatus` flake: untouched, still out of scope,
  not observed in my runs.

## C. Contract-change adjudication: yield-on-assigned

**Recommendation: RATIFY WITH A BOUNDED CONDITION.**

Reasoning against the six questions:

1. **Parking is the correct recovery behavior.** The old absorb
   stranded builds `assigned` forever with no recovery path
   (assign already dispatched or gateway running; no event
   could ever move the build again). Parking converts certain
   death into a recoverable, truthful state.
2. **No false liveness.** Event dedup (`ON CONFLICT DO NOTHING`
   on `(tenant, build, event_id)`) runs *before* `applyEvent`
   (`tools/ingest.go:85-102`), so duplicates can never re-park.
   A *fresh* yield event is HMAC-authenticated evidence of a
   gateway emission; treating it as proof of a live run is
   sound modulo gateway/key trust (already the system's
   foundation). The only false-positive source is a gateway bug
   or out-of-band drive emitting yields without runs — in
   which case parking + later gateway 404/stale handling still
   converges (verified paths), and the alternative (absorb)
   strands.
3. **Terminal-before-started joins truthfully.** Same dedup
   protection; the signed terminal proves the run; late
   `started` absorbs harmlessly afterward.
4. **State stays accurate.** Transitions reuse the existing
   `assigned→running→waiting_user` table; supersede/expiry
   semantics unchanged; callbacks for terminal builds still
   absorb. Gateway-side turn truth is untouched (Studio turns
   and gateway turns are separate counters by design; freeze
   uses only gateway-acked values, reconciling on divergence).
5. **Late/duplicate/stale/contradictory callbacks:** duplicates
   ack via dedup; stale turns 409 via fencing + reconcile
   supersede; contradictions 400 terminally (duplicate question
   IDs); terminal races serialize on the build lock with both
   orders safe in-band (out-of-band strand is F4).
6. **Contract update required:** yes — the bounded condition.

**Condition:** (a) record the trust assumption where the code
lives (an authenticated yield implies a live run; out-of-band
drives can park questions for runs Studio didn't dispatch —
convergent via reconcile, not silent); (b) implement F4
(expire open questions on terminal-while-waiting) so the one
remaining absorb-then-strand path closes. Neither reopens the
decision; both are small, and C6 may proceed with them tracked.

The old test was correctly rewritten rather than preserved:
it pinned the wedging behavior, and the report discloses the
change with rationale instead of burying it.

## D. Evidence verdict

Precise category split (no promotion):

1. **Unit-tested sender behavior** — 23 sender tests incl.
   sweep/shutdown/exhaustion/waits/isolation; busy/indeterminate/
   stale paths scripted against `FakeAdapter`. Real logic,
   fake agent.
2. **Integration-tested real Gateway HTTP contract with
   FakeAdapter** — sender tests drive the real
   `gateway.Server` (claims, dedup, fencing, outbox paths) over
   loopback HTTP; `TestSendWithIDCollapsesAcrossClients` proves
   collapse; adapter behavior scripted. The HTTP contract is
   proven; agent behavior is not.
3. **Browser-tested Studio persistence/callbacks/SSE/answer
   flow** — real Chrome 144 + live server + isolated DB:
   login, conversation, message, build open, signed started/
   question callbacks, live SSE render, UI answer with frozen
   turn 2 verified in DB. Genuine local-system evidence.
4. **Sender→Gateway dispatch** — proven ONLY in (2)
   (FakeAdapter). The browser pass explicitly did NOT dispatch
   (rows held `queued`, no Gateway URL). The report states this
   plainly in §5/§6; it must not be cited as dispatch proof.
5. **OpenCode inference** — never ran (correctly banned).
6. **Same-session continuation** — never live; resume path only
   via FakeAdapter + signed-callback simulation.

Also verified: new-test count is exactly 39 as named; full
serial suite reproduced green (13/13, timings match); race
runs reproduced green; `gofmt`/`vet`/`build` clean. The
report's residual list (§6) is honest — every item I checked
(send-vs-ingest deadlock fix verified by lock-graph inspection
plus race; workspace correction pinned by chain test;
password-rotation hazard real and pre-existing).

## E. Residual register

- **Structural blockers:** none.
- **Bounded local-demo residuals (ratification-compatible,
  tracked):** F1 (singleton enforcement before any 2nd
  instance), F2 (prompt freeze), F3 (terminal-skip +
  cancel-ack), F4 (terminal-while-waiting expiry — also the §C
  condition), F5 (conflict warn log), F6 (fresh-DB upgrade
  note), F9 (exhausted-row surfacing → C6). Plus test additions
  T1–T3 and the F4/F3/F2 unit tests above.
- **C5/live-only requirements:** live gateway interop,
  OpenCode execution, same-session continuation proof.
- **C6 dependencies:** command-row states + `gateway_turn`
  (exist); exhausted/terminal surfacing in-stream (F9);
  live answer forms; mid-run-message UX (F8).
- **Deferred scaling work:** F7 (M5-SSE lifetime + stream
  budget + per-poll invocation cost), single-worker
  head-of-line blocking, multi-instance operation.

## F. Acceptance recommendation

No structural or foundational blocker remains. **I recommend
accepting Boundary A (C1–C4 ratified) with the bounded residuals
in §E tracked, and proceeding to C6** — no further broad review
required before Console work. The §C condition (F4 expiry +
trust-assumption note) should land with the early C6 backend
touch-ups, not gate its start. C5 live proof remains a separate
gate and is not required for this acceptance.
