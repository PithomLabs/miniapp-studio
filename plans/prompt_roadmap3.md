Use this prompt with the coding agent in the Miniapp Studio repository. It directs the agent to produce an implementation plan only, preserving our existing architectural decisions and project state.

# TASK: Write the Miniapp Studio Agent Gateway Implementation Plan

You are acting as the **Principal Software Architect and Technical Project Planner** for the Miniapp Studio and Miniapp repositories.

Your task is to write a comprehensive, implementation-ready technical plan for the Agent Gateway architecture we have approved.

**This is a planning task, not an implementation task. Do not write application code or begin implementing the roadmap.**

## 1. Objective

Create a durable implementation plan that establishes how Miniapp Studio communicates with coding agents running inside persistent Fly.io Sprites.

The objective is to enable a business user to describe a business process in Miniapp Studio, have Studio delegate the implementation work to a coding agent in a Builder Sprite, and maintain a continuous two-way conversation:

- Business user → Miniapp Studio.
- Miniapp Studio → Coding agent in Sprite.
- Coding agent → Miniapp Studio.
- Miniapp Studio → Business user.
- Business user → Miniapp Studio → Coding agent.

The business user must experience one continuous conversation, even though the coding agent operates remotely in a separate execution environment.

The plan must explain how to implement this reliably, securely, and incrementally without coupling Studio to a particular coding agent or making Studio dependent on a continuously running Sprite.

## 2. Locked architectural decisions

Treat the following as approved architectural decisions. Do not reopen them as optional alternatives unless repository evidence reveals a genuine foundational problem.

### A. Ownership boundaries

**Miniapp Studio owns:**

- Business-user conversations and permanent conversation history.
- Business requirements and decisions.
- Application and build assignments.
- Agent-session references and lifecycle status.
- Pending questions and user responses.
- Human approvals and authorization decisions.
- Build status, artifacts, test results, and deployment records.
- The business-user-facing chat interface.

**Miniapp owns:**

- The application framework and governed execution model.
- Runtime invocation and static tool registration.
- Policy enforcement and tenant context.
- Controlled access to consequential application capabilities.
- Execution evidence and audit boundaries.

**Fly.io Builder Sprite owns:**

- Persistent coding-agent working environment.
- Repository checkout and source files.
- Coding-agent execution.
- Build and test commands.
- Development dependencies and temporary working files.
- Agent-specific session data.
- Local build artifacts.

**Fly.io Application Runtime Sprite owns:**

- Running the approved application.
- Application services and runtime dependencies.
- The application's production execution environment.

**Fly.io Connectors own the provider credential boundary:**

- External API connections.
- Organization-managed provider credentials.
- Routing requests without exposing raw provider credentials to Sprites.

### B. Two distinct conversations

There are two communication relationships:

1. Business user ↔ Miniapp Studio.
2. Miniapp Studio ↔ Coding agent.

The first is the permanent business-facing conversation. The second is an internal agent communication protocol.

The user must never need to communicate directly with a Sprite or its terminal.

Miniapp Studio is the authoritative conversation owner and communication hub.

### C. Agent Gateway

Introduce an Agent Gateway (also called Agent Bridge) running within the Builder Sprite.

It must:

- Accept authenticated build assignments from Studio.
- Start and resume coding-agent sessions.
- Translate between Studio's stable communication protocol and the coding agent's native interface.
- Receive messages and structured events from the coding agent.
- Send those events to Studio through authenticated callbacks.
- Receive user answers and continuation commands.
- Support status inspection, artifact delivery, and cancellation.
- Survive ordinary Sprite sleep/wake cycles through persistent state and recoverable execution.

The gateway must remain independent of a particular coding-agent implementation.

Codex, Claude Code, and other compatible agents should be replaceable through adapters rather than requiring changes to Studio's conversation model.

### D. Communication channels

Use the following communication pattern as the baseline:

| Direction        | Mechanism                     |
| ---------------- | ----------------------------- |
| Browser → Studio | HTTP                          |
| Studio → Browser | Server-Sent Events (SSE)      |
| Studio → Sprite  | Authenticated HTTPS commands  |
| Sprite → Studio  | Authenticated HTTPS callbacks |

Do not require a permanent Studio-to-Sprite WebSocket connection.

The Sprite may sleep between work sessions. An incoming command must be able to wake it.

Studio must continue to function when the browser disconnects, the Sprite sleeps, or an agent process exits.

### E. Separation of environments

Builder Sprite and Application Runtime Sprite must be separate logical environments.

The coding agent must not automatically inherit unrestricted access to live business integrations.

The Builder Sprite is for development and testing. The Application Runtime Sprite hosts the approved Miniapp application.

Promotion of generated work from builder to runtime must have an explicit review and authorization boundary.

### F. Human control

Distinguish three kinds of interaction:

1. **Clarification:** The agent needs missing business information.
2. **Proposal:** The agent recommends an architectural or business decision.
3. **Authorization:** The agent requests permission for a consequential action.

An agent question must not be interpreted as authorization.

The agent cannot grant itself permission by emitting an event or changing its own working files.

Authorization must be represented and enforced by trusted Studio/Miniapp components.

## 3. Required roadmap

Structure the implementation into the following five phases. Preserve their intent and sequencing.

### Phase A — Conversation Infrastructure

Implement the authoritative conversation model within Miniapp Studio.

Plan for:

- Conversations and messages.
- User and agent message attribution.
- Build assignments.
- Agent sessions and turns.
- Structured agent events.
- Pending questions.
- User answers and decisions.
- Approval requests and authorization records.
- Build lifecycle and status transitions.
- Artifact metadata and references.
- Persistent conversation history.
- Browser reconnection and SSE event recovery.

Define the data model, relationships, uniqueness constraints, indexes, tenant scoping, and migration approach.

Establish which records are authoritative in Studio and which are merely references to files or sessions in a Sprite.

### Phase B — Sprite Agent Gateway

Implement the gateway service in a Builder Sprite.

Plan for:

- Sprite creation and lifecycle integration.
- Gateway installation and service startup.
- Authenticated command reception.
- Coding-agent adapter interface.
- Starting a new agent session.
- Maintaining a stable agent-session reference.
- Agent process supervision.
- Structured event extraction.
- Authenticated callbacks to Studio.
- Artifact discovery and delivery.
- Status reporting.
- Cancellation.
- Error handling and recovery.

Use Fly.io's actual documented Sprite APIs, SDKs, services, sessions, and execution semantics.

Do not assume that terminating an interactive process preserves an agent's native session automatically.

Determine how each supported coding agent can safely start, stop, and resume work. Where capabilities differ, document them in the adapter contract.

### Phase C — Genuine Two-Way Interaction

Prove the complete conversation bridge.

The implementation must support:

1. Business user submits a message.
2. Studio persists the message.
3. Studio creates a build assignment.
4. Studio wakes the Builder Sprite.
5. Gateway starts the coding agent.
6. Agent reports progress.
7. Studio persists and displays the progress.
8. Agent asks a question.
9. Studio persists and displays the question.
10. Agent stops or yields control cleanly.
11. User answers through the original Studio conversation.
12. Studio persists the answer and forwards it to the same build.
13. Sprite wakes.
14. Agent resumes with the answer and previous context.
15. Agent continues working and eventually reports its result.

The same conversation must remain intact throughout.

Explicitly test delayed responses, disconnected browsers, duplicate messages, duplicate callbacks, out-of-order events, Sprite sleep/wake, agent crashes, and interrupted builds.

### Phase D — Build and Review

Plan for the coding agent to generate a small Miniapp application artifact.

The agent must be able to:

- Inspect the assigned project.
- Understand the requirements and constraints.
- Produce a proposed implementation.
- Generate a MiniappSpec or application artifact as appropriate.
- Run tests.
- Produce machine-readable test results.
- Report failures honestly.
- Deliver diffs and artifacts to Studio.
- Request human review.

