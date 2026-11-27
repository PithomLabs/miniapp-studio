The next step is to move from **T1 repo-local implementation to T1 live Neon verification**.

Since both Neon and Fly CLI are already authenticated, the implementation agent should reuse those sessions. It should not ask you to log in again or request API keys in chat.

I recommend one controlled execution prompt: inspect first, establish the Neon non-production environment, complete the owner gates, then run the live verification suite.

One important distinction: Fly CLI authentication is useful for future T2/T3 work, but **do not provision Fly infrastructure yet**. Option A prioritizes the Neon-backed B2 demonstration.

## **Prompt28.4 — T1 Live Neon Provisioning and Verification**

# Prompt28.4 — T1 Live Neon Environment Provisioning and Verification

## Objective

Advance Miniapp Studio from approved T1 repo-local implementation to verified T1 live Neon development.

The following work is already completed and must be preserved:

- C4 loopback-to-Neon endpoint redesign.
- R312 Neon-aware fingerprint.
- `dbenv.Pool`, `PoolConfig`, and `TLSConfigFor`.
- Exclusive `sslmode=verify-full` enforcement.
- Pinned TLS configuration: system certificate roots, endpoint hostname verification, TLS 1.2 minimum, no verification bypass.
- All ten database contract-path pool sites migrated.
- `studio dev` revisions and `task dev:setup`.
- R302 repository audit.
- Updated ADR-006, `docs/dev-contract.md`, and `docs/t1-evidence.md`.

Latest repo-local evidence:

- `go build ./...`: PASS.
- `go vet ./...`: PASS.
- gofmt: CLEAN.
- Full suite: 396 pass, 304 skip, 0 fail.
- TLS refusal matrix and refusal-before-dial: PASS.

Do not redesign or regress these already-approved contracts.

The remaining task is to establish an actual Neon development environment and demonstrate the same behavior against a real Neon PostgreSQL 19 endpoint.

## 1. Governing constraints

Neon is the mandatory PostgreSQL provider.

No:

- Local PostgreSQL server.
- PGDG installation.
- Docker PostgreSQL.
- PostgreSQL Testcontainers.
- PostgreSQL inside Fly Machines or Sprites.
- Production database or credentials in development.
- Weakening of TLS verification.
- Unapproved endpoint-policy exceptions.
- Secret exposure in terminal output, logs, source control or reports.

Go implements database and provisioning behavior.

Existing Taskfiles orchestrate workflows.

`studio dev` remains the canonical development interface.

Preserve the three-lifecycle separation: agent execution, CI and PR preview.

The current objective is T1 only.

Do not begin T2 Fly-native CI, T3 previews or T4 background-agent infrastructure.

## 2. Phase A — Read-only environment and repository preflight

Reuse the existing authenticated Neon and Fly CLI sessions.

Do not repeat login procedures unless an authenticated session is demonstrably unavailable.

Inspect:

1. Current Git branch, HEAD, working-tree changes and untracked files.
2. Existing `docs/t1-evidence.md`, including its TLS addendum.
3. ADR-004, ADR-005, ADR-006 and ADR-010.
4. `plans/stage2-tracks.md` and the T1 acceptance requirements.
5. Current `Taskfile.dev.yml` and `studio dev` command implementation.
6. Existing Neon CLI version and exact supported commands using its current help.
7. Current Neon authenticated organization/account context.
8. Existing Neon projects, relevant non-production projects, available regions, and PostgreSQL version capabilities.
9. Current Fly CLI authentication and organization context, read-only.

Never print API keys, passwords, complete connection strings or tokens.

For authenticated context, report only necessary account, organization and project identifiers.

Do not assume that CLI authentication grants permission to create or destroy infrastructure.

### Deliverable

Produce a concise preflight report that identifies:

- Existing Neon projects suitable for non-production use.
- Whether a dedicated non-production project is necessary.
- Candidate region and justification.
- Actual Neon plan and available quota information.
- Whether PG19 is selectable.
- Existing secret-management infrastructure.
- Exact unresolved owner decisions.
- Exact provisioning operations that would follow approval.

If an existing project already satisfies ADR-005, prefer evaluating reuse over unnecessarily creating another project.

## 3. Phase B — Formal owner decisions

Prepare the formal acceptance packet for:

- ADR-004: Neon-first PostgreSQL.
- ADR-005: Neon project and branch topology.
- ADR-006: Neon endpoint and TLS security contract.
- ADR-010: Credential and secret-management architecture.

Summarize each decision, its implications, alternatives rejected, and operational consequences.

Do not mark an ADR accepted merely because its repo-local implementation is complete.

Obtain explicit owner acceptance for the four ADRs.

### Secret-store selection

