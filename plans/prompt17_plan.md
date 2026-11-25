APPROVED — GO

The Prompt17 plan is structurally sound. **No foundational or architectural blocker prevents this documentation-only pass.** The placement of §8b, the HOME/XDG verification gate, explicit STOP behavior, model mapping, and honest fallback-gap treatment are all appropriate.

I checked the key assumptions and have three refinements to carry into execution.

### **1. Correction: don't require an unnecessary OpenCode credential**

I inspected the pinned OpenCode v1.19.6 implementation:

- `provider.ts` retains zero-input-cost OpenCode models even when no OpenCode credential exists.
- In that case it configures the provider with the `public` placeholder.
- Paid OpenCode models are filtered when no credential is configured.

Source: [OpenCode v1.19.6 provider implementation](https://github.com/anomalyco/opencode/blob/v1.19.6/packages/opencode/src/provider/provider.ts?utm_source=chatgpt.com).

Therefore, the directive's assumption that two credentials are always required is too strong.

**My decision:** use the OpenCode free tier without a separate OpenCode credential, provided the actual catalog confirms both selected models have zero input cost and the supervised inference test succeeds. The OpenRouter API key remains necessary for the fallback.

The pinned version supports this exact model-catalog inspection command:

```
opencode models opencode --verbose --refresh
opencode models openrouter --verbose --refresh
```

Source: [OpenCode v1.19.6 models command](https://github.com/anomalyco/opencode/blob/v1.19.6/packages/opencode/src/cli/cmd/models.ts?utm_source=chatgpt.com).

### **2. Model IDs are currently supported**

The current catalogs support the requested mapping:

| Primary OpenCode model              | OpenRouter fallback                       |
| ----------------------------------- | ----------------------------------------- |
| `opencode/deepseek-v4.1-flash-free` | `openrouter/deepseek/deepseek-v4.1-flash` |
| `opencode/mimo-v2.6-flash-free`     | `openrouter/xiaomi/mimo-v2.6-flash`       |

The OpenCode IDs appear in its model catalog and documentation.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://models.opencode.ai\&sz=32)

Models.dev+2



Both OpenRouter alternatives are metered, not free. Current published rates are $0.051/$0.567 per million input/output tokens for DeepSeek and $0.14/$0.28 for MiMo. These prices must be rechecked at execution.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://openrouter.ai\&sz=32)

OpenRouter+1



Keep the live catalog verification and STOP-on-mismatch requirement.

### **3. One explicit security warning**

The native credential file's `0600` permissions protect it from other operating-system users, but **not from arbitrary commands running under the same&#x20;**`studio-agent`**&#x20;identity**.

The agent's shell tools can potentially read that file. Document this honestly. The OpenRouter key must be dedicated, spending-limited, have automatic recharge disabled, and be revoked after the proof unless continued use is explicitly authorized.

### **Sprite console uncertainty is resolved**

The official Sprites documentation confirms:

```
sprite console -s '<SPRITE>'
```

It provides an interactive terminal. There is no need to leave this command as an unsupported guess.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Linux computers for agents+1



## **Final instruction to the agent**

Proceed with Prompt17, incorporating these refinements:

1. Make OpenCode authentication optional for the two free models, conditional on zero-cost catalog metadata and successful supervised inference. OpenRouter authentication remains required for fallback.
2. Document the same-UID credential exposure risk explicitly.
3. Use the verified Sprite console command and pinned OpenCode model-catalog command.
4. Preserve all existing STOP conditions and the distinction between model selection and automatic failover.
5. Keep the task strictly documentation-only.

**Approval is for the Prompt17 runbook update only.** No code, tests, credential entry, Sprite provisioning, OAuth, live inference, commits, or pushes are authorized.

The architecture is settled. The next milestone is the real Builder proof, once the documentation is complete and you separately authorize Part 2.