# Yes — This Is the Correct Entry

The 12-idea list was the wrong framing. It treated the Miniapp framework as a foundation for building individual products, when the framework's actual thesis is that **individual products should be generated, not built**. Building 12 hand-made agents would be an entry that contradicts its own foundation. Building the generator is the thesis.

The entry is: **a Studio where an end user describes a personal agent in plain language, and a working agent appears — deployed to its own Sprite, wired to a connector, with approval gates, checkpoints, and durable execution underneath.**

That is the hackathon entry. The judges see the Studio, not the framework.

---

## Why this wins where the 12 ideas would lose

**The judges remember the factory, not the products.** On Tuesday, comparing 30 entries, "an agent that triages email" is indistinguishable from the other four email-triage entries. "The one where you typed a sentence and an agent appeared on its own Sprite, connected to Box without ever seeing my password, with an approval gate and a checkpoint button" is not.

**The five partners are all visible in a single demo.** Box is the connector the generated agent uses. Pinecone is the memory layer. Ollama is the local reasoning model the agent opts into. Meta Muse Spark is the LLM the Studio uses for generation. TypeSafe AI is the type-safe schema the generator emits against. Every partner sees their product being *structurally* used, not bolted on.

**It uses every Sprite primitive for its actual purpose:**

| Sprite primitive | Why the Studio needs it |
|---|---|
| **Hardware-isolated compute** | Every generated agent gets its own machine. The "personal agent" story only works if the user's data is isolated from every other user's. |
| **Persistence + checkpointing** | The agent learns over weeks. Checkpoint before upgrade, restore on regression. The Studio has a button for this. |
| **Connector gateway** | The agent uses Box/Pinecone/Ollama without the credentials ever touching the Sprite. The Studio shows the policy. |
| **Sprite services** | Long-running agents (the SRE watching your projects, the Chief of Staff draining email overnight) need to keep running between sessions. |
| **Fly MCP** | The Studio provisions Sprites programmatically via `mcp.fly.io` with limited OAuth scope. |

**Every Miniapp framework differentiator is a *visible product feature*, not an implementation detail.** Approval gates are the "you approve before I execute" UX. Durable execution is the "your agent runs overnight and doesn't lose work" promise. Effect ledger is the "your agent won't double-charge you" guarantee. Credential isolation is the "your passwords never reach the AI" trust story. Per-family versioning is the "upgrade your agent without breaking it" button.

**The "weird uncle" angle is in the meta-level.** "This agent builds agents. I typed one sentence and it made me a personal CFO." That's the strange-in-a-good-way story the Weird Uncle award is for.

---

## The architecture

```
┌─────────────────────────────────────────────────────────────┐
│  MINIAPP STUDIO (the entry the judges see)                  │
│  Web UI: describe → preview → deploy → monitor → approve    │
│  Backend: LLM generates MiniappSpec (type-safe schema)      │
│           → validates deterministically                     │
│           → calls Fly MCP to provision Sprite               │
│           → configures connector policy                     │
│           → deploys agent runtime                           │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       │ Fly MCP (mcp.fly.io, OAuth)
                       │ Sprites API (connectors, checkpoints)
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  GENERATED AGENT (runs on its own Sprite)                   │
│  - Miniapp runtime (from the framework)                     │
│  - MCP server exposing the generated miniapps               │
│  - Harness (Claude Code / Codex / Cursor) inside the Sprite │
│  - Connector calls go through the Sprites API gateway       │
│    (credentials never touch the Sprite)                     │
│  - Approval gates for consequential actions                 │
│  - DBOS for durable execution                               │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       │ Sprites connector gateway
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  CONNECTORS (Box / Pinecone / Ollama / Meta Muse Spark)     │
└─────────────────────────────────────────────────────────────┘
```

**What the Studio must expose as generic primitives:**

