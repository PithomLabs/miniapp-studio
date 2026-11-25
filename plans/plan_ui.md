## prompt

reimagine the UI of Miniapp Studio that covers the critical and essential UI elements and workflow, how do you envision this for optimal experience for both devs and ops? the goal is to provide a minimal clutter-free UI, I do not want the standard UI of today, here is what I want: a minimal left sidebar but display only the essentials, bottom bar for chat, right sidebar to display crucial UI information like activity, notifications but feel free to suggest and a middle pane that displays what needs to be shown (again feel free to suggest more inventive modern-looking UI without the clutter


## deepseek

# Miniapp Studio — Reimagined UI

The design brief is unusual in a useful way: you've asked for a shape that already excludes the two most common failure modes of modern B2B products — the deep-tree sidebar and the dashboard landing page. What remains, if you hold to the constraint, is a four-zone surface where each zone does exactly one thing, and nothing is decorative.

Below is what I'd build, and why.

---

## 1. Design principles

**Three principles drive every decision in what follows.**

**Conversation is the workspace.** The central artifact of Miniapp Studio is not a project, a build, or a deployment — it is the durable conversation between the user and the agent that produced them. Questions, proposals, authorization requests, artifacts, approvals, and deployments all happen *inside* that conversation. The middle pane is not a tab-switched content area; it is the conversation stream, with rich interactive cards rendered inline as its events.

**Every zone owns one question.** Left rail answers *where am I?* Right rail answers *what needs me now?* Bottom bar answers *what do I want next?* Middle pane shows *what I'm looking at.* A user who is lost can ask any one of these questions and find the answer in one place, at a fixed position on screen, always.

**Signal, not feed.** The right rail is a triage surface, not a notification stream. Every item has a state (`needs you` / `in flight` / `done`), and the item auto-dismisses when handled. The user is never scrolling to find what to do next.

Everything else follows from these.

---

## 2. The anatomy

```
┌──────────┬─────────────────────────────────────┬────────────┐
│          │  Friday Box Digest      ● running   │  Needs you │
│  ◉ me    │  ─────────────────────────────────  │  ──────────│
│          │                                     │            │
│  Builds  │  │ User · 2h ago                     │  Q. Which  │
│  ──────  │  │ "Every Friday, summarize the     │  folder?   │
│  Deploy  │  │  reports in my Box folder..."    │  FBD       │
│  Conns   │  │                                  │  → answer  │
│          │  │ Agent · 2h ago                    │            │
│          │  │ ┌─ Analyzing workspace ──────┐   │  ────────  │
│          │  │ │ ✓ checkout loaded          │   │  Pending   │
│          │  │ │ ✓ MiniappSpec draft ready  │   │  approval  │
│          │  │ │ ◐ writing handler...       │   │  Invoicer  │
│          │  │ └────────────────────────────┘   │  → review  │
│          │  │                                  │            │
│          │  │ Agent · 1h ago                    │  ────────  │
│          │  │ ┌─ Question ─────────────────┐   │  Recently  │
│          │  │ │ "Which folder — top-level  │   │  ────────  │
│          │  │ │  or including subfolders?" │   │  GitHub    │
│          │  │ │ [Answer in bottom bar ↓]   │   │  Triage    │
│          │  │ └────────────────────────────┘   │  deployed  │
│          │  │                                  │  5m ago    │
│          │  │ User · 58m ago                    │            │
│          │  │ "Top-level only."                 │  ────────  │
│          │  │                                  │  Health    │
│          │  │ Agent · 58m ago                   │  ────────  │
│          │  │ ┌─ Artifact ready ───────────┐   │  ● 2/2     │
│          │  │ │ diff · spec · 24 tests ✓   │   │  healthy   │
│          │  │ │ [Review →]                 │   │            │
│          │  │ └────────────────────────────┘   │            │
│          ├─────────────────────────────────────┤            │
│          │  Building: Friday Box Digest        │            │
│          │  Ask, answer, or say what's next ⌘↵ │            │
└──────────┴─────────────────────────────────────┴────────────┘
```

Four zones:

| Zone | Width | What it does |
|---|---|---|
| **Left rail** | ~200px, collapsible to ~56px | Workspace switcher at top, then three flat destinations: **Builds**, **Deployments**, **Connectors**. Badge count on Builds if any build is waiting on you. Status dot on Deployments indicating fleet health. Nothing else. |
| **Middle pane** | Flex, primary | The conversation (in Create mode) or the fleet (in Operate mode). A thin header strip with the current item's name, status, and one or two contextual actions. |
| **Right rail** | ~280px, collapsible | Signal inbox. Three sections always in the same order: **Needs you**, **In flight**, **Recently done**. Bottom pinned strip shows fleet health dot. |
| **Bottom bar** | Full width of middle+right, or just middle | The **universal describe**. Context chip on the left ("Building: Friday Box Digest"), input in the center, send button on the right. Keyboard hint `⌘↵`. |

That's the entire chrome. No top bar, no tabs, no breadcrumbs (except a back arrow inside the middle pane's header when you've drilled into a sub-view), no notifications bell, no settings gear in the main view, no user avatar in the corner.

Settings and account live in the workspace switcher menu at the top of the left rail, which is a click-to-open popover, not a modal.

---

## 3. Two modes, one shape

The UI has exactly two modes, and they're not dev-versus-ops — they're **Create** and **Operate**. Same person, different tasks. Left rail switches between them.

### Create mode (left rail: Builds selected)

Middle pane = the conversation for the currently selected build. This is where the user describes, answers, reviews, approves, and eventually deploys. It covers Phases A through D.

The build progresses through a sequence of **typed events** in the conversation. Each event is a card, and the card's visual style, affordance, and interaction match its type:

| Event type | Card style | Primary action |
|---|---|---|
| User message | Text-only, left-bordered in neutral gray | — |
| Agent progress | Collapsible block (auto-opens while running, auto-collapses on completion) | — |
| Agent question | Highlighted card with pulsing left border | **Answer** → opens bottom bar in answer mode |
| Agent proposal | Card with 2–3 numbered options | **Choose one** |
| Authorization request | Card with payload preview and expiry countdown | **Approve / Reject** (payload-bound) |
| Artifact ready | Summary card: diff size, spec status, test count | **Review** → expands the artifact card in place |
| Artifact review (expanded) | Full diff viewer + MiniappSpec tree + test report + digests | **Approve / Request rework** |
| Deployment | Card with URL, digest, health check, version | **Open URL / Rollback** |
| Checkpoint saved | Slim single-line card | **Rewind here** (on hover) |
| Error | Red-bordered card with typed error and one suggested action | **Retry / Ask for help** |

Critically, **artifacts expand in place**. Reviewing a diff does not navigate anywhere; the card grows vertically, the conversation scrolls to accommodate it, and the user can collapse it again to return to the conversation context. This is the single most important invention in the design — most products would navigate to a "diff page" or "artifact tab," which breaks the conversation continuity. Here, review happens in the conversation because review *is* part of the conversation.

### Operate mode (left rail: Deployments selected)

Middle pane = fleet view. Each deployed application is a card in a grid:

```
┌──────────────────────┐  ┌──────────────────────┐
│ Friday Box Digest    │  │ Invoicer             │
│ ● healthy · v3 · 2h  │  │ ● healthy · v1 · 1d  │
│ digest: sha256:abc1…│  │ digest: sha256:def4… │
│                      │  │                      │
│ [Open] [Rollback]    │  │ [Open] [Rollback]    │
└──────────────────────┘  └──────────────────────┘
┌──────────────────────┐
│ GitHub Triage        │
│ ◐ degraded · v2 · 5h │
│ 1 UNKNOWN effect     │
│                      │
│ [Investigate]        │
└──────────────────────┘
```

Click a card → the middle pane switches to that app's **run timeline**: chronological event log (deploys, health checks, tool invocations, receipts), with the kill switch pinned in the top-right corner of the middle pane header, always visible.

The conversation for that app is one click away in the left rail (Builds → select the corresponding build). The deployment and its build are related but distinct views, because they answer different questions: "what did the agent build and why?" (build conversation) versus "what is the deployed app doing right now?" (deployment timeline).

---

## 4. The three inventions

The four-zone layout is conventional. Three things within it make it feel genuinely different from "a chat panel bolted onto an admin dashboard."

### 4.1 Left-border status rhythm

Every conversation event has a **thin colored left-border** (2px) indicating its state:

- **Neutral gray** — informational (user messages, completed agent progress)
- **Amber** — needs the user (questions, pending approvals, authorization requests)
- **Animated amber pulse** — actively waiting on the user, unresolved for more than a threshold
- **Green** — completed (finished agent work, approved artifacts, healthy deployments)
- **Red** — error state

The result is that scrolling the conversation shows, at a glance, where the user's attention has been required and where the current block is. There is no need for a separate "status" column, no badges on every message, and no chat bubbles. The left border does the work quietly.

On a per-build basis, the same left border on the left rail's "Builds" item shows the aggregate state of the most urgent build: amber if any build needs you, gray otherwise.

### 4.2 Signal inbox, not notification feed

The right rail has exactly three sections, always in the same order:

**Needs you** — items that require a decision or action to unblock. Highest priority first. Each item is one row:

```
Q. Which folder?                     FBD  → answer
Pending approval: v3                 Inv  → review
1 UNKNOWN effect                     Git  → reconcile
```

Each row has:
- **What** ("Q. Which folder?" — a truncated description)
- **Where** (three-letter code for the build/app)
- **What to do** (a verb, not a "view" link)

Clicking the row navigates the middle pane to that item. Answering, approving, or reconciling makes the row disappear. **The inbox empties itself when the user has done the work.** This is the anti-feed: the user is never scrolling through history in the right rail; history is in the middle pane.

**In flight** — items the system is actively working on that may need user awareness but not action. A slow build, a deployment in progress, a rollback propagating. Collapsed by default; shows a count. Expands to a compact list if the user wants to see.

**Recently done** — the last 5–10 completed items, older items fall off. Each is a one-line timestamp. Clicking navigates to the completed item in the middle pane if the user wants to review it.

**Fleet health** at the bottom is a single dot: green if all deployments healthy, amber if any degraded, red if any down. Hovering shows the count. Clicking switches to Operate mode.

The discipline: the right rail never contains *announcements*. No "Welcome to Miniapp Studio." No "Your build completed 3 hours ago" toasts. Only items that need you, are working, or recently resolved.

### 4.3 Universal describe (contextual bottom bar)

The bottom bar is not "chat." It is the **universal describe** — the place where the user tells the system what they want to do next, and the system routes based on context.

```
┌─────────────────────────────────────────────────────────┐
│  Building: Friday Box Digest      [x]  │ Describe…  ⌘↵│
└─────────────────────────────────────────────────────────┘
```

The **context chip** on the left is the key idea. It tells the user *what their next message will attach to*. It changes based on where they are:

- In a build conversation: **"Building: Friday Box Digest"** → next message continues that conversation
- In a deployment view: **"Operating: Invoicer"** → next message can be "rollback to v2", "show me yesterday's receipts", "why did this fail at 3am"
- In the fleet view: **"Fleet context"** → next message like "which apps are unhealthy?"
- In the connectors view: **"Connectors"** → next message like "add a custom API for my niche CRM"
- Empty state: **"New build"** → next message starts a new build

The `[x]` on the chip clears context; the placeholder updates to a generic "What do you want to do?"

The input is a single line by default, expands to multi-line on `Shift+↵`, and always submits on `⌘↵`. The right side of the input shows a small keyboard hint that disappears after first use.

This solves a real problem: in most AI products, the user has one global chat box and has to remember to say which thing they're talking about. Here, the context is always explicit, always visible, and always correct because it's derived from where they clicked.

---

## 5. The developer journey, walked through

To make this concrete, here's a developer building an agent from scratch.

**Morning — new build.**
Opens Studio. Left rail → Builds. Empty state in the middle pane: a big describe bar in the center with placeholder "What do you want to build?" and three example prompts below (clickable): *"Summarize my Box reports to Slack every Friday"*, *"Triage GitHub issues with approval gates"*, *"Reconcile invoice payments against my ledger"*.

Clicks the first example, or types their own. Hits Enter. The big describe bar **shrinks and moves** to the bottom bar's context chip. The first user message appears in the middle pane. The agent begins.

**The build progresses.**
The agent's progress appears as collapsible blocks. While running, blocks auto-expand. When done, they collapse to a one-line summary ("Analyzed workspace · 12 seconds" — user can click to expand). The user can scroll up to review what the agent did, or ignore it.

**A question arrives.**
An amber card appears in the middle pane with a pulse. The right rail's "Needs you" gets a new row. The bottom bar's context chip changes: **"Building: Friday Box Digest · Answer pending"**.

The user clicks the question card (or the row in the right rail — both do the same thing). The bottom bar's placeholder becomes: *"Answer the question…"*. The user types "top-level only, not subfolders." Submits. The amber card resolves to green. The right rail row disappears. The bottom bar context chip returns to normal.

**The artifact is ready.**
An "Artifact ready" card appears in the conversation with a summary: `diff · 3 files · spec valid · 24 tests passed`. The right rail shows "Review: artifact ready · FBD".

The user clicks Review. The card **expands in place** to show:
- Tabs at the top of the card (not the page): **Diff · Spec · Tests · Transcript**
- The Diff view (default): three files with syntax highlighting, collapsed by default to file headers; clicking a file expands it inline.
- The Spec view: the MiniappSpec rendered as a structured tree, with the framework-only invariant explicitly confirmed ("✓ Go only · ✓ Miniapp runtime only · ✓ MiniappSpec valid").
- The Tests view: a list of 24 tests, each with a pass indicator. Clicking expands to the test's assertion.
- The Transcript view: the agent's reasoning, collapsible.

At the bottom of the expanded card: **Approve** (green) and **Request rework** (neutral). The user clicks Approve. The card **morphs** in place: the review controls fade out, and the card becomes a **deployment card** with a "Deploying to Runtime Sprite…" progress indicator. No navigation happened. The conversation continued naturally.

**Deployment completes.**
The card settles into the deployed state: `● healthy · v1 · digest sha256:abc1… · open URL`. The right rail's "Recently done" gets an entry. The left rail's Deployments badge increments.

**Iteration a week later.**
The user comes back, clicks Friday Box Digest in the left rail. The full conversation is there. They type into the bottom bar: *"Also check for duplicate reports before posting."* The build starts a new turn in the same conversation. The previous deployment card stays in the history; the new card will appear below when the change is approved.

**Rewind, if needed.**
Every completed build step has a "Rewind to here" affordance that appears on hover. Clicking it opens an inline confirmation, then triggers the checkpoint/rewind flow. The user sees the plan resume from that checkpoint. This makes the "I don't like where this is going, undo to step 3" workflow feel like clicking undo in an editor, not like a recovery operation.

---

## 6. The ops journey, walked through

Now the same system from an ops person's perspective.

**Morning — fleet check.**
Opens Studio. **The UI opens in Operate mode** if the user has previously used Operate mode more than Create mode, or if there are items in "Needs you" — the platform learns from usage or defaults to attentiveness. Left rail → Deployments selected. Middle pane shows the fleet.

Right rail top row: *"1 UNKNOWN effect · GitHub Triage"*. The user clicks it.

**Investigating.**
The middle pane switches to GitHub Triage's run timeline. The UNKNOWN effect is highlighted in the timeline: *"Attempted to comment on issue #442 at 03:14. Provider timed out after 30s. No receipt returned."* Below the event, an inline card: **"Reconcile: read issue #442's comments to determine if the comment was posted."** with a single button **"Reconcile now"**.

The user clicks it. The system reads the issue's comments, determines the comment was posted, and the UNKNOWN resolves to CONFIRMED. The card morphs into a **receipt card** showing what the platform saw: *"Confirmed: comment was posted at 03:14 by the expected account. Receipt recorded."* The right rail row disappears.

**A degraded app.**
Left rail → Deployments → GitHub Triage card (now `◐ degraded · v2 · 5h ago · 1 UNKNOWN reconciled`). Click to open the timeline. The timeline shows the reconciliation event, and a health-check failure at 04:20 with the reason: *"1 reconciliation required in the last 24h — status downgraded by policy."* The user clicks **Rollback** in the middle pane header.

**Rollback.**
An inline confirmation appears: *"Rollback GitHub Triage to v1 (digest sha256:xyz9…, approved 3 days ago)? [Yes] [No]"*. The user clicks Yes. The middle pane shows a slim progress strip: *"Rolling back… 1/3 steps · transferring artifact → reinstalling → health check"*. When complete, the card updates to `● healthy · v1 · digest sha256:xyz9…` and a new event appears in the timeline: *"Rolled back to v1 by operator@example.com at 09:12. Reason: reconciliation policy."*

**Post-incident.**
The user clicks "View receipts" on the deployment card. A **receipts panel** expands in place, showing:
- The last 24 hours of actions (comment posted, issue labeled, comment reconciled)
- Each with a timestamp, the account that acted, the digest of the artifact that acted, and a link to the audit record
- A "Copy digest" button on each

The user pastes the digest into the incident report. Total time from detection to close: under 5 minutes, all in one window, no navigation, no tab-switching.

**A quiet morning.**
When nothing needs attention, Operate mode shows the fleet with green dots and the right rail is empty (just the "Health: ● all healthy" strip). The user gets a sense of "everything is fine" without having to read anything.

---

## 7. Visual design specifics

The visual identity has to support calm. Here's the palette and material choices.

**Color.** Neutral background (dark #0a0a0b or light #fafafa, following system preference but dark by default for the developer audience). One accent color per semantic role:

| Role | Color | Use |
|---|---|---|
| Attention | Amber (#f5a524) | Pulses, "needs you" borders, pending-approval cards |
| Success | Green (#3fb950) | Healthy dots, completed states, approved artifacts |
| Error | Red (#da3633) | Failed states, UNKNOWN effects, kill switch |
| Active | Blue (#539bf5) | In-flight progress, current-focus highlight |
| Muted | Gray (#6e7681) | Informational borders, secondary text, disclaimers |

The four semantic colors appear **only** for their semantic role. Green never appears as decoration. Red is never used for emphasis. Amber is never used for branding. The user learns the palette in a day and reads the system by color without thinking about it.

**Typography.** Two typefaces, disciplined:

- **Prose** — a humanist sans (Inter, IBM Plex Sans, or system UI) for conversation, card titles, descriptions.
- **Technical** — a monospace (JetBrains Mono, IBM Plex Mono, or system mono) for code, digests, IDs, JSON, MiniappSpec fragments, timestamps.

Monospace appears in a distinctive weight (regular, not bold), often with a subtle background tint (`#1a1a1d` on dark) to distinguish it from prose. This is the visual signal that says *"this is a technical value, don't edit it casually."*

**Density.** Compact in the rails (44–48px rows), generous in the middle pane (16px padding, 20px line-height, 12px between conversation events). The middle pane is where the user spends the most time, so it gets the most breathing room.

**No drop shadows. No rounded corners beyond 6px. No gradients. No illustrations.** Borders are 1px, in muted gray, used only where they carry information (separating the four zones, defining card boundaries, marking the left-border status indicator). Surfaces are flat. The design's weight comes from typography and whitespace, not from chrome.

**Motion.** Minimal, and only to convey state:

- New items in the right rail fade in (150ms), don't slide.
- Pulses are slow (2s period), subtle (opacity 0.6 → 1.0), and stop after a threshold (the user has seen it).
- Cards expand inline (250ms ease-out); nothing slides in from off-screen.
- Progress indicators animate smoothly; spinners don't exist (use a pulsing dot or a progress bar).
- No page-transition animations. Clicking a build in the left rail swaps the middle pane instantly.

**Icons.** Filled icons, 16px, used only on the left rail's three destinations. Everything else uses text labels. No icons for decoration, no emoji, no avatars except the workspace switcher.

---

## 8. What's deliberately absent

The design's strength is largely in what it *doesn't* have. To be explicit:

- **No top bar.** Context lives in the bottom bar's chip and the middle pane's header. No breadcrumbs. No global search bar (the command palette handles search).
- **No dashboard / landing page.** The first screen is contextual: Create mode's empty state, Operate mode's fleet, or the last-viewed item. There is no "home" to navigate away from.
- **No tabs.** The middle pane's content switches based on left-rail selection, and artifacts expand inline within the conversation. No tab bars inside cards either — sub-views (Diff, Spec, Tests, Transcript) are tabs *within the artifact card*, which is small and contained, not within the page.
- **No modal dialogs** for routine actions. Approve, reject, rollback, and reconcile all happen inline with a confirmation card. The only modal is the command palette, which is invoked deliberately.
- **No toast notifications.** The right rail is the notification surface. Items appear there and dismiss there.
- **No settings page prominence.** Settings live in the workspace switcher menu, accessible but not competing for attention.
- **No file tree.** The diff is the file tree; files are grouped by change type within the diff view.
- **No terminal in Studio.** The Builder Sprite has a terminal for developers who want one, but Studio is the business surface. The terminal is a Sprite capability, not a Studio feature.
- **No sidebar navigation tree.** Left rail is three items. There is no sub-navigation; depth lives in the middle pane.
- **No chat bubbles.** Conversation events are text with left-border status indicators, not rounded bubbles. This is important: bubbles imply a chat app; the left-border timeline implies a log with structure.
- **No loading spinners.** Skeleton states in the middle pane, subtle pulsing dots in the bottom bar during inference, progress bars during deployment. Never a spinner.
- **No "empty state illustrations."** Empty states are one sentence of guidance and one primary action.
- **No account avatar in the corner.** Workspace switcher is at the top of the left rail, and it shows the workspace name, not the user.

The cumulative effect is that **the user's attention always has exactly one place to go, and nothing competes for it.**

---

## 9. Shortcuts and command palette

Power users will live on the keyboard. The mapping:

| Shortcut | Action |
|---|---|
| `⌘K` | Command palette |
| `⌘1` | Focus left rail |
| `⌘2` | Focus middle pane |
| `⌘3` | Focus right rail |
| `⌘↵` | Submit current input |
| `⌘N` | New build |
| `⌘/` | Show keyboard shortcuts |
| `Esc` | Clear context chip, or close expanded card |
| `J` / `K` | Navigate conversation events (when middle pane is focused) |
| `Y` | Approve the currently-focused approval (when one is focused) |
| `N` | Reject the currently-focused approval |
| `?` | Help overlay |

The **command palette** (`⌘K`) is the escape hatch for anything the four zones don't surface. It's a single input that fuzzy-matches against:

- Builds (by name)
- Deployments (by name)
- Connectors (by provider name)
- Actions (`approve all`, `rollback <name>`, `new build`, `add custom connector`, `kill <name>`, `reconcile <effect-id>`)
- Settings (`appearance`, `account`, `workspace`)
- Help (`keyboard shortcuts`, `show me a tutorial`, `what is an approval?`)

The palette is how the design avoids needing a top-bar search, a settings page, a help modal, and a "fleet management" page. Those things exist — they're just accessed through one input.

---

## 10. Responsive behavior

The four zones map cleanly to narrow screens without any new UI:

**Desktop (≥1024px):** four zones as designed.

**Tablet (768–1024px):** left rail collapses to icon-only (~56px). Right rail becomes a horizontal strip across the top of the middle pane showing just counts (`Needs you: 2 · In flight: 1`); clicking expands it as an overlay.

**Phone (<768px):** this is where most products collapse into unusable. Miniapp Studio should **reduce to the essential loop**:

- Default view: **Operate mode** (fleet as a vertical list of cards)
- Bottom bar remains, taking full width
- Right rail items appear as a horizontal scrolling strip at the top of the screen: pill-shaped badges that, when tapped, navigate to the item
- Left rail becomes a bottom drawer icon (three lines) that slides up to show Builds / Deployments / Connectors
- The conversation view for a single build is fully readable on phone; artifacts expand full-width
- Approval cards render full-screen when tapped; the inline-then-expand pattern becomes tap-then-full-screen because the screen is small

The phone experience is not "the desktop experience but narrower." It is the **operate + approve** loop: fleet health at a glance, actionable items at the top, describe bar always at the bottom. Approving an artifact or reconciling an UNKNOWN from the phone should work as well as it does on desktop.

---

## 11. Tradeoffs and open questions

**Three honest tradeoffs** in this design:

**Conversation-as-workspace prioritizes continuity over parallelism.** If a user has 5 builds in flight and wants to compare their diffs side-by-side, the current design requires opening them one at a time. Mitigation: the fleet view can list *builds* as well as *deployments*, and each row can preview its diff state. But this is a compromise — side-by-side comparison is not the design's strength.

**No dashboard means no "at a glance" analytics.** A user wanting "how many builds did I run this week?" needs to either ask the describe bar ("how many builds this week?") or open the command palette. The design leans on the conversational interface to answer analytical questions, which works for occasional queries but is not a BI surface. If analytics become a real need, a fourth mode (Analytics) can be added to the left rail, but I'd defer it until usage data proves it's needed.

**Flat left rail puts pressure on naming.** With three items, "Builds / Deployments / Connectors" works. If the product grows to need "Agents", "Runtimes", "Audit", "Integrations" as separate destinations, the flatness becomes a liability. The design assumes the domain stays shallow; if it doesn't, the left rail becomes a tree, and the design's quietness suffers.

**Three open questions** worth resolving before implementation:

1. **Where does the build ↔ deployment relationship live in the UI?** A build produces an artifact; the artifact deploys to a Runtime. In Create mode, the deployment is a card in the conversation. In Operate mode, the deployment is a card in the fleet. Clicking a deployment in the fleet should let the user jump to the build that produced it; clicking the deployment card in the conversation should let the user jump to the fleet. This two-way navigation is essential but needs visual affordance design (a subtle "source build" link on the deployment card, and a "deployed as" link on the artifact card).

2. **How does the user set up a tenant / workspace the first time?** The workspace switcher is at the top of the left rail, but the first-run flow is not described. Options: a minimal setup modal on first visit (email, workspace name), or a deferred setup (use a default workspace, let the user rename later). I'd lean toward deferred setup — the user should be able to describe their first build before being asked for a workspace name.

3. **What is the visual language for "needs you" vs "needs you critically"?** The amber left-border covers both, but a question that's been open for 3 hours is different from a question that's been open for 3 minutes, and an UNKNOWN effect is different from a pending approval. The design needs at most one more signal (e.g., "critical" adds a subtle red tint to the amber, or doubles the pulse rate) — but adding too many signals defeats the quietness. This needs to be tested with real usage.

---

The design's job is to disappear. The user should be able to spend 30 minutes in Miniapp Studio and afterwards remember *what they built*, *what decisions they made*, and *what is running* — without remembering *the UI*. The four-zone shape with the three inventions above gets as close to that as I can see without user testing. The rest is refinement against real usage, which is the only thing that can settle the open questions.



## claude

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




## Z

Picking up from the recalibrated frame (Studio = scaffolding, real deployment = Fly.io Machines): I'll turn the reimagined UI into a concrete, interactive concept you can click through, not just describe it. Loading the web-dev skill first, then building.
The file is written. Now verifying the render — let me load the browser skill and snapshot desktop + mobile viewports:
Desktop renders exactly as designed. Now testing the scripted golden path — clicking the build chip, then approving the approval card:
The pipeline ran through the spine correctly, but the approval card is missing from the Pulse — let me check for a JS error:
Found the bug — `d.apr` is undefined somewhere in `pipeline`. Let me look at the exact line:
Line 432 is the approval `addEv` — let me inspect what `RUNDEMO['invoice-desk']` actually holds in the browser:
`apr` really is missing. Let me read the RUNDEMO block in the file to see what got written:
The file has `appr` at line 205, but the browser sees undefined — let me read the exact block:
File looks right — so let me see what keys the live object actually has:
Found it — a property-name mismatch: the data objects use `appr:{...}` but `pipeline()` reads `d.apr.t`. Three one-word fixes:
All three renamed. Now reload and re-run the full golden path end-to-end:
The stale error log is from before the reload — the eval confirms the approval card now exists (1 pending). Clearing errors, then approving:
Approve → receipt → evidence scene works. Checking the render visually:
Evidence scene is exactly as designed — full green spine, gates table, receipt trail in the Pulse. Now testing the refusal path and the DEV persona:
Refusal path works — spine Validate turns red, policy violations listed, refusal receipted in the Pulse. Checking DEV persona:
DEV persona works (hashes, rule ids, ledger detail lines all render) — one cosmetic bug: the approved receipt's dev line shows `[object Object]` (bad `dd` shape in `decide()`). Fixing:
Now the mobile check — 375px viewport plus a programmatic horizontal-overflow test:
Mobile verified — no horizontal overflow, Pulse opens as a slide-over sheet, DEV detail lines render correctly. Closing the browser and wrapping up:
The concept is built, clicked through, and verified — [miniapp-studio-focus-console.html](/home/z/my-project/download/miniapp-studio-focus-console.html), a single self-contained interactive page (works offline, no build step). I verified it in a real headless browser: the full golden path runs, the refusal path works, no JS errors, and no horizontal overflow at 375px.

## The concept: a state machine, not an app shell

The core idea that kills clutter: **the UI is the pipeline**. MiniApp Studio has exactly one lifecycle — Describe → Spec → Validate → Connect → Deploy → Run → Evidence — so the interface shows nothing that isn't a state of that lifecycle. There are no dashboards-of-dashboards, no file trees, no tabs-within-tabs. Four zones, each doing one job:

**Left rail — the essentials, as icons (64px, never wider).** Gallery, Connectors, Evidence, Settings, Pulse. Bottom: the **OPS/DEV persona toggle** and the **hold-to-confirm kill switch** (press-and-hold 700ms — a red fill confirms before it fires; no modal dialogs anywhere in the UI). Nothing else earns a place.

**Stage header — the spine (the inventive element).** The seven lifecycle phases as live pills: gray = untouched, amber-pulsing = in progress, green = *proven*, red = refused. This replaces breadcrumbs, progress bars, and status pages — position and color *are* the state, and every pill is clickable.

**Middle pane — one object at a time, morphing between seven scenes:** gallery → the generated spec card (plain-language sections; DEV mode exposes the raw JSON in a collapsed `<details>`) → the red/green fixture report with the adversarial case shown refused → the 3-state connector catalog with the **Custom API wizard** (Define → Test green → Scope default-deny, mirroring `custom_api` → `custom_api/test` → `AccessPolicy`) → deploy terminal ending in the persisted fly-issued URL → the live app preview → the **evidence ledger**: every gate G0–G12 with its state, and in DEV mode the exact proof (`ci: pin-gate PG19`, `probe P2 refused`, …). That last one is the thesis made visible: *the UI may only assert what a passing test proves.*

**Right sidebar — the Pulse.** One chronological stream instead of four scattered widgets: approvals (with expiry countdown and inline Approve/Reject), receipts, alerts, events. The UNKNOWN quarantine lives here visibly ("held — not retried"), which is your durability story made tangible without claiming a composer.

**Bottom bar — the Line.** Your chat bar, but not a chat *window*: one input line where submissions become **events in the Pulse, not chat bubbles**. The transcript never accumulates in the middle of the screen — that single decision removes ~60% of the clutter of every "chat-first builder" on the market.

## Devs and ops, one product

The OPS/DEV toggle is a *density* switch, not a second UI: ops reads "It will ask before it sends anything — approve in the Pulse," dev reads `sha256 9f3c…e2 · rule gov.approval.required · ledger PENDING→UNKNOWN · settle via SettleEffect`. Same data, same scenes, two languages — no split-brain navigation.

**To explore it:** click a Build chip (or type "build an invoice review app"), watch the spine light up, approve the card in the Pulse, and land on Evidence. Then toggle DEV and re-look at the same screen. On a phone, the Pulse slides over from the right edge.