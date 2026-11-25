Boundary A deserves an independent adversarial review before we ratify C1–C4 and authorize C6. I recommend a **focused implementation and invariant review**, not another architecture review.

The most important areas are sender recovery, delivery idempotency, turn/question binding, tenant isolation, SSE replay, and the changed yield-on-assigned contract. The report also reveals an important evidence limitation: the browser test exercised signed callbacks and the UI answer flow, but the sender rows remained queued because no Gateway URL was provisioned. The reviewer must not mistake that for a complete live Gateway execution.&#x20;

prompt18-boundary-a.md



Here is the prompt.

# Adversarial Code Review — Miniapp Studio Boundary A (Phase C Backend C1–C4)

**Review mode: Independent, read-only, adversarial.**

## 1. Objective

Perform an independent adversarial code review of the Phase C backend implementation delivered under Prompt18 Boundary A.

Review the actual working-tree implementation, not merely the completion report.

The objective is to determine whether C1–C4 establish their claimed correctness, durability, security, concurrency, and recovery invariants.

The review must identify structural or foundational defects that could invalidate the implementation or make the local-first demo unreliable.

Do not reopen accepted architecture without concrete evidence of an invariant violation.

Do not implement fixes during this review.

## 2. Required source material

Read:

- `docs/prompt18-boundary-a.md`
- `plans/prompt18_plan.md`
- `docs/hackathon-demo-critical-path.md`
- `plans/agent-gateway-implementation-plan.md`
- `plans/miniapp-studio-implementation-agent-guidelines.md`

Inspect the actual code and tests associated with Boundary A.

### New files

- `server/sender.go`
- `server/sender_test.go`
- `server/sse_resume_test.go`
- `gateway/sender_contract_test.go`
- `tools/dispatch_contract_test.go`
- `tools/binding_test.go`
- `store/migrations/000017_sender_tracking.up.sql`
- `store/migrations/000017_sender_tracking.down.sql`
- `store/migrations/000018_sender_schedule.up.sql`
- `store/migrations/000018_sender_schedule.down.sql`

### Modified implementation

- `gateway/client.go`
- `gateway/server.go`
- `tools/build.go`
- `tools/question.go`
- `tools/ingest.go`
- `tools/conversation.go`
- `tools/register.go`
- `server/server.go`
- `server/config.go`
- `cmd/studio/main.go`
- `server/gateway_callback.go`

Also inspect the relevant existing store, transaction, SSE, authorization, RLS, and migration code.

Use `git diff` and `git status` to distinguish Boundary A changes from earlier unrelated working-tree changes. Do not assume every current modification belongs to this implementation.

## 3. Review discipline

Treat every statement in `docs/prompt18-boundary-a.md` as a claim to verify, not as proof.

For every significant finding:

- Identify the violated or potentially violated invariant.
- Trace the complete execution path.
- Identify the component that owns the invariant.
- Cite exact files, functions, and line ranges.
- Explain a concrete failure scenario.
- Explain tenant, data-integrity, execution, or recovery impact.
- Identify the missing or inadequate regression test.
- Distinguish demonstrated failures from theoretical concerns.

Do not report generic best-practice preferences as defects.

Do not recommend redesign merely because another implementation is possible.

Prioritize defects that can cause lost work, duplicate execution, cross-tenant access, incorrect authorization, corrupted conversation state, unrecoverable builds, or fabricated success.

## 4. C1 — Durable sender loop

Review `server/sender.go` and its interactions with `gateway.Client`, `gateway.Server`, `studio.commands`, and `studio.builds`.

Adversarially investigate:

### Dispatch and idempotency

- Is the persisted command row ID always the stable Gateway message ID?
- Does `SendWithID` preserve identity across retries, process restarts, and multiple sender instances?
- Can an HTTP timeout or lost response cause duplicate execution?
- Can two workers claim or dispatch the same row concurrently?
- Are claim, dispatch, and settlement transitions atomic where required?
- Can an accepted command be incorrectly marked failed?
- Can a failed settlement leave a permanently stuck claim?

### Retry and attempt accounting

