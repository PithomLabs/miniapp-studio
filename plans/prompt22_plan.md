# Prompt22 Plan — Phase D Architecture and Implementation Plan

Document-only task. Produce `plans/phase-d-implementation-plan.md`
per `plans/prompt22.md` §14 (18 required sections), then stop for
architectural acceptance. No Phase D implementation, no code changes,
no live operations, no commits.

## Goal

Deliver an implementation-ready Phase D plan (Miniapp artifact
pipeline and verification for the Invoice Review Desk) grounded in
actual repository evidence, with the three critical decisions —
artifact identity, execution isolation, human authorization —
answered as mechanisms, and every undecidable point marked
unresolved rather than guessed.

## Success Criteria

- `plans/phase-d-implementation-plan.md` exists with all 18 §14
  sections, each traceable to inspected repo paths.
- Every capability claim cites an exact file path; anything not
  found in the tree is marked missing/unresolved, never assumed.
- The D0–D6 sequence is dependency-ordered with evidence for
  each ordering choice; no UI work precedes trustworthy backend
  state; no Phase E work leaks in.
- Negative matrix covers at least the eight §11 classes plus any
  the threat model adds, each with an expected result.
- Working tree gains exactly one file (the plan); `git status`
  shows no other new modifications.

## Approach

Four stages, strictly ordered. Investigation before design;
design before drafting; audit before delivery.

- **S0 — Baseline sweep.** Read the §4 source list end to end
  (gateway plan, agent guidelines, roadmap4, demo critical path,
  Boundary B / U5.12 / U5.12.1 reports), record accepted
  decisions and their dates, and confirm the locked UX specs
  (`plans/plan_ui.html`, `plans/plan_ui_z.md`). Output: decision
  baseline with no new interpretation.
- **S1 — Capability inventory (verify, don't assume).** Trace in
  code: artifact/build/app/deployment models (`store/`,
  migrations `000011`–`000020`, `tools/build.go`,
  `tools/ingest.go`); tools registry and manifest conventions
  (`tools/register.go`, `contract.ProfileDigest`); RLS/migration
  architecture; approval and authorization-request mechanisms;
  Console scene handlers/templates; `testdb` endpoint guards;
  Miniapp framework/runtime contracts (pinned SHA). Record
  discrepancies between accepted reports and source. Do not
  reopen Boundary B decisions without invariant-violation
  evidence.
- **S2 — Decisions.** Resolve with evidence: artifact contract
  shape (MiniappSpec versioning, layout, packaging, digest,
  identity, correlation); retrieval/admission threat model
  (traversal, links, bombs, substitution, secrets);
  `artifact.verify` determinism boundary;
  `artifact.test` isolation — survey what actually exists on
  this machine (read-only: container runtime, unshare/userns,
  cgroups) and, if inadequate, design the honest
  blocked/unavailable result with trusted-fixture-only local
  testing; `artifact.decide` human-only binding (digest-tied,
  idempotent, race-safe); minimal migration deltas on top of
  000020; minimal Console scenes; full failure/recovery matrix.
- **S3 — Draft, audit, report.** Write the 18 sections, then
  audit: every cited path exists, no section contradicts
  another or the §3 invariants, unresolved points are explicit,
  Phase E boundary is crisp. Report findings, structural
  blockers (if any), and stop.

## Steps

1. S0: read and baseline the seven §4 documents + locked UX
   specs. Depends on: nothing.
2. S1a: trace persistence (migrations, RLS, artifact/approval
   tables) and tool-registry/manifest surfaces. Depends on: 1.
3. S1b: trace approval/auth, Console scenes, testdb guards,
   framework contracts. Depends on: 1. (Parallelizable with 2.)
4. S1c: discrepancy log — reports vs. source. Depends on: 2, 3.
5. S2a: artifact contract + admission security design.
   Depends on: 4.
6. S2b: verify/test/decide contracts + isolation survey
   (read-only environment probing) + persistence deltas.
   Depends on: 4. (Parallelizable with 5.)
7. S2c: failure/recovery matrix + negative matrix + Console
   scene deltas + D0–D6 sequencing. Depends on: 5, 6.
8. S3: draft the plan document. Depends on: 7.
9. S3: citation/contradiction/unresolved audit; fix; final
   readback. Depends on: 8.
10. Report findings and blockers; stop. Depends on: 9.

## Validation Plan

- Completeness: all 18 §14 headings present, in order, each
  with content (manual readback).
- Citation audit: spot-check every cited path with
  `ls`/`grep`; any missing path fails the draft.
- Invariant audit: each §3 invariant maps to at least one
  enforcing mechanism in the design (table check).
- Boundary audit: `git status --short` shows only
  `?? plans/phase-d-implementation-plan.md` as a new entry;
  no code, migration, or config file modified.
- Negative coverage: each §11 class appears in the matrix
  with an expected result and a named regression test.

## Risks / Open Questions

- Highest risk: adequate local execution isolation may not
  exist; the plan must then define the honest blocked state
  rather than a weak sandbox. The prompt anticipates this
  (§7); treat it as a likely outcome, not a failure.
- Scope risk: artifact verification easily expands toward
  generic builds or deployment. Guard: every section ends at
  the Phase D boundary (human decision recorded); deployment
  language fails review.
- No clarifying questions for the owner before drafting: the
  prompt is decision-complete as a planning brief, and its
  three critical decisions are the drafter's to propose, not
  the owner's to pre-answer. Assumptions: PG19 `:5434`
  remains the only database touched and only if schema
  questions arise that migration files cannot answer;
  environment isolation survey is read-only (`docker images`,
  `unshare --help`-class probes, no containers started).
