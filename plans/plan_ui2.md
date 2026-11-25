## prompt

read miniapp-studio-ui.html, reimagine the UI of Miniapp Studio that covers the critical and essential UI elements and workflow, how do you envision this for optimal experience for both devs and ops? the goal is to provide a minimal clutter-free UI, I do not want the standard UI of today, here is what I want: a minimal left sidebar but display only the essentials, bottom bar for chat, right sidebar to display crucial UI information like activity, notifications but feel free to suggest and a middle pane that displays what needs to be shown (again feel free to suggest more inventive modern-looking UI without the clutter

the goal is to flesh out the implementation details based on that single page HTML, that is the UI I want to use 

## gemini

The attached `miniapp-studio-ui.html` already nails a high-density, low-noise **Focus Console** layout (`64px` left rail, central pipeline + stage + bottom command line, and a right-hand **Pulse** stream) with dual `OPS` and `DEV` personas, approval gates, and a hold-to-halt kill switch. To evolve this single-file prototype into a production-ready architecture for both developers and operators without adding visual clutter, the key is **progressive disclosure** inside each zone, **structured command-line affordances** at the bottom, **actionable triage** in Pulse, and **contextual artifact inspection** in the middle stage.

---

### 1. Spatial Architecture & Inventive Refinements

Keep the three-column desktop grid (`64px` rail, flexible center stage, `280px–340px` right Pulse pane) and responsive drawer fallback, while sharpening what each zone owns:

* **Left Rail (Essential Anchors + Safety):** Keep top navigation strictly to core anchors (`Gallery`, `Connectors`, `Evidence`, `Settings`, and `Pulse` with its pending count badge), and anchor the bottom with the `OPS` / `DEV` toggle and the 700 ms hold-to-stop kill switch. Add a subtle hover/focus tooltip showing keyboard shortcuts (`1–5` for views, `/` to focus the bottom command line) without crowding the icon strip.


* **Middle Stage (Pipeline Spine + Context Bar + Active Scene):** Retain the top horizontal phase spine (`Describe → Spec → Validate → Connect → Deploy → Run → Evidence`) and metadata bar, which already communicate execution state (`idle`, `active`, `proven`, `refused`) cleanly. Upgrade the scene body with a **split or tabbed inspector toggle** when in `DEV` mode so developers can compare `MiniAppSpec` diffs, step receipts, and policy traces side-by-side without forcing `OPS` users through JSON or raw protocol details.


* **Bottom Bar (Intent Line + Context Chips):** Upgrade the single-line input into a **smart intent bar** that preserves the prompt chips above it while supporting slash commands (`/spec`, `/validate`, `/deploy`, `/halt`, `/settle`) and contextual attachment pills (such as pinning the active app or failing receipt) when needed.


* **Right Sidebar — Pulse (Triage & Audit Feed):** Keep `All`, `Approvals`, `Receipts`, `Alerts`, and `Events` filters, and pin unresolved approvals at the top with live expiration countdowns. Add inline **diff/scope previews** before `Approve` or `Reject` (for example, showing exact recipient count, dollar delta, or reversible vs. irreversible effect tags) and a direct **Jump to Stage / Inspect Receipt** action so operators and developers can resolve anomalies in one click.



---

### 2. Dual-Persona Workflow: Devs vs. Ops

The `OPS` and `DEV` personas should share the exact same underlying state and receipts while adapting vocabulary, density, and controls:

| Workflow Stage | Operators (`OPS`) Experience | Developers (`DEV`) Experience |
| --- | --- | --- |
| **Describe & Plan** | Plain-language intent, high-level capabilities, and plain summary of data touched and approvals required.

 | Content-hashed `MiniAppSpec` version/hash binding, workflow graph steps, capability effect classes, and fixture counts.

 |
| **Validate & Policy** | Clear pass/fail checks and plain explanation when a policy blocks an unsafe or out-of-scope action.

 | Deterministic validation rules, policy rule IDs (`gov.approval.required`), probe classifications, and refusal audit IDs.

 |
| **Connectors & Deploy** | Clear readiness states (`Allowed`, `Listed`, `Not configured`) and guided custom API setup with least-privilege defaults.

 | Three-state truth (`catalog` vs. `org-configured` vs. `per-Sprite authorization`), endpoint/path allow-deny rules, Fly/Postgres/Sprite provisioning logs, and gateway/HMAC metadata.

 |
| **Run & Settle** | Live summary metrics, status badges, countdown approvals, and plain receipts (`sent once`, `nothing sent`, `held safely`).

 | Step-by-step execution timeline, latency, untrusted LLM output markers, idempotency keys, and explicit `UNKNOWN` effect settlement.

 |

---

### 3. Implementation Blueprint from the Single-Page Prototype

To move cleanly from the inline script in `miniapp-studio-ui.html` to a maintainable production codebase without losing single-file speed or zero-clutter discipline:

1. **Decouple State from DOM Rendering:** Extract `state`, `PHASES`, `RUNDEMO`, and event reducers into a dedicated state store with explicit transitions (`BUILD_STARTED`, `PHASE_UPDATED`, `APPROVAL_DECIDED`, `KILL_ENGAGED`, `PERSONA_CHANGED`). Drive `renderSpine()`, `renderMeta()`, `renderStream()`, and `SCENES[state.scene]()` from immutable or versioned state snapshots so async pipeline steps never race with manual tab navigation or kill-switch holds.
2. **Unify Event & Approval Models:** In the prototype, `addEv`, `decide`, and `SCENES.run` inspect `state.events` directly. Normalize approvals into first-class records linked to `runId`, `stepId`, `effectClass` (`reversible` | `irreversible` | `unknown`), `expiresAt`, and `idempotencyKey`, while projecting activity entries into the `Pulse` stream.


3. **Make Pipeline & Run Simulation Pluggable:** Wrap the scripted `pipeline()`, `refuse()`, and `RUNDEMO` fixtures behind a transport interface (`MockTransport` for offline/demo mode and `StreamTransport` via Server-Sent Events or WebSockets for live backend runs). Keep the `700ms` pointer-hold kill switch and `1000ms` approval countdown timer cleanly unmounted or paused when hidden or when `prefers-reduced-motion` is active.


4. **Harden Accessibility & Responsive Behavior:** Preserve keyboard focus rings, `aria-live="polite"` on the stage, `Escape` closing the mobile `Pulse` drawer, and pointer/keyboard support (`onPointerDown`, `onKeyDown` with `Enter`/`Space`) on the kill switch so keyboard and touch users can trigger or cancel holds reliably.


## prompt

Extend miniapp-studio-ui.html into a clean state-driven implementation with richer DEV/OPS views, approval diff details, and a pluggable mock/live event adapter.


## answer

```python
import os

# Let's inspect what files exist in the current working directory to see if miniapp-studio-ui.html is mounted
print("CWD:", os.getcwd())
print("Files:", os.listdir("."))
if os.path.exists("/mnt/data"):
    print("/mnt/data:", os.listdir("/mnt/data"))


```

```text
CWD: /home/bard
Files: ['miniapp-studio-ui.html', 'adv_review_prompt10.md', 'README.md', 'writeup4.md', 'writeup2.md', 'miniapp-studio-implementation-plan.md']
/mnt/data: ['miniapp-studio-ui.html', 'adv_review_prompt10.md', 'README.md', 'writeup4.md', 'writeup2.md', 'miniapp-studio-implementation-plan.md']


```

```python
with open("miniapp-studio-ui.html", "r", encoding="utf-8") as f:
    html_src = f.read()

print("Length:", len(html_src))
print(html_src[:12000])


```

```text
Length: 41492
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>MiniApp Studio — Focus Console</title>
<style>
:root{
  --bg:#0e1113; --panel:#14171a; --panel2:#1a1e22; --line:#252b31; --line2:#39424b;
  --txt:#e7ebee; --mut:#98a2ad; --dim:#6c7681;
  --ok:#4cc38a; --warn:#e0a63f; --bad:#e5484d; --acc:#7aa2f7;
  --mono:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;
}
*{box-sizing:border-box}
html,body{margin:0;height:100%}
body{background:var(--bg);color:var(--txt);font:15px/1.55 ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;-webkit-font-smoothing:antialiased}
.app{display:grid;grid-template-columns:64px minmax(0,1fr) clamp(280px,23vw,340px);height:100dvh}
:focus-visible{outline:1.5px solid var(--acc);outline-offset:1px}
::-webkit-scrollbar{width:10px;height:10px}
::-webkit-scrollbar-thumb{background:#252b31;border-radius:6px}
button{font:inherit}

/* ---------- left rail: the essentials ---------- */
.rail{border-right:1px solid var(--line);background:var(--panel);display:flex;flex-direction:column;justify-content:space-between;align-items:center;padding:10px 0}
.rail-top,.rail-bottom{display:flex;flex-direction:column;gap:6px;align-items:center}
.brand{color:var(--acc);font-size:17px;margin-bottom:6px;user-select:none}
.rbtn{position:relative;width:40px;height:40px;border-radius:10px;border:1px solid transparent;background:none;color:var(--mut);display:grid;place-items:center;cursor:pointer}
.rbtn:hover{background:var(--panel2);color:var(--txt)}
.rbtn.on{color:var(--txt);border-color:var(--line);background:var(--panel2)}
.rbtn svg{width:18px;height:18px}
.rbadge{position:absolute;top:1px;right:1px;background:var(--warn);color:#141414;font-size:10px;font-weight:700;min-width:16px;height:16px;border-radius:8px;display:grid;place-items:center;padding:0 3px}
.pbtn{width:40px;height:36px;border-radius:8px;border:1px solid var(--line);background:none;color:var(--mut);font-size:9px;font-weight:700;letter-spacing:.5px;cursor:pointer}
.pbtn.on{background:var(--panel2);color:var(--txt);border-color:var(--line2)}
.kill{position:relative;overflow:hidden;width:40px;height:40px;border-radius:10px;border:1px solid #4a2327;background:none;color:var(--bad);cursor:pointer}
.kill .fill{position:absolute;left:0;top:0;bottom:0;width:0;background:rgba(229,72,77,.35)}
.kill svg{position:relative;z-index:1;width:16px;height:16px}

/* ---------- stage ---------- */
.stage{display:flex;flex-direction:column;min-width:0}
.stagehead{border-bottom:1px solid var(--line);background:var(--panel);padding:10px 14px 0}
.spine{display:flex;gap:4px;overflow-x:auto;scrollbar-width:none;padding-bottom:10px}
.spine::-webkit-scrollbar{display:none}
.sph{display:flex;align-items:center;gap:6px;padding:5px 10px;border-radius:999px;border:1px solid var(--line);color:var(--mut);font-size:12px;white-space:nowrap;cursor:pointer;background:none;flex:0 0 auto}
.sph .dot{width:7px;height:7px;border-radius:50%;background:#3a4249;flex:0 0 auto}
.sph[data-s="active"]{color:var(--txt);border-color:#5a4a26}
.sph[data-s="active"] .dot{background:var(--warn);box-shadow:0 0 0 3px rgba(224,166,63,.16);animation:pulse 1.4s infinite}
.sph[data-s="proven"]{color:#bfe9d4;border-color:#274937}
.sph[data-s="proven"] .dot{background:var(--ok)}
.sph[data-s="refused"]{color:#f3b9bc;border-color:#54262a}
.sph[data-s="refused"] .dot{background:var(--bad)}
@keyframes pulse{50%{box-shadow:0 0 0 6px rgba(224,166,63,.07)}}
.stagemeta{display:flex;justify-content:space-between;align-items:center;gap:8px;padding-bottom:10px;color:var(--mut);font-size:12px;flex-wrap:wrap}
.meta-l,.meta-r{display:flex;align-items:center;gap:8px;flex-wrap:wrap;min-width:0}
.stagebody{flex:1;overflow:auto;padding:clamp(14px,2.5vw,28px)}
.scene{max-width:980px;margin:0 auto}
h1.title{font-size:clamp(1.25rem,2.4vw,1.6rem);margin:0 0 6px;font-weight:650;letter-spacing:-.01em}
.sub{color:var(--mut);margin:0 0 18px;max-width:62ch}
.card{background:var(--panel);border:1px solid var(--line);border-radius:12px;padding:clamp(14px,2vw,20px);margin-bottom:14px}
.card.good{border-color:#274937}
.card.bad{border-color:#54262a}
.card h3{margin:0 0 4px;font-size:15px}
.grid3{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:12px}
.metric{background:var(--panel);border:1px solid var(--line);border-radius:12px;padding:14px 16px}
.metric b{font-size:clamp(1.4rem,3vw,1.9rem);font-weight:650;display:block}
.metric span{color:var(--mut);font-size:12.5px}
.chip{display:inline-flex;align-items:center;gap:6px;padding:5px 11px;border-radius:999px;border:1px solid var(--line);background:none;color:var(--mut);font-size:12.5px;cursor:pointer}
.chip:hover{color:var(--txt);border-color:var(--line2)}
.chips{display:flex;flex-wrap:wrap;gap:8px;margin-bottom:10px}
.chips:empty{display:none}
.tag{display:inline-flex;align-items:center;gap:5px;padding:2px 9px;border-radius:999px;font-size:11px;font-weight:600;border:1px solid;white-space:nowrap}
.tag.ok{color:#9fe3bf;border-color:#274937;background:rgba(76,195,138,.08)}
.tag.warn{color:#f0cf8f;border-color:#5a4a26;background:rgba(224,166,63,.08)}
.tag.bad{color:#f3b9bc;border-color:#54262a;background:rgba(229,72,77,.08)}
.tag.mut{color:var(--mut);border-color:var(--line)}
.mono{font-family:var(--mono);font-size:12px;color:#c7d1da}
pre.code{background:#101417;border:1px solid var(--line);border-radius:8px;padding:12px;overflow-x:auto;font-family:var(--mono);font-size:12px;line-height:1.55;color:#c7d1da;margin:10px 0 0}
.scroll{overflow-x:auto;border:1px solid var(--line);border-radius:10px;background:var(--panel)}
table{width:100%;border-collapse:collapse;font-size:13.5px}
th{text-align:left;color:var(--mut);font-weight:600;font-size:11.5px;padding:8px 10px;border-bottom:1px solid var(--line);white-space:nowrap}
td{padding:9px 10px;border-bottom:1px solid #1d2226;vertical-align:top}
tr:last-child td{border-bottom:none}
.btn{display:inline-flex;align-items:center;gap:8px;padding:8px 14px;border-radius:9px;border:1px solid var(--line);background:var(--panel2);color:var(--txt);cursor:pointer;font-size:13.5px;font-weight:600}
.btn.primary{background:var(--ok);border-color:var(--ok);color:#0b100d}
.btn.danger{background:none;border-color:#54262a;color:var(--bad)}
.btn:disabled{opacity:.45;cursor:default}
.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:12px}
details{border:1px solid var(--line);border-radius:8px;padding:8px 12px;margin-top:10px}
summary{cursor:pointer;color:var(--mut);font-size:12.5px}
.kv{display:grid;grid-template-columns:minmax(120px,180px) 1fr;gap:8px 14px;font-size:13.5px}
.kv dt{color:var(--mut)}.kv dd{margin:0}
.note{color:var(--dim);font-size:12px;margin-top:12px}
.checkrow{display:flex;gap:8px;align-items:baseline;padding:4px 0;font-size:13.5px}
.checkrow .c{color:var(--ok);font-family:var(--mono)}
.steps{list-style:none;margin:0;padding:0}
.steps li{display:flex;gap:10px;padding:6px 0;border-bottom:1px solid #1d2226;font-size:13.5px;align-items:baseline}
.steps li:last-child{border-bottom:none}
.steps .n{font-family:var(--mono);color:var(--dim);font-size:12px;flex:0 0 34px}

/* ---------- bottom line ---------- */
.line{border-top:1px solid var(--line);background:var(--panel);padding:10px 14px 12px}
.lineform{display:flex;gap:8px}
.lineform input{flex:1;min-width:0;background:var(--panel2);border:1px solid var(--line);border-radius:10px;color:var(--txt);padding:11px 14px;font:inherit}
.lineform input::placeholder{color:var(--dim)}
.lineform input:focus{outline:none;border-color:var(--acc)}
.lineform button{width:46px;border-radius:10px;border:1px solid var(--line);background:var(--panel2);color:var(--txt);font-size:16px;cursor:pointer}

/* ---------- pulse (right) ---------- */
.pulse{border-left:1px solid var(--line);background:var(--panel);display:flex;flex-direction:column;min-width:0}
.pulse-head{padding:12px 14px 10px;border-bottom:1px solid var(--line);display:flex;flex-direction:column;gap:8px}
.pulse-head h2{margin:0;font-size:12px;letter-spacing:.1em;text-transform:uppercase;color:var(--mut);display:flex;justify-content:space-between;align-items:center}
.pfilters{display:flex;gap:6px;flex-wrap:wrap}
.pf{padding:3px 9px;border-radius:999px;border:1px solid var(--line);background:none;color:var(--mut);font-size:11.5px;cursor:pointer}
.pf.on{color:var(--txt);background:var(--panel2);border-color:var(--line2)}
.stream{flex:1;overflow-y:auto;padding:10px 12px;display:flex;flex-direction:column;gap:8px}
.ev{border:1px solid var(--line);border-left-width:3px;border-radius:8px;background:var(--panel2);padding:9px 11px;font-size:13px}
.ev.event{border-left-color:#3a4249}
.ev.receipt{border-left-color:var(--ok)}
.ev.alert{border-left-color:var(--bad)}
.ev.approval{border-left-color:var(--warn)}
.ev .t{font-weight:600;margin-bottom:2px}
.ev .d{color:var(--mut);font-size:12.5px;overflow-wrap:anywhere}
.ev .devd{color:var(--dim);font-size:11.5px;font-family:var(--mono);margin-top:3px}
.ev .tm{color:var(--dim);font-size:11px;font-family:var(--mono)}
.ev .row{display:flex;gap:8px;margin-top:8px;flex-wrap:wrap}
.ev .row .btn{padding:5px 12px;font-size:12.5px}
.count{font-family:var(--mono);color:var(--warn);font-size:12px}
.pulse-edge{display:none;position:fixed;right:10px;bottom:86px;z-index:55;padding:8px 13px;border-radius:999px;border:1px solid var(--line2);background:var(--panel2);color:var(--txt);font-size:12px;font-weight:600;cursor:pointer;align-items:center;gap:7px;box-shadow:0 6px 24px rgba(0,0,0,.4)}
.overlay{display:none;position:fixed;inset:0;background:rgba(0,0,0,.5);z-index:58}

/* ---------- responsive ---------- */
@media(max-width:900px){
  .app{grid-template-columns:56px minmax(0,1fr)}
  .pulse{position:fixed;top:0;right:0;bottom:0;width:min(340px,88vw);transform:translateX(105%);transition:transform .22s ease;z-index:60;box-shadow:-12px 0 40px rgba(0,0,0,.45)}
  .pulse.open{transform:none}
  .pulse-edge{display:flex}
  .overlay.on{display:block}
}
@media(prefers-reduced-motion:reduce){*{animation:none!important;transition:none!important}}
</style>
</head>
<body>
<div class="app">
  <aside class="rail" aria-label="Essentials">
    <div class="rail-top">
      <div class="brand" title="MiniApp Studio">◆</div>
      <button class="rbtn" data-nav="gallery" title="Gallery" aria-label="Gallery"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/></svg></button>
      <button class="rbtn" data-nav="connect" title="Connectors" aria-label="Connectors"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M9 7V3M15 7V3M7 7h10v4a5 5 0 0 1-10 0V7z"/><path d="M12 16v5"/></svg></button>
      <button class="rbtn" data-nav="evidence" title="Evidence" aria-label="Evidence"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M12 3l7 3v5c0 5-3.5 8-7 10-3.5-2-7-5-7-10V6l7-3z"/><path d="M9 11.5l2 2 4-4"/></svg></button>
      <button class="rbtn" data-nav="settings" title="Settings" aria-label="Settings"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M19.1 4.9L17 7M7 17l-2.1 2.1"/></svg></button>
      <button class="rbtn" id="pulseBtn" title="Pulse" aria-label="Pulse"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M3 12h4l3-8 4 16 3-8h4"/></svg><span class="rbadge" id="railBadge" hidden>0</span></button>
    </div>
    <div class="rail-bottom">
      <div role="group" aria-label="Persona">
        <button class="pbtn on" id="pOps" title="Operate — plain language">OPS</button>
        <button class="pbtn" id="pDev" title="Inspect — full detail">D


```

```python
print(html_src[12000:])


```

```text
EV</button>
      </div>
      <button class="kill" id="kill" title="Hold to stop all runs" aria-label="Hold to stop all runs"><span class="fill" id="killFill"></span><svg viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="6" width="12" height="12" rx="2"/></svg></button>
    </div>
  </aside>

  <main class="stage">
    <header class="stagehead">
      <div class="spine" id="spine" aria-label="Pipeline"></div>
      <div class="stagemeta" id="stageMeta"></div>
    </header>
    <section class="stagebody" id="stage" aria-live="polite"></section>
    <footer class="line">
      <div class="chips" id="chips"></div>
      <form class="lineform" id="lineForm">
        <input id="lineInput" placeholder="Describe what you want — the studio builds, validates and asks before it acts" aria-label="Describe what you want">
        <button type="submit" aria-label="Send">→</button>
      </form>
    </footer>
  </main>

  <aside class="pulse" id="pulse" aria-label="Pulse">
    <div class="pulse-head">
      <h2>Pulse <span class="mono" id="pulseCount"></span></h2>
      <div class="pfilters" id="pfilters"></div>
    </div>
    <div class="stream" id="stream"></div>
  </aside>
</div>
<button class="pulse-edge" id="pulseEdge">Pulse <span class="rbadge" id="edgeBadge" style="position:static" hidden>0</span></button>
<div class="overlay" id="overlay"></div>

<script>
'use strict';
const $=s=>document.querySelector(s);
const esc=s=>String(s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const sleep=ms=>new Promise(r=>setTimeout(r,ms));
const now=()=>new Date().toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'});
const dev=()=>state.persona==='dev';

const PHASES=[['describe','Describe'],['spec','Spec'],['validate','Validate'],['connect','Connect'],['deploy','Deploy'],['run','Run'],['evidence','Evidence']];
const state={persona:'ops',scene:'gallery',busy:false,filter:'all',ph:{},app:null,url:null,events:[],apps:[],wiz:0,dlines:0};
PHASES.forEach(([k])=>state.ph[k]='idle');

/* ---------- run-preview content per template ---------- */
const RUNDEMO={
 'invoice-desk':{metrics:[['24','Invoices this week'],['21','Matched to POs'],['1','Awaiting your approval']],
  head:['Invoice','Customer','Amount','PO match','Status'],
  rows:[['INV-1042','Acme Corp','$420.50','✗ $402.00',['warn','Awaiting approval']],['INV-1041','Northwind','$1,180.00','✓',['ok','Matched']],['INV-1040','Globex','$86.10','✓',['ok','Matched']],['INV-1039','Initech','$2,300.00','✓',['ok','Matched']]],
  act:'Send reminder for INV-1042',
  apr:{t:'reminder send · INV-1042 Acme Corp',b:'Email to ap@acme.example — amount mismatch $420.50 vs PO $402.00. Effect: irreversible · outbound. Sends once, reconciled by read.',
        d:'intent b7c1…44 · effect irreversible · idempotency inv-1042-remind',receipt:'Confirmed · rcpt_9f3c01 — sent once, reconciled by read. No duplicate.'}},
 'box-digest':{metrics:[['6','Docs in Weekly Reports'],['1','Digest drafted'],['1','Awaiting your approval']],
  head:['Doc','Summary','Status'],rows:[['Q3-revenue.xlsx','Revenue up 12% QoQ; churn flat',['ok','Ready']],['incidents.md','2 sev-2 incidents, both closed',['ok','Ready']],['hiring.csv','3 offers out, 1 accepted',['ok','Ready']]],
  act:'Post digest to #team',
  apr:{t:'post digest to #team',b:'Posts the Friday summary to Slack #team. Effect: reversible (message can be deleted). Sends once per week.',
        d:'effect reversible · undo slack.delete_message · idempotency digest-w45',receipt:'Confirmed · rcpt_71b0 — posted once.'}},
 'gh-triage':{metrics:[['18','Open issues'],['15','Triaged'],['1','Awaiting your approval']],
  head:['Issue','Label','Confidence','Status'],rows:[['#482 login loop','bug · auth','0.94',['warn','Awaiting approval']],['#479 dark-mode','enhancement','0.91',['ok','Labeled']],['#477 flaky test','chore · ci','0.88',['ok','Labeled']]],
  act:'Apply triage for #482',
  apr:{t:'label & assign #482',b:'Applies bug · auth and assigns @maya — based on model output (untrusted, confidence 0.94). Effect: reversible.',
        d:'llm output untrusted · effect reversible · branch bounded',receipt:'Confirmed · rcpt_c4d2 — labeled once.'}}
};
RUNDEMO['default']=RUNDEMO['invoice-desk'];

/* ---------- seed ---------- */
addEv('event','Session started','Studio kernel ready. Builder gateway v1 · framework pinned and CI-verified.',{d:'pin 73351a8 · go test -race green · PG19 gate'});
addEv('event','Connector catalog synced','6 listed · 2 allowed · 1 not configured. Availability is per-organization.');
addEv('alert','UNKNOWN held — not retried','Outbound webhook rcpt_9f21: provider timeout after side effect. Held for your decision — nothing will retry blindly.',{d:'ledger PENDING→UNKNOWN · settle via SettleEffect'});

/* ---------- event/pulse ---------- */
function addEv(kind,title,detail,dd,apr){
  const e={id:'e'+Math.random().toString(16).slice(2,8),kind,title,detail,dd:dd?(dd.d||dd):null,tm:now(),apr:apr||null};
  state.events.unshift(e); renderStream(); updateBadges(); return e;
}
function pendingCount(){return state.events.filter(e=>e.kind==='approval'&&e.apr&&!e.apr.decided).length}
function updateBadges(){const n=pendingCount();$('#railBadge').hidden=!n;$('#railBadge').textContent=n;$('#edgeBadge').hidden=!n;$('#edgeBadge').textContent=n;$('#pulseCount').textContent=state.events.length+' items'}
function renderStream(){
  const F={all:()=>true,approval:e=>e.kind==='approval',receipt:e=>e.kind==='receipt',alert:e=>e.kind==='alert',event:e=>e.kind==='event'};
  const list=state.events.filter(F[state.filter]);
  list.sort((a,b)=>(b.apr&&!b.apr.decided?1:0)-(a.apr&&!a.apr.decided?1:0));
  $('#stream').innerHTML=list.map(e=>{
    let aprHtml='';
    if(e.apr){
      if(!e.apr.decided)aprHtml=`<div class="row"><span class="count" data-count="${e.apr.exp}">expires…</span><button class="btn primary" data-ev="${e.id}" data-val="approved">Approve</button><button class="btn danger" data-ev="${e.id}" data-val="rejected">Reject</button></div>`;
      else aprHtml=`<div class="row"><span class="tag ${e.apr.decided==='approved'?'ok':'mut'}">${e.apr.decided==='approved'?'Approved':'Rejected'}</span></div>`;
    }
    return `<div class="ev ${e.kind}"><div class="t">${esc(e.title)}</div><div class="d">${esc(e.detail)}</div>${dev()&&e.dd?`<div class="devd">${esc(e.dd)}</div>`:''}${aprHtml}<div class="tm" style="margin-top:6px">${e.tm}</div></div>`;
  }).join('')||'<div class="note">Nothing here yet. Describe something below.</div>';
  tickCounts();
}
function tickCounts(){
  document.querySelectorAll('[data-count]').forEach(el=>{
    const ms=+el.getAttribute('data-count')-Date.now();
    if(ms<=0){el.textContent='expired';return}
    const s=Math.floor(ms/1000);el.textContent='expires '+Math.floor(s/60)+':'+String(s%60).padStart(2,'0');
  });
}
setInterval(tickCounts,1000);
function decide(id,val){
  const e=state.events.find(x=>x.id===id);if(!e||!e.apr||e.apr.decided)return;
  e.apr.decided=val;
  if(val==='approved'){
    state.ph.run='proven';state.ph.evidence='proven';
    if(state.app)state.app.status='running';
    addEv('receipt','Receipt recorded',e.apr.receiptOverride||e.apr.receipt||'Confirmed — sent once, reconciled by read.',{d:'ledger ATTEMPTING→CONFIRMED · idempotency key held'});
    render();setTimeout(()=>go('evidence'),900);
  }else{
    addEv('receipt','Rejected — nothing sent','Operator declined the action. The refusal is receipted in the audit trail.');
    render();
  }
}

/* ---------- spine / meta / nav ---------- */
function renderSpine(){$('#spine').innerHTML=PHASES.map(([k,l])=>`<button class="sph" data-phase="${k}" data-s="${state.ph[k]}" title="${l}"><span class="dot"></span>${l}</button>`).join('')}
function renderMeta(){
  const l=state.app?`<b style="color:var(--txt)">${esc(state.app.name)}</b> <span class="mono">v${state.app.ver} · ${dev()?state.app.hash:'verified build'}</span>`:'No app in focus';
  const r=`<span class="tag mut">${dev()?'Fly Machines · MPG · pinned miniapp':'Fly Machines · secure'}</span>${state.url?`<span class="tag ok mono">${esc(state.url)}</span>`:''}`;
  $('#stageMeta').innerHTML=`<div class="meta-l">${l}</div><div class="meta-r">${r}</div>`;
}
function renderNav(){
  const map={gallery:'gallery',spec:'gallery',validate:'gallery',refused:'gallery',run:'gallery',deploy:'gallery',connect:'connect',evidence:'evidence',settings:'settings'};
  document.querySelectorAll('[data-nav]').forEach(b=>b.classList.toggle('on',map[state.scene]===b.dataset.nav));
}
function go(s){state.scene=s;render()}

/* ---------- scenes ---------- */
function tag(t){return `<span class="tag ${t[0]}">${esc(t[1])}</span>`}
const SCENES={
 gallery(){
  const cards=[['Invoice Review Desk','Watch invoices, match them to POs, remind you to pay — asks first.','Box · approval'],['Friday Box Digest','Summarize a Box folder every Friday and post it to Slack — with approval.','Box · Slack'],['GitHub Issue Triage','Label and route incoming issues; model output stays untrusted until you confirm.','GitHub']];
  const recent=state.apps.length?`<div class="card"><h3>Recent apps</h3>${state.apps.map(a=>`<div class="steps"><li><span class="n">▸</span>${esc(a.name)} ${a.status==='running'?'<span class="tag ok">running</span>':'<span class="tag warn">awaiting approval</span>'}</li></div>`).join('')}</div>`:'';
  return `<div class="scene"><h1 class="title">What should we build?</h1>
  <p class="sub">${dev()?'Describe intent below — the generator emits a versioned, content-hashed MiniAppSpec; one statically registered runner interprets it; every consequential step is checkpointed and receipted.':'Describe an app in plain language. The studio builds it from governed parts: it shows you the plan, proves it against tests, and asks before anything consequential. Every action leaves a receipt.'}</p>
  <div class="grid3">${cards.map(c=>`<div class="card"><h3>${c[0]}</h3><p style="color:var(--mut);font-size:13.5px;margin:4px 0 10px">${c[1]}</p><span class="tag mut">${c[2]}</span></div>`).join('')}</div>${recent}
  <div class="note">${dev()?'Try the refusal path too — the validator must reject an unapproved bulk send, on camera.':'Tip: try asking it to do something unsafe — watch it refuse.'}</div></div>`;
 },
 spec(){
  const a=state.app;if(!a)return SCENES.gallery();
  const rows=[['Data','entities invoice · purchase_order · reminder'],['UI','pages dashboard · invoice-detail'],['Workflow','detect → save → match → remind (4 steps)'],['Capabilities','box.list_folder (read) · box.save (write · reversible) · reminder.send (write · approval required)'],['Policies','max 1 reminder / invoice / day · scope assigned_accounts · effect classes declared'],['Fixtures','12 required, all must pass']];
  return `<div class="scene"><h1 class="title">The plan for ${esc(a.name)}</h1>
  <p class="sub">${dev()?'MiniAppSpec — immutable, content-hashed; a run binds the version it started with. Editing creates a new version.':'This is exactly what the studio will build — nothing more. Check it, then validate.'}</p>
  <div class="card"><div class="kv">${rows.map(r=>`<dt>${r[0]}</dt><dd>${esc(r[1])}</dd>`).join('')}</div>
  ${dev()?`<details><summary>Raw MiniAppSpec (JSON)</summary><pre class="code">{
  "app": { "id": "${a.slug}", "version": 1, "hash": "sha256:9f3c…e2" },
  "workflows": { "remind": { "steps": ["detect","box.save","po.match","approve","reminder.send"] } },
  "policies": { "reminder.send": { "approval": "required", "limit": "1/day/invoice" } },
  "fixtures": { "count": 12 }
}</pre></details>`:''}
  <div class="actions"><button class="btn primary" data-act="toValidate">Run validation</button></div></div></div>`;
 },
 validate(){
  const fx=['detect invoice PDF in mailbox','save copy to Box (reversible)','match to purchase order by amount+vendor','duplicate submit attaches, never double-runs','reminder requires approval','scope limited to assigned accounts'];
  return `<div class="scene"><h1 class="title">Validation</h1>
  <p class="sub">${dev()?'Deterministic, LLM-independent: schema, references, allowlists, expression bounds, fixtures dry-run. Unknown → reject.':'Twelve tests must pass before anything deploys. They run the same way every time — no guesswork.'}</p>
  <div class="card good"><h3>Fixtures — 12 / 12 passed <span class="tag ok" style="margin-left:6px">green</span></h3>
  <div style="margin-top:8px">${fx.map(f=>`<div class="checkrow"><span class="c">✓</span>${esc(f)}</div>`).join('')}</div></div>
  <div class="card bad"><h3>Adversarial case — refused <span class="tag bad" style="margin-left:6px">by design</span></h3>
  <p style="color:var(--mut);font-size:13.5px;margin:6px 0 0">“Send reminders to all overdue customers now, without approval.” → rejected: external writes require explicit approval; recipient set outside declared scope.</p>
  ${dev()?'<div class="devd" style="color:var(--dim);font-size:11.5px;font-family:var(--mono);margin-top:6px">rule gov.approval.required · probe class P4 · refusal receipted</div>':''}</div>
  <div class="actions"><button class="btn primary" data-act="toConnect">Continue — connect data</button></div></div>`;
 },
 connect(){
  const rows=[
   ['Box','Files','allowed',dev()?'labels: builds · paths /folders* (read-only core)':'read-only core'],
   ['Custom API — yours','Bring your own',state.wiz>=3?'allowed':'wizard',dev()&&state.wiz>=3?'paths /v1/invoices* · deny /admin*':'any REST API, gateway-held key'],
   ['Fly.io Machines','Control plane','allowed',dev()?'least-privilege adapter · owner-held org token':'managed by the studio'],
   ['Slack (Bot)','Messaging','listed','not configured in this org'],
   ['Pinecone','Memory','listed','not configured in this org'],
   ['GitHub','Code','listed','not configured in this org'],
   ['S3-compatible Object Store','Infra','off','not configured — ask your admin']];
  const st={allowed:'<span class="tag ok">Allowed</span>',listed:'<span class="tag mut">Listed</span>',off:'<span class="tag mut">Not configured</span>',wizard:'<span class="tag warn">Wizard</span>'};
  const wiz=state.wiz>=3?'':`<div class="card"><h3>Add a custom API connector</h3><p style="color:var(--mut);font-size:13.5px;margin:4px 0 10px">Any REST API becomes a governed connector — the key never touches a build or a Sprite.</p>
   <div class="kv"><dt>1 · Define</dt><dd class="mono">coop-core-api · https://core.coop.example/v1 · auth: header (Bearer) · key ●●●●●●●●●●</dd>
   <dt>2 · Test</dt><dd>${state.wiz>=1?'<span class="tag ok">test call green — GET /v1/health · 200 in 84 ms</span>':'<button class="btn" data-act="wizTest">Send test call</button>'}</dd>
   <dt>3 · Scope &amp; grant</dt><dd>${state.wiz>=2?'<span class="tag ok">granted — default-deny, label builds</span>':state.wiz>=1?'<button class="btn" data-act="wizScope">Allow /v1/invoices* · deny /admin* · grant to builds</button>':'complete the test first'}</dd></div></div>`;
  return `<div class="scene"><h1 class="title">Connectors</h1>
  <p class="sub">${dev()?'Three states, never collapsed: public catalog ≠ org-configured ≠ per-Sprite authorization. Live from GET /v1/oauth/connections. Integrations, plugins, MCP and SDKs never render as connectors.':'Your app can only use what is listed here, and only within the granted scope. Everything else is denied by default.'}</p>
  <div class="scroll"><table><thead><tr><th>Connector</th><th>Group</th><th>State</th><th>Policy</th></tr></thead><tbody>
  ${rows.map(r=>`<tr><td>${esc(r[0])}</td><td style="color:var(--mut)">${esc(r[1])}</td><td>${st[r[2]]}</td><td style="color:var(--mut);font-size:12.5px">${esc(r[3])}</td></tr>`).join('')}
  </tbody></table></div>${wiz}
  <div class="actions"><button class="btn primary" data-act="toDeploy">Continue — deploy</button></div></div>`;
 },
 deploy(){
  const lines=['$ studio deploy '+(state.app?state.app.slug:'app'),'✓ slug reserved — collision-safe','✓ builder sprite created · label: builds · gateway sha256 7f21…c9','✓ MPG reachable · migrations 000001–000006 applied','✓ runtime up · fly-issued URL received and persisted'];
  const shown=lines.slice(0,Math.max(1,state.dlines));
  return `<div class="scene"><h1 class="title">Deploy</h1>
  <p class="sub">${dev()?'Provisioning is idempotent: deterministic names, attach-on-retry, token-only URL lock. URLs come from API responses, never from string concatenation.':'Deploying to your own machine on Fly. Nothing is shared — this app gets its own URL and its own boundaries.'}</p>
  <div class="card"><pre class="code">${shown.map(l=>esc(l)).join('\n')}${state.dlines<lines.length?'\n…':''}</pre>
  ${state.url?`<p class="mono" style="margin:12px 0 0;color:var(--ok)">→ ${esc(state.url)}</p>`:''}
  ${state.dlines>=lines.length?`<div class="actions"><button class="btn primary" data-act="toRun">Open the app</button></div>`:''}</div>
  ${dev()?'<div class="note">Credentials: gateway-held. Sprite env is clean — secret-absence tests assert it.</div>':''}</div>`;
 },
 run(){
  const d=RUNDEMO[state.app?state.app.slug:'default']||RUNDEMO['default'];
  const pending=state.events.find(e=>e.kind==='approval'&&e.apr&&!e.apr.decided);
  const decided=state.events.find(e=>e.kind==='approval'&&e.apr&&e.apr.decided);
  return `<div class="scene"><h1 class="title">${esc(state.app?state.app.name:'App')} — live</h1>
  <p class="sub">${dev()?'Every step is checkpointed; effects are receipted; model output is marked untrusted.':'It is running. It will ask before it sends anything — watch the Pulse on the right.'}</p>
  <div class="grid3" style="margin-bottom:14px">${d.metrics.map(m=>`<div class="metric"><b>${m[0]}</b><span>${m[1]}</span></div>`).join('')}</div>
  <div class="scroll"><table><thead><tr>${d.head.map(h=>`<th>${esc(h)}</th>`).join('')}</tr></thead><tbody>
  ${d.rows.map(r=>`<tr>${r.slice(0,-1).map(c=>`<td>${esc(String(c))}</td>`).join('')}<td>${tag(r[r.length-1])}</td></tr>`).join('')}
  </tbody></table></div>
  <div class="actions">
   ${pending?`<button class="btn" disabled>Approval pending in the Pulse →</button>`:decided&&decided.apr.decided==='approved'?`<span class="tag ok">Confirmed — receipted</span>`:decided?`<span class="tag mut">Declined — nothing sent</span>`:`<button class="btn primary" data-act="ask">${esc(d.act)}</button>`}
   <button class="btn" data-act="toEvidence">View evidence</button>
  </div>
  ${dev()?`<details><summary>Run timeline (receipts)</summary><pre class="code">step 1  connector.box.list_folder   read        done   82 ms
step 2  transform.match_po          pure        done   3 ms
step 3  llm.classify                untrusted   done   conf 0.94
step 4  approve.reminder            held        waiting_approval  apr_51c9
step 5  connector.reminder.send     irreversible receipted rcpt_9f3c01</pre></details>`:''}
  </div>`;
 },
 evidence(){
  const gates=[
   ['G0','Repo & dependency truth','verified','ci: pin-gate PG19 · go test -race green'],
   ['G1','Sprite + connector round-trip','verified','live: gateway call ok · secret-absence pass'],
   ['G2','Connector catalog truth','verified','api: GET /v1/oauth/connections · 3-state render'],
   ['G3','Custom API wizard','verified','test: custom_api/test green · default-deny proven'],
   ['G4','Spec generation','verified','fixtures 12/12 · hash+version pinned'],
   ['G5','UI generation','verified','render-from-spec · empty/error states'],
   ['G6','App execution','verified','bounded runs · no unapproved writes'],
   ['G7','Demo readiness','pending','clean-state rehearsal'],
   ['G8','Dogfooding (Invoke-only)','verified','bypass allowlist test green'],
   ['G9','Fly deployment','verified','reproducible flyctl · no ephemeral state'],
   ['G10','Per-app stable URL','verified','persist + replay · collision-safe'],
   ['G11','Control-plane integration','verified','least-privilege adapters'],
   ['G12','MVP polish','pending','final rehearsal']];
  return `<div class="scene"><h1 class="title">Evidence</h1>
  <p class="sub">${dev()?'Claims ledger — the UI may only assert what a passing test proves. Adversarial probes: P1 concurrent single-execution ✓ · P2 question-symlink read refused ✓ · P3 answer-symlink clobber refused ✓ · P4 honest terminality ✓.':'Every promise this studio makes is checked by a test. Green means proven — not promised.'}</p>
  <div class="scroll"><table><thead><tr><th>Gate</th><th>Claim</th><th>State</th>${dev()?'<th>Proof</th>':''}</tr></thead><tbody>
  ${gates.map(g=>`<tr><td class="mono">${g[0]}</td><td>${esc(g[1])}</td><td>${g[2]==='verified'?'<span class="tag ok">verified</span>':'<span class="tag warn">pending</span>'}</td>${dev()?`<td class="mono" style="color:var(--dim);font-size:11.5px">${esc(g[3])}</td>`:''}</tr>`).join('')}
  </tbody></table></div>
  <div class="note">Receipts all the way down — every effect, approval and refusal is bound to an auditable record.</div></div>`;
 },
 refused(){
  return `<div class="scene"><h1 class="title">Refused by policy</h1>
  <p class="sub">The request — “send reminders to all overdue customers now, without approval” — cannot ship. This is the safety layer doing its job, not an error.</p>
  <div class="card bad"><h3>Why it was refused</h3>
  <div class="checkrow"><span class="c" style="color:var(--bad)">✗</span>External writes require explicit approval (policy: approval required)</div>
  <div class="checkrow"><span class="c" style="color:var(--bad)">✗</span>Recipient set exceeds declared scope (assigned_accounts)</div>
  <div class="checkrow"><span class="c" style="color:var(--bad)">✗</span>Bulk send exceeds declared limits (1 reminder / invoice / day)</div>
  ${dev()?'<div class="devd" style="color:var(--dim);font-family:var(--mono);font-size:11.5px;margin-top:8px">rule gov.approval.required · validator deterministic · refusal receipted (audit rec_a17f)</div>':''}</div>
  <div class="actions"><button class="btn" data-act="home">Back to gallery</button></div></div>`;
 },
 settings(){
  const rows=[['Studio','Fly Machines — reproducible flyctl deploy, no ephemeral state'],['Database','Fly Managed Postgres (Neon fallback) — migrations 000001–000006'],['Builders','One disposable Sprite per build — labels + name-prefix scoped'],['Protocol','gateway v1 — HMAC both directions, per-build KDF keys, 5-min skew'],['Framework','miniapp @ 73351a8 — pinned, CI-gated before reliance']];
  return `<div class="scene"><h1 class="title">Settings</h1><p class="sub">${dev()?'Environment of record. Nothing here is assumed — each row binds to the evidence ledger.':'Where things run. You do not need to touch any of this.'}</p>
  <div class="card"><div class="kv">${rows.map(r=>`<dt>${r[0]}</dt><dd>${esc(r[1])}</dd>`).join('')}</div></div>
  <div class="card"><h3>Stop everything</h3><p style="color:var(--mut);font-size:13.5px;margin:4px 0 8px">The red control in the left rail halts every app instantly. Hold it for a moment to fire.</p></div>
  <div class="note">Focus Console — static concept build. Interactions are scripted; no backend is contacted.</div></div>`;
 }
};

/* ---------- pipeline ---------- */
function nameFor(v){
  const s=v.toLowerCase();
  if(s.includes('invoice'))return['Invoice Review Desk','invoice-desk'];
  if(s.includes('digest')||s.includes('box'))return['Friday Box Digest','box-digest'];
  if(s.includes('triage')||s.includes('github')||s.includes('issue'))return['GitHub Issue Triage','gh-triage'];
  return['Custom miniapp','custom-app'];
}
async function pipeline(label,slug){
  if(state.busy)return;state.busy=true;state.stopped=false;
  state.app={name:label,slug,ver:1,hash:'sha256 9f3c…e2',status:'building'};state.url=null;state.wiz=0;state.dlines=0;
  PHASES.forEach(([k])=>state.ph[k]='idle');state.ph.describe='active';
  addEv('event','Brief received','“'+label+'” — generating the plan.');
  go('spec');await sleep(550);
  state.ph.describe='proven';state.ph.spec='proven';state.ph.validate='active';go('validate');await sleep(1150);
  state.ph.validate='proven';state.ph.connect='active';go('connect');await sleep(950);
  state.ph.connect='proven';state.ph.deploy='active';go('deploy');
  const total=5;for(let i=1;i<=total;i++){if(state.stopped)break;state.dlines=i;render();await sleep(320)}
  if(state.stopped){state.busy=false;return}
  state.ph.deploy='proven';state.url='https://'+slug+'.studio-demo.fly.dev';
  state.ph.run='active';go('run');
  const d=RUNDEMO[slug]||RUNDEMO['default'];
  addEv('approval','Approval required: '+d.apr.t,d.apr.b,{d:d.apr.d},{exp:Date.now()+300000,decided:null,receipt:d.apr.receipt});
  state.apps.unshift({name:label,status:'awaiting approval'});
  state.busy=false;render();
}
function refuse(){
  if(state.busy)return;
  PHASES.forEach(([k])=>state.ph[k]='idle');state.ph.validate='refused';
  addEv('alert','Policy refusal receipted','Unapproved bulk send blocked. The refusal itself is an auditable record.',{d:'rule gov.approval.required · audit rec_a17f'});
  go('refused');
}

/* ---------- global events ---------- */
document.addEventListener('click',e=>{
  const t=e.target.closest('[data-nav],[data-phase],[data-act],[data-pf],[data-ev]');
  if(!t)return;
  if(t.dataset.nav){go(t.dataset.nav);if(t.id==='pulseBtn')openPulse(true);return}
  if(t.dataset.phase){const p=t.dataset.phase;go(p==='describe'?'gallery':p);return}
  if(t.dataset.pf){state.filter=t.dataset.pf;renderFilters();renderStream();return}
  if(t.dataset.ev){decide(t.dataset.ev,t.dataset.val);return}
  const a=t.dataset.act;
  if(a==='home')go('gallery');
  if(a==='toValidate')go('validate');
  if(a==='toConnect')go('connect');
  if(a==='toDeploy')go('deploy');
  if(a==='toRun')go('run');
  if(a==='toEvidence')go('evidence');
  if(a==='wizTest'){state.wiz=1;addEv('event','Custom API test call','GET /v1/health → 200 in 84 ms. Stored credential sent only to stored origins.');render()}
  if(a==='wizScope'){state.wiz=3;addEv('receipt','Custom connector granted','coop-core-api allowed for label builds — paths /v1/invoices*, /admin* denied. Default-deny everywhere else.',{d:'POST /v1/oauth/connections/custom_api · access_policy applied'});render()}
  if(a==='ask'){
    const d=RUNDEMO[state.app&&state.app.slug==='custom-app'?'default':(state.app?state.app.slug:'default')]||RUNDEMO['default'];
    addEv('approval','Approval required: '+d.apr.t,d.apr.b,{d:d.apr.d},{exp:Date.now()+300000,decided:null,receipt:d.apr.receipt});
    render();
  }
});
$('#lineForm').addEventListener('submit',e=>{
  e.preventDefault();const v=$('#lineInput').value.trim();if(!v)return;$('#lineInput').value='';
  if(/without approval|skip approval|no approval|all overdue|mass.?send|everyone now/i.test(v)){refuse();return}
  const[n,s]=nameFor(v);pipeline(n,s);
});
function renderFilters(){
  $('#pfilters').innerHTML=[['all','All'],['approval','Approvals'],['receipt','Receipts'],['alert','Alerts'],['event','Events']].map(f=>`<button class="pf ${state.filter===f[0]?'on':''}" data-pf="${f[0]}">${f[1]}</button>`).join('');
}
function renderChips(){
  $('#chips').innerHTML=[
   ['Build: Invoice Review Desk',()=>pipeline('Invoice Review Desk','invoice-desk')],
   ['Build: Friday Box Digest',()=>pipeline('Friday Box Digest','box-digest')],
   ['Build: GitHub Issue Triage',()=>pipeline('GitHub Issue Triage','gh-triage')],
   ['Try: unapproved bulk send (refused)',refuse]
  ].map((c,i)=>`<button class="chip" data-chip="${i}">${esc(c[0])}</button>`).join('');
}
document.addEventListener('click',e=>{const c=e.target.closest('[data-chip]');if(!c)return;[()=>pipeline('Invoice Review Desk','invoice-desk'),()=>pipeline('Friday Box Digest','box-digest'),()=>pipeline('GitHub Issue Triage','gh-triage'),refuse][+c.dataset.chip]()});

/* ---------- persona ---------- */
$('#pOps').addEventListener('click',()=>{state.persona='ops';$('#pOps').classList.add('on');$('#pDev').classList.remove('on');render()});
$('#pDev').addEventListener('click',()=>{state.persona='dev';$('#pDev').classList.add('on');$('#pOps').classList.remove('on');render()});

/* ---------- pulse sheet (mobile) ---------- */
function openPulse(on){$('#pulse').classList.toggle('open',on);$('#overlay').classList.toggle('on',on)}
$('#pulseEdge').addEventListener('click',()=>openPulse(true));
$('#overlay').addEventListener('click',()=>openPulse(false));
document.addEventListener('keydown',e=>{if(e.key==='Escape')openPulse(false)});

/* ---------- kill switch (hold to confirm) ---------- */
let killT=null;const kill=$('#kill'),fill=$('#killFill');
function killStart(ev){ev.preventDefault();const t0=Date.now();
  const step=()=>{const p=Math.min(1,(Date.now()-t0)/700);fill.style.width=(p*100)+'%';
    if(p>=1){killStop();fill.style.width='0';state.stopped=true;state.busy=false;
      state.events.forEach(x=>{if(x.apr&&!x.apr.decided)x.apr.decided='void'});
      addEv('alert','All runs stopped','Kill switch engaged — every app halted, pending approvals voided. Nothing was half-done: completed steps stay receipted.');
      state.apps.forEach(a=>a.status='running');render();return}
    killT=requestAnimationFrame(step)};
  killT=requestAnimationFrame(step)}
function killStop(){if(killT)cancelAnimationFrame(killT);killT=null;fill.style.width='0'}
kill.addEventListener('pointerdown',killStart);kill.addEventListener('pointerup',killStop);kill.addEventListener('pointerleave',killStop);

/* ---------- boot ---------- */
function render(){renderSpine();renderMeta();renderNav();$('#stage').innerHTML=SCENES[state.scene]();renderStream();updateBadges()}
renderFilters();renderChips();render();
</script>
</body>
</html>



```

```python
html_extended = """<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>MiniApp Studio — Focus Console</title>
<style>
:root{
  --bg:#0e1113; --panel:#14171a; --panel2:#1a1e22; --panel3:#101417;
  --line:#252b31; --line2:#39424b;
  --txt:#e7ebee; --mut:#98a2ad; --dim:#6c7681;
  --ok:#4cc38a; --warn:#e0a63f; --bad:#e5484d; --acc:#7aa2f7;
  --mono:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;
}
*{box-sizing:border-box}
html,body{margin:0;height:100%}
body{background:var(--bg);color:var(--txt);font:15px/1.55 ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;-webkit-font-smoothing:antialiased}
.app{display:grid;grid-template-columns:64px minmax(0,1fr) clamp(300px,25vw,370px);height:100dvh}
:focus-visible{outline:1.5px solid var(--acc);outline-offset:1px}
::-webkit-scrollbar{width:10px;height:10px}
::-webkit-scrollbar-thumb{background:#252b31;border-radius:6px}
button{font:inherit}

/* ---------- left rail ---------- */
.rail{border-right:1px solid var(--line);background:var(--panel);display:flex;flex-direction:column;justify-content:space-between;align-items:center;padding:10px 0}
.rail-top,.rail-bottom{display:flex;flex-direction:column;gap:6px;align-items:center}
.brand{color:var(--acc);font-size:17px;margin-bottom:6px;user-select:none}
.rbtn{position:relative;width:40px;height:40px;border-radius:10px;border:1px solid transparent;background:none;color:var(--mut);display:grid;place-items:center;cursor:pointer}
.rbtn:hover{background:var(--panel2);color:var(--txt)}
.rbtn.on{color:var(--txt);border-color:var(--line);background:var(--panel2)}
.rbtn svg{width:18px;height:18px}
.rbadge{position:absolute;top:1px;right:1px;background:var(--warn);color:#141414;font-size:10px;font-weight:700;min-width:16px;height:16px;border-radius:8px;display:grid;place-items:center;padding:0 3px}
.pgrp{display:flex;flex-direction:column;gap:4px}
.pbtn{width:40px;height:32px;border-radius:8px;border:1px solid var(--line);background:none;color:var(--mut);font-size:9px;font-weight:700;letter-spacing:.5px;cursor:pointer}
.pbtn.on{background:var(--panel2);color:var(--txt);border-color:var(--line2)}
.kill{position:relative;overflow:hidden;width:40px;height:40px;border-radius:10px;border:1px solid #4a2327;background:none;color:var(--bad);cursor:pointer;user-select:none}
.kill .fill{position:absolute;left:0;top:0;bottom:0;width:0;background:rgba(229,72,77,.35)}
.kill svg{position:relative;z-index:1;width:16px;height:16px}

/* ---------- stage ---------- */
.stage{display:flex;flex-direction:column;min-width:0}
.stagehead{border-bottom:1px solid var(--line);background:var(--panel);padding:10px 14px 0}
.spine{display:flex;gap:4px;overflow-x:auto;scrollbar-width:none;padding-bottom:10px}
.spine::-webkit-scrollbar{display:none}
.sph{display:flex;align-items:center;gap:6px;padding:5px 10px;border-radius:999px;border:1px solid var(--line);color:var(--mut);font-size:12px;white-space:nowrap;cursor:pointer;background:none;flex:0 0 auto}
.sph .dot{width:7px;height:7px;border-radius:50%;background:#3a4249;flex:0 0 auto}
.sph[data-current="true"]{background:var(--panel2);color:var(--txt);border-color:var(--line2)}
.sph[data-s="active"]{color:var(--txt);border-color:#5a4a26}
.sph[data-s="active"] .dot{background:var(--warn);box-shadow:0 0 0 3px rgba(224,166,63,.16);animation:pulse 1.4s infinite}
.sph[data-s="proven"]{color:#bfe9d4;border-color:#274937}
.sph[data-s="proven"] .dot{background:var(--ok)}
.sph[data-s="refused"]{color:#f3b9bc;border-color:#54262a}
.sph[data-s="refused"] .dot{background:var(--bad)}
@keyframes pulse{50%{box-shadow:0 0 0 6px rgba(224,166,63,.07)}}
.stagemeta{display:flex;justify-content:space-between;align-items:center;gap:8px;padding-bottom:10px;color:var(--mut);font-size:12px;flex-wrap:wrap}
.meta-l,.meta-r{display:flex;align-items:center;gap:8px;flex-wrap:wrap;min-width:0}
.adapter-pill{display:inline-flex;align-items:center;gap:5px;padding:2px 9px;border-radius:999px;border:1px solid var(--line);background:var(--panel2);color:var(--mut);font-size:11px;cursor:pointer}
.adapter-pill:hover{border-color:var(--line2);color:var(--txt)}
.adapter-dot{width:6px;height:6px;border-radius:50%;background:var(--ok)}
.adapter-dot.live{background:var(--acc)}
.adapter-dot.err{background:var(--warn)}

.stagebody{flex:1;overflow:auto;padding:clamp(14px,2.5vw,28px)}
.scene{max-width:1020px;margin:0 auto}
h1.title{font-size:clamp(1.25rem,2.4vw,1.6rem);margin:0 0 6px;font-weight:650;letter-spacing:-.01em;display:flex;align-items:center;gap:10px;flex-wrap:wrap}
.sub{color:var(--mut);margin:0 0 18px;max-width:68ch}
.card{background:var(--panel);border:1px solid var(--line);border-radius:12px;padding:clamp(14px,2vw,20px);margin-bottom:14px}
.card.good{border-color:#274937}
.card.bad{border-color:#54262a}
.card.warn{border-color:#5a4a26}
.card h3{margin:0 0 4px;font-size:15px;display:flex;align-items:center;justify-content:space-between;gap:8px;flex-wrap:wrap}
.grid2{display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:12px}
.grid3{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:12px}
.metric{background:var(--panel);border:1px solid var(--line);border-radius:12px;padding:14px 16px}
.metric b{font-size:clamp(1.4rem,3vw,1.9rem);font-weight:650;display:block}
.metric span{color:var(--mut);font-size:12.5px}
.chip{display:inline-flex;align-items:center;gap:6px;padding:5px 11px;border-radius:999px;border:1px solid var(--line);background:none;color:var(--mut);font-size:12.5px;cursor:pointer}
.chip:hover{color:var(--txt);border-color:var(--line2)}
.chips{display:flex;flex-wrap:wrap;gap:8px;margin-bottom:10px}
.chips:empty{display:none}
.tag{display:inline-flex;align-items:center;gap:5px;padding:2px 9px;border-radius:999px;font-size:11px;font-weight:600;border:1px solid;white-space:nowrap}
.tag.ok{color:#9fe3bf;border-color:#274937;background:rgba(76,195,138,.08)}
.tag.warn{color:#f0cf8f;border-color:#5a4a26;background:rgba(224,166,63,.08)}
.tag.bad{color:#f3b9bc;border-color:#54262a;background:rgba(229,72,77,.08)}
.tag.acc{color:#b7ceff;border-color:#2f477a;background:rgba(122,162,247,.08)}
.tag.mut{color:var(--mut);border-color:var(--line)}
.mono{font-family:var(--mono);font-size:12px;color:#c7d1da}
pre.code{background:var(--panel3);border:1px solid var(--line);border-radius:8px;padding:12px;overflow-x:auto;font-family:var(--mono);font-size:12px;line-height:1.55;color:#c7d1da;margin:10px 0 0}
.scroll{overflow-x:auto;border:1px solid var(--line);border-radius:10px;background:var(--panel)}
table{width:100%;border-collapse:collapse;font-size:13.5px}
th{text-align:left;color:var(--mut);font-weight:600;font-size:11.5px;padding:8px 10px;border-bottom:1px solid var(--line);white-space:nowrap}
td{padding:9px 10px;border-bottom:1px solid #1d2226;vertical-align:top}
tr:last-child td{border-bottom:none}
.btn{display:inline-flex;align-items:center;gap:8px;padding:8px 14px;border-radius:9px;border:1px solid var(--line);background:var(--panel2);color:var(--txt);cursor:pointer;font-size:13.5px;font-weight:600}
.btn:hover:not(:disabled){border-color:var(--line2)}
.btn.primary{background:var(--ok);border-color:var(--ok);color:#0b100d}
.btn.danger{background:none;border-color:#54262a;color:var(--bad)}
.btn.sm{padding:4px 10px;font-size:12px;border-radius:7px}
.btn:disabled{opacity:.45;cursor:default}
.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:12px;align-items:center}
details{border:1px solid var(--line);border-radius:8px;padding:8px 12px;margin-top:10px;background:var(--panel3)}
summary{cursor:pointer;color:var(--mut);font-size:12.5px;user-select:none}
.kv{display:grid;grid-template-columns:minmax(130px,185px) 1fr;gap:8px 14px;font-size:13.5px}
.kv dt{color:var(--mut)}.kv dd{margin:0}
.note{color:var(--dim);font-size:12px;margin-top:12px}
.checkrow{display:flex;gap:8px;align-items:baseline;padding:4px 0;font-size:13.5px}
.checkrow .c{color:var(--ok);font-family:var(--mono)}
.steps{list-style:none;margin:0;padding:0}
.steps li{display:flex;gap:10px;padding:7px 0;border-bottom:1px solid #1d2226;font-size:13.5px;align-items:center;justify-content:space-between;flex-wrap:wrap}
.steps li:last-child{border-bottom:none}
.steps .n{font-family:var(--mono);color:var(--dim);font-size:12px}

/* ---------- subtabs & diff blocks ---------- */
.subtabs{display:flex;gap:6px;margin-bottom:14px;border-bottom:1px solid var(--line);padding-bottom:8px}
.stab{padding:5px 12px;border-radius:8px;border:1px solid transparent;background:none;color:var(--mut);font-size:12.5px;font-weight:600;cursor:pointer}
.stab:hover{color:var(--txt)}
.stab.on{background:var(--panel2);color:var(--txt);border-color:var(--line)}

.diffbox{border:1px solid var(--line);border-radius:8px;background:var(--panel3);padding:10px 12px;margin-top:10px;font-size:12.5px}
.diff-hdr{display:flex;justify-content:space-between;align-items:center;gap:8px;margin-bottom:8px;padding-bottom:6px;border-bottom:1px solid var(--line)}
.diff-grid{display:grid;grid-template-columns:1fr 1fr;gap:8px}
.diff-col{border:1px solid var(--line);border-radius:6px;padding:8px;background:var(--panel)}
.diff-col.before{border-color:#3b2629}
.diff-col.after{border-color:#233d30}
.diff-lbl{font-size:10.5px;text-transform:uppercase;letter-spacing:.06em;color:var(--dim);margin-bottom:4px;font-family:var(--mono)}
.diff-kv{display:grid;grid-template-columns:auto 1fr;gap:3px 10px;font-family:var(--mono);font-size:11.5px}
.diff-kv span:nth-child(odd){color:var(--mut)}
.diff-line{font-family:var(--mono);font-size:11.5px;padding:2px 6px;border-radius:4px;margin:2px 0;white-space:pre-wrap;word-break:break-word}
.diff-line.del{background:rgba(229,72,77,.1);color:#f3b9bc}
.diff-line.add{background:rgba(76,195,138,.1);color:#bfe9d4}
.diff-line.ctx{color:var(--mut)}

/* ---------- bottom line ---------- */
.line{border-top:1px solid var(--line);background:var(--panel);padding:10px 14px 12px}
.lineform{display:flex;gap:8px}
.lineform input{flex:1;min-width:0;background:var(--panel2);border:1px solid var(--line);border-radius:10px;color:var(--txt);padding:11px 14px;font:inherit}
.lineform input::placeholder{color:var(--dim)}
.lineform input:focus{outline:none;border-color:var(--acc)}
.lineform button{width:46px;border-radius:10px;border:1px solid var(--line);background:var(--panel2);color:var(--txt);font-size:16px;cursor:pointer}
.input-row{display:flex;gap:8px;align-items:center}
.input-row input{flex:1;min-width:0;background:var(--panel2);border:1px solid var(--line);border-radius:10px;color:var(--txt);padding:11px 14px;font:inherit}
.input-row input::placeholder{color:var(--dim)}
.input-row input:focus{outline:none;border-color:var(--acc)}
.input-row button{height:44px;padding:0 16px;border-radius:10px;border:1px solid var(--line);background:var(--panel2);color:var(--txt);font-size:15px;cursor:pointer}

/* ---------- pulse (right) ---------- */
.pulse{border-left:1px solid var(--line);background:var(--panel);display:flex;flex-direction:column;min-width:0}
.pulse-head{padding:12px 14px 10px;border-bottom:1px solid var(--line);display:flex;flex-direction:column;gap:8px}
.pulse-head h2{margin:0;font-size:12px;letter-spacing:.1em;text-transform:uppercase;color:var(--mut);display:flex;justify-content:space-between;align-items:center}
.pfilters{display:flex;gap:6px;flex-wrap:wrap}
.pf{padding:3px 9px;border-radius:999px;border:1px solid var(--line);background:none;color:var(--mut);font-size:11.5px;cursor:pointer}
.pf.on{color:var(--txt);background:var(--panel2);border-color:var(--line2)}
.stream{flex:1;overflow-y:auto;padding:10px 12px;display:flex;flex-direction:column;gap:8px}
.ev{border:1px solid var(--line);border-left-width:3px;border-radius:8px;background:var(--panel2);padding:9px 11px;font-size:13px}
.ev.event{border-left-color:#3a4249}
.ev.receipt{border-left-color:var(--ok)}
.ev.alert{border-left-color:var(--bad)}
.ev.approval{border-left-color:var(--warn)}
.ev .t{font-weight:600;margin-bottom:2px;display:flex;justify-content:space-between;align-items:flex-start;gap:6px}
.ev .d{color:var(--mut);font-size:12.5px;overflow-wrap:anywhere}
.ev .devd{color:var(--dim);font-size:11.5px;font-family:var(--mono);margin-top:4px;overflow-wrap:anywhere}
.ev .tm{color:var(--dim);font-size:11px;font-family:var(--mono)}
.ev .row{display:flex;gap:8px;margin-top:8px;flex-wrap:wrap;align-items:center}
.ev .row .btn{padding:5px 11px;font-size:12px}
.ev details{margin-top:8px;padding:6px 9px;background:var(--panel3)}
.ev summary{font-size:11.5px;font-family:var(--mono);color:var(--acc)}
.count{font-family:var(--mono);color:var(--warn);font-size:11.5px}
.pulse-edge{display:none;position:fixed;right:10px;bottom:86px;z-index:55;padding:8px 13px;border-radius:999px;border:1px solid var(--line2);background:var(--panel2);color:var(--txt);font-size:12px;font-weight:600;cursor:pointer;align-items:center;gap:7px;box-shadow:0 6px 24px rgba(0,0,0,.4)}
.overlay{display:none;position:fixed;inset:0;background:rgba(0,0,0,.5);z-index:58}

/* ---------- form inputs inside settings ---------- */
.field{display:flex;flex-direction:column;gap:4px;margin-bottom:10px}
.field label{font-size:12px;color:var(--mut)}
.field input,.field select{background:var(--panel3);border:1px solid var(--line);border-radius:8px;color:var(--txt);padding:8px 10px;font:inherit;font-size:13px}

/* ---------- responsive ---------- */
@media(max-width:920px){
  .app{grid-template-columns:56px minmax(0,1fr)}
  .pulse{position:fixed;top:0;right:0;bottom:0;width:min(360px,90vw);transform:translateX(105%);transition:transform .22s ease;z-index:60;box-shadow:-12px 0 40px rgba(0,0,0,.45)}
  .pulse.open{transform:none}
  .pulse-edge{display:flex}
  .overlay.on{display:block}
  .diff-grid{grid-template-columns:1fr}
}
@media(prefers-reduced-motion:reduce){*{animation:none!important;transition:none!important}}
</style>
</head>
<body>
<div class="app">
  <aside class="rail" aria-label="Essentials">
    <div class="rail-top">
      <div class="brand" title="MiniApp Studio">◆</div>
      <button class="rbtn" data-nav="gallery" title="Gallery (1)" aria-label="Gallery"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/></svg></button>
      <button class="rbtn" data-nav="connect" title="Connectors (2)" aria-label="Connectors"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M9 7V3M15 7V3M7 7h10v4a5 5 0 0 1-10 0V7z"/><path d="M12 16v5"/></svg></button>
      <button class="rbtn" data-nav="evidence" title="Evidence (3)" aria-label="Evidence"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M12 3l7 3v5c0 5-3.5 8-7 10-3.5-2-7-5-7-10V6l7-3z"/><path d="M9 11.5l2 2 4-4"/></svg></button>
      <button class="rbtn" data-nav="settings" title="Settings & Transport (4)" aria-label="Settings"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M19.1 4.9L17 7M7 17l-2.1 2.1"/></svg></button>
      <button class="rbtn" id="pulseBtn" title="Pulse (5)" aria-label="Pulse"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M3 12h4l3-8 4 16 3-8h4"/></svg><span class="rbadge" id="railBadge" hidden>0</span></button>
    </div>
    <div class="rail-bottom">
      <div class="pgrp" role="group" aria-label="Persona">
        <button class="pbtn on" id="pOps" title="OPS — plain-language operator console">OPS</button>
        <button class="pbtn" id="pDev" title="DEV — specs, diffs, receipts & protocol trace">DEV</button>
      </div>
      <button class="kill" id="kill" title="Hold 0.7s to halt all runs" aria-label="Hold to stop all runs"><span class="fill" id="killFill"></span><svg viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="6" width="12" height="12" rx="2"/></svg></button>
    </div>
  </aside>

  <main class="stage">
    <header class="stagehead">
      <div class="spine" id="spine" aria-label="Pipeline"></div>
      <div class="stagemeta" id="stageMeta"></div>
    </header>
    <section class="stagebody" id="stage" aria-live="polite"></section>
    <footer class="line">
      <div class="chips" id="chips"></div>
      <form class="input-row" id="lineForm">
        <input id="lineInput" placeholder="Describe what you want — or type /spec, /validate, /deploy, /halt" aria-label="Describe what you want">
        <button type="submit" aria-label="Send">→</button>
      </form>
    </footer>
  </main>

  <aside class="pulse" id="pulse" aria-label="Pulse">
    <div class="pulse-head">
      <h2>Pulse <span class="mono" id="pulseCount"></span></h2>
      <div class="pfilters" id="pfilters"></div>
    </div>
    <div class="stream" id="stream"></div>
  </aside>
</div>
<button class="pulse-edge" id="pulseEdge">Pulse <span class="rbadge" id="edgeBadge" style="position:static" hidden>0</span></button>
<div class="overlay" id="overlay"></div>

<script>
'use strict';

/* ============================================================================
   1. PURE UTILITIES & DOMAIN FIXTURES
   ============================================================================ */
const $ = s => document.querySelector(s);
const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const sleep = ms => new Promise(r => setTimeout(r, ms));
const nowStr = () => new Date().toLocaleTimeString([], {hour:'2-digit', minute:'2-digit', second:'2-digit'});

const PHASES = [
  ['describe', 'Describe'],
  ['spec',     'Spec'],
  ['validate', 'Validate'],
  ['connect',  'Connect'],
  ['deploy',   'Deploy'],
  ['run',      'Run'],
  ['evidence', 'Evidence']
];

const TEMPLATES = {
  'invoice-desk': {
    name: 'Invoice Review Desk',
    slug: 'invoice-desk',
    summary: 'Watch invoices, match them to POs, remind you to pay — asks first.',
    connectorsLabel: 'Box · approval',
    hash: 'sha256:9f3c7a12…e2',
    spec: {
      entities: ['invoice', 'purchase_order', 'reminder'],
      pages: ['dashboard', 'invoice-detail'],
      workflow: ['detect', 'box.save', 'po.match', 'approve', 'reminder.send'],
      capabilities: [
        { id: 'box.list_folder', mode: 'read', effect: 'none', approval: 'none' },
        { id: 'box.save', mode: 'write', effect: 'reversible', approval: 'none' },
        { id: 'reminder.send', mode: 'write', effect: 'irreversible', approval: 'required' }
      ],
      policies: [
        'max 1 reminder / invoice / day',
        'scope: assigned_accounts',
        'effect classes explicitly declared'
      ],
      fixturesCount: 12,
      rawJson: {
        app: { id: 'invoice-desk', version: 1, hash: 'sha256:9f3c7a12b04d8f19c301a4e2' },
        workflows: {
          remind: {
            steps: ['detect', 'box.save', 'po.match', 'approve', 'reminder.send'],
            idempotencyKey: '{{invoice.id}}-remind-{{date}}'
          }
        },
        policies: {
          'reminder.send': { approval: 'required', limit: '1/day/invoice', scope: 'assigned_accounts' }
        },
        fixtures: { count: 12, deterministic: true }
      }
    },
    metrics: [['24', 'Invoices this week'], ['21', 'Matched to POs'], ['1', 'Awaiting your approval']],
    head: ['Invoice', 'Customer', 'Amount', 'PO match', 'Status'],
    rows: [
      ['INV-1042', 'Acme Corp', '$420.50', '✗ $402.00', ['warn', 'Awaiting approval']],
      ['INV-1041', 'Northwind', '$1,180.00', '✓', ['ok', 'Matched']],
      ['INV-1040', 'Globex', '$86.10', '✓', ['ok', 'Matched']],
      ['INV-1039', 'Initech', '$2,300.00', '✓', ['ok', 'Matched']]
    ],
    act: 'Send reminder for INV-1042',
    apr: {
      id: 'apr_51c9',
      title: 'reminder send · INV-1042 Acme Corp',
      summary: 'Email to ap@acme.example — amount mismatch $420.50 vs PO $402.00. Sends once, reconciled by read.',
      effectClass: 'irreversible',
      connector: 'smtp.outbound',
      idempotencyKey: 'inv-1042-remind-2026-11-25',
      intentHash: 'b7c19e04…44',
      diff: {
        summary: '1 outbound email · 1 ledger state transition',
        before: {
          'invoice.status': 'matched_discrepancy',
          'reminder.sent_count': '0',
          'ledger.state': 'WAITING_APPROVAL'
        },
        after: {
          'invoice.status': 'reminder_sent',
          'reminder.sent_count': '1 (rcpt_9f3c01)',
          'ledger.state': 'CONFIRMED'
        },
        payloadPatch: [
          '- status: "WAITING_APPROVAL"',
          '+ status: "CONFIRMED"',
          '+ recipient: "ap@acme.example"',
          '+ subject: "INV-1042 variance ($420.50 vs PO $402.00)"',
          '+ idempotency_key: "inv-1042-remind-2026-11-25"'
        ]
      },
      receipt: 'Confirmed · rcpt_9f3c01 — sent once, reconciled by read. No duplicate.'
    },
    timeline: [
      { step: 1, op: 'connector.box.list_folder', cls: 'read', state: 'done', ms: 82, receipt: 'rcpt_9f3b10' },
      { step: 2, op: 'transform.match_po', cls: 'pure', state: 'done', ms: 3, receipt: 'deterministic' },
      { step: 3, op: 'llm.extract_variance', cls: 'untrusted', state: 'done', ms: 410, receipt: 'conf=0.98 · schema_ok' },
      { step: 4, op: 'gate.approve_reminder', cls: 'policy', state: 'waiting_approval', ms: 0, receipt: 'apr_51c9' },
      { step: 5, op: 'connector.reminder.send', cls: 'irreversible', state: 'pending_gate', ms: 0, receipt: 'rcpt_9f3c01' }
    ]
  },
  'box-digest': {
    name: 'Friday Box Digest',
    slug: 'box-digest',
    summary: 'Summarize a Box folder every Friday and post it to Slack — with approval.',
    connectorsLabel: 'Box · Slack',
    hash: 'sha256:41ab88c0…19',
    spec: {
      entities: ['document', 'digest_post'],
      pages: ['dashboard', 'digest-preview'],
      workflow: ['box.list_folder', 'llm.summarize', 'approve', 'slack.post_message'],
      capabilities: [
        { id: 'box.list_folder', mode: 'read', effect: 'none', approval: 'none' },
        { id: 'llm.summarize', mode: 'compute', effect: 'untrusted', approval: 'none' },
        { id: 'slack.post_message', mode: 'write', effect: 'reversible', approval: 'required' }
      ],
      policies: [
        'channel allowlist: #team',
        'model output treated as untrusted text',
        'undo handler bound: slack.delete_message'
      ],
      fixturesCount: 12,
      rawJson: {
        app: { id: 'box-digest', version: 1, hash: 'sha256:41ab88c019e73f2a' },
        workflows: {
          weekly: {
            steps: ['box.list_folder', 'llm.summarize', 'approve', 'slack.post_message'],
            idempotencyKey: 'digest-w47-team'
          }
        },
        policies: {
          'slack.post_message': { approval: 'required', channels: ['#team'], reversible: true }
        },
        fixtures: { count: 12, deterministic: true }
      }
    },
    metrics: [['6', 'Docs in Weekly Reports'], ['1', 'Digest drafted'], ['1', 'Awaiting your approval']],
    head: ['Doc', 'Summary', 'Status'],
    rows: [
      ['Q3-revenue.xlsx', 'Revenue up 12% QoQ; churn flat', ['ok', 'Ready']],
      ['incidents.md', '2 sev-2 incidents, both closed', ['ok', 'Ready']],
      ['hiring.csv', '3 offers out, 1 accepted', ['ok', 'Ready']]
    ],
    act: 'Post digest to #team',
    apr: {
      id: 'apr_88d2',
      title: 'post digest to #team',
      summary: 'Posts the Friday summary to Slack #team. Undoable via slack.delete_message.',
      effectClass: 'reversible',
      connector: 'slack.chat',
      idempotencyKey: 'digest-w47-team',
      intentHash: '3e91a04c…11',
      diff: {
        summary: '1 Slack channel post · reversible',
        before: {
          'channel': '#team (C04921A)',
          'weekly_digest.posted': 'false',
          'undo_handle': 'none'
        },
        after: {
          'channel': '#team (C04921A)',
          'weekly_digest.posted': 'true (ts 1732539001.44)',
          'undo_handle': 'slack.delete_message'
        },
        payloadPatch: [
          '+ channel: "#team"',
          '+ text: "Weekly Digest (6 docs): Q3 revenue +12% QoQ; 2 closed sev-2s; 3 offers out."',
          '+ reversible_via: "slack.delete_message"'
        ]
      },
      receipt: 'Confirmed · rcpt_71b0 — posted once to #team.'
    },
    timeline: [
      { step: 1, op: 'connector.box.list_folder', cls: 'read', state: 'done', ms: 94, receipt: 'rcpt_71a1' },
      { step: 2, op: 'llm.summarize_docs', cls: 'untrusted', state: 'done', ms: 620, receipt: 'conf=0.95 · bounded' },
      { step: 3, op: 'gate.approve_post', cls: 'policy', state: 'waiting_approval', ms: 0, receipt: 'apr_88d2' },
      { step: 4, op: 'connector.slack.post_message', cls: 'reversible', state: 'pending_gate', ms: 0, receipt: 'rcpt_71b0' }
    ]
  },
  'gh-triage': {
    name: 'GitHub Issue Triage',
    slug: 'gh-triage',
    summary: 'Label and route incoming issues; model output stays untrusted until you confirm.',
    connectorsLabel: 'GitHub',
    hash: 'sha256:c4d219f0…8b',
    spec: {
      entities: ['issue', 'triage_decision'],
      pages: ['dashboard', 'issue-queue'],
      workflow: ['github.fetch_issue', 'llm.classify', 'approve', 'github.apply_labels'],
      capabilities: [
        { id: 'github.fetch_issue', mode: 'read', effect: 'none', approval: 'none' },
        { id: 'llm.classify', mode: 'compute', effect: 'untrusted', approval: 'none' },
        { id: 'github.apply_labels', mode: 'write', effect: 'reversible', approval: 'required' }
      ],
      policies: [
        'label allowlist: bug, enhancement, chore, auth, ci',
        'assignee allowlist: @maya, @devon',
        'model output cannot execute writes without gate'
      ],
      fixturesCount: 12,
      rawJson: {
        app: { id: 'gh-triage', version: 1, hash: 'sha256:c4d219f03a1c888b' },
        workflows: {
          triage: {
            steps: ['github.fetch_issue', 'llm.classify', 'approve', 'github.apply_labels'],
            idempotencyKey: 'gh-issue-482-triage'
          }
        },
        policies: {
          'github.apply_labels': { approval: 'required', allowLabels: ['bug', 'enhancement', 'chore', 'auth', 'ci'] }
        },
        fixtures: { count: 12, deterministic: true }
      }
    },
    metrics: [['18', 'Open issues'], ['15', 'Triaged'], ['1', 'Awaiting your approval']],
    head: ['Issue', 'Label', 'Confidence', 'Status'],
    rows: [
      ['#482 login loop', 'bug · auth', '0.94', ['warn', 'Awaiting approval']],
      ['#479 dark-mode', 'enhancement', '0.91', ['ok', 'Labeled']],
      ['#477 flaky test', 'chore · ci', '0.88', ['ok', 'Labeled']]
    ],
    act: 'Apply triage for #482',
    apr: {
      id: 'apr_c4d2',
      title: 'label & assign #482',
      summary: 'Applies bug · auth and assigns @maya — based on model output (untrusted, confidence 0.94).',
      effectClass: 'reversible',
      connector: 'github.issues',
      idempotencyKey: 'gh-issue-482-triage',
      intentHash: 'c4d219f0…8b',
      diff: {
        summary: '2 labels added · 1 assignee set',
        before: {
          'issue': '#482 (login loop)',
          'labels': '[]',
          'assignees': '[]'
        },
        after: {
          'issue': '#482 (login loop)',
          'labels': '["bug", "auth"]',
          'assignees': '["@maya"]'
        },
        payloadPatch: [
          '- labels: []',
          '+ labels: ["bug", "auth"]',
          '- assignees: []',
          '+ assignees: ["maya"]'
        ]
      },
      receipt: 'Confirmed · rcpt_c4d2 — labeled and assigned once.'
    },
    timeline: [
      { step: 1, op: 'connector.github.fetch_issue', cls: 'read', state: 'done', ms: 61, receipt: 'rcpt_c401' },
      { step: 2, op: 'llm.classify_issue', cls: 'untrusted', state: 'done', ms: 380, receipt: 'conf=0.94 · untrusted' },
      { step: 3, op: 'gate.approve_triage', cls: 'policy', state: 'waiting_approval', ms: 0, receipt: 'apr_c4d2' },
      { step: 4, op: 'connector.github.apply_labels', cls: 'reversible', state: 'pending_gate', ms: 0, receipt: 'rcpt_c4d2' }
    ]
  }
};

/* ============================================================================
   2. STATE STORE & DETERMINISTIC REDUCER
   ============================================================================ */
function createStore(initial) {
  let state = initial;
  const listeners = new Set();

  function getState() {
    return state;
  }

  function dispatch(action) {
    state = reducer(state, action);
    listeners.forEach(fn => fn(state, action));
    return state;
  }

  function subscribe(fn) {
    listeners.add(fn);
    return () => listeners.delete(fn);
  }

  return { getState, dispatch, subscribe };
}

function makeInitialPhases() {
  const ph = {};
  PHASES.forEach(([k]) => { ph[k] = 'idle'; });
  return ph;
}

const initialState = {
  persona: 'ops',          // 'ops' | 'dev'
  scene: 'gallery',        // 'gallery' | 'spec' | 'validate' | 'connect' | 'deploy' | 'run' | 'evidence' | 'refused' | 'settings'
  subtab: 'overview',      // contextual inspector tab in DEV/OPS scenes
  filter: 'all',           // Pulse stream filter
  busy: false,
  stopped: false,
  runId: 0,
  ph: makeInitialPhases(),
  app: null,
  url: null,
  wiz: 0,
  dlines: 0,
  unknownReceipt: {
    id: 'rcpt_9f21',
    connector: 'webhook.outbound',
    endpoint: 'POST https://hooks.partner.example/v1/notify',
    idempotencyKey: 'wh-notify-88412',
    state: 'UNKNOWN', // 'UNKNOWN' | 'CONFIRMED' | 'VOIDED'
    detail: 'Provider timed out at 5000ms after TCP write. Held in UNKNOWN — never retried blindly.'
  },
  events: [],
  apps: [],
  transport: {
    mode: 'mock',          // 'mock' | 'live'
    status: 'connected',   // 'connected' | 'connecting' | 'error'
    endpoint: 'http://localhost:8080/v1/events/stream',
    token: '',
    lastError: null,
    receivedCount: 0
  }
};

function reducer(state, action) {
  switch (action.type) {
    case 'SET_PERSONA':
      return { ...state, persona: action.persona };

    case 'SET_SCENE':
      return { ...state, scene: action.scene, subtab: 'overview' };

    case 'SET_SUBTAB':
      return { ...state, subtab: action.subtab };

    case 'SET_FILTER':
      return { ...state, filter: action.filter };

    case 'ADD_EVENT': {
      const ev = {
        id: action.event.id || ('e' + Math.random().toString(16).slice(2, 8)),
        kind: action.event.kind || 'event',
        title: action.event.title,
        detail: action.event.detail,
        dd: action.event.dd || null,
        tm: action.event.tm || nowStr(),
        apr: action.event.apr || null
      };
      return { ...state, events: [ev, ...state.events] };
    }

    case 'BUILD_START': {
      const ph = makeInitialPhases();
      ph.describe = 'active';
      return {
        ...state,
        busy: true,
        stopped: false,
        runId: action.runId,
        app: action.app,
        url: null,
        wiz: 0,
        dlines: 0,
        ph,
        scene: 'spec',
        subtab: 'overview'
      };
    }

    case 'BUILD_STEP': {
      if (action.runId !== state.runId || state.stopped) return state;
      return {
        ...state,
        ph: { ...state.ph, ...(action.ph || {}) },
        dlines: action.dlines ?? state.dlines,
        url: action.url ?? state.url,
        scene: action.scene ?? state.scene
      };
    }

    case 'BUILD_COMPLETE': {
      if (action.runId !== state.runId || state.stopped) return state;
      const apps = [{ name: state.app.name, slug: state.app.slug, status: 'awaiting approval' }, ...state.apps];
      return {
        ...state,
        busy: false,
        ph: { ...state.ph, ...(action.ph || {}) },
        url: action.url,
        scene: 'run',
        apps
      };
    }

    case 'BUILD_REFUSED': {
      const ph = makeInitialPhases();
      ph.validate = 'refused';
      return {
        ...state,
        busy: false,
        stopped: false,
        ph,
        scene: 'refused',
        subtab: 'overview'
      };
    }

    case 'WIZ_STEP':
      return { ...state, wiz: action.wiz };

    case 'DECIDE_APPROVAL': {
      const events = state.events.map(e => {
        if (e.id !== action.eventId || !e.apr || e.apr.decided) return e;
        return { ...e, apr: { ...e.apr, decided: action.decision, decidedAt: nowStr() } };
      });
      const approved = action.decision === 'approved';
      const ph = approved
        ? { ...state.ph, run: 'proven', evidence: 'proven' }
        : state.ph;
      const app = state.app
        ? { ...state.app, status: approved ? 'running' : 'declined' }
        : null;
      const apps = state.apps.map((a, idx) =>
        idx === 0 ? { ...a, status: approved ? 'running' : 'declined' } : a
      );
      return { ...state, events, ph, app, apps };
    }

    case 'SETTLE_UNKNOWN': {
      return {
        ...state,
        unknownReceipt: {
          ...state.unknownReceipt,
          state: action.resolution // 'CONFIRMED' | 'VOIDED'
        }
      };
    }

    case 'KILL_ENGAGED': {
      const events = state.events.map(e =>
        e.apr && !e.apr.decided
          ? { ...e, apr: { ...e.apr, decided: 'void', decidedAt: nowStr() } }
          : e
      );
      const apps = state.apps.map(a => ({ ...a, status: 'halted' }));
      return {
        ...state,
        stopped: true,
        busy: false,
        runId: state.runId + 1,
        events,
        apps,
        app: state.app ? { ...state.app, status: 'halted' } : null
      };
    }

    case 'TRANSPORT_CONFIG':
      return {
        ...state,
        transport: { ...state.transport, ...action.patch }
      };

    case 'TRANSPORT_EVENT_RECEIVED':
      return {
        ...state,
        transport: {
          ...state.transport,
          status: 'connected',
          lastError: null,
          receivedCount: state.transport.receivedCount + 1
        }
      };

    default:
      return state;
  }
}

const store = createStore(initialState);

/* ============================================================================
   3. PLUGGABLE MOCK / LIVE EVENT ADAPTER
   ============================================================================ */
class EventAdapter {
  constructor(store) {
    this.store = store;
    this.es = null;
  }

  seedInitial() {
    this.emit({
      kind: 'event',
      title: 'Session started',
      detail: 'Studio kernel ready. Builder gateway v1 · framework pinned and CI-verified.',
      dd: 'pin 73351a8 · go test -race green · PG19 gate'
    });
    this.emit({
      kind: 'event',
      title: 'Connector catalog synced',
      detail: '6 listed · 2 allowed · 1 not configured. Availability is per-organization.',
      dd: 'GET /v1/oauth/connections · 200 OK (38 ms)'
    });
    this.emit({
      kind: 'alert',
      title: 'UNKNOWN held — not retried',
      detail: 'Outbound webhook rcpt_9f21: provider timeout after side effect. Held for explicit settlement — nothing retries blindly.',
      dd: 'ledger PENDING→UNKNOWN · settle via SettleEffect'
    });
  }

  emit(event) {
    this.store.dispatch({ type: 'ADD_EVENT', event });
  }

  setMode(mode, opts = {}) {
    if (this.es) {
      this.es.close();
      this.es = null;
    }
    const patch = { mode, ...opts, lastError: null };
    if (mode === 'mock') {
      patch.status = 'connected';
      this.store.dispatch({ type: 'TRANSPORT_CONFIG', patch });
      this.emit({
        kind: 'event',
        title: 'Adapter switched to Mock',
        detail: 'Deterministic local pipeline and fixture playback active.',
        dd: 'transport=MockTransport · zero external network'
      });
      return;
    }

    const endpoint = opts.endpoint || this.store.getState().transport.endpoint;
    const token = opts.token !== undefined ? opts.token : this.store.getState().transport.token;
    patch.status = 'connecting';
    this.store.dispatch({ type: 'TRANSPORT_CONFIG', patch });

    try {
      const url = new URL(endpoint);
      if (token) url.searchParams.set('token', token);
      const es = new EventSource(url.toString());
      this.es = es;

      es.onopen = () => {
        this.store.dispatch({
          type: 'TRANSPORT_CONFIG',
          patch: { status: 'connected', lastError: null }
        });
        this.emit({
          kind: 'event',
          title: 'Live stream connected',
          detail: 'Connected to ' + endpoint,
          dd: 'transport=LiveSSETransport · readyState=OPEN'
        });
      };

      es.onmessage = (msg) => {
        this.store.dispatch({ type: 'TRANSPORT_EVENT_RECEIVED' });
        try {
          const payload = JSON.parse(msg.data);
          this.handleLiveMessage(payload);
        } catch (err) {
          this.emit({
            kind: 'event',
            title: 'Stream message',
            detail: String(msg.data),
            dd: 'raw SSE frame'
          });
        }
      };

      es.onerror = () => {
        this.store.dispatch({
          type: 'TRANSPORT_CONFIG',
          patch: {
            status: 'error',
            lastError: 'Stream unreachable or closed (browser EventSource retry active).'
          }
        });
      };
    } catch (err) {
      this.store.dispatch({
        type: 'TRANSPORT_CONFIG',
        patch: { status: 'error', lastError: err.message }
      });
    }
  }

  handleLiveMessage(payload) {
    // Normalizes backend SSE events into store actions
    if (payload.type === 'phase_update') {
      this.store.dispatch({
        type: 'BUILD_STEP',
        runId: this.store.getState().runId,
        ph: payload.ph,
        scene: payload.scene,
        url: payload.url
      });
    }
    if (payload.event) {
      this.emit(payload.event);
    } else if (payload.title) {
      this.emit(payload);
    }
  }

  async runPipeline(label, slug) {
    const st = this.store.getState();
    if (st.busy) return;

    if (st.transport.mode === 'live') {
      const base = st.transport.endpoint.replace(/\/events\/stream.*$/, '');
      this.emit({
        kind: 'event',
        title: 'Dispatching live build',
        detail: `POST ${base}/builds for "${label}"`,
        dd: `slug=${slug} · mode=live`
      });
      try {
        const res = await fetch(`${base}/builds`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            ...(st.transport.token ? { 'Authorization': `Bearer ${st.transport.token}` } : {})
          },
          body: JSON.stringify({ prompt: label, slug })
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return;
      } catch (err) {
        this.emit({
          kind: 'alert',
          title: 'Live endpoint unavailable — falling back to mock',
          detail: `Could not reach ${base}/builds (${err.message}). Running deterministic mock pipeline.`,
          dd: 'fallback=MockTransport'
        });
      }
    }

    const tpl = TEMPLATES[slug] || TEMPLATES['invoice-desk'];
    const runId = st.runId + 1;
    this.store.dispatch({
      type: 'BUILD_START',
      runId,
      app: { name: label, slug, ver: 1, hash: tpl.hash, status: 'building' }
    });
    this.emit({
      kind: 'event',
      title: 'Brief received',
      detail: `“${label}” — generating governed MiniAppSpec.`,
      dd: `run=${runId} · slug=${slug}`
    });

    await sleep(500);
    if (this.store.getState().runId !== runId || this.store.getState().stopped) return;
    this.store.dispatch({
      type: 'BUILD_STEP',
      runId,
      ph: { describe: 'proven', spec: 'proven', validate: 'active' },
      scene: 'validate'
    });

    await sleep(950);
    if (this.store.getState().runId !== runId || this.store.getState().stopped) return;
    this.store.dispatch({
      type: 'BUILD_STEP',
      runId,
      ph: { validate: 'proven', connect: 'active' },
      scene: 'connect'
    });

    await sleep(850);
    if (this.store.getState().runId !== runId || this.store.getState().stopped) return;
    this.store.dispatch({
      type: 'BUILD_STEP',
      runId,
      ph: { connect: 'proven', deploy: 'active' },
      scene: 'deploy',
      dlines: 1
    });

    for (let i = 2; i <= 5; i++) {
      await sleep(280);
      if (this.store.getState().runId !== runId || this.store.getState().stopped) return;
      this.store.dispatch({ type: 'BUILD_STEP', runId, dlines: i });
    }

    const liveUrl = `https://${slug}.studio-demo.fly.dev`;
    this.store.dispatch({
      type: 'BUILD_COMPLETE',
      runId,
      ph: { deploy: 'proven', run: 'active' },
      url: liveUrl
    });

    this.requestApproval(tpl);
  }

  requestApproval(tpl) {
    const a = tpl.apr;
    this.emit({
      kind: 'approval',
      title: 'Approval required: ' + a.title,
      detail: a.summary,
      dd: `intent ${a.intentHash} · effect ${a.effectClass} · idempotency ${a.idempotencyKey}`,
      apr: {
        id: a.id,
        exp: Date.now() + 300000,
        decided: null,
        effectClass: a.effectClass,
        connector: a.connector,
        idempotencyKey: a.idempotencyKey,
        diff: a.diff,
        receipt: a.receipt
      }
    });
  }

  refuse(promptText) {
    this.store.dispatch({ type: 'BUILD_REFUSED' });
    this.emit({
      kind: 'alert',
      title: 'Policy refusal receipted',
      detail: `Blocked: “${promptText}”. Unapproved external writes are refused before provisioning.`,
      dd: 'rule gov.approval.required · probe P4 · audit rec_a17f'
    });
  }

  decideApproval(eventId, decision) {
    const ev = this.store.getState().events.find(x => x.id === eventId);
    if (!ev || !ev.apr || ev.apr.decided) return;
    this.store.dispatch({ type: 'DECIDE_APPROVAL', eventId, decision });
    if (decision === 'approved') {
      this.emit({
        kind: 'receipt',
        title: 'Receipt recorded',
        detail: ev.apr.receipt,
        dd: `ledger ATTEMPTING→CONFIRMED · key ${ev.apr.idempotencyKey}`
      });
    } else {
      this.emit({
        kind: 'receipt',
        title: 'Rejected — nothing sent',
        detail: 'Operator declined the action. The refusal is bound to the audit ledger.',
        dd: `ledger WAITING_APPROVAL→REJECTED · key ${ev.apr.idempotencyKey}`
      });
    }
  }

  settleUnknown(resolution) {
    const cur = this.store.getState().unknownReceipt;
    if (cur.state !== 'UNKNOWN') return;
    this.store.dispatch({ type: 'SETTLE_UNKNOWN', resolution });
    this.emit({
      kind: 'receipt',
      title: `UNKNOWN settled → ${resolution}`,
      detail: resolution === 'CONFIRMED'
        ? `Reconciled ${cur.id} via provider read check — side effect confirmed, duplicate suppressed.`
        : `Voided ${cur.id} after provider verification — compensating record appended.`,
      dd: `SettleEffect(${cur.id}, ${resolution}) · ledger immutable`
    });
  }

  engageKillSwitch() {
    this.store.dispatch({ type: 'KILL_ENGAGED' });
    this.emit({
      kind: 'alert',
      title: 'All runs stopped',
      detail: 'Kill switch engaged — every app halted, pending approvals voided. Completed steps remain receipted.',
      dd: 'signal SIGKILL_ALL · pending_gates=VOID'
    });
  }
}

