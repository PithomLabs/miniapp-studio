## prompt

read roadmap4.md and README.md for context, educate me about the five phases, why are those important and elaborate its context in the overall scheme of things


## gemini

The attached roadmap outlines how **Miniapp Studio**—a multi-tenant platform for turning plain-language business processes into governed software—is engineered and rolled out in five staged phases (**Phases A through E**).

Rather than building everything at once or trusting an AI coding agent on its word, the five phases separate **durable conversation plumbing**, **isolated execution infrastructure**, **two-way clarification**, **strict artifact verification**, and **controlled production deployment**.

---

### The Five Phases at a Glance

| Phase | Core Focus | Key Exit / Gate |
| --- | --- | --- |
| **Phase A: Conversation Infrastructure** | Durable business conversation state, tenant-isolated storage (RLS), secret sanitization, SSE replay, and crash/restart recovery.

 | **Accepted** (as of Nov 25, 2026); 10 tables migrated with RLS enabled/forced and recovery tested.

 |
| **Phase B: Sprite Agent Gateway** | Command/callback transport between Studio and a per-workspace **Builder Sprite** (and coding agent, initially OpenCode), OS/key isolation, and service lifecycle.

 | **Part 1 (fake-backed)** authorized; **Part 2 (live Sprite & recovery)** requires a separate explicit GO.

 |
| **Phase C: Genuine Two-Way Interaction** | End-to-end live loop where the coding agent asks real clarification questions, Studio persists/renders them, and the user's answer resumes the same session—plus failure/recovery drills.

 | Proven live Q&A session continuity and recovery matrix (kill, sleep/wake, restart, out-of-order/stale events).

 |
| **Phase D: Miniapp Build, Artifact Verification & Review** | Enforcing the binding **Go + Miniapp framework + `MiniappSpec**` rule, verifying SHA-256 digests and test reports, and presenting review evidence before any deploy.

 | A real Go/Miniapp artifact built, validated, and explicitly approved (or sent for rework) with non-Go/bypass artifacts rejected.

 |
| **Phase E: Approved Deployment to Runtime Sprite** | Promoting *only* approved, digest-verified artifacts into a separate **Runtime Sprite** with scoped production connectors, health checks, and rollback.

 | Live health check pass, clean Runtime without dev/agent tools, and tested rollback to a prior approved digest.

 |

---

### Detailed Breakdown of Each Phase

#### 1. Phase A — Conversation Infrastructure

* **What it establishes:** The authoritative business conversation between the user and Studio (`user ↔ Studio`). It implements tenant-scoped database tables with PostgreSQL Row-Level Security (**ENABLED and FORCED**), write-path secret sanitization, Server-Sent Events (SSE) with cursor replay, and ordered shutdown/recovery.


* **Why it matters:** A business owner's requirements, questions, answers, and audit trail cannot live inside a coding agent's ephemeral terminal or local process. If a browser disconnects or a server restarts, the conversation and state must survive intact.



#### 2. Phase B — Sprite Agent Gateway

* **What it establishes:** The bridge (`Studio ↔ Gateway ↔ coding agent`) into a dedicated, long-lived **Builder Sprite** per application workspace. It covers signed command/callback envelopes (HMAC-SHA256 + per-build keys), command deduplication, a durable on-disk callback outbox with retries/backoff, OS-level separation (`studio-gw` vs. `studio-agent` so the agent gets `EACCES` on signing keys), and scoped Fly permissions (no production connectors or Sprite-creation/destroy/network-mutation privileges).


* **Why it matters:** Coding agents execute untrusted build/test code. Isolating them in a per-workspace Builder Sprite—without production credentials or signing keys—prevents lateral tenant compromise and keeps development messiness out of production. Splitting fake-backed work (Part 1) from live Sprite/recovery runs (Part 2) prevents simulated tests from being mistaken for live infrastructure proof.



#### 3. Phase C — Genuine Two-Way Interaction

