## claude

# App Categories for Miniapps — Why We Classify, How to Organize Business Logic, and What to Do and Not Do

- **Audience:** developers and coding agents building miniapps (agent-invokable business-process automations) in Go on DBOS, River, Fly.io and Neon Postgres
- **Purpose:** give one shared vocabulary and a set of enforceable rules so that every piece of business logic is organized according to *how it must behave when things go wrong*, not according to which feature it belongs to
- **Companion documents:** ADR-001 (`DBOS.md`: selective adoption of DBOS), `rea.md` (pattern source), the plan4.1 adversarial review
- **Status of code samples:** illustrative shapes. Verify every SDK call against the pinned versions of DBOS Go and River before copying

---

## 1. The short version

1. Every miniapp tool is classified into one of four **execution categories** (A, B, C, D) by what it requires to survive failure, not by how complicated its business rules are.
2. The category decides the **substrate** (plain handler, River, DBOS), the **contract with the agent** (inline result vs run handle), the **tests** you must write, and the **runbook** operators will use.
3. Business rules live in a **pure domain core** that does not know its category. Only a thin **orchestration shell** is category-specific. That is what makes a tool promotable (A → B → D) without a rewrite.
4. Default to the **lowest category that satisfies the requirement**. Escalation must be justified in writing (ADR-001); de-escalation is welcome.
5. Categories are for **invocation boundaries** (a tool, a trigger, a scheduled task). They are not a grading system for code and not a rule for every function.

---

## 2. Why classify at all

### 2.1 The problem: logic organized by feature hides the failure contract

Two tools can look equally simple in a feature list:

- "Update a contact in the CRM"
- "Follow up with this lead for two weeks"

One is a single request that either happened or did not. The other must survive restarts, deployments and multi-day waits, and must never send a follow-up twice. If both are written as "an async function that calls some APIs", the second one will lose follow-ups after the first deployment, and the first one will be dragged into queue and workflow machinery it never needed.

Classification forces the question that determines correctness: **what happens if the process dies at every important transition?**

### 2.2 What the category decides

| Decision | Determined by the category |
|---|---|
| Where the code runs | Plain Go handler, River worker, or DBOS workflow |
| What "success" means after a crash | Retry from scratch, resume a batch, or resume from a checkpoint |
| What the agent receives | Inline result (A) or a run handle (B/C/D) |
| How duplicates are prevented | Row lock and idempotency key (A), unique job key (B/C), workflow ID and step idempotency (D) |
| What you must test | Unit only (A) up to crash-boundary and replay tests (D) |
| What operators monitor | Request errors (A) up to stuck workflows and quarantined effects (D) |
| What upgrade rules apply | None (A) up to workflow-version compatibility (D) |

### 2.3 Failure modes classification prevents

| Failure | Cause | Category discipline that prevents it |
|---|---|---|
| Lost follow-ups after deploy | Multi-day process built as an in-memory timer or a plain job | Class D requires durable waits and a version strategy |
| Duplicate emails or payments | Retry of a step whose outcome was unknown | Effect ledger, idempotency keys, UNKNOWN quarantine for every consequential call |
| "DBOS everywhere" | Using a workflow engine to wrap a single database transaction | Classes A and B default to *no workflow engine* |
| Re-implementing a workflow engine | Hand-rolled timers, reapers and progress tables on a job queue | ADR-001 Gate 4: compare custom machinery against DBOS before building it |
| Agent latency and token waste | Every read returns a run handle that must be polled | Class A returns inline |
| Two agents corrupt one entity | Concurrent runs on the same invoice or lead | Keyed serialization pattern required for writes |
| Business rules entangled with I/O | Rules written inside handlers or workflows | Pure domain core; category-specific code is only a shell |

### 2.4 Secondary benefits

- **Shared vocabulary.** "This is a Class C with a keyed-serial pattern" is a complete design sentence for a reviewer or a coding agent.
- **Review speed.** Reviewers know which checklist items apply.
- **Cost control.** Durable workflows cost database writes, connection budget and operational procedures. Classification keeps that cost where it pays for itself.
- **Prompting coding agents.** An agent given the category and the rules below produces code with the right structure the first time.

---

## 3. The categories

Classification is **per tool or trigger**, not per miniapp. A single miniapp may contain a Class D `start_followup` tool and a Class A `get_followup_status` tool.

### 3.1 Class A — Ordinary operation

| Aspect | Rule |
|---|---|
| Shape | Completes within one request; no independent lifecycle |
| Failure contract | Caller retries the whole request; idempotency key protects writes |
| Substrate | Go handler + Postgres transaction. **No queue, no workflow engine** |
| Agent contract | Inline result within the sync budget (suggested 5 s) |
| Typical examples | Read a record, validate input, create or update one record, check availability |
| External effects | A single provider call is allowed but must go through the effect ledger (a timeout after send is an *unknown* outcome, not a failure) |
| Tests | Unit, contract, authorization probes, idempotency replay |

**Use when:** the work finishes inside the request and nothing needs to resume.
**Not when:** the work may exceed the sync budget, needs retries beyond the request, or waits for a person.

### 3.2 Class B — Simple background job

| Aspect | Rule |
|---|---|
| Shape | One independently retryable operation with a clear beginning and end |
| Failure contract | Safe to retry using an idempotency key; no progress worth preserving |
| Substrate | River (transactional enqueue in the same transaction as the motivating write) |
| Agent contract | Run handle; poll or long-poll for the result |
| Typical examples | Send one notification, process one webhook, generate one report, sync one record |
| Tests | Unit, enqueue atomicity, retry and terminal-failure behavior, duplicate submit collapse |

**Use when:** one operation, retryable from scratch, no human or long wait.
**Not when:** several dependent stages must not repeat after partial progress.

### 3.3 Class C — Bounded multi-step operation

| Aspect | Rule |
|---|---|
| Shape | Several operations or many records, finishing within a bounded period |
| Failure contract | Resume without replaying completed records; per-record idempotency |
| Substrate | River by default with an explicit progress record; **pilot** a DBOS parent/child design when custom machinery starts to resemble a workflow engine (concurrency limits, durable progress, per-step retry) |
| Agent contract | Run handle with progress (`processed`, `failed`, `remaining`) |
| Typical examples | Import contacts, process a batch of appointments, send to a bounded recipient list |
| Tests | Kill-and-resume, per-record idempotency, partial failure policy, rate-limit behavior |

**Use when:** multiple steps or records, bounded duration, recovery means "continue the batch".
**Not when:** the process waits for people or days, or needs durable checkpoints between stages.
**Watch for:** multiple steps alone does not justify DBOS; but hand-building concurrency control, progress tracking, reapers and timers on a job queue is the cost ADR-001 Gate 4 asks you to compare.

### 3.4 Class D — Durable business workflow

| Aspect | Rule |
|---|---|
| Shape | Progress must survive process replacement; spans hours, days or weeks; durable waits; checkpoints; human or external events |
| Failure contract | Resume from the last completed step; no repeated *confirmed* effects |
| Substrate | DBOS Transact Go, subject to ADR-001 Gates 1–4 and a written decision record |
| Agent contract | Run handle with status, signal and cancel |
| Typical examples | Lead follow-up sequence, invoice-to-ledger with approval, employee provisioning with saga, appointment reminder journey |
| Tests | Crash at every transition, replay determinism, worker-kill recovery, version-compatibility, approval-while-stopped |

**Use when:** durable progress and waiting are part of the business promise.
**Not when:** a restart from the beginning is harmless, or the "wait" is just a scheduling convenience.

### 3.5 Cross-cutting patterns (not categories)

These combine with a category rather than replace it.

| Pattern | Applies to | Mechanism |
|---|---|---|
| **Approval-gated invocation** | Any write tool in `confirm` mode | One approval table with state machine (`PENDING → APPROVED/REJECTED/EXPIRED`, `expires_at`, material-change hash, approver identity). Single-step tools resume via a one-shot River job; multi-stage workflows wait on a DBOS message/event with timeout. The table is always the source of truth |
| **Keyed serial execution** | Any write that targets an entity | A: row lock; B/C: unique key + row lock; D: workflow on a queue partitioned by entity key with concurrency 1 |
| **Saga / compensation** | D (sometimes C) | Register the compensation *before* the action; run in LIFO order; compensators idempotent; never auto-compensate an UNKNOWN outcome without a safety basis |
| **Reconciliation** | Any category with external writes | Reconcile-by-read before retrying an UNKNOWN effect; sweeper job on River |
| **Policy guardrails** | All | Strict/warn/disabled policy modes (strict in CI and production), idempotency-key validation, server-side authorization |

---

## 4. How to classify

### 4.1 Ask these questions in order

1. **Does it complete inside one request and need no resume?** → **A**
2. **Is it one independently retryable operation?** → **B**
3. **Is it several operations or records that finish within a bounded period, where recovery means continuing?** → **C**
4. **Does it need durable waits, human decisions, external events, multi-day lifetime, or checkpointed resume?** → **D candidate** — complete the ADR-001 decision record before building

If the answer to question 1 is yes but the tool performs an external write, keep it Class A **and** record the effect in the ledger.

### 4.2 Complexity score (ADR-001 §4 decision aid)

Score each dimension 0 (negligible), 1 (limited), 2 (important):

`lifetime · stages · waiting · recovery · external effects · coordination · human involvement`

| Total | Guidance |
|---|---|
| 0–4 | Ordinary code or a simple job |
| 5–7 | Evaluate queue and database capabilities first |
| 8–10 | Strong DBOS candidate; do an architectural comparison |
| 11–14 | Very strong DBOS candidate; justify any custom workflow engine |

The score is an aid, not a selector. A critical multi-day wait can qualify with a low total; a high total can still have a reason to avoid DBOS.

### 4.3 Worked examples

| Tool | First impression | Actual category | Why |
|---|---|---|---|
| `get_contact` | Read | **A** | One request, no effects |
| `update_contact` (autonomous) | Write | **A** + ledger + keyed lock | One request; external write needs effect protocol |
| `update_contact` (`confirm` mode) | Write | **A/B** + approval table | Bounded human wait; approval resumes a one-shot job. Not reclassified to D |
| `send_notification` | Async | **B** | One retryable operation |
| `import_contacts` (2,000 rows) | Batch | **C** | Per-record idempotency, resume, rate limits |
| `follow_up_lead` (7 days, replies cancel it) | Sequence | **D** | Durable waits, events, cancellation, consequences if duplicated |
| `post_invoice` (extract → validate → approval → ledger → reconcile) | Pipeline | **D** | Multi-stage, human wait, saga, expensive recovery |
| "Remind me in 10 minutes" | Delay | **B** | Scheduling convenience, not a business process |
| `sync_accounts` (pull, merge, push, conflicts) | Sync | **C or D** | Score ~8; run the comparison and record the decision |

### 4.4 Reclassification triggers

| Move | Trigger |
|---|---|
| A → B | Exceeds sync budget, needs retries that outlive the request |
| B → C | Becomes several dependent operations with meaningful partial progress |
| C → D | Adds durable waits, human involvement, multi-day lifetime, or hand-built timers and reapers |
| D → C/B | The waits and approvals are removed; record **RETIRE DBOS** for that tool |

A reclassification is a design change: update the manifest, the decision record, the tests and the runbook in the same pull request.

---

## 5. When to use classification — and when not to

### 5.1 Use it

- At every **invocation boundary**: each tool, each webhook trigger, each scheduled task.
- At **design time**, before choosing a substrate.
- In **review**, to select the applicable checklist.
- When **prompting a coding agent**, to state the required structure.

### 5.2 Do not use it

| Situation | Why |
|---|---|
| Internal helper functions, utilities, pure libraries | Categories describe invocation lifecycles, not code units |
| One-off scripts and data migrations | Run once, supervised; use normal operational practice |
| Throwaway spikes | Classify when the spike becomes a product |
| UI work, browser automation, screen scraping | Outside the miniapp model; separate capability with its own security design |
| High-throughput event streams and real-time pipelines | Not business-process automation; use purpose-built stream tooling |
| As a quality grade ("D is more advanced") | Higher classes cost more; the right class is the lowest sufficient one |

### 5.3 When the category is unclear

1. Start with the **lower** candidate class.
2. Write the ADR-001 decision template (REJECT, DEFER, PILOT, ADOPT) for the higher one.
3. If still unclear, **PILOT** with measurable acceptance criteria rather than guessing.

---

## 6. Organizing business logic by category

### 6.1 The central rule: a pure domain core plus a thin shell

Business rules belong in a **domain core** that:

- does no I/O, no logging of secrets, no clock reads, no randomness;
- takes explicit inputs (including `now` and any prior state) and returns explicit decisions;
- imports nothing from the framework, River or DBOS.

Everything category-specific lives in a thin **orchestration shell** that loads state, calls the core, performs effects through ports, and records results.

```text
miniapps/<name>/
  domain/        pure rules and decision types              (no I/O, no clock, no framework imports)
  ports/         interfaces for effects: capabilities, clock, id generation
  app/
    handler.go   Class A shell      (or)
    job.go       Class B/C shell    (or)
    workflow.go  Class D shell
  manifest/      miniapp.json, adr001.md (C/D required)
  fixtures/      independent expected results
  tests/         unit (domain), shell tests, crash-boundary tests
```

Dependency direction: `domain ← app ← adapters`. Never the reverse.

**Why this matters:** when a tool is promoted from B to D, only `app/` changes. The rules, the fixtures and most tests carry over.

### 6.2 What belongs where

| Concern | Domain core | Shell | Framework/platform |
|---|---|---|---|
| Eligibility, pricing, scheduling rules | ✔ | | |
| Deciding the next action from state | ✔ | | |
| Reading current time | | ✔ (pass into core) | |
| Provider calls | | ✔ via port | Capability proxy, token broker |
| Idempotency identity | ✔ (derivation rule) | ✔ (applies it) | Ledger |
| Authorization | | | ✔ (before dispatch; never in domain) |
| Retries, backoff | | Declared | River / DBOS |
| Audit, redaction | | | ✔ |

### 6.3 Class A skeleton

```go
// app/handler.go — one request, one transaction, inline result.
func (h Handler) UpdateContact(ctx context.Context, inv framework.Invocation) (framework.Result, error) {
    in, err := framework.Decode[UpdateContactIn](inv)           // schema validation, terminal on failure
    if err != nil { return framework.Result{}, err }

    return h.tx.Run(ctx, inv.Tenant(), func(tx framework.Tx) (framework.Result, error) {
        cur, err := h.repo.LockContact(ctx, tx, in.ContactID)   // keyed serial: row lock
        if err != nil { return framework.Result{}, err }

        decision := domain.PlanContactUpdate(cur, in, h.clock.Now()) // pure rules
        if !decision.Allowed { return framework.Rejected(decision.Reason), nil }

        // external write: ledger records intent first; unknown outcomes are quarantined
        out, err := h.cap.Call(ctx, "crm.contact.update", decision.Request,
            effects.Intent{Key: inv.IdempotencyKey("contact.update")})
        return framework.OK(out), err
    })
}
```

