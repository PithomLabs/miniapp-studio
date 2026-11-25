# Adversarial Review — Gateway Phase A (per plans/prompt8_phase_a_adv.md)

- Scope: Phase A conversation infrastructure only (migrations 000011–000015,
  `tools/` conversation/build/question/ingest, `gateway/protocol.go`,
  `server/` callback/SSE/conversation handlers + templates).
- Read-only review; no repository file created, modified, or deleted except
  this report. No closure work performed.
- Reviewer posture: security and reliability engineer reporting to the Senior
  Architect, adversarial at phase boundaries.
- Date: 2026-11-24. Studio root: `/home/chaschel/Documents/go/miniapp-studio`.

## Verdict

**PHASE A NOT YET ACCEPTED — bounded closure outstanding.**

Both authorized findings reproduce exactly as described — the closure task in
`plans/prompt8_phase_a_adv.md` has not been performed (code, tests, and
`docs/gateway-phase-a-evidence.md` are unchanged on both points, verified by
reading and by live probes). Additionally, this review found one **new MAJOR**
the prior review missed: **RLS is not enabled on any of the 10 Phase A tables**
(`FORCE` without `ENABLE` — the exact trap migration 000004 documents for the
original tables). Tenant isolation currently rests solely on explicit
`tenant_id` predicates, which I audited exhaustively and found complete, so
nothing is cross-tenant-readable today — but the mandated second layer is
absent and the evidence claim of "RLS FORCE + grants on all tables" as security
proof is materially misleading at the enforcement layer.

Recommendation: execute the bounded closure with **three** items (the two
authorized findings + the RLS ENABLE migration and catalog test), re-run the
gates, update the evidence doc — then Phase A is accepted and Phase B begins
with no further review cycle. The RLS fix is a trivial migration in the same
shape as the authorized work, not a reopening of architecture.

## Status of the two authorized findings (both verified OPEN)

### Finding 1 (MAJOR): conversation-body secret handling — OPEN, confirmed live

The requirement: secrets must not appear in persisted agent messages, logs,
SSE responses, or rendered conversation history.

Current state, traced end to end and proven with a live `sk-live-` canary
driven through the real tool path on a disposable database:

- **Agent → persistence (verbatim, 3 surfaces).** `agent.question` /
  `agent.proposal` / `agent.authorization_request` payloads persist verbatim
  into `studio.events.payload` (`tools/ingest.go:78-83`), and the question
  text/options are denormalized verbatim into `studio.questions.body` /
  `.options` (`tools/ingest.go:196-213,258-264`). Nothing on the write path
  redacts. (Phase A writes `messages.body` only with `user` and `system`
  roles — verified all four `appendMessage` call sites — so agent prose
  reaches the transcript exactly via event payloads and question bodies.)
- **Persistence → reads (verbatim for question bodies).**
  `buildStatusView` scrubs only `options` (`tools/build.go:396-398`);
  question `body` flows verbatim through `conversation.get` →
  `shapeBuilds` (`server/handlers_conversation.go:143-146`) → template, and
  through the 409 conflict page. Live probe: canary present in
  `build.status` output (`leaks: true`), `conversation.get` output
  (`leaks: true`), and `conversation.stream` output (`leaks: true`, via the
  event item).
- **Scrubper limitation compounding the gap.** `scrubValue`
  (`tools/activity.go:116-143`) redacts denylisted *keys* and string values
  that *start with* a known prefix. A mid-string secret (`"paste
  sk-live-… to proceed"`) passes through even MAP payload scrubbing —
  demonstrated live: the stream event item carrying it was unredacted.
  The committed test `TestAgentContentScrubbedOnRead` only exercises
  key-positioned secrets and a benign question body (`"Pick one"`), so
  neither the question-body path nor embedded secrets are pinned.
- **Legs that hold.** SSE emits the same shaped data (no *additional* leak
  beyond the read path); page rendering is autoescaped and the EventSource
  JS uses only `textContent`/`createTextNode` (XSS-safe, secret-visible);
  logs carry metadata + static error strings only (callback handler,
  `dbErr` SQLSTATE mapping, `decode` without excerpts); per-invocation
  framework audit rows carry `details = nil` (verified all call sites in
  pinned `runtime/service.go`); `commands.payload` (queued assign/continue,
  answers included) has no Phase A read path (write-only, verified by
  search).

