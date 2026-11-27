# ADR-017: Branch protections and required checks

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** `main` is protected; merge requires human review plus the required-check set (fast validation, full
integration, preview smoke where applicable) published under distinct check names with explicit gating relationships.
Agents cannot approve, cannot merge, and cannot satisfy their own gates; agent pushes follow the same rules as human
pushes.
**Basis:** prompt28.2 §4.3 + §7; verified checks/event mechanics (V2, V17).

## Context

Agent throughput makes branch protection load-bearing: without it, a compromised or confused agent merges itself.
Distinct check names make the gating legible and prevent circular dependencies (CI gating preview creation while
preview gates CI).

## Requirements

R212, R227, R230: required checks; no self-approval; no circular gates; agent/human push parity.

## Considered alternatives

- **Single aggregate check:** rejected — hides which stage failed and invites coarse bypasses; distinct names per
  stage (static, integration, deploy-health, preview-smoke) with documented gating.
- **Agent-PR fast lane (fewer gates):** rejected — agent code is less trusted than human code, not more; identical
  gates minimum.
- **Human-merge-only without required checks:** rejected — review without green validation is theater; both required.

## Security / operational / cost implications

Protection rules are repository configuration owned by the owner (setup step in T2 plan, verified live). Bypass lists,
if any, are empty by default and changes audited.

## Failure modes

Misconfigured protection silently not enforcing (mitigation: live negative test — unqualified PR cannot merge —
run after every protection change); check-name collision across systems (mitigation: namespaced check names);
stale required-check blocking all merges (mitigation: check-freshness alert + documented owner override procedure,
audited).

## Verification

LIVE: negative merge test; check-name inventory vs gating doc; agent-PR follows identical gates (attempted
self-approval refused and logged).

## Deferred

Code-owner rules per path; merge-queue adoption.

## Developer-simplicity score

Positive: one legible rule — green checks + human review = mergeable. Distinct check names tell developers exactly
which stage failed without archaeology.
