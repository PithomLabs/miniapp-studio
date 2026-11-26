Miniapp Studio — Consolidated Engineering Operating Charter
Architectural decision: ADOPT WITH CORRECTIONS.

I accept the Technical Director role and the operating model described in your proposal. It is consistent with our goal of making Miniapp Studio the control plane for its own development and, eventually, the entire software engineering lifecycle.

However, there are several inconsistencies with decisions we have already made. I want these resolved now so coding agents do not receive conflicting instructions.
1. Authority and responsibility
This is the governance model I will enforce.
Role	Responsibility	Authority
Technical Director (me)	Architecture, sequencing, execution location, environment selection, technical instructions, acceptance requirements	Direct technical work; cannot authorize consequential cloud operations
Project Owner (you)	Business priorities, risk acceptance, consequential operations, commits and pushes	Sole authorization authority
Coding agents	Implement specifically assigned tasks	Only the named scope and permissions
Reviewers	Independently inspect code, tests, database state and evidence	Accept or reject technical claims; cannot authorize operations

I will not leave execution-environment decisions to the coding agent.

Every substantive directive will contain the seven decision-brief fields:
1. Execution location.
2. Application target.
3. Database target.
4. Infrastructure actions.
5. Authorization class.
6. Verification and acceptance evidence.
7. Recovery and rollback.
This becomes mandatory, including for small tasks that touch infrastructure, databases, or security boundaries.
2. Consolidated infrastructure rules
There are two independent axes, and every coding agent must understand the difference.
Axis A — Where development executes
Situation	Decision
Editing Go source, small changes, unit tests	Local Desktop
Fast feedback and ordinary development	Local Desktop
Remote development and persistent agent work	Fly Sprite
Local-to-remote handoff	Fly Sprite
Reproducible dependency environment	Fly Sprite
Running generated or untrusted code requiring broader execution privileges	Fly Sprite
Constrained artifact verification	Existing hardened sandbox, provided its security guarantees are sufficient
Using a Fly Machine as a development workspace	Forbidden

One logical Builder workspace identity per application.

A persistent Sprite can represent that workspace. Additional task-scoped Sprites may be used when isolation is valuable.

The Studio database owns development history. The Sprite is execution infrastructure, not the source of truth.

Important: No untrusted application code executes directly on the Studio or coding-agent host. Our existing hardened Docker sandbox remains appropriate for constrained artifact testing; a Sprite is the stronger execution environment for broader agent-driven work.
Axis B — Where the application runs
Target	Compute	Database	Purpose
Local	Desktop	Local PostgreSQL	Development and rapid testing
Preview	Temporary Fly Machine	Temporary Neon branch	Validate a verified candidate
Staging	Persistent Fly Machine	Persistent Neon branch	Rehearse a release
Production	Protected Fly Machines	Protected Neon production branch	Serve real users

The creation gates are binding:
- Preview requires a verified deployable candidate.
- Staging requires a candidate that has passed Preview.
- Production requires Staging acceptance and explicit human approval.
Machines are application runtime infrastructure. Sprites are development and execution infrastructure. They are not interchangeable.
A terminology rule we must enforce
There are two kinds of preview:
- Workspace Preview: an application or development server exposed from a Sprite.
- Application Preview: an immutable candidate deployed to a Fly Machine with an isolated Neon database branch.
A Sprite URL is never evidence of a release-grade Application Preview.

This distinction is particularly important for our hackathon demonstration.
3. Database constitution
I strongly endorse the database rules in your proposal.

The following are binding:

Migrations live in Git. Every schema change is versioned and applied explicitly to each target environment.

Database branches are isolation, not promotion. Preview, Staging and Production have separate database state. We do not copy Preview or Staging records into Production to deploy software.

Preview data is synthetic or sanitized. A Neon branch is not safe merely because it is a separate branch. Its parent and inherited data must be approved.

Schema compatibility matters. Use expand-and-contract migrations when an application release may overlap with an older runtime.

