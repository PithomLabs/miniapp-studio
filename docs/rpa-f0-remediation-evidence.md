# RPA F0 Remediation Evidence (prompt36)

Remediation of the six HIGH/MEDIUM findings from
`plans/adv_review_prompt35.md` (verdict: NOT APPROVED), plus LOW
triage, adversarial re-review, and live-gate preparation.

## Worktree identity

Base commit `df25e17` ("prompt 34 inc 3") plus uncommitted
remediation work. Modified: `.gitignore`, `Taskfile.dev.yml`,
`broker/{broker,store,exchange,kek}.go`,
`broker/{db,exchange,crypto}_test.go`, `capability/capability_test.go`,
`cmd/studio/{dev,devbroker,devinvite,devserve,bind,migrate,provision}.go`,
`cmd/studio/{devinvite,devserve}_test.go`,
`dbenv/{neon,provision}.go`, `dbenv/neon_test.go`,
`docs/{broker,invitations,secrets,taskfile}.md`,
`invite/db_test.go`, `secrets/{load,registry}.go`,
`secrets/secrets_test.go`, `server/config.go`,
`store/{provision,privilege_parity_test}.go`,
`store/provision_guard_test.go`, `plans/prompt36_plan.md`
(amended per owner verdict before implementation).
New: `store/migrations/000031_offboard_correction.{up,down}.sql`,
`000032_broker_revocation.{up,down}.sql`, `secrets/{secure,dsn}.go`,
`secrets/{secure,dsn}_test.go`, `invite/offboard_test.go`,
`cmd/studio/poolredact_test.go`. Removed: `secrets/redact.go`
(F35-09). No commits or pushes (per boundaries).

## Dispositions

| Finding | Severity | Disposition |
|---|---|---|
| F35-01 stale-invite reactivation | HIGH | FIXED + 9 regression tests (live-pending) |
| F35-02 brokerPool nil-cancel panic | MEDIUM | FIXED + 5 regression tests (green) |
| F35-03 DSN leakage | MEDIUM | FIXED at CLI + dbenv source + 4 canary tests (green) |
| F35-04 OAuth redirect exfil | MEDIUM | FIXED + 3 regression tests (green) |
| F35-05 revocation races | MEDIUM | FIXED (versioned) + 5 regression tests (live-pending) |
| F35-06 file TOCTOU | MEDIUM | FIXED (OpenSecure) + 5 regression tests (green) |
| F35-07 stale auto-revoke | LOW | FIXED with F35-05 (conditional revoke + test) |
| F35-08 owner-DSN outside contract | LOW | PARTIAL: secure audited reads; full registry routing deferred (owner GO needed) |
| F35-09 dead Redact | LOW | FIXED by removal + docs |
| F35-10 unused KEK_FILE entry | LOW | FIXED by removal + docs |
| F35-11 TTL NaN/Inf | LOW | FIXED + test (green) |
| F35-12 untrimmed refresh | LOW | FIXED (trim + empty-reject) |
| F35-13 error-code echo | LOW | FIXED with F35-04 (allowlist + test) |
| F35-14 DELETE grants | LOW | FIXED (revoked in 000031/000032 + mirror + parity) |
| F35-15 docs drift | LOW | FIXED (secrets.md rewritten sections); gateway half REBUTTED (warning exists, server.go:170) |
| F35-16 loser errors | LOW | FIXED (assertions added) |
| F35-17 scan gaps | LOW | FIXED (needles added); aliased-import residual documented |
| LOW count | — | CORRECTED to 11 (F35-07…17 enumerated; none invented) |
| R36-01 secrets/ git-ignored (new) | HIGH integrity | FIXED (.gitignore narrowed; package committable; no commit per boundaries) |
| R36-02 migrate DSN raw read (new) | LOW | FIXED (ReadSecureFile) |
| R36-03 CheckEndpoint unredacted (new) | MEDIUM-equivalent | FIXED (redact at dbenv source + test) |
| R36-04 dbTestPackages rotted (new) | HIGH verification | FIXED (packages added + completeness test); prior F0 live claims CORRECTED below |

## Root causes and corrections

- **F35-01**: accept unconditionally reactivated. 000031 gates
  accept on operator-active, never flips false→true, adds
  owner-only `operator_offboard` (revoke-first ordering), and a
  reactivation trigger revoking outstanding invites on every
  false→true path. Lock-free by design: an explicit FOR UPDATE
  scheme deadlocks when the operator row is absent (documented
  in the migration); atomicity comes from guard re-checks,
  single-tx rollback, and session_create's active backstop. New
  `dev operator-offboard` verb + Taskfile entry.
- **F35-02**: deferred cleanup called the nil-valued named
  `cancel` return. Cancel now lives in a separate variable;
  all 10 DB verbs (incl. new offboard) covered by one
  no-panic/error-preserved test.
