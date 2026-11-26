# Miniapp Studio — Engineering Constitution and Operating Skills

**Document type:** Repository-level engineering constitution  
**Status:** Governing guidance  
**Intended readers:** Project owner, senior architect, coding agents, reviewers, and operators  
**Applies to:** The full Miniapp Studio software lifecycle, from local development through remote execution, CI, Preview, Staging, Production, operations, and recovery  
**Last revised:** 2026-11-26

---

## 1. Purpose and authority

This document establishes one consistent engineering process for Miniapp Studio. It is intended to outlive any hackathon, demo, individual coding agent, model, or infrastructure experiment.

Miniapp Studio is being built to manage its own development and, ultimately, the development and operation of other Miniapp applications. The platform must therefore become useful before it is complete. Every meaningful increment should strengthen the same durable development-to-production workflow rather than create a disconnected demo path that must later be discarded.

### 1.1 Roles and decision rights

**Project Owner**
- Sets business priorities, product goals, budget constraints, and acceptable risk.
- Provides explicit authorization for consequential external actions, especially cloud resource mutations, production changes, destructive operations, and releases.
- Has final authority when product priorities or business trade-offs must be decided.

**Senior Architect / Technical Director**
- Owns the coherent technical direction across architecture, implementation sequence, infrastructure, security boundaries, and lifecycle operations.
- Turns goals into sequenced work, selects the appropriate execution and application environments, identifies dependencies, and defines acceptance evidence.
- Resolves design questions within the approved product direction; records material decisions and trade-offs.
- Directs when to use Local Desktop, Fly Sprites, Fly Machines, Neon branches, CI, Preview, Staging, and Production.
- Stops or escalates work when it conflicts with a binding invariant, lacks required authorization, or risks damaging user data or infrastructure.
- Does not override the Project Owner's authorization boundaries.

**Coding Agent**
- Investigates before changing code, follows this constitution, and executes only the assigned scope.
- Solves underlying problems rather than applying superficial symptom patches.
- Reports actual evidence, test results, limitations, and residual risks honestly.
- Does not self-authorize releases, broaden its own permissions, or infer permission to mutate infrastructure from access to a CLI or credential.

**Reviewer / Acceptance Authority**
- Reviews the implementation and evidence against explicit acceptance criteria.
- Challenges unsupported claims, missing tests, security assumptions, and regressions.
- Does not treat an agent's completion statement as proof.

### 1.2 Instruction precedence

When instructions conflict, use this order:

1. Applicable law, platform safety requirements, and security obligations.
2. The Project Owner's latest explicit instruction and authorization.
3. This Engineering Constitution and binding architecture decisions.
4. The task-specific approved specification or implementation plan.
5. Existing implementation conventions, where they do not conflict with the above.

A task prompt cannot silently waive a higher-level invariant. If an instruction conflicts with a binding rule, stop the conflicting action, explain the conflict, and propose the smallest safe resolution. Do not silently reinterpret the architecture.

### 1.3 Authorization is explicit and scoped

Planning approval is not implementation approval. Implementation approval is not cloud-operation approval. Permission to inspect an environment is not permission to mutate it. Permission to mutate Preview is not permission to mutate Staging or Production.

Before a consequential operation, identify the exact target, operation, expected effect, data impact, and authorization. Production releases and destructive operations require separate explicit human authorization. Never infer approval from urgency, a hackathon deadline, possession of credentials, an agent's recommendation, or a previous approval for another environment.

---

## 2. Product and architecture north star

Miniapp Studio is an AI-native software engineering and operations control plane. It coordinates the lifecycle of Miniapp applications while keeping coding-agent execution, application runtime, durable workflow state, and external infrastructure distinct.

### 2.1 Binding architecture principles

1. **Miniapp Studio is the control plane.** It owns project lifecycle coordination, durable workflow state, evidence, approvals, environment identity, release records, and operational visibility.
2. **Coding agents execute separately.** They are supervised through controlled execution boundaries such as the Gateway and Builder workspace. The browser does not connect directly to Builder processes.
3. **A dedicated Builder workspace exists for each application.** Its execution location may be Local Desktop or a Remote Fly Sprite. Runtime applications are separate from Builder workspaces.
4. **Miniapp is the exclusive application framework/runtime.**
5. **Go is the exclusive application implementation language.**
6. **MiniappSpec is the canonical application specification.** Requirements are enforced at validation, admission, build, and deployment boundaries—not only through agent instructions.
7. **PostgreSQL is the durable source of truth for Studio workflow state.** Conversations, turns, questions, answers, commands, attempts, approvals, artifacts, builds, deployments, events, and history must survive process restarts.
8. **Execution state is not durable control-plane truth.** Filesystems, terminals, running processes, Sprite disks, and agent-native sessions are execution resources. Studio must retain the durable identities, checkpoints, state transitions, receipts, and references needed to recover the workflow.
9. **Runtime consumes an exact approved artifact.** A deployed application must not depend on a mutable Builder workspace, a coding-agent process, the Gateway process, or development credentials.
10. **Human authority is distinct from agent execution.** An agent may prepare, test, recommend, and request approval; it cannot authorize its own consequential action.
11. **G-01 remains binding:** use synthetic data and simulated approval effects for the constrained demo/workflow; do not perform consequential external business writes unless a later, explicit product decision changes this boundary.
12. **Provider neutrality is an architectural direction, not a reason to overbuild early.** Do not bind the product permanently to one agent runtime, model provider, database provider, or deployment provider. Implement only the abstractions justified by a real boundary or near-term use case.

### 2.2 Keep four concepts separate

Never collapse these concepts into one selector or one process:

- **Project identity:** the logical application being developed.
- **Workspace execution location:** Local Desktop or Remote Builder Sprite.
- **Application runtime environment:** Local, Preview, Staging, or Production.
- **Agent runtime and model provider:** the tool that executes coding tasks and the model service it uses.

For example, a developer may use Muse on a remote Sprite to modify a project and deploy the resulting candidate to Preview. The workspace location and application destination are independent.

### 2.3 Sources of truth

| Information | Authoritative source |
|---|---|
| Application source and source history | Git repository and commit history |
| Product/workflow state, questions, approvals, events, and receipts | Miniapp Studio PostgreSQL database |
| Specification | Versioned MiniappSpec and its verified digest |
| Build and deployment candidate | Immutable artifact and its provenance/digest |
| Actual infrastructure state | Fly.io and Neon APIs/CLIs, reconciled into Studio |
| Runtime telemetry | Actual application/runtime log and telemetry sources |
| Human authorization | Durable Studio approval record tied to the exact action/target/digest |
| Agent execution context | Durable Studio task journal/checkpoints plus workspace state; provider-native session where supported |

