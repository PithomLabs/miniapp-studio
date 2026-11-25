**Consolidated verdict: APPROVED.** I would adopt the proposed additions, with two corrections: use a more precise name than “single-writer,” and correct the audit-volume arithmetic.

The central diagnosis is right: Cause 4 makes every discovery expensive; Cause 1 makes the resulting resets unreliable. The other two causes have already received appropriate dispositions.&#x20;

writeup6(2).md



## **Consolidated decisions**

| Issue                                   | Decision                                                                                            |
| --------------------------------------- | --------------------------------------------------------------------------------------------------- |
| Cause 1 — shared PostgreSQL credentials | Temporary operational restriction during U5; permanent role separation immediately afterward        |
| Cause 2 — Evidence contamination        | Keep implemented SQL-level exclusion; quantify remaining audit write amplification                  |
| Cause 3 — provisioning wake-up          | Keep polling; improve test timing; defer notification architecture                                  |
| Cause 4 — monolithic browser matrix     | Restructure immediately; add prerequisites, state ledger, sabotage test, and two clean-slate passes |

### **Important corrections and refinements**

**1. The interim restriction is valid, but I would not call it a single-writer rule.**

Studio itself must continue writing to PostgreSQL during U5. What we actually need is an exclusive credential-mutation window:

> During a live U5 pass, no Go test suite, provisioning command, or other process that can mutate cluster-global PostgreSQL role credentials may run against PG19. If this restriction is violated, abort the pass, verify database authentication and server health, and restart the acceptance run.

This closes the immediate hazard without confusing it with ordinary database writes.

**2. The fixed&#x20;**`studio_app_test`**&#x20;role is my preferred permanent solution.**

Choose the smaller-blast-radius design:

- Create `studio_app_test` once through explicit test-environment bootstrap.
- Set its password once and never rotate it from test code.
- Give it access only to isolated test databases.
- Isolate concurrent test runs through run-specific databases, not shared-role password changes.
- Remove password mutation from ordinary `EnsureAppRole` behavior.
- Keep password rotation as an explicit administrative provisioning operation.

The invariant is stronger than merely introducing another role:

**No test suite may rotate a cluster-global application-role credential.**

**3. The audit-volume arithmetic needs correction.**

The attached writeup reports approximately 120 audit rows per minute for a browser polling at 1 Hz, with two audit records per invocation.&#x20;

writeup6(2).md



That implies:

| Metric                | Per continuous stream |
| --------------------- | --------------------- |
| Poll invocations/hour | 3,600                 |
| Audit rows/hour       | 7,200                 |
| Poll invocations/day  | 86,400                |
| Audit rows/day        | 172,800               |

The proposed 8,600/hour and 200,000/day figures are not consistent with the writeup's 120 rows/minute observation.

Record the derived numbers as estimates and measure actual rates during acceptance. Multiply by the number of simultaneous active streams. Importantly, query exclusion prevents receipt displacement; it does not prevent the underlying audit writes.

**4. Two clean-slate passes are a good strengthening.**

Each pass must use an independently provisioned clean database. The second consecutive green pass becomes the final repeatability evidence. One successful pass followed by a failure is not acceptance.

## **Consolidated execution directive**

# Boundary B — Consolidated U5 Execution and Verification Plan

## Objective

Finish Boundary B by correcting the browser-matrix verification architecture, proving repeatability, and producing trustworthy acceptance evidence.

The work must address the underlying causes identified in Writeup 6. Do not merely make the next browser run pass.

## 1. Immediate work — Cause 4: restructure the browser matrix

Adopt the original §3.4 approach, incorporating these additional requirements.

### 1.1 Independent phase scripts

Split `drive.py` into independently executable acceptance phases.

Each phase must:

- Be runnable against existing database and server state.
- Have deterministic phase-specific setup.
- Assert its prerequisites before performing work.
- Record its results and generated identifiers.
- Fail with a specific diagnostic when prerequisites are absent.
- Never require a complete environment reset merely because an assertion fails.

Use prerequisite preambles that verify the actual server, database, operator, conversation, build, and expected lifecycle state.

