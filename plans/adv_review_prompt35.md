# Retrospective Adversarial Review — RPA F0 Increments 1–3 (prompt35)

## Scope and identity

Reviewed worktree: commit `df25e17` ("prompt 34 inc 3"), tree clean
except untracked `plans/prompt35.md`. Scope: `secrets/`, `broker/`,
`invite/`, `capability/`, migrations `000028`–`000030`,
`cmd/studio/dev{broker,invite,refresh,secrets}.go`, Taskfile F0
commands, `dbenv` guard/mirror, `server/config.go` + login/errors,
`store` tenant/session/parity, `tools` dispatch, `docs/{secrets,
broker,invitations,capabilities,rpa-f0-1/2/3-evidence}.md`.

Constraint note: prompt35 §5/§7 authorize remediation and a `docs/`
deliverable. The session instruction (read-only; report to `plans/`)
overrides both — no code was modified, no fixes implemented, and
this file is the deliverable. Every finding carries a recommended
correction instead of a fix.

## Executive verdict: NOT APPROVED

One HIGH authorization-bypass (stale invitations defeat operator
offboarding), five MEDIUMs (CLI panic on all guard paths, DSN in
errors, OAuth redirect exfiltration, revocation races, secret-file
TOCTOU), and twelve LOWs. No Critical. The cryptography, RLS
design, atomicity shapes, and canary discipline are strong, and
several attack hypotheses were disproven by probe — but the HIGH
sits on the session-issuance boundary and all 17 F0 DB tests plus
parity skipped here (no Neon), so database behavior is
source-verified only.

Closure checklist: fix F35-01 (HIGH) + F35-02…F35-06 (MEDIUMs),
re-run this review's probes, then execute the owner-side live
commands in §8. Increment 4 design may proceed, but nothing may
depend on invitation/offboarding interplay until F35-01 is fixed,
and broker Mint/Consume production wiring must include the F35-04
redirect refusal and F35-05 revocation re-check.

## Coverage

- **A. Secrets:** registry, precedence/conflicts, dotenv subset,
  file-mode audit + races, `_FILE`, redaction coverage, Taskfile
  tracing, server adoption, DSN/canary handling. Probed with
  canary values through parsers and the live CLI.
- **B. Broker:** GCM/AAD/ciphertext format, KEK loading, refresh
  plaintext handling, OAuth exchange (redirects/TLS/limits/
  errors), mint scope/TTL, lease atomicity, revoke races,
  auto-revoke targeting, RLS/grants, cross-tenant/scope binding.
  Mint/Consume have no production callers (nil exchanger
  everywhere); the CLI deliberately exposes no mint verb.
- **C. Invitations:** entropy/hash persistence, token-file
  reservation (O_EXCL + held fd), tenant stamping, expiry/
  revocation/resend/replay, uniform failures, `invitation_accept`
  (search path, privileges, atomicity), accept/session atomicity,
  concurrency, rollback injection, token leakage.
- **D. Capability:** source authority, drift-as-data, publish
  atomicity, last-good retention, concurrency, tenant scoping,
  read/absent/corrupt distinction, descriptive-only in BOTH
  directions (scan test forward; zero execution-path readers
  reverse, verified by search).
- **E. Cross-increment:** full chain traced (registry → KEK →
  credential → lease → capability → invite → sessions); 000028–30
  up/down + dependency order; RLS/FORCE everywhere on F0 tables;
  tenant context single-site (`SET LOCAL`); definer search paths;
  grant mirror + parity (new tables + `invitation_accept`
  covered); CLI env/prod guards incl. DSN↔policy host binding;
  down-migration orphans (none — drops carry grants/policies).

## Findings

### F35-01 HIGH — Stale invitation reactivates offboarded operators

- **Increment:** 3 (invite + sessions composition). **Files:**
  `store/migrations/000029_invitations.up.sql:58-60`,
  `store/migrations/000002_schema.up.sql:186`.
- **Invariant violated:** offboarding (`active=false`, the
  documented session-issuance kill switch per `docs/p4-evidence.md`
  and ADR-003) must stop session issuance.
- **Attack:** operator holds invite issued before offboarding
  (TTL ≤ 30d). After `active=false`, they run `dev invite-accept`:
  `invitation_accept` consumes + `ON CONFLICT DO UPDATE SET
  active = true`, then `session_create` succeeds — full session
  as the offboarded identity, no member cooperation needed.