Do not treat a README, plan, dashboard cache, agent assertion, or old CLI output as authoritative proof of current infrastructure state.

---

## 3. The unified execution and application environment model

Miniapp Studio has two independent axes: where development work executes and where the application runs.

### 3.1 Workspace execution locations

**Local Desktop**
- Use for rapid iteration, source editing, local unit tests, focused integration tests, and low-cost development.
- Use the local PostgreSQL development database for local application and Studio tests.
- Use the installed local coding agent when it is the fastest and safest option.
- Local use of Fly and Neon CLIs is allowed only under the operation and authorization policy in this document.
- Do not assume local CLI credentials are automatically safe to expose to an agent or pass into a remote workspace.

**Remote Builder Sprite**
- Use when the work benefits from a persistent, isolated, remotely accessible Linux workspace; remote agent execution; reproducible dependencies; long-running development tasks; integration testing; or access to services that are awkward to reach from a developer's desktop.
- The default long-term model is a dedicated Builder Sprite/workspace identity per application, with separate task-scoped or disposable execution environments where additional isolation is justified.
- Sprites are the preferred place to execute generated or untrusted code, builds, and integration tests rather than running that code inside the Studio control-plane process or the agent host.
- Use checkpoints before risky refactors, dependency upgrades, database experiments, or other changes where restoring the workspace is valuable.
- Use Sprite URLs for temporary inspection of a running development process when useful. A Sprite URL is a workspace/service preview; it is not automatically a release-grade application Preview deployment.
- Use explicit network-egress policies and narrowly scoped connectors/credentials. Do not assume a new Sprite is safely restricted by default; inspect and configure its actual network and credential policy.
- Keep a durable task journal in Studio. A Sprite checkpoint is a recovery aid for workspace state, not a replacement for Git, Studio's durable workflow records, or a guaranteed migration of an agent's live process/session.

**Local-to-remote handoff**
- Preserve logical project/workspace identity, branch and commit, staged/unstaged/untracked changes, source snapshot, relevant conversation and clarification history, agent/provider configuration, task journal, checkpoints, environment references, test results, artifact/evidence references, and a durable handoff receipt.
- Enforce one active writer/execution owner at a time. Never allow Local Desktop and Remote Sprite to concurrently modify the same logical working state without an explicit, safe coordination mechanism.
- Before handoff, inspect dirty Git state and report what will transfer. Never silently discard uncommitted work.
- Resume the provider-native session only when supported and verified. Otherwise reconstruct context from Studio's durable journal and a workspace snapshot. Never promise that a live process or in-memory model context can migrate.
- On return to Local, reconcile the remote workspace and Git state before reactivating local writes.

### 3.2 Fly Sprites and Fly Machines are different products in the lifecycle

| Dimension | Fly Sprite | Fly Machine |
|---|---|---|
| Primary purpose | Agent development and isolated code execution | Running application or long-lived service |
| Typical lifecycle | Persistent workspace that can sleep/wake; checkpoints and restore | Explicitly managed runtime/deployment lifecycle |
| Main contents | Repository, tools, dependencies, build/test processes, temporary services | Immutable application release and runtime configuration |
| Main user | Coding agent / developer | Application users and operators |
| Best use | Build, experiment, execute, test, inspect, recover workspace | Serve Local-like remote, Preview, Staging, or Production application deployments |
| Must not be confused with | A production application environment | A remote Builder workspace |

**Use Sprites when the problem is where and how an agent works or where code executes safely. Use Machines when the problem is where the application runs for its intended users or testers.**

Do not use a production Fly Machine as a substitute for a Builder Sprite. Do not treat a Sprite as the production release target. A Sprite-hosted development URL may help demonstrate an in-progress feature, but acceptance of a release-grade Preview requires the controlled artifact and database workflow described below.

The long-term architecture may use other Fly Machines for Studio's own long-running control-plane services if justified; that does not change the Builder Sprite versus application-runtime distinction.

### 3.3 Application runtime environments

| Environment | Purpose | Compute | Database | Lifecycle |
|---|---|---|---|---|
| Local | Fast development and local verification | Desktop/local services | Local PostgreSQL | Developer-controlled |
| Preview | Validate a specific candidate, branch, or PR | Temporary Fly application/Machine(s) | Isolated Neon branch, normally per PR/candidate | Created and cleaned up by policy |
| Staging | Validate a complete release candidate under production-like configuration | Persistent Fly application/Machine(s) | Persistent isolated Neon staging branch | Long-lived and controlled |
| Production | Serve real users and real operational data | Protected Fly application/Machine(s) | Protected Neon production branch | Long-lived, highly controlled |

Environment identities, secrets, network access, resources, databases, and permissions must be isolated. A Preview app must never accidentally connect to Staging or Production data. Staging must not share a mutable database with Production.

---

## 4. Fly.io operating doctrine

Fly.io provides two complementary capabilities in this architecture: Sprites for development/execution and Machines for application runtimes.

### 4.1 When to use Fly Sprites

Use a Sprite when one or more of the following is true:

- A coding agent needs a persistent remote development workspace.
- The task needs an isolated Linux environment with reproducible dependencies.
- Generated or untrusted code must run away from the Studio control plane and agent host.
- Builds, tests, migrations against non-production data, or integration services need to run remotely.
- The developer needs a remotely accessible running service for inspection or collaboration.
- A risky change benefits from a workspace checkpoint and restore path.
- Local-to-remote development continuity is needed.

Recommended Sprite practices:
- Keep a stable logical workspace identity per application; record the Sprite identity and lifecycle state in Studio.
- Separate long-lived Builder workspaces from disposable task/test execution where isolation or cleanup demands it.
- Checkpoint before risky operations and record checkpoint identity in Studio.
- Treat checkpoints as workspace recovery points, not source control or durable business records.
- Keep source changes in Git and preserve a durable task/attempt journal in Studio.
- Apply least-privilege access and network egress restrictions. Expose only required services and endpoints.
- Use short-lived or brokered credentials where supported. Prefer connectors or controlled access mechanisms that avoid placing provider secrets inside the Sprite.
- Never give a Sprite Production database credentials or unrestricted Production infrastructure authority for ordinary development work.
- Inspect current Sprite CLI/API/MCP capabilities and supported operations. Do not assume the regular `fly` CLI manages every Sprite capability or invent a `sprite` command, API, or MCP tool.
- Validate checkpoint and restore semantics from current Fly documentation before relying on them for a workflow guarantee. Do not claim that restoring a disk checkpoint resumes an agent's in-memory reasoning or an arbitrary process exactly where it stopped.

