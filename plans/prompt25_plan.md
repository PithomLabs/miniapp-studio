# Prompt25 Execution Plan — B1 Local Development Loop (implementation)

## 0. Objective and authorization

Implement prompt25 (`plans/prompt25.md`): B1 checkpoints, progress,
interruption/resume, minimal Git awareness, narrow Console
integration. Binding decision brief is prompt25 §1 (Local Desktop
only, local PG + run-unique test DBs/roles, no cloud/Sprites/
Machines/Neon/GitHub, no commits/pushes/destructive ops).
B2+ explicitly NOT authorized. Stop with evidence + report.

Baseline 2026-11-26: 85 tree entries (drift from B0's 73 is
owner-side arrivals — re-verify at execution, preserve all);
migration chain ends at 000023 with 000024 free and unreferenced;
B0 artifacts present (`build.packet`, `RecordAttempt`, 000023,
runbook, evidence); sender held-assign confirmed at
`server/sender.go:23-24,377-385` (prerequisite-hold, no burn, no
network); root `SKILLS.md` read in full this session; one hygiene
item: my B0 debug leftover `server/zz_dbg_test.go` must be removed
in P0 (my scratch, not user work).

## 1. Execution phases

### P0 — Baseline and hygiene

- Record `git status --porcelain`; remove ONLY `server/zz_dbg_test.go`.
- Re-confirm 000024 availability and B0 artifact presence.
- Output: baseline note. Abort on unexpected tree state.

### P1 — Investigate and produce the approach brief (prompt25 §10.3–5)

- Trace: B0 packet/attempt paths, `builds`/`turns`/`events`/
  `conversation_stream` writes, sender hold path, migration/test-role
  conventions, run scene + Pulse shaping, scrub helpers.
- Answer with evidence: (a) held-assign — change dispatch or not
  (standing recommendation: no change; held rows burn nothing;
  revisit only if a B1 invariant forces it); (b) checkpoint
  authorship — operator-triggered server ops + auto-capture on
  attempt record (registry excluded: agents must not drive
  server-filesystem reads); (c) workspace path — server config
  root + traversal confinement, path recorded per checkpoint as
  data; (d) progress events as `kind='event'` rows with typed
  payloads (no CHECK change); (e) resume as a new attempt linked
  to checkpoint + parent (extends B0 linkage, no new engine);
  (f) workbench slice bounds (read-only listing depth/ignore
  rules, secret-safe output view).
- Output: concise approach + §22 brief. STOP and report on any
  structural conflict with the plan or constitution.

### P2 — Schema: migration 000024 (+down)

- `checkpoints` table per §4.1 (tenant, build, attempt ref, turn
  ref, version, git refs + manifest digest, bounded scrubbed
  context JSONB, message/event/evidence refs, complete flag, seq),
  composite tenant FKs, ENABLED+FORCED RLS, least-privilege grants,
  test-role mirror + parity updates, down/up round-trip test.
- No other tables. No blobs, credentials, transcripts in columns.

### P3 — Checkpoint write/read APIs

- Create (idempotent keys, concurrent-write ordering: newer never
  clobbered by stale), latest-complete retrieval, history,
  resume-safety validation, resume recording with source link —
  server-side, tenant/ownership enforced, payloads versioned,
  bounded, scrubbed; atomic checkpoint + event/receipt writes.

### P4 — Resume + Git manifest

- Interruption identification, history preservation, latest-complete
  selection (incomplete never selected), build/attempt/tenant/git
  validation, worktree-compat check with explicit-reconciliation
  refusal (never silent reset — no reset/clean/discard commands
  anywhere), reconstructed packet, resume-as-linked-attempt.
- Server-side Git inspection: safe bounded invocation, porcelain/
  machine-readable parsing, clean/staged/unstaged/untracked/
  deleted/renamed coverage, deterministic manifest digest,
  worktree-mutation proof test. Resume honesty states only:
  reconstructed vs refused (no native-resume claims).
- Manifest content-sensitivity invariant (binding): a status-only
  digest is insufficient — a file can change contents while staying
  "modified", which would let Studio wrongly declare an
  incompatible workspace safe. The manifest MUST include file
  paths + Git states, content digests (sha256) for staged,
  unstaged, AND untracked changes, deleted/renamed-file entries
  (with rename source/destination), and file modes where relevant.
  Hashing is bounded (per-file byte cap + total file-count cap);
  exceeding either fails closed: checkpoint creation refuses with
  an explicit over-limit error, never a silent partial manifest.
  Digests and references persist; source contents never enter PG.

### P5 — Progress events + minimal UI

- Progress lifecycle events (started/recorded/verified/
  interrupted/resume-requested/resumed/completed) via existing
  stream/SSE/Pulse; structured, bounded, scrubbed; no fabricated
  percentages; durable receipts distinct from agent claims.
- Console: context-bar branch/commit/dirty/checkpoint/resume
  state (+stale honesty), Run/Evidence attempt-checkpoint history,
  Pulse kinds, read-only workbench slice (git panel, bounded
  listing, output view), checkpoint/resume actions with server-side
  preconditions. Spine/stages/OPS/DEV/a11y untouched; navigation
  mutates nothing; no terminal, no full workbench.

### P6 — Tests (§7 matrix)

- Persistence: round trip, latest-complete, version/malformed
  refusal, size bounds, secret canaries, isolation, atomicity,
  idempotency, concurrent-write ordering.
- Recovery: attempt+checkpoint → simulated interruption → resume
  from same checkout; context preserved; interrupted attempt
  visible; resume links checkpoint+parent; incomplete never
  selected; stale/mismatched git refused-or-reconciled; no
  duplicate progress on retry.
- Git: six worktree states vs actual `git` output; determinism;
  no-mutation proof; content-sensitivity regression set:
  same-status-new-content changes the manifest digest (modify one
  file twice, status identical, digests differ); staged vs
  unstaged content distinguished; untracked file contents digested
  (not just listed); deleted paths and rename pairs exact; mode
  change detected; over-limit file and over-limit file count both
  refuse checkpoint creation with explicit errors (fail-closed,
  no partial manifest persisted).
- UI: restart survival, history-from-durable-state, Pulse
  fidelity, stale honesty, precondition enforcement, isolation,
  OPS/DEV + a11y intact.
- B0 compatibility: packet version/shape, attempt idempotency/
  numbering/parentage, verdict semantics, held-assign non-burning.

### P7 — Full gates

`gofmt`, focused packages, migration up/down + schema checks, RLS/
parity tests, `go vet ./...`, `go build ./...`,
`go test -p 1 -count=1 ./...`, targeted `-race` (checkpoint
sequencing, event persistence, concurrent resume). Red = stop;
never suppress/skip/weaken/relabel.

### P8 — Evidence + report + stop

- Extend `docs/bootstrap-evidence.md` with the B1 record.
- Report per prompt25 §12 (9 items) with A–I evidence mapping.
- Reap scratch DBs, confirm tree (only B1 files changed), STOP.
  No B2.

## 2. Explicit non-goals (from §5 + constitution)

No Sprites/handoff/transfer, Machines/deployments, Preview/Staging/
Production, Neon, GitHub, agent auto-launch, agent registry, full
workbench, terminal, metrics/traces, artifact-pipeline redesign, or
B0 packet/attempt contract changes beyond narrowly scoped
backward-compatible fixes with proven defects.

## 3. Acceptance of this planning task

Done when this file exists and the owner issues GO (or asks for
changes). No code, migration, test, or UI work happens in the
planning turn. Then wait for next instruction.
