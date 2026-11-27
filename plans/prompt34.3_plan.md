# Prompt34.3 Plan — Increment 3: Tenant Invitations + Capability Refresh

## Goal

Implement prompt34.3 in the current working tree: a secure tenant-invitation
lifecycle and a framework-native capability-refresh mechanism, reusing existing
Studio infrastructure (sessions, operators, tool registry, connector manager,
RLS/grant-mirror doctrine) and preserving the Increment 1 (`secrets/`) and
Increment 2 (`broker/`, migration `000028`) work untouched.

## Success Criteria

- Invitations: create → deliver-out-of-band → accept → session, plus
  revoke/resend/expiry, all under the existing identity model; raw tokens
  never persisted or logged; acceptance idempotent-safe under retry and
  concurrency; no enumeration or cross-tenant leakage.
- Capability refresh: snapshot derived only from registered tool
  implementations and connector attestations; states
  implemented/unavailable/planned/stale/failed; atomic publish (partial or
  failed refresh never masquerades as complete); refresh output grants
  nothing (execution paths byte-identical in behavior).
- Persistence: new migrations apply up/down cleanly with RLS, grants,
  mirror, and parity updates; existing data compatible.
- Gates: gofmt clean, build + vet pass, full suite green (DB tests honest
  about skips), race on the concurrency tests, adversarial review performed
  with substantiated findings fixed.
- Evidence: `docs/rpa-f0-3-evidence.md` plus exact owner-side live-Neon
  verification commands. No live runs, commits, or pushes by the agent.

## Approach

Reuse, don't duplicate. Every security primitive already exists:

- Invitation tokens reuse `store.MintToken` (256-bit opaque) and
  `store.HashToken` (sha256 hex) — the same hash-only persistence rule as
  sessions.
- Acceptance reuses `store.Store.Create` for session issuance and follows
  the `session_create` SECURITY DEFINER precedent: the app role holds no
  direct privilege on operator provisioning; one new fixed-body function,
  `studio.invitation_accept`, performs guarded accept + operator upsert +
  consume atomically.
- Capability identity reuses the framework `contract.CapabilityRef`
  (`miniapp/contract/manifest.go`: provider/operation/version, pinned).
  Refresh sources are the registered Studio tool defs (`tools/register.go`,
  via the miniapp registry) and `ConnectorManager.ListConnections`
  (`fly/interfaces.go`; `FakeConnectorManager` in tests). No broker minting
  during refresh — refresh reports, never acquires credentials.
- Tenant isolation reuses `store.InTenantTx` + FORCE RLS + the
  `dbenv.GrantTestPrivileges` mirror + `TestPrivilegeParity` additions.

## Key Decisions

1. **No role hierarchy exists — none will be invented.** Verified: the
   identity model is flat `(tenant_id, binding_id, active)` operator pairs
   (`000002_schema`), no role/permission columns anywhere. The prompt's
   "existing role and permission model" therefore maps to: inviter must
   hold a valid session for the tenant (any authenticated operator of that
   tenant may invite/revoke/resend); acceptance binds the fixed invited
   binding. A future role hierarchy is explicitly out of scope. (Highest
   review-value decision.)
2. **Acceptance is one atomic transaction.** Accept (token capability,
   unauthenticated endpoint) runs in a single pgx transaction:
   `invitation_accept` (guard → operator upsert → consume) then
   `session_create` for the returned `(tenant, binding)` — no
   `store.Store.Create` round-trip outside the tx. If session creation
   fails, the whole transaction rolls back and the invitation stays
   consumable, so a failed accept is safely retryable and can never strand
   a consumed invite without a session. Fallback (only if composition
   proves impossible): a durable idempotent recovery keyed by invite id
   that resumes session issuance without allowing token replay to mint
   additional sessions. Replays after success return a typed
   already-consumed response with no new session and no state change
   (replay-safe; "idempotent" means safe retry, not session replay).
3. **Enumeration resistance by uniform response.** One accept-failure
   message for bad/expired/revoked/consumed/foreign tokens; hash lookup
   only; no tenant-existence oracle in errors or timing beyond indexed
   lookup.
4. **Capability snapshot is stored, published atomically.**
   `studio.capability_snapshots`, one row per tenant, replaced in a single
   transaction only after a fully successful collection pass. Partial or
   failed runs leave the previous snapshot with its recorded `refreshed_at`
   (staleness is honest data, computed from it). Rejected alternative:
   on-demand computation — cannot record provenance or distinguish
   stale/failed states.
5. **Two migrations: `000029_invitations`, `000030_capability_snapshots`.**
   One topic each (000028 broker precedent: tables grouped by topic, but
   invitations and snapshots are independent features with independent
   revert stories). Each with RLS + grants + mirror + parity + reversible
   down.
