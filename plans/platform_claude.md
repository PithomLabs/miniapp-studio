# Hacksgiving, Revision 2 — A Generic Miniapp Platform on Your Sprite

- **Date:** 2026-11-23
- **Supersedes:** the "12 ideas / build Rewind" recommendation in `hacksgiving_plan.md`. The 12 ideas survive as **platform primitives and gallery templates**, not as 12 builds.
- **New input:** the Sprites ecosystem and connectors page (Fly MCP, the API gateway, the Meta connector walkthrough, checkpoint and confirmation notes).
- **Premise from you:** the framework predates the hackathon and is only the foundation; the judges see the UI platform that lets end users generate miniapp-driven apps, GigaCatalyst-style.

---

## 1. My verdict

**Yes, take the platform direction. It is the right instinct, and it solves the problem I created with 12 separate ideas.** But two things have to change from "a generic platform" to "a platform that wins":

1. **Pitch a guarantee, not a builder.** "Build apps from a description" is the most crowded pitch in the room. The line nobody else can say: **"Apps that can't do anything you haven't seen first, and that you can always undo or stop."** Approvals, undo, audit and a kill switch are properties of the platform, not features of one app.
2. **Generate specs, not programs.** The generator (a coding agent already installed on every Sprite) writes a small declarative app spec, built only from vetted primitives. A fixed runtime executes it. That is safe, inspectable, diffable, undoable, and far easier to finish in six days than "the agent writes arbitrary code."

The 12 ideas collapse into **one platform plus a few generated demo apps** (Section 8). That is how you "give end users the primitives, UI and workflow" without building 12 apps.

**What the new connectors page adds to your argument:** it says confirmation behavior varies by plugin, and that terminal backends such as Hermes do not provide Sprites-specific confirmation hooks. That is a documented gap. Your approvals layer is **harness-agnostic**: confirmation enforced by the platform on the server side, so even harnesses without confirmation hooks get safe approvals. Use that sentence in the pitch.

---

## 2. The trap to avoid

A general platform tempts you to build everything: visual workflow editor, marketplace, multi-user accounts, billing, dozens of connectors. A one-person, six-day platform demo fails when the judge cannot see value in 90 seconds.

**The cure is one golden path, proven with three different apps:**

> Describe a chore → the platform generates a guarded app → it runs on your Sprite → it asks before it acts → you undo it → an attack fails.

Show that path **three times with three different apps** in the same video. Generality is proven by the variety, not by feature count.

---

## 3. The thesis

**Your Sprite becomes your personal app factory.** One Sprite per person. You describe a chore; the platform generates a small app for it; every app inherits **ask first, undo, audit and stop** by construction.

Keep the framing "**personal agent**," because the brief requires a personal agent that lives on a Sprite. One user per Sprite also removes multi-tenant complexity (no tenant isolation work, no accounts system) and fits Sprites' hardware isolation.

---

## 4. What the judges see (the 90-second demo)

| Time | Scene | What it proves |
|---|---|---|
| 0:00–0:10 | Hook line: "Fly made it safe to give an agent the keys. We make it safe to let it act, and to take it back." | Thesis |
| 0:10–0:35 | **Generate:** type "every Friday summarize my Box reports folder and post it to Slack, but ask me first." The agent (running on the Sprite) generates the app. A **permissions card** shows what it will touch: which connector paths, which effects, which limits | Generation, least privilege, connectors without keys |
| 0:35–0:55 | **Run and approve:** the app runs; the approval card appears on a phone; the Sprite was asleep and woke on request | Approvals, wake on request, durable wait |
| 0:55–1:10 | **Undo:** Rewind restores the Sprite to a checkpoint and reconciles the outside world; the irreversible effect is flagged "cannot unsend" | Checkpoint plus effect ledger |
| 1:10–1:25 | Generate two more apps in two prompts (a GitHub triage and a Thanksgiving coordinator) | Generality |
| 1:25–1:30 | Attack scoreboard: a booby-trapped message tries six attacks; all blocked | Guardrails |

---

## 5. Architecture

