# Credential broker (F0, option A: server-side mint)

Tenant automation credentials (provider OAuth refresh tokens) are
NEVER ambient. The broker holds them sealed in Neon under a
deployment-held KEK and mints short-lived, execution-scoped access
tokens bound to (tenant, provider, capability, execution intent).
Refresh tokens never reach Minispecs, agents, terminals, logs, or
audit rows. Lease rows record issue + use WITHOUT token material:
a minted access token exists only in broker memory between Mint
and the provider call.

## Cryptography

- AES-256-GCM (stdlib only), blob = `0x01 || nonce(12) ||
  sealed`.
- AAD binds every ciphertext to its (tenant, provider): a row
  confused across tenants fails decryption instead of minting
  for the wrong owner.
- KEK: 32 bytes, stored as 64 hex chars in a 0600 file, loaded
  via `broker.LoadKEK` (mode-audited, strictly validated).
- `kek_id` fingerprints the encrypting KEK (first 8 hex of its
  SHA-256) so rotation generations stay distinguishable.

## Tables (`000028`)

- `studio.broker_credentials(tenant_id, provider)` — sealed
  refresh + `kek_id` + `revoked_at`. Re-enrollment overwrites
  and clears revocation (operator re-consent).
- `studio.broker_leases(tenant_id, id)` — scope (provider,
  capability, intent sha256), `single_use`, issue/expiry/use
  timestamps. No token columns exist by design.

Both carry the standard tenant RLS policy + FORCE + `studio_app`
grants mirrored to the test role.

## Lease lifecycle

1. **Mint** validates scope (non-empty tenant/provider/
   capability, 64-hex intent, TTL within (0, 1h]), decrypts the
   refresh, trades it at the provider, then finalizes under a
   credential row lock: a revoke or re-enrollment that landed
   mid-exchange fails the mint (`ErrCredentialSuperseded`)
   instead of issuing a lease for a dead generation. Each lease
   binds its issuing credential version. The lease TTL never
   exceeds the provider's `expires_in`. A provider
   `invalid_grant` auto-revokes the dead credential — but only
   when that exact generation is still current, so a concurrent
   re-enrollment is never killed by a stale grant failure.
2. **Consume** validates one use atomically: tenant, capability,
   and intent must match the grant; expiry, single-use, and
   credential generation are enforced in the gating UPDATE, so
   revoked or re-enrolled generations refuse with
   `ErrLeaseSuperseded`. Failures are typed so adapters react
   without string parsing.
3. **Revoke** marks the credential dead; mints fail thereafter.
   Revoking a missing credential is a typed NotFound (a typo'd
   provider must not silently "succeed").

## Operator ceremony (owner terminal)

```
task dev:broker-init-kek                                   # once per env
task dev:broker-enroll TENANT=t PROVIDER=p REFRESH_FILE=/path/0600
task dev:broker-leases TENANT=t [-- --provider p]
task dev:broker-revoke TENANT=t PROVIDER=p
```

Refresh tokens travel by 0600 file only — never flags (process
listings), never chat. `broker-enroll` targets `studio_dev_ui`
under the standard production guard.

## Deliberate non-surfaces

- **No mint verb.** A CLI mint would print an access token to a
  terminal; minting happens only inside broker process memory
  (server runtime / capability execution).
- **No KEK rotation verb yet.** Rotation = generate new KEK,
  re-seal rows, retire old KEK. The `kek_id` column already
  supports generations; the re-seal procedure lands with the
  first rotation requirement, not before.
- **No provider registry yet.** `HTTPExchanger` owns the generic
  OAuth2 refresh wire shape; per-provider endpoints and client
  credentials arrive with the connector increments.
- **No server wiring yet.** `studio serve` does not construct a
  broker in this increment — no consumer exists, so no boot
  requirement was added. Capability execution wires Mint/Consume
  when the first brokered primitive lands.