### 4.2 When to use Fly Machines

Use Fly Machines to run application deployments and long-running services:
- Preview application candidates.
- Persistent Staging application.
- Production application.
- Other explicitly approved runtime workloads.

Each deployment must be tied to an application identity, environment, source commit, immutable artifact digest, runtime configuration version, and deployment receipt. Design Machine count, health checks, regions, rollout strategy, and availability to match the environment's purpose. Do not assume a single Machine is sufficient for Production simply because it works for a demo.

Application filesystems should be treated as ephemeral unless a specifically designed persistent-storage requirement exists. Store application data in Neon or a deliberate durable storage service, not in a Machine's mutable root filesystem. Do not introduce Fly Volumes as an accidental substitute for the managed PostgreSQL architecture.

### 4.3 Two meanings of “preview”

The product must distinguish:

1. **Workspace preview:** a service running inside a Sprite and exposed through its URL for quick inspection of in-progress work. This is useful during development but is not proof of a verified release.
2. **Application Preview environment:** a controlled Fly Machine deployment of a verified candidate artifact connected to an isolated Neon branch. This is a release-validation environment with recorded provenance and acceptance evidence.

Never label a Sprite URL as a successfully deployed application Preview unless the actual Preview deployment contract has been satisfied.

---

## 5. Neon PostgreSQL operating doctrine

Neon provides PostgreSQL projects and branches for isolated database development and environment lifecycle management. Branches support safe testing, but they do not replace schema migration discipline, access control, backups, or production change management.

### 5.1 Recommended database layout

- **Local:** local PostgreSQL instance, disposable test databases, and isolated test-run databases where needed.
- **Developer/workspace branches:** optional branches for independent development when shared database state would cause interference.
- **Preview:** a temporary Neon branch associated with a Preview candidate, PR, or controlled test lifecycle.
- **Staging:** a persistent branch with its own credentials and data policy.
- **Production:** protected production branch, distinct credentials, and stricter operation policy.

A Neon branch can be created from a designated parent. Choose the parent deliberately. Preview data must be synthetic, sanitized, or otherwise explicitly approved. Do not assume that branching a database makes sensitive Production records safe to expose to a development agent.

### 5.2 Branch lifecycle

**Preview branch**
- Create for a candidate/PR or test cycle when a separate database state is needed.
- Apply the candidate's versioned migrations to that branch.
- Seed with synthetic or approved sanitized data.
- Run integration and end-to-end tests.
- Associate the branch ID, parent, migration version, candidate, and expiration/cleanup policy with the Preview record in Studio.
- Delete or expire it only after confirming it is no longer needed for review, evidence, or debugging.

**Staging branch**
- Keep persistent and isolated from Preview and Production writes.
- Apply the same release migration set that has been validated in Preview.
- Use representative, non-production data unless an approved data-handling policy says otherwise.
- Preserve sufficient state for repeatable integration, release, and operational rehearsals.

**Production branch**
- Protect from ordinary agent and development credentials.
- Permit only explicitly authorized operational identities to perform schema-changing operations.
- Preserve backups/recovery mechanisms and migration history.
- Never reset, reseed, replace, or recreate it as a routine development action.

### 5.3 Database migrations are not database promotion

- Store schema changes as versioned, reviewed migration files in Git.
- Test every migration against an isolated Preview database before Staging.
- Apply the same migration sequence to Staging and then Production under their respective authorization gates.
- Do not copy Preview or Staging data into Production as a deployment mechanism.
- Do not assume that merging or promoting a Neon branch is equivalent to deploying application code or safely applying schema changes.
- Use backward-compatible **expand-and-contract** changes for schema evolution where application versions may overlap during deployments or rollback.
- Keep application artifact promotion and database migration as coordinated but separately verified operations.
- Record migration identity, source commit, target database/branch, execution result, and verification evidence.

### 5.4 Schema evolution for Miniapp Studio itself

Do not attempt to design the final PostgreSQL schema before the workflow has been used. Build the minimum durable schema required by the next usable vertical slice.

Before adding a table or column:
1. Inspect the existing schema and migration history.
2. Determine whether an existing durable concept already represents the information.
3. Identify the workflow invariant the new field/table must support.
4. Add only the minimum persistent concept required now.
5. Define migration, rollback/recovery, authorization, and tests.
6. Record future requirements as deferred rather than speculatively encoding them.

Prefer additive, comprehensible migrations. Never duplicate task, artifact, event, conversation, or approval systems without proving the existing model cannot support the needed workflow.

---

## 6. Local CLI and infrastructure-operation policy

The installed Fly.io and Neon CLIs are first-class operational interfaces in the development workflow. Coding agents must be able to use them where authorized. CLI access is not unrestricted infrastructure authority.

### 6.1 Discover actual capabilities; never invent commands

Before relying on a CLI:
- Resolve the executable from the actual environment and inspect its version.
- Read the installed version's help and relevant command help.
- Verify authentication status and account/organization context without printing credentials.
- Determine which commands are read-only, mutating, or destructive.
- Verify that the coding agent's actual process can resolve and invoke the executable.
- Capture only redacted, non-secret command evidence.
- Record version-specific command contracts in the operational documentation.

Do not assume executable names, flags, API support, command syntax, or Sprite management integration. Recheck current vendor documentation and local help because CLIs evolve.

### 6.2 Operation classes

| Class | Examples | Default policy |
|---|---|---|
| Read-only inspection | List apps/Machines/branches, inspect status, health, deployment metadata, and logs | Allowed under workspace and account scope; audit where relevant |
| Preview mutation | Create/update temporary Preview app/Machine, create Preview branch, deploy candidate, clean up Preview | Requires explicit environment-scoped authorization or an approved delegated policy; record operation and receipt |
| Staging mutation | Deploy Staging, run staging migration, restart/scale staging resources | Requires explicit staging permission and auditable operation; human approval according to release policy |
| Production mutation | Deploy Production, change Production configuration, execute Production migration, rollback | Requires separate explicit human approval for the exact operation and target |
| Destructive operation | Drop branch/database, delete persistent app/Machine, destroy resources, reset data | Deny by default; require exceptional, explicit authorization and verified target safeguards |

