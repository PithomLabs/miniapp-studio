# Winning Fly.io Hacksgiving on top of miniapp — 12 ideas, top 3 picked

**Rules that shape every idea** (from `upload/hacksgiving.md`):
- Build a **personal agent** that (1) **runs on a Sprite** and (2) **uses at least one Sprites connector** (partners: TypeSafe AI, Box, Fly.io, Ollama, Pinecone). Everything else is open.
- Hackathon window Nov 23–29 (today is Nov 24 → **~5 build days left**), one entry per developer, submit code + demo + description, winners Dec 1.
- Prizes: Grand $2,500, Second $1,500, **Weird Uncle $1,000 for the strangest agent**. Video demos earn "bonus points."
- Judges: Scott Johnston (Fly.io) + founders/leads of agent & infra companies (TypeSafe AI, Box, Ollama, Pinecone) — they will reward **platform-native** builds and agent-safety literacy.
- "Must be built during Hacksgiving week" applies to the agent; using your own pre-existing open-source framework (miniapp) as a dependency is standard practice — the agent is the new artifact.

**The unfair advantage:** the hackathon hard-requires Sprites, and miniapp already carries `superfly/sprites-go` with a frozen `deploy.SpriteTasks` interface and `Placement` sprite/machine/hybrid in the contract. You are one of the few entrants whose *framework* was designed for the required substrate. Second advantage: miniapp's differentiators — durable waits, intent-bound approvals, effect receipts with UNKNOWN quarantine, entity serialization, kill switches — are exactly the machinery most hackathon agents fake with `time.Sleep` and hope.

Legend per idea: **connector(s)** used · miniapp machinery showcased · demo moment · build risk (L/M/H over ~5 solo days).

---

## The 12 ideas

**1. The Receipt-Keeping Butler — the personal agent that never double-fires an action.**
A general personal assistant (organize files, draft replies, run chores) where miniapp is the spine: every plan is a Class D workflow, every consequential action an effect with a receipt, sends gated behind phone approvals, ambiguous outcomes quarantined for your verdict instead of guessed.
*Connectors:* Box (personal files) + Pinecone (memory) · *Machinery:* approvals, effect ledger, durable resume · *Demo:* give it 3 chores, **kill the Sprite mid-task with `fly machine kill`**, watch it come back and finish with nothing repeated, approval cards arriving on your phone. · *Risk:* M — needs curation to not look like a generic chatbot.

**2. Fly Fleet Janitor — your personal SRE, living on the platform it babysits.**
An agent on a Sprite that watches YOUR Fly.io apps (the Fly.io connector is first-class), diagnoses misbehavior (errors, restart loops, cert expiry, scale waste), proposes fixes as approval cards, executes through the Machines API as receipted effects — restart, rescale, scale-to-zero, roll back. "The agent that pays rent to the landlord it manages."
*Connectors:* Fly.io (+ Ollama locally for cheap triage) · *Machinery:* entity serialization per app (never two concurrent repairs on one app), kill switch, blast-radius declarations · *Demo:* break an app on stage (deploy a crash), agent notices → proposes → you approve from your phone → machine restarts → receipt filed → Grafana green. · *Risk:* L–M — the Fly connector does heavy lifting; miniapp adds governance. · *Why it can win:* deepest platform-native story in the field, real infrastructure reacting live on video, and every judge understands the pain instantly.