### 6.4 Class B/C skeleton (River)

```go
// Args are small: identifiers, never records or credentials.
type SendNotificationArgs struct {
    Tenant string `json:"tenant"`
    NoticeID string `json:"notice_id"`
}
func (SendNotificationArgs) Kind() string { return "notify.send" }

type SendNotificationWorker struct {
    river.WorkerDefaults[SendNotificationArgs]
    deps Deps
}

func (w *SendNotificationWorker) Work(ctx context.Context, job *river.Job[SendNotificationArgs]) error {
    n, err := w.deps.Repo.GetNotice(ctx, job.Args.Tenant, job.Args.NoticeID)
    if err != nil { return err }
    plan := domain.PlanDelivery(n, w.deps.Clock.Now())       // pure
    if plan.AlreadyDelivered { return nil }                   // idempotent
    return w.deps.Send(ctx, plan)                             // through ledger + capability proxy
}

// Enqueue in the same transaction as the motivating write (River InsertTx),
// with a unique key derived from the business identity.
```

Class C additions: a **progress record** (`processed`, `failed`, `cursor`), per-record idempotency keys (`tenant:batch:{id}:row:{n}`), bounded concurrency, and a defined partial-failure policy.

### 6.5 Class D skeleton (DBOS)

```go
// app/workflow.go — deterministic control flow; every nondeterministic thing is a step.
func FollowUp(ctx dbos.Context, in FollowUpIn) (FollowUpOut, error) {
    state := domain.NewFollowUpState(in)

    for !state.Done() {
        // 'now' enters the workflow only through a step, so replay sees the recorded value.
        now, err := dbos.RunAsStep(ctx, func(c context.Context) (time.Time, error) { return time.Now(), nil },
            dbos.WithStepName("clock"))
        if err != nil { return FollowUpOut{}, err }

        action := domain.NextAction(state, now)               // pure decision

        switch action.Kind {
        case domain.Wait:
            dbos.Sleep(ctx, action.Duration)                  // durable wait (check return values per SDK)
        case domain.Send:
            _, err := dbos.RunAsStep(ctx, func(c context.Context) (Receipt, error) {
                return deps.Send(c, action.Message)           // ledger + proxy; idempotency key from workflow ID + step
            }, dbos.WithStepName("send"), dbos.WithStepMaxRetries(3))
            if err != nil { return FollowUpOut{}, err }
        }
        state = domain.Apply(state, action)
    }
    return FollowUpOut{Status: state.Status()}, nil
}
// Start with a stable workflow ID: tenant:lead:followup:v1  (dbos.WithWorkflowID)
// No goroutines, no select, no map iteration, no time.Now, no rand inside the workflow body.
```

---

## 7. Dos and don'ts

### 7.1 Universal

| Do | Don't |
|---|---|
| Classify each tool and record the category in the manifest | Classify once for the whole miniapp |
| Choose the lowest category that satisfies the requirement | Choose a category for technology enthusiasm or consistency |
| Keep business rules in the pure domain core | Put rules inside handlers, River workers or workflow bodies |
| Give every consequential operation a stable idempotency identity (`tenant:entity:kind:op`) | Use random or per-attempt identifiers |
| Route every external write through the effect ledger and capability proxy | Call providers directly or hold credentials in miniapp code |
| Treat an unknown outcome as UNKNOWN; reconcile by read before any retry | Retry blindly after a timeout |
| Put identifiers and compact context in job args and workflow inputs | Put records, secrets, tokens or database URLs in payloads |
| Declare `entity_key` on write tools (or justify its absence) | Allow concurrent writes to one entity without serialization |
| Declare limits (rate, volume, per-run) and honor the kill switch | Ship a write tool without blast-radius limits |
| Mark free-text provider content as untrusted in results | Pass provider text through to the agent as if it were instructions |
| Write the crash-boundary tests for your category | Rely on "it worked locally" |
| Update decision record, tests and runbook when you reclassify | Change the substrate quietly |

### 7.2 Class A

| Do | Don't |
|---|---|
| Return inline within the sync budget | Return a run handle for a simple read |
| Use one transaction for domain write, audit row and effect intent | Split related writes across transactions |
| Lock the entity row for writes | Read-modify-write without a lock |
| Use transaction-local tenant context (`SET LOCAL` / `set_config(..., true)`) | Use session-level `SET` on pooled connections |
| Enforce authorization before dispatch | Check permissions inside domain logic |
| — | **Wrap a Class A operation in a workflow to "standardize execution"** |
| — | Poll or sleep inside a request |

### 7.3 Class B

| Do | Don't |
|---|---|
| Enqueue in the same transaction as the motivating write | Enqueue after commit and hope it succeeds |
| Use unique keys derived from the business identity | Allow duplicate jobs for the same operation |
| Make the worker safe to run twice | Depend on exactly-once job delivery |
| Define retry limits and terminal-failure handling | Retry forever or swallow errors |
| Keep job args small | Embed full payloads |
| — | Add DBOS to run a single task asynchronously |
| — | Build multi-day state machines out of delayed jobs |

### 7.4 Class C

| Do | Don't |
|---|---|
| Keep an explicit progress record and per-record idempotency keys | Resume by replaying completed records |
| Bound concurrency and respect provider rate limits | Fan out unbounded |
| Define a partial-failure policy up front (skip, quarantine, abort) | Discover the policy in production |
| Report progress to the agent (`processed`, `failed`, `remaining`) | Return only "done" or "failed" |
| Pilot and record a decision when custom machinery grows (timers, reapers, concurrency control) | Quietly accumulate a home-grown workflow engine |
| — | Assume multiple steps justify DBOS |
| — | Assume your job queue's free tier has global concurrency limits (verify) |

### 7.5 Class D

| Do | Don't |
|---|---|
| Complete the ADR-001 decision record before implementation | Move from "considering" to "adopted" implicitly |
| Use a stable, intentional workflow ID scoped by tenant and business operation | Let a client retry create a second workflow |
| Put every nondeterministic call (time, randomness, I/O) in a step | Read `time.Now`, use `rand`, or touch the database in the workflow body |
| Use the SDK's durable helpers for concurrency; sort map keys before iterating | Start goroutines, use `select`, or range over a map to call steps |
| Use datasource transactions when the app write and checkpoint must commit atomically | Assume an ordinary Postgres transaction includes the DBOS checkpoint |
| Remember steps run **at least once**; make them idempotent | Assume a step is exactly-once |
| Keep workflow code on the SDK's rules: no transactions, enqueues or message calls inside step bodies | Call durable operations from inside steps |
| Register compensation **before** the action; keep compensators idempotent | Auto-compensate an UNKNOWN outcome without a safety basis |
| Define a version strategy for pending workflows before every deploy | Change workflow control flow without a version bump, patch or drain |
| Define retry limits, terminal handling, cancellation, alerting and recovery ownership | Leave workflows able to become silently stuck |
| Run the determinism linter and replay tests in CI | Rely on code review to catch non-determinism |
| — | Pass large objects through workflow inputs/outputs |
| — | Make DBOS a mandatory dependency for every module |

### 7.6 Approval-gated invocation

| Do | Don't |
|---|---|
| Use the approval table as the single source of truth | Hold approval state only in a workflow or in the harness UI |
| Require out-of-band approver authentication and separation of duties | Let the invoking agent approve its own request |
| Invalidate approval when material parameters change | Reuse an approval for different inputs |
| Set an expiry and handle `EXPIRED` explicitly | Wait forever |
| Keep single-step approvals on River; use a DBOS wait only inside a larger workflow | Promote a whole tool to Class D just because it needs one approval |

---

## 8. Anti-patterns and fixes

| Anti-pattern | Symptom | Fix |
|---|---|---|
| **DBOS everywhere** | Simple reads and single transactions run as workflows; checkpoint writes dominate cost | Move to Class A or B; record **RETIRE DBOS** |
| **Shadow workflow engine on River** | Progress tables, timers, reapers, and ad-hoc state machines for a multi-day process | Run the ADR-001 comparison; pilot DBOS for that tool |
| **Giant step** | One step does ten provider calls; a crash replays all of them | Split into steps at natural durable boundaries with their own idempotency keys |
| **Rules in the shell** | Business logic duplicated across handler and worker; cannot be promoted | Extract to `domain/`; shell calls it |
| **Clock in the workflow** | Replay takes a different branch; workflow fails or behaves differently | Fetch time in a step; pass into the pure core |
| **Secrets in payloads** | Credentials or tokens visible in job tables, checkpoints, logs | IDs only; resolve credentials inside the capability proxy |
| **Blind retry of unknown outcome** | Duplicate email, payment, or record | Ledger + UNKNOWN state + reconcile-by-read |
| **Polling loop in a request** | Request timeouts; harness gives up | Class B/C/D with a run handle and long-poll status |
| **Unkeyed writes** | Lost updates when an agent retries or two agents collide | Declare `entity_key`; apply the keyed serial pattern |
| **Reclassify silently** | Docs, tests and alerts describe the old category | Treat reclassification as a design change in one PR |
| **Approval inside the agent** | Agent "confirms" itself | Out-of-band approver and server-side enforcement |
| **Wrapping a transaction in a workflow** | Same atomicity, extra complexity | Use the plain transaction |

---

## 9. Required artifacts per category

| Artifact | A | B | C | D |
|---|---|---|---|---|
| Manifest entry with `class` and tool kind | ✔ | ✔ | ✔ | ✔ |
| Idempotency identity declared | ✔ (writes) | ✔ | ✔ | ✔ |
| `entity_key` declared (writes) | ✔ | ✔ | ✔ | ✔ |
| Pure `domain/` package with unit tests | ✔ | ✔ | ✔ | ✔ |
| Authorization probes | ✔ | ✔ | ✔ | ✔ |
| Limits and kill-switch hooks (writes) | ✔ | ✔ | ✔ | ✔ |
| Enqueue-atomicity test | | ✔ | ✔ | |
| Progress and resume test | | | ✔ | |
| Partial-failure policy | | | ✔ | |
| `adr001.md` decision record | | | ✔ | ✔ |
| Crash-boundary tests at every transition | | | ✔ | ✔ |
| Determinism lint + replay test | | | | ✔ |
| Version-compatibility strategy | | | | ✔ |
| Recovery policy, alerts, runbook | basic | basic | ✔ | ✔ |

---

## 10. Review checklist

Copy into pull requests.

```markdown
## Category review — <miniapp>/<tool>

- [ ] Category declared in manifest: A | B | C | D (per tool)
- [ ] Lowest sufficient category? Justification (one line): …
- [ ] Domain rules live in `domain/`; no I/O, clock, rand or framework imports
- [ ] Idempotency identity and `entity_key` declared; keyed serialization applied
- [ ] All external writes go through ledger + capability proxy; UNKNOWN path handled
- [ ] No secrets or full records in args, inputs, outputs, logs
- [ ] Limits, rate caps, kill-switch hooks configured (writes)
- [ ] Untrusted provider text marked in results; result size capped
- [ ] Tests for this category present (see §9) and passing on real Postgres
- [ ] C/D: `adr001.md` complete: score, gates, Alternative A vs B, decision (REJECT/DEFER/PILOT/ADOPT)
- [ ] D: determinism lint clean; replay test; version strategy; recovery policy and alerts
- [ ] Approval-gated? approval table used; out-of-band approver; expiry; material-change invalidation
- [ ] If reclassified: manifest, decision record, tests, runbook updated together
```

---

## 11. Quick reference card

| | A | B | C | D |
|---|---|---|---|---|
| One-line test | Finishes in the request | One retryable job | Bounded batch, resume | Durable process, waits, checkpoints |
| Substrate | Handler + tx | River | River (pilot DBOS) | DBOS |
| Agent gets | Inline result | Run handle | Run handle + progress | Run handle + status/signal/cancel |
| Crash means | Caller retries | Job retries | Batch resumes | Workflow resumes from last step |
| Duplicate guard | Idempotency key + row lock | Unique job key | Per-record keys | Workflow ID + step idempotency |
| Never | Wrap in a workflow | Use DBOS for one task | Assume multi-step means DBOS | Non-determinism in the body |

**Governing rule (ADR-001 §10):** use the simplest execution mechanism that reliably satisfies the business requirement. Ordinary operations belong in ordinary code, atomic data changes in Postgres transactions, simple background work in the job queue, and durable multi-stage business processes in DBOS — exactly where *not* having it would make the system unnecessarily difficult to build, prove, operate and trust.


## gemini

# Guidelines: Classifying Business Logic by Miniapp Category

## 1. Why Classifying Miniapp Categories Is Necessary

Agent-invokable RPA miniapps execute real business processes against SaaS and custom REST APIs on behalf of harnesses like Claude Code, Codex, Hermes, and OpenClaw. Unlike human browser sessions, AI agents retry calls, re-plan after partial failures, and operate across sessions that may terminate while a business process is still waiting.

Classifying every miniapp into an explicit execution category prevents two failure modes:

* **Under-engineering:** Treating a multi-day workflow or a consequential multi-system mutation as a plain synchronous handler or ad-hoc script leads to lost state, duplicate side effects, and orphaned work when a process restarts.


* **Over-engineering:** Routing every simple read or lightweight background task through a durable workflow engine adds replay-determinism constraints, checkpoint write amplification, connection overhead, and operational complexity that do not match the workload.



The core execution taxonomy separates miniapps into **Class A**, **Class B**, **Class C**, and **Class D**, while treating **per-entity serialization**, **saga compensation**, and **human-in-the-loop approvals** as composable patterns rather than standalone categories. Each category maps to a single authoritative substrate on **Go + Neon Postgres + River + DBOS Transact Go** deployed on **Fly.io**.

| Category | Execution Form | Authoritative Substrate | Primary Purpose |
| --- | --- | --- | --- |
| **Class A** | Synchronous single-request operation | Plain Go handler + Neon transaction | Bounded reads or immediate transactional operations with no multi-step durable lifecycle.

 |
| **Class B** | One-shot or delayed background job | **River** (transactional enqueue) | Fire-and-forget tasks, scheduled runs, webhooks, and single-unit retries.

 |
| **Class C** | Bounded multi-step batch operation | **River** by default (or **DBOS** when justified by checkpointed recovery) | Multi-item or multi-stage batch processing with explicit per-record progress and idempotency.

 |
| **Class D** | Durable business workflow | **DBOS Transact Go** | Multi-day processes, durable sleeps, external signals/approvals, step checkpoints, and crash recovery.

 |

---

## 2. When to Use Each Category — and When Not To

### Class A: Synchronous Operations (Go Handler + Neon Transaction)

* **When to use:**
* The operation completes within a normal synchronous request/response budget (such as looking up a record, validating input, or executing a self-contained database mutation).


