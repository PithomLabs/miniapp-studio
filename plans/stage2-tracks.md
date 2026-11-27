# Stage-2 track execution plans (S1.8)

One plan per track: entry criteria, work units, acceptance proofs, authorizations. All tracks share the common rules:
small verifiable increments; live proofs only; per-operation GOs for consequential steps; fail closed on surprise.
Tracks are independent once their entry criteria hold; recommended order is T1 → T2 → T3 → T4 → T5 (demo slice may
interleave after T1 per the S1.7 decision).

## T1 — Local Neon development flow

- **Entry:** ADR-004/005/006/010 accepted; non-prod Neon project exists; secret-store chosen.
- **Work units:** (1) C4 redesign: endpoint policy + TLS + branch roles in `testdb`/`dbenv`; (2) `studio dev` verbs
  revised (doctor/init/test/fingerprint with R312 fields); (3) `task dev:setup` Neon flow (R307); (4) R302 prohibition
  audit + removal of loopback-PG paths; (5) `pg_muse.md` procedure rewrite; (6) extension re-proof on PG19 (N2).
- **Proofs:** clean-host setup; idempotent re-run; credential preservation; PG19 actual-version proof; dev-branch
  suite green twice; parity-report dev column filled.
- **Authorizations:** Neon project use; dev-branch credential issuance; each live verification run.

## T2 — Fly-native CI

- **Entry:** T1 green; ADR-007/008/012/013/017 accepted; controller host (Fly Machine) designated; GitHub App installed.
- **Work units:** (1) Controller: webhook verify/dedupe/persist/schedule (V2/V4/V5/V17); (2) Sprite supervision via
  sessions+markers (V6–V9); (3) per-run Neon branch lifecycle + SHA association (R309–R310); (4) checks publishing +
  required-check wiring (R211–R212); (5) reconciler + caps (ADR-013); (6) `ci.yml` replacement (no parallel system).
- **Proofs:** full §13 CI matrix; N8 inheritance proof; failure/cancel honesty; orphan reconciliation; restart recovery;
  parity-report CI column filled; merge-blocked negative test.
- **Authorizations:** controller deploy; App install + permission inventory; each live run at first, then batched
  run-windows per owner direction.

## T3 — Per-PR previews

- **Entry:** T2 green; ADR-009/013/015/018 accepted; N14 quotas verified sufficient for preview concurrency.
- **Work units:** (1) PR-branch lifecycle + association records (R313); (2) persistent-branch-per-PR migration
  discipline + fencing (R314; fallback to per-commit generations on evidence); (3) Fly app deploy + exclusive config +
  smoke sequence (R218); (4) exposure policy + read-only agent diagnostics (R220/R222); (5) close-triggered +
  reconciled destruction (R315); (6) storage/integration isolation incl. Neon-storage evaluation (ADR-018).
- **Proofs:** full §13 preview matrix; two-PR isolation; stale-job rejection; exclusivity audit; orphan-GC proof.
- **Authorizations:** preview App provisioning authority (bounded); per-PR live proofs at first; destruction authority
  scoped by ownership metadata.

## T4 — Background coding agents

- **Entry:** T2 green (validation path exists); ADR-008/011/014 accepted; model-key ceremony performed with budgets.
- **Work units:** (1) Label-trigger + eligibility pipeline (R223); (2) per-run Sprite + policy + narrowed token
  (R224–R225); (3) headless-agent workflow + validation gate + draft-on-failure (R226–R227); (4) supervision
  (sessions/markers/heartbeats/timeouts/cancel) (R228); (5) adapters + budgets + egress (R229); (6) V15 wake proof
  for same-PR continuation.
- **Proofs:** full §13 agent matrix; containment negatives; no-main-push; failure honesty; E2E PR creation; budget
  enforcement.
- **Authorizations:** trigger-label activation; per-run GOs at first; model-spend budget approval.

## T5 — Customer-miniapp provisioning

- **Entry:** ADR-016 accepted; N14 project-quota headroom verified; T1 patterns reusable.
- **Work units:** (1) Provisioning automation implementing the allocation strategy established by accepted ADR-016 (current proposal: project-per-miniapp default; the track decides nothing prematurely); (2) lifecycle binding (migrate/operate/backup/
  delete-export); (3) per-miniapp billing/quota/observability; (4) reconciler extension to miniapp resources.
- **Proofs:** isolation inventory audit; delete-path proof; quota-headroom check.
- **Authorizations:** miniapp-project provisioning authority; billing-account scope confirmation.

## Cross-cutting (all tracks)

Taskfile surface per R320/R233 (orchestration only; `Taskfile.prod.yml` iff prod tasks materialize); secret-store
integration per ADR-010; cost attribution + alerts per ADR-013; threat-model traceability (each T-proof maps to T1–T13).