**3. The Procrastination Enforcer — an accountability agent with actual enforcement power.**
You sign a goal contract; it uses durable waits (miniapp's native `WaitUntil`/signal machinery) to wait out deadlines, checks your evidence (Box docs, git push receipts), and on failure escalates through its approval-gated punishment ladder: emails your accountability partner, drafts a public confession, donates $5 to a cause you *hate* (sandbox). The comedy writes itself; the tech underneath (durable multi-day waits, resumable after Sprite restarts, approval-gated irreversible actions) is real.
*Connectors:* Box + email-as-effect · *Machinery:* durable sleep/signal, approvals, receipts · *Demo:* you skip the gym, the agent files the receipt of your shame. · *Risk:* L. · *Why:* Weird Uncle Award ($1,000) has few credible contenders, and this one hides a genuinely hard durability problem under the joke.

**4. Autopay Chaperone — the bill-payer that shows you a receipt before and after.**
Watches invoice PDFs in Box, extracts amount/due-date (Ollama locally), waits for your approval card, pays via a sandbox payment channel, reconciles on any ambiguity ("gateway timed out" → quarantine → ask you), files the full receipt ledger. Personal finance + safety + receipts.
*Connectors:* Box + Ollama · *Machinery:* the whole effects plane — this is the RPA thesis in one demo · *Demo:* pay three fake bills, kill the network mid-payment, watch it refuse to blindly retry and instead ask you. · *Risk:* M — payments must stay sandboxed (ToS rule 8).

**5. Inbox Chief of Staff — triage with a paper trail.**
Reads mail, clusters by priority (Pinecone), drafts replies, and **sends nothing without approval**; every send is a receipted effect, so "did the agent email my boss twice?" is answerable with ledger rows. Resumable across Sprite restarts — the outage that eats other agents' queues just pauses this one.
*Connectors:* Pinecone + Box · *Machinery:* approvals, receipts, resume · *Demo:* firehose inbox → approve 3 drafts from phone → audit view of exactly what was sent. · *Risk:* M — email credentials/IMAP plumbing eats time.

**6. Second Brain That Acts — research that ends in consequences.**
Not another research chatbot: the pipeline (search → fetch → extract → Pinecone) is durable and resumable, and the *actionable* tail (archive to Box, email a digest, book the thing) goes through approvals and receipts. Uses three connectors — maximum partner coverage.
*Connectors:* Pinecone + Box + Ollama · *Machinery:* durable pipelines, receipts · *Demo:* overnight research job survives a Sprite reboot and still delivers the digest + filed sources. · *Risk:* M — crowded category, needs the "acts, not just answers" twist to stand out.

**7. The Disposalist — a privacy agent with scorched-earth hygiene.**
For tasks you wouldn't trust to a persistent assistant (check a sketchy site, query a data broker, test an untrusted file), it spins a **fresh disposable Sprite** per task, runs an Ollama local model so nothing leaves your hardware, uploads only the receipt/evidence, then **burns the VM**. Persistent-home results, zero persistent exposure.
*Connectors:* Ollama + Fly.io · *Machinery:* placement abstraction (machine vs sprite per sensitivity), audit · *Demo:* side-by-side of a "normal" agent's leaked history vs. the Disposalist's post-burn `ls`. · *Risk:* M — Sprites-per-task orchestration is the unknown; the placement seam exists but is unimplemented (that's your build).

**8. Deep Research Bureau — the resumable literature review.**
Long-running structured research (50+ sources → categorized → cited report in Box) as checkpointed workflows: close the laptop, come back tomorrow, it's still going and nothing re-fetches. Receipts for every source fetched; determinism lint keeps runs reproducible.
*Connectors:* Pinecone + Box · *Machinery:* checkpointing, resume · *Demo:* kill the Sprite at minute 40 of an hour-long run, resume, show zero duplicate fetches. · *Risk:* M — polished research agents will be everywhere this week.

**9. Forms Gofer — the 5-day seed of your RPA platform.**
The vertical slice of the RPA build plan: headless browser inside a disposable Sprite (perfect isolation pitch), autocompletes tedious forms (expense reports, applications) from Box documents, approval card before every submit, screenshots as evidence refs, UNKNOWN quarantine when the confirmation page hangs.
*Connectors:* Box + Ollama (local extraction) · *Machinery:* effects ledger = form submissions with receipts · *Demo:* expense report filed end-to-end with a screenshot receipt for each field. · *Risk:* H in 5 days — selectors are time-hungry; scope to ONE form type done perfectly.

**10. Thanksgiving Logistics Agent — on-theme and charming.**
Coordinates the actual meal: parse shared recipes (Box), build the shopping list, budget-approve large purchases, place the grocery order (connector effect), schedule durable reminders per dish, split costs with receipts. Judges are literally eating pie.
*Connectors:* Box + Ollama · *Machinery:* approvals, receipts, durable reminders · *Demo:* "agent, dinner for 12, two vegetarians" → approved budget → order placed → reminder timeline. · *Risk:* L–M. · *Why:* memorability through theme; honest ceiling is Second Place at best.

**11. Hired Help — the contract-bound agent (a demo about agents, for agent judges).**
Two agents, same tools, one task: one is unconstrained (goes rogue: over-sends, double-fires, no receipts), one is *hired under a miniapp contract* (declared scope, rate limits, approval gates, kill switch, revocation). The side-by-side IS the pitch: governance as the product. Every judge at this table sells or builds agent tooling — they will get it in 10 seconds.
*Connectors:* Fly.io + Box · *Machinery:* literally the framework's thesis (manifest, approvals, ledger, kill switch) · *Demo:* rogue agent emails 47 people; hired help asks permission, sends 1, files a receipt, gets revoked live when it drifts. · *Risk:* M — the art is keeping the rogue agent funny rather than cringe.

**12. Renewal Watchdog — deadlines with a paper trail.**
Watches things that quietly expire (passport, domains, subscriptions, TLS certs on your hobby domains) as durable entity-serialized watches; proposes renewals as approval cards; files receipts. The most *boringly useful* idea on the list — and "useful" wins hackathons more often than clever.
*Connectors:* Box + Ollama (document reading) · *Machinery:* durable waits (this is THE use case for multi-month `WaitUntil`), entity serialization per tracked item · *Demo:* show a dashboard of 6 tracked deadlines and a receipted renewal that happened while you slept. · *Risk:* L.

---

## Top 3 picks, and why

### 🥇 Pick 1: **Fly Fleet Janitor (idea 2)** — the grand-prize play
1. **It is the most platform-native entry possible.** The hackathon's soul is Sprites; the judges' table is Fly.io plus agent-platform companies. An agent that runs on Fly, uses Fly's own connector, and manages Fly workloads through the Machines API makes the judges' own product the star — that is how you win a vendor hackathon.
2. **The recursion is the story.** "It lives on the platform it babysits" is a one-line pitch that survives being retold. Demo moments are physical and undeniable: a real machine crashes on stage, a real approval lands on your phone, a real receipt is filed.
3. **miniapp contributes exactly the hard 10% and nothing you'd waste time on.** Entity serialization per app (no concurrent double-repairs), approval gates before destructive actions, receipts and audit for every change, kill switch for the agent itself. The Fly connector + Machines API cover the rest. Buildable in the remaining ~5 days: **scope to 3 tools (diagnose, restart, scale-to-zero), 1 demo app, 1 scripted incident** — tight, honest, and complete.
4. **Risk hedge:** ops-flavored agents are a known category, so the differentiator must be the *governance spine* — lead the video with the approval card and the ledger, not the chat window.

### 🥈 Pick 2: **The Receipt-Keeping Butler (idea 1)** — the polished-product play
1. **It turns miniapp's thesis into the demo.** Every entrant this week will show an agent doing tasks; almost none can show their agent *surviving `fly machine kill` mid-task* and finishing with zero double-fired actions, because almost none have durable execution. The kill-and-resume moment is the most demoable expression of the framework on this list.
2. **It maximizes connector surface** (Box + Pinecone) and frames safety literacy — approval cards, receipts, quarantine — which resonates with judges who build agent platforms for a living. The pitch writes itself: *"Agents fail; mine files receipts."*
3. **Why second, not first:** the category is broad, which means crowded; winning requires curation and polish, and the demo must be choreographed tightly to avoid reading as "yet another assistant." Higher variance than the Janitor, higher ceiling on product-feel.

### 🥉 Pick 3: **The Procrastination Enforcer (idea 3)** — the Weird Uncle play, and the smart-money hedge
1. **It is the cheapest path to prize money in the field.** The Weird Uncle Award ($1,000) rewards strangeness, and strangeness is under-supplied relative to polished assistants. A goal-contract agent whose punishments are approval-gated and whose multi-day deadline waits *actually survive Sprite restarts* is both funny and technically deeper than the joke suggests.
2. **Durable waits are the hidden flex.** The hard part — an agent reliably waiting 6 days and then acting once, not zero or twice times — is precisely miniapp's native machinery. The comedy is the packaging; the durability is the entry fee nobody else can pay.
3. **Why third:** single-prize ceiling ($1,000 vs $2,500), and humor is judged subjectively. But as a portfolio decision, note the rules allow **one entry** — so the real choice is appetite: Janitor for the top prize, Enforcer for the highest-probability prize.

---

## Recommendation

**Submit the Fly Fleet Janitor.** It uniquely satisfies both hard requirements in a way the judges will recognize as platform-native (Sprite runtime via the placement seam your framework already froze; Fly.io connector as first-class), it demos against real infrastructure where crash-and-resume and receipted repairs are visible rather than claimed, and it scopes honestly into the ~5 days remaining: one demo app, three tools, one scripted incident, one approval card, one ledger view. Borrow two things from the other picks for the demo: the Janitor's video opens with the Butler's signature moment (kill the Sprite mid-diagnosis, show resume) and ends with one Enforcer-style flourish (the agent refuses an out-of-scope "helpful" action and files a receipt saying why). One week, one agent, receipts all the way down.