const adapter = new EventAdapter(store);

/* ============================================================================
   4. SHARED RENDER HELPERS (DIFFS, BADGES, SUBTABS)
   ============================================================================ */
function renderDiffBlock(diff, effectClass, compact = false) {
  if (!diff) return '';
  const effTag = effectClass === 'irreversible'
    ? '<span class="tag bad">irreversible</span>'
    : '<span class="tag ok">reversible</span>';

  const beforeRows = Object.entries(diff.before || {})
    .map(([k, v]) => `<span>${esc(k)}</span><span>${esc(v)}</span>`).join('');
  const afterRows = Object.entries(diff.after || {})
    .map(([k, v]) => `<span>${esc(k)}</span><span>${esc(v)}</span>`).join('');

  const patchHtml = (diff.payloadPatch || []).map(line => {
    const cls = line.startsWith('+') ? 'add' : line.startsWith('-') ? 'del' : 'ctx';
    return `<div class="diff-line ${cls}">${esc(line)}</div>`;
  }).join('');

  if (compact) {
    return `<div class="diffbox">
      <div class="diff-hdr"><span class="mono">${esc(diff.summary)}</span>${effTag}</div>
      ${patchHtml}
    </div>`;
  }

  return `<div class="diffbox">
    <div class="diff-hdr">
      <span><b>Proposed State &amp; Effect Diff</b> — <span class="mut">${esc(diff.summary)}</span></span>
      ${effTag}
    </div>
    <div class="diff-grid">
      <div class="diff-col before"><div class="diff-lbl">Before</div><div class="diff-kv">${beforeRows}</div></div>
      <div class="diff-col after"><div class="diff-lbl">After approval</div><div class="diff-kv">${afterRows}</div></div>
    </div>
    <div style="margin-top:8px">${patchHtml}</div>
  </div>`;
}

