# The Agent Factory — building the platform, not the 12 apps

**Question on the table:** instead of submitting one of the 12 hackathon agents, build the *generic platform* on top of miniapp — primitives, UI, and workflow so end users generate miniapp-driven agents without building 12 different apps (the GigaCatalyst pattern applied to agents). The framework predates the hackathon and stays the foundation; what judges see is the UI platform on top.
**Verdict up front:** **Yes — this is the right pivot, and it is the honest expression of miniapp's thesis. One big caveat about how platforms lose hackathons, and a concrete design that avoids it.** (I'm working from your framing of GigaCatalyst as your template-generation platform; the design below is generic to that pattern.)

---

## 1. Why the pivot is right — and the one thing that can kill it

**Right, because the 12 ideas were never 12 products.** Look at them structurally: every one is the same five primitives — *trigger → durable plan → consequential action (effect + receipt) → human gate (approval) → evidence (ledger/audit)* — with different connectors and copy. The Fleet Janitor is `cron + fly-tool + approval + restart`; the Renewal Watchdog is `cron + document-read + approval + renewal`; the Enforcer is `durable-wait + evidence-check + escalation`. Anyone who builds one of them this week ships an agent; you ship the *generator* of all of them. That's the difference between an entry and a platform — and it's exactly the "don't build 12 apps" pitch to your real target market after the hackathon.

**Right, because the rules still fit.** Hacksgiving requires "an agent that runs on a Sprite using a connector." The platform satisfies this if — and only if — the demo remains **agent-first**: the factory UI creates a described agent, the generated agent runs on its own Sprite, uses connectors, acts, gets approved, files receipts. The platform is the wrapper; the generated agent is the star of every scene.

**The one thing that can kill it:** platforms demo badly when the demo shows infrastructure instead of magic. "Here's my schema editor" loses to "here's my agent restarting a crashed machine because I approved it from my phone." The mitigation is baked into the demo script (§6): every platform feature must be shown *through a generated agent doing something visibly consequential*. Two agents generated live in three minutes, receipts all the way down — that's the whole video.

---

## 2. What `connectors.md` changes — two gifts and one unlock

1. **Fly MCP (`mcp.fly.io`) is an official connector** with OAuth sign-in and scoped permissions (delete apps, read secrets metadata, run commands on Fly Machines). This means the highest-value template — the Fleet Janitor — becomes a *configuration exercise*: the generated agent speaks MCP to Fly, and miniapp wraps every tool call as a governed effect (approval gate → ledger → receipt). You don't build Fly integration; you govern it.
2. **The Sprites API gateway keeps provider credentials off the Sprite** (configure a connector in the dashboard, grant per-Sprite access by prefix/labels, gateway attaches the credential at call time). This solves, at the platform level, the single hardest gap in the RPA/agent analysis (the credentials vault, G6): *the generated agent never holds your keys.* Your security story becomes two sentences: "credentials never touch the agent's machine, and every action is approval-gated and receipted." The gateway is the vault; miniapp is the conscience.
3. **The unlock:** Sprites' checkpoint/restore (full-filesystem snapshot in ~1s) composes with miniapp's durable execution for a demo flourish nobody else will have — kill the Sprite mid-task, restore the checkpoint, watch the *plan* resume from its checkpointed step and the filesystem resume from the snapshot. State-level and disk-level recovery in one demo.

Also in the file, worth using: the gateway exposes OpenAI-compatible and Anthropic-compatible chat endpoints with model discovery — so an **LLM step** is just another connector primitive (and with Meta/Muse Spark, the model call carries no key inside the Sprite).

---

## 3. The generic primitives — the catalog the platform ships

This is the heart of "end users generate miniapp-driven apps." Five families, each mapping to machinery miniapp **already has**:

| Family | Primitives | miniapp machinery underneath |
|---|---|---|
| **Triggers** | manual (UI button) · cron · webhook · connector event (file added, inbox) · threshold | River periodic jobs; the canonical `Invoke` path — no side doors |
| **Actions** | connector call (Fly MCP tool, Box file op, gateway LLM step, Pinecone query/upsert) · notify (email/webhook) | `effects.Channel` — every action is a receipted effect with retry/backoff declared per step |
| **Control** | approval gate (who, expiry, payload preview) · durable wait · condition/branch · entity serialization · deadline | `parkApproval` + intent-hash binding; `WaitUntil`/signal; `entity_key` + advisory lock |
| **Policy** | rate limit per target · blast-radius declaration · kill switch · revocation | manifest limits + kill-switch path (enforcement of limits is on your build list) |
| **Evidence** | receipt timeline · step evidence (screenshots/files as refs) · audit view · UNKNOWN inbox | effect ledger + `Receipt.ProviderReply` refs + audit + quarantine→human |

End users never see Go. They see primitives, forms, and a timeline. You never write 12 apps. You write **one interpreter and N JSON plans**.

---

## 4. The architecture that makes it work in five days: plans as data, one interpreter

The hackathon-scale way to "generate miniapps" is **not codegen** — it's a **data-driven plan interpreter**:

- A **plan** is JSON: named entity, trigger, declared limits, and a list of typed steps (the primitives above), with simple `as:`/`${...}` references between steps.
- One **generic Class D workflow** (the interpreter) executes any plan: each step runs as a checkpointed `RunAsStep`; connector steps go through the effects ledger; approval steps park durably and resume on `decide`; waits are durable; UNKNOWN outcomes route to the exception inbox.
- The **determinism lint stays green by construction** — the workflow body is static Go; plans are data. This is the quiet elegance of the fit: the framework's strictest rule is precisely what forces the generator architecture, and the generator inherits every runtime property (resume, at-most-once, serialization, audit) without a line of per-app code.
- **UI is generated from the manifest, which already carries JSON Schema** per tool — forms render from schemas, not hand-built. The whole operator surface is a thin layer over verbs the runtime already exposes: `invoke | status | signal | cancel | decide` (`cmd/miniapp/main.go:68`) plus the approval park/decide endpoints and `Signal{Payload}` for corrections.

```json
{
  "name": "fleet-janitor",
  "entity": "app",
  "trigger": { "type": "cron", "expr": "*/5 * * * *" },
  "limits": { "maxRunsPerHour": 12 },
  "onUnknown": "ask",
  "steps": [
    { "use": "fly.tool", "tool": "list_machines", "as": "apps" },
    { "use": "fly.tool", "tool": "machine_status", "args": { "id": "${apps.unhealthy[0]}" }, "as": "status" },
    { "if": "status.restarts > 3" },
    { "approval": "Restart ${apps.unhealthy[0]}?", "expires": "30m" },
    { "use": "fly.tool", "tool": "restart_machine", "args": { "id": "${apps.unhealthy[0]}" } },
    { "receipt": "restarted ${apps.unhealthy[0]}" }
  ]
}
```

A template is just a plan with a friendly name and pre-filled primitives. The Fleet Janitor, Renewal Watchdog, and Enforcer from the 12-ideas doc become **three shipped templates** — breadth without bespoke code.

---

## 5. What the judges see (the UI platform)

Four screens, phone-friendly, single-page, over the existing REST verbs:

1. **Factory / gallery** — pick a template or start from primitives; edit the plan as forms (schema-driven) with a JSON view for power users; one button: **Deploy → Sprite**. Show placement (this Sprite, fresh disposable Sprite per run) — the placement seam your framework already froze.
2. **Run timeline** — per run: steps, durations, receipts with evidence refs, approval moments, waits. This is the receipt culture made visible.
3. **Approval inbox** — payload preview, approve/deny/resolve-with-correction, expiry countdown. The phone moment of the demo.
4. **Control room** — kill switches, declared limits, the UNKNOWN/exception inbox. Governance as a product surface, not a hidden feature.

That's the platform. It is deliberately *not* a drag-and-drop editor, not a marketplace, not multi-tenant SaaS polish — those are post-hackathon roadmap items (and the natural continuation of the GigaCatalyst line into the business plan's verticals).

---

## 6. The demo script (three minutes, agent-first)

