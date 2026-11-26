## question

per adv_review_prompt10.md what phase are we in now and how severe is the review

## answer

You are currently in **Phase B, Part 1** (the Sprite Agent Gateway build/verification stage), with **Part 2** (live Sprite & recovery) explicitly **not authorized** yet.

The review severity is **NOT ACCEPTED — 1 Blocker, 8 Majors, and 16 Minors**. While the core protocol, atomic persistence patterns, and tenant/Studio boundaries are solid, there is a structural blocker and four proven adversarial exploits that must be fixed before Part 2 can proceed.

---

### Where You Stand Right Now

* **Current phase:** **Phase B — Sprite Agent Gateway (Part 1)**. Phase A is already accepted, while Phases C–E remain ahead.


* **Part 2 gate:** **Blocked.** Part 2 requires landing the bounded correction bundle in §J and passing its regression tests and re-probes; no separate full re-review is needed afterward, only verification of those fixes.



---

### Severity Summary

#### 1 Blocker (Structural Failure)

* **B1 — Missing OS-identity boundary (`gateway/supervisor.go`, `fly/interfaces.go`, `tools/build_provision.go`, `tools/provisiondata/setup.sh`):** The `studio-gw` and `studio-agent` users exist, but no code drops privileges or assigns users when spawning agents or provisioning. Worse, the file-drop directory/permission model as written cannot work securely under either single-user or dual-user uid assignments, and `0644` staging copies of keys linger in `/srv/studio-staging`.



#### 8 Majors (Including 4 Proven Exploits)

* **F2 & F3 — Symlink attacks on file-drop Q&A (Proven):** `ReadFile` on `question.json` follows symlinks (allowing key/file exfiltration into callbacks), and `WriteFile` on `answer.json` truncates through symlinks without `O_EXCL`/`O_NOFOLLOW` (allowing target file clobbering).


* **F4 — Concurrent duplicate command execution (Proven):** Check-then-act/launch without an in-progress claim or lock allows concurrent identical continues to return `200` and invoke the adapter twice.


* **F5 — Premature terminal failure on mid-run errors (Proven):** Any mid-run `error` line immediately emits `EventFailed` even if the run recovers and exits cleanly (`0`), causing false terminal failures.


* **F6 — Dead end after recovery:** Recovery marks builds failed, but subsequent assign/continue calls hit `409`s with no re-drive path—contradicting runbook §11.5.


* **F7 — Shared-builder service restarts:** Provisioning deletes/recreates the service unconditionally, restarting the gateway and orphaning co-located builds' agents.


* **F8 & F9 — Callback durability & freshness gaps:** Terminal outbox failures have no requeue/backstop path (the cited §16 poll does not exist), and envelopes are signed once at emit (`SentAt`), so a ~6-minute outage pushes redeliveries past the 5-minute skew window and causes `401` → permanent failure.



#### 16 Minors & 10 Live Risks (R1–R10)

* **Minors (M1–M16):** Covers missing directory `fsync` after rename, `http.DefaultClient` with no timeout in the deliverer, `SIGTERM` handling/process groups/undrained stderr (`>64KB` hang risk), collector TOCTOU window, stale-turn gateway fencing, CLI/provenance polish, and untested `KDFProvider` round-trip.


* **Live Risks (R1–R10):** Live OpenCode execution, real Sprite behavior, wake/sleep timing, OAuth ceremony, and unvendored `run.ts` @ v1.19.6 claims remain unverified until Part 2 checkpoints.



---

### Recommended Fix Order

1. **Resolve B1 + F2/F3 together:** Lock in the uid/group/sticky-bit/key-cleanup design (`studio-gw` vs. `studio-agent`, `2730` inbox, `0640` answers, `O_NOFOLLOW` + `O_EXCL` + size caps) and re-run the P2/P3 symlink probes.


2. **Fix F4, F5, F6, F8, F9:** Add per-build/session concurrency claims, base terminality on exit code + question drop rather than mid-run error lines, add post-recovery resume (`resumed`), re-sign callbacks per attempt, and add outbox inspect/requeue.


3. **Fix F7 + M1–M16 + doc updates:** Avoid restarting active shared builders during provisioning, close out the minor items and evidence/runbook mismatches (G1–G8, §11.5, §12), and re-run P1–P4 plus the full test/race suite before starting Part 2.


## question

just to clarify, what we are building for Miniapp Studio is basically it acts as scaffolding only, the real deal is actual deployment to Fly.io Machines which are different environment from here, makes sense? so why all the fuss

## answer

Not quite—Miniapp Studio isn't just disposable scaffolding before Fly.io Machines, and the review isn't nitpicking hypotheticals. First, **Miniapp Studio is the control plane / orchestrator itself**, and second, **the findings hit real code that actually runs on the target environment** (and the review deliberately sets aside later Fly.io runtime deployment work for Phase E).

### 1. Miniapp Studio Isn't Just Scaffolding

* **It is the system of record and control plane:** Studio owns the business conversation, approvals, audit trail, and state machine (Invariant 1), while users interact with Studio rather than raw terminals or cloud consoles (Invariant 4).
* **The gateway is a real long-lived service, not throwaway glue:** The `gateway` (`cmd/gateway` + `gateway/`) is the service running on the **Builder Sprite** that receives signed commands from Studio, supervises the coding agent, handles file-drop Q&A, collects artifacts, and sends signed callbacks back to Studio.
* **Actual Fly.io Machine / Runtime Sprite deployment comes later:** Your intuition about a separate production runtime environment is right—that's the **Runtime Sprite** (Phase E), which gets *only* approved, digest-verified artifacts, with no coding agent or gateway left on it. The review explicitly deferred runtime-target / Phase E concerns rather than holding Part 1 to them.

### 2. Why "It Runs in Another Environment" Doesn't Excuse the Fuss

* **B1 isn't a local-test quirk—it's a broken security design in the target setup:** `setup.sh` already creates `studio-gw` and `studio-agent` and `0600` key files, and `builder.md` promises keys are unreadable to the agent, yet no code drops privileges when spawning the agent. Worse, the file-drop directory and permission layout as written breaks under *any* uid setup: single-user lets the agent read HMAC keys and forge callbacks, while dual-user blocks the agent from writing questions or reading answers.
* **F2 and F3 are direct filesystem attacks on the gateway's own files:** Even inside a sandbox or container, if the gateway reads `question.json` following symlinks (`F2`) or writes `answer.json` with a truncating `WriteFile` without `O_EXCL`/`O_NOFOLLOW` (`F3`), an untrusted agent process can trick the gateway into reading sensitive files into callbacks or clobbering files at agent-chosen paths.
* **F4–F9 are logic, concurrency, and durability bugs:**
* **F4:** Two identical concurrent continues can both succeed and run the adapter twice.
* **F5:** A recoverable mid-run error line can mark a run `failed` even when the process exits cleanly (`0`).
* **F6 & F7:** Recovered builds hit a dead end (`409`s with no resume path), and provisioning restarts the service on shared builders—killing co-located builds' agents.
* **F8 & F9:** Outbox items that hit terminal failure have no requeue path, and envelopes signed once at emit expire past the 5-minute skew window during a ~6-minute outage—permanently killing queued callbacks.