| Primitive | UI element | What it maps to |
|---|---|---|
| **Describe** | A single text box + example prompts | The intent input to the generator |
| **Spec** | A readable "here's what I'll do" card with the miniapps it will invoke | The generated `MiniappSpec`, rendered for non-technical review |
| **Validate** | A green-check "ready to deploy" with a fixtures-pass indicator | The deterministic validation (red/green from the framework) |
| **Deploy** | One button; progress shows Sprite provisioning | Fly MCP `create_sprite` |
| **Connect** | A picker of available connectors + policy preview | Sprites Connector API |
| **Approve** | An inbox of pending agent actions with evidence and approve/reject | The framework's approval gates |
| **Checkpoint** | A button and a timeline of checkpoints | Fly MCP `checkpoint_sprite` / `restore_sprite` |
| **Upgrade** | A "new version available" with diff | The framework's per-family versioning |
| **Talk** | A chat interface to the running agent | The generated agent's MCP endpoint |

That's nine primitives. Every generated agent uses them. The UI is small — a few screens — because the framework does the hard part.

---

## The 3-minute demo

This is the exact sequence the judges will watch.

**Beat 1 — The problem (15 sec).**
"Every personal agent demo you'll see today is one agent, hand-built. What if the agent itself was generated? What if you described what you wanted and it appeared?"

**Beat 2 — The describe (30 sec).**
"Here's what I want: an agent that watches my email for invoices, saves them to Box, matches each one to a purchase order, and reminds me to pay anything that's due but not paid. It should ask me before it sends a reminder."
*Types into the Studio.*

**Beat 3 — The spec (30 sec).**
The Studio shows a card: "I'll create 4 miniapps: `email.invoice.detect`, `box.invoice.save`, `po.match`, `reminder.send`. I'll need read access to email, write to Box, and approval before sending reminders. Here are 12 test fixtures I'll pass before deploying."
*The red/green indicator is green.*
"Validation is deterministic. The framework doesn't ship agents that don't pass their own fixtures."

**Beat 4 — The deploy (60 sec).**
*Clicks Deploy.* The Studio calls `mcp.fly.io` to create the Sprite, sets the connector policy to "Box only, this Sprite only," deploys the agent.
"Notice the connector policy. My Box credentials never enter the Sprite. The gateway attaches them. If the agent went rogue, it couldn't exfiltrate my Box token — it doesn't have one."

**Beat 5 — The use (45 sec).**
*The Studio opens the agent.* The agent has already processed three invoices. Two matched POs. One didn't.
*The approval inbox shows one pending reminder.*
"Here's what I need to approve. The agent shows me the invoice, the PO, and why the amount doesn't match. I can approve or reject. If I approve and the send times out, the framework won't double-send — it reconciles by reading the provider."

**Beat 6 — The checkpoint (30 sec).**
*Clicks Checkpoint.* "Before I upgrade the agent to handle a new invoice format, I checkpoint. If the upgrade breaks anything, I restore."
*Clicks Restore.* "Same agent, same memory, two seconds ago."

**Beat 7 — The close (15 sec).**
"That's one agent, in three minutes, on its own Sprite, with a connector, an approval gate, and a checkpoint. Every one of those properties is generic. Try the Studio — describe any agent you want and watch it appear."

**The 12 ideas are now demos of the Studio.** "Watch me generate a Chief of Staff in 60 seconds. Now a Weekend Ops. Now a Personal CFO." The Studio's demo can show one end to end and mention the others as gallery examples.

---

## What's already done vs. what's left

**Already done (the framework):**

- `MiniappSpec` schema and validation
- DBOS execution substrate with checkpointing
- Effect ledger with `PENDING → ATTEMPTING → CONFIRMED | FAILED | UNKNOWN` and reconciliation
- Approval gates with out-of-band channel
- Capability proxy (credential isolation)
- MCP surface (the generated agent talks MCP)
- Per-family versioning

**Left for the hackathon week:**

