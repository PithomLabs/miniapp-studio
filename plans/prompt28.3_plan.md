# prompt28.3_plan — Consolidated: Fly-native lifecycle (28.2) + Neon-first PostgreSQL (28.3)

**Status:** Planning only. No implementation, provisioning, or live operations authorized by this document.
**Amendment (owner decision 2026-11-27):** mandated PG major is 18, not 19 (see ADR-004). All PG19 references in this
plan read as PG18; the Neon-only mandate is unchanged.
**Inputs:** `plans/prompt28.2.md`, `plans/prompt28.3.md`, `plans/prompt28_plan.md` (master plan — amendment specified in the last section),
`plans/docker_vs_sprites.md` and `plans/app_category_guidelines.md` (locked context, unchanged).
**Program context:** prompt28 B2-demo work is mid-flight (Phase 0/0B complete; local reference + Sprite phases pending).
This plan reconciles the new lifecycle + Neon mandates with that in-flight work before any new infrastructure is built.

## Governing product goal

One of Miniapp Studio's ultimate goals is to **simplify ops complexity for developers**. That goal binds this plan:

- Every lifecycle decision (local setup, CI, previews, agents, database provisioning) is evaluated on the
  **developer ops burden** it creates, alongside isolation, cost, and recoverability.
- Complexity belongs in the platform (controllers, reconciliation, automation), not in developer workflows.
  One-command flows (`task dev:setup`, canonical `studio dev` verbs) are preserved as UX even as backends change.
- Each ADR scores alternatives on developer simplicity explicitly; a design that is "more correct" but pushes
  manual provisioning, credential juggling, or environment-specific rituals onto developers loses to a simpler one
  with equivalent safety properties.

## Goal

Reconcile prompt28.2 (Fly-native CI, per-PR previews, background agents) and prompt28.3 (mandatory Neon-first
PostgreSQL) into one coherent program: produce the requirement map, resolve conflicts with frozen/in-flight decisions
by explicit supersession, record ADRs, update the authoritative docs and master plan, verify every external capability
claim, and define gated implementation tracks with live acceptance proofs. Near-term executable work is the
reconciliation-and-design pass (Stage 1); platform implementation (Stage 2) starts only through its entry gates.

## Success Criteria

- Every prompt28.2 (§3–§13) and prompt28.3 (§1–§10) requirement is mapped to existing implementation, missing
  implementation, owning ADR, stage, acceptance test, and required evidence. Nothing is silently dropped.
- All conflicts with frozen or in-flight decisions are resolved by explicit supersession records — never by
  dual-satisfying incompatible requirements (per prompt28.3 §2).
- One deduplicated ADR set covers the prompt28.2 §11 topics and prompt28.3 §10 topics, each with context,
  alternatives, decision, security/ops/cost implications, failure modes, verification strategy, and
  developer-simplicity scoring.
- Authoritative docs are amended (not duplicated): `docs/architecture/` lifecycle/threat-model/cost docs,
  `docs/pg_muse.md`, Taskfile docs, parity report, and `plans/prompt28_plan.md`.
- Every external capability claim relied upon (Fly and Neon) is verified against current official sources or marked
  conditional with a verification step; no design rests on an unverified platform capability.
- Stage 2 tracks are defined with entry gates, work units, and live acceptance proofs; the demo-vs-platform split is
  decided at the Stage-1 exit gate on evidence, not on assumptions.

## Approach

- **Reconciliation before implementation** (prompt28.2 §13 and its §8 recommendation). The Neon mandate contradicts
  in-flight native-PG work and frozen ADR-002; building platform infrastructure before reconciling guarantees rework.
- **Neon-first is binding; conflicts resolve by supersession.** Where prompt28.2 or the master plan assumed
  self-hosted/digest-pinned PostgreSQL, prompt28.3 supersedes those elements explicitly (Conflict inventory below). Compatible
  decisions (Go ownership, Taskfile orchestration, three-lifecycle separation, Sprite isolation, GitHub review) carry over.
- **Three lifecycles stay separate** (agent execution / CI / preview) with explicitly named contracts between them;
  controller topology (one service vs modules) is an ADR decision, not a premise.
