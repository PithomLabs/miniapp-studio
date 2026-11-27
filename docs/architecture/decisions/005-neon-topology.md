# ADR-005: Neon project and branch topology

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** Separate Neon production project(s) from a non-production project; all ephemeral environments are
branches of the non-production project; untrusted branches are never derived from sensitive production data.
**Basis:** prompt28.3 §3; verified branching semantics (N3–N6); conflict C2.

## Context

Neon projects are the top-level resource boundary; branches within a project give lightweight isolation. The design
must place persistent developer branches, agent branches, CI branches, PR branches, and production workloads so that
a compromised ephemeral credential cannot reach production data.

## Requirements

Prod/non-prod separation; branch-per-developer (persistent), per-agent-run, per-CI-run, per-PR; synthetic/schema-only
or anonymized parents for untrusted branches; project-per-x for miniapps decided in ADR-016.

## Considered alternatives

- **Single project, branches for everything:** rejected — a leaked branch credential plus project-scoped API key stays
  inside one blast radius containing production; violates least privilege.
- **Project per environment class (chosen):** production project(s) hold protected branches; one non-production project
  hosts dev/agent/CI/PR branches. Stronger isolation boundary at project level; branch-level isolation within.
- **Project per PR:** rejected for Studio PRs — multiplies project-quota consumption and API-key scope for isolation
  that branch-level controls already provide; revisit only if N14 quotas force it.

## Security / operational / cost implications

Production branches are unreachable from non-prod credentials by construction (distinct projects, distinct keys).
Operations: project IDs pinned in config + preflight validation. Cost: branch metering concentrated in non-prod;
prod history-window/storage tuned independently.

## Failure modes

Mis-pinned project ID routes ephemeral branches at production (mitigation: preflight project-identity check + deny
branch-create on prod project from automation keys); parent-branch contamination with sensitive data (mitigation:
ADR-015 lineage rule + audit).

## Verification

LIVE: topology inventory (project IDs, branch parents) on every lifecycle operation; negative test proving non-prod
credentials cannot list/read prod branches; lineage audit per R305.

## Deferred

Miniapp project boundaries (ADR-016); exact plan/limits (ADR-013, needs N14).

## Developer-simplicity score

Positive: developers interact with one named dev branch; project topology is invisible to them. Ephemeral-branch
mechanics are fully automated (create → use → delete) with no developer cleanup duty.