* **What it establishes:** A real, non-interactive ask-and-resume loop where the agent surfaces genuine clarification questions instead of guessing, Studio records them in the business conversation, the user answers, and the same coding session resumes with that answer. It also tests messy operational edge cases: agent kills, Gateway restarts with pending callbacks, Sprite sleep/wake, Studio restarts, stale/duplicate callbacks, and cancellation.


* **Why it matters:** Most AI app builders silently guess when specs are ambiguous and fail late. Real two-way Q&A—backed by durable session resumption and failure testing—turns ambiguity into an auditable record rather than a hidden assumption.



#### 4. Phase D — Miniapp Build, Artifact Verification and Review

* **What it establishes:** Programmatic enforcement of the **Go + Miniapp framework + `MiniappSpec**` invariant (rejecting alternative languages/frameworks/runtimes). Studio verifies artifact digests and size limits on untrusted bytes, checks test reports (never treating claimed completion without passing required tests as success), and surfaces diffs, specs, test results, transcripts, and provenance for human review.


* **Why it matters:** An agent claiming "it works" is not verification. Gating review on concrete evidence and strict framework/digest checks protects business owners from broken logic, unverified claims, or rogue runtimes before anything goes live.



#### 5. Phase E — Approved Deployment to Runtime Sprite

* **What it establishes:** Promotion of *only* explicitly approved, digest-pinned artifacts into a separate **Runtime Sprite** running the Miniapp runtime, wired only to approved production connectors and network policies, verified via health checks, and supporting rollback to prior approved digests—with no coding agent or Gateway left on the Runtime.


* **Why it matters:** Production apps must run from immutable, verified artifacts rather than a mutable build checkout. Separating the Runtime from the Builder, enforcing health checks, and enabling digest-based rollback give business owners safe operations and reversible changes.



---

### Why This Phasing Matters & Its Place in the Bigger Picture

1. **Correctness before features & no premature shortcuts:** The roadmap explicitly forbids jumping ahead (e.g., building Phase C–E early, adding a second agent adapter, or wiring production connectors into Builder Sprites). Each phase proves a foundational guarantee before the next layer depends on it.


2. **Separating "what the AI did" from "what the business owns":** Business owners maintain the **conversation, approvals, receipts, and rollbacks**, not raw code or cloud terminals. Over time, this supports compounding value—reusable patterns, shared connectors, and a personal trust/audit history—while keeping human authority intact.


3. **Readiness beyond the 5 code phases:** Even after all five phase exits have evidence, Section 5 and Section 8 highlight operational readiness items needed before broad production scale—such as Builder/tenant lifecycle & compromise policies, usage metering/quotas, and SSE stream budgets—as well as open governance items (`F1` and `G-01`) that must not be bypassed.



Which part of the architecture would you like to dig into next—such as the security/isolation boundaries (Builder vs. Runtime and key separation), the Phase C Q&A/recovery mechanics, or how the business-user benefits in `writeup4.md` map to specific phase gates?


## deepseek

# The Five Phases — What They Prove and Why They're Sequenced This Way

The roadmap is not a build plan in the ordinary sense. It is a **de-risking sequence** where each phase establishes one property the product depends on, and the next phase would be irresponsible to start without that property being real. Read in isolation, each phase looks like "build a chunk of the system." Read in sequence, each phase answers a specific question that the following phase's claim depends on.

The correct frame for understanding the phases:

> **Each phase proves one thing. If it fails, the next phase has nothing to stand on. If it succeeds, the next phase's claim is meaningful.**

Here is what each phase proves, why the proof matters, and how it fits into the whole.

---

## The whole picture first

The product claim is:

> *"A business owner describes a process. An agent implements it. The owner reviews verified evidence. The owner approves. The app deploys through the Miniapp runtime. The owner can see, audit, evolve, and roll back everything."*

That claim has five load-bearing pieces, and each requires its own proof:

| Claim piece | What could be fake about it | What proves it |
|---|---|---|
| "A business owner describes a process" | The conversation is a chat log that forgets | Phase A |
| "An agent implements it" | The agent is a demo that only runs while you watch | Phase B |
| "The agent asks genuine questions and consumes answers" | The questions are theater; the agent guesses | Phase C |
| "The owner reviews verified evidence" | The output is the agent's self-report | Phase D |
| "The app deploys through the Miniapp runtime" | Deployment is a shortcut that bypasses the boundary | Phase E |

