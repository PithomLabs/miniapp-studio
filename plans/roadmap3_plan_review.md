I’ve reviewed both uploaded documents. My decision is to use a **separate OpenCode amendment document first**, rather than directly rewriting the approved Agent Gateway plan.

This preserves the gateway plan as an independently reviewable architectural baseline. The amendment can be reviewed, accepted, and then merged through an explicit section-by-section change list.

There is one important issue I want the amendment to resolve before we approve that merge: the official Sprites OpenCode plugin describes OpenCode running outside the Sprite and using Fly MCP tools to work inside a remote Sprite. That is not automatically the same topology as our proposed Agent Gateway supervising an OpenCode process inside the Builder Sprite. We must resolve this from the official documentation rather than assume the two arrangements are interchangeable.

**GO: Proceed with the separate amendment document.** Do not directly revise the Agent Gateway implementation plan yet.

The current gateway plan places the Agent Gateway inside the Builder Sprite and expects it to supervise a coding-agent process.&#x20;

agent-gateway-implementation-plan.md

&#x20;The OpenCode amendment must resolve how that model works with Fly's documented integration.



Send the coding agent this instruction:

GO — execute Part 2 of `plans/roadmap3_plan.md`.

**Deliverable:** `plans/agent-gateway-opencode-amendment.md`

Use a separate, reviewable amendment document containing all nine sections (A–I), an exact merge list, and proposed changelog text.

Critical requirements:

1. OpenCode is our designated primary coding agent, using the official `@flydotio/sprites-opencode-plugin`.
2. Resolve the topology discrepancy using official sources. The plugin README describes OpenCode running outside the Sprite and using remote Sprite tools. Our gateway plan places the agent and gateway inside the Builder Sprite. Investigate both arrangements. Recommend a technically supported topology and explain its implications for Agent Gateway, session control, conversation bridging, filesystem access, and recovery.
3. Map every AgentAdapter operation to an actual, supported OpenCode interface. Do not invent APIs, CLI flags, or parse terminal output as a substitute for a structured interface.
4. Resolve version pins, plugin configuration, OAuth authentication, and authentication persistence. Clearly distinguish verified behavior from unproven assumptions.
5. Preserve the Studio-owned conversation protocol and the required real question → user answer → same-session continuation acceptance test.
6. Preserve F1, G-01, P5 scope, Invoice Review Desk as the current golden path, and Lead-to-Booking as the candidate second demonstration.
7. Identify any architectural amendment required if the supported OpenCode topology conflicts with the existing gateway design. Do not silently change locked decisions.

Execute one continuous drafting pass, followed by a concise final report.

Document-only authorization. Do not implement, install, authenticate, provision, modify the existing gateway plan or prompts, commit, or push.

The amendment must be technically grounded in current official documentation, version-stamped, and ready for my review and explicit approval before merging.

The official [Fly.io Agent Ecosystem](https://fly.io/sprites/ecosystem/?utm_source=chatgpt.com) confirms the OpenCode integration and browser OAuth setup. The [official plugin README](https://github.com/superfly/sprites-opencode-plugin/blob/main/README.md?utm_source=chatgpt.com) is the critical reference for resolving the topology.

**My preference:** preserve the Agent Gateway's purpose and Studio-owned conversation model, but do not force the implementation to preserve an unsupported execution topology.