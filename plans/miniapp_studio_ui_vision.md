# Miniapp Studio — UI Vision: Quiet by Default, Deep on Demand

- **Date:** 2026-11-25
- **Inputs:** the Studio roadmap (revised 25 Nov 2026), the four platform reviews you attached, the Fly.io Sprites ecosystem page and the Sprites connectors API reference
- **Your brief:** minimal left sidebar with essentials only, bottom bar for chat, right sidebar for crucial information, a middle pane that shows what needs to be shown; one UI that serves developers and operators; modern, inventive, no clutter
- **Honesty rule for this document:** everything described is a **design**. Which surfaces are live, simulated or roadmap is stated in Section 11 and must be bound to evidence records, not to this document.

---

## 1. The idea in one paragraph

Studio is not a dashboard with a chat widget. **The conversation is the product, and everything else is a lens the conversation (or your attention) summons.** There is one **Stage** in the middle that shows exactly one thing at a time: the thread, a question, a build, a diff, a connector, a deployed app. A slim **rail** on the left holds only the five places you actually go. A **Pulse** on the right tells you what needs you, what is happening now, and what just happened. A **command bar** at the bottom is how you speak to the system, and it always knows what you are looking at.

The roadmap already says the user should interact "through one durable Studio conversation rather than managing terminals, coding-agent sessions, or cloud infrastructure." This UI makes that literal.

---

## 2. Design principles

| # | Principle | What it means in the interface |
|---|---|---|
| 1 | **One thing at a time** | The Stage shows a single focus object. No tab bars, no multi-widget dashboards |
| 2 | **Attention is budgeted** | Pulse shows at most three "Needs you" items, then a count. Nothing blinks, nothing nags |
| 3 | **Verbs, not menus** | Actions are named by what they do (Answer, Approve, Rework, Authorize), and each verb looks different, because the roadmap keeps clarification, proposal and authorization distinct |
| 4 | **Weighty actions feel weighty** | Authorizing a deployment is a deliberate press-and-hold showing the digest. Answering a question is Enter |
| 5 | **Evidence over assertion** | Every claim carries a status chip (verified, simulated, roadmap, unknown). A fake-backed result never looks live |
| 6 | **Progressive disclosure, not two UIs** | Owners see plain language. A single Technical toggle reveals digests, raw events, commands and cursors for developers and operators |
| 7 | **Calm motion** | Motion explains state change (a builder waking, a stream replaying). It never decorates |
| 8 | **Keyboard first, touch ready** | Every Stage and Pulse action has a key. On a phone the same model collapses to a card stack |

---

## 3. Layout

