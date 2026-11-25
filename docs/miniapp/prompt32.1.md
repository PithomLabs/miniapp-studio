This prompt keeps Prompt32.1 tightly scoped to architectural reconciliation. It requires the agent to correct the technical inconsistencies without redesigning the six documents or prematurely authorizing implementation.

# Prompt32.1 — Miniapp Foundational Architecture Reconciliation

## Mission

Act as the principal architect and adversarial technical reviewer of Miniapp.

Reconcile the six Prompt32 foundational architecture documents following the initial architectural review.

The goal is to eliminate contradictions, overstated guarantees, incomplete conformance mappings, and ambiguous execution semantics before owner approval of F-0.

**This is an architecture-documentation work order only. No production implementation is authorized.**

The result must be one internally consistent, technically defensible architecture that can guide subsequent implementation without forcing developers to understand or manually assemble Miniapp's internal machinery.

## 1. Governing architectural doctrine

The following decisions are locked:

1. Security and Observability are co-equal, critical foundational capabilities.
2. Developer Experience and Enforcement Architecture makes both automatic, mandatory, and testable.
3. The developer writes business logic; Miniapp owns the mechanisms required to execute it safely, reliably, securely, and observably.
4. Framework guarantees must be enforced by construction, not delegated to developer discipline.
5. API, MCP, CLI, and background workers share one authoritative security and execution boundary.
6. Classes A–D preserve their distinct execution and recovery semantics.
7. No execution class is production-ready without Security, Observability, and Developer Experience conformance.
8. Internal framework complexity is acceptable; developer-facing complexity must remain minimal.

Do not weaken or reinterpret these decisions.

## 2. Review inputs

Read all six existing documents in full:

- `docs/architecture/security-contract.md`
- `docs/architecture/observability-contract.md`
- `docs/architecture/developer-experience-enforcement.md`
- `docs/architecture/foundational-architecture.md`
- `docs/architecture/foundational-conformance-matrix.md`
- `docs/architecture/foundational-roadmap.md`

Also inspect the governing source material:

- `docs/plan4.2.md`
- Original Phase 1 invocation lifecycle contract.
- Phase 2 authorization and effect design.
- Phase 0 threat model.
- Prompt21–Prompt31 findings and dispositions.
- Prompt30 residual closeout.
- Existing security, audit, effect, registry, runtime, and transport implementations.

Use actual source and contract definitions. Do not rely exclusively on the architectural documents' descriptions of existing functionality.

Preserve the `[Existing]`, `[Partial]`, `[Documented]`, and `[Proposed]` classification discipline.

## 3. Correction A — Exactly-once execution claims

**Highest-priority correction.**

Inspect every occurrence of:

- Exactly once.
- Exactly-once dispatch.
- Exactly-once execution.
- Exactly-once business semantics.
- At-most-once.
- At-least-once.
- Idempotency.
- Stable operation identity.
- Provider delivery guarantees.

The unified architecture currently describes the kernel as dispatching exactly once per invocation. The Workflow Kit proposal also associates transaction and effect-ledger guarantees with exactly-once business semantics.

These descriptions must be corrected to match the actual guarantees.

Define these concepts separately:

1. Logical invocation identity.
2. Stable BusinessOpID across retries and attempts.
3. Durable dispatch intent and enqueue atomicity.
4. Job delivery and worker execution semantics.
5. Transactional business execution guarantees.
6. External-effect execution and provider-delivery guarantees.
7. UNKNOWN outcomes and evidence-based recovery.

Preserve these established properties:

- Class A transactional business writes and invocation completion share the appropriate transaction.
- Class A recovery is bounded and preserves stable BusinessOpID.
- Class B delivery and worker execution may repeat; idempotency and durable state must govern recovery.
- Class C pipeline progress and item-level recovery remain explicit.
- Class D replay, durable waits, checkpoints, and recovery remain DBOS-owned.
- Attempt fencing prevents stale workers from overwriting newer ownership.
- Ambiguous provider outcomes remain quarantined.
- External effects are never blindly resent merely because an invocation was retried.