Without Phase A, the conversation is a lie. Without Phase B, the agent is a demo. Without Phase C, the questions are theater. Without Phase D, the trust is unearned. Without Phase E, the deployment collapses the security boundary the whole architecture was designed for. **Each phase is the thing that would be missing if the corresponding claim were false.**

---

## Phase A — Conversation Infrastructure: the trust-the-record layer

### What it delivers

The Studio-side conversation model: tenant-scoped database tables for conversations, messages, builds, turns, questions, answers, commands, events, artifacts, approvals, and deployment records. RLS both `ENABLED` and `FORCED` on all ten new tables. Write-path secret sanitization before persistence. Server-sent-events (SSE) with cursor replay for the browser. A callback endpoint that receives Gateway events idempotently. Genuine ordered-shutdown → fresh-boot recovery that a parked question can survive.

### Why it exists

Phase A answers a deceptively simple question: **"If everything in the system crashed right now, could you reconstruct the conversation exactly?"**

Most systems cannot. The conversation lives in a chat log that gets trimmed. The session lives in memory. The "build" is a temporary artifact. The user can't tell what actually happened three weeks ago because the system never had a durable record of it in the first place.

The architecture's central claim — "Studio owns the work, its history, its permissions, and its outcome" — only means something if **the history survives process death, restart, and every failure mode in between**. Phase A is where that claim becomes testable.

### What it de-risks in the whole system

Every subsequent phase writes to and reads from the Phase A tables. If the tables are not tenant-isolated, every later phase leaks tenant data. If the conversation is not durable, every later phase's "I'll show the user later" is a lie. If the question-answer lifecycle is not idempotent, every later phase will surface duplicate or stale answers.

**Phase A is the layer every other layer trusts.** If Phase A is not real, every later phase's claim is built on sand.

### What Phase A honestly does *not* prove

The roadmap is explicit: *"No live Sprite, OpenCode, plugin, or OAuth integration was exercised in Phase A."* Phase A proves the Studio side. It does not prove that any agent exists, works, or can interact. That is the correct scope — Phase A is the contract, Phase B is the counterparty.

### The honest cost of how Phase A was done

The consolidated review surfaced three defects during Phase A's closure: RLS was `FORCED` but not `ENABLED`; secret sanitization was applied on read but not on write; the restart test started a second server before shutting down the first. Those were found *because* the review exercised the Phase A tables with real multi-tenant traffic, real question content, and a real shutdown sequence. The correct read is: **Phase A was built against fakes, which is why the fakes' gaps didn't surface until review.** That is a general lesson the roadmap carries into Phases B–E: the fake proves logic, the live proves behavior, and you want to discover the gap between the two as early as possible. Phase B is explicitly structured — fake-backed Part 1, owner-run Part 2 — to make that gap visible instead of hidden.

---

## Phase B — Sprite Agent Gateway: the plumbing works end to end

### What it delivers

A running Agent Gateway inside a real Sprite, with a signed command/callback protocol, a per-build deduplication and retry outbox, a process supervisor for the coding agent, an initial OpenCode adapter, the dual-user OS layout (`studio-gw` vs `studio-agent`), the HMAC key provisioning drop that only the `studio-gw` user can read, and the install path as a persistent Sprite service.

Split deliberately into two parts:
- **Part 1 — fake-backed implementation.** All the logic, tests, and config, without touching a live Sprite. Proves the Gateway can correctly sign, dedup, retry, supervise a scripted double, and recover from simulated failures.
- **Part 2 — live installation, owner-run.** The owner runs the exact runbook. Provisions one Sprite, installs the Gateway as a service, verifies a signed `status` round trip, runs a no-op agent turn, puts the Sprite to sleep, wakes it with a new command, and verifies `agent.resumed` or an honest `agent.failed`. Records timings and resource cleanup.

### Why it exists

