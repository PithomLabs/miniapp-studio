# prompt28_plan — Execute: unblock DB testing, deliver the B2 demo

## Goal

Execute `plans/prompt28.md` plus amendments `plans/prompt28.1.md`–`plans/prompt28.3.md` as consolidated in `plans/prompt28.3_plan.md`: establish the Neon-first development contract (no self-hosted PostgreSQL in any environment), prove it in dev-branch and Fly-native CI execution via one version-controlled interface, then deliver working B2 demo evidence (resident agent, genuine GitHub Connector, Studio authorization boundary, evidence/recovery, Workspace Preview if core is green). Deadline-driven, single active slice, fail-closed on missing authorization.

## Success Criteria

- Neon dev-branch reference verified (project/branch identity, actual PG19 version, roles, conventions) and the DB-dependent `go test` suite green against it with exact counts.
- Builder Sprite inspected read-only (distro, permissions, PG packages, service manager, Go toolchain, module/network path, checkpoint support); findings recorded, no writes made.
- Zero self-hosted PostgreSQL anywhere (R302 audit clean); Sprite execution connects to Neon branches over TLS — no PG in Sprite, ever.
- The same DB suite green from Sprite execution against its Neon branch with exact counts; tenant/RLS tests green in both environments; branch isolation and cleanup verified.
- B2-01B live proofs recorded: Tasks hold, attach/detach, pause/wake observation, receipt persistence, recovery/reconciliation.
- Demo slices evidenced live: real agent round-trip, genuine Connector read, one approved write plus one refused write, evidence visible in Studio; preview only if core is green.
- Every VERIFIED checklist item has reproducible evidence; nothing mocked, skipped, or inaccessible is reported green.
- A version-controlled canonical dev interface exists and all three environments run the same commands through it; clean bootstrap is demonstrated (not assumed) locally and remotely with repeat runs.
- Fly-native CI reproduces the DB suite from the same commit against an ephemeral Neon branch with zero unexplained skips; the version-controlled parity report records dev + CI (+ preview when T3 lands) with explicit deviations.

## Approach

- One active slice at a time; hard gates between phases; stop and report precise blockers instead of provisioning or improvising.
- Neon endpoint model (ADR-006): same repo, same migrations, same suite everywhere; environments differ by Neon branch, never by database kind. No loopback database; endpoint policy names pinned project + branch + TLS + least-privilege role. Tests run where their credentials live (owner-terminal shell for dev-branch work, Sprite for remote execution).
- Reuse, don't rebuild: `testdb` harness (run DBs, fixed test role, allowlist), migration chain, accepted B2-01A lifecycle contract + pinned Tasks argv, existing gateway/OpenCode path, `sprite-env services` for PG lifecycle. No Docker in the Sprite. HostExec and testrunner stay preserved but uninstalled and off the critical path.
- Locked context from `plans/docker_vs_sprites.md` PARTIALLY SUPERSEDED for PostgreSQL by ADR-004: native-PG-in-Sprite is retired (no PG in Sprite, ever). The remainder stands and is not relitigated: Docker-in-Sprite rejected as default; Docker/OCI kept only for the Machine build pipeline. Verify the remaining Sprite properties live in Phase 2 rather than assuming them.
- No Taskfiles exist in the repo (prompt28 §4 mentions them; verified absent) — the test entrypoint is `go test` directly; environment setup/teardown beyond what the CLI covers stays as explicit documented commands per environment (no second command system).
- `plans/app_category_guidelines.md` is vocabulary only for this mission: tag demo operations with execution category / placement / authorization mode where useful, but implement no classification framework, admission machinery, or DBOS/River substrate (explicit non-goal per prompt28 §1/§7).
- Authorization follows `docs/pg_muse.md` §2.1: owner-terminal `sprite login`; explicit GO for Sprite changes (PG install), GitHub writes (per operation), Connector/App grants, and any URL exposure. Planning approval is not operation approval.
- Canonical interface = new `studio dev` subcommands reusing `testdb` + `store` packages (endpoint check, PG19 check, migrate, role ensure), not shell reimplementation, not Taskfiles (none exist), not a new binary. Service lifecycle (docker/container, `sprite-env`, CI service) stays explicit documented commands per environment — the CLI operates through DSNs and never controls Docker (Docker authority stays owner-terminal per `pg_muse.md` §2.3).
- CI is Fly-native per ADR-007 (T2): Machine controller + per-run Sprite + ephemeral Neon branch; head SHA from the webhook payload; canonical Taskfile targets; Go pinned to the `go.mod` line (`1.27`); verification per R312 (project/branch IDs, actual version, extensions). Digest-pinned-image language retired for the DB layer.
- Required extensions: none — verified by grep over both migration chains (25 Studio + framework baseline) and repo code. The parity fingerprint re-runs that proof rather than asserting it.
- Host-side execution runs owner-terminal with exact pasted commands. Escalated agent execution was explicitly denied this session — do not retry it; treat owner-terminal as the mechanism and keep every command copy-pasteable with expected output stated.

