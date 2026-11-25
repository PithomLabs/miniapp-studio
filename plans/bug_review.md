The underlying problem is that **our verification process is not sufficiently isolated, reproducible, or resilient to partial failure.** The browser matrix is exposing that weakness, but repeatedly fixing the latest failing assertion will not solve it.

My approach would be to fix the causes at the layers that own them, then make Boundary B acceptance reproducible without repeatedly rebuilding the entire environment.

I would divide the work into two tracks: finish Boundary B and eliminate the underlying environment defects that make future development fragile.

## **1. What is actually broken?**

The writeup identifies four causes. I agree with its diagnosis.

**Cause 1 — Shared PostgreSQL credentials**

Structural defect

Test suites rotate the cluster-global `studio_app` password while the live Studio server is using that same role. This makes concurrent testing and live verification unsafe by design.

**Cause 2 — Evidence contamination**

Fix implemented

SSE polling generates audit records that displace meaningful receipts from the Evidence scene. The correct fix is to filter at the database query layer before applying `LIMIT`, not hide records in the UI.

**Cause 3 — Missing provisioning wake-up**

Deferred improvement

The sender discovers provisioning completion through scheduled polling. That is eventually correct, but creates unnecessary latency and timing sensitivity in acceptance tests.

**Cause 4 — Monolithic browser matrix**

Immediate test-design defect

One script controls the entire acceptance journey. A small browser-script mistake forces a complete reset, destroying useful test state and making each failure expensive to diagnose.

writeup6(2).md

## **2. My proposed architectural corrections**

### **A. Eliminate shared credential mutation**

This is the most important infrastructure correction.

I would not settle for a rule saying developers must avoid running tests while Studio is running. That is an operational workaround for an architectural defect.

The real correction is to make credential ownership explicit.

**Separate three operations:**

| Operation                      | Responsibility                                                  |
| ------------------------------ | --------------------------------------------------------------- |
| Create application role        | Create role if absent; never rotate an existing password        |
| Grant application privileges   | Apply required grants without changing authentication           |
| Rotate application credentials | Explicit administrative operation, invoked by provisioning only |

Then introduce a dedicated `studio_app_test` role for test harnesses.

The test role must have access only to test databases. Test setup must never rotate or otherwise mutate the live `studio_app` credential.

I would also inspect every caller of `EnsureAppRole`, including all four test harnesses and the provisioning CLI, rather than changing just the function and hoping the behavior is corrected.

**Required regression proof:**

1. Start a live Studio server using `studio_app`.
2. Record its database role configuration.
3. Run the complete Go test suite concurrently.
4. Verify the live server continues serving authenticated requests.
5. Verify the `studio_app` password has not changed.
6. Verify test operations cannot access or mutate application data outside their isolated databases.

This proves the invariant rather than merely testing that a function returned successfully.

writeup6(2).md

### **B. Keep the audit system authoritative; correct the Evidence read model**

I would retain the implemented `activity.list exclude` design.

The distinction is important: the Miniapp runtime owns which tool invocations are audited. Studio owns how it classifies and presents those records.

We should not fork or modify the runtime's audit behavior just to make one Studio screen look cleaner.

The structural invariant is:

> Transport activity must not displace business evidence in a bounded Evidence query.

The exclusion must happen in SQL before `LIMIT`, with validation and a bounded exclusion list.

I would add an acceptance test that generates substantial `conversation.stream` polling activity while preserving access to the meaningful receipts.

The interface must disclose that transport-poll records are excluded from the Evidence view; it must not suggest that the underlying audit records were deleted.

This correction is already reported as implemented, so I would verify it rather than rewrite it.

writeup6(2).md

### **C. Make provisioning completion event-driven only when justified**

I would not immediately introduce `LISTEN/NOTIFY`.

The current polling model has an important property: it converges even if the sender misses an opportunity to wake up.

The problem is latency, not correctness.

For the current Boundary B milestone, I would retain polling and make the browser test respect the actual 30-second scheduling behavior. The documented SQL wake can remain explicitly labeled test scaffolding, not product behavior.

For a subsequent improvement, I would prefer:

1. A durable provisioning-completed record or outbox event written when the Gateway URL becomes available.
2. A notification used to wake the affected sender work immediately.
3. The sender re-reads authoritative database state before dispatch.
4. Periodic polling/reconciliation remains as the recovery mechanism if a notification is missed or a process restarts.

This is important: `NOTIFY` must be a wake-up optimization, not the source of truth.

I would implement that only when provisioning latency becomes a real product requirement. Introducing additional asynchronous machinery just to improve this one acceptance test would be unnecessary complexity.

