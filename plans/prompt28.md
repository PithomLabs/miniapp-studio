I recommend giving the coding agent one **deadline-driven execution mission**, not another architecture-review assignment.

The priority is to unblock database testing without HostExec, preserve local-first development, and then return immediately to the B2 demo path. The agent must not spend the remaining Hacksgiving window building infrastructure that does not directly advance the demonstration.

# EXECUTION MISSION: Unblock Miniapp Studio and Deliver the B2 Hacksgiving Demo

## Role

Act as the Senior Engineer and Technical Execution Lead for PithomLabs/Miniapp Studio.

Your objective is to deliver the strongest credible, working demonstration of Miniapp Studio's B2 capabilities before the Hacksgiving deadline.

You are not being asked to produce another architecture proposal. You are being asked to turn our existing engineering foundations into a working product, using evidence from actual execution.

## 1. Mission and priorities

Our roadmap remains locked:

- B0 — Studio manages one development task: ACCEPTED.
- B1 — Studio supports its local development loop: ACCEPTED.
- B2 — Remote development and resident engineering agent: CURRENT MILESTONE.
- B3 — Formal Application Preview: FUTURE.
- B4 — Studio troubleshoots itself: FUTURE.
- B5 — Release management: FUTURE.

Do not redesign this roadmap.

Our immediate goal is to make B2 demonstrably real. Miniapp Studio must manage an actual coding agent operating inside its own persistent Fly Sprite.

Prioritize:

1. Unblock database-dependent development and testing.
2. Complete B2-01B's outstanding verification.
3. Run a real coding agent inside the Builder Sprite.
4. Prove genuine Fly Sprites GitHub Connector integration.
5. Prove Studio-controlled authorization for consequential GitHub operations.
6. Return execution evidence to Studio.
7. Demonstrate Workspace Preview if the core path is working.

B3 is a stretch goal. Do not divert time into B3, B4, B5, generalized execution-classification machinery, or speculative DBOS/River abstractions.

## 2. Current engineering baseline — verify before changing anything

The following is our known baseline. Inspect the actual repository, reports, current execution environment and Git state before relying on it.

### Accepted foundations

- B0 and B1 are accepted.
- B2-01A lifecycle contract/controller work is implemented and accepted.
- Sprite-per-app topology has been proven.
- The local development and remote Builder Sprite execution models are architecturally separate.
- One active workspace writer per app remains a governing invariant.
- Git is source history; Studio PostgreSQL is execution and authorization truth; Sprite checkpoints preserve workspace state.

### B2-01B

The lifecycle implementation and deterministic tests have been developed.

The remaining important proof is actual Fly behavior: provision/use the designated Sprite, test Tasks API holds, attachment/detachment, automatic pause and wake behavior, persistence of lifecycle receipts, and recovery/reconciliation.

Do not claim live lifecycle verification based on SDK mocks or scripted test doubles.

### Host Execution Broker and test runner

The `hostexec/` and `testrunner/` implementations have undergone extensive adversarial review.

The latest review classified the implementation as ready for installation, not installed or operational. Fake-Docker testing is not real-Docker verification.

Do not make installation of this broker a prerequisite for database testing or B2 progress.

Do not spend the remaining demo window repeatedly debugging the host sandbox or trying Docker-in-Sprite workarounds.

Preserve both implementations. They may remain useful for future host-specific capabilities.

## 3. Establish two supported database development environments

This is an architectural requirement, not a temporary workaround.

**Local development must remain fully supported. Remote development must achieve functional parity.**

### Local environment

Preserve the existing local development workflow.

Our known reference setup is a PostgreSQL 19 Docker container on the host, exposed at `127.0.0.1:5434`. Verify the actual running server version, endpoint, credentials configuration and current repository test conventions before modifying anything.

Do not dismantle, replace or casually alter the existing local environment.

Local development must remain usable independently of Fly.io and without requiring HostExec.

### Remote Builder Sprite

Use native PostgreSQL installed directly inside the existing, owner-designated Builder Sprite.

Do not install Docker inside the Sprite.

