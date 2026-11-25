# MiniApp Studio — Consolidated Hacksgiving Plan

**What this is:** a synthesis of the four platform reviews (`platform_chatgpt.md`, `platform_claude.md`, `platform_gemini.md`, `platform_z.md`) into one coherent entry, with the connector-extensibility requirement elevated to a first-class product surface and the durable-workflow claim scoped honestly.

---

## 1. Verdict and framing

**Build the factory, not the products.** The 12 ideas were never 12 products — structurally they are the same five primitives (*trigger → durable plan → consequential action → human gate → evidence*) with different connectors and copy. Anyone can build one of them this week; the entry is the generator of all of them.

**What the judges see:** a Studio where you describe an agent in plain language, it is generated as a **plan** (data, not code), deployed to its own Sprite, wired to a connector through the gateway, and run with approvals, receipts, and a checkpoint/rewind button.

**The one-sentence thesis:** *"A personal AI app factory on a Sprite: describe an agent, get a governed one. Credentials never touch the machine, every action is receipted, and you can always rewind."*

**The guarantee line (from `platform_claude.md`, the strongest framing across the four reviews):** *"Apps that can't do anything you haven't seen first, and that you can always undo or stop."* Approvals, undo, audit, and a kill switch are **platform properties**, not features of one app. That is the line no other entry in the room can say.

**What I changed from the prior files:** `platform_chatgpt.md` contributed the three-layer architecture and the six primitive families; `platform_claude.md` contributed the guarantee-first pitch, the off-Sprite effect ledger, and the rule-compliance discipline; `platform_gemini.md` contributed the three judge traps and the "one rock-solid miniapp beats a sprawling studio" scope discipline; `platform_z.md` contributed the plans-as-data interpreter (the key architectural move) and the demo script. The connector requirement below is new and is the reason the Studio is genuinely extensible rather than a fixed catalog of integrations.

---

## 2. What must not be lost

### The three judge traps (`platform_gemini.md`)

1. **The rules require a personal agent on a Sprite using at least one Sprites connector.** A static form-builder UI that happens to run on Fly infrastructure does not satisfy this. The demo must visibly show the Sprite executing the generation, the connector being used through the gateway, and the agent acting.
2. **"Generic generator" can become "prompt + wishful thinking."** Technical judges will ask what happens when the generator produces a broken plan or an unsafe action. The answer must be visible: schema validation, dry-run on fixtures, approval gates, receipts.
3. **Scope compression before midnight PST Nov 29.** One rock-solid generated agent running end-to-end on a real Sprite with a real connector beats a sprawling, half-wired studio.

### The rule-compliance discipline (`platform_claude.md`)

The framework predates the hackathon. That is fine, but it must be disclosed cleanly:

- **New repository** for everything the judges see: UI, generator, plan interpreter, connector adapters, templates, demo. Commits from Nov 23.
- **Framework as an external dependency**, pinned, credited by name and link in the README.
- **The submission states plainly:** *"Built this week: the Studio, the plan interpreter, the connector adapters, the templates, and the demo. Foundation: the miniapp framework, developed earlier, imported as a library."*
- **Email the organizers today** in two sentences asking whether an earlier open-source dependency is acceptable. Keep the reply. **Fallback:** if the answer is no or silent, rebuild the small kernel (ledger, approvals, audit, kill switch) fresh in the new repo. The platform design does not change.

### The pinned-framework discipline (`platform_claude.md`)

Freeze the framework at a pinned version. Known lifecycle defects are still in remediation; do not let remediation consume the hackathon week. **Claim only what a test on the pinned version proves.** A judge who double-clicks or retries can embarrass an overclaim; narrow, true claims beat big, shaky ones.

---

## 3. Architecture — three layers

```
┌──────────────────────────────────────────────────────────────┐
│  LAYER 1 — MINIAPP STUDIO (what judges see)                  │
│  Describe · Plan view · Connector catalog · Run timeline ·   │
│  Approval inbox · Control room · Rewind · Deploy to Sprite   │
├──────────────────────────────────────────────────────────────┤
│  LAYER 2 — PLAN COMPILER                                     │
│  LLM → Plan (JSON) → schema validation → dry-run on fixtures │
│  → connector-path derivation → permission derivation         │
├──────────────────────────────────────────────────────────────┤
│  LAYER 3 — RUNTIME (pinned miniapp framework)                │
│  One generic Class-D-shaped interpreter: each step is a      │
│  checkpointed step; effects go through the ledger; approvals │
│  park durably; receipts are recorded; Sprite checkpoint      │
│  provides disk-level rewind                                  │
└──────────────────────────────────────────────────────────────┘
            │
            ▼
   Sprites API gateway → Box / GitHub / Slack / OpenAI / Ollama /
   Custom API / …
```