## Steps

### Phase 0 — Status audit (read-only; no new auth needed)

- Record repo state: branch, HEAD, worktree diffs vs HEAD, untracked material.
- Confirm accepted baseline files exist: B2-01A (`fly/lifecycle*`, contract tests), B2-01B prereqs (`store/lifecycle_receipts*`, migration 000025, Tasks bindings), gateway/OpenCode wiring.
- Inventory GitHub surface: Connector read path (`fly/connectors.go`, `connector/registry.go`) and any Studio-side write broker/approval enforcement; record precisely what exists vs what the demo must still build. Do not assume a broker exists.
- Re-establish test baseline via owner-terminal `go test` runs; record exact pass/fail/skip counts. If the owner cannot run them, that is finding #1 blocking all local verification.
- Output: audit note + the single smallest blocker to the next real demonstration.

### Phase 0B — Canonical dev interface + fingerprint (new code; no live infra)

- Read `cmd/studio/{main,migrate,provision}.go` dispatch conventions first, then add `studio dev` verbs reusing `testdb` + `store` (no reimplementation): `doctor` (diagnose + toolchain + readiness + fingerprint), `init` (dev/test DB init + migrate + verify), `reset` (safe cleanup guarded by the existing run-pattern + disposable-only rules), `migrate` (delegate to existing path), `test` (DB suite + full suite wrappers), `fingerprint` (machine-readable record).
- Fingerprint content: git commit, Go version, PG server version/build, encoding/locale/timezone, extensions proof (grep command + result), migration versions (both chains), endpoint shape (host/port only — never credentials), required-env presence by name.
- Unit-test the new verbs with fakes; run existing suite to prove no regressions. No Docker control, no Sprite contact, no credential handling beyond the existing DSN-from-environment convention (never print values).
- Gate: `studio dev doctor` passes locally (via owner-terminal) and the fingerprint is recorded before Phase 1 uses the interface.

### Phase 1 — Neon dev-branch reference (T1; ratified Stage 1)

- Preflight: Neon CLI/API + auth + pinned non-prod project verified via `task dev:setup`; reconcile the persistent dev branch + least-privilege creds; no local PG server involved.
- Clean-bootstrap proof: fresh branch state via the canonical interface, full migration chain from zero, schema + roles verified; an already-running database is not evidence.
- Run the representative DB subset, then the full DB-dependent suite against the dev branch through the canonical interface; record counts, skips (with reasons), failures.
- Fingerprint (R312 fields) + commit recorded; second full run proving repeatability.
- Gate: dev-branch reference reproducible and suite green (modulo pre-existing designed skips) before Sprite-execution work.

### Phase 2 — Sprite access + inspection (login is owner-terminal)

- Owner runs `sprite login` (org `isagani-mendoza`); agent verifies with read-only `sprite list`/status.
- Identify the designated Builder Sprite; resolve exact CLI subcommands from `sprite --help` at execution time (do not assume subcommand names).
- Read-only inspection only: distro/version, user + sudo, available PG packages and versions, service manager, `sprite-env services` behavior, Go toolchain presence/version, Go module path (proxy reachability vs pre-seed need), checkpoint support, pause/wake observables.
- Output: Sprite fact sheet. No writes, no installs, no provisioning.