```text
                 You (browser/phone, via the Sprite URL)
                              |
+--------------------------- YOUR SPRITE (one per user) --------------------------+
|  Platform UI (served on :8080)                                                   |
|   Describe bar | Gallery | App view | Approvals | Timeline + Rewind | Permissions|
|                              |                                                   |
|  Generator: coding agent on the Sprite (Claude Code / Codex)  --writes-->  SPEC  |
|     validate: schema + static safety checks + dry run on fixtures                |
|                              |                                                   |
|  Spec Runner (one statically registered "miniapp")  <-- interprets specs         |
|     built on the miniapp framework: capability proxy, effects, approvals,        |
|     audit, limits, kill switch, durable steps                                    |
|                              |                                                   |
|  Local state: SQLite on the Sprite disk (checkpointed)                           |
+------------------------------|---------------------------------------------------+
                               | no credentials leave the gateway
                 Sprites API gateway (Connectors): Box, Slack, GitHub, Fly Machines,
                 an LLM connector, object store for the off-Sprite ledger
```

### 5.1 Key design decisions

| Decision | Why |
|---|---|
| **One generic "Spec Runner"**, registered once | The existing framework registers miniapps as compiled code. A generic interpreter lets end users create apps at run time without compiling anything: each generated app is data, not a new binary |
| **Specs are immutable and hashed** | A run pins its spec hash. Replays and recovery see the same definition; changes make a new version |
| **Deterministic interpreter** | All nondeterminism (time, randomness, network, model calls) happens inside steps whose results are recorded, the same discipline as durable workflows |
| **Off-Sprite effect ledger** | A restore rewinds the disk, so the record of what the agent did to the world must live somewhere restore cannot touch (object store through a connector, or an external database) |
| **Effects declared in the spec** | Each action declares its reversibility (reversible, compensatable, irreversible). The platform enforces approval for irreversible ones and offers compensation for the rest |
| **Permissions derived from the spec** | The platform computes the connector paths and scopes an app needs, shows them, and tells you what to grant. The Sprites gateway is default-deny per Sprite and supports allowing or blocking request paths, so least privilege is real, not decorative |
| **Generation inside a checkpoint** | The generator works between a checkpoint and a validation gate. If it produces junk, restore and retry |

### 5.2 Why this is more feasible than "the agent writes code"

Free-form code needs sandboxing, review and conformance testing. A spec needs only a schema validator and a small interpreter. The generator's feedback loop is simple: write spec, run `validate`, fix, repeat.

---

## 6. The primitives (what you give end users)

### UI primitives (the generic UI)

| Primitive | Purpose |
|---|---|
| Describe bar | Natural-language entry for creating or editing an app |
| Form | Inputs for manual triggers and parameters |
| List and table | Records, results, queues |
| Detail card | One item with actions |
| Approval card | Shows exactly what will happen, with expiry and a content-bound approve or reject |
| Run timeline | Every step, effect and checkpoint; the Rewind control |
| Status and progress | Running, waiting, needs attention |
| Permissions card | What the app touches and what to grant |
| Notifications | Push to the phone (for example through a Pushover connector) |

### Workflow primitives

| Primitive | Purpose |
|---|---|
| Trigger | Manual, schedule, webhook |
| Connector call | A request through the Sprites gateway with declared path and effect |
| LLM step | A model call through a connector; output marked untrusted |
| Transform | Pure data shaping |
| Branch and bounded loop | Control flow, with caps |
| Wait | Durable delay or event |
| Approve | Human decision, out of the agent's control |
| Notify | Message the user |

### Safety primitives (the part nobody else ships)

| Primitive | Purpose |
|---|---|
| Effect declaration and reversibility class | Drives approvals and undo |
| Idempotency key per effect | No duplicate confirmed effects; unknown outcomes are held, not retried |
| Limits | Max actions per run and per hour, recipient allow-lists |
| Kill switch | Stops every app instantly |
| Audit log | Append-only record of agent, action and approver |
| Checkpoint before run, Rewind | Disk undo plus world reconciliation |
| Untrusted-content marking | Provider and model text is data, not instructions |
| Egress view | The Sprite's network policy and connector allow-list, shown in the UI |

---

## 7. What a generated app looks like (illustrative spec)

```yaml
app: friday-box-digest
version: 1
trigger: { schedule: "fri 17:00" }

steps:
  - id: list
    use: connector.box.list_folder
    with: { folder: "Weekly Reports" }
    effect: read

  - id: summary
    use: llm.summarize
    with: { input: "${list.items}" }
    output: untrusted

  - id: confirm
    use: approve
    with: { show: "${summary.text}", expires: 24h }

  - id: post
    use: connector.slack.post_message
    with: { channel: "#team", text: "${summary.text}" }
    effect: { class: reversible, undo: slack.delete_message }
    idempotency: "friday-box-digest:${run.week}"

limits: { max_posts_per_run: 1, max_runs_per_day: 1 }
ui: { view: detail, card: summary, actions: [run_now, pause] }
```