**The critical architectural move (`platform_z.md`, adopted as the centerpiece):** plans are **data**, not generated code. One generic interpreter executes any plan. This is what makes the generator feasible in five days, keeps the framework's determinism guarantees intact, and gives every generated app the runtime's properties (resume, at-most-once, serialization, audit) for free.

**Why not codegen:** free-form generated code needs sandboxing, review, and conformance testing. A plan needs only a schema validator and a small interpreter. The generator's feedback loop is `write plan → validate → fix → repeat`. That loop is a demo, not a research project.

---

## 4. The connector catalog — the Studio's extensibility surface

This is the requirement that turns the Studio from "a builder for the integrations we shipped" into "a builder that treats any token-authenticated HTTP API as a first-class primitive." It is also the requirement most aligned with the hackathon rules, because *using a connector* becomes a visible, inspectable Studio operation rather than a hidden implementation detail.

### 4.1 How the gateway works (what the Studio must understand)

A connector has three pieces: **a credential** (OAuth token, your own API key, or managed), **an access policy** (deny-by-default; name prefix, labels, or allow-all; endpoint allow/block lists), and **a gateway endpoint**:

```
https://api.sprites.dev/v1/gateway/<provider>/<connection_id>/<path>
```

The Sprite calls that URL. **No `Authorization` header is sent** — the gateway identifies the calling Sprite from Fly.io's request signature, checks the policy, and forwards to the provider with the stored credential attached. The Sprite never holds the provider token; it only knows the connection ID and the gateway URL. Endpoint allow- and block-lists exist but are currently **API-only** (not dashboard-editable); block rules win over allow rules.

**Discovery endpoint.** The Sprite base image ships a `sprite-api-gateway` skill, and the gateway exposes `GET https://api.sprites.dev/v1/gateway/list`, which returns:

- Providers **connected** in the org, with `gateway_base_url` and `usage_snippet`.
- Providers **available but not set up**, with a `setup_url` to share with the admin.
- If a connector exists but is missing scopes, a `request_scopes_url`.

This is the endpoint the Studio's Connector Catalog calls to render the full picture without hard-coding provider names.

### 4.2 The provider catalog the Studio must display

The Studio's **Connector Catalog** screen renders whatever `GET /v1/gateway/list` returns for the org, grouped as:

| Group | Providers (per current docs; **availability varies by org during rollout**) |
|---|---|
| **Managed by Sprites** | OpenRouter (managed access), Sprites plan-provided credentials |
| **OAuth** | GitHub, Slack (User), Slack (Bot), Discourse, Anthropic (API), Meta Muse Spark (API key) |
| **Bring your own key** | OpenAI, TypeSafeAI (Jev), Ollama Cloud, Fireworks, Deepgram, Firecrawl, Steel Browser Tools |
| **Infrastructure** | Fly.io Machines, S3-compatible Object Store |
| **Custom** | **Custom API** — any token-authenticated HTTP API |
| **Payments** | Link Agent Payments (single-use card) |

Sources: the connector concept page and the ecosystem catalog list these providers with their auth styles and path restrictions.

**Honest note the Studio must show, not hide:** the docs state availability varies by organization during rollout. If Box or Pinecone is not present in the org's catalog, the Studio says so and offers Custom API or an alternative; it does not pretend the connector exists. The demo picks connectors from **what the org actually has**.

### 4.3 Custom API connectors — the extensibility primitive

Custom API is the fallback for anything without a dedicated connector: **base URL, token, where the token goes (header / query string / URL path), and an optional test request**. The upstream must be publicly reachable; private and loopback addresses are blocked. Bearer or custom headers, query parameters, and token-bearing URL paths are supported.

The Studio creates a Custom API connector by calling the Connectors API:

```
POST https://api.sprites.dev/v1/oauth/connections/api_key
{
  "provider": "custom",          // or "openrouter", "upstash", …
  "api_key": "<the token>",
  "base_api_url": "<publicly reachable base URL>",
  "access_policy": { "name_prefix": "studio-", "allowed_endpoints": ["/v1/*"] }
}
```

The response returns a **sanitized** connection (`conn_abc123`, provider, policy) — the token is never returned.

**What this unlocks in the Studio:** a user who has an internal REST API, a niche SaaS, or a legacy system can add it as a Connector in the Studio UI, and the generator can then use it as a primitive like any other. The plan declares the capability (`custom_api.invoke` with a declared path), the Studio's Permission Card shows the derived endpoint list, and the Sprite calls it through the gateway without the token.