Residual that closure must disclose, not fix in Studio: the framework
persists full tool inputs and results in `app.invocations`
(`input`/`result` columns, pinned `runtime/store.go:62-67,226-227`), so
verbatim bodies/payloads exist there by framework design (tenant-RLS'd, no
Studio read path exposes them; `activity.list` scrubs details). Any closure
claim of "no unsanitized copy" must scope itself to Studio-owned tables and
name this explicitly.

### Finding 2 (MINOR): genuine restart recovery — OPEN, confirmed as described

`TestPhaseARestartRecovery` (`server/conversation_e2e_test.go:150-196`) boots
`srv2` while `srv1` is still running and never shuts `srv1` down (both closes
defer to `t.Cleanup`). It proves fresh-instance reconstruction against live
rows — sessions, parked answer, SSE replay from cursor 0, duplicate ack —
but not a clean shutdown → cold boot lifecycle. The suite passes (verified in
my gate run), so the reconstruction half is green; the lifecycle half
remains unproven as the prior review states.

Feasibility note for closure (code-level, not live-verified since I may not
edit tests): `Server.Close` (`server/server.go:221-228`) discards shutdown
errors and `pgxpool.Close` is idempotent, so an explicit mid-test
`srv1.Close()` followed by the cleanup re-close is tolerated in practice;
making `Close` `sync.Once`-idempotent would remove the hazard the evidence
doc cites. Also note `bootServerOn` rotates the app-role password on every
boot (`server/server_test.go:68-74`) — harmless once srv1 is down first,
another reason the ordered lifecycle matters.

## New finding: RLS never enabled on Phase A tables (MAJOR)

- Severity: MAJOR. Not a blocker to eventual acceptance (no boundary is
  crossable today — see containment), but it must join the bounded closure:
  it is the same one-line-per-table fix shape, and the security-proof claim
  in the evidence doc is false without it.
- Location: `store/migrations/000011_conversation.up.sql` through
  `000014_stream_artifact.up.sql` — every new table gets `FORCE ROW LEVEL
  SECURITY`, none gets `ENABLE ROW LEVEL SECURITY`. Only `000004` (original
  six tables) enables. The catalog test
  (`store/store_test.go:266-271`) asserts both flags for the original six
  tables only, which is why this slipped through.
- Live proof (disposable PG16, trust auth): catalog shows all 10 Phase A
  tables `enabled=false force=true` (originals `enabled=true force=true`);
  as `studio_app`, unscoped `SELECT count(*) FROM studio.messages` returns
  **both tenants' rows**, and arming `app.tenant_id='tA'` changes nothing
  (policy inert).
- Containment (why MAJOR, not BLOCKER): I audited every production
  read/write on all 10 tables — all carry explicit `tenant_id` predicates
  (including both sides of the stream JOINs); no handler queries studio
  tables directly (all through tools); cross-tenant behavioral tests pass.
  No untrusted principal (user, callback, agent) can issue arbitrary SQL,
  so no trust boundary is crossable. The missing layer matters the moment
  any future tool forgets a predicate — which is precisely what the layer
  exists to catch.
- Durable correction: migration `000016` enabling RLS on the 10 tables +
  extend the catalog test table list (fails before, passes after) +
  correct the evidence claim. Optionally add one behavioral test with an
  intentionally unscoped owner-asserted query proving policy enforcement.
- Regression test: extend `TestRoleAttributes`-style loop with the 10 table
  names asserting `relrowsecurity = true`. Fails today, passes after.

## Criterion assessment (the six review lenses)

1. **Architectural invariants — hold.** Ownership correct (migrations own
   DDL; tools own transitions; handlers authenticate + invoke only —
   verified no direct studio queries in non-test server code); tenant
   isolation holds at the predicate layer on every query (exhaustively
   audited); execution boundaries hold (Class A tools via public registry
   API, no framework internals; `workspace_id` re-validated by credential
   + tenant-scoped build lookup, never trusted alone); scope holds (no
   provisioning/OpenCode/F1/G-01/P5 movement in Phase A paths; Sprite
   calls exist only in pre-existing `tools/deploy.go`).
2. **Security — holds except Finding 1 (+ RLS layer gap).** HMAC envelope
   verified (version/kind/shape/freshness/constant-time compare;
   `workspace_id` covered by the MAC); forged/tampered/stale/wrong-key →
   401; unknown build → 404 without oracle; 1MB cap; sessionless callback
   by design with no cookie surface; CSRF via `requireOrigin` on all
   mutating browser routes; injection escaped at every render layer
   (server autoescape + DOM-safe JS). Secret handling is the open hole
   (Finding 1), and RLS-absent is the missing backstop (new MAJOR).
