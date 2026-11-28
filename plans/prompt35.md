# Retrospective Adversarial Security Review — RPA F0 Increments 1–3

## Mission

Perform an independent, adversarial code review of the completed F0 implementation across:

- **Increment 1 — Secrets:** `secrets/`, Taskfile commands, server configuration adoption, and `docs/secrets.md`.
- **Increment 2 — Credential Broker:** `broker/`, migration `000028`, KEK handling, OAuth2 refresh exchange, credential enrollment, leases, consumption, revocation, and grant/parity changes.
- **Increment 3 — Invitations + Capability Refresh:** `invite/`, `capability/`, migrations `000029` and `000030`, invitation acceptance, token-file handling, capability snapshots, Taskfile commands, and grant/parity changes.

Read the actual source code, migrations, tests, architecture decisions, and evidence documents:

- `docs/rpa-f0-1-evidence.md`
- `docs/rpa-f0-2-evidence.md`
- `docs/rpa-f0-3-evidence.md`
- `docs/secrets.md`
- `docs/broker.md`
- `docs/invitations.md`
- `docs/capabilities.md`
- Relevant store, server, gateway, tools, dbenv, and Taskfile implementations.

Evidence documents describe claims to verify; they are not proof that the implementation is correct.

**Do not begin with a presumption that the increments are secure because their tests pass. Attempt to break their security invariants.**

## 1. Review principles

- Review the current working tree, not an assumed GitHub version.
- Preserve existing user changes. Inspect Git status before modifying anything.
- Trace data and authorization flows end to end, including database boundaries and CLI entry points.
- Investigate root causes, not just individual symptoms.
- Treat tenant isolation, credential confidentiality, authorization, and fail-closed behavior as security boundaries.
- Reuse existing Studio contracts. Do not introduce replacement authentication, authorization, approval, effect, or tenancy machinery.
- Do not invent new product scope or reopen settled architectural decisions.
- A passing unit test is not sufficient evidence when the invariant can fail at a different boundary.

## 2. Adversarial attack surfaces

### A. Secrets management — Increment 1

Attempt to find ways to disclose, override, bypass, or corrupt secrets.

Examine:

- Registry completeness, duplicate variables, unknown keys, invalid names, and secret defaults.
- Precedence across flags, process environment, `_FILE`, dotenv, and defaults.
- Conflicting sources and whether every ambiguous configuration fails closed.
- Dotenv parsing limits, malformed input, duplicate keys, empty values, and unexpected syntax.
- Secret-file permissions, symlinks, path replacement, TOCTOU races, and permission-check/read races.
- `_FILE` loading behavior and direct-variable conflicts.
- Error messages, logs, panic paths, subprocess output, and redaction coverage.
- DSN and credential-shaped values that could bypass redaction.
- Whether Taskfile commands expose secret contents or leak them through shell tracing or errors.
- Whether server configuration adoption preserves valid legacy behavior while enforcing the documented security changes.

Specifically challenge the claim that secrets never appear in terminal output, logs, errors, or evidence.

### B. Credential broker — Increment 2

Attempt to decrypt, redirect, reuse, steal, or incorrectly authorize tenant credentials.

Examine:

- AES-256-GCM nonce generation, ciphertext format, AAD binding to tenant and provider, and tamper detection.
- KEK loading, file permissions, key identity, rotation assumptions, and failure behavior.
- Whether refresh tokens are ever persisted or exposed in plaintext.
- OAuth2 refresh exchange behavior, HTTP timeouts, redirects, TLS validation, response limits, malformed responses, and error handling.
- Whether provider-controlled responses or error strings can leak credentials.
- Mint scope validation: tenant, provider, capability, intent hash, and TTL.
- Whether access tokens remain only in broker memory and are excluded from database rows, logs, CLI output, errors, and audit metadata.
- Lease binding, expiry, single-use consumption, and atomicity under concurrent consumers.
- Revoke-versus-mint and revoke-versus-consume races.
- Whether `invalid_grant` revocation can affect the wrong credential.
- RLS, database grants, service-role boundaries, and cross-tenant access.
- Whether a caller can use a valid lease for a different tenant, capability, or intent.

Challenge the design's central promise: refresh tokens remain broker-controlled, while executions receive only narrowly scoped, short-lived access.

### C. Tenant invitations — Increment 3

Attempt to accept, replay, enumerate, forge, revoke, or misuse invitations.

Examine:

- Token entropy and hash-only persistence.
- Token creation, reservation, file permissions, held file descriptors, path replacement, and cleanup.
- Tenant and binding assignment: can untrusted input select another tenant or change the invited binding?
- Inviter authorization at the service and store boundaries.
- Expiry, revocation, resend rotation, duplicate invitations, and replay.
- Uniform failure responses and possible timing or error-based enumeration.
- The `invitation_accept` SECURITY DEFINER function: fixed search path, privileges, tenant context, function ownership, and atomic behavior.
- Atomicity across invitation consumption, operator provisioning, and session creation.
- Concurrent acceptance and retry behavior.
- Session creation failure, transaction rollback, and whether any partial state can survive.
- Whether raw tokens can leak through logs, errors, tests, terminal output, or persisted files.

Verify the documented failure-injection guarantee: session creation refusal must roll back the invitation consumption and operator provisioning, allowing a legitimate retry.

### D. Capability refresh — Increment 3

Attempt to make the capability inventory lie or become an authorization bypass.

