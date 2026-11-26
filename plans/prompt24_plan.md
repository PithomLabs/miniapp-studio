# Prompt24 Execution Plan — Unified Lifecycle UI Implementation Plan (planning-only)

## 0. Objective

Execute `plans/prompt24.md`: investigate the repository and produce ONE
planning document at
`docs/architecture/unified-lifecycle-ui-implementation-plan.md`.
No implementation, no cloud operations, no commits. Stop after
delivering the plan and the §10 report.

Verified 2026-11-26: the target path does not exist yet
(`docs/architecture/` holds only `decisions/` and
`miniapp-studio-implementation-plan.md`), so this is a fresh document;
re-check at execution and preserve on collision. Working tree holds
68 modified/untracked entries to preserve exactly. Canonical inputs
exist: `plans/plan_ui.html`, `plans/plan_ui_z.md`,
`plans/phase-d-implementation-plan.md`, `docs/phase-d-evidence.md`
(D5/D5b done, 23.1 verdict filed). CLIs present but unprobed:
`fly`/`flyctl` under `~/.fly/bin`, `neon` under the nvm node bin.

## 1. Strategy reconciliation (binding for this execution)

Prompt24 has three layers; they combine in this order, no re-debate:

1. **Base prompt (§1–10)** sets the destination: unified
   dev→prod workflow, Focus Console extension (seven-stage spine
   binding, no environment-as-stage), two workstreams for
   observability, 15-section document structure, demo-vs-roadmap
   separation throughout.
2. **Bootstrap Q&A (the "chicken-and-egg" revision)** sets the
   strategy: minimum viable development control plane first —
   Studio useful before complete. The plan document MUST answer
   its 7 bootstrap questions in order (§"question" item 6) before
   presenting the full platform: smallest workflow on existing
   code, reusable Phase A–D capabilities, minimum schema delta
   per slice, human-assisted bridges, first genuine
   Studio-through-Studio milestone, then remote/Preview/
   troubleshooting, with the rest deferred. Bootstrap 0–5 are
   the closest thing to a fixed milestone spine; adjust only
   with repository evidence. Schema rule: add a persistent
   concept only when the next usable increment requires it;
   minimum viable schema contract per slice, never a
   speculative final schema.
3. **CLI amendment** is binding infrastructure policy:
   locally installed Fly/Neon CLIs are first-class operational
   interfaces (discover real versions/commands/auth via safe
   read-only inspection; invent nothing); operation classes
   (read-only → non-prod → staging → production →
   destructive) with server-side enforcement; agent-invoked
   CLIs now under supervision, Studio-owned infrastructure
   adapter later; no unrestricted shell tool; planning task
   performs zero mutating cloud operations.

Demo target is Miniapp Studio developing itself (Invoice Review
Desk is superseded as the demo; prior phases' evidence still
stands as implementation fact). Demo-critical path must use the
existing local Muse workflow, not a new multi-agent registry.

## 2. Execution phases

### P0 — Baseline and preserve (minutes)

- Record `git status --porcelain` (expect 68 entries) and
  the target-path absence check.
- Confirm read-only posture: no server/process starts, no
  DB writes, no cloud mutations, no commits.
- Output: baseline note; abort on unexpected tree state
  (e.g. target doc appeared with content → switch to
  preserve-and-merge mode).

### P1 — Repository investigation (current-state assessment)

Trace code and tests only; never infer from plans/TODOs/agent
claims. Cover, at minimum:

- Focus Console: `server/templates/focus/*`, handlers,
  spine builder, Pulse/Line, OPS/DEV, tests; scene-by-scene
  status (shipped vs placeholder — Deploy is the known
  placeholder) per `plans/plan_ui.html` + `plans/plan_ui_z.md`.
- Phase A–D backend: conversations/turns/questions/commands,
  Gateway protocol + sender, artifact pipeline
  (claim→fetch→verify→test→decide), approvals, stream/SSE,
  audit; current evidence docs as the capability ledger.
- Data model: full migration chain; map existing tables to
  the 8 conceptual workflow responsibilities
  (project/workspace/work-item/attempt/checkpoint/evidence/
  approval/deployment) to find the true delta per slice.
- Boundaries already enforced: tenant isolation, authorization
  points (F1/G-01 residuals), idempotency, secret scrubbing,
  sandbox/toolchain, registry/tool surface.
- Docs: architecture decisions, phase plans/evidence,
  roadmap states; note where repo evidence supersedes the
  prompt's §7 notes.
