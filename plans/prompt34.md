Got it. This is a core platform requirement, not an optional deployment detail.

I inspected the current `Taskfile.dev.yml`, `docs/dev-contract.md`, `docs/dev-serve.md`, and `docs/taskfile.md`. The existing convention is a good foundation: Taskfile delegates to Go commands, sensitive files live outside the repository with restrictive permissions, and credentials are never printed.

I'll incorporate the following into the plan prompt:

- **One configuration contract:** identical environment-variable names and validation across local development and Fly Machines.
- **Local development:** environment-specific `.env` files outside the repository, defaulting to `$HOME/.config/miniapp-studio/`, with restrictive permissions.
- **Fly deployment:** securely import the corresponding environment's values into Fly's encrypted secrets store. Never ship `.env` files in images or repositories.
- **Environment separation:** distinct development, preview, staging, and production values. Development credentials must never be reused in production.
- **Unified workflow:** Taskfile commands for validation, setup, and environment synchronization, delegating to the canonical Go configuration/secrets interface.
- **Tenant credentials:** provider OAuth tokens and customer-specific API keys must be tenant-scoped through the credential/token-broker architecture—not indiscriminately exposed as global process environment variables.
- **Security:** redaction, rotation, revocation, least privilege, fail-closed behavior, and no credential leakage through logs, agent context, or evidence.

Here is the complete revised prompt, including that requirement.

# Mission: Design the RPA Platform Capabilities Required to Power Real Small-Business Workflows

You are the Principal Software Architect for `PithomLabs/miniapp-studio`.

Your task is to study the reference repository:

[https://github.com/hewi333/mom-n-pop-skills](https://github.com/hewi333/mom-n-pop-skills)

and produce an implementation plan describing how Miniapp Studio's Go Miniapp framework can support the business-automation patterns demonstrated there.

**This is a platform architecture assignment, not a request to port the skills.**

The goal is to give AI agents reusable, composable, secure, durable business capabilities so they can orchestrate workflows for small businesses without requiring every business to implement its own infrastructure, authorization, state management, integrations, and recovery machinery.

Produce:

`plans/mom-n-pop-rpa-platform-plan.md`

## 1. Study the reference implementation

Read the repository README and inspect the relevant skills, their reference documents, scripts, examples, dependencies, and workflows.

Pay particular attention to:

- `lead-to-payment`: orchestration across the business lifecycle.
- `crm-lite`: customer, lead, estimate, activity, communication, and task records.
- `estimator-engine`: deterministic, configuration-driven business calculations.
- `stripe-payments` and `stripe-link-cli`: financial operations and authorization.
- `quickbooks-online`: accounting and financial synchronization.
- `outlook-graph`: email, contacts, and calendar integration.
- `mailchimp-integration` and `marketing-campaign-builder`: audience and campaign operations.
- `construction-agent`: multi-provider business operations and document routing.
- `grasshopper-voicemail-monitor`: event intake, polling, deduplication, and notifications.
- `financial-analysis`: structured data analysis and owner-facing reporting.
- `small-business-ai-transformation`: business discovery and the six-stage business lifecycle.
- `agent-cheatsheet-builder`: presenting automation capabilities to non-technical owners.
- `hermes-production-ops`: production reliability and end-user communication.

Inspect the actual implementation, not just the README descriptions. Identify reusable patterns, assumptions, limitations, coupling, failure modes, and places where the reference relies on Hermes-specific behavior.

Do not treat every documented capability as production-ready or assume its reference implementation is the correct architecture.

## 2. Target business outcome

The platform should make it possible to implement workflows such as:

**Lead-to-payment**

Inbound inquiry / voicemail / website form / email\
→ capture and qualify lead\
→ identify or create customer\
→ calculate estimate\
→ present estimate to owner\
→ obtain approval\
→ create payment request\
→ prepare customer communication\
→ obtain sending approval\
→ execute communication\
→ monitor payment\
→ update business records\
→ schedule work\
→ produce owner reports.

Other examples include:

- Email and document intake → classification → CRM/project routing → owner notification.
- Customer follow-up and appointment scheduling.
- Marketing campaign drafting → audience selection → owner approval → campaign execution.
- Invoice/payment reconciliation → accounting updates → exception reporting.
- Voicemail and website-lead monitoring → deduplication → CRM entry → notification.
- Financial reporting and business-performance analysis.
- Recurring briefings, reminders, and follow-up workflows.

These are reference use cases, not a requirement to implement every business workflow.

## 3. Define the platform abstractions

Determine the smallest coherent set of reusable platform primitives required to support these workflows.

Evaluate and specify the following capability areas:

**A. Business domain primitives**

Customer, organization, tenant scope, workspace, contact, lead, opportunity, estimate, job, appointment, invoice, payment, communication, activity, task, campaign, document, and business event.

Determine which should be framework-level contracts, domain-specific Miniapp types, or integration-owned records. Avoid creating a giant universal CRM schema.

**B. Miniapp capability model**

Define how a Miniapp exposes business capabilities to agents through typed contracts:

- Inputs and outputs.
- Preconditions and validation.
- Execution category (A/B/C/D).
- Idempotency identity and entity key.
- Trust mode and authorization policy.
- External effects and blast-radius limits.
- Result, error, and recovery semantics.

An agent should discover what a Miniapp can do from its declared contract rather than depending on hardcoded provider-specific instructions.

**C. Workflow orchestration**

Explain how agents compose Miniapp capabilities into multi-step business workflows.

Distinguish business logic from orchestration and infrastructure.

A workflow should support:

- Conditional branching.
- Typed results and explicit dependencies.
- Human decisions and approval waits.
- Durable timers and scheduled follow-ups.
- Retries and error handling.
- Cancellation and compensation.
- Progress and execution history.
- Recovery after process or worker failure.

Use the existing A/B/C/D classification and ADR-001 gates. Do not select a workflow engine merely because the examples contain multiple steps. Do not introduce DBOS, River, or another engine without evidence-backed architectural justification.

**D. Integration and connector architecture**

Design provider-neutral contracts for connecting external systems.

Examples include CRM, email, calendar, payment, accounting, marketing, project management, file storage, telephony, and notification providers.

Address:

- Provider discovery and capability discovery.
- Typed API adapters.
- Credential and OAuth lifecycle.
- Tenant-scoped authorization.
- Credential rotation and revocation.
- Rate limits and retry policies.
- Pagination and webhooks.
- Provider-specific feature differences.
- Sandbox/test versus production environments.
- Connector health and diagnostics.

A business must be able to change providers without rewriting its business workflow wherever the underlying capability contracts are compatible.

Do not assume all providers support identical features. Model capability differences explicitly.

**E. Human approval and authority**

Preserve the reference repository's defining pattern:

`Agent drafts → owner approves → agent executes`

Design one authoritative, framework-owned approval mechanism with:

- PENDING / APPROVED / REJECTED / EXPIRED states.
- Approval bound to the exact material being approved.
- Expiry and invalidation when material changes.
- Explicit approver identity and authority scope.
- Separation between agent and approval principal.
- Audit evidence.
- Safe resume after approval.

Distinguish low-risk reads and calculations from customer communications, financial operations, accounting writes, purchases, destructive actions, and other consequential effects.

The agent must never approve its own action. An approval for one operation must not authorize a different operation.

**F. External effects and financial safety**

Design the platform-level effect protocol around:

PENDING → ATTEMPTING → CONFIRMED / FAILED / UNKNOWN

Support idempotency, keyed serialization, reconciliation by reading provider state, UNKNOWN quarantine, and safe retry decisions.

Explicitly cover duplicate payment requests, duplicate emails, webhook replay, network timeouts after provider acceptance, partial workflow completion, refunds, and accounting synchronization.

Never blindly retry an external action with an ambiguous outcome.

**G. Event intake and background operations**

Support external events and recurring business work:

- Email/form/voicemail intake.
- Provider webhooks.
- Scheduled polling.
- Cron-like recurring jobs.
- Event deduplication and ordering.
- Cursor/UID-based incremental processing.
- Checkpointing.
- Owner notifications.
- Silent idle execution where appropriate.
- Monitoring, retry, disablement, and recovery.

Determine when polling, webhooks, or scheduled jobs are appropriate. Do not assume an LLM must run for every event; deterministic code should handle suitable ingestion and routing.

**H. Multi-tenant data and security**

Preserve the existing organization → tenant scope → workspace model, explicit delegation, Neon PostgreSQL, RLS, transaction-local tenant context, and fail-closed isolation.

Every workflow, execution, business record, connector credential, approval, effect, audit event, and artifact must have the correct tenant context.

No ambient cross-tenant access. Shared templates must not imply shared credentials or business data.

Neon remains mandatory. Do not propose SQLite as the production platform database or introduce an alternate database provider.

**I. Agent and owner interaction**

Design how agents discover available business capabilities, receive typed outcomes, request human decisions, resume workflows, and report results.

Support messaging-oriented owner experiences, including Telegram-style interaction, without requiring non-technical business owners to operate a technical dashboard.

The owner-facing experience should use plain language and expose only the decisions and actions relevant to that owner.

Do not couple the core RPA framework to Hermes, Telegram, or a particular LLM provider.

## 4. Secrets management and unified environment configuration

This is a first-class platform requirement.

Miniapp Studio must provide a seamless, consistent configuration and secrets-management experience across local development and Fly.io deployment.

The owner must not have to maintain two unrelated credential-management procedures or manually translate environment-variable names between local development and production.

First inspect and reuse the existing conventions:

- `Taskfile.dev.yml`
- `Taskfile.yml`
- `docs/dev-contract.md`
- `docs/dev-serve.md`
- `docs/taskfile.md`
- Existing Go configuration and secret-loading code.

The current development convention uses Taskfile commands as the operator interface, delegates actual behavior to Go commands, keeps sensitive files outside the repository with restrictive permissions, and avoids printing credentials. Extend this design rather than replacing it unnecessarily.

### 4.1 One configuration contract

Define a canonical, machine-readable inventory/schema for all required environment variables and configuration values.

It must describe:

- Variable name and purpose.
- Owning application or Miniapp.
- Environment scope.
- Whether it is secret or non-secret.
- Whether it is required or optional.
- Validation requirements.
- Consumer and access scope.
- Rotation/revocation expectations.
- Whether it is a platform bootstrap secret or tenant/provider credential.

The same variable names, contracts, validation rules, and configuration semantics must apply across local development and Fly-hosted environments.

Avoid duplicated configuration definitions in Taskfiles, Go code, Docker/build files, deployment scripts, and documentation.

### 4.2 Local development: external `.env`

Local development credentials must live outside the Git repository.

Use an environment-specific configuration directory under:

`$HOME/.config/miniapp-studio/`

A recommended convention is separate `.env` files for development and any other local profiles, for example:

- `$HOME/.config/miniapp-studio/dev/.env`
- `$HOME/.config/miniapp-studio/staging/.env`
- `$HOME/.config/miniapp-studio/prod/.env` only when genuinely needed for an explicitly authorized operator workflow.

Determine the canonical path and document it.

Requirements:

- Directory permissions restrictive (normally `0700`).
- `.env` files restricted to the owning user (normally `0600`).
- Never store actual secrets in the repository.
- Never commit `.env` files.
- Provide a safe committed example/template containing names and placeholders only.
- Ensure `.gitignore` prevents accidental inclusion of secret files.
- Validate file ownership, permissions, and format before use.
- Never print values in Task output, command output, logs, errors, diagnostics, fingerprints, or evidence.
- Do not source arbitrary shell code from `.env`; parse it using a controlled, documented mechanism.
- Fail closed on malformed configuration or missing required secrets.
- Preserve existing file-based secret contracts such as the Neon DSN and generated application keys/passwords where they are intentionally file-backed.

The developer experience should be straightforward: initialize the environment, edit one secure local `.env` file, and use the normal Taskfile commands.

### 4.3 Fly.io deployment: same variable contract

The same configuration contract must work when Studio or a Miniapp is deployed to Fly.io.

Use Fly's encrypted secrets mechanism for runtime secrets.

The deployment workflow must:

1. Select an explicit deployment environment and Fly app.
2. Read the corresponding environment-specific `.env` source outside the repository.
3. Validate required names, environment scope, and values without displaying them.
4. Synchronize the approved values into Fly's encrypted secret store.
5. Preserve the same environment-variable names expected by the Go runtime.
6. Verify that required secret names are configured without revealing their values.
7. Record a secret-free configuration fingerprint and synchronization evidence.

Use a secure, repeatable operator interface, preferably integrated into the existing Taskfile and canonical Go command conventions.

The agent must investigate the currently supported Fly CLI/API mechanisms and choose the simplest reliable approach. Do not invent commands or depend on unverified Fly behavior.

**Important:** the local `.env` is a secure source for configuration, not a deployment artifact. Never copy it into a Docker image, Fly image filesystem, Git repository, or public artifact. Fly receives the approved values through its encrypted secrets store. Runtime processes receive the same named environment variables through Fly's environment injection.

Local development and Fly production must use the same configuration schema and application configuration loader, but distinct environment-specific values and credentials.

Never synchronize development secrets into production accidentally. The selected Fly app and environment must be explicitly verified before any secret synchronization.

### 4.4 Unified Taskfile experience

Plan a consistent operator interface for:

- Initializing local environment configuration.
- Validating local configuration and required secrets.
- Running local development with the selected `.env`.
- Checking configuration completeness and permissions.
- Preparing Fly deployment configuration.
- Synchronizing secrets to the explicitly selected Fly app.
- Verifying deployed configuration without exposing values.
- Rotating or revoking secrets.
- Diagnosing missing, expired, or invalid credentials.

Taskfiles should orchestrate the canonical Go configuration/secrets interface and appropriate Fly operations. Do not reimplement configuration parsing, validation, or security policy separately in YAML or shell scripts.

Provide a simple, documented developer workflow. The developer should not need to manually export dozens of variables or copy values between unrelated tools.

### 4.5 Configuration and secret loading

The Go runtime must use one well-defined configuration-loading contract across local and Fly environments.

It should support the existing `_FILE` convention where appropriate, environment-injected secrets, validation, and deterministic environment selection.

Define precedence explicitly and avoid ambiguous cases where one source silently overrides another.

Required behavior:

- Missing required secret: fail closed with a useful, secret-free diagnostic.
- Invalid or malformed secret: fail closed.
- Conflicting sources: reject or apply a clearly documented precedence rule.
- Optional integrations: remain unavailable until properly configured; unrelated application capabilities should continue working.
- Secret rotation: support deliberate rotation without accidental credential loss.
- Revocation: fail closed and audit the outcome.
- Diagnostics: report variable names and state only, never values.

### 4.6 Distinguish deployment secrets from tenant connector credentials

Do not confuse platform-level environment variables with each customer's external provider credentials.

**Platform/deployment secrets** may include database connection credentials, signing keys, encryption keys, and platform service credentials. These belong to the selected deployment environment and are loaded through the environment configuration contract.

**Tenant/provider credentials** may include a customer's Stripe, QuickBooks, Microsoft, Google, Mailchimp, or project-management OAuth/API credentials.

These must be tenant-scoped and managed through the platform's credential/token-broker abstraction. They must not be indiscriminately exposed to the entire Studio process or injected into untrusted Miniapp/Sprite environments.

Define how the secure credential service:

- Stores or references tenant-specific credentials.
- Encrypts credentials at rest using an appropriate key-management boundary.
- Restricts access to the authorized tenant, Miniapp, provider, operation, and execution context.
- Supports OAuth authorization, refresh, rotation, expiry, and revocation.
- Provides short-lived or narrowly scoped access to provider adapters.
- Prevents credentials from entering agent prompts, workflow payloads, logs, checkpoints, artifacts, or audit records.
- Separates sandbox/test credentials from production credentials.

Evaluate reuse of existing project infrastructure before proposing another secret-management system. Make any new infrastructure dependency an explicit architecture decision rather than silently introducing one.

### 4.7 Environment isolation

Preserve distinct configuration and credential sets for:

- Local development.
- Builder Sprite execution.
- CI.
- Engineering PR Preview.
- Application Preview.
- Staging.
- Production.

Use the same configuration contracts where appropriate, but separate credentials, identities, database branches, and deployment scopes.

Local development must never connect to the production database or inherit production credentials. Production secrets must never be exposed to local development, CI, or untrusted Sprites by default.

Respect the existing Neon-only database contract and production-branch guards.

### 4.8 Secrets-management acceptance criteria

The plan must define executable tests proving:

- `.env` files are outside the repository and correctly permissioned.
- Secret files cannot accidentally enter Git, build contexts, or artifacts.
- Local configuration loads successfully through the canonical Taskfile workflow.
- Fly synchronization uses the same variable contract and correct environment-specific values.
- A wrong Fly app/environment selection is rejected before mutation.
- Missing, malformed, or conflicting configuration fails closed.
- No secret values leak into logs, diagnostics, fingerprints, test output, agent context, or evidence.
- Tenant A cannot access tenant B's connector credentials.
- Revoked credentials are no longer usable.
- A credential rotation does not silently break unrelated integrations.
- Runtime parity is demonstrated across local and Fly environments without requiring identical secret values.

The resulting developer experience must be **one unified configuration and secrets-management workflow, with environment-specific secure storage adapters**.

## 5. Map the requirements to Miniapp Studio's existing architecture

Use the ultimate roadmap as the canonical authority:

`plans/miniapp-studio-ultimate-roadmap.md`

Respect its Roadmap A / Roadmap B / F0 separation and the existing RT, T, S, E, H, and O tracks.

Explain precisely:

- What belongs in the Miniapp framework.
- What belongs in shared F0 platform services.
- What belongs in connector adapters.
- What belongs in individual Miniapps.
- What belongs to agent orchestration.
- What belongs in the Engineering Studio UI.
- What belongs in the separate Automation Operations Studio.
- Where the configuration/secrets contract belongs.
- Which components own local `.env` loading, Fly synchronization, runtime configuration, and tenant credential brokerage.
- Which roadmap phases and existing workstreams provide the dependencies.

The Miniapp framework should own reusable guarantees, not business-specific policies. For example, the framework owns approval enforcement; an individual business workflow decides which actions require approval under its declared policy.

Do not duplicate existing functionality. Inspect the current repository and identify capabilities already implemented or accepted.

## 6. Produce an actionable architecture and implementation plan

The deliverable must include:

1. Executive summary and target operating model.
2. Reference-repository workflow and capability inventory.
3. Current Miniapp Studio capability gap analysis.
4. Proposed domain and platform abstraction model.
5. Component architecture and responsibility boundaries.
6. Typed API/contract examples in Go.
7. Connector and provider abstraction strategy.
8. Workflow orchestration and execution-category mapping.
9. Approval, effect ledger, idempotency, and recovery design.
10. Multi-tenant security and data model implications.
11. Secrets-management architecture covering local `.env`, Fly secrets, runtime loading, tenant credentials, and environment parity.
12. At least one complete lead-to-payment workflow mapped onto the proposed primitives.
13. At least two additional workflows demonstrating reuse across different business domains.
14. Dependency-ordered implementation roadmap.
15. Testing and verification strategy, including crash, retry, duplicate, authorization, tenant-isolation, and secrets-leakage tests.
16. Risks, tradeoffs, open architectural decisions, and explicit non-goals.
17. Traceability matrix from reference capabilities to platform primitives and implementation milestones.

For each proposed primitive, identify:

- The problem it solves.
- Its owning layer.
- Its contract.
- Its failure and recovery semantics.
- Its security boundary.
- Its required tests.
- Whether it already exists, needs extension, or is new.

Include concrete Go interface/type sketches where they clarify the architecture. Avoid speculative code dumps.

## 7. Scope and product discipline

The goal is a reusable RPA platform, not a port of 16 Hermes skills.

Do not:

- Execute or copy all skills from the reference repository.
- Adopt Python scripts or Hermes as a platform dependency.
- Build a giant CRM as the framework's core.
- Hardcode Stripe, QuickBooks, Mailchimp, Google, Microsoft, or JobTread into workflow logic.
- Implement every connector before defining the contracts.
- Build a generic visual workflow canvas merely because workflows exist.
- Build billing or metering productization.
- Reopen locked architecture decisions.
- Invent a universal schema that erases provider-specific semantics.
- Introduce an unapproved workflow engine.
- Expand Roadmap A into Roadmap B.
- Put actual credentials in the repository or application images.
- Create a second, incompatible local-versus-production configuration system.
- Expose all tenant credentials to a shared process or untrusted execution environment.
- Use development credentials in production or production credentials in development.

Prefer a small set of strong primitives that can support many workflows.

The first demonstration should prove that the platform can compose a few real capabilities into a safe, owner-approved business workflow. It should not attempt to reproduce the entire reference repository.

## 8. Execution autonomy and review

This is a planning task only.

Autonomously inspect the repository, reference materials, existing contracts, tests, accepted decisions, and relevant ADRs. Make ordinary architectural recommendations with explicit rationale. Do not ask routine clarification questions.

Raise questions only when a genuine owner decision materially affects the architecture. Record those as explicit OWNER-DECISION items with a recommended default and consequences; do not block the plan on minor uncertainties.

Perform a self-review and a distinct adversarial architecture review of the proposed plan. Challenge:

- Over-abstraction and unnecessary complexity.
- Duplicate ownership of business state.
- Unsafe approval semantics.
- Missing failure and recovery contracts.
- Tenant isolation gaps.
- Provider lock-in.
- Unnecessary workflow-engine adoption.
- Missing idempotency and reconciliation.
- Premature UI or connector implementation.
- Secrets leakage or ambiguous configuration precedence.
- Local/production credential crossover.
- Unsafe Fly secret synchronization.
- Confusion between deployment secrets and tenant connector credentials.
- Conflicts with the ultimate roadmap.

Resolve substantive findings before delivery. Classify remaining issues as blocking decisions, material risks, or non-blocking improvements.

## 9. Guardrails

Planning only:

- Do not implement code.
- Do not modify the roadmap or existing architecture decisions.
- Do not change migrations or database schemas.
- Do not provision cloud resources or perform provider authentication.
- Do not request, expose, or store credentials.
- Do not execute Fly secret synchronization.
- Do not commit or push.

Create only:

`plans/mom-n-pop-rpa-platform-plan.md`

## 10. Definition of done

The document must enable the next implementation assignment to proceed without another large planning cycle.

A senior Go engineer should be able to read it and understand what platform primitives to build, where they belong, how they compose, how they fail safely, how secrets are configured consistently across local and Fly environments, and how the first business workflow will use them.

A reviewer should be able to trace every significant recommendation to either the reference repository, current Miniapp Studio architecture, or an explicitly labeled proposal.

Finish with:

- The recommended minimum viable primitive set.
- The first vertical-slice implementation target.
- A dependency-ordered implementation sequence.
- The unified local/Fly configuration and secrets workflow.
- Explicit owner decisions, if any.
- Adversarial review findings and dispositions.

Do not implement anything. Deliver the architecture plan and a concise completion report.

One important architectural distinction is preserved: **the same configuration contract does not mean sharing the same secret values or storage location.** Local `.env` files and Fly's encrypted secrets store are environment-specific sources; tenant-owned provider credentials belong in the tenant-scoped credential service. This gives you one developer/operator experience without compromising tenant isolation or production security.