3. **State integrity — holds.** Dedup on `(tenant, build, event_id)` PK
   with correct `ErrNoRows`-on-conflict branch (duplicates ack without
   stream/transition reapply); row-locked, table-guarded transitions;
   terminal/out-of-order absorb without wedging; supersede semantics
   correct; advisory xact lock + MAX+1 allocation (monotonic, gap-tolerant
   replay — see minor M4 on the "gapless" comment); per-attempt framework
   keys avoid recorded-failure stranding (documented from live observation).
4. **Recovery — half-proven (Finding 2).** Reconstruction proven green;
   ordered shutdown→boot lifecycle unproven. Supporting facts verified:
   no in-memory broadcaster (SSE polls persisted rows), locks are
   xact-scoped (no crash-stuck state), idempotency keys are PKs, Close is
   double-call tolerant in practice.
5. **Evidence — executed and mostly demonstrative, with two integrity
   dents.** I re-ran everything (record below): full suite green, zero
   skips, race green. Negative cases exist across the boundary (cross-
   tenant, forged callbacks, stale answers, double cancel/answer,
   unknown-type opacity). Dents: (a) the RLS "security proof" claim is
   false at the enforcement layer (new MAJOR); (b) the secret tests pin
   the *current* behavior including its gaps (benign question body,
   key-positioned canaries only) — they demonstrate the scrubper, not the
   requirement.
6. **Scope discipline — holds.** No Gateway v1 change needed by findings;
   no F1/G-01 movement; no P5/Lead-to-Booking; no new infra or workflow
   engine; Miniapp untouched (clean tree, pin `73351a8` is an ancestor of
   the checkout); everything uncommitted (this repo has no commits at all).

## Minor findings (record and fold into closure where cheap)

- **M1 — Lock-ordering deadlock risk (answer vs yield).**
  `questionAnswer` locks question → build (`tools/question.go:69,102`);
  ingest locks build → turn → question rows
  (`tools/ingest.go:69,214-237`). Concurrent answer + superseding yield
  can deadlock (40P01 → `dbErr` → "unavailable"): the callback side
  retries via 503 correctly, but the UI side surfaces 400 and the
  recorded-failure semantic wedges that opkey until page reload mints a
  fresh one. Fix: lock build first in `questionAnswer` too. Low rate,
  self-healing — minor.
- **M2 — `questionIDsFor` is dead code** (`tools/conversation.go:284-298`,
  zero callers). Remove or wire it.
