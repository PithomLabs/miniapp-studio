Use this prompt with the coding agent operating in the Miniapp Studio repository. It directs the agent to investigate the current implementation and produce a phased plan only—not start implementation.

# Task: Write the Phased Implementation Plan for Miniapp Studio’s Unified Development-to-Production Workflow

Act as the **Senior Software Architect and Technical Product Planner for Miniapp Studio**.

Your task is to investigate the existing repository, architecture documents, implementation state, and accepted evidence, then write a comprehensive, dependency-aware implementation plan for the next evolution of the **Miniapp Studio UI and its unified application lifecycle workflow**.

The objective is to make Miniapp Studio feel like a cohesive development and operations platform, combining ideas from Encore.dev and Kilo Desktop while retaining Miniapp Studio’s existing architecture and distinctive Focus Console design.

**This is a planning task only. Do not implement the plan.**

Write the resulting document to:

`docs/architecture/unified-lifecycle-ui-implementation-plan.md`

If that path already contains work, inspect it and preserve relevant content rather than blindly overwriting it. Do not modify unrelated files.

---

## 1. Product objective

The user should experience one unified workflow for developing, validating, deploying, operating, and troubleshooting an application across:

- Local development
- Remote development
- Preview
- Staging
- Production

The central product proposition is:

> One application, one continuous development journey, from local development to production.

The workflow must make it straightforward to move between development environments, understand which version is running where, inspect application health, investigate failures, and promote verified releases safely.

Miniapp Studio itself is the first dogfooding target. The near-term demo is Miniapp Studio developing itself—not the previous Invoice Review Desk concept.

The plan must distinguish two scopes throughout:

1. **Hackathon/demo-critical path:** the minimum credible end-to-end workflow for demonstrating Miniapp Studio developing itself.
2. **Long-term product roadmap:** the complete, production-grade platform vision.

Do not confuse a demo shortcut with a permanent architectural decision.

---

## 2. Binding architectural principles

Treat the following decisions as constraints, not suggestions.

### 2.1 Product architecture

- Miniapp Studio is the web application and control plane.
- Coding agents execute separately and are supervised by Studio through the Gateway.
- Each application has a dedicated Builder workspace/environment.
- Runtime applications are separate from Studio and Builder.
- Miniapp is the exclusive application framework/runtime.
- Go is the exclusive application implementation language.
- MiniappSpec is the canonical application specification.
- Enforce framework, language, and specification requirements at validation, admission, build, and deployment boundaries—not through prompts alone.
- Studio's durable database is authoritative for conversations, turns, questions, approvals, artifacts, builds, deployments, events, and history.
- Workspace files, processes, and the filesystem are execution state, not the source of truth for durable Studio state.
- The runtime receives only the exact approved artifact. It must not depend on a mutable Builder workspace, agent process, Gateway process, or development credentials.
- Agents cannot authorize their own work. Human approval is separate from agent execution.
- G-01 remains binding: synthetic data and simulated approval effects; no consequential external business writes.

Do not introduce a generic workflow canvas, marketplace, broad multi-agent framework, or large fleet/quota/billing subsystem without evidence that it is necessary.

### 2.2 Workspace execution and application environments are independent

This distinction is essential to the UI and architecture.

**Workspace execution location:**

- Local Desktop
- Remote Builder Sprite

**Application runtime environment:**

- Local
- Preview
- Staging
- Production

These are two independent dimensions and must be represented by separate controls.

For example, a developer may use a Remote Builder Sprite to modify an application that is ultimately deployed to Preview. Do not model this as one combined Local/Remote/Preview/Staging/Production selector.

### 2.3 Environment model

Use this as the target environment model:

| Environment        | Compute                                            | Database                                                |
| ------------------ | -------------------------------------------------- | ------------------------------------------------------- |
| Local development  | Desktop Studio, local coding agent, local services | Local PostgreSQL / development database                 |
| Remote development | Dedicated Fly.io Sprite                            | Workspace development database / controlled DB services |
| Preview            | Temporary Fly Machine or suitable preview runtime  | Ephemeral Neon branch per candidate or PR               |
| Staging            | Persistent Fly Machine                             | Persistent isolated Neon staging branch                 |
| Production         | Dedicated Fly.io Machines                          | Protected production Neon branch                        |

Important distinctions:

- Fly Machines are application runtime infrastructure.
- Sprites are remote development execution environments.
- Do not treat a Fly Machine as a Sprite.
- Preview is normally temporary and isolated.
- Staging is persistent and shared pre-production.
- Production is protected and requires explicit human release approval.
- Do not promote Preview database records into Production.
- Database schema migration and database data promotion are separate concerns.
- Preview must use sanitized or approved seed data.
- Staging and Production must not share mutable database state.

The plan may propose alternatives where repository evidence justifies them, but must explain the tradeoffs and preserve these safety boundaries.

### 2.4 Local-to-remote continuity

A developer must be able to hand off work from Local Desktop to Remote Builder Sprite while preserving the logical project and development journey.

The handoff design must account for:

- Project identity
- Source files
- Staged, unstaged, and untracked Git changes
- Current branch and relevant commit state
- Conversation and turn history
- Clarification questions and answers
- Agent/provider/session context
- Task state and checkpoints
- Environment configuration
- Test and verification results
- Artifact and evidence references
- Handoff receipts and synchronization state

Do not promise that a live OS process or every provider-native agent session can migrate between machines.

Use durable journals, checkpoints, workspace snapshots, and provider session resumption where supported. If native session resumption is unsupported, reconstruct the agent context from persisted state and make this visible to the user.

There must be one active execution owner at a time. Prevent split-brain local and remote modifications. Make synchronized, unsynchronized, and unsafe handoff states explicit.

### 2.5 AI providers and coding agents

Keep agent runtime and model provider separate concepts.

Initial agent-runtime targets include OpenCode, Codex CLI, Claude Code, Antigravity CLI, and Meta Muse Code. Model providers may include OpenAI, Anthropic, Google, Meta Muse, and other compatible providers.

Do not assume every agent runtime can use every model provider.

The long-term architecture should support platform allowlists, workspace defaults, authorized per-build overrides, encrypted server-side credentials, immutable execution configuration snapshots, and no silent provider fallback.

For the hackathon/demo-critical path, use the existing local Muse workflow where appropriate. Do not build an entire multi-agent/model registry merely to demonstrate the lifecycle.

### 2.6 GitHub and CI

The long-term separation of responsibilities is:

- GitHub: source control, branches, commits, pull requests, reviews, and collaboration.
- GitHub Actions: CI checks, tests, builds, security checks, and artifact production.
- Miniapp Studio: lifecycle orchestration, artifact verification, human approval, environment management, and release control.

The architecture should support:

- Repository connection
- Branch and Git-change visibility
- Pull-request visibility
- GitHub Actions status, jobs, and logs
- Required-check policies
- Commit-to-build-to-artifact provenance
- Preview environments tied to exact commits and artifacts
- Staging and Production consuming the same immutable artifact
- Short-lived credentials where supported and minimum GitHub App permissions

Passing CI must not automatically mean Studio has verified or approved the artifact.

Full GitHub integration is a product roadmap concern. Implement only the subset required for a credible demo when justified.

---

## 3. Preserve the existing Focus Console design

The Focus Console is the existing UI foundation. Extend it; do not replace it with a conventional IDE, dashboard, or generic workflow canvas.

Inspect the canonical UI references and existing implementation before proposing changes:

- `plans/plan_ui.html`
- `plans/plan_ui_z.md`
- Existing Focus Console templates, handlers, JavaScript, styles, and tests
- Relevant architecture and phase evidence documents

The existing seven-stage spine is binding:

**Describe → Spec → Validate → Connect → Deploy → Run → Evidence**

Do not add Local, Remote, Preview, Staging, or Production as additional stages. Those are execution or runtime contexts, not lifecycle stages.

The existing design has four major zones plus the Line composer:

1. **Left rail:** Gallery, Connectors, Evidence, Settings, Pulse. Retain the 64px minimal rail and the OPS/DEV density switch.
2. **Stage spine:** exactly seven stages, with evidence-based status.
3. **Middle scene:** one contextual scene/object at a time.
4. **Right Pulse:** chronological events, receipts, approvals, alerts, and clarification cards.
5. **Bottom Line:** the sole natural-language composer. Submissions become Pulse events, not an expanding chat transcript.

Stage colors and meanings remain:

- Gray: untouched
- Amber: active
- Green: proven through inspectable persisted evidence
- Red: refused or failed

Navigation does not grant authorization. OPS/DEV changes information density over the same underlying state and workflow. DEV may expose identifiers, digests, and evidence, but never secrets.

Retain responsive behavior, accessibility, keyboard support, live announcements, reduced-motion support, and honest loading, empty, stale, error, offline, and reconnect states.

Do not claim a stage is green merely because an agent says it succeeded. It must be supported by durable, inspectable evidence.

---

## 4. Target UI capabilities

Investigate the current implementation and plan the following cohesive additions.

### 4.1 Persistent project context bar

Design a compact project context area above the existing stage spine.

It should make the following visible where applicable:

- Current project and repository
- Current branch, pull request, and commit
- GitHub Actions/CI state
- Active coding agent and model
- Workspace execution location: Local Desktop or Remote Sprite
- Workspace synchronization and handoff status
- Application runtime environment: Local, Preview, Staging, or Production

The workspace selector and application environment selector must remain independent.

Do not overload the context bar. Plan progressive disclosure for detailed information.

### 4.2 Contextual Developer Workbench

Plan an on-demand Developer Workbench, not another permanently visible navigation rail.

Potential capabilities:

- Source and file browser
- Git changes, branches, commits, and pull requests
- Integrated terminal
- Agent tasks and sessions
- Test and build output
- Workspace status
- Environment configuration visibility, without exposing secrets

It should support the existing Focus Console workflow rather than become a competing application inside the application.

Determine which capabilities should be embedded, opened as a panel, or delegated to existing tools. Avoid duplicating functionality unnecessarily.

### 4.3 Workspace continuity and handoff