### 3. Why the Review Is Actually Good News (Not a Rewrite)

The reviewer specifically noted that **none of this requires a redesign**. The core protocol, atomic persistence pattern, supervisor basics, and Studio-side tenant checks are already in good shape. Everything blocking Part 2 is a bounded fix bundle (§J) that can be verified directly with regression tests and re-running probes P1–P4—without needing another full review cycle.

If you want, we can map out how the **Builder Sprite** (`gateway` + coding agent) and **Runtime Sprite** (Phase E) divide responsibilities so it's crystal clear what lives where.



## question

do you see security lapses or lack of security fundamentals from the implementation agent per the review above, the goal is to uncover insights that the agent might be lacking so we have a robust methodology from this point on


## answer

Yes—the review shows clear gaps in **applied security fundamentals**, particularly around privilege enforcement, confused-deputy filesystem handling, key/secret lifecycle hygiene, and replay-window vs. retry-duration assumptions.

Crucially, the agent isn't failing at basic cryptography or high-level tenant routing (HMAC signatures, constant-time comparisons, per-build KDF keys, and Studio-side tenant checks are present). Instead, the lapses appear at the **seams** where security policy meets OS primitives, untrusted inputs, retries, and lifecycle state.

---

### Security Lapses & Missing Fundamentals Exposed by the Review

#### 1. Policy vs. Enforcement & Unworkable Permission Modeling (B1)

* **What happened:** `setup.sh` creates `studio-gw` and `studio-agent` and `0600` key files, and documentation promises keys are unreadable to the agent, yet no code drops privileges when spawning the agent or sets a user during provisioning. Furthermore, the file-drop directory/permission setup as written cannot work under either single-user or dual-user uid assignments.
* **Missing fundamental:** **Least privilege and permission-matrix verification.** Security users or permissions only exist if enforced at process spawn/provision time and tested across both allowed and denied identities (`EACCES` on secrets, permitted on intended I/O). Before writing code, map a concrete `actor × resource × operation` permission matrix.

#### 2. Confused-Deputy Symlink & File-Drop Vulnerabilities (F2, F3, M8)

* **What happened:** Reading `question.json` with `os.ReadFile` follows symlinks (exfiltrating sensitive/key material into callbacks), writing `answer.json` with a truncating `WriteFile` without `O_EXCL`/`O_NOFOLLOW` clobbers target files, and the artifact collector checks/stats paths before re-resolving reads (TOCTOU).
* **Missing fundamental:** **Safe handling of untrusted or agent-writable paths.** Any path touched by or adjacent to an untrusted/less-privileged agent must assume symlinks, race conditions, or oversized payloads. Use `O_NOFOLLOW`, `O_EXCL` where appropriate, file-type/ownership checks, size caps, and open-then-`fstat`/read-from-fd patterns rather than check-then-open.

#### 3. Key & Secret Lifecycle Hygiene (B1, F9, M12, M15)

* **What happened:** World-readable `0644` key copies linger in `/srv/studio-staging` after provisioning. Envelopes are signed once at emit with a `SentAt` timestamp, so retries that cross the 5-minute skew window fail with `401` and become terminally failed. The client does not enforce `https`, `flyErr` assumes API errors never leak staged key bytes, and `KDFProvider` round-trips are untested.
* **Missing fundamental:** **End-to-end secret lifecycle and cryptographic freshness.** Generating a strong key or signing a payload once is only half the job—you also must manage staging cleanup, permission modes on *all* copies, re-signing/freshness across retries, transport scheme constraints, error-path redaction, and tests for the actual production key path.

#### 4. Concurrency, State-Machine, and Failure-Path Integrity (F4, F5, F6, F8, M11)

* **What happened:** Deduplication and `Continue` use check-then-act without claiming in-flight work (allowing duplicate execution under concurrency). A mid-run `error` line triggers terminal `failed` even before process exit. Recovery leaves builds stuck with no resume path. Terminal outbox items have no requeue path. Stale turn IDs are not fenced at the gateway layer.
* **Missing fundamental:** **Security-relevant state transitions and fail-safe boundaries.** Race conditions, false terminal states, dead-end recovery, and missing turn/state checks aren't just reliability bugs—they let duplicate or stale commands replay actions or corrupt operational state.

#### 5. Unverified Assumptions & "Happy-Path-Only" Testing (G2, G3, G4, M15, R1–R10)

* **What happened:** Claims of unreadability, CLI behavior (`run.ts` @ v1.19.6), and readiness rested on unverified assumptions, mocked/canned runners, or single-user tests without adversarial abuse tests.
* **Missing fundamental:** **Adversarial & negative test coverage.** Security claims require tests that try to *break* the boundary (symlinks, concurrent duplicates, stale timestamps/turns, unauthorized reads) and explicit tracking of what is proven vs. assumed.

---

### A Repeatable Methodology for Future Implementation Turns

To prevent these gaps from recurring, require the implementation agent to follow this checklist on every security-sensitive task:

1. **Write the Boundary & Threat Model First:** State which component is trusted vs. untrusted (e.g., `studio-gw` vs. `studio-agent`), what assets must stay isolated (keys, tokens, other tenants' files), and how an untrusted actor could abuse inputs or shared paths.
2. **Define the Permission & Ownership Matrix:** Specify exact `uid:gid` and file/directory modes for every artifact, directory, and staged file—including creation, post-setup permissions, and cleanup/deletion.
3. **Apply Safe I/O Primitives by Default:** On untrusted or shared directories, enforce `O_NOFOLLOW`, `O_EXCL` (when creating new files), `fstat`/fd-bound reads to avoid TOCTOU, explicit byte limits, and no secret logging or lingering staging copies.
4. **Check State, Concurrency, and Expiry Together:** Ensure atomic claim/locking for idempotency/dedup, verify how retries interact with timestamps/nonces/skew windows, and check every terminal or recovery state for a valid next transition.
5. **Require Abuse & Production-Path Tests Before Sign-Off:** Pair happy-path unit tests with adversarial/negative tests (symlink reads/writes, concurrent duplicate requests, expired signatures, stale turns) and exercise the real auth/KDF path rather than only test doubles.
6. **Label Assumptions Explicitly:** Separate *code-verified facts*, *live-tested facts*, and *unverified assumptions*—never treat a doc comment or uninspected upstream file as proof.

Would you like to turn this into a concise pre-flight checklist or apply it directly to drafting the fixes for **B1, F2, and F3**?


## question

turn this into a concise pre-flight checklist as general guidelines to achieve the goal of completing all phases securely, reliably and agent-friendly

## answer

Here is a concise pre-flight checklist designed to keep future implementation across **Phases B–E** secure, reliable, and clear for both human reviewers and coding agents.

---

### Miniapp Studio Pre-Flight Checklist

#### 1. Identity & Boundary Enforcement

* [ ] **Enforce OS-level separation:** Ensure distinct identities (such as privileged/gateway vs. agent/runtime users) are actively applied at process spawn and provisioning—never left as unused setup scripts or documentation promises.
* [ ] **Verify permission symmetry across actors:** Confirm every shared or handoff directory, file mode, and group/sticky bit allows intended reads/writes while blocking unauthorized access (`EACCES`) across both privileged and unprivileged identities.
* [ ] **Protect secrets & staging artifacts:** Keep signing keys, credentials, and tokens out of unprivileged paths and logs, restrict key/staging permissions, and clean up temporary or staged key files once installed.

#### 2. Safe File & Input Handling

* [ ] **Prevent symlink & TOCTOU abuse:** Use `O_NOFOLLOW` on untrusted or shared paths, `O_EXCL` when creating new files where clobbering is unsafe, and open-then-`fstat`/read-from-fd patterns rather than check-then-open.
* [ ] **Enforce bounds & contract invariants:** Apply explicit byte/size caps when reading files or drops, fence turns/state transitions, and validate required contracts (such as Go + Miniapp framework + `MiniappSpec` and digest checks) before trusting outputs.

#### 3. Concurrency, Retries & State Integrity

* [ ] **Claim work atomically:** Acquire an in-flight claim, lock, or singleflight guard before executing/launching so concurrent or retried commands cannot double-run side effects.
* [ ] **Keep signatures, retries, and state machines aligned:** Re-sign or refresh time-bound credentials/envelopes across retries if skew windows can be exceeded, base terminal states on actual outcomes rather than mid-run noise, and ensure recovery/failure states have an explicit, honest next step or requeue path.
* [ ] **Avoid collateral restarts & resource leaks:** Do not restart shared/active services or orphan child processes during provisioning, cancellation, shutdown, or recovery.

#### 4. Agent-Friendly Specs & Proof Standards

* [ ] **Give agents concrete, unambiguous rules:** State exact file paths, payload shapes, allowed/forbidden actions, and exit/terminal conditions up front so coding agents do not guess or invent workarounds.
* [ ] **Pair happy-path tests with abuse & live checks:** Include negative/adversarial tests (symlinks, concurrent duplicates, stale turns/expiry) and production-path tests alongside unit tests, and explicitly label what is code-verified, live-tested, or still unverified.

---

### Quick Sign-Off Gate Before Moving Forward

* **Scope check:** Does this change belong to the current phase gate, or is it jumping ahead into a later phase or deferred item?
* **Proof check:** Are the required regression tests, live checks, and docs updated to match actual code behavior?

Would you like to tailor this checklist to a specific upcoming stage—such as **Phase B Part 2** live Sprite verification or **Phase C** two-way Q&A and failure recovery?


## question

if all goes well as planned up to phase E what does this mean for software engineering  reliable deployment-ready robust business apps


## answer

Short answer: reaching Phase E as planned would mean you've demonstrated something the AI-coding discourse keeps promising but almost nobody has actually engineered — **agentic code generation wrapped in the same trust machinery that makes third-party software deployable.** Not "AI writes apps" as a demo trick, but "AI writes apps inside a boundary that lets you ship them and say true things about them."

Here's the chain, phase by phase — each completed phase certifies one link:

| Phase | What completing it certifies | SE principle it embodies |
|---|---|---|
| A — Protocol | Commands and callbacks are authenticated, replay-safe, per-build keyed | Authenticated transport, key derivation, replay protection |
| B — Gateway execution | Builds run on disposable builders under an OS-enforced identity boundary; agent never sees keys | Least privilege, untrusted-code containment |
| C — Dispatch & delivery | Commands sent once, delivered at-least-once, terminal states honest, failures requeueable | Idempotency, durable outbox, honest terminality |
| D — Artifacts & contract | Output is a scrubbed manifest with provenance, on an enforced app contract (Go + Miniapp) | Supply-chain provenance, declarative interface contracts |
| E — Runtime deployment | Built app lands on Fly Machines with owner-held OAuth, builder/runtime separation, consequential writes authorized per action | Scoped credentials, separation of build and run, audit receipts |

**What "reliable, deployment-ready, robust" concretely means at that point** — not adjectives, but checkable properties:

- **Reliable** = every command executes exactly once under concurrency and crash-retry; callbacks survive Studio outages; a recovered builder produces honest state, never a fake resume; a Studio outage of six minutes doesn't permanently destroy queued work. These are the exact properties the adversarial review proved or refuted with probes.
- **Deployment-ready** = the artifact is a manifest, not a vibe. Reproducible, idempotently provisioned via API, static binary, health-verified, with credentials the build environment never touched — the OAuth ceremony is owner-held and the builder never sees a production token.
- **Robust business apps** = the generated app inherits the guarantees of its pipeline: signed provenance, scoped connector access, human approval on consequential actions, and a receipt trail for everything it did. The last mile where generated business software usually dies — auth, connectors, deployment, auditability — is industrialized instead of hand-waved.

**What this means for software engineering, more broadly — three transferable lessons:**

1. **Treat LLM-written code as untrusted third-party code.** This is the core move. The industry's instinct with agents is to trust the output and review it by eyeballing diffs. Your architecture instead applies the classic open-source supply-chain playbook — identity boundaries, signed transport, least privilege, provenance manifests, scrubbed artifacts — to a builder that happens to be an LLM. If Phase E holds, the lesson generalizes far beyond Miniapp: *agent output becomes shippable precisely when you stop trusting it.*

2. **Reliability is architecture, not aspiration.** Dedup stores, singleflight claims, re-signing at delivery, fail-closed refusals, never-restarting supervisors — none of that is glamorous, and the review found real holes in it anyway (B1, F2–F9). That's the point: the system *refuses to call itself ready until someone actively fails to break it.* The methodology — evidence matrices, claims audits, adversarial probes, "claim only what tests prove" — is itself a reusable engineering practice, arguably the most valuable artifact of the whole project.

3. **Autonomy at the safe seams, humans at the irreversible ones.** The agent works autonomously through the entire build; a human gate appears exactly where actions become consequential (approval, OAuth, deploy). That asymmetric design is the operational pattern agentic software engineering has been missing — not "human reviews everything" (unscaleable) nor "human reviews nothing" (unaccountable).

**The honest scope — what it does *not* mean** (your own claims-audit rule applies to this pitch too): it means one framework (Miniapp/Go), one cloud (Fly), one operator, demo-scale tenancy — multi-tenant operations are Phase C+ work and R1–R10 remain live-unverified until Part 2 runs. The pipeline certifies *provenance and governance*, not the semantic fitness of every generated feature — the app's own tests and the approval loop still carry that weight. And B1 is the standing reminder: "all goes well" is earned per-phase by surviving review, which is exactly why the end state, if reached, is worth believing.

So the one-sentence version: **Phase E reached = proof that the missing layer between "an LLM can write your app" and "you can deploy that app on Monday" is not more model capability — it's governance infrastructure, and you built it.**


## question

just to clarify about the workflow, so Miniapp Studio runs as Web server app along with coding agent specified in the UI, is that right? if so, users should be able to set in the UI configuration settings which coding agent is the default along with other sane default choices like LLM model etc


## answer

Yes. That's the intended product workflow, with one important architectural distinction: **Miniapp Studio is the web application and orchestration control plane; the coding agent is a separate execution process managed by Studio.**

The agent should not run inside the Studio web server process.

## **1. How the workflow should work**

**User's Browser**

Describe application · Configure settings · Review · Approve

**Miniapp Studio Web Server**

Conversations · Settings · Builds · Agent selection · Approvals · Evidence

Control plane

Durable commands through Gateway

**Dedicated Builder Environment**

Workspace · Gateway · Process supervision · Credentials isolation

**Selected Coding Agent**

OpenCode

Codex

Claude Code

Antigravity

Meta Muse Code

Works on the Miniapp workspace using the configured model and permitted tools.

**Studio receives events and artifacts**

Pulse · Line · Validation · Testing · Human approval

In local development, the Studio server and agent process could reside on the same machine. In the intended production architecture, the agent runs in its isolated Builder environment. Their separation is logical and architectural, not dependent on physical machines.

## **2. Yes — agent and model selection belong in Settings**

I recommend making this a first-class feature of Miniapp Studio.

The Settings scene should let administrators configure defaults rather than forcing every user to repeatedly select an agent.

**Illustrative Settings experience**

## **AI Execution Defaults**

Workspace defaults for new application builds

**Default coding agent**

OpenCode

OpenAI Codex

Claude Code

Google Antigravity

Meta Muse Code

**Default LLM model**

Model selected according to agent

Available choices depend on the selected agent, configured model provider, and verified compatibility.

**Default reasoning effort**

LowMediumHigh

**Default execution permission**

Restricted — ask when additional access is needed

Illustrative controls, not existing functional settings. Platform security restrictions always take precedence.

I would organize the real configuration into these categories:

| Configuration              | Purpose                                               |
| -------------------------- | ----------------------------------------------------- |
| Default coding agent       | Which agent runtime executes new builds               |
| Default LLM provider/model | Which inference model the agent uses, where supported |
| Reasoning effort           | Model-specific reasoning configuration                |
| Permission profile         | Tool and workspace permissions                        |
| Agent version              | Reproducible runtime version                          |
| Execution limits           | Time, resource and optional spending limits           |
| Provider connections       | Configure credentials securely and check availability |

## **3. Three levels of configuration**

This is important for a multi-tenant SaaS product.

**Platform level:** You, as platform administrator, decide which agent providers and model providers are available.

**Workspace/tenant level:** The business owner chooses the default agent, model and execution preferences for their organization.

**Application/build level:** An authorized user may override defaults for a particular application or build, subject to workspace policy.

For example:

A business might set Claude Code with a particular model as its workspace default, but use Codex for one application and Muse Code for another.

Studio should never silently switch providers because one becomes unavailable. It should explain the problem and ask for an authorized alternative.

## **4. One important architectural distinction**

We need to model two different things independently:

- **Agent runtime:** OpenCode, Codex, Claude Code, Antigravity or Muse Code. This controls coding behavior, tools, sessions, file operations and agent orchestration.
- **Model provider:** OpenAI, Anthropic, Google, Meta or another supported inference service. This supplies the underlying model.

Not every agent supports every model provider or configuration option. Studio must discover and validate compatible combinations rather than promise universal interchangeability.

For every build, Studio should persist an immutable execution-configuration snapshot:

```
build_id
agent_provider
agent_version
model_provider
model_id
reasoning_configuration
permission_profile
configuration_version
```

This makes historical builds reproducible and lets the operator see exactly which agent and model produced an artifact.

Credentials must remain encrypted and server-side, never inside browser settings, conversation transcripts or ordinary build payloads.

## **5. What this means for Phase D**

I recommend one small adjustment to our current roadmap:

- **D0:** Define how agent/model configuration is represented and recorded in build provenance.
- **D1–D4:** Continue artifact retrieval, verification, testing and approval.
- **D5:** Integrate the relevant Settings and Console experiences only to the extent necessary for the approved Phase D scope.
- **Separate provider-expansion milestone:** Implement the full provider registry, availability checks, authentication management, compatibility discovery and selectable agent runtimes.

Do not make Phase D dependent on implementing all five coding agents.

**My architectural recommendation:** Miniapp Studio should be an agent-agnostic application development platform. The business owner describes what they want to build; Studio selects the configured agent, coordinates its work, verifies the resulting Miniapp artifact, and obtains human approval. The choice of coding agent or model should be an implementation detail the business owner can control—not a permanent architectural dependency on OpenCode.

This is consistent with the existing dedicated Builder architecture and the initial OpenCode-only adapter boundary.&#x20;

LOCKED — PRODUCT ROADMAP

Agreed. **Agent-agnostic execution and configurable AI providers are now part of the long-term Miniapp Studio product roadmap, not part of the hackathon deliverables.**

I am treating the following as binding product architecture and roadmap direction.

## **Product roadmap decision: Agent-agnostic Miniapp Studio**

**Product vision:** Miniapp Studio is a web-based application development and orchestration platform that allows business owners to choose which AI coding agent and model power their application development, while Studio retains control over execution, verification, security and approval.

### **1. Architectural principles**

1. Miniapp Studio is the web application and orchestration control plane.
2. Coding agents execute as separate, supervised processes in dedicated Builder environments.
3. Agent runtimes and model providers are independent architectural concepts.
4. Business owners can configure default agents, models and execution preferences through Studio Settings.
5. Authorized users can override workspace defaults for individual applications or builds, subject to policy.
6. Studio discovers and validates provider capabilities rather than assuming universal compatibility.
7. Every build records an immutable snapshot of its execution configuration.
8. Provider credentials remain securely managed by Studio and isolated by tenant.
9. Provider failure never causes an undisclosed, automatic switch to another agent.
10. All providers remain subject to the exclusive Go/Miniapp application architecture.

### **2. Planned provider ecosystem**

| Layer           | Planned support                                                            |
| --------------- | -------------------------------------------------------------------------- |
| Agent runtimes  | OpenCode, Codex, Claude Code, Antigravity, Muse Code                       |
| Model providers | OpenAI, Anthropic, Google, Meta and compatible future providers            |
| Configuration   | Agent, model, reasoning effort, permissions, versions and execution limits |
| Management      | Provider availability, authentication, capabilities and compatibility      |
| Execution       | Dedicated Builder environments and supervised processes                    |

Meta Muse Code is an agent runtime; Meta Model API is a model-provider integration. They must not be conflated.

### **3. Roadmap placement**

| Milestone                | Scope                                                                   |
| ------------------------ | ----------------------------------------------------------------------- |
| Current Phase D          | Artifact pipeline, verification, isolated testing and human approval    |
| Phase D0                 | Define execution-configuration provenance and immutable build snapshots |
| Future product milestone | Agent/provider registry and capability discovery                        |
| Future product milestone | Workspace Settings for default agent, model and execution preferences   |
| Future product milestone | Multiple agent adapters and model-provider integrations                 |
| Future product milestone | Per-application/build configuration overrides                           |

The full provider ecosystem is a separate product investment. It does not delay Phase D.

### **4. Explicit hackathon boundary**

This decision does **not** expand the hackathon's implementation scope.

The existing OpenCode-first path remains the initial implementation. No additional coding agents, provider-management UI, or model integrations are required for the hackathon.

**Final architectural principle:**

Miniapp Studio must not be built around one particular AI coding agent or model provider. Business owners choose the intelligence powering their development workflow; Miniapp Studio governs how that intelligence produces, verifies and delivers applications.

This is now a product roadmap commitment, independent of the hackathon schedule.


## question

and where does the Studio-generated miniapp runs, on separate Fly.io Machine or Sprite?

## answer

The current Miniapp Studio architecture calls for a **separate Runtime Sprite for each approved application**, distinct from the Builder Sprite where the coding agent works.

The generated Miniapp does **not** run inside the Studio web server or inside the Builder environment.

This is the previously established Phase E deployment model:

- **Miniapp Studio:** web application, control plane, persistence and orchestration.
- **Builder Sprite:** isolated workspace where the selected coding agent produces the application.
- **Runtime Sprite:** separate execution environment where the approved Miniapp runs using the Miniapp runtime.

The artifact transferred to the Runtime environment must be the exact approved digest. It must not be rebuilt from mutable Builder files.

I need to refine that answer after checking Fly.io's current architecture guidance: **for a production Miniapp, I recommend a separate Fly.io Machine, not a Runtime Sprite.**

Fly.io explicitly distinguishes the two: Sprites are designed for agent work and isolated code execution; Fly Machines are designed to host production applications.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

fly.io+1



## **Recommended Miniapp Studio architecture**

**Miniapp Studio**

Central web application and control plane

Development

**Builder Sprite**

Coding agent, source workspace, development services

Verification

**Isolated test environment**

Artifact build and tests, no production credentials

Human approval of exact artifact digest

Production

**Dedicated Fly Machine deployment**

Approved Go application running on the Miniapp runtime. No coding agent or Builder Gateway.

Optional: Preview applications can run in separate Sprites before production approval.

| Environment | Compute                    | Responsibility                  |
| ----------- | -------------------------- | ------------------------------- |
| Studio      | Web server                 | Orchestration and management    |
| Builder     | Sprite                     | Coding agent and development    |
| Testing     | Isolated container/sandbox | Artifact verification and tests |
| Preview     | Sprite (optional)          | Temporary application preview   |
| Production  | Fly Machine(s)             | Approved Miniapp application    |

## **Why I prefer Fly Machines for production**

Production applications generally need predictable service availability, routing, health monitoring and scaling. Fly positions Machines for that workload, while Sprites provide persistent, isolated environments that sleep when idle.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

fly.io+1



There is another important security boundary:

**The production Machine must receive only the approved artifact.** It must not receive the Builder's filesystem, agent credentials, Gateway process or development session.

The deployment process should:

1. Retrieve the exact approved artifact digest.
2. Build the production image using the pinned Go toolchain and Miniapp runtime.
3. Record the image digest and artifact digest.
4. Deploy to a separate Fly Machine deployment.
5. Verify health and expose the application URL.
6. Support rollback to a previously approved deployment.

## **One product architecture decision to update**

I recommend replacing the earlier Phase E term *Runtime Sprite* with **Runtime Deployment**, initially implemented using Fly Machines.

Keep the compute provider behind a deployment-provider abstraction so Miniapp Studio is not permanently tied to Fly.io.

This preserves the long-term product vision: business owners choose their coding agent and model, while Miniapp Studio controls where and how their approved applications run.

**My recommendation: Builder Sprite for development; Fly Machine for production; optional Preview Sprite for previews.**


LOCKED — PRODUCT ARCHITECTURE AND HACKATHON STRATEGY

Agreed. We will preserve the long-term product architecture while deliberately reducing implementation scope for the hackathon. **We are building the first credible vertical slice of Miniapp Studio, not trying to finish the entire platform before the deadline.**

## **1. Locked product architecture**

These are now binding roadmap decisions:

| Component               | Long-term product direction                         |
| ----------------------- | --------------------------------------------------- |
| Miniapp Studio          | Web application and orchestration control plane     |
| Builder environment     | Dedicated Sprite per application workspace          |
| Coding agent            | Independently selectable agent runtime              |
| Model                   | Independently configurable model provider and model |
| Verification            | Isolated artifact verification and testing          |
| Approval                | Human-only, digest-bound authorization              |
| Preview                 | Optional temporary Preview Sprite                   |
| Production              | Dedicated Fly.io Machine deployment                 |
| Deployment architecture | Provider abstraction, initially Fly Machines        |

The production Miniapp never runs inside Studio or the Builder Sprite. It receives only the exact approved artifact. It has no coding agent, Builder Gateway, or development credentials.

## **2. Separate product roadmap from hackathon MVP**

The hackathon should deliver a narrow but convincing end-to-end experience.

## **Hackathon MVP — Invoice Review Desk**

1. **Describe**

   Business owner describes the Invoice Review Desk in plain English.
2. **Build**

   Studio uses the initial OpenCode integration to generate a Go/Miniapp application.
3. **Verify**

   Studio checks the MiniappSpec, source files, dependencies and artifact digest.
4. **Test**

   Studio tests the artifact in an isolated environment, or honestly reports that testing is blocked.
5. **Approve**

   The business owner explicitly approves the exact artifact.
6. **Deploy**

   When justified by the judging criteria, deploy the approved artifact to one dedicated Fly Machine.
7. **Demonstrate**

   Open the running Invoice Review Desk and demonstrate its synthetic invoice workflow.

## **3. What we deliberately defer**

To protect the deadline, these are not hackathon requirements:

- Five coding-agent integrations. OpenCode remains the initial adapter.
- Full model-provider marketplace and compatibility discovery.
- Advanced workspace/provider configuration UI.
- Multiple production deployment providers.
- Automatic scaling, multi-region deployment and fleet management.
- Multiple simultaneous production applications.
- Generic application templates and workflow builders.
- Production CRM, accounting and payroll integrations.

The product architecture must not rule these out, but the MVP does not need to implement them.

## **4. Recommended implementation sequence**

| Order | Milestone                                                          | Hackathon priority             |
| ----- | ------------------------------------------------------------------ | ------------------------------ |
| 1     | D0: freeze artifact contract and boundaries                        | Critical                       |
| 2     | D1–D2: artifact retrieval and verification                         | Critical                       |
| 3     | D3: isolated testing or honest blocked capability                  | Critical                       |
| 4     | D4: human-only artifact approval                                   | Critical                       |
| 5     | D5–D6: Console integration and local rehearsal                     | Critical                       |
| 6     | Phase B Part 2/C5: one real Builder Sprite and live agent proof    | After local acceptance         |
| 7     | Minimal Phase E: one approved artifact deployed to one Fly Machine | Conditional on judging benefit |

The seventh step is intentionally narrow. No generic deployment platform is necessary to demonstrate the production architecture.

## **5. Hackathon deployment acceptance**

A minimum credible production deployment should demonstrate:

1. An approved immutable artifact is selected.
2. Its digest is verified before deployment.
3. The application is built with the pinned Go/Miniapp toolchain.
4. One dedicated Fly Machine runs the application.
5. Health checks succeed and Studio records the deployment identity, artifact digest and URL.
6. The application displays synthetic invoice data.
7. The running application is independent of its Builder environment.

No mutable Builder source should be copied directly into production.

## **6. The governing principle**

I want us to optimize for one thing:

**Demonstrate that a business owner can describe an application, have an AI coding agent build it, have Miniapp Studio verify it, approve it and ultimately run it without needing to understand the underlying engineering.**

That is a much stronger hackathon story than demonstrating five agent integrations or a sophisticated deployment control plane without a complete working application.

**Final decision:** Product architecture is locked around Builder Sprites and production Fly Machines. Hackathon implementation remains OpenCode-first, Invoice Review Desk-only, local-first through Phase D, followed by the shortest necessary live Builder and Fly deployment path.

We preserve the long-term vision without allowing it to become the hackathon's scope.


## question

to clarify more, once the Miniapp Studio is done per prompt 22, I should be able to run the Studio locally on my desktop and it will talk to my local OpenCode, right?

## answer

**Yes. That is the intended local-development experience, but there is one missing acceptance requirement in Prompt 22.**

Once the Phase D work is complete, you should be able to run Miniapp Studio on your desktop, open it in your browser, and have it communicate with your locally installed OpenCode.

The architecture should look like this:

**Your desktop**

Local development environment

**Browser**

[http://localhost](http://localhost?utm_source=chatgpt.com) — Focus Console

**Miniapp Studio**

Go web server · PostgreSQL · conversations · builds · approvals

Durable Gateway commands and signed callbacks

**Local Gateway + OpenCode adapter**

Separate supervised process and isolated workspace

**Your locally installed OpenCode**

Writes Go/Miniapp application code in its assigned workspace

The agent's progress, questions, artifacts and results flow back into Studio's Pulse and Line.

## **One important distinction**

Prompt 22 currently specifies the artifact pipeline. It does not complete the live local OpenCode integration. Phase B Part 2 and C5 remain separately gated, and Prompt 22 explicitly excludes live agent sessions.&#x20;

phase-d-implementation-plan.md

phase-d-implementation-plan.md



I recommend adding a separate **Local Development Mode milestone** to the hackathon MVP acceptance criteria:

1. Start Miniapp Studio locally.
2. Connect to the local Gateway and OpenCode adapter.
3. Select OpenCode for a build.
4. Have OpenCode generate an Invoice Review Desk.
5. Demonstrate streaming, clarification, answering, cancellation and recovery.
6. Retrieve the resulting artifact, verify it, test it and approve it through Studio.
7. No Sprite or Fly dependency for this local demonstration.

This is a development mode, not a replacement for the production architecture. Eventually, Builder Sprites handle remote development, while production applications run on dedicated Fly Machines.

**I recommend locking this expectation:** completing Phase D alone is insufficient for the local MVP. The MVP must also demonstrate a complete Studio-to-local-OpenCode execution round trip.


## prompt

educate me about preview, staging and prod environments on both Fly and Neon, when do we encounter these use cases, I want to incorporate it in our development, testing and deployment process, just high level overview


## answer

Think of **Preview, Staging, and Production as three different levels of confidence before and after releasing software.** They are not three different versions of your codebase. They are separate places where you run and test your application, with different purposes and different levels of risk.

Fly.io provides the application runtime: where your Go application actually runs. Neon provides the PostgreSQL databases: where your application's data lives.

The two work together, but their environments serve different roles.

## **1. The three environments, in plain English**

**Preview**

Experiment safely

# **Does this change work?**

Preview is where you test a particular change before it becomes part of the shared application. It might be created for a feature branch, pull request, or candidate build.

Example: You're changing Miniapp Studio's workspace handoff. You deploy that version to Preview and test whether local-to-remote handoff works.

**Staging**

# **Is this release ready?**

Staging is a persistent environment that resembles Production. It is where you test the release as a whole, including integrations, configuration, migrations, and operational behavior.

Example: The workspace handoff feature passed Preview. You now test it alongside the rest of Miniapp Studio, with the intended release configuration and a staging database.

**Production**

# **Can real users rely on it?**

Production is the actual application used by customers. It contains real operational state and potentially important customer data. Changes require the strongest safeguards.

Example: You have accepted the release in Staging. You authorize deployment of that exact release to the live Miniapp Studio.

A useful analogy:

- Preview is your workshop prototype.
- Staging is the dress rehearsal.
- Production is opening night.

## **2. How Fly.io and Neon work together**

The key idea is that **each application environment should connect to its corresponding database environment.**

## **Your application lifecycle**

**PREVIEW**

Temporary

**Fly Machine — Preview app**

**Neon branch — Preview database**

Isolated test data, usually one branch per feature or pull request.

Accepted candidate

**STAGING**

**Fly Machine — Staging app**

**Neon branch — Staging database**

Persistent environment for release-candidate testing and integration validation.

Human-approved release

**PRODUCTION**

**Fly Machines — Production app**

**Neon branch — Production database**

Real users, protected data, controlled migrations, explicit release authorization.

Conceptual architecture: the Fly runtime and Neon database are separate resources connected through environment-specific configuration.

Fly Machines run your application. Neon branches provide isolated PostgreSQL database environments. A Preview Fly Machine must not accidentally connect to the Production Neon branch.

This is particularly important when coding agents are creating migrations or testing changes. A faulty migration should damage only its isolated test database, not your live application data.

Fly describes per-PR preview environments as separate applications and databases, while Neon supports branching for previews, individual developers, and staging.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.fly.io\&sz=32)

Learn+2



## **3. What happens to the databases?**

This is where Neon becomes especially useful.

Think of a Neon branch as a separate database development line. You can create a branch from another branch, experiment with schema changes and data, and keep the parent environment separate.

For example:

[Practical Guide to Database Branching  - Neon](https://images.openai.com/static-rsc-4/9K41tb0VD7NsrP9DiPkPUvYw8-ULc02eMBAm9cTFuJGlcqkxYRchuAPt81I55tmLoQXf03luJG6IWRB2b95iWKd9OY2dboZLrSBxD9gujJMEMfs3q_mEnq0ib1QR2cVOpcRghIVDXVGx8CYto37eKoQgbrr_xP4u8t7FSnTJAso?purpose=inline)

[neon.com](https://neon.com/blog/practical-guide-to-database-branching?utm_source=chatgpt.com)

A simplified branch arrangement could look like this:

```
Production branch
       |
       +---- Staging branch
       |
       +---- Preview PR-101
       |
       +---- Preview PR-102
       |
       +---- Developer branch
```

There is an important distinction here: **branching a database is not the same as promoting application code.**

Suppose you change a database schema by adding a `workspace_checkpoints` table.

Your process could be:

1. Create a Preview database branch.
2. Apply the migration to that branch.
3. Test the Go application against it.
4. Confirm that existing functionality still works.
5. Apply the migration separately to Staging.
6. Verify the application against Staging.
7. Apply the approved migration to Production through a controlled release procedure.

You promote the *schema change through migrations*, not blindly copy Preview's database contents into Production.

Neon documents workflows for separate branches per developer, PR, preview, and test run.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://neon.com\&sz=32)

Neon+2



## **4. When do you encounter each environment in actual development?**

Let's follow a realistic feature-development example for Miniapp Studio.

Suppose you want to implement **workspace checkpoints and remote handoff**.

| Development activity                            | Environment | Fly.io                      | Neon                        |
| ----------------------------------------------- | ----------- | --------------------------- | --------------------------- |
| Write code and run unit tests                   | Local       | Not required                | Local PostgreSQL            |
| Test against a shared cloud database, if needed | Development | Optional                    | Developer/task branch       |
| Validate a feature branch                       | Preview     | Temporary Fly app/Machine   | Temporary database branch   |
| Test an integrated release candidate            | Staging     | Persistent Fly app/Machines | Persistent staging branch   |
| Release to customers                            | Production  | Production app/Machines     | Protected production branch |

Here's when each becomes useful.

### **Preview: during feature development**

You typically encounter Preview when:

- A coding agent finishes a substantial feature.
- You open or update a pull request.
- You introduce a database migration.
- You need to test the whole application, not just individual functions.
- Someone needs to review the actual running application.

Imagine Muse changes the workspace handoff implementation.

Unit tests pass. But does the application actually start? Does the migration work? Can you transfer a workspace? Does the remote agent see the correct files?

You deploy the candidate to Preview and test those questions.

A particularly useful arrangement is one Preview per pull request:

```
PR #101
  Fly: miniapp-pr-101
  Neon: miniapp-pr-101

PR #102
  Fly: miniapp-pr-102
  Neon: miniapp-pr-102
```

The two candidates don't interfere with each other's infrastructure or database changes. When a pull request closes, its Preview resources can be cleaned up automatically.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.fly.io\&sz=32)

Learn+1



### **Staging: before releasing a version**

You encounter Staging after a feature or collection of changes has passed its individual Preview checks.

Staging answers a different question:

Does the entire release work together under conditions resembling Production?

For example, Miniapp Studio might have several changes:

- Workspace handoff
- Gateway recovery
- Artifact verification
- Environment configuration
- Logging

Each feature might work independently. Staging tests them together.

You also test things that are easy to miss locally:

- Database migrations.
- Authentication and authorization.
- Configuration and secrets.
- External integrations.
- Application startup and shutdown.
- Recovery after process restarts.
- Performance and resource constraints.
- Deployment and rollback procedures.

Staging should normally remain available. Unlike Preview, it shouldn't disappear whenever a pull request closes.

### **Production: after approval**

Production is where the actual users and their persistent data live.

Before releasing, you want evidence that:

- The candidate passed automated tests.
- Preview checks passed.
- Staging acceptance passed.
- Required database migrations are ready.
- The exact artifact has been approved.
- Rollback or recovery procedures are understood.

You then deploy the same immutable application artifact that passed Staging, rather than rebuilding potentially different code for Production.

The Production database is not replaced with the Staging database. You apply the approved schema migrations and preserve Production's actual data.

## **5. An important distinction: database branching versus application deployment**

This is worth understanding carefully.

Suppose you have this migration:

```
CREATE TABLE workspace_checkpoints (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL,
    checkpoint JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
```

You should not immediately apply that migration to Production just because your coding agent wrote it.

Instead:

1. **Preview database**

   Apply the migration to the Preview Neon branch. Test the application and migration together.
2. **Staging database**

   Apply the migration separately to the Staging branch. Test the complete release and ensure it is compatible with the application.
3. **Production database**

   Apply the approved migration under the Production release procedure.

The database branches provide isolated places to experiment. They do not automatically synchronize changes back into their parent or promote your schema changes to Production.

This is an important distinction: application promotion and database migration are coordinated operations, but they are not the same operation.

Neon's branching workflows explicitly support this approach to testing database changes and migrations before release.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://neon.com\&sz=32)

Neon+1



One additional practice worth learning is *expand and contract* database migrations. Instead of immediately removing or renaming a column that existing application versions depend on, introduce compatible changes first, migrate the application, and remove the old structure only after it is no longer needed.

This becomes particularly important when deploying multiple Machines or rolling back application versions.

## **6. How I recommend incorporating this into Miniapp Studio**

I would use the following development lifecycle:

## **Miniapp Studio development lifecycle**

1. **Local development**

   Muse works on the Studio repository. Local PostgreSQL supports rapid development and unit tests. Changes remain in Git.
2. **Automated verification**

   Run formatting, static analysis, unit tests, integration tests, and build checks.
3. **Preview deployment**

   Create an isolated Neon branch, provision a temporary Fly environment, apply migrations, deploy the candidate, and run end-to-end tests.
4. **Human review**

   Inspect the running application, logs, test evidence, and Git changes. Accept or return the feature for further work.
5. **Staging deployment**

   Deploy the accepted immutable artifact. Apply staging migrations and perform integrated acceptance and operational testing.
6. **Production authorization**

   A human approves the exact artifact digest and associated migration plan.
7. **Production deployment**

   Deploy the approved artifact, apply the controlled Production migration, verify application health, and preserve release/rollback evidence.

## **7. A few practical lessons for your Fly.io and Neon setup**

**First, Preview should be disposable.** Do not let temporary environments accumulate indefinitely. Give them ownership, expiration, and cleanup policies.

**Second, Staging should be stable.** It is a shared testing environment, not somewhere every experimental agent should freely modify the database.

**Third, Production must be isolated.** Development agents should not inherit unrestricted Production credentials simply because they can access Fly and Neon CLIs.

**Fourth, database data needs protection too.** A Preview environment should generally use synthetic or sanitized data. A branched database that exposes sensitive Production records is not automatically safe because it is technically isolated.

**Fifth, infrastructure costs need lifecycle management.** Fly bills for provisioned resources and running Machines; stopped Machines still have storage costs. Neon branches and compute also have usage and retention implications. Automating cleanup is therefore part of the design, not an afterthought.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.fly.io\&sz=32)

Fly Docs+1



## **8. What I would implement first for Miniapp Studio**

Don't immediately build three elaborate environments.

Start with these three concrete capabilities:

| Priority | Capability                               | What it proves                                      |
| -------- | ---------------------------------------- | --------------------------------------------------- |
| 1        | One Preview Fly app + Neon branch        | We can deploy and test a real candidate safely.     |
| 2        | Persistent Staging Fly app + Neon branch | We can validate a release independently of Preview. |
| 3        | Protected Production app + Neon branch   | We can safely release an approved artifact.         |

For your initial dogfooding exercise, I would implement Preview first, because it provides the fastest feedback on the features being built in Miniapp Studio itself.

Then add Staging as soon as the Preview workflow is repeatable. Production comes after the promotion, authorization, and rollback processes are demonstrated.

**One final distinction for the Studio UI:** Local Desktop versus Remote Builder Sprite describes *where the coding agent works*. Preview, Staging, and Production describe *where the application runs*. These are independent selectors. Keeping them separate will prevent a lot of confusion as the platform grows.

## **Further reading**

These are useful primary references:

- [Fly.io — Per-PR Preview Environments for Coding Agents](https://fly.io/learn/per-pr-preview-environments-for-coding-agents/?utm_source=chatgpt.com) — particularly relevant to your AI coding-agent workflow.
- [Neon — Database Branching Workflows](https://neon.com/branching?utm_source=chatgpt.com) — covers development, Preview, Staging, and database branching.
- [Neon — Branching in CI/CD](https://neon.com/blog/adopting-neon-branching-in-ci-cd-pipelines-a-practical-story-by-shepherd?utm_source=chatgpt.com) — shows how migrations can move through development, Preview, Staging, and Production.

The mental model to remember is simple: **Preview validates a change, Staging validates a release, and Production serves users.** Fly runs the application at each stage; Neon gives each stage the appropriate database.


## prompt

remember that the hackathon is marketing Fly Sprites, how do Sprites figure in our process? it must be integrated as well where applicable or where absolutely essential not just for the sake of the hackathon but to leverage it where it is really important, brief reply only


## answer

Absolutely. **Sprites should be a core part of Miniapp Studio, not merely a hackathon demonstration.**

The important distinction:

- **Sprite = where coding agents work and execute untrusted code.**
- **Fly Machine = where the application or long-running production service runs.**

I recommend integrating Sprites at three points:

1. **Remote development:** Dedicated persistent Sprite per application/workspace. Agents can work remotely, preserve dependencies and files, and resume work.
2. **Isolated execution and testing:** Use separate Sprites to run generated code, builds, integration tests, and potentially risky operations. Checkpoint and restore before risky changes.
3. **Preview and demonstration:** Give users a live preview of development work through a Sprite URL when appropriate. Final accepted application releases run on Fly Machines.

This gives us a compelling architecture:

**Miniapp Studio**

Development control plane

**Coding Agent**

Muse, OpenCode, Codex, etc.

Fly Sprites

**Development and execution environments**

Persistent workspace · Checkpoints · Isolated testing

**Fly Machines**

Preview · Staging · Production application runtimes

Neon provides isolated databases for the respective application environments.

Fly specifically promotes this separation between agents, isolated execution Sprites, and application-serving Machines.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+1



**My recommendation:** Make Sprite integration part of the first meaningful development workflow, not a later optional feature. Local development remains available, but remote development and isolated agent execution should demonstrate why Sprites are valuable in real engineering work.

That is a much stronger hackathon story: we're not just deploying on Fly; we're demonstrating how Fly enables the entire AI-native software development lifecycle.


## architect roles

you are the senior architect of this project and entire process based on discussions above, I want you to drive the entire direction and process as I ask you every now and then so ensure that you give instructions when to use Fly Sprites or Machines, or when to use preview, staging and prod environments, got it?


Got it. I will act as the **Senior Architect and technical director of Miniapp Studio**, driving the architectural direction, implementation sequence, infrastructure choices, and acceptance criteria as we work through the project.

I will take responsibility for recommending not just *what to build*, but also **where, when, and how we should build and test it.**

These are the operating rules I will follow:

| Infrastructure         | When I will direct you to use it                                                                                       |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| Local Desktop          | Fast development, local coding agents, unit tests, and inexpensive experimentation.                                    |
| Fly Sprites            | Remote Builder workspaces, persistent agent execution, isolated builds, integration testing, and risky code execution. |
| Fly Machines           | Running actual application deployments: Preview, Staging, and Production.                                              |
| Neon Preview branches  | Isolated database testing for candidates and pull requests.                                                            |
| Neon Staging branch    | Persistent release-candidate and integration testing.                                                                  |
| Neon Production branch | Protected customer-facing database and controlled migrations.                                                          |

### **How I will direct each implementation task**

For every significant task, I will explicitly establish:

1. **Objective:** What problem are we solving?
2. **Execution environment:** Local Desktop or Fly Sprite?
3. **Application environment:** Local, Preview, Staging, or Production?
4. **Database:** Which Neon branch or local PostgreSQL instance?
5. **Infrastructure operations:** Which Fly and Neon CLI operations are appropriate?
6. **Authorization:** What can the coding agent do independently, what needs your approval, and what is prohibited?
7. **Acceptance criteria:** What evidence proves the work is complete?

I will also distinguish planning authorization from implementation authorization and explicitly identify when cloud operations are involved.

### **My governing architectural principle**

**Miniapp Studio must progressively become the system that manages its own development, testing, deployment, and operations.**

We will dogfood it continuously, use Sprites where remote execution and isolation provide genuine value, use Machines for application runtime, and use Preview → Staging → Production as progressively stronger release gates.

I will not recommend infrastructure complexity simply to demonstrate a technology. Every component must serve the product.

**You focus on the business vision and priorities. I will drive the technical direction, sequencing, and execution discipline.**