### Phase 3 — RETIRED (no PostgreSQL in Sprites, ever — ADR-004)

This phase previously installed native PG19 in the Sprite. That design is superseded: Sprite execution connects to
Neon branches over TLS (ADR-006/007/008). Sprite-side prerequisites (toolchain, egress policy, checkpoint behavior)
are verified in Phase 2 inspection and the T2 execution plan. Do not execute the retired install procedure.

### Phase 4 — Neon parity proof: dev branch vs CI branch (ratified Stage 1)

- Run the representative subset (migrations, integration, RLS, tenant isolation), then the complete DB-dependent suite, in each environment against its Neon branch.
- Compare dev-branch vs CI-branch counts; investigate any divergence to root cause (environment defect vs product defect) before proceeding.
- Verify branch/credential isolation and branch cleanup (CI branch deleted; dev branch retained).
- No Docker/Testcontainers PostgreSQL anywhere — R302 audit must be clean; Docker-bound tests, if any remain genuinely Docker-bound for non-PG reasons, run in a separate clearly-labeled job, never conflated with parity.
- Run everything through the canonical interface; record fingerprint + commit; repeat the full run proving stability; a skip caused by environment inability is a failure to investigate, not a pass.
- Gate: both environments produce equivalent results for the same suite from clean bootstrap, or every divergence is root-caused and evidenced.
- Record exact commands, outputs, versions, counts, failures.

### Phase 4B — Fly-native CI proof (T2; needs controller + GitHub App authorizations)

- Build the T2 controller path per ADR-007/012 (Machine controller, per-run Sprite, ephemeral Neon branch); `ci.yml` Actions workflow is replaced, not modified in place — no parallel validation system.
- CI runs the canonical interface per run: branch create → migrate-from-zero → DB suite (RLS, tenant isolation, integration) → full suite → checks published → branch deleted; logs + reports preserved and associated with the exact head SHA.
- Skip policy: a checked-in allowlist names the designed skips; a run fails on any skip outside it. No Docker-PG job exists to conflate.
- Gate: green CI run from the same commit with zero unexplained skips. Controller deploy, App install, and live runs each need their own explicit GO; never provision on plan approval alone.

### Phase 5 — B2-01B live proof (per-operation GOs)

- Follow the accepted B2-01B card + `plans/plan1.md` sequence: Tasks hold create/verify/release, attach/detach, idle→pause observation, wake with latency + warm/cold classification, lifecycle receipt persistence, reconcile-after-restart.
- Each live operation needs its own GO; destroy carries its own explicit authorization. Stop on any surprise.

### Phase 6 — Demo slices (prompt28 §5 order; GO each live step)

1. Resident agent: dispatch one bounded real task via the gateway/OpenCode path; agent works in the Sprite repo copy; round-trip status/logs/evidence lands in Studio; interruption/resume shown if supported.
2. Genuine Connector read through the agent; verify the actual permission boundary live.
3. Authorization boundary: one approved consequential operation end-to-end plus one refused unapproved write, both enforced Studio-side. Board fields/chat text never count as authorization.
4. Evidence/recovery: status, logs, test results in Studio; checkpoint create/restore only where the provider supports it; no checkpoint-implies-DB-recovery claims.
5. Workspace Preview only if 1–4 are green; never labeled as Application Preview.

### Phase 7 — Rehearsal + acceptance (time-boxed)

- Maintain the NOT STARTED / IN PROGRESS / VERIFIED / BLOCKED / DEFERRED checklist from the first slice onward; every VERIFIED needs reproducible evidence.
- Full demo rehearsal, failure-recovery drill, submission prep. Cut list, in cut order: preview, checkpoint depth, connector-write breadth — never the core agent→evidence loop.

## Validation Plan

