# ADR-014: Background agent execution policy

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** Event-driven agent runs on isolated Sprites: authorized label trigger → verified webhook → eligibility
checks → clean Sprite → headless agent → canonical validation → branch → PR gated on validation (stop or marked draft
on failure) → record → retain/destroy per policy. Pluggable agent/model adapters; per-run budgets; restricted egress;
issue/repo content always untrusted.
**Basis:** prompt28.2 §6 + §7; verified Sprite/GitHub mechanics (V4, V6–V11).

## Context

Agents amplify both throughput and risk: they execute untrusted instructions against real credentials and real APIs.
The policy binds what can trigger a run, what a run can touch, and what a run may claim.

## Requirements

R223–R229: trigger authorization, isolation, credential treatment, workflow, main-branch protection, failure honesty,
supervision, budgets, egress.

## Considered alternatives

- **Interactive-terminal agents only:** rejected — abandons the background-automation objective; kept as a complementary
  mode, not the policy.
- **Broad agent permissions for convenience:** rejected — a compromised run must be containable to one repo, one hour,
  one Sprite (V4 + V10 + ADR-008).
- **Auto-ready PRs on agent completion:** rejected — PRs open as ready only when validation passes; failures stop the
  run or open explicitly marked drafts. No self-approval path exists (ADR-017).

## Security / operational / cost implications

Trigger authorization is the front door (label + user + repo + eligibility). Egress policy per run (V10). Model keys
controller-held with per-run budgets; runaway spend bounded before provisioning. Agent-accessible diagnostics are
read-only (ADR-009/011).

## Failure modes

Prompt injection via issue/repo content (mitigation: untrusted-content doctrine + least privilege + egress policy);
runaway long run (mitigation: timeouts + heartbeats + budgets + kill path); validation-gaming (tests pass, feature
broken — mitigation: preview + human review as the real gates, R230).

## Verification

LIVE: full §13 agent matrix — authorized trigger, untrusted-content containment, no-main-push, failure honesty,
E2E PR creation, cancel/cleanup, budget/egress enforcement.

## Deferred

Multi-issue/orchestrated runs; review-comment continuation UX (needs V15 proof).

## Developer-simplicity score

High positive: labeling an issue yields a reviewed PR candidate. Developers supervise outcomes (preview + diff +
checks) instead of performing mechanical implementation.