- **Stage 1 is fully specified and immediately executable. Stage 2 is defined at phase granularity here** — its
  detailed execution plans are drafted from Stage-1 evidence, not invented before the reconciliation exists.
- **Existing evidence doctrine carries over:** live proofs, per-operation authorizations for consequential steps,
  fail-closed gates, exact counts, no mock-green, secrets never in output.

## Key Decisions

- **D1 — Reconciliation first.** Stage 1 (map → conflicts → ADRs → docs → gate) precedes all new infrastructure.
  Rejected: starting CI/preview/agent builds now (would entrench the contradicted PG assumptions).
- **D2 — Explicit supersession, single-stack outcome.** The program ends on Neon-only PostgreSQL; superseded
  native-PG/Docker-image designs are recorded as superseded, not maintained in parallel.
- **D3 — Developer simplicity is a scored ADR criterion** (governing goal above), alongside security, cost,
  operability, and recoverability.
- **D4 — Demo-vs-platform split decided at the Stage-1 exit gate** with cost/effort evidence (pivot the demo stack
  to Neon now vs parallel track). Clarification (owner, binding): demo and platform may differ in scope, sequencing,
  and feature completeness, but BOTH respect the mandatory Neon constraint — native and Docker PostgreSQL remain
  superseded approaches, never alternative implementation tracks. Asking before the reconciliation exists would decide without facts.
- **D5 — No new command/orchestration layer.** Taskfiles orchestrate; Go implements; `studio dev` stays canonical.
  `Taskfile.prod.yml` is created only if prod-scoped tasks materialize (it does not exist today — verified).
- **D6 — Stage-2 detail deferred honestly.** This plan names Stage-2 tracks, gates, and proofs; per-track execution
  plans follow the reconciliation. This is scoping discipline, not vagueness.

## Conflict inventory (verified this planning pass)

| # | Conflict | Status in this plan |
|---|----------|---------------------|
| C1 | ADR-002 decides Fly Managed Postgres PG17 with Neon as fallback; prompt28.3 mandates Neon-only PG19. | Requires a superseding ADR (Stage 1, S1.4). |
| C2 | Master-plan Phases 1/3/4 build native PG19 (local reference, Sprite loopback install, dual-env proof); prompt28.3 forbids self-hosted PG in every environment. | Superseded elements listed in the Master-plan integration table; formal revision in Stage 1. |
| C3 | Master-plan Phase 4B pins a digest-verified PG19 CI image; prompt28.3 replaces it with Neon CI branches + Neon-specific verification. | Superseded for the DB layer (Go/toolchain pins unaffected). |
| C4 | `testdb`/`dbenv`/`studio dev`/Taskfiles/fingerprint assume loopback `127.0.0.1:5434` + local trust; Neon means TLS remote endpoints, branch-scoped credentials, no loopback. | Deepest technical conflict: endpoint/security model redesign owned by Stage 1 ADRs. |
| C5 | `docs/pg_muse.md` Docker-authority/local-env sections vs no-Docker + Neon-only direction. | Accuracy rewrite scoped in Stage 1; never left stale. |
| C6 | Demo deadline vs full platform scope (12 + 10 ADR topics, controllers, previews, agents). | Resolved by the Stage-1 exit gate (D4), not by silent scope creep. |
| C7 | prompt28.2 assumes existing `Taskfile.dev.yml` + `Taskfile.prod.yml`; only dev exists. | D5: extend dev; create prod only on demonstrated need. |
| C8 | Customer-miniapp database story (project-per-miniapp vs shared) undecided; affects platform control plane. | ADR topic; must not default to single-shared-project. |

## Existing-implementation map (verified, not assumed)

- **Present:** `fly/` sprites adapter + Connectors REST client + lifecycle contract; `gateway/` OpenCode adapter path;
  `connector/registry.go` (metadata only, no credentials); server approval surface (`server/artifact_decide.go`,
  console handlers); `testdb`/`dbenv` + `studio dev` CLI + `Taskfile.dev.yml` (built this session, loopback-shaped);
  migration chains through 000025; `decisions/001–003`; `ci.yml` (GitHub Actions, digest-pinned PG service).
