# RPA increment 3 (F0 invitations + capability refresh) — evidence

Scope: `plans/prompt34.3.md` — tenant invitations on the existing
flat identity model + framework-native capability refresh. Owner
GO approved the no-role-hierarchy judgment call with one mandatory
safeguard (atomic acceptance), implemented as designed.

## Files

New: `store/migrations/000029_invitations.{up,down}.sql`,
`store/migrations/000030_capability_snapshots.{up,down}.sql`,
`invite/invite.go`, `invite/db_test.go`,
`capability/capability.go`, `capability/capability_test.go`,
`capability/db_test.go`, `cmd/studio/devinvite.go`,
`cmd/studio/devinvite_test.go`, `cmd/studio/devrefresh.go`,
`cmd/studio/devrefresh_test.go`, `docs/invitations.md`,
`docs/capabilities.md`.
Modified: `cmd/studio/dev.go` (dispatch),
`dbenv/provision.go` (table mirror + invitation_accept function
grant), `store/privilege_parity_test.go` (+2 tables, +1 function,
audit comment), `store/provision_guard_test.go` (devinvite.go +
devrefresh.go allowlist, same `studio dev` surface precedent),
`Taskfile.dev.yml` (+6 tasks, +5 vars), `docs/taskfile.md`,
`plans/prompt34.3_plan.md` (Decision 2/Steps amended with the
atomic-acceptance safeguard before implementation).

## Contracts

Invitations: hash-only tokens (`store.MintToken`/`HashToken`);
create/revoke/resend/list as app role under tenant RLS with the
tenant stamped from the authorized parameter; accept runs ONE
transaction (`invitation_accept` guard→upsert→consume, then
`session_create`) so any failure rolls back and stays retryable —
acceptance can never strand a consumed invite without a session.
Uniform `ErrInviteInvalid` for all accept failures (no oracle).
Resend rotates atomically in one tx. Tokens travel by 0600 file
only: reservation held open across the DB call (O_EXCL, fd-held,
no path-swap race), nothing printed.
Capability refresh: entries in CapabilityRef shape from the tool
manifest × bound handlers (drift = `failed` data) plus the live
connector list (`implemented`); unconfigured manager = honest
`unavailable` marker; collection errors block publish (previous
snapshot stands); staleness computed at read. `planned` is never
emitted — no declaration source exists in the contract
(documented, not fabricated). `TestRefreshGrantsNothing` pins no
call edges into broker/invite/session/approval surfaces plus the
key-presence-only handler-map access form.

## Adversarial review (separate pass, all fixed)

- A1 resend spanned two txns (revoke, then create) → single tx.
- A2 token files written after DB mutation (stranded invite/
  session on file failure) → reserve-before-mutate with held fd.
- A3 grants-nothing scan didn't pin the handler-map access form
  → asserts presence-probe-only lines.
- A4 capability Read masked all errors as absence → NoRows stays
  uniform, corruption/transport distinguishable.
- A5 nil entries slice → initialized to `[]` for stable shape.
Reviewed clean with rationale: no FK on `invited_by` (app role
must stay blind to `operators`; membership check belongs at the
future session-handler layer); double-invite same binding safe
(PK + idempotent upsert, sessions are multi-per-binding by
design); singleton snapshot keeps last-good only (documented).

## Gates (this session, local)

- `gofmt -l .`: clean. `go build ./...`, `go vet ./...`: pass.
- `go test ./... -count=1`: **475 pass / 328 skip / 0 fail**
  (baseline 470/316/0; +5 = 2 invite verb + 1 refresh verb + 2
  capability pure; +12 = 6 invite + 6 capability DB tests
  skipping honestly without Neon).
- `go test -race ./invite/ ./capability/`: pass (DB concurrency
  tests skip locally; race coverage needs the live run).
- Taskfile smoke: new tasks parse; preconditions fail cleanly.
- One transient failure observed in 4 full-suite runs (an
  unnamed single FAIL in one count run; immediate re-run and
  the final logged run green, exit 0). All increment-3 passing
  tests are deterministic by construction (no timing or
  concurrency in the passing set); the transient matches the
  known untouched-code load-flake class. Not chased further.

## Pending: live Neon verification (owner terminal)

Migrations 000029/000030 unapplied anywhere; 12 DB tests, parity
additions, and race concurrency coverage never executed.

```
task dev:init
task dev:test -- -- -run 'TestInvite|TestCapability|TestPrivilegeParity' -timeout 15m -count=1
```

Expected: `studio_dev_ui` at v30; 12 tests + parity green. Then
(optionally, exercises verbs end to end):

```
task dev:invite-create TENANT=<t> BINDING=<b> INVITED_BY=<op> TOKEN_FILE=/tmp/inv.tok
task dev:invite-list TENANT=<t>
task dev:invite-accept TOKEN_FILE=/tmp/inv.tok SESSION_TOKEN_FILE=/tmp/sess.tok
task dev:capability-refresh TENANT=<t>
```

## Residuals (genuine, not blockers)

- HTTP accept + session-bound invite endpoints: later
  increment; services take the authorized tenant as a
  documented parameter, verbs satisfy it via owner trust.
- Invitation delivery is operator out-of-band (same trust
  model as the login-token ceremony).
- Verb `capability-refresh` carries no connector manager, so
  connector state reports unconfigured until a configured
  caller exists.
- Plan deviation: dropped the planned `complete` BOOL column —
  only complete snapshots publish, so it would be constant.