* The miniapp primarily provides a typed, permission-checked, audited tool boundary for an agent without needing background scheduling or durable waits.




* **When NOT to use:**
* The process involves multi-day waits, human approval pauses, or multi-stage recovery across restarts.


* The operation triggers unguarded external mutations before effect tracking, reconciliation, and approval controls are in place.





### Class B: One-Shot, Scheduled, or Delayed Jobs (River)

* **When to use:**
* A single task or scheduled run needs to be enqueued atomically inside the same Neon transaction as a business state change (`InsertTx`).


* The workload needs automatic retries with backoff, unique job deduplication, or delayed/periodic execution (such as sending a notification, generating a report, or handling a verified webhook).




* **When NOT to use:**
* The business process spans days, waits on external events or human decisions, or requires fine-grained step-by-step workflow checkpointing and replay.





### Class C: Bounded Multi-Step Batch Operations (River Default / DBOS When Justified)

* **When to use:**
* A bounded batch processes multiple records or dependent stages (such as importing or syncing a batch of records) and needs explicit per-record progress tracking, per-record idempotency keys, and resume-without-replaying completed items.


* Consider promoting to DBOS when mid-batch checkpointed recovery, cross-stage durability, or concurrency/rate-limit guarantees outweigh River's simpler job model.




* **When NOT to use:**
* The operation is a single atomic call (use Class A) or a single independent background task (use Class B).


* Do not treat a River job chain as a substitute for a durable workflow engine when long waits or complex workflow recovery are required.





### Class D: Durable Business Workflows (DBOS Transact Go)

* **When to use:**
* The process must survive agent session exits, process crashes, or deployments across multi-day waits (`Sleep`), external events (`Recv`/signals), or human approvals.


* The workflow coordinates multi-step business logic across systems where step checkpointing, rate-limited workflow queues, or explicit saga compensation is required.




* **When NOT to use:**
* Async execution, scheduling, or having more than one step occurs on its own—those properties alone do not justify DBOS.


* Never use DBOS as a default wrapper for every tool call, as checkpoint writes and determinism rules add overhead to straightforward operations.





---

## 3. Developer Dos and Don'ts by Category

### General Rules Across All Categories

* **DO** keep agents out of credential and durability ownership: never pass raw SaaS tokens, refresh tokens, or secrets into workflow payloads, logs, or agent outputs.


* **DO** enforce authorization, trust mode (`dry-run`, `confirm`, `autonomous`), blast-radius limits, and tenant isolation server-side on every invocation.


* **DO** route consequential external mutations through an idempotency/effect-tracking path (`PENDING -> ATTEMPTING -> CONFIRMED | FAILED | UNKNOWN`) and quarantine `UNKNOWN` outcomes for reconciliation rather than retrying blindly.


* **DO** keep Fly.io compute co-located with the Neon primary database and separate DBOS direct connections from pooled connections where required.


* **DON'T** let an agent approve its own gated write or rely solely on harness UI prompts to enforce permissions.


* **DON'T** dispatch the same miniapp invocation to both River and DBOS.



### Class A (Synchronous Handlers)

* **DO** execute state changes inside a clean Neon transaction with transaction-local tenant context (`SET LOCAL` / `set_config(..., true)`).


* **DO** return inline results synchronously for fast Class A operations and surface typed validation or authorization errors immediately.


* **DON'T** force agents through multi-step async polling loops for simple synchronous lookups or commands.


* **DON'T** assume an external SaaS write inside a handler has zero timeout/unknown-outcome risk without effect tracking and reconciliation.



### Class B & Class C (River Jobs & Bounded Batches)

* **DO** enqueue jobs atomically inside the database transaction that creates or updates the motivating business state (`InsertTx`).


* **DO** assign deterministic unique job keys and per-record idempotency keys (such as `tenant:batch:{id}:row:{n}`) so retries and duplicate webhooks collapse safely.


* **DO** record explicit batch progress for Class C workloads so interrupted batches resume without re-executing already-completed records.


* **DON'T** rely on River alone if the workload needs multi-day durable waits or full workflow step-replay semantics.



### Class D (DBOS Workflows)

* **DO** keep workflow functions deterministic: place clocks, randomness, external API calls, and non-deterministic I/O inside steps (`RunAsStep`) or transactions (`RunAsTransaction`).


* **DO** use deterministic workflow IDs (`tenant:entity:kind:op`) and partitioned queues (`entity_key`) when serializing operations per business entity.


* **DO** register explicit, idempotent compensation steps *before* executing reversible external mutations when building sagas, and plan workflow versioning/compatibility across deployments.


* **DON'T** spawn raw goroutines, iterate over Go maps non-deterministically, or perform side-effecting calls directly inside a workflow body outside step/transaction boundaries.


* **DON'T** assume a single always-on DBOS executor provides high availability without explicit restart, drain, and alerting procedures.



---

## 4. Organizing Business Logic Cleanly

When structuring a miniapp's codebase, organize logic by **boundary** rather than dumping orchestration, database transactions, and provider HTTP calls into one file:

1. **Contract & Policy Boundary:** Define typed input/output structs (generating JSON Schemas), idempotency key derivation, entity key rules, trust-mode requirements, and capability scope declarations in the manifest.


2. **Invocation & Guardrail Boundary:** Validate inputs, authenticate the caller/principal, check tenant/resource permissions, enforce rate/volume limits and kill switches, and return either a synchronous result or a handle (`run_id`) with status/signal/cancel support.


3. **Execution & State Boundary:** Isolate category-specific mechanics—plain transactional handlers for Class A, River job workers and batch progress trackers for Class B/C, and deterministic DBOS workflows, steps, events, and compensations for Class D.


4. **Capability & Effect Boundary:** Keep SaaS and custom REST clients behind capability adapters and a credential/token broker (with SSRF protections for custom REST endpoints), recording effect intents and reconciling `UNKNOWN` outcomes centrally.



If a business process combines multiple patterns—for example, validating input synchronously, running a batch import, and then waiting days for human sign-off—split the cohesive durable lifecycle into a **Class D** miniapp that delegates bounded sub-tasks to appropriate steps or jobs, rather than scattering half-finished state across agent prompts.



## deepseek

# App Category Classification: A Technical Guide for Business Logic Organization

**Audience:** Developers, architects, and reviewers building miniapps on the framework
**Scope:** Why we classify miniapps into categories, how to pick the right category, what each category obligates, and what goes wrong when classification is treated as an afterthought
**Governing principle:** The simplest execution mechanism that reliably satisfies the business requirement

---

## 1. Why classify at all

Classification exists to force explicit reasoning about four properties that are invisible in code review but decisive in production: **lifecycle**, **durability**, **failure semantics**, and **coordination**.

A function that "looks like" a CRUD handler can be a workflow in disguise. A function that "uses async" can be a request-scoped read in disguise. A function that is "one call" can have side effects that cannot be undone. Without an explicit category, these distinctions are invisible until a crash test or a production incident surfaces them — and then the code is already committed, tested, and deployed.

Classification makes four things possible that are otherwise impossible:

1. **Auditable decisions.** ADR-001's Gate 4 ("does the framework reduce total complexity?") is a judgment call. Categories make that judgment reviewable. Without them, "we chose DBOS" or "we chose River" is an assertion. With them, it is a decision with a record.

2. **Substrate-specific obligations.** Class A must own its transaction boundary. Class B must own its idempotency key. Class C must own its per-record progress. Class D must own its determinism. These obligations are not interchangeable. Without classification, they get applied inconsistently — Class C tools get built without per-record idempotency, Class D workflows get written with `time.Now()` inside the workflow body, Class A handlers get built with silent fire-and-forget side effects.

3. **Falsifiable acceptance criteria.** "This works" is not a test. "This is a Class C tool and it resumes without replaying completed records after a worker kill" is a test. Classification produces testable claims.

4. **Prevention of the two failure modes.** Under-classification (calling a Class D workflow "async") produces silent data loss. Over-classification (building a Class A read as a DBOS workflow "to be safe") produces unnecessary complexity, latency, and cost. Both are prevented by a taxonomy that names the difference.

There is also a fifth, less obvious benefit: **shared vocabulary**. When a reviewer says "this is Class C, where is your per-record idempotency key?" the conversation is precise. Without categories, the same review becomes "I'm worried about this" or "looks fine."

Classification is not bureaucracy. It is the minimal ceremony required to make workflow complexity a design decision rather than an accident.

---

## 2. The four categories, defined precisely

Each category is defined by **execution semantics**, not by code shape, provider choice, or function size.

### Class A — Request-scoped operation

**Definition:** The operation completes within the initiating request's lifecycle and has no independent execution state. If the request fails, the operation is safe to retry from the beginning. If the process dies mid-operation, nothing durable is lost.

**Semantic test:** *If the process dies immediately after the last line of this function executes, and the client retries, is anything corrupted, duplicated, or lost?* If "no" for all three, it is Class A.

**Substrate:** Plain Go handler + Postgres transaction.

**Examples:** read a contact, validate a form, compute a total, update a single record, check availability.

**What it is not:** A Class A function that "schedules" an email, enqueues a webhook, or "kicks off" a process is not Class A. It is Class B or C, because the work it initiates has an independent lifecycle.

### Class B — One-shot background job

**Definition:** A single independently retryable operation with a clear beginning and end, no durable progress across stages, and no dependency on the outcome of a prior job.

**Semantic test:** *Can this be safely retried with an idempotency key from a clean start, with no meaningful intermediate state that must be preserved?* If yes, it is Class B.

**Substrate:** River (`InsertTx` in the same transaction as the motivating write).

**Examples:** send a single notification, generate a report, process one webhook, sync one record.

**What it is not:** A Class B job that "also updates the CRM, then sends an email, then posts to Slack" is not Class B. It is Class C, because the chain has intermediate state that must be preserved on failure.

### Class C — Bounded multi-step operation

**Definition:** Multiple dependent operations with intermediate state, but the operation has a bounded duration, no durable waits longer than the retry timeout, and no requirement to resume from a checkpoint after a process death (as opposed to restarting the current step).

**Semantic test:** *Does this require checkpoint-level recovery — resuming from the exact step that succeeded rather than retrying from the start of the current operation?* If "no, restarting the current operation is fine" and the operation has multiple steps, it is Class C. If "yes, we must resume from step 4 not step 1," it is Class D.

**Substrate:** River with an explicit per-record progress table, **or** DBOS if a PILOT demonstrates that DBOS reduces total complexity for the specific operation (see §4.3).

**Examples:** import 500 contacts with validation, run a batch of dependent sync operations, send a campaign to a bounded recipient list.

**What it is not:** A Class C operation that "might take a few days" or "waits for approval" is not Class C. It is Class D, because those are durable waits.

### Class D — Durable business workflow

**Definition:** A business process whose progress must survive process death, deployments, and arbitrary time gaps; that contains durable waits (for external events, human decisions, or scheduled times); or that requires checkpoint-level recovery across multiple consequential stages.

**Semantic test:** *If the process is stopped for 6 days and then resumed, must it continue from the exact point it paused?* If yes, and if any of the following is also true — durable wait, human approval, multi-stage external effects, or checkpoint recovery required — it is Class D.

**Substrate:** DBOS workflow with deterministic code, checkpointed steps, and stable workflow ID.

**Examples:** multi-day lead follow-up, employee provisioning with approval gates, invoice-to-ledger with saga compensation, approval flow with escalation.

**What it is not:** A Class D workflow that "doesn't really need durability, but we wanted consistency with other workflows" is over-classified. See §6 for why that is a real cost.

### The borrowed-wait case (approval-gated commands)

An approval gate introduces a durable wait into a tool that would otherwise be Class A or B. This is not a fifth class — it is a Class A or B tool that borrows a bounded D-style wait. The mechanism is a single approval table with a state machine (`PENDING → APPROVED | REJECTED | EXPIRED`), `expires_at`, `material_hash`, and an out-of-band approver channel.

**Bounded wait** (e.g., ≤24h) → one-shot River job triggered on approval, no DBOS.
**Unbounded or multi-stage wait** → the containing process is Class D and the approval is a DBOS event.

The category of the *tool* does not change; the category of the *invocation* does. This is the one exception to "category is intrinsic to the tool," and it must be declared explicitly in the manifest.

---

## 3. The distinguishing questions

Every new tool should be classified by answering these questions in order. The first "yes" stops the walk.

1. **Does this operation have any state that must survive the current request?**
   - No → **Class A**. Complete the classification.
   - Yes → continue.

2. **Is it a single independently retryable operation with no meaningful intermediate state?**
   - Yes → **Class B**. Complete the classification.
   - No → continue.

3. **Does it require a durable wait (for a human, an external event, or a scheduled time longer than the retry timeout)?**
   - Yes → **Class D**. Complete the classification.
   - No → continue.

4. **Does it require checkpoint-level recovery — resuming from the exact successful step rather than restarting the current operation?**
   - Yes → **Class D**. Complete the classification.
   - No → continue.

5. **Does it have multiple dependent steps with intermediate state?**
   - Yes → **Class C**. Note the substrate decision (River + progress, or DBOS via PILOT).
   - No → return to question 1. The classification walk has a gap, and the tool needs to be re-scoped.

**Two questions this walk deliberately does not ask:**

- *"Is this async?"* — Asynchrony is a property, not a category. A Class A handler can call an async SDK. A Class D workflow can be entirely synchronous internally. The presence of `async`/`await`, goroutines, or queues is not diagnostic.
- *"How many lines of code?"* — A 30-line Class D workflow and a 300-line Class A handler are both valid. Size is not semantics.

If the classification walk produces "Class D" for a tool that a reviewer cannot explain in one sentence of business terms, the tool is probably over-classified. Revisit §4.4.

---

## 4. When each category is the right choice — and when it's tempting but wrong

### 4.1 Class A: The right choice for request-scoped correctness

**Use when:** The business promise is satisfiable within the request. "Show me this record." "Update this field to this value." "Validate this input against these rules."

**Tempting but wrong:**
- **"It needs to send an email, but the email can happen in the background."** — Then the tool is not the email send. The tool is the state change; the email is a Class B job enqueued in the same transaction. Class A can *enqueue* a Class B job. It cannot *silently perform* a side effect that isn't part of its transaction.
- **"It calls the provider twice, but both calls are idempotent."** — Then it is still Class A only if both calls are safe to retry from a clean start. If a crash between the two calls leaves the provider in an inconsistent state that the retry cannot resolve, it is Class B (with an idempotency key that covers both calls) or Class C (with intermediate state).
- **"The approval is just a flag on the record."** — Then it is not Class A. An approval is a wait; a wait is a lifecycle; a lifecycle is Class D. See the borrowed-wait case in §2.

**Over-classification trap:** Wrapping a Class A read in a DBOS workflow "for consistency." This is Class A over-classified. It costs a workflow record, a checkpoint write, a DBOS worker thread, and 30–200ms of latency, and returns no benefit.