- **Absent:** any webhook receiver, GitHub App token minting, Neon client/code, preview provisioning, CI/Sprite
  execution controller, `Taskfile.prod.yml`, `docs/dev-parity-report.md`, `closed`-type PR handling.
- **External references inspected:** both Fly articles fetched and read (background-agents article in full; previews
  article substantially — teardown/cost tail truncated in transit, full re-read required in S1.1). Relied-upon mechanics
  (exec-timeout/marker polling, warm checkpoints, token narrowing, `closed`-type requirement, per-PR concurrency
  groups) are carried as to-verify items, not facts.

## Steps — Stage 1: reconciliation and design (immediately executable)

No infrastructure is built in Stage 1. No provisioning, no live Sprite/Fly/Neon/GitHub mutations, no commits
without authorization. Read-only inspection plus design documents only.

### S1.1 — Fly lifecycle reference verification

- Re-read both Fly articles in full; extract every platform capability the design will rely on.
- Verify each against current official sources (Sprites CLI/API docs: exec timeouts, background execution, checkpoints,
  network policy, sleep/wake; Connectors: GitHub connector path scoping; Machines API; GitHub docs: App token
  narrowing parameters, webhook signature scheme, `pull_request` event types).
- Deliverable: verified-vs-assumed capability table. Anything unverifiable stays conditional with a named proof step.

### S1.2 — Neon claim verification

- Verify every prompt28.3-cited claim: PG19 availability; branching semantics and copy-on-write behavior; role/password
  inheritance on branch; branch protection availability and plan gating; project-scoped API keys; branch lifecycle
  API/CLI + expiration semantics; compute scale-to-zero/resume latency; quotas, branch/project limits, pricing;
  branch-aware object storage (evaluated separately — adoption needs its own justification, not the PG mandate).
- Deliverable: same verified-vs-assumed treatment as S1.1. Credential-inheritance behavior must have an executable
  proof plan (S1.2 defines it; Stage 2 runs it) before any untrusted-workload design is finalized.

### S1.3 — Requirement map

- Map every prompt28.2 (§3–§13) and prompt28.3 (§1–§10) requirement to: existing implementation / missing
  implementation / owning ADR / stage / acceptance test / required evidence.
- Deliverable: `REQ-###` matrix document. Gaps and double-coverage (28.2 vs 28.3) are called out, not smoothed over.

### S1.4 — Conflict resolution records

- Resolve C1–C8 with explicit supersession records: ADR-002 successor; master-plan phase revisions (the amendment in the last section becomes the ratified revision); endpoint/security-model redesign for `testdb`/`dbenv`/`studio dev`/Taskfiles/CI (C4 — expect
  this to be the largest design item: Neon endpoints, TLS, branch credentials, revised allowlist doctrine, revised
  fingerprint fields); `docs/pg_muse.md` rewrite scope; Taskfile.prod decision; miniapp-DB framing.
- Rule: superseded text is marked superseded with a pointer, never silently deleted from history.

### S1.5 — Architecture decision records

- Merge the prompt28.2 §11 list (12 topics) and prompt28.3 §10 list (10 topics) into one deduplicated, numbered ADR
  set under `docs/architecture/decisions/`.
- Each ADR carries the §11-required fields (context, requirements, alternatives, choice, rejections + rationale,
  security/ops/cost implications, failure modes, verification, deferred items) **plus developer-simplicity scoring**
  per the governing goal.
- Controller topology, Sprite/checkpoint strategy, Neon project/branch hierarchy, credential rotation, preview
  generation strategy (persistent branch vs per-commit), and cleanup-as-reconciliation are decided here.

### S1.6 — Documentation updates

- Amend (never duplicate): lifecycle architecture + component/trust-boundary diagrams, three-lifecycle sequence
  diagrams, controller design, preview + agent architectures, state machines, threat model, cost/capacity model;
  `docs/pg_muse.md` accuracy pass; Taskfile docs; parity report (create/update `docs/dev-parity-report.md`);
  `plans/prompt28_plan.md` (ratify the Master-plan integration amendment).
- Remove stale local-PG/Docker-image provisioning references as the supersession records direct.

### S1.7 — Demo/platform split + Stage-1 exit gate