| Task | Effort | Notes |
|---|---|---|
| **Studio UI** — the nine primitives as screens | 1.5 days | This is the visual product. Highest-value work. |
| **Generator** — LLM call → `MiniappSpec` with type-safe schema, deterministic validation | 1 day | The framework already validates; the generator produces the spec. |
| **Fly MCP client** — create Sprite, set connector policy, checkpoint/restore | 1 day | `mcp.fly.io` is documented; OAuth flow. |
| **Sprite image** — pre-installed miniapp runtime + agent harness | 0.5 day | Build once, every generated agent uses it. |
| **Connector integration** — Box (or Pinecone) through the Sprites gateway | 1 day | Choose Box: the invoice demo is the strongest story. |
| **Approval inbox UI** | 0.5 day | Minimal: list of pending actions with evidence. |
| **Demo staging** — record the video, prepare realistic invoice data | 1 day | The video is the entry. |
| **Polish + submission** | 0.5 day | |

Total: ~7 days, tight but doable because the framework is the hard part.

**The single highest-risk item:** the Studio must feel like a product, not a config file. Every minute spent on the UI is a minute that improves the entry's chance of winning. The framework underneath should be invisible.

---

## Risks and mitigations

**Risk 1 — "This is just a prompt wrapper."**
The demo must show the generated artifact, not just the output. When the user describes an agent, the Studio shows the `MiniappSpec` with its miniapps, its capability dependencies, its fixtures, and its validation status. The judges must see that the platform is doing real work (deterministic validation, red/green tests, connector policy enforcement) that a prompt wrapper cannot.

**Risk 2 — The demo is too abstract.**
Don't demo the platform generically. Demo **one concrete agent** (invoice handling) end to end, then mention the gallery. Concrete + one beats generic + twelve.

**Risk 3 — Sprite provisioning is slow.**
First `create_sprite` can take 30+ seconds. In the video, cut. In the demo (if live), show progress. In the Studio UI, the progress bar shows "creating Sprite → installing runtime → validating → ready." The wait is a feature if it's well-wrapped.

**Risk 4 — The connector story is invisible.**
The judges might not understand why "credentials never touch the Sprite" matters. Show it: after deployment, a diagnostic screen shows "Box credentials: held by gateway, not by Sprite." A button "attempt to read my Box token from inside the Sprite" returns an error. That moment is the trust story.

**Risk 5 — The generator produces a broken spec.**
The framework's deterministic validation catches this. In the demo, deliberately show a rejected spec (e.g., "I tried to make it send emails without approval, and the Studio refused — that's the safety layer"). This demonstrates the red/green discipline working.

**Risk 6 — "Why does this need a Sprite?"**
The pitch answers this in one sentence: "Every user's agent gets its own machine, its own filesystem, its own connector policy. That's what makes it personal. A shared runtime would mean my agent learns about your data." That's the privacy story the Sprite uniquely tells.

---

## Submission checklist

- [ ] Studio deployed on Fly.io
- [ ] At least one agent generating → deploying → running end to end
- [ ] At least one Sprites connector used through the gateway (Box recommended for the invoice story)
- [ ] Demo video, 3 minutes, showing the seven beats
- [ ] Public post on X/Bluesky/LinkedIn/Reddit tagging @flydotio
- [ ] Link to code + link to demo submitted by midnight PST Nov 29
- [ ] Two sentences: "Miniapp Studio lets you describe a personal agent in plain language and deploy it to its own Sprite, connected to your SaaS tools without ever seeing your credentials. Every generated agent has approval gates, checkpoints, and durable execution — because the framework underneath was built for Class D workflows."

---

## The name

Pick something that hints at the meta-level. "Miniapp Studio" is descriptive. Better options: **Seed** (you describe it, it grows), **Sprout** (personal, small, isolated), **Hearth** (personal, warm, the agent that tends your fire), **Forge** (the factory metaphor), **Perch** (the Sprite is the perch the agent lives on). Or stick with Miniapp Studio — clarity beats cleverness for judges evaluating a new concept in three minutes.

---

## The single sentence to lead the submission with

**"We built the studio that generates personal agents — because the agent you want tomorrow shouldn't be hand-built today, and every generated agent deserves its own Sprite, its own credentials boundary, and its own approval gate."**

That sentence wins the grand prize, because it's the only entry where the framework is the star.
