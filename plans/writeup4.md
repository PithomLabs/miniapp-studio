## prompt

if all goes well as planned what benefits does the target market end users expect to reap 

## gemini

If everything goes according to plan, the target market end users—**business owners** describing business processes in plain language—reap value because the architecture hides low-level infrastructure and coding-agent messiness while keeping business ownership and safety intact:

### Core Benefits for End Users (Business Owners)

* **Plain-Language Workflow Creation Without DevOps or CLI Overhead:** Business owners can describe a business process in ordinary language and get working software built for them without opening a terminal, managing coding-agent sessions, or dealing with underlying cloud infrastructure.


* **One Continuous, Reliable Conversation:** Everything happens in a single permanent Miniapp Studio conversation that survives browser disconnects, Sprite sleep/wake cycles, and server/process restarts without losing history or context.


* **Meaningful Clarification Instead of Guesswork:** When requirements are ambiguous, the system surfaces clear clarification questions and choices in the conversation rather than silently guessing or conflating questions with approval.


* **Verifiable, Auditable Output Before Going Live:** Before deploying anything, users can inspect concrete review artifacts—such as source diffs, generated specs, test results, and provenance/digest verification—so they know what was built and whether tests actually passed rather than taking an agent's word for it.


* **Explicit Control Over Real-World Actions & Production Rollouts:** Consequential business permissions and production deployments require explicit human approval. Only approved artifacts are promoted into a separate Runtime Sprite with scoped production capabilities, and previous approved versions can be rolled back cleanly.


* **Safer Multi-Tenant Isolation & Credential Boundaries:** Untrusted build/code execution stays isolated in a Builder Sprite without direct production credentials, helping protect tenant data and business accounts from runaway or unsafe agent actions.



---

### One Practical Caveat for the Business Pitch

Even when the technical workflow runs smoothly, **end users still have to review diffs/specs/tests and make real business or authorization decisions**—so the biggest win for business owners is **dramatically faster, governed software creation and safe iteration**, rather than "zero human thought required."



## deepseek

# What End Users Reap If the Architecture Works

**Frame first.** The target end user is a business owner or operator — a co-op manager, an SME founder, an MFI operations lead — who has a process in their head, runs it today on spreadsheets, email threads, three SaaS subscriptions, and a part-time contractor. They are not technical. They have never successfully bought custom software because the cost of a bespoke build exceeds the value of the process, and off-the-shelf SaaS never quite fits.

The architecture described in `writeup2.md` delivers benefits at six moments in the user's journey. Each maps to a specific architectural property, and the strength of each benefit depends on which phase is actually working.

---

## 1. The describe moment — from "I need software" to "I described software"

**What the user does today.** Writes a requirements doc, gets quotes from three agencies, picks one, waits 3–6 months, pays £15k–£80k, receives software that solves 70% of the problem, hires a contractor for the other 30%.

**What the architecture delivers.** The user opens Studio, types: *"Every Friday, summarize the reports in my Box folder and post them to Slack — but ask me before posting."* Studio acknowledges, assigns the build to their Builder Sprite, and the agent starts working. The user has not installed anything, has not been asked for a technical decision, and is not waiting for a quote.

**Architectural property this depends on:** Studio as the permanent conversation surface (§1, §2.1), Builder Sprite assigned to the workspace (§2.2), agent assigned the build (§3).

**What makes this different from "chat with an AI."** The conversation is durable. If the agent asks a question at 3am and the user answers at 9am, the work resumes from exactly where it stopped. If the Builder Sprite sleeps overnight, the user's next message wakes it. The conversation does not reset because the environment did.

**What the user should not expect.** The describe moment does not produce a deployed app. It produces a conversation and a build that the user will review.

---

## 2. The question moment — genuine clarification without a technical translator

**What the user does today.** Answers a developer's questions through a product manager, losing fidelity at every hop. *"What I actually meant was..."* appears three weeks after the wrong assumption was baked in.

**What the architecture delivers.** The agent asks the user directly: *"You said 'reports folder' — do you mean the top-level folder or subfolders too? And should the summary include file names or just content?"* The user answers in the same conversation. The agent continues with the answer as authoritative.

