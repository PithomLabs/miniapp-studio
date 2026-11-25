# Prompt18 Boundary A — Phase C Backend (C1–C4, Local-First)

**Verdict: implementation complete, all gates green. STOPPING here per
authorization. Two items need owner ratification (contract change §2.1,
unverified-live list §5); everything else is evidenced below.**

Scope executed: Phase C backend only (U1–U3). No Part 2, no C5 live
proof, no Phase E, no commits, no Sprite/Fly invocations of any kind
(shell history: docker/psql/go/curl/python/chrome only), no live
OpenCode, synthetic invoice text only, PG19 `127.0.0.1:5434` only.

## 1. Files

New:

- `server/sender.go` — durable dispatch loop (claim/dispatch/settle,
  classification, reconcile, recovery rows, backoff, sweep).
- `server/sender_test.go` — 23 sender tests incl. the U2 Q&A chain.
- `server/sse_resume_test.go` — kill-and-resume, backlog drain,
  concurrent no-gap/no-dupe proofs.
- `gateway/sender_contract_test.go` — status turn, conflict codes,
  indeterminate marker, stable-id collapse across clients.
- `tools/dispatch_contract_test.go` — model persistence/validation,
  assign `message_id` linkage.
- `tools/binding_test.go` — turn freeze, terminal-on-assigned,
  cancel-stale, answer-answer race.
- `store/migrations/000017_sender_tracking.{up,down}.sql` —
  `builds.model`, `builds.gateway_turn`.
- `store/migrations/000018_sender_schedule.{up,down}.sql` —
  `commands.next_attempt_at` + due index.

Modified (this session only; tree holds earlier unrelated work):

- `gateway/client.go` — `SendWithID`, `CallError{Status,Body}`.
- `gateway/server.go` — status `turn`, six conflict/indeterminate codes.
- `tools/build.go` — `buildRow` model/turn, `build.create` model,
  `LogCommandTx`, advisory-lock ordering fix in `appendStream`.
- `tools/question.go` — submit-time turn freeze.
- `tools/ingest.go` — assigned-join for yields and terminal events.
- `tools/conversation.go`, `tools/register.go` — assign `message_id`,
  model schema.
- `server/server.go`, `server/config.go`, `cmd/studio/main.go` —
  sender lifecycle, `STUDIO_SENDER` knob.
- `server/gateway_callback.go` — duplicate question id → 400.
- `server/gateway_callback_test.go`,
  `tools/ingest_answer_test.go`,
  `server/conversation_ui_test.go` — new pins + contract updates.
- `docs/hackathon-demo-critical-path.md` — C1–C4 labels + evidence.

## 2. Invariants (established)

- Row id = stable gateway message id; retries collapse via claim replay.
- 2xx = launched (outcomes via callbacks); never inferred otherwise.
- `gateway_turn` advances only from gateway accepts/status, never Studio
  counters. Continue payloads freeze `(turn, content)` at creation.
- Recovery fires once per lineage (`recovered_from`); second stale or
  second ambiguous fails visibly — no unbounded row chains.
- Waits never burn attempts; 25-attempt budget bounds every retry.
- RLS FORCE preserved; sender polls per configured tenant only.
- Lock order: conversation KeyShare → advisory → fresh inserts, in
  `appendStream` for all callers (deadlock fix, §3).
- `workspace_id` = tenant (plan identifiers); sender→gateway→callback
  continuity pinned by the chain test (§3).

### 2.1 Contract change — NEEDS OWNER RATIFICATION

`TestIngestYieldRequiresRunning` pinned yield-on-assigned → absorb. That
wedges the build `assigned` forever when `started` is lost (the later
terminal event absorbs too; no recovery path exists). U2 parks instead:
assigned → running → waiting_user, and terminal events join the same
way. The old test was rewritten as `TestIngestYieldOnAssignedParks`
with the rationale inline. If rejected, the wedge returns and C3 must
be re-scoped around an operator replay procedure.

## 3. Red-before/green-after log

- Turn-less continue row → sender fails fast, zero gateway traffic
  (`TestSenderContinueRequiresFrozenTurn`).
- Stale turn (gateway 3, Studio 1) → reconcile → supersede at 4 with
  original content; second stale fails (`TestSenderStale*`).
- Indeterminate dispatch → poisoned id retires, content re-issues fresh;
  second ambiguity fails (`TestSenderIndeterminate*`).
- Crash-adopt: pre-crash assign + queued row → adopted turn 1, one
  gateway execution (`TestSenderAlreadyAssignedAdopts`).