Plan the complete Local Desktop ↔ Remote Builder Sprite workflow.

Include:

- Handoff initiation and explicit user intent
- Preflight checks
- Dirty-worktree handling
- Snapshot and transfer strategy
- Conversation and task checkpointing
- Agent session resume or reconstruction
- Single execution-owner enforcement
- Handoff receipt
- Recovery after interruption
- Conflict handling
- Return from remote to local
- Visibility into synchronization and stale state

A handoff must not silently discard uncommitted changes or create two active writers.

### 4.4 Deploy scene: environment manager

Develop the existing Deploy placeholder into an environment-management scene.

Each environment should expose relevant, persisted facts, including:

- Environment identity and type
- Runtime URL
- Health and deployment state
- Git commit and artifact digest
- Fly Machine identity
- Neon branch identity
- Migration status
- Last successful release
- Deployment and rollback history

Plan the user actions and their authorization boundaries:

- Create or refresh Preview
- Promote to Staging
- Request/approve Production release
- Roll back a release

Actions must not occur simply because the user navigates to a stage or environment card.

Production approval must be tied to the exact immutable artifact digest and a distinct human authorization.

### 4.5 Run scene: Operations workspace

Develop the Run scene as the operational view of an application environment.

Plan these views:

- Overview
- Logs
- Metrics
- Traces

**Logs are the first fully functional observability capability.**

Metrics and traces may be represented in the UI architecture, but must be visibly unavailable until real data collection and ingestion exist. Never fabricate operational telemetry.

Logs should support useful filters such as:

- Environment
- Deployment
- Service
- Severity
- Request or correlation ID
- Time range

Plan collection, ingestion, storage, indexing, querying, live streaming, reconnection, retention, and secret redaction. Distinguish operational telemetry from durable business audit, approval, and artifact evidence.

### 4.6 Pulse and Line

Pulse must become the chronological record of meaningful workflow events, including:

- Agent activity
- Clarification questions and answers
- Workspace handoffs
- Git changes and GitHub CI
- Artifact verification
- Neon branch lifecycle
- Deployment and release events
- Logs and operational alerts
- Approvals and refusals

Line should accept context-aware natural-language commands and display resulting activity through Pulse.

Plan explicit confirmation and authorization for consequential actions. An agent must never self-authorize a release or gain access to secrets through a natural-language request.

### 4.7 Settings

Plan a coherent settings structure, including:

- AI and Agents
- Workspaces
- Environments
- Infrastructure (Fly.io and Neon)
- Observability
- Security

Avoid creating settings that are not backed by an actual capability or a justified roadmap requirement.

### 4.8 Integrated troubleshooting workflow

The product should eventually support a complete troubleshooting loop:

1. Select a production or staging error.
2. Inspect correlated logs, request details, deployment, and source commit.
3. Create a sanitized evidence bundle.
4. Ask the coding agent to investigate.
5. Reproduce and fix the issue in Local or Remote development.
6. Run tests and GitHub Actions CI.
7. Deploy the exact candidate to Preview with an isolated Neon branch.
8. Verify the change.
9. Promote the same immutable artifact to Staging.
10. Obtain human approval for Production release.
11. Deploy and inspect the result.

Secrets and sensitive production data must not be sent to the coding agent. The plan must identify which steps are demo-ready, which are existing capabilities, and which require future work.

---

## 5. Observability must be planned as two coordinated workstreams

Do not treat observability as a single UI-only task.

Plan these workstreams concurrently:

**Workstream A — Miniapp framework/runtime**

- Standardized structured logs, initially using Go `log/slog` as the base.
- Timestamp, severity, application, environment, deployment, request/correlation identifiers, and contextual fields.
- Secret scrubbing and safe serialization.
- Extensible output and export mechanisms.
- A path toward OpenTelemetry compatibility.

**Workstream B — Miniapp Studio**

- Collection and ingestion.
- Durable or appropriately bounded telemetry storage.
- Indexing and query.
- Live streaming and reconnection.
- Filtering and navigation.
- Retention policies.
- UI integration across Local, Sprite, Fly Machine, and application environments.

Explain the dependency between the two workstreams. The framework must emit useful structured events while Studio makes them available and actionable.

Metrics and traces must be planned honestly, with explicit readiness gates.

---

## 6. Phased implementation planning

Do not simply divide the work into arbitrary UI features. Sequence phases by architectural dependencies, risk, and demonstrable user value.

Start with a **Phase 0: Discovery and architecture reconciliation**. Then propose the remaining phases.

At a minimum, evaluate and sequence these capabilities:

1. Project and environment context in the existing Focus Console.
2. Contextual Developer Workbench.
3. Local/Remote workspace continuity and handoff.
4. GitHub repository, Git, PR, and Actions integration.
5. Deploy scene and environment manager.
6. Neon branch lifecycle and environment isolation.
7. Miniapp structured logging and Studio log ingestion/UI.
8. Preview-to-Staging-to-Production artifact promotion.
9. Metrics and tracing.
10. Integrated end-to-end lifecycle and troubleshooting rehearsal.