The platform derives from this: the connector paths needed (Box list, Slack post), the effects (one read, one reversible write), the limits, and the approval requirement. The Permissions card shows them before the app is installed.

---

## 8. What happens to the 12 ideas

| Idea | Becomes |
|---|---|
| 1 Rewind | **Platform capability:** Rewind on every app |
| 2 Safe Hands | **Platform capability:** Approval card and phone approvals |
| 3 Black Box | **Platform capability:** Run timeline and audit |
| 4 Sleepwalker | **Primitive:** Wait, plus wake on request |
| 5 Fortress | **Platform capability:** guardrails and the attack scoreboard |
| 6 Shepherd | **Template:** ops app using the Fly Machines connector, with approvals for destructive actions |
| 7 Toolsmith | **The generator itself** (now safer, because it generates specs) |
| 8 Pie Chief | **Template** (durable reminders plus notifications) |
| 9 Receipts | **Template** (Box in, approval, post) |
| 10 Treaty | **Stretch:** two Sprites, dual approval |
| 11 Red Tape | **Joke template** (committee approval), only if time remains |
| 12 Amnesia | **Stretch:** forget guarantee across memory and checkpoints |

---

## 9. The three apps to generate in the demo

Pick for **variety of primitives** and **variety of connectors**, so more judges see their world:

| App | Primitives exercised | Connectors | Who it speaks to |
|---|---|---|---|
| **Friday Box digest to Slack** | Connector call, LLM step, approval, reversible effect, schedule | Box, Slack, an LLM connector | Box, Fly |
| **GitHub issue triage with approval** | Connector call, LLM step (untrusted input), branch, approval, undo | GitHub | Fly, TypeSafe AI, Ollama if a local model is used for classification |
| **Thanksgiving coordinator** | Wait, loop, notify, durable reminders | Pushover or Slack | Everyone (charm) and the "durable on a sleeping Sprite" story |

Check the connectors available in **your organization's** dashboard first: the page notes that available providers vary by organization during rollout. If Pinecone, Ollama or TypeSafe AI connectors are present, swap one app to use them. If not, do not bend the design.

---

## 10. Rule compliance, updated

The rule: **the project must be built during Hacksgiving week.** Your framing ("the framework only serves as the foundation; the judges see the platform") is reasonable, but it is also exactly what a judge could challenge. Reduce the risk:

1. **New repository today**, commits from November 23, containing everything the judges see: UI, generator, Spec Runner, primitives, templates, demo.
2. **The framework as an external dependency**, not copied code: a separate repository with its own earlier history, imported at a pinned version and credited in the README by name and link. The submission says plainly: "Built this week: the platform, generator, spec runtime, UI and apps. Foundation: [framework], developed earlier, imported as a library."
3. **Make the new work the majority of what runs.** If the Spec Runner and the effects handling you demo are mostly new code written this week, the foundation is clearly a foundation.
4. **Ask the organizers** by email today, in two sentences, whether an earlier open-source library as a dependency is acceptable. Keep the reply.
5. **Plan B:** if the answer is no, or you hear nothing, rebuild the small kernel you need (ledger, approvals, audit, kill switch) fresh in the new repo; the platform design does not change.

---

## 11. Reality check on the framework

| Point | Why it matters |
|---|---|
| **Pin the framework version.** Known lifecycle defects are still in remediation (the Prompt 15.2 work on approvals, cancellation races, duplicate submission and so on) and the P1 contract review is not closed | Do not let remediation consume the hackathon week. Freeze it. Hackathon work targets a pinned version |
| **Only claim what is tested.** Do not say "no duplicate effects" unless a test on the pinned version proves it | A judge who tries a retry or a double click can embarrass an overclaim. Narrow, true claims beat big, shaky ones |
| **Portability spike on day one.** The framework was built around PostgreSQL and a durable workflow engine; a Sprite has a local disk, and credentials should not live in it | Decide: SQLite on the Sprite for working state (the durable workflow library reportedly has a SQLite backend; verify), plus an off-Sprite ledger. If the spike fails, fall back to a lean kernel |
| **Static registration** | New apps cannot be compiled in at run time. This is why the single Spec Runner is the key design move |
| **Approvals must be outside the agent's control** | In a one-Sprite product, the approval page is served from the same Sprite. Make sure the agent process cannot call the approve endpoint (separate service identity or token it cannot read) |