- Produce the split proposal: what the B2 demo needs (minimal, date-driven) vs platform roadmap, with effort/cost
  evidence for pivot-now vs parallel-track.
- **Gate:** owner decision required before any Stage-2 work. Planning approval is not implementation approval.

### S1.8 — Stage-2 execution plans

- Draft per-track execution plans from Stage-1 evidence (entry criteria, work units, proofs). These are separate
  reviewable documents, not improvisation licenses.

## Steps — Stage 2: gated implementation tracks (defined here, detailed in S1.8)

Common entry: Stage-1 exit gate passed; owning ADRs accepted; track execution plan reviewed. Common rules: small
independently verifiable increments; live proofs per prompt28.2 §13 + prompt28.3 §10; per-operation authorization
for consequential steps; fail-closed on surprise.

- **T1 — Local Neon development flow.** `task dev:setup` → Neon CLI/API preflight, auth, project validation,
  developer-branch reconcile, credential provisioning, env config, canonical `studio dev` verification, connectivity +
  engine-version proof. No local PG server dependency. Acceptance: clean-host setup, idempotent re-run, credential
  preservation, PG19 proof.
- **T2 — Fly-native CI.** Controller on Fly Machine (webhook verify → dedupe → persist → schedule → Sprite execution
  → checks) + ephemeral Neon CI branch per run (create → migrate → test → collect → delete → reconcile). Acceptance:
  exact-SHA runs, canonical Taskfile targets, GitHub checks, no false-green on failure/cancel, orphan reconciliation.
- **T3 — Per-PR previews.** Isolated Fly app + Neon branch per PR; serialized per-PR deployment with SHA/generation
  fencing; migration + fixture + smoke sequence; URL/health reporting; close-triggered + reconciled destruction;
  per-PR object-storage/external-service isolation; cost controls. Acceptance: the full prompt28.2 §13 preview matrix.
- **T4 — Background coding agents.** Labeled-issue → verified webhook → isolated Sprite → headless agent → validation
  → branch → PR (draft-on-failure policy); scoped short-lived GitHub App credentials; checkpoint strategy; spend and
  concurrency caps; restricted egress. Acceptance: the §13 agent matrix, incl. no-push-to-main and no self-approval.
- **T5 — Customer-miniapp provisioning.** Neon project/database allocation per the accepted ADR; lifecycle tied to
  app lifecycle; billing/quota/recovery story. Acceptance: provisioned miniapps are Neon-backed with isolation proofs.
- **Cross-cutting:** Taskfile surface for all of the above (orchestration only); secret-management selection and
  credential-domain separation (Neon API keys vs DB roles vs app secrets vs agent tokens); cost attribution/alerts.

## Validation Plan

- **S1.1/S1.2:** verified-vs-assumed tables reviewed claim-by-claim; every "verified" cell cites the inspected source
  (URL + checked date); every "assumed" cell names its proof step and owning stage.
- **S1.3:** matrix audit — each prompt28.2/28.3 section number appears at least once; each row has all six columns;
  spot-check rows against the cited existing code paths.
- **S1.4:** each C1–C8 record states old decision, superseding decision, rationale, and affected files; no
  dual-satisfying language survives review.
- **S1.5:** ADR checklist per record (all §11 fields + simplicity score + verification strategy); ADR numbers
  sequential under `docs/architecture/decisions/`.
- **S1.6:** doc diffs reviewed; no stale local-PG/Docker-image provisioning references remain except in explicitly
  marked superseded-history blocks.
- **S1.7:** split proposal carries effort/cost evidence; gate decision recorded with date and decider.
- **Stage 2:** per-track live proofs per prompt28.2 §13 and prompt28.3 §10, evidenced live (receipts, URLs, branch IDs,
 SHAs, counts) — never mock-green, never skip-green.
- **Highest-risk validations:** the C4 endpoint/security-model redesign (it reshapes the just-built dev contract) and
  the Neon credential-inheritance executable proof (untrusted-workload safety rests on it).

## Risks / Open Questions

- **R1 — Neon account/plan/cost unknowns.** Plan selection, branch/project limits, and concurrent preview+CI capacity
  are unverified until S1.2; quota/cost findings may force topology changes. No Neon provisioning happens in Stage 1.