The architecture must distinguish:

- At-most-one committed transactional business execution within the defined transaction boundary.
- Potentially repeated delivery or execution attempts.
- Idempotent business operations.
- At-most-one accepted effect-ledger transition for a given authoritative attempt.
- External provider guarantees, which depend on provider idempotency and evidence.

Do not promise exactly-once arbitrary external delivery.

Update the Runtime Kernel description, execution diagrams, Workflow Kit discussion, conformance requirements, and acceptance criteria wherever necessary.

## 4. Correction B — Separate durable audit from operational telemetry

Inspect the Observability Contract's event taxonomy.

It currently includes telemetry pipeline health events such as:

- `telemetry-degraded`
- `exporter-down`
- `exporter-recovered`

within the audit action vocabulary.

Determine whether each event represents an authoritative security/business transition or merely operational infrastructure health.

Apply this normative distinction:

**Durable audit**

Records authoritative, security-relevant, and consequential state transitions. Examples include authorization decisions, approval decisions, effect transitions, delegation changes, and operator settlement.

**Operational logs and metrics**

Record exporter health, telemetry delivery failures, queue saturation, dropped spans, exporter recovery, and other infrastructure diagnostics.

Do not introduce high-volume exporter-health records into PostgreSQL audit storage merely to make telemetry observable.

For each event category, define:

- Whether it belongs in durable audit, logs, metrics, or traces.
- Whether it is mandatory or sampled.
- Whether it is transactionally coupled to business state.
- Its retention and failure behavior.

Do not weaken the guarantee that audit and ledger state remain authoritative when external telemetry is unavailable.

Clarify which worker pickups, retries, recovery events, and reconciliation events warrant durable audit records versus ordinary telemetry.

The classification must be consistent across the Observability Contract, unified architecture, and conformance matrix.

## 5. Correction C — Make sampling technically implementable

Resolve the contradiction between:

- SDK head sampling for routine spans.
- Collector-side tail sampling that retains all error, recovery, and UNKNOWN traces.

A Collector cannot retain a complete trace that the SDK discarded before export.

Evaluate implementable alternatives:

**Option A:** Preserve candidate traces for Collector-side tail-sampling decisions, using bounded export queues and explicit volume controls.

**Option B:** Apply SDK head sampling and accept that some errors and recovery traces will not reach the Collector.

**Option C:** Propose another technically sound strategy, explaining its resource, latency, privacy, and cost implications.

For whichever strategy is proposed:

- Explain where sampling occurs.
- Explain which traces can be lost.
- Explain parent/child and asynchronous trace propagation.
- Define replay behavior for Class D.
- Preserve bounded telemetry queues.
- Ensure telemetry exporter failure cannot block admission, authorization transactions, effect attempts, or business execution.
- Ensure audit records are never sampled.

Do not claim that all errors are retained unless the architecture actually guarantees that property.

Where a material trade-off requires an owner decision, record it explicitly in the decision register and identify which implementation milestone it blocks.

## 6. Correction D — Reconcile roadmap and conformance criteria

Audit every F-0–F-8 stage against:

- The Security Contract's acceptance criteria.
- The Observability Contract's acceptance criteria.
- The DX acceptance criteria.
- The shared and class-specific conformance matrix.

### D1. F-1 Security criteria

The current F-1 exit criteria require Security acceptance criteria 1–5 on Classes A/D.

However, Security criterion 3 includes worker-checkpoint verification, which also applies to B/C, not yet implemented.

Split the acceptance criteria accurately:

- F-1 proves the shared security obligations applicable to existing A/D.
- F-5 proves Class B worker pickup and checkpoint obligations.
- F-6 proves Class C pipeline/item obligations.
- F-7 proves integrated A–D conformance.

Do not require nonexistent B/C behavior to pass F-1.

