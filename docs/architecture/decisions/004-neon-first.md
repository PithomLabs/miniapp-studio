# ADR-004: Neon-first PostgreSQL across every environment (supersedes ADR-002)

**Status:** Decided (owner hard constraint, prompt28.3; recorded 2026-11-27, Stage 1).
**Decision:** Neon-managed PostgreSQL is the mandatory provider for every PostgreSQL workload: Studio control plane,
development, agents, CI, per-PR previews, production, generated miniapps, internal services.
**Amendment (owner decision 2026-11-27):** the mandated major version is **PG18** (was PG19), tracking the Neon
project default. Rationale: comply with the provisioned project rather than recreating it; PG18 also sidesteps
Neon's PG19 feature caveats (N2: Data API/Realtime/anonymization/some extensions unavailable on 19). The Neon-only
mandate is unchanged — only the version number moved. Acceptance condition: miniapp framework compatibility must be
re-proven on PG18 via a Neon branch (the P0 gate ran on PG19); recorded pending in `docs/t1-evidence.md`.
**Supersedes:** ADR-002 (Fly Managed Postgres PG17, Neon as fallback).
**Basis:** prompt28.3 §1–§2, §11; verified Neon PG19 availability (N1); conflict records C1–C4.

## Context

ADR-002 chose MPG PG17 on cost and Fly-native grounds with Neon as fallback. The owner has since mandated Neon-only
as a hard architectural constraint at the same level as Fly.io, driven by the branch-per-environment lifecycle
(CI runs, PR previews, agent workspaces) that per-cluster provisioning cannot serve economically.

## Requirements

Single provider; PG19; branch-per-run/PR lifecycle; no self-hosted PG anywhere (apt/PGDG, Docker, Testcontainers,
in-Machine, in-Sprite); client tools (psql) remain allowed; exceptions need owner-approved amendment.

## Considered alternatives

- **Keep ADR-002 (MPG) with Neon fallback:** rejected — violates the owner mandate; per-PR/per-run clusters are
  operationally and economically incoherent next to copy-on-write branches.
- **Dual-stack (MPG prod + Neon ephemeral):** rejected — two providers doubles credential domains, verification
  contracts, and failure modes; explicitly forbidden by prompt28.3 §2 ("do not attempt to satisfy both").
- **Neon-first (chosen):** one provider, one credential model, one verification contract (R312).

## Security / operational / cost implications

Security concentrates on Neon API-key custody + branch credential rotation (ADR-010). Operations simplify (no PG
installation, patching, or container lifecycle anywhere). Cost moves from fixed cluster spend to branch/compute/storage
metering — requires the quota/cost architecture of ADR-013 before scale commitments.

## Failure modes

Neon control-plane/API outage blocks provisioning (mitigation: durable desired-state + retry/reconcile, ADR-012/013);
wrong-project misconfiguration (mitigation: project-ID pinning + preflight checks, ADR-005/006); stale local-PG
procedures followed by habit (mitigation: S1.6 doc rewrite + prohibition audit R302).

## Verification

LIVE inventory audit in Stage 2: zero non-Neon PostgreSQL servers across dev/CI/preview/prod; grep audit over repo
for forbidden provisioning paths; `SHOW server_version` = 19.x on every managed endpoint.

## Deferred

Exact Neon plan selection (needs N14 live quota/pricing check); per-miniapp project boundaries (ADR-016).

## Developer-simplicity score

High positive: developers never install, patch, back up, or port-map PostgreSQL again. One `task dev:setup` yields a
working branch. Complexity absorbed by controllers and automation, per the governing goal.
