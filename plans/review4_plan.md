# Prompt27.1-R4 Revision Plan — Focused Correction (planning-only)

## 0. Objective and authorization

Execute `plans/review4.md`: apply its 2 blockers + 2 security details
as a focused R4 revision of
`docs/architecture/issue-driven-engineering-workflow-plan.md`.
Preserve all accepted work byte-for-byte outside the corrected
passages — including the review's §4 accepted table, which needs
no action. Planning and documentation only: no implementation,
GitHub/Project writes, cloud operations, commits, or pushes. Stop
with a correction report and wait for the acceptance decision.

Baseline 2026-11-26: target doc is 683 lines (8 sections, 11
cards); tree count noted at execution and preserved exactly
(owner-side drift expected — touch nothing but the deliverable).

## 1. Work items

### R4-1 — Split the write-denial proofs (review §1)

- B2-02 keeps the Connector-boundary proof ONLY: the Sprite
  cannot mutate GitHub through Fly's Connector (per-operation
  allowed/refused matrix on the actual authorized Connector).
- B2-08 gains the DISTINCT broker-enforcement proof: Studio
  refuses the unapproved write before the GitHub App executes
  it. Required cases: missing, expired, revoked, and
  mismatched-target approvals; cross-operation approval use;
  accepted approval produces exactly its authorized effect and
  nothing else. Plus the invariant: the broker never lets the
  Sprite obtain the App token or call its authenticated API.
- §3 write-path split updated to name both properties and their
  separate proofs; coverage matrix rows for push/PR/merge
  reference the split.

### R4-2 — Merge boundary despite broad permissions (review §1)

- B2-08 acceptance: PR creation/updates require their own scoped
  approvals; merge is refused without a separate merge approval
  even though the installation token technically permits it; a
  creation approval cannot replay into a merge; broker bypass is
  refused. Same principle stated for Issues and Discussions
  (create/update/delete granularity — write ≠ create-only).

### R4-3 — Fallback decision: Option B (review §2)

- RECOMMENDATION (rationale below): build the independent
  Studio-side read-only integration as new card B2-10, keeping
  it separate from the write broker so the graph stays acyclic:
  B2-10 registers the GitHub App (read perms) and implements
  fallback reads ONLY for operations B2-02 proves unavailable
  (possibly the empty set, recorded with evidence); deps B2-02
  (needs its matrix). B2-03's fallback branch points at B2-10;
  B2-08 gains B2-10 as a dependency (App exists before binding);
  B3-01's "full B2 core" includes B2-10. Rationale: the incident
  demo is the mission centerpiece — it must not hinge on
  unverified third-party capabilities — and the App is needed
  for writes regardless, so B2-10 is never speculative scaffolding.
- Flip path (one paragraph, no rework if overruled): under Option
  A, drop B2-10, mark B2-04's CI acceptance and demo beats 4–6
  conditional on Connector verification, remove fallback edges.
- Update the independence map, parallelizable pairs, and demo
  beats (evidence names which path served each read).

### R4-4 — Safe probes (review §3)

- B2-02 probe procedure: disposable test repository or explicitly
  authorized canary resources, defined cleanup path, stop on the
  first unexpected successful mutation, cleanup under its own
  authorization. B2-08's positive-effect test uses the same
  canary discipline. Note Fly's path/identity-based (not
  per-method) controls as the reason probes can surprise.

## 2. Verification before delivery

- Structure intact: 8 sections; 12 cards each carrying all seven
  execution fields; dependency map consistent (B2-10 edges,
  no cycles — mechanically check every dep points backward).
- Acceptance mapping: (1) two named proofs in B2-02 vs B2-08;
  (2) merge/no-replay/no-bypass tests + Issues/Discussions
  principle; (3) fallback decided with acyclic deps; (4) safe-
  probe procedure + cleanup authorization in B2-02 (and B2-08
  positive test).
- Focused-diff review: every hunk maps to exactly one item
  above; §4-accepted passages byte-identical; no new scope, no
  invented capabilities, no secrets.
- Tree holds only the revised doc as modified content.

## 3. Explicit non-goals

Implementation of any kind; creating GitHub issues, Discussions,
Project cards, or categories; editing `AGENTS.md`; cloud/Fly/Neon
operations; diagnosing the CI incident; committing or pushing.

## 4. Acceptance of this planning task

Done when this file exists and the owner issues GO for the R4
revision (or asks for changes). No revision work happens in the
planning turn. Then wait for next instruction.
