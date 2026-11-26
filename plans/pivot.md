# **My recommendation: make Fly Sprites the heart of the hackathon story**

PIVOT RECOMMENDED

I like your direction, but I would sharpen it considerably.

**Don't present Miniapp Studio as a development platform that happens to use Sprites. Present it as a personal AI software engineer whose computer is a Fly Sprite, with Miniapp Studio as its command center.**

That is a much stronger proposition.

There is an important distinction in the official Hacksgiving rules: the entry must be a personal agent that actually runs on a Sprite and uses at least one Sprites Connector. It also has to be built during Hacksgiving week. Merely deploying a website or staging environment on a Sprite is not enough to demonstrate those requirements.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly



There is no published weighted judging rubric, so my recommendations about what will win are strategic judgments, not confirmed judging criteria.

## **1. Your two proposed uses**

### **A. Sprites as remote development environments**

Absolutely essential

This is the strongest use case and the one I would prioritize.

Imagine this:

You open Miniapp Studio locally, describe a development task, and assign it to your resident engineering agent.

The agent lives inside a persistent Fly Sprite, where it has:

- A repository checkout.
- Its own Linux development environment.
- Preinstalled coding agents such as Codex, Claude Code, Gemini CLI or Antigravity.
- Dependencies and development tools.
- Persistent files across sessions.
- Checkpoint and restore capabilities.

Fly documents Sprites as persistent, hardware-isolated Linux computers, with checkpoint/restore and support for coding agents.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Linux computers for agents+1



**Why judges should care:** You're demonstrating that Sprites solve a real problem for AI software engineering: agents need their own persistent computer, not merely another shell session.

### **B. Sprites as Preview/Staging environments**



Preview: yes

Staging: not as the canonical environment



This needs more nuance.

I would absolutely use a Sprite to host a development preview.

For example, while Codex is implementing a Miniapp Studio feature, it starts the application inside its Sprite. The application gets a Sprite URL that you can open in your browser.

This gives you:

- Live inspection of work in progress.
- A remotely accessible application.
- A persistent environment between sessions.
- On-demand wake-up when someone visits its URL.
- The ability to checkpoint or restore the entire environment.

These are documented Sprite capabilities.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Linux computers for agents+1



However, I would not use this as our formal Staging environment.

Why?

A Sprite is a mutable development/execution computer. Our formal release environments must validate immutable application artifacts with controlled database connectivity, migrations, and deployment receipts.

I would therefore establish two types of preview:

| Preview type        | Infrastructure            | Purpose                                 |
| ------------------- | ------------------------- | --------------------------------------- |
| Workspace Preview   | Fly Sprite URL            | Inspect the agent's work in progress    |
| Application Preview | Fly Machine + Neon branch | Validate an immutable release candidate |

Staging and Production should remain Fly Machines with their respective Neon branches.

This is not a compromise. It gives Sprites an additional valuable role without weakening our release architecture.

## **2. Three additional Sprite use cases I strongly recommend**

These are where I think we can make the demonstration substantially more interesting.

### **C. Checkpoint, experiment, break things, restore**

Very compelling demonstration

This could be our signature moment.

The coding agent is about to perform a risky change to Miniapp Studio.

1. It creates a Sprite checkpoint.
2. It upgrades dependencies or performs a substantial refactor.
3. The tests fail, or the application stops working.
4. Miniapp Studio displays the failed attempt.
5. The agent restores the Sprite to the previous checkpoint.
6. It retries using a corrected approach.

The impressive part is that the checkpoint covers the whole Sprite filesystem, including installed packages, files and on-disk services—not merely Git changes.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Sprites · Fly+1



This is a particularly good demonstration of why persistent, recoverable computers are useful for autonomous agents.

It also complements our B1 checkpoint architecture:

- Studio checkpoints preserve the durable development journal and evidence.
- Sprite checkpoints preserve the remote execution environment.
- Git preserves source history.

Three different responsibilities, working together.

### **D. Isolated execution and independent verification**

Strong engineering use case

Consider using a separate disposable Sprite as an independent verification worker.

The main Builder Sprite implements a feature. A second Sprite receives the candidate and performs:

- Full test execution.
- Integration tests.
- Dependency and build verification.
- Security checks.
- Independent acceptance checks.

The reviewer Sprite cannot modify the primary development workspace.