- **F35-03**: decoded-only replacement + unredacted parse
  errors. Structural spans (raw userinfo, parts, decoded forms,
  full DSN, longest-first) now live in `secrets.RedactDSNText`,
  applied at the CLI layer AND at the dbenv source
  (CheckEndpoint/PoolConfig), closing pre-existing setup/doctor
  leaks of the same class.
- **F35-04**: default redirect following. Refusing
  CheckRedirect on cloned clients (caller clients unmutated);
  301/302/303/307/308 catcher matrix proves zero credential
  bytes forwarded.
- **F35-05**: no generation binding. `version` on credentials
  (+1 on enroll/revoke), `credential_version` on leases, Mint
  finalizes under a post-exchange row lock, Revoke takes the
  same lock (linearizable), Consume joins generation match.
  New typed `ErrCredentialSuperseded`/`ErrLeaseSuperseded`.
- **F35-06**: Stat-then-Read. `secrets.OpenSecure`
  (O_NOFOLLOW + fstat regular + 0600 on the fd) and
  `ReadSecureFile` (bounded read from the audited fd) at every
  secret-read site incl. DSN, KEK, refresh, tokens, operators,
  keys/grants, serve-init, and bind paths. No uid check
  (open(2) already enforces; documented); parent-dir
  replacement explicitly out of scope.
- **R36-01**: `.gitignore`'s `secrets/` data-dir rule silently
  excluded the secrets Go package — zero secrets/ files were
  tracked, so committed HEAD cannot build. Rule removed with
  rationale (credential patterns retained); package now
  committable.

## Reproduction

F35-02 (live CLI panic at devbroker.go:80), F35-03 (encoded
password survives), F35-04 (307 forwards refresh+secret),
F35-06 (symlink passes Stat) reproduced pre-fix via throwaway
probes; F35-01/F35-05 source-verified (DB-gated). Post-fix,
probes re-run: clean errors, no panic, no leak, ELOOP refusal.

## Migrations

- `000031_offboard_correction`: accept gate (CREATE OR REPLACE;
  000029 untouched), `operator_offboard` (owner-only EXECUTE via
  session_user pattern), reactivation trigger + function,
  `revoked_at` column grant to studio_lookup, DELETE revoked on
  invitations + snapshots. Down restores the byte-identical
  000029 accept body (diff-verified), drops trigger/functions,
  re-grants DELETE.
- `000032_broker_revocation`: version columns (coherent v1
  backfill), DELETE revoked on broker tables. Down reverses.
- Mirror split (`studioDeletableList`/`f0NoDeleteList`) + parity
  per-table DELETE exceptions; audit comment extended /31/32.
- 000031/000032 applied nowhere; up/down source-reviewed only.

## Gates

- `gofmt -l .`: clean. `go build ./...`, `go vet ./...`: pass.
- `go test ./... -count=1`: **492 pass / 342 skip / 0 fail**
  (baseline 475/328/0; +17 = 7 pool/redact + 1 TTL + 5 secure
  + 3 exchange + 1 secrets-DSN + 1 dbenv − 1 removed Redact;
  +14 = 9 offboard + 5 revocation DB tests skipping honestly).
- `go test -race` on secrets/broker/invite/capability/store/
  dbenv: 6/6 ok (DB concurrency paths skip locally).
- One transient `TestCheckAgentSpawnIdentity` failure observed
  (`main_test.go:242: read |0: file already closed`,
  cmd/gateway, untouched by F0): 3/3 isolated green, suite
  green on immediate re-run and on the final logged run.
  Verbatim log preserved in the session record per the review's
  recommendation; not implicated in F0.

## Prior live evidence reconciliation (§9) — CORRECTED (R36-04)

R36-04: `dbTestPackages` (the `dev:test --db-only` package list)
never included `./broker/...`, `./invite/...`, or
`./capability/...` — the list rotted when the F0 packages were
added, and its test only asserted entries exist, never
completeness. Consequence: **no F0 DB test has ever executed
against live Neon, in any run.** Every `-run` pattern naming
them matched zero tests (proven by the package list and the
`[no tests to run]` lines in owner output).

- v28 owner run (`TestBroker|TestPrivilegeParity`, 197.469s):
  executed ONLY `TestPrivilegeParity` (store). The 5 broker
  tests matched nothing. Earlier claims that broker tests
  "passed live" are RETRACTED.
- v30 owner run (`TestInvite|TestCapability|TestPrivilegeParity`,
  201.027s): executed ONLY `TestPrivilegeParity`. The 6 invite
  + 6 capability tests matched nothing. Same retraction.
- This remediation's first live gate (236.875s store ok):
  again ONLY parity — the F0 packages were still unlisted.
- Fix: the three packages added to `dbTestPackages`, plus a
  completeness assertion (recursive testdb-import scan, e2e
  excluded by design) proven red-green. The corrected gate
  below executes all 31 F0 DB tests for the first time.