**Architectural property this depends on:** The clarification/proposal/authorization distinction (§6.2), the question-and-answer lifecycle in the adapter (§4.3), Studio as the durable conversation record (§7.1).

**Why this matters more than it sounds.** Most "AI builds apps" products silently guess when they lack information, and the user discovers the guess in the output. Studio's architecture forces the agent to *ask* rather than assume, and the user's answer becomes part of the durable record — not lost in a chat log, but attached to the build, auditable later, and available if the agent needs to re-derive a decision.

**What the user should not expect.** Questions are not a sign the system is broken. They are the mechanism by which the system avoids producing software based on guesses.

---

## 3. The review moment — evidence, not vibes

**What the user does today.** Receives a demo from the agency, sees the happy path work, signs off, discovers the failure cases in production.

**What the architecture delivers.** Studio shows, for each completed build:

- A unified source-code diff.
- The generated `MiniappSpec` (what the app will actually do, in structured form).
- A machine-readable test report (which fixtures passed, which failed).
- The agent's execution transcript (the reasoning the agent did).
- Artifact provenance (SHA-256 digests, what the agent actually produced vs. what was claimed).

The user does not need to read the code. The user needs to see: *here are the specific behaviors the app has been tested against, here are the specific paths it touches, here is what happens when it fails.*

**Architectural property this depends on:** Phase D artifact pipeline (§10, Phase D), digest verification (§3, §10), MiniappSpec admission and validation (§10 Phase D).

**The concrete benefit.** A build that reports success but fails its tests is *not presented as success* (§10, Phase D). The user sees the same evidence the platform sees. That is a different class of trust than "the agency said it works."

**What the user should not expect.** A working app on first build. Most meaningful builds require 1–3 rounds of correction, which is why the conversation continues rather than being a one-shot.

---

## 4. The approval moment — authority without technical literacy

**What the user does today.** Grants a contractor admin access to their business systems and hopes. Or manually approves each individual action in a SaaS dashboard, one click at a time.

**What the architecture delivers.** Every consequential action the app takes is gated by a decision the user makes, with the exact payload shown and the decision bound to the content. *"Restart the crash-looping machine (expires 30m)?"* The user approves on their phone. The action executes and a receipt appears. If the action times out mid-execution, it is not silently retried — it enters the UNKNOWN state and asks the user to reconcile.

**Architectural property this depends on:** Approval gates in the Miniapp framework, payload-bound approval (§6.1), separate authorization requests for consequential actions (§6.2), UNKNOWN quarantine (§ effect ledger).

**The concrete benefit.** The user does not have to choose between *granting access and hoping* or *manually approving every step*. They grant the app a defined envelope (this connector, these paths, this limit) and get asked only when something consequential happens. The envelope is visible in the permissions card before the app is deployed.

**What the user should not expect.** Zero approvals. An app that acts autonomously on the user's business systems without their involvement is not what the architecture delivers; an app that never requires the user to do anything is a policy choice the user makes about which actions are consequential, not a property of the platform.

---

## 5. The operate moment — receipts, not reconstruction

**What the user does today.** When something goes wrong, reconstructs what happened from logs, email threads, and memory. If the contractor is gone, the reconstruction is impossible.

**What the architecture delivers.** A run timeline showing every step the app took, every approval it requested, every receipt it filed, and every wait it is in. If the app sent an email three weeks ago, the receipt exists. If the app was cancelled mid-run, the in-flight effects were reported, not hidden. If the app was checkpointed before an upgrade, the user can rewind to the checkpoint and the plan resumes from where it stopped.

**Architectural property this depends on:** Studio as the permanent record (§2.1), run timeline (§6), effect ledger, checkpoint and rewind (§10 Phase E references, Sprite checkpoint).

**The concrete benefit.** The user can *answer the question "what did my app do?"* without technical help. That is the difference between an app that the user trusts and one they tolerate.

**What the user should not expect.** A guarantee that the app can always undo external effects. Some actions are irreversible (an email sent, an invoice paid). The architecture flags those as irreversible in the plan and requires explicit approval; it does not undo them after the fact.

---

## 6. The evolve moment — compounding capability, not a new project

