# Phase A evidence record — Conversation Infrastructure (2026-11-25)

Implementation of `plans/prompt8.md` (Phase A only) against the merged
gateway plan, in one uninterrupted pass. No live Sprite touched (all
gateway interaction via HMAC-signed envelopes to the real callback
endpoint); no OpenCode/plugin/OAuth; no F1/G-01 movement; no P5 change;
Miniapp untouched; everything uncommitted.

## Gates executed

- Baseline (untouched tree): `gofmt -l .` empty, `go vet ./...` clean,
  `go test -p 1 ./...` 12/12 ok on PG19.
- Final: `gofmt` clean, `go vet ./...` clean, `go build ./...` clean,
  `go test -p 1 ./...` 13/13 ok on PG19 (127.0.0.1:5434) and PG17
  (127.0.0.1:5435, 17.12), `go test -race` clean on
  `store`/`tools`/`server`/`gateway`.
- Migration up/down round-trip green (`TestMigrateDownRoundTrip`, scratch
  DB): down to v10 removes exactly the new tables, older tables survive,
  back up to v15 clean. Schema is at v15.

## Files changed

- `store/migrations/000011_conversation` (conversations, messages),
  `000012_build` (builds, turns, questions),
  `000013_command_event` (commands incl. payload JSONB, events),
  `000014_stream_artifact` (conversation_stream, artifacts, approvals),
  `000015_question_provenance` (options, source_event_id) — each with
  down files, composite tenant FKs, CHECK state enums, RLS FORCE,
  `studio_app` grants.
- `tools/`: `build.go` (create/status/cancel, transition tables,
  stream allocator), `conversation.go` (create/get/list/stream/send),
  `question.go` (answer), `ingest.go` (event ingest), `register.go` +
  `tools.go` (10 new tools: 23 registered total).
- `gateway/protocol.go`: v1 envelope, HMAC sign/verify, skew window,
  key-provider boundary + test-only static provider.
- `server/`: `gateway_callback.go` (POST /gw/v1/callback),
  `sse.go` (GET /conversations/{id}/events),
  `handlers_conversation.go` (list/create/view/send/answer/409),
  templates (`conversations_list`, `conversation` incl. EventSource JS,
  `answer_conflict`), home nav link, routes, `Options.GatewayKeys`.

## Data model and protocol decisions

- **D1 as ordered:** unified `conversation_stream(tenant, conversation,
  seq)` with DB-enforced uniqueness; seq allocated transactionally with
  the source insert under a per-conversation advisory xact lock
  (two-integer variant — NUL-concatenated keys are illegal in Postgres
  text, found empirically). Messages carry no own seq; per-build event
  seq stays separate for protocol ordering. Honest invariant: monotonic
  per conversation (aborted txns can gap; cursor replay needs only
  greater-than, never contiguity).