```text
┌────┬──────────────────────────────────────────────────────────────┬─────────────────────┐
│ ◐  │ ━━ Builder ● awake · stream ● live · Renewals workspace ━━━━ │ NEEDS YOU        2  │
│    │                                                              │ ▸ Answer: which CRM │
│ ⌂  │                                                              │ ▸ Review build 14   │
│ ▦  │                         S T A G E                            │─────────────────────│
│ ⌁  │            (the one thing that matters right now)            │ NOW                 │
│ ☰  │                                                              │ ● Builder: tests    │
│    │                                                              │   running 2 of 5    │
│    │                                                              │─────────────────────│
│ ⚙  │                                                              │ THEN                │
│    │                                                              │ · 10:42 you answered│
│    │                                                              │ · 10:31 build 13 ✓  │
├────┴──────────────────────────────────────────────────────────────┴─────────────────────┤
│ ◇ Re: Build 14 · diff   Ask, instruct, or type / for commands                        ⏎  │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

| Region | Size and behavior |
|---|---|
| **Horizon line** (top, one thin strip) | Replaces a header bar. Shows builder state, stream state and workspace. Becomes a one-line banner only when something is wrong |
| **Left rail** | 56 px, icons only, labels on hover. Never expands into a tree |
| **Stage** | Fluid, centered content column (about 720 px for reading, full width for diffs and galleries) |
| **Pulse** | 280 px, collapsible with `]`. Auto-collapses on narrow screens into a bottom sheet |
| **Command bar** | Docked bottom, always visible, one line that grows to a few when you type. Carries a **context chip** for whatever the Stage is showing |

---

## 4. Interaction model

### 4.1 Focus object

The Stage always shows one **focus object**. A breadcrumb above it (`Thread › Build 14 › Diff`) lets you go back. The command bar's context chip (`Re: Build 14 · diff`) means "when I type, I am talking about this." You can clear it with `Esc`.

### 4.2 Decision verbs

| Verb | Used for | Gesture | Look |
|---|---|---|---|
| **Answer** | The agent asked a clarification | Type and press Enter | Calm blue |
| **Approve or Rework** | A proposal or an artifact review | Click, or `a` / `r` | Green and amber |
| **Authorize** | Deploy, rollback, grant a permission | **Press and hold** (about 0.8 s) with the digest and target read out | Deep indigo with a ring that fills |

An agent can ask, recommend or request. It cannot grant itself anything, and the UI never offers a path that implies otherwise.

### 4.3 Keyboard

`⌘K` focus command bar · `.` next "Needs you" item · `[` and `]` toggle rail and Pulse · `1`–`5` switch lens on the Stage · `j` / `k` move through files or events · `a` approve · `r` rework · `t` Technical toggle · `g h` History · `g a` Apps · `g c` Connectors.

---

## 4a. Inputs consolidated from the four reviews

| Source | What I kept | What I did not carry over, and why |
|---|---|---|
| ChatGPT | Declarative specification and compiler layer; three working areas (builder, preview, properties); GigaCatalyst principles (permission inheritance, versioned governed contracts); Fly connector discovery versus business capability discovery as two separate concepts | The "visual editor" emphasis: the roadmap forbids a workflow canvas in this phase |
| DeepSeek | Nine primitives as visible UI elements (describe, spec, validate, deploy, connect, approve, checkpoint, upgrade, talk); the "diagnostic proof that credentials are not in the Sprite" moment | Claims that approval channels, reconciliation and per-family versioning are "already done"; our own contract and roadmap say those are not yet proven. Also its statement that TypeSafe AI is a schema emitted against, and that Studio drives Sprite creation through named Fly MCP tools: neither is supported by the ecosystem page (TypeSafeAI is a decision-model connector; tool names are unverified) |
| Gemini | Judge-level traps: agent-first demo, visible guardrails, one live generated app beats a half-wired studio | Nothing substantive discarded; it is short and mostly strategy |
| Z | Receipts culture (run timeline with evidence), a control room for limits and an exception inbox, plans as data, the Fly MCP and gateway observations, checkpoint-and-resume demo flourish | Limits enforcement and several framework details assumed as present; treat each as a claim to verify |

---

## 5. The left rail: five places, nothing else

| Icon | Place | Why it earns a seat |
|---|---|---|
| ◐ (top) | **Workspace switcher** (tenant and application workspace) | The roadmap's isolation unit is one Builder per application workspace; switching must be one gesture |
| ⌂ | **Home** (the thread and "Now") | Default landing; the durable business conversation |
| ▦ | **Apps** | Everything deployed, with health. The ops front door |
| ⌁ | **Connectors** | The catalog, access policies and custom APIs |
| ☰ | **History** | Replayable record of conversations, builds, decisions and audit |
| ⚙ (bottom) | **Settings** | Roles, notification policy, capability ledger, quotas later |

**Deliberately absent:** a terminal, a file browser, a task list, a workflow canvas, a marketplace and billing screens. Builds are not a nav item: they are reached from the conversation and from Pulse, because a build is an event in a business conversation, not a place.

---

## 6. The Stage: one component, many objects

Every Stage view has the same anatomy, so the interface feels like one thing.

```text
┌──────────────────────────────────────────────────────────────────────────┐
│  Title · state chip · evidence chip                    lens ribbon  1 2 3 4 5│
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Body (the object, rendered for its type)                                │
│                                                                          │
├──────────────────────────────────────────────────────────────────────────┤
│  Decision bar:  [Rework]  [Approve]                    scope · expires   │
└──────────────────────────────────────────────────────────────────────────┘
```

### 6.1 Home: the thread as moments

The default Stage is the conversation. It is not a chat log of bubbles; it is a vertical thread of **moments**:

| Moment | Look | Opens into |
|---|---|---|
| Your message | A quiet line of text | Nothing |
| Status | One-line progress with a breathing dot | The Build view |
| Question | A prominent card with the question and an inline answer field | The Question view |
| Proposal | A plain-language summary card | The Proposal view |
| Evidence | A compact card: tests passed, validation passed, digest | The Artifact review |
| Decision receipt | A small closed card: "You approved sha256:ab12… at 10:50" | The audit entry |

Cards expand into the Stage. The thread never turns into a wall of logs: raw events live behind the Technical toggle.

### 6.2 Clarification (the question)

```text
 Which CRM should the renewals app read from?             awaiting your answer
 ────────────────────────────────────────────────────────────────────────────
  The builder needs this before it can continue.     asked 10:38 · turn 3 · build 14
  ( ) HubSpot    ( ) Salesforce    ( ) GoHighLevel    ( ) Something else…
  ┌──────────────────────────────────────────────────────────────────────┐
  │ Type an answer…                                                  ⏎   │
  └──────────────────────────────────────────────────────────────────────┘
  Stale if the build is cancelled or the turn moves on. Answers are bound to this turn.
