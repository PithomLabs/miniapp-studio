I strongly recommend adding this to Miniapp Studio's architecture, but as a **roadmap-ready Agent Provider and Execution Adapter layer**, not as another feature bolted directly into the application builder.

This is an important extension because it separates two things that should never be coupled:

- **Agent intelligence:** Gemini, Claude, Codex, OpenAI Agents SDK and other agent providers.
- **Agent execution environment:** Fly Sprites, Fly Machines and other supported execution environments.

Miniapp Studio should eventually let a customer choose the agent technology they prefer without changing how their generated applications are defined, governed, rendered or managed.

The official Fly documentation confirms that both Gemini Managed Agents and Claude Managed Agents can use Sprites as their execution environment.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+2



## **1. The architecture I recommend**

Introduce a new architectural layer:

**Agent Provider Abstraction**

**Miniapp Studio**

Create applications, choose an agent provider, configure integrations, preview, deploy and manage.

**Agent Provider Abstraction**

**Gemini Managed Agents**

Roadmap adapter

**Claude Managed Agents**

Roadmap adapter

**OpenAI Agents**

Roadmap adapter

**Other providers**

Extensible adapters

**Execution Environment Abstraction**

Fly Sprites for isolated agent tools and sessions; Fly Machines for long-lived trusted services and production control planes.

**Miniapp Foundation**

MiniappSpec, capability registry, authorization, connectors, UI renderer, application lifecycle, audit and eventually DBOS durable workflows.

This architecture means a generated application does not have to know whether its AI agent is powered by Gemini or Claude.

## **2. What the future Studio UI should provide**

Add an **AI Agents** section alongside Connectors.

