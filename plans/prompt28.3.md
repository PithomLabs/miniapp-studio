Understood. This is a **hard architectural constraint**, not a preference or an optional database integration.

We must incorporate Neon into the architecture before implementation decisions are finalized.

I am making one important correction to our previous prompt28.2: **PostgreSQL must be provided by Neon across every environment.** We should not independently provision PostgreSQL through Ubuntu packages, Docker, Fly Machines or Sprites.

This means the earlier proposal for native PG19 provisioning through `task dev:setup` must be reconsidered. Taskfiles still own declarative setup, but their database provisioning responsibility should be implemented through Neon.

The same applies to CI and per-PR previews: Neon must be the database infrastructure behind those environments, not merely an optional supported provider.

## **1. What this changes**

| Area              | Previous proposal                 | Corrected architecture                     |
| ----------------- | --------------------------------- | ------------------------------------------ |
| Local development | Native PostgreSQL 19              | Neon developer branch                      |
| Local setup       | Install PostgreSQL through PGDG   | Provision/verify Neon branch               |
| Integration tests | Local PostgreSQL / Testcontainers | Ephemeral Neon branch                      |
| CI                | Digest-pinned PostgreSQL 19 image | Neon PostgreSQL 19 branch                  |
| PR preview        | Dedicated PostgreSQL instance     | Neon branch per PR                         |
| Coding agents     | Database supplied by Sprite       | Isolated Neon branch per agent when needed |
| Production        | PostgreSQL deployment             | Neon production project                    |
| Customer miniapps | Database provisioning undecided   | Neon-backed database provisioning          |

This is technically viable today: Neon announced PostgreSQL 19 availability on November 19, 2026.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://neon.com\&sz=32)

Neon+1



Neon's branching model is particularly well suited to this architecture. Branches use copy-on-write storage, have independent compute endpoints, and can be created for PRs and individual test runs.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://neon.com\&sz=32)

Neon Branching+2



## **2. The Neon architecture I recommend evaluating**

## **Neon database topology**

Production boundary

**Neon Production Project(s)**

Studio control plane and customer-facing production workloads

Protected production branches; no CI or agent write access.

Non-production boundary

**Neon Development Project**

Built from synthetic or sanitized data; separate from production

**Developer branches**

Persistent local development

**Agent branches**

Isolated coding tasks

**CI branches**

One per validation run

**PR branches**

One per preview lifecycle

Illustrative topology. The project-per-service, project-per-miniapp and branch-allocation decisions must be validated through ADRs.

I favor separate production and non-production Neon projects. Neon projects are the top-level resource boundary; branches within a project provide lightweight environment isolation. Neon itself describes project-per-user as its strongest database isolation pattern for platform architectures.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://neon.com\&sz=32)

Neon+1



For customer-generated miniapps, we must also explicitly decide whether each miniapp receives its own Neon project or whether certain applications share a project with stronger database-level isolation.

That decision affects security, billing, quotas, recovery and the long-term platform control plane.

## **3. One critical Neon security consideration**

There is a detail I particularly want the implementation agent to investigate.

Neon branches inherit PostgreSQL roles and passwords from their parent by default. This means creating a child branch alone does not guarantee unique credentials. Neon provides branch protection to regenerate role passwords, but its availability depends on the subscription plan.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://neon.com\&sz=32)

Neon Docs+1



Therefore, the design must explicitly address credential rotation.

Also:

- Neon API keys must remain with the control plane.
- Use project-scoped API keys rather than broad personal or organization credentials.
- Never give coding agents unrestricted Neon API access.
- Create appropriately restricted database roles for application and migration workloads.
- Do not use the Neon project-owner role as an application runtime credential.
- Use sanitized or synthetic data for agent and CI branches.
- Verify that inherited credentials cannot access production.

Neon distinguishes management API keys from application database credentials, and its documentation explicitly recommends keeping API keys out of application runtimes.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://neon.com\&sz=32)

Neon Docs+1



## **4. Prompt amendment to add to prompt28_plan.md**

# Prompt28.3 — Mandatory Neon-First PostgreSQL Architecture

## 1. Hard architectural constraint

Neon is the mandatory PostgreSQL provider across the entire Miniapp Studio ecosystem.

This is a non-negotiable architectural constraint.

Every PostgreSQL workload must use Neon-managed PostgreSQL, including:

- Studio control-plane persistence.
- Studio development.
- Developer workspaces.
- Background coding-agent execution.
- CI and integration testing.
- Per-PR preview environments.
- Production deployments.
- Generated miniapps and customer application databases.
- Other internal PostgreSQL-backed services.

Do not introduce self-hosted PostgreSQL as an alternative database provider.

