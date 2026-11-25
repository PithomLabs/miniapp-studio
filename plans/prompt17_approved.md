The ambiguity is that approval of an execution plan and authorization to execute it are two different things. The instruction below explicitly resolves that distinction.

**My recommendation is to issue an explicit GO for Phase B Part 2 live execution.** The architecture and runbook have been approved. The agent must now execute the existing procedure, not produce another plan.

# EXPLICIT OWNER GO — Execute Phase B Part 2 Live Builder Gateway Verification

**AUTHORIZATION STATUS: GO — LIVE EXECUTION AUTHORIZED**

This is the owner's separate, explicit authorization to EXECUTE Phase B Part 2, not merely prepare, review, or plan it.

Do not interpret this instruction as documentation-only authorization.

## 1. Objective

Execute the owner-approved Phase B Part 2 live Builder installation and recovery verification using:

`docs/gateway-phase-b-live-runbook.md`

Phase A is accepted.\
Phase B Part 1 is accepted with bounded residuals.\
The Part 2 runbook and its corrections are approved.

Your task is to perform the live verification and collect attributable evidence.

Do not create another implementation plan or request another GO confirmation.

## 2. Binding owner decisions

Use these decisions:

| Decision                  | Binding value                                                                         |
| ------------------------- | ------------------------------------------------------------------------------------- |
| D1 Fly organization       | `isagani-mendoza`                                                                     |
| D2 primary model provider | OpenCode                                                                              |
| D2 default model          | `opencode/deepseek-v4.1-flash-free`                                                   |
| D2 alternative model      | `opencode/mimo-v2.6-flash-free`                                                       |
| D2 fallback provider      | OpenRouter                                                                            |
| D2 fallback models        | Same underlying model family, using the approved provider-specific IDs in the runbook |
| D3 Studio deployment      | Fly-hosted Studio through Fly Proxy                                                   |
| D4 Builder retention      | Destroy to zero after successful proof                                                |
| D5 packages               | Discover at execution; record any required installation                               |
| D6 pins                   | Hold existing pins; no re-pinning without separate authorization                      |

Do not substitute organizations, providers, models, versions, or authentication mechanisms.

## 3. Execute the runbook in order

Follow the approved runbook sections and their observation points.

- §2: Verify branch, HEAD, working tree, and all pinned dependencies.
- §3: Execute pre-live gates and stop on any failure.
- §4–§5: Build the pinned Gateway binary and prepare owner-held environment and secrets.
- §6: Provision the Builder Sprite in `isagani-mendoza`.
- §7–§7b: Verify packages, service privilege, process identity, key isolation, and inbox permissions.
- §8: Complete the owner-driven Sprites MCP OAuth ceremony.
- §8b: Complete the approved OpenRouter credential ceremony and model verification.
- §9: Verify Studio readiness and callback reachability.
- §10: Execute the supervised no-op assignment using the explicit default model.
- §11: Perform sleep/wake, PID-absence, and recovery observations.
- §13: Destroy the Builder and verify cleanup, as required by D4.
- §14: Produce the complete evidence report.

Record actual outputs, exit codes, timestamps, process identities, callback events, and observed outcomes.

Never convert an expected observation into a claimed result.

## 4. Critical safety restrictions

These restrictions remain absolute despite this GO.

**N2 — Gateway upgrade prohibited**

- Do not run `gateway upgrade`.
- Do not use `--force` to bypass N2.
- If binary drift is detected, STOP and report.

**N8 — Re-provisioning permission window**

Before every re-provision beside running agents, perform the required staging-mode and sibling-process checks. Drain first when either condition cannot be established.

**N5 — Cold recovery**

Record agent PIDs before sleep. After a cold recovery, explicitly prove that every pre-restart PID is absent. Never rely on `agent.failed` alone.

**Credential protection**

- Never print or transmit API keys, OAuth tokens, HMAC secrets, or credential-file contents.
- Never put secrets in command arguments, source files, Git, artifacts, or transcripts.
- OpenRouter credentials must be dedicated and spending-limited, with automatic recharge disabled.
- Remember that OS mode 0600 protects credentials from other users, not from arbitrary processes running as `studio-agent`.
- Do not weaken the Gateway's identity separation.