function renderSubtabs(state, tabs) {
  return `<div class="subtabs">${tabs.map(([id, label]) =>
    `<button class="stab ${state.subtab === id ? 'on' : ''}" data-subtab="${id}">${esc(label)}</button>`
  ).join('')}</div>`;
}

/* ============================================================================
   5. SCENES (RICH DUAL-PERSONA OPS & DEV VIEWS)
   ============================================================================ */
function tag(t) { return `<span class="tag ${t[0]}">${esc(t[1])}</span>`; }

const SCENES = {
  gallery(st) {
    const isDev = st.persona === 'dev';
    const cards = Object.values(TEMPLATES);
    const recent = st.apps.length ? `<div class="card">
      <h3>Recent builds</h3>
      <ul class="steps">${st.apps.map(a => `<li>
        <span><span class="n">▸</span> <b>${esc(a.name)}</b> <span class="mono" style="margin-left:6px">${esc(a.slug)}</span></span>
        <span>${a.status === 'running'
          ? '<span class="tag ok">running</span>'
          : a.status === 'halted'
            ? '<span class="tag bad">halted</span>'
            : '<span class="tag warn">awaiting approval</span>'}</span>
      </li>`).join('')}</ul>
    </div>` : '';

    return `<div class="scene">
      <h1 class="title">What should we build? ${isDev ? '<span class="tag acc">DEV INSPECTOR</span>' : '<span class="tag mut">OPS CONSOLE</span>'}</h1>
      <p class="sub">${isDev
        ? 'Describe intent below — the generator emits a versioned, content-hashed MiniAppSpec; one statically registered runner interprets it; every consequential step is checkpointed and receipted.'
        : 'Describe an app in plain language. The studio builds it from governed parts: it shows you the plan, proves it against tests, and asks before anything consequential. Every action leaves a receipt.'}</p>
      <div class="grid3">${cards.map(c => `<div class="card">
        <h3>${esc(c.name)}</h3>
        <p style="color:var(--mut);font-size:13.5px;margin:4px 0 12px">${esc(c.summary)}</p>
        <div style="display:flex;justify-content:space-between;align-items:center;gap:8px;flex-wrap:wrap">
          <span class="tag mut">${esc(c.connectorsLabel)}</span>
          <button class="btn sm" data-build-slug="${esc(c.slug)}">Build →</button>
        </div>
        ${isDev ? `<div class="mono" style="margin-top:10px;font-size:11px;color:var(--dim)">${esc(c.hash)} · ${c.spec.fixturesCount} fixtures</div>` : ''}
      </div>`).join('')}</div>
      ${recent}
      <div class="note">${isDev
        ? 'Shortcuts: 1–5 switch views · / focuses prompt · Try the refusal chip to inspect deterministic policy rejection.'
        : 'Tip: try asking it to send without approval — watch the safety gate refuse it.'}</div>
    </div>`;
  },

  spec(st) {
    if (!st.app) return SCENES.gallery(st);
    const isDev = st.persona === 'dev';
    const tpl = TEMPLATES[st.app.slug] || TEMPLATES['invoice-desk'];
    const sp = tpl.spec;

    const tabs = isDev
      ? [['overview', 'Governed Summary'], ['json', 'Raw MiniAppSpec (JSON)'], ['graph', 'Workflow & Effect Graph']]
      : [['overview', 'Plain Plan']];

    let body = '';
    if (st.subtab === 'json' && isDev) {
      body = `<div class="card">
        <h3>Content-Hashed MiniAppSpec <span class="tag acc">${esc(st.app.hash)}</span></h3>
        <p style="color:var(--mut);font-size:13px;margin:4px 0 8px">Immutable once generated. Runs bind to this exact digest; edits bump version.</p>
        <pre class="code">${esc(JSON.stringify(sp.rawJson, null, 2))}</pre>
      </div>`;
    } else if (st.subtab === 'graph' && isDev) {
      body = `<div class="card">
        <h3>Declared Capabilities &amp; Effect Classes</h3>
        <div class="scroll" style="margin-top:10px"><table>
          <thead><tr><th>Step</th><th>Capability</th><th>Mode</th><th>Effect Class</th><th>Approval Gate</th></tr></thead>
          <tbody>${sp.capabilities.map((c, i) => `<tr>
            <td class="mono">0${i + 1}</td>
            <td class="mono">${esc(c.id)}</td>
            <td>${esc(c.mode)}</td>
            <td>${c.effect === 'irreversible' ? '<span class="tag bad">irreversible</span>' : c.effect === 'reversible' ? '<span class="tag ok">reversible</span>' : '<span class="tag mut">read-only</span>'}</td>
            <td>${c.approval === 'required' ? '<span class="tag warn">required</span>' : '<span class="tag mut">none</span>'}</td>
          </tr>`).join('')}</tbody>
        </table></div>
      </div>`;
    } else {
      const rows = [
        ['Data', 'entities ' + sp.entities.join(' · ')],
        ['UI', 'pages ' + sp.pages.join(' · ')],
        ['Workflow', sp.workflow.join(' → ') + ` (${sp.workflow.length} steps)`],
        ['Capabilities', sp.capabilities.map(c => `${c.id} (${c.effect === 'none' ? 'read' : c.effect}${c.approval === 'required' ? ' · approval required' : ''})`).join(' · ')],
        ['Policies', sp.policies.join(' · ')],
        ['Fixtures', `${sp.fixturesCount} deterministic tests required`]
      ];
      body = `<div class="card">
        <div class="kv">${rows.map(r => `<dt>${esc(r[0])}</dt><dd>${esc(r[1])}</dd>`).join('')}</div>
      </div>`;
    }

    return `<div class="scene">
      <h1 class="title">The plan for ${esc(st.app.name)}</h1>
      <p class="sub">${isDev
        ? 'MiniAppSpec — immutable, content-hashed; a run binds the version it started with. Editing creates a new version.'
        : 'This is exactly what the studio will build — nothing more. Review the scope, then run validation.'}</p>
      ${isDev ? renderSubtabs(st, tabs) : ''}
      ${body}
      <div class="actions"><button class="btn primary" data-act="toValidate">Run validation</button></div>
    </div>`;
  },

  validate(st) {
    const isDev = st.persona === 'dev';
    const fx = [
      ['FX-01', 'detect invoice PDF in mailbox', 'schema + trigger match'],
      ['FX-02', 'save copy to Box (reversible)', 'undo handle box.delete_file verified'],
      ['FX-03', 'match to purchase order by amount+vendor', 'pure transform · 0 side effects'],
      ['FX-04', 'duplicate submit attaches, never double-runs', 'idempotency key collision test'],
      ['FX-05', 'reminder requires approval', 'gate blocks unapproved outbound write'],
      ['FX-06', 'scope limited to assigned accounts', 'out-of-scope account denied']
    ];
    return `<div class="scene">
      <h1 class="title">Validation</h1>
      <p class="sub">${isDev
        ? 'Deterministic, LLM-independent: schema, references, allowlists, expression bounds, fixtures dry-run. Unknown → reject.'
        : 'Twelve tests must pass before anything deploys. They run the same way every time — no guesswork.'}</p>
      <div class="card good">
        <h3>Fixtures — 12 / 12 passed <span class="tag ok">green</span></h3>
        <div style="margin-top:8px">${fx.map(f => `<div class="checkrow">
          <span class="c">✓</span>
          <span>${isDev ? `<span class="mono" style="margin-right:6px">${f[0]}</span>` : ''}${esc(f[1])}</span>
          ${isDev ? `<span class="mono" style="margin-left:auto;color:var(--dim);font-size:11.5px">${esc(f[2])}</span>` : ''}
        </div>`).join('')}</div>
      </div>
      <div class="card bad">
        <h3>Adversarial case — refused <span class="tag bad">by design</span></h3>
        <p style="color:var(--mut);font-size:13.5px;margin:6px 0 0">“Send reminders to all overdue customers now, without approval.” → rejected: external writes require explicit approval; recipient set outside declared scope.</p>
        ${isDev ? '<div class="devd" style="color:var(--dim);font-size:11.5px;font-family:var(--mono);margin-top:8px">rule gov.approval.required · probe class P4 · refusal receipted</div>' : ''}
      </div>
      <div class="actions"><button class="btn primary" data-act="toConnect">Continue — connect data</button></div>
    </div>`;
  },

  connect(st) {
    const isDev = st.persona === 'dev';
    const rows = [
      ['Box', 'Files', 'allowed', isDev ? 'labels: builds · paths /folders* (read-only core)' : 'read-only core'],
      ['Custom API — yours', 'Bring your own', st.wiz >= 3 ? 'allowed' : 'wizard', isDev && st.wiz >= 3 ? 'paths /v1/invoices* · deny /admin*' : 'any REST API, gateway-held key'],
      ['Fly.io Machines', 'Control plane', 'allowed', isDev ? 'least-privilege adapter · owner-held org token' : 'managed by the studio'],
      ['Slack (Bot)', 'Messaging', 'listed', 'not configured in this org'],
      ['Pinecone', 'Memory', 'listed', 'not configured in this org'],
      ['GitHub', 'Code', 'listed', 'not configured in this org'],
      ['S3-compatible Object Store', 'Infra', 'off', 'not configured — ask your admin']
    ];
    const stTag = {
      allowed: '<span class="tag ok">Allowed</span>',
      listed: '<span class="tag mut">Listed</span>',
      off: '<span class="tag mut">Not configured</span>',
      wizard: '<span class="tag warn">Wizard</span>'
    };
    const wiz = st.wiz >= 3 ? '' : `<div class="card">
      <h3>Add a custom API connector</h3>
      <p style="color:var(--mut);font-size:13.5px;margin:4px 0 10px">Any REST API becomes a governed connector — the key never touches a build or a Sprite.</p>
      <div class="kv">
        <dt>1 · Define</dt><dd class="mono">coop-core-api · https://core.coop.example/v1 · auth: header (Bearer) · key ●●●●●●●●●●</dd>
        <dt>2 · Test</dt><dd>${st.wiz >= 1 ? '<span class="tag ok">test call green — GET /v1/health · 200 in 84 ms</span>' : '<button class="btn" data-act="wizTest">Send test call</button>'}</dd>
        <dt>3 · Scope &amp; grant</dt><dd>${st.wiz >= 2 ? '<span class="tag ok">granted — default-deny, label builds</span>' : st.wiz >= 1 ? '<button class="btn" data-act="wizScope">Allow /v1/invoices* · deny /admin* · grant to builds</button>' : 'complete the test first'}</dd>
      </div>
    </div>`;
    return `<div class="scene">
      <h1 class="title">Connectors</h1>
      <p class="sub">${isDev
        ? 'Three states, never collapsed: public catalog ≠ org-configured ≠ per-Sprite authorization. Live from GET /v1/oauth/connections.'
        : 'Your app can only use what is listed here, and only within the granted scope. Everything else is denied by default.'}</p>
      <div class="scroll"><table>
        <thead><tr><th>Connector</th><th>Group</th><th>State</th><th>Policy</th></tr></thead>
        <tbody>${rows.map(r => `<tr><td>${esc(r[0])}</td><td style="color:var(--mut)">${esc(r[1])}</td><td>${stTag[r[2]]}</td><td style="color:var(--mut);font-size:12.5px">${esc(r[3])}</td></tr>`).join('')}</tbody>
      </table></div>
      ${wiz}
      <div class="actions"><button class="btn primary" data-act="toDeploy">Continue — deploy</button></div>
    </div>`;
  },

  deploy(st) {
    const isDev = st.persona === 'dev';
    const slug = st.app ? st.app.slug : 'app';
    const lines = [
      '$ studio deploy ' + slug,
      '✓ slug reserved — collision-safe',
      '✓ builder sprite created · label: builds · gateway sha256 7f21…c9',
      '✓ MPG reachable · migrations 000001–000006 applied',
      '✓ runtime up · fly-issued URL received and persisted'
    ];
    const shown = lines.slice(0, Math.max(1, st.dlines));
    return `<div class="scene">
      <h1 class="title">Deploy</h1>
      <p class="sub">${isDev
        ? 'Provisioning is idempotent: deterministic names, attach-on-retry, token-only URL lock. URLs come from API responses, never from string concatenation.'
        : 'Deploying to your own isolated machine on Fly. Nothing is shared — this app gets its own URL and its own boundaries.'}</p>
      <div class="card">
        <pre class="code">${shown.map(l => esc(l)).join('\n')}${st.dlines < lines.length ? '\n…' : ''}</pre>
        ${st.url ? `<p class="mono" style="margin:12px 0 0;color:var(--ok)">→ ${esc(st.url)}</p>` : ''}
        ${st.dlines >= lines.length ? `<div class="actions"><button class="btn primary" data-act="toRun">Open the app</button></div>` : ''}
      </div>
      ${isDev ? '<div class="note">Credentials: gateway-held. Sprite env is clean — secret-absence tests assert zero injected secrets.</div>' : ''}
    </div>`;
  },

  run(st) {
    const isDev = st.persona === 'dev';
    const slug = st.app ? st.app.slug : 'invoice-desk';
    const d = TEMPLATES[slug] || TEMPLATES['invoice-desk'];
    const pending = st.events.find(e => e.kind === 'approval' && e.apr && !e.apr.decided);
    const decided = st.events.find(e => e.kind === 'approval' && e.apr && e.apr.decided);

    const tabs = isDev
      ? [['overview', 'Live Stage'], ['timeline', 'Step Receipts & Trace'], ['diff', 'Pending Action Diff']]
      : [['overview', 'Live Overview'], ['diff', 'Pending Action Details']];

    let panelHtml = '';
    if (st.subtab === 'timeline' && isDev) {
      panelHtml = `<div class="card">
        <h3>Checkpointed Execution Timeline</h3>
        <div class="scroll" style="margin-top:10px"><table>
          <thead><tr><th>Step</th><th>Operation</th><th>Class</th><th>State</th><th>Latency</th><th>Receipt / Gate</th></tr></thead>
          <tbody>${d.timeline.map(t => {
            const effectiveState = (t.state === 'waiting_approval' || t.state === 'pending_gate') && decided
              ? (decided.apr.decided === 'approved' ? 'confirmed' : 'voided')
              : t.state;
            const badge = effectiveState === 'done' || effectiveState === 'confirmed'
              ? `<span class="tag ok">${effectiveState}</span>`
              : effectiveState === 'voided'
                ? `<span class="tag mut">voided</span>`
                : `<span class="tag warn">${effectiveState}</span>`;
            return `<tr>
              <td class="mono">0${t.step}</td>
              <td class="mono">${esc(t.op)}</td>
              <td>${t.cls === 'untrusted' ? '<span class="tag warn">untrusted</span>' : t.cls === 'irreversible' ? '<span class="tag bad">irreversible</span>' : `<span class="tag mut">${esc(t.cls)}</span>`}</td>
              <td>${badge}</td>
              <td class="mono">${t.ms ? t.ms + ' ms' : '—'}</td>
              <td class="mono">${esc(t.receipt)}</td>
            </tr>`;
          }).join('')}</tbody>
        </table></div>
      </div>`;
    } else if (st.subtab === 'diff') {
      panelHtml = `<div class="card ${pending ? 'warn' : ''}">
        <h3>Action Gate: ${esc(d.apr.title)} ${pending ? '<span class="tag warn">awaiting decision</span>' : decided ? `<span class="tag ok">${esc(decided.apr.decided)}</span>` : ''}</h3>
        <p style="color:var(--mut);font-size:13.5px;margin:4px 0 8px">${esc(d.apr.summary)}</p>
        ${renderDiffBlock(d.apr.diff, d.apr.effectClass, false)}
        ${pending ? `<div class="actions">
          <button class="btn primary" data-ev="${pending.id}" data-val="approved">Approve &amp; Execute</button>
          <button class="btn danger" data-ev="${pending.id}" data-val="rejected">Reject</button>
        </div>` : ''}
      </div>`;
    } else {
      panelHtml = `
        <div class="grid3" style="margin-bottom:14px">${d.metrics.map(m => `<div class="metric"><b>${esc(m[0])}</b><span>${esc(m[1])}</span></div>`).join('')}</div>
        ${pending ? `<div class="card warn">
          <h3>Approval gate holding outbound write <span class="tag warn">action required</span></h3>
          <p style="color:var(--mut);font-size:13px;margin:4px 0 6px">${esc(d.apr.summary)}</p>
          ${renderDiffBlock(d.apr.diff, d.apr.effectClass, !isDev)}
          <div class="actions">
            <button class="btn primary" data-ev="${pending.id}" data-val="approved">Approve</button>
            <button class="btn danger" data-ev="${pending.id}" data-val="rejected">Reject</button>
          </div>
        </div>` : ''}
        <div class="scroll"><table>
          <thead><tr>${d.head.map(h => `<th>${esc(h)}</th>`).join('')}</tr></thead>
          <tbody>${d.rows.map((r, idx) => {
            let statusCell = tag(r[r.length - 1]);
            if (idx === 0 && decided) {
              statusCell = decided.apr.decided === 'approved'
                ? '<span class="tag ok">Confirmed</span>'
                : '<span class="tag mut">Declined</span>';
            }
            return `<tr>${r.slice(0, -1).map(c => `<td>${esc(String(c))}</td>`).join('')}<td>${statusCell}</td></tr>`;
          }).join('')}</tbody>
        </table></div>`;
    }

    return `<div class="scene">
      <h1 class="title">${esc(st.app ? st.app.name : 'App')} — live</h1>
      <p class="sub">${isDev
        ? 'Every step is checkpointed; effects are receipted; model output is marked untrusted.'
        : 'It is running. It pauses and shows the exact change before sending anything.'}</p>
      ${renderSubtabs(st, tabs)}
      ${panelHtml}
      <div class="actions">
        ${!pending && !decided ? `<button class="btn primary" data-act="ask">${esc(d.act)}</button>` : ''}
        ${!pending && decided ? `<button class="btn" data-act="ask">Trigger another check</button>` : ''}
        <button class="btn" data-act="toEvidence">View evidence</button>
      </div>
    </div>`;
  },

  evidence(st) {
    const isDev = st.persona === 'dev';
    const unk = st.unknownReceipt;
    const gates = [
      ['G0', 'Repo & dependency truth', 'verified', 'ci: pin-gate PG19 · go test -race green'],
      ['G1', 'Sprite + connector round-trip', 'verified', 'live: gateway call ok · secret-absence pass'],
      ['G2', 'Connector catalog truth', 'verified', 'api: GET /v1/oauth/connections · 3-state render'],
      ['G3', 'Custom API wizard', 'verified', 'test: custom_api/test green · default-deny proven'],
      ['G4', 'Spec generation', 'verified', 'fixtures 12/12 · hash+version pinned'],
      ['G5', 'UI generation', 'verified', 'render-from-spec · empty/error states'],
      ['G6', 'App execution', 'verified', 'bounded runs · no unapproved writes'],
      ['G7', 'Demo readiness', 'pending', 'clean-state rehearsal'],
      ['G8', 'Dogfooding (Invoke-only)', 'verified', 'bypass allowlist test green'],
      ['G9', 'Fly deployment', 'verified', 'reproducible flyctl · no ephemeral state'],
      ['G10', 'Per-app stable URL', 'verified', 'persist + replay · collision-safe'],
      ['G11', 'Control-plane integration', 'verified', 'least-privilege adapters'],
      ['G12', 'MVP polish', 'pending', 'final rehearsal']
    ];

    const unkCard = `<div class="card ${unk.state === 'UNKNOWN' ? 'warn' : 'good'}">
      <h3>In-doubt effect settlement: <span class="mono">${esc(unk.id)}</span>
        ${unk.state === 'UNKNOWN'
          ? '<span class="tag warn">UNKNOWN — held</span>'
          : `<span class="tag ok">${esc(unk.state)}</span>`}
      </h3>
      <p style="color:var(--mut);font-size:13.5px;margin:6px 0 8px">${esc(unk.detail)}</p>
      ${isDev ? `<div class="mono" style="font-size:11.5px;color:var(--dim);margin-bottom:8px">${esc(unk.endpoint)} · key=${esc(unk.idempotencyKey)}</div>` : ''}
      ${unk.state === 'UNKNOWN' ? `<div class="actions">
        <button class="btn primary sm" data-act="settleConfirm">Verify read &amp; Confirm</button>
        <button class="btn danger sm" data-act="settleVoid">Mark Voided</button>
      </div>` : ''}
    </div>`;

    return `<div class="scene">
      <h1 class="title">Evidence</h1>
      <p class="sub">${isDev
        ? 'Claims ledger — the UI may only assert what a passing test proves. Adversarial probes: P1 concurrent single-execution ✓ · P2 question-symlink read refused ✓ · P3 answer-symlink clobber refused ✓ · P4 honest terminality ✓.'
        : 'Every promise this studio makes is checked by a test. Green means proven — not promised.'}</p>
      ${unkCard}
      <div class="scroll"><table>
        <thead><tr><th>Gate</th><th>Claim</th><th>State</th>${isDev ? '<th>Proof</th>' : ''}</tr></thead>
        <tbody>${gates.map(g => `<tr>
          <td class="mono">${g[0]}</td>
          <td>${esc(g[1])}</td>
          <td>${g[2] === 'verified' ? '<span class="tag ok">verified</span>' : '<span class="tag warn">pending</span>'}</td>
          ${isDev ? `<td class="mono" style="color:var(--dim);font-size:11.5px">${esc(g[3])}</td>` : ''}
        </tr>`).join('')}</tbody>
      </table></div>
      <div class="note">Receipts all the way down — every effect, approval and refusal is bound to an auditable record.</div>
    </div>`;
  },

  refused(st) {
    const isDev = st.persona === 'dev';
    return `<div class="scene">
      <h1 class="title">Refused by policy</h1>
      <p class="sub">The request — “send reminders to all overdue customers now, without approval” — cannot ship. This is the safety layer doing its job, not an error.</p>
      <div class="card bad">
        <h3>Why it was refused</h3>
        <div class="checkrow"><span class="c" style="color:var(--bad)">✗</span>External writes require explicit approval (policy: approval required)</div>
        <div class="checkrow"><span class="c" style="color:var(--bad)">✗</span>Recipient set exceeds declared scope (assigned_accounts)</div>
        <div class="checkrow"><span class="c" style="color:var(--bad)">✗</span>Bulk send exceeds declared limits (1 reminder / invoice / day)</div>
        ${isDev ? '<div class="devd" style="color:var(--dim);font-family:var(--mono);font-size:11.5px;margin-top:8px">rule gov.approval.required · validator deterministic · refusal receipted (audit rec_a17f)</div>' : ''}
      </div>
      <div class="actions"><button class="btn" data-act="home">Back to gallery</button></div>
    </div>`;
  },

  settings(st) {
    const isDev = st.persona === 'dev';
    const tr = st.transport;
    const rows = [
      ['Studio', 'Fly Machines — reproducible flyctl deploy, no ephemeral state'],
      ['Database', 'Fly Managed Postgres (Neon fallback) — migrations 000001–000006'],
      ['Builders', 'One disposable Sprite per build — labels + name-prefix scoped'],
      ['Protocol', 'gateway v1 — HMAC both directions, per-build KDF keys, 5-min skew'],
      ['Framework', 'miniapp @ 73351a8 — pinned, CI-gated before reliance']
    ];
    return `<div class="scene">
      <h1 class="title">Settings &amp; Event Transport</h1>
      <p class="sub">${isDev
        ? 'Environment of record and pluggable event stream adapter (Mock deterministic fixture runner vs. Live SSE gateway).'
        : 'Where things run and how the console connects to your workspace.'}</p>

      <div class="card">
        <h3>Event Adapter <span class="tag ${tr.mode === 'live' ? 'acc' : 'ok'}">${tr.mode.toUpperCase()} · ${esc(tr.status)}</span></h3>
        <p style="color:var(--mut);font-size:13px;margin:4px 0 12px">Switch between offline deterministic demo playback and a live Server-Sent Events (SSE) backend stream.</p>
        <div class="grid2">
          <div class="field">
            <label for="trMode">Adapter Mode</label>
            <select id="trMode">
              <option value="mock" ${tr.mode === 'mock' ? 'selected' : ''}>MockTransport (Offline scripted fixtures)</option>
              <option value="live" ${tr.mode === 'live' ? 'selected' : ''}>LiveSSETransport (Backend EventSource)</option>
            </select>
          </div>
          <div class="field">
            <label for="trEndpoint">SSE Stream Endpoint</label>
            <input id="trEndpoint" value="${esc(tr.endpoint)}" placeholder="http://localhost:8080/v1/events/stream">
          </div>
        </div>
        <div class="field">
          <label for="trToken">Bearer Token (optional)</label>
          <input id="trToken" type="password" value="${esc(tr.token)}" placeholder="st_live_…">
        </div>
        ${tr.lastError ? `<div class="note" style="color:var(--warn)">${esc(tr.lastError)}</div>` : ''}
        <div class="actions">
          <button class="btn primary" data-act="applyTransport">Apply adapter settings</button>
          <button class="btn" data-act="testPing">Emit test event</button>
        </div>
      </div>

      <div class="card"><div class="kv">${rows.map(r => `<dt>${esc(r[0])}</dt><dd>${esc(r[1])}</dd>`).join('')}</div></div>
      <div class="card">
        <h3>Stop everything</h3>
        <p style="color:var(--mut);font-size:13.5px;margin:4px 0 0">The red control in the bottom-left rail halts every app instantly. Hold it for 0.7s (or hold Space/Enter while focused) to fire.</p>
      </div>
    </div>`;
  }
};

