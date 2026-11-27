# ADR-010: Credential architecture (Neon, GitHub, agents, runtimes)

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** Strict credential-domain separation: Neon project-scoped API keys (control plane only, approved secret
store); per-branch least-privilege DB roles (SQL/`--access`-created, rotated before untrusted access); per-run
GitHub App installation tokens (repo+permission narrowed, 1h); model keys controller-held with spend caps. No domain's
credential ever appears in another domain's environment, logs, commits, artifacts, or checkpoints.
**Basis:** prompt28.2 §6.3 + §9; prompt28.3 §7; verified key/role/token mechanics (N7, N10–N11, V4–V5, V11).

## Context

Agents, CI, previews, and the controller each need different powers. A single shared credential (or a credential that
wanders across domains) turns any one compromise into a total compromise.

## Requirements

R225, R316–R317: domain table (controller / agent-run / CI-run / preview-app / migration / model); rotation-before-
untrusted-access; N8 inheritance proof; N9 protection investigation; `neon_superuser`/owner never as app credential.

## Considered alternatives

- **Single org-wide Neon key for all automation:** rejected — violates least privilege and makes per-project blast
  radius meaningless.
- **Default Neon roles for app/migration use:** rejected — Console/API/CLI-created roles carry `neon_superuser`
  membership (N7); app roles must be SQL-created or `--access`-scoped.
- **Long-lived agent GitHub tokens:** rejected — per-run 1h narrowed tokens bound the leak window (V4).
- **Connectors as the only GitHub path:** rejected as exclusive — connectors are Sprite-side only (V11 design
  consequence 3); controller API calls use installation tokens via its own path.

## Security / operational / cost implications

Secret store selection is an owner decision (candidate recorded in S1.8; approval before T-track use). Rotation is
automated per-branch (reset-before-use) and event-driven for keys (revoke+replace). Audit: every credential issuance
logged with domain, scope, and expiry.

## Failure modes

Inherited branch password reaching untrusted run (rotation-before-use + N8 proof); key material in checkpoint/log
(hygiene scans as CI gates); secret-store outage (fail closed: no new runs, existing runs unaffected until expiry).

## Verification

LIVE: domain-placement audit (each environment holds exactly its domains); rotation proof; cross-domain negative
tests (agent creds cannot provision; preview creds cannot reach other branches or prod); hygiene scans green.

## Deferred

Secret-store product choice; automated Neon key rotation cadence.

## Developer-simplicity score

Positive: developers never handle any of these credentials — provisioning, narrowing, rotation, and revocation are
fully automated. Developer-facing auth stays at the Studio session level.