- Phase 0: `git status`, `git log`, file-existence checks; baseline `go test` counts via owner-terminal (escalated execution was denied — do not retry).
- Phase 0B: unit tests for new verbs (fakes); `gofmt`/`vet`; `studio dev doctor` + `fingerprint` output reviewed field-by-field.
- Phase 1: Neon preflight + dev-branch reconcile, `SHOW server_version`, clean-bootstrap proof, `go test -p 1 ./...` through the canonical interface (twice), skip-reason audit.
- Phase 2: `sprite list`/status/info outputs recorded; Sprite fact sheet with distro, PG candidates, Go toolchain, network findings. Zero writes.
- Phase 3: RETIRED — no validation; any Sprite PG install procedure found in history must not be executed.
- Phase 4: identical `go test` invocations both environments; count comparison table; isolation (cross-tenant denial, branch exclusivity) and cleanup (CI branch deleted, no orphans) checks; stability repeat.
- Phase 4B: CI run record + log excerpts; fingerprint step output; skip-policy green; migration-from-zero evidence; branch↔SHA association; checks visible on the commit.
- Parity report: version-controlled `docs/dev-parity-report.md` (skeleton ratified Stage 1; result cells filled by live proofs only) plus explicit deviations.
- Phase 5–6: per-operation receipts/IDs/latencies, approval refs for writes, refusal reasons verbatim, Studio-visible evidence screenshots-or-exports.
- Highest-risk validation: C4 endpoint-model redesign proven live (dev-branch suite + CI-branch suite + R302 audit) and the Neon credential-inheritance proof.

## Risks / Open Questions

- R1: `sprite login` still pending (owner terminal) — blocks all live work; all local phases ordered first so no idle waiting.
- R2: RETIRED with Phase 3 (no Sprite PG install exists to fail). Replaced by N14/V16 live quota/limit verification before scale commitments.
- R3: Go toolchain / module availability inside the Sprite is unknown until Phase 2; fallback is pre-seeded module cache or vendoring, decided on evidence.
- R4: GitHub write-broker implementation state is unverified; Phase 0 inventory determines build-vs-wire scope before any demo commitment.
- R5: Deadline pressure vs verification depth — the cut list (Phase 7) protects the core story; infrastructure polishing stops when E2E is unproven. Prompt28.1 priority order (local → Sprite → CI → B2 demo) is the execution order.
- R6: CI proof needs controller deploy + App install + live-run authorizations; until then the T2 plan is validated by review, contract tests against fakes, and local simulation of controller logic where possible — never presented as live verification.
- R7: Host-side execution is owner-terminal only — escalated agent execution was explicitly denied this session and must not be retried unasked. Every host command ships copy-pasteable with expected output.
- Non-goals: HostExec install/operation, B3/B4/B5, A/B/C/D classification framework or admission machinery (guidelines doc is vocabulary only), DBOS/River substrates, any self-hosted PostgreSQL as a convenience workaround, production data/credentials anywhere near tests, GitHub-Actions-based validation as a gate.
- Open questions: None blocking. Remaining unknowns (Sprite internals, broker code state, PG packaging) are scheduled as verification steps with fail-closed outcomes, and authorizations are structured as gates, not questions.

## Amendment P28.2/P28.3 — Fly-native lifecycle + Neon-first PostgreSQL (planned, not yet executed)

`plans/prompt28.2.md` and `plans/prompt28.3.md` are consolidated in `plans/prompt28.3_plan.md`, which governs
all lifecycle + database-platform work from here. Neon-first PostgreSQL is binding across every environment.
The following master-plan elements are SUPERSEDED in mechanism (formal phase revisions ratified in Stage 1,
step S1.4): local/native PG19 reference work, Sprite PostgreSQL install, dual-environment native proof,
digest-pinned CI image. Sprite read-only inspection, B2 live-proof intent, demo slices, and rehearsal are
RETAINED; their execution stack is decided at the Stage-1 exit gate. Completed Phase 0/0B work stands as
history. No Stage-2 implementation starts before that gate.

**Ratified Stage 1 (2026-11-27):** Phases 1/3/4/4B, Success Criteria, Approach, Validation Plan, and Risks above now
reflect the Neon-first revisions; see `plans/stage1-conflicts.md` (C1–C8) and ADR-004–018 for the records.