This creates a meaningful separation between the agent that produces the work and the environment that challenges it.

I would eventually use this for adversarial code review and generated-code execution. Fly itself emphasizes isolated execution of agent-generated code.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

fly.io+1



For the hackathon, however, start with one Builder Sprite. Add a second verification Sprite only if the first workflow is already reliable.

### **E. Connectors as secure access to the outside world**

Mandatory for Hacksgiving

This is not optional.

The hackathon requires at least one actual Sprites Connector. I want us to make the connector part of the product story rather than simply satisfy a checkbox.

My first choice is the **GitHub Connector**.

Imagine the agent inside its Sprite:

1. Receives a task from Miniapp Studio.
2. Uses the GitHub Connector to retrieve relevant issue, repository or pull-request context.
3. Examines the source code.
4. Implements the requested change.
5. Runs tests inside the Sprite.
6. Reports the result and evidence back to Studio.

The important security property is that the GitHub credential stays in Fly's connector control plane. The Sprite makes the API request through the connector without receiving the raw token. Connector access can be restricted to specific Sprites and API paths.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+2



Later, we can introduce a custom Miniapp Studio Connector so the agent can request task packets and submit progress/evidence directly through Studio's API. That would be even more integrated, provided we expose a securely reachable endpoint.

**Do not confuse Fly MCP with a Sprites Connector.** Fly MCP is useful for managing Fly resources, but the hackathon specifically asks for a Sprites Connector. We should visibly demonstrate an actual GitHub, Slack, OpenRouter or Custom API connector call.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+1



## **3. The architecture I would pitch**

This is the architecture I believe best combines your existing product direction with Fly's value proposition.

## **Miniapp Studio**

**Local engineering cockpit**

Tasks · conversations · approvals · evidence · project history

**Persistent Builder Sprite**

Hackathon centerpiece

## **Resident Engineering Agent**

Preinstalled Codex / Claude Code / Gemini CLI / Antigravity

**Repository**

Source and dependencies

**Checkpoints**

Rollback and recovery

**Build and test**

Isolated execution

**Workspace Preview**

Live Sprite URL

**Sprites Connectors**

GitHub initially · Custom Studio API later · Slack or other APIs when justified

Credentials remain outside the Sprite.

**Verified immutable application artifact**

Source bundle → OCI image digest → provenance

**Fly Machine**

Application Preview

**Neon**

Isolated Preview branch

There is an important long-term design distinction here: Fly's own agent architecture describes the agent's persistent home separately from the isolated environment where generated code executes. We should preserve this separation when we evolve beyond the initial single-Sprite demonstration.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

fly.io+1



## **4. The hackathon demo I would build**

I would give the entry a more distinctive identity:

**Miniapp Studio — The Self-Developing Software Engineer**

The pitch:

*A personal AI software engineer that lives inside its own persistent, recoverable computer. Miniapp Studio directs its work, Fly Sprites give it a secure place to execute, and Connectors give it controlled access to the outside world.*

Here is the demonstration sequence I would aim for:

1. **Start locally**

   Open Miniapp Studio. Create a real task for Miniapp Studio itself. Show its existing durable task history.
2. **Wake the resident engineer**

   Dispatch the task to the persistent Builder Sprite. Show the actual agent process running inside the Sprite, not merely a local agent controlling a remote shell.
3. **Use a real Connector**

   Have the agent retrieve repository or issue context using the GitHub Connector. Show that it uses the real connected service without storing its raw credentials in the Sprite.
4. **Demonstrate checkpoint and restore**

   Checkpoint before a risky change, deliberately demonstrate an unsuccessful attempt, restore, and recover.
5. **Show workspace preview**

   Start Miniapp Studio or its feature branch inside the Sprite. Open its live Sprite URL and inspect the running application.
6. **Return evidence**

   Bring the Git state, test results, agent report and checkpoint references back into Studio. Have the human review and accept the result.
7. **Graduate to application Preview**

   As the next step, deploy the verified immutable image to an actual Fly Machine with a Neon Preview branch, demonstrating the difference between development preview and release preview.

This tells a much better story than simply saying we use a Sprite as a remote staging server.

## **5. Pros and cons of the pivot**