6. **Minimal surfaces, no UI.** Service/store layer + `studio dev` verbs
   for owner verification (`invite-create/revoke/list`,
   `capability-refresh`), following the `devbroker.go` precedent
   (production guard, `studio_dev_ui` target, names-only output). No new
   Console scenes — not requested, and the honesty rule forbids placebo UI.
   No mint/accept verb prints tokens except create-once to the inviter's
   terminal (documented, out-of-band delivery is the operator's step).

## Steps

1. **Invitation contract + migration 000029.** Table
   `studio.invitations(tenant_id, id, token_hash UNIQUE, binding_id,
   invited_by, expires_at, consumed_at, revoked_at, created_at)`; RLS +
   FORCE + `studio_app` grants; SECURITY DEFINER
   `studio.invitation_accept(p_token_hash)` doing guard-check → operator
   upsert → consume → return `(tenant_id, binding_id)` in one statement
   scope; down migration drops function + table. Mirror + parity updates.
2. **Invitation service + verbs.** `store` or new `invite/` package:
   `Create` (session-authorized tenant operator; returns raw token once),
   `Accept` (single tx: accept fn + session_create; uniform failure),
   `Revoke`, `Resend` (rotate: revoke old hash + new token),
   `List` (metadata only). `dev invite-*` verbs with guard + redaction.
3. **Invitation tests.** Authz (non-member cannot invite/revoke/resend;
   service-boundary, not UI), tenant isolation (cross-tenant token use
   fails uniformly), expiry, revocation, replay (no second session),
   resend rotation (old token dead), concurrent accept (exactly one
   session, one membership — race test), token canary (hash-only in rows;
   no raw token in errors/logs), idempotent retry safety,
   failure-injection (forced session_create failure rolls back
   consumption: invite stays valid, no operator row, no session; retry
   then succeeds).
4. **Capability refresh contract + migration 000030.** Table
   `studio.capability_snapshots(tenant_id PK, snapshot JSONB,
   refreshed_at, provenance JSONB, complete BOOL)`; RLS + grants + mirror
   + parity; down drops table.
5. **Refresh service + verb.** Collect from tool registry +
   `ConnectorManager` (+ declared-vs-registered diff for planned state);
   classify per-`CapabilityRef` into the five states; unknown providers,
   malformed contracts, and unverified entries fail closed (state
   `failed`/`unavailable`, never assumed); atomic publish only on full
   success; `dev capability-refresh` verb. Provenance records source,
   check time, and failure reason per entry.
6. **Refresh tests.** Authoritative-source-only (registry double with an
   unregistered capability → not implemented; manager double down →
   failed, old snapshot retained); unknown/stale/failed/cross-tenant
   fail-closed; refresh-grants-nothing proof (execution/authz/approval
   paths untouched — assert no new call edges from refresh into
   mint/approve/execute, plus unchanged golden behavior of one approval
   test); concurrent refresh safety (one published snapshot, losers
   retry-or-keep-previous, never partial).
7. **Gates + evidence.** gofmt, build, vet, full suite, race on new
   concurrency tests; `docs/rpa-f0-3-evidence.md` (files, contracts,
   migration details, gates, skips, limitations); exact owner live-Neon
   commands (`dev:init` → v30, focused `-run` gate).
8. **Adversarial review + fixes.** Separate pass against actual code and
   tests (invite authz bypass, token leakage, RLS gaps, snapshot
   integrity, migration reversibility, regression); fix substantiated
   findings; report dispositions. No architecture reopening.

## Validation Plan

- `gofmt -l .`, `go build ./...`, `go vet ./...` — clean.
- `go test ./... -count=1` — 0 fail; new DB tests skip honestly without
  Neon; exact pass/skip counts reported against the 470/316 baseline.
- `go test -race` on invitation-concurrency + refresh-concurrency tests.
- Owner live gate (prepared, not run): `task dev:init` (v30 clean) then
  `task dev:test -- -- -run 'TestInvite|TestCapability|TestPrivilegeParity'
  -timeout 15m -count=1`.
- Adversarial review: findings enumerated with file/line, each fixed or
  rebutted with evidence.

## Risks / Open Questions

- **Decision 1 is the plan's main judgment call.** If the owner wants a
  real role hierarchy (e.g., tenant admin vs member inviter rights), that
  is new product scope beyond prompt34.3's "existing model" constraint and
  needs a separate decision — the plan does not block on it.
- Invitation delivery itself (email/message) is out of scope: create-once
  terminal output + operator out-of-band delivery, same trust model as the
  existing login-token ceremony.
- Provider-queryable capability depth is bounded by what
  `ConnectorManager` exposes; anything unqueryable reports honestly as
  unavailable — no fabricated support.
- None of the steps require live Neon, Fly, OAuth, commits, or pushes;
  those remain owner-side or unauthorized per prompt §6.
