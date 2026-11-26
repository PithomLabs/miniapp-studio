Got it. I'm taking the Technical Director seat as defined in §1.1 of the constitution: I direct *where work executes, where applications run, and what evidence each step must produce*; you (Project Owner) remain the only authorization authority for consequential cloud operations; coding agents execute only named scope; reviewers verify claims against evidence. Every directive I issue will carry the §22 decision-brief fields — execution location, application target, database target, infrastructure actions, authorization class, verification, recovery — so agents never self-select environments. Here is the charter and the first set of standing directives.

## 1. Standing rule — the one-line version

**Agents build on Local or Sprites; applications run on Fly Machines; candidates prove themselves in Preview; releases rehearse in Staging; users are served in Production — and only immutable, digest-bound artifacts cross those boundaries.**

## 2. Standing decision rules (binding on all agents)

**Axis 1 — Workspace execution (where the agent works):**

| Situation | Execution location |
|---|---|
| Editing, unit tests, quick integration tests, small diffs, fastest feedback | **Local Desktop** (default) |
| Generated/untrusted code must run away from Studio and the agent host | **Fly Sprite** (non-negotiable) |
| Long-running or remotely accessible dev tasks, reproducible deps, checkpoint-before-risky-change, local↔remote handoff | **Fly Sprite** |
| Anything that would use a Machine as a dev host | **Forbidden** — Machines are not workspaces |

One logical Builder workspace identity per application (§3.1); disposable task-scoped sprites where isolation justifies it. Checkpoint before risky refactors or DB experiments; the journal lives in Studio, not on the Sprite.

**Axis 2 — Application runtime (where the app runs):**

| Environment | Compute | Database | Create it when |
|---|---|---|---|
| **Local** | Desktop + local PG (per-run test DBs) | Local PostgreSQL | Always — dev default |
| **Preview** | Temporary Fly Machine(s), expiry policy mandatory | Temporary Neon branch off an approved parent, synthetic/sanitized data | A verified candidate artifact exists and needs smoke/e2e/log validation |
| **Staging** | Persistent Fly Machine(s) | Persistent Neon branch | A release candidate passed Preview and needs rehearsal incl. restart/rollback drills |
| **Production** | Protected Fly Machine(s) | Protected Neon production branch | Explicit human approval bound to the exact digest, after Preview+Staging gates |

**Database rules** (§5): migrations are versioned Git files applied per environment under its authorization class — **branch ≠ promotion**; never copy data across environments to "deploy"; expand-and-contract for schema evolution. The writeup6 lesson is now a standing DB rule: *test code may only mutate roles/databases it created with run-unique names — no shared role rotation, ever.*

**Two meanings of "preview"** (§4.3 — I will enforce this wording everywhere): a Sprite workspace URL is a *workspace preview*, never a release-grade Application Preview. Any agent that labels a Sprite URL as a deployed Preview fails acceptance.

## 3. Immediate directives for the current streams

**D1 — Gateway §J correction bundle: highest priority, Local-only.**
Execution: Local Desktop. Target: none (control-plane library code). DB: local disposable PG (PG16 substitute is acceptable, keep recording the deviation). Infrastructure: **none** — no Sprite, no Neon, no Fly. Verification: the named §J regression tests green + my four probes re-run (P1 single-execution, P2/P3 refused, P4 honest terminality) + full suite and `-race` green. Rationale: this unblocks Part 2; nothing live may be provisioned until the identity boundary (B1) is enforced.

**D2 — Boundary B U5 matrix restructure: Local-only, in parallel.**
Execution: Local Desktop (browser, local Studio server, local PG). Per my writeup6 plan: per-phase scripts with prerequisite preambles, state-ledger file, relative assertions, sabotage test, then **two consecutive clean-slate passes** as acceptance evidence. Standing rule while any live pass is in flight: the PG cluster is single-writer — no `go test` against the live cluster (interim Cause-1 discipline until role separation lands as D3).

**D3 — Follow-up work item, queued behind D1/D2:** split `EnsureAppRole` (create-if-missing+grants | explicit rotate, provision-CLI-only), per-run-unique test roles, red-before/green-after proof: *live server survives a concurrent suite run*.

**D4 — Version control: first authorization I'll request.** The tree is all-untracked with zero commits — the built-during-week disclosure and every evidence gate depend on history. After D1 lands and is verified, I will bring you a commit plan (initial commits, branch layout, CI bootstrap) for your explicit authorization. No push without it.

**D5 — Provisioning posture until further notice: nothing cloud exists, nothing cloud gets created.** No Sprites (Part 2's live runbook R1–R10 is gated on D1's verification + your per-operation authorization). No Neon branches (there is no candidate artifact yet). No Fly Machines (there is no verified, approval-bound release). The first authorized Sprite operation will be the Part 2 live runbook; the first Preview Machine comes at Bootstrap 3, not before.

**D6 — Hackathon demo path stays Local-only** (local Studio + local OpenCode + Invoice Review Desk + verified artifact + human approval), per the locked roadmap. The demo *shows* Sprites/Machines in the narrative; it does not *require* provisioning them for the MVP path. Any request to provision mid-demo-week gets a no from me unless you explicitly override.

## 4. Cadence

When you check in, I'll respond with: current stream states → directives (each in decision-brief shape) → open authorization requests → risks. Agents' completion claims route through me as claims-to-verify, not proof (§17.1); environment wording violations, missing receipts, or invented CLI capabilities get bounced. If any instruction from you conflicts with a constitution invariant, I'll stop and flag it rather than silently reinterpret (§1.2).

Open authorization requests right now: **none** — D1 and D2 require none. The first one will arrive with the D4 commit plan.