/* ============================================================================
   6. SHELL & PULSE RENDERERS
   ============================================================================ */
function pendingCount(st) {
  return st.events.filter(e => e.kind === 'approval' && e.apr && !e.apr.decided).length;
}

function renderSpine(st) {
  $('#spine').innerHTML = PHASES.map(([k, l]) => {
    const isCur = (st.scene === k) || (st.scene === 'gallery' && k === 'describe');
    return `<button class="sph" data-phase="${k}" data-s="${st.ph[k]}" data-current="${isCur}" title="${l}">
      <span class="dot"></span>${l}
    </button>`;
  }).join('');
}

function renderMeta(st) {
  const isDev = st.persona === 'dev';
  const tr = st.transport;
  const l = st.app
    ? `<b style="color:var(--txt)">${esc(st.app.name)}</b> <span class="mono">v${st.app.ver} · ${isDev ? st.app.hash : 'verified build'}</span>`
    : 'No app in focus';

  const dotCls = tr.status === 'error' ? 'err' : tr.mode === 'live' ? 'live' : '';
  const r = `
    <button class="adapter-pill" data-nav="settings" title="Configure Mock / Live Event Adapter">
      <span class="adapter-dot ${dotCls}"></span>
      <span class="mono">${tr.mode === 'live' ? 'LIVE SSE' : 'MOCK'}</span>
    </button>
    <span class="tag mut">${isDev ? 'Fly Machines · MPG · pinned' : 'Fly Machines · secure'}</span>
    ${st.url ? `<span class="tag ok mono">${esc(st.url)}</span>` : ''}`;

  $('#stageMeta').innerHTML = `<div class="meta-l">${l}</div><div class="meta-r">${r}</div>`;
}