**The Studio must therefore expose four connector operations as first-class UI actions:**

1. **List** — call `GET /v1/gateway/list`; render connected, available-not-set-up (with `setup_url`), and missing-scope connectors.
2. **Add Custom API** — a small form (base URL, auth method, token, test request); on submit, create via the Connectors API and set the access policy.
3. **Set access policy** — name-prefix / labels / allow-all, plus endpoint allow/block lists (marked "API-only" per the docs).
4. **Test** — hit the Gateway Playground equivalent or a Studio button that runs a request from the deployed Sprite and reports the result.

This is the difference between "we have three hard-coded integrations" and "the Studio treats the connector gateway as an extensible capability registry." It is also the requirement that keeps the Studio honest: the catalog screen shows what exists, and the Custom API form covers what does not.

### 4.4 What the Studio must not do

- **Not store or display raw tokens.** The gateway is the vault; the Studio is the console.
- **Not assume every org has every provider.** The catalog screen shows the org's actual state.
- **Not treat connector discovery and capability discovery as the same thing.** Fly connector discovery establishes which external services the Sprite can reach. The miniapp framework's capability registry establishes which business operations the generated app is actually allowed to perform. They are complementary, not interchangeable (`platform_chatgpt.md`, retained).

---

## 5. The primitives — what end users compose