Examine:

- Whether the snapshot derives exclusively from authoritative manifests, bound handler registrations, and connector state.
- Whether a declaration without a registered implementation can appear executable.
- Whether handler-map checks merely inspect registration or accidentally imply execution authorization.
- Unknown providers, malformed contracts, stale data, unavailable connectors, and failed collection.
- Atomic publication and preservation of the last known-good snapshot.
- Concurrent refresh behavior and snapshot consistency.
- Tenant isolation and connector-manager scoping.
- Read behavior for missing rows versus corruption, transport failures, or malformed snapshots.
- Stable JSON shape and provenance.
- Whether refreshed capability data can affect execution authorization, trust, approval, broker minting, or effect dispatch.

Prove that capability refresh is descriptive only. It must never grant permission or expand execution authority.

### E. Cross-increment and database boundaries

This is a critical part of the review. Do not review each package in isolation.

Trace the complete path:

`secrets/configuration → KEK → broker credential → scoped lease → capability selection → invitation/tenant identity → existing authorization and execution machinery`

Determine whether a weakness in one increment can undermine another.

Inspect:

- Migrations `000028`, `000029`, and `000030`, including up/down correctness and dependency ordering.
- RLS enablement and FORCE RLS where required.
- Tenant context setup, reset, pooling, and fail-closed behavior.
- SECURITY DEFINER function ownership, search paths, grants, and revocation.
- `dbenv` grant mirrors, privilege-parity tests, and provisioning guards.
- Direct table privileges and unintended access paths.
- Environment and production guards on every new CLI verb.
- Consistency between application-level checks and database enforcement.
- Whether down migrations leave orphaned privileges, functions, or state.
- Whether concurrent operations can cross tenant or execution boundaries.

Do not assume the grant mirror or parity tests are complete merely because they currently pass. Independently inspect the actual grants and policies.

## 3. Required adversarial techniques

Use concrete attack scenarios rather than checklist-only inspection.

Where practical:

- Add negative tests that reproduce suspected vulnerabilities.
- Use malformed, oversized, conflicting, and adversarial configuration inputs.
- Exercise concurrent mint/consume, invitation accept/resend/revoke, and capability refresh.
- Use failure injection around transaction boundaries.
- Verify database permissions with the actual role model.
- Search for secret-bearing values in logs, command output, errors, and persisted data using canary values.
- Inspect migrations directly and verify assumptions against executable tests.
- Trace call graphs to prove that capability refresh cannot reach execution or authorization paths.

Do not weaken a security assertion merely to make a test pass.

## 4. Finding classification

Record every substantiated finding with:

- ID and severity: Critical / High / Medium / Low.
- Affected increment and exact file/line.
- The violated security invariant.
- A concrete attack or failure scenario.
- Root cause.
- Reproduction steps or a failing regression test.
- Recommended correction.
- Actual disposition and verification evidence.

Prioritize exploitable vulnerabilities, tenant-isolation failures, credential leakage, authorization bypasses, transaction-integrity defects, and destructive migration errors.

Do not inflate severity for stylistic preferences. Do not bury a material issue as a minor improvement.

## 5. Remediation authority

You are authorized to make local code, migration, test, and documentation changes within the three-increment review scope.

- Fix substantiated Critical, High, and Medium findings.
- Fix Low findings when they create a real security or correctness defect; otherwise record them without expanding scope.
- Add regression tests demonstrating the invariant.
- Re-run affected tests after each meaningful fix.
- Do not perform unrelated refactoring or introduce speculative abstractions.
- Do not reopen settled architecture merely because another design is possible.
- Do not stop for routine implementation choices; make the narrowest safe decision and document it.
- Stop and report if a consequential owner decision or genuine architectural blocker is unavoidable.

## 6. Verification gates

Run and report:

- `gofmt -l .`
- `go build ./...`
- `go vet ./...`
- `go test ./... -count=1`
- Relevant package-specific race tests, including broker, invitation, and capability concurrency tests.
- Relevant migration, RLS, grant, and privilege-parity tests.

Use only the existing Neon-only database contract.

**Do not run live Neon migrations, Fly operations, provider OAuth enrollment, credential enrollment, production mutations, or external spending.** Those require separate owner authorization.

If database tests skip because Neon is unavailable, state that explicitly. Never describe skipped tests as passing. Prepare exact owner-side commands for any remaining live verification.

Investigate the previously observed transient full-suite failure enough to determine whether it is reproducible or materially affects confidence. Do not attribute it to an unrelated flake without evidence.

## 7. Deliverable

Create:

`docs/rpa-f0-adversarial-review.md`

Include:

1. Scope and reviewed commit/worktree identity.
2. Executive verdict: APPROVED / APPROVED WITH LIMITATIONS / NOT APPROVED.
3. Review coverage by increment and cross-increment boundary.
4. Findings, severity, reproduction, and exact code references.
5. Fixes made and regression tests added.
6. Findings not fixed, with evidence-based rationale.
7. Test and race results with exact pass/skip/fail counts.
8. Database and migration verification status.
9. Remaining risks and exact owner-side verification commands.

Finish with a concise terminal report stating whether F0 increments 1–3 are safe to close from a code-review perspective, what remains unverified, and whether Increment 4 may proceed.

**Do not claim the review is clean simply because all existing tests pass. The review must demonstrate that the security invariants withstand adversarial attempts.**

Do not commit or push changes.