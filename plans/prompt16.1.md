I reviewed `prompt16_report.md`. **I agree with the BLOCKED verdict.** The agent has identified two legitimate runbook inconsistencies, and there is one small documentation improvement worth including before asking you to authorize Part 2.

The findings are:

1. **B1 — Stale repository-state expectation:** Runbook §2 assumes an all-untracked repository with no commits. That no longer describes the actual checkout.
2. **B2 — Unsafe N2 instruction:** Runbook §12 suggests retrying a live Gateway upgrade with `--force`, contradicting the explicit N2 prohibition in §6.5.
3. **§9 clarity:** The instruction to start Studio using the standard deployment procedure should identify the actual serve command and required environment from §5.

These are documentation blockers, not reasons to reopen the Gateway architecture.&#x20;

prompt16_report.md



**My recommendation:** authorize a bounded documentation-only correction now. Do not authorize Part 2 yet. One important refinement: don't hardcode `de85523` as the permanent expected commit. The durable correction is to record the current branch, HEAD SHA, and working-tree status, then stop if the checkout is not owner-recognized.

Give the agent this prompt:

# Directive — Close Phase B Part 2 Runbook Readiness Findings

## Objective

Correct the two documented blockers identified in `plans/prompt16_report.md` and resolve the Studio startup ambiguity in `docs/gateway-phase-b-live-runbook.md`.

This is a documentation-only task.

The objective is to make the owner-run procedure accurate, internally consistent, and resistant to future repository-state drift.

## Authorized scope

Only modify:

`docs/gateway-phase-b-live-runbook.md`

Do not modify code, tests, architecture plans, evidence reports, or other documentation.

## Required corrections

### B1 — Correct runbook §2 repository-state handling

The current instruction assumes the repository has no commits and expects an all-untracked tree. This is stale.

Replace that assumption with a durable procedure that:

- Records the current branch.
- Records the current HEAD commit SHA.
- Records `git status --porcelain` verbatim.
- Does not assume zero commits or an all-untracked checkout.
- Requires the operator to stop if the branch, revision, or working-tree changes are unexpected or cannot be reconciled with the owner-approved checkout.
- Does not permanently hardcode the current commit SHA as an expected future value.

Keep the purpose of this gate: establish exactly which source tree is being used before live operations.

### B2 — Correct the N2 contradiction in §12

The §12 upgrade failure row must not suggest retrying a live upgrade with `--force`.

Replace it with an unambiguous instruction:

**STOP and report. Live `gateway upgrade` remains prohibited until N2 is fixed and tested, as stated in §6.5. Do not use `--force` to bypass the prohibition.**

Verify that no other runbook instruction contradicts the §6.5 prohibition.

### §9 — Clarify Studio startup

Replace the vague phrase "start it per the standard deploy" with the actual Studio serve command and required environment setup, grounded in the existing §5 configuration and repository implementation.

Do not invent a command, environment variable, or deployment mechanism.

Make explicit that Studio must be reachable from the Builder Sprite using the previously selected D3 callback URL.

## Verification

1. Re-read §2, §5, §6.5, §9, and §12 in full context.
2. Confirm §2 no longer assumes zero commits or all-untracked state.
3. Confirm §12 contains no permission to use `--force` for a live upgrade while N2 remains open.
4. Search the entire runbook for other contradictory upgrade instructions.
5. Confirm §9's startup command and environment agree with actual repository implementation.
6. Confirm only `docs/gateway-phase-b-live-runbook.md` changed.
7. Confirm no code or tests changed.
8. Do not execute Fly, Sprite, Gateway provision, upgrade, assign, continue, OAuth, or other live operations.
9. Do not stage, commit, or push.

## Final report

Report:

- Exact changes made.
- Verification performed and results.
- Final runbook diff scope.
- Whether B1, B2, and §9 ambiguity are closed.
- Explicit confirmation that Part 2 was not executed or authorized.

Stop after verification. Do not proceed to Part 2.

After that correction, the next owner decision packet will still need D1, D2, and D3, followed by D6 pin confirmation at execution time. **No Part 2 GO has been granted.** The report itself correctly identifies these as outstanding owner decisions.&#x20;

prompt16_report.md
