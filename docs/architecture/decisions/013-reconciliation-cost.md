# ADR-013: Cleanup, reconciliation, quotas, and cost controls

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** Cleanup-as-reconciliation: every provisioned resource carries ownership metadata (repo, PR, SHA,
controller record); a periodic reconciler discovers expired, abandoned, and partially provisioned resources and
destroys them safely. Cost controls are parameterized caps (max previews, per-PR limits, lifetimes, idle expiry,
spend budgets) with attribution and alerts — no hardcoded vendor figures.
**Basis:** prompt28.2 §5.7 + §8; prompt28.3 §8; N4/N6/N14–N15, V13/V16.

## Context

Webhooks get missed, controllers crash mid-provision, branches outlive apps. Without reconciliation, the system leaks
money and state. Vendor figures (V13/N14) are unverified or volatile, so caps must be parameters, not constants.

## Requirements

R221, R315, R318–R319: explicit expiry on all ephemeral branches; orphan discovery; spend/concurrency caps enforced at
the scheduling point (controller refuses over-budget work before provisioning).

## Considered alternatives

- **Webhook-only cleanup:** rejected (ADR-012) — single point of failure for cost control.
- **Hardcoded price arithmetic in code:** rejected — figures drift; caps expressed in platform units (branches,
  previews, runs, budgets) with periodic recalibration against live pricing.
- **Manual cleanup runbooks:** rejected as the mechanism — runbooks document the reconciler's behavior for operators;
  humans are the backstop, not the loop.

## Security / operational / cost implications

Reconciler holds destroy authority scoped by ownership metadata (it cannot touch unowned resources). Alerts fire
before caps bite (warning thresholds) and on GC actions (audit trail). Attribution tags per PR/run enable cost review.

## Failure modes

Reconciler destroys a live resource (mitigation: ownership + liveness checks + generation fencing; dry-run mode for
audit); reconciler outage lets orphans accumulate (mitigation: orphan-age alerts; bounded blast radius via per-PR caps);
quota exhaustion blocks legitimate runs (mitigation: capacity alerts + graceful degraded scheduling).

## Verification

LIVE: orphan-injection tests (planted orphans collected); cap-enforcement tests (over-budget run refused pre-provision);
destroy-audit review; quota-behavior observation against live account limits.

## Deferred

Exact cap values (set from N14 live data + usage history); alert routing/ownership.

## Developer-simplicity score

Positive: developers never clean up after the platform — no stale previews, no orphan branches, no surprise bills from
leaked resources. Cost attribution is visible per PR without developer bookkeeping.
