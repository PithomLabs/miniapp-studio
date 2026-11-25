I recommend treating this as a **separate Agent Provider Expansion milestone**, not silently adding it to C6. C6 is already authorized with OpenCode as its initial adapter. We should preserve that scope while designing the provider architecture so Codex, Claude Code, and Antigravity can become selectable backends.

The prompt below asks the coding agent to inspect the actual implementation and produce an implementation plan first. It does not authorize provider implementation or live agent execution.

# Prompt 21 — Agent Provider Expansion: Codex, Claude Code and Antigravity

## Role

Act as the Senior Go Architect of Miniapp Studio. You own the architecture, implementation sequencing, provider integration contracts, security boundaries, and acceptance criteria.

Your task is to design the expansion of Miniapp Studio's agent-provider support beyond OpenCode.

## 1. Objective

Extend the existing agent adapter architecture to support four selectable agent providers:

1. OpenCode — existing implementation; must remain fully supported.
2. OpenAI Codex CLI.
3. Anthropic Claude Code CLI.
4. Google Antigravity CLI (`agy`).

The goal is to let an operator choose an available agent provider for a Miniapp Studio build without changing the Studio conversation model, Gateway contract, Miniapp execution model, or tenant boundaries.

The providers must be interchangeable at the Studio orchestration boundary, but their internal execution mechanisms must remain provider-specific where necessary.

Do not assume that these CLIs have identical process, session, streaming, authentication, permission, or recovery semantics.

## 2. Planning boundary

**This task is planning and repository investigation only. Do not implement the adapters yet.**

Inspect the actual repository and identify the existing implementation before proposing changes.

The current C6 Focus Console implementation has already been authorized separately. Do not silently expand or rewrite its approved scope. Treat provider expansion as a separate milestone that follows C6 Boundary B unless your dependency analysis demonstrates that a small, necessary change must be made earlier.

Do not alter the locked product architecture or implementation guidelines.

## 3. Mandatory repository investigation

Trace the complete execution path from Studio to the current OpenCode process and back.

Inspect:

- Existing agent adapter interface and OpenCode implementation.
- Gateway command, callback, and event contracts.
- Process creation, supervision, cancellation, exit handling, and restart/recovery.
- Session and conversation identity.
- Streaming output parsing and event normalization.
- The existing `agent.started`, yield, terminal, and waiting-user behavior.
- Studio question and answer lifecycle.
- MCP server configuration and tool exposure.
- Provider/model configuration and persisted build state.
- Authentication and credential injection.
- Builder workspace provisioning and binary installation.
- Tenant isolation and workspace ownership.
- Tests, fakes, fixtures, and existing operational documentation.

Trace actual call paths. Do not infer capabilities from interface names or documentation alone.

Identify which existing abstractions are genuinely provider-neutral and which accidentally encode OpenCode-specific assumptions.

## 4. Provider capability investigation

Prepare a provider comparison based on current official documentation and actual CLI behavior.

Use these as starting points, not immutable specifications:

**Codex CLI**

- Non-interactive execution through `codex exec`.
- JSONL event output through `--json`.
- Session/thread identity and supported resume behavior.
- Sandbox and approval configuration.
- MCP/tool integration.
- Cancellation, exit codes, and recovery behavior.

**Claude Code**

- Non-interactive execution through `claude -p`.
- Streaming output through `--output-format stream-json`.
- Session persistence, continuation, and resume.
- Permission modes and permission-prompt handling.
- MCP/tool integration.
- Cancellation, exit codes, and recovery behavior.

**Antigravity**

- Non-interactive execution through `agy -p`.
- Machine-readable output through `--output-format stream-json`.
- Conversation identity and continuation.
- Multi-turn streaming input/output capabilities.
- Permission modes and scoped command/file permissions.
- Cancellation, exit codes, and recovery behavior.

Verify current CLI versions, flags, output schemas, and permission semantics against official documentation and, where appropriate, local help output. Record the sources and verification date.

Do not invent flags, event types, or capabilities. Mark undocumented or unverified behavior explicitly.

## 5. Provider capability matrix

Produce a table comparing all four providers:

| Capability                      | OpenCode | Codex | Claude Code | Antigravity |
| ------------------------------- | -------- | ----- | ----------- | ----------- |
| Headless execution              | <br>     | <br>  | <br>        | <br>        |
| Structured event stream         | <br>     | <br>  | <br>        | <br>        |
| Persistent process / multi-turn | <br>     | <br>  | <br>        | <br>        |
| Resume after process restart    | <br>     | <br>  | <br>        | <br>        |
| Conversation/session identity   | <br>     | <br>  | <br>        | <br>        |
| Prompt delivery                 | <br>     | <br>  | <br>        | <br>        |
| Studio clarification support    | <br>     | <br>  | <br>        | <br>        |
| MCP tool support                | <br>     | <br>  | <br>        | <br>        |
| Tool activity events            | <br>     | <br>  | <br>        | <br>        |
| Permission handling             | <br>     | <br>  | <br>        | <br>        |
| Workspace restrictions          | <br>     | <br>  | <br>        | <br>        |
| Cancellation                    | <br>     | <br>  | <br>        | <br>        |
| Exit/terminal semantics         | <br>     | <br>  | <br>        | <br>        |
| Usage/cost metadata             | <br>     | <br>  | <br>        | <br>        |
| Authentication model            | <br>     | <br>  | <br>        | <br>        |

Use `supported`, `partially supported`, `unsupported`, or `unverified`, followed by evidence and limitations.

Do not label a provider fully supported merely because it can execute a prompt and print a response.

## 6. Architecture requirements

Propose a provider-neutral contract that preserves the existing Gateway and Studio semantics.

Evaluate whether the current adapter interface can support:

- Provider discovery and availability.
- Provider-specific configuration.
- Starting a new agent session.
- Sending a new turn.
- Streaming normalized events.
- Resuming or reconciling an existing session.
- Cancellation and process termination.
- Health and version reporting.
- Capability reporting.
- Structured error reporting.

Avoid building a generic plugin marketplace or a second agent orchestration framework.

Prefer the smallest durable extension of the existing architecture.

Provider-specific process management and event parsing should remain inside provider-specific adapters. Do not scatter CLI-specific branching throughout Gateway or Studio business logic.

Define how provider-specific events map into the existing internal lifecycle. Preserve the distinction between:

- Agent started.
- Agent is working.
- Agent yielded and is waiting for the user.
- Agent completed successfully.
- Agent failed.
- Agent was cancelled.
- Agent outcome is indeterminate and requires reconciliation.

A provider's normal response completion must not automatically be interpreted as successful Miniapp build completion.

### Clarification compatibility

Investigate how each provider can invoke Studio's clarification mechanism.

Determine whether the provider can use the required Studio MCP tool or another supported, authenticated mechanism to yield a structured question.

Do not depend on parsing arbitrary natural-language output to infer that the agent is waiting for the user.

A provider that cannot satisfy the clarification contract must be identified as having a bounded capability limitation. Do not fabricate equivalent behavior.

### Session identity and recovery

Define how Studio stores and correlates:

- Studio build and turn IDs.
- Provider identity and provider version.
- Provider-specific session/thread/conversation ID.
- Gateway execution identity.
- Current lifecycle state.
- Last acknowledged event or recoverable cursor.

Address process crashes, duplicate events, lost responses, retries, stale callbacks, and provider session expiration.

Never assume that resuming a process is equivalent to resuming an agent conversation.

## 7. Security and credential requirements

Security is a technical contract, not a prompt instruction.

Investigate the authentication model of each CLI and propose a safe tenant-aware design.

Requirements:

- Never mix provider credentials between tenants.
- Never persist raw provider credentials in conversation history, command payloads, logs, Pulse, or ordinary build configuration.
- Never expose credentials to browser code.
- Never mount a user's entire home directory into an untrusted Builder workspace.
- Never inherit the entire host environment indiscriminately.
- Use explicit environment allowlists and secret redaction.
- Preserve workspace isolation and process ownership.
- Use least-privilege provider permissions.
- Never silently escalate to full-access or approval-bypass modes.
- Never use Codex `--yolo`, Claude Code `bypassPermissions`, or Antigravity `--dangerously-skip-permissions` as ordinary execution defaults.
- Identify any provider whose authentication or permission model cannot be safely supported under the current Builder architecture.

If safe credential injection or tenant isolation cannot be demonstrated, report the blocker rather than inventing a workaround.

## 8. Process and protocol reliability

Design for real process behavior, not idealized CLI output.

Include:

- Argument-vector execution; no shell-string interpolation.
- Absolute executable resolution.
- Bounded stdout and stderr handling.
- Incremental JSONL/NDJSON parsing.
- Partial lines and malformed event handling.
- Process-group cancellation and child-process cleanup.
- Startup, execution, and idle timeouts.
- Resource limits.
- Exit-code interpretation.
- Signal handling.
- Duplicate and out-of-order events.
- Provider upgrades and version compatibility.
- Recovery when the process exits after accepting a prompt but before Studio records the result.

Do not assume all output belongs to stdout or that stderr is always non-semantic.

Define a common contract-test suite that every provider adapter must pass.

## 9. Local-first implementation strategy

Preserve the current local-first decision.

For this planning task:

- Do not invoke Sprite CLI, including read-only commands.
- Do not perform Fly operations.
- Do not provision cloud resources.
- Do not install or authenticate provider CLIs.
- Do not execute real model inference.
- Do not modify production credentials or account state.
- Do not run live OpenCode, Codex, Claude Code, or Antigravity sessions.

Propose fake-executable tests for all four adapters.

Use provider-accurate captured or synthetic protocol fixtures, clearly labeled as such. The test suite must not claim live-provider validation from fake process tests.

Actual provider smoke tests belong to a later, explicitly authorized environment-validation milestone.

## 10. Studio user experience

Determine how operators should select and configure an agent provider.

Evaluate:

- Provider selection during build creation or configuration.
- Provider availability and version display.
- Model selection where supported.
- Permission profile visibility.
- Authentication status without exposing secrets.
- Clear unavailable-provider and authentication-error states.
- Provider identity in build history and Pulse.
- Behavior when a previously selected provider becomes unavailable.

Do not expand C6 into a full provider-management UI. Identify the minimal UI implications and place broader configuration UI in the appropriate future milestone.

The operator must always know which provider is executing the current build.

## 11. Testing and acceptance strategy

Propose red-before/green-after tests for the actual integration boundaries.

At minimum cover:

- Shared adapter contract tests.
- One provider-specific parser suite per provider.
- Prompt delivery and stable execution identity.
- Structured event mapping.
- Clarification and answer round-trip.
- Process startup failure.
- Authentication failure.
- Malformed or truncated stream.
- Duplicate and out-of-order events.
- Lost response after accepted execution.
- Retry and reconciliation.
- Cancellation and child-process cleanup.
- Tenant isolation.
- Workspace isolation.
- Credential and log redaction.
- Provider version mismatch.
- Provider unavailable at startup.
- Provider session not resumable.

Require concurrency, crash, retry, and recovery tests wherever the invariant is relevant.

## 12. Required deliverable

Create:

`plans/agent-provider-expansion-plan.md`

The plan must contain:

1. Executive decision and scope.
2. Current architecture inventory with exact file paths.
3. Complete OpenCode execution-path trace.
4. Provider capability matrix with sources.
5. Proposed provider-neutral adapter contract.
6. Provider-specific integration designs.
7. Event and session lifecycle mapping.
8. Authentication and security design.
9. Schema and migration impact assessment.
10. Studio UI implications.
11. Test strategy and concrete acceptance criteria.
12. Sequenced implementation units with dependencies.
13. Risks, unresolved questions, and explicit blockers.
14. Local-first and later live-validation boundaries.
15. Definition of done for each provider.

Update no implementation code and do not modify the approved C6 plan.

## 13. Decision discipline

The objective is not to make four CLIs appear superficially interchangeable.

The objective is to make four genuinely usable agent providers operate through one stable Studio orchestration contract without weakening security, recovery, or user control.

If a provider cannot satisfy a required contract, identify the limitation and recommend an explicit capability boundary. Do not conceal it behind compatibility shims.

Solve underlying architectural problems, not symptoms. Require durable invariants and regression tests in the eventual implementation plan.

Stop after producing the plan. Report the important findings, identify any structural blockers, and request architectural acceptance before implementation.

The official interfaces already show why provider-specific handling matters: Codex documents `codex exec` and JSONL events, Claude Code documents headless streaming, and Antigravity documents both streaming events and conversation continuation. [Codex documentation](https://learn.chatgpt.com/docs/non-interactive-mode?utm_source=chatgpt.com) · [Anthropic guidance](https://www.anthropic.com/research/claude-code-best-practices?utm_source=chatgpt.com) · [Antigravity headless CLI](https://antigravity.google/docs/cli/headless/?utm_source=chatgpt.com). The implementation plan should verify the exact versions and contracts again when it inspects the repository.

**Decision:** Keep C6 moving independently. Run this as a separate planning task; review the resulting provider-expansion plan before authorizing implementation.