A CLI invocation must be classified by its effect, not by the command name. A seemingly routine command may be destructive depending on arguments and target.

### 6.3 Credential rules

- Never print or persist raw tokens, passwords, connection strings, private keys, or credential files in prompts, logs, artifacts, screenshots, or reports.
- Never copy a local credential store wholesale into a Sprite.
- Prefer short-lived, narrowly scoped, environment-specific credentials or a secure connector/broker when supported.
- Keep Preview, Staging, and Production credentials separate.
- Do not silently fall back to a broader credential, another account, or another environment when access fails.
- Ensure logs and command output are redacted before they enter Studio or agent context.
- Revoke or rotate credentials through an explicit, controlled process; never rotate a credential merely to make a test pass.

### 6.4 From agent-invoked CLI to Studio-controlled infrastructure adapter

The bootstrap may allow an authorized local coding agent to invoke the installed Fly and Neon CLIs under supervision. The long-term product should coordinate privileged operations through a controlled infrastructure execution interface.

That interface must provide:
- An allowlisted operation contract, not unrestricted shell access.
- Argument and target validation.
- Tenant, project, and environment scoping.
- Authorization and human approval enforcement on the server side.
- Idempotency or duplicate-operation protection where appropriate.
- Timeouts, cancellation, output limits, and secret redaction.
- Durable operation state, status, and receipts.
- Actual-state reconciliation against Fly and Neon.
- Audit history identifying requester, approver, executor, target, artifact, and outcome.

The browser must never execute Fly or Neon commands directly. Hiding a button is not authorization. The server-side operation boundary must enforce permission regardless of which agent or UI initiated the request.

---

## 7. Standard development-to-production lifecycle

This is the default lifecycle for meaningful application changes. A task may use fewer stages when justified, but skipped controls must be explicit and appropriate to the risk.

### Stage 1 — Describe and scope

- State the user/business problem and intended outcome.
- Identify the target application, repository, workspace, and affected environments.
- Define acceptance criteria, non-goals, risks, and data/security implications.
- Determine whether the task is planning, code, schema, infrastructure, deployment, or operations work.

### Stage 2 — Specify

- Update or reference MiniappSpec and relevant architecture decisions.
- Identify API, data model, event, and integration contracts.
- Identify the smallest schema delta and migration strategy, if any.
- Define the test/evidence plan before implementation.

### Stage 3 — Prepare execution

- Select Local Desktop or Remote Builder Sprite based on the task's isolation, resource, persistence, and collaboration needs.
- Select the agent runtime and model provider supported by that environment; do not assume universal compatibility.
- Confirm the single active execution owner.
- Load relevant repository guidance, task history, checkpoints, and environment context.
- Confirm permissions and ensure secrets are not exposed to the agent unnecessarily.

### Stage 4 — Implement and test locally/remotely

- Investigate the existing call path and root cause before changing code.
- Follow existing architecture unless a justified change is approved.
- Add regression tests demonstrating the required invariant.
- Run formatting, static analysis, unit tests, relevant integration tests, and build checks.
- Use a Sprite for isolated execution of generated/untrusted code or where persistent remote development provides real value.
- Record exact commands, versions, outcomes, and limitations without exposing secrets.

### Stage 5 — Source-control and CI verification

- Review staged, unstaged, and untracked changes.
- Inspect diffs for accidental scope expansion, generated files, credentials, and unrelated modifications.
- Commit/branch/PR operations follow the user's authorization and repository policy; never commit or push by default.
- GitHub Actions (or the configured CI) runs required formatting, lint, tests, build, security, and artifact checks.
- A passing CI run is evidence of those checks only. It is not automatically Studio verification, approval, or authorization to release.

### Stage 6 — Build and verify an immutable candidate

- Build the candidate from a known source commit and pinned/recorded toolchain.
- Bind the artifact digest to project, tenant, build, specification, source revision, and relevant toolchain provenance.
- Verify artifact structure, ownership, contents, policy, dependencies, and security requirements.
- Run the approved test suite in an isolated environment.
- Preserve machine-readable results and human-readable evidence.
- Do not run untrusted artifact code on the Studio control-plane host.

### Stage 7 — Preview

- Create or refresh a temporary Fly application runtime for the candidate.
- Create the corresponding isolated Neon branch using an approved, safe parent/data policy.
- Apply candidate migrations to that branch.
- Configure only Preview-specific credentials and services.
- Deploy the exact verified artifact.
- Run smoke, integration, end-to-end, migration, and relevant UX checks.
- Inspect real application health and logs.
- Record commit, artifact digest, Fly resource identity, Neon branch identity, migration status, test results, URL, and cleanup policy.
- Review the running candidate; return failures to development rather than promoting an unverified candidate.

### Stage 8 — Human review and acceptance

- Compare results with acceptance criteria.
- Review test and deployment evidence, logs, known limitations, and security impact.
- Accept, reject, or request changes.
- The agent cannot approve its own artifact or release.
- An approval must identify the exact artifact digest and relevant environment/action.

### Stage 9 — Staging

- Deploy the same immutable artifact digest that passed Preview; do not rebuild a different artifact for Staging.
- Apply the corresponding versioned migration set to the persistent Staging Neon branch.
- Validate configuration, authentication, authorization, integration behavior, startup/shutdown, restart/recovery, and operational health.
- Exercise release and rollback procedures where relevant.
- Use realistic but non-production data under an approved policy.
- Record all evidence and any differences from Preview.

### Stage 10 — Production release

- Require explicit human approval of the exact artifact digest, target, migration plan, and release action.
- Confirm required CI, artifact, Preview, and Staging gates.
- Confirm Production credentials and database target independently; never infer the target from a default context.
- Apply the approved backward-compatible Production migration procedure.
- Deploy the same immutable artifact digest tested in Staging.
- Verify Machines, health checks, runtime connectivity, logs, and key user flows.
- Record deployment, migration, approver, artifact provenance, and verification receipts.
- Keep rollback/recovery options available and do not destroy prior release evidence.

### Stage 11 — Operate and learn