Studio should present the artifact and evidence through the original conversation and suitable review interfaces.

No automatic production deployment at this phase.

### Phase E — Deployment

After human approval, promote the reviewed artifact into its separate Application Runtime Sprite.

Plan for:

- Artifact identity and integrity verification.
- Review and approval records.
- Deployment assignment.
- Application Sprite provisioning.
- Configuration and connector setup.
- Service deployment and health verification.
- Deployment records and rollback strategy.
- Stable application URL.
- Operational visibility through Studio.

The application runtime must not become an unrestricted coding-agent environment.

## 4. Protocol design requirements

Define a stable, versioned communication protocol between Studio and Agent Gateway.

Evaluate the following event types as a starting vocabulary:

- `agent.started`
- `agent.progress`
- `agent.message`
- `agent.question`
- `agent.proposal`
- `agent.approval_required`
- `agent.artifact_ready`
- `agent.test_result`
- `agent.waiting_for_user`
- `agent.completed`
- `agent.failed`

Evaluate the following identifiers:

- `workspace_id`
- `application_id`
- `conversation_id`
- `build_id`
- `agent_session_id`
- `turn_id`
- `event_id`
- `question_id`
- `correlation_id`

Specify event envelopes, schema versioning, payload validation, ordering, delivery acknowledgement, deduplication, retries, replay, and compatibility rules.

Do not blindly adopt the example vocabulary. Refine it based on actual requirements and explain the decisions.

The protocol must be independent of the coding agent's native CLI output format.

## 5. Reliability and failure handling

Treat this as a distributed system, not a simple request/response integration.

Investigate and plan for:

- Duplicate command delivery.
- Duplicate event delivery.
- Callback authentication failures.
- Lost callbacks.
- Delayed and out-of-order callbacks.
- Agent process termination.
- Sprite sleeping during work.
- Sprite restoration from checkpoints.
- Gateway restart.
- Studio restart.
- Browser disconnection and reconnection.
- User answering an expired or superseded question.
- Multiple simultaneous builds.
- Build cancellation.
- Repeated user submissions.
- Failed artifact transfer.
- Partially completed builds.
- Conflicting agent responses.
- Unauthorized continuation attempts.

Define explicit lifecycle state machines for conversations, builds, agent turns, questions, and approvals where appropriate.

Do not equate HTTP 200 with successful command execution. Respect Fly's documented execution response and exit-status semantics.

Require idempotency, durable state transitions, and recoverable delivery.

## 6. Security and trust boundaries

This section is mandatory and must be treated as a foundational part of the architecture.

Investigate and specify:

- Workspace and tenant isolation.
- Authentication between Studio and Gateway.
- Callback authentication and replay protection.
- Authorization of commands and continuation requests.
- Secure binding between a Sprite and its assigned workspace/application.
- Safe handling of user-provided business requirements.
- Protection against prompt injection and untrusted repository contents.
- Coding-agent filesystem permissions.
- Secret isolation.
- Fly Connector permissions and access boundaries.
- Builder versus runtime permissions.
- Artifact provenance and integrity.
- Approval enforcement.
- Audit records.
- Sensitive data redaction in logs and agent output.
- Secure handling of uploaded documents or other user-provided artifacts.

Do not assume that a Fly Connector automatically prevents an authorized process inside a Sprite from using the associated integration.

Prefer limiting the Builder Sprite to development-only capabilities.

The trusted Miniapp runtime must remain responsible for consequential business operations.

Also reconcile this work with existing security decisions and unresolved issues in the repository, particularly F1 and G-01. Do not silently mark them resolved or bypass their approval requirements.

## 7. Coding-agent adapter architecture

Define an interface separating Studio's protocol from the underlying coding agent.

The plan should explain the responsibilities of:

- Agent adapter.
- Agent session manager.
- Process supervisor.
- Event translator.
- Gateway command handler.
- Callback delivery mechanism.
- Artifact collector.

