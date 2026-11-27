# Stage-1 exit proposal (S1.7)

**Date:** 2026-11-27. **Stage 1 status:** complete — all S1.1–S1.8 deliverables delivered for review.
**Decision requested:** (1) accept ADR-004–018 (004 already decided by owner mandate; 005–018 proposed);
(2) choose the demo/platform sequencing below; (3) authorize Stage 2 entry per the gate checklist.

## What Stage 1 established (evidence-backed)

- 33 capability claims reviewed and classified against official sources (18 Fly/GitHub, 15 Neon — verified, attested, assumed, disputed, and unverified distinguished in the table); 1 disputed claim removed from the
  design (`sprite exec` timeout — sessions model adopted instead); 4 items explicitly conditional with named proofs
  (password inheritance N8, branch protection N9, pricing/quotas V13/N14, wake mechanics V15).
- 61 requirements traced (R201–R237, R301–R324) with existing/missing/ADR/stage/acceptance columns; double-coverage
  resolved with 28.3 winning each mechanism conflict.
- 8 conflicts resolved by explicit supersession (C1–C8); deepest conflict C4 owned by ADR-006 with a sized redesign
  (endpoint policy, TLS, branch roles, fingerprint, verbs, prohibition audit).
- 15 ADRs (004–018) covering all 22 source topics; each with verification strategy + developer-simplicity score.
- Docs amended without duplication: lifecycle architecture + diagrams + state machines, threat model (13 vectors),
  parameterized cost model, parity-report skeleton, `pg_muse.md` + `dev-contract.md` accuracy passes, ADR-002
  supersession, master-plan formal revision.

## Demo/platform sequencing (D4 as clarified — both options Neon-compliant)

- **Option A — Demo-first minimal slice (recommended):** T1 dev-branch flow → minimal B2 demo on Neon (agent run,
  connector read, approval boundary, evidence) → T2/T3/T4/T5 as roadmap tracks. Smallest path to a credible,
  constraint-honest demo; platform tracks keep independent entry gates.
- **Option B — Platform depth first:** T1 → T2 (CI) → T3 (previews) → demo on the fuller platform. Stronger story,
  later date, larger pre-demo live surface.
- Ruled out: any sequencing that retains native/Docker PostgreSQL as an implementation track (owner clarification).

## Stage-2 entry gate checklist (track-specific per review2)

- [ ] ADR-005–018 formally accepted — recorded separately (review2: full ADR texts not individually reviewed at exit)
- [x] Sequencing chosen: Option A — demo-first, minimal Neon-compliant slice (review2, 2026-11-27)
- [ ] T1 prerequisites: Neon non-prod project provisioned by owner; secret-store product chosen; ADR-004/005/006/010 accepted; T1 work units reviewed
- [ ] T2/T4 prerequisites (later, not T1 gates): GitHub App installed with minimal permissions; controller credential ceremony performed
- [ ] T3 prerequisites (later, not T1 gates): N14 live quota/concurrency verification; preview provisioning authority bounded
- [ ] Per-operation authorizations structured per track (provision/run/destroy each gated)

No Stage-2 work starts before this gate. Planning approval is not implementation approval.