A missing prerequisite must produce a clear failure at the beginning of that phase, not an unrelated browser assertion later.

### 1.2 Run-scoped state ledger

Introduce one JSON state ledger for each acceptance run.

Store it in a run-specific location. It must include:

- Schema version and unique run ID.
- Database identity.
- Relevant environment and server identity.
- Build IDs and conversation IDs.
- Phase-unique titles.
- Relevant timestamps and epochs.
- Phase completion status and evidence references.

Each phase writes its verified outputs atomically. The next phase reads the exact recorded identifiers.

Replace all `latest row`, epoch-second ID, and absolute-count assumptions with ledger-driven identifiers and relative assertions.

Reject stale, malformed, incompatible, or mismatched ledgers. Do not store credentials or secrets in the ledger.

### 1.3 Relative assertions

Replace assertions such as:

- "Stopped 2 builds."
- "Use the latest row."
- "Find the build created most recently."

With assertions against the exact resources created by the current run.

Examples:

- Count changes relative to the pre-operation count.
- Confirm cancellation for the exact build IDs in the ledger.
- Find phase-owned records through unique IDs.
- Assert event ordering through persisted sequence and correlation identifiers.

### 1.4 Resumability sabotage test

Demonstrate that the new structure actually eliminates the world-reset cycle.

Intentionally break one assertion in one isolated phase.

Expected demonstration:

1. Only that phase reports the assertion failure.
2. The harness preserves the existing database and previous phase results.
3. The phase reports its failure and evidence.
4. Restore the assertion.
5. Rerun only the affected phase.
6. The phase passes without a global reset.

Dependent phases remain blocked rather than being reported as independent failures.

This sabotage test is a mandatory gate.

### 1.5 Two final clean-slate passes

Reserve full database teardown and recreation for exactly two final acceptance passes.

Each pass must start with an independently provisioned clean database and execute the complete matrix.

The acceptance criterion is two consecutive complete green passes.

The second pass is the final repeatability evidence.

Do not weaken the matrix to achieve this.

## 2. Interim operational restriction — Cause 1

Until permanent PostgreSQL role separation is implemented:

During a live U5 pass, no Go test suite, provisioning command, or other process capable of rotating cluster-global PostgreSQL application-role credentials may run against PG19.

Place this restriction prominently in the U5 runbook header.

If the restriction is violated:

- Abort the live acceptance pass.
- Verify the database role configuration.
- Verify live server database connectivity.
- Restart or reprovision the server as required.
- Begin a new acceptance pass.

This is an explicitly temporary operational restriction, not the permanent solution.

## 3. Complete U5 against the preserved implementation

After the matrix restructuring and sabotage gate pass, finish all remaining browser acceptance cases.

Retain the complete original U5 scope:

- Evidence receipts.
- OPS/DEV density.
- Offline/reconnect.
- Server restart and resume.
- Black-hole retrying.
- Keyboard kill.
- Tenant-B isolation.
- Mobile Pulse sheet.
- Screenshots and inspected evidence.
- All seven scenes and the required UI states.

Pay particular attention to offline/reconnect and restart-resume as high-risk integration boundaries.

Do not dismiss failures as script mechanics without tracing the actual execution path.

For mobile and keyboard behavior, distinguish DOM/test-harness failures from genuine accessibility or interaction defects.

### Provisioning timing

Keep the existing polling model.

The SQL wake remains explicitly labeled test scaffolding.

Derive the assertion deadline from the actual configured 30-second sender hold plus bounded scheduling and dispatch allowance. Do not use a 25-second timeout against a 30-second hold.

Use bounded polling for the expected persisted outcome rather than arbitrary sleeps.

Do not introduce LISTEN/NOTIFY during U5.

## 4. Cause 2 — Evidence exclusion and audit-volume residual

Retain the implemented `activity.list exclude` behavior.

The exclusion must remain at the query layer, before LIMIT, and must be disclosed by the Evidence UI.

Do not modify the frozen Miniapp runtime's audit semantics merely to improve Studio's presentation.

Extend the F7 residual documentation with the quantified audit-volume estimate.

At one poll per second and two audit records per invocation:

- 86,400 tool invocations/day per continuously active stream.
- 172,800 audit records/day per continuously active stream.
- Approximately 7,200 audit records/hour.

Label these as derived estimates from the observed 120-records/minute rate. Measure actual volume under the browser matrix and distinguish invocation counts from audit-row counts.

The exclusion fixes Evidence visibility. It does not fix audit write amplification.

Keep audit categories, retention policy, or partitioning as future Miniapp-runtime considerations.

## 5. Cause 3 — provisioning completion

Keep polling as the authoritative convergence mechanism.

Document the future structural improvement in the roadmap:

- Durable provisioning-completed signal or outbox record.
- Notification to wake only the affected build's sender rows.
- Sender re-reads authoritative database state before dispatch.
- Polling and reconciliation remain available after missed notifications and restarts.

Do not introduce this additional machinery into Boundary B.

## 6. Immediately after Boundary B acceptance — permanent Cause 1 correction

After U5 produces two consecutive clean-slate passes and the Boundary B report is filed, implement the PostgreSQL role separation as a dedicated follow-up.

### Required changes

1. Split `EnsureAppRole` into create-if-missing and grants, without changing an existing password.
2. Keep password rotation as an explicit provisioning-only administrative operation.
3. Introduce `studio_app_test`, initialized once through explicit test-environment bootstrap.
4. Never rotate `studio_app_test` from test code.
5. Isolate test suites using run-specific databases.
6. Grant the test role access only to the appropriate test databases.
7. Audit and update all four existing test harnesses.
8. Update provisioning documentation and runbooks.

### Mandatory regression proof

Start a live Studio server authenticated as `studio_app`.

While it is serving requests, run the complete Go test suite.

Demonstrate:

- The live `studio_app` credential remains unchanged.
- The live server remains healthy.
- Authenticated database-backed application requests continue to work.
- Test code cannot access the live application database.
- Tests use isolated databases.
- Concurrent test runs do not rotate or invalidate one another's credentials.

Do not accept a solution that merely moves the password-rotation race to a shared test role.

## 7. Required Boundary B report

The report must contain:

- Restructured matrix architecture.
- Individual phase execution evidence.
- State-ledger design and sample.
- Prerequisite and relative-assertion examples.
- Sabotage-test results.
- Both consecutive clean-slate acceptance results.
- Screenshots and browser evidence.
- Four-cause postmortem.
- Interim credential-mutation restriction.
- Quantified audit write amplification.
- Provisioning-poll timing and SQL-wake evidence.
- Remaining residuals and risks.

Update `docs/hackathon-demo-critical-path.md` with actual evidence references.

## 8. Stop conditions and prohibited scope

After the two clean-slate passes and Boundary B report, STOP and request architectural acceptance.

Do not:

- Modify the Miniapp framework or runtime.
- Change F1, G-01, or P5.
- Provision Sprites or perform Fly operations.
- Launch live OpenCode or other provider sessions.
- Execute Phase B Part 2 or C5.
- Begin Phase D or E.
- Commit or push without explicit authorization.

This task is about making the current verification process dependable, repeatable, and honest—not expanding the product.

## Definition of success

A browser-matrix failure must be locally diagnosable and rerunnable without destroying previously verified state.

A successful test run must not invalidate a live Studio server.

Evidence must remain visible despite transport polling.

Provisioning latency must be handled with correct test timing rather than arbitrary timeouts.

Boundary B is accepted only after two independently provisioned, consecutive clean-slate passes and a complete evidence report.

## **Final sequencing decision**

1.

   GO

   &#x20;Restructure U5, add the state ledger and preambles, and prove phase-level recovery through the sabotage test.

2. Run the entire matrix twice from fresh isolated databases, with the interim credential-mutation restriction enforced.

3. File Boundary B report and stop for acceptance.

4. After acceptance, implement permanent PostgreSQL role separation and prove concurrent live-server/test-suite survival.

**My approval is for this sequence.** Causes 2 and 3 remain dispositioned; Cause 1 receives a temporary safety restriction and then a permanent correction; Cause 4 is addressed immediately. No reason to broaden the current U5 scope.