Do not silently weaken the Security Contract's eventual B/C requirements.

### D2. Workflow Kit conformance

The Developer Experience Contract requires a three-fixture reuse demonstration for the Class D Workflow Kit.

The current conformance matrix does not give that specific requirement an identifiable independent conformance row.

Add an explicit requirement with:

- A unique conformance identifier.
- Owning contract reference.
- Required behavior.
- Evidence/test shape.
- F-7 milestone linkage.
- Acceptance criterion for actual fixture reuse rather than interface presence.

Keep the existing adversarial demo requirement distinct.

Reconcile row counts, identifiers, roadmap references, and DX acceptance criteria.

### D3. General consistency

Verify that:

- Every acceptance criterion has an identifiable conformance row or explicitly documented non-testable owner decision.
- Every conformance row has an owning contract.
- Every roadmap exit proof maps to concrete evidence.
- Shared requirements and class-specific requirements are not confused.
- No milestone claims conformance before its relevant suite exists.

## 7. Correction E — Define delegation-revocation semantics

The Security Contract describes delegation revocation as immediate, while execution-class checkpoints differ.

Resolve this architectural ambiguity.

Define the behavior for:

1. New synchronous invocations after revocation.
2. Queued Class B jobs admitted before revocation but not yet picked up.
3. Class B jobs already executing when revocation occurs.
4. Class C pipelines between individual items.
5. Class D workflows during durable waits.
6. Class D workflows during replay or resumed execution.
7. Consequential external-effect authorization after revocation.
8. A revocation race with an already-authorized provider attempt.

Distinguish:

- Immediate refusal of new admission.
- Re-verification at worker pickup.
- Re-verification at pipeline/item checkpoints.
- Re-verification before consequential external effects.
- Operations already past the authorization/attempt boundary.

Do not assume that pickup-time verification alone provides immediate revocation.

Do not add advisory locks, invert the global lock order, use the Class A attempt transaction for authorization, or otherwise violate C1–C7.

Explicitly preserve the existing grant configuration hash and ten-minute revocation-staleness semantics. Do not silently redefine them as delegation semantics.

Where a revocation guarantee cannot be provided without a new owner-approved mechanism, state the exact limitation and add a decision-register item. Do not invent a false real-time guarantee.

## 8. Correction F — Clarify Go library enforcement versus isolation

The DX Contract correctly acknowledges that arbitrary trusted Go code can bypass a library boundary.

Ensure this limitation is consistently expressed in:

- Runtime Kernel description.
- Escape-hatch rules.
- Static analysis requirements.
- Developer demonstration matrix.
- Conformance criteria.

Use precise language:

- Miniapp can provide no supported framework-mediated bypass path.
- Static analysis can detect known prohibited direct provider calls.
- CI can reject detectable violations.
- Public APIs can make trusted authority difficult to forge or construct.
- Arbitrary code executing in the same trusted Go process is not a sandbox.
- Genuinely hostile or agent-authored code requires a process, VM, or other isolation boundary.

Do not claim that static analysis makes bypass impossible for arbitrary Go code.

Define what the framework guarantees and what it merely detects.

## 9. Correction G — Normalize milestone identifiers

The roadmap currently uses M1–M12 for new foundational milestones, while Miniapp already has lifecycle M1–M6 and the M5 sequence-verification requirement.

Rename the new foundational milestones to:

- `F-M01` through `F-M12`.

Keep existing lifecycle M1–M6 and M5 sequence terminology unchanged.

Update every reference across all six documents.

Do not rename F-0 through F-8 stages.

Ensure that cross-document references and decision IDs remain unambiguous.

## 10. Preserve all established invariants and residuals

Line-by-line verify the following:

- Original C1–C7 meanings.
- Global lock ordering.
- Authorization uses its own transaction/pool checkout.
- Class A bounded recovery and stable BusinessOpID.
- Invocation and transactional business-write atomicity.
- Attempt fencing and stale-outcome protection.
- `runtime.Service.SettleEffect` as the sole shared Settle boundary.
- Exact `miniapp.settle.operator` authority.
- M5 sequence activation requirements.
- P24-008 capability-membership requirement before real provider integration.
- Existing grant configuration hash and ten-minute staleness semantics.
- UNKNOWN quarantine and no blind resend.
- Class-specific recovery semantics.

Carry forward without silently closing:

- P31-001: degenerate-binding refusal test at Settle boundary.
- P31-001: boundary conflicting-repeat test.
- P29-005: abandoned-PENDING cross-invocation wedge, documented but execution-unverified.
- Operational revocation drill: NOT EXECUTED pending an authorized isolated multi-replica environment.
- CI execution limitations where applicable.

Architecture documentation does not close implementation or execution-evidence residuals.

No new guarantee may contradict existing tested behavior.

## 11. Cross-document reconciliation audit

After changes, inspect all six documents together.

Perform a mechanical audit:

1. Search for every exactly-once and at-most-once claim.
2. Search for audit, log, metric, trace, exporter, sampling, and retention terminology.
3. Verify revocation terminology and checkpoints by class.
4. Verify Workflow Kit references and conformance identifiers.
5. Verify all F-M01–F-M12 references.
6. Verify F-1–F-8 entry and exit criteria.
7. Verify every owner decision ID appears in the decision register and every reference resolves.
8. Verify the capability-status tags remain accurate.
9. Resolve every internal section reference and relative document link.
10. Ensure no proposal is accidentally described as implemented.

Create a requirement-to-document ownership matrix covering every correction and original Prompt32 requirement.

A requirement must have one authoritative owner document and may be cross-referenced elsewhere, never duplicated inconsistently.

## 12. Required deliverables

Update the six existing files:

- `docs/architecture/security-contract.md`
- `docs/architecture/observability-contract.md`
- `docs/architecture/developer-experience-enforcement.md`
- `docs/architecture/foundational-architecture.md`
- `docs/architecture/foundational-conformance-matrix.md`
- `docs/architecture/foundational-roadmap.md`

Create:

`docs/architecture/prompt32.1-reconciliation.md`

The reconciliation report must contain:

- Baseline HEAD and worktree status.
- Each correction and its disposition.
- Exact sections changed.
- Technical rationale.
- Owner decisions added or revised.
- Updated conformance-to-roadmap traceability.
- Invariant and residual preservation checklist.
- Cross-document terminology and link audit.
- Explicit statement that no production implementation occurred.
- Remaining blockers before F-0 approval.

Do not create a second competing architecture or duplicate the six specifications.

## 13. Explicit prohibitions

- No production implementation.
- No Go source changes.
- No migrations or dependency changes.
- No Class B/C adapters.
- No telemetry exporters or OTel dependencies.
- No identity-provider integrations.
- No generic IAM or policy engine.
- No new sequence subsystem.
- No real provider integration.
- No changes to existing tested runtime behavior.
- No changes to historical evidence or closeout verdicts.
- No commits, staging, pushes, deployments, or releases.
- Do not overwrite outside author activity.
- Do not silently resolve owner decisions that require explicit approval.

This remains architecture reconciliation only.

## Final instruction

Be precise about guarantees.

A design that sounds reassuring but cannot be implemented without contradicting the existing execution model is not acceptable.

Prefer a smaller, enforceable guarantee over an expansive but misleading promise.

Solve the underlying architectural inconsistency rather than merely editing the wording that exposed it.

Stop after producing the reconciled architecture package and evidence-backed report.

Return the package for owner review. Do not begin F-1, F-2, or any other implementation stage without explicit owner approval.

## **Acceptance standard**

The most important outcome is that the documents make **no false reliability promises**, have a technically implementable telemetry architecture, and establish clear revocation behavior across A–D.

Once Prompt32.1 is complete, we can make an informed F-0 approval decision and authorize the first shared Security and Observability implementation milestones.