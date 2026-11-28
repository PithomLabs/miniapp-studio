# Prompt36 Plan — Remediate RPA F0 Security Findings

## Goal

Fix the six HIGH/MEDIUM findings (F35-01…F35-06) from
`plans/adv_review_prompt35.md`, regression-test each, triage the
LOWs, re-run adversarial verification, reconcile prior live Neon
evidence, and deliver `docs/rpa-f0-remediation-evidence.md` with
exact owner-side live commands. No live runs, commits, or pushes.

## Success Criteria

- F35-01: stale invitations can never reactivate offboarded
  operators; offboarding revokes outstanding invites; NO
  pre-offboard invitation can become valid after reactivation on
  ANY deactivation path (reactivation trigger safeguard); all 7
  prompt §2 race/rollback cases plus the raw-SQL
  deactivate→reactivate→accept case tested; uniform failures
  preserved.
- F35-02: no panic on any F0 CLI error path; intended errors +
  cleanup preserved; per-stage regression tests for all 9 DB verbs.
- F35-03: no DSN/password/userinfo (decoded, encoded, or full)
  in any CLI error, log, or wrapped error; canary tests.
- F35-04: OAuth exchange refuses all redirects (301/302/303/307/
  308); catcher-proven credential containment.
- F35-05+F35-07: revocation linearizable — no lease issued or
  consumable after revocation wins; each lease binds its issuing
  credential version and Consume requires a match (re-enrollment
  retires old leases); Mint finalization serialized vs revocation
  by a post-exchange row lock; stale `invalid_grant` cannot kill
  a re-enrolled credential; no locks held across network.
- F35-06: one `OpenSecure` helper (NOFOLLOW + fstat + read-from-fd)
  at every secret-file site; symlink/swap regression tests.
- LOWs F35-07…F35-17 explicitly dispositioned; count corrected to
  11 (enumerated, none invented).
- Gates green, fresh adversarial re-review with re-run probes,
  evidence doc complete, verdict honest (APPROVED requires the
  live Neon gate, which stays owner-side).

## Approach

Reproduce-first per finding against the current tree, then fix
root causes (all six verified present as reported):

- **F35-01** (DB): new migration `000031` (000029 may be applied;
  never rewrite). `CREATE OR REPLACE invitation_accept` with an
  inactive-operator gate: lock the operator row first (`SELECT …
  FOR UPDATE`, consistent lock order operators→invitations in
  both functions — no deadlock), refuse when a row exists with
  `active=false`, consume only otherwise, and upsert operators
  without ever flipping false→true. New `operator_offboard`
  function (SECURITY DEFINER, owner-only EXECUTE via the dynamic
  `session_user` grant pattern from `operator_upsert`/Card 001):
  lock operator row, set inactive, revoke outstanding invites
  for that binding in the same statement scope. Revoke-on-
  offboard (not epoch counters) satisfies prompt §2's lifecycle
  rule with the least machinery: a pre-offboard invite is
  revoked, so reactivation cannot revive it; only a fresh
  post-reactivation invite works. Path-independence (raw-SQL
  deactivation bypasses the function) is closed by a second
  safeguard: an `operators` reactivation trigger (`AFTER UPDATE
  OF active WHEN OLD.active=false AND NEW.active=true`) that
  revokes outstanding invites for the binding — triggers fire on
  every path, function or raw SQL. Combined invariant: an invite
  outstanding across an offboard is dead in all cases (revoked at
  offboard via the function, or refused-while-inactive plus
  revoked-at-reactivation via raw SQL). Lock order stays
  operators→invitations on all three paths (accept, offboard
  function, trigger-after-row-lock) — no deadlock cycle. New `dev
  operator-offboard` verb (owner ceremony; offboarding today is
  owner-SQL only — verified no deactivate entry point exists).
  8 DB regression tests (prompt's 7 + raw-SQL
  deactivate→reactivate→accept refusal) incl. both race orders
  with deterministic end-state assertions (consumed ⟺
  session-exists; operator inactive).
