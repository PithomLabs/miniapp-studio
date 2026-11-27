# Prompt27.1-R3 Revision Plan — Focused Correction (planning-only)

## 0. Objective and authorization

Execute `plans/review3.md`: apply its 4 blocking-issue fixes, 3
additional corrections, and 2 clarifications to
`docs/architecture/issue-driven-engineering-workflow-plan.md` as a
focused R3 revision. Preserve all accepted planning work
byte-for-byte outside the corrected passages. Planning and
documentation only: no implementation, GitHub/Project writes,
cloud operations, commits, or pushes. Stop with a correction
report and wait for the acceptance decision.

Baseline 2026-11-26: target doc is 616 lines (8 sections, 11
cards); all review-quoted passages verified present (B2-01B/B2-03
locations, "non-mutating", interim-governance rule). Tree count
noted at execution and preserved exactly; owner-side drift since
R2 is expected and must not be touched.

## 1. Work items (one per review finding)

### R3-1 — Connector call location (review §1)

- B2-03: move actual read operations inside the Builder Sprite
  (`Builder Sprite agent → Fly GitHub Connector → GitHub →
  sanitized evidence returned to Studio`); Studio receives and
  persists evidence, never poses as the Connector caller. Update
  Location, evidence (Sprite-side transcripts + Studio receipts),
  and recovery accordingly.
- Ripple: §1 end-to-end workflow line and §5 Connector read
  surface must name Sprite-side execution explicitly.

### R3-2 — Demonstrated read-only boundary (review §2)

- B2-02 acceptance becomes an explicit matrix procedure: enumerate
  every needed endpoint/operation; test the actual authorized
  Connector (never a fake or direct-API substitute); attempt
  unauthorized writes including on read-used routes; record
  allowed/refused per operation.
- Fallback rule (binding): if read-only cannot be demonstrated,
  restrict the Connector to demonstrably safe operations and move
  affected reads behind a Studio-side read-only integration —
  never describe an unproven boundary as read-only.
- B2-08 negative proof reframed: it must test the real Connector
  boundary, not only the Studio App's own refusal. §3 refusal
  text updated (path policy ≠ method proof).

### R3-3 — No assumed capabilities (review §3)

- B2-02 live verification gates every operation B2-03 needs;
  B2-03 proceeds only with verified operations and names a
  defined alternative (Studio-side read path or deferred scope)
  for anything unsupported (Discussions/Actions runs/jobs/logs
  explicitly, per the published guide gap).
- Demo script beats depending on Actions/logging evidence become
  conditional on verification, with degraded-but-honest evidence
  when a capability is unavailable.

### R3-4 — Studio-controlled lifecycle proof (review §4)

- B2-01B rewrite: exercise the Fly adapter through Studio's own
  lifecycle operations with durable receipts and server-side
  authorization enforcement. Owner-terminal bootstrap demoted to
  a separately identified prerequisite — never presented as the
  proof. Destruction gets its own explicit authorization bound
  to the exact Sprite and effect (scope + auth + evidence).

### R3-A — Preview access policy (additional)

- B2-07: private organization-authenticated access by default;
  synthetic/sanitized data only; no production credentials or
  private production data; public exposure requires separate
  explicit authorization for the exact URL; acceptance evidence
  verifies the actual access mode, not URL presence.

### R3-B — App permission matrix (additional)

- B2-08 gains a precise least-privilege matrix: minimum GitHub
  App permissions per write operation (Issues, Discussions,
  pull requests, and Contents for installation-token git access
  stated explicitly), repository installation scope, and
  deliberately excluded permissions. Private key and
  installation credentials server-side only — outside Sprite,
  agent context, repository, and Studio database. §3 references
  the matrix.

### R3-C — AGENTS.md governance rule (additional)

- Replace the interim-governance sentence ("new statements govern
  new work") with the stop rule: until the owner approves the
  reconciliation, any implementation task hitting a conflict
  stops and requests a decision; the Proposed plan never
  silently supersedes governing instructions.

### R3-D — Two clarifications (additional)

- State explicitly (Task Cards passage + exclusions): Project #3
  Draft cards are manually created by the owner for the initial
  workflow; any Project V2 integration is a separate, deferred,
  authorized capability.
- Replace every "non-mutating" describing checkpoint capture
  with "non-destructive, task-scoped" (snapshots are persistent
  side effects); verify by grep that none remain.

## 2. Verification before delivery

- Structure intact: 8 sections; 11 cards each carrying all seven
  execution fields; dependency map and demo path consistent.
- The 6 acceptance tests mapped to doc assertions: (1) Sprite-
  origin reads → B2-03; (2) write-denial or safe fallback →
  B2-02/B2-08; (3) verified-ops-only → B2-02 gate + B2-03 scope;
  (4) Studio-controlled lifecycle → B2-01B; (5) Preview privacy →
  B2-07; (6) App matrix + AGENTS.md rule → B2-08/§4.
- Focused-diff review: every hunk maps to exactly one item above;
  accepted passages byte-identical; no new scope, no invented
  capabilities, no secrets.
- Tree holds only the revised doc as modified content.

## 3. Explicit non-goals

Implementation of any kind; creating GitHub issues, Discussions,
Project cards, or categories; editing `AGENTS.md`; cloud/Fly/Neon
operations; diagnosing the CI incident; committing or pushing.

## 4. Acceptance of this planning task

Done when this file exists and the owner issues GO for the R3
revision (or asks for changes). No revision work happens in the
planning turn. Then wait for next instruction.
