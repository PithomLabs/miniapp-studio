# MiniApp Studio — Consolidated Platform Plan

**What this document is:** the single source of truth merging the best of all four platform reviews (`platform_chatgpt.md`, `platform_claude.md`, `platform_deepseek.md`, `platform_gemini.md`), plus two new hard requirements from you — **(1) Studio is extensible via custom API connectors** per `docs.fly.io/sprites/api/connectors`, and **(2) Studio displays the full supported connector catalog** per the Sprites ecosystem page — plus the **claims discipline that keeps durable workflow a roadmap differentiator, not an overclaimed demo**.
**Grounding:** the Connectors API was verified against the live OpenAPI spec (`docs.fly.io/sprites/api/openapi.json`) and the ecosystem page was scraped today (2026-11-24). Everything quoted below is from those sources or from code previously read in the miniapp framework.

---

## 1. Product thesis (merged from all four specs)

> **MiniApp Studio — describe a personal agent in plain language; get a working, governed app on its own Sprite, wired to your connectors without ever seeing your credentials. Apps that can't do anything you haven't seen first — and that you can always inspect, stop, and rewind.**

- From **chatgpt**: the platform generates *specifications, not code*; three-layer architecture; six primitive families; "don't build software for every business problem — build the platform that lets every business build its own software."
- From **claude**: pitch the **guarantee** (ask first, undo, audit, stop) rather than the builder — the one line no other entry can say; one golden path proven with three different apps; the off-Sprite ledger; harness-agnostic approvals filling the documented gap (terminal backends such as Hermes provide no Sprites-specific confirmation hooks).
- From **deepseek**: all five hackathon partners used *structurally* (Box as the connector, Pinecone as memory, Ollama Cloud as the model choice, Meta Muse Spark as the generator's LLM, TypeSafeAI as the type-safe spec schema); the nine Studio primitives; the seven-beat demo; the "attempt to read my Box token from inside the Sprite → error" trust moment.
- From **gemini**: the three judge-traps — the entry must visibly be *a personal agent on a Sprite using a connector* (not a static CRUD UI); "generic generator" dies as "prompt + wishful thinking" unless validation is visible; one rock-solid end-to-end run beats a sprawling half-wired studio.

**Division of responsibility (chatgpt's closing, kept verbatim in spirit):** Fly.io Sprites give the builder agent and each generated app their own isolated, persistent machine · the miniapp framework provides trusted execution, approvals, effects, audit · MiniApp Studio gives ordinary users the surface to create apps without programming.

---

## 2. Architecture (three layers + the generated agent)

```
                    You (browser / phone)
                            │
┌───────────────────────────┴────────────────────────────────────────┐
│ L1 · MINIAPP STUDIO  (what judges see)                             │
│  Describe bar · App gallery · Spec card · Connector picker ·       │
│  Approvals inbox · Run timeline + receipts · Checkpoint/Rewind ·   │
│  Control room (limits, kill switch, UNKNOWN inbox)                 │
├────────────────────────────────────────────────────────────────────┤
│ L2 · APP COMPILER / SPEC RUNNER  (the bridge)                      │
│  Generator agent (on its own Sprite) → MiniAppSpec (type-safe)     │
│  → deterministic validator + fixtures dry-run                      │
│  → ONE generic Spec Runner miniapp (registered once) interprets    │
│    specs as data: every step a checkpointed RunAsStep              │
│  → capability proxy resolves connector calls                       │
├────────────────────────────────────────────────────────────────────┤
│ L3 · MINIAPP FRAMEWORK  (the foundation, pre-existing, pinned)     │
│  Auth/RLS tenancy · approvals (intent-hash bound) · effect ledger  │
│  (receipts, UNKNOWN quarantine) · audit · kill switches · MCP      │
│  transport · versioned manifests · deterministic lint              │
└───────────────┬───────────────────────────────────┬────────────────┘
                │ Fly MCP (mcp.fly.io, OAuth)        │ Sprites gateway
                │ provisioning: create/checkpoint/   │ connector calls —
                │ restore; connector policy          │ credentials NEVER on the Sprite
        ┌───────▼────────┐                  ┌────────▼─────────────────┐
        │ Generated agent │                  │ Connectors: Box, GitHub, │
        │ on its own Sprite│                 │ Pinecone, Slack, LLMs,   │
        └─────────────────┘                  │ Custom API (yours), …    │
                                             └──────────────────────────┘
```

Key decisions kept from the specs (claude §5.1, chatgpt §4, deepseek §"already done"):

| Decision | Why |
|---|---|
| **Specs, not programs** — the generator emits a declarative MiniAppSpec; a fixed runtime executes it | Safe, inspectable, diffable, undoable; validator feedback loop instead of code review; no arbitrary AI code in the trusted runtime |
| **One generic Spec Runner, registered once** | The framework registers miniapps as compiled code; a data-driven interpreter lets users create apps at run time without compiling — each app is data, not a binary |
| **Specs immutable + hashed** | Runs pin their spec hash; replays and recovery see the same definition; changes create a new version |
| **Fly discovery ≠ business authority** (chatgpt §6) | Connector discovery says what the *infrastructure* can reach; the manifest says what the *app is allowed to do* (approvals, limits, entity keys). Studio composes both, and says so explicitly |
| **Off-Sprite effect ledger** (claude §5.1) | Checkpoint restore rewinds the disk; the record of what was done to the world must live where restore can't touch it (S3-compatible store via connector, or external DB) |
| **Approvals outside the agent's control** | The approve endpoint must be unreachable from the agent process (separate service identity/token) — in a one-Sprite product this is the difference between governance and theater |

---

## 3. The primitive catalog (what end users receive)

Merged: chatgpt's six families + claude's UI/workflow/safety tables + deepseek's nine Studio primitives. The families are platform capabilities, not separate products.

| Family | Primitives |
|---|---|
| **Data** | Entities/collections, fields, filters, aggregations, bindings; one working connected data source at Hacksgiving (chatgpt's vertical-slice rule) |
| **UI** | Describe bar · form · list/table · detail card · approval card · run timeline (with receipts) · status/progress · permissions card · notifications · themes/branding (post-hackathon) |
| **Workflow** | Trigger (manual / schedule / webhook / connector event) · connector call · LLM step (output marked untrusted) · transform · branch (bounded) · durable wait · approve · notify |
| **Integration** | Connector catalog browser · policy preview · capability discovery · **custom API connector wizard** · webhooks · auth references |
| **Governance** | Roles/permissions · approval requirements · effect declaration with reversibility class (reversible / compensatable / irreversible) · idempotency keys per effect · limits (per run / per hour / recipient allow-lists) · kill switch · append-only audit · egress view |
| **Evidence & recovery** | Receipt timeline · checkpoint-before-run · Rewind (disk undo + world reconciliation; irreversible effects flagged "cannot unsend") · UNKNOWN/exception inbox |

deepseek's nine screens are the L1 rendering of these: **Describe → Spec → Validate → Deploy → Connect → Approve → Checkpoint → Upgrade → Talk.**

---

## 4. The connector system — catalog display + custom API extensibility (verified)

This is the section your two requirements live in, grounded in the actual Sprites API.

### 4.1 The catalog the Studio displays

The Studio's **Connector picker** shows the ecosystem catalog (scraped 2026-11-24 from `fly.io/sprites/ecosystem`; the page itself notes availability varies by organization during rollout — the Studio must show **your org's live state**, not the marketing list, by calling `GET /v1/oauth/connections` at `api.sprites.dev/v1`):

| Group | Connectors |
|---|---|
| Models / AI | Meta (Muse Spark), OpenAI, Anthropic, OpenRouter, Ollama Cloud, Fireworks, Parasail, Hugging Face, Deepgram, ElevenLabs, TypeSafeAI |
| Knowledge / memory | Pinecone ("query and manage your Pinecone vectors and run hosted inference"), Upstash Redis, Mnemom |
| Files & SaaS | Box, GitHub, Slack (User), Slack (Bot), Discourse, AgentMail, Resend, Usable, Blotato |
| Infra & data | Fly.io Machines, Neon, Supabase Database, Tigris, S3-compatible Object Store, Depot build insights |
| Automation & tools | Steel Browser Tools, Firecrawl, Composio Tools, SenseLab, Keenable, LemonSlice, Era, Link Agent Payments |
| **Bring your own** | **Custom API** — the extensibility path below |

For each connector the Studio shows: name, group, what it can do (from the ecosystem descriptions), its **access policy state** (which Sprites may use it — `allow_all`, `name_prefix`, `sprite_labels`), and its endpoint rules. Demo beat: open the picker, show Box granted to exactly one Sprite, everything else default-deny.

### 4.2 Custom API connectors — the extensibility requirement, grounded

This is what makes Studio *open* rather than a walled catalog: any REST API a business already runs (HubSpot, GoHighLevel, a coop's core system, a government portal) becomes a governed connector **without keys ever living on a Sprite**. Verified endpoint and schema facts from the OpenAPI spec:

- **Create:** `POST /v1/oauth/connections/custom_api` — body fields: `name`, `description`, `base_api_url`, `test_url`, `auth_method` (enum: `header` | `url_path` | `query_param` | `custom_header`), plus the per-method auth fields (`auth_custom_header_name`, `auth_header_prefix`, `auth_query_param_name`, `auth_url_path_template`), `access_token`, `refresh_token`, and an `access_policy`.
- **Test:** `POST /v1/oauth/connections/custom_api/test` — same fields plus `connection_id`; the Studio runs this as its wizard's "green check" before offering Deploy.
- **Update:** `PUT /v1/oauth/connections/custom_api/{id}`. **List:** `GET /v1/oauth/connections`. **Provision:** `POST /v1/oauth/connections/provision`. Managed connectors also expose `/balance` and `/usage`; OAuth-type connectors authorize via `GET /v1/oauth/{provider}/authorize` → `POST /v1/oauth/{provider}/callback`.
- **AccessPolicy** (the least-privilege contract): `allow_all` (default-deny when false), `allowed_endpoints[]` (exact paths and trailing-`*` wildcards, e.g. `/chat.*`), `blocked_endpoints[]` (case-insensitive, **checked before allow rules**), `name_prefix` (calling Sprite must match), `sprite_labels[]` (all listed labels required). Note: database connectors reject endpoint rules — the Studio form must hide endpoint rules for those types.

**Studio UX — the "Add connector" wizard (three steps):**
1. **Define** — name, base API URL, test URL, auth method (one of the four, with the matching field revealed), paste the token. The credential goes straight into the gateway; it is never written to a Sprite, a manifest, or a spec.
2. **Scope** — endpoint allow/block rules with wildcard hints ("the generated app may call `/v1/invoices*`, never `/admin*`") + per-Sprite policy (name prefix / labels; leave `allow_all` off).
3. **Prove & grant** — the test call runs green; the permissions card shows exactly which paths the app can reach; grant to the app's Sprite. The connector now appears in the catalog as first-class and is callable from any generated app's spec (`use: connector.my_api.list_invoices`).

Strategic value (the line for the post-hackathon pitch): the catalog is Fly's, the **custom connector is the user's**, and the *authority* layer — approvals, limits, entity serialization, receipts — is miniapp's. Three parties, three trust layers, no overlap.

---

## 5. What ships at Hacksgiving vs. the roadmap — the overclaiming fence

**The rule (claude §11, kept): claim only what is tested on the pinned version; narrow, true claims beat big, shaky ones.** Durable workflow is the framework's real, tested foundation — but the *Studio surfaces for authoring and observing durable workflows* are roadmap. The demo must show receipts, not promise an editor.

### Ships at Hacksgiving (the golden path, gemini's scope-compression rule)

| Capability | Scope |
|---|---|
| Generator | Describe bar → MiniAppSpec → deterministic validation with a visible red/green fixtures indicator; a deliberately rejected spec shown live (safety layer refuses an unapproved send) |
| Spec Runner | One generic interpreter miniapp: connector call, LLM step, approve, wait, notify, branch — every step checkpointed |
| Connectors | Catalog browser (live from the API) · one governed native connector (Box) · **one custom API connector created live through the wizard** |
| Approvals | Phone-approvable cards with expiry, payload preview, content binding; approvals outside the agent's control |
| Evidence | Run timeline with receipts; checkpoint before run; Rewind on reversible effects; irreversible flagged "cannot unsend"; the "read my Box token from the Sprite → denied" diagnostic |
| Control | Limits declared per spec and enforced for the demoed ones; kill switch; app gallery with three generated apps |

### Roadmap — after Hacksgiving, on the public roadmap (say this, not more)

1. **Durable Workflow Studio (the flagship differentiator):** the visual composer that compiles end-user intent into governed durable workflows — long-running schedules and waits with a real timeline UX, saga/compensation designer, **reconciliation console for UNKNOWN outcomes** (the "your agent won't double-charge you" guarantee made manageable), per-family version upgrades with diff, deterministic-replay debugging view. The engine underneath already exists and is tested; **the Studio authoring surfaces are what's being built next.** One sentence for the README: *"Durable workflow authoring is on the roadmap; today's demo runs each generated step on the framework's durable execution substrate — receipts are live, the visual composer is next."*
2. Data & presentation depth: multi-source entities, theming/branding, publishing and sharing (chatgpt families 1 and 6).
3. Embedded/white-label distribution (GigaCatalyst's B2B model) after one coherent standalone experience (chatgpt §8).
4. Marketplace of digest-pinned templates; multi-user accounts; billing (claude's cut list — deliberately not in the demo).

**Language fence for the demo and README:** ✅ "every step is checkpointed; the run survives a Sprite restart" · ✅ "every consequential action is receipted; ambiguous outcomes are held for you" · ❌ "no duplicate effects ever" (claim only what the pinned version's tests prove) · ❌ "visual workflow designer" (it's roadmap) · ❌ "the agent can never misbehave" (say: "it can't act without a receipt and a gate, and you can stop it").

---

## 6. Demo script (three minutes, merged: deepseek's beats + claude's undo/attack + gemini's live-run)

| Beat | Time | Scene |
|---|---|---|
| 1 · Hook | 0:00–0:15 | "Every demo you'll see today is one hand-built agent. We built the studio that generates them — on its own Sprite, with a gate and a receipt." |
| 2 · Describe | 0:15–0:45 | Type the invoice-agent prompt (email → Box → PO match → remind, ask first). Spec card appears: miniapps it will create, connector paths it needs, fixtures count. |
| 3 · Validate | 0:45–1:00 | Red/green fixture indicator; then deliberately describe an unapproved mass-send — the Studio refuses, showing the rejected spec. Safety layer, on camera. |
| 4 · Connect | 1:00–1:30 | Connector picker: show the catalog, then the **custom API wizard live** — paste a public API's base URL, pick header auth, allow `/v1/todos*` only, test green, grant to the Sprite. "Any API becomes a governed connector in a minute, and the key never touches the Sprite." |
| 5 · Deploy & run | 1:30–2:00 | Deploy via Fly MCP; the agent processes items; approval card lands on the phone; approve. Receipt lands in the timeline. |
| 6 · Break it | 2:00–2:30 | `fly machine kill` mid-run (or checkpoint restore). The run resumes; the ledger shows **one approval, one send, one receipt** — nothing double-fired. Attack scoreboard: a booby-trapped message tries to exfiltrate; denied and receipted. |
| 7 · Close | 2:30–3:00 | Gallery: three different generated apps from one platform. "Describe any agent. Get a governed one on a Sprite. Receipts all the way down." |

---

## 7. Rule compliance and repo strategy (claude §10, unchanged in substance)

1. **New repository today** (Nov 23 commits), containing everything judges see: Studio UI, generator, Spec Runner, primitives, templates, demo.
2. **The framework as an external dependency** — separate repo, pinned version, credited by name and link; submission says plainly: *"Built this week: the platform, generator, spec runtime, UI and apps. Foundation: [miniapp], developed earlier, imported as a library."*
3. **Email the organizers today** (two sentences) asking whether a pre-existing open-source library as a dependency is acceptable; keep the reply. **Plan B:** rebuild a lean kernel (ledger, approvals, audit, kill switch) fresh in the new repo — the design doesn't change.
4. **Freeze framework remediation** until Dec 1; hackathon targets the pinned version. **Day-one portability spike:** framework (Postgres-bound) on a Sprite → SQLite locally for working state if the durable-execution library supports it, off-Sprite ledger for world-state; if the spike fails, fall back to the lean kernel.
5. **Connectors differ per organization** — check the org dashboard first; design around what's actually enabled; don't bend the demo around a connector that isn't there.

## 8. Consolidated six-day plan

| Day | Ship (from claude's plan + deepseek's task table) |
|---|---|
| Mon 23 | Organizer email · new repo · Sprite up · org connector check · portability spike · spec schema v0 |
| Tue 24 | Spec Runner v0 (connector call, approve, wait, ledger) · `validate` command · hand-written spec runs end-to-end |
| Wed 25 | Generator loop (agent writes spec → validator feeds back → fixtures dry run) · permissions card |
| Thu 26 | Studio UI: gallery, spec card, approvals page, timeline · **connector catalog browser (live list) + custom API connector wizard against the real endpoints** · phone approvals while the Sprite sleeps |
| Fri 27 | Rewind + compensations · second and third apps · custom-connector demo beat rehearsed · attack scoreboard |
| Sat 28 | Record video · README with built-this-week disclosure + roadmap section (durable workflow authoring = next) · post publicly, tag @flydotio · **submit Saturday night** |
| Sun 29 | Buffer only (deadline midnight PST ≈ 4 PM Monday PH time) |

## 9. Risk register (merged)

| Risk | Likelihood | Mitigation |
|---|---|---|
| "Built during the week" challenged | Medium | §7 route: disclosure + organizer email + lean-kernel plan B |
| Platform scope overruns | High | Golden path only; three apps not twelve; cut list enforced |
| Reads as "just a prompt wrapper" | High | Spec card + fixtures + rejected-spec beat on camera; deterministic validation visible |
| Framework won't run on a Sprite | Medium | Day-one spike; SQLite/lean-kernel fallback; off-Sprite ledger |
| Org lacks a specific connector | Medium | Live catalog from `GET /v1/oauth/connections`; custom API connector as the universal fallback beat |
| Overclaiming durable guarantees | Medium | §5 language fence; claim only pinned-version-tested properties |
| Sprite provisioning slow on camera | Medium | Progress-as-feature ("creating Sprite → installing runtime → validating → ready"); cut in the video |
| Connector types differ (DB connectors reject endpoint rules) | Low | Studio form adapts to connector type (§4.2 note) |

## Sources

- `platform_chatgpt.md`, `platform_claude.md`, `platform_deepseek.md`, `platform_gemini.md` (the four consolidated reviews)
- `docs.fly.io/sprites/api/connectors` + `docs.fly.io/sprites/api/openapi.json` (Connectors API: custom API connector create/test/update, list, provision, access policy schema, auth methods — verified 2026-11-24)
- `fly.io/sprites/ecosystem/` (connector catalog: 37 connectors including Pinecone, Box, Custom API; SDKs incl. official Go SDK `superfly/sprites-go`; integrations incl. Claude Code/Codex plugins and Fly MCP at `mcp.fly.io` — scraped 2026-11-24)
- `upload/hacksgiving.md` (rules, dates, prizes, judges)
- miniapp framework code read earlier at commit `1a67ba2` (approvals, effects ledger, Spec-Runner-relevant runtime, manifests with JSON Schema)
