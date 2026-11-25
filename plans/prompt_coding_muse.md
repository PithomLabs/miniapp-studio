Yes. I would include **Meta Muse Code and Meta Model API**, rather than treating Muse Spark as just another CLI.

There are two distinct integration opportunities:

- **Muse Code:** Meta's first-party terminal coding agent, using the `muse` executable. It supports headless execution, JSONL output, session continuation, and a versioned programmatic session protocol through `muse serve`. [Official documentation](https://dev.meta.ai/docs/muse-code?utm_source=chatgpt.com).&#x20;

  [image](https://www.google.com/s2/favicons?domain=https://dev.meta.ai\&sz=32)

  Meta Model API+1

- **Meta Model API:** Exposes Muse Spark and Muse Wonder through OpenAI-compatible and Anthropic-compatible interfaces. This could also let Codex, Claude Code, and OpenCode use Meta models without replacing their agent runtimes. [Official documentation](https://dev.meta.ai/docs/coding-agents?utm_source=chatgpt.com).&#x20;

  [image](https://www.google.com/s2/favicons?domain=https://dev.meta.ai\&sz=32)

  Meta Model API+1



Append this amendment to Prompt 21. It supersedes the earlier provider count and adds requirements to the capability matrix and architecture proposal.

# Prompt 21 Amendment — Meta Muse Support

This amendment extends Prompt 21. All its original requirements remain binding.

## 1. Additional integration target: Meta Muse

Add support for Meta's Muse ecosystem, including both:

1. **Muse Code** — Meta's first-party terminal coding agent, installed as the `muse` executable.
2. **Meta Model API** — hosted inference API exposing Muse Spark, Muse Wonder, and other available Muse models.

Do not conflate an agent runtime with an inference/model provider. They are separate architectural concerns.

The target agent runtimes are now:

| Agent runtime           | Provider  |
| ----------------------- | --------- |
| OpenCode                | OpenCode  |
| Codex CLI               | OpenAI    |
| Claude Code CLI         | Anthropic |
| Antigravity CLI (`agy`) | Google    |
| Muse Code CLI (`muse`)  | Meta      |

Additionally, investigate Meta Model API as a selectable model backend for compatible agent runtimes.

## 2. Muse Code integration

Investigate the actual capabilities of the current Muse Code release.

Official starting points:

- [https://dev.meta.ai/docs/muse-code](https://dev.meta.ai/docs/muse-code)
- [https://dev.meta.ai/docs/muse-code/extending](https://dev.meta.ai/docs/muse-code/extending)
- [https://dev.meta.ai/docs/muse-code/permissions](https://dev.meta.ai/docs/muse-code/permissions)

Specifically investigate:

- Native executable resolution and version pinning.
- Interactive versus headless execution.
- `muse exec` and its JSONL event output.
- Session creation and continuation using `--session-id`.
- Programmatic session management through `muse serve`.
- The versioned protocol schema exposed by `muse schema`.
- Process termination, cancellation, and exit semantics.
- MCP server configuration and tool availability.
- Workspace trust and project-level configuration.
- Permission profiles, approval modes, and OS sandbox enforcement.
- Browser authentication and API-key authentication.
- `META_API_KEY` handling in non-interactive environments.
- Model selection and availability.

### Important architecture decision

Do not automatically build the Muse adapter around one-shot `muse exec`.

Investigate whether the versioned `muse serve` session protocol is a better fit for the existing Gateway's long-lived, multi-turn interaction contract.

Compare both approaches:

**Option A — Headless process execution**

- Launch `muse exec`.
- Consume structured JSONL events.
- Persist the session identity.
- Reconcile and resume using the documented session continuation mechanism.

**Option B — Programmatic session protocol**

- Launch and supervise `muse serve`.
- Use its versioned protocol.
- Derive supported lifecycle, streaming, input, and control capabilities from `muse schema`.
- Determine how the protocol maps to Gateway's existing lifecycle and recovery requirements.

Recommend the smallest reliable implementation based on evidence. Do not assume that the TypeScript SDK must be adopted by the Go Gateway.

A completed Muse Code turn is not proof that a Miniapp was built correctly. Exit code zero is not an artifact acceptance signal. Preserve the existing Phase D verification boundary.

## 3. Meta Model API integration

Investigate:

[https://dev.meta.ai/docs/overview](https://dev.meta.ai/docs/overview)

[https://dev.meta.ai/docs/coding-agents](https://dev.meta.ai/docs/coding-agents)

[https://dev.meta.ai/docs/muse-spark](https://dev.meta.ai/docs/muse-spark)

Evaluate the API as a distinct model-provider capability.

Investigate:

- OpenAI-compatible Responses API.
- OpenAI-compatible Chat Completions API.
- Anthropic-compatible Messages API.
- Model identifiers and current model availability.
- Muse Spark and Muse Wonder.
- Reasoning configuration and multi-turn reasoning continuity.
- Tool calling and streamed tool-call arguments.
- Context and output limits.
- Usage, rate limits, and billing.
- Authentication using `MODEL_API_KEY`.
- Provider-specific error and retry semantics.

Determine whether Codex, Claude Code, and OpenCode can use Meta Model API as an alternative inference backend.

Do not assume that API compatibility guarantees equivalent functionality across agents. Test the actual protocol and feature subset.

### Critical architectural distinction

Keep these concepts separate:

- Agent runtime: the program that orchestrates tool calls, edits files, runs commands, and manages the agent session.
- Model provider: the inference service that generates reasoning and tool-call responses.
- Model: the selected Muse Spark, Muse Wonder, or other supported model identifier.

A Codex session using Muse Wonder is still a Codex agent session. A Muse Code session using Muse Spark is a Muse Code agent session.

Avoid duplicating agent adapters merely because two runtimes use the same model backend.

## 4. Revised capability matrices

Add Muse Code as a fifth agent runtime to the original provider matrix.

Add a separate model-provider matrix:

| Model provider | API compatibility | Model selection | Streaming | Tool calling | Multi-turn continuity | Authentication |
| -------------- | ----------------- | --------------- | --------- | ------------ | --------------------- | -------------- |
| OpenAI         | Verify            | Verify          | Verify    | Verify       | Verify                | Verify         |
| Anthropic      | Verify            | Verify          | Verify    | Verify       | Verify                | Verify         |
| Google         | Verify            | Verify          | Verify    | Verify       | Verify                | Verify         |
| Meta Model API | Verify            | Verify          | Verify    | Verify       | Verify                | Verify         |

Do not fill the table using assumptions. Provide current evidence and limitations.

## 5. Additional security requirements

Muse Code has its own approval and OS sandbox mechanisms. Evaluate how these coexist with Miniapp Studio's security boundaries.

In particular:

- Never use unrestricted or `--yolo` execution as the normal Builder profile.
- Do not silently disable the Muse sandbox.
- Verify that its Linux sandbox functions in the actual intended Builder environment.
- Treat sandbox initialization failure as an execution failure.
- Do not allow Muse workspace trust to expand access outside the assigned Builder workspace.
- Keep Meta API credentials isolated from other tenants and providers.
- Distinguish the authentication requirements of Muse Code from Meta Model API.
- Never assume that a browser-authenticated local session is transferable to an isolated Builder environment.

## 6. Revised definition of done

The provider expansion is not complete merely because five executable names can be launched.

Each agent runtime must demonstrate:

- Stable session correlation.
- Structured event normalization.
- Correct Studio clarification and answer behavior.
- Cancellation and process cleanup.
- Recovery or explicitly documented inability to resume.
- Tenant and workspace isolation.
- Secret redaction.
- Accurate terminal semantics.
- Provider-specific contract tests.

Meta Model API support must separately demonstrate that its model backend can be used through the selected compatible agent interfaces without breaking tool execution, streaming, or multi-turn continuity.

## 7. Deliverable updates

Retain the original deliverable:

`plans/agent-provider-expansion-plan.md`

Update it to include:

- Muse Code execution-path analysis.
- Comparison of `muse exec` and `muse serve`.
- Muse protocol and session lifecycle mapping.
- Muse permission and sandbox compatibility.
- Meta Model API as a separate model-provider layer.
- Codex/Claude Code/OpenCode compatibility with Muse models.
- Revised architecture diagrams and capability matrices.
- Revised implementation sequencing and acceptance criteria.

Do not implement the adapters or run real Muse sessions during this planning exercise.

Preserve the existing local-first restrictions and stop for architectural acceptance before implementation.

**Architectural recommendation:** Make the distinction between agent runtime and model backend a first-class part of this investigation. That gives Miniapp Studio the potential to support five agent runtimes and multiple model providers without creating a separate adapter for every possible combination.