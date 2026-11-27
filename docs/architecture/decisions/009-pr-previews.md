# ADR-009: Per-PR preview architecture

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** Each eligible Studio PR gets an isolated Fly app + a dedicated Neon branch; deployments serialized
per PR with SHA+generation fencing; DB strategy is persistent-branch-per-PR with per-commit migration discipline
(first choice, verified in T3 — per-commit-generation branches as fallback if migration-across-commits proves
unsafe); preview serves Studio development PRs, not customer miniapps.
**Basis:** prompt28.2 §5 + §7; prompt28.3 §6; verified branching + GitHub mechanics (N3–N6, V2, V17).

## Context

Previews must prove the deployed app starts, migrates, and works against its own database. The 28.2 cluster-vs-database
choice is re-expressed for Neon: branch-per-PR (chosen) vs project-per-PR (rejected in ADR-005).

## Requirements

R213–R222, R313–R315, R230: isolation invariant, lifecycle events, fencing, naming, fixtures, exposure policy,
observability, reconciliation with Fly state.

## Considered alternatives

- **Per-commit-generation branches:** fallback — strongest per-commit isolation but multiplies branch churn and
  complicates data continuity; adopted only if persistent-branch migration proves unsafe in T3 proofs.
- **Shared staging with per-PR schemas:** rejected — violates the isolation invariant (one bad migration breaks all PRs).
- **Persistent-branch-per-PR (chosen):** one branch per PR lifecycle; migrations applied in commit order with
  generation fencing; atomic app-config switch per deployment.

## Security / operational / cost implications

Preview apps connect exclusively to their PR branch (exclusivity proof required). Exposure defaults to authenticated;
public only with explicit justification (V12 warning honored). Cost: per-PR branch+compute metered with caps (ADR-013).

## Failure modes

Stale deployment overwrites newer commit (fencing); migration order violation across rapid pushes (serialization +
generation check); orphaned branch/app on missed close event (reconciler, never webhook-only); prod-data leak into
preview (ADR-015 lineage rule).

## Verification

LIVE: full §13 preview matrix — two-PR isolation, rapid-push fencing, stale-job rejection, URL/health reporting,
close-triggered destruction, missed-webhook recovery, exclusivity audit.

## Deferred

Preview lifetime/idle-expiry tuning (needs usage data); branch-aware storage adoption (ADR-018).

## Developer-simplicity score

High positive: every PR automatically yields a working deployment with URL on the PR. Developers and reviewers click
a link instead of checking out branches and running migrations locally.