### 4.2 Class B: The right choice for independently retryable single operations

**Use when:** The business promise is "this one thing will happen, exactly once (or at-least-once with provider idempotency)." One notification. One sync. One scheduled report.

**Tempting but wrong:**
- **"It sends an email and then updates a tracking table."** — Then it is Class C, because the tracking table is intermediate state that must survive a crash between the two steps. If the tracking table is the *only* side effect and the email is best-effort, it can be Class B with a compensating read on retry, but the classification must be stated.
- **"It retries up to 5 times."** — Retry count is not a class. River jobs retry; DBOS steps retry; plain handlers retry inside a transaction. Retry count is a policy, not a category.
- **"It's scheduled."** — Scheduling is a trigger, not a class. A scheduled Class A operation is still Class A. A scheduled Class B is still Class B (River periodic). A scheduled Class D is still Class D. Scheduling does not upgrade the class.

**Over-classification trap:** Wrapping a Class B job in a Class C chain "because eventually we might need to add a step." The classification is for the tool as it exists, not for the tool as it might be. If a future step is added, reclassify then. Write the seam (an explicit `Workflow` interface for a future Class C wrapper), but do not build the wrapper.

### 4.3 Class C: The right choice for bounded multi-step work — with a caveat

**Use when:** The operation has multiple dependent steps, a bounded duration, no durable wait, and no requirement to resume from a checkpoint (as opposed to restarting the current operation). Batch import with per-record validation. Synchronize a bounded set of records.

**The caveat — River vs. DBOS:** This is where ADR-001's Gate 4 must be run as a **PILOT**, not asserted. The default recommendation of "River by default, DBOS only when checkpointed recovery matters" is under-justified without measurement because River's OSS tier does not provide global concurrency limits, workflows, or partitioned concurrency (those are Pro-tier features). For RPA against rate-limited SaaS, global and per-tenant concurrency control is a core requirement, and DBOS queues provide it in the open-source library.

**The obligation:** A Class C tool must either (a) be a documented River implementation with an explicit progress table, per-record idempotency keys, and bounded worker concurrency, or (b) be a documented DBOS parent workflow with queued child workflows. The decision must be recorded as an ADR §9 PILOT with measured custom-code volume, crash-test results, and rate-limit behavior — not chosen by default.

**Tempting but wrong:**
- **"It has multiple steps and might wait for approval."** — Then the approval wait makes it Class D. Class C does not have durable waits.
- **"It runs for a few days, but each step is bounded."** — Then it is not Class C. Duration is not the criterion; durable resumption is. If the process resumes from the exact step after a 3-day gap, it is Class D.
- **"It's a River chain."** — A River chain that restarts from step 1 on failure is a Class B with retries, not a Class C. The distinction is whether intermediate state (progress) is preserved across retries. If it is not, the chain is Class B-shaped even if it has multiple steps.

**Over-classification trap:** Using DBOS for a Class C batch "because it's already in the stack." If the batch is bounded, has no durable waits, and does not require checkpoint resumption, River is the correct substrate. The DBOS overhead (deterministic code, checkpoint writes, worker thread) is a cost that only amortizes if the tool genuinely needs Class D semantics.

### 4.4 Class D: The right choice for durable business processes — and the only class that must be earned

**Use when:** The business promise cannot be satisfied without durable progress. "Onboard this employee over 14 days, with manager approval and provisioning across three SaaS systems." "Chase this invoice until paid, up to 90 days, with escalation after 30." "Post this invoice to the ledger, with saga compensation if any step fails."

**The discipline:** Class D is the only category that must pass ADR-001's Gates 1–4 before adoption. A Class D tool that cannot articulate (a) the genuine durability requirement, (b) at least two orchestration-complexity characteristics, (c) the business consequence of failure, and (d) the net complexity reduction versus the custom machinery it replaces, must not use DBOS. It is over-classified.

**Tempting but wrong:**
- **"It calls two APIs in sequence."** — Then it is Class A or B. Sequence is not durability.
- **"It might need to wait for something someday."** — YAGNI. Classify for the tool that exists. Add a seam if a future wait is plausible; do not classify for a hypothetical future.
- **"It's a Class A tool but we're using a DBOS workflow for consistency."** — Then it is over-classified. See §4.1's over-classification trap.
- **"It's a batch, but we want durability."** — A batch that does not resume from the exact step is Class C. Durability of individual retries is River's job. Checkpoint-level recovery is what distinguishes D from C, and if the batch does not need it, it is not D.

**Under-classification trap:** Calling a Class D workflow "async." An async Class A tool that enqueues a chain of follow-up operations, with no durable state between them, is a Class D tool that has been under-classified. The chain will lose progress when the process dies, and the failure will be silent. If a tool has a chain and the chain's progress matters, it is Class D. Do not let "it's async" hide a workflow.

---

## 5. The classification workflow

Use this process for every new tool, and re-run it whenever the tool's semantics change.

**Step 1 — Describe the business promise in one sentence.** "When invoked, this tool will [promise]." The promise must be customer-observable, not implementation-observable. "Send an email" is implementation. "Notify the customer within 5 minutes" is a promise.

**Step 2 — Apply the four distinguishing questions** from §3.

**Step 3 — Score against the seven dimensions** from ADR-001: lifetime, stages, waiting, recovery, effects, coordination, human. The score is diagnostic, not dispositive — a low score with a critical durable wait is still Class D.

**Step 4 — Select the class.** Write it down.

**Step 5 — Verify against the manifest.** The manifest (`miniapp.json`) must declare the class per tool. A code reviewer must be able to verify that the code's behavior matches the declared class. Drift is a correctness bug.

**Step 6 — Record the decision.** For Class A and B, the record is the manifest. For Class C, the record is the manifest plus the ADR §9 PILOT note. For Class D, the record is the manifest plus an `adr001.md` file (see the plan's §5.2 template).

**Step 7 — Re-review on any semantic change.** Adding a durable wait, removing a durable wait, changing the entity key, changing the idempotency identity, or adding/removing a compensating action are all re-classification triggers. Re-classification is not a new tool; it is a new version of the same tool, with a new decision record.

**Step 8 — Test at the boundary of the declared class.** Class A tests crash-boundary safety of the transaction. Class B tests idempotency-key retries. Class C tests per-record resume-without-replay. Class D tests checkpoint recovery with no repeated confirmed effects. A Class D tool tested only for happy-path is untested.

---

## 6. The DOS

1. **Classify before coding.** Write the class in the manifest before the first handler. A class chosen after implementation is a justification, not a decision.

2. **Declare the class per tool, not per miniapp.** A miniapp with three tools may have three classes. Do not classify "the miniapp" — classify each tool's business promise.

3. **State the business promise explicitly.** The promise is what the tests verify. "Post invoice to ledger" is a promise. "Call `accounting.bill.create`" is not.

4. **Declare the idempotency identity** for every tool that has external effects. The identity is `tenant:entity:kind:op` or an equivalent stable key. A tool with external effects and no declared idempotency identity is not classified — it is deferred.

5. **Declare the entity key** for every write tool. The entity key is the business identifier (invoice ID, employee ID, lead ID) that serializes concurrent writes. Tools without an entity key must justify why (`no_entity: <reason>` in the manifest).

6. **Mark external effects as intents, not actions.** The workflow expresses intent; the capability proxy performs the action. This is the discipline that makes effects auditable and reconciliable.

7. **Register compensation before the action** in Class D workflows with sagas. A compensation registered after the action cannot undo a crash that occurs between the action and the registration.

8. **Declare the failure mode for each step.** `retryable`, `terminal`, `unknown → quarantine`, `needs_approval`, `needs_reauth`. An undeclared failure mode defaults to "unknown," which is the safest default but the least actionable.

9. **Test at the crash boundary of the declared class.** Class C tested only for happy path is not tested. Class D tested only for happy path is not tested. Each class has a specific crash boundary (per-record for C, per-step for D) and the tests must exercise it.

10. **Specify the version strategy for Class D.** Every Class D workflow must declare what happens when the workflow code changes while instances are running: version pin, patch, drain, or route. "We'll figure it out" is a production incident.

11. **Review the classification in code review.** A reviewer's first question is "what class is this tool?" If the answer is not immediate, the classification is not done.

12. **Re-classify on semantic change.** A new wait, a new external effect, a new compensating action, a changed entity key, or a changed idempotency identity all trigger re-classification. Re-classification is a version, not a patch.

---

## 7. The DON'TS

1. **Don't classify by code shape.** The presence of `async`, `await`, `go func`, or a queue library is not diagnostic. Classification is by execution semantics, not syntax.

2. **Don't classify by provider choice.** Calling Stripe does not make a tool Class C. Calling two Stripe endpoints does not make it Class D. The class is determined by lifecycle, durability, failure semantics, and coordination — not by which vendor is on the other end.

3. **Don't mix classes within a tool.** A tool is one class. If it needs Class A and Class D behaviors, it is two tools or it is Class D. Mixing produces a tool that is untestable at either class's crash boundary.

4. **Don't upgrade a class "to be safe."** Building a Class A read as a Class D workflow costs a workflow record, a checkpoint write, a worker thread, and 30–200ms of latency, and returns no benefit. Over-classification is as much a bug as under-classification.

5. **Don't downgrade a class "because it's simpler."** Calling a Class D workflow "async" hides the durable state that the chain actually needs. The simplification is fictional; the data loss is real.

6. **Don't use the workflow ID as an idempotency key for external effects.** Workflow IDs are for workflow deduplication. External effects need provider-level idempotency or read-before-write reconciliation. The two are different mechanisms for different problems.

7. **Don't put credentials, PII, or large payloads in workflow checkpoints.** Workflow checkpoints persist step inputs and outputs. Persist IDs and compact context. Store PII in application tables with a deletion path. This is a data protection obligation, not a preference.

8. **Don't run non-deterministic code outside steps in Class D.** `time.Now()`, `rand.Int()`, `uuid.New()`, HTTP calls, and DB queries inside the workflow body (as opposed to inside `RunAsStep`/`RunAsTransaction`) will produce divergent replays and silent corruption. The determinism linter must catch this; if it does not, it is broken.

9. **Don't use goroutines, `select`, or map iteration inside workflow bodies.** Use the durable execution framework's equivalents (`dbos.Go`, `dbos.Select`, sorted keys). The linter must catch this; if it does not, it is broken.

10. **Don't skip the reconciler for any tool with external effects.** An external effect with `UNKNOWN` outcome and no reconciler is a silently lost operation. The reconciler is not optional for B/C/D tools with external effects, and it is the reason Class B/C tools cannot ship before the reconciler exists.

11. **Don't approve in-band.** An agent that can approve its own `confirm` request defeats the approval gate. The approver must be a separate authenticated channel (out-of-band UI, magic link with step-up), with separation of duties. Harness elicitation is convenience; it is not the control.

12. **Don't trust the manifest to match the code without a drift test.** A manifest that declares "Class A" for a tool that enqueues a Class D workflow is a metadata bug that becomes a correctness bug. CI must verify that declared classes match observable behavior (via a drift test that inspects the tool's substrate usage).

13. **Don't use "might need to be durable later" as a classification.** YAGNI applies. Classify for the tool that exists. Write the seam for the future class if a future change is plausible. Do not build for a hypothetical.

14. **Don't classify a miniapp by its most complex tool.** If a miniapp has one Class A read and one Class D workflow, the miniapp's manifest declares both classes, per tool. There is no "miniapp class."

15. **Don't let "confirm" mode hide a Class D workflow inside a Class A tool.** An approval gate on a Class A tool introduces a durable wait. If the wait is bounded and single-step, it is the borrowed-wait case (approval table + one-shot River job). If the wait is unbounded or multi-stage, the tool is Class D. The classification must match the actual wait semantics, not the pre-approval shape.

---

## 8. Class-specific engineering obligations (quick reference)

| Obligation | Class A | Class B | Class C | Class D |
|---|---|---|---|---|
| Substrate | Plain handler + Postgres tx | River | River + progress **or** DBOS via PILOT | DBOS |
| Transaction boundary | Single Postgres tx | `InsertTx` in domain tx | Per-record tx | `RunAsTransaction` |
| Idempotency identity | Optional (retry from start is safe) | Required | Required per record | Required per step |
| Entity key | Required for writes | Required for writes | Required per record | Required per workflow |
| External effects | Through capability proxy | Through capability proxy | Through capability proxy | Through capability proxy |
| Reconciler | Required if external effects | Required | Required | Required |
| Approval gate | Borrowed-wait if needed | Borrowed-wait if needed | Borrowed-wait if needed | DBOS event |
| Checkpoint recovery | N/A | N/A | Not required (restart current op) | Required |
| Determinism lint | N/A | N/A | N/A (unless DBOS) | Required |
| Crash-boundary tests | Transaction | Idempotency retry | Per-record resume | Per-step resume |
| Version strategy | N/A | N/A | N/A (unless DBOS) | Required |
| ADR-001 record | Manifest | Manifest | Manifest + PILOT note | Manifest + `adr001.md` |
| Complexity score | N/A | N/A | Required for PILOT | Required |

---

## 9. The reclassification rule

A tool's class is not permanent. Semantic changes trigger re-classification:

**Triggers:**
- A new durable wait is added (Class A/B/C → D)
- A durable wait is removed (Class D → C or lower)
- A new external effect is added (may trigger reconciliation requirements)
- A compensating action is added or removed (affects saga classification)
- The entity key changes (affects serialization)
- The idempotency identity changes (affects retry safety)
- The duration changes from bounded to unbounded (Class C → D)
- The duration changes from unbounded to bounded (Class D → C, if no other D criteria apply)

**The mechanics:** Re-classification is a new version of the tool, with a new decision record. Old workflow instances continue to use their original class and substrate. Migration from old to new class is a separate decision, with its own ADR §9 PILOT and migration plan.

**The cost of re-classification:** Upgrading from Class A to Class D is a rewrite of the substrate. Downgrading from Class D to Class C is a migration of in-flight workflows. Both are expensive. This is why classification happens before coding and not after — the cost of a misclassification discovered in production is the cost of an emergency re-architecture.

---

## 10. What this discipline prevents

Classification is not a process for its own sake. It prevents four specific failures that have killed more workflow projects than any technical issue:

**1. The DBOS-everywhere trap.** Without classification, every async operation looks like a candidate for durable execution. The substrate gets applied to everything, the write amplification compounds, the latency degrades, and the cost of running the platform becomes dominated by DBOS checkpoints for operations that were never durable. Classification forces the ADR-001 question per tool.

**2. The "it's just async" trap.** Without classification, a chain of follow-up operations looks like "async work." It gets built as a series of fire-and-forget River jobs with no per-record progress, no reconciliation, and no checkpoint. When a worker dies mid-chain, the chain is silently lost and the business promise is broken. Classification forces the chain to be named as Class C or Class D and to carry the corresponding obligations.