- Output: capability ledger (implemented / partial /
  placeholder / planned) with file+test citations. Every
  later claim about "existing" must cite this ledger.

### P2 — CLI capability discovery (safe read-only)

- Resolve executables via PATH (no assumed names); record
  `--version` output.
- Record `help` / command-list output for lifecycle-relevant
  areas (Fly: org/apps/machines/deploy/logs/secrets; Neon:
  projects/branches/databases/roles/connections).
- Check authentication *state* (logged-in identity/org only)
  via documented status commands; NEVER print, copy, or
  persist tokens, connection strings, or secrets; redact
  command output in notes by default.
- Probe agent reachability dimensions from the prompt:
  PATH resolution, auth-context accessibility from the
  agent process, cwd/shell constraints, read-vs-mutate
  distinguishability. No mutations, no provisioning.
- Output: CLI discovery table (executable, version, auth
  state, supported commands relevant to Preview/Staging/
  Production) with unknowns marked unknown. Version-pin
  every command contract cited later.

### P3 — Architecture reconciliation

- Map P1 capabilities onto the prompt's target: context bar,
  workbench, handoff, Deploy/Run scenes, Pulse/Line growth,
  settings, troubleshooting loop, observability A/B.
- For each Bootstrap 0–5 slice: minimum workflow, reused
  capabilities (no second task/artifact/approval engines),
  minimum schema delta, human-assisted bridges, exit
  milestone. Flag any slice whose dependency forces a
  merge/split, with evidence.
- Environment model check: workspace-location × runtime-
  environment as independent controls; Neon-branch and Fly
  Machine lifecycle per environment; promotion = same
  immutable artifact + digest-bound human approval.
- Record discrepancies where implementation contradicts
  binding principles, with durable resolutions (never silent
  redesign to fit accidents).
- Output: reconciled phase skeleton + schema-delta-per-slice
  table + discrepancy log. This is the document's backbone.

### P4 — Draft the plan document

Write `docs/architecture/unified-lifecycle-ui-implementation-plan.md`
in the §9 15-section structure, with the bootstrap answers
up front (they govern sequencing) and the CLI amendment woven
through architecture, phases, and acceptance criteria:

- Every phase carries all 16 required fields (§6), S/M/L/XL
  sizing, independently verifiable acceptance (no "works
  correctly"), and named evidence for completion.
- Dependency graph + separate demo-critical-path sequence
  (tables required; Mermaid only where it clarifies).
- Status vocabulary used exactly: implemented /
  implemented-but-not-integrated / demonstrated-with-evidence
  / planned / blocked-by-environment-or-permissions /
  not-yet-tested. No invented demo evidence.
- Enforcement-point table for §8 invariants
  (database/server/Gateway/Builder/runtime/deployment/UI).
- Explicit non-goals incl. deferred multi-agent registry,
  metrics/traces gates, marketplace/canvas/fleet.

### P5 — Self-verification (before delivery)

- Every "existing capability" cites P1 file+test evidence;
  every CLI command cites P2 version+help output; proposals
  marked proposals, unknowns marked unknown.
- Boundaries re-checked: one new file only, no source/test/
  migration/UI edits, no processes started, no cloud
  mutations, no secrets in the doc or report, no commits.
- Re-read against prompt24 §1–10 + bootstrap 7 questions +
  CLI §1–7: each requirement traceable to a section.
- Output: corrections applied; delivery checklist signed.

### P6 — Deliver and stop

Report exactly the §10 six items: document path, phase/
dependency summary, recommended demo-critical path, top
risks/unresolved decisions, sources/evidence used, and the
no-implementation/cloud/commit/push confirmation. Then stop
and wait for architectural review + implementation
authorization.

## 3. Explicit non-goals for THIS execution

Writing code, tests, or migrations; starting Studio/Gateway/
agents; any Fly/Neon/GitHub mutation (including "harmless"
preview creates); destructive or mutating DB operations;
touching `docs/phase-d-evidence.md` (implementation evidence,
not planning output); treating the aborted D6 rehearsal as
evidence; redesigning binding principles.

## 4. Acceptance of this planning task

Done when: (1) the document exists at the exact path with all
15 sections; (2) bootstrap questions answered in order with
per-slice schema deltas; (3) CLI discovery real (versions +
commands recorded, zero invented flags); (4) every phase has
all 16 fields + verifiable acceptance; (5) demo path separated
from roadmap with honest status labels; (6) §10 six-item
report delivered; (7) tree contains exactly one new file and
zero implementation diffs. Then wait for next instruction.
