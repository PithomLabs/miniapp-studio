## Goal

Implement prompt32's high-autonomy delivery directive as an operating pipeline: GitHub Project #3 becomes the
single coordination surface, work flows as bounded authorized Task Cards with seven-field briefs, and Roadmap A
(B0–B5) advances by verified vertical slices while invariants, authorization boundaries, and evidence discipline
hold. No R-series implementation; no new process beyond what the directive requires.

## Success Criteria

- A repeatable card lifecycle exists and has completed one full trial cycle: draft → authorize → implement →
  verify → report → board reflects reality.
- The first card sequence is drafted, authorized one at a time, and each carries outcome, acceptance criteria,
  dependencies, the seven-field brief, and honest status.
- Every executed card ends with the §7 report (changes, rationale, files, tests actually run with results,
  limits, cloud/db ops, next card) and traceable evidence.
- No parallel tracking system; no silent scope or authorization expansion; no commits/pushes/provisioning
  outside explicit per-card authorization.
- Suite-green on Neon at measured timeouts is re-established as the standing quality bar before B3 slices start.

## Approach

Bounded autonomy with explicit seams. The directive grants implementation freedom inside authorized cards; the
plan designs only the seams the directive leaves open — how cards get written without agent-side GitHub auth
(verified this turn: `gh` has no usable credential here and standing rules forbid requesting one), how board
truth stays synchronized, and which cards come first given actual repo state (clean tree, HEAD includes T1 +
prompt30 work, 19 untracked plans-only files, gates are gofmt/vet/build/tests).

Key decisions:

- **D1 — File-first cards, owner-mirrored board.** Agent authors each card as `plans/cards/NNN-slug.md`
  (seven-field brief + outcome + acceptance + dependencies + status). Owner creates/moves the Project #3 item
  via exact owner-terminal `gh` commands (verified via `--help` at execution time, never invented here).
  Board reads come back the same way (`gh project item-list --format json` to a gitignored file the agent
  reads). Rejected: agent-side board writes (no credential path exists that survives the no-secrets rule) and
  chat-pasted card text (drifts from the file of record).
- **D2 — Single active card default.** One human owner + shared Neon branch + one checkout means WIP > 1 is
  the failure mode Kanban exists to prevent. Parallel cards allowed only with isolated worktrees + isolated
  database state, stated on the card. Rejected: parallel-by-default (provenance and interference risk).
- **D3 — First sequence follows prompt32 §5 from measured state.** Card 1 re-establishes suite-green on Neon
  (the sized-timeout rerun adjudicates the last known failures); Card 2 executes B2 live proofs from the
  existing gateway runbook; Cards 3+ take B3 preview slices from the T2/T3 tracks. Rejected: starting new
  features before the quality bar is green, and re-planning B2 (runbook exists).
- **D4 — F0 enters only as B-dependencies.** Shared-contract work appears as linked cards pulled by B-slices
  that need them, never as a parallel program that gates delivery. Rejected: F0-first sequencing.
- **D5 — Evidence standard = the §7 report + artifacts.** Each card closes with files changed, tests/commands
  actually executed with counts, limits/risks, cloud/db ops, and the recommended next card; logs live in
  gitignored evidence files. Rejected: lighter-weight closes (they caused the current verification debt).
- **D6 — Docs triage is Card 0, not a program.** plans/ holds 19 untracked imports plus prior prompts; Card 0
  classifies them (active/superseded/reference) once, then the rule is: new cards reference, never duplicate.
- **D7 — Roadmap B stays design-only.** R0 architecture work continues only as explicit design cards; any
  R1+ implementation card is out of scope until B-gates say otherwise. Rejected: letting catalog/scheduling
  work masquerade as B-slices.

## Steps

1. **Card 0 — pipeline standup.** Write the card template + file convention (`plans/cards/`, seven fields,
   status values); verify Project #3 columns/fields via owner-terminal `gh` read; triage plans/ imports;
   draft Cards 1–3. Trial-run Card 0 itself through the full loop as the pipeline shakedown.
2. **Card 1 — suite green on Neon.** Re-run full suite at measured timeouts (`-timeout 30m`, streaming);
   triage every failure as defect/flake/capacity with the bounded-evidence discipline; fix defects at root
   cause; record counts + durations as the new baseline. Gate for all feature cards.
3. **Card 2 — B2 live proofs.** Execute the gateway live runbook owner-terminal per its own GO structure;
   record receipts, timings, cleanup; close or carry each proof item explicitly.
4. **Cards 3+ — B3 preview slices.** Ordered per T2/T3 tracks (controller → Sprite execution → per-PR branch →
   checks); each card independently verifiable with live proofs gated per operation.
5. **Operating cadence.** Per card: authorize → implement with root-cause discipline → gates → adversarial
   self-review → §7 report → owner mirrors board status from the report. Agent never moves board items
   directly, never infers commit/push/provision authority, never starts the next card on a red board.

## Validation Plan

- Card 0 acceptance: template exists; board columns/fields confirmed from live read; plans/ triage recorded;
  Cards 1–3 drafted with complete briefs; the Card-0 report itself follows §7.
- Card 1 acceptance: full suite green at measured timeouts with exact counts, or every red item triaged to a
  named defect card with evidence attached. Baseline durations recorded for future timeout sizing.
- Per-card gates: `gofmt`, `go vet ./...`, `go build ./...`, focused tests + affected suites; live steps only
  via owner-terminal with evidence files; `git status` reviewed before any authorized commit/push.
- Pipeline audit (monthly or on suspicion): sample one closed card and trace board status → report → evidence
  files → code/tests; any break fails the audit and pauses new cards until repaired.
- Highest-risk validation: Card 1's live full-suite run — it re-proves every layer (Neon binding, migrations,
  tenant isolation, suite health) after the prompt30 rework, and its failures define the real backlog.

## Risks / Open Questions

- Owner-terminal bottleneck: every board write and live run needs the owner. Mitigated by batching (one
  command block per card boundary) and making each block exact; a scoped installation-token path for board
  writes is a future explicit decision, not smuggled in here.
- Demo pressure vs invariant discipline: prompt32 §4/§6 already settle this (invariants win); the plan adds
  no escape hatches, and Card 1 exists to prove the bar before features stack on it.
- Doc sprawl recurring: the Card-0 triage plus the no-duplicate rule; new prompts arrive as inputs, roadmaps
  stay single-source in the reconciled ultimate roadmap.
- Open questions: none blocking. Card content details (exact B3 slice boundaries) are decided per card at
  authorization time from the T-track plans, not pre-planned here.