Five families, each mapped to machinery the pinned framework already has (`platform_z.md`, consolidated with `platform_chatgpt.md`'s six-family view):

| Family | Primitives | Underneath |
|---|---|---|
| **Triggers** | Manual, cron, webhook, connector event | River periodic jobs; canonical invoke path |
| **Data** | Entity, field, filter, aggregate, list/detail binding | Capability reads through the gateway; transform steps |
| **Actions** | Connector call (any catalog provider or Custom API), LLM step, notify | Effect ledger — every action is a receipted effect with retry/backoff |
| **Control** | Approval gate, durable wait, branch, entity serialization, deadline | Approval parking with intent-hash binding; durable step boundary |
| **Policy & Evidence** | Rate limit, blast radius, kill switch, receipt timeline, audit view, UNKNOWN inbox | Manifest limits; effect ledger; audit; quarantine→human |

**UI primitives** (the generic renderer): describe bar, form, list/table, detail card, approval card, run timeline, status/progress, permissions card, notifications.

**The Studio renders the UI from the manifest's JSON Schema per primitive** — forms are schema-driven, not hand-built per template. The generated app is a plan; the UI is a renderer over the plan.

---

## 6. The plan format — data, not code

A plan is JSON. Named entity, trigger, declared limits, typed steps, simple `${...}` references between steps.

```json
{
  "name": "friday-box-digest",
  "entity": "folder",
  "trigger": { "type": "cron", "expr": "0 17 * * 5" },
  "limits": { "maxRunsPerDay": 1, "maxPostsPerRun": 1 },
  "onUnknown": "ask",
  "steps": [
    { "use": "connector.box.list_folder", "with": { "folder": "Weekly Reports" }, "as": "items", "effect": "read" },
    { "use": "llm.summarize", "with": { "input": "${items}" }, "output": "untrusted", "as": "summary" },
    { "use": "approve", "with": { "show": "${summary.text}", "expires": "24h" } },
    { "use": "connector.slack.post_message",
      "with": { "channel": "#team", "text": "${summary.text}" },
      "effect": { "class": "reversible", "undo": "connector.slack.delete_message" },
      "idempotency": "friday-digest:${run.week}" },
    { "receipt": "posted digest for week ${run.week}" }
  ]
}
```

The Studio derives from this plan, before deployment:

- **The connector paths needed** (Box list, Slack post) → shown in the **Permissions Card**.
- **The effects and their reversibility class** → drives approval requirements and the Undo button.
- **The limits** → enforced by the runtime (a small token-bucket table; on the critical path the moment limits appear in the UI).
- **The approval requirement** → the generated app cannot execute the consequential step without one.

**The interpreter stays tiny.** Three operators (`if`, `${...}`, `and/or`) and no more. Every step is a checkpointed unit. The framework's determinism lint stays green because the workflow body is static Go and the plan is data.

---

## 7. The Studio UI — four screens

Deliberately not a drag-and-drop canvas, not a marketplace, not multi-tenant SaaS polish. That is post-hackathon.

| Screen | Purpose |
|---|---|
| **Factory / Gallery** | Pick a template or start from primitives; edit the plan as schema-driven forms with a JSON view; **Connector Catalog** panel; **Deploy → Sprite** button showing placement |
| **Connector Catalog** | The full provider list from §4.2; connected / available / missing-scope states; **Add Custom API** form; access-policy editor; test button |
| **Run timeline** | Per run: steps, durations, receipts with evidence refs, approval moments, waits, and the **Rewind** control |
| **Approval inbox + Control room** | Payload preview, approve/deny/resolve, expiry countdown; kill switches, declared limits, the UNKNOWN/exception inbox |

Phone-friendly, single-page, over the existing runtime verbs (`invoke | status | signal | cancel | decide`). No visual workflow editor in the hackathon build — the plan view *is* the editor.

---

## 8. The demo script — three minutes, agent-first

Every platform feature is shown **through a generated agent doing something visibly consequential**. No infrastructure screens.

| Time | Beat | What it proves |
|---|---|---|
| 0:00–0:10 | Hook: "Fly made it safe to give an agent the keys. We make it safe to let it act, and to take it back." | Thesis |
| 0:10–0:40 | **Generate:** type the Box-digest prompt. The agent on the Sprite writes a plan; the **Permissions Card** shows the connector paths, effects, and limits before deploy | Generation, least privilege, connectors without keys |
| 0:40–0:55 | **Deploy:** the Studio provisions a Sprite, sets the connector policy, deploys | Fly MCP, connector gateway |
| 0:55–1:25 | **Run + approve:** the app runs; the approval card lands on the phone; approve; the receipt appears with the before/after | Approvals, receipts, wake-on-request |
| 1:25–1:45 | **Rewind:** checkpoint before a change, restore, and the plan resumes from its checkpointed step; the ledger proves **nothing double-fired** | Checkpoint + effect ledger |
| 1:45–2:20 | **Breadth:** generate a second, different agent (GitHub triage) live from primitives in under a minute; deploy to a fresh Sprite | Generality — proven by variety, not feature count |
| 2:20–2:40 | **Connector extensibility:** open the Connector Catalog, show the full provider list, add a **Custom API** connector live, and use it in a third agent | The Studio's extensibility claim, demonstrated |
| 2:40–3:00 | **Governance close:** ask the second agent to do something outside its declared scope — denied, receipted. Flip a kill switch. End card: *"Don't build 12 agents. Describe the agent. Get a governed one on a Sprite. Receipts all the way down."* | Guardrails |

The 12 ideas survive as **templates in the gallery** (Fleet Janitor, Renewal Watchdog, Thanksgiving Coordinator, and the rest), not as separate builds. Breadth is shown by generating two different agents live, not by shipping twelve.

---

## 9. Durable workflows — what ships, what is roadmap

This is the delineation the user asked for. The Studio's pitch depends on being honest about which durability properties are **demonstrated** and which are **designed**.

### Ships in the hackathon demo (tested on the pinned framework)

| Property | What the demo shows |
|---|---|
| **Checkpointed execution** | Each plan step is a checkpointed unit; a Sprite kill mid-run resumes from the last completed step |
| **Disk rewind** | Sprite checkpoint (~1s filesystem snapshot) + restore; the plan resumes and the ledger proves no double-fire |
| **Effect receipts** | Every consequential step writes a ledger record before and after; the timeline shows them |
| **At-most-once confirmed effects** | Idempotency key per effect; a retry that would duplicate is refused |
| **UNKNOWN quarantine** | A timed-out send is held for human reconciliation, not blindly retried |
| **Approval parking** | Approval is a durable step; the run parks and resumes on decision or expiry |
| **Kill switch** | Every generated app can be stopped instantly |

### Designed, not demoed (roadmap; the vision slide names it, the demo does not claim it)

| Property | Why it is roadmap |
|---|---|
| **Multi-day durable waits** | The framework's DBOS-backed long-wait path is in remediation; the demo uses short waits and checkpoint-based recovery |
| **Per-family versioning with drain** | Needs the DBOS family infrastructure; not needed for a one-afternoon agent |
| **Retry-as-re-entry across state machines** | The framework's state machine is pinned but not yet production-hardened |
| **Sweeper-based lost-signal recovery** | The framework's approval sweeper is in Phase 2 |
| **Cross-replica grant coherence** | Deployment property, not a single-Sprite concern |

**The honest formulation for the vision slide:** *"Every generated agent gets checkpointed execution, effect receipts, approval gates, and a rewind button today. The same runtime is being hardened for multi-day durable workflows — the roadmap is on the repo."* That sentence is true, defensible, and does not depend on a property the judges can falsify in the demo.

**Why this delineation matters beyond the hackathon:** the durable-workflow claim is the long-term differentiator for the business (co-ops, MFIs, SMEs whose workflows genuinely span days). Overclaiming it at the hackathon would be a credibility debt paid at the first customer. Underclaiming it would be leaving the strongest story on the table. The middle path — *"checkpointed execution and receipts today, multi-day durability on the roadmap"* — is what `platform_claude.md` meant by "claim only what tests prove," applied to the product pitch as well as the code.

---

## 10. Scope and five-day plan

**Must have (the golden path):** describe bar, generator with validation, one interpreter (connector call, LLM step, approve, wait, notify, branch), approvals page, **Connector Catalog + Custom API form**, permissions card, run timeline with Rewind, off-Sprite ledger, three generated agents, a small attack scoreboard.

**Cut list:** visual workflow editor, accounts and multi-user, marketplace, billing, dozens of hard-coded connectors, custom domain, theming.

| Day | Goal | Done when |
|---|---|---|
| **Mon 23** | Organizer email; new repo; Sprite up; `GET /v1/gateway/list` returns the org's actual catalog; portability spike; plan schema v0 | A Sprite calls a connector with no token inside; framework imports on the Sprite or the fallback kernel is chosen |
| **Tue 24** | Interpreter v0: connector call, approve, wait, effect ledger; `validate` command | A hand-written plan runs end to end with an approval and a receipt |
| **Wed 25** | Generator loop: coding agent writes a plan, validator feeds back, dry-run on fixtures; **Connector Catalog screen + Custom API form** | "Describe it" produces a valid plan for the Box digest; a Custom API connector is created and used |
| **Thu 26** | UI shell: gallery, app view, approvals, timeline; phone-friendly; Sprite wake test | Approve from a phone while the Sprite sleeps |
| **Fri 27** | Rewind and compensations; the second and third agents; attack scoreboard | The golden path runs three times with three different agents |
| **Sat 28** | Record the video; README with the built-this-week disclosure; post publicly tagging `@flydotio`; **submit** | Submission sent Saturday night |
| **Sun 29** | Buffer only | No new work |

**Deadline note:** midnight PST Nov 29 is roughly 4:00 PM Monday Nov 30 Philippine time. Treat Saturday as the deadline.

---

## 11. Risks and mitigations

| Risk | Mitigation |
|---|---|
| "Built during the week" challenged | §2: separate repo, dependency disclosure, organizer email, fallback kernel |
| Platform scope overruns | Golden path only; cut list; three agents not twelve |
| Connector availability varies by org | `GET /v1/gateway/list` on day one; design around what exists; Custom API as the fallback proof |
| Custom API upstream blocked (private/loopback) | Use a publicly reachable test endpoint; the docs state the restriction explicitly |
| Framework portability to a Sprite fails | Day-one spike; lean-kernel fallback |
| Overclaiming durability | §9 delineation; every demo claim is a tested property on the pinned version |
| Generated plans are low quality | Strict schema, validator, fixtures, retry inside a checkpoint |
| Demo flakiness around Sprite sleep/wake | Rehearse; keep a Service listening; record a clean take |
| Pitch sounds like every app builder | Lead with the guarantee line, show undo and the attack, not "describe an app" |
| Endpoint allow/block lists not editable in dashboard | The Studio sets them via the Connectors API and labels them "API-only" in the UI |

---

## 12. Decisions to make today

1. **Rule-compliance route:** send the organizer email now and use the dependency-plus-disclosure approach, or rebuild a small kernel fresh?
2. **Off-Sprite ledger location:** an S3-compatible Object Store connector, a Custom API connector to an external database, or Box as the store. (The S3 connector exists in the catalog.)
3. **Which three agents** to generate, given the connectors the org actually has (check the dashboard before choosing templates).
4. **Custom API demo target:** which publicly reachable token-authenticated API to use as the live extensibility proof.
5. **Freeze framework remediation** until after Nov 29 (recommended: yes, pinned version only).

---

## 13. The consolidated thesis, in one paragraph

**The Studio turns a description into a governed agent, deployed to its own Sprite, connected through the Sprites gateway to any provider the org has enabled — including any token-authenticated HTTP API via Custom API — with approval gates, effect receipts, checkpoint/rewind, and a kill switch as platform properties rather than features of any one app.** The framework underneath provides the execution, ledger, approval, and audit primitives; the Studio is the visible product; the generated agents are the output. Durable multi-day workflows are the roadmap differentiator, named honestly and not demoed until tested. The 12 ideas are templates in the gallery, and the demo shows two of them generated live plus a third wired to a Custom API connector — because generality is proven by variety, not by feature count.
