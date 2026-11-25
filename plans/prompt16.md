The next step is **Phase B Part 2 readiness and owner authorization**. The documentation corrections are closed, so another review cycle is unnecessary. The critical-path dependency graph puts the live Builder Gateway verification before the Phase B exit gate and Phase C.

Part 2 is still unauthorized. The next agent task should prepare the exact owner decision and GO checklist—not execute live operations.

# Directive — Prepare Phase B Part 2 GO Readiness

## Objective

Prepare Miniapp Studio for the owner-run Phase B Part 2 live Builder Gateway verification.

Phase A is accepted. Phase B Part 1 is accepted with bounded residuals. The five demo critical-path review findings are closed.

The next dependency in `docs/hackathon-demo-critical-path.md` is Phase B Part 2. However, **Part 2 has not been authorized.**

Your task is a read-only readiness assessment. Do not execute Part 2.

## 1. Governing sources

Read and follow:

- `docs/gateway-phase-b-live-runbook.md` — authoritative owner-run procedure.
- `docs/gateway-phase-b-evidence.md` — accepted implementation evidence and residual register.
- `docs/prompt12-residual-closure-report.md` — accepted closure evidence.
- `docs/hackathon-demo-critical-path.md` — current demo sequence and dependencies.
- `plans/agent-gateway-implementation-plan.md` — governing Gateway architecture.

Do not reopen accepted architecture or repeat broad adversarial reviews.

## 2. Verify Part 2 readiness from existing evidence

Inspect the repository and documents to establish:

- Whether the working tree and branch are in an understandable state.
- Whether the pinned dependencies and expected versions are documented.
- Whether the pre-live gates in runbook §3 have current, attributable evidence.
- Whether the runbook's commands, prerequisites, and expected observations are internally consistent.
- Whether N2, N5, and N8 restrictions are respected at their relevant operations.
- Whether the cleanup and evidence-capture requirements are executable by the owner.
- Whether any documented blocker or ambiguity remains.

Clearly distinguish:

- Verified from current repository evidence.
- Previously tested but not re-executed.
- Live-unverified.
- Owner decision required.
- Blocked or unresolved.

Do not infer that a previously green test is currently green without fresh evidence.

## 3. Produce the owner decision packet

Return a concise table containing every decision in runbook §1:

| Decision | Required owner input                                                                      |
| -------- | ----------------------------------------------------------------------------------------- |
| D1       | Fly organization for Builder Sprites                                                      |
| D2       | OpenCode model provider and owner-provisioned authentication method                       |
| D3       | Studio callback reachability: owner machine plus tunnel, or Fly-hosted Studio             |
| D4       | Builder retention: destroy after proof or retain a named Builder                          |
| D5       | System packages: preinstalled image or live installation, subject to discovery            |
| D6       | Pin drift: confirm existing pins or explicitly authorize a re-pin after evidence of drift |

For each decision, explain:

- Why it is needed.
- Whether it blocks the start of Part 2 or a later step.
- What exact information the owner must supply.
- Any security or operational consequence of the choice.

Identify D3 as the largest technical dependency because the Builder Sprite must reach Studio's callback endpoint.

Do not request or expose credentials, tokens, API keys, or secrets in chat.

## 4. GO prerequisites

Restate the exact conditions that must be satisfied before Part 2 begins:

- F-R1/N8 and F-R2/N5 runbook corrections remain present.
- F-R3 count correction remains present.
- D1, D2, and D3 are resolved.
- D6 pin policy is confirmed and pins are re-verified at execution time.
- Runbook §3 pre-live gates are green.
- Owner accepts the N2 live-upgrade prohibition and N8 re-provisioning restriction.
- Owner explicitly authorizes the live run.

D4 and D5 must be resolved at their relevant runbook steps.

Make clear that readiness is not authorization.

## 5. Strict execution boundary

This task is read-only.

Forbidden:

- Creating, destroying, provisioning, or modifying Sprites.
- Running `flyctl` mutations or Fly API mutations.
- Running `sprite` commands against live infrastructure.
- Running Gateway provision, upgrade, assign, continue, or other live commands.
- Starting OAuth or installing OpenCode on a live Builder.
- Requesting, printing, copying, or exposing production credentials.
- Implementing Phase C, D, or E.
- Changing the Miniapp framework, Gateway architecture, F1, G-01, or P5.
- Editing files, staging, committing, or pushing.

Do not execute runbook procedures as a test. Inspect their documented requirements only.

## 6. Output

Return:

1. **Readiness verdict:** READY FOR OWNER DECISIONS, BLOCKED, or NOT READY.
2. **Evidence table:** each prerequisite, its current status, and its source.
3. **D1–D6 owner decision packet:** exact inputs needed.
4. **Remaining blockers:** only concrete unresolved issues.
5. **GO checklist:** observable conditions that must be true before live execution.
6. **Recommended next action:** the precise owner response needed to proceed.

Do not create another roadmap or broad review document. Do not modify the existing runbook.

Stop after producing the readiness report. Wait for the owner's explicit, separate Part 2 GO authorization.

**Success means the owner can make an informed authorization decision without ambiguity, not that any live operation has been performed.**

Once the agent returns the packet, the next decision is yours: settle the required owner inputs and explicitly authorize Part 2. After its live evidence is accepted, proceed to the Phase B exit gate—M5-SSE and `STUDIO_GATEWAY_BINARY` serve-time wiring—then Phase C.