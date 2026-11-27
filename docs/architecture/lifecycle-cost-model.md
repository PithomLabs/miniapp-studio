# Lifecycle cost and capacity model (Stage 1, S1.6)

**Rule:** no hardcoded vendor figures — Sprite pricing (V13) and Neon quotas/pricing (N14) are unverified or volatile.
Caps are expressed in platform units with periodic recalibration against live pricing. This doc defines the dimensions,
the control points, and the calibration procedure; numeric values land in T-track plans after live verification.

## Cost dimensions

| Dimension | Unit | Driven by | Control (ADR) |
|-----------|------|-----------|---------------|
| Sprite compute (CI) | runs × duration × size | PR/commit velocity | Concurrency cap + per-run timeout (007, 013) |
| Sprite compute (agents) | runs × duration × size | Trigger rate × task size | Eligibility + per-run budget + timeout (014) |
| Neon branch count | concurrent branches | PRs + runs + dev branches | Max previews, TTLs, GC (013) |
| Neon compute | branch-hours × size | Branch lifetimes + activity | Idle expiry, scale-to-zero config (013) |
| Neon storage/history | GB + history window | Data size × retention | History-window tuning, fixture-first data (013, 015) |
| Model tokens | per agent run | Task size × attempts | Per-run budget, attempt caps (014) |
| Controller machine | fixed + scale | Always-on baseline | Single small machine; no per-run scaling (007) |

## Control points (all pre-provision)

1. Scheduling refuse: over-budget/over-cap work is refused before any resource exists, with an auditable reason.
2. Warning thresholds below caps; alerts on GC actions and spend anomalies.
3. Attribution tags (repo, PR, SHA, run ID) on every resource for per-PR cost review.

## Calibration procedure (Stage 2 entry requirement)

1. Read live Sprite pricing + Neon plan quotas/pricing; record source + date (closes V13/N14).
2. Set initial caps conservatively from the Demo-scale workload (single-digit concurrent PRs/runs).
3. Measure three live runs per track; recalibrate caps and budgets from observed consumption.
4. Re-read pricing quarterly or on vendor change notice; caps are parameters, never constants.

## Capacity risks

- Org active-Sprite limit bounds parallelism (V16 assumed) — verify before scale claims.
- Neon branch/project limits bound PR+run concurrency (N14) — verify before T2/T3 scale commitments.
- Preview lifetime × branch compute dominates idle cost — idle-expiry policy is the highest-leverage control.