Phase B answers: **"Can the agent be a real, resumable, supervised worker on infrastructure it doesn't own?"**

This is where "the agent is a worker, not the application" stops being a diagram and becomes a process. It's the smallest end-to-end that could possibly prove the architecture: one Sprite, one Gateway, one no-op turn, one sleep, one wake. Nothing about the agent's intelligence is tested here. Only the plumbing.

### What it de-risks in the whole system

- **The Sprite lifecycle.** Every later phase assumes "the Sprite can sleep and wake, and the Gateway survives it." Phase B is the only phase where that assumption is tested in isolation.
- **The protocol.** Every later phase's Studio↔Gateway communication depends on the signed envelope, dedup, retry, and outbox behaving correctly under duplicates, losses, delays, and replay. Phase B is where those are proven.
- **The OS boundary.** The dual-user split is the only thing separating the agent from the Gateway's signing keys. If Phase B can't enforce it on a real Sprite, the security model is fictional and Phases C–E are moot.

### Why the fake-backed / owner-run split

This split is the roadmap's most underappreciated discipline. Part 1 is fake-backed because:
- It lets the logic be proven deterministically, without burning Sprite-hours.
- It puts the *test surface* under the assistant's control (scripted doubles, simulated sleeps, forced failures).
- It removes the temptation to prove the live system works by accident.

Part 2 is owner-run because:
- Credentials stay with the owner, as the security model requires.
- Live Sprite mutations are consequential and shouldn't be automated under a "the assistant probably knows what it's doing" assumption.
- The moment of truth is "the live Sprite behaves differently than the fake" — and that moment should be owned, not buried in a log.

The roadmap explicitly refuses to let a fake-backed test claim live behavior. That refusal is the whole point: **fake proves logic, live proves behavior, and the roadmap never confuses them**.

---

## Phase C — Genuine Two-Way Interaction: the questions are real

### What it delivers

A complete user ↔ Studio ↔ Gateway ↔ OpenCode round trip:
- User writes a requirement.
- Studio wakes the right Builder.
- OpenCode starts in the workspace and emits structured progress.
- OpenCode **asks a real clarification question.**
- Studio persists it and shows it in the original conversation.
- User answers through Studio.
- Gateway **resumes the same OpenCode session.**
- OpenCode **demonstrably consumes the answer as the question's resolution.**
- Work continues, and completion or failure is honestly reported.
- Studio's history and SSE replay remain correct across disconnect/reconnect.

### Why it exists

Phase C answers: **"When the agent asks a question, does the answer actually resolve it?"**

This is not rhetorical. The roadmap is explicit that "OpenCode's native question behavior under noninteractive `opencode run` is not proven." Three possible outcomes:
- The run emits a structured question event and exits/can be terminated safely. (Best.)
- The process blocks in a way the supervisor can manage. (Workable.)
- The process errors or behaves in a way that can't reliably accept a continuation. (Fatal to the native path.)

The roadmap is disciplined: test native first, fall back to a file-drop handshake second, and **if neither works, stop and return for owner decision — do not invent behavior**.

### What it de-risks in the whole system

- **The clarification ≠ authorization invariant.** Phase C is the first phase where the agent legitimately needs user input, and the platform's job is to make sure the input is *consumed as input*, not *inferred as authorization*. If Phase C proves "the question was asked and the answer was used," the invariant has teeth. If Phase C proves only "the transcript contains a question and an answer," the invariant is theater.
- **The recovery/failure matrix.** Phase C is where the full matrix (agent kill, Gateway restart with pending outbox, Sprite sleep/wake, Studio restart, duplicate/delayed/out-of-order callbacks, cancellation) gets exercised against a real agent. If Phase C can't survive these, Phase D and E will surface them later — at a worse time.
- **The line between "the agent is running" and "the agent is stuck."** Phase C is where Studio learns to distinguish `waiting_question` from `in_flight` from `failed`. Without that distinction, the user sees "in progress" for six hours and doesn't know whether to wait or intervene.

### What Phase C deliberately does not do