You may split, merge, or reorder these phases if repository evidence or dependencies justify doing so. Explain every material sequencing decision.

For every phase, include all of the following:

- Phase name and objective
- User-visible outcome
- Scope
- Explicit exclusions and non-goals
- Existing capabilities that can be reused
- Codebase and architectural areas affected
- Data model/API/protocol implications
- Dependencies and prerequisites
- Security and authorization boundaries
- Failure modes and recovery behavior
- Tests and verification strategy
- Acceptance criteria
- Evidence required to declare completion
- Risks and unresolved questions
- Estimated relative complexity: S/M/L/XL
- Blocking dependencies and exit criteria

Use independently verifiable acceptance criteria. Avoid vague phrases such as "works correctly", "fully integrated", or "production-ready" without defining what proves them.

### Demo-critical path

Identify a narrow, realistic sequence of phases that demonstrates:

1. Miniapp Studio running locally.
2. A local coding agent working on the Studio repository.
3. A handoff to a Remote Builder Sprite with durable development continuity.
4. A Preview deployment of Miniapp Studio to a Fly Machine using a dedicated Neon branch.
5. Real log inspection and a credible agent-assisted troubleshooting cycle.
6. Staging and Production promotion only to the extent that their authorization, provenance, isolation, and rollback boundaries can be demonstrated reliably.

Production promotion is desirable but must not compromise demo integrity or lead to claims unsupported by evidence.

Clearly distinguish:

- Implemented
- Implemented but not integrated
- Demonstrated with evidence
- Planned
- Blocked by environment or permissions
- Not yet tested

Do not invent demo evidence.

---

## 7. Current implementation and evidence must be verified

Before designing phases, inspect the actual repository and relevant documentation.

At minimum:

- Check `git status` and preserve all existing modifications.
- Read the canonical Focus Console references.
- Inspect the existing stage scenes, templates, handlers, JavaScript, styles, and related tests.
- Inspect relevant architecture decisions and implementation plans.
- Inspect the current Phase A–D implementation/evidence documents.
- Determine precisely what is implemented, what is partial, what is a placeholder, and what is only planned.
- Investigate existing data models, APIs, Gateway protocols, artifact contracts, environment abstractions, and security boundaries relevant to the proposed work.

The latest Phase D evidence is important:

- D0–D5b have implementation evidence and bounded residuals.
- The Deploy scene remains a placeholder in the current Focus Console.
- The D6 browser rehearsal was aborted and produced no acceptance evidence.
- Do not cite the aborted D6 rehearsal as a successful demonstration.
- Do not assume that the prior Invoice Review Desk demo remains the target.
- The current dogfooding demo target is Miniapp Studio itself.

Check whether newer repository evidence supersedes these notes. The repository and its current evidence are authoritative.

Do not infer implementation from a plan, TODO, mockup, or agent assertion. Trace the actual code and tests.

If the current implementation contradicts a binding architectural principle, document the discrepancy and propose a durable resolution. Do not silently redesign the architecture to accommodate an accidental implementation constraint.

---

## 8. Engineering and security standards

The plan must explicitly consider:

- Tenant isolation and authorization enforcement
- Durable state and replay/recovery
- Idempotency and concurrency
- One active workspace execution owner
- Handoff interruption and recovery
- Artifact immutability and provenance
- Human approval and digest binding
- Secret isolation and redaction
- Runtime isolation from Builder and agent execution
- Neon branch isolation and lifecycle cleanup
- Fly Machine deployment and rollback behavior
- Log ingestion failure and backpressure
- SSE/live-stream reconnect and stale data
- GitHub credential scope and token expiry
- Network and process boundaries
- Failure recovery after crashes, restarts, and partial operations

For each important invariant, specify where it must be enforced: database, server, Gateway, Builder, runtime, deployment boundary, or UI.

The UI is never the security boundary. Hiding a button is not authorization.

Implementation phases must require root-cause investigation rather than symptom-level patches. Where a bug or invariant is addressed, require regression tests that demonstrate the invariant, including relevant concurrency, crash, retry, and recovery cases.

---

## 9. Required document structure

Write the plan with this structure, adapting headings only where necessary:

1. Executive Summary
2. Product Goal and Design Principles
3. Current-State Repository Assessment
4. Existing Capabilities vs. Gaps
5. Target User Journey
6. Target UI Architecture
7. Workspace and Environment Model
8. Target System Architecture and Data Flows
9. Security and Authorization Model
10. Phased Implementation Roadmap
11. Hackathon/Demo-Critical Path
12. End-to-End Acceptance and Rehearsal Plan
13. Risks, Dependencies, and Open Decisions
14. Explicit Non-Goals
15. Definition of Done

Include useful tables and Mermaid diagrams where they clarify relationships or dependencies. Do not add diagrams merely for decoration.

Provide a high-level dependency graph across phases, and a separate critical-path sequence for the demo.

---

## 10. Quality bar and operating constraints

The plan must be actionable by another coding agent without requiring this agent to make architectural decisions again.

Be concrete about:

- Which capabilities already exist
- What must change
- Where the changes belong
- Which invariants must hold
- What depends on what
- How each phase will be tested
- What evidence is necessary for acceptance
- What must remain out of scope

Do not overengineer. Prefer the smallest architecture that preserves the long-term product direction and allows a credible demonstration.

Do not create a broad abstraction merely because a future provider may exist. Introduce provider abstractions where they solve a real boundary or prevent a costly architectural lock-in.

Do not invent API contracts, database tables, existing files, completed features, test results, or infrastructure capabilities. Mark proposals as proposals and unknowns as unknowns.

### Strict execution boundaries

This assignment authorizes **repository inspection and creation or revision of the planning document only**.

- Do not implement any phase.
- Do not modify application source code, tests, migrations, runtime, Gateway, deployment configuration, or UI.
- Do not start or stop Studio, Gateway, Builder, coding agents, or other long-running processes.
- Do not create, change, or destroy Fly.io resources, Sprites, Machines, Neon databases/branches, or GitHub resources.
- Do not run destructive database operations.
- Do not commit, push, or alter branches.
- Do not overwrite unrelated owner changes or uncommitted work.
- Do not treat this prompt as authorization for D6, a cloud rehearsal, or any live operation.

Read-only tests or inspection commands may be used only when safe and necessary; do not change persistent state as a side effect.

At completion, report:

1. The exact planning document path.
2. A short summary of the proposed phases and their dependencies.
3. The recommended demo-critical path.
4. The most important architectural risks and unresolved decisions.
5. The repository sources and evidence used.
6. Confirmation that no implementation, cloud operation, commit, or push was performed.

**Stop after delivering the plan. Wait for explicit architectural review and implementation authorization before doing any implementation work.**

The key requirement is that the agent must produce **one coherent, dependency-ordered roadmap**, while clearly separating the hackathon demonstration path from the full product roadmap.


## prompt

we have a chicken and egg problem here, I want Miniapp Studio to drive my development process but we are still developing it and the final Postgres schema to support this process, my goal is to build the absolute minimal Miniapp Studio to support this process, it need not be perfect in coverage but that is the goal, how do you plan to address this


## question

I would address this by changing the strategy from **"finish Miniapp Studio, then use it to develop Miniapp Studio"** to **"build just enough Miniapp Studio to manage its own development, then let actual development drive the next increments."**

You're right: the previous implementation-plan prompt risks producing too much platform before we have a useful platform. It describes the destination well, but it needs a much sharper bootstrap strategy.

The solution is to build a **minimum viable development control plane**, not a miniature version of the entire final product.

## **1. Establish the first usable development loop**

Miniapp Studio does not need to automate every step initially. It needs to own the process, preserve its state, and make progress verifiable.

I would define the first useful loop as:

1. **Describe the development task in Studio**

   Define the objective, scope, acceptance criteria, and relevant project context.
2. **Studio prepares the execution**

   Assemble the agent instructions, repository/branch context, relevant evidence, and workspace identity.
3. **A coding agent performs the work**

   Initially use the available local Muse workflow. Studio does not need a complete agent-management platform yet.
4. **Return the result to Studio**

   Record the changes, Git state, test results, agent report, and any artifacts produced.
5. **Studio evaluates the evidence**

   Tests, verification results, and acceptance evidence—not just the agent's assertion—determine whether the work is acceptable.
6. **Human reviews and decides**

   Accept, reject, or send the task back for another attempt.
7. **Studio advances the development lifecycle**

   Eventually build an immutable candidate, deploy it to Preview, inspect its logs, and use the findings to create the next development task.

This gives us something valuable immediately: Miniapp Studio becomes the place where development work is defined, tracked, reviewed, and evidenced.

Initially, some execution steps can be manual. That is a deliberate bootstrap bridge, not the intended final architecture.

## **2. Don't wait for the final PostgreSQL schema**

This is the most important architectural decision.

**We should derive the schema from the minimum workflow we actually need to operate, not attempt to design the final platform schema before the platform has been used.**

We already have important foundations from the previous implementation:

- Phase A provides durable conversations, turns, questions, commands, artifacts, approvals, and events.
- Phase C provides the Focus Console UI.
- Phase D provides artifact retrieval, verification, test execution, and human artifact decisions.

Those capabilities should be reused. We should not create a second task engine, a second artifact system, or a replacement approval system simply to support the new workflow.

The first task is to inspect the existing PostgreSQL schema and identify the smallest missing set of relationships.

Conceptually, the development workflow needs these durable concepts:

| Concept           | What it needs to preserve                                          |
| ----------------- | ------------------------------------------------------------------ |
| Project           | Application identity and repository association                    |
| Workspace         | Logical workspace identity and execution location                  |
| Work item         | Development objective and acceptance criteria                      |
| Execution attempt | Which agent performed which attempt, its configuration and outcome |
| Checkpoint        | Recoverable development context and handoff state                  |
| Evidence          | Test results, artifact references, verification and Git provenance |
| Approval          | Human decision tied to the exact object being approved             |
| Deployment        | Environment, artifact digest and deployment receipt                |

