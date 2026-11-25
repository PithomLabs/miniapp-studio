**Authorized. Run the bounded Phase A closure.** Do not proceed to Phase B yet.

The new finding changes the acceptance decision: `FORCE ROW LEVEL SECURITY` without `ENABLE ROW LEVEL SECURITY` leaves the policy layer inactive. The audited tenant predicates contain the current exposure, but they are not a substitute for the required database enforcement layer. The live secret-canary probes also establish that the original redaction gap is real, not theoretical.&#x20;

adv_review_prompt8_phase_a.md

adv_review_prompt8_phase_a.md

adv_review_prompt8_phase_a.md



The closure is now three mandatory items. Fold in M1–M4 only where they remain small, isolated fixes; carry M5–M8 into the next phase as documented. No broad re-review afterward.

# Authorized Coding Task: Phase A Bounded Closure

**Decision:** Proceed with the bounded closure. Phase A remains unaccepted until the three mandatory items below are corrected, tested, and documented.

**Primary review report:** `plans/adv_review_prompt8_phase_a.md`

**Evidence report to update:** `docs/gateway-phase-a-evidence.md`

## 1. Agent-content secret handling — mandatory

Investigate the complete path from callback ingestion through persistence, retrieval, audit, SSE delivery, and rendered conversation history.

The live review demonstrated that an embedded `sk-live-` canary in an agent question body survives persistence and is returned verbatim by `build.status`, `conversation.get`, and `conversation.stream`.

Implement a durable correction at the appropriate ownership boundary. At minimum:

- Redact recognized secret-shaped content before it reaches Studio-owned persisted agent-message surfaces.
- Cover question bodies, question options, proposals, authorization requests, and event-derived conversation content.
- Cover embedded or mid-string secrets, not just values beginning with a known prefix or secrets appearing under denylisted keys.
- Ensure all relevant read and delivery paths remain safe: build status, conversation retrieval, stream events/SSE, and rendered history.
- Preserve useful surrounding text and make redaction behavior testable and consistent.
- Do not solve this by adding isolated output filters while leaving the same unsanitized content in Studio-owned tables.

Add regression tests using secret-shaped canaries, including embedded canaries. Exercise the real tool/callback paths and assert both database persistence and every relevant read/delivery surface. Include the previously failing question-body scenario.

**Required disclosure:** The review identified that the pinned Miniapp framework persists full tool inputs and results in `app.invocations`. This is framework-owned behavior with no Studio read path exposing those values. Do not silently claim that no unsanitized copy exists anywhere. State this residual explicitly in the evidence report, and do not modify Miniapp or its invocation-storage design as part of this task.

## 2. Genuine shutdown → restart recovery — mandatory

Correct `TestPhaseARestartRecovery` so the first server is actually shut down before the replacement server starts.

Prove the ordered lifecycle against the same persistent database:

1. Start the first server and create the relevant conversation/build/question state.
2. Park the question and preserve the database state.
3. Shut down the first server and close its resources exactly once.
4. Confirm shutdown has completed before starting the replacement server.
5. Start a fresh server instance against the same database.
6. Apply the parked question's answer.
7. Verify conversation history reconstruction and SSE replay from the prior cursor.
8. Deliver the duplicate callback and verify it is acknowledged without reapplying the state transition.

The test must not leave both server instances running concurrently or rely on process-local state surviving shutdown. Handle shutdown errors honestly; do not mask a failed shutdown merely to make the test pass.

## 3. Enable RLS on all Phase A tables — mandatory

Add migration `000016` that enables row-level security on all ten Phase A tables identified in the review.

- Keep the existing FORCE setting and ensure both `relrowsecurity` and `relforcerowsecurity` are true.
- Provide a correct down migration.
- Extend the catalog test so all ten Phase A tables are checked, not just the original six.
- Add a behavioral enforcement test if practical: an intentionally unscoped query under the application role must demonstrate that the tenant policy is active.
- Correct the evidence report's current RLS security claim to reflect the verified state.

The regression test must fail against the current migration state and pass after migration `000016` is applied.

## 4. Small incidental findings

Fold in M1–M4 from the review only within this bounded scope:

- **M1:** Resolve the question-answer versus ingest lock-order inconsistency. Use a consistent build-first lock order and add a targeted concurrency regression test if the race can be reproduced or deterministically exercised.
- **M2:** Remove the unused `questionIDsFor` function, unless a real caller is identified.
- **M3:** Make `gateway.Parse` reject trailing JSON data consistently with the strict decoder behavior elsewhere.
- **M4:** Correct the inaccurate "gapless" stream-sequence comment. The documented guarantee is monotonic, gap-tolerant cursor semantics.

Do not spend this task expanding into M5–M8. Record those for the appropriate later phase as recommended in the review.

## 5. Verification and evidence

Run the full Phase A gates, including formatting, vet, build, complete tests, and race tests for the touched packages. Run migration up/down checks and verify the RLS catalog state.

Use the required PostgreSQL versions from the existing Phase A plan where available. If a required version cannot be run, disclose the exact environment limitation and do not represent a substitute-version run as equivalent evidence.

Update `docs/gateway-phase-a-evidence.md` with:

- The three corrections and their implementation locations.
- Regression tests and their actual results.
- Database persistence and output-path results for secret canaries.
- The ordered shutdown/restart test result.
- RLS enabled/forced catalog results for all ten tables.
- Exact PostgreSQL versions used and any deviations.
- The `app.invocations` residual disclosure.
- Any M1–M4 items not completed and why.

## 6. Strict boundaries

- No protocol or schema redesign beyond the required RLS migration and directly necessary corrections.
- No Sprite, OpenCode, OAuth, or Agent Gateway Phase B implementation.
- No F1 or G-01 changes.
- No P5 scope changes or Lead-to-Booking implementation.
- No Miniapp framework modifications.
- No infrastructure changes.
- No commits or pushes.

Investigate root causes and fix the underlying invariants. Do not weaken tests, suppress failures, or substitute superficial output filtering for a durable correction.

If an unexpected foundational issue requires an architectural decision, stop and report the concrete evidence rather than improvising a redesign.

## 7. Acceptance rule

When all three mandatory items are corrected, the relevant regression tests pass, and the Phase A gates are green, update the evidence report and declare Phase A closure complete.

**No additional general adversarial review cycle is required.** I will accept Phase A from the corrected evidence and move directly to Phase B, subject to the existing Phase B plan and its unchanged boundaries.

**Acceptance remains evidence-based:** a green test suite alone is insufficient. The corrected evidence must demonstrate the three invariants, especially that RLS is actually enabled and that secret canaries no longer traverse Studio-owned persistence and delivery paths. Once that evidence is complete, Phase A closes and Phase B begins without another broad review.