Avoid introducing a generalized multi-agent framework unless necessary.

The initial implementation should prove one coding-agent integration end-to-end, but the architecture must permit another compatible agent to be added without changing Studio's conversation model.

## 8. Repository and implementation boundaries

Before designing the plan, inspect the actual repositories and current implementation.

Relevant modules:

- `github.com/PithomLabs/miniapp-studio`
- `github.com/PithomLabs/miniapp`

Inspect:

- Repository `AGENTS.md` instructions.
- Governing implementation plan: `plans/miniapp-studio-implementation-plan-consolidated.md`, Revision 2.
- Existing architecture decision records.
- P0–P4 evidence.
- Existing Miniapp SpriteManager and ConnectorManager abstractions.
- Existing deployment and application lifecycle implementation.
- Existing runtime invocation and audit architecture.
- Current Miniapp dependency pin and its relationship to the published `v0.1.0` tag.
- Existing test infrastructure and available fakes.

Do not assume a path or API exists merely because it appeared in an earlier discussion. Verify it in the current checkout.

Respect these boundaries:

- Preserve frozen historical plans:
  - `plans/plan4.md`
  - `plans/plan5.1.md`
  - `plans/plan6.md`
- Do not modify Miniapp unless a demonstrated general framework requirement exists and the established approval process permits it.
- Keep framework changes separate from Studio changes.
- Do not overwrite historical evidence.
- Do not implement the existing P5 builder roadmap as part of this planning task.
- Do not expand this into a generic task-management system.
- Do not introduce DBOS or a new workflow engine without a demonstrated requirement and explicit justification.
- Do not commit, push, or publish anything.

The Agent Gateway roadmap is a new, explicitly approved architectural direction, but it must be reconciled with the existing governing plan rather than silently replacing it.

## 9. Required deliverable

Create:

`plans/agent-gateway-implementation-plan.md`

The document must be a comprehensive, implementation-ready plan, not merely a conceptual architecture essay.

Use the following structure:

1. Executive summary.
2. Product objective and user experience.
3. Locked architectural decisions.
4. Current repository state and verified capabilities.
5. Architecture and component responsibilities.
6. Communication topology and sequence diagrams.
7. Conversation and build data model.
8. Agent Gateway protocol.
9. Coding-agent adapter contract.
10. Sprite provisioning and lifecycle.
11. Session persistence and recovery.
12. Security and trust boundaries.
13. Human questions, decisions, and approvals.
14. Artifact management and build review.
15. Deployment and application promotion.
16. Failure modes and recovery strategies.
17. Observability and audit requirements.
18. Testing and acceptance strategy.
19. Five-phase implementation roadmap.
20. Dependencies and prerequisite decisions.
21. Risks, unresolved questions, and owner decisions.
22. Explicit non-goals and scope boundaries.
23. Definition of done.

Include Mermaid diagrams for:

- Overall architecture.
- User-to-agent round-trip conversation.
- Agent question and answer lifecycle.
- Build and artifact promotion lifecycle.
- Relevant state machines.

## 10. Each implementation phase must be actionable

For every phase, define:

- Objective.
- Scope.
- Architectural ownership.
- Prerequisites.
- Expected source files/packages, based on verified repository structure.
- API and interface changes.
- Data model and migration changes.
- Implementation sequence.
- Unit tests.
- Integration tests.
- Failure and recovery tests.
- Security tests.
- Acceptance criteria.
- Deliverables.
- Exit criteria.
- Dependencies on later phases.

Identify the critical path and parallelizable work.

Do not invent API methods, SDK capabilities, or existing code structures. Clearly label proposed interfaces as proposals.

## 11. Mandatory engineering discipline

The plan must direct future coding agents to solve root causes, not symptoms.

Every implementation task should require:

- Investigation of the complete execution path.
- Identification of systemic failure modes.
- Preservation of architectural invariants.
- Regression tests that fail before and pass after the correction.
- Testing of replay, retry, race, concurrency, and failure cases where relevant.
- Durable corrections at the appropriate architectural ownership boundary.
- Explicit separation between code inspected and behavior actually executed.
- Honest reporting of unverified assumptions and environmental limitations.

Avoid superficial workarounds, local bypasses, duplicated subsystems, and speculative abstractions.

Do not let optional polish block foundational progress.

## 12. Project governance

The plan must clearly distinguish:

- Already implemented and verified.
- Accepted architecture not yet implemented.
- Proposed implementation details.
- Unresolved owner decisions.
- External dependencies.
- Blockers.

Use the existing project approval standard:

**Approve progress when there are no foundational or structural blockers.**

Do not create endless general review cycles.

A finding should be considered a blocker only when supported by concrete evidence, a reproducible failure or exploit, an identified root cause, an invariant breached, a durable correction, and a regression test.

Do not convert every uncertainty into a blocker.

Do not invent completion claims or test results.

Do not treat approval of this roadmap as authorization for live external writes, deployment, or other consequential actions.

## 13. Scope of this task

You are authorized to:

- Inspect the repositories and documentation.
- Inspect relevant Fly.io Sprite documentation and actual SDK/API references where needed.
- Analyze the existing architecture.
- Identify implementation dependencies and gaps.
- Write the requested implementation plan.
- Add a reference to the new plan in an existing plan index only if there is an established convention and doing so does not alter a frozen document.

You are NOT authorized to:

- Implement the Agent Gateway.
- Modify application source code.
- Make database migrations.
- Provision or destroy Sprites.
- Configure external connectors.
- Enable live business integrations.
- Change security decisions.
- Implement unresolved F1 or G-01 fixes.
- Modify the governing implementation plan without explicit justification and approval.
- Commit or push changes.

## 14. Final report

After writing the plan, report:

1. Exact path of the new plan.
2. Sections completed.
3. Existing architecture and code reused.
4. Important gaps discovered.
5. Decisions that require owner input.
6. Dependencies and critical path.
7. Any repository assumptions that could not be verified.
8. Confirmation that no application implementation, migrations, provisioning, commits, or pushes were performed.

Do not begin implementation after writing the document.

**The objective is to leave the project with a clear, technically grounded, sequenced implementation roadmap that another coding agent can execute phase by phase without losing architectural context or reinventing previous decisions.**

This gives us a durable implementation plan beneath the existing governing plan, without accidentally authorizing the coding agent to start building the subsystem.


Agreed. **OpenCode is now the designated initial coding agent for Miniapp Studio's Builder Sprites.** This is an important architectural decision, not merely an implementation preference.

I verified Fly.io's ecosystem page and the official [`superfly/sprites-opencode-plugin`](https://github.com/superfly/sprites-opencode-plugin?utm_source=chatgpt.com) README.

The integration provides several useful capabilities:

- OpenCode connects to Fly MCP at `https://mcp.fly.io`.
- The plugin provides Sprite management, remote execution, filesystem operations, checkpoints, and services.
- It supports browser-based OAuth.
- It supplies workflow guidance and safeguards for destructive Sprite operations.
- OpenCode 1 requires version 1.18.23 or later; OpenCode 2 has a separate beta configuration.
- OpenCode remains the agent, while its remote development and execution work happens inside Sprites.

One especially important distinction: **the Fly plugin connects OpenCode to Sprites; it does not itself implement our Miniapp Studio conversation bridge.** We still need the Agent Gateway and the Studio-owned conversation protocol.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly



## **Updated instruction for the coding-agent planning prompt**

Add the following section to the previous prompt, preferably under the locked architectural decisions and coding-agent adapter architecture.

## 15. Mandatory Coding Agent: OpenCode + Official Fly.io Sprites Plugin

**OpenCode is the designated initial and primary coding agent for Miniapp Studio Builder Sprites.**

This is an approved project decision, not an optional technology comparison.