| Strategic choice                | Advantage                                                                  | Risk                                              |
| ------------------------------- | -------------------------------------------------------------------------- | ------------------------------------------------- |
| Persistent Builder Sprite       | Directly showcases Sprites' core strengths                                 | Remote handoff complexity                         |
| Sprite workspace preview        | Instantly visible result, easy to share                                    | Can be confused with release Preview              |
| Checkpoint/restore              | Memorable demonstration of recoverability                                  | Needs a deliberately prepared scenario            |
| GitHub Connector                | Satisfies mandatory connector requirement with genuine engineering utility | Connector permissions must be carefully scoped    |
| Disposable review Sprite        | Strong isolation and independent verification                              | Extra implementation and demonstration complexity |
| Fly Machine application Preview | Proves the complete development-to-deployment story                        | Larger artifact and database integration scope    |

## **6. What I would prioritize given the deadline**

The official deadline is midnight PST, Sunday, November 29, 2026. The requirements include building the agent during Hacksgiving week, using a Sprite and at least one Connector, and submitting the code and demo link.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly



I would prioritize as follows:

| Priority | Capability                                             | Hackathon importance                   |
| -------- | ------------------------------------------------------ | -------------------------------------- |
| P0       | Personal coding agent actually running inside a Sprite | Essential                              |
| P0       | Real Sprites Connector integration                     | Essential                              |
| P0       | Persistent workspace and checkpoint/restore            | Essential differentiator               |
| P1       | Local Studio → remote Sprite task workflow             | Very high                              |
| P1       | Live Sprite workspace preview                          | Very high                              |
| P2       | Isolated independent reviewer Sprite                   | Optional                               |
| P2       | Fly Machine + Neon Application Preview                 | Stretch goal                           |
| P3       | Full Staging/Production lifecycle                      | Product roadmap, not hackathon blocker |

One compliance concern: because the rules say the project must be built during Hacksgiving week, make the new Sprite-resident personal agent and its Connector integration the identifiable Hacksgiving deliverable. Miniapp Studio is the existing host/control plane that gives it a useful real-world purpose.

## **7. My final strategic recommendation**

I would make three decisions:

**Decision 1 — Keep Miniapp Studio local as the command center.**

No need to deploy Studio itself to the cloud just to prove Sprite integration.

**Decision 2 — Make the persistent Sprite-hosted coding agent the centerpiece.**

Use one real coding-agent runtime inside the Sprite, one real Connector, real checkpoints, real code changes and a visible result. Don't build a fleet.

**Decision 3 — Use Sprite URLs for development previews, Machines for release previews.**

Do not call a Sprite Staging or Production. Preserve the formal environment model.

My most important advice: judges should leave remembering **why an AI engineer needs its own persistent, recoverable, hardware-isolated computer**, not merely that you managed to run a coding agent in a VM.

That is the differentiator I would bet on.

The immediate architectural next step is to revise B2 into a hackathon-focused **Resident Engineering Agent + Builder Sprite + Connector** vertical slice, while retaining the full Local → Remote → Preview → Staging → Production roadmap.


## prompt

