---
type: project
description: R4 baseline approval and token-scoping constraint
---

# R4 approved as implementation baseline (2026-11-26)

Owner approved `docs/architecture/issue-driven-engineering-workflow-plan.md` Rev R4 as the architectural baseline. No R5 cycle before B2-01A. Architecture is frozen — no cosmetic reopening.

## Standing implementation constraint (owner, at approval)
- Read-only GitHub integration MUST use permission-scoped installation tokens (selected repos, subset of App's granted permissions).
- Write-capable installation tokens are confined to the Studio broker (minted per approved operation only).
- Applies when B2-10 / B2-08 implementation begins.

## Authorization state
- B2-01A is first task but requires its own explicit GO — NOT yet given.
- Live Fly ops, App registration, GitHub writes, and other consequential actions retain separate approvals.
- Planning-only posture continues until an explicit implementation GO arrives.