These are conceptual responsibilities, not instructions to create eight new tables. Much of this may already be represented by existing tables. The implementation plan must map the actual schema before proposing migrations.

### **The schema evolution rule**

I would establish one strict rule:

> Add a persistent database concept only when the next usable workflow increment requires it.

For example, we do not need a complete deployment-history data model before we can manage development tasks. We need it when we first implement the Preview deployment workflow.

Similarly, we do not need a generic agent registry before using one agent.

PostgreSQL remains the durable source of truth for Studio's process and decisions. Git remains the source of truth for source-code history. Workspace storage holds execution files. Studio records references, identities, digests, state transitions, and evidence.

Use additive, reversible migrations wherever practical. Do not repeatedly redesign the database to anticipate hypothetical future features.

## **3. Build in thin vertical slices, not platform-wide subsystems**

This is how I would sequence the bootstrap.

**Bootstrap 0 — Studio can manage a development task**

First usable milestone

One project, one local workspace, one coding-agent workflow.

Studio can create a task, prepare instructions, preserve context, record an execution attempt, capture test results, and obtain a human decision.

The first execution bridge can be manual: invoke Muse locally and return the results to Studio.

**Bootstrap 1 — Studio can manage its own local development**

Connect the available local agent workflow sufficiently for Studio to coordinate task execution, record progress, resume from checkpoints, and associate changes with the right Git state.

Do not build a multi-agent framework. Start with one proven execution path.

**Bootstrap 2 — Studio can hand off development remotely**

Introduce the Remote Builder Sprite.

Transfer the workspace snapshot, uncommitted changes, task context, conversation/checkpoints, and evidence references. Enforce a single active execution owner and issue a durable handoff receipt.

Provider-native session migration is optional; reconstructing the session from durable context is an acceptable fallback.

**Bootstrap 3 — Studio can deploy a Preview of itself**

Introduce one working Preview path: verified immutable artifact, Fly Machine, isolated Neon branch, deployment receipt, and a usable application URL.

Do not build the complete environment manager before this one path works.

**Bootstrap 4 — Studio can troubleshoot itself**

Implement structured logging in the Miniapp runtime and real log collection and presentation in Studio.

Select an error, inspect its context, prepare sanitized evidence, ask the agent to investigate, and bring the resulting fix back through the same development loop.

**Bootstrap 5 — Studio can manage releases**

Extend the proven Preview workflow to persistent Staging and protected Production.

Reuse the same immutable artifact for promotion. Add human approval, release provenance, health verification, and rollback.

These are capability milestones, not necessarily six large releases. Some may be implemented together when their dependencies permit it.

## **4. What I would deliberately postpone**

The minimum viable Studio should not attempt to solve every future problem.

| Capability                              | Initial treatment          |
| --------------------------------------- | -------------------------- |
| One coding agent integration            | Build                      |
| Multi-agent/model registry              | Defer                      |
| Local development workflow              | Build                      |
| Remote Sprite handoff                   | Build after local loop     |
| Git branch, diff and commit awareness   | Build minimally            |
| Full GitHub PR management               | Defer                      |
| Basic test and artifact evidence        | Build                      |
| Verified Preview deployment             | Build                      |
| Staging and Production                  | Later milestone            |
| Structured application logs             | Build with troubleshooting |
| Full metrics and traces                 | Defer                      |
| Generic deployment-provider marketplace | Defer                      |
| Full infrastructure management console  | Defer                      |
| Broad workflow canvas                   | Do not build               |

The distinction is that a capability can be part of the long-term architecture without being part of the first useful implementation.

For example, the architecture should not prevent other agents from being supported later. But the initial implementation only needs one working agent path.

## **5. The critical architectural compromise: Studio-led, initially human-assisted**

There is one thing I would not compromise: Studio must become the durable control plane from the beginning.

However, I would allow execution to remain partially manual while its integrations are being built.

For example:

- Studio creates the task and produces the execution packet.
- You launch Muse in the local workspace.
- Studio records the execution attempt and its associated context.
- You return the agent's result and test evidence.
- Studio records the decision and next action.

Then we replace individual manual steps with automation without changing the underlying workflow or losing its history.

This means we do not need to finish the Gateway, Sprite orchestration, all agent providers, or the final environment manager before Studio becomes useful for its own development.

The important distinction is:

**Manual execution is acceptable temporarily. Manual loss of development state is not.**

## **6. What this means for the implementation-plan prompt**

I would revise the previous prompt's central instruction. Instead of asking the coding agent to plan the entire unified lifecycle platform as though every subsystem must exist before the first release, require it to answer these questions in order:

1. What is the smallest end-to-end development workflow that Miniapp Studio can support using the existing code?
2. Which existing Phase A–D capabilities can be reused without modification?
3. What is the smallest PostgreSQL schema delta needed for that workflow?
4. What can initially use a human-assisted bridge rather than a new integration?
5. What is the first useful milestone at which we can genuinely develop Miniapp Studio through Miniapp Studio?
6. What capabilities must come next to support remote handoff, Preview, and troubleshooting?
7. Which long-term capabilities can safely remain deferred?