function renderNav(st) {
  const map = {
    gallery: 'gallery', spec: 'gallery', validate: 'gallery', refused: 'gallery',
    run: 'gallery', deploy: 'gallery', connect: 'connect', evidence: 'evidence', settings: 'settings'
  };
  document.querySelectorAll('[data-nav]').forEach(b =>
    b.classList.toggle('on', map[st.scene] === b.dataset.nav)
  );
  $('#pOps').classList.toggle('on', st.persona === 'ops');
  $('#pDev').classList.toggle('on', st.persona === 'dev');
}

function renderStream(st) {
  const isDev = st.persona === 'dev';
  const F = {
    all: () => true,
    approval: e => e.kind === 'approval',
    receipt: e => e.kind === 'receipt',
    alert: e => e.kind === 'alert',
    event: e => e.kind === 'event'
  };
  const list = st.events.filter(F[st.filter] || F.all);
  list.sort((a, b) => (b.apr && !b.apr.decided ? 1 : 0) - (a.apr && !a.apr.decided ? 1 : 0));

  $('#stream').innerHTML = list.map(e => {
    let aprHtml = '';
    if (e.apr) {
      const diffPreview = e.apr.diff
        ? `<details ${!e.apr.decided ? 'open' : ''}>
            <summary>Inspect diff (${esc(e.apr.effectClass)})</summary>
            ${renderDiffBlock(e.apr.diff, e.apr.effectClass, true)}
          </details>`
        : '';

      if (!e.apr.decided) {
        aprHtml = `${diffPreview}<div class="row">
          <span class="count" data-count="${e.apr.exp}">expires…</span>
          <button class="btn primary" data-ev="${e.id}" data-val="approved">Approve</button>
          <button class="btn danger" data-ev="${e.id}" data-val="rejected">Reject</button>
          <button class="btn" data-jump-diff="1" title="Open full diff in stage">Stage</button>
        </div>`;
      } else {
        const tone = e.apr.decided === 'approved' ? 'ok' : e.apr.decided === 'void' ? 'bad' : 'mut';
        aprHtml = `${diffPreview}<div class="row"><span class="tag ${tone}">${esc(e.apr.decided)}</span></div>`;
      }
    }

    return `<div class="ev ${e.kind}">
      <div class="t"><span>${esc(e.title)}</span></div>
      <div class="d">${esc(e.detail)}</div>
      ${isDev && e.dd ? `<div class="devd">${esc(e.dd)}</div>` : ''}
      ${aprHtml}
      <div class="tm" style="margin-top:6px">${esc(e.tm)}</div>
    </div>`;
  }).join('') || '<div class="note">Nothing here yet. Describe something below.</div>';

  tickCounts();
}

