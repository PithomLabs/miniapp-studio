# ADR-008: Sprite provisioning, lifecycle, and checkpoint strategy

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** One Sprite per independent execution, from fresh create or verified-clean checkpoint restore; long-lived
work supervised via sessions + detach + marker/heartbeat polling (never a held exec — V7 disputed); warm checkpoints
per repository for sequential tasks only, never shared mutable state across concurrent runs.
**Basis:** prompt28.2 §6.2, §6.5; verified CLI/networking/connector mechanics (V6–V12).

## Context

CI runs and agent runs need isolated, reproducible starting states and supervision that survives controller restarts
and network interruptions. The Fly article's exec-timeout model is contradicted by the official CLI reference.

## Requirements

R224, R228; per-run Sprite identity; checkpoint lifecycle; abandonment cleanup; pause/wake observation for same-PR
continuation (V15 assumed until live proof).

## Considered alternatives

- **Held-exec supervision with timeout flags:** rejected — no such flags exist (V7); design on contradiction would
  fail at first contact.
- **Shared warm Sprite across concurrent runs:** rejected — mutable sharing breaks run isolation and auditability.
- **Fresh-everything always:** rejected as the only mode — dependency reinstall cost per run is real; warm checkpoints
  are kept for sequential same-repo work with restore+pull semantics.
- **Chosen:** fresh default; checkpoint-restored warm start as an explicit per-run mode with clean-state verification.

## Security / operational / cost implications

Each Sprite gets an egress policy at creation (V10) scoped to the run's needs. No credentials baked into checkpoints
(R317). Cost: fresh-create cost vs warm-restore savings measured live before scale commitments.

## Failure modes

Checkpoint captures a secret (mitigation: checkpoint hygiene audit + no-secret-in-Sprite rule); restore yields dirty
state (mitigation: clean-state verification before use); orphaned Sprite on controller crash (mitigation: ownership
metadata + reconciler, ADR-013).

## Verification

LIVE: fresh vs restored equivalence test; secret-hygiene scan of checkpoints; kill/timeout/cancel supervision proofs;
pause/wake behavior observation (V15 proof).

## Deferred

Checkpoint retention/rotation policy tuning; cross-region Sprite placement.

## Developer-simplicity score

Neutral: developers never touch Sprites directly — provisioning, policy, and cleanup are controller-owned. Same-PR
continuation (sleep/wake) reduces developer wait without developer action.