Inspect existing available secret-management infrastructure.

Evaluate the existing OpenBao setup as a candidate before introducing another product.

Consider:

- Local development credential storage.
- Neon management API key storage.
- Database credential issuance and rotation.
- Fly application runtime secrets (future T2/T3).
- Agent and CI credential isolation.
- Auditability and recovery.
- Developer operational burden.

Keep Neon management credentials completely separate from application database credentials.

Present one recommended secret-store arrangement and its rationale for owner acceptance.

Do not put secrets in plaintext `.env` files or repository configuration.

### Gate

Do not create Neon resources or write secrets before the necessary owner approvals.

If the required decisions are already explicitly accepted, record that evidence and proceed without asking again.

## 4. Phase C — Provision Neon non-production environment

Once the prerequisites are approved, provision or reconcile the designated Neon non-production project.

Use the current documented Neon CLI or supported Neon API. Verify exact commands and flags against current installed CLI help and official documentation.

Requirements:

1. Use the approved Neon organization and project.
2. Select PostgreSQL 19.
3. Select a region based on the accepted architecture and future Fly deployment topology.
4. Use synthetic or schema-only data.
5. Establish the designated non-production parent branch.
6. Create or reconcile the developer branch.
7. Record project ID, branch ID and parent branch ID.
8. Verify actual PostgreSQL version through the live connection.
9. Verify TLS endpoint details.
10. Apply explicit resource ownership and lifecycle metadata.

Never derive the development branch from production data.

Do not create customer-miniapp projects at this stage.

### Database roles

Follow ADR-010.

Create least-privilege database roles appropriate to development and migration operations.

Do not use Neon management credentials as database credentials.

Do not use `neon_superuser` or an unrestricted project-owner credential as an application runtime credential.

The existing fixed test-role design may be retained only within the approved non-production synthetic-data scope, pending the T2 per-run credential implementation.

Investigate role and password inheritance behavior. Do not assume a new branch automatically receives independent credentials.

Never expose credentials in terminal output.

Persist credentials directly to the approved secret store.

Do not write them to committed files or reusable agent checkpoints.

## 5. Phase D — Live development setup

Inspect the actual implementation of:

- `task dev:setup`
- `studio dev setup`
- `studio dev doctor`
- `studio dev fingerprint`
- `studio dev init`
- `studio dev migrate`
- `studio dev test`

Reconcile any discrepancy between the documented workflow and actual command responsibilities.

Do not bypass the canonical interface with manually reconstructed shell commands.

The intended developer experience should be:

`task dev:setup`

Followed by the documented canonical initialization and validation commands.

Prove that:

- The Neon project is recognized.
- The expected developer branch is recognized.
- The connection uses the expected branch endpoint.
- `sslmode=verify-full` is enforced.
- The certificate chain is trusted.
- The endpoint hostname is verified.
- The actual PostgreSQL major version is 19.
- Database initialization and migrations work from a clean branch.
- The fingerprint reports the correct R312 identity.
- The fingerprint never contains credentials.

Preserve idempotency. Repeated setup must not create uncontrolled duplicate branches or rotate valid credentials unexpectedly.

## 6. Phase E — Live TLS security verification

Complete the two previously documented pending handshake tests.

### N-TLS-1: Untrusted CA

Use the actual provisioned Neon developer endpoint with an intentionally untrusted CA configuration.

Expected result:

- TLS handshake fails.
- The connection is rejected.
- No SQL is executed.
- No fallback to `require`, `verify-ca` or disabled certificate verification occurs.

### N-TLS-2: Hostname mismatch

Use the actual endpoint with an intentionally incorrect TLS ServerName.

Expected result:

- Hostname verification fails.
- The connection is rejected.
- No SQL is executed.
- No verification bypass occurs.

Also prove that the correctly configured Neon endpoint connects successfully using `verify-full`.

Record precise error classifications and safe diagnostic evidence.

Do not weaken the policy to make any negative test pass.

## 7. Phase F — Live PostgreSQL 19 and database contract proof

Execute the complete T1 verification matrix against the real Neon developer branch.

### PostgreSQL identity

Verify:

- `SHOW server_version`.
- `SHOW server_version_num`.
- PostgreSQL major version 19.
- Actual project, branch and parent branch identity.
- Endpoint hostname, port and TLS configuration.

### Extensions

Re-prove the required extension inventory against PG19.

The current expected inventory is none, but verify this from both repository migration chains and the actual database.

Do not infer PG19 compatibility from earlier versions.

### Schema and migrations

Prove:

1. Fresh branch starts with no application migrations.
2. Canonical migration sequence completes.
3. Both migration chains reach expected versions.
4. Expected tables, indexes and constraints exist.
5. Tenant isolation and RLS tests execute successfully.
6. Application and migration roles retain their intended privileges.