Verify the actual Sprite environment, Linux distribution, permissions, available PostgreSQL packages, service manager and persistence behavior before selecting the installation approach.

Requirements:

- Match the local PostgreSQL server major version and relevant extensions.
- Use a dedicated development/test cluster.
- Bind PostgreSQL to Sprite loopback only, preferably `127.0.0.1:5434`, preserving existing test endpoint restrictions where possible.
- Manage its lifecycle using Fly's native `sprite-env services`.
- Use separate development/test roles and credentials.
- Support deterministic database initialization, migrations and reset.
- Never use production credentials or customer data.
- Do not expose the database through a public Sprite endpoint.

If the required major version cannot be installed safely using available packages, report the evidence and propose the smallest valid alternative. Do not silently substitute a different PostgreSQL version and call it equivalent.

### Shared development contract

Both environments must support the same meaningful development workflow:

- Same migrations and schema.
- Same test database initialization.
- Same tenant/RLS and transaction testing.
- Same Go test commands and acceptance criteria.
- Same database version and relevant extensions.
- Separate environment-specific connection configuration and service lifecycle.

Do not introduce a second testing architecture just to accommodate Sprites.

Testcontainers-specific tests may remain a separate Docker-capable concern. Do not pretend that native PostgreSQL verifies Docker/Testcontainers behavior.

## 4. Execute the database-unblocking slice

First inspect the current Taskfiles, database configuration, migration system, test endpoint guard, test helpers and actual integration tests.

Implement only the minimum necessary changes to establish a reliable dual-environment workflow.

The acceptance sequence is:

1. Verify the existing local PostgreSQL reference environment and its compatibility with the repository.
2. Verify access to the existing Builder Sprite.
3. Install and configure native PostgreSQL there, subject to applicable owner authorization.
4. Bootstrap an isolated test database.
5. Run representative migrations, integration tests, RLS tests and tenant-isolation tests.
6. Run the complete relevant database-dependent Go test suite.
7. Confirm that test databases and credentials are isolated and cleanup is safe.
8. Record exact commands, outputs, server versions, test counts and failures.

Do not claim a test is green if it was skipped, mocked, inaccessible or not actually executed.

If the Builder Sprite is unavailable or an operation needs additional owner authority, do not provision resources or change external infrastructure without authorization. Report the precise blocker and the smallest approval required.

## 5. Resume the B2 demo immediately after the database path works

Do not turn the database environment into a separate infrastructure project.

Return to the approved B2 implementation plan, particularly the accepted B2-01B baseline and the approved subsequent B2 work.

The central demo must prove the following:

### A. Resident engineering agent

- A real coding agent executes inside the persistent Builder Sprite.
- Studio dispatches a bounded development task to it.
- The agent works in the intended repository and reports actual execution.
- The workspace survives interruption and can resume.

### B. Genuine Fly GitHub Connector

- The agent invokes the actual Fly-provided GitHub Connector from inside the Sprite.
- Demonstrate an actual permitted read operation.
- Do not substitute a PAT, hardcoded fake connector or simulated response.
- Verify the actual permission boundary rather than assuming that read-only access is enforced.

### C. Studio authorization boundary

- The agent can recommend consequential actions.
- Unapproved GitHub writes are refused by the trusted Studio-side broker.
- Issue creation, repair/PR creation and other consequential operations have appropriately scoped authorization.
- Demonstrate both an approved operation and an unauthorized refusal.
- Do not treat a Project board field, chat instruction or agent recommendation as authorization.

### D. Evidence and recovery

- Studio receives execution status, logs, test results and useful evidence.
- Checkpoint creation and restoration are demonstrated where the actual provider supports them.
- The agent must not claim that a checkpoint proves database or business-workflow recovery.

### E. Workspace Preview

When the core workflow is working, demonstrate a real application running in the Sprite and accessible through its Workspace Preview URL.

Do not confuse Workspace Preview with formal Application Preview.

## 6. Demo scope control

Our preferred story is:

1. Create an engineering task in Miniapp Studio.
2. Wake or attach the existing Builder Sprite.
3. Dispatch the actual coding agent.
4. Investigate a real repository or CI problem through GitHub evidence.
5. Prepare a proposed bug Issue.
6. Request and obtain explicit approval for Issue creation.
7. Separately authorize the code repair.
8. Reproduce the defect, repair it and run the relevant tests.
9. Produce a PR only when the required authorization exists.
10. Show the execution evidence and Workspace Preview in Studio.

Use a real, reproducible problem. An existing CI incident may be reused only after verifying that its evidence remains available and its cause is understood.

Do not invent successful outcomes or conceal unsupported capabilities.

The demo must show a working system, not merely screens that simulate one.

## 7. Non-negotiable engineering discipline

Follow the existing governing architecture and Reality-First Engineering principles.

- Investigate root causes rather than patching symptoms.
- Preserve accepted decisions unless actual evidence demonstrates a material defect.
- Do not reopen settled architecture without concrete evidence.
- Prefer the smallest complete vertical slice.
- Build, test, observe and adapt.
- Keep exactly one active implementation slice unless a genuine dependency permits parallel work.
- Do not produce endless review documents or speculative plans.
- Do not ask the Project Owner to perform repetitive terminal, Docker or database operations when an already-authorized agent capability can perform them.
- Do not bypass security, isolation or authorization to meet the deadline.
- Do not perform unapproved cloud, GitHub or destructive operations.
- Keep generated secrets, credentials and local state out of Git.

The three governing dimensions remain independent:

1. Execution category: A, B, C or D.
2. Compute placement: Sprite, Machine or Hybrid.
3. Authorization: dry-run, confirm or autonomous.

Apply these principles when relevant. Do not implement a generalized admission framework during this demo unless a concrete B2 requirement demands it.

## 8. Deadline-driven execution

The Hacksgiving deadline is a hard constraint.

Do not spend the remaining time pursuing technical completeness across B0-B5.

Work in this order:

**First:** unblock the agent and the database testing path.

**Second:** demonstrate one real remote engineering task from Studio to Sprite and back.

**Third:** establish genuine GitHub Connector access and demonstrate the authorization boundary.

**Fourth:** add checkpoint recovery and Workspace Preview.

**Finally:** reserve time for a complete rehearsed demo, failure recovery and submission preparation.

At every stage, identify what can be cut without undermining the central product story.

Do not continue polishing infrastructure while the end-to-end experience remains unproven.

## 9. Reporting format

At the end of each implementation slice, report:

1. What changed.
2. Exact files changed.
3. Exact commands executed.
4. Actual test results, including counts, skips and failures.
5. Which results are real-provider evidence versus mocks/fakes.
6. Security and authorization implications.
7. Remaining demo acceptance criteria.
8. The single highest-impact next action.

Maintain a concise demo acceptance checklist with:

- NOT STARTED
- IN PROGRESS
- VERIFIED
- BLOCKED
- DEFERRED

Every VERIFIED item must have reproducible evidence.

Do not report percentages based on lines of code or amount of planning.

## 10. First action

Start now with a short evidence-based status audit:

- Inspect repository and current branch/worktree.
- Read the accepted B0-B5 roadmap, current B2 plan, approved B2-01B implementation and most recent reviews.
- Inspect the database test configuration and existing local PostgreSQL assumptions.
- Inspect the current Builder Sprite and connector availability without performing unapproved writes.
- Identify the smallest blocker preventing the next real B2 demonstration.

Then begin implementation. Do not return with another broad roadmap proposal.

**Definition of success: Before the deadline, Miniapp Studio demonstrably dispatches and manages a real coding agent inside its persistent Fly Sprite, the agent uses a genuine GitHub Connector, Studio enforces consequential-action authorization, and the resulting work and evidence are visible in Studio.**

Our guiding principle remains:

**Plan enough to act safely. Build. Test. Observe. Learn. Adapt.**

**My instruction alongside this prompt:** don't let the agent report database-environment work as demo completion. The real milestone is the functioning B2 experience. The dual PostgreSQL setup is the means of unblocking it, not the prize-winning deliverable.