## 5. OpenRouter authentication and model execution

Use OpenCode v1.19.6's native credential store.

The owner enters the OpenRouter API key interactively through the hidden password prompt:

`opencode auth login --provider openrouter`

Verify credential presence without exposing its value.

OpenCode is the primary provider. The DeepSeek free model is the default. MiMo is the approved alternative.

The OpenCode free model does not require a separate OpenCode credential if its pinned catalog metadata confirms zero cost and the real supervised inference succeeds.

Before selecting a model, verify the exact IDs in the provider catalogs. Any mismatch means STOP; do not silently substitute.

Use `opencode/deepseek-v4.1-flash-free` explicitly for §10.

**Important:** automatic cross-provider failover is not implemented in the accepted Gateway. Configured OpenRouter credentials do not prove automatic fallback. Do not claim or simulate automatic fallback. A fallback execution test is conditional on a separately implemented and accepted failover capability.

## 6. D3 deployment prerequisite

The selected topology is Fly-hosted Studio through Fly Proxy.

Require:

- Studio listening on the approved `0.0.0.0:<internal_port>` address.
- Fly Proxy internal port matching Studio's listener.
- `/healthz` 200 for liveness.
- `/readyz` 200 for database and schema readiness.
- Builder-side unsigned callback POST returning 401.
- Actual signed callback delivery demonstrated by `agent.started` during §10.

The approved repository does not contain an established Studio-on-Fly deployment procedure. Do not invent one or improvise infrastructure changes.

If Studio is not already deployed and reachable, STOP and report the exact deployment prerequisite. Do not substitute an owner-machine tunnel or request-catcher without authorization.

## 7. STOP conditions

Stop immediately on:

- Any failed pre-live gate.
- Unexpected repository state or pin drift.
- Insufficient privileges or credential-isolation failure.
- HOME/XDG/data-directory mismatch between the authentication ceremony and supervised OpenCode process.
- Unverified model IDs.
- Studio readiness or callback reachability failure.
- N2 or N8 safety violation.
- Ambiguous process ownership or unsuccessful orphan cleanup.
- Any operation requiring a new owner decision or deviation from the approved runbook.

Do not bypass a STOP condition to finish the demonstration.

Report the root cause and evidence.

## 8. Scope boundaries

This GO authorizes Phase B Part 2 live verification only.

It does NOT authorize:

- Phase C implementation.
- Phase D artifact pipeline implementation.
- Phase E Runtime deployment implementation.
- New Gateway APIs or credential-injection architecture.
- Automatic model failover implementation.
- Changes to Miniapp, F1, G-01, P5, or accepted architecture.
- Unapproved production business effects.
- Commits or pushes.

No code modifications are authorized as incidental fixes during this run.

## 9. Completion report

Return:

1. A section-by-section runbook execution matrix.
2. Exact evidence for each passed, failed, blocked, or skipped check.
3. Studio readiness and signed callback proof.
4. Builder provisioning and identity-isolation evidence.
5. OpenRouter credential and actual DeepSeek inference proof, without secrets.
6. Sleep/wake and recovery evidence, including PID absence where applicable.
7. Cleanup or explicit reason cleanup could not be completed.
8. Any remaining N2–N10 residual implications.
9. A final Phase B Part 2 verdict: ACCEPTED, ACCEPTED WITH BOUNDED RESIDUALS, or NOT ACCEPTED.

Do not declare Part 2 accepted solely because provisioning or the no-op assignment succeeded.

Stop after the report. Wait for a separate owner decision before Phase C.

**FINAL INSTRUCTION: GO. Execute Phase B Part 2 now, strictly within these boundaries. Do not produce another plan and do not ask for another authorization confirmation.**

This makes the authorization unmistakable: **live Part 2 execution is authorized**, but Phase C, D, and E remain outside this GO.