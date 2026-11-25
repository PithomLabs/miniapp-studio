Yes. **An adversarial review is worthwhile before P2**, because P1 establishes the specification's trust boundary. The most important claim to challenge is that writes are structurally unrepresentable—not merely rejected by a few validator tests.

I would make this a single, read-only review with a strict approval threshold. No more review cycles for optional improvements.

# Adversarial Code Review — Miniapp Studio P1 Specification and Validator

## Role

Act as a principal Go architect and application-security reviewer. Conduct an adversarial, evidence-driven review of the implemented P1 `spec/` package in the Miniapp Studio repository.

Your job is to determine whether P1 is structurally sound enough to become the foundation for P2 (generic runner and renderer).

Do not act as a feature developer. Do not modify files, implement fixes, change dependencies, or commit anything. This is a read-only review.

## 1. Context and established implementation claims

P1 is reported implemented and verified:

- `spec/types.go`: versioned MiniappSpec and closed v1 vocabularies.
- `spec/validate.go`: strict decoding, 13-code error taxonomy, deterministic collection order.
- `spec/hash.go`: canonical JSON and SHA-256 content hash, excluding the hash field itself.
- Fifteen fixtures under `spec/testdata/valid_invoice_review.json` and associated test data.
- Eleven tests, including deliberate rejection of an unapproved send.
- Standard-library-only package; `go list -m all` reports no framework reliance.
- `gofmt`, `go vet`, `go test`, and `go test -race` reportedly pass.

Most importantly, v1 is deliberately read-only:

- No write step type exists.
- Capability method/effect combinations accept only GET/read.
- `policies.confirmation.externalWrites` must equal `disabled`.
- The validator rejects anything outside these constraints.

The schema also establishes `app.owner` as required and includes per-tenant uniqueness rules. Persistence-level tenant scoping is deferred to P2 and later designs.

P0 is not complete. The PostgreSQL-backed framework validation and Studio import spike remain blocked by Docker socket permissions. Docker Postgres is the designated testing path once daemon access is available.

## 2. Review objective

Determine whether the P1 implementation actually enforces its stated invariants, including at hostile-input boundaries.

Do not accept comments, documentation, type names, or happy-path tests as proof. Trace the actual code and tests.

### A. Structural read-only guarantee

Attempt to defeat the claim that v1 cannot represent external writes.

Inspect:

1. Every step type and discriminator.
2. Every capability method and effect vocabulary.
3. All policy and confirmation fields.
4. Optional, omitted, zero-valued, null, and unknown fields.
5. JSON decoding and validation order.
6. Cross-field constraints and interactions between capabilities, workflows, and policies.
7. Any alternate representation that could smuggle an undeclared operation into an otherwise valid specification.

Test adversarial inputs such as:

- POST, PUT, PATCH, DELETE, and other non-GET methods.
- Write-like effect values and unknown effect values.
- Missing or disabled confirmation fields.
- `null`, empty strings, omitted fields, and unexpected nested objects.
- Unknown workflow step discriminators.
- Unknown fields in nested structures.
- Conflicting or contradictory capability and policy declarations.
- Extra JSON documents or trailing data.
- Duplicate JSON object keys and case-variant field names, where applicable to the decoder.

Pay particular attention to Go JSON decoding behavior. Do not assume that `DisallowUnknownFields` alone proves complete strictness.

**Required conclusion:** Is a write genuinely unrepresentable in the v1 data model, or is it merely rejected by the current validator implementation? Identify the precise boundary and any residual limitation.

Do not demand runtime enforcement that cannot exist before P2. Clearly distinguish the P1 schema/validator guarantee from future interpreter enforcement.

### B. Type system and schema integrity

Inspect whether the Go types faithfully implement the intended v1 contract.

Challenge:

- Closed vocabularies and enum handling.
- Required versus optional fields.
- Pointer, zero-value, nil, and default semantics.
- Unknown-version behavior.
- Forward-compatibility assumptions.
- Serialization/deserialization consistency.
- Whether invalid states can be constructed internally without passing through validation.

Determine whether another developer could accidentally bypass the validator through ordinary package usage.

### C. Canonical serialization and content hashing

Independently examine `spec/hash.go`.

Verify:

1. The hash is computed from the intended canonical representation.
2. The hash field is excluded without excluding unrelated data.
3. Semantically identical valid specs produce the same hash.
4. Meaningfully different specs produce different canonical inputs.
5. Hash calculation does not mutate the original spec unexpectedly.
6. Map ordering, optional fields, empty collections, numeric representations, and string handling do not introduce unintended instability.
7. Validation and hash computation cannot disagree about which representation is authoritative.
8. Repeated operations and independent processes produce stable results.

Challenge the canonicalization implementation rather than simply confirming that it calls `json.Marshal` and SHA-256.

Identify which determinism guarantees are actually established and which are merely assumed.

### D. Validator correctness and error taxonomy

Verify the complete validation pipeline:

- Strict decode before semantic validation.
- Version and vocabulary checks.
- Required-field validation.
- Cross-field consistency.
- Capability and policy constraints.
- Collection traversal and deterministic ordering.
- Stable error codes and useful field paths.
- No panic or silent acceptance on malformed input.