- **Root cause:** unconditional reactivation in the accept
  function; no operator-state gate. Test suite never covers
  invite×deactivation interplay.
- **Reproduction:** needs Neon (owner cmd §8): deactivate a
  binding with a live invite, accept, observe session. Statically
  certain from the SQL.
- **Correction:** refuse accept when an operator row exists with
  `active=false` (check before consume; lock the operator row to
  close the mid-accept deactivation race); legitimate re-onboarding
  = owner reactivates first, then invite. Gate Create-side too (or
  document that insider Create for inactive bindings is refused at
  accept). Add DB regression test.
- **Disposition:** NOT FIXED (read-only constraint).

### F35-02 MEDIUM — `brokerPool` panics on every guard/policy/DSN error

- **Increment:** 2+3 (all 9 F0 DB verbs). **File:**
  `cmd/studio/devbroker.go:76-99` (deferred `cancel()` on the
  nil-valued named return after `return nil, nil, …`).
- **Invariant violated:** guard refusals must fail closed with a
  readable remedy (the guard's contract); error paths must run
  cleanup.
- **Scenario:** any policy load, production-guard, DSN read, or
  DSN parse failure → nil-pointer panic + stack trace instead of
  the error; token-file reservation cleanup skipped (stray empty
  0600 file); exit 2.
- **Evidence:** reproduced live — malformed DSN via real CLI
  panicked at `devbroker.go:80` (`brokerPool.func1`).
- **Correction:** capture the cancel func in a separate variable
  (or nil-check); add error-path regression tests per verb.
  Fix together with F35-03 (the panic currently masks it).
- **Disposition:** NOT FIXED.

### F35-03 MEDIUM — DSN/password reaches CLI errors and logs

- **Increment:** 1+2. **Files:**
  `cmd/studio/devbroker.go:94-97`, `cmd/studio/migrate.go:61-75`.
- **Invariant violated:** secrets never appear in terminal
  output, logs, errors, or evidence.
- **Attack/vectors:** (a) `brokerPool`'s `url.Parse` error wraps
  the raw DSN and is returned UNREDACTED (reachable once F35-02
  is fixed — verified by code path); (b) `redactDSN` replaces
  only the *decoded* password, so URL-encoded passwords in
  messages survive — demonstrated: decoded `p@ss:word/x` absent
  from message while encoded `p%40ss…` present (probe C3).
  Worst case: malformed-DSN errors in CI/automation logs.
- **Correction:** redact the parse error against the raw DSN
  string; make `redactDSN` replace decoded + encoded + full-DSN
  forms (or redact the userinfo span structurally).
- **Disposition:** NOT FIXED.

### F35-04 MEDIUM — OAuth exchange follows redirects (latent exfil)

- **Increment:** 2. **File:** `broker/exchange.go:34-62`
  (default client, no `CheckRedirect`).
- **Invariant violated:** refresh tokens + client secrets go to
  the configured endpoint only.
- **Evidence:** probe B1 — 307 redirect delivered the full form
  (refresh token AND client secret) to the catcher
  (`body-has-refresh=true body-has-clientsecret=true`). 302 is
  safe (Go rewrites to GET, body dropped — probe B1b).
- **Status:** LATENT — no production `HTTPExchanger` wiring
  exists (nil exchanger everywhere; endpoints arrive with
  connector increments). Must be fixed before wiring.
- **Correction:** `CheckRedirect` refusing (or `useLastResponse`
  + explicit handling); regression tests for 307/302.
- **Disposition:** NOT FIXED.

### F35-05 MEDIUM — Revocation races in Mint and Consume

- **Increment:** 2. **File:** `broker/store.go:100-177`
  (`refreshFor` → network exchange → insert with no re-check;
  `Consume` gate never reads `broker_credentials`).
- **Invariant violated:** revocation stops credential use.
- **Scenarios:** (a) revoke lands during an in-flight Mint's
  provider exchange → lease still issued; (b) outstanding leases
  stay consumable until expiry after revocation. Bounded by
  TTL ≤ 1h; no test covers either race.
- **Correction:** conditional lease insert (`… AND NOT EXISTS
  (SELECT 1 … revoked)`); add revocation check to the Consume
  gate (or document TTL-boundedness as accepted and test it).
- **Disposition:** NOT FIXED.

### F35-06 MEDIUM — Secret-file permission-check/read TOCTOU

- **Increments:** 1+2+3 (shared root cause). **Files:**
  `secrets/load.go:75-87` (`CheckFileMode` Stat + separate
  `ReadFile`, symlinks followed); call sites: `_FILE` loads,
  `broker/kek.go`, refresh/token/DSN/operators/keys/grants files.
- **Invariant violated:** the audited file is the file read.
- **Evidence:** probe S1a — symlink to a 0600 file PASSES the
  audit; S1b — 17/1500 loads returned outside bytes under a
  rename-swap loop. Max impact at KEK/refresh substitution
  (decrypt oracle / credential swap), gated on directory/file
  write access (sandbox-contained in the standard install, where
  all secret paths sit outside the repo).
- **Correction:** `O_NOFOLLOW` open + `fstat` + read-from-fd at
  every site (one helper; hostexec-claim precedent); regression
  test with symlink + swap loop.
- **Disposition:** NOT FIXED.

### F35-07 LOW — `invalid_grant` auto-revoke can kill a re-enrolled credential

- **File:** `broker/store.go:271-278` (unconditional `revoked_at`
  update). Re-enroll landing between the refresh read and
  `markRevoked` gets revoked by the stale grant failure.
  Fail-closed availability wart, narrow window, operator-visible.
- **Correction:** conditional revoke (`… AND refresh_ct = $3`
  with the exchanged ciphertext); DB test with interleaved
  re-enroll. Doc `broker.md` should condition the auto-revoke
  claim. NOT FIXED.

### F35-08 LOW — Owner-DSN handling bypasses the secrets contract

- **Files:** `cmd/studio/migrate.go:25-33`,
  `cmd/studio/provision.go:66-76`, `cmd/studio/dev.go:97-110`
  (`devDSN`), `provision.go:131-141` (`readSecretFile`, no mode
  check). Direct env reads; file silently wins conflicts (no
  fail-closed); no `_FILE` convention; no 0600 audit on
  `--dsn-file`. Production/host-binding guards bound the blast
  radius; operator-terminal only.
- **Correction:** route through `Registry.Load`; mode-audit DSN
  files; conflict-fail in `devDSN`. NOT FIXED.

### F35-09 LOW — `Registry.Redact` is dead code with narrow coverage

- **File:** `secrets/redact.go` — zero callers repository-wide;
  documented as the redactor. Demonstrated bypasses (probe S3):
  JSON, bare values, `export` lines, lowercase names, and
  multi-line continuations all leak; only `NAME=` lines and
  DSN shapes are covered. Structural prevention (files,
  identities-only output) is the actual control and holds.
- **Correction:** wire it where output is rendered, or remove it
  and document `RedactSpans`/`redactDSN` scope honestly. NOT FIXED.

### F35-10 LOW — `STUDIO_BROKER_KEK_FILE` registered but never read

- Broker verbs take `--kek-file` (a path flag, fine) and never
  consult the registry entry; `docs/secrets.md` claims every
  surface goes through the registry. Dead entry + doc overclaim.
- **Correction:** adopt the variable or remove it. NOT FIXED.

### F35-11 LOW — `--ttl-hours` NaN/negative silently becomes 168h

- **Files:** `cmd/studio/devinvite.go:125,226`,
  `invite/invite.go:80-82` (`ttl <= 0` → default). Demonstrated:
  `NaN`/`+Inf` convert to min-int64 duration → default 168h, no
  error. A typo'd TTL grants longer-lived invites than intended.
- **Correction:** CLI-side range validation (`0 < h ≤ 720`).
  NOT FIXED.

### F35-12 LOW — Enroll seals untrimmed refresh bytes

- **File:** `cmd/studio/devbroker.go:121` (`os.ReadFile`, no
  trim; contrast KEK/invite trimming). A trailing newline
  (typical editor file) seals a wrong token → first Mint gets
  `invalid_grant` → auto-revoke destroys the enrollment.
  Fail-closed but self-destructive; confusing UX.
- **Correction:** `TrimSpace` + empty-reject before `Enroll`.
  NOT FIXED.

### F35-13 LOW — Provider error codes echoed verbatim into errors

- **File:** `broker/exchange.go:80` — provider-controlled
  `error` string lands in broker errors → callers/logs.
  Demonstrated with a marker (probe B2b). Precondition is exotic
  (rogue endpoint echoing material it already holds).
- **Correction:** allowlist known OAuth codes, bucket the rest.
  NOT FIXED.

### F35-14 LOW — Unused DELETE grants on F0 tables

- `000028`/`29`/`30` grant full CRUD; no code path deletes
  (revoke/consume/upsert only). Least-privilege hardening —
  migration + `dbenv` mirror + parity test must move together.
  NOT FIXED.

### F35-15 LOW — Documentation debt

- `docs/secrets.md` still frames the broker as "future/open
  decision" post-increment-2 and omits that tenant scope has no
  entries; the gateway boot-warning comment (`server/config.go:40`)
  has no implementation (empty secret silently disables
  callbacks — verbs requiring it still fail closed). NOT FIXED.

### F35-16 LOW — Concurrent-accept test ignores loser error quality

- `invite/db_test.go:210-243` asserts win-count + one session
  but never that losers get `ErrInviteInvalid` (a 500-style
  loser error would pass). Test-hardening. NOT FIXED.

### F35-17 LOW — Capability scan needle gaps

- `TestRefreshGrantsNothing` misses `Revoke(`/`session_revoke`
  and aliased-import-only references (partly mitigated by
  paren-needles on call sites). Reverse direction verified
  separately by search (zero execution-path snapshot readers).
  Test-hardening. NOT FIXED.

## Hypotheses disproven by probe (no finding)

- CRLF dotenv corruption: values cut from the trimmed line —
  `K=val\r\n` parses as `val` (probe S2). Leading-space
  preservation and inline-comment-as-data are documented + pinned.
- `expires_in` overflow escape: `2^62` yields `ttl=0`, and the
  `expires_at > issued_at` CHECK fails the lease insert — Mint
  errors and the token is dropped (probe B2a + `000028:34`).
- 302 redirect exfil: Go rewrites to GET, body dropped (B1b).
- Production-guard bypass via `--dsn-file`: `CheckEndpoint`
  binds DSN host to policy hosts before dial (B3a) — metadata
  and DSN cannot disagree onto prod.
- serve-marker credential: the `credential` field is a service
  credential NAME (identifier), not a value; marker gitignored.
- F0 HTTP/execution integration: none exists — no server,
  gateway, or tools imports of broker/invite/capability, so
  several handler-layer concerns are future preconditions, not
  current defects. F0 is CLI + library against the dev database.

## Remediation status

NONE — the session's read-only instruction overrides prompt35 §5.
All 18 items above are recorded with recommended corrections and
reproduction evidence; none implemented, no regression tests added,
no commits. Prompt35 §4 "actual disposition" for every finding is
therefore: reported, unverified fix pending.

## Test and race evidence

| Check | Command | Result |
|---|---|---|
| gofmt | `gofmt -l .` | clean, exit 0 |
| build | `go build ./...` | clean, exit 0 |
| vet | `go vet ./...` | clean, exit 0 |
| full serial ×2 | `go test -p 1 -count=1 ./...` | exit 0 both; 25 `ok`, 0 FAIL/panic each |
| full verbose | `go test -p 1 -count=1 -v ./...` | exit 0; **475 pass / 328 skip / 0 fail** |
| race F0+boundary | `go test -race ./broker/ ./invite/ ./capability/ ./secrets/ ./dbenv/ ./store/` | ok all six |
| gateway ×10 | `go test -count=10 ./cmd/gateway/` | ok (transient hunt) |
| S1 file race | `/tmp/p35probe/s12` | symlink passes; 17/1500 outside reads |
| S2 dotenv | `/tmp/p35probe/s12` | CRLF fine; inline-comment pinned behavior |
| S3 redact | `/tmp/p35probe/s12` | 5 bypass shapes leak; NAME=/DSN covered |
| B1 redirect | `/tmp/p35probe/b12` | 307 exfils refresh+secret; 302 safe |
| B2 overflow/echo | `/tmp/p35probe/b12` | 2^62→ttl=0 (contained); errcode echoed |
| B3 guards | `/tmp/p35probe/b3c3` | host/sslmode/loopback/prod-name all refuse |
| C1 CLI panic | real `studio dev broker-leases` run | nil-deref at `devbroker.go:80` |
| C3 encoding | `/tmp/p35probe/b3c3` | decoded pw absent, encoded present |
| T1 ttl float | `/tmp/p35probe/b3c3` | NaN/Inf→minint64→168h default |
| V1 lister | `/tmp/p35probe/b3c3` | 28 tools, 0 unbound, v1.0.0 |
| canary docs | grep + dev-*.json shape audit | no DSN/tokens; binding files identities-only, gitignored |

Skips are environmental (no Neon: `TEST_DATABASE_URL`/`NEON_API_KEY`
unset; loopback DB prohibited by contract), never described as
passing. F0 DB tests skipped: 5 broker + 6 invite + 6 capability +
`TestPrivilegeParity` (+ pre-existing skips elsewhere = 328).

## Transient-failure hunt (§6)

Two historical observations: (1) F0-2 evidence names one
`TestCheckAgentSpawnIdentity` failure "under full-suite load";
(2) F0-3 evidence notes one unnamed single FAIL in 4 runs. No
failure logs were preserved. Investigation: the named test body
is deterministic (fixed assertions, no sleeps/ports/shared
state; supervisor `Start` is mutex-guarded); gateway package
10/10 green here; two full serial suites exit 0 (25 ok each);
verbose run 475/328/0 with zero failures. NOT REPRODUCED.
No evidence implicates F0 code (gateway untouched by F0; F0 DB
tests skip in both environments' reports). Confidence in F0 is
unaffected; recommend preserving the next failure log verbatim
and adding CI `-count` stress on `cmd/gateway` before calling it
a known flake.

## Database and migration verification status

Source-verified: 000028/29/30 up/down SQL, RLS + FORCE on all F0
tables, tenant-context single-site, definer search paths and
function ACLs, grant mirror + parity list coverage, down-migration
orphan-freedom. NOT executed anywhere (migrations unapplied; all
F0 DB tests skip): RLS enforcement, `invitation_accept`
behavior (incl. F35-01), lease atomicity under concurrency,
parity, rollback injection, canary row dumps.

## Owner-side live verification commands

```sh
task dev:bind
task dev:init                    # expect studio_dev_ui at v30
task dev:test -- -- -run 'TestBroker|TestInvite|TestCapability|TestPrivilegeParity' -timeout 15m -count=1
# race the concurrency paths live:
TEST_DATABASE_URL=$TEST_DATABASE_URL go test -race ./broker/ ./invite/ ./capability/ -count=1
# verb smoke (dev database only):
task dev:broker-init-kek
task dev:broker-enroll TENANT=<t> PROVIDER=<p> REFRESH_FILE=<0600 path>
task dev:broker-leases TENANT=<t>
task dev:broker-revoke TENANT=<t> PROVIDER=<p>
task dev:invite-create TENANT=<t> BINDING=<b> INVITED_BY=<op> TOKEN_FILE=/tmp/inv.tok
task dev:invite-list TENANT=<t>
task dev:invite-accept TOKEN_FILE=/tmp/inv.tok SESSION_TOKEN_FILE=/tmp/sess.tok
task dev:capability-refresh TENANT=<t>
# after the F35-01 fix: deactivate a binding holding a live invite,
# accept must fail uniform; owner re-activates, fresh invite works.
```

## Remaining risks and Increment 4 guidance

- F35-01 must close before invitations meet real offboarding;
  until then, treat outstanding invites as reactivation-capable
  and delete invite rows when offboarding (owner SQL;
  `revoked_at` suffices).
- F35-04/F35-05 fixes are prerequisites for broker
  Mint/Consume production wiring (connector increments).
- HTTP invite endpoints (later increment) must enforce
  inviter-membership + session-derived tenant at the handler —
  the service layer deliberately cannot see `operators`.
- Lease IDs printed by `broker-leases` are safe to display
  (consume needs the unprinted intent hash too) — keep it that way.
- DB-trusting conclusions above (RLS, atomicity, parity) await
  the live run; treat them as source-verified, not proven.

## Terminal report

F0 increments 1–3 are NOT safe to close from a code-review
perspective: one HIGH (invitation/offboarding bypass) and five
MEDIUMs require correction, and every DB-backed invariant is
unexecuted (17 F0 DB tests + parity skip without Neon). What was
verifiable statically or DB-free is largely strong — crypto,
atomicity shapes, guard design, canary discipline, and several
attack hypotheses were falsified by probe. Unverified: all live
database behavior (commands above). Increment 4 may proceed in
design only, gated on the F35-01 fix and the live verification
run; broker production wiring additionally gated on F35-04/F35-05.