Testing must own what it mutates. Test code may modify only roles and databases it created with run-unique names. It must not rotate shared roles, alter shared credentials, or reuse someone else's test database.

The last rule is especially important given our earlier PostgreSQL testing and isolation lessons.

Our recorded U5.12 and U5.12.1 work already addressed role separation, run-specific test databases, and endpoint isolation. I will not direct agents to reimplement that work without identifying a new, demonstrable gap. 
4. The immutable artifact rule needs one explicit addition
This is an important architectural requirement that emerged from the review of the unified lifecycle plan.

Our deployment pipeline must preserve the complete chain:
Verified Miniapp application bundle

Source artifact digest


Immutable runtime image

Pinned build process, image digest and provenance


Preview

Fly Machine

Same image digest


Staging

Fly Machine

Same image digest


Production

Fly Machine

Same image digest




The application must be built once and promoted using the same immutable image digest.

No rebuilding separately for Staging or Production.

Release evidence must link the source artifact digest, runtime image digest, Git commit, migration version/checksum, environment, deployment receipt and human approval.

The approval must identify the exact artifact and target environment; where migrations exist, it must also identify the migration set being authorized.

This is now a required correction to the implementation plan. The coding agent's outstanding revision must address it before we grant implementation GO.
5. Adjudication of the immediate directives
Here is my architectural decision on D1–D6.
Directive	Decision	Conditions
D1 — Gateway §J corrections	GO	Local-only, disposable PG, named regressions, probes, full suite and race tests
D2 — Boundary B U5 restructuring	GO	Local-only; two clean-slate passes. Parallel with D1 only with isolated worktrees and database state
D3 — Role separation follow-up	RECONCILE	Compare against accepted U5.12/U5.12.1 evidence before creating another implementation task
D4 — Version control	AWAIT APPROVAL	Agent submits an exact initial commit plan; no commit or push without your GO
D5 — Cloud provisioning	HOLD	Read-only discovery is permitted where authorized; no cloud mutation without separate owner approval
D6 — Hackathon demo	REVISE	The proposed local-only Invoice Review Desk demo is outdated

D6 deserves special attention
Our agreed direction is Miniapp Studio developing itself, using local Muse as the initial development agent.

The demonstration should progressively prove:
1. Studio manages its own development locally.
2. Work transfers to a real Fly Sprite with preserved context.
3. Studio deploys an actual Application Preview to a Fly Machine with an isolated Neon branch.
4. Studio inspects real application logs and drives a troubleshooting cycle.
5. The release moves to Staging, and we demonstrate the Production authorization boundary.
6. A real Production release is shown only when all acceptance and authorization requirements have been met.
Fly Sprites are a central part of this demonstration because they are commercially and architecturally important to the product. They must not exist only in presentation slides.

We will not provision anything merely to make the demo look impressive. We will provision when the capability is ready, useful, and authorized.

The Invoice Review Desk is superseded as the demonstration target. The latest dogfooding decision is Miniapp Studio itself, with Muse—not the previous OpenCode/Invoice Review Desk flow. 
6. Reconcile the Sprite sequence with the implementation roadmap
There are two related but distinct pieces of work:
- Phase B Part 2: prove the underlying live Builder/Sprite execution and recovery contract.
- Bootstrap B3: productize remote workspace continuity in Studio, including handoff, synchronization, ownership and receipts.
I want the underlying Part 2 capability proven before we build the user-facing B3 workflow around it.

The first authorized Sprite operation should therefore be the existing Part 2 live runbook, after D1 verification and your explicit per-operation authorization.

The current unified lifecycle roadmap numbers its milestones:
- B1: one local development task.
- B2: checkpoints and Git awareness.
- B3: Sprite handoff.
- B4: Preview deployment.
- B5: logs and troubleshooting.
- B6: Staging and Production releases.
Use those identifiers consistently. In particular, Preview is B4 in the current plan, not B3. This prevents agents from implementing against inconsistent milestone numbering.
7. The unified engineering lifecycle
This is the process I will direct going forward.
1. Task definition
   
   Studio records objective, scope, acceptance criteria, project identity and development context.