**What the user does today.** Wants to add a feature to last year's build. The agency quotes a new project. The user either pays or lives with what they have.

**What the architecture delivers.** The user returns to the same conversation, types *"also check for duplicate invoices before posting,"* and the agent produces a new version of the plan. The conversation retains every prior decision. The Builder Sprite retains the checkout, the session history, the last-reviewed artifact. The user reviews only the diff. If the new version regresses, they rewind to the prior approved digest.

**Architectural property this depends on:** Phase E promotion by artifact digest (§10 Phase E), Runtime Sprite rollback to a previously approved digest (§10 Phase E), conversation history preservation (§7.1).

**The concrete benefit.** The user's second app is faster than the first. The third is faster still. The conversation, the patterns, and the corrections accumulate. Over 12 months, the user has a portfolio of apps they understand, all built in the same interface, all subject to the same governance.

**What the user should not expect.** That the platform will produce code they can maintain themselves. The user maintains the *conversation*, not the code. If they wanted to maintain code, they would hire a developer.

---

## What compounds over time

Three things accumulate that the user should feel:

**1. The pattern library.** Each successfully generated app contributes to the templates available for the next generation. A user who builds five apps sees the sixth generated in minutes because their patterns are already in the library.

**2. The connector catalog.** Every Custom API connector one user adds becomes visible to the platform. A user who connects their niche accounting system on Monday sees their neighbor's niche CRM connector on Tuesday. The catalog grows without a central integration team.

**3. The trust record.** Approvals, receipts, and reversions accumulate. Six months in, the user can look at *"here are the 40 actions my apps took on my behalf, here are the 6 I approved, here are the 3 I rewound."* That record is the user's own governance history, not a compliance artifact imposed from outside.

---

## What makes this different from the AI app builders that exist today

| Property | Generic AI app builder | Studio with this architecture |
|---|---|---|
| Durable conversation | Session-scoped chat | Permanent, replayable, auditable record |
| Clarification | Guesses when missing info | Asks, records, resumes on answer |
| Review evidence | "It works, trust me" | Diff, spec, test report, transcript, digests |
| Consequential actions | Executes directly | Approval-gated, payload-bound, receipted |
| Failure handling | Retries or fails silently | UNKNOWN quarantine, receipts, rewind |
| Evolution | New project or fork | Same conversation, diff-only review |
| What the user owns | An app | The conversation that produced the app |

The user's trust in the system is not "the AI is good." It is "I can see what it did, I can approve what it proposes, and I can undo what I don't want."

---

## The honest bounds — what "if all goes well" actually requires

The benefits above are contingent on five things actually happening. If any of these fail, the corresponding benefit degrades:

1. **Phase A–E actually complete.** The writeup is explicit that Phase A is not yet accepted, Phase B–E are pending. The benefits of the operate and evolve moments require Phase D (artifact review) and Phase E (deployment) to be working. If the project stalls before Phase E, the user gets a described-and-built app they cannot deploy.

2. **The agent's question lifecycle actually works.** If OpenCode cannot reliably ask-and-resume (F2 in the prior review), the clarify moment degrades to guessing, and the review moment becomes the *only* place the user catches wrong assumptions. That is still better than today's agency model but worse than the design promises.

3. **The connector catalog grows.** The custom API connector support is in the architecture, but its value depends on users adding connectors. A platform with 5 connectors is not the same product as one with 50.

4. **The approvals layer works on phones.** The design assumes mobile approval. A user who has to open a laptop to approve a receipt is a user who stops using the system.

5. **The 30-day, 90-day, and 12-month retention holds.** GigaCatalyst's 89% day-30 retention is the category benchmark. If Studio's retention falls below that, the compounding benefits (pattern library, trust record) do not accrue, and the platform becomes another "I tried an AI app builder once."

**The honest summary.** If everything works, the end user gets: software that fits their process, built in an afternoon, with governance they can see and approve, and evolution they can drive without hiring anyone. If some things work, they get a better agency alternative. If most things fail, they get a demo.

The architecture is sound. The benefit delivery depends on execution — the phases landing, the adapters working, the retention holding. The writeup's §12 discipline of "correctness first, features second" is the correct posture for ensuring the benefits are real rather than promised.