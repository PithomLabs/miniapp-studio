# T1 repo-local implementation evidence (2026-11-27)

**Scope:** C4 contract redesign, R312 fingerprint, `studio dev` revisions, R302 audit, Taskfile setup, docs + tests.
No Neon provisioning, credentials, live connections, or migrations — all live proofs PENDING (see below).

## Files changed

| File | Change |
|---|---|
| `dbenv/neon.go` | NEW: EndpointPolicy, PolicyFromEnv (NEON_*), CheckEndpoint (exact-host + TLS + anti-loopback), ErrNoPolicy |
| `dbenv/neon_test.go` | NEW: policy accept/reject matrix (incl. loopback/private refusals), empty-policy fail-closed, env tests |
| `dbenv/dbenv.go` | Removed loopback allowlist + CheckTestEndpoint; Neon-first trust model; kept WantPGMajor/CheckPGMajor |
| `dbenv/dbenv_test.go` | Retired loopback TestEndpointAllowlist (expectations inverted by approved ADR-004/006, not weakened) |
| `dbenv/provision.go` | CreateRunDB takes EndpointPolicy; Neon-scoped comments; fixed-role residual documented (T2 retires) |
| `testdb/testdb.go` | Policy-gated provisioning; missing metadata skips as "live Neon proof pending" |
| `testdb/testdb_test.go` | Neon-shaped fixtures; subprocess refusal test with branch policy |
| `cmd/studio/dev.go` | Policy gating; NeonEndpoint R312 shape; new `setup` preflight verb (provisions nothing) |
| `cmd/studio/dev_test.go` | Fingerprint R312 keys incl. project/branch/TLS; leak scan retained |
| `Taskfile.dev.yml` | NEW `dev:setup`; metadata preconditions on setup/init/test/reset; bootstrap chains setup first |
| `docs/dev-contract.md` | Contract rewritten Neon-first (rules, entry points, per-env notes) |
| `.github/workflows/ci.yml` | T2-replacement notice header only (no workflow change) |

Untouched by design: `testrunner/` (superseded-history, unimported), `hostexec/` (preserved-uninstalled, only its own
cmd imports it), HTTP listen addresses (not database endpoints; out of R302 scope).

## Architectural decisions implemented

- ADR-004/006 (repo-local portion): loopback endpoint model removed at the contract level — no shim, no exception;
  loopback/private hosts refused with the `unapproved endpoint` marker; TLS (`require`+) proven pre-dial.
- R312: fingerprint carries project/branch/parent/host/port/TLS; credentials still impossible by construction.
- Credential separation: this interface carries DB credentials only; Neon management API keys never enter it (R308).
- Fixed test role retained for process convergence, scoped to non-prod synthetic-data branches; per-run issuance is
  T2 work (recorded residual, not hidden).

## Test counts (observed)

`go build ./...` OK · `go vet ./...` OK · `gofmt` clean on touched packages.
Full suite `go test -p 1 -count=1 ./...`: **396 pass, 304 skip, 0 fail** (after TLS hardening; was 393/304/0).
- 287 skips carry "live Neon proof pending" (DB-backed tests; TEST_DATABASE_URL and NEON_* both unset here).
- Remaining skips are pre-existing designed families (toolchain-image-absent, unconfigured gateway/daemon envs).
- Red-green proof: removing the old entry point broke the build first (no silent fallback); new matrix green after.
- Refusal-before-dial proven end-to-end via the policy-bound subprocess test.

## TLS hardening addendum (T1 approval condition, 2026-11-27)

pgx v5.13.0 source verified: `require` without sslrootcert sets InsecureSkipVerify (encryption only, MITM-able);
`verify-ca` verifies the chain but skips the hostname; only `verify-full` proves both. Changes:
- Policy accepts `sslmode=verify-full` exclusively; all weaker modes refused pre-dial (matrix updated + extended).
- New `dbenv.Pool` / `PoolConfig` / `TLSConfigFor`: every contract-path pool carries a code-pinned verifying
  `tls.Config` (system roots, ServerName = endpoint host, TLS 1.2+, no bypass). All 10 pool sites migrated
  (dbenv ×2, testdb ×3, dev.go ×5); `EnsureClusterRoles` takes the policy.
- String-consumers (golang-migrate pgx5 driver, testkit) inherit verify-full via the policy-constrained DSN —
  documented layering, not an assumption: the DSN cannot carry a weaker mode past the policy.