```

Binds visibly to the roadmap's rule that answers are turn-bound and stale answers are rejected.

### 6.3 Artifact review (Phase D surface)

```text
 Build 14 · Renewals dashboard                  ● Awaiting your review   ✔ Verified by Studio
 Diff   Spec   Tests   Provenance   Transcript                              (lens 1–5)
 ────────────────────────────────────────────────────────────────────────────
  Admission  ✔ MiniappSpec valid   ✔ Go/Miniapp only   ✔ Dependencies allowed
  Tests      ✔ 24 passed · 0 failed     Digest sha256:7f3c…  size 1.2 MB
  ────────────────────────────────────────────────────────────────────────
   + handlers/renewals.go                                  (unified diff, j/k to move)
  ────────────────────────────────────────────────────────────────────────
  [Rework…]                                              [Approve for deployment]
```

- The **lens ribbon** replaces tabs: one lens at a time, keys `1`–`5`.
- **Admission strip** shows the roadmap's checks (MiniappSpec admission, Go-only implementation, dependency allowlist, tests). A completion claim without a passing required test report is shown as "Not accepted."
- **Inline comments** on the diff collapse into one rework message in the thread.
- An **Approve** here is not a deployment. Deployment is a separate Authorize action in Apps (Phase E), as the roadmap requires.
- A discreet footer notes: *"Secret patterns are redacted from views. This is not complete data-loss prevention."* (the roadmap forbids claiming otherwise).

### 6.4 Build in progress (honest builder states)

```text
 Build 14 · running                                       Builder ● awake
 ────────────────────────────────────────────────────────────────────────────
  ✔ Understood requirement          ✔ Asked 1 question, answered
  ● Writing Go implementation       ○ Running tests        ○ Preparing evidence
  Builder woke 10:36 (took 6 s)      Last signal 4 s ago   Stream live
```

States are explicit and honest: **asleep → waking → awake → resuming → failed**, with times. If the Builder was asleep, the UI says so and shows the wake as part of the story, never as an unexplained spinner.

### 6.5 Apps (the ops front door)

```text
 Apps                                                              search ⌕
 ────────────────────────────────────────────────────────────────────────────
  ● Renewals dashboard   v3 · sha256:7f3c…   Runtime awake   healthy   10:50
  ● Invoice reminders    v1 · sha256:21ab…   Runtime asleep  healthy   yesterday
  ▲ Onboarding           v2 · sha256:98de…   Runtime awake   health check failing