- **Deadlock (pre-existing):** U3 hammer went red — PG log showed
  `deadlock detected` between conversation-row holders and advisory
  holders (FK KeyShare vs explicit order). Fixed in `appendStream`;
  green twice + `-race`, PG log clean since.
- **Wrong workspace (mine):** sender first sent the operator binding;
  the callback handler looks up tenants, so every callback would 404.
  Caught by inspection against the plan; chain/assign/isolation tests
  proven red with a bogus value, green with tenant.
- **Retry poison (pre-existing):** reused question id returned 500
  (observed live). Mapped to 400 + regression test.
- All 25-attempt exhaustion, busy-retry, shutdown-no-burn, sweep,
  tenant-isolation, and defensive-stub branches pinned by named tests.

## 4. Exact test results

Gates (fresh, this session, PG19 `:5434`):

```text
gofmt -l .            → clean (no output)
go vet ./...          → clean (no output)
go build ./...        → ok
go test -p 1 -count=1 ./... →
  ok  apps  cmd/gateway  cmd/studio  connector  fly  gateway (2.9s)
  ok  render  runner  server (23.9s)  spec  spike  store (5.2s)  tools (11.2s)
go test -race ./tools/ -run 'TestAnswerAnswerConcurrency|TestAnswerYieldConcurrency' → ok
go test -race ./server/ -run 'TestSSEConcurrentTrafficNoGapsNoDupes|TestSenderSweepRecoversOrphans|TestSenderRestartResumesLoop' → ok
```

New tests: 4 gateway + 4 tools dispatch + 4 tools binding + 23 sender
(+1 chain inside) + 3 SSE resume + 1 callback mapping = 39, all green.
Migration round-trip/idempotency suites cover 000017/000018.
Pre-existing `TestAdapterStatus` flake: not observed this session; still
out of scope, untouched.

## 5. Browser evidence (real Chrome 144, CDP-driven, live server)

Isolated DB `studio_boundary_a` (created, migrated, dropped after);
server `serve` + `STUDIO_SENDER=1` on `:18081`; signed Python poster
(byte-exact envelope replication, verified by 200s):

```text
PASS login -> /
PASS conversation /conversations/88af2f1f64780d4d3bdcfc7a11bceb33
PASS message sent, build opened (assigned)
agent.started 200 {"ack":true,"build_status":"running",...,"transitioned":true}
agent.question 200 {"ack":true,"build_status":"waiting_user",...,"transitioned":true}
PASS question streamed live via SSE (no reload)
PASS answer posted via UI
PASS screenshot /tmp/boundary-a/final.png
```

Screenshot visually verified: full transcript (user message, Build
opened, agent.started, agent.question, CC-42 answer), build `running
(turn 1: answered)`. DB cross-check on the pass build:

```text
assign   | queued | 0 |    | build has no gateway URL; waiting… | running |
continue | queued | 0 | 2  | build has no gateway URL; waiting… | running |
```

Continue froze turn 2 through the real UI path; the live sender held
both rows without burning (no gateway URL provisioned — honest wait).
P4 renders answer forms server-side only (SSE appends text), so the
driver reloads once for the form; live forms arrive with C6.

## 6. Fake/live residuals (unverified live)

- C5 live proof: NOT run (gated). All gateway interop is against the
  real `gateway.Server` + `FakeAdapter` over loopback HTTP.
- Gateway binary runs OpenCode only — no fake-adapter mode exists, so
  no live gateway process was started (would be banned OpenCode
  execution). The sender↔gateway HTTP contract is proven by tests.
- Dispatch state has no UI surface until C6 (rows are DB-visible only).
- Out-of-band gateway drives: reconcile-once bounds them; tested via
  scripted out-of-band continues, not a second live driver.
- SSE polls cost one tool invocation per tick per stream (pre-existing
  P4 design; flagged, not changed).
- Test boots rotate the cluster-global `studio_app` password: any live
  server sharing PG19 needs re-provision after a suite run (observed
  twice; pre-existing hazard, flagged).
- Terminal dispatch failures are queryable rows; no stream event yet
  (deferred to Console phase per plan).

## 7. Next dependency (C6 Console, U4 — NOT STARTED)

Needs from this boundary: command-row states + `gateway_turn` for the
Line/Pulse/rail; SSE contract as proven; live answer forms (server
render + SSE text today). No code beyond Boundary A was written.
Secrets: live-pass credentials lived in `/tmp/boundary-a` (0600) +
process env only; none in the repo, none in this report; pass DB
dropped, processes killed. Tree left uncommitted, nothing pushed.