1. **(0:00–0:40) Create.** Open the factory, pick the **Fleet Janitor** template, show the plan as editable primitives (five families — name them out loud: triggers, actions, gates, limits, receipts), hit Deploy. Agent comes up on its own Sprite.
2. **(0:40–1:20) Consequence.** Scripted incident: a demo app starts crash-looping. The agent notices, files its diagnosis, and an **approval card lands on the phone**: "Restart crash-looping machine? expires 30m." Approve. Machine restarts. Receipt appears in the timeline with the before/after status. Real infrastructure, visibly moved.
3. **(1:20–2:00) Resilience.** Kill the agent's Sprite mid-run (`fly machine kill`). Restore from checkpoint / let it come back. The plan resumes from its checkpointed step — show the ledger proving **nothing double-fired**: one approval, one restart, one receipt. "Agents fail. Mine files receipts."
4. **(2:00–2:40) Breadth.** Back in the factory: build a **second, different agent** (Renewal Watchdog: `cron + Box document + LLM extract + approval + receipt`) from primitives, live, in under a minute. Deploy to a fresh disposable Sprite. It runs and files a receipt. Same platform, different agent — the generator claim demonstrated, not asserted.
5. **(2:40–3:00) Governance close.** Ask the second agent to do something outside its declared scope: denied, receipted, visible in the control room. Flip a kill switch. End card: *"Don't build 12 agents. Describe the agent. Get a governed one on a Sprite. Receipts all the way down."*

---

## 7. Five-day build plan (today = Nov 24; deadline Nov 29 midnight PST)

| Day | Ship |
|---|---|
| **1** | Plan schema + generic interpreter workflow (Class D): steps as `RunAsStep`, approval parking, durable waits, UNKNOWN routing. Unit tests with the `mockcap` scripted-failure pattern. |
| **2** | Connector adapters as `effects.Channel` implementations: Fly MCP client (governed tool calls), one gateway LLM step (OpenAI-compatible via the Sprites gateway), Box file op, Pinecone upsert/query. End-to-end: plan → effect → receipt. |
| **3** | The four UI screens over existing REST verbs; schema-driven forms from manifests; deploy-to-Sprite button (placement seam → `sprites-go`). |
| **4** | Three templates (Fleet Janitor, Renewal Watchdog, Enforcer-lite), demo app + scripted incidents, checkpoint/restore flourish, limits enforcement for the `maxRunsPerHour` you declare (small token-bucket table — it's on the critical path the moment you show limits in the UI). |
| **5** | Video, submission copy ("built during Hacksgiving week: the factory UI, plan interpreter, connector adapters, and templates — on top of my pre-existing open-source miniapp framework as the reliability layer"), buffer for the inevitable Sprites rollout surprises (connector availability "varies by organization"). |

**Honest risk list:** the Fly MCP permission scopes must be requested narrowly (least-privilege is itself part of the pitch); connector availability varies per org — have the LLM/Box adapters as the fallback "connector" proof; the interpreter's expression language (`if`, `${...}`) must stay tiny — three operators, no more, or Day 4 evaporates; and if Fly MCP integration stalls, the Fleet Janitor template ships with a simulated-Fly adapter so the *platform* story never depends on one connector.

---

## 8. Why this wins, stated plainly

1. **It's the only entry that answers "build a personal agent" with "here's the machine that builds them."** Same requirement, higher altitude — and every generated instance still satisfies it literally: an agent, on a Sprite, with connectors.
2. **The judges' own products are co-stars.** Sprites are the runtime, the gateway is the vault, Fly MCP is the highest-value connector — and the reliability spine that makes the demo's kill-and-resume moment possible is yours, pre-existing, and MIT-licensed for anyone to inspect after the demo.
3. **The post-hackathon story is intact.** The factory *is* the seed of the platform line for the target market (coops, MFIs, SMEs describing apps instead of hiring for them) — the hackathon entry and the GigaCatalyst-pattern business converge on the same artifact, which is the best possible outcome of a week of building: one demo, two audiences, zero throwaway work.
