**Prompt30 closeout acknowledged.** This is a good transition point: the five Prompt29 findings have been addressed, and the full validation battery is reported green.

I accept the reported completion, with one distinction: I have not independently inspected the Prompt30 closeout artifacts in this turn.

Stage 3A remains released

## **1. Residuals to carry forward**

Three items must not disappear from the roadmap:

| Item                               | Disposition                                                             |
| ---------------------------------- | ----------------------------------------------------------------------- |
| Revocation operational drill       | NOT EXECUTED; requires an authorized isolated multi-replica environment |
| P29-005 abandoned-PENDING behavior | Documented, but execution-unverified                                    |
| P31-001 Settle boundary tests      | Two LOW coverage gaps: degenerate binding and conflicting repeat        |

P31-001 was identified after Prompt30 was planned, so I do not consider it closed based on the completion report. It is non-blocking but should be retained in the evidence register.

Also preserve the M5 and P24-008 activation requirements. No sequence workload or real provider integration should bypass those requirements.

## **2. Next major workstream: Prompt32**

I recommend moving directly into the coordinated foundational architecture program.

The first deliverable should be a **unified architecture specification**, not implementation. Security, Observability, and Developer Experience must be designed together because they determine how developers interact with every Miniapp execution capability.

# Prompt32 — Miniapp Foundational Architecture: Security, Observability, and Developer Experience

## Mission

Act as the principal architect of Miniapp.

Design the next foundational architecture of Miniapp around three co-equal, critical capabilities:

1. Security Contract.
2. Observability Contract.
3. Developer Experience and Enforcement Architecture.

The ultimate objective is to make Miniapp the business logic layer for Go applications.

Application developers should primarily write business logic. Miniapp should automatically provide and enforce the security, execution reliability, observability, and operational mechanisms required to run that logic safely.

**This is an architecture and specification work order. Do not implement the architecture.**

## 1. Governing doctrine

The following principles are mandatory:

> The developer writes business logic. Miniapp owns everything required to execute that business logic safely, reliably, securely, and observably.

> Framework guarantees must be enforced by construction, not delegated to developer discipline.

> The complexity of the framework must not leak into the complexity of the application.

Additional rules:

- The framework owns reliability; miniapps own business logic.
- Applications declare business policies and required permissions; Miniapp enforces them.
- API, MCP, CLI, and background workers share one authoritative security and execution boundary.
- Security and Observability are co-equal foundational capabilities.
- Developer Experience and Enforcement Architecture ensures both are automatic and mandatory.
- Classes A–D retain their distinct execution and recovery semantics.
- No execution class is production-ready without satisfying all three foundational contracts.

## 2. Source and architecture investigation

Before proposing designs, independently inspect the current repository.

Read the governing architecture, lifecycle, authorization, effect, audit, revocation, and execution contracts, including:

- `docs/plan4.2.md`
- Phase 1 invocation lifecycle contract.
- Phase 2 authorization and effect contracts.
- Current runtime and transport implementations.
- Prompt21–Prompt31 findings and dispositions.
- Prompt30 residual closeout.
- Existing roadmap and observability/security documents.
- Existing Class A and D implementation.
- Current capability registration and execution APIs.

Determine what already exists, what is partially implemented, what is only documented, and what remains absent.

Do not redesign working mechanisms simply because a different abstraction appears cleaner.

Create an explicit inventory mapping every proposed architectural capability to existing source, existing contract, or genuinely new work.

## 3. Workstream S — Miniapp Security Contract

Design a framework-owned security model covering:

### Identity and authentication

- Human identities.
- AI-agent identities.
- Service and workload identities.
- Authentication adapters for API, MCP, and CLI.
- Trusted principal resolution.
- Tenant context.
- Credential validation and protection.
- Integration with external identity providers.
- Separation between identity issuance and authorization enforcement.

Miniapp owns authorization enforcement. Host applications and identity providers may own identity issuance.

Do not build a complete identity-provider or IAM product unnecessarily.

### Authorization and delegation

- Capability-based authorization.
- Application-declared permissions.
- Resource and tenant boundaries.
- Explicit agent delegation.
- Least privilege.
- Authority expiration and revocation.
- Approval requirements.
- Server-derived identity and trusted context.
- Protection against caller-supplied identity, role, tenant, or authority spoofing.

### Execution-time security