### Database test suite

Run the representative database subset first.

Then run the complete DB-dependent suite using the canonical interface.

Repeat the clean-bootstrap and validation sequence to demonstrate reproducibility.

Record exact:

- Commands.
- Commit SHA.
- Project and branch IDs.
- Actual PostgreSQL version.
- Migration versions.
- Pass counts.
- Skip counts and reasons.
- Failures.
- Cleanup results.

A live-test skip is never a pass.

If anything fails, identify the root cause and implement the structural correction within the approved scope. Do not weaken the underlying invariant to obtain green results.

## 8. Phase G — Runtime R302 audit

Perform a repository and runtime audit confirming:

- No self-hosted PostgreSQL server provisioning exists in live Go execution paths.
- No Docker PostgreSQL is needed for development.
- No PostgreSQL server exists inside Sprites or Machines as a workaround.
- No PostgreSQL Testcontainers path is silently represented as Neon testing.
- Historical `testrunner` and `hostexec` code remains clearly off-path unless separately approved for removal.
- Superseded CI configuration is accurately identified for T2 replacement.
- No active development path accepts unapproved loopback/private database endpoints.

Inspect actual imports and call paths, not merely filenames and grep results.

## 9. Phase H — Documentation and evidence

Update:

- `docs/t1-evidence.md`
- `docs/dev-contract.md`
- `docs/dev-parity-report.md`
- `plans/stage2-tracks.md` if the execution evidence requires it.
- Any affected ADR verification records.

Fill the development environment column of the parity report with actual live evidence.

Keep CI and preview columns pending.

Maintain explicit statuses:

- VERIFIED.
- BLOCKED.
- PENDING.
- DEFERRED.
- SUPERSEDED.

Do not call T1 live verification complete until all mandatory acceptance proofs pass or have an explicitly owner-approved disposition.

Never report planned work as completed.

## 10. Authorization and resource safety

Use the currently authenticated Neon CLI session.

Fly CLI is available but must remain read-only during T1.

No Fly apps, Machines, Sprites or secrets are to be created or modified as part of this stage.

Before creating Neon projects or branches, identify the exact intended resource and confirm the applicable owner authorization.

Every destructive operation, including branch deletion, requires separate explicit authorization.

Never delete or modify existing projects or branches based solely on naming conventions.

Never destroy the original parent branch.

Never reveal credentials.

If an operation would exceed the approved scope or requires an unapproved account permission, stop and report the precise blocker.

## 11. Required final report

Deliver a T1 live-verification report containing:

1. Provisioned Neon project and developer branch identifiers.
2. Actual PostgreSQL version.
3. Actual TLS verification results.
4. N-TLS-1 and N-TLS-2 results.
5. Extension inventory proof.
6. Migration-from-zero evidence.
7. Complete database test counts.
8. RLS and tenant-isolation evidence.
9. Credential and secret-handling audit.
10. R302 runtime audit.
11. Final developer fingerprint.
12. Updated parity report.
13. Remaining unresolved risks and conditional proofs.
14. Exact next recommendation for Option A B2 demonstration.

The report must clearly distinguish live-verified behavior from repo-local unit-test evidence.

## 12. Completion criteria

T1 is complete only when the actual Neon-backed developer environment is reproducible, securely configured, and validated through the canonical interface.

Do not begin T2, T3 or T4 automatically.

The next owner decision after successful T1 completion is whether to proceed with the minimal B2 demonstration.

**Governing principle: Make the real system work. Prove the security boundaries. Preserve reproducibility. Never trade architectural correctness for a green test count.**

## **My additional recommendation**

There is one subtle issue I would prioritize: **do not let the developer database branch become the parent of agent or CI branches by default.**

Use a deliberately controlled, synthetic-data parent with an explicit lifecycle. Developer branches, CI branches, agent branches and PR branches should have clearly defined ancestry and credential rotation rules.

This matters because Neon role/password inheritance is still conditional in the evidence, and that assumption must be resolved before untrusted workloads receive access.

The immediate sequence should be:

1. Read-only Neon/Fly authentication and account preflight.
2. Formal ADR acceptance and secret-store selection.
3. Neon non-production project and developer branch provisioning.
4. Live TLS, PG19, migration and database-suite verification.
5. Complete T1 evidence and proceed to the minimal B2 demonstration.

The Fly CLI is already logged in, so the agent can record the authenticated organization context now, but should leave Fly infrastructure untouched until the appropriate roadmap track.

**The next milestone is not CI or preview infrastructure. It is a fully working, genuinely Neon-backed development environment that proves the architecture we just built.**