I like the GitHub connector thing, right now there is a real issue on GitHub Actions [https://github.com/PithomLabs/miniapp-studio/actions/runs/45666647515](https://github.com/PithomLabs/miniapp-studio/actions/runs/45666647515), how can we incorporate the process where for example I want the coding agent to file that issue on GitHub Issues, then I can tell it locally or remotely that it can work on it, fix it and deploy it then report to me if it has indeed fixed the issue, got what I mean? we do not need to fix it right away because I want to highlight that in the demo video story but give me the high level idea on how this will work out


## answer

Yes. You are describing an **issue-driven autonomous engineering workflow**: an agent that can discover a problem, turn it into a tracked GitHub issue, wait for your authorization, work on it in its own remote development environment, and prove whether the fix actually worked.

The important idea is that **GitHub Issues becomes the work queue, Miniapp Studio becomes the control plane, and a Fly Sprite becomes the agent's computer.**

I inspected your linked Actions run. It is run **#4**, workflow `ci`, on `main`, at commit `5942fef8`, and it failed on November 25, 2026. The `gofmt`, `go vet`, and `go build` steps passed; `go test -p 1 ./...` failed. The visible failures report that test database provisioning rejected endpoint port `5432`, expecting `5434`. That is a useful real-world incident for the demo. No changes have been made. [Open the run](https://github.com/PithomLabs/miniapp-studio/actions/runs/45666647515?utm_source=chatgpt.com).

## **1. The user experience**

Imagine this sequence in Miniapp Studio:

**Miniapp Studio — Focus Console**

Connected to GitHub

INCIDENT DETECTED

**GitHub Actions: CI verification failed**

`PithomLabs/miniapp-studio`

Failed step: `go test -p 1 ./...`

Source: Actions run #4 · main · 5942fef8

Suspected failure: test database endpoint port mismatch

Agent action

**I investigated the failed run and prepared a GitHub issue.**

The issue includes the failing job, relevant log excerpts, commit SHA, and reproduction context.

**Proposed GitHub Issue**

`CI tests fail because test database provisioning rejects port 5432`

Awaiting approval

Not yet authorized to modify code

**What should I do?**

Investigate

Fix

Test

Prepare deployment

No code changes or deployments occur just because the issue was created.

**You — from your local computer or remotely**

"Approved. Work on issue #123. Fix it, verify the tests, and deploy to Preview. Do not deploy to Production."

Authorized execution

**The agent starts work in its persistent Fly Sprite.**

It reports progress, asks questions when necessary, and keeps evidence in Miniapp Studio.

The issue number above is illustrative, not an actual GitHub issue.

## **2. How the workflow operates**

I would structure it into eight stages.

1. **Detect**

   The agent receives an incident from GitHub Actions, or you ask it to inspect a run. The GitHub Connector retrieves run metadata, failed jobs, steps, logs, and relevant repository context.
2. **Investigate and create an issue**

   The agent analyzes the evidence, identifies likely root causes, and drafts an issue with reproduction details, suspected causes, impact, and acceptance criteria. It creates the issue using the GitHub Connector.
3. **Wait for your authorization**

   The issue is visible in Miniapp Studio. You can approve it from the local console or remotely. Approval is recorded against that specific issue, repository, scope, and requested actions.
4. **Dispatch to the Sprite**

   Miniapp Studio hands the task to the persistent Builder Sprite. The coding agent runs inside that Sprite, retrieves the issue through the actual GitHub Connector, checks out the relevant repository revision, and works in an isolated branch.
5. **Fix and verify**

   The agent investigates the underlying problem, implements a durable fix, runs targeted tests and the complete required verification suite, and records the outputs. It should not declare success just because the code compiles.
6. **Open a pull request**

   The agent pushes its branch and opens a PR linked to the GitHub issue. CI runs again. The agent inspects the new Actions results and responds to failures rather than treating the existence of a PR as proof of success.
7. **Deploy to Preview**

   Once the candidate passes the release gates, Miniapp Studio builds an immutable artifact and deploys it to a Fly Machine with an isolated Neon Preview branch. You authorize the deployment. This is formal Application Preview, not merely the mutable Sprite workspace preview.
8. **Report verified outcome**

   The agent checks deployment health, application behavior, relevant logs, and CI status. It reports exactly what succeeded, what remains unverified, and links to the issue, PR, commit, Actions run, and deployment evidence.

## **3. The critical design: approval is not just a chat message**

The system should distinguish these states:

| State                  | Meaning                                  |
| ---------------------- | ---------------------------------------- |
| Detected               | An incident has been discovered          |
| Issue proposed         | Agent prepared an issue                  |
| Issue filed            | GitHub issue exists                      |
| Awaiting authorization | Agent cannot modify code                 |
| Authorized             | Owner approved a defined task            |
| In progress            | Sprite agent is working                  |
| Verification           | Tests and independent checks are running |
| PR review              | Candidate awaits review/merge            |
| Preview deployed       | Immutable candidate is running           |
| Verified               | Acceptance criteria have evidence        |
| Failed / blocked       | Agent must report the reason             |

This is important because you may approve a fix without approving a deployment. Likewise, approving Preview must not silently authorize Staging or Production.

A single approval should be scoped to a specific issue and action. A later change in scope, a different target environment, or a production deployment requires another authorization.

## **4. How the three main components cooperate**

**You — Project Owner**

Local Desktop or remote browser · Approvals and decisions

**Miniapp Studio**

Control plane · Task state · Authorization · Evidence · Dispatch · Release gates

**GitHub Connector**

Issues, repository context, PRs, Actions and CI results

**Fly Sprite**

Resident coding agent, repository, tools, tests and checkpoints

**Release pipeline**

Verified bundle → immutable image → Fly Machine → isolated Neon Preview

**Evidence and report**

CI results · commit SHA · PR · deployment digest · health checks · acceptance status

One important security detail: the Sprite should not receive your raw GitHub credentials. Use the Sprites Connector as the controlled external-access mechanism, with explicit permissions and default-deny access. Miniapp Studio should remain the durable record of approvals and execution receipts.

## **5. What makes the final report trustworthy?**

The agent must distinguish between these claims:

- **Code fixed:** the implementation change is present.
- **Tests passed:** the required test suite passed on a specific commit.
- **CI passed:** GitHub Actions completed successfully for that commit.
- **Preview deployed:** the exact approved image digest is running.
- **Issue resolved:** the issue's acceptance criteria were met and independently verified.

It must never collapse these into one vague green "Done" status.

For example, the final report might say:

**Engineering completion report**

Verified

**Issue #123 — CI test database provisioning failure**

- Root cause: confirmed and documented.
- Fix: committed on `fix/issue-123-db-port`.
- Tests: full required suite passed.
- GitHub Actions: successful on the fix commit.
- Pull request: linked and awaiting your review.
- Preview: deployed using immutable image digest.
- Production: untouched.

Illustrative report; not a claim that the real issue has been fixed.

## **6. How I would fit this into Miniapp Studio's roadmap**

I would not turn this into a separate issue-tracking product. GitHub remains the source of truth for issues and pull requests; Miniapp Studio owns the execution lifecycle and evidence.

The roadmap connection is:

| Roadmap                           | Contribution                                                     |
| --------------------------------- | ---------------------------------------------------------------- |
| B0 — Local task management        | Existing task and execution records                              |
| B1 — Local development loop       | Existing checkpoints, recovery, and evidence                     |
| **B2 — Sprite engineering agent** | Remote execution, GitHub Connector, issue authorization, handoff |
| B3 — Preview deployment           | Immutable artifact and Fly Machine + Neon Preview                |
| B4 — Troubleshooting              | Observe CI and runtime failures, investigate and repair          |
| B5 — Release management           | Staging, Production, approvals, rollback                         |

I would make the demo's headline capability the complete journey from a real failure to a verified Preview fix, even though the initial implementation can be staged.

**My recommendation for the demo story:** start with the red GitHub Actions failure, show the agent investigate and file the issue, then deliberately stop at the approval boundary. Approve the issue from another device or session. Show the resident agent waking inside its Fly Sprite, creating a fix, opening a PR, watching CI turn green, deploying the candidate to Preview, and returning a verifiable report.

That tells a much stronger story than simply showing an AI agent writing code: **you have an engineering agent that accepts work, respects authority, operates remotely, and can prove the result.**



## prompt

just to clarify, when you say "deploy to Preview" the preview environment will try to replicate the environment and the issue, and from there it will  try to fix it without deploying live to prod, is that right? brief reply only


## answer

Almost. The distinction is:

1. **Reproduce:** The agent attempts to recreate the failure in an isolated environment, using equivalent configuration and synthetic or sanitized data.
2. **Fix:** The agent modifies the code inside its Fly Sprite and runs tests.
3. **Preview:** It deploys the fixed candidate to an isolated Fly Machine + Neon Preview branch, then verifies the fix under deployment-like conditions.
4. **Production:** Remains untouched. A separate explicit approval is required to release there.

So Preview is the **safe proving ground for the proposed fix**, not where the agent initially writes the fix. It should reproduce the relevant conditions, but we must not assume it perfectly replicates Production.



## prompt

that is exactly what I want, let us generalize this into agents filing bugs to GitHub Issues itself but only 1) if coding agents suggest if the issue is relatively complex or 2) if I say it myself but in either case, agents cannot do consequential actions without my authorization (that is my standing principle), let us get to business, give prompt for agent to write implementation plan based on my last turn where I asked you to lock it in up to this moment, remember to put only the absolute essentials that we can incorporate into our process and to Miniapp Studio itself and the hackathon demo


## answer

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

GitHub Actions run:
https://github.com/PithomLabs/miniapp-studio/actions/runs/45666647515

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