- **M3 — `gateway.Parse` ignores trailing data** (single `Decode`,
  unlike `tools.decode`'s two-decode strictness). Harmless in practice —
  verification re-canonicalizes the parsed struct, so trailing bytes are
  dropped, never interpreted — but inconsistent for a "fixed envelope".
  One-line hardening.
- **M4 — "Gapless" sequence comment is wrong**
  (`000014_stream_artifact.up.sql:1-6` header). MAX+1 under an xact lock
  gaps on abort/rollback; the evidence doc states the honest invariant
  (monotonic, cursor needs greater-than only). Fix the comment so Phase B
  gateway-side ordering work doesn't assume contiguity.
- **M5 — SSE streams never terminate server-side**
  (`server/sse.go:79-87`: idle ping resets the counter forever). Fine for
  Phase A scale; Phase B should add a max stream lifetime / concurrent-
  stream budget per operator. Improvement.
- **M6 — Framework `invocations` copies (disclose, don't fix).**
  Full inputs/results persist verbatim in `app.invocations` by framework
  design (tenant-RLS'd; no Studio read path). Closure evidence must name
  this as the sanctioned residual, and the Phase B contract must forbid
  any future tool from selecting invocation inputs into a response.
- **M7 — `artifacts.manifest` is agent-supplied JSON with no Phase A read
  path** (only counts asserted). Latent secret surface for Phase D
  bundle review — scrub-on-read from day one there.
- **M8 — Evidence nits:** "13/13" counts 12 test packages + the
  test-less root; "gapless" contradiction (M4); the restart-test caveat
  is honest and matches the code.

## What I accept without further review

Per the prior review, and re-verified independently: unified stream + D1
references-only shape, atomic `message.send`, callback idempotency
layering (fresh attempt keys + event-id dedup), HMAC verification and
unsigned-route scoping, transition tables and guards, terminal/out-of-order
absorb semantics, authorization-answer strictness, CSRF + session wrapping
on all mutating routes, XSS-safe rendering at both layers, `message.send`
concurrency serialization, down-migration round trip (suite-green here
too), and audit-row presence. Gateway-side sequence ordering and live
OpenCode proofs correctly remain Phase B/C.

## Verification record

Environment: sandbox shell, Go 1.27.3, `GOTOOLCHAIN=local`, module/build
caches in `/tmp` (default cache is read-only in sandbox); disposable
PostgreSQL **16.15** (user-space extract, fresh `initdb`, trust auth,
`127.0.0.1`, destroyed after each run). The evidence cites PG19/PG17 on
ports that are unreachable from this sandbox (connection refused, no
`TEST_DATABASE_URL` set); PG16-substitution is the disclosed deviation —
plain DDL/RLS/pgx features, no version-sensitive constructs involved.
Each bash tool call runs in a private network namespace, so Postgres and
its clients ran inside single calls.

| Check | Result |
| ----- | ------ |
| `gofmt -l .` | clean (empty), exit 0 |
| `go vet ./...` | clean, exit 0 |
| `go build ./...` | clean, exit 0 (`-buildvcs=false`; repo has both `.git` and `.hg` stubs) |
| `go test -p 1 -count=1 ./...` on disposable PG16 | **12/12 packages ok, 0 skips** (root has no test files; evidence "13/13" counts it) |
| `go test -race -p 1 ./store/ ./tools/ ./server/ ./gateway/` on PG16 | all 4 ok |
| Live probe: RLS catalog flags (10 Phase A tables) | `enabled=false force=true` on all 10; originals `true/true` |
| Live probe: app-role unscoped `SELECT count(messages)` | 2 rows across 2 tenants (policy inert; tenant-setting arming changes nothing) |
| Live probe: `sk-live-` canary in question body via real tool path | persisted verbatim; returned verbatim by `build.status`, `conversation.get`, `conversation.stream` |
| Framework source checks (pinned `73351a8` module cache) | audit details `nil` on per-invocation rows; full input/result in `invocations`; `Shutdown` error-swallowing path; `Close` double-call tolerance by inspection |
| Miniapp repo | working tree clean, pin is ancestor of checkout — untouched |
| Not executed | PG19/PG17 runs (environment-blocked, recorded, not counted); any code change (forbidden by instruction) |

Probe code lives outside the repo at `/tmp/phasea_probe` (drives
`tools.NewImplementation` + a migrated disposable DB); it modifies
nothing in the working tree.

## Phase B contract additions (from this review)

1. Secrets: question bodies + event payloads must be redacted before
   they reach persisted agent-message surfaces and every read/delivery
   path — including substring/embedded secrets, not just key- and
   prefix-positioned ones (M1-context: extend patterns or redact at
   write with provenance).
2. Treat `commands.payload`, `artifacts.manifest`, and
   `app.invocations` input/result as sensitive-by-content; the Phase B
   sender and Phase D bundle review scrub on read from day one; no tool
   ever selects invocation inputs into a response.
3. RLS ENABLE on the Phase A tables (closure item) + keep the catalog
   test list in lockstep with every future migration — no new table
   without an ENABLE assertion.
4. Callback route stays sessionless/HMAC-only; production must never
   configure `gateway.StaticProvider` (fail-closed 503 today — keep it).
5. Consistent lock ordering (build first) wherever a transaction touches
   build + question/turn rows.
6. No reliance on stream-seq contiguity anywhere (gateway-side ordering
   included); cursor semantics are greater-than only.

## Final decision

The two authorized findings stand verified and open; closure has not been
performed. The bounded closure task in `plans/prompt8_phase_a_adv.md` should
now run with **three** items — Finding 1 (secret handling, covering
question bodies, all read/delivery paths, and embedded-secret patterns),
Finding 2 (ordered shutdown→boot restart proof), and the new RLS ENABLE
migration + catalog test + evidence correction. Minors M1–M4 are cheap
enough to fold in; M5–M8 are record-and-carry.

Once that closure lands and the Phase A gates re-pass, Phase A is accepted
under the file's own terms and Phase B begins. No general re-review is
needed — but the acceptance gate must be the *corrected* evidence (with the
RLS claim fixed and the invocations residual disclosed), not the current
document. Per instruction, I changed nothing and await the next step.