- **F35-02+F35-03** (CLI, fixed together — the panic masks the
  leak): capture `cancel` in a separate variable (or nil-check)
  in `brokerPool`; audit all 9 DB verbs (enroll/revoke/leases,
  invite×5, capability-refresh — all share `brokerPool`) for
  error-path cleanup incl. token reservations. Rewrite
  `redactDSN` structurally (spans: user, decoded password,
  encoded password, full userinfo, full DSN; longest-first) and
  redact `brokerPool`'s parse error; audit migrate/provision/dev
  error paths for DSN-bearing wraps. Canary tests per stage.
- **F35-05+F35-07** (DB, shared versioning defect): migration
  `000032` adds `broker_credentials.version` (bumped on enroll
  and revoke) and `broker_leases.credential_version` (bound at
  issue). Mint reads `(ciphertext, version)`, exchanges over the
  network lock-free, then finalizes in one tx: `SELECT …
  FOR UPDATE` on the credential row (lock held only for the
  finalize, never across the network), re-check not-revoked +
  version-unchanged, insert the lease with its version — loss →
  new typed `ErrCredentialSuperseded` (retryable; same operator
  action for both causes). Revoke takes the same row lock, so
  revoke-vs-finalize serializes (linearizable). Consume gate
  joins the credential and requires `revoked_at IS NULL AND
  version = lease.credential_version`, so re-enrollment retires
  old leases. Auto-revoke becomes conditional on matching
  ciphertext+version. Deterministic interleaving tests
  (revoke-before-finalize, revoke-before-consume, revoke→
  re-enroll→consume-old-lease refusal, re-enroll between read
  and auto-revoke) + concurrency tests.
- **F35-04**: `CheckRedirect` refusing on both the default and
  caller-supplied clients (clone, don't mutate, the supplied
  one); 3xx → controlled exchange failure; httptest matrix
  301/302/303/307/308 with a catcher asserting zero credential
  bytes received. Fix F35-13 in the same function (allowlist
  known OAuth codes, bucket the rest).
- **F35-06**: `secrets.OpenSecure(path)` — `O_RDONLY|O_NOFOLLOW`,
  fstat asserts regular file + `perm&0077==0`, read-from-fd;
  returns bytes (+ file identity for tests). No uid check:
  open(2) already enforces access and 0600 ⇒ opener is owner or
  root (documented). Parent-directory replacement explicitly
  out of scope (documented residual, per prompt §7). Apply at:
  `Registry.Load` _FILE branch, KEK, refresh, invite-token,
  DSN files (`devDSN` — also the cheap half of F35-08),
  operators/keys/grants reads, `provision readSecretFile`;
  path-only audits (keys/grants pointers) open+fstat+close.
  Tests: symlink rejection, post-open swap still reads audited
  bytes (deterministic proof), rename-swap loop integrity.

## Key Decisions

1. **Two corrective migrations** (`000031` invitations/offboard,
   `000032` broker versioning), each with a reversible down that
   restores prior behavior (down-000031 re-installs the original
   accept body). Never touch 000028–30 (may be applied live).
2. **Revoke-on-offboard + reactivation trigger, not epochs.**
   The function revokes outstanding invites at offboard; a
   reactivation trigger revokes them on every false→true path
   (function or raw SQL), so no pre-offboard invite can revive
   after reactivation regardless of path. Epochs add a second
   clock without improving on this. Offboarding contract still
   prefers `operator_offboard` (new verb) for immediate hygiene.
3. **DELETE grants revoked** (F35-14) inside the two corrective
   migrations (invitations+snapshots in 000031, broker tables
   in 000032) + `dbenv` mirror split + parity per-table
   exceptions. Verified no code path deletes; retention purge
   (if ever needed) re-adds deliberately.
4. **Dead code removed, not wired:** `Registry.Redact` (F35-09,
   zero callers, demonstrated bypasses — wiring it would grant
   false confidence) and `STUDIO_BROKER_KEK_FILE` (F35-10, never
   read; explicit `--kek-file` stays the ceremony). Docs updated
   to describe the actual controls.
5. **`devDSN` stays file-wins** (F35-08): add OpenSecure audit,
   defer full `Registry.Load` routing — conflict-fail could
   break the owner's working env+file setup; needs owner GO,
   recorded as follow-up.
