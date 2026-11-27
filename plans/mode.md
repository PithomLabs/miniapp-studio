# Local-First Modes of Work

Your proposed mode is correct — but it's one mode among several, and it's the most expensive one. The right architecture keeps it as the baseline while adding cheaper modes as the catalog matures. Below is what those modes are, when each applies, and what the local-first interaction actually looks like.

---

## 1. Your proposed mode is the right baseline

> "User issues prompt → local coding agent uses Miniapp framework → agent produces rails-guarded miniapps following app categories."

This is **Mode 1: Prompt → Go Miniapp**, and it is the correct baseline for two reasons:

1. **It's the most general.** Any automation can be built this way — no template fits it, no spec expresses it, the agent writes Go against the framework.
2. **It fits the framework's binding invariant.** Go is the language, Miniapp is the framework, categories are enforced by the framework's admission step. The agent produces exactly what the framework can validate.

But it is also the **most expensive mode** — the agent must write Go, the code must compile, tests must pass, fixtures must be authored, the manifest must declare the right category. For a Class D durable workflow, this is correct effort. For "read these three fields from an invoice and post to Slack," it's 30 minutes of agent time where 30 seconds should suffice.

The framework enables this mode well. The question is whether to make it the *only* mode, which would be a mistake for an RPA product at scale.

---

## 2. The six modes of work

Each mode is a different interaction pattern between the user, the agent, and the framework. They differ in cost, expressiveness, determinism, and review burden.

| Mode | User does | Agent does | Framework does | Cost | Best for |
|---|---|---|---|---|---|
| **M1 — Prompt → Go** | Describes automation in English | Writes manifest + Go + tests + fixtures | Validates, classifies, admits | Highest (~15–45 min) | Class D, novel workflows, anything not templated |
| **M2 — Prompt → Spec → Generated** | Describes automation in English | Emits `MiniappSpec` (declarative) | Compiles spec into Go miniapp | Medium (~2–5 min) | Common patterns (Class A reads, Class B jobs, standard Class C) |
| **M3 — Template + Parameters** | Picks a template, fills in fields | Nothing (or fills in defaults) | Renders template + parameters into miniapp | Lowest (~30 sec) | 80% of RPA catalog — variations of common patterns |
| **M4 — Bring Your Own (BYO)** | Says "here's my existing Go code" | Wraps it in framework contracts, classifies, adds tests | Validates the wrapped miniapp | Medium (depends on existing code) | Migration from existing automation tools |
| **M5 — Conversational Authoring** | Iterates with agent over multiple turns | Proposes, revises, refines | Validates each revision | Highest (~30–60 min) | Ambiguous requirements, novel business logic |
| **M6 — Catalog → Fork → Modify** | Browses catalog, finds a similar automation | Nothing (or minimal edits) | Forks the miniapp with new parameters | Lowest (~1–2 min) | Mature deployments (20+ automations in catalog) |

Two of these are **variations of M1** (M5 adds iteration; M4 adds a code starting point). Three are cheaper alternatives (M2, M3, M6).

---

## 3. The progression I'd recommend

You do not need all six modes on day one. Build them in this order:

**Months 1–2: M1 only.**
The founder writes prompts, the agent writes Go. This validates that the framework's discipline works — that an agent *can* produce a rails-guarded miniapp, and that the admission step catches what it gets wrong. If M1 doesn't work, none of the others will matter.

**Months 3–5: Add M6 (Catalog → Fork → Modify).**
The moment the catalog has 3 automations, the founder notices that automation #4 is "automation #1 with a different field mapping." Forking is the natural interaction. This requires almost no new code — it's `cp -r automation-A automation-B`, edit the manifest, edit the field mappings, validate.

**Months 5–8: Add M3 (Template + Parameters).**
Once the catalog has 10 automations, the founder notices that 6 of them are "read from system X, transform, post to system Y." That's a template. Building templates requires (a) identifying the pattern, (b) writing a template that takes parameters, (c) a small parameter-fill UI/CLI. This is where most of the cost reduction lives — M3 covers 70–80% of an RPA catalog.

**Months 8–12: Add M2 (Prompt → Spec → Generated).**
This is the hardest mode to build well because it requires a `MiniappSpec` compiler that produces safe, tested Go. It's the GigaCatalyst-style approach. Build it only when M1 (agent writes Go) is the bottleneck because the catalog is at 30+ automations and the founder is spending hours per automation writing Go by hand.