- Skipped in review/agent environment: all 31 F0 DB tests
  (10 broker + 15 invite + 6 capability) + parity — no Neon.
- New/changed and NEVER run live: 000031, 000032, ALL F0 DB
  tests (old and new), parity DELETE exceptions. The live
  gate below is mandatory, not ceremonial.

## Re-review (this remediation)

Fresh pass, 7 §10 properties probed (CLI/pure live;
DB-side source + committed tests):
no offboard bypass (all interleavings walked;
session_create backstop verified); no CLI secret leakage
(symlink/encoded/malformed probes re-run clean);
no redirect forwarding (5-status catcher green);
no post-revocation lease use / no mid-exchange mint win
(finalize lock + version join verified);
no check/read race (deterministic swap proof green);
no guard panic (10 verbs + live CLI probes).
New findings R36-01/02/03 above, all fixed. Down-000031 body
diff-identical to 000029. Zero DELETE references in Go.
No other ignored source dirs (sweep clean).

## Owner live gate — FOCUSED RUN EXECUTED GREEN (v32)

`task dev:init`: `studio_dev_ui` re-verified clean at v32
(000031 + 000032 applied; preceded by the one-time dirty-flag
clear after the 42501 incident — the Neon SQL editor had opened
on `neondb`, corrected to `studio_dev_ui`).

Focused gate (corrected package list — first real execution of
all F0 DB tests live):
`task dev:test -- -- -run
'TestBroker|TestInvite|TestOffboard|TestCapability|TestPrivilegeParity'
-timeout 30m -count=1`

| Package | Result | Duration |
|---|---|---|
| store (parity + suite) | ok | 283.800s |
| broker (10 tests) | ok | 587.238s |
| invite (15 tests) | ok | 735.051s |
| capability (6 tests) | ok | 330.425s |
| server/tools/testdb/runner/spike/sandbox/cmd | ok, no tests to run | — |

Zero failures. This closes the R36-04 gap: 31 F0 DB tests +
parity verified against live Neon.

Live `-race` run — EXECUTED GREEN. (A first attempt as direct
`go test -race` with only TEST_DATABASE_URL silently skipped
every DB test — `PolicyFromEnv` finds no NEON_* policy —
observed as a vacuous ~1s "ok" and caught before it could be
misread. The race run MUST go through the task harness, which
injects the policy env from the binding file.)
`task dev:test -- -- -race -run
'TestBroker|TestInvite|TestOffboard|TestCapability' -timeout
30m -count=1`

| Package | Result | Duration |
|---|---|---|
| broker | ok | 476.830s |
| invite | ok | 595.802s |
| capability | ok | 368.569s |
| all others | ok, no tests to run | ~1s |

## Remaining items

- F35-08 full routing (Registry.Load for devDSN): deferred —
  conflict-fail could break the working env+file owner setup;
  needs explicit owner GO. Secure audited reads are in place.
- Parent-directory replacement races: documented non-goal.
- Aliased-import scan evasion: documented residual with the
  reverse direction verified by search.
- Live Neon execution of everything above (owner-side).

## Live incident: 000031 42501 on first apply (fixed)

The owner's first `task dev:init` failed 000031 live:
`must be owner of function invitation_accept (SQLSTATE 42501)`.
Root cause: `CREATE OR REPLACE` on a `studio_lookup`-owned
function as the migrator-as-member. The 000027 precedent (SET
ROLE studio_lookup / RESET ROLE) was not applied to the new
function DDL. No side effects persisted (failure at the first
statement; remediation: owner clears dirty flag, re-runs —
commands in the final report, not here).

Corrections (all local, no live state touched by the agent):
- 000031 up/down restructured: function DDL/ACL inside a SET
  ROLE block; CREATE TRIGGER + table ACLs outside it
  (studio_lookup holds no TRIGGER privilege; tables are
  migrator-owned).
- 000029 down amended per the 000026 Card-001 precedent (downs
  never ran on persistent DBs; runner checks dirty-flag only):
  DROP FUNCTION wrapped in SET ROLE — same latent 42501.
- New `TestMigrationFunctionOwnershipDiscipline` pins the
  invariant statically (function DDL inside the role block,
  table-owner ops outside).
- Down-000031 body re-verified byte-identical to 000029's.

## Verdict

**APPROVED** — all HIGH/MEDIUM defects are fixed,
regression-tested, independently re-reviewed, and the required
live Neon gates pass: v32 clean, 31 F0 DB tests + parity green
(283–735s per package), live `-race` green on all three F0
packages (368–595s). Four new review findings (R36-01…R36-04,
incl. the live 42501 and the rotted package list) were found
and fixed in the course of verification. F0 is closed;
Increment 4 implementation is unblocked.