Phase C does not attempt to prove the agent's output is good, verifiable, or reviewable. That is Phase D. Phase C is narrow: the conversation works, the questions are genuine, the recovery matrix holds.

The roadmap is explicit: *"No new architecture should be added merely to make the demo look complete."* This is the phase where the temptation to over-demo is highest, and the discipline is to resist it.

---

## Phase D — Miniapp Build, Artifact Verification and Review: the trust is earned

### What it delivers

The coding agent produces a real, small Go/Miniapp application. Studio:
- Fetches the artifact and computes digests.
- Validates the MiniappSpec against the existing admission boundary.
- Validates the implementation against the **Go/Miniapp allowlist** — rejects Python, Node, React, Django, or any other application framework.
- Builds and tests with the real toolchain. (An agent's claim is not a passing test.)
- Presents the user with: a unified diff, the MiniappSpec, a machine-readable test report, the relevant transcript, provenance, and digests.
- Waits for the user to explicitly approve or request rework. **No auto-deploy in this phase.**

### Why it exists

Phase D answers: **"Can the user trust what they're approving?"**

Every "AI app builder" produces output the user can't meaningfully review. This phase is where the platform's central claim — "verifiable implementation evidence" — becomes testable. The user's trust comes not from "the agent said it works" but from "here is the diff, here is the spec, here is the test report, here is the digest."

### What it de-risks in the whole system

- **The locked application invariant.** The roadmap's binding constraint — Go only, Miniapp only, MiniappSpec canonical — is enforced *programmatically* at artifact validation. This is where a prompt-based instruction becomes a rejection boundary. If Phase D can't reject a Python artifact, the invariant is a slogan.
- **The artifact contract.** Digests, sizes, paths, symlinks, and source content are treated as untrusted. Phase D is where the platform proves it can validate an artifact *without* trusting the agent that produced it.
- **The review UX.** Phase D is where the platform learns what a meaningful review surface looks like. Without it, the "approve" button is a rubber stamp. With it, approval means the user saw evidence.

### What "no auto-deploy" means and why

This is one of the most important boundaries in the roadmap. Phase D produces an artifact. It does not deploy it. The boundary between Phase D and Phase E is a **human approval gate**.

If Phase D could auto-deploy, the entire governance model collapses. The "user approves" moment is the product's central value proposition — the user has authority, the agent proposes, the platform gates. Phase D is where that gate is built. Phase E is where it's used.

---

## Phase E — Approved Deployment to Runtime Sprite: the claim is delivered

### What it delivers

A specific, verified, digest-pinned artifact is promoted from the Builder to a **new Runtime Sprite**. The Runtime contains only the approved application (running through the Miniapp runtime), the approved production connectors, and the network policy the application needs. No coding agent, no Agent Gateway, no checkout, no development tools. Health checks pass. The deployment is recorded in Studio with digest, URL, and status. Rollback to a previously approved digest is tested.

### Why it exists

Phase E answers: **"Can the approved artifact become a real running application without the Builder bleeding into production?"**

This is where the Builder/Runtime separation proves itself. The Builder is where the agent works. The Runtime is where the app runs. If they merge — if the Runtime just copies the Builder, or if the deployment rebuilds from a mutable checkout — the security boundary the entire architecture was designed for collapses.

### What it de-risks in the whole system

- **The security boundary.** The Builder can run agent code, install dev dependencies, and mutate the checkout. The Runtime must not. Phase E proves the boundary is real.
- **The rollback promise.** If the deployed app has a problem, the user can revert to a previously approved digest. This requires that deployment is *by digest*, not by *checkout state*. Phase E is where that's proven.
- **The "no rebuild from uncontrolled state" invariant.** The Runtime installs the exact artifact that was approved. It does not re-fetch source, re-run the build, or trust anything from the Builder's working directory. This is the difference between "I deployed the thing you approved" and "I deployed a thing that was built from the thing you approved."

### Why it comes last

Because deployment is the least reversible action in the entire system. Everything else is preparatory — fake-backed, sim-testable, staging, rollback-testable. Phase E is the first moment where the user's business process is genuinely running on infrastructure that the user's customers will hit.

Everything before Phase E can be done in isolation. Phase E requires all the prior phases to have succeeded, because the artifact it deploys is the artifact Phase D verified, produced by an agent Phase C proved could ask questions, supervised by a Gateway Phase B installed, recorded in a conversation Phase A made durable.

**Phase E is the moment the product claim is delivered.** Everything before is what makes the claim honest.

---

## Why this order and not another

The sequence is not arbitrary. Try any other order and the argument breaks:

- **B before A**: The Gateway would talk to a conversation record that can't survive restart. Phase A's closure findings (RLS not enabled, restart test using two servers) would surface in Phase B, where they'd be blamed on the Gateway and harder to diagnose.
- **C before B**: The agent would be asked to ask questions through a Gateway that hadn't been proven to survive Sprite sleep. The conversation would work until the Sprite slept, and then the "waiting for an answer" state would hang.
- **D before C**: The agent would be producing artifacts before it had proven it could ask a question and consume an answer. The artifacts would be built on silent guesses, and the review evidence would show a passing test suite that didn't reflect what the user actually asked for.
- **E before D**: Deployment would bypass the verification step. The user would approve something they couldn't see; the artifact would be whatever the agent last produced; the Runtime would be the Builder.

The order is the order because **each phase's claim would be unprovable if an earlier claim were false**. The phases are not steps in a build; they are gates in a proof.

---

## The cross-cutting ideas that make the sequence work

**1. Each phase has an explicit "what it does not prove" statement.**

- Phase A: no live Sprite, no OpenCode.
- Phase B: no agent intelligence, only plumbing.
- Phase C: no artifact quality, only Q&A consumption.
- Phase D: no deployment, only verification.
- Phase E: nothing new — everything has been proven.

This is the honest distinction between inspected / simulated / live behavior. Every phase is explicit about its scope so that later phases don't over-claim what earlier phases proved.

**2. Each phase's fake-vs-live boundary is explicit.**

Phase B is the clearest example: Part 1 is fake-backed (no live Sprite, no OAuth, no Fly credentials), and Part 2 is owner-run (live Sprite, owner-held credentials, explicit GO). The discipline is: **the fake proves logic, the live proves behavior, and the roadmap never confuses them.** Phase A is fake-heavy (built against test harnesses), which is why its closure findings exist and why the review disciplines Phase B's split.

**3. Each phase preserves the invariants established by prior phases.**

- Phase B preserves Phase A's RLS, sanitization, and idempotency.
- Phase C preserves Phase B's OS separation and signed protocol.
- Phase D preserves Phase C's clarification/authorization distinction.
- Phase E preserves Phase D's digest-pinning and admission boundary.

At no point does a phase relax an earlier invariant to make its own work easier. If a phase needs to relax one, that's a design decision that must be surfaced, not absorbed.

**4. Each phase's exit criterion is a real proof, not a demo.**

- Phase A: closing migration evidence, RLS catalog+behavioral tests, sanitization canary tests, genuine restart test, M1–M4 corrections.
- Phase B Part 1: full gates, capability matrix (executed vs. unknown), owner-run live runbook.
- Phase B Part 2: live Sprite, live sleep/wake, live signed `status`, live `agent.resumed`.
- Phase C: documented live two-way interaction, with the recovery/failure matrix proven.
- Phase D: rejected non-Go artifacts, verified digests, presented evidence, explicit approval.
- Phase E: approved digest deployed, health check passed, rollback tested.

No phase exits on "it looks good." Each exits on a specific piece of evidence that the claim is real.

---

## The one-sentence summary

**The five phases are the five de-risking gates of the product claim: the conversation is durable (A), the Gateway survives the Sprite lifecycle (B), the agent's questions are genuine (C), the artifacts are verified (D), and the deployment is governed (E) — and each claim is only meaningful if the earlier claims are true, which is why the sequence exists.**

Read this way, the roadmap is not a build plan. It is a **proof structure**. The phases are the lemmas; the theorem is the product claim; and the discipline of "each phase proves one thing before the next phase starts" is what makes the theorem's proof valid rather than asserted.