- **`message.send` (directive's "append" made atomic):** submit appends
  the user message AND ensures an open build in one transaction, because
  the framework records failures against their key — a two-call
  append-then-create could strand retries on a recorded partial failure
  (observed live during this pass). Conversation-row locking keeps
  concurrent sends to one build.
- **Callback idempotency layering:** fresh framework key per HTTP attempt;
  exactly-once effects come from the `(tenant, build, event_id)` dedup.
  Keying by message id would replay recorded failures instead of
  retrying transient ones (observed live; documented in code).
- **Envelope `workspace_id` (§8 clarification):** carried as an
  HMAC-authenticated routing hint, re-validated by the tenant-scoped
  build lookup inside the tool. Trust derives server-side at enforcement,
  never from the claim alone.
- **Ingest semantics:** history append-only (every well-formed event
  persists + streams, terminal builds included); transitions strictly
  state-guarded with `transitioned` reported. Terminal-on-terminal and
  out-of-order arrivals absorb without wedging the sender. Yields apply
  on running (park) and waiting (supersede) builds only.
- **Authorization answers:** must be exactly `approved`/`rejected` and
  land in a separate `approvals` row; vague answers are refused. Answers
  never silently grant.
- **Scrub boundary:** persist verbatim (evidence), scrub secret-shaped
  MAPS on read (event payloads, question options) following the
  `activity.list` pattern; flat conversation bodies stay verbatim
  (operator's own transcript; autoescaping handles injection). HMAC keys
  never enter tool inputs, logs, or payloads (handler logs metadata
  only). Scrub-test fixtures use fake `sk-` values.
- **Artifacts:** metadata persisted with `verified_at NULL` (bytes
  unfetchable without a gateway); UI/review of bundles is Phase D.

## Acceptance evidence

- **15-step functional proof:** `TestPhaseAEndToEnd` executes all steps
  in order (create → send → assigned → progress callback → SSE receipt
  → started → question → parked page → answer → running → continue
  command row → completed+artifacts → succeeded page + artifact row).
  Note: `started` precedes the question per the §6 machine (progress
  while assigned absorbs as history — arrival-ordered transitions).
- **Recovery proof:** `TestPhaseARestartRecovery` — fresh server boot,
  no truncate: pre-restart session works, parked answer applies, SSE
  replays full history from cursor 0, duplicate callback acks without
  reapply. (Old process left running only to avoid double-shutdown
  hazards; nothing in-memory is read after reboot by construction.)
- **Security proof:** cross-tenant 404s (stream/append/ingest/cancel/
  view/send), RLS FORCE + grants on all tables [CORRECTED below: FORCE
  was set without ENABLE — policies inert until closure item 3],
  forged/tampered/stale/
  wrong-key callbacks → 401, malformed/oversized → 400, unknown build →
  404 (no oracle), unauthorized credential denied (all-tools matrix),
  stale answers → 409 with current question, injection strings escaped,
  secret-absence on read paths, all transitions inside tools (handlers
  authenticate + invoke only).
- **Edge coverage:** exhaustive pure transition tables (6×6/5×5/4×4);
  every reachable tool path tested incl. supersede, terminal absorb,
  unknown-type opacity, artifact skip/dedup, defensive turn branch
  (SQL-staged), double cancel/answer refusals.
- **Audit:** framework emits per-invocation rows; proven present for new
  tools (`TestConversationToolsAudited`); correlation via build/event ids.

## Deviations from the governing plan (with reasons)

1. `message.send` replaces a bare append (§7 named no such tool): the
   atomic submit+ensure is required by recorded-failure semantics (above).
2. `conversation.get` added: the page needs builds+questions with the
   stream; one consistent read instead of N status calls.
3. `commands.payload` and `questions.(options, source_event_id)` columns
   added beyond the §7 sketch: the command log must store payloads to be
   a retry source; options need single-row render; provenance needs the
   source event link.
4. System stream notes for build open/cancel: Studio-initiated
   transitions have no gateway event, so every SSE viewer would otherwise
   miss them.

## Risks and unverified assumptions

- Per-build gateway `seq` is accepted but unused for ordering (stream
  seq rules Phase A replay); gap-tolerant render ordering is specified
  for the gateway side in Phase B.
- SSE poll cadence (1s default) is unmeasured under load; the acceptance
  load proof stays a Phase A item only in that cursor replay bounds cost
  — no fan-out infra until measured (per plan §21 item 9).
- P-E/P-G/C-track live proofs are Phase B/C gates, unchanged.

## Verdict

Phase A exit gate: **fully satisfied**. The conversation infrastructure
is durable, tested, and red-then-green at every correction in this pass.
The v1 envelope is frozen as implemented. **Phase B can safely begin**:
its data model (builds/turns/commands/events/stream) and protocol
(envelope/HMAC/dedup/ack) need no redesign — only the gateway binary,
provisioning, and the adapter proofs the plan already assigns it.

---

## Bounded closure record (2026-11-25, per plans/prompt9.md)

Review basis: `plans/adv_review_prompt8_phase_a.md` (two authorized
findings + new RLS finding + M1–M8). All three mandatory items corrected,
tested red-then-green, and re-gated. M1–M4 folded; M5–M8 recorded below
for their owning phases.

### Item 3 — RLS enabled on all Phase A tables

- Fix: `store/migrations/000016_enable_rls.up.sql` (ENABLE on all ten
  tables, FORCE retained) + down file restoring DISABLE-with-FORCE
  exactly (verified via catalog flags both directions).
- Catalog test: extended table list in `TestRoleAttributes`
  (`store/store_test.go`) to all 16 tables asserting both flags.
- Red/green: `TestRLSEnable000016` on scratch DB — v15 leg shows all ten
  at `enabled=false forced=true` (bug reproduced); v16 leg shows
  `true/true`. A temporary probe running the v16 assertions against v15
  failed 10/10 (captured, then removed).
- Behavioral: `TestPhaseARLSEnforced` — as `studio_app`, unscoped read
  sees 0 rows; armed read sees exactly the armed tenant's rows; armed
  own-tenant insert succeeds, cross-tenant insert refused. Temporary
  v15 probe showed the unscoped read exposing 2 rows across tenants
  (captured, removed).
- Evidence correction: the "RLS FORCE + grants" security claim above is
  superseded — enforcement was absent until 000016; tenant predicates
  held isolation meanwhile (review-audited, no boundary ever crossable).

### Item 1 — write-path secret sanitization

- Fix (`tools/ingest.go`): the verified envelope payload is scrubbed
  before the `events` insert and before question/artifact derivation —
  one choke point, so `events.payload`, `questions.body`/`.options`,
  `artifacts.manifest`, and everything downstream can never hold
  recognized spans. Signature verification still runs first in the HTTP
  handler on the original bytes (order preserved).
- Engine (`tools/scrub.go`, wired into `scrubValue`): token-aware
  substring patterns with left boundaries and credential-length minima
  (`sk-live`/`sk-test` + `sk_live_`/`sk_test_` + ≥20 chars, `AKIA`+16,
  `xox[baprs]`+10, `ghp_`/`gho_`/`github_pat_`/`gsk_` +20). Only the
  matched span is replaced; surrounding prose survives. Deliberately
  NOT exhaustive (documented limitation); no bare generic `sk-`.
- Read-path scrub retained as defense-in-depth (also covers pre-closure
  rows). User/system message bodies stay verbatim (operator's own
  transcript); `agent_session_id` stays an opaque resume correlator.
- Tests: `TestScrubSpansDetects` (13 shapes incl. embedded/at-edge/
  multi-secret), `TestScrubSpansPreserves` (16 false-positive guards:
  task-ids, mid-word `sk`, short fragments, uuids, code),
  `TestWritePathCanariesAreSanitized` (embedded canary via real ingest:
  DB `questions.body` + `events.payload` sanitized with prose intact;
  `build.status`, `conversation.get`, `conversation.stream` clean),
  `TestAgentCanaryEndToEnd` (signed callback → DB + rendered page +
  SSE bytes all clean). Red demonstrated by unwiring the one scrub line
  (FAIL), then restored.
- Residual (required disclosure): the pinned framework persists full
  tool inputs/results in `app.invocations` by design; tenant-RLS'd, no
  Studio read path exposes them, `activity.list` scrubs on read. No
  claim is made that zero unsanitized copies exist framework-wide.
  Miniapp untouched per boundary.

### Item 2 — ordered shutdown → restart

- Fix (`server/server.go`): `Close() error` with mutex-guarded
  exactly-once semantics and retained first error; all call sites
  handle it (`cmd/studio` logs, `New` error path warns, test cleanup
  `t.Errorf`s).
- Test (`TestPhaseARestartRecovery` rewritten): srv1 boots truncated →
  state parked → explicit `Close()` asserted nil → completion proven
  observably (srv1 pool refuses new acquires) → second `Close()` proves
  idempotency → srv2 boots untruncated → pre-restart cookie works →
  parked answer applies → SSE replays full history → duplicate callback
  acks without reapply. Green.

### M1–M4 (all completed)

- M1: `questionAnswer` reordered to build → turn → question (unlocked
  pre-read, lock, re-validate) matching ingest; `TestAnswerYieldConcurrency`
  (50 racy iterations) failed pre-fix at iter 2 with a deadlock and
  passes 3/3 post-fix (150 iterations, zero deadlocks). Supporting fix:
  `notFoundOrDbErr` helper — `require*` loaders previously masked ALL
  errors (incl. 40P01) as "not found", which hid the deadlock; now only
  `ErrNoRows` maps to not-found (applied to all loaders incl. P4's
  `requireApp`/`specGet`/`loadDeployment`; all callers already handle
  errors correctly).
- M2: `questionIDsFor` deleted (zero callers verified).
- M3: `gateway.Parse` rejects trailing data (+ table case), matching
  `tools.decode` strictness.
- M4: `000014` header "gapless" → monotonic/gap-tolerant wording,
  comment-only. Pre-approved by checksum inspection: golang-migrate
  v4.20.1 tracks `(version, dirty)` only, no migration checksums —
  verified in module source before editing. Round-trip + version gates
  confirm zero semantic drift.

### M5–M8 carried (not expanded, per authorization)

- M5 (SSE stream lifetime/budget) → Phase B non-functional hardening.
- M6 (`app.invocations` copies) → disclosed above; Phase B contract
  must forbid selecting invocation inputs into any response.
- M7 (`artifacts.manifest` unread in Phase A) → Phase D bundle review
  scrubs on read from day one.
- M8 (evidence nits) → counts in THIS section are exact (12 test
  packages including `gateway`, plus the test-less root = 13 output
  lines); "gapless" fixed (M4); restart caveat superseded by item 2.

### Closure gates (observed)

- `gofmt -l .` empty; `go vet ./...` clean; `go build ./...` clean.
- `go test -p 1 ./...` green on PG19 **19.0**
  (127.0.0.1:5434, Debian 19.0-1.pgdg13+1) and PG17 **17.12**
  (127.0.0.1:5435, Debian 17.12-1.pgdg13+1): 12 test packages ok, 0
  skips, 0 failures.
- `go test -race` green on `store`/`tools`/`server`/`gateway`.
- Migration round-trip green incl. v16 (down to v10 removes new tables
  incl. Phase A set; back up clean; catalog true/true throughout).
- No live Sprite contact (signed-envelope fake paths only); Miniapp
  tree clean; frozen docs untouched; no commits.

### Closure verdict

All three invariants demonstrated by corrected evidence: RLS enforced
(catalog + behavioral), secret canaries sanitized at write and absent
from every Studio-owned persistence and delivery surface, and genuine
ordered shutdown → restart recovery green. **Phase A closure complete;
Phase B may begin** under its existing plan and unchanged boundaries.
