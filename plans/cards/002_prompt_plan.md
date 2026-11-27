# Plan: execute `plans/cards/002_prompt.md` → `plans/cards/card002_plan.md`

Planning-only meta-plan. No implementation, no live ops, no commits.

## 0. Starting position (verified 2026-11-28)

- `002_prompt.md` demands exactly one new file (`plans/cards/002_prompt_plan.md`), no edits to Card 002 /
  runbook / code, no commits or pushes, no credentials, no live operations.
- Directly overlapping work already exists: Card 002 prep (runbook §0b/§3/§5b/§9 rewrite + card update) is
  committed LOCALLY but was unpushed at last check — so local tree is AHEAD of `main`. The prompt says to
  read "`main` versions". First job is to pin which side each input comes from and reconcile; never silently
  mix them.
- Known input sizes: runbook ~730 lines, Card 002 ~110 lines, Card 001 + closure evidence, roadmap 52 KB
  (`plans/miniapp-studio-ultimate-roadmap.md`), gateway package + provision scripts + CLI surface.
- `plans/cards/TEMPLATE.md` (referenced by the cards README) was present during prep but is absent from the
  latest listing — re-verify at execution; if still missing, note it as an input anomaly, do not recreate it
  (out of scope: create ONLY `card002_plan.md`).

## 1. Execution phases

### Phase A — Input inventory (read-only)

1. Record exact input versions: `git rev-parse HEAD`, `git status --porcelain`, and for each input file
   whether it is taken from local tree or `main` (if they differ, cite both and state which governs each
   conclusion).
2. Read in full: `002-b2-live-proofs.md`, `gateway-phase-b-live-runbook.md`, `001-suite-green-neon.md`
   (§7 closure + `test-run-card001.log` existence/counts), the B2-relevant roadmap sections (Phase 5 /
   H-track / exit evidence — grep-guided, not the whole 52 KB blind), gateway implementation + tests +
   `tools/provisiondata/setup.sh` + `cmd/studio/gateway.go` + server routes/config/callback.
3. Output of this phase: a dated input table (file → version → status) that goes into the deliverable as
   its evidence baseline.

### Phase B — Runbook audit (the core; requirement 1)

For each runbook section (§0–§15 incl. new §0b/§3/§5b/§9): re-verify every command, flag, pin, env var,
expected output, and recovery row against the CURRENT tree and installed CLIs (`sprite`/`opencode --help`
surface only — never authenticated or mutating invocations). Classify each item as VERIFIED (with
path + line), STALE (what changed), or UNVERIFIABLE-LOCALLY (what live observation must confirm).
Pay special attention to anything the prep commit touched (§3 gate, §5b/§9, 400/401 probes, §12 rows):
audit it, do not grandfather it.

### Phase C — Reconciliations (requirements 4–6)

1. §3 vs Card 001: restate the prep's reconciliation (B2 gate = repo-local + Card-001 citation-or-rerun),
   then stress-test it — explicitly argue why each B2 stage's dependencies are covered WITHOUT a full
   suite, and state what the focused gate does NOT prove. No silent waivers, no equivalence claims.
2. D3/Fly-hosted: confirm no in-tree deploy procedure exists (grep, don't assume); keep deployment
   owner-established with §5b as the acceptance gate; never invent deploy steps.
3. Boundaries: carry R vs C0–C6, per-class GOs, N2 prohibition, N8 precautions, OAuth/provider separation
   into every stage spec; any conflict found during audit becomes a blocker, not a footnote.

### Phase D — Draft `card002_plan.md` (requirements 2, 3, 7, 8)

Sequenced stages: preflight → pins/tree → build → provision → privilege/identity → OAuth → provider auth →
callback readiness → no-op turn → sleep/wake/resume → cleanup → evidence. Per stage: prerequisites,
responsible party, exact action/command, expected result, evidence to capture, stop conditions, recovery
path. Plus: seven-field brief, dependencies, evidence/report structure, unresolved blockers, and a final
GO/NO-GO recommendation with conditions. Voice: actionable and concise; every claim tagged VERIFIED /
ASSUMPTION / PROPOSAL / OWNER-DECISION with repo-path citations.

### Phase E — Self-verification (before reporting)

1. Re-read the deliverable against all 8 requirements + 5 strict boundaries, one by one.
2. Confirm: single new file only; no other file touched (`git status` diff vs Phase A baseline must show
   exactly `?? plans/cards/card002_plan.md`); no credentials; no live ops performed.
3. Report: file path + critical blockers + decisions needing owner attention. No implementation GO implied.

## 2. Boundaries (binding, from the prompt)

Planning only; no code/runbook/card edits; no live Fly/Sprite/Neon/OAuth/provider/provision/deploy/install/
cleanup ops; no credential handling in any form; no commit/push; create ONLY `plans/cards/card002_plan.md`.

## 3. Risks / open points

- Local-vs-`main` divergence on the prep commit could confuse "current versions" — Phase A pins it first.
- Roadmap is large; undisciplined reading burns the session — grep-guided extraction (B2/H/exit-evidence).
- The 400/401 probe correction and §5b/§9 are prep-authored and unaudited-by-a-second-pair — Phase B
  re-verifies rather than trusts.
- No clarification needed from the owner before starting: the prompt is complete and the boundaries are
  explicit. Questions arising mid-audit (e.g. a stale command with two plausible fixes) get recorded as
  PROPOSAL + OWNER-DECISION in the deliverable, not asked interactively.
