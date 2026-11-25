## Goal

Execute `plans/prompt8.md`: implement Phase A (Conversation Infrastructure)
in one continuous pass per the merged gateway plan, then deliver the §9
final report. Exit state: durable Studio-side conversations/builds/turns/
questions/callback-ingest/SSE proven against a deterministic fake Gateway,
with a Phase A evidence report in the working tree (no Git commit) — and
no live Sprite touched.

## Success Criteria

- §4 A–G implemented: §7 data model via fix-forward migrations `000011+`
  (all tables except `deployments_gateway`); `build.*` tools through
  `runtime.Service`; §8 Studio-side protocol (envelope, HMAC verify,
  replay window, dedup, opaque unknowns); authenticated callback
  receiver; SSE with durable replay; minimal conversation UI in existing
  templates; audit + redaction wiring.
- Decision D1 resolved with written rationale: a monotonic, durable
  conversation-stream cursor covering messages + events (per §4E —
  smallest correct schema extension; see Approach).
- Fake Gateway exercises the real callback endpoint and verification path
  (deterministic test-only keys; no production key distribution).
- §8 exit gate green: 15-step functional proof, restart/reconnect/dedup
  recovery proof, full security-proof matrix, `gofmt` + `go vet ./...` +
  `go test -p 1 ./...` + race on touched packages + migration up/down +
  PG integration, 100% new transition-edge coverage, SSE renders
  persisted rows only.
- §6 exclusions honored in full: no Sprites, OpenCode, plugin, OAuth,
  gateway service, adapter, live proofs, integrations, writes, F1/G-01
  movement, P5/Lead-to-Booking change, DBOS, frozen-doc edits, commits.
- §9 final report delivered (10 items + blocker protocol if any).

## Approach

Baseline-first, dependency-ordered (§7 sequence), fake-driven: no
implementation before the baseline gates are recorded on the untouched
tree; each layer lands behind its tests; the fake Gateway is built early
so every Studio behavior is proven through the same paths Phase B will
use. One design judgment is front-loaded to step 2 (Decision D1, below)
so no schema rework happens mid-pass.

**Decision D1 (SSE cursor).** The plan sketches per-conversation message
seqs + per-build event seqs; §4E requires one monotonic conversation
stream under a single `Last-Event-ID`. Options: (a) a unified
conversation-stream table (one row per renderable item, gapless seq);
(b) a global sequence assigned at ingest across both tables;
(c) composite cursor (type, seq) with merge-ordered replay. Evaluate
against: gapless monotonicity under concurrency, restart safety, query
simplicity, and smallest schema delta. Recommendation mechanism, not
pre-decision: prototype the cursor math against (a)/(b) in step 2,
document the choice + rejected alternatives in the evidence report, then
build once. Default fake location: `gateway/fake.go` (protocol-level,
exercising the real endpoint) unless step-2 inspection shows a better fit.

## Steps

1. Read governing sources (§2 items 1–8) in the live checkout; record
   `git status`; run baseline `gofmt -l .`, `go vet ./...`,
   `go test -p 1 ./...` (escalated shell for Docker PG).
2. Finalize Phase A schema + protocol decisions: §7 tables minus
   `deployments_gateway`, indexes/uniqueness/RLS/grants, D1 cursor
   design, v1 envelope freeze, test-only key provider shape.
3. Migrations `000011+` (up/down) + tenant/composite-FK/RLS protections;
   `LatestVersion()`-derived expectations; up/down round-trip tests.
4. `build.*` tools (`tools/build.go` + tests): create/append/answer/
   cancel/status; transition tables in tools, never handlers;
   idempotency per P4 conventions; three question kinds distinct;
   answers never self-authorize.
5. Protocol (`gateway/protocol.go`) + callback receiver
   (`server/gateway_callback.go`): sign/verify, replay window, tenant+
   build binding, `(build_id, event_id)` dedup, opaque unknowns, acks;
   forged/replayed/duplicate/cross-tenant/malformed matrix green.
6. Fake Gateway (`gateway/fake.go`): scripted signed event streams,
   continuation-command capture, sleep/kill/duplication hooks for later
   phases where cheap.
7. SSE (`server/sse.go`) + cursor replay: auth, initial state, live fan-out
   over persisted rows only, `Last-Event-ID` resume, mixed-traffic replay
   tests, restart recovery.
8. Conversation/question UI (existing templates + `server/conversation.go`):
   open/submit/history/progress/question/answer/continuation/reconnect —
   minimal, no new framework, no canvas.
9. Audit + redaction: tenant/actor/ids/action/result/timestamps on new
   paths; secret-absence tests for logs/SSE/persisted messages.
10. Full gates (PG19 suite, PG17 DB spot-check, race on touched
    packages), 15-step fake proof end to end, recovery + security proofs,
    evidence report at `docs/gateway-phase-a-evidence.md`
    (naming follows the gateway plan's own convention), §9 final report.

## Validation Plan

- Gates: `gofmt -l .` empty, `go vet ./...` clean,
  `go test -p 1 ./...` green, `-race` on `store`/`tools`/`server`/
  `gateway`, migration up/down round-trip, PG19 + PG17.
- Proofs executed (not inspected): 15-step fake script; Studio restart
  mid-question + SSE resume + answer-still-applicable + duplicate-ack;
  security matrix (§8: cross-tenant, RLS, forged, replayed, unauthorized/
  expired answers, inert rendering, secret absence, no handler bypass).
- Coverage: every new transition edge tested legal + illegal; SSE output
  diffed against persisted rows in replay tests.
- Boundaries: `git status` shows only intended new/modified files; frozen
  docs byte-identical; Miniapp untouched (§14); zero live-Sprite contact
  asserted (fake-only; no `fly` adapter constructor runs outside existing
  tests); no commit created.

## Risks / Open Questions

- Root `AGENTS.md` is absent (verified earlier); execution falls back to
  user-level instructions + project memory, recorded in the evidence
  report.
- Docker/PG access requires the escalated shell (established pattern);
  no new environment needs anticipated.
- D1 is the only design judgment with schema impact; it is quarantined
  in step 2 with a documented rationale requirement, so mid-pass rework
  should not occur.
- No clarifying questions blocking a GO: scope (§4), exclusions (§6),
  sequence (§7), and acceptance (§8) are explicit; the evidence-report
  path follows the plan's own convention. If the owner wants a different
  report path or a pre-pass D1 review, say so before GO — otherwise the
  pass runs uninterrupted per §7.
