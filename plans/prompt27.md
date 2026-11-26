Use this as the next coding-agent directive. It asks for an implementation plan only—no implementation or infrastructure changes.

# Implementation Plan: Issue-Driven, Owner-Governed Engineering Workflow

You are the implementation planning agent for Miniapp Studio. Act as a senior software architect. Produce a practical, minimal implementation plan that incorporates the decisions below into the existing architecture, engineering process, B0–B5 roadmap, and Hacksgiving demo.

**Do not implement anything. Do not modify application code, create GitHub issues, operate Fly or Neon, deploy, commit, or push. This assignment authorizes planning and documentation only.**

## 1. The intended capability

Miniapp Studio should support an issue-driven engineering workflow:

1. A coding agent encounters a defect during its work, investigation, testing, or verification.
2. It decides whether the problem is sufficiently complex to warrant a GitHub issue.
3. It suggests creating an issue only when the problem is relatively complex. Simple, localized problems should normally remain part of the current task.
4. The Project Owner can also explicitly instruct an agent to file a GitHub issue at any time.
5. The agent prepares a useful issue containing observed behavior, evidence, suspected root cause, reproduction details, and acceptance criteria.
6. The issue is created only with the required owner authorization. An agent's recommendation is never authorization.
7. The Project Owner can authorize work on that specific issue from the local Studio or remotely.
8. An authorized coding agent works inside its designated Fly Sprite, investigates the root cause, implements a durable fix, and verifies it.
9. The agent opens a pull request and reports the evidence. Any consequential external action must be within the owner's explicit authorization.
10. An approved candidate can be deployed to formal Application Preview for isolated deployment verification. Production remains untouched unless separately and explicitly authorized.

GitHub Issues remains the system of record for issues and GitHub PRs. Miniapp Studio owns the execution lifecycle, authorization records, progress, and evidence. Do not build a second issue-tracking system.

## 2. Standing owner-authorization principle

This is non-negotiable:

**Agents may recommend, investigate, prepare, and report. They cannot perform consequential actions without Project Owner authorization.**

The plan must define a minimal authorization model that covers at least:

- Creating or modifying GitHub issues.
- Writing code and changing the authorized workspace.
- Pushing branches and creating pull requests.
- Merging changes.
- Creating or modifying cloud resources.
- Deploying to Preview, Staging, or Production.
- Applying database migrations or performing destructive operations.

Authorization must be explicit, attributable, durable, and scoped to the requested issue, repository, action, and environment. One approval must not silently authorize unrelated or higher-impact actions. A direct owner instruction can authorize the specific action it clearly requests; it must not be interpreted as blanket authorization for subsequent actions.

The agent must stop and request authorization when the next consequential action falls outside its granted scope.

## 3. Reproduction, repair, and Preview

Preserve this distinction:

- **Reproduction environment:** The agent first attempts to reproduce the reported failure in an isolated environment that approximates the relevant failing conditions. For CI problems, account for the runner, toolchain, containers, database version, configuration, and test fixtures. Use synthetic or sanitized data.
- **Repair environment:** The coding agent implements the fix inside its Fly Sprite workspace, with checkpoints and recovery.
- **Formal Application Preview:** After verification, build the candidate as an immutable artifact and deploy it to an isolated Fly Machine with a temporary Neon branch. Use this environment to validate the deployed candidate and its acceptance criteria.
- **Production:** Never changed without separate explicit authorization.

Do not claim that Preview perfectly reproduces Production or a GitHub Actions runner. The plan must explain how the workflow reports a failure that cannot be reproduced, or one whose reproduction is only partial.

Preserve the established immutable artifact lifecycle: build once, deploy the same digest through environments, and bind release approvals to the exact artifact and migration set.

## 4. Existing architecture and roadmap constraints

Respect the locked Engineering Operating Charter and the existing Miniapp Studio architecture.

