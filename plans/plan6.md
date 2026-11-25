## Goal

Implement [plan5.1_review.md](/home/chaschel/Documents/go/miniapp-studio/plans/plan5.1_review.md): resolve the external-effect execution boundary, correct the attempt-start sequence and sweeper trigger, fix the two acceptance criteria, and carry multi-tenant readiness into the governing plan as a default requirement. Doc-only revision of the consolidated plan ([plan5.1.md](/home/chaschel/Documents/go/miniapp-studio/plans/plan5.1.md) stays the frozen snapshot of the pre-revision state).

## Success Criteria

- §4.2/§4.5/§6.3 made mutually consistent via an explicit framework-controlled external-effect boundary with all guarantees retained and no background engine.
- Attempt sequence corrected to intent → committed attempt-start → execute → outcome → UNKNOWN+reconcile, with a no-background-jobs recovery trigger.
- Provider test redefined as spec/hash immutability across provider change; auth coverage (expiry, revocation, unauthenticated denial, ownership, cross-tenant denial) added to G8.
- Seven multi-tenant requirements reflected in architecture, data model, auth contract, and gates; MVP keeps one operator per workspace with no admin UI.
- Scope, cut line, pin, and `plan4_z` conditions otherwise untouched.

## Approach

Evidence already points at the resolution: `effects.Service.Perform` (`effects/effects.go:182`) implements exactly the needed shape — `claim` commits PENDING with a framework-minted attempt ID, `authorizeAttempt` commits ATTEMPTING in a separate authorization transaction, `Channel.Send` executes outside any transaction, `transition` commits the outcome. Step 1 verifies each function's transaction boundaries and lock-ordering statements in code, then specifies the §6.3 path as Class A handler → `Perform` (claim/authorize txs → send → outcome tx) synchronously in-request, with the gateway adapter as a `Channel`. Recovery without background jobs follows the existing lease + adopt/stale→UNKNOWN + `ListUnknown`/`SettleEffect` operator path. Tenant grounding exists too (`tenant:local` BusinessOpID enforcement, server-stamped `Operation.TenantID`, RLS tenant context) and extends to all Studio records.

## Steps

1. Verify `claim`/`transition`/`authorizeAttempt` transaction boundaries, `adopt`/`adoptStale`/`reconcile` semantics, and lock-ordering evidence in code.
2. Specify the external-effect boundary (§4.2/§4.5/§6.3 reconciliation) and the corrected attempt sequence with the no-background recovery trigger.
3. Fix provider test (§12.1), add auth coverage to G8, and carry the seven multi-tenant requirements into §§3/4/5/8/10 as applicable.
4. Apply to the consolidated governing doc; readback audit; resubmit for approval.

## Validation Plan

- Consistency walkthrough: one connector call traced through §4.2 → §4.5 → §6.3 with every guarantee attributed to a verified mechanism; crash-at-every-boundary check (pre-send, post-send-pre-commit, post-commit).
- Traceability: review items 1, 2, 3a, 3b, and multi-tenancy 1–7 each map to a revised section, test, or gate; no invented APIs; proposed-vs-executed tests kept distinct.
- Scope check: no new features, no DBOS, no P1/P3, no commits.

## Risks / Open Questions

- If step 1 shows `transition` shares the Class A attempt tx anywhere, the boundary needs a §14 framework correction proposal instead of a pure doc fix — will escalate rather than assume.
- Auth/ownership single-operator scope is unchanged by multi-tenancy (isolation without admin UI); any pressure beyond that escalates as scope.