The agent should not make a speculative final schema design its starting point. It should produce a **minimum viable schema contract for each vertical slice**, with dependencies and migration requirements.

## **My architectural recommendation**

I would make this the governing principle:

**Miniapp Studio must be useful before it is complete, and every subsequent development increment must make Miniapp Studio more capable of developing itself.**

The first milestone should not be a polished IDE, a complete cloud deployment platform, or even a full agent orchestration engine.

It should be a working Studio that can reliably manage one real development task on its own repository, preserve its history, collect evidence, and record a human decision.

Then we expand that same workflow to remote execution, Preview, and troubleshooting.

That is how we break the chicken-and-egg problem: **we build the control plane just far enough ahead of the development process that it can lead the next increment, without requiring it to be finished before it can be used.**&#x20;


## prompt

one more thing, I already have fly and neon CLIs installed so the coding agent must be able to use it to drive our use of preview, staging and prod environments and databases in Fly and Neon as part of the process, this is absolutely essential, do you have it in the prompt as well so we are both aligned along with the coding agents locally


## answer

**Partially, but not strongly enough.** The previous prompt included Fly.io and Neon in the target architecture, but it did not explicitly require the coding agent to use your locally installed CLIs as the actual operational interface. It also prohibited cloud operations during the planning task.

We should make this a binding requirement. The CLIs are not optional conveniences; they are part of how we will build, operate, and demonstrate Miniapp Studio.

Add the following section to the previous prompt.

## Binding Requirement: CLI-Driven Fly.io and Neon Infrastructure Operations

This requirement is essential to the Miniapp Studio development workflow and must be reflected throughout the architecture, phased implementation plan, and acceptance criteria.

### 1. Existing local CLI tools are first-class infrastructure interfaces

The development environment already has the Fly.io and Neon command-line interfaces installed.

The coding agent must be able to use these locally installed tools to inspect, provision, configure, deploy, verify, troubleshoot, and manage the application's infrastructure across:

- Preview
- Staging
- Production

Local development must also be able to inspect and manage its relevant development infrastructure when necessary.

Do not assume the exact Neon executable name, CLI version, command syntax, or available features. Discover the installed executable, version, authentication state, and supported commands through safe inspection.

At minimum, investigate:

- Fly.io CLI: executable, version, authentication status, organization, applications, Machines, deployment status, logs, secrets/configuration interfaces, and supported lifecycle commands.
- Neon CLI: executable, version, authentication status, projects, branches, databases, roles, connection management, branch lifecycle, and supported lifecycle commands.

Use the installed CLIs and their actual help output as the basis for the implementation plan. Do not invent flags, commands, or capabilities.

Never print, expose, copy into agent context, or persist raw credentials, access tokens, connection strings, passwords, or secrets in logs, plans, reports, or execution receipts.

### 2. The CLI must drive actual infrastructure operations

The intended workflow is not merely to display Fly.io and Neon information in Miniapp Studio.

The coding agent must be able to use the CLIs to perform the relevant authorized operations.

Examples include:

**Preview**

- Create or refresh an isolated Neon branch for a candidate or pull request.
- Provision the corresponding Fly Machine/application runtime.
- Configure the appropriate non-production database connectivity without exposing credentials.
- Deploy the exact verified application artifact.
- Inspect deployment health, runtime logs, and database connectivity.
- Remove temporary Preview resources when no longer needed, subject to explicit cleanup policy.

**Staging**

- Maintain a persistent staging Fly Machine environment.
- Maintain a separate persistent Neon staging branch.
- Deploy the same immutable artifact that passed Preview verification.
- Apply and verify database migrations under controlled procedures.
- Inspect application health, logs, and deployment state.

**Production**

- Maintain protected production Fly Machines.
- Maintain a protected production Neon branch.
- Release only an exact immutable artifact that has passed required verification.
- Require explicit human authorization tied to the artifact digest.
- Verify deployment health and preserve release history.
- Support controlled rollback.
- Never allow an agent to independently authorize a production release or execute destructive production operations.

The implementation plan must identify the actual CLI operations needed for each lifecycle action, including preconditions, expected results, failure behavior, verification, and recovery.

Where exact command syntax is version-dependent, record the discovered version and mark the command contract for validation during implementation.

### 3. Local coding agents must be able to operate these tools

The current local development workflow uses a local coding agent.

The agent must have a documented, tested way to invoke the installed Fly.io and Neon CLIs from its execution environment, subject to explicit permissions and environment-specific policies.

Do not assume that installing a CLI is sufficient. Investigate:

- Executable resolution and PATH.
- CLI authentication availability.
- Whether the agent's process can access the authenticated CLI context.
- Working-directory and shell constraints.
- Credential isolation.
- Command execution and timeout handling.
- Safe capture and redaction of command output.
- How to distinguish read-only inspection from resource-mutating operations.

When development moves to a Remote Builder Sprite, determine how the required CLIs will be installed, authenticated, and operated there.

Do not copy local credential stores or raw credentials into a Sprite. Define a secure, separately authorized authentication mechanism for remote execution.

A local CLI session must not implicitly become unrestricted authority for every environment.

