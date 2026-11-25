My architectural decision: **use OpenCode's native credential store for OpenRouter authentication. Do not build a new Gateway credential-injection API or change the Sprites service API for this demo.**

I investigated the exact OpenCode version pinned by Miniapp Studio: **v1.19.6**. There is already a supported mechanism.

### **Approved credential ceremony**

OpenCode v1.19.6 implements:

- `opencode auth login --provider openrouter`
- A hidden API-key password prompt (`Prompt.password`).
- Native credential persistence in `Global.Path.data/auth.json`.
- File creation with permissions `0600`.

Sources: [OpenCode v1.19.6 provider login implementation](https://github.com/anomalyco/opencode/blob/v1.19.6/packages/opencode/src/cli/cmd/providers.ts?utm_source=chatgpt.com), [credential storage implementation](https://github.com/anomalyco/opencode/blob/v1.19.6/packages/opencode/src/auth/index.ts?utm_source=chatgpt.com), [data-directory implementation](https://github.com/anomalyco/opencode/blob/v1.19.6/packages/core/src/global.ts?utm_source=chatgpt.com).

This avoids putting the key in command arguments, application configuration, or the Gateway's environment.

**The ceremony I approve:**

1. Provision the Builder under the separately authorized Part 2 procedure.
2. Open its interactive console using `sprite console`.
3. Become the `studio-agent` identity.
4. Verify the effective `HOME` and `XDG_DATA_HOME` match the directories used by the supervised OpenCode process. This is critical because the adapter inherits the Gateway environment.
5. Run `opencode auth login --provider openrouter` interactively and enter the key at its hidden prompt.
6. Verify `opencode auth list` identifies OpenRouter as configured, and verify the credential file's owner and `0600` permissions without displaying its contents.
7. Complete §10's actual supervised no-op assignment. Successful model inference and the expected callbacks are the end-to-end proof.

Fly documents `sprite console` as an interactive TTY session.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.sprites.dev\&sz=32)

Sprites



### **One security qualification**

The OpenCode process and its tools run under the same `studio-agent` identity. Consequently, the agent's execution environment can potentially access its own provider credential. Use a dedicated OpenRouter key with a strict spending limit, and revoke it after the live proof if it is no longer required.

Do not use the personal key if it has broader permissions or an unrestricted budget.

### **Next instruction to the implementation agent**

# Directive — Finalize OpenRouter Owner Authentication Ceremony

## Decision

The approved D2 credential-delivery method is OpenCode v1.19.6's native provider credential store.

Use the existing command:

`opencode auth login --provider openrouter`

The owner enters the key interactively through the hidden password prompt. Do not implement a new Gateway credential-injection API or modify the Sprites service API.

## Required documentation

Update `docs/gateway-phase-b-live-runbook.md` only.

Document the owner ceremony at the appropriate point after Builder provisioning and identity verification, but before §10's no-op agent assignment.

Include:

1. Interactive Sprite console access.
2. Execution as `studio-agent`, not root or `studio-gw`.
3. Verification that the interactive login and supervised OpenCode process use the same effective `HOME`, `XDG_DATA_HOME`, and `Global.Path.data`.
4. The exact native OpenCode login command.
5. Verification through `opencode auth list` without displaying credentials.
6. Verification of file owner, parent-directory permissions, and credential-file mode `0600`.
7. The distinction between OpenRouter model authentication and Sprites MCP OAuth.
8. The §10 supervised no-op assignment as the authoritative proof of successful inference.

## Critical path check

The Gateway child inherits its parent environment. Do not assume a login shell's XDG paths match the supervised process.

Trace the actual environment setup and effective identity. If the OpenCode child would read a different credential directory from the one populated by the owner ceremony, STOP and report the root cause. Do not enter the key into the wrong home directory or create a workaround.

## Security

- Never put the key in argv, source code, Git, application artifacts, or logs.
- Never print the credential file.
- Do not use `opencode auth export`.
- Do not copy credentials into the Builder workspace.
- Do not weaken the OS identity boundary.
- Use the owner's dedicated, spending-limited OpenRouter key.

## Execution boundary

This is documentation and read-only implementation verification only.

Do not provision a Sprite, open an interactive console, enter credentials, run OpenCode authentication, or execute §10.

Do not stage, commit, or push.

## Final report

Report the exact ceremony, the verified credential path and identity assumptions, remaining runtime conditions, and confirmation that no live operation occurred.

Stop and await separate explicit Part 2 authorization.

**Architectural verdict:** no foundational reason to change the Gateway design. The missing credential injection mechanism is addressed by an existing, version-pinned OpenCode capability. We only need to verify the identity and data-directory alignment before using it.

Part 2 remains unauthorized. We are now one clearly defined owner authentication ceremony away from the real Builder proof.



Add this to the existing agent directive as a new D2 model-routing requirement.

**My decision:** OpenCode is the primary provider. Both models are approved, with DeepSeek as the default and MiMo as the approved alternative. OpenRouter is the only permitted fallback, and it must use the corresponding same model—not a different model.

One important distinction: the `-free` offerings are OpenCode Console models. OpenRouter exposes the corresponding underlying models under different IDs, and those routes are paid.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://opencode.ai\&sz=32)

OpenCode+3



# Addendum — OpenCode Primary Model and OpenRouter Same-Model Fallback

## 1. Model-routing decision