- Runnable negatives: TLSConfig pinning assertions, PoolConfig pinning + unbound rejection.
- ADR-006 amended; dev-contract TLS rule tightened to verify-full.

## Verified vs pending live evidence

VERIFIED (repo-local, this run): policy accept/reject matrix (verify-full-only); empty-policy fail-closed; env
binding; refusal-before-dial; TLS config pinning (chain+hostname, no bypass); PoolConfig enforcement; grant/
run-pattern guards; fingerprint shape + leak scan; build/vet/gofmt; full unit suite green (396/304/0).
PENDING (need owner prerequisites + live GO): dev-branch suite green; N8 inheritance
proof; R302 runtime audit; parity-report result cells; `dev setup/doctor/init` against a real branch; live TLS
handshake negatives below.

## Live fingerprint verified (owner-terminal 2026-11-27)

First live proof from the bound dev branch, verified from `./dev-parity.local.json` (values withheld here):
bound verify-full connection succeeded (positive TLS control); server major 18; extension inventory =
plpgsql-only (pre-installed, consistent with the zero-migration-extensions rule); locale UTF8/C.UTF-8/GMT;
migrations clean at 0/25 applied (fresh branch, as expected); all 4 fingerprint checks pass. This closes the
extension re-proof on PG18 and the live `dev fingerprint` item; `setup/doctor/init`/suite remain below.

### Pending live TLS negatives (exact procedures, not yet executed)

- **N-TLS-1 untrusted CA:** positive control — `dbenv.Pool` connects to the dev-branch endpoint (verify-full,
  system roots). Negative — same endpoint with `RootCAs` replaced by an empty pool: expect handshake failure
  (`x509: certificate signed by unknown authority`), no session established. Redact endpoint hostname to branch
  ID in evidence.
- **N-TLS-2 hostname mismatch:** same endpoint with `ServerName` replaced by a sibling non-existent hostname:
  expect `x509: certificate is valid for <endpoint>, not <wrong>` (or equivalent hostname error), no session.
- Both run from the owner terminal against the provisioned dev branch after the credential ceremony; DSNs via
  mode-0600 file, never pasted. No mock may satisfy these.

## Zero-transfer handoff addendum (2026-11-27)

Owner standing rule: no credentials in chat or anywhere, ever — extended to NO copy-paste of any values at all.
Chat is untrusted transport. Implemented file-based handoff (`docs/live-handoff-protocol.md`):
- `studio dev bind` (owner-terminal): resolves project/branch BY NAME via local `neon` CLI; writes
  `./dev-neon.json` (identities only, gitignored, agent-readable) + streams the DSN straight to
  `~/.config/miniapp-studio/dev.dsn` (0600, outside repo, never overwritten, never logged); enforces
  verify-full; round-trip validates the binding. Missing branch fails with dashboard instructions (no auto-provision).
- All `dev` verbs accept `--metadata-file`/`--dsn-file`; `dev test` bridges files into the `go test` subprocess env
  (owner-side process only). Env-only path preserved for CI/T2.
- Strict binding schema (unknown fields rejected, bare-hostname + non-local validation); secret-scrubbing after use.
- Taskfile: `dev:bind` + file preconditions on every live task (verified: `task --list` parses; preconditions fail
  fast with the bind instruction).
- Tests: binding loader matrix, bind JSON parsing (miss/ambiguity/malformed), DSN normalize (upgrade/preserve/0600),
  policy precedence (file wins, no fallback on bad file). Guard allowlist extended to bind.go with justification.
- Suite after handoff work: **402 pass, 304 skip, 0 fail**; build/vet/gofmt clean. One self-caught failure during the
  pass (map-order assertion + guard allowlist) fixed before delivery — no weakening.
- Agent-shell Neon API access is BLOCKED by sandbox (read-only fs breaks CLI auth refresh) — all live Neon work is
  owner-terminal by necessity, which coincides with the security design.

## PG18 amendment addendum (owner decision 2026-11-27)