### 4. CLI-first does not mean unrestricted agent authority

Treat infrastructure operations as privileged actions.

Define an explicit operation classification:

| Class                   | Examples                                                                                   | Authorization                                                      |
| ----------------------- | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------------ |
| Read-only               | List apps, inspect Machines, list Neon branches, inspect deployment status, read logs      | Allowed under workspace/environment policy                         |
| Non-production mutation | Create or refresh Preview resources, deploy Preview, create Preview Neon branch            | Explicit environment-scoped permission and auditable execution     |
| Staging mutation        | Deploy staging, migrate staging database, restart staging resources                        | Explicit approval according to staging policy                      |
| Production mutation     | Deploy Production, change protected database state, rollback, delete or recreate resources | Separate human authorization tied to the operation and target      |
| Destructive operation   | Drop database/branch, delete application or Machine, destroy persistent infrastructure     | Deny by default; explicit exceptional authorization and safeguards |

The UI is not the authorization boundary.

Enforcement must exist in the server-side operation layer and the infrastructure execution boundary. Agent prompts or hidden buttons are insufficient.

Production credentials and privileged operations must be separated from ordinary development execution. Prefer short-lived, narrowly scoped credentials where supported. Do not silently fall back to broader credentials or another environment.

### 5. Miniapp Studio must evolve from agent-invoked CLIs to an auditable infrastructure control plane

The initial bootstrap may allow the local coding agent to invoke the installed CLIs directly under human supervision.

However, the architecture must provide a path toward Studio coordinating these operations through a controlled infrastructure execution interface.

Studio should eventually own:

- Environment identity and configuration.
- Fly application and Machine references.
- Neon project and branch references.
- Operation requests and authorization.
- Execution status and failure state.
- Deployment and migration receipts.
- Artifact digest and source commit provenance.
- Resource lifecycle and cleanup history.
- Audit history and environment-specific policy.

The CLI is an operational adapter, not the durable source of truth.

Do not make the browser execute infrastructure commands. Do not make the Studio database dependent on parsing terminal output after the fact.

Use structured operation records and normalized results. Preserve appropriately redacted command evidence for troubleshooting and audit.

Investigate whether a controlled CLI wrapper or adapter should be introduced. It must enforce an allowlisted operation contract, argument validation, environment isolation, timeouts, output limits, credential redaction, and idempotency where possible. Avoid building a generic unrestricted shell-execution service.

### 6. Required phases and acceptance criteria

The phased roadmap must explicitly include:

1. **CLI capability discovery:** verify the actual local Fly.io and Neon CLI installations, versions, authentication context, and supported commands.
2. **Local infrastructure operations:** demonstrate authorized, auditable CLI-driven operations without unnecessarily changing remote resources.
3. **Preview lifecycle:** create an isolated Neon branch and corresponding Fly runtime, deploy a verified artifact, and verify connectivity and health.
4. **Staging lifecycle:** maintain separate persistent resources and promote the same immutable artifact from Preview.
5. **Production lifecycle:** implement protected infrastructure operations, human approval, deployment verification, release history, and rollback.
6. **Remote Builder support:** establish a secure CLI and credential strategy for remote execution without copying local credential stores.
7. **Studio integration:** expose environment state and controlled infrastructure operations through the Focus Console and Pulse.

Each phase must demonstrate actual CLI invocation and verified infrastructure state where cloud operations are authorized. A mock response or a manually entered status must not be represented as real infrastructure integration.

### 7. Current planning-task boundary

For the present assignment, inspect the installed CLIs and document the proposed operational workflows, but do not provision, change, restart, deploy, migrate, or delete any Fly.io or Neon resources.

The present task remains planning-only.

During later implementation, actual non-production cloud operations may be performed only when separately authorized. Staging and Production operations require their own explicit approvals.

Do not treat this planning prompt as permission to perform live cloud operations.

**Acceptance principle:** Miniapp Studio must eventually be capable of driving its own Preview, Staging, and Production infrastructure lifecycle through real Fly.io and Neon CLI operations, with durable evidence, least-privilege execution, environment isolation, and human-controlled production authorization.

## **One architectural distinction I want us to preserve**

There are two stages to this capability:

- **Now:** Muse or another authorized local coding agent invokes the installed CLIs to operate infrastructure under your supervision.
- **Later:** Miniapp Studio takes ownership of coordinating those operations through a controlled infrastructure adapter, with the agent requesting actions and Studio enforcing permissions and recording evidence.

We should not build an unrestricted shell tool and call that infrastructure integration.

The resulting development loop is:

**Miniapp Studio**

Tasks · Evidence · Approvals · Environment state

**Local / Remote Coding Agent**

Development, investigation, authorized CLI invocation

**Fly.io CLI**

Machines · Deployments · Runtime · Logs

**Neon CLI**

Projects · Branches · Databases · Migrations

Actual infrastructure state is verified and recorded back in Studio.

**My recommendation: make this amendment binding before giving the planning prompt to the coding agent.** It ensures the roadmap plans for real infrastructure operations from the beginning, without prematurely authorizing changes to your live environments.