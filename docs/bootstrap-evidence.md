# Bootstrap evidence

## B0 — Studio manages one local task (implementation record)

Authorization: owner GO (Bootstrap B0, Local Desktop only, local PG,
no cloud/commits). Constitution: root `SKILLS.md` (read before work;
§22 decision brief filed in-session).

- Migration `000023_build_execution_location` (+down): one column
  on `studio.builds`, `local_desktop` default, CHECK over the two
  workspace locations. No grant/RLS change; `LatestVersion` derives
  from embedded files; down round-trips through the existing
  version-staged test.
- `build.packet` registry read tool (`tools/build_packet.go`):
  versioned `v0` packet (ids, workspace, objective, bounded task
  messages, branch hint, prior attempts as excerpts + event refs,
  artifact/decision refs, manual-bridge pointer). Tenant-owned
  reads; truncation always flagged, never silent.
- `RecordAttempt` operator-only op (`server/build_attempt.go`,
  POST `/console/attempt-result`, session+Origin+opkey):
  result + verdict recorded as one `attempt_result` event plus a
  stream row, atomically. Event id derives from the opkey (retries
  replay; fresh keys append); numbering serializes on the locked
  build row; parent links automatically (n−1). Accept closes the
  build through assigned→running→succeeded (ingest-passthrough
  precedent) and closes open turns; reject/rework leave it open;
  terminal builds refuse. Reports/notes/summaries scrub secret
  spans before storage (ingest write-path precedent).
- Console: run scene gains the Execution packet card (version,
  objective, branch hint, attempts table, DEV raw JSON) and the
  Record attempt result form (open builds only); layout gains the
  v0 context bar (project/branch/agent/workspace/env, durable or
  honest dashes) above the spine on focused renders.
- Runbook: `docs/local-agent-bridge.md` (procedure, packet v0
  reference, idempotency, secrets, limitations).
- Corrections with rationale: scene template `{with` single-brace
  typo broke parsing (fixed at the typo); new tool enrolled in
  the `allTools` list after `TestAllHandlersRequireTenant`
  failed red (28 tools now covered by the tenant/authz/
  malformed sweeps).
- Residuals: (1) manual builds carry a held gateway `assign` row
  (sender prerequisite-hold: no burn, no network, no cloud) —
  revisit when execution location drives dispatch (B1); (2) no
  browser proof in B0 (route-level acceptance instead; browser
  rehearsal belongs to the §12 joint rehearsals).
- Tests: `tools/build_packet_test.go` (round trip, truncation,
  isolation, unknown build); `server/build_attempt_test.go`
  (round trip via public stream, parent linkage, replay
  single-card proof, accept-closes + terminal refusal, tenant
  isolation, full HTTP form round trip, validation matrix incl.
  anonymous 401). Route-level B0 loop test IS the acceptance
  proof for criteria 1–3.
- Gates: GREEN. `go test -p 1 -count=1 ./...` rc=0, 17/17
  packages ok (sandbox 524.3s incl. daemon tests, server 39.7s,
  tools 14.0s, store 7.6s, rest <4s). gofmt/vet/build clean.
  Scratch run DBs reaped (0 remain). No sandbox flake this run.
- Verdict: B0 DONE per its three acceptance criteria (route-level
  proof in `TestConsoleAttemptFormRoundTrip` + `TestRecordAttempt*
  `): real task end-to-end through Console routes with all state
  durable; Evidence/Pulse show attempt + tests + decision; second
  attempt links its parent. Submitted for review; B1 not started.

## B1 — Local development loop (implementation record)

Authorization: owner GO (prompt25; B1 only, Local Desktop + local PG,
no cloud/commits). Constitution: root `SKILLS.md` (§22 brief filed
in-session before execution).

- Migration `000024_checkpoints` (+down): `checkpoints` table
  (tenant/build/attempt/turn, version CHECK `v1`, complete flag,
  per-build seq, repo path, branch, HEAD, clean flag, manifest +
  context JSONB), composite tenant FKs with CASCADE, ENABLED+FORCED
  RLS, least-privilege grants, test-role mirror + parity-list
  updates. References and digests only — no blobs, credentials,
  transcripts.
- Ops (`server/build_checkpoint.go`, all operator-only, POST routes
  with session+Origin+opkey): `StartAttempt` (one open attempt,
  409 on double-start, same-key replay), `CreateCheckpoint`
  (server-side Git inspection → manifest → atomic row + progress
  event, same-key replay), `InterruptAttempt` (marker event),
  `ResumeAttempt` (latest-complete check, superseded/stale 409,
  worktree-compat validation with explicit-reconcile path, opens
  the resumed attempt linked to its checkpoint). `RecordAttempt`
  extended backward-compatibly: result payloads inherit
  `source_checkpoint` from a resumed open attempt (absent for B0
  flows). Accept/terminal/404 semantics unchanged.