**3. The Zerodha v1 trap applied to workflow classification.** Zerodha's failure was building a bounded expression language for an unbounded domain and growing the language until it became a bad programming language. The same failure mode applies here: building all operations as Class A until a Class D requirement surfaces, then bolting DBOS onto a Class A handler, then discovering the handler was never structured for durability. Classification prevents the growth-by-accretion pattern by forcing the class decision at design time.

**4. The configuration graveyard applied to workflow sprawl.** A catalog of hundreds of workflows, each individually reasonable, collectively unmanageable — no one knows which is which, what any given tool's failure semantics are, or which tools would be affected by a substrate change. Classification with mandatory decision records prevents the graveyard by making each workflow's category, substrate, and failure mode inspectable.

The discipline is the point. Categories without discipline become labels. Categories with discipline become the interface between business intent and technical execution — the one place where "what the business needs" and "what the substrate provides" are explicitly compared, and the comparison is recorded.


## chatgpt

The central message for developers should be: **choose an execution category based on the business operation's lifecycle and failure-recovery requirements, not based on how complicated its Go code looks.**

The writeup below follows `plan4.2`, including the A/B/C/D taxonomy, selective DBOS adoption, the River-versus-DBOS decision for Class C, and the separation between execution semantics and compute placement.

# Choosing the Right Execution Category for a Miniapp

## A developer's guide to organizing business logic, execution, and responsibility

### 1. Why classify miniapps at all?

A business process can look deceptively simple when viewed through its API.

Consider an operation called `follow-up-lead`.

At first glance, it might appear to be nothing more than updating a CRM record and sending an email. A developer could implement it as an ordinary HTTP handler, call the CRM API, send the message, and return a response.

But consider what happens when:

- The CRM accepts the update but the connection times out.
- The email provider accepts the message, but the application crashes before recording success.
- The customer does not respond for three days.
- A follow-up must happen again after seven days, unless the customer has already converted.
- An operator needs to cancel the sequence.
- The application is deployed twice while the process is waiting.
- An agent retries the original request.

The original implementation was not necessarily incorrect because its Go code was poorly written. It was incorrect because its execution model did not represent the actual business process.

This is the problem that execution categories solve.

The purpose of classification is to make developers answer an important architectural question before implementation:

**What execution guarantees does this business operation actually require?**

The answer determines how the operation is executed, how its state is managed, how failures are recovered, and which infrastructure responsibilities it must own.

Without classification, developers tend to make one of two mistakes:

1. **Under-engineering:** implementing a durable business process as a simple handler or background job, then discovering that retries, recovery, and partial failures require substantial custom machinery.
2. **Over-engineering:** introducing a workflow engine, durable state, queues, and additional infrastructure for an operation that could have been completed by a simple Go handler and a PostgreSQL transaction.

Neither is desirable.

The objective is not to make every miniapp sophisticated. It is to use the simplest execution mechanism that satisfies the business requirement.

That is the governing principle behind ADR-001.

---

## 2. What exactly are we classifying?

We classify the execution semantics of a business operation, not its business domain, programming language, API provider, or deployment location.

A miniapp represents a versioned, typed, permissioned business outcome.

Examples include:

- Look up a CRM contact.
- Send a notification.
- Import a collection of contacts.
- Follow up with a lead over several days.
- Onboard an employee across multiple business systems.
- Post an approved invoice to an accounting system.

These operations have different execution requirements, even though they may all use the same Go framework, PostgreSQL database, and SaaS integrations.

The framework defines four execution categories:

| Category | Name                                  | Primary responsibility                     | Execution mechanism                        |
| -------- | ------------------------------------- | ------------------------------------------ | ------------------------------------------ |
| A        | Ordinary operation                    | Complete one request                       | Go handler + PostgreSQL transaction        |
| B        | One-shot background job               | Execute one independently retryable unit   | River by default                           |
| C        | Bounded batch or multi-step operation | Coordinate a bounded collection of work    | River or DBOS queue, selected per workload |
| D        | Durable business workflow             | Coordinate a long-running business process | DBOS                                       |

The category is declared per tool in the miniapp manifest. Do not assume that every operation in a business domain must belong to the same category.

A related but separate decision determines where the code executes: local process, Fly Machine, Sprite, or hybrid execution. This is covered in Section 9.

---

## 3. Class A — Ordinary operations

**The defining characteristic: the operation can complete within one request without requiring an independent execution lifecycle.**

Class A is the simplest and should be the default.

A normal Go handler performs the business operation, uses PostgreSQL for its transaction, and returns a typed result directly to its caller.

There is no need to introduce a job queue or workflow engine.

### When to use Class A

Use Class A when:

- The operation is short-running and request-bound.
- The caller expects an immediate result.
- The operation does not need durable waiting.
- The operation does not need independent background retries.
- Its database changes can be handled in an ordinary transaction.
- If interrupted before completion, it can safely be retried under its defined idempotency rules.

Typical examples:

| Business operation   | Reason                                     |
| -------------------- | ------------------------------------------ |
| `record-lookup`      | Retrieve and return a contact              |
| `validate-customer`  | Perform a bounded validation               |
| `calculate-quote`    | Calculate a result and return it           |
| `get-account-status` | Return current account state               |
| `record-update`      | Apply a short, authorized database command |

Class A must return its typed result inline. It should still record an invocation and audit identity, but it does not require the caller to poll `get_run`.

### Where its business logic belongs

A typical organization is:

```
miniapps/
  crm-read/
    miniapp.json
    handler.go
    service.go
    repository.go
    fixtures/
```

The handler handles the invocation contract and authorization context.

The service contains the business operation.

The repository handles database access.

The handler should not become a collection of raw SQL statements and provider API calls.

### Do

- Keep request execution synchronous.
- Use normal Go context cancellation.
- Keep database transactions short.
- Apply application authorization before execution.
- Use explicit idempotency identity for consequential commands.
- Serialize conflicting writes to the same entity.
- Record invocation and audit information.
- Return typed, bounded results.

### Don't

- Add DBOS merely because a handler calls two or three functions.
- Add River merely to make an ordinary API call asynchronous.
- Return a run handle when an inline result is sufficient.
- Move business rules into HTTP or MCP transport adapters.
- Assume that a PostgreSQL transaction makes an external SaaS operation atomic.
- Blindly retry a provider operation after an unknown outcome.

### An important exception: external writes

A single-request operation can still modify an external system.

For example, `record-update` may be Class A from an execution perspective.

But if it writes to a SaaS provider, its external effect must use the framework's effect protocol.

The relevant state machine is:

```
PENDING
   |
   v
ATTEMPTING
   |
   +----> CONFIRMED
   |
   +----> FAILED
   |
   +----> UNKNOWN
```

A timeout after the provider accepted the request is UNKNOWN, not FAILED.

The operation must not be blindly repeated.

The effect ledger, reconciliation, authorization, and required approval controls must be available before real autonomous or confirmed writes are released.

**Execution Category A does not mean low-risk or permission-free. It means request-bound execution.**

---

## 4. Class B — One-shot background jobs

**The defining characteristic: the operation should execute independently of the original request, but it does not require a durable multi-stage business workflow.**

Class B is for work that can be represented as one independently retryable unit.

The caller submits the work. The application records it and a worker executes it later.

River is the default execution mechanism.

### When to use Class B

Use Class B when:

- The operation does not need to finish within the original request.
- It can execute independently.
- A failed attempt can be retried safely.
- A stable unique identity can prevent duplicate submissions.
- It has a bounded amount of work.
- It does not need to retain a complex multi-stage business process across long waits or external decisions.

Examples:

| Business operation | Reason                              |
| ------------------ | ----------------------------------- |
| `notify-send`      | One message delivery                |
| `report-generate`  | Generate and deliver one report     |
| `webhook-process`  | Process one received event          |
| `send-reminder`    | Dispatch one scheduled notification |
| `document-export`  | Produce one export artifact         |

### Where its business logic belongs

```
miniapps/
  notify-send/
    miniapp.json
    handler.go
    job.go
    service.go
    provider.go
    fixtures/
```

The invocation handler validates and enqueues the job.

The worker executes the job.

The service contains the actual business operation.

The provider adapter handles communication with the external system.

A job payload should normally contain identifiers and compact execution context, not entire customer records or credentials.

### Do

- Use stable job identities.
- Use unique constraints or River uniqueness where appropriate.
- Enqueue transactionally with the motivating database change when required.
- Set bounded retry policies and backoff.
- Make job execution idempotent.
- Record terminal failures.
- Apply provider rate limits and shared concurrency controls.
- Make job status observable.

### Don't

- Use River as a substitute for a durable workflow engine.
- Assume a unique job key makes the external provider operation exactly-once.
- Put large customer objects or secrets in job payloads.
- Start unbounded workers for provider operations.
- Assume per-process worker concurrency is a global provider rate limiter.
- Build an application-owned workflow engine merely to make one-shot jobs work.

### A useful test

Ask:

"If this job crashes, can I safely retry the complete unit from the beginning without reconstructing a complex sequence of previously completed business stages?"

When the answer is yes, Class B is a strong candidate.

If not, investigate Class C or D.

---

## 5. Class C — Bounded batches and multi-step operations

**The defining characteristic: the operation coordinates a bounded amount of related work, but it does not automatically require a full durable business workflow.**

Class C exists because many business operations are more complicated than a single job but do not necessarily justify DBOS workflow orchestration.

Examples include importing hundreds of contacts, synchronizing a bounded data set, or processing a collection of invoices.

These operations often require:

- Per-record progress.
- Per-record idempotency.
- Bounded concurrency.
- Retry and resume behavior.
- Provider rate limits.
- Fan-out and aggregation.
- Error collection and reporting.

The critical decision is not simply whether the operation contains multiple steps.

It is how much coordination and recovery machinery the operation genuinely needs.

### When to use Class C

Consider Class C when:

- The operation handles a finite batch or bounded collection.
- Each item has an independently identifiable outcome.
- Completed records should not be repeated unnecessarily.
- The process needs explicit progress.
- Parallelism and rate limits matter.
- The operation may resume after a worker interruption.
- The complete process does not necessarily require the richer semantics of a Class D workflow.

Examples:

| Business operation  | Why Class C is plausible                            |
| ------------------- | --------------------------------------------------- |
| `contact-import`    | Per-record validation, persistence, retry, progress |
| `bulk-update`       | Bounded records with independent outcomes           |
| `data-sync`         | Pull, merge, push, conflict handling                |
| `bulk-notification` | Many independently tracked deliveries               |

### River versus DBOS: a deliberate decision

Class C has no automatic substrate.

There are two competing implementations.

**Alternative A — River plus application-owned progress**

The application maintains the progress records and item-level idempotency.

River provides job execution, retries, scheduling, and uniqueness.

This may be simpler when the batch is bounded and the application-owned state is straightforward.

**Alternative B — DBOS queue-based execution**

DBOS queues may provide more useful built-in concurrency, partitioning, rate limiting, and checkpoint semantics.

This may be simpler when the workload requires substantial orchestration or coordination.

Neither option is automatically correct.

For the forcing workload `contact-import`, implement and compare the alternatives when the relevant capabilities materially affect the result.

Measure:

- Custom code volume.
- Transaction behavior.
- Throughput.
- Per-tenant and per-provider rate-limit correctness.
- Per-entity serialization.
- Crash recovery.
- Resume behavior.
- Operational burden.
- Licensing and infrastructure cost.

Then record the decision in the workload's `adr001.md`.

### Where its business logic belongs

```
miniapps/
  contact-import/
    miniapp.json
    handler.go
    batch.go
    record.go
    progress.go
    repository.go
    fixtures/
```

The batch coordinator defines the batch's lifecycle and progress model.

The record processor handles one individual record.

The provider and repository adapters handle their respective I/O.

The underlying business rules should remain reusable rather than becoming tied to River-specific or DBOS-specific interfaces.

### Do

- Define a stable batch identity.
- Define a stable record identity.
- Track progress explicitly.
- Make individual record processing idempotent.
- Bound concurrency.
- Enforce shared provider rate limits.
- Distinguish retryable, terminal, and unknown outcomes.
- Test interruption and resume behavior.
- Record the River-versus-DBOS decision with evidence.

### Don't

- Assume every batch belongs on River.
- Assume every batch belongs on DBOS.
- Implement an unbounded custom workflow engine using job tables.
- Maintain multiple competing progress authorities.
- Treat worker concurrency as global rate limiting.
- Reprocess completed records unnecessarily.
- Hide partial failures behind an overall successful batch status.

**Class C is where architectural comparison matters most.**

A small amount of explicit progress state is often reasonable. Rebuilding durable timers, recovery, checkpointing, workflow versioning, and orchestration is a different proposition.

---

## 6. Class D — Durable business workflows

**The defining characteristic: the business process must survive the lifetime of the process that initiated it and retain meaningful execution state across stages, waits, interruptions, and deployments.**

Class D is for business processes whose execution lifecycle is itself part of the product's correctness contract.

DBOS is the execution engine for Class D.

### When to use Class D

A process becomes a strong Class D candidate when it requires durable execution and meaningful orchestration.

Examples include:

| Business operation       | Why Class D is appropriate                            |
| ------------------------ | ----------------------------------------------------- |
| `follow-up-sequence`     | Multi-day waits, customer responses, cancellation     |
| `employee-provisioning`  | Multiple SaaS systems, manager approval, compensation |
| `invoice-to-ledger`      | Validation, approval, posting, reconciliation         |
| `appointment-journey`    | Reminders, responses, rescheduling, cancellation      |
| `campaign-orchestration` | Fan-out, rate limits, results and recovery            |

Consider an employee-provisioning process:

1. Receive an onboarding request.
2. Validate the employee and organization.
3. Create an identity-provider account.
4. Provision access to business applications.
5. Wait for manager approval.
6. Provision payroll access.
7. Confirm completion.
8. Notify the relevant parties.

If the process crashes after step four, it must not forget which systems have already been provisioned.

If the manager takes three days to approve, the process must remain meaningful while it waits.

If a deployment occurs while the process is waiting, the old execution must remain recoverable under its established workflow version.

This is not merely a collection of API calls.

It is a durable business process.

### Where its business logic belongs

```
miniapps/
  employee-provisioning/
    miniapp.json
    workflow.go
    steps.go
    approval.go
    policy.go
    service.go
    effects.go
    compensation.go
    fixtures/
```

The workflow defines the sequence and durable control flow.

Individual steps perform nondeterministic operations through appropriate DBOS durable boundaries.

Domain services contain reusable business rules.

Capability clients communicate with external providers.

The framework owns identity, authorization, effects, approvals, audit, and operational recovery.

DBOS owns workflow execution and checkpoint recovery. It does not own business authorization or make external effects exactly-once.

### Do