Official references:

- Fly.io Agent Ecosystem: [https://fly.io/sprites/ecosystem/](https://fly.io/sprites/ecosystem/)
- Official plugin: [https://github.com/superfly/sprites-opencode-plugin](https://github.com/superfly/sprites-opencode-plugin)
- Plugin README: [https://github.com/superfly/sprites-opencode-plugin/blob/main/README.md](https://github.com/superfly/sprites-opencode-plugin/blob/main/README.md)

### A. Adopt the official integration

Use the official `@flydotio/sprites-opencode-plugin` package.

For OpenCode 1, the documented configuration is:

```
{
  "$schema": "https://opencode.ai/config.json",
  "plugin": ["@flydotio/sprites-opencode-plugin"]
}
```

The plugin requires OpenCode 1.18.23 or later. Its documented installation uses OpenCode's automatic Bun-based plugin installation.

The official repository also supports OpenCode 2 through a separate plugin entry point and configuration format. Do not assume the two versions have identical plugin APIs.

The implementation plan must identify the recommended OpenCode version to pin for the first production implementation. Prefer the stable OpenCode 1 integration unless repository evidence or an explicit owner decision supports OpenCode 2.

If the package is unavailable from npm at implementation time, follow the official repository's instructions for a local checkout and absolute `file://` configuration. Do not invent an alternative installation procedure.

### B. Understand exactly what this plugin provides

The official plugin connects OpenCode to Fly MCP at:

`https://mcp.fly.io`

It provides:

- Hosted MCP access to Fly.io Sprites.
- Browser-based OAuth authentication.
- Sprite listing and creation.
- Remote command execution.
- Remote filesystem operations.
- Service management.
- Filesystem checkpoints and restoration.
- Outbound network-policy operations.
- Sprite workflow guidance.
- Safeguards and approval prompts for destructive operations.

Do not recreate these capabilities unnecessarily in Miniapp Studio.

The official plugin must be used as the supported integration between OpenCode and Fly.io Sprites.

### C. Preserve the distinction between OpenCode, the plugin, and Agent Gateway

These are three different things:

**OpenCode:** The actual coding agent. It understands implementation instructions, examines repositories, writes code, runs tests, and produces artifacts.

**Official Sprites OpenCode Plugin:** The Fly-provided integration that gives OpenCode access to Sprite infrastructure through Fly MCP.

**Miniapp Studio Agent Gateway:** Our own integration component responsible for the business-facing two-way conversation protocol, build assignments, user questions, agent messages, event delivery, session association, and artifact reporting.

The Fly plugin does NOT automatically provide the Miniapp Studio conversation bridge.

Do not incorrectly treat the plugin's MCP connection as the mechanism for forwarding business-user chat messages to Studio.

### D. Define the OpenCode execution topology

The implementation plan must investigate and explicitly distinguish:

1. Where the OpenCode process runs.
2. Where the Sprite-management plugin is installed and configured.
3. Where the coding agent's actual development commands execute.
4. Where the repository and persistent working files reside.
5. How the Agent Gateway interacts with the OpenCode process.
6. How agent messages, questions, progress, and completion events are captured.
7. How a session is resumed after Sprite sleep or process termination.

The desired execution model is that the coding agent performs its development work in the persistent Builder Sprite, rather than accidentally running builds and file changes on Studio's host machine.

The official README explains that OpenCode can run outside the Sprite while using the plugin's remote execution tools to perform work in the Sprite. Therefore, do not assume that installing the plugin alone moves the OpenCode process into the Sprite.

Investigate the correct supported execution arrangement for our architecture. In particular, determine how to run OpenCode inside the Builder Sprite, how the official plugin is loaded there, and how its authentication and session state persist across sleep/wake cycles.

Where the official plugin is designed for a different topology, document the implications and recommend a supported approach rather than silently bypassing it.

### E. Authentication and credential boundaries

The official plugin uses browser-based OAuth for Fly MCP.

The plan must specify:

- How the initial OAuth authorization is performed.
- Which identity and Fly organization the Builder Sprite operates under.
- How authentication state survives normal Sprite lifecycle events.
- How access is restricted to the intended Sprites and organization.
- Whether restricted Sprite-name prefixes or creation limits are appropriate.
- How OpenCode's Fly MCP permissions are configured.
- Which destructive operations require human confirmation.

Do not place Fly API tokens or unrelated third-party API credentials in build prompts, source files, repositories, logs, or application artifacts.

Do not invent a non-interactive authentication workaround or install the Sprite CLI merely to bypass the supported plugin authentication flow.

### F. Versioning and compatibility

Pin and document:

- OpenCode version.
- `@flydotio/sprites-opencode-plugin` version.
- Node.js/Bun runtime requirements.
- Plugin configuration format.
- MCP endpoint.
- Authentication expectations.

Inspect the official plugin's `package.json`, README, release history, source, and tests where necessary.

Record verified compatibility facts and distinguish them from assumptions.

Do not depend on an unpinned floating Git branch in a reproducible production setup.

### G. OpenCode adapter contract

The Miniapp Studio Agent Gateway must have an OpenCode adapter as its first fully implemented coding-agent integration.

The adapter must translate between Studio's stable event protocol and OpenCode's actual supported session and execution interfaces.

Investigate OpenCode's supported programmatic interfaces, SDKs, CLI modes, event streams, session management, and non-interactive execution behavior.

Do not invent CLI flags or assume that terminal output is a reliable structured event protocol.

The implementation plan must establish how Studio can:

- Start an OpenCode build.
- Send an initial assignment.
- Receive assistant messages.
- Receive progress and tool activity.
- Detect when OpenCode requires user input.
- Forward a user's answer.
- Resume the same conversation and session.
- Detect completion or failure.
- Retrieve generated files, diffs, and test reports.

If OpenCode does not expose a particular capability through a stable supported interface, explicitly identify that limitation and design a tested adapter boundary around it.

### H. Acceptance proof

The first OpenCode integration must demonstrate this complete round trip:

1. Studio creates a build assignment.
2. Studio wakes or provisions the assigned Builder Sprite.
3. The Sprite starts the pinned OpenCode environment.
4. The official Fly Sprites plugin is loaded.
5. OpenCode verifies its authenticated Sprite connectivity.
6. OpenCode inspects the assigned repository inside the Sprite.
7. OpenCode sends a progress event to Studio.
8. OpenCode asks the business user a real clarification question.
9. Studio displays that question in the original conversation.
10. The user answers.
11. Studio forwards the answer to the same OpenCode session.
12. OpenCode resumes work.
13. OpenCode generates a small artifact and runs a test inside the Sprite.
14. OpenCode reports its result and evidence to Studio.
15. Studio displays the result in the original conversation.

Test process interruption, Sprite sleep/wake, duplicate events, reconnects, and session recovery.

Do not consider the integration complete merely because OpenCode can list Sprites or execute a remote shell command.

### I. Future agent portability

OpenCode is the first and primary implementation.

Keep the Agent Gateway and Studio conversation protocol sufficiently independent that other coding agents can be added later through adapters.

However, do not prematurely implement multiple coding-agent adapters or a generalized multi-agent orchestration framework.

The first milestone is a reliable, production-quality OpenCode integration.

**My architectural recommendation:** Run OpenCode as the coding agent inside the Builder Sprite, with the official Fly plugin providing Sprite-management and Fly MCP integration. Build our own Agent Gateway around OpenCode's supported session interfaces. This preserves the official Fly integration while giving Miniapp Studio ownership of the user-facing conversation.

The plugin's actual README and package metadata should be treated as the technical source of truth during planning, since its OpenCode 1 and 2 entry points differ.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly



One critical acceptance condition: prove the complete question → user answer → same OpenCode session resumes round trip. A successful `/sprites-status` check alone does not prove our architecture works.