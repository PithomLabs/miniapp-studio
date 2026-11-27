# ADR-015: Test and preview data strategy

**Status:** Proposed 2026-11-27 (Stage 1; acceptance at Stage-1 exit gate).
**Decision:** Default lineage is synthetic fixtures or schema-only branches; anonymized production derivatives only
under an explicit data-handling approval with a documented anonymization procedure. No untrusted branch (agent, CI,
preview) is ever derived from sensitive production data. Sequence-copy ID behavior (N3) is accounted for in
cross-branch comparisons.
**Basis:** prompt28.3 §3 + §7; verified branching/anonymization facts (N3–N6; N2 notes PG19 anonymization unavailable).

## Context

Branches make production-data copies trivially easy — which is exactly why the default must make them hard. PG19
currently lacks Neon's data-anonymization support (N2), so anonymized derivatives are unavailable on the mandated
version until verified otherwise.

## Requirements

R218 (deterministic non-sensitive fixtures), R305 (lineage rule), R317 (no prod reachability from untrusted workloads).

## Considered alternatives

- **Production-data branches for "realistic" previews:** rejected — violates the untrusted-workload boundary; one
  leaked preview credential becomes a data breach.
- **Anonymized derivatives as default:** rejected for now — PG19 anonymization support unverified/absent (N2);
  available only after support lands + procedure approved. Recorded as a future upgrade path, not today's default.
- **Synthetic/schema-only default (chosen):** deterministic fixtures checked into the repo; schema-only branches where
  shape suffices. Reproducible, reviewable, breach-proof by construction.

## Security / operational / cost implications

Fixture determinism doubles as reproducibility proof for CI (replaces part of what image digests used to guarantee).
Anonymization, if later adopted, needs its own verification (re-identification review), not just tool availability.

## Failure modes

Fixture drift from production shape (mitigation: schema-conformance checks in CI); accidental sensitive data in
fixtures (mitigation: fixture review + secret scanning on fixture files); future anonymization falsely assumed
sufficient (mitigation: re-identification review gate before any adoption).

## Verification

LIVE: lineage audit (every untrusted branch traces to synthetic/schema-only parent); fixture determinism check
(same fixtures → same post-migration state hash); negative test (sensitive-data detection scan on fixture set).

## Deferred

Anonymized-derivative adoption (blocked on PG19 support + procedure + review); fixture-scaling strategy for
large-shape testing.

## Developer-simplicity score

Positive: deterministic fixtures mean previews and CI behave identically every time — no "works on my branch"
mysteries. Developers extend fixtures with normal code review instead of managing data extracts.