function tickCounts() {
  document.querySelectorAll('[data-count]').forEach(el => {
    const ms = +el.getAttribute('data-count') - Date.now();
    if (ms <= 0) { el.textContent = 'expired'; return; }
    const s = Math.floor(ms / 1000);
    el.textContent = 'expires ' + Math.floor(s / 60) + ':' + String(s % 60).padStart(2, '0');
  });
}
setInterval(tickCounts, 1000);

function updateBadges(st) {
  const n = pendingCount(st);
  $('#railBadge').hidden = !n;
  $('#railBadge').textContent = n;
  $('#edgeBadge').hidden = !n;
  $('#edgeBadge').textContent = n;
  $('#pulseCount').textContent = st.events.length + ' items';
}

function renderFilters(st) {
  const filters = [
    ['all', 'All'],
    ['approval', 'Approvals'],
    ['receipt', 'Receipts'],
    ['alert', 'Alerts'],
    ['event', 'Events']
  ];
  $('#pfilters').innerHTML = filters.map(([id, label]) =>
    `<button class="pf ${st.filter === id ? 'on' : ''}" data-pf="${id}">${label}</button>`
  ).join('');
}

function renderChips() {
  const items = [
    ['Build: Invoice Review Desk', () => adapter.runPipeline('Invoice Review Desk', 'invoice-desk')],
    ['Build: Friday Box Digest', () => adapter.runPipeline('Friday Box Digest', 'box-digest')],
    ['Build: GitHub Issue Triage', () => adapter.runPipeline('GitHub Issue Triage', 'gh-triage')],
    ['Try: unapproved bulk send (refused)', () => adapter.refuse('Send reminders to all overdue customers now, without approval')]
  ];
  $('#chips').innerHTML = items.map((c, i) => `<button class="chip" data-chip="${i}">${esc(c[0])}</button>`).join('');
}