- Inspect actual logs and operational health.
- Correlate runtime incidents with request IDs, deployment, commit, and artifact.
- Create sanitized evidence bundles for agent investigation; never send secrets or uncontrolled Production data to the agent.
- Reproduce and fix issues in Local or Remote development, then repeat the lifecycle at a risk-appropriate level.
- Feed recurring operational findings into product backlog, tests, and architecture decisions.

---

## 8. Environment-specific release policy

### 8.1 Local

Purpose: fast feedback and safe experimentation.

- Run local Studio and application services where practical.
- Use local PostgreSQL and isolated test databases.
- Use synthetic data.
- Do not assume local success proves cloud deployment or production configuration.
- Keep local execution separate from Preview/Staging/Production application state.

### 8.2 Preview

Purpose: verify one candidate or PR in a realistic but isolated environment.

- Temporary runtime and database resources, normally linked to a candidate/PR.
- Per-candidate/per-PR database branch where useful; do not use one shared mutable Preview database for unrelated changes.
- Explicit cleanup/expiry policy and ownership.
- Synthetic or sanitized test data.
- No Production credentials or customer writes.
- Exact artifact and source provenance recorded.
- Preview failure blocks promotion of that candidate until resolved or explicitly waived by the proper authority.

### 8.3 Staging

Purpose: release-candidate rehearsal and integration confidence.

- Persistent, clearly identified Fly runtime and Neon branch.
- Separate credentials and access policy from Preview and Production.
- Controlled deployment and migration process.
- Stable enough for repeatable tests and stakeholder review.
- No uncontrolled experimental agent activity or arbitrary database resets.
- Release promotion must use the same immutable artifact digest approved for Staging.

### 8.4 Production

Purpose: reliable customer-facing operation.

- Protected Fly Machines and protected Neon Production branch.
- Strictly scoped operator identities and credentials.
- Explicit human authorization for deployment, migration, rollback, and destructive changes.
- No development agent direct access to Production data or unrestricted infrastructure mutation.
- No ad hoc schema edits; use reviewed, versioned migrations.
- Preserve logs, release history, audit evidence, backups, and recovery procedures.
- Monitor health after release and respond through a documented incident path.

---

## 9. GitHub, CI, provenance, and release integrity

Use the tools according to their responsibilities:

- **Git:** source history, branches, commits, and local change tracking.
- **GitHub:** repository collaboration, pull requests, reviews, and source governance.
- **GitHub Actions / CI:** formatting, lint, tests, build, security checks, and artifact production.
- **Miniapp Studio:** lifecycle orchestration, evidence, artifact verification, approvals, environment management, release authorization, and operational history.
- **Fly.io:** agent workspaces through Sprites and application runtime through Machines.
- **Neon:** managed PostgreSQL environments and database branching.

Every release candidate should have traceable provenance:

`source commit → specification/version → build/toolchain → artifact digest → verification/test evidence → environment deployment → human decision`

A release must not rely on a mutable branch name or an agent's statement. Promote the same immutable artifact through Preview, Staging, and Production. If a rebuild is required, it is a new candidate with a new digest and must satisfy the required gates again.

Do not put long-lived provider secrets in repository files or workflow logs. Prefer short-lived, narrowly scoped identity mechanisms where supported. Apply least-privilege permissions to GitHub Apps and CI tokens.

---

## 10. Agent runtimes and model providers

Agent runtime and model provider are separate configuration dimensions.

Potential agent runtimes include OpenCode, Codex CLI, Claude Code, Antigravity CLI, and Meta Muse Code. Potential model providers include OpenAI, Anthropic, Google, Meta Muse, and compatible providers. Support is determined by the actual runtime's capabilities; do not assume every combination works.

### Bootstrap policy

- Use the currently available local agent (including Muse where appropriate) for the immediate dogfooding workflow.
- Add the simplest proven remote Sprite execution path when it provides meaningful persistence, isolation, or collaboration value.
- Do not build a full agent/model marketplace or registry before the workflow requires it.
- Preserve a path to multiple agents and providers through clean interfaces and recorded configuration.

### Execution configuration

For each meaningful execution attempt, record an immutable configuration snapshot containing the agent runtime, version, model provider/id where available, reasoning/configuration settings, permission profile, workspace identity, and relevant configuration version. Never record the secret values themselves.

Use explicit permission profiles and no silent provider fallback. If a provider or runtime is unavailable, report the failure and request an approved alternative rather than quietly changing execution behavior.

---

## 11. Miniapp Studio UI and workflow experience

Extend the existing Focus Console. Do not replace it with a conventional IDE/dashboard or a generic workflow canvas.

### 11.1 Seven-stage spine

The canonical stages remain:

**Describe → Spec → Validate → Connect → Deploy → Run → Evidence**

Local, Remote, Preview, Staging, and Production are contexts or destinations—not additional lifecycle stages.

Stage status must be evidence-based:
- Gray: untouched.
- Amber: active or awaiting action.
- Green: proven by inspectable, persisted evidence.
- Red: refused, failed, or blocked.

Navigation never grants authorization. A green state cannot be assigned solely because an agent says a task succeeded.

### 11.2 Focus Console zones

Preserve the existing design contract:
- Minimal left rail for Gallery, Connectors, Evidence, Settings, and Pulse, with the OPS/DEV density switch.
- Seven-stage spine.
- One contextual middle scene at a time.
- Chronological right-side Pulse for events, receipts, approvals, alerts, and clarification cards.
- Bottom Line as the sole natural-language composer; submissions become durable workflow events rather than an ever-growing chat transcript.

OPS/DEV changes information density over the same underlying state. DEV may expose identifiers, digests, and evidence, never secrets. Preserve responsive behavior, accessibility, keyboard support, live announcements, reduced-motion behavior, and honest loading/empty/stale/error/offline/reconnect states.

### 11.3 Context and Workbench

The UI should expose two independent controls:
- Workspace execution: Local Desktop or Remote Builder Sprite, including sync/ownership/handoff state.
- Application environment: Local, Preview, Staging, or Production.

Provide project/repository, branch/PR/commit, CI state, active agent/model, and relevant environment identity without overloading the main Console.

A contextual Developer Workbench may expose source files, Git changes, terminal, agent tasks, test output, and workspace status on demand. It must complement rather than compete with the Focus Console.

### 11.4 Deploy and Run scenes

**Deploy** becomes the environment manager. Show actual environment identity, URL, health, source commit, artifact digest, Fly app/Machine references, Neon project/branch references, migration status, release history, and rollback evidence. Actions require authorization; navigation must not trigger them.