[AI Catalog | Manage Models, MCP Servers, Tools & Agents | Gravitee](https://images.openai.com/static-rsc-4/I_awpsUHgSYDNAnDQURHdm8om4Z-5d531Qup2MeZKRfqSWBRTx022dKJwiRJRpb332jIlXJncS6Ens5vPGS4o6Zl9funcvhfe4MliR7MvULq4mVGRHAHceWhGxsEaGuw_HZtI5VompbZQeJIZq3uU7JXlZjD8Oodyenx27kZ7n8?purpose=inline)

[gravitee.io](https://www.gravitee.io/platform/ai-catalog?utm_source=chatgpt.com)

The eventual provider selection screen should show:

| Provider              | Execution integration                | Studio status                        |
| --------------------- | ------------------------------------ | ------------------------------------ |
| Gemini Managed Agents | Google agent loop + Sprite worker    | Roadmap                              |
| Claude Managed Agents | Anthropic agent loop + Sprite worker | Roadmap                              |
| OpenAI Agents SDK     | Agent SDK + Sprite sandbox           | Roadmap                              |
| Claude Code           | Coding agent + Sprite integration    | Existing Fly integration to evaluate |
| Codex                 | Coding agent + Sprite integration    | Existing Fly integration to evaluate |
| Custom agent          | User-provided adapter                | Future extension                     |

The distinction between managed agents and coding-agent integrations is important. They are different integration types and should not be presented as interchangeable.

The configuration screen should eventually allow users to select:

- Agent provider and model.
- Provider account and authorization.
- Execution environment.
- Session or agent configuration.
- Available tools and connectors.
- Permission boundaries.
- Agent-specific usage and cost limits.

Provider credentials and agent-specific secrets must be handled through appropriately scoped control-plane integrations, not embedded in generated Miniapp specifications or exposed to untrusted Sprite processes.

## **3. Why this matters commercially**

Imagine two companies creating the same invoice-management application.

Company A prefers Gemini.

Company B prefers Claude.

Both should be able to generate the same Miniapp application definition, connect the same business systems and use the same governance policies.

Only the agent provider configuration changes.

That is an important form of platform independence.

It also avoids making Miniapp Studio dependent on one LLM vendor's agent framework.

## **4. Important implementation distinction**

I inspected Google's Fly Machines integration documentation through its GitHub repository.

It describes a self-hosted Gemini environment that can run inside either a Sprite or a raw Fly Machine. The integration uses a `gemini-api` worker, environment ID and environment key, a supervised service, and a task heartbeat to keep a Sprite active while the worker is running.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly.io Changelog



This should be treated as a separate adapter from Gemini Managed Agents.

I recommend three distinct provider integration categories:

| Integration category           | Example                                      | Roadmap treatment                                        |
| ------------------------------ | -------------------------------------------- | -------------------------------------------------------- |
| Managed agent providers        | Gemini Managed Agents, Claude Managed Agents | Planned                                                  |
| Self-hosted agent environments | Gemini API CLI on Sprites or Machines        | Planned                                                  |
| Coding-agent integrations      | Claude Code, Codex, Cursor                   | Existing Fly ecosystem; evaluate and expose where useful |

For Claude Managed Agents, the official reference implementation uses a Fly dispatcher and Sprite worker architecture.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

GitHub+1



## **5. Addition to the coding-agent prompt**

Add the following section to the existing implementation-plan prompt.

## Additional Roadmap Requirement: Pluggable Managed Agents and Agent Providers

Miniapp Studio must be architecturally ready to support managed and self-hosted AI agent providers, including Gemini Managed Agents, Claude Managed Agents, and future providers.

This is a POST-HACKATHON ROADMAP REQUIREMENT. Do not implement these integrations as part of the Hacksgiving MVP unless a minimal architectural seam is necessary to avoid costly redesign.

### A. Product objective

Non-technical users must eventually be able to select their preferred AI agent provider when creating or managing a generated Miniapp application.

The application specification, UI, connectors, business capabilities, authorization policies and Miniapp execution contracts must not become coupled to one AI vendor.

Changing the agent provider must not require rebuilding the generated application.

### B. Required provider categories

Distinguish the following integration types:

1. Managed agent providers:
   - Gemini Managed Agents using Fly Sprites.
   - Claude Managed Agents using Fly Sprites.
   - Future managed-agent providers.
2. Self-hosted agent environments:
   - Gemini API CLI self-hosted environment on Sprites.
   - Gemini API CLI self-hosted environment on Fly Machines.
   - Future self-hosted agent runtimes.
3. Coding-agent and harness integrations:
   - Claude Code.
   - Codex.
   - Cursor.
   - Other Fly-supported coding agents and MCP clients.

Do not conflate these categories. They have different agent loops, execution contracts, authentication requirements, lifecycle semantics and operational responsibilities.

### C. Architecture requirements

Design a provider-neutral Agent Provider abstraction with separately implemented adapters.

The plan must define contracts for:

- Provider discovery and registration.
- Provider account and authorization references.
- Agent and model selection.
- Agent/session creation and termination.
- Execution-environment selection.
- Sprite or Machine provisioning.
- Tool-call and worker lifecycle.
- Session status and event handling.
- Callbacks/webhooks where applicable.
- Logs and usage reporting where supported.
- Error classification and recovery.
- Credential and secret isolation.
- Provider capability discovery and compatibility validation.

Keep the provider control plane separate from the generated Miniapp application runtime.

The provider must not be able to bypass Miniapp authorization, capability restrictions, business-operation approvals or audit requirements.

### D. Fly-native integration requirements

Use official provider integration documentation, official SDKs and repositories as primary evidence.

Evaluate:

- Gemini Managed Agents on Sprites:\
  [https://docs.fly.io/sprites/integrations/gemini-managed-agents](https://docs.fly.io/sprites/integrations/gemini-managed-agents)
- Claude Managed Agents on Sprites:\
  [https://docs.fly.io/sprites/integrations/claude-managed-agents](https://docs.fly.io/sprites/integrations/claude-managed-agents)
- Gemini API CLI self-hosted Fly provider:\
  [https://github.com/google-gemini/gemini-api-cli/tree/main/docs/self-hosted/providers/fly-machines](https://github.com/google-gemini/gemini-api-cli/tree/main/docs/self-hosted/providers/fly-machines)
- Fly Sprites ecosystem:\
  [https://fly.io/sprites/ecosystem/](https://fly.io/sprites/ecosystem/)

Record the differences between managed providers, self-hosted environments, Sprites and raw Fly Machines.

Do not assume that the Gemini Managed Agents integration and Gemini API CLI self-hosted integration are the same product or use the same credential and lifecycle model.

### E. Future Studio UI

Reserve an AI Agents section with:

- Supported provider catalog.
- Provider type and integration status.
- Connect/disconnect account.
- Model and agent configuration.
- Execution target selection.
- Session and agent management.
- Usage and cost visibility.
- Provider health and diagnostics.

Clearly distinguish available, configured, authorized, deployed and operational states.

### F. Security requirements

Investigate provider-specific trust boundaries.

Never expose provider API keys, environment keys, Fly API tokens or webhook secrets in generated MiniappSpecs, user-visible logs, untrusted Sprite processes or browser responses.

Use least-privilege credentials, isolated execution environments, revocation and auditable lifecycle operations.

The agent provider must remain an execution/intelligence dependency, not the authority over Miniapp business actions.

### G. Implementation-plan deliverables

Add a dedicated roadmap phase for managed and self-hosted agent integrations.

Include:

- Provider-neutral architecture and Go interfaces.
- Provider-by-provider capability matrix.
- Authentication and secret-management architecture.
- Fly Sprite/Machine lifecycle adapters.
- Gemini and Claude integration sequence.
- Session persistence and recovery considerations.
- Webhook and event-handling contracts.
- Security threat model.
- Contract tests and acceptance criteria.
- Cost, resource and operational trade-offs.

Separate architectural seams needed in the MVP from actual post-hackathon provider integrations.

Do not claim Gemini Managed Agents, Claude Managed Agents or Gemini self-hosted integration is implemented merely because the Studio exposes a provider selector.

The final deliverable remains an implementation plan only. Do not implement source code.

## **6. My final architectural recommendation**

I would now describe Miniapp Studio as having four independent dimensions:

1. **Application definition:** What software does the user want?
2. **Agent provider:** Which AI agent or managed service powers the intelligence?
3. **Execution environment:** Where does the agent run—Sprite or Machine?
4. **Business execution:** Which Miniapp capabilities, permissions, integrations and workflow operations can it invoke?

This separation is fundamental to making Miniapp vendor-neutral and Fly-native at the same time.

The hackathon MVP should establish the seams without implementing all the provider adapters.

**The long-term vision: users choose what they want to build, which AI they prefer, and where it should run. Miniapp governs how the resulting business application operates.**