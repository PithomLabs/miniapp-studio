# ADR-016: Customer-miniapp database provisioning

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** One Neon project per customer miniapp as the default provisioning unit; sharing a project across
miniapps requires explicit justification against the criteria below. Database lifecycle is bound to the miniapp
lifecycle (provision → migrate → operate → backup → destroy/export on delete).
**Basis:** prompt28.3 §3; verified project/branch model (N3, N10); conflict C8.

## Context

This decision shapes the platform control plane: billing, quotas, recovery, and tenant isolation all follow the
project boundary. Neon itself positions project-per-tenant as the strongest isolation pattern for platforms.

## Requirements

R306: cost/isolation/quota/lifecycle/observability/recovery evaluation; no shared-project default.

## Considered alternatives

- **Single shared project, database-per-miniapp:** rejected as default — shared project concentrates blast radius,
  complicates per-miniapp billing/quotas/recovery, and makes credential scoping coarser.
- **Project per tenant (multi-miniapp tenants):** allowed as a justified variant — same isolation reasoning applied at
  tenant granularity when one tenant owns several miniapps; still one project per security boundary.
- **Project per miniapp (chosen default):** strongest isolation, cleanest billing/quotas/recovery, project-scoped
  keys per miniapp. Cost evaluated per plan (ADR-013); revisit only on N14 evidence.

## Security / operational / cost implications

Per-miniapp project keys bound least privilege naturally. Provisioning/deprovisioning is control-plane automation
(T5), never developer ritual. Project-count quotas (N14) are the scaling constraint to verify before launch-scale
claims.

## Failure modes

Project sprawl beyond plan limits (mitigation: quota monitoring + T5 entry-gate N14 check); orphaned miniapp
projects on failed delete (mitigation: ownership metadata + reconciler extends to T5 resources); cross-miniapp
credential confusion (mitigation: project-key binding, never shared keys).

## Verification

LIVE: provisioned miniapp uses its own project (inventory audit); isolation negative tests; delete-path destroys or
exports per policy; quota-headroom check.

## Deferred

Tenant-vs-miniapp granularity tuning (needs real tenancy data); backup/export format policy.

## Developer-simplicity score

Positive: miniapp developers get a database that simply exists with their app — no provisioning requests, no shared-
database coordination, no noisy-neighbor mysteries. Deletion is complete and automatic.