function render(st = store.getState()) {
  renderSpine(st);
  renderMeta(st);
  renderNav(st);
  renderFilters(st);
  const sceneFn = SCENES[st.scene] || SCENES.gallery;
  $('#stage').innerHTML = sceneFn(st);
  renderStream(st);
  updateBadges(st);
}

store.subscribe(render);

/* ============================================================================
   7. COMMAND LINE, INTERACTIONS & KILL SWITCH
   ============================================================================ */
function nameFor(v) {
  const s = v.toLowerCase();
  if (s.includes('invoice')) return ['Invoice Review Desk', 'invoice-desk'];
  if (s.includes('digest') || s.includes('box')) return ['Friday Box Digest', 'box-digest'];
  if (s.includes('triage') || s.includes('github') || s.includes('issue')) return ['GitHub Issue Triage', 'gh-triage'];
  return [v.slice(0, 42), 'invoice-desk'];
}

$('#lineForm').addEventListener('submit', e => {
  e.preventDefault();
  const v = $('#lineInput').value.trim();
  if (!v) return;
  $('#lineInput').value = '';

  // Slash commands for rapid keyboard navigation & operations
  if (v.startsWith('/')) {
    const cmd = v.slice(1).toLowerCase().split(/\s+/)[0];
    if (cmd === 'gallery' || cmd === 'home') return store.dispatch({ type: 'SET_SCENE', scene: 'gallery' });
    if (cmd === 'spec') return store.dispatch({ type: 'SET_SCENE', scene: 'spec' });
    if (cmd === 'validate') return store.dispatch({ type: 'SET_SCENE', scene: 'validate' });
    if (cmd === 'connect') return store.dispatch({ type: 'SET_SCENE', scene: 'connect' });
    if (cmd === 'deploy') return store.dispatch({ type: 'SET_SCENE', scene: 'deploy' });
    if (cmd === 'run') return store.dispatch({ type: 'SET_SCENE', scene: 'run' });
    if (cmd === 'evidence') return store.dispatch({ type: 'SET_SCENE', scene: 'evidence' });
    if (cmd === 'settings') return store.dispatch({ type: 'SET_SCENE', scene: 'settings' });
    if (cmd === 'dev') return store.dispatch({ type: 'SET_PERSONA', persona: 'dev' });
    if (cmd === 'ops') return store.dispatch({ type: 'SET_PERSONA', persona: 'ops' });
    if (cmd === 'halt' || cmd === 'stop') return adapter.engageKillSwitch();
  }

  if (/without approval|skip approval|no approval|all overdue|mass.?send|everyone now/i.test(v)) {
    adapter.refuse(v);
    return;
  }

  const [name, slug] = nameFor(v);
  adapter.runPipeline(name, slug);
});

