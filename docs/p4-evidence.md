# P4 evidence record (2026-11-24)

Studio server and UI per prompt6_plan (approved), Phases A–D. No credential
value was emitted anywhere; DB work ran against Docker Postgres with
file-held secrets. Everything uncommitted; Miniapp untouched (no §14
evidence arose); frozen plans untouched.

## Gates (all green, this session)

- `gofmt -l .` — no output.
- `go vet ./...` — clean.
- `go test -p 1 ./...` on PG19 (127.0.0.1:5434) — all 12 packages ok.
- DB suites (`store`, `server`, `tools`, `cmd/studio`, `spike`) on PG17
  (127.0.0.1:5435, 17.12) — all ok.
- `go test -race -p 1` on `store`, `tools`, `server`, `cmd/studio` — clean.
- Framework pin unchanged: `73351a8` throughout; G-01 still open and
  external writes still disabled.

## Phase A — F2, F3, F7 (first, per approval)

- F2: migrations 000005 (USAGE on schema) + 000006 (audit sequence grant);
  the audit INSERT/WITH CHECK path tested empirically after the grants.
- F3: migration 000007 (composite tenant FKs) + admission pre-check with
  encoded-traversal cases (`%2e%2e`, encoded separators/backslashes).
- F7: `AfterRelease` tenant reset on every pooled connection; a reset
  failure destroys the connection instead of returning it.
- `MemoryRegistrar` deleted as dead code after deterministic naming won.

## Phase B — Studio tools (after Phase A green)

13 `studio` tools via `runtime.Service` with ten-case matrices. Idempotency
per the approved correction: a fresh unpredictable key per intentional
write (per form render), scoped to tenant/tool/resource, bound to a
canonical payload fingerprint on first accept; retries replay the original
outcome; same key + different payload conflicts; the spec content hash is
spec identity only. Tests prove replay and independence of two identical
payloads under different keys. Secrets never enter tool inputs (they
persist); Custom API credentials stay memory-only in handlers behind a
sanitizing attest tool.

## Phase C — Six screens (after real handlers)

Async deploy worker with stuck-threshold staleness, invoke/CSRF/opkey
helpers, six screens with loading/empty/error/success states; loading
reflects genuine async work only (deployment interstitial). Two fixes
during green-up, both root-caused:

- Resubmit-payload mismatch: the test compared byte-wise; replayed results
  compare semantically (jsonb text form), not byte-wise.
- Activity-list self-observation: `activity.list` excluded its own dispatch
  audit row via `invocation_id` (`tools/activity.go`). The record still
  exists; the view excludes the observer so the empty state stays reachable.

## Phase D — contained fixes

| Finding | Fix | Proof |
|---|---|---|
| F5 | `store.ErrSessionRefused` sentinel; login maps it to the identical 401 page | `TestLoginUnprovisionedOperator401` (deactivated row, correct token → 401, never the 500 page) |
| F6 | `safeNext` shared by login/logout; backslash rejected (browsers parse `\/` as a separator → open redirect) | `TestSafeNextTable` (22 cases incl. controls, encoded CRLF/NUL, double-encoded) |
| F7 | (Phase A, above) | pool tests |
| F8 | all secret-file reads trimmed: `loadTokenFile` (Sprites token), provision token/password/DSN files | `TestLoadTokenFileTrims`, `TestProvisionFresh` (newline-terminated files) |
| F9 | migration 000009 returns the stable session id from `session_lookup` | lookup id == create id asserted in `store` suite |
| F10 | duplicate `login_token_sha256` rejected at startup (last-match-wins would be ambiguous) | `TestLoadOperatorsValidation/dup hash` + message test |
| F11 | `notFound` wired to the mux fallback; README deploy section written (000001 pointer now resolves); `CreateWithToken` doc-fenced test-only | `TestUnknownRoute404`, README |
| F12 | startup version gate (`New` refuses drift) + per-request `readyz` 503; migration 000010 grants `studio_app` version-read | `TestNewRefusesUnmigratedDatabase`, `TestReadyzSchemaGate`, `TestReadyzDirtySchema` (scratch databases) |
| F13 | `cmd/studio` (`serve`/`migrate`/`provision`) with DSN-password redaction | provision fresh/rotation/idempotency/0600 tests, `TestRedactDSN`, `TestMigrateRedactsPassword`, `TestServeHappyPath` (ephemeral port, real shutdown) |

Two more root-cause fixes found while proving the above:

- Attach-path liveness: `deploy.provision` verified the sprite on create
  but trusted the record on attach. It now checks `GetSprite` when a
  manager is configured; a vanished sprite is an honest failure naming it
  (operator retires explicitly), never a silent re-provision.
  `TestStaleProvisioningRetry` + status reconciliation to
  `failed`/`target missing`.
- Stale version assertion: the migrate-idempotency test hardcoded `v8`
  while the message drifted. It now derives the expectation from
  `LatestVersion()`, so new migrations cannot desync it. Schema is at v10.

## F1 / F4 records (document only)

- F1: `docs/architecture/decisions/003-session-issuance-proof.md` compares
  HMAC stateless proof vs DB-held verification with a decision table and
  owner questions. No selection, no implementation — owner decision pending.
- F4: offboarding gap recorded in the README deploy section (manual in P4:
  `active=false` stops issuance at the `session_create` guard —
  `TestLoginUnprovisionedOperator401` proves it — plus session-row delete).
  No automated offboarding command exists; the restriction stands.

## §11 risk disposition for P4

- Pin/framework gaps: pin held at `73351a8`; full suite + race green; no
  §14 correction warranted.
- Secret leakage: file/env-only secrets, 0600 files, DSN redaction with
  tests, no secret values in tool inputs/logs/errors; dual-path
  verification banned until the F1 decision.
- Misleading claims: unconfigured backends refuse honestly; attach paths
  verify liveness; readyz reports drift/dirty as 503; writes still
  rejected at validation and execution boundaries (G-01 open).
- Scope: P5 untouched, no external writes, one-app cut line intact.

## Conditions carried forward

No P5, no external writes, G-01 untouched, Miniapp unchanged absent §14
evidence, F1 unimplemented until the owner decides, F4 manual, everything
uncommitted. Next review is an implementation acceptance check.
