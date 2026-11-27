# RPA increment 2 (F0 broker, option A) — evidence

Scope: `plans/mom-n-pop-rpa-platform-plan.md` §4.7 + P13
(`tenant.credential.lease`) — server-side mint per owner decision
A. Refresh tokens sealed in Neon under a deployment KEK, decrypted
only in broker memory; executions get short-lived access tokens
bound to (tenant, provider, capability, intent).

## Files

New: `broker/broker.go` (typed errors), `broker/crypto.go`
(AES-256-GCM + AAD), `broker/kek.go` (KEK loader),
`broker/exchange.go` (OAuth2 refresh exchanger), `broker/store.go`
(Enroll/Mint/Consume/Revoke/Leases), `broker/crypto_test.go`,
`broker/exchange_test.go`, `broker/db_test.go`,
`store/migrations/000028_broker.{up,down}.sql`,
`cmd/studio/devbroker.go`, `cmd/studio/devbroker_test.go`,
`docs/broker.md`.
Modified: `cmd/studio/dev.go` (dispatch), `secrets/registry.go`
(+`STUDIO_BROKER_KEK_FILE`), `secrets/secrets_test.go` (12),
`dbenv/provision.go` (grant mirror +2 tables),
`store/privilege_parity_test.go` (+2 tables, audit comment),
`store/provision_guard_test.go` (devbroker.go allowlist, same
`studio dev` surface precedent as devserve.go),
`Taskfile.dev.yml` (+4 tasks), `docs/taskfile.md`,
`docs/secrets.md` (KEK row).

## Contract (behavioral, all test-pinned)

- Ciphertext: `0x01||nonce||sealed`, AAD binds (tenant,
  provider); cross-tenant/provider/KEK opens and version/nonce/
  body tampering all fail closed (unit).
- KEK file: 0600-audited, exactly 64 hex chars; `kek_id` =
  first 8 hex of SHA-256(KEK) for rotation generations.
- Mint: validates scope (64-hex intent, TTL within (0,1h]),
  decrypts refresh, trades at provider, stores lease row WITHOUT
  token, returns `(leaseID, access, expires)` in memory only;
  lease TTL capped by provider `expires_in`; `invalid_grant`
  auto-revokes the dead credential.
- Consume: atomic gating UPDATE (tenant+capability+intent match,
  live, single-use-once); typed Expired/Consumed/Mismatch/
  NotFound refusals.
- Revoke: marks dead; missing credential is typed NotFound.
- Exchanger: generic OAuth2 refresh POST; wire shape asserted
  against httptest (grant fields posted, token parsed,
  invalid_grant typed).
- Verbs: `broker-init-kek` (0600, refuse-overwrite),
  `broker-enroll` (0600 refresh file, never a flag),
  `broker-revoke`, `broker-leases` (metadata only). NO mint verb
  by design (tokens must never reach a terminal). All live verbs
  run the production guard and target `studio_dev_ui`.
- Parity: broker tables in the grant mirror + parity list.

## Gates (this session, local)

- `gofmt -l .`: clean. `go build ./...`, `go vet ./...`: pass.
- `go test ./... -count=1`: **470 pass / 316 skip / 0 fail**
  (baseline 460/311/0; +10 = 8 broker unit + 2 verb,
  +5 = broker DB tests skipping honestly without Neon).
- Taskfile smoke: `broker-init-kek` writes 0600 + refuses
  overwrite; verb preconditions fail cleanly. (One smoke run
  wrote a stray KEK to the owner config dir via a wrong
  override syntax; removed immediately — config dir verified
  back to its 6 pre-existing files.)
- Incidents: (1) transient `cmd/gateway`
  TestCheckAgentSpawnIdentity failure under full-suite load —
  passes isolated and as a package; untouched code, known flake
  class. (2) `TestNoProdDbenvImport` failed on the new
  devbroker.go import — fixed by allowlisting under the guard's
  own `studio dev` surface precedent (documented in-test).

## Pending: live Neon verification (owner terminal)

Migration 000028 is unapplied anywhere; the 5 broker DB tests
and parity additions have never executed. Minimum owner run:

```
task dev:init
task dev:test -- -- -run 'TestBroker|TestPrivilegeParity' -timeout 15m -count=1
```

Expected: `studio_dev_ui` at v28; 5 broker tests + parity pass.
Then (optional, exercises the verbs end to end):

```
task dev:broker-init-kek
task dev:broker-enroll TENANT=<t> PROVIDER=<p> REFRESH_FILE=<0600 path>
task dev:broker-leases TENANT=<t>
task dev:broker-revoke TENANT=<t> PROVIDER=<p>
```

## Deliberate non-surfaces (carried, not forgotten)

No KEK re-seal rotation verb (kek_id generations ready); no
provider endpoint registry (connector increments); no `serve`
wiring (no boot requirement added until a consumer exists).