Inspect whether multiple invalid fields produce deterministic error ordering and whether error messages leak sensitive content.

Verify that the claimed 13-code taxonomy is actually closed and consistently used.

### E. Tenant ownership foundation

Review the P1 schema and types only; do not pretend persistence isolation is already implemented.

Verify:

- `app.owner` cannot be absent or silently defaulted.
- Ownership is represented unambiguously.
- Per-tenant uniqueness rules include the tenant boundary rather than accidentally imposing global uniqueness.
- No conflicting owner representation can create ambiguity.
- Hashing and versioning preserve ownership semantics.
- The schema provides a sound foundation for later server-derived tenant enforcement.

Record the distinction between schema readiness and actual database/RLS isolation.

### F. Dependency and architectural boundary

Verify the standard-library-only claim independently.

Inspect imports, module dependencies, package references, and build/test dependency closure.

Determine whether `spec/` depends on Miniapp runtime, framework internals, external providers, Fly, Sprite APIs, or other implementation-specific packages.

The P1 specification package must remain independent of the execution framework.

### G. Tests and evidence quality

Inspect every relevant test, not just the test count.

Determine whether tests assert the invariant or merely exercise a code path.

Check for:

- Positive and negative fixtures.
- Every write-like input being rejected.
- Unknown enum and discriminator rejection.
- Malformed JSON handling.
- Duplicate keys and trailing-document behavior where applicable.
- Hash stability and mutation safety.
- Deterministic error ordering.
- Tenant-owner and uniqueness validation.
- Regression coverage for any identified historical failure.

Run the applicable P1 checks if the environment permits:

```
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go list -m all
```

Do not run unrelated infrastructure tests just to make the review appear comprehensive. P1 is standard-library-only; PostgreSQL is not a prerequisite for its unit suite.

Report exactly which commands were executed, their exit status, and any environmental limitation. Do not repeat reported test results as independently verified results.

## 3. Root-cause standard

For every defect:

1. Trace the failure to its actual root cause.
2. Identify the architectural boundary where the invariant should be enforced.
3. Explain the exploit or failure scenario.
4. Specify a durable correction.
5. Name regression tests that would fail before the correction and pass afterward.

Reject superficial fixes that only add another validator check while leaving an invalid state representable or a bypass available elsewhere.

Do not recommend an abstraction or dependency merely because it would make the implementation more elaborate.

## 4. Review boundaries

This review is limited to P1 and its immediate architectural contract.

Do not:

- Implement P2.
- Change the Miniapp framework.
- Change the provisional framework pin.
- Modify the implementation plan or frozen plans.
- Add write support.
- Add managed-agent integrations.
- Introduce DBOS or a workflow engine.
- Redesign multi-tenancy.
- Reopen settled architecture decisions without concrete contradictory evidence.
- Treat P0's outstanding PostgreSQL gate as a P1 defect.

Identify future P2 requirements where relevant, but do not turn them into P1 blockers.

## 5. Approval standard

The owner has explicitly established this principle:

**Approve if there are no foundational or structural blockers.**

Apply it strictly.

A finding is a blocker only when there is evidence of a material defect in a foundational invariant, such as:

- A write operation can be represented or accepted in v1.
- Malformed or adversarial input can bypass required validation.
- Hash identity is unstable, incomplete, or inconsistent with the actual spec.
- Tenant ownership can be omitted or misrepresented.
- The specification package is coupled to framework implementation in violation of its boundary.
- A critical claimed invariant is not actually tested or enforced.

Classify findings as:

- **BLOCKER:** foundational or structural defect. P1 cannot safely become the basis for P2 until corrected.
- **MAJOR:** meaningful weakness, but not a reason to stop P2 if it can be safely contained and tracked.
- **MINOR:** polish, maintainability, or optional improvement.
- **PASS:** invariant verified with code and test evidence.

Do not inflate optional improvements into blockers. Do not demand perfection.

## 6. Required report

Return:

### Executive verdict

`APPROVE P1` or `BLOCK P1`, followed by a concise evidence-based explanation.

### Invariant scorecard

| Invariant | Verdict | Code evidence | Test evidence | Remaining limitation |
| --------- | ------- | ------------- | ------------- | -------------------- |

Cover read-only structure, type/schema integrity, canonical hashing, validator correctness, tenant ownership, dependency isolation, and regression coverage.

### Findings

For each finding provide:

- Severity.
- Exact file and line.
- Reproducible failure or attack scenario.
- Root cause.
- Durable correction.
- Required regression test.
- Whether it blocks P2.

### Verification record

List commands actually executed, results, and limitations.

### P2 handoff

List only the requirements that P2 must preserve when it introduces interpretation and execution. Explicitly preserve the v1 read-only boundary and ensure runtime enforcement cannot weaken it.

### Final decision

State clearly whether implementation should proceed.

If no foundational or structural blockers remain, approve P1 and recommend starting P2 after the separate P0 dependency gate passes. Do not request another general review cycle.

**My recommendation:** run this once. The most valuable adversarial checks are JSON decoder edge cases, write-shaped input bypasses, canonical hash stability, and whether ownership constraints are genuinely tenant-scoped. A clean result means P1 is accepted; P0 remains a separate gate before P2.