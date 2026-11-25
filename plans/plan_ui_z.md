## prompt

read plan_ui.html reimagine the UI of Miniapp Studio that covers the critical and essential UI elements and workflow, how do you envision this for optimal experience for both devs and ops? the goal is to provide a minimal clutter-free UI, I do not want the standard UI of today, here is what I want: a minimal left sidebar but display only the essentials, bottom bar for chat, right sidebar to display crucial UI information like activity, notifications but feel free to suggest and a middle pane that displays what needs to be shown (again feel free to suggest more inventive modern-looking UI without the clutter

the goal is to flesh out the implementation details based on that single page HTML, that is the UI I want to use 

## answer

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