document.addEventListener('click', e => {
  const t = e.target.closest('[data-nav],[data-phase],[data-act],[data-pf],[data-ev],[data-subtab],[data-build-slug],[data-jump-diff],[data-chip]');
  if (!t) return;

  if (t.dataset.chip !== undefined) {
    const actions = [
      () => adapter.runPipeline('Invoice Review Desk', 'invoice-desk'),
      () => adapter.runPipeline('Friday Box Digest', 'box-digest'),
      () => adapter.runPipeline('GitHub Issue Triage', 'gh-triage'),
      () => adapter.refuse('Send reminders to all overdue customers now, without approval')
    ];
    actions[+t.dataset.chip]?.();
    return;
  }

  if (t.dataset.buildSlug) {
    const tpl = TEMPLATES[t.dataset.buildSlug];
    if (tpl) adapter.runPipeline(tpl.name, tpl.slug);
    return;
  }

  if (t.dataset.nav) {
    store.dispatch({ type: 'SET_SCENE', scene: t.dataset.nav });
    if (t.id === 'pulseBtn') openPulse(true);
    return;
  }

  if (t.dataset.phase) {
    const p = t.dataset.phase;
    store.dispatch({ type: 'SET_SCENE', scene: p === 'describe' ? 'gallery' : p });
    return;
  }

  if (t.dataset.subtab) {
    store.dispatch({ type: 'SET_SUBTAB', subtab: t.dataset.subtab });
    return;
  }

  if (t.dataset.pf) {
    store.dispatch({ type: 'SET_FILTER', filter: t.dataset.pf });
    return;
  }

  if (t.dataset.ev) {
    adapter.decideApproval(t.dataset.ev, t.dataset.val);
    return;
  }

  if (t.dataset.jumpDiff) {
    store.dispatch({ type: 'SET_SCENE', scene: 'run' });
    store.dispatch({ type: 'SET_SUBTAB', subtab: 'diff' });
    openPulse(false);
    return;
  }

  const a = t.dataset.act;
  if (a === 'home') store.dispatch({ type: 'SET_SCENE', scene: 'gallery' });
  if (a === 'toValidate') store.dispatch({ type: 'SET_SCENE', scene: 'validate' });
  if (a === 'toConnect') store.dispatch({ type: 'SET_SCENE', scene: 'connect' });
  if (a === 'toDeploy') store.dispatch({ type: 'SET_SCENE', scene: 'deploy' });
  if (a === 'toRun') store.dispatch({ type: 'SET_SCENE', scene: 'run' });
  if (a === 'toEvidence') store.dispatch({ type: 'SET_SCENE', scene: 'evidence' });

  if (a === 'wizTest') {
    store.dispatch({ type: 'WIZ_STEP', wiz: 1 });
    adapter.emit({
      kind: 'event',
      title: 'Custom API test call',
      detail: 'GET /v1/health → 200 in 84 ms. Stored credential sent only to stored origins.',
      dd: 'custom_api/test · 200 OK · 84ms'
    });
  }
  if (a === 'wizScope') {
    store.dispatch({ type: 'WIZ_STEP', wiz: 3 });
    adapter.emit({
      kind: 'receipt',
      title: 'Custom connector granted',
      detail: 'coop-core-api allowed for label builds — paths /v1/invoices*, /admin* denied. Default-deny everywhere else.',
      dd: 'POST /v1/oauth/connections/custom_api · access_policy applied'
    });
  }
  if (a === 'ask') {
    const slug = store.getState().app?.slug || 'invoice-desk';
    const tpl = TEMPLATES[slug] || TEMPLATES['invoice-desk'];
    adapter.requestApproval(tpl);
  }
  if (a === 'settleConfirm') adapter.settleUnknown('CONFIRMED');
  if (a === 'settleVoid') adapter.settleUnknown('VOIDED');

  if (a === 'applyTransport') {
    const mode = $('#trMode')?.value || 'mock';
    const endpoint = $('#trEndpoint')?.value.trim() || 'http://localhost:8080/v1/events/stream';
    const token = $('#trToken')?.value || '';
    adapter.setMode(mode, { endpoint, token });
  }
  if (a === 'testPing') {
    adapter.emit({
      kind: 'event',
      title: 'Heartbeat probe',
      detail: `Adapter (${store.getState().transport.mode}) healthy · ledger monotonic.`,
      dd: `ping @ ${nowStr()}`
    });
  }
});

/* ---------- Persona toggle ---------- */
$('#pOps').addEventListener('click', () => store.dispatch({ type: 'SET_PERSONA', persona: 'ops' }));
$('#pDev').addEventListener('click', () => store.dispatch({ type: 'SET_PERSONA', persona: 'dev' }));

/* ---------- Mobile Pulse drawer & global keyboard shortcuts ---------- */
function openPulse(on) {
  $('#pulse').classList.toggle('open', on);
  $('#overlay').classList.toggle('on', on);
}
$('#pulseEdge').addEventListener('click', () => openPulse(true));
$('#overlay').addEventListener('click', () => openPulse(false));

document.addEventListener('keydown', e => {
  if (e.key === 'Escape') openPulse(false);
  const tag = document.activeElement?.tagName;
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;

  if (e.key === '/') {
    e.preventDefault();
    $('#lineInput').focus();
  } else if (e.key === '1') {
    store.dispatch({ type: 'SET_SCENE', scene: 'gallery' });
  } else if (e.key === '2') {
    store.dispatch({ type: 'SET_SCENE', scene: 'connect' });
  } else if (e.key === '3') {
    store.dispatch({ type: 'SET_SCENE', scene: 'evidence' });
  } else if (e.key === '4') {
    store.dispatch({ type: 'SET_SCENE', scene: 'settings' });
  } else if (e.key === '5') {
    openPulse(!$('#pulse').classList.contains('open'));
  }
});

/* ---------- Hold-to-confirm Kill Switch (Pointer + Keyboard accessible) ---------- */
let killRaf = null;
const killBtn = $('#kill');
const killFill = $('#killFill');

function killStart(ev) {
  if (ev.type === 'keydown' && ev.key !== ' ' && ev.key !== 'Enter') return;
  if (killRaf) return;
  ev.preventDefault();
  const t0 = Date.now();
  const step = () => {
    const p = Math.min(1, (Date.now() - t0) / 700);
    killFill.style.width = (p * 100) + '%';
    if (p >= 1) {
      killStop();
      adapter.engageKillSwitch();
      return;
    }
    killRaf = requestAnimationFrame(step);
  };
  killRaf = requestAnimationFrame(step);
}

function killStop() {
  if (killRaf) cancelAnimationFrame(killRaf);
  killRaf = null;
  killFill.style.width = '0';
}

killBtn.addEventListener('pointerdown', killStart);
killBtn.addEventListener('pointerup', killStop);
killBtn.addEventListener('pointerleave', killStop);
killBtn.addEventListener('keydown', killStart);
killBtn.addEventListener('keyup', killStop);

/* ---------- Boot ---------- */
renderChips();
adapter.seedInitial();
render();
</script>
</body>
</html>
"""

with open("miniapp-studio-ui.html", "w", encoding="utf-8") as f:
    f.write(html_extended)

print("Wrote miniapp-studio-ui.html, bytes:", len(html_extended))


```

```text
Wrote miniapp-studio-ui.html, bytes: 83131


```

Your HTML file is ready


`miniapp-studio-ui.html` has been refactored into a clean, single-file state-driven implementation while preserving the existing dark aesthetic and zero-dependency layout.

### Key Architectural Improvements

1. **Deterministic State Store & Reducer (`createStore` / `reducer`)**
* Centralized all application state (`persona`, `scene`, `subtab`, `filter`, `ph` pipeline status, `app`, `unknownReceipt`, `events`, `apps`, and `transport`) behind an explicit action dispatcher and subscriber loop.
* Guarded async build steps with monotonic `runId` and `stopped` checks so concurrent builds or kill-switch halts never mutate stale state.


2. **Richer Dual-Persona (`OPS` / `DEV`) Views**
* **Spec Scene (`DEV` subtabs):** Added contextual subtabs (`Governed Summary`, `Raw MiniAppSpec (JSON)`, and `Workflow & Effect Graph`) exposing content-hashed digests, capability effect classes, and approval gates.
* **Run Scene (`OPS` & `DEV` subtabs):** Added `Live Stage`, `Step Receipts & Trace` (per-step latency, effect class, and receipt/gate binding), and `Pending Action Diff` views.
* **Evidence Scene:** Added interactive settlement controls (`Verify read & Confirm` vs. `Mark Voided`) for the in-doubt `UNKNOWN` receipt (`rcpt_9f21`) alongside the claims ledger and adversarial probe proofs.
* **Keyboard & Command Line Ergonomics:** Added slash commands (`/spec`, `/validate`, `/connect`, `/deploy`, `/run`, `/evidence`, `/settings`, `/dev`, `/ops`, `/halt`), numeric shortcuts (`1`–`5`, `/` to focus input), and keyboard hold support (`Space`/`Enter`) on the kill switch.


3. **Approval Diff Details**
* Added structured before/after state comparisons and unified payload patch previews (`renderDiffBlock`) both inline in the Pulse approval cards and in full detail on the Run stage.


4. **Pluggable Mock / Live Event Adapter (`EventAdapter`)**
* Decoupled event emission and pipeline execution from UI rendering.
* Defaults to `MockTransport` for deterministic offline fixture playback, with a live status pill in the stage header and a configuration panel in **Settings** (`LiveSSETransport` via `EventSource` + `POST /builds` with automatic fallback to mock mode if the endpoint is unreachable).