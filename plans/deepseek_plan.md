# Hacksgiving: 12 Agent Ideas and Top 3 Picks

## Winning strategy

The hackathon rules are strict but shallow: **an agent that runs on a Sprite, uses at least one connector**. The judges (TypeSafe AI, Box, Pinecone, Ollama, Fly.io) determine what "wins." Based on the judging panel and the hackathon's tone ("personal agent", "weird uncle award"), a winning entry will:

1. **Be obviously a personal agent** — one person, one job, done well. Not a multi-tenant SaaS.
2. **Use the Sprite for something the Sprite alone enables** — persistence, checkpointing, cost-zero when idle, hardware isolation for agent-authored code. A 24/7 Fly Machine would work for many ideas; the winners should be those where a Sprite is the right primitive.
3. **Use a connector deeply** — not for a token "Pinecone integration" line item, but as the substrate for the agent's memory, documents, or reasoning.
4. **Show human-in-the-loop** — the interesting personal-agent question is "how much can I delegate safely?" The Miniapp framework's approval gates are the answer.
5. **Demo in under 3 minutes** — the judges' time is finite. One clear flow beats five.

The Miniapp framework's differentiators map directly to what makes a strong entry:

| Framework feature | What it enables in a personal agent |
|---|---|
| Approval gates | The agent proposes, the human disposes — the trust pattern for delegated authority |
| Durable execution (DBOS) | Runs overnight or for days without losing work |
| Effect ledger with UNKNOWN quarantine | Safe external actions — no double-sends, ambiguous outcomes are reconciled, not retried |
| Credential isolation (capability proxy) | The agent never sees passwords; accounts stay safe even if the agent goes rogue |
| Per-family versioning | Safe upgrades while workflows are in flight |
| MCP surface | Works with Claude Code, Codex, Cursor — the harnesses inside the Sprite |

The framework turns the personal agent from "a chat interface with tools" into "a durable operating layer with delegated authority." That is what the winners should showcase.

---

## The 12 ideas

Each idea listed with: **what it does · connector · Sprite-essential angle · framework differentiator · "why it wins or doesn't"**

### 1. Chief of Staff
Email, calendar, and task triage with a morning approval inbox. The agent drains the inbox overnight, drafts replies, proposes calendar changes, and surfaces decisions needing human input. The user approves the batch in ten minutes.
**Connectors:** Box (drafts and docs), Pinecone (past-email context), Ollama (private reasoning on sensitive emails). **Sprite angle:** runs 24/7, checkpointed between days. **Framework:** approval gates, effect ledger for sends, durable execution for overnight batches. **Why it wins:** universal, familiar, uses all three partner connectors. **Why it doesn't:** many Chief of Staff agents exist; the judges have seen this shape.

### 2. Weekend Ops
A personal agent that only works weekends. Friday night, kick off. Saturday and Sunday, it drains the household admin backlog (bills, subscriptions, meal planning, appointments, warranties). Sunday evening, present an approval inbox. Monday, everything is executed and the Sprite is checkpointed and stopped. Next Friday, resume from checkpoint — smarter than last week. **Connectors:** Box (receipts, contracts), Pinecone (past decisions). **Sprite angle:** the entire product is the Sprite's persistence + checkpointing + cost-zero when idle. **Framework:** durable execution for weekend-long batches, approval gates for consequential actions. **Why it wins:** Sprite-native framing turns platform characteristics into a product feature; novel; memorable. **Why it doesn't:** might be seen as gimmicky; hard to demo the "resumes next weekend" story in a 3-minute video.

### 3. Personal SRE for Solo Developers
A 24/7 agent that watches your side projects, diagnoses incidents, and proposes fixes. Uses the Sprite's hardware isolation to safely run agent-authored code — tests, rollback drills, log analysis — without touching production. Approves on your phone before rolling back.
**Connectors:** Sentry (errors), Fly Redis (state), Box (postmortems). **Sprite angle:** hardware isolation is the whole point — the agent can run untrusted code safely. **Framework:** approval gates for rollbacks, effect ledger for deploy actions, per-family versioning for safe upgrades. **Why it wins:** Fly.io-native, uses Sprites for their actual purpose, deep developer audience appeal. **Why it doesn't:** narrow audience (only developers), hard to demo without a live incident.

### 4. Second Brain
Continuous knowledge curator. Reads RSS, newsletters, email, saved articles, and social, indexes semantically, and surfaces daily insights. The "second brain" improves over time because the Sprite is persistent and the corpus is durable.
**Connectors:** Pinecone (vector index), Ollama (local summaries), Box (long-term archive). **Sprite angle:** continuous ingestion + persistent corpus. **Framework:** durable execution for ingestion, approval gates for "publish to shared". **Why it wins:** demonstrates Pinecone + Ollama + Box together; the "recall a fact from three months ago" demo is genuinely impressive. **Why it doesn't:** crowded space (Readwise Reader, Mem, Notion AI); differentiation is unclear.

