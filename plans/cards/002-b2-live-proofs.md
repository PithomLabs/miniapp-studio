# Card 002: B2 live proofs (gateway runbook execution)

- Status: `draft`
- Roadmap: A (B2 — blocked on Card 001 green)
- Owner decision required: GO per the runbook's own authorization structure (per-operation GOs for provision,
  OAuth, provider auth; this card covers execution + evidence, each live step already gated inside the runbook).

## Outcome

The Phase B live runbook (`docs/gateway-phase-b-live-runbook.md`) is executed end to end: B2 regression gate
green, Fly-hosted Studio ready, builder provisioned, identity/inbox proofs recorded, OAuth + provider
ceremonies completed, callback route proven, no-op supervised turn demonstrated, sleep/wake/resume observed,
cleanup verified — or each unproven item explicitly carried with its blocker.

## Acceptance criteria

- [ ] Runbook §0–§5b pre-live gates pass (pins, tree, B2 gate, binary, env, Studio ready — recorded).
- [ ] Builder provision + privilege discovery recorded (§6–§7, honest unknowns preserved).
- [ ] Identity/inbox proofs recorded (§7b); OAuth ladder P-E1→P-E3 completed (§8); provider ceremony (§8b).
- [ ] Callback route proven (§9: 400 + 401 probes + status) before any agent run.
- [ ] No-op turn + sleep/wake/resume observations recorded (§10–§11) or explicitly deferred with reason.
- [ ] Cleanup executed (§13); receipts, timings, and file references attached; no orphaned resources.
- [ ] Per-class GO record complete (each C0–C6 step ran under its own explicit authorization).

## Dependencies

- Card 001 green — SATISFIED (seven-test Neon gate GREEN 2026-11-28, commit `0c027ae`, 468.5s; no live
  proofs on a red suite — cited by §3, re-run only if HEAD moved).
- Fly-hosted Studio DEPLOYED + §5b-passing (C0 prerequisite, owner-established; blocks §6).
- Owner identity: Fly org holds, Sprite login, Neon branch, OpenRouter key ceremony (all owner-terminal).
- Pins + tree state re-verified at execution time (runbook §2 — never floated).
- Full-suite live verification stays DEFERRED to the pre-demo gate (never a Card 002 requirement).

## Decision brief

1. Execution location: owner terminal for all live steps; agent prepares, reviews output, records evidence.
2. Application target: Builder Sprite + gateway binary + Phase B integration surface.
3. Database target: Neon dev branch (run-pattern DBs for any DB touch); no migration changes expected.
4. Infrastructure actions: as scripted in the runbook only, per authorization class (§0b): C1 provision,
   C2 package install, C3 OAuth grants, C4 provider key, C5 agent runs, C6 destroy/retention.
5. Authorization class: R steps under the Part 2 GO; EACH C-class step needs its own explicit owner GO at
   execution time (a GO for one class never covers another); §6.5 upgrade PROHIBITED until N2 closes
   (no GO may authorize it); no commits/pushes; no scope beyond the runbook's named steps.
6. Verification and evidence: runbook section outputs + receipts + timings in a gitignored log; agent
   verifies each proof against its stated expectation before marking it; §14 evidence list is mandatory.
7. Recovery and rollback: runbook §13 cleanup always runs; failure interpretation per §12; partial state is
   reported, never force-cleaned.

## Report (§7)

**Prep complete (2026-11-28, docs + local analysis only — zero live execution).** No credentials touched,
no provisioning, no infrastructure changes. Changes made:

- Runbook §3 rewritten as the B2 regression gate (fmt/vet/build + `env -u TEST_DATABASE_URL go test`
  repo-local + `command -v sprite` + Card 001 citation-or-rerun); full live suite stays deferred.
- New §0b authorization map (R vs C0–C6, per-class GOs, N2 prohibition restated); class markers on §5b–§13.
- New §5b Studio readiness gate (Fly deployment record, Proxy listener, double-200) blocking §6; new §9
  callback route proof (400 + 401 probes + status) blocking §10. D3 tunnel language removed (fallback only
  with a new decision); the dangling §9 references (§1 note, §6.5) now resolve.
- Corrected the route proof: a bare POST yields 400 "malformed envelope", not 401 (verified in
  `server/gateway_callback.go:32-35`); the 401 needs a well-formed bogus envelope (schema verified in
  `gateway/protocol.go:39-50,173-185`), prescribed exactly.
- §6 STUDIO_URL = §5b-verified URL; §12 +§5b/§9 outcome rows; §13 destroy-prompt note (no `--force`);
  §14 evidence list extended (pins, gate, §5b record, §9 probes, GO record).
- Verified against tree/CLI (all match, no drift): §2 pins (`setup.sh:13-18`, `go.mod`, `provision_test.go`),
  `sprite` info/exec/console/destroy/list + `-s` (CLI `--help`), `opencode` mcp/models/auth surface incl.
  `login --provider` and `models --verbose --refresh` (CLI `--help`, local 1.19.6), gateway
  provision/upgrade/command flags (`cmd/studio/gateway.go:43-47,92-95,141-145`), §5 env incl. `_FILE`
  convention (`server/config.go:63-106,142-156`), routes (`server/routes.go:19-23`), §12 strings
  (setup.sh:30, build_provision.go:230, gateway/server.go:217,496,533,539,547,564), §10 payload keys
  (gateway/server.go:460-482), fetch_artifact 400 (`gateway/server.go:705-709,853-861`), readyz shape
  (`server/routes.go:72-90`), §15 backlog still open (`server/server.go:79-82`).
- N2 respected: §6.5 prohibition preserved verbatim; no GO may authorize upgrade; §12 upgrade rows unchanged.

**Remaining for execution:** C0 (Fly-hosted Studio deployed + §5b-passing) is the critical path — everything
from §6 on hangs off it. Then per-class GOs C1→C6 in runbook order. Awaiting owner review + authorization.