```

Opening an app shows, one lens at a time: **Overview** (health, URL, runtime identity), **Versions** (approved digests with provenance and rollback targets), **Activity**, **Connectors** (what it may call), **Limits** (clearly labelled "External writes disabled while G-01 is open" when true). The ops actions are **Pause**, **Roll back to a previously approved digest** and **Retire**, each an Authorize gesture.

### 6.6 History (the scrubber)

A horizontal event timeline under the Stage body. Dragging the cursor shows the thread, build and decision state **as of that event**. It reuses the cursor replay the platform already has for the stream, so history and live view are the same mechanism. Developers and operators use it to answer "what did the system know when it did that?"

---

## 7. Pulse (the right rail)

Three bands, always in this order:

| Band | Contents | Rules |
|---|---|---|
| **Needs you** | Questions awaiting an answer, artifacts awaiting review, approvals expiring, connectors needing re-authorization or about to expire | Max three shown, then "+n more". Each item is verb-first ("Answer: which CRM?"). `.` jumps to the next |
| **Now** | Builder state, current build step, stream state (live, replaying from event n, offline), in-flight deployments | Only things in motion. Empty means quiet |
| **Then** | The last few events in plain language | Collapsed by default after five |

### Notification policy

- Notifications are **Needs you** items, nothing else. There is no separate notification center.
- Severity is expressed by position and a left accent line, not by badges everywhere.
- Quiet by default: no toasts for routine progress. A toast appears only when your decision is requested while the Stage is elsewhere, and it carries the same verb as the Pulse item.
- Optional daily digest and quiet hours live in Settings.

### Technical toggle in Pulse

Adds for developers and operators: callback outbox depth and last acknowledgment, event cursor number, command deduplication state, retry timers. Hidden for owners.

---

## 8. The command bar

One bar, three jobs: **talk**, **command**, **navigate**.

| Input | Meaning |
|---|---|
| Plain text | Sent to the business conversation, scoped by the context chip |
| `/status` | Builder, stream and workspace health |
| `/connectors`, `/connect <provider>` | Open the catalog, or start adding one |
| `/apps`, `/history` | Navigation |
| `/deploy`, `/rollback` | Start an Authorize flow for an approved digest (disabled until one exists) |
| `/why` | Ask the system to explain the object on the Stage |
| `/roadmap` | Open the capability ledger (Section 11) |
| `@` | Mention an app, build, artifact or connector to attach it as context |

**Preview chip before sending.** A command shows what will happen ("Will wake Builder and start a turn") so the system never surprises you. When the Builder is asleep, the bar says so ("Builder asleep, waking takes a moment") instead of letting the message disappear.

**Context chip.** `◇ Re: Build 14 · diff` is attached automatically. Ask "why this file?" and the system knows which file.

---

## 9. One UI, three people

The roadmap's actors are owners, developers who review, and operators who run. The interface does not fork. A **role preset** changes defaults only.

| | Owner | Developer | Operator |
|---|---|---|---|
| Lands on | Home thread | The next artifact awaiting review | Apps with health |
| Left rail order | Home, Apps, Connectors | Home, History, Apps | Apps, History, Connectors |
| Default lens | Summary | Diff or Tests | Overview or Activity |
| Pulse filter | Questions and approvals | Reviews and build state | Health, expiring connectors, failed callbacks |
| Technical toggle | Off | On | On |
| Language | Plain | Precise | Precise, with runbook hints |

The same toggle (`t`) works for everyone. A developer can read an owner's view and an owner can peek at the technical layer without switching accounts.

### Incident mode (operators)

When something fails, the Stage reorganizes into three short parts: **What happened** (in one sentence), **What it affected** (apps, builds, connectors), **Safe actions** (pause, roll back, retry a callback), each with the same weight rules. No dashboards to hunt through.

---

## 10. Visual language

| Element | Direction |
|---|---|
| Mood | Calm, editorial, a quiet instrument panel; generous whitespace; soft surfaces instead of boxed grids |
| Color | Neutral base with **one** accent (indigo). Semantic colors only for state: green (verified), amber (needs you), red (failed), **violet dashed (roadmap)**. Dark theme by default, light available |
| Type | A clean sans (such as Inter or Geist) at 14 px base; large Stage titles; monospace only for digests, paths and ids |
| Surfaces | 12–16 px radius, soft elevation, no heavy borders; the Stage floats slightly above the rail and Pulse |
| Motion | 150–220 ms ease-out; a slow "breathing" dot for the Builder; the stream replay shows a thin progress sweep; honors reduced-motion |
| Iconography | One thin-line set, 20 px; no icon without a label on hover |
| Density | Two settings only: comfortable (default) and compact |

Components needed (about a dozen, reused everywhere): Moment card, Stage frame, Lens ribbon, Decision bar, Hold-to-authorize, Status chip, Evidence chip, Diff view, Timeline scrubber, Pulse item, Command bar with context chip, Connector tile.

---

## 11. Honesty system: status vocabulary and the capability ledger

The roadmap insists on "an honest distinction between inspected, simulated, and live-executed behavior." The UI enforces it.

| Chip | Meaning | Example |
|---|---|---|
| ● **Live** (solid) | Executed against the real system; links to an evidence record | "Live: question answered in the same session" |
| ○ **Simulated** (hollow) | Fake-backed or fixture-backed | "Simulated: scripted agent double" |
| ▨ **Roadmap** (violet, dashed) | Designed, not built | "Durable workflows: roadmap" |
| ? **Unknown** (grey) | Not yet verified | "Connector token expiry: not yet read" |
| ✔ **Verified** | A check Studio itself ran, with digest | "Verified by Studio: 24 tests, digest sha256:…" |

### The capability ledger (Settings → Capabilities, also `/roadmap`)

A single page listing each capability with its chip, the evidence record that justifies it, and a one-line "what this does not prove." It is the antidote to overclaiming: a visitor can see exactly what is live.

### Durable workflows: shown as the differentiator, labelled as roadmap

Durable workflows are the long-term differentiator, but they are not in the hackathon build. The UI handles that without overclaiming:

- In the Apps view a **"Durable runs"** lens appears with the roadmap chip, showing a concept frame: a run timeline with checkpoints, durable waits, approvals and resume after restart.
- The frame is plainly marked **"Concept. Not available in this build."** It cannot be clicked into a live run.
- Claims such as "no duplicate confirmed effects" appear only after evidence exists, and consequential external writes remain disabled while G-01 is open; the UI says exactly that.

---

## 12. Connectors: catalog, policy and extensibility

This section covers your two requirements: show every supported connector, and let users extend Studio with custom API connectors.

### 12.1 What the platform provides (from the Fly documentation)

- The connectors API lists connectors configured for an organization. Each record carries the provider, connection type (oauth, api_key, provisioned, internal or database), granted scopes, token expiry, transport (http or postgres) and an **access policy**: whether all Sprites may use it, allowed endpoints, blocked endpoints, a Sprite name prefix and required Sprite labels. Block rules are checked before allow rules, and a connector with no access policy denies all Sprites. Database connectors reject endpoint rules.
- The ecosystem page lists the supported providers (61 entries when I read it, ten marked community-built) and states that availability varies by organization during rollout.
- A **Custom API** connector connects a token-authenticated HTTP API through the gateway. It supports bearer or custom headers, query parameters and token-bearing URL paths, requires the upstream to be publicly reachable (private and loopback addresses are blocked), and asks for a test URL before saving.

### 12.2 The gallery

```text
 Connectors                                   ⌕ search     Show: All · Configured · Needs attention
 ────────────────────────────────────────────────────────────────────────────
  Models (15)   Work (8)   Messaging (4)   Data (6)   Agent tools (4)   Web (9)
  Voice (5)     Money (3)  Infrastructure (6)   Custom (1)

  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
  │ GitHub      │ │ Slack (Bot) │ │ Box         │ │ Pinecone    │ │ ＋ Custom API│
  │ OAuth       │ │ OAuth       │ │ OAuth       │ │ API key     │ │ Token+endpoint│
  │ ● configured│ │ ○ not set up│ │ ▲ reauth    │ │ ● configured│ │ your own API │
  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘
```

- **Facets:** category, auth type (OAuth, API key, token plus endpoint, database), origin (Fly or Community), status (configured, not configured, needs re-authorization, expiring).
- **Catalog source:** a versioned catalog file in Studio, stamped "as of" a date and generated from the ecosystem page, so the gallery shows all supported connectors even when the organization has none configured. **Live status** overlays it from the connectors API.
- **Honest labels:** "Community" on community-built entries; "availability varies by organization" on tiles the live list does not return.
- **Adding or re-authorizing** deep-links to the Fly dashboard (Sprites, Connectors, Add connector), because the documented API I read only lists connectors. If creation is also available by API, that can come later; confirm before promising it.

### 12.3 The connector detail Stage

```text
 Box                                          OAuth · Fly connector · ● configured
 Overview   Access policy   Used by   Bindings   Technical                    (1–5)
 ────────────────────────────────────────────────────────────────────────────
  Allowed endpoints    /2.0/folders/*   /2.0/files/*       (editable list)
  Blocked endpoints    /2.0/*/collaborations   ← block rules win
  Name prefix          studio-          Required labels   tenant-renewals
  Notes from provider  Requires read-write permission even for downloads…
  Token                refreshes on use · reconnect after idle expiry
```

- **Access policy** is rendered as plain lists with the rule "blocked paths are checked first."
- **Used by** lists the apps and builds whose MiniappSpec declares these endpoints, so you can see the blast radius of changing a policy.
- **Warnings** surface provider notes that matter (for example a connector that bypasses row-level security, or one that grants broad file access) in plain words.
- **Technical** shows the connector id and the gateway usage snippet.

### 12.4 Two lanes: where a connector may be bound

The roadmap's invariant is that **production connector bindings never enter Builder Sprites**, and Runtime Sprites receive only what the application needs.

```text
  Builder lane (build and test)          Runtime lane (production, per approved app)
  ───────────────────────────            ─────────────────────────────────────────
  Sandbox or test credentials only       Only after an approved deployment
  🔒 production connectors cannot        Bound by app, with its declared endpoints
     be granted here