- Git inspection (`server/gitworktree.go`): read-only commands only
  (`rev-parse`, `branch`, `status --porcelain=v1 -z
  --untracked-files=all`, `ls-files -s -z`, `show-ref`), direct exec
  (no shell), timeout + output caps, root confinement (absolute,
  top-level, symlink-resolved). Manifest binds branch + HEAD +
  per-path entries with staged blob SHAs and worktree sha256,
  modes, rename pairs, unmerged/conflict coverage; deterministic
  digest. Bounds fail closed (4 MiB/file, 64 MiB total, 5000
  entries). Plus cheap `QuickStatus` (render path) and bounded
  read-only `ListWorktree` (.git skipped, depth 3, 200 entries).
- Config: optional `STUDIO_WORKSPACE_ROOT` (absolute dir, validated
  at Load); empty means Git features refuse/report unavailable
  instead of guessing.
- Console: run scene gains checkpoints/recovery card (history
  table, start/capture/interrupt/resume forms with server-side
  preconditions) and a Workspace card (live git panel, bounded
  file listing, recorded test outputs); evidence scene gains
  Attempts + Checkpoints sections; context bar gains live
  tree/commit/dirty, checkpoint ref, and a stale marker (live
  branch ≠ checkpoint branch). Git failures degrade to explicit
  unavailable states — never a 500. Spine/stages/OPS/DEV/a11y
  untouched; navigation mutates nothing.
- Runbook: `docs/local-agent-bridge.md` B1 addendum (extended loop,
  one-open-attempt, resume/reconcile rules, reconstruction honesty,
  read-only inspection, progress streaming).
- Corrections with rationale: `normal` untracked mode collapsed
  pure-untracked dirs (hid per-file contents) → `all` with caps;
  `with`+`else` template pattern skipped empty states → plain `if`;
  new FK table added to both test truncate lists after the
  no-CASCADE failure (established pattern); listing depth off-by-one
  fixed before running.
- Residuals: (1) manual-build held `assign` rows unchanged (still
  non-burning; dispatch stays untouched per §3/§9); (2) history
  queries are per-build (no cross-build checkpoint search);
  (3) unborn-HEAD repos checkpoint with empty HEAD (valid, tested).
- Tests: `server/gitworktree_test.go` (clean/states/content-
  sensitivity/staged-vs-unstaged/unborn/detached/mode/unmerged/
  caps/no-mutation/quick/list/validation); `server/
  build_checkpoint_test.go` (start round trip + replay + conflict,
  capture round trip incl. manifest-honesty vs fresh inspection,
  open-attempt gate, replay single-row, scrub canary, bounds,
  unconfigured refusal, incomplete exclusion with higher seq,
  version-CHECK pin, interrupt, happy-path resume with packet
  linkage, drift refusal + reconcile, superseded refusal,
  no-checkpoint refusal, concurrent seq uniqueness + start
  single-winner, isolation, restart survival + post-restart
  resume, full Console HTTP loop, stale marker, unsafe-409 page,
  unavailable degradation); `TestLoadWorkspaceRoot` (valid/
  relative/missing). `build.packet` unchanged (B0 compat).
- Gates: GREEN. `go test -p 1 -count=1 ./...` rc=0, 17/17
  packages ok (sandbox 520.0s incl. daemon tests, server 47.1s,
  tools 14.5s, store 8.1s, rest <4s). gofmt/vet/build clean.
  Race detector clean on checkpoint sequencing, event
  persistence, and concurrent resume/write paths. Scratch run
  DBs reaped (0 remain). No sandbox flake this run.
- Verdict: B1 DONE per acceptance criteria A–I (see report).
  Submitted for review; B2 not started.

## B1-R1 — Adversarial-review remediation (implementation record)

Authorization: owner decision in `plans/prompt26.1.md` (fix F-01,
F-02, F-03, F-04, F-06, F-07, F-09, F-10, F-11; document F-05, F-08;
no B2, no cloud, no commits). Baseline F-12–F-14 accepted as
written; P1–P3 noted.

- F-01 TOCTOU/symlink/FIFO: hashing now opens every component
  relative to a held root fd (`O_NOFOLLOW` throughout,
  `O_NONBLOCK` on the final open); type and size come from fstat
  on the open descriptor; cancellation threads per file. Symlink,
  vanished, unreadable, and non-regular paths record entry notes
  (deterministic, resume-sound: same condition replays, changed
  conditions mismatch); only budget breaches and unexpected
  failures abort capture. FIFO opens can no longer block (proven
  by direct unit call with a timeout guard — git omits FIFOs from
  status, so the live hazard was always the swap race). Uses
  `golang.org/x/sys/unix` (already in the module graph; zero
  go.mod/go.sum delta). Residual: privileged parent-dir swap
  between top-level resolution and first open recorded in the
  runbook as accepted.
