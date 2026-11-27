# B2-01B Reality-First Capability Discovery

**Status:** Discovery complete. Investigation only — no live operations
performed, no code changed, no credentials touched.

**Evidence tier:** Specified / code-verified throughout. Nothing in this
document is live-tested; live proof is the B2-01B execution gate.

## Sources (all read-only)

- sprites-go v0.2.2 module source (pinned SDK; confirmed latest —
  no newer release exists on the Go proxy).
- `sprite` CLI v2026-11-03 help output (installed locally at
  `~/.local/bin/sprite`). Its command list is complete and contains
  no power verbs: create/list/info/use/console/file/config/
  checkpoint/restore/destroy/exec/attach/sessions/api/proxy/url/
  login/org/auth.
- Fly vendor docs fetched 2026-11-26:
  - https://docs.fly.io/sprites/concepts/lifecycle
  - https://docs.fly.io/sprites/working-with-sprites
  - https://docs.fly.io/sprites/keeping-sprites-running
- Prior repo knowledge: `docs/gateway-phase-b-live-runbook.md` §11
  (sleep/wake/resume observation procedure) and
  `docs/gateway-phase-b-evidence.md` (warm/cold recorded as
  third-party/unverified — now superseded by vendor docs below).

## Classification

| Verb | Class | Real mechanism |
|---|---|---|
| provision | Native | SDK `CreateSprite` / `sprite create`. |
| destroy | Native | SDK `DeleteSprite` / `sprite destroy`. |
| sleep | Documented, no verb | Automatic: ~30s idle → warm pause
  (billing stops, 100–500ms resume, processes frozen in memory);
  prolonged idle → cold (memory dropped, 1–2s resume, processes
  start fresh). Cannot be forced. Three sources agree no power
  endpoint exists: SDK source, CLI command list, vendor docs
  ("You don't choose between them and you can't see the
  transition"). |
| wake | Documented, no verb | Any request resumes the sprite.
  Studio wakes via a cheap exec and measures latency. |
| connect | Documented (Tasks API) | Create a named hold (`studio-mgmt`)
  via the in-sprite Tasks API (unix socket `/.sprite/api.sock`,
  `POST /v1/tasks`) over exec — a genuine server-side hold that
  keeps the sprite awake, with an expiring backstop. |
| disconnect | Documented (Tasks API) | `DELETE /v1/tasks/studio-mgmt`;
  the sprite is free to pause. A real close, and it stops
  compute billing. |

## Consequences for the contract

- The adapter can implement Attach/Detach with the existing SDK
  `Exec` verb (curl over the Tasks socket). No endpoints are
  invented. Only SetPower stays unbound — correctly, because
  sleep/wake are observe/act, not calls.
- The B2-01A rule "sleep-from-connected is illegal" turned out to
  be physical reality — a live Tasks hold inhibits pause — not
  just Studio policy.
- The B2-01A contract needs no redesign. Real provider
  capabilities determine only the provider bindings.

## Receipt persistence determination

No existing table fits: `studio.commands`, `studio.events`, and
`studio.approvals` are all build-scoped with cascading FKs to
`studio.builds`, while lifecycle receipts are tenant+sprite scoped
and build-less (B2-01B: App none, infra only). Forcing them into a
build-scoped table would mutate existing invariants for a poor fit.

Minimal addition: ONE table, `studio.sprite_lifecycle_receipts`
(tenant_id, key, sprite, transition, from_state, to_state, at,
approver, auth_ref, provider, latency_ms), tenant RLS mirroring the
000013 pattern, grants to studio_app, `UNIQUE (tenant_id, key)` for
cross-restart replay dedup. Current position per sprite derives as
the latest `To`, so no second table is needed. One additive
migration in B2-01B execution, whose card already authorizes local
PG lifecycle receipts.

## Open questions for live discovery (in order)

All are read-only or near-read-only, but need a live sprite, so
provisioning approval comes first:

1. Exact `Status` vocabulary: what do `GetSprite` / `sprite list
   --json` report across active/warm/cold?
2. Do control-plane reads (`GetSprite`, `ListSessions`) reset the
   idle timer or wake a paused sprite?
3. Exec wake latency: wall-clock cost of waking via `exec true`
   from warm and from cold?
4. Warm process survival: do exec'd PIDs survive a ~40s idle? Two
   vendor docs conflict (lifecycle: warm preserves processes
   mid-thought; working-with-sprites: exec'd processes stop on
   sleep). Resolve by observation, not by picking a doc.
5. Tasks API round-trip via exec: create, verify hold inhibits
   pause across a 60s untouched idle, delete, verify release.

## Smallest viable live-proof sequence

Prerequisite (separate, never proof): owner-terminal `sprite login`
with org `isagani-mendoza` selected.

1. Task-level reads: capture the `Status` vocabulary baseline.
2. Owner-GO provision of ONE sprite through Studio's adapter;
   provision receipt recorded.
3. Owner-GO connect: create the Tasks hold; connect receipt.
4. 60s untouched idle proving the hold inhibits pause (no
   intermediate reads — reads may disturb the observed idle).
5. Owner-GO disconnect: delete the hold; disconnect receipt.
6. 45s untouched idle, then a single status read proving warm
   pause; sleep observation recorded.
7. Per-operation-GO wake via `exec true`: latency recorded,
   warm/cold classified by survival of a marker `sleep` PID
   (detached TTY session; no agent runtime — that is B2-05).
8. Separately authorized destroy bound to the exact sprite;
   absence verified by list; destroy receipt. Teardown complete.

Recovery: stop on any surprise per the runbook STOP conditions;
destroy orphans only under their own explicit authorization;
already-recorded receipts stand as evidence; reconcile via
Describe.

## Approvals required before execution

- Per-operation owner GO: provision, connect, disconnect, wake.
- Own explicit authorization bound to the exact sprite: destroy.
- Reads are task-level. No provisioning or destructive operation
  may run until its respective approval is given.