- Does the 25-attempt budget actually bound every retry path?
- Can an attempt be burned without a real dispatch?
- Do prerequisite waits preserve attempts?
- Are backoff and `next_attempt_at` calculations safe against clock changes and process restarts?
- Can retry exhaustion strand a build without a visible terminal outcome?
- Are non-retryable refusals correctly distinguished from transient failures?

### Indeterminate outcomes

- Is the reconcile-once behavior genuinely bounded?
- What happens if the Gateway accepted a command but its response was lost?
- Can a delayed original command execute after a fresh replacement has been issued?
- Can the same logical action execute twice across a crash/restart boundary?
- Is a second ambiguous outcome visible and terminal rather than retried indefinitely?

### Recovery and lifecycle

- Does the crash sweep adopt or recover rows correctly?
- Can shutdown interrupt a dispatch between Gateway acceptance and database settlement?
- Can sender startup race with database migrations, callbacks, or build creation?
- Is `STUDIO_SENDER` configuration fail-safe?
- Are multiple Studio instances safe, or is singleton behavior assumed without enforcement?
- Are tenant polling queries constrained by the authenticated tenant/RLS context?

Verify the claimed invariants:

- Stable row ID.
- 2xx means launched, not completed.
- Gateway turn is authoritative.
- Recovery once per lineage.
- Waits do not burn attempts.
- 25-attempt budget.
- Per-tenant RLS polling.

Do not accept these claims without tracing their implementation.

## 5. C2/C3 — Question binding, turn freezing, and atomic handoff

Inspect `tools/question.go`, `tools/ingest.go`, `tools/build.go`, `tools/conversation.go`, `tools/register.go`, and related database transactions.

Investigate:

- Is every answer bound to the exact tenant, build, turn, and question ID?
- Can an answer for an earlier question be applied to a newer turn?
- Can two simultaneous answers both be accepted?
- Can a question be superseded between validation and handoff?
- Is turn freezing performed atomically with the answer and command creation?
- Can an answer be recorded without its continuation command?
- Can a continuation command exist without a durable answer record?
- Can duplicate answer submission create two continuations?
- Are cross-tenant references rejected at the owning boundary?
- Does Gateway-reported turn state remain authoritative?
- Can a stale or out-of-band Gateway command invalidate Studio's turn assumptions?
- Are error classifications stable and actionable?

### Specific contract change requiring adjudication

`TestIngestYieldRequiresRunning` previously asserted that yield-on-assigned was absorbed.

The implementation changes this behavior:

`assigned → running → waiting_user`

Terminal events also join the assigned build rather than being absorbed.

The report argues the old behavior could leave a build permanently assigned when `agent.started` was lost.

Review this change independently.

Determine:

1. Is parking the correct recovery behavior?
2. Can an out-of-order or duplicate yield falsely imply that an agent started?
3. Can a terminal event arrive before the corresponding started event?
4. Does the new transition preserve accurate build and agent state?
5. Are late, duplicate, stale, and contradictory callbacks handled safely?
6. Does this change require any corresponding contract or documentation update?

Give a direct recommendation: RATIFY, RATIFY WITH A BOUNDED CONDITION, or REJECT.

Do not silently preserve the old contract merely because a historical test asserted it.

## 6. Gateway protocol and refusal taxonomy

Inspect `gateway/client.go` and `gateway/server.go`.

Review:

- Stable command-ID semantics.
- `SendWithID` behavior across independent clients.
- Status and turn reporting.
- HTTP response interpretation.
- `CallError{Status, Body}` exposure and logging safety.
- The six-code refusal/indeterminate taxonomy.
- Whether every code is mapped to the correct retry, reconcile, reject, or terminal behavior.
- Whether malformed, missing, or unknown error responses fail safely.
- Authentication and request-signing boundaries.
- Whether callback and command identities can be confused across builds or tenants.

Verify that a 2xx response is never treated as agent completion.

Check that a lost or delayed response cannot bypass fencing or cause duplicate logical work.

## 7. Callback handling and event ordering

Inspect `server/gateway_callback.go` and `tools/ingest.go`.

Adversarially test the assumptions around:

- Signed callback verification.
- Tenant/build/workspace binding.
- Duplicate callback delivery.
- Duplicate question IDs.
- Out-of-order started, question, completed, and failed events.
- Callback replay after process restart.
- A terminal callback racing with an answer.
- A callback arriving after a build has moved to a new turn.
- Invalid callback payloads and unknown event types.

**Specifically challenge the change that maps duplicate question IDs to HTTP 400.**

Determine whether 400 is the correct protocol response for a duplicate delivery under the actual Gateway outbox retry semantics.

A duplicate event must not produce retry poison, but a response must not silently conceal a conflicting question or corrupted binding either.

Verify the distinction between:

- Exact duplicate, safe to acknowledge.
- Same identifier with conflicting content, must fail visibly.
- Stale event, must be rejected or reconciled according to contract.

## 8. C4 — Conversation history and SSE replay

Inspect the stream persistence, polling, cursor, and HTTP event-stream implementation.

Review:

- Durable sequence allocation.
- Cursor ordering and uniqueness.
- Replay after browser reconnect.
- Replay after Studio process restart.
- Backlog drain.
- Concurrent writers and concurrent readers.
- No-gap/no-duplicate claims.
- Commit-before-delivery ordering.
- Events committed during the query/subscribe boundary.
- Multiple tabs subscribing to the same conversation.
- Cross-tenant stream access.
- Slow or disconnected clients.
- Polling cancellation and resource cleanup.
- Duplicate suppression and ordering guarantees.

Investigate the reported design cost:

One tool invocation per polling tick per SSE stream is currently a known residual.

Determine whether this is merely a bounded local-demo limitation or can cause correctness, resource, or isolation problems at the current scope.

Do not turn this review into a general SSE scaling project.

## 9. Database and migration review

Review migrations `000017` and `000018`, their down migrations, and the related SQL/query code.

Inspect:

- `builds.model`.
- `builds.gateway_turn`.
- `commands.next_attempt_at`.
- Due-row indexes.
- Defaults and backfill semantics.
- NULL and zero-value handling.
- Foreign-key behavior.
- Transaction isolation.
- Lock ordering.
- Rollback correctness.
- Migration idempotency.
- RLS ENABLE and FORCE requirements.
- Tenant context reset across pooled connections.
- Cross-tenant reference prevention.

### Explicitly investigate the reported deadlock fix

The implementation changed advisory-lock ordering in `appendStream`.

The report identifies a pre-existing deadlock between conversation-row holders and advisory-lock holders, with a correction described as:

`conversation KeyShare → advisory → fresh inserts`

Trace every caller of `appendStream` and every relevant transaction path.

Determine whether all callers follow a compatible global lock order or whether another code path can still create the inverse order.

The two green reruns and race tests are useful evidence, but they do not replace a lock-graph inspection.

## 10. Tenant isolation and authorization

Treat tenant isolation as a foundational invariant.

Review all new and modified paths for:

- Explicit tenant derivation.
- No caller-controlled tenant substitution.
- RLS context set and reset.
- RLS FORCE remaining effective.
- Sender polling under the correct tenant.
- Build, conversation, question, answer, command, and event ownership checks.
- Cross-tenant object references.
- Authorization on cancellation and continuation.
- No service-level bypass of human authorization.
- No secrets in errors, callbacks, event payloads, or transcripts.

Investigate the reported workspace correction:

The sender now uses `workspace_id = tenant`, because callbacks look up tenants rather than operator bindings.

Verify this consistently throughout the full assignment → Gateway → callback path.

## 11. Test and evidence audit

Independently inspect the 39 new tests and their relationship to the claimed invariants.

Do not rely solely on test counts.

Look for:

- Tests that assert implementation details rather than behavior.
- Missing red-before evidence.
- Weak mocks that bypass transaction or protocol semantics.
- Tests that cannot reproduce the claimed crash window.
- Missing concurrent duplicate delivery.
- Missing conflicting duplicate callback.
- Missing delayed original command after indeterminate replacement.
- Missing cross-tenant sender polling.
- Missing out-of-order event sequences.
- Migration rollback paths not exercised.
- Race tests that omit the relevant code path.

Review the reported full serial suite and race evidence.

Important known test-environment hazard:

Test boots rotate the cluster-global `studio_app` password and have disrupted a concurrently running local server.

