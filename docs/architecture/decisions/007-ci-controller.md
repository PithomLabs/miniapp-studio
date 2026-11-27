# ADR-007: CI controller topology and execution model

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** Small Go controller on a Fly Machine (webhook → verify → dedupe → persist → schedule → Sprite execution →
checks) with PostgreSQL-backed state + leases + outbox (no DBOS); each run executes on a fresh-or-verified-clean
Sprite against an ephemeral Neon CI branch; no GitHub Actions dependency for validation.
**Basis:** prompt28.2 §4 + §7; prompt28.3 §5; verified Sprite/GitHub mechanics (V2, V4–V11, V17).

## Context

CI must validate exact SHAs reproducibly without inheriting GitHub Actions' limits, while reporting standard commit
checks. The controller is narrowly scoped (schedule/supervise/report), not a workflow engine.

## Requirements

R204–R212, R309–R312, R230 (no CI↔preview circular gate): fast validation gates preview creation, never the reverse.

## Considered alternatives

- **GitHub Actions retained for CI:** rejected as the mandatory path — Actions' per-PR model cannot express the
  Sprite+Neon-branch execution contract; retained only as an optional mirror, not a gate.
- **DBOS-backed controller:** rejected for this scope — PG-backed state + leases + outbox + reconciliation covers the
  narrow controller; revisit only if durable-execution needs outgrow it (explicitly deferred, not assumed).
- **Machine-local execution (no Sprite):** rejected — untrusted PR code must not share the controller's machine.
- **Chosen:** Machine controller + per-run Sprite + per-run Neon branch.

## Security / operational / cost implications

Untrusted code runs only in Sprites with egress policy (V10) + ephemeral Neon branch. Controller holds GitHub App
private key + Neon project key (ADR-010/011) and never exposes them to runs. Cost per run = Sprite compute + Neon
branch/compute (parameterized caps, ADR-013).

## Failure modes

Controller restart mid-run (leases + reconcile, ADR-012/013); Sprite acquisition failure (bounded retry, honest
unknown-state); missed webhook (reconciliation, never webhook-only); stale SHA execution (payload head-SHA binding).

## Verification

LIVE: full §13 CI matrix — exact-SHA runs, canonical targets, checks published, failure/cancel honesty, orphan
reconciliation, restart recovery.

## Deferred

Multi-region controllers; run prioritization/queuing policy beyond simple caps.

## Developer-simplicity score

Positive: developers push and read checks; no YAML workflow authoring required for standard validation. Custom CI
needs are Taskfile targets, not controller changes.
