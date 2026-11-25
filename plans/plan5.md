# Plan to implement plan4_review.md

## Goal

Revise the implementation plan ([plan4.md](/home/chaschel/Documents/go/miniapp-studio/plans/plan4.md), verbatim copy of `docs/architecture/miniapp-studio-implementation-plan.md`) to resolve the three structural issues and three additional decisions in [plan4_review.md](/home/chaschel/Documents/go/miniapp-studio/plans/plan4_review.md), then resubmit for final approval. Doc-only: no code, deps, migrations, Fly resources, repo init, or edits to governing plans. Framework pin stays provisional per the standing condition.

## Review assessment

I agree with all three structural findings — each names a real contract gap, not a feature request:

1. **Approval resumption (agree).** §4.2 specified park/decide/query but not what executes the business operation after approval. The framework already supports the recommended fix: `executor.Request.ApprovalID` names the gating approval, so a separately submitted invocation can reference the persisted decision with defined idempotency.
2. **Class A tx vs external effects (agree, most important).** A PG transaction cannot span Fly/connector APIs; the Sprite-provisioning crash case is correct. The fix builds on the existing ledger (`PENDING/ATTEMPTING/CONFIRMED/FAILED/UNKNOWN`, `authorizeAttempt`) plus a durable intent/attempt/outcome/reconcile lifecycle for external operations. Connector reads stay low-risk but Sprite provisioning is a write and must use the full lifecycle regardless.
3. **Gateway path vs dogfooding (agree).** A direct agent→gateway call bypasses `Invoke`/audit. The recommended path (agent → narrow capability tool → `runtime.Service` → validated adapter → gateway, result recorded) lets G1 and G8 test one end-to-end path and keeps the agent untrusted.

Additional decisions: agreed on all three. The provider-in-metadata coupling (§12.1) is my own error — provider selection moves to generation/session config. The shared-router URL equivalence was left ambiguous and must be settled. Auth/ownership needs a minimum viable contract (framework `auth` is demo scaffolding per `cmd/miniapp/main.go`; production auth is a later phase, but the MVP contract must be defined).

## Resolutions to specify

1. **Approval→execution contract (§4.2 revised):** approval authorizes a subsequent separately submitted invocation carrying `approvalID` + idempotency key; runtime verifies granted decision + intent hash before executing; duplicate decisions/submissions attach without duplicate actions. Tests: no protected action pre-approval; approval leads to exactly the intended path; duplicates safe. Gates touched: G4, G6, G8.
2. **Durable external-operation lifecycle (new §4.5):** persisted intent before execution; stable idempotency/resource identity; provisioning/failure states; separately committed attempt + outcome records; crash/ambiguous-response recovery; orphan reconciliation sweep. MVP connector ops read-only unless the lifecycle is proven for a write. Gates touched: G1, G6, G9, G10, G11.
3. **Governed gateway path (§6.3 + §8 revised):** agent requests via narrow tool → `Service.Invoke` → policy/declared-op validation → trusted adapter executes gateway call → result recorded on the invocation and returned. G1 and G8 test this same path. Gates touched: G1, G8.
4. **URL semantics (§5 revised):** shared-route option declared non-compliant with the stable persisted Fly-issued URL requirement unless the requirement itself is explicitly amended; spikes compare (a)/(c) as compliant candidates.
5. **Provider independence (§4.1 + §12.1 revised):** provider selection lives in generation/session config only; MiniappSpec carries no provider references; spec byte-identical across provider changes (test).
6. **Auth/ownership contract (new §8.x):** identity model, app ownership, session security, tenant-isolation mapping; minimum viable for exposing the UI, with production auth still roadmap.

## Steps

1. Re-read the cited framework surfaces (`runtime/service.go` approval flow, `executor/executor.go`, `approvals/`, `effects/`) to ground the resumption and lifecycle contracts in existing APIs.
2. Draft revised §4.2, new §4.5, revised §6.3/§8, revised §5 URL ruling, revised §4.1/§12.1 provider rule, new auth/ownership contract.
3. Update affected acceptance tests and gate definitions (G1, G4, G6, G8–G11); keep scope otherwise unchanged (no DBOS, no new features).
4. Apply the revision to the implementation-plan doc, preserving all still-valid sections; resubmit for final approval.

## Validation plan

- Readback audit: each of the 3 structural + 3 additional items traceable to a revised section; every new mechanism cites an existing framework API or is marked new-with-tests; no invented symbols, versions, or green results.
- Contract checks: post-approval execution path executable on paper with no missing step; crash-at-every-point walkthrough for one Sprite provisioning and one connector write; gateway call traceable through `Invoke`/audit in both directions.
- Scope check: diff touches only the cited contracts, tests, and gates — no redesign, no DBOS creep.

## Risks / open questions

- Auth/ownership minimum contract may surface a larger decision (single-user vs accounts) — resolve within the revision, escalate if it changes scope.
- If the approval-flow re-read shows `grantedRun` already resumes Class A execution in-request, prefer documenting the verified mechanism over the two-invocation design — evidence wins.
- Pin stays provisional; §2.1 gate unchanged and still blocking P1+.