**M4 (BYO) and M5 (Conversational) are variations, not phases.**
- M4 is a convenience wrapper on M1 — the agent starts from existing code instead of a blank file.
- M5 is M1 with more turns — no new mechanism, just a different interaction cadence.

You don't build them. You notice them happening.

---

## 4. The classification question that none of these modes answer

Your proposed mode is silent on a crucial question: **who classifies the automation, and how is the classification reviewed?**

This matters because classification is where the framework's value lives. A miniapp that's classified as Class A but secretly needs durable waits is a production incident. A miniapp classified as Class D when it could be Class B wastes cost and latency on every run.

There are three approaches:

| Approach | How it works | When it works | When it fails |
|---|---|---|---|
| **Agent decides** | The coding agent reads the requirement and picks a class | Fast, no user friction | Agent hallucinates the wrong class; user doesn't understand the consequence |
| **User decides** | The user explicitly picks A/B/C/D in the prompt or manifest | User understands the tradeoffs | Most RPA users don't know what "durable" means; they can't pick correctly |
| **Framework decides** | The framework classifies based on the manifest's declared capabilities (does it use `WaitUntil`? does it retry independently? does it have multiple steps?) | Deterministic, always correct | Requires the framework to be able to detect these properties, which is hard for Go code |

The right answer is a **hybrid**:

1. The agent proposes a classification based on the requirements.
2. The framework validates the classification against the manifest's declared capabilities (e.g., "you declared Class A but the code uses `dbos.Recv`, which is Class D — either downgrade the code or upgrade the class").
3. The user reviews the classification as a one-line summary ("This will run as Class C — bounded batch with per-record idempotency — proceed?").
4. The framework's admission step rejects mismatches.

This is where "rails-guarded" actually lives. The agent doesn't get to freely produce a miniapp; it produces one that the framework validates against the class it declared, and the class must match the code's actual behavior.

**The manifest is the review surface.** The user reviews:
- What the automation does (one sentence)
- What class it's classified as (and why)
- What external systems it touches
- What trust mode it requires
- What approvals it needs
- What limits apply (max runs, max records, max spend)

This is the "prompt → miniapp" review moment, and it's where the value is. If the review is a Go diff, it's a developer tool. If the review is a manifest summary, it's an RPA product.

---

## 5. What "rails-guarded" concretely means

In a local-first setup, "rails-guarded" does not mean "the agent is guided by instructions." It means **the framework's admission step refuses to accept code that violates its contracts.** The rails are enforcement, not suggestion.

Concretely, the admission step rejects:

- A miniapp that declares Class A but uses `dbos.Recv` or `WaitUntil` (class mismatch)
- A miniapp that declares `idempotency_key` but doesn't call `framework.Idempotent` for a write tool
- A miniapp that performs external effects without an `effects.Intent` record
- A miniapp that has write tools without an `entity_key` or an explicit `no_entity` justification
- A miniapp that declares Class D but fails the determinism lint (workflow body has `time.Now()` or `rand.Int()`)
- A miniapp that declares Class D but has no crash-boundary tests
- A miniapp that doesn't compile
- A miniapp whose tests fail
- A miniapp whose declared dependencies aren't in the approved Go module list
- A miniapp that uses a framework other than Miniapp (e.g., imports `net/http` directly in the workflow body)

The agent's job is to produce something the admission step accepts. The framework's job is to reject what it shouldn't accept. The user's job is to review the manifest summary and decide whether the automation is worth running.

**The local coding agent should be given the framework's admission step as a tool.** The agent iterates: write code → run `miniapp validate` → read the errors → fix → rerun → until admission passes. This makes the agent productive without making it trusted — the agent's output is validated by the framework, not by the agent's confidence.

This is what makes Mode 1 (agent writes Go) safe despite being expressive. The agent's job is to satisfy the admission step, and the admission step's job is to enforce the framework's discipline.

---

## 6. The concrete local-first workflow I'd build

Given the above, the local-first interaction I'd actually build for the first 20 automations looks like this:

```
$ studio new invoice-reconciler
? Describe what this automation does (one sentence):
> Every night at 2 AM, read unpaid invoices from Stripe, match to POs in
  the ERP, and post discrepancies to Slack for review.

? Class: the agent suggests Class C (bounded batch with per-record idempotency)
  because it has multiple records, per-record effects, and bounded duration.
  [Accept suggested class] [Choose different class]
> Accept

? Trust mode: Class C writes Slack messages (external effect). Recommend
  confirm mode (requires approval before posting).
  [Accept confirm] [Choose dry-run] [Choose autonomous]
> Accept

? External systems touched:
  - Stripe (read: invoices)
  - ERP (read: POs)
  - Slack (write: messages)
  Each requires a connector. Configure now or later?
> Later

Generating miniapp skeleton... done.
Running admission step... pass.
Generating tests... done.
Running tests... 14 pass, 0 fail.
Generating fixtures... done.

Next steps:
  studio connector add stripe    # required before first run
  studio connector add erp       # required before first run
  studio connector add slack     # required before first run
  studio run invoice-reconciler --dry-run    # test against synthetic data
  studio schedule invoice-reconciler --cron "0 2 * * *"

$ studio run invoice-reconciler --dry-run
Class C run started: run_id=abc123, mode=dry-run
Progress: 47 records processed, 12 matched, 3 discrepancies found
Discrepancies:
  - Invoice INV-001 (Stripe $450) vs PO-234 (ERP $425) — amount mismatch
  - Invoice INV-002 (Stripe $890) — no matching PO
  - Invoice INV-004 (Stripe $120) vs PO-238 (ERP $150) — amount mismatch
Dry-run complete. In real mode, 3 messages would be queued for approval.
```

This workflow has:
- **Mode 1** underneath (agent writes Go)
- **Classification review** (agent proposes, user accepts)
- **Trust mode review** (agent proposes, user accepts)
- **Framework validation** (admission step runs automatically)
- **Dry-run first** (safety before scheduling)
- **Scheduling as a separate deliberate action** (not bundled)

The user sees a manifest-shaped review, not a Go diff. The agent sees a framework-shaped error loop. The framework sees a validate/reject loop that runs in seconds.

**This is the local-first experience for the first 20 automations.** It doesn't require M2 (spec compiler), M3 (templates), or M6 (catalog fork) — though those become valuable as the catalog grows. It requires M1 (agent writes Go) plus the framework's admission step as the review/validation surface.

---

## 7. The modes added later, and why

At 20+ automations, the founder notices the catalog is dominated by three patterns:
- "Read from system X, transform, write to system Y" (Class C)
- "Listen for webhook from X, do Y, notify Z" (Class B)
- "Scheduled report from X" (Class B)

At this point, add **M3 (Template + Parameters)**. Templates are Go code with holes in it: the agent fills the holes with connector names, field mappings, and schedule. The user picks the template, fills the form, the framework compiles.

At 50+ automations, the founder notices that new automations are increasingly variations of existing ones. Add **M6 (Catalog → Fork → Modify)**: `studio fork invoice-reconciler my-variant`, edit the field mapping, done in 2 minutes.

At 100+ automations, the founder notices that 90% of new requests fall into 5 templates, but 10% are genuinely novel. The 10% get M1; the 90% get M3/M6. **M2 (spec compiler) never becomes necessary** — the templates + the agent handle the common case, and M1 handles the novel case. M2 is a solution in search of a problem that M3 solves more cheaply.

This is the progression that an RPA product follows naturally: start general (M1), add templated (M3), add catalog-forking (M6). M2 (the GigaCatalyst model) is elegant but unnecessary for most real RPA catalogs — the pattern library is easier to maintain than a spec compiler.

---

## 8. What you were right about, and what to add

**You were right:** the local coding agent + Miniapp framework + categories is the correct baseline. It's Mode 1, and Mode 1 is where the framework proves itself.

**What to add:**
1. **The admission step as the review surface.** The agent's output is validated by the framework, not by the agent's confidence. The user reviews the manifest, not the code.
2. **The classification review.** The agent proposes the class; the user accepts it; the framework enforces it. Don't let the agent classify silently.
3. **The dry-run-first pattern.** Every automation runs in dry-run first, against synthetic or recorded data, before being scheduled. This is the equivalent of `terraform plan` for RPA — the safety primitive that makes "schedule it and hope" into "schedule it and know."
4. **The six modes as a roadmap, not as day-one requirements.** M1 first, M6 next, M3 after 20 automations, M2 probably never.

The right mental model: **the framework is the rails, the agent is the driver, the user is the passenger.** The user doesn't drive; they don't need to. But they see the route (the manifest), they approve the destination (the class and trust mode), and they can stop the car (the kill switch). The rails make the agent's driving safe. The agent makes the rails useful. Neither is sufficient alone.

Your local-first mode is the right shape. It just needs those four additions to be complete.

## prompt