```

The UI makes the invariant visible: the Builder lane shows a lock for production connectors; the Runtime lane shows what an approved app may call.

### 12.5 Extensibility: custom API connectors

The key idea: **a connector is a credential plus a capability map.**

- The **credential** lives in the Fly connector and never passes through Studio.
- The **capability map** lives in Studio: the operations the API offers, their effect class (read, write, destructive) and the default policy. It feeds the MiniappSpec's capability declarations.

The wizard has five steps:

1. **Describe.** Name, base URL, authentication method (bearer, custom header, query parameter, token in path) and a test URL. A note states that private and loopback addresses are blocked.
2. **Map operations.** Paste or import an OpenAPI document, or list operations by hand. Studio proposes endpoint rules.
3. **Classify.** Each operation is marked read, write or destructive. Defaults: reads allowed, writes require human approval, destructive operations blocked.
4. **Review the policy.** Studio shows the allowed and blocked endpoint lists and the Sprite name prefix or labels it recommends, and which apps would use them.
5. **Hand off.** Studio opens the Fly dashboard with the non-secret values prefilled. **The token is entered only in Fly.** Studio stores the connector id and the capability map.

Studio's own policy is a second, earlier line of defense: the gateway's path rules remain the hard stop.

### 12.6 Tenant partitioning (a design question to verify)

Fly connectors belong to a Fly organization, while Studio is multi-tenant. Access policies support Sprite name prefixes and labels (all listed labels are required), so the likely pattern is one label or prefix scheme per tenant or workspace so that one tenant's Sprites can never use another tenant's connector. Verify the label constraints and how this scales before relying on it.

---

## 13. Responsive and accessible

| Context | Behavior |
|---|---|
| Phone | Stage fills the screen; command bar stays at the bottom; Pulse is a swipe-up sheet; the rail becomes a long-press drawer. Approvals are full-screen cards; hold-to-authorize works by touch |
| Tablet | Rail and Stage; Pulse as an overlay |
| Keyboard | Every action reachable; visible focus; logical order rail, Stage, Pulse, command bar |
| Screen readers | Pulse is a live region with polite announcements; decision verbs are labelled with their consequence |
| Contrast and motion | WCAG AA or better; reduced-motion removes breathing and sweeps |

---

## 14. First-run and empty states

There is no empty dashboard. The first screen is the Stage showing one sentence ("Describe a process you want automated") above the command bar, with three example chips. Rail items that have no content yet show a single line explaining what will appear, not an empty grid.

---

## 15. Mapping to the roadmap, and what is demoable

| UI surface | Roadmap dependency | Honest status to show until evidence says otherwise |
|---|---|---|
| Home thread, moments, replay on reconnect | Phase A accepted (conversation infrastructure, cursor replay) | Live for stored conversations; the agent side is simulated until the live proofs |
| Builder state and wake | Phase B Part 1 fake-backed, Part 2 needs a separate GO | Simulated |
| Question and answer in the same session | Phase C (genuine two-way interaction) | Simulated until proven |
| Artifact review and admission strip | Phase D | Simulated with fixtures, or roadmap |
| Apps, deployment, rollback | Phase E | Roadmap or simulated |
| Connectors gallery and detail | Sprites connectors API (list) plus a catalog file | Live if executed against a real organization; mark unverified until then |
| Custom API descriptor and wizard | Studio-only capability map; hand-off to the Fly dashboard | Live as a Studio feature; the hand-off is manual |
| History scrubber | Existing stored events and cursors | Live for stored events |
| Durable runs | Roadmap | Concept frame, clearly labelled |
| Capability ledger | None | Always live; the honesty backbone |

**For a demo:** build Home, Pulse, the command bar, the Connectors gallery with the Custom API wizard, and one Artifact review on fixtures. Every simulated surface wears the hollow chip. A demo that is honest about what is simulated is stronger than one that is not.

---

## 16. Open decisions

1. **Role model:** are owner, developer and operator distinct permissions in a tenant, or presets of one account?
2. **Connector creation:** can the Sprites API create and update connectors, or is the dashboard the only path? (The page I read documents listing.)
3. **Tenant to connector mapping:** the label or prefix scheme, and its limits.
4. **Technical toggle defaults** per role, and whether owners can see it at all.
5. **Theme default:** dark first as proposed, or follow the system setting.
6. **Mobile priority:** is phone approval a launch requirement or later?
7. **Naming:** "Stage" and "Pulse" are working names; keep, or use plainer labels in the product.

---

## Appendix A — Connector catalog (61 entries when read, 2026-11-25)

Source: the Fly.io Sprites ecosystem page. † marks entries labelled community-built. Availability varies by organization.

| Category | Connectors |
|---|---|
| **Models and decisions (15)** | Meta (Muse Spark), OpenAI, Anthropic, Google Gemini, OpenRouter, Ollama Cloud, Fireworks, Parasail, Hugging Face, DigitalOcean Inference, Nscale†, ZeroGPU†, Codex, TypeSafeAI (a decision model), Echo by Fulcrum |
| **Work and collaboration (8)** | GitHub, Slack (User), Slack (Bot), Discourse, Linear†, Notion†, Granola†, Box |
| **Messaging and notifications (4)** | AgentMail, Resend, Pushover†, Blotato |
| **Data and storage (6)** | S3-compatible Object Store, Tigris, Neon, Supabase Database, Upstash Redis, Pinecone |
| **Agent tools and memory (4)** | SenseLab, Mnemom, Composio Tools, Usable |
| **Web and research (9)** | Steel Browser Tools, Firecrawl, Browserbase, Apify, Kagi, Exa, Linkup, Tavily†, Keenable |
| **Voice and media (5)** | Deepgram, ElevenLabs, AssemblyAI†, Vocal Bridge, LemonSlice |
| **Money (3)** | Stripe†, Era, Link Agent Payments |
| **Infrastructure and observability (6)** | Fly.io Machines, Depot build insights, Temporal Cloud, Gigapipe, Sentry†, Nebius Serverless AI Jobs |
| **Custom (1)** | Custom API (token plus endpoint) |

### Notes that matter for governance (from the provider notes)

- Some connectors carry broad authority, which the connector detail should state in plain words: Supabase Database uses a service-role key that bypasses row-level security; Box requires read-write permission even for downloads; Neon lets authorized Sprites fetch database credentials; Link Agent Payments routes spend requests to the account owner for approval in the Link app.
- Connectors that send mail, publish posts or place calls have real external effects; Studio should classify these as write effects, which matters while consequential writes remain disabled.

## Appendix B — Ecosystem beyond connectors (names only)

- **Agents and plugins (16):** Claude Code, Codex, Cursor, Antigravity, Grok Build, Cortex Code, DeepSeek Harness, OpenCode, Pi, VS Code, Herdr, Droid, Hermes Agent, Orca, any MCP client, Agent Plugins v1. The roadmap's first coding agent is OpenCode.
- **SDKs (8):** Python, JavaScript, Go (`github.com/superfly/sprites-go`), Elixir, Google ADK, LlamaIndex, OpenAI Agents SDK, OpenRouter Agent SDK.
- **Integrations (10):** TanStack AI, Claude Managed Agents, Gemini Managed Agents, Anthropic Connectors Directory, QM, Cursor Cloud Agents, Mastra, Pydantic AI Harness, Goose, Datasette Agent.

A future "Ecosystem" lens inside Connectors could show these, but only the connectors catalog is needed for the current scope.

---

## Sources

- Miniapp Studio roadmap (attached `README_1.md` and `roadmap4.md`, revision 25 November 2026)
- Fly.io Sprites ecosystem: https://fly.io/sprites/ecosystem/ (connectors, integrations, SDKs including Go)
- Sprites API, List connectors: https://docs.fly.io/sprites/api/connectors
- Four attached platform reviews (`platform_chatgpt.md`, `platform_deepseek.md`, `platform_gemini.md`, `platform_z.md`)