**Run** becomes the operations workspace. Logs are the first fully functional telemetry capability. Metrics and traces may be designed for later, but must visibly remain unavailable until real collection, ingestion, and query paths exist. Never fabricate data.

### 11.5 Pulse and Line

Pulse should record meaningful agent, task, question, handoff, Git/CI, artifact, database branch, infrastructure, deployment, log/alert, approval, and refusal events. Line commands are context-aware requests, not authorization. Consequential actions require explicit confirmation and server-side permission checks.

---

## 12. Observability and operations

Observability is a coordinated framework/runtime and Studio capability.

### 12.1 Miniapp runtime workstream

- Emit structured logs, initially based on Go `log/slog`.
- Include timestamp, severity, application, environment, deployment, request/correlation identifiers, and useful contextual fields.
- Use safe serialization and secret scrubbing.
- Provide extensible output/export mechanisms.
- Keep a path toward OpenTelemetry compatibility.

### 12.2 Studio workstream

- Collect and ingest logs from Local, Sprite, and Fly Machine application runtimes as applicable.
- Store, index, query, filter, and stream logs with reconnect behavior.
- Support environment, deployment, service, severity, request/correlation ID, and time filtering.
- Apply retention and cost controls.
- Correlate telemetry with deployment, source commit, and artifact digest.

Start with real logs. Add metrics and traces only when their instrumentation, ingestion, storage, query, and UI paths are implemented. Label absent capabilities honestly.

Operational telemetry is not the same as durable business audit, human approval history, or artifact evidence. Keep those records separate and preserve their appropriate retention and integrity guarantees.

### 12.3 Troubleshooting loop

The long-term workflow is:

1. Select an actual staging or Production error.
2. Inspect correlated logs, request context, deployment, artifact, and source commit.
3. Produce a sanitized evidence bundle.
4. Ask an authorized agent to investigate in Local or a Builder Sprite.
5. Reproduce and fix the problem with regression tests.
6. Run CI and artifact verification.
7. Deploy to Preview with an isolated Neon branch.
8. Validate, then promote the same artifact to Staging.
9. Obtain explicit Production approval.
10. Deploy and verify; preserve incident and release evidence.

Never send raw credentials, secrets, or unapproved Production data to an agent.

---

## 13. Security, isolation, and authorization invariants

Security must be enforced at the correct system boundary, not by relying on UI visibility or agent instructions.

| Invariant | Required enforcement locations |
|---|---|
| Tenant and project isolation | Database policies, server authorization, Gateway, resource identity mapping |
| One active workspace writer | Durable Studio state, execution coordinator, Builder lease/ownership mechanism |
| Agent cannot approve its own work | Server-side approval policy and durable approval records |
| Production release is human-authorized and digest-bound | Studio server, approval transaction, deployment operation boundary |
| Runtime receives only approved immutable artifact | Artifact admission, deployment boundary, runtime configuration |
| Builder and runtime are separate | Architecture, deployment packaging, credentials, network and process boundaries |
| Secrets do not leak to agent or telemetry | Credential broker, execution environment, redaction, logging, UI, artifact checks |
| Environment database isolation | Neon branch/project identity, environment-specific credentials, server-side mapping |
| Infrastructure operations are scoped | Controlled adapter, server authorization, CLI/API operation policy |
| Generated/untrusted code is isolated | Sprite or equivalent sandbox, process/resource/network controls |
| No arbitrary browser-to-cloud control | Server-side execution and authorization; no direct browser CLI access |
| G-01 synthetic-data/no-consequential-write boundary | Application policy, tools, credentials, server-side checks, tests |

### 13.1 Agent access is not user authorization

An agent may request an operation, but permission must be checked by the trusted server-side execution boundary. A prompt, skill, hidden button, terminal alias, or convention is not a security control.

### 13.2 Network and process boundaries

Keep the control plane, Gateway, Builder/Sprite, and application runtime separate. Use explicit network policies and resource limits for code execution. Deny unnecessary egress and allow only the services required by the task. Review any request to weaken sandbox controls as a security decision, not a convenience patch.

### 13.3 Auditability

For important operations, record who or what requested the action, who approved it, the tenant/project/environment, target resource, artifact/migration identity, execution identity, start/end status, and redacted result. Keep audit/approval records distinct from operational logs.

---

## 14. Bootstrap strategy: make Studio useful before it is complete

The chicken-and-egg problem is resolved by building a minimum viable development control plane and letting real development use drive later capabilities.

Do not build the complete platform before using Studio to develop Studio.

### Bootstrap 0 — Studio manages one development task

Minimum workflow:
- Create a task with objective, scope, and acceptance criteria.
- Prepare an execution packet with repository, branch, and context.
- Use the available local agent through a supervised bridge.
- Record the execution attempt, task progress, Git state, tests, and evidence.
- Obtain a human accept/reject/request-changes decision.

Manual agent invocation and manual result capture are acceptable temporary bridges. Loss of durable task history is not.

### Bootstrap 1 — Studio supports its own local development loop

- Use the local coding agent to modify Miniapp Studio itself.
- Preserve source changes and task context.
- Record tests and evidence.
- Resume interrupted work from durable Studio state.

### Bootstrap 2 — Studio supports remote development with Sprites

- Use the dedicated Remote Builder Sprite for persistent, isolated agent work.
- Implement a safe Local ↔ Remote handoff with single-writer enforcement and recovery receipts.
- Use checkpoints and Git together with Studio's durable task journal.
- Avoid requiring provider-native agent-session migration.

### Bootstrap 3 — Studio deploys itself to Preview

- Build and verify an immutable candidate.
- Provision a temporary Fly Machine application environment and an isolated Neon branch.
- Deploy, test, inspect logs, and clean up through authorized operations.

### Bootstrap 4 — Studio troubleshoots itself

- Implement real structured logging and Studio log ingestion/viewing.
- Select an error, create sanitized evidence, ask the agent to investigate, and return the fix through the same lifecycle.

### Bootstrap 5 — Studio manages releases

- Add persistent Staging and protected Production environments.
- Promote the exact same immutable artifact through release gates.
- Implement human approval, controlled migrations, health verification, release history, and rollback.

These bootstrap milestones may be split or combined where repository dependencies justify it. They are capability outcomes, not permission to skip required safety gates.

### 14.1 Schema grows with the workflow