- F-02 hidden flags: `ls-files -v` tripwire in both inspection
  and quick status; any assume-unchanged/skip-worktree path
  refuses capture/resume/UI-live-state with an explicit message
  naming count + sample. Sparse checkouts out of B1 scope.
- F-03 submodules: gitlinks enumerate from the index (including
  status-invisible clean ones) and verify initialized + clean +
  non-nested + worktree HEAD == index SHA; anything else refuses
  explicitly. Deinit trap closed: the subdir must prove itself a
  repo (toplevel check) before its git output is trusted.
- F-04 replay identity: resume owns the `resume:` event namespace;
  the checkpoint conflict path verifies row ownership (cross-build
  reuse → 409 naming the other build); same-key cross-operation
  reuse → 409 via a per-build usage check spanning all five B0/B1
  op namespaces (record included). Same-op retries still replay.
- F-06 secrets: secret-shaped branch names and paths refuse
  capture/resume/UI-live-state outright (redacting identity
  strings could collide; the operator renames). B0 typed branch/
  digest scrub at the op persistence boundary (defense in depth
  with the form pass; redaction idempotent). Commits need no
  scrubbing (hex validation admits nothing the span table
  matches). Canary coverage across DB rows, stream, packet,
  Console render, and refusal paths.
- F-07 waiting_user: `refuseNonLocal` names the actual state in
  the checkpoint gate and the record gate (shared helper);
  runbook documents gateway-owned ineligibility.
- F-09 framing: canonical length-prefixed mf2 with injectivity
  unit proof (an mf1-colliding pair diverges) plus separator/
  newline filename fixtures; mf1 framer retained frozen with
  per-row format dispatch, so pre-upgrade checkpoints resume
  honestly (proven by an mf1-row resume test). No schema change:
  the format tag lives in the manifest JSON.
- F-10 config: `status.renames=true` pinned on every status call;
  audit documented in code (quotepath neutralized by -z,
  untracked/relative/color moot by explicit flags and top-level
  execution, no diff invoked, ignorecase accepted platform
  variance, fsmonitor staleness fail-closed by construction).
  Skew test proves identical entries under hostile config.
- F-11 listing: skip count returned and rendered ("N entries
  unreadable"); depth bound labeled statically; unreadable
  subdirectories fail loudly. Deterministic injection test for
  the skip path.
- F-05 ignored files: runbook boundary section + reconciliation
  rule (verify by hand, record in the reconcile note; digests
  only, never secret values) + rendered limitation line on the
  checkpoints card (guarded by test). No ignored-file tripwire:
  counts false-negative and hashing risks secrets — documented.
- F-08 repo path: documented as provenance-only (runbook + code
  comments); workbench shows the workspace root with the rule;
  moved-tree resume test pins the semantics (succeeds, original
  path preserved).
- Extra coverage: concurrent-resume single-winner test,
  resume-vs-capture race test, 401/403 sweep over all four B1
  routes plus a no-leak stream assertion.
- Corrections with rationale: deinit trap (git answers from the
  parent when the subdir is empty — toplevel proof added);
  untracked mode collapsed pure-untracked dirs (→ `all` with
  caps); `mf1`/`mf2` dispatch defaults tagless rows to mf1.
- Residuals: review P1 (test-workspace isolation — temp repos
  already per-test), P2 (covered by F-01 tests), P3 (queue
  accounting: B1 surfaces counts; deeper accounting deferred),
  F-12 (B0 checklist), F-13 (ASSESSMENT unchanged), F-14 (plan-doc
  amendments belong to the owner-held doc process).
- Tests: 30+ new/extended tests across `gitworktree_test.go`,
  `build_checkpoint_test.go`, `config_test.go` (pre-existing).
- Gates: GREEN. `go test -p 1 -count=1 ./...` rc=0, 17/17
  packages ok (sandbox 514.4s incl. daemon tests, server 48.3s,
  tools 14.4s, store 7.8s, rest <4s). gofmt/vet/build clean.
  Race detector clean on checkpoint sequencing, start/resume
  single-winner, resume-vs-capture, replay, and record paths.
  Migration up/down + RLS/parity suites green (no schema change
  in R1). Scratch run DBs reaped (0 remain). No sandbox flake.
- Verdict: B1-R1 DONE — all eleven findings closed per their
  dispositions (F-01,02,03,04,06,07,09,10,11 fixed; F-05,F-08
  documented), extra coverage delivered, B0 compatibility
  preserved (all B0 tests green unmodified; shared-path changes
  limited to additive safety: cross-op 409, op-boundary scrub,
  accurate refusal message). Submitted for review; B2 not started.