2. Execution assignment
   
   I select Local Desktop or Sprite, specify the target environment and database, and assign the coding agent.
3. Development and checkpointing
   
   The agent works only within its authorized workspace. Changes, progress, Git state and checkpoints are preserved.
4. Verification
   
   Automated tests and independent review produce evidence. Agent assertions do not count as acceptance.
5. Artifact creation
   
   Build and verify the immutable deployable runtime image. Preserve source-to-image provenance.
6. Preview
   
   Use Fly Machine + temporary Neon branch. Verify the candidate and clean up temporary resources when appropriate.
7. Staging
   
   Use persistent Fly Machine + persistent Neon branch. Rehearse migrations, integration, restart and rollback.
8. Production
   
   Obtain your explicit approval for the exact release. Deploy the same image digest, apply the authorized migrations, and verify health.
9. Operations and troubleshooting
   
   Inspect real telemetry, prepare sanitized evidence, return a fix to the development workflow, and repeat the release process.

No phase advances merely because the coding agent reports completion. The acceptance evidence must exist.
8. Standing authorization and safety policy
These are the boundaries I will apply to every instruction.
Operation	Policy
Local source modifications	Authorized within assigned scope
Local disposable tests	Authorized with database ownership safeguards
Read-only Fly/Neon inspection	Allowed within explicit scope
Sprite provisioning or mutation	Project Owner authorization
Preview resource creation/deployment	Project Owner authorization
Staging mutations	Separate Project Owner authorization
Production deployment/rollback/migration	Explicit, operation-specific Project Owner authorization
Destructive cloud or database operation	Deny by default; exceptional explicit authorization
Commits	Separate authorization
Pushes	Separate authorization

An agent cannot expand its own permissions. An approval for Preview cannot be interpreted as approval for Staging or Production.

Likewise, authorization to test an infrastructure capability is not authorization to destroy or recreate unrelated resources.

One correction to the phrase "nothing cloud exists": we should say no cloud mutations are currently authorized, and inventory existing infrastructure through read-only discovery where needed. We should not claim that no resources exist unless we have verified that fact.
9. How I will manage the process with you
Every time you check in, I will use this operating format:

A. Current stream state

What is complete, what is being implemented, what is blocked, and what remains unverified.

B. Technical directives

Each directive includes the seven decision-brief fields and an explicit coding-agent scope.

C. Open authorization requests

I will identify precisely which action needs your approval, its target, expected effect and risk.

D. Risks and architectural decisions

I will flag conflicting instructions, missing evidence, security boundaries, technical debt and dependency changes.

E. Acceptance review

I will review coding-agent reports as claims to verify—not as proof. Failed tests, incomplete evidence, and unresolved safety invariants remain visible.

10. Immediate operating position
Current process posture
Controlled execution

Stream	Direction
D1 Gateway corrections	Local-only
D2 Boundary B matrix	Local-only
D3 Role separation	Reconcile existing accepted evidence
D4 Initial commits	Await owner authorization
Phase B Part 2	On hold until D1 verification and owner GO
B1/B2	Local-first development loop
B3	Real Sprite handoff after prerequisites
B4	First Application Preview on Fly + Neon
B5	Real logs and troubleshooting
B6	Staging and Production release controls


Open cloud authorization requests: none.

Implementation-plan approval: Still pending the immutable runtime-image provenance correction and B2/B3 dependency clarification.

My final architectural position
We will not build Miniapp Studio as a collection of unrelated features. We will build a single, durable engineering process in which every task has a known execution location, every application has a known runtime environment, every database mutation is controlled, and every release has independently verifiable provenance.

The hackathon is an opportunity to demonstrate this architecture, but it does not define its lifetime.

The real product is the engineering lifecycle itself. Fly Sprites, Fly Machines, Neon branches, Go, Miniapp, Studio, and coding agents must work together as one coherent system.

I will enforce that direction consistently, including when an agent proposes something faster that violates an architectural invariant.