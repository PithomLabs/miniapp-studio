# Stage 1 — Conflict resolution records (S1.4)

**Rule:** superseded text is marked superseded with a pointer, never silently deleted. No dual-satisfying language.
Each record: old decision → superseding decision → rationale → affected files → ratification step.

## C1 — Platform database: MPG PG17 vs Neon-only PG19

- **Old:** ADR-002 — Fly Managed Postgres, Developer plan, PG17, same region; Neon as fallback.
- **Superseding:** ADR-004 — Neon is the mandatory PostgreSQL provider, PG19, across every environment (R301/R302/R324).
- **Rationale:** Owner hard constraint (prompt28.3). Neon PG19 verified same-day available (N1); MPG cannot satisfy
  branch-per-PR/CI-run lifecycle economics or the single-provider mandate.
- **Affected:** `docs/architecture/decisions/002-postgres.md` (marked superseded by ADR-004), `docs/p3-evidence.md`
  MPG sections (marked historical), any MPG connection code/flags (inventory in S1.8 T-track plans).
- **Ratification:** ADR-004 published in S1.5; ADR-002 header amended in S1.6.

## C2 — Master-plan native-PG phases vs Neon-only constraint

- **Old:** prompt28_plan Phases 1 (local PG19 reference), 3 (Sprite PG install), 4 (dual-env native proof).
- **Superseding:** T1 Neon dev flow + T2 Neon CI execution (R307/R309–R312); Phase 2 read-only Sprite inspection retained.
- **Rationale:** Self-hosted PG in any environment violates R302. Sprite inspection facts are still required for T2/T4.
- **Affected:** `plans/prompt28_plan.md` Phases 1/3/4, Success Criteria, Approach, Validation Plan, R2.
- **Ratification:** Master-plan amendment already appended; formal phase revisions applied in S1.6.

## C3 — Digest-pinned CI image vs Neon CI branches

- **Old:** Phase 4B — digest-pinned PG19 service image + `127.0.0.1:5434` mapping in `ci.yml`.
- **Superseding:** T2 — ephemeral Neon CI branch per run + Neon-specific verification (R312); Go/toolchain pins unaffected.
- **Rationale:** Image-level reproducibility is unclaimable for managed infrastructure (prompt28.3 §5); branch identity +
  actual-version proof replaces digest proof. Reproducibility duty moves to migration determinism + fixture determinism.
- **Affected:** `plans/prompt28_plan.md` Phase 4B, `.github/workflows/ci.yml` (rewritten in T2), fingerprint fields.
- **Ratification:** ADR-004/007 in S1.5; Phase 4B revision in S1.6.

## C4 — Loopback endpoint/security model vs Neon endpoints (deepest conflict)

- **Old:** `testdb`/`dbenv`/`studio dev`/Taskfiles/fingerprint assume loopback `127.0.0.1:5434`, local trust, fixed test
  credential, run-pattern isolation; endpoint allowlist forbids anything else.
- **Superseding:** ADR-006 — Neon endpoint model: TLS remote endpoints, per-branch credentials, branch-scoped roles,
  redesigned endpoint policy (Neon project/branch allowlisting replaces loopback allowlisting), revised fingerprint
  (R312 fields), revised `studio dev` verbs. Migration path: loopback model retired in code, not adapted by exception.
- **Rationale:** Neon endpoints cannot be loopback; adapting by allowlist exception would recreate the exact
  silent-substitution hazard the endpoint invariant exists to prevent. The invariant is re-expressed (known Neon
  project + known branch + TLS + least-privilege role), not weakened.
- **Affected:** `testdb/`, `dbenv/`, `cmd/studio` dev verbs, `Taskfile.dev.yml`, fingerprint, `ci.yml`, parity report.
- **Ratification:** ADR-006 in S1.5 (largest design item); implementation sized in S1.8 T1/T2 plans before any demo commitment.

## C5 — `docs/pg_muse.md` Docker/local-PG authority vs no-Docker + Neon-only

- **Old:** Docker authority, local-env, and runner-design sections assume Docker PG + loopback as the operating reality.
- **Superseding:** Accuracy rewrite: Neon-first operating guide; Docker-PG/runner sections marked superseded-history
  with pointers; secrets discipline and owner-terminal mechanism preserved.
- **Rationale:** Stale operating docs cause live misoperation. History preserved for audit; procedure replaced.
- **Affected:** `docs/pg_muse.md` (§2.3, §3, §5, §7 or as found).
- **Ratification:** Amendment pass in S1.6; full procedure rewrite owned by T1 execution plan.

## C6 — Demo deadline vs full platform scope

- **Old (tension):** B2 demo date vs 22 ADR topics + controllers + previews + agents.
- **Resolving:** D4-as-clarified — S1.7 exit gate splits demo scope from platform roadmap; BOTH Neon-compliant; native/Docker
  PG are not alternative tracks. T1–T5 are roadmap tracks with independent entry gates; the demo takes the minimal
  Neon-compliant slice.
- **Rationale:** Owner clarification (binding). Scope creep into Stage 2 without the gate decision is the named failure mode (R5).
- **Affected:** `plans/stage1-exit-proposal.md` (S1.7), `plans/stage2-tracks.md` (S1.8).
- **Ratification:** Owner exit decision after Stage 1 delivery.

## C7 — Taskfile.prod.yml assumed vs absent

- **Old (assumption in 28.2):** existing `Taskfile.dev.yml` + `Taskfile.prod.yml`, both extended.
- **Resolving:** D5 — extend `Taskfile.dev.yml`; create `Taskfile.prod.yml` only when prod-scoped tasks materialize
  (T3/T5 design may produce them; absence until then is not a gap).
- **Rationale:** No parallel orchestration layer; no empty placeholder files.
- **Affected:** `Taskfile.dev.yml` (extended in T1–T3); `Taskfile.prod.yml` (iff needed).
- **Ratification:** Task inventory in S1.8 track plans names which tasks land where.

## C8 — Customer-miniapp database story undecided

- **Old:** Undecided (prompt28.2 scope-excluded miniapps; prompt28.3 §3 demands the decision).
- **Resolving:** ADR-016 decides project-per-x with cost/isolation/quota/lifecycle/observability/recovery evaluation;
  single-shared-project is not the default. T5 implements the ADR.
- **Rationale:** The decision shapes the platform control plane; defaulting without evaluation mortgages multi-tenancy.
- **Affected:** New ADR-016; T5 execution plan; future B3+ work.
- **Ratification:** ADR-016 in S1.5; T5 entry gate requires its acceptance.