Mandated PG major moved 19 → 18 to comply with the provisioned Neon project (Neon default). Changed: `WantPGMajor`,
gate messages, version fixtures, dev-contract, ADR-004 (amended, rationale recorded). Unchanged: Neon-only mandate,
endpoint policy, TLS rules, all other gates. Live gate behavior already proven: a PG18 server was correctly refused
under the old PG19 gate with the exact mismatch stated; the gate now accepts 18 and refuses everything else.
PENDING acceptance condition: miniapp framework compatibility re-proof on PG18 via a Neon scratch branch (P0 ran on
PG19). Procedure (owner-terminal, after live GO): point the framework suite at a fresh Neon PG18 branch DSN via
mode-0600 file, run the P0 subset, record counts; no self-hosted PG involved.
Naming footgun noted: Neon's default root branch is named `production`; our `dev` branch derives from it while it is
empty. Lineage audits must check DATA (empty/synthetic), not the branch name (ADR-015 controls).

## Live migration findings (owner-terminal 2026-11-27)

- **F1 — migrate driver scheme:** Studio `withMigrationsTable` rewrote only `postgres://` to `pgx5://`; Neon DSNs use
  `postgresql://` → "unknown driver postgresql". Fixed with a both-schemes rewrite + `TestWithMigrationsTableScheme`
  regression (TLS preservation asserted). Framework-baseline path already handled both.
- **F2 — OWNER TO membership:** `ALTER FUNCTION ... OWNER TO studio_lookup` (000002/000009) requires SET ROLE ability
  on the new owner; Neon branch owners are not superusers → 42501. Fixed in 000001 (pre-first-live-application edit,
  reasoned in-file): idempotent `GRANT studio_lookup TO session_user`. App-facing boundaries unchanged; the
  no-memberships-for-runtime-roles test still passes (owner, not a runtime role, is the member).
- **F2b — OWNER TO schema CREATE + self-sufficiency:** same error persisted after F2, implicating the second half of
  the Postgres ownership-transfer rule (new owner must hold CREATE on the schema — studio_lookup held only USAGE via
  000003, which runs later). Added `GRANT CREATE ON SCHEMA studio TO studio_lookup` plus an idempotent membership
  re-assert at the top of 000002's function section, making the migration self-sufficient regardless of prior state.
  NOLOGIN + fixed function bodies mean CREATE grants no app-facing power. (All init error output passes through
  redactDSN, so pasted error lines carry no credentials.)
- **Implicit proof:** the framework baseline chain applied cleanly on Neon PG18 (Studio 000002 began executing),
  retiring part of the PG18 framework-compat question; the full P0-subset re-proof stays pending.
- Failed bootstrap attempts leave their dirty state in abandoned run-pattern databases only; each re-run provisions a
  fresh database, so no manual cleanup is needed (stale reaping covers the orphans).

## 42501 root cause: proven live, repaired (architect diagnosis, 2026-11-27)

- **Proven:** `task dev:diag-roles` returned MEMBER true / SET false / INHERIT false / ADMIN true on the dev branch —
  the exact PostgreSQL 16+ CREATEROLE auto-membership signature. Membership rows exist branch-wide (fresh databases
  never reset them), so every bootstrap re-hit the trap.
- **Repair (canonical, in 000001):** unconditional `GRANT studio_lookup TO SESSION_USER WITH ADMIN FALSE, INHERIT
  FALSE, SET TRUE` (all options explicit per approval refinement; bootstrap grant untouched) + `pg_has_role(...SET)`
  postcondition abort. 000002 asserts both preconditions (SET ability, schema CREATE) and implements no second repair.
  Applied-migration rule documented in-file: verification recreates disposable run DBs; no persistent DB carries 000001.
- **Instrumentation:** migration failures now capture version, dirty flag, embedded-byte SHA-256, SQLSTATE, server
  position/routine, and a dollar-quote-aware statement locator (file-level pinning guaranteed; statement mapping
  best-effort with explicit fallback). Zero credential exposure by construction.
- **Regression:** `TestStudioLookupSetRepair` (10-step live scenario on a scratch probe role, full cleanup, honest
  skip without Neon) + static embedded-SQL pins + locator/diagnostic unit tests. Suite: 409 pass / 305 skip / 0 fail.
- **Freshness:** embed-vs-disk comparison test at test time; runtime SHA-256 on every migration failure; no cache
  clearing performed or needed.
- **Awaiting live run:** owner-terminal `task dev:bootstrap` (exercises repaired chain + live scenario test).

## Remaining blockers (owner gates, not code)

1. Neon non-prod project provisioned; 2. secret-store product chosen; 3. dev-branch credential ceremony;
4. formal acceptance of ADR-004/005/006/010. Live T1 proofs start only after 1–4.
