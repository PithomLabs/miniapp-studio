## Goal

Execute `plans/prompt9_imp.md`: implement Phase B (Sprite Agent Gateway)
per the merged gateway plan §§9–11 + §10, ending with a real, recoverable
Builder Gateway proven against the Phase B acceptance target. Phase A is
accepted and closed; this plan covers Phase B only, in two gated parts:
fake-backed implementation first, live installation/recovery only after
explicit owner authorization.

## Success Criteria

- Fake-backed program green: Studio gateway client (`gateway/client.go`:
  sign/send/retry/wake/backoff) against a fake command handler; command
  dedup (per-build processed-key set, stored-response replay); durable
  on-disk outbox with ack tracking; supervisor + session manager +
  OpenCode adapter skeleton against scripted agent doubles; capability
  matrix v1 with executed-vs-unknown cells marked.
- Security/provisioning specified-as-code where local: dual-user layout
  scripts, HMAC key handling (memory + 0700 provisioning drop, never
  agent-visible), pinned `opencode.json` (exact rules,
  `guidance: false`, Builder instructions), install flow (static binary
  build + WriteFile + service create + verify-by-status-round-trip).
- Live segment (gated): real Builder Sprite provisioned, gateway
  installed as a service, signed `status` round trip, supervised
  OpenCode no-op turn, sleep → wake → `resumed` (or honest `failed`),
  callback received by Studio — with timings recorded and zero leftover
  resources (or an explicitly retained builder, owner-decided).
- Gates green throughout (fmt/vet/build/tests/race/migrations);
  evidence report extended; F1/G-01/P5 untouched; M5–M8 carried;
  no commits without authorization.

## Approach

Strict fake-first sequencing with a hard authorization gate between the
local program and any live action. Nothing in the fake-backed part
requires Fly credentials, OAuth, or network access beyond the module
proxy; the live part is specified as an exact runbook (commands, expected
outputs, cleanup) submitted for authorization before anything executes.
The gateway binary is built from this repo as a static Linux binary
(`GOOS=linux`, static) and deployed via the existing `fly` boundary —
no new control-plane client.

## Steps

### Part 1 — Fake-backed (no authorization needed beyond this plan)

1. Baseline: `git status`, gofmt/vet/build/tests green on the untouched
   tree; re-verify OpenCode/plugin pins against live sources (amendment
   requires confirming v1.19.6 still latest 1.x; record outcome, do not
   float).
2. Transport: `gateway/client.go` (Studio side) + gateway command
   handler skeleton; signed-command round trip against an in-process
   fake; wake-on-command as a client retry policy over `fly` errors.
3. Durability: per-build command dedup store, on-disk outbox (single
   file, fsync), Studio ack tracking, exponential backoff with jitter;
   red-then-green duplicate/out-of-order/loss tests.
4. Supervision: process supervisor (spawn/signal/reap, never
   auto-restart), on-disk session bookkeeping, `AgentAdapter` interface
   + OpenCode implementation against scripted doubles, NDJSON event
   translator, artifact collector (hash-then-send).
5. Security/provisioning as code: install script (build → WriteFile →
   service + verify), dual-user + key-drop layout, pinned
   `opencode.json`, restricted-grant parameter sheet; all exercised
   against `FakeSpriteManager` extended with service-state/
   sleep-simulation/kill hooks.
6. Evidence checkpoint: Part 1 report (what's implemented vs stubbed,
   matrix v1, live runbook draft) + full gates. STOP for live
   authorization — do not proceed to Part 2 in the same pass.

### Part 2 — Live (explicit owner authorization required)

7. Authorized live runbook only: provision Builder → install → signed
   `status` → supervised no-op turn → sleep → wake → `resumed` →
   Studio callback; record timings + transcript; prove C1–C3b carry
   over (config, scope bindings, `EACCES` boundary) where observable.
8. Cleanup to zero (or owner-retained builder, explicitly recorded);
   extend the evidence report; final Phase B report with exit-gate
   verdict.

## Validation Plan

- Part 1: unit (outbox fsync/replay, dedup, backoff, translator),
  integration (fake-sprite command→callback loop, install flow incl.
  service-state verification), failure (kill agent, restart gateway
  with full outbox, sleep mid-command, restore marker), security
  (forged commands, key/channel separation, permission posture) —
  plus full gates on PG19/PG17 and race on touched packages.
- Part 2: each acceptance bullet observed live with timings; no
  substitution of fake results for live proofs; cleanup attested by
  get-after-delete/list-empty.
- Boundaries: live steps never execute without the explicit GO in a
  later instruction; owner-held decisions (Fly org, model provider,
  builder retention) arrive with that authorization, not guessed.

## Risks / Open Questions

- One blocking question (see message): who executes the live runbook —
  owner-run in the authorized shell with results reported back, or
  owner-provided scoped credentials for direct execution? The Part 2
  plan shape depends on it (handoff checklist vs executable procedure).
- Sandbox has no Fly/maker network path (established): all live work is
  environment-gated regardless of authorization mechanics.
- Pin drift since verification (2026-11-24): step 1 re-verifies; if the
  ecosystem moved (npm publication, new 1.x, repo changes), the plan
  follows the amendment's own re-verification rules and reports before
  building on new facts.
- No other unknowns blocking Part 1: scope (§§9–11), acceptance target,
  and boundaries are explicit in the directive + merged plan.