6. **Cheap LOWs fixed:** TTL range validation (F35-11), enroll
   trim+empty-reject (F35-12), OAuth code allowlist (F35-13),
   docs drift (F35-15), loser-error assertions (F35-16), scan
   needles (F35-17 + documented aliased-import residual).
7. **No scope expansion:** existing sessions surviving
   offboarding (`session_lookup` checks no active flag —
   verified pre-existing) stays as-is; prompt's race tests
   accept "accept-serialized-first wins" semantics. No new
   abstractions beyond the two helpers the fixes require
   (`OpenSecure`, offboard path).

## Steps

1. **Reproduce all six** against the current tree (re-derive the
   review's /tmp probes as repo tests/scripts; confirm line
   numbers and behavior; record deltas).
2. **F35-01**: 000031 up/down (accept gate + offboard fn +
   reactivation trigger), `operator_offboard` + verb, 8 DB
   tests, guard allowlist for the dbenv-importing verb.
3. **F35-02+F35-03**: cancel fix, 10-verb error-path/guard/
   cleanup audit (incl. the new `operator-offboard` verb),
   structural `redactDSN`, canary tests per stage.
4. **F35-05+F35-07**: 000032 up/down, versioned Mint/Consume/
   auto-revoke with post-exchange finalize lock, interleaving
   + concurrency tests incl. revoke→re-enroll→old-lease.
5. **F35-04+F35-13**: redirect refusal + code allowlist, 5-status
   catcher matrix.
6. **F35-06+F35-08-half**: `OpenSecure`, all call sites, symlink/
   swap tests, honesty docs on parent-dir scope.
7. **LOWs**: F35-11/12/15/16/17 fixes, F35-09/10 removals,
   F35-14 revokes (in 000031/000032), F35-08 deferral record,
   count reconciliation (11).
8. **Gates**: gofmt, build, vet, full suite with exact counts,
   `-race` on secrets/broker/invite/capability/store/dbenv,
   migration up/down source review + ordering.
9. **Adversarial re-review**: fresh pass, re-run attack probes
   and regression tests, verify the 7 prompt §10 properties
   explicitly; fix substantiated findings.
10. **Evidence + handoff**: `docs/rpa-f0-remediation-evidence.md`
    (identity, dispositions, root causes, tests, migrations,
    counts/skips, v28/v30 reconciliation mapped to exact test
    names, re-review findings, owner commands, LOW rationale),
    owner live gate commands (from inspected Taskfile/harness:
    `dev:init` → v32, focused run incl. new offboarding tests,
    live race form), honest verdict (APPROVED only with live
    green, which stays owner-side).

## Validation Plan

- `gofmt -l .`, `go build ./...`, `go vet ./...` — clean.
- `go test ./... -count=1` — 0 fail; exact pass/skip delta vs
  the 475/328 baseline; new DB tests skip honestly locally.
- `go test -race` on the six listed packages.
- Per-finding regression suites (offboarding 7, CLI stages,
  canaries, redirect matrix, revocation interleavings, symlink/
  swap) — each fails-before/passes-after demonstrated where
  the pre-fix code is runnable (CLI/pure); DB-only ones carry
  source-verified-before + live-pending status.
- Owner live gate (prepared, not run): `dev:init` then focused
  `-run` covering broker/invite/capability/parity/offboarding
  + live race; migration v32 expected.
- Re-review checklist: the 7 §10 properties each with probe +
  verdict.

## Risks / Open Questions

- 000029/000030 may be applied live: corrective-only DDL is
  mandatory; down migrations must restore exact prior function
  bodies (review before relying on them).
- Prior v28/v30 live evidence predates remediation: it proves
  the old code, never the new — the plan reconciles by exact
  test name and marks all new/changed behavior live-pending.
- `/tmp/p35probe` artifacts are ephemeral and outside the repo:
  all probes are re-derived as committed regression tests.
- No questions require the owner before starting: the prompt
  mandates autonomy, every decision above is explicit with a
  reversible rationale, and the two deferrals (full devDSN
  routing, parent-dir races) are recorded as follow-ups, not
  blockers.
