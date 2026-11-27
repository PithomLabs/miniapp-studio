# Card 002: B2 live proofs (gateway runbook execution)

- Status: `draft`
- Roadmap: A (B2 — blocked on Card 001 green)
- Owner decision required: GO per the runbook's own authorization structure (per-operation GOs for provision,
  OAuth, provider auth; this card covers execution + evidence, each live step already gated inside the runbook).

## Outcome

The Phase B live runbook (`docs/gateway-phase-b-live-runbook.md`) is executed end to end: builder provisioned,
identity/inbox proofs recorded, OAuth + provider ceremonies completed, no-op supervised turn demonstrated,
sleep/wake/resume observed, cleanup verified — or each unproven item explicitly carried with its blocker.

## Acceptance criteria

- [ ] Runbook §0–§5 pre-live gates pass (pins, tree state, binary, environment recorded).
- [ ] Builder provision + privilege discovery recorded (§6–§7, honest unknowns preserved).
- [ ] Identity/inbox proofs recorded (§7b); OAuth ladder P-E1→P-E3 completed (§8); provider ceremony (§8b).
- [ ] No-op turn + sleep/wake/resume observations recorded (§10–§11) or explicitly deferred with reason.
- [ ] Cleanup executed (§13); receipts, timings, and file references attached; no orphaned resources.

## Dependencies

- Card 001 green (no live proofs on a red suite).
- Owner identity: Fly org holds, Sprite login, Neon branch, OpenRouter key ceremony (all owner-terminal).
- Pins + tree state re-verified at execution time (runbook §2 — never floated).

## Decision brief

1. Execution location: owner terminal for all live steps; agent prepares, reviews output, records evidence.
2. Application target: Builder Sprite + gateway binary + Phase B integration surface.
3. Database target: Neon dev branch (run-pattern DBs for any DB touch); no migration changes expected.
4. Infrastructure actions: as scripted in the runbook only (provision, OAuth grants, provider keys,
   retirement) — each already individually gated there.
5. Authorization class: live infrastructure operations per runbook gates; no commits/pushes; no scope beyond
   the runbook's named steps.
6. Verification and evidence: runbook section outputs + receipts + timings in a gitignored log; agent
   verifies each proof against its stated expectation before marking it.
7. Recovery and rollback: runbook §13 cleanup always runs; failure interpretation per §12; partial state is
   reported, never force-cleaned.

## Report (§7)

(Pending execution.)