Do not install PostgreSQL server packages through apt/PGDG as a development prerequisite.

Do not introduce local Docker PostgreSQL containers, PostgreSQL Testcontainers, or PostgreSQL instances inside Fly Machines or Sprites.

Local tools such as psql and compatible PostgreSQL client utilities remain permissible for connecting to Neon.

## 2. Superseded decisions

The following previously documented decisions must be reconciled:

1. Native PostgreSQL 19 installation on the development host is superseded.
2. Local PGDG repository configuration is no longer required.
3. The owner-side PostgreSQL server installation procedure must be removed.
4. The local PostgreSQL maintainer-role provisioning procedure must be replaced with Neon credential and role provisioning.
5. The digest-pinned PostgreSQL 19 container requirement for CI is superseded by Neon-managed PostgreSQL 19.
6. PostgreSQL Testcontainers-based integration tests must be replaced by tests against isolated Neon branches.

Preserve all other compatible prompt28.1 and prompt28.2 decisions.

This is an explicit architecture amendment. Do not attempt to satisfy incompatible Docker-image and Neon-only requirements simultaneously.

## 3. Neon database topology

Produce an architecture decision record defining the Neon project and branch hierarchy.

At minimum, evaluate:

- Production Neon projects.
- Non-production Neon projects.
- Persistent developer branches.
- Agent execution branches.
- Ephemeral CI branches.
- Per-PR preview branches.
- Customer miniapp projects and their lifecycle.

Strongly consider separate Neon production and non-production projects.

Never derive untrusted agent or CI branches directly from a production database containing sensitive customer information.

Use synthetic fixtures, schema-only branches, or properly anonymized data.

Decide whether customer applications require one Neon project per miniapp, per tenant, or per application security boundary. Evaluate cost, isolation, quotas, lifecycle, observability and recovery.

Do not assume every application should share a single project.

## 4. Local development through Neon

Replace native PostgreSQL provisioning with Neon provisioning.

The existing `task dev:setup` entry point must:

1. Verify the Neon CLI or supported API client prerequisites.
2. Verify required Neon authentication.
3. Validate the configured non-production Neon project.
4. Create or reconcile the developer's persistent Neon branch.
5. Provision or verify appropriate database credentials.
6. Obtain the corresponding connection details securely.
7. Configure the development environment.
8. Invoke the canonical `studio dev` verification interface.
9. Verify actual database connectivity and engine version.

Use the Neon API or supported CLI for project and branch lifecycle management.

Keep project-scoped management credentials outside agent processes and application runtime environments.

The local development process must not depend on a local PostgreSQL server.

Taskfiles orchestrate this workflow. Go implements the canonical application behavior.

## 5. Neon-backed integration testing and CI

Every independent integration-test execution must receive its own disposable Neon branch.

The workflow must:

1. Select the designated sanitized non-production parent branch.
2. Create a fresh child branch for the specific execution.
3. Apply the canonical migration sequence.
4. Provision appropriate branch-specific credentials.
5. Execute database-backed integration tests.
6. Collect test reports and database diagnostics.
7. Delete the branch after completion.
8. Reconcile abandoned branches following failed or interrupted executions.

CI must run on the previously proposed isolated Fly Sprite infrastructure.

The Sprite connects to Neon over its PostgreSQL endpoint. It does not run its own PostgreSQL server.

The database branch and execution must be associated with the exact commit SHA.

Preserve:

- Go 1.27.
- Existing port mapping and canonical application URL.
- Existing validation targets.
- Zero allowlist changes.
- Required extension inventory, currently specified as none, subject to reproducible verification.

Replace image-digest verification with Neon-specific verification:

- Neon project identity.
- Branch identity and parent identity.
- Configured PostgreSQL major version.
- Actual `SHOW server_version` result.
- Actual `SHOW server_version_num` result.
- Required extension verification.
- Compute and connection configuration where relevant.

Record the actual managed PostgreSQL version. Do not claim container-image reproducibility for managed Neon infrastructure.

## 6. Per-PR Neon database environments

Every eligible Studio pull request must have its own Neon database branch.

The branch must be associated with:

- GitHub repository.
- PR number.
- PR head commit SHA.
- Preview application identity.
- Neon project ID.
- Neon branch ID.
- Database connection configuration.
- Lifecycle state and expiration.

The preview application deployed on Fly.io must connect exclusively to its associated Neon branch.

Subsequent commits must update the correct PR environment.

Investigate the safest strategy for:

- Reusing one persistent branch per PR.
- Creating a new generation branch per commit.
- Handling database migrations between commits.
- Preserving or resetting preview data.
- Preventing stale deployments from overwriting newer commits.
- Atomically switching application configuration to the correct branch.