- B0 and B1 are accepted. Do not redesign or regress them.
- B2 is the next planned capability and has not yet been authorized for implementation.
- B2 should incorporate the essential Sprite-resident engineering agent, actual Sprites Connector use, safe owner-authorized GitHub issue workflow, and local-to-Sprite execution handoff.
- B3 owns formal Application Preview and immutable artifact deployment.
- B4 owns troubleshooting and evidence-driven repair.
- B5 owns Staging and Production release governance.
- Keep Miniapp Studio local-first: implement and test control-plane behavior locally before depending on live cloud resources. Use small, explicitly authorized live integration proofs when the local capability is sound.
- Preserve one designated Builder workspace per application and enforce one active writer.
- Use the existing Studio PostgreSQL task, build, event, approval, and evidence records wherever possible. Reuse existing engines. Propose schema additions only when a specific workflow requirement cannot be represented adequately.
- Keep agent execution separate from the Studio control plane.
- The real GitHub Sprites Connector must be used by the coding agent inside the Sprite. A Fly management API or ordinary GitHub API call alone does not satisfy this requirement.
- Do not copy raw GitHub credentials or local credentials into the Sprite.
- Do not introduce a generic issue tracker, unrestricted shell service, multi-agent fleet, or unnecessary workflow engine.

Inspect the current repository and canonical architecture/roadmap documents before proposing changes. Reconcile any stale roadmap numbering or conflicting descriptions against the accepted B0–B5 sequence and current implementation evidence. Do not assume the documentation is already consistent.

## 5. Hacksgiving demo story

Use this real incident as the proposed demonstration starting point:

GitHub Actions run:\
[https://github.com/PithomLabs/miniapp-studio/actions/runs/45666647515](https://github.com/PithomLabs/miniapp-studio/actions/runs/45666647515)

The run failed in `go test -p 1 ./...`. The visible logs include test database provisioning failures reporting an unapproved endpoint port `5432` where `5434` was expected. Treat this as an observed failure signal, not a fully established root cause. Do not fix it as part of this planning assignment.

Design a compelling, honest demo:

1. Show the real failing GitHub Actions run.
2. Have the coding agent inspect the run and its evidence.
3. Have the agent recommend filing a GitHub issue because the problem appears sufficiently complex.
4. Show the proposed issue and its evidence.
5. The owner authorizes issue creation.
6. Show the issue appear in GitHub and Miniapp Studio.
7. The owner authorizes the specific repair scope from a local or remote session.
8. Show the coding agent executing inside its persistent Fly Sprite and using a genuine Sprites Connector.
9. Show reproduction, checkpoint, root-cause investigation, repair, and verification.
10. Show the agent opening a PR and inspecting the resulting CI run, subject to the granted authorization.
11. Show the verified candidate deployed to formal Application Preview, with isolated Neon data.
12. Show an evidence-based report that distinguishes confirmed results from unverified claims. Production remains untouched.

Make the actual Sprite-resident agent and real Connector integration the Hacksgiving deliverable. Miniapp Studio is its command center. Do not imply the pre-existing Studio codebase was built during Hacksgiving week.

## 6. Required planning output

Write a focused implementation plan at:

`docs/architecture/issue-driven-engineering-workflow-plan.md`

Include only:

1. **Architecture and workflow:** The end-to-end path, component responsibilities, and GitHub Connector interaction.
2. **Authorization boundaries:** The minimum states and scoped approvals required to preserve owner authority.
3. **Lifecycle integration:** How the capability fits B2–B5 without duplicating existing task, approval, or evidence systems.
4. **Minimal data/API/UI changes:** Reuse existing structures; identify only necessary changes. Show how the owner sees proposed issues, approvals, execution progress, and final evidence in the existing Focus Console.
5. **Execution sequence:** Small, dependency-ordered implementation slices, with Local Desktop versus Fly Sprite execution and Local versus cloud application targets clearly identified.
6. **Acceptance criteria:** Local tests, connector proof, Sprite execution proof, authorization/refusal tests, recovery tests, CI verification, and Preview verification.
7. **Hacksgiving demo script:** A coherent, achievable sequence with clear evidence at each step.
8. **Risks and explicit exclusions:** The smallest set of meaningful risks, unresolved prerequisites, and features deliberately deferred.

For each implementation slice, state the execution location, application target, database target, infrastructure actions, authorization class, verification evidence, and recovery/rollback.

Keep the plan implementation-ready but deliberately narrow. Prioritize the smallest complete vertical slice that demonstrates the principle. Do not design speculative abstractions or future enterprise features.

## 7. Final instruction

Before writing the plan, inspect the actual repository, existing B0/B1 implementation, current B2–B5 roadmap, and relevant architecture documents. Base recommendations on what exists. Identify conflicts or missing prerequisites rather than inventing capabilities.

At completion, report:

- The plan file created.
- The important existing components reused.
- Any genuine architectural blockers.
- The proposed first implementation slice.

**Planning only. No code, GitHub writes, cloud mutations, deployments, commits, or pushes.**