- Give every workflow a stable business identity.
- Define a distinct DBOS `AppName` for each Class D miniapp/workflow family.
- Manage `ApplicationVersion` explicitly.
- Retain old workflow artifacts and worker versions until their executions drain or a verified compatibility strategy permits retirement.
- Put nondeterministic operations and external I/O behind appropriate durable steps or transactions.
- Use stable effect identities and the effect ledger.
- Keep workflow payloads small and avoid storing credentials or unnecessary personal information.
- Define cancellation, retry, terminal failure, and operator recovery policies.
- Test replay and worker-restart behavior.
- Monitor pending, failed, stuck, and quarantined workflows.

### Don't

- Use DBOS just because a handler has several function calls.
- Put arbitrary HTTP requests or database reads directly in deterministic workflow control flow.
- Use current time, randomness, uncontrolled goroutines, or nondeterministic iteration in workflow logic.
- Assume a stable workflow ID serializes writes to a business entity.
- Assume DBOS guarantees exactly-once external side effects.
- Deploy incompatible workflow code without a version strategy.
- Retire an old workflow version while active executions still depend on it.
- Put full customer records, access tokens, passwords, or connection strings in workflow payloads.
- Treat a workflow engine as a replacement for authorization, idempotency, reconciliation, or operational ownership.

DBOS is justified when its durability and orchestration reduce the total complexity of satisfying the business promise.

It is not justified merely because it has useful features.

---

## 7. The decision framework: how to choose a category

Developers should make the classification decision before creating the implementation.

Start with the business promise, then follow the execution requirements.

### Step 1 — Is the operation request-bound?

Can it complete within one request and return a result without independent retries or durable waiting?

- Yes: Class A is the default candidate.
- No: continue.

### Step 2 — Is it one independently retryable unit?

Can the operation execute as one bounded job, with stable identity and safe retry behavior?

- Yes: Class B is the default candidate.
- No: continue.

### Step 3 — Is it a bounded batch or multi-step operation?

Does it process a finite collection with record-level progress, idempotency, concurrency, and resume requirements?

- Yes: evaluate Class C.
- Compare River and DBOS queues where the workload's coordination requirements make the difference material.
- No: continue.

### Step 4 — Does the business process require durable orchestration?

Must it survive process termination, retain checkpoints, wait durably for people or external events, resume across deployments, or coordinate consequential stages with expensive recovery?

- Yes: evaluate Class D.
- Require the ADR-001 workload decision and human approval.

### Decision summary

| Question                         | A                   | B                      | C                   | D                    |
| -------------------------------- | ------------------- | ---------------------- | ------------------- | -------------------- |
| Returns inline result            | Yes                 | No                     | No                  | No                   |
| Independent background execution | No                  | Yes                    | Yes                 | Yes                  |
| Independently retryable unit     | Optional            | Core property          | Per item/stage      | Per durable step     |
| Explicit progress model          | Usually unnecessary | Job status             | Required            | Workflow checkpoints |
| Durable waits                    | No                  | Simple scheduling only | Workload-dependent  | Yes                  |
| Multi-stage orchestration        | No                  | No                     | Bounded             | Durable              |
| Engine                           | Go + PostgreSQL     | River                  | River or DBOS queue | DBOS                 |
| Stable run handle                | No                  | Yes                    | Yes                 | Yes                  |

The table describes typical execution shapes, not a substitute for reviewing the business requirement.

---

## 8. ADR-001: when DBOS earns its place

Class C and Class D miniapps require a workload-specific `adr001.md`.

The developer must not select DBOS merely by declaring a manifest category.

The decision must answer four qualification gates.

### Gate 1 — Is durable execution genuinely required?

Does the process need recovery after process termination, durable waits, checkpoint resumption, continuity across deployments, or durable waiting for an event or human decision?

If a complete restart is safe and no meaningful progress must be preserved, DBOS may be unnecessary.

### Gate 2 — Is there meaningful orchestration complexity?

Look for at least two substantial characteristics:

- Multiple independently committed stages.
- Delays or long waits.
- External events or callbacks.
- Human approval.
- Multiple dependent external services.
- Inter-stage retries.
- Fan-out and concurrency management.
- Compensation or partial-progress recovery.
- Cancellation, resumption, or operator intervention.
- Execution-history requirements.

### Gate 3 — Does failure have meaningful business consequences?

Ask what happens when the operation stops halfway.

Could it cause duplicate payments, incomplete employee onboarding, lost customer follow-ups, incorrect bookings, repeated messages, or expensive manual reconciliation?

A durable execution engine becomes more valuable when failure can break a business promise.

### Gate 4 — Does DBOS reduce total complexity?

Compare:

**Alternative A: ordinary Go/PostgreSQL/River**

What state, timers, retries, checkpointing, recovery, coordination, versioning, and reconciliation machinery would the team build and maintain?

**Alternative B: DBOS**

What does DBOS provide, and what still remains the application's responsibility?

DBOS may reduce workflow infrastructure, but the application still owns business authorization, external-effect idempotency, provider reconciliation, policy, and operational oversight.

The decision is a human architectural judgment, supported by automated checks and tests.

A complexity score is a supporting decision aid, not an automatic engine selector.

### Required decision record

Every Class C/D decision should record:

```
Miniapp:
Business promise:
Execution category:
Business failure consequence:

Complexity score:
Gate 1 — durable requirement:
Gate 2 — orchestration characteristics:
Gate 3 — business consequences:

Alternative A — Go/Postgres/River:
Custom machinery required:

Alternative B — DBOS:
Capabilities reused:
Responsibilities still owned by the application:

Operational cost:
Residual complexity:
Evidence and tests:

Decision: REJECT / DEFER / PILOT / ADOPT
Reviewer:
Date:
```

A Class C decision must include the River-versus-DBOS comparison when the relevant concurrency, rate-limit, partitioning, or recovery behavior is material.

---

## 9. Do not confuse execution category with compute placement

This is one of the most important rules in the architecture.

Execution category answers:

**How must this business operation execute and recover?**

Compute placement answers:

**Where should the code run, considering its trust profile and infrastructure requirements?**

These are independent decisions.

For trusted production code, all four categories normally execute on Fly Machines:

| Execution category | Typical placement |
| ------------------ | ----------------- |
| A                  | Fly Machine       |
| B                  | Fly Machine       |
| C                  | Fly Machine       |
| D                  | Fly Machine       |

Sprites are reserved for explicitly isolated development workspaces and untrusted or agent-authored code.

A durable workflow may orchestrate a bounded Sprite task, but DBOS owns workflow durability and Neon owns authoritative task identity and results.

An agent invoking a miniapp does not automatically imply that the miniapp must run in a Sprite.

### Don't do this

```
Agent call
    |
    v
Always create a Sprite
    |
    v
Run miniapp
```

### Do this instead

```
Agent call
    |
    v
Evaluate execution category
    |
    v
Evaluate code trust and placement policy
    |
    v
Select execution mechanism and deployment target
```

Never choose a more complicated execution category merely because a deployment target supports it.

---

## 10. Cross-cutting patterns are not additional categories

Some architectural concerns apply across A, B, C, and D.

They must not be turned into new execution categories.

### Approval

Approval is a policy gate, not a fifth execution class.

A single-step Class A or B command can create a durable approval request with:

- `PENDING`
- `APPROVED`
- `REJECTED`
- `EXPIRED`

Approval binds to a canonical hash of the material operation and its inputs, requester, approver, expiry, and decision audit.

Once approved, a valid short Class A operation can execute inline, or a Class B operation can be queued.

A larger Class D workflow can wait on a DBOS event while the same approval record remains authoritative.

Do not introduce DBOS merely because a single-step command needs approval.

### Entity-level serialization

This is a concurrency pattern, not a category.

Every consequential write declares an `entity_key` or an explicitly reviewed `no_entity` justification.

Typical implementation:

| Category | Entity serialization                                                     |
| -------- | ------------------------------------------------------------------------ |
| A        | PostgreSQL row lock or transaction-scoped advisory lock                  |
| B        | Stable unique job/effect identity plus entity lock                       |
| C        | Per-record identity plus entity lock or selected DBOS partition          |
| D        | DBOS partitioned queue where appropriate, concurrency one per entity key |

Workflow IDs alone do not protect an entity against conflicting operations.

Test that operations on the same entity serialize while independent entities can proceed concurrently.

### External effects and idempotency

Provider writes use the framework's effect ledger irrespective of category.

Use stable business identities, provider-supported idempotency where available, and reconciliation for unknown outcomes.

Do not confuse a category with a delivery guarantee.

### Saga and compensation

A Saga describes how to recover or compensate for partial business progress.

It may be useful inside a Class D workflow, but Saga by itself does not make an operation Class D.

Register compensation before the forward side effect, make compensators idempotent, and do not automatically compensate an UNKNOWN outcome without a safe basis.

### Security and policy

All categories retain the same fundamental boundaries:

- Agent principal and authority principal remain separate.
- Permissions are checked server-side.
- Effective provider permissions are constrained by manifest policy, granted scopes, and SaaS-side privileges.
- Blast-radius controls and kill switches apply irrespective of category.
- Provider-returned free text is untrusted data.
- Credentials never become business-workflow payloads or agent-visible results.

A simple operation still requires proper security engineering.

---

## 11. Organizing business logic in Go

Execution category should influence how the use case is executed, not cause the business rules to be duplicated across different implementations.

A recommended conceptual structure is:

```
miniapps/
  crm-read/
    miniapp.json
    handler.go
    service.go
    repository.go
    fixtures/

  notify-send/
    miniapp.json
    handler.go
    job.go
    service.go
    provider.go
    fixtures/

  contact-import/
    miniapp.json
    handler.go
    batch.go
    record.go
    progress.go
    repository.go
    fixtures/

  lead-followup/
    miniapp.json
    handler.go
    workflow.go
    steps.go
    service.go
    effects.go
    fixtures/
```

This is an illustrative organization, not a requirement to create every file in every miniapp.

### The important separation of responsibilities

**Business logic**

Defines what the business operation means, what rules apply, and what constitutes a valid outcome.

**Execution logic**

Determines whether the operation runs synchronously, as a job, as a bounded batch, or as a durable workflow.

**Integration logic**

Communicates with SaaS and custom REST providers through declared capabilities.

**Persistence logic**

Owns application data, progress records, and relevant state under the correct transactional boundaries.

**Framework logic**

Owns common identity, policy, authorization, audit, invocation, effects, approvals, and execution adapters.

The framework should not import reference miniapps. Miniapps should depend on the framework's public APIs, not on its internal packages or on one another.

### Reuse the business operation, not the executor

Suppose a business requires sending a notification.

The notification rules should not be duplicated just because one caller invokes the operation synchronously and another invokes it through River.

Likewise, a DBOS workflow should orchestrate reusable application operations rather than embedding provider-specific HTTP and database implementation details in its control flow.

Keep the execution adapter thin and the business operation explicit.

---

## 12. Common classification mistakes

| Mistake                                       | Why it is wrong                                                                     | Correct approach                                                     |
| --------------------------------------------- | ----------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| Every asynchronous operation becomes D        | Asynchronous execution alone does not imply durable orchestration                   | Evaluate A/B/C/D requirements                                        |
| Every scheduled operation becomes D           | A simple scheduled notification can be B                                            | Use River where one-shot retryable execution suffices                |
| Every multi-step function becomes D           | Multiple function calls do not equal a durable workflow                             | Distinguish bounded processing from durable business stages          |
| Every batch becomes River                     | Global limits, partitioning, and recovery may change the complexity comparison      | Pilot River versus DBOS for the actual Class C workload              |
| Every write becomes D                         | A short single-request command can remain A                                         | Apply authorization, effects, and serialization independently        |
| Every approval requires D                     | A single-step approval can use a durable approval record and inline/River execution | Use DBOS when the approval belongs to a larger durable process       |
| Every miniapp gets its own worker             | This creates unnecessary infrastructure                                             | Share the runtime where execution and release semantics permit       |
| One workflow ID protects an entity            | Workflow identity is not the same as entity serialization                           | Declare and enforce `entity_key`                                     |
| DBOS guarantees exactly-once external effects | Provider outcomes may be unknown across failure boundaries                          | Use the effect ledger and reconciliation                             |
| Class D versions can be casually replaced     | Existing workflows may still require old implementation behavior                    | Preserve per-family AppName/version compatibility and drain strategy |

---

## 13. A practical classification checklist for developers

Before creating a miniapp, answer these questions.

**Business outcome**

- What specific business outcome does this tool promise?
- Is it a business operation or merely a wrapper around one provider endpoint?
- What should happen if the caller retries the operation?
- What is the stable identity of the operation?

**Execution**

- Does it return immediately or continue independently?
- Can the entire operation safely be retried?
- Does it need per-record progress?
- Does it have bounded fan-out or concurrency?
- Must it wait for people or external events?
- Must it resume from a previous checkpoint?

**Correctness**

- Are there consequential external effects?
- How are unknown outcomes reconciled?
- Can two agents modify the same business entity concurrently?
- Is an `entity_key` required?
- Is approval necessary?
- What are the blast-radius limits?

**Infrastructure**

- Which mechanism owns execution?
- If Class C, has River been compared with DBOS for this workload?
- If Class D, is the workload-specific ADR-001 decision approved?
- What is the AppName and workflow-version strategy?
- Who owns recovery and operational intervention?

**Evidence**

- Are the important failure cases tested?
- Does a retry duplicate a confirmed effect?
- Can a worker stop and recover safely?
- Is partial progress visible?
- Are the declared guarantees demonstrated rather than merely documented?

If the developer cannot answer these questions, the classification is not ready to be treated as an architectural decision.

---

## 14. Final rules to remember

The framework's categories exist to prevent developers from solving every problem with the same mechanism.

Remember these principles:

1. **Class A is the default.** Prefer ordinary Go handlers and PostgreSQL transactions when a request-bound operation is sufficient.
2. **Class B is for independent work.** Use River by default for one-shot retryable jobs.
3. **Class C requires comparison.** A bounded batch may use River or DBOS. Measure the actual requirements before choosing.
4. **Class D earns durability.** Use DBOS when the business process genuinely requires durable multi-stage execution, waits, recovery, and orchestration.
5. **Business outcome comes before technology.** Organize miniapps around meaningful business verbs and outcomes, not individual API endpoints.
6. **Approval, entity serialization, idempotency, and Saga are cross-cutting patterns.** Do not invent additional execution classes for them.
7. **External writes always need honest effect handling.** A simple handler is not exempt from unknown-outcome recovery.
8. **Execution category and compute placement are independent.** A trusted Class D workflow can run on a Fly Machine; an agent caller does not automatically require a Sprite.
9. **DBOS is not a reward for complicated code.** It is justified only when its durability and orchestration reduce the total complexity of delivering the business promise.
10. **Evidence decides.** Every meaningful execution guarantee should have a corresponding test, failure case, and operational owner.

The ultimate objective is straightforward:

**Organize business logic around what the business operation promises, then choose the simplest execution model capable of keeping that promise—even when the process crashes, a provider times out, an agent retries, or a deployment occurs halfway through the work.**

This is intended as a developer-facing guide, not another architecture proposal. It can live at `docs/guides/execution-categories.md` and serve as the reference when developers design new miniapps or review a proposed category.



## Z

# App Category Guidelines — Organizing Business Logic by Execution Class

| Field | Value |
|---|---|
| Scope | Execution class (Dimension A) and placement (Dimension B) selection |
| Basis | plan4.1 frozen baseline; ADR-001 selective-adoption policy; platform contracts (effect ledger, run envelope, approval-gated invocation) |
| Audience | Developers writing miniapps against the framework |
| Version | 1.0 — 2026-11-22 |

---

## 1. Executive Summary

Every tool a miniapp exposes runs somewhere, under some crash semantics, with some recovery story. The framework's app categories exist to make those answers deliberate instead of accidental. This document explains why business logic must be classified, how to choose among the categories, when each category is the wrong answer, and the working rules developers must follow once a category is chosen. It covers the four execution classes (Dimension A) and the placement decision (Dimension B), which are independent selectors and must never be collapsed into one.

The four classes in one sentence each. **Class A** is an ordinary operation that completes within one request and one database transaction. **Class B** is a one-shot, independently retryable background job on River. **Class C** is a bounded multi-step batch with explicit per-record progress, on River by default. **Class D** is a durable business workflow on DBOS that must outlive its initiator, and it earns the workflow engine only by passing the ADR-001 gates. Placement is orthogonal: trusted production code runs on Fly Machines; untrusted or agent-authored code runs in isolated Sprites regardless of its class.

The golden rules developed below are these. Fit the class to the shape of the work, not to ambition. When torn between two classes, choose the simpler one and record what the alternative would have required. Escalation, especially into Class D, is a reviewed and recorded decision, never a quiet refactor. And the hard parts are framework-owned: the effect ledger, idempotency, approvals, reconciliation, and recovery are implemented once by the platform, so individual tools must never reimplement them.

## 2. Purpose, Scope, and Basis

### 2.1 What this document governs

This guideline governs one decision that every developer making a miniapp repeats for every tool they expose: which execution class the tool belongs to, where it is allowed to run, and what that choice obligates them to write and to declare. It defines the selection procedure, the negative selection rules (when a category must not be used), the per-category do-and-do-not lists, and the records that make the choice auditable. Operational runbooks, infrastructure scaling, and product roadmap questions are out of scope here; plan4.1 sections 17 and 19 own those topics.

### 2.2 Sources and precedence

The content is a developer-facing distillation of three governing sources: the plan4.1 frozen baseline (particularly the section 3 reconciliation table, the section 10 decision matrices, and ADR-001 through ADR-017), the ADR-001 selective-adoption policy in docs/DBOS.md (default reject; Gates 1 through 4; the complexity score; the REJECT / DEFER / PILOT / ADOPT decision set), and the platform contracts for the effect ledger, the run envelope, and approval-gated invocation. Where plan4.1 or ADR-001 and this document appear to differ, the plan and the ADR govern. Concrete numbers used in examples, such as a five-second synchronous budget or a 24-hour approval expiry, are illustrative defaults that Phase 0 freezes into the contracts; they are not commitments of this document.

## 3. Why Business Logic Must Be Classified

### 3.1 The problem an unclassified codebase develops

Every tool, whether its author thinks about it or not, answers a set of hard questions. What happens if the process dies between the database write and the provider call? What happens if the work takes an hour, or five weeks? Who retries it, who can see its progress, and what state survives a restart? If each developer answers these questions individually with ad-hoc machinery, a codebase accumulates several incompatible answers at once, and three failure patterns appear reliably.

The first pattern is **workflow-engine default**: everything becomes a durable workflow because durability feels safe. The team then pays the engine tax on every change, including version-compatibility discipline for in-flight workflows, determinism constraints on workflow code, replay debugging, and an executor that must always be on. The second pattern is **shadow machinery**: hand-rolled state tables, pollers, retry flags, and status columns that quietly re-implement a workflow engine, badly and without tests for the crash windows they claim to cover. The third pattern is **silent effect loss or duplication**: a handler commits its transaction and then dies before calling the provider, or calls the provider and dies before committing, and nobody owns that window. Classification exists precisely so these questions have one uniform, tested answer per category instead of one improvised answer per tool.

### 3.2 What the class decision fixes

Choosing a class binds the tool to platform machinery that is implemented once and tested against real crash windows: the effect ledger and its UNKNOWN quarantine, transactional enqueue, per-record progress, durable workflow recovery, and the approval binding. The class is declared per tool in the `miniapp.json` manifest, and admission checks enforce the declaration: a Class D tool without a passing ADR-001 record, or a manifest declaring the wrong target for its trust profile, fails the validate step with typed errors before anything is deployed. In other words, a classification mistake becomes a build failure instead of a production incident. The class also makes cost visible at design time, because each category carries a different operational posture, from an always-on DBOS executor to River workers that may idle.

### 3.3 Two independent dimensions, never one selector

Dimension A decides **how** the tool executes: as a plain handler (Class A), a queued job (Class B), a tracked batch (Class C), or a durable workflow (Class D). Dimension B decides **where** the code runs: on Fly Machines for trusted production code, or in isolated Sprites for untrusted, generated, or agent-authored code, with the hybrid pattern reserved for a durable workflow orchestrating one bounded, isolated task. The dimensions are decided independently, and conflating them forbids valid designs. A trusted Class D workflow runs on a Machine. An agent-authored prototype runs in a Sprite even though it might be a trivial Class A script. A hybrid flow keeps the DBOS workflow authoritative on a Machine while the Sprite executes an isolated step with no credentials and no authoritative state. One rule makes the boundary memorable: a Sprite checkpoint is a filesystem snapshot, never a workflow checkpoint; business recovery always lives in DBOS and the Neon database.

## 4. The Categories and What Each Provides

### 4.1 Dimension A: the four execution classes

The table below is the taxonomy of Dimension A. Read the crash-behavior column carefully: it is the essence of each category, because it says what the platform guarantees when a process dies mid-work. Everything else a class provides follows from that guarantee.

| Class | What it is | Substrate | Crash behavior | Platform provides | You still write |
|---|---|---|---|---|---|
| A | One operation in one request | Go handler + Postgres transaction | All-or-nothing rollback; nothing partial survives | Authorization before dispatch, audit row, effect ledger for external calls, duplicate-submit collapse | The handler, the transaction logic, the idempotency key |
| B | One-shot, independently retryable job | River | Attempt retried with backoff until terminal | Transactional enqueue (`InsertTx`), unique jobs, retries, dead-letter inspection, scheduling | An idempotent job handler |
| C | Bounded multi-step batch over many records | River by default; DBOS only on recorded need | Resume without replaying completed records | Progress record, per-record idempotency keys, bounded concurrency, batch resume contract | Batch logic, per-record identity, conflict policy |
| D | Durable business workflow | DBOS | In-flight workflows recover after restart at the correct step | Durable waits, events, checkpointed steps, recovery, queues | Deterministic workflow body, steps, compensations, stable workflow IDs |

Two readings of the table are worth stating explicitly. First, the platform-provides column is a promise, not a suggestion: when a Class B job is enqueued transactionally, the framework guarantees that the job exists if and only if the motivating write committed, and developers must not create a second path around it. Second, the you-still-write column is small on purpose; a category that required you to rebuild recovery machinery would not be a category, it would be a burden. If you notice yourself writing anything from that column's neighbors, you are in the wrong class.

### 4.2 Dimension B: where the code runs

Placement is decided after, and independently of, the class. The single input is the trust profile of the code itself: who wrote it, whether it was reviewed, and what it is allowed to reach. The table below restates the plan4.1 placement matrix in developer terms.

| Workload | Placement | Notes |
|---|---|---|
| Trusted production miniapp code (any class) | Machine | The shared platform runtime on Fly Machines; process groups per plan4.1 section 17 |
| Agent-authored or generated code | Sprite | Isolated, policy-gated execution; no credentials ever staged |
| Persistent development workspace or preview | Sprite | Per-workspace leases; decommissioned when idle past policy |
| One isolated, bounded task inside a Class D workflow | Hybrid | DBOS orchestrates on a Machine; Neon holds task identity and results |
| Promoted prototype | Machine (as trusted build) | Promotion produces a reviewed, immutable build artifact; a Sprite snapshot is never the artifact |

The rule to internalize is that **placement follows the trust profile of the code, never the identity of the caller**. An agent invoking a trusted Class A read tool still exercises Machine code; the agent's identity is handled by the authorization layer, not by moving execution into a sandbox. Conversely, code an agent wrote is untrusted no matter how harmless it looks, and it runs in a Sprite with a confirmed restrictive network policy, proxied egress, and no staged credentials.

### 4.3 One miniapp, many tools; one tool, one class

The class is declared per tool, not per miniapp. A single miniapp can expose a Class A read tool and a Class D workflow tool side by side, because the manifest carries the class in each tool declaration. What is never valid is mixing classes inside one tool: a handler that silently enqueues a follow-up job and a workflow step inside one request path is three tools pretending to be one, and none of the guarantees hold. The granularity rule applies here as well: one tool maps to one business outcome with a stable idempotency identity, not to one provider endpoint. If an agent must chain three of your tools to achieve one outcome, and a crash between them leaves a mess, those three belong inside one miniapp, usually as a Class D workflow.

## 5. Choosing a Category: The Decision Procedure

### 5.1 The ordered questions

Classification is a sequence, not a vibe. Ask the questions in order and stop at the first match, because the sequence is arranged from simplest to most expensive machinery. If two answers seem true at once, the simpler class wins and you record what the alternative would have required.

1. **Q1. Does the tool complete within one request and one local transaction, with external calls only through ledgered capability calls that return in time?** Then it is Class A.
2. **Q2. Is it exactly one unit of work, independently retryable, with no multi-step progress to track?** Then it is Class B on River.
3. **Q3. Is it a bounded batch over many records where you need progress, per-record resume, and bounded concurrency?** Then it is Class C, on River by default; escalate to DBOS only when checkpointed recovery genuinely matters, and record the comparison.
4. **Q4. Must the work outlive its initiator: waits spanning hours or days, a human approval inside a larger process, a cross-system saga with compensation, restart-safe multi-day state?** Then it is a Class D candidate, and it proceeds only if the ADR-001 gates pass and the decision record is written.

One boundary deserves its own sentence because it is the most common confusion: a tool that pauses for a human approval before executing a single step is still Class A or B with a bounded approval wait, handled by the framework's approval-gated-invocation invariant. It is not a Class D workflow unless the process around the approval has genuine multi-stage, restart-safe structure of its own.

### 5.2 The scoring rubric and the four gates

For Class C and D candidates, ADR-001 requires a written comparison rather than an instinct. The complexity score rates seven dimensions from 0 to 2 each, for a maximum of 14. The gates then decide, because a score alone cannot weigh business consequence: **Gate 1** asks which business promise must never break; **Gate 2** asks whether at least two durable-workflow characteristics are genuinely present; **Gate 3** asks what a crash mid-process actually costs the business; **Gate 4** asks whether total complexity goes down versus the simpler alternative, which requires listing the custom machinery you would otherwise build. Every Class C and D candidate ends with a recorded decision of REJECT, DEFER, PILOT, or ADOPT in its `adr001.md` file. The scoring table below shows how to rate each dimension.

| Dimension | 0 points | 1 point | 2 points |
|---|---|---|---|
| Lifetime | Single request | Minutes to hours inside one process | Days, or unbounded until an external event |
| Stages | One | Two or three fixed steps | Many steps, or dynamic structure |
| Waiting | None | Short bounded waits | Human or timed waits spanning long periods |
| Recovery | Transaction rollback suffices | A retry suffices | Must resume at the exact interrupted step |
| External effects | None, or one ledgered call | A few, each compensable | Many, ordered, saga-worthy |
| Coordination | One entity | A few related rows | Cross-system orchestration |
| Human in the loop | None | One approval | Multiple approvals or escalations |

As illustrative guidance, scores of 0 to 2 land in Class A or B, scores of 3 to 6 indicate Class C territory, and scores of 7 or more make a Class D candidate, but the gates decide, not the number. A low score with a devastating Gate 3 answer can still be a workflow; a high score with a trivial Gate 3 answer usually is not. The reference catalog in plan4.1 applies the rubric exactly this way, and the `adr001.md` record captures both the score and the gate answers so a reviewer can reconstruct the reasoning.

### 5.3 Worked examples

The table below maps realistic business logic, including the reference catalog, to categories. The deciding-signals column names the signals, which is the habit to copy: classify by naming the signal, not by analogy to a previous tool.

| Business logic | Class | Deciding signals |
|---|---|---|
| Read a contact with permission checks | A | One request, one transaction, no external effects beyond a ledgered read |
| Update a record in confirm mode | A + approval wait | Single step; bounded human wait via the approval table, not an engine |
| Process an inbound webhook | B | One-shot, independently retryable; transactional enqueue from the receiver |
| Render and deliver a nightly report | B (scheduled) | One-shot on a schedule; scheduling is not durability |
| Import 50,000 contacts with resume | C | Bounded batch; per-record idempotency; resume without replay |
| Two-CRM pull-merge-push sync | C (PILOT) | Batch with conflict policy; compare River vs DBOS queue on the record |
| Lead follow-up over days with quiet hours | D | Multi-day waits, events, cancel-safe recovery: passes the gates |
| Invoice-to-ledger saga | D | Ordered external effects with compensation; failure consequence is high |
| Employee provisioning with manager approval | D | Human approval inside a multi-system process that must survive restarts |
| Run an agent-written prototype safely | Placement, not class | Untrusted code: Sprite workspace; promotion goes through a trusted build |

### 5.4 Same verb, different category

Category choice often looks arbitrary until two similar verbs are placed side by side. Sending one email is Class B: one shot, one retry policy, one receipt. Running a five-week nurture sequence with quiet hours, rescheduling, and cancellation is Class D: the lifetime and the waiting dimension carry it across the line, because a crash on week three must resume at week three, not re-send weeks one and two. Updating one record is Class A; merging duplicate accounts across two systems is at least Class C and often Class D, because the coordination and recovery dimensions change shape even though the vocabulary did not. Fetching a report is Class A; generating and delivering one on a schedule is Class B, because scheduling is a queue feature, not a workflow feature. The discipline is to name which scoring dimension moved, and to be suspicious when the honest answer is none of them moved but the class still went up.

## 6. When Not to Use Each Category

### 6.1 When not to use Class D (the most important list)