The preview controller must reconcile Neon branch state with Fly deployment state.

Closing or merging a PR must trigger branch deletion, with independent expiration and garbage-collection safeguards.

## 7. Neon credential and security architecture

Treat Neon management credentials and PostgreSQL credentials as separate security domains.

Requirements:

- Store Neon API keys only in approved secret-management infrastructure.
- Use project-scoped API keys with minimum required permissions.
- Never expose organization-wide Neon management credentials to coding agents.
- Never inject Neon management API keys into deployed application environments.
- Use distinct credentials for production, development, CI, previews and agent workloads.
- Create least-privilege application and migration roles.
- Never use the default project-owner or unrestricted neon_superuser role as a normal application credential.
- Verify branch role/password inheritance behavior.
- Investigate Neon branch protection and credential regeneration.
- Rotate or replace inherited role passwords before untrusted workloads can access a new branch.
- Ensure CI and agent workloads cannot reach production branches through inherited credentials or application configuration.
- Ensure secrets do not enter logs, commits, build artifacts or reusable Sprite checkpoints.

Verify all assumptions about branch-scoped database role behavior through an executable technical proof.

## 8. Neon lifecycle, quotas and economics

The design must account for actual Neon plan capabilities and limits.

Document:

- Maximum supported branch concurrency.
- Project limits.
- Branch limits.
- Additional branch charges.
- Compute usage.
- Storage and history usage.
- Scale-to-zero behavior.
- Resume latency.
- Autoscaling configuration.
- Branch expiration.
- Branch deletion.
- Project provisioning costs.
- Recovery and retention behavior.

API- and CLI-created branches must receive explicit expiration policies where applicable. Do not assume they automatically expire.

Implement a reconciliation process for expired, abandoned and partially provisioned resources.

Account for concurrent PR previews and CI runs when selecting the Neon plan and project topology.

## 9. Taskfile architecture

Extend existing Taskfiles without creating a parallel task orchestration layer.

The declarative setup must expose documented commands for:

- Neon authentication verification.
- Neon project preflight.
- Developer branch provisioning.
- Development environment configuration.
- Database migration.
- Integration testing.
- Ephemeral CI branch lifecycle.
- Preview database lifecycle.
- Database diagnostics.
- Resource reconciliation.

Taskfiles must invoke the canonical Go `studio dev` commands.

Do not reproduce Neon API management logic using shell scripts when it belongs in the Go application or controller.

## 10. Documentation and acceptance evidence

Update:

- `plans/prompt28_plan.md`
- `docs/dev-parity-report.md`
- `pg_muse.md`
- Existing development setup documentation.
- CI and preview architecture documentation.
- Relevant architecture decision records.

Remove stale references to local PostgreSQL provisioning and PostgreSQL Docker-image execution.

Produce ADRs covering:

1. Neon project boundaries.
2. Branch hierarchy and lifecycle.
3. Developer and local database workflow.
4. CI branch lifecycle.
5. PR preview database lifecycle.
6. Customer miniapp database provisioning.
7. Credential inheritance and rotation.
8. Data anonymization and fixture strategy.
9. Neon quotas, costs and concurrency.
10. Failure recovery and abandoned-resource cleanup.

Acceptance requires executable proof that:

- Local development works without a PostgreSQL server.
- Developer environments connect to Neon.
- CI runs against an isolated Neon PostgreSQL 19 branch.
- Each test run has isolated database state.
- Each PR has an associated isolated Neon branch.
- Customer miniapp provisioning uses Neon.
- Untrusted workloads cannot access production credentials or production databases.
- Branches are cleaned up reliably.
- Local setup and CI execute the same canonical validation contract.

Do not claim completion from a written design alone.

## 11. Governing principle

Neon is the PostgreSQL infrastructure layer. Fly provides application compute and isolated execution. Go implements platform behavior. Taskfiles declare reproducible workflows. GitHub provides collaboration and review.

Do not introduce another PostgreSQL infrastructure provider, including local or containerized PostgreSQL, as a convenience workaround.

Any exception requires an explicit architectural amendment approved by the project owner.

## **5. One final design improvement**

Neon also launched branch-aware Object Storage, where database branches and associated object-storage contents can be forked together. That is worth evaluating for PR previews because it could eliminate another source of cross-preview file contamination. It is not part of your mandatory PostgreSQL constraint, though, so adoption should be justified separately.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://neon.com\&sz=32)

Neon



**My calibration going forward:** Neon is now a foundational infrastructure decision, at the same level as Fly.io—not an interchangeable database adapter. I will treat any design that introduces PostgreSQL outside Neon as an architectural violation unless you explicitly authorize an exception.