### 5. Travel Concierge
Plans and books travel with approval gates. The agent researches options, proposes an itinerary, holds flights (gated), books after approval, and reconciles if any step times out.
**Connectors:** Box (itineraries, receipts), Pinecone (past trip preferences). **Sprite angle:** persistent memory of travel preferences; runs while you sleep. **Framework:** approval gates for expensive actions, effect ledger for booking reconciliation (timed-out bookings are reconciled, never double-booked). **Why it wins:** the "effect ledger saved me from a double-booking" story is compelling. **Why it doesn't:** flight/hotel APIs are painful to integrate in a week; the demo depends on a mock provider.

### 6. Personal CFO
Watches accounts, categorizes spending, forecasts bills, and suggests transfers or cancellations. All financial actions are gated behind approval.
**Connectors:** Box (statements, tax docs), Pinecone (natural-language search over finances). **Sprite angle:** holds sensitive financial context that shouldn't be in an LLM's context window. **Framework:** approval gates for all transfers, effect ledger for banking reconciliation, credential isolation for account access. **Why it wins:** compelling trust story ("my agent handles money but can't move it without me"). **Why it doesn't:** regulatory/compliance concerns; bank APIs are restricted; hard to demo without real accounts.

### 7. Content Pipeline
Drafts posts, schedules them, publishes with approval. Learns the user's voice from past content.
**Connectors:** Box (drafts, assets), Pinecone (past content for voice), Ollama (local model for drafts). **Sprite angle:** runs persistently, drafts overnight, publishes with approval. **Framework:** approval gates for publishing, effect ledger for cross-platform posts (a timed-out tweet doesn't double-post). **Why it wins:** clear demo (write → review → publish), uses all three partner connectors. **Why it doesn't:** content generation is crowded; the differentiation is the safety layer, which is harder to demo than generation quality.

### 8. Legal/Admin Assistant
Processes paperwork — contracts, government forms, insurance claims — with review and signature gates. Tracks deadlines.
**Connectors:** Box (documents), Pinecone (past contracts, templates). **Sprite angle:** persistent memory of ongoing cases; runs while you sleep. **Framework:** approval gates for signatures, effect ledger for form submissions (a timed-out submission is reconciled before resubmission). **Why it wins:** Box judges will love document workflows; universal pain point; trust story is obvious. **Why it doesn't:** narrow use case; hard to demo without realistic documents.

### 9. Health Tracker
Ingests health data (wearables, labs, portal data), coordinates with providers, requests prescription refills (gated).
**Connectors:** Pinecone (semantic search over personal health docs), Box (records). **Sprite angle:** persistent memory of health history; runs nightly. **Framework:** approval gates for prescription refills (safety-critical), effect ledger for provider portal requests (a timed-out request is reconciled before resubmission), credential isolation for portal access. **Why it wins:** safety-critical framing makes approval gates essential, not optional; strong trust story. **Why it doesn't:** health data is sensitive; HIPAA concerns; hard to demo without real data.

### 10. Personal Chef
Plans meals, generates recipes, builds shopping lists, orders groceries. Learns preferences from past meals.
**Connectors:** Box (recipes), Pinecone (past meals, preferences), Ollama (local recipe generation for privacy). **Sprite angle:** persistent memory of "what's in the fridge", "what you liked", "what you threw out". **Framework:** approval gates for grocery orders over a threshold. **Why it wins:** **Thanksgiving-themed!** Judges will smile; the "plan my Thanksgiving dinner" demo is memorable. **Why it doesn't:** feels small compared to Chief of Staff or SRE; harder to justify a Sprite versus a cron job.

### 11. Research Assistant
Deep dives on demand. The agent spends hours researching a topic, collecting notes on the persistent filesystem, and produces a report with citations.
**Connectors:** Pinecone (research corpus), Box (published reports). **Sprite angle:** persistent memory of past research; the Sprite is the agent's "study." **Framework:** durable execution for hours-long jobs, effect ledger for external API rate limits. **Why it wins:** showcases long-running durable execution; the "resume after a crash" story is compelling. **Why it doesn't:** research agents are common; hard to differentiate.

### 12. Board of Advisors
Simulated advisors (optimist, skeptic, operator, contrarian) that help you make decisions. Tracks decisions and outcomes to improve advice over time.
**Connectors:** Pinecone (past decisions, outcomes), Box (decision journal), TypeSafe AI (type-safe agent interactions). **Sprite angle:** persistent memory of every decision and outcome; the agent's value compounds over months. **Framework:** durable execution for long decision-tracking workflows. **Why it wins:** "weird uncle" potential — genuinely strange and charming; uses TypeSafe AI prominently. **Why it doesn't:** less obviously useful; hard to demo value in 3 minutes; might be seen as a toy.

---

## Top 3 picks and reasoning

### 🥇 1. Weekend Ops

**The pitch.** A personal agent that works only on weekends. Friday night at 6pm, the agent wakes up (the Sprite un-checkpoints from last Sunday's snapshot). Saturday and Sunday, it drains the household admin backlog — reviewing bills, flagging unused subscriptions, proposing meal plans, scheduling appointments, checking warranties, requesting quotes for repairs. Sunday at 6pm, it presents a single approval inbox: "here are 14 things I found, 9 I recommend executing, here's what each costs." You approve the 9, the agent executes (calling miniapps via MCP), and Monday morning everything is done. The Sprite checkpoints and stops. **$0 the rest of the week.** Next Friday, it resumes with everything it learned.

**Why it wins the grand prize.**

- **Sprite-native as a product feature.** The "agent that only works weekends" framing turns the Sprite's persistence + checkpointing + cost-zero-when-idle into a genuine product story. A 24/7 Fly Machine would be wasteful for this workload; a serverless container would lose context between weekends; only a checkpointable, persistent, disposable machine makes the economics and the product work.
- **The approval gate is the feature, not a safety add-on.** The pitch to the user is "you delegate the entire weekend's admin, and you trust the agent because nothing executes without your approval." That's the Miniapp framework's approval gates as a product surface, not a compliance checkbox.
- **The effect ledger is the invisible hero.** Every bill payment, subscription cancellation, appointment booking, or grocery order goes through the effect ledger. If a payment times out, it's reconciled, not double-charged. The demo can *show* this: "the agent's payment timed out, and instead of double-charging you, it reconciled by reading the provider — this is what makes it safe to delegate money."
- **The durable execution is the other invisible hero.** A weekend is 48 hours; the agent runs many flows. If the Sprite crashes at hour 30, DBOS resumes from the last checkpoint. The demo can show the Sprite being stopped and restarted, and the flow continuing.
- **The narrative arc fits Hacksgiving.** "My agent takes Thanksgiving weekend off. It works the *next* weekend." The judges will smile.
- **Demo is 3 minutes.** Friday night kickoff → Saturday morning "watch the agent work" (a screen showing the approval inbox populating) → Sunday morning "here are 14 items, I recommend 9" → one click to approve → Monday "done, $0 for the week, checkpointed, resumes next Friday".

**Risks.**
- The "checkpointed and stopped" story is the strongest differentiator but the hardest to demo compellingly in a short video. Mitigation: pre-record the checkpoint/restore in a screen capture with timestamps.
- The breadth of "household admin" might dilute the demo. Mitigation: pick *three* concrete flows for the demo — bills, subscriptions, meal plans — and show them end to end.
- Requires integrating real connectors (Box for receipts, Pinecone for decisions) in addition to the framework itself. Mitigation: use realistic mock data for the demo, integrate the real Box connector for the "receipts are stored in Box" moment.

**Connector choice:** Box (primary, for receipts and contracts) + Pinecone (past decisions to improve recommendations).

---

### 🥈 2. Personal SRE for Solo Developers

**The pitch.** A 24/7 agent that watches your side projects. When something breaks, it diagnoses in an isolated Sprite (hardware-isolated, so it can safely run agent-authored code — tests, rollback drills, log analysis — without risking the production environment). It proposes a fix, you approve on your phone, and it executes. Weekly, it writes you a postmortem in Box that you can share.

**Why it wins the grand prize.**

- **Uses Sprites for their intended purpose.** The hardware isolation is the whole point. The agent can run untrusted code — code the agent wrote, or code from a bug report — inside the Sprite, without the "will this break production" fear that makes every other agent demo feel sketchy. This is exactly the kind of Fly.io-native story judges will reward.
- **The approval gate is the safety feature.** Rolling back a deployment is consequential. The agent's job is to *propose* the rollback with evidence, then the human approves. The Miniapp framework's approval gates make this a first-class flow.
- **The effect ledger matters for deploy actions.** "The agent's deploy command timed out after the platform already started rolling — did it work or not?" The framework's UNKNOWN-quarantine means the agent reconciles state before retrying. This is the "production-grade" story that developer judges appreciate.
- **The audience is the judges.** Fly.io and its developer audience *are* solo developers with side projects. The judges will see their own pain in this.
- **"Weird uncle" potential.** An agent that writes its own postmortems — and complains about its own agent-authored bugs — is genuinely strange and charming.

**Risks.**
- Narrow audience (only developers). Might be seen as "for a subset of judges, not for the grand prize".
- Requires integrating Sentry + Fly Redis + Box in addition to the framework. More integration work than Weekend Ops.
- Demo requires triggering a realistic incident. Mitigation: use a real side project with real Sentry data, or a carefully staged demo.

**Connector choice:** Sentry (primary) + Fly Redis (state) + Box (postmortems).

---

### 🥉 3. Chief of Staff

**The pitch.** The universal personal agent. The Sprite drains your inbox overnight, drafts replies in your voice (using past sent emails as context), proposes calendar changes, and surfaces the five things you actually need to decide today. You open the approval inbox in the morning with coffee, approve 12 of 15 drafts, and the batch executes.

**Why it wins the grand prize.**

- **Universally understood.** Every judge has an email backlog. The problem is immediately obvious, the demo is immediately compelling.
- **Uses all three partner connectors naturally.** Box for documents, Pinecone for past-email semantic context (voice matching), Ollama for local reasoning on sensitive emails (privacy). Judges from each partner company see their product being genuinely useful, not bolted on.
- **The approval gate is the core interaction.** The user's whole morning routine is "review the batch, approve, done." The framework's approval gates become the product experience, not a compliance feature.
- **The effect ledger matters invisibly.** "The agent tried to send a reply to your boss, the SMTP timed out, did it send twice?" The framework's reconciliation means no double-emails — a nightmare avoided.
- **Durable execution is essential.** Overnight batches mean the Sprite runs for hours; if it crashes, DBOS resumes. The demo can show this.

**Risks.**
- **Many Chief of Staff agents exist.** Lindy, Superhuman AI, and every "AI executive assistant" startup are doing this. The differentiation must be sharp in the pitch ("we use Sprites for persistent, private, checkpointed runs") or the entry gets lost in the crowd.
- **Requires a real email account** for a compelling demo. Mitigation: use a dedicated hackathon email with realistic staged content.
- **Ollama integration** adds complexity (local model in Sprite, choosing which emails go to local vs. hosted model). Mitigation: for the demo, route a specific category of sensitive email through Ollama and everything else through a hosted model, and say so.

**Connector choice:** Box (drafts, docs) + Pinecone (past-email voice) + Ollama (local privacy reasoning).

---

## Why this order

**Weekend Ops first** because it has the best "novel + useful + demo-able + Sprite-native" combination. It uses every Miniapp framework differentiator as a *product feature*, not just an implementation detail. The "agent works only weekends" framing is memorable, the checkpoint/restore story is a Fly.io-native slam dunk, and the demo arc fits Hacksgiving's pie-and-turkey tone.

**Personal SRE second** because it's the most Fly.io-native. It uses Sprites for exactly what Sprites are built for (hardware isolation of agent-authored code), and it appeals directly to the judges' own pain. The "weird uncle" postmortem angle is a bonus.

**Chief of Staff third** because it's the safest but also the most crowded. It will score well on all axes but may not stand out. If the weekend framing feels too gimmicky after a day of prototyping, Chief of Staff is the fallback that still has a strong shot.

---

## Buildability in one week, given Miniapp exists

All three are buildable in 7 days because the framework's hard parts (DBOS, effect ledger, approval gates, capability proxy, MCP surface, credential isolation) already exist. The hackathon work is:

1. **Define the miniapps** — the business operations the agent invokes (e.g., `bills.check`, `bills.pay` (gated), `subscriptions.audit`, `subscriptions.cancel` (gated)).
2. **Build the MCP orchestration** — the agent's system prompt and tool-use patterns (the harness inside the Sprite, e.g., Claude Code or Cursor, calls the miniapps via MCP).
3. **Integrate the connectors** — Box, Pinecone, Ollama (or Sentry + Fly Redis for SRE).
4. **Build the approval inbox** — a minimal web UI that shows pending approvals and lets the user approve/reject. (This is small — a few hundred lines.)
5. **Demo the flows** — pick two or three concrete flows and stage them for the video.

Weekend Ops is the most novel framing, Personal SRE is the most Fly.io-native, Chief of Staff is the most universal. Pick one and go deep. For "win the top prize," I'd bet on **Weekend Ops** — it's the one entry the judges will *remember* on Tuesday when they're picking winners, because nobody else will have built an agent that works only on weekends.
