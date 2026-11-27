# ADR-006: Endpoint and security-model redesign for Neon (resolves C4)

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** Retire the loopback endpoint model and re-express the endpoint invariant for Neon: pinned Neon project +
known branch + TLS + least-privilege branch-scoped role. `testdb`/`dbenv`/`studio dev`/Taskfiles/fingerprint are
redesigned to this model; no allowlist exceptions.
**Amendment (T1 hardening, 2026-11-27):** "TLS" above means `sslmode=verify-full` exclusively — certificate-chain AND
hostname verification. Lesser modes are refused pre-dial (`require` encrypts without identity proof — pgx implements
it as InsecureSkipVerify without a root cert; `verify-ca` skips the hostname; both verified in pgx v5.13.0 source),
and `dbenv.Pool` pins the verifying `tls.Config` (system roots, ServerName = endpoint host, TLS 1.2+) in code rather
than trusting the DSN string. Live handshake negatives (untrusted CA, hostname mismatch) are acceptance tests, recorded
pending in `docs/t1-evidence.md` until Neon execution.
**Basis:** prompt28.3 §4–§5, §9; conflict C4; verified Neon roles/endpoints model (N7, N10–N11).

## Context

The current dev contract assumes `127.0.0.1:5434` + local trust + fixed test credential. Neon endpoints are remote
TLS hostnames with per-branch credentials. Pointing the old model at Neon via exceptions would destroy the invariant
the allowlist exists to enforce.

## Requirements

New endpoint policy (Neon project/branch allowlisting); TLS verification; per-branch least-privilege roles
(SQL-created or `--access`-scoped, never default `neon_superuser` members); revised fingerprint (R312 fields);
revised `studio dev` verbs; prohibition audit for non-Neon PG (R302).

## Considered alternatives

- **Allowlist exception for Neon hostnames:** rejected — exception-per-hostname recreates silent substitution; the
  invariant must name projects/branches, not carve holes for hosts.
- **Loopback relay/proxy to Neon:** rejected — adds a spoofable middlebox, hides TLS identity, complicates every
  environment for zero safety gain.
- **Model redesign (chosen):** policy names (project ID, branch role, TLS required); connection strings resolved from
  branch metadata at runtime, never hardcoded.

## Security / operational / cost implications

Security: endpoint identity is cryptographic (TLS + project/branch binding) rather than topological (loopback).
Operations: `studio dev doctor` gains Neon preflight (auth, project, branch, version). Cost: none directly.

## Failure modes

Stale hardcoded DSNs pointing at loopback PG (mitigation: prohibition audit + removal in T1); TLS verification
disabled for convenience (mitigation: no such flag exists in the redesigned verbs); branch confusion across
environments (mitigation: branch↔purpose association records, ADR-009/012).

## Verification

LIVE: redesigned verbs against real branches; negative tests (wrong project, wrong branch, non-TLS, overprivileged
role all refused); fingerprint field audit; R302 grep + runtime audit.

## Deferred

Exact verb/flag UX (T1 execution plan); migration of existing local run DBs (one-way, documented).

## Developer-simplicity score

Neutral-to-positive: developers keep one-command flows (`task dev:setup`, `studio dev test`); the added preflight
output is diagnostic, not ritual. No developer-managed certificates or connection strings.