At each milestone, inspect the existing database and add only the minimum durable schema delta required by the next usable workflow. Reuse existing conversation, event, command, artifact, evidence, and approval concepts wherever possible. Do not invent a second task engine, artifact registry, or approval system without evidence that the current model cannot support the requirement.

---

## 15. Standard task execution protocol for coding agents

Every coding agent working in this repository must follow this protocol.

### Before work

1. Read this `SKILLS.md` and the task-specific instructions.
2. Inspect repository status and preserve existing user modifications.
3. Read relevant architecture decisions, plans, and current implementation/evidence.
4. Trace the existing code path and tests; do not infer implementation from documentation alone.
5. Classify the task: planning, implementation, test, schema, local operations, Sprite operations, Preview, Staging, Production, or destructive operation.
6. State the chosen execution location (Local or Sprite), application target environment (if any), database target, expected infrastructure actions, and authorization requirements.
7. Identify invariants and define verification before implementation.
8. If required authorization is absent, perform only safe inspection/planning and request authorization for the restricted action.

### During work

1. Solve the underlying problem; do not patch only the immediate symptom.
2. Prefer the smallest coherent change that respects the architecture.
3. Add regression tests for the invariant being changed.
4. Include relevant concurrency, retry, crash, restart, and recovery tests for durable or distributed workflows.
5. Keep migrations additive and deliberate; never make unrequested destructive database changes.
6. Keep secrets out of code, test output, agent context, logs, screenshots, and artifacts.
7. Do not start, stop, or interfere with unrelated user processes.
8. Do not modify unrelated files or overwrite concurrent owner work.
9. Do not commit, push, merge, deploy, or mutate cloud resources unless explicitly authorized for that exact operation.
10. Record actual commands, versions, results, and environmental blockers honestly.

### After work

Report:
- What changed and why.
- Files and migrations changed.
- Tests/checks run, exact outcomes, and what was not run.
- Evidence supporting each acceptance criterion.
- Fly Sprite/Machine and Neon resources inspected or changed, if any.
- Authorization used for any mutation.
- Known limitations, residual risks, and follow-up work.
- Git status and confirmation that unrelated work was preserved.
- Whether the task is ready for review or requires another decision.

Never claim completion based solely on an agent-generated summary. The evidence must support the claim.

---

## 16. Planning and implementation discipline

### 16.1 Plan before broad or risky work

For work involving multiple components, database changes, infrastructure, security, deployment, or more than one meaningful phase:
- Produce a concise implementation plan.
- Identify dependencies and sequencing.
- Separate the demo-critical path from the long-term product roadmap.
- List non-goals and required approvals.
- Define independently verifiable acceptance criteria.

For a small, clearly bounded task, do not create process overhead unnecessarily. Still preserve the invariants and report evidence.

### 16.2 Root-cause engineering

When something fails:
- Trace the full call path and relevant state transitions.
- Identify the underlying invariant and failure mechanism.
- Inspect system-level interactions, not just the line that surfaced the error.
- Add a regression test that fails before the fix and passes after it where practical.
- Test important recovery paths, including concurrency, retries, interrupted operations, and restarts.
- Implement a durable fix within scope.
- Do not hide failures, weaken checks, skip tests, or add broad retries as a substitute for understanding the cause.

### 16.3 Repository hygiene

- Inspect `git status` before and after work.
- Preserve all pre-existing modifications and untracked files.
- Avoid broad formatting or generated-file churn outside the task scope.
- Never reset, clean, stash, amend, or overwrite work without explicit permission.
- Do not commit or push without explicit authorization.
- Update relevant documentation and decision records when architecture or operational behavior changes.

### 16.4 Decisions and documentation

Record material architectural decisions with context, options, rationale, consequences, and status. Mark proposals as proposals, unknowns as unknowns, and implemented capabilities only when supported by code and evidence. Keep operational runbooks aligned with actual CLI versions and deployed behavior.

---

## 17. Acceptance and evidence standards

A capability is complete only when its observable behavior and required invariants are demonstrated.

### 17.1 Evidence hierarchy

Prefer evidence in this order:
1. Reproducible automated tests and machine-verifiable receipts.
2. Actual runtime/infrastructure state obtained from the authoritative system.
3. Durable database records and immutable artifact/deployment provenance.
4. Reproducible manual test steps with captured results.
5. Screenshots or narrative reports as supporting context.
6. Agent assertions as claims to be verified, never as sole proof.

### 17.2 Required status vocabulary

Use these labels consistently:
- **Implemented:** code exists and relevant local checks support it.
- **Implemented but not integrated:** component exists but is not connected to the end-to-end workflow.
- **Demonstrated with evidence:** the stated workflow was actually executed and evidence is retained.
- **Planned:** designed but not implemented.
- **Blocked by environment or permissions:** cannot proceed because a real dependency or authorization is unavailable.
- **Not yet tested:** implementation exists but the relevant acceptance path has not been verified.

Do not describe planned, mocked, or aborted workflows as demonstrated.

### 17.3 Definition of done

A task is done when:
- Scope and acceptance criteria are satisfied.
- Relevant formatting, static analysis, tests, and builds pass—or failures are explicitly explained and accepted by the proper authority.
- Security and data boundaries remain intact.
- Database changes have safe migrations and tests where applicable.
- Infrastructure mutations, if authorized, have verifiable receipts and correct environment targets.
- Documentation and operational procedures reflect actual behavior.
- No unrelated user work has been lost or overwritten.
- Remaining risks and limitations are explicit.

For a release, additionally require artifact provenance, digest-bound approval, Preview/Staging evidence as applicable, migration verification, Production authorization, deployment health evidence, and rollback/recovery readiness.

---

## 18. Failure, rollback, and recovery principles

- Prefer reversible operations and staged rollout over irreversible changes.
- Checkpoint a Sprite before risky workspace changes; retain Git history and Studio task state independently.
- A failed Preview should be discarded or repaired without affecting Staging or Production.
- Staging can be rebuilt or reset only under its explicit data-retention and approval policy.
- Production rollback should normally restore a prior known-good application artifact/configuration while preserving valid Production data.
- Database rollback is not automatically equivalent to application rollback. Prefer forward-compatible migrations and recovery procedures that preserve data.
- Never restore an old database snapshot over current Production data without explicit authorization and a verified recovery plan.
- After partial infrastructure failure, inspect actual Fly and Neon state before retrying. Avoid blind repeated create/deploy/delete commands.
- Record failed operations and cleanup outcomes; do not hide them by deleting the evidence.