Define how security context survives:

- Class A transactions.
- Class B queued jobs and retries.
- Class C pipelines and individual items.
- Class D durable waits, signals, checkpoints, replay, and recovery.

A background worker must not gain privileges simply because execution moved away from the original authenticated request.

Specify the relationship between invocation authorization, existing `authorizeAttempt`, effect authorization, approvals, and the Settle operator boundary.

MCP must never become a privileged backdoor.

## 4. Workstream O — Miniapp Observability Contract

Design one coherent observability model covering:

### Four complementary pillars

1. Audit: authoritative, durable records of consequential and security-relevant transitions.
2. Logs: structured operational diagnostics.
3. Metrics: quantitative health, reliability, performance, and throughput measurements.
4. Traces: end-to-end causal visibility across execution boundaries.

### OpenTelemetry

Define:

- Instrumentation architecture.
- Trace propagation.
- Metrics collection.
- Structured logging integration.
- OTLP export.
- External Collector integration.
- Exporter abstraction and configuration.
- Failure behavior when telemetry infrastructure is unavailable.

The authoritative audit and execution ledgers must remain independent of external telemetry availability.

### Correlation

Define common correlation conventions for:

- `principal_id`
- `actor_id`
- `tenant_id`
- `invocation_id`
- `run_id` / `workflow_id`
- `business_op_id`
- `effect_id`
- `attempt_id`
- `trace_id`
- `span_id`

Do not use high-cardinality identifiers as ordinary metric labels.

### Operational guarantees

Define:

- Redaction and sensitive-data handling.
- Metric cardinality limits.
- Sampling policy.
- Retention and storage boundaries.
- Diagnostic events.
- Error taxonomy.
- Dashboard and alert requirements.
- SLO and reliability measurements.
- Instrumentation testability.

Instrumentation must automatically cover invocation, authorization, transaction, dispatch, retry, recovery, approval, effect, provider, reconciliation, Settle, and workflow lifecycle events.

Business-specific diagnostic events should remain optional and domain-owned.

## 5. Workstream DX — Developer Experience and Enforcement Architecture

This is the mechanism that makes the Security and Observability Contracts real.

Design a small, coherent public programming model.

Investigate a Business Capability abstraction through which applications declare:

- Capability identity.
- Request and response types.
- Execution class.
- Required authorization.
- Business handler.
- Explicit business policies and approval requirements.

The application should not manually assemble:

- Authentication middleware.
- Tenant extraction.
- Transaction lifecycle.
- Invocation identity.
- Audit infrastructure.
- Structured logging.
- Metrics.
- Tracing.
- Retry and recovery.
- Effect ledgers.
- Provider authorization.

Design the mandatory enforcement mechanisms:

1. Typed APIs and compile-time constraints.
2. Startup validation.
3. Runtime enforcement.
4. Framework-owned conformance suites.
5. Static analysis and CI enforcement.

Define explicit escape hatches for legitimate advanced use cases. No silent opt-outs.

Be realistic about Go: arbitrary trusted application code can still import external packages and bypass a library boundary. Distinguish ordinary API-level enforcement from the process or sandbox isolation needed for genuinely hostile code.

Do not invent a massive public configuration API to solve framework complexity. Minimize developer-facing concepts.

## 6. Execution-class integration

Specify how all three foundational contracts apply to:

| Class | Execution model                                 |
| ----- | ----------------------------------------------- |
| A     | Plain Go and PostgreSQL transaction             |
| B     | River one-shot asynchronous job                 |
| C     | Bounded pipeline, River-first; DBOS selectively |
| D     | DBOS durable business workflow                  |

For each class, define:

- Invocation and identity propagation.
- Authorization checkpoints.
- Transaction and execution boundaries.
- Audit events.
- Logs, metrics, and traces.
- Retry and recovery instrumentation.
- Conformance obligations.
- Failure behavior.

Do not flatten the different semantics of the four classes.

Do not begin B/C implementation in this workstream.

## 7. Developer experience acceptance test

Design a realistic demonstration application.

A developer should be able to register and expose a business capability through API, MCP, and CLI while writing only its actual business logic and declaring its business permissions.

The framework should automatically handle the necessary security and observability mechanisms.

Define adversarial tests covering:

- Unauthenticated callers.
- Unauthorized humans and agents.
- Delegation violations.
- Cross-tenant access.
- Credential and identity spoofing.
- Interrupted execution.
- Retries and recovery.
- Ambiguous provider outcomes.
- Missing telemetry exporter.
- Sensitive-data leakage.
- Attempts to bypass canonical effect boundaries.

The acceptance test must demonstrate that developers do not need to remember and manually implement dozens of framework rules.

## 8. Conformance architecture

Design a reusable Miniapp Framework Conformance Suite.

It must test real behavior, not merely interface presence.

Define shared conformance requirements and class-specific test suites.

An execution adapter that fails Security, Observability, or execution-reliability conformance must not be declared production-ready.

Specify how adapter authors and miniapp developers can run these tests without having to recreate the framework's internal fixtures.

## 9. Preserve existing invariants and residuals

The design must preserve:

- Original Prompt21 C1–C7 definitions and established behavior.
- Global lock ordering.
- Separate authorization transaction requirements.
- Class A bounded recovery and stable BusinessOpID.
- Atomic effect-state transitions and attempt fencing.
- The `runtime.Service.SettleEffect` shared boundary.
- M5 sequence-verification activation requirements.
- P24-008 capability-membership requirement before real providers.
- Existing revocation-staleness semantics.

Carry forward explicitly:

- P31-001: missing degenerate-binding and boundary conflicting-repeat tests.
- P29-005: abandoned-PENDING behavior is documented but execution-unverified.
- Operational revocation drill: NOT EXECUTED pending an authorized isolated multi-replica environment.
- CI execution limitations, where still applicable.

Do not silently promote characterization or source inspection into execution proof.

## 10. Required deliverables

Create a coordinated specification package:

1. `docs/architecture/security-contract.md`
2. `docs/architecture/observability-contract.md`
3. `docs/architecture/developer-experience-enforcement.md`
4. `docs/architecture/foundational-architecture.md` — unified architecture and interaction between the three contracts.
5. `docs/architecture/foundational-conformance-matrix.md`
6. `docs/architecture/foundational-roadmap.md`

Include:

- Executive architectural rationale.
- Existing-capability inventory.
- Responsibilities and boundaries.
- Public programming model proposal.
- Internal architecture.
- Execution lifecycle diagrams.
- Threat model.
- Observability event and metric taxonomy.
- Conformance matrix.
- Migration and compatibility strategy.
- Risks and unresolved decisions.
- Staged implementation plan.
- Explicit acceptance criteria.

Distinguish existing code, proposed designs, assumptions, and decisions requiring owner approval.

## 11. Implementation sequencing recommendation

Recommend a concrete sequence:

1. Approve the coordinated contracts.
2. Implement shared foundations against existing Class A and D paths.
3. Verify shared security and observability conformance.
4. Accelerate Class B and C with all three foundations incorporated from their first working versions.
5. Establish cross-class A–D conformance.
6. Demonstrate the intended developer experience with a realistic Go application.

Do not assume all foundational capabilities must be completed in one enormous implementation release. Identify independently testable milestones without fragmenting the architecture.

## 12. Prohibitions

- No production implementation.
- No Class B/C adapter implementation.
- No migrations or dependency changes.
- No generic IAM platform.
- No competing telemetry backend.
- No UI or dashboard product implementation.
- No weakening of existing authorization, audit, execution, or recovery guarantees.
- No new sequence subsystem.
- No real provider integration.
- No commits, staging, pushes, deployments, or releases.
- Do not overwrite or reset outside author activity.
- No Phase 3 work.

## Final instruction

Think like the architect of a framework that will be used by developers who should not need to understand its internal machinery.

The specification must be technically rigorous internally and radically simpler externally.

Solve the underlying complexity problem, not merely its visible symptoms.

Stop after producing the architecture specification package and an evidence-backed implementation roadmap. Wait for owner review before implementation.

## **3. My architectural direction**

I want to emphasize one thing before this work begins: **do not design three independent subsystems that developers have to assemble.**

Security, Observability, and Developer Experience are three perspectives on one runtime architecture.

- Security establishes who may do what.
- Observability establishes what happened, why, and how.
- Developer Experience ensures neither responsibility is forgotten or manually reimplemented.

That is the foundation on which we should build the remaining execution classes.

**Prompt32 should be architecture-only.** Once its specification is reviewed and approved, we can authorize implementation in carefully bounded stages.