Do not run the full test suite while another local Studio server depends on that credential. Do not modify cluster configuration, user data, or databases outside the isolated test scope.

If additional tests are needed, propose exact commands and isolation requirements. Do not execute unsafe tests merely to complete the review.

## 12. Evidence honesty audit

The completion report claims a real Chrome 144 pass against a local live server.

However, it also records:

- Gateway URL absent on the build.
- Assign and continue rows remained queued.
- Sender did not dispatch to a Gateway.
- Signed Python callbacks were posted directly.
- The callback, question streaming, and UI answer path were exercised.
- The real Gateway interop tests used `gateway.Server` with `FakeAdapter`.
- C5 live proof was not run.

Assess these claims precisely.

Distinguish:

1. Unit-tested sender behavior.
2. Integration-tested real Gateway HTTP contract with FakeAdapter.
3. Browser-tested Studio persistence, callbacks, SSE, and answer flow.
4. Actual sender-to-Gateway dispatch.
5. Actual OpenCode inference.
6. Actual same-session agent continuation.

Do not promote one category into another.

The browser pass is valuable evidence, but it must not be described as a full sender/Gateway/agent end-to-end proof when the sender remained queued.

## 13. Scope restrictions

This is a read-only review.

Do not:

- Modify source, tests, migrations, plans, or reports.
- Stage, commit, or push.
- Invoke the Sprite CLI, including read-only commands.
- Invoke Fly commands or APIs.
- Provision Sprites.
- Execute live OpenCode.
- Run OAuth or handle credentials.
- Perform C5 live verification.
- Implement C6 Console.
- Implement Phase D or E.
- Change Miniapp, F1, G-01, P5, or accepted architecture.

Do not reopen the broader accepted Gateway architecture absent a concrete invariant violation.

## 14. Required review output

Produce a focused adversarial report with:

### A. Executive verdict

Choose exactly one:

- **ACCEPTED** — no material defect remains within C1–C4.
- **ACCEPTED WITH BOUNDED RESIDUALS** — no structural blocker; identified residuals are contained, explicit, and do not invalidate the local Phase C backend.
- **NOT ACCEPTED** — at least one foundational invariant is violated or a material correctness/security defect remains.

State whether C1–C4 can be ratified and whether the implementation may proceed to C6.

### B. Findings

For every finding include:

- Severity: Critical / High / Medium / Low.
- Exact source location.
- Violated invariant.
- Concrete failure sequence.
- Evidence and reproducibility.
- Required correction or bounded containment.
- Whether it blocks C1–C4 ratification or only a later live phase.

Do not inflate severity.

### C. Contract-change adjudication

Give an explicit decision on yield-on-assigned behavior and justify it against event ordering, recovery, and state truthfulness.

### D. Evidence verdict

State exactly what the current tests and browser pass prove, and what remains fake-backed or live-unverified.

### E. Residual register

Separate:

- Structural blockers.
- Bounded local-demo residuals.
- C5/live-only requirements.
- C6 dependencies.
- Deferred scaling work.

### F. Acceptance recommendation

If no structural or foundational blocker remains, say so directly and recommend acceptance without requiring another broad review.

Do not demand C5 live proof to accept the local C1–C4 implementation. It remains a separate live gate.

## 15. Review philosophy

Be adversarial toward the implementation, not performatively adversarial toward the project.

Find root-cause defects, not cosmetic preferences.

The standard is whether the implementation preserves durable, tenant-safe, correctly ordered, idempotent work under retries, crashes, concurrency, and recovery.

If the invariant is demonstrated and no structural defect remains, approve it.

Do not extend this review indefinitely. Once the concrete invariants have been examined, deliver the verdict and stop.

### **My recommendation**

Have the reviewer focus especially on three high-risk questions:

1. Can the sender ever execute the same logical continuation twice after an ambiguous Gateway response?
2. Is the new yield-on-assigned transition truthful under out-of-order callbacks?
3. Does duplicate question handling correctly distinguish safe duplicate delivery from conflicting data?

These are the areas where a green test suite can still conceal a serious state-machine defect.

The review should end with a direct ratification decision on Boundary A and a recommendation on proceeding to **C6 — Focus Console**, not another round of general architecture planning.