so for local-first, I just issue prompts to local coding agent but the agent uses Miniapp framework as foundation to build rails-guarded robust task and workflow as miniapps following the app categories where it fits, right? or do you have other modes of work in mind

## answer

# Local-First Modes of Work

Your proposed mode is correct — but it's one mode among several, and it's the most expensive one. The right architecture keeps it as the baseline while adding cheaper modes as the catalog matures. Below is what those modes are, when each applies, and what the local-first interaction actually looks like.

---

## 1. Your proposed mode is the right baseline

> "User issues prompt → local coding agent uses Miniapp framework → agent produces rails-guarded miniapps following app categories."

This is **Mode 1: Prompt → Go Miniapp**, and it is the correct baseline for two reasons:

1. **It's the most general.** Any automation can be built this way — no template fits it, no spec expresses it, the agent writes Go against the framework.
2. **It fits the framework's binding invariant.** Go is the language, Miniapp is the framework, categories are enforced by the framework's admission step. The agent produces exactly what the framework can validate.

But it is also the **most expensive mode** — the agent must write Go, the code must compile, tests must pass, fixtures must be authored, the manifest must declare the right category. For a Class D durable workflow, this is correct effort. For "read these three fields from an invoice and post to Slack," it's 30 minutes of agent time where 30 seconds should suffice.

The framework enables this mode well. The question is whether to make it the *only* mode, which would be a mistake for an RPA product at scale.

---

## 2. The six modes of work

Each mode is a different interaction pattern between the user, the agent, and the framework. They differ in cost, expressiveness, determinism, and review burden.

| Mode | User does | Agent does | Framework does | Cost | Best for |
|---|---|---|---|---|---|
| **M1 — Prompt → Go** | Describes automation in English | Writes manifest + Go + tests + fixtures | Validates, classifies, admits | Highest (~15–45 min) | Class D, novel workflows, anything not templated |
| **M2 — Prompt → Spec → Generated** | Describes automation in English | Emits `MiniappSpec` (declarative) | Compiles spec into Go miniapp | Medium (~2–5 min) | Common patterns (Class A reads, Class B jobs, standard Class C) |
| **M3 — Template + Parameters** | Picks a template, fills in fields | Nothing (or fills in defaults) | Renders template + parameters into miniapp | Lowest (~30 sec) | 80% of RPA catalog — variations of common patterns |
| **M4 — Bring Your Own (BYO)** | Says "here's my existing Go code" | Wraps it in framework contracts, classifies, adds tests | Validates the wrapped miniapp | Medium (depends on existing code) | Migration from existing automation tools |
| **M5 — Conversational Authoring** | Iterates with agent over multiple turns | Proposes, revises, refines | Validates each revision | Highest (~30–60 min) | Ambiguous requirements, novel business logic |
| **M6 — Catalog → Fork → Modify** | Browses catalog, finds a similar automation | Nothing (or minimal edits) | Forks the miniapp with new parameters | Lowest (~1–2 min) | Mature deployments (20+ automations in catalog) |

Two of these are **variations of M1** (M5 adds iteration; M4 adds a code starting point). Three are cheaper alternatives (M2, M3, M6).

---

## 3. The progression I'd recommend

You do not need all six modes on day one. Build them in this order:

**Months 1–2: M1 only.**
The founder writes prompts, the agent writes Go. This validates that the framework's discipline works — that an agent *can* produce a rails-guarded miniapp, and that the admission step catches what it gets wrong. If M1 doesn't work, none of the others will matter.

**Months 3–5: Add M6 (Catalog → Fork → Modify).**
The moment the catalog has 3 automations, the founder notices that automation #4 is "automation #1 with a different field mapping." Forking is the natural interaction. This requires almost no new code — it's `cp -r automation-A automation-B`, edit the manifest, edit the field mappings, validate.

**Months 5–8: Add M3 (Template + Parameters).**
Once the catalog has 10 automations, the founder notices that 6 of them are "read from system X, transform, post to system Y." That's a template. Building templates requires (a) identifying the pattern, (b) writing a template that takes parameters, (c) a small parameter-fill UI/CLI. This is where most of the cost reduction lives — M3 covers 70–80% of an RPA catalog.

**Months 8–12: Add M2 (Prompt → Spec → Generated).**
This is the hardest mode to build well because it requires a `MiniappSpec` compiler that produces safe, tested Go. It's the GigaCatalyst-style approach. Build it only when M1 (agent writes Go) is the bottleneck because the catalog is at 30+ automations and the founder is spending hours per automation writing Go by hand.