ADR-001 is a default-reject policy, and for good reason: the durable engine is the most expensive machinery in the platform, so the negative list for Class D is the one every developer must know by heart. **Async execution alone never justifies DBOS**; that is Class B. **Scheduled execution alone never justifies it**; River schedules. **Multi-step execution alone never justifies it**; that is Class C with a progress record. An approval-gated single-step tool is Class A or B with a bounded wait. A reconciliation sweeper that compares provider state to local state is a periodic River job, not a workflow. Per-entity serialization alone is a row lock or a partitioned queue, not an engine. And the phrase "we might need durability later" is the signature of a DEFER decision, which is recorded and revisited, never a speculative adoption. Finally, Gate 4 has teeth: if the custom machinery you would build without DBOS amounts to one state row and a retry policy, the net-complexity answer is no, and the workflow is rejected.

### 6.2 When not to use River (Class B)

River is the right answer for one-shot work, and the wrong answer for three neighboring shapes. If the work completes within the synchronous budget, keep it in Class A, because a queued round trip costs the agent extra latency and a failure mode nobody asked for. If the work needs per-record progress and resume, it is Class C, and a chain of jobs pretending to be a batch will replay records after a crash. If the work needs durable waits, it is Class D. One operational note matters when a batch does stay on River: global concurrency limits are a River Pro feature, not an open-source one, so rate control against a provider must come from per-provider limiters, queue-level controls, or the recorded DBOS-queue comparison from the Class C pilot.

### 6.3 When Class C machinery is overkill

The Class C contract asks you to name a stable per-record identity, because per-record idempotency is the whole guarantee. If you cannot name that identity, the work is not a batch: it is either one unit (Class B) or a fan-out of independent one-shots, which is many Class B jobs rather than one Class C batch. A batch of one, or a loop over three rows inside what is really a single request, should stay in Class A. The test is simple: if a crash after record 999 of 1,000 should resume at record 1,000, use Class C; if it should just re-run the whole thing because re-running is harmless and cheap, one of the simpler classes is honest.

### 6.4 When the answer is no miniapp at all

Not every capability deserves a catalog entry, and the granularity rule cuts both ways. A thin wrapper around one provider endpoint is not a miniapp; if an agent genuinely needs raw access to a provider operation, the platform exposes it through a read-only, policy-bounded capability call rather than through a tool that pretends to be a business outcome. At the other end, three tools that an agent must chain to produce one outcome, with a mess left behind if a crash lands between them, are one miniapp, usually Class D. The question to ask is what the agent is trying to accomplish as a business verb: the miniapp boundary follows the verb, and the idempotency identity follows with it.

## 7. Dos and Don'ts for Developers

The rules in this chapter exist because the platform's guarantees depend on them. Violating a do-not rarely fails immediately; it fails during a crash, a restart, or a provider timeout, which is the worst possible moment to discover that a guarantee was never real. Each subsection states the one invariant that defines its category, then gives the working rules as paired do-and-do-not rows. Where a rule names a contract, such as the effect ledger or the approval table, that machinery is framework-owned; calling it correctly is your job, rebuilding it is not.

### 7.1 Class A: the all-or-nothing request

The defining invariant of Class A is that everything either commits or does not: one request, one transaction, no partial effects. An external operation inside the request is allowed only through the effect ledger, which records the intent before the provider call so that an unknown outcome is quarantined and reconciled rather than lost or blindly retried. Keep the handler stateless, declare the idempotency key so duplicate submits collapse, and return the result inline under the sync-first envelope.

| Do | Don't |
|---|---|
| Run the whole operation in one transaction and return the result inline. | Enqueue a job "to be safe"; that is Class B work hiding in an A declaration. |
| Record the effect intent before any provider call; treat provider free text as untrusted data. | Call a provider directly with credentials from configuration. |
| Serialize concurrent writers with a row lock on the declared `entity_key`. | Touch dozens of entities in a loop inside one request; that is Class C work. |
| Declare the idempotency key so duplicate submits collapse into one run. | Sleep, poll, or wait inside the request path. |
| Keep the handler stateless and read policy through the framework. | Build a multi-step state machine in session or global variables. |

### 7.2 Class B: the transactional one-shot

The defining invariant of Class B is that the enqueue commits in the same transaction as the motivating write, using River's transactional enqueue, so the job exists if and only if the write committed. The handler must be idempotent, because at-least-once delivery means an attempt can run more than once; the unique job key gives you deduplication on the stable business identity.

| Do | Don't |
|---|---|
| Enqueue with `InsertTx` inside the domain transaction. | Enqueue after the commit; that window loses jobs. |
| Use the business identity as the unique key, for example `tenant:order:{id}:notify`. | Chain jobs into a hidden workflow; steps that depend on each other are Class C or D. |
| Give the job an explicit retry policy and a terminal-failure path; review dead-letter regularly. | Put large payloads in job arguments; pass IDs and read state from the database. |
| Declare `entity_key` when the job writes, so concurrent writers serialize. | Loop over thousands of records inside one job; that is Class C work. |
| Keep the handler idempotent; assume an attempt can repeat. | Assume open-source River enforces global concurrency limits; it does not, plan for a limiter. |

### 7.3 Class C: resume without replay

The defining invariant of Class C is that a crashed batch resumes at the first unfinished record, never replaying completed ones. The framework owns the progress record and the per-record idempotency keys; you own the batch logic, the per-record identity, and the conflict policy. Bounded concurrency protects the provider, and typed per-record outcomes let one bad row fail without aborting the batch.

| Do | Don't |
|---|---|
| Use the framework progress record and per-record idempotency keys, for example `tenant:batch:{id}:row:{n}`. | Restart the whole batch from scratch on failure. |
| Cap worker concurrency and set typed outcomes per record. | Build timers, events, or recovery machinery; that is the Class D line, and crossing it needs a record. |
| Write the ADR-001 comparison when checkpointed recovery plausibly matters. | Keep progress checkpoints in files; progress lives in the database. |
| Declare `entity_key` so concurrent batches serialize on the same entity. | Assume records are processed in a guaranteed global order. |
| Test kill-and-resume as part of the batch suite. | Treat "bounded" as optional; unbounded batches become shadow workflows. |

### 7.4 Class D: deterministic body, authoritative database

Class D has two invariants that everything else follows from. First, the workflow body is deterministic: the same replayed history must produce the same decisions, so all nondeterminism lives in steps and transactions. Second, the database, not the engine, is the authority for business state; checkpoints carry IDs and compact context so that recovery reads the truth from Neon, not from the workflow's own memory. The workflow earns its complexity through the gates, and it keeps that trust through determinism discipline and a recorded version strategy.

| Do | Don't |
|---|---|
| Put every nondeterministic call in a step; keep the workflow body deterministic. | Start goroutines, select on channels, rely on map iteration order, or call `time.Now` or `rand` in the workflow body. |
| Use the stable workflow ID form `tenant:entity:kind:op`. | Call `RunAsTransaction`, `Enqueue`, or `Send`/`Recv` inside a step body. |
| Carry IDs and compact context in payloads; register compensation before the action, idempotent, unwound LIFO. | Put credentials or personal data in checkpoints or events. |
| Wait with durable sleeps and events; serialize entities through a partitioned queue keyed by entity. | Treat a Sprite checkpoint as workflow recovery; files are not business state. |
| Record the version strategy: a pinned application version with patching, or the per-family worker topology. | Auto-compensate on UNKNOWN outcomes without a safety basis, or hot-fix by replaying an old version. |

### 7.5 Placement: trust decides where code runs

Placement rules are short because the principle is short: the trust profile of the code selects the target. What deserves emphasis is the negative side, because the two most dangerous mistakes in this dimension are moving execution into a sandbox because of who called the code, and letting workspace code reach production without the governed promotion path.

| Do | Don't |
|---|---|
| Run trusted production miniapps on Machines. | Choose a Sprite because the caller is an agent; trust of the code decides. |
| Run agent-authored or generated code, previews, and workspaces in Sprites. | Stage credentials into a Sprite; provider calls go through the capability proxy. |
| Use the hybrid pattern for one bounded, isolated task inside a Class D workflow. | Promote a Sprite snapshot to production; only a reviewed, trusted-build artifact promotes. |
| Confirm the restrictive network policy and read back its digest before any untrusted code runs. | Deploy agent-authored code to Machines, even "temporarily". |

## 8. Cross-Cutting Invariants

A handful of contracts apply to every category, which is exactly why they are framework-owned and must never be reimplemented inside a miniapp. The table below lists them with the practical obligation each one places on your code. When one of these invariants seems to block a design, treat that as a signal: either the design is wrong, or the exception must be recorded and reviewed like any other architecture change.

| Invariant | What it means | Your obligation |
|---|---|---|
| Effect protocol | External effects move `PENDING -> ATTEMPTING -> CONFIRMED`, `FAILED`, or `UNKNOWN`, with quarantine and reconciliation toward a confirmed or not-applied outcome | Record intent before the call; never blindly retry an UNKNOWN outcome |
| Idempotency identity | Every tool declares a stable business idempotency key so duplicate submits collapse | Choose the key from business identity, not from timestamps or random values |
| Entity serialization | Write tools declare `entity_key`, enforced by row locks in A/B/C and partitioned queues in D | Declare `entity_key`, or record an explicit `no_entity` justification |
| Approval-gated invocation | One approval mechanism: a table with `PENDING -> APPROVED / REJECTED / EXPIRED`, material-change invalidation, an out-of-band approver, and separation of duties | Use the framework approval wait; bounded for A/B, a DBOS event wait inside D |
| Sync-first envelope | Results return inline within the synchronous budget; otherwise a run handle with `wait_ms` long polling | Design tools to finish in one request when the work honestly allows it |
| Blast-radius limits | Per-principal, per-tenant, per-capability rate and volume caps, a circuit breaker, and a kill switch that takes effect without redeploy | Declare limits in the manifest; never assume policy alone prevents a retry loop |
| Untrusted results | Provider free text is data, not instructions: typed fields, untrusted marking, size caps, cursors | Never concatenate provider content into instructions or error messages |
| Audit and fail-closed | Every transition is audited; revocation fails closed | Emit nothing outside the framework's audit path; never widen access on error |

## 9. Recording and Enforcing the Decision

### 9.1 What the manifest carries

The manifest is the artifact of record, and the class decision lives in it as concrete declarations rather than as prose. For every tool, the manifest states the execution class, the idempotency key template, the entity key or its explicit absence justification, the trust modes the tool supports, the placement profile with its policy references, and the blast-radius limits. Because referenced deployment profiles are content-addressed at publish time, a change to a profile forces a new manifest version and fresh validation, which keeps the class decision and the running code from drifting apart.

### 9.2 The adr001.md record for Class C and D

Every Class C and D candidate carries a short decision record with a fixed shape: the business promise that must never break, the seven-dimension score, the answers to the four gates, the list of custom machinery the no-engine alternative would need, the checklist items linked to the tests that prove them, and the final decision of REJECT, DEFER, PILOT, or ADOPT with an owner and a date. For Class C the record additionally captures the River-versus-DBOS comparison when promotion is plausible. The record is small on purpose; its value is that the reasoning is reconstructible months later, and that DEFER and REJECT are visible, reversible outcomes rather than silent omissions.

### 9.3 How the platform enforces the choice

Enforcement is mechanical, which is what makes the guidelines trustworthy. The executor admission check rejects Class D declarations that lack a passing record. The placement admission check rejects manifests whose trust profile does not match their declared target. CI runs contract drift tests that keep the manifest, the generated schemas, and the documentation aligned, and the import checker keeps miniapps on the framework's public API. A classification mistake therefore surfaces at the validate step with a typed error, at the cheapest possible moment.

### 9.4 Changing class later

Reclassification is a normal event with a formal path. Moving from C to D requires the decision record and review, because the tool is acquiring the engine tax permanently. Moving from D back down is rarely valid once durable state exists, and it demands a migration plan under the documented workflow compatibility strategy rather than a rewrite. Any class or placement change produces a new manifest version with fresh validation and review, so the audit trail shows who decided what, and when.

## 10. Quick Reference

### 10.1 The decision tree in six lines

1. One request, one transaction? **Class A.** External effects go through the ledger; results return inline.
2. One independent, retryable unit? **Class B.** Enqueue transactionally; write an idempotent handler.
3. A bounded batch with per-record identity? **Class C** on River, with progress and resume-without-replay.
4. Must outlive its initiator, with real waits, approvals inside a process, or a saga? **Class D candidate:** pass the gates, write the record.
5. Untrusted or agent-authored code? **Placement changes, not the class:** Sprite, policy-confirmed, proxy-only egress.
6. Torn between two classes? Take the **simpler one** and record what the alternative would have required.

### 10.2 Definition of done per category

The checklist below condenses the chapter 7 rules into the shortest possible completion test for a tool. Treat a row that cannot be checked as a finding, not a formality.

| Class | Done means |
|---|---|
| A | One transaction; ledgered external calls; idempotency key declared; entity key declared or justified; inline result within the sync budget |
| B | Transactional enqueue; business-identity unique key; idempotent handler; explicit retry and terminal policy; entity key when it writes |
| C | Framework progress record; per-record idempotency keys; bounded concurrency; kill-and-resume test green; decision record written |
| D | Gates passed and record written; deterministic body lint-clean; steps own all effects; compensations registered before actions; version strategy recorded; recovery drill green |
| Placement | Target matches trust profile; policy read-back confirmed for Sprites; promotion, if any, through a trusted-build artifact with review |

### 10.3 Glossary

The glossary below defines the terms used throughout this document, in the sense the framework gives them. When a term is used in discussion, it means exactly this and nothing narrower or broader.

| Term | Definition |
|---|---|
| Execution class (Dimension A) | How a tool runs: A handler, B job, C batch, D durable workflow; declared per tool in the manifest |
| Placement (Dimension B) | Where the code runs: Machine for trusted code, Sprite for untrusted code, hybrid for isolated tasks inside D |
| Effect ledger | The framework record of external effects and their outcomes, including the UNKNOWN quarantine and reconciliation path |
| Idempotency identity | The stable business key under which duplicate submits collapse into one run |
| Entity key / keyed serialization | The declared entity a write tool acts on, enforced by row locks or partitioned queues so writers serialize |
| Trust mode | The per-version, per-tenant authorization posture of a tool: dry-run, confirm, or autonomous |
| Approval-gated invocation | The single framework mechanism that pauses a tool for a human decision, with expiry, material-change invalidation, and an out-of-band approver |
| Durable workflow | A Class D process that survives initiator and process death, recovering at the exact step from DBOS checkpoints |
| Compensation | A registered, idempotent undo operation, registered before the action it compensates and unwound in LIFO order |
| Trusted-build promotion | The governed path from a Sprite workspace to production: reviewed source, immutable artifact, digests, and a review decision |

A final note on posture. The categories are not bureaucracy; they are the compressed record of every crash window the platform has already thought about for you. When the procedure in this document and your instinct disagree, slow down and write the record: either the record shows your instinct was right and the class changes through review, or it shows where the design would have lost effects, replayed records, or stranded a workflow. Both outcomes are cheaper than the production incident the record prevents.