---

## 12. Scope and plan

### Must have (golden path)

Describe bar, generator with validation, Spec Runner (connector call, LLM step, approve, wait, notify, branch), approvals page, permissions card, run timeline with Rewind, off-Sprite ledger, three generated apps, attack scoreboard (small).

### Cut list (do not build)

Visual workflow editor (show the generated spec instead), accounts and multi-user, marketplace, billing, many connectors, custom domain, theming.

### Six-day plan

| Day | Goal | Done when |
|---|---|---|
| Mon 23 | Email the organizers; new repo; Sprite up; connectors available checked (Box, Slack, GitHub, LLM, an object store or custom connector for the ledger); portability spike; spec schema v0 | A Sprite calls a connector with no token inside; the framework imports and runs on the Sprite or the fallback kernel is chosen |
| Tue 24 | Spec Runner v0 with connector call, approve, wait and the effect ledger; `validate` command | A hand-written spec runs end to end with an approval |
| Wed 25 | Generator loop: coding agent writes a spec, validator feeds back, dry run on fixtures; permissions card | "Describe it" produces a valid spec for the Box digest |
| Thu 26 | UI shell: gallery, app view, approvals page, timeline; phone-friendly; Sprite wake test | Approve from a phone while the Sprite sleeps |
| Fri 27 | Rewind and compensations; the second and third apps; attack scoreboard | The golden path runs three times with three apps |
| Sat 28 | Record the video; README with the built-this-week disclosure; post publicly tagging @flydotio; **submit** | Submission sent Saturday night |
| Sun 29 | Buffer only. The deadline is midnight PST, which is about 4:00 PM Monday November 30 in Philippine time | No new work |

---

## 13. Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| "Built during the week" challenged | Medium | Section 10: separate repo, dependency disclosure, organizer email, fallback kernel |
| Platform scope overruns | High | Golden path only; cut list; three apps not twelve |
| Framework portability to a Sprite fails | Medium | Day-one spike; lean-kernel fallback |
| Off-Sprite ledger path unavailable through a connector | Medium | Verify day one; fall back to Box, GitHub or an external endpoint |
| Overclaiming safety guarantees | Medium | Claim only what tests on the pinned version prove |
| Generated specs are low quality | Medium | Strict schema, validator, fixtures, retry inside a checkpoint |
| Connectors differ by organization | Medium | Check the dashboard today; design around what exists |
| Demo flakiness around Sprite sleep and wake | Medium | Rehearse; keep a Service listening; record a clean take |
| The pitch sounds like every app builder | High | Lead with the guarantee line, show undo and the attack, not "describe an app" |

---

## 14. What the connectors page changes

- **Per-Sprite, default-deny access** (turn off "allow all", restrict by name prefix or labels; both must match if both are set; a connector with no policy denies every Sprite): your Permissions card can tell the user exactly which access policy to save for each generated app.
- **Path-level allow and block** (for example the Meta connector allows model discovery and blocks other paths): declare each effect's path in the spec and show it. This is least privilege the user can see.
- **OpenAI- and Anthropic-compatible model access through a connector:** the LLM step can run through a connector with no key on the Sprite, which also keeps the platform model-agnostic.
- **Fly MCP** can be pointed at by any agent, but it can request powerful permissions (deleting apps, running commands). A "Shepherd" template wrapping it with approvals and a checkpoint before destructive steps is a ready-made demo of why your layer matters.
- **Checkpoints** snapshot the filesystem in about a second and are the Rewind foundation; the page itself notes confirmation varies by plugin, which is the gap you fill.

---

## 15. Decisions for you

1. **Rule compliance route:** send the organizer email today, and use the dependency-plus-disclosure approach, or rebuild a small kernel fresh?
2. **Ledger location:** an S3-compatible object store connector, a custom HTTP connector to an external database, or Box as the store?
3. **Which three apps** to generate, given the connectors your organization actually has.
4. **How much of Rewind** to promise: two reversible effects and one irreversible flagged, or more?
5. **Whether to freeze framework remediation** until after November 29.

---

## Sources

- Sprites ecosystem and connectors page (attached `connectors.md`)
- Earlier research for `hacksgiving_plan.md`: https://fly.io/connectors, https://docs.sprites.dev/concepts/checkpoints/, https://fly.io/sprites.md, https://fly.io/changelog/two-new-sprites-connectors/
- Hacksgiving brief (attached earlier): dates, rules, prizes, judges