---

## 19. Cost and resource lifecycle

Use infrastructure proportionate to the environment:
- Local for fast, repeated, low-cost development.
- Sprites for remote/persistent agent work and isolated execution; allow idle workspaces to sleep where supported and manage stored data/checkpoint retention.
- Temporary Fly Machines and Neon branches for Preview; define expiry and cleanup.
- Persistent resources only for Staging and Production where their purpose requires them.
- Do not leave temporary Machines, branches, volumes, or test resources running after their useful lifetime.
- Never apply generic cleanup to resources whose ownership or environment is uncertain.
- Record cost-sensitive decisions and resource ownership in Studio.

Cost optimization must never silently weaken Production availability, data durability, tenant isolation, or release safety.

---

## 20. Hackathon and demonstration policy

The hackathon is an opportunity to demonstrate the genuine product direction and the value of Fly Sprites. It does not define the architecture's expiration date or justify throwaway designs that undermine the long-term platform.

The dogfooding target is **Miniapp Studio developing Miniapp Studio**.

The strongest credible story is:
1. Miniapp Studio runs locally and manages a real development task on its own repository.
2. A local agent works on Studio, with durable task and evidence state.
3. Work is handed to a persistent Fly Builder Sprite with a clear handoff receipt and recoverable context.
4. The Sprite is used for remote development, builds, tests, checkpoints, and safe execution where those capabilities matter.
5. A verified Studio artifact is deployed to a Preview Fly Machine with an isolated Neon branch.
6. Real logs are inspected; an agent investigates a sanitized failure and returns a tested fix.
7. Staging and Production promotion are demonstrated only when artifact provenance, database isolation, approval, health verification, and recovery are genuine.

Demonstrate Sprites because they materially improve remote development, persistent agent workspaces, checkpoints, isolation, and execution—not merely because the hackathon sponsor is Fly.io. Demonstrate Machines because they run the application release. Demonstrate Neon because database branching makes Preview and release validation safer.

Never misrepresent a Sprite workspace preview as an immutable application deployment, a mock as a real infrastructure action, or a planned capability as a working feature. A reliable, narrow end-to-end workflow is stronger than a broad demo held together by manual claims.

---

## 21. Non-goals unless separately approved

Do not introduce these merely to appear comprehensive:
- A generic visual workflow canvas.
- A marketplace or broad plugin ecosystem.
- A universal multi-agent orchestration framework.
- A full agent/model marketplace or registry before multiple runtime integrations are needed.
- A broad fleet, quota, billing, or resource-management product unrelated to demonstrated lifecycle requirements.
- A generic unrestricted shell service.
- Metrics and tracing before real instrumentation and ingestion exist.
- Multi-cloud or multi-database abstractions without an actual second provider use case.
- A complete IDE that duplicates established developer tools.
- A speculative final PostgreSQL schema.

Future extensibility should be preserved through clean boundaries, not by implementing hypothetical features prematurely.

---

## 22. Required decision brief for significant work

For every significant implementation or operational task, the architect or coding agent should provide a short decision brief before execution:

| Field | Required content |
|---|---|
| Objective | User/business outcome and reason |
| Scope | What changes and what does not |
| Execution location | Local Desktop or which Fly Sprite/workspace |
| Application target | Local, Preview, Staging, Production, or none |
| Database target | Local PostgreSQL or exact Neon project/branch/environment |
| Infrastructure actions | Fly/Neon/GitHub commands or API operations expected |
| Authorization | Read-only, Preview, Staging, Production, or destructive approval required |
| Dependencies | Existing capabilities, schema/API prerequisites, blockers |
| Verification | Tests, runtime checks, infrastructure state, evidence receipts |
| Recovery | Expected rollback or recovery path |

For routine low-risk code changes this may be compact. For cloud and database operations it must be explicit. If any target or authorization is ambiguous, do not guess.

---

## 23. Final governing principles

1. **Build the control plane just far enough ahead of the work that it can lead the next increment.** Studio must be useful before it is complete.
2. **One application, one durable development journey.** Preserve identity, history, context, and evidence across Local and Remote execution.
3. **Sprites are for agent work and isolated execution; Machines are for application runtime.** Use each where it solves the real problem.
4. **Preview validates a candidate; Staging validates a release; Production serves users.** Keep the three environments separate and purposeful.
5. **Neon branches isolate database work; migrations advance schema deliberately.** Never confuse branching with promotion of application code or Production data.
6. **Promote one immutable artifact.** The artifact tested in Preview and Staging is the artifact authorized for Production.
7. **Agents execute; humans authorize consequential actions.** Credentials and CLI access do not confer approval authority.
8. **PostgreSQL preserves Studio's durable process.** Workspace state and processes are recoverable execution resources, not the system of record.
9. **Evidence determines status.** No claim of completion without verifiable support.
10. **Solve root causes and preserve invariants.** Do not trade durable correctness for a short-lived demo shortcut.
11. **Keep the process uniform, but scale the controls to the risk.** A local unit test does not need a Production release ceremony; a Production migration does.
12. **The hackathon demonstrates the enduring architecture.** Do not let the demo become a separate product or a source of permanent technical debt.

---

## 24. Reference documentation

Vendor interfaces and capabilities change. Before implementing a Fly or Neon integration, verify current official documentation and the installed CLI's own version/help output. These links are starting points, not substitutes for version-specific verification.

### Fly.io
- [Fly.io Sprites](https://fly.io/sprites/)
- [Fly.io — Run agent-generated code on Sprites](https://fly.io/run-agent-code/)
- [Fly.io — AI agents from prototype to production](https://fly.io/ai-agents/)
- [Fly.io — Design and implementation of Sprites](https://fly.io/blog/design-and-implementation/)
- [Fly.io Volumes](https://fly.io/docs/volumes/)

### Neon
- [Neon branching workflows](https://neon.com/branching)
- [Neon database branching workflows guide](https://neon.com/blog/database-branching-workflows-a-guide-for-developers)

### Project-specific canonical references
Consult the current repository copies of:
- `plans/plan_ui.html`
- `plans/plan_ui_z.md`
- `docs/architecture/` decision records and approved implementation plans
- Current phase evidence and acceptance reports
- Current MiniappSpec and artifact-contract documentation

When project-specific evidence conflicts with this document, do not silently choose one. Identify the conflict, determine which instruction is newer and authoritative, and obtain an explicit decision before changing a binding invariant.