- **R2 — Sprite execution unknowns.** Exec semantics, checkpoint support, and network-policy mechanics are carried as
  to-verify (S1.1); the T2/T4 designs stay conditional until proven.
- **R3 — GitHub App + secrets-backend ownership.** App registration, installation scoping, and the approved secret store
  are owner decisions with per-operation authorizations; Stage 1 frames them, it does not execute them.
- **R4 — C4 blast radius.** The endpoint-model redesign touches `testdb`/`dbenv`/`studio dev`/Taskfiles/CI/fingerprint —
  including work built this session. S1.4 must size this before any demo-stack commitment.
- **R5 — Demo timeline pressure.** The full platform (T1–T5) is not a demo prerequisite; S1.7 exists to protect the
  date. Scope creep into Stage 2 without the gate decision is the failure mode to watch.
- **R6 — Fixture/PII strategy.** Synthetic-vs-anonymized parent-branch data needs its ADR before any branch is derived
  from sensitive data; default is synthetic/schema-only.
- **Non-goals (carried):** HostExec install/operation, B3/B4/B5 beyond T5's provisioning ADR, classification/admission
  machinery, DBOS/River substrates, production data near tests, any self-hosted PG as a convenience workaround.
- **Open questions:** None blocking Stage 1. Remaining unknowns are scheduled as verification steps with fail-closed
  outcomes; the demo-stack decision is structured as the S1.7 gate, not a question.

## Sources

- `https://www.fly.io/learn/per-pr-preview-environments-for-coding-agents` (inspected 2026-11-27; teardown/cost tail
  truncated in transit — full re-read required in S1.1).
- `https://www.fly.io/learn/background-coding-agents-pr-workflow` (inspected in full 2026-11-27).

## Master-plan integration — amendment to the plan above (`plans/prompt28_plan.md`)

The master plan is amended by appending the block quoted beneath the table (applied once, in this planning turn). It marks
superseded elements by pointer; formal phase revisions are ratified in S1.4. In-flight completed work (Phase 0 audit,
Phase 0B interface) is preserved as history regardless of outcome.

| Master-plan element | Disposition under 28.2 + 28.3 |
|---|---|
| Goal / Success Criteria (local PG19, native Sprite PG, digest-pinned CI) | Superseded in mechanism; re-expressed on Neon in S1.4 (same proof discipline). |
| Approach: dual-env loopback, digest CI, "no testdb changes" | Under revision via C4; the endpoint invariant is redesigned, not silently kept. |
| Phase 1 local reference (native/Docker PG19) | Replaced by T1 Neon dev flow. |
| Phase 2 Sprite inspection (read-only) | Retained — facts still needed for T2/T4 Sprite-execution design. |
| Phase 3 remote PG install | Superseded — no PostgreSQL inside Sprites, ever. |
| Phase 4 dual-env native proof | Replaced by Neon-based parity (dev branch vs CI branch). |
| Phase 4B digest-CI proof | Replaced by T2 Fly-native CI. |
| Phases 5–7 B2 proofs, demo slices, rehearsal | Retained in intent; execution stack decided at the S1.7 gate. |
| R2 (Sprite PG installability), R6 (digest-CI push) | Retired with the phases they served; R1/R3–R5/R7 carry over. |

Quoted amendment block appended to `plans/prompt28_plan.md`:

> ## Amendment P28.2/P28.3 — Fly-native lifecycle + Neon-first PostgreSQL (planned, not yet executed)
>
> `plans/prompt28.2.md` and `plans/prompt28.3.md` are consolidated in `plans/prompt28.3_plan.md`, which governs
> all lifecycle + database-platform work from here. Neon-first PostgreSQL is binding across every environment.
> The following master-plan elements are SUPERSEDED in mechanism (formal phase revisions ratified in Stage 1,
> step S1.4): local/native PG19 reference work, Sprite PostgreSQL install, dual-environment native proof,
> digest-pinned CI image. Sprite read-only inspection, B2 live-proof intent, demo slices, and rehearsal are
> RETAINED; their execution stack is decided at the Stage-1 exit gate. Completed Phase 0/0B work stands as
> history. No Stage-2 implementation starts before that gate.
