Increment 2 — CLOSED

Live Neon verification passed: `studio_dev_ui` is clean at v28, and the targeted `store` tests passed in 197.469 seconds. Proceed to Increment 3: **tenant invitations + capability refresh**.

Use this implementation assignment with the coding agent.

# Increment 3 — Tenant Invitations + Capability Refresh

Implement Increment 3 in the **current working tree**, preserving the completed Increment 1 secrets package and Increment 2 credential broker. Increment 2's live Neon gate is green: `studio_dev_ui` verified at migration v28; targeted `TestBroker|TestPrivilegeParity` tests passed.

## Mission

Deliver secure tenant invitations and a framework-native capability refresh mechanism. Reuse existing Studio infrastructure. Do not duplicate approvals, effects, intent-hash, trust, identity, tenancy, or credential-broker machinery.

## 1. Investigate before changing code

Inspect the current source and its governing contracts, especially:

- `AGENTS.md`, `PRINCIPLES.md`, and relevant architecture decisions.
- Existing organization/tenant/workspace, identity, session, membership, and authorization models.
- RLS policies, store transactions, grant mirrors, and privilege-parity tests.
- Existing capability definitions, provider/tool registries, tool attestations, and gateway contracts.
- The new `secrets/` and `broker/` packages and migration `000028`.

Treat the current working tree as authoritative. Do not reset, overwrite, or replace uncommitted work with GitHub `main`.

First establish what already exists and identify only the missing contracts and behavior. Make reasonable implementation decisions autonomously; do not create another plan-only cycle.

## 2. Tenant invitations

Implement the smallest complete invitation lifecycle consistent with the existing identity and tenancy model.

Requirements:

- Invitations are scoped to the existing tenant/workspace hierarchy and use the existing role and permission model.
- Only principals authorized by the existing policy may invite, revoke, or manage invitations.
- Invitation acceptance must bind the authenticated identity to the intended tenant and permitted role. Never trust tenant, role, or principal identifiers supplied by an untrusted client without authorization checks.
- Use high-entropy, single-use invitation tokens. Persist only a cryptographic token hash; never persist or log the raw token.
- Enforce expiry, revocation, replay resistance, and safe resend/rotation semantics.
- Make acceptance idempotent under retries and concurrent requests. Prevent duplicate or conflicting memberships.
- Prevent tenant enumeration and cross-tenant access through API responses and database behavior.
- Enforce authorization at the service/store boundary, not merely in UI controls.
- Apply RLS and explicit tenant context consistently with existing Studio patterns.

Do not invent a second authentication or membership system. Reuse existing session, identity, and membership contracts.

## 3. Capability refresh

Implement capability refresh using the existing capability model and provider/tool contracts.

Requirements:

- Derive refreshed capability state from authoritative registered implementations and provider attestations, not client claims or AI-generated assertions.
- Preserve the canonical `CapabilityRef` and existing trust/authority semantics.
- Clearly distinguish implemented, unavailable, planned, stale, and failed capability states where supported by the existing contract.
- Record refresh time and sufficient provenance to explain the resulting capability state.
- Make refresh safe under retries and concurrent execution. A partial or failed refresh must not publish a misleading complete snapshot.
- Unknown providers, malformed contracts, and unverified capabilities fail closed.
- A refreshed capability inventory must never itself grant execution permission, bypass approval, raise trust, or authorize an effect.
- Execution must continue to use the existing authorization, approval, effect-ledger, intent-hash, and broker-lease paths.

Do not fabricate provider support or implement unrelated provider integrations. When a provider cannot yet be queried, report that state honestly.

## 4. Persistence and security

Add a migration only where inspection proves it is required. Any new tenant-owned tables must have:

- Tenant-scoped keys and appropriate foreign keys.
- RLS enabled and restrictive policies.
- Explicit grants aligned with the existing role model.
- Matching updates to grant mirrors and privilege-parity tests.
- A reversible down migration.

Preserve the Neon-only database contract and production guard. Do not introduce local PostgreSQL, SQLite, or another persistence backend.

Do not expose refresh tokens, access tokens, invitation tokens, KEKs, or other secret values in logs, CLI output, test failures, or evidence documents.

## 5. Tests and evidence

Add tests that demonstrate the actual security invariants, including:

- Invitation authorization, tenant isolation, expiry, revocation, replay, resend, idempotent acceptance, and concurrent acceptance.
- Raw invitation tokens are not persisted or logged.
- Capability refresh derives only from authoritative registrations/attestations.
- Unknown, stale, failed, and cross-tenant refresh cases fail closed.
- Refresh cannot grant permission or bypass approval/effect controls.
- RLS, grant-mirror, and privilege-parity behavior for any new tables.
- Migration up/down behavior and existing data compatibility.

Run `gofmt`, build, vet, the full test suite, and relevant race tests where practical. Report database-backed tests as skipped when Neon is unavailable; never describe skipped tests as passing.

Write implementation evidence to `docs/rpa-f0-3-evidence.md`, including files changed, contracts, migration details, gates, skipped tests, and unresolved limitations.

## 6. Boundaries

This assignment authorizes local code, migrations, tests, and documentation only.

Do not perform live Neon migrations, Fly operations, provider OAuth enrollment, real invitation delivery, credential enrollment, production changes, commits, or pushes. Do not use real tenant credentials. Prepare the exact owner-side verification commands for any live database gate.

## 7. Completion standard

After implementation:

1. Perform a separate adversarial review against the actual code and tests.
2. Fix substantiated security, correctness, isolation, migration, and regression findings.
3. Do not reopen settled architecture or add unrelated abstractions.
4. Report concise results: implemented behavior, test evidence, migration version, adversarial findings, owner-side verification commands, and genuine remaining blockers.

Deliver the working increment, not another planning document. Stop only for a consequential owner decision or a genuine blocker; otherwise make the narrowest safe decision and document it.

The key boundary is that capability refresh reports what the platform can actually do; it must not silently expand what an agent is authorized to do.