**M4 (BYO) and M5 (Conversational) are variations, not phases.**
- M4 is a convenience wrapper on M1 — the agent starts from existing code instead of a blank file.
- M5 is M1 with more turns — no new mechanism, just a different interaction cadence.

You don't build them. You notice them happening.

---

## 4. The classification question that none of these modes answer

Your proposed mode is silent on a crucial question: **who classifies the automation, and how is the classification reviewed?**

This matters because classification is where the framework's value lives. A miniapp that's classified as Class A but secretly needs durable waits is a production incident. A miniapp classified as Class D when it could be Class B wastes cost and latency on every run.

There are three approaches:

| Approach | How it works | When it works | When it fails |
|---|---|---|---|
| **Agent decides** | The coding agent reads the requirement and picks a class | Fast, no user friction | Agent hallucinates the wrong class; user doesn't understand the consequence |
| **User decides** | The user explicitly picks A/B/C/D in the prompt or manifest | User understands the tradeoffs | Most RPA users don't know what "durable" means; they can't pick correctly |
| **Framework decides** | The framework classifies based on the manifest's declared capabilities (does it use `WaitUntil`? does it retry independently? does it have multiple steps?) | Deterministic, always correct | Requires the framework to be able to detect these properties, which is hard for Go code |

The right answer is a **hybrid**:

1. The agent proposes a classification based on the requirements.
2. The framework validates the classification against the manifest's declared capabilities (e.g., "you declared Class A but the code uses `dbos.Recv`, which is Class D — either downgrade the code or upgrade the class").
3. The user reviews the classification as a one-line summary ("This will run as Class C — bounded batch with per-record idempotency — proceed?").
4. The framework's admission step rejects mismatches.

This is where "rails-guarded" actually lives. The agent doesn't get to freely produce a miniapp; it produces one that the framework validates against the class it declared, and the class must match the code's actual behavior.

**The manifest is the review surface.** The user reviews:
- What the automation does (one sentence)
- What class it's classified as (and why)
- What external systems it touches
- What trust mode it requires
- What approvals it needs
- What limits apply (max runs, max records, max spend)

This is the "prompt → miniapp" review moment, and it's where the value is. If the review is a Go diff, it's a developer tool. If the review is a manifest summary, it's an RPA product.

---

## 5. What "rails-guarded" concretely means

In a local-first setup, "rails-guarded" does not mean "the agent is guided by instructions." It means **the framework's admission step refuses to accept code that violates its contracts.** The rails are enforcement, not suggestion.

Concretely, the admission step rejects:

- A miniapp that declares Class A but uses `dbos.Recv` or `WaitUntil` (class mismatch)
- A miniapp that declares `idempotency_key` but doesn't call `framework.Idempotent` for a write tool
- A miniapp that performs external effects without an `effects.Intent` record
- A miniapp that has write tools without an `entity_key` or an explicit `no_entity` justification
- A miniapp that declares Class D but fails the determinism lint (workflow body has `time.Now()` or `rand.Int()`)
- A miniapp that declares Class D but has no crash-boundary tests
- A miniapp that doesn't compile
- A miniapp whose tests fail
- A miniapp whose declared dependencies aren't in the approved Go module list
- A miniapp that uses a framework other than Miniapp (e.g., imports `net/http` directly in the workflow body)

The agent's job is to produce something the admission step accepts. The framework's job is to reject what it shouldn't accept. The user's job is to review the manifest summary and decide whether the automation is worth running.

**The local coding agent should be given the framework's admission step as a tool.** The agent iterates: write code → run `miniapp validate` → read the errors → fix → rerun → until admission passes. This makes the agent productive without making it trusted — the agent's output is validated by the framework, not by the agent's confidence.

This is what makes Mode 1 (agent writes Go) safe despite being expressive. The agent's job is to satisfy the admission step, and the admission step's job is to enforce the framework's discipline.

---

## 6. The concrete local-first workflow I'd build

Given the above, the local-first interaction I'd actually build for the first 20 automations looks like this:

```
$ studio new invoice-reconciler
? Describe what this automation does (one sentence):
> Every night at 2 AM, read unpaid invoices from Stripe, match to POs in
  the ERP, and post discrepancies to Slack for review.

? Class: the agent suggests Class C (bounded batch with per-record idempotency)
  because it has multiple records, per-record effects, and bounded duration.
  [Accept suggested class] [Choose different class]
> Accept

? Trust mode: Class C writes Slack messages (external effect). Recommend
  confirm mode (requires approval before posting).
  [Accept confirm] [Choose dry-run] [Choose autonomous]
> Accept

? External systems touched:
  - Stripe (read: invoices)
  - ERP (read: POs)
  - Slack (write: messages)
  Each requires a connector. Configure now or later?
> Later

Generating miniapp skeleton... done.
Running admission step... pass.
Generating tests... done.
Running tests... 14 pass, 0 fail.
Generating fixtures... done.

Next steps:
  studio connector add stripe    # required before first run
  studio connector add erp       # required before first run
  studio connector add slack     # required before first run
  studio run invoice-reconciler --dry-run    # test against synthetic data
  studio schedule invoice-reconciler --cron "0 2 * * *"

$ studio run invoice-reconciler --dry-run
Class C run started: run_id=abc123, mode=dry-run
Progress: 47 records processed, 12 matched, 3 discrepancies found
Discrepancies:
  - Invoice INV-001 (Stripe $450) vs PO-234 (ERP $425) — amount mismatch
  - Invoice INV-002 (Stripe $890) — no matching PO
  - Invoice INV-004 (Stripe $120) vs PO-238 (ERP $150) — amount mismatch
Dry-run complete. In real mode, 3 messages would be queued for approval.
```

This workflow has:
- **Mode 1** underneath (agent writes Go)
- **Classification review** (agent proposes, user accepts)
- **Trust mode review** (agent proposes, user accepts)
- **Framework validation** (admission step runs automatically)
- **Dry-run first** (safety before scheduling)
- **Scheduling as a separate deliberate action** (not bundled)

The user sees a manifest-shaped review, not a Go diff. The agent sees a framework-shaped error loop. The framework sees a validate/reject loop that runs in seconds.

**This is the local-first experience for the first 20 automations.** It doesn't require M2 (spec compiler), M3 (templates), or M6 (catalog fork) — though those become valuable as the catalog grows. It requires M1 (agent writes Go) plus the framework's admission step as the review/validation surface.

---

## 7. The modes added later, and why

At 20+ automations, the founder notices the catalog is dominated by three patterns:
- "Read from system X, transform, write to system Y" (Class C)
- "Listen for webhook from X, do Y, notify Z" (Class B)
- "Scheduled report from X" (Class B)

At this point, add **M3 (Template + Parameters)**. Templates are Go code with holes in it: the agent fills the holes with connector names, field mappings, and schedule. The user picks the template, fills the form, the framework compiles.

At 50+ automations, the founder notices that new automations are increasingly variations of existing ones. Add **M6 (Catalog → Fork → Modify)**: `studio fork invoice-reconciler my-variant`, edit the field mapping, done in 2 minutes.

At 100+ automations, the founder notices that 90% of new requests fall into 5 templates, but 10% are genuinely novel. The 10% get M1; the 90% get M3/M6. **M2 (spec compiler) never becomes necessary** — the templates + the agent handle the common case, and M1 handles the novel case. M2 is a solution in search of a problem that M3 solves more cheaply.

This is the progression that an RPA product follows naturally: start general (M1), add templated (M3), add catalog-forking (M6). M2 (the GigaCatalyst model) is elegant but unnecessary for most real RPA catalogs — the pattern library is easier to maintain than a spec compiler.

---

## 8. What you were right about, and what to add

**You were right:** the local coding agent + Miniapp framework + categories is the correct baseline. It's Mode 1, and Mode 1 is where the framework proves itself.

**What to add:**
1. **The admission step as the review surface.** The agent's output is validated by the framework, not by the agent's confidence. The user reviews the manifest, not the code.
2. **The classification review.** The agent proposes the class; the user accepts it; the framework enforces it. Don't let the agent classify silently.
3. **The dry-run-first pattern.** Every automation runs in dry-run first, against synthetic or recorded data, before being scheduled. This is the equivalent of `terraform plan` for RPA — the safety primitive that makes "schedule it and hope" into "schedule it and know."
4. **The six modes as a roadmap, not as day-one requirements.** M1 first, M6 next, M3 after 20 automations, M2 probably never.

The right mental model: **the framework is the rails, the agent is the driver, the user is the passenger.** The user doesn't drive; they don't need to. But they see the route (the manifest), they approve the destination (the class and trust mode), and they can stop the car (the kill switch). The rails make the agent's driving safe. The agent makes the rails useful. Neither is sufficient alone.

Your local-first mode is the right shape. It just needs those four additions to be complete.