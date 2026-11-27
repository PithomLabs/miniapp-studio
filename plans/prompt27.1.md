# Coding Agent Directive — Execute Prompt27-R1

## Mission

Execute the Project Owner's directive published in:

https://github.com/PithomLabs/miniapp-studio/discussions/2

Repository: `PithomLabs/miniapp-studio`

GitHub Project: [Miniapp Studio Engineering](https://github.com/orgs/PithomLabs/projects/3)

Act as the senior implementation-planning agent. Investigate the actual repository, reconcile architecture and implementation evidence, and deliver the approved planning document.

**This assignment authorizes planning and read-only investigation only. It does not authorize implementation or consequential GitHub, Fly, Neon or production actions.**

## 1. Read the governing context first

Before acting, read:

1. GitHub Discussion #2 in full.
2. Repository `SKILLS.md`.
3. Repository `AGENTS.md`.
4. Canonical architecture documents and ADRs.
5. Current B0–B5 roadmap and accepted implementation evidence.
6. Relevant B0/B1 code, tests and documentation.
7. Current GitHub Project configuration.

Follow the latest Project Owner instructions and the repository's Engineering Constitution.

Do not rely on old chat summaries or stale planning documents when actual repository evidence is available.

## 2. Understand the established project-management model

The Project Owner has established the following division of responsibilities:

- **GitHub Discussions:** Feature initiatives, architectural proposals, design decisions, product direction and milestone discussions.
- **GitHub Issues:** Possible or confirmed bugs and defects only.
- **GitHub Projects:** Small, independently verifiable implementation Task Cards derived from accepted Discussions and implementation plans.
- **GitHub Pull Requests:** Proposed source-code changes and technical reviews.
- **Miniapp Studio:** Execution coordination, durable authorization records, agent activity, evidence and acceptance.

Do not create feature Issues merely to group implementation work.

A Discussion is the initiative record. An accepted implementation plan is the specification. Project Task Cards are the executable work packages.

## 3. Current GitHub Project configuration

The owner has configured Project #3, `Miniapp Studio Engineering`.

Verify its actual configuration read-only. The following is the owner-reported and screenshot-verified baseline:

### Access and governance
- Visibility: Private.
- Organization base role: No access.
- Project Owner: Admin.
- Make template: Off.
- Project README: Saved.
- No task cards have been created.

### Views
1. Execution Board — Kanban grouped by Status.
2. Initiative Roadmap — Table grouped by Workstream, sorted by Task Priority.
3. Owner Inbox — Proposed, Blocked and In Review tasks.
4. Agent Queue — Ready tasks assigned to Local Agent or Sprite Agent, with Authorization = Approved.

### Status options
- Proposed (default)
- Backlog
- Ready
- In Progress
- Blocked
- In Review
- Done
- Deferred

Moving an item to Ready is not authorization.

### Project fields
- Workstream: B2, B3, B4, B5.
- Executor: Human, Local Agent, Sprite Agent, Reviewer.
- Execution Location: Local, Sprite.
- Application Target: None, Local, Workspace Preview, Application Preview, Staging, Production.
- Authorization: Not requested, Pending, Approved, Refused, Expired, Revoked.
- Initiative: Text reference to originating GD.
- Task Priority: P0 — Critical, P1 — High, P2 — Normal, P3 — Low.
- Existing `Priority` Issue Field remains separate and is for bug triage, not feature Task Cards.

### Workflows
The owner has disabled the default workflows to prevent unintended state synchronization, automatic Issue closure, automatic task intake and automatic completion based on PR events.

Verify all default workflows are disabled. Do not change them.

If any discrepancy exists, report it; do not mutate the Project settings.

## 4. Execute the original Prompt27-R1 procedure

Preserve its P0–P6 sequence.

### P0 — Baseline and preserve
- Verify Git status and current commit.
- Confirm the target document does not already exist.
- Verify the permitted one-new-file boundary.
- Stop and report if the tree is unexpectedly dirty or the target already exists.

### P1 — Repository and architecture investigation
Inspect actual:
- B0/B1 implementation and tests.
- B2–B5 roadmap.
- Engineering Constitution and accepted ADRs.
- Connector interfaces and real/fake coverage.
- Approval and authorization mechanisms.
- Focus Console extension points.
- Current GitHub Project integration capabilities.
- Discussion and Issue integration capabilities.
- GitHub Actions CI and local reproduction setup.

Investigate the actual available GitHub integration for:
- Reading Discussions.
- Discovering Discussion categories.
- Preparing and creating Discussions.
- Reading and creating Project V2 Draft Items.
- Reading and updating Project fields.
- Reading workflow and PR evidence.

Distinguish ordinary GitHub integration capabilities from the genuine Fly Sprites Connector.

Do not claim a capability works merely because its API exists. Identify what is available in the actual connected agent environment, including required permissions.

### P2 — Read-only incident investigation
Inspect:

https://github.com/PithomLabs/miniapp-studio/actions/runs/45666647515

Establish confirmed facts and unknowns. The observed port mismatch is not an established root cause.

Do not diagnose or fix the incident as part of this assignment.

### P3 — Architecture and process reconciliation
Reconcile:
- Product vision.
- Current B0–B5 roadmap.
- Prompt27-R1 requirements.
- Issue versus Discussion responsibilities.
- Project Task Card lifecycle.
- Owner authorization boundaries.
- Builder Sprite and Fly Machine separation.
- Workspace Preview versus formal Application Preview.
- Hacksgiving scope and priorities.

Resolve conflicts using the latest accepted owner decisions and actual evidence. Document unresolved questions.

### P4 — Produce the planning document
Create exactly:

`docs/architecture/issue-driven-engineering-workflow-plan.md`

Follow the eight substantive sections specified in Prompt27-R1:

1. Product vision, architecture and end-to-end workflow.
2. GitHub Issues and Discussions intake and classification.
3. Authorization boundaries and refusal behavior.
4. Project memory, milestone governance and B2–B5 integration.
5. Minimal data, Connector/API and Focus Console changes.
6. Dependency-ordered implementation slices.
7. Acceptance evidence and Hacksgiving demo script.
8. Risks, prerequisites and explicit exclusions.

## 5. Add the proposed Kanban execution breakdown

This is an essential addition to the planning deliverable.

The implementation plan must include a proposed set of small, dependency-ordered Project Task Cards.

These are proposals for owner review, not actual Project items.

For each proposed card include:

- Unique identifier.
- Outcome-oriented title.
- Source Discussion #2.
- Exact implementation-plan section and revision.
- Scope and explicit exclusions.
- Dependencies.
- Workstream.
- Task Priority.
- Executor.
- Seven execution-planning fields:
  1. Execution location.
  2. Application target.
  3. Database target.
  4. Infrastructure actions.
  5. Authorization class.
  6. Verification/acceptance evidence.
  7. Recovery/rollback.

Tasks must be independently verifiable and small enough to execute and review as separate work packages.

Do not arbitrarily divide work by file count or produce excessive microtasks.

Identify:
- Which tasks can execute independently.
- Which tasks depend on other tasks.
- Which require owner clarification or approval.
- Which are local-first.
- Which need a live Sprite or Connector proof.
- Which belong to B3, B4 or B5 and must remain deferred.

The plan should contain copy-ready proposed Task Card descriptions so the Project Owner can approve the breakdown before anything is created in GitHub.

Do not create Project items or Draft Issues.

## 6. Standing authorization principle

Agents may investigate, recommend, draft and report.

They cannot independently authorize consequential actions.

Planning approval does not authorize implementation. Implementation approval does not authorize cloud changes. Preview authorization does not authorize Staging or Production.

Every consequential action must be attributable, explicit, durable and scoped to its exact target and operation.

The plan must explain how Miniapp Studio enforces this independently of Project status fields.

## 7. Preserve the original Hacksgiving story

The product north star remains:

**Miniapp Studio — The Self-Developing Software Engineer.**

A personal AI software engineer living inside its own persistent, recoverable Fly Sprite, directed by Miniapp Studio and connected to the outside world through genuine Sprites Connectors.

The centerpiece is:
- Actual coding agent executing inside a persistent Sprite.
- Genuine GitHub Sprites Connector interaction.
- Local Studio to remote Sprite handoff.
- Checkpoint, deliberate failure and restoration.
- Sprite Workspace Preview.
- Evidence returned to Miniapp Studio.

The real CI incident is the demonstration mission, not a replacement product story.

Fly Machine + Neon formal Application Preview remains a stretch capability for the hackathon and B3 roadmap work. Do not make it a prerequisite that compromises the resident agent and Connector demonstration.

## 8. P5 — Self-verification

Before delivering:
- Every existing-capability claim has repository or test evidence.
- Proposed functionality is explicitly labeled proposed.
- Unknown Connector capabilities are explicitly identified.
- All eight document sections are present.
- Every proposed implementation slice has all seven required fields.
- Every consequential action has an explicit approval and refusal path.
- Project Task Cards are proposals only.
- Issues remain reserved for possible or confirmed bugs.
- No duplicate task, approval, evidence or workflow engine is proposed without demonstrated necessity.
- The Hacksgiving story remains Sprite-centered.
- The real CI incident is not treated as diagnosed.
- Only one new document is created.

## 9. P6 — Delivery and stop

Report exactly:

1. Planning document created.
2. Important existing components reused.
3. Genuine architectural blockers and unverified integration capabilities.
4. Proposed first implementation slice.
5. Number and summary of proposed Project Task Cards.
6. Exact authorization required for the next step.

Then stop and await owner review.

## Seven-field execution directive

For this assignment:

- Execution location: Local Desktop.
- Application target: None; planning only.
- Database target: None; no database writes.
- Infrastructure actions: Read-only inspection only.
- Authorization class: Planning and read-only inspection.
- Verification evidence: Repository, documentation, tests and read-only integration evidence.
- Recovery/rollback: No mutations; preserve the existing working tree.

**Strict exclusions:** No application implementation, migrations, task-card creation, GitHub Issues or Discussions writes, Project changes, PRs, commits, pushes, Fly/Neon mutations, deployments or incident fixes.

This directive authorizes only the planning deliverable described above.