writeup6(2).md

### **D. Replace the monolithic browser test with a resumable acceptance suite**

This is the immediate work needed to finish Boundary B.

The browser test should become an orchestrated collection of independently runnable acceptance phases.

**Proposed Boundary B acceptance structure**

1. **Environment and authentication**

   Verify database connectivity, server health, login, operator identity, and tenant context.
2. **Conversation and dispatch**

   Verify conversation creation, Line submissions, sender dispatch, persisted acknowledgments, and callbacks.
3. **Clarification lifecycle**

   Verify question delivery, frozen turn binding, answer submission, continuation, stale answers, and concurrency.
4. **Approvals and Evidence**

   Verify approval and rejection, persisted receipts, dispatch outcomes, and audit exclusions.
5. **Console behavior**

   Verify seven scenes, OPS/DEV, placeholders, loading, empty, failed, pending, and refused states.
6. **Recovery and isolation**

   Verify offline/reconnect, server restart, dispatch retry, tenant B isolation, and keyboard/mobile behavior.
7. **Final clean-slate acceptance**

   Execute the complete sequence against a fresh isolated database and preserve the final evidence.

Each phase must:

- Be executable independently against existing state.
- Have deterministic setup for its own fixtures.
- Use stable correlation IDs rather than `latest row` lookups.
- Assert relative changes rather than absolute counts.
- Record its checkpoints and failure diagnostics.
- Avoid destroying the database after an assertion failure.
- Fail explicitly when its prerequisite state is missing.

A single orchestration script can still execute all seven phases. The difference is that an error in phase five no longer requires destroying the results of phases one through four.

The final clean-slate run remains mandatory. We are making the test suite easier to debug, not weakening its acceptance gate.

writeup6(2).md

## **3. Recommended implementation sequence**

I would not combine all four corrections into one large implementation task.

| Order | Work                                                | Decision                                |
| ----- | --------------------------------------------------- | --------------------------------------- |
| 1     | Finish and verify the audit exclusion               | Already implemented; validate           |
| 2     | Refactor the browser matrix into independent phases | Proceed now                             |
| 3     | Run U5 and complete Boundary B                      | Proceed after matrix restructuring      |
| 4     | Separate application and test database roles        | Dedicated infrastructure-hardening task |
| 5     | Event-driven provisioning wake-up                   | Defer until justified                   |

One important distinction: **I would not hold Boundary B hostage to the provisioning notification enhancement.** Nor would I silently add credential redesign to U5.

However, I would treat the shared-role problem as a real outstanding architectural defect. Before adopting a workflow that runs full Go suites while a long-lived Studio server is operating, it must be corrected.

The document itself reports U1–U4 complete with the full Go gates green, while U5 has several remaining browser acceptance cases. That is where we should concentrate the immediate effort.&#x20;

writeup6(2).md



## **4. What I would require from the coding agent**

The implementation task should explicitly prevent superficial fixes. In particular:

- Do not merely add longer browser timeouts.
- Do not add arbitrary sleeps to hide scheduling races.
- Do not replace absolute assertions with weaker assertions that allow failures to pass.
- Do not delete audit records to clean up Evidence.
- Do not hardcode a shared database password.
- Do not introduce notifications without a durable recovery mechanism.
- Do not rewrite the Miniapp runtime to accommodate Studio's Evidence presentation.

Every change must have a regression test that demonstrates the underlying invariant.

For the credential correction, the most valuable test is concurrent live-server survival. For the browser matrix, it is independently rerunning a failed phase against the existing database. For the Evidence view, it is demonstrating that noisy transport activity cannot displace business receipts.

## **5. My architectural decision**

Proceed with U5 restructuring

I would authorize the matrix restructuring and completion of Boundary B as the immediate task.

I would track the PostgreSQL role separation as a separate, mandatory infrastructure-hardening item before allowing test-suite/live-server concurrency. I would keep provisioning notifications deferred.

The larger lesson is that Miniapp Studio needs a reliable **verification environment**, not just a reliable product. A passing test suite should not accidentally invalidate a running application, a browser test failure should not destroy all previous evidence, and transport activity should never obscure the business events we claim to verify.

That is the level at which I would fix this—not by making the next browser run pass, but by making the entire development and acceptance process structurally dependable.&#x20;

## Z

The writeup's diagnosis is sound — this is a verification-architecture problem, not a product bug: Cause 4 makes every finding cost a full world-reset, and Cause 1 makes those resets *unreliable* (a green test run silently kills the live server minutes later). Causes 2 and 3 are already correctly dispositioned (fixed at the query layer; deferred with rationale). So my plan is to adopt §3 as written, with **two additions the writeup misses** and an explicit sequence with gates. Here it is:

## 1. Now, before resuming U5 — restructure the matrix (Cause 4, immediate)

Adopt §3.4 exactly: per-phase scripts, relative assertions, wipe-and-rerun reserved for exactly one final clean-slate pass. I'm adding three things to that restructure, all in the script layer:

- **Per-phase prerequisite preambles.** Each phase asserts its own preconditions first (server up, operator provisioned, build X in state Y) and fails fast with a precise message — so a phase fails with "server not reachable" at line 3, not with a confusing DOM assert at step 14. This is what makes a script bug cost seconds instead of a cycle.
- **A state-ledger file** (one JSON: build ids, phase-unique titles, epochs) written by each phase and read by the next, replacing every "latest row" lookup. Relative assertions become data-driven instead of positional.
- **The clean-slate pass runs twice.** Acceptance evidence = the second consecutive green run; one pass proves correctness, two prove repeatability — the same standard the demo rehearsal (G7) will be held to.

**Gate for this step:** sabotage test — intentionally break one assertion in one phase, show only that phase fails, rerun only that phase, green. That demonstration *is* the proof the circling is dead; no full reset was performed.

## 2. During all live U5 passes — a written single-writer rule (the writeup's gap)

§3.1 defers the role fix, but never states the interim hazard rule, and the remaining matrix steps make it acute: **restart-resume restarts the server mid-matrix**, so if anyone runs `go test ./...` against the shared cluster while a live pass is in flight, Cause 1 recurs exactly where it hurts. Interim rule, one line in the runbook header: *while a live U5 pass is in progress, the PG19 cluster is single-writer — no suite runs until the pass finishes or aborts.* Costs nothing, closes the recursion window until the real fix lands.

## 3. Immediately after U5 greens and the report is filed — land the role separation (Cause 1)

Agree with §3.1: bounded follow-up, not smuggled into U5. Scope as written (split `EnsureAppRole` into create-if-missing+grants vs. explicit rotate used only by the provision CLI; dedicated test role; all four harnesses; runbook wording) with the red-before/green-after proof: **live server survives a concurrent suite run.**

One strengthening of the invariant, because `studio_app_test` alone only moves the bug: the durable rule is **test code may only rotate roles it created with run-unique names** — otherwise two overlapping suite runs rotate `studio_app_test` out from under each other and the same failure recurs inside CI. Minimal-blast-radius version: test role's password is set once at cluster bootstrap and never rotated at all; suites isolate via per-run *databases*, not per-run password changes. Either satisfies the invariant; pick the smaller diff.

## 4. Keep Causes 2 and 3 as dispositioned — with one sizing addition

Cause 2's fix is already at the right layer (query-level `exclude`, schema-pinned, disclosed in UI, tested) and correctly does not fork the frozen runtime. Add one cheap item to the F7 residual note: **quantify it** — at 1Hz polling, audit rows accumulate at roughly 8,600/conversation/hour, so a day-active conversation writes on the order of 200k audit rows daily regardless of display exclusion. The exclusion fixes the evidence view; the write amplification is the actual residual and should be *sized*, not just named — that number is what later justifies framework audit categories or partitioning when the owner revisits it.

Cause 3: no product change now — polling converges, and adding LISTEN/NOTIFY machinery during a verification phase is new failure modes (missed notifies, restart replay) bought at the worst possible time. Keep the deferred design as a two-line sketch in the roadmap (trigger on `builds.gateway_url` + outbox row, wake only affected builds), and in the matrix keep the documented SQL wake with its assertion window derived from the 30s hold, so the 25s-vs-30s mismatch can't bite a fifth cycle.

## 5. Stop conditions — unchanged from §5, made explicit

Restructured matrix green → double clean-slate pass → Boundary B report (which now includes this four-cause postmortem, the interim single-writer rule, and the sized F7 residual) → **stop**. Explicitly out of scope, per §4: no framework changes, no F1/G-01/P5, no Sprite/Fly/OpenCode/Part 2 work, no commits without authorization. Expected-finding budget for the remaining steps, so they're planned discoveries rather than surprises: offline/reconnect and restart-resume are the two most likely to surface a real product defect; mobile sheet and keyboard kill are script-mechanics risks, not product risk.

Net: the writeup's fixes are all root-cause, not band-aid — my additions are the interim single-writer rule (which closes the hazard window the deferral leaves open), the run-unique-role invariant (which keeps the fix from recurring between suites), and the repeatability standard (two clean passes) so the matrix ends with evidence that survives the same scrutiny the project applies everywhere else.