OpenCode (`opencode`) is the PRIMARY provider.

Configure both approved OpenCode models:

| Priority             | Provider   | Model ID                   | Role              |
| -------------------- | ---------- | -------------------------- | ----------------- |
| Default              | `opencode` | `deepseek-v4.1-flash-free` | Primary default   |
| Approved alternative | `opencode` | `mimo-v2.6-flash-free`     | Alternative model |

The default model selection is:

`opencode/deepseek-v4.1-flash-free`

The alternative is:

`opencode/mimo-v2.6-flash-free`

Do not silently switch from one model family to the other.

## 2. OpenRouter fallback mapping

OpenRouter is the ONLY permitted fallback provider.

Use this exact model mapping:

| Primary OpenCode model              | OpenRouter fallback model                 |
| ----------------------------------- | ----------------------------------------- |
| `opencode/deepseek-v4.1-flash-free` | `openrouter/deepseek/deepseek-v4.1-flash` |
| `opencode/mimo-v2.6-flash-free`     | `openrouter/xiaomi/mimo-v2.6-flash`       |

These are corresponding underlying model families with provider-specific catalog IDs. They are not identical provider offerings. OpenRouter fallback is not free merely because the OpenCode primary model is free.

Verify the IDs against the actual OpenCode v1.19.6 model catalog and current OpenRouter catalog before provisioning. Never guess or silently substitute another model/version.

## 3. Credential requirements

Two distinct provider credentials are required:

**Primary — OpenCode**

- The owner must authenticate the `opencode` provider using its own OpenCode Console credential or supported account authentication.
- Use OpenCode's native credential mechanism.
- The OpenRouter key does NOT authenticate the OpenCode provider.

**Fallback — OpenRouter**

- Use the owner's already-provisioned OpenRouter API key.
- Store it through OpenCode's native provider credential store.
- Do not request the key in chat or copy it into source code, artifacts, command-line arguments, or logs.

The owner authentication ceremony must configure both providers under the `studio-agent` identity and the exact data directory used by the supervised OpenCode process.

Verify credential presence without printing credential values. Both credential files and parent directories must have appropriately restricted ownership and permissions.

## 4. Fallback behavior

Fallback must preserve the SAME underlying model family.

Examples:

- DeepSeek primary failure → OpenRouter DeepSeek V4.1 Flash.
- MiMo primary failure → OpenRouter MiMo V2.6 Flash.

Never fall back to another model, another provider, or a generic "best available" model.

Before implementing or claiming automatic fallback, inspect the pinned OpenCode v1.19.6 CLI and existing Gateway adapter/supervisor execution path.

OpenCode's `--model` selection chooses one provider/model for a run. Do not assume native cross-provider automatic failover exists.

If the current Gateway does not implement typed provider failover, document that as a concrete implementation gap. Do not represent a manual retry or model-selection capability as automatic failover.

Any eventual automatic fallback must:

- Trigger only on a positively classified provider/inference failure, such as upstream unavailability, transient transport failure, timeout, rate limiting, or provider-side 5xx.
- Not trigger on coding errors, tool failures, permission denials, policy refusals, malformed requests, user cancellation, or application/test failures.
- Never retry indefinitely.
- Preserve the build, workdir, session lineage, and turn-fencing invariants.
- Avoid replaying consequential tool actions or duplicating work.
- Record the primary failure, fallback provider/model, attempt identity, and final outcome without recording secrets.
- Surface failure honestly if the fallback also fails.

The fallback mechanism must not bypass Gateway deduplication, fencing, durable state, or the existing supervised execution boundary.

## 5. Cost and owner controls

OpenRouter fallback is metered.

Before enabling automatic fallback, require:

- Owner-confirmed OpenRouter spending limit or prepaid budget.
- No automatic recharge unless separately authorized.
- A bounded fallback attempt count.
- Usage evidence attributable to the build and model attempt.

Do not allow a free OpenCode model failure to silently create unlimited paid OpenRouter usage.

## 6. Runbook and verification

Update `docs/gateway-phase-b-live-runbook.md` to document:

- The primary provider and default model.
- The approved alternative.
- The exact OpenRouter fallback mapping.
- The separate credential ceremonies.
- The model catalog verification procedure.
- The cost and retry limits.
- The distinction between configured fallback and proven automatic fallback.

At §10, the no-op supervised assignment must explicitly select the primary model and prove actual inference. A separate controlled test must prove fallback behavior if automatic fallback is implemented and accepted.

Record the exact provider ID, model ID, OpenCode version, and attempt outcome.

## 7. Execution boundary

This addendum does NOT authorize Part 2 live operations.

For the current task:

- Inspect implementation and pinned OpenCode v1.19.6 behavior.
- Update documentation only.
- Do not provision Sprites or run OpenCode authentication.
- Do not enter or expose credentials.
- Do not implement a new fallback engine as an incidental runbook change.
- Do not stage, commit, or push.

If automatic fallback requires code not present in the accepted Gateway implementation, report the precise gap and recommend a separately scoped implementation decision before live GO.

Stop after documentation and verification.

**One important approval condition:** the agent must not claim automatic fallback merely because both credentials and models are configured. Model selection is already supported through `--model`; automatic provider failover is a separate behavior that must be demonstrated.

This keeps the accepted Gateway architecture intact while giving Miniapp Studio a deterministic primary model and a controlled, same-model fallback.