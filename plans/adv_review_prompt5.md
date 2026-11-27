# Focused Adversarial Verification — Host Execution Broker (prompt5)

## Executive verdict: BLOCKED

Every original exploit path from the prompt3 review is closed and
demonstrated closed — except that two new HIGH defects stand in the
way of installation, both small and local, neither architectural:

1. **P5-HIGH-1:** `image.build` can never succeed against real Docker:
   the argv omits `-f Containerfile` while the context contains no
   `Dockerfile` (Docker defaults to `PATH/Dockerfile` per its own
   docs). The deployment workflow is provably non-functional, the
   regression test pins the broken argv, and fake-Docker masks it.
2. **P5-HIGH-2:** authoritative anchors are not first-wins: deleting a
   repo result copy and resubmitting the id silently overwrites the
   owner-side anchor, and `hostexec verify` still reports clean.
   History destruction the anchor system exists to prevent.

Both are fail-closed defects (no privilege gain demonstrated), but
installation cannot be authorized for a broker whose core workflow
errors out and whose evidence history is silently replaceable. No
broad re-review is needed after these two fixes: re-verify the build
argv against real Docker behavior and the anchor first-wins probe,
then install.

Verification-only review. No remediation performed, no install, no
Docker/PostgreSQL/systemd operations. The `docker` CLI exists on this
machine but was never invoked.

## Original findings status (prompt3 H1/H2/M1–M5/L1–L6)

| ID | Status | Evidence |
|---|---|---|
| H1 incomplete pins | CLOSED | Full-context `WalkContext` + staged verification; live P7 plant refused (`staged 8 files, pins cover 7`) |
| H2 pin TOCTOU | CLOSED | Copy-then-verify on staged bytes; build from stage only (conditional on StateDir outside agent write — see P5-MED-2) |
| M1 queue TOCTOU | CLOSED | fd-pinned claim; 1500+ adversarial claims, zero outside bytes (V4) |
| M2 unbounded read | CLOSED | Capped fd read; 256 MB refused in 272 µs, queue live |
| M3 forgeable `done/` | CLOSED for verdicts; PARTIAL for suite detail | Anchors + `verify` + conflict names; `test-results/` detail still unhashed (P5-MED-1) |
| M4 sweep overwrite | CLOSED | Terminal verdicts preserved; unparseable residue quarantined; parseable-plant residual is LOW (P5-LOW-1) |
| M5 config placement | CLOSED in code | Resolved-path placement + ownership + `/tmp`,`/dev/shm` deny + `ExtraAgentRoots`; sandbox attestation remains operational (P5-MED-2) |
| L1 trailing/dup JSON | CLOSED | Second-decode EOF + `rejectDupKeys`, tested |
| L2 newline rejection | CLOSED | Pretty-print + trailing newline accepted, tested |
| L3 quarantine overwrite | CLOSED | Unique quarantine names, tested |
| L4 error/unknown vocab | CLOSED | Timeouts/partial/indeterminate → UNKNOWN, tested |
| L5 hollow green | ADDRESSED per accepted limitation | Snapshot + manifest + `disarm.json` + mandatory diff review documented; gaps are P5-MED-1/P5-LOW-2 |
| L6 hardcoded paths | CLOSED | Broker-passed `GOROOT`/`GOMODCACHE`, entrypoint requires + verifies, tested |
| Digest binding (new) | IMPLEMENTED | `verified-image` record + tag-resolution refusal, tested; live-Docker proof out of scope here |

## V1 — Build integrity and immutable staging: PASS (modulo P5-HIGH-1)

Established by inspection + tests + live probes (Q6*):

1. Full `testrunner/` context pinned: `WalkContext` enumerates every
   entry; live tree pins 7 files including both `*_test.go`.
2. Missing/altered/unexpected/symlinked/subdir inputs refused
   (`TestStageContextRefusesDrift` + Q6b/Q6c on live bytes).
3. Verified bytes copied to owner-side `StateDir/stage-image`.
4. Verification hashes the STAGED copy (`StageContext` reads only
   stage paths after `cp`); pins compare against staged bytes.
5. `docker build` argv takes only the stage path (`Argv` refuses empty
   stage; `TestArgvFixed` pins it).
6. Staged bytes unreachable from the workspace iff StateDir is (V7).
7. P7 marker attack refused (test + live Q6b).
8. `TestStageSwapConsistency` (20 snapshots under rename attack) is
   genuine: rename atomicity + fd-pinned `cp` reads force each
   snapshot to exactly one version; content-membership + ≤2-manifest
   assertions prove it. 10/10 volume runs pass.

**Acceptance:** no workspace modification can introduce executable
code into a verified build — holds. But the build then fails for a
non-security reason (P5-HIGH-1).

## V2 — Per-job snapshots and provenance: PASS WITH GAP (P5-MED-1)

- Every run stages the whole repo minus queue/results and mounts the
  snapshot `:ro` (`StageRepo`, `Argv`, `TestStageRepo*`).
- Snapshot excludes runtime dirs; any symlink/fifo/socket refuses the
  run (live tree verified free of symlinks/special files incl. `.git`).
- Manifest (tree hash) frozen per run and carried on the result.
- `disarm.json` freezes job, git HEAD/status, image ID, manifest, and
  package counts; first-wins tested (`TestDisarmProvenance`).
- Hollow-green limitation is honestly documented (`docs/host-exec-broker.md`:
  "test-gutting is indistinguishable… owner diff review") and the
  acceptance workflow requires it. Hashing is not oversold.
- Gap: package-level *detail* (`gotest.json`, `summary.txt`,
  `exitcode`) is not hashed into any owner-side record — only counts
  are. Repeat runs overwrite it, post-hoc edits are undetectable
  (P5-MED-1). "Subsequent modifications cannot alter earlier-run
  evidence" holds for counts/manifest/image, not for detail.

## V3 — Image digest binding: PASS (trusted-daemon residual accepted)

- Verified build records image ID owner-side (`writeVerifiedImage`).
- Every run resolves the tag and refuses on mismatch/missing
  (`TestRunStaleTagRefused`, `TestRunWithoutBuildRefused`,
  `TestBuildInspectFailUnknown`).
- The verified digest is the executed selection: refusal precedes
  `docker run`; no fallback path.
- Residual: inspect→run tag TOCTOU closable only by a hostile-actor
  with Docker access — outside the stated trusted-daemon model.
  Accept; recommend a one-line doc note. Live-daemon proof was not
  possible in this verification (no Docker operations authorized).

## V4 — Queue claim integrity: PASS, anomaly classified (see §4)

- Claim is fully descriptor-relative: pinned dirfds, `openat` with
  `O_NOFOLLOW|O_NONBLOCK`, fstat-regular gate, capped fd read,
  `O_EXCL` running-stage of OUR bytes, dev/ino-verified retire.
- Symlinks/FIFOs/sockets skipped without blocking; size cap enforced
  during the read (~68 KB max allocation); claimed bytes cannot
  diverge from executed bytes (same buffer staged, verified, retired).
- Poison jobs quarantine-and-continue (`TestClaimOversizeQuarantined`
  proves the queue stays live; Q2: 256 MB refused in 272 µs).
- Directory-swap survival proven (`TestClaimSurvivesDirSwap`).
- Zero outside-byte claims established: `TestClaimSwapLoop` 30/30 +
  10/10 under `-race`, plus independent Q1a (300 claims, running-file
  identity checked each round) and Q1b (300 FIFO claims in 2.6 ms,
  none blocking, none claimed). Total 1500+ adversarial claims, zero
  outside bytes, zero hangs, zero errors.
- P3 anomaly: see §4 — classified non-blocking with evidence.

## V5 — Snapshot tearing: DEFENSIBLE GUARANTEE DEMONSTRATED, not a blocker

No quiesce protocol exists in code or docs (searched), and none is
needed for security. The implemented guarantee is narrower and
sufficient:

- **Build snapshots:** torn bytes fail the pin comparison → refuse.
  Tearing cannot smuggle (a pin match IS the reviewed bytes).
- **Run snapshots:** the manifest is computed from staged bytes
  AFTER the copy, so it always describes exactly what executes.
  Demonstrated: Q5 ran 15 snapshots under hostile in-place
  truncation/rewrite churn (0–100 KB, fsync'd); 0/15 manifests
  disagreed with an independent re-hash of the stage.
- A torn snapshot may execute a never-coherent tree — but the agent
  could submit any tree directly, and provenance truthfully records
  the executed bytes. Quiescence affects run *meaningfulness* (owner
  workflow: don't mutate mid-run; review the diff), never boundary
  soundness. Classification: acceptable with documented workflow;
  recommend one doc line ("the manifest describes executed bytes;
  quiesce the tree for a meaningful run").

## V6 — Evidence integrity: FAIL (P5-HIGH-2), sweep honest (P5-LOW-1 noted)

Verified holding:

- Anchors (bytes + `.sha256`, independent inodes — separate write
  paths, never hardlinks) land owner-side before repo copies.
- Repo squats land under `conflict-` names, loudly (`TestFinishSquattedPreserved`).
- `VerifyResults` detects mismatch/unpublished/missing/conflict
  (`TestVerify*`; semantics re-confirmed by probe).
- Sweep preserves terminal verdicts and quarantines unparseable
  residue (tests + Q4: `done/w.json` intact).
- Uncertain ops stay UNKNOWN; reconcile guidance present.

Failing:

- **P5-HIGH-2:** `writeAnchor` uses temp+rename OVERWRITE. Delete
  `done/x.json` (workspace write) → resubmit id `x` → claim+finish
  succeeds → anchor + hash silently replaced. Q3 demonstrated:
  `anchor overwritten=true`, then `verify: checked=1 problems=[]`.
  The duplicate/single-shot rule is defeated AND the erasure is
  verify-blind. V6's "Finish cannot silently overwrite" fails.
  Fix (small, local): first-wins anchors — `O_EXCL` anchor create,
  conflict-preserve on collision, mirroring the repo-side handling.
- **P5-LOW-1:** parseable planted residue becomes an ANCHORED unknown
  (`TestSweepGhostValidated` enshrines it). Bounded (UNKNOWN only,
  needs a restart) but: (a) the reason asserts a mid-execution
  restart the broker cannot know happened — reword to "residue found
  at startup; execution state unknown"; (b) it is a verify-clean id
  squat. Cheap fix, low urgency.

## V7 — Owner-side state: CODE PASS, OPERATIONAL PREREQUISITE (P5-MED-2)

- `checkPlacement` enforces on RESOLVED paths: queue inside repo;
  state/env outside repo, `/tmp`, `/dev/shm`, declared extras;
  symlink-escape refused; env outside any checkout, owner-owned,
  0600; state not world-writable. Tested (`TestPlacement`,
  `TestLoadConfigRefusesAgentRoots`) + Q7 (production `LoadConfig`
  refuses in-`/tmp` state).
- Correct: V7's warning stands — code cannot prove the sandbox
  confines the agent. `~/.config/hostexec` (or wherever the install
  puts state) and the DSN file are protected only if the runtime
  sandbox denies the agent those paths, and nothing in the repo can
  attest to that.
- **P5-MED-2 (operational, explicitly scoped):** installation
  requires a recorded owner attestation: exact StateDir/EnvFile,
  sandbox deny evidence (attempted agent-side write refused), and
  `ExtraAgentRoots` covering the sandbox's temp roots. Make it step
  0 of install; fail closed (no attestation → no install). This is
  the single operational residual for a future READY-WITH-ACCEPTANCE.

## V8 — Disarm, provenance, L1–L6: PASS (P5-LOW-2 noted)

- Disarm only on green `mode=run` after digest-bound execution;
  `disarm.json` first-wins (tested, incl. second-green no-op);
  re-arm owner-only (filesystem, conditional on P5-MED-2).
- Git HEAD/status recorded; git failure degrades honestly
  (`unavailable: …`, code-inspected).
- Trailing data + duplicate keys refused; pretty-print + trailing
  newline accepted (tested both directions).
- Unique quarantine names; UNKNOWN vocabulary; broker-passed
  toolchain env with entrypoint enforcement (`sh -n` clean +
  `TestEntrypointToolchainEnv`).
- Fixed argv policy unchanged in scope (all ops still constant +
  config + broker-generated; stage paths are the only addition).
- **P5-LOW-2:** git provenance reads the LIVE repo at disarm time,
  not the executed snapshot — mid-run repo changes misattribute the
  tree (manifest stays authoritative). Capture at snapshot time
  instead (the stage contains `.git`).

## New findings (prompt5 numbering)

### P5-HIGH-1 — Build argv missing `-f Containerfile`: deployment cannot succeed

- **Files:** `hostexec/exec.go:245-249` (`{d,"build","-t",tag,stage}`,
  no `-f`); pinned by `hostexec/exec_test.go:37`;
  `testrunner/Containerfile` (no `Dockerfile` exists); echoed in
  `docs/test-runner.md:57`.
- **Fact:** `docker build` defaults to `PATH/Dockerfile`
  ([Docker docs](https://docs.docker.com/reference/cli/docker/buildx/build/):
  "`-f`, `--file` — Name of the Dockerfile (default: `PATH/Dockerfile`)").
  Only Podman/Buildah default to `Containerfile`.
- **Effect:** every `image.build` fails against real Docker
  ("cannot locate Dockerfile"); the deployment chain (build →
  connect → netcheck → run) dead-ends at step one. All build
  evidence to date is fake-Docker. Fail-closed (no exploit), but
  installation is pointless until corrected.
- **Correction:** add `-f Containerfile` (constant) to the build
  argv + update the pinned test + docs line; re-run one REAL
  `docker build` at install (owner GO) as the first genuine
  build proof. One-line functional fix; no boundary change.

### P5-HIGH-2 — Anchors silently overwritten on delete + resubmit

- **Files:** `hostexec/queue.go:238-260` (`writeAnchor` rename-over);
  `hostexec/serve.go` finish path; demonstrated by `/tmp/p5probe/q6`.
- **Attack:** finish id `x` → `rm done/x.json` → resubmit `x` →
  claim (no duplicate anywhere) → finish → anchor + hash replaced;
  `verify` clean. Defeats id single-shot-ness and erases the first
  verdict undetectably.
- **Correction:** first-wins anchors (`O_EXCL`; on collision preserve
  under a conflict name and report, mirroring `publishResult`).

### P5-MED-1 — Suite-evidence detail not frozen owner-side

- Only package COUNTS enter `disarm.json`; `gotest.json`,
  `summary.txt`, `exitcode`, logs stay live-repo-mutable and are
  overwritten by repeat runs. Post-hoc detail edits preserving
  counts are undetectable.
- **Correction:** sha256 the `test-results/` files into the disarm
  record at disarm time (bounded, local).

### P5-MED-2 — Sandbox attestation prerequisite (operational)

- As V7: record StateDir/EnvFile + sandbox-deny evidence +
  `ExtraAgentRoots` before install; no attestation → no install.

### P5-LOW-1 — Sweep UNKNOWN wording + ghost id squat (V6 residual)

- Reword sweep reason to assert only what is known; optionally mark
  swept records. Bounded: UNKNOWN only, restart required.

### P5-LOW-2 — Git provenance from live tree at disarm time (V8 residual)

- Capture from the snapshot (or live tree at snapshot time) instead.

## P3 anomaly investigation (§4)

The remediation report disclosing "one unexplained swap-test failure
among 25+ passes" is not present in the repo (searched `plans/`,
`docs/`), so the original failing invariant, log line, and code
version are unavailable — that history cannot be recovered here.

What was established instead, with evidence:

- **Volume:** `TestClaimSwapLoop` 30/30 clean, plus 10/10 under
  `-race` (different interleavings), plus 10/10
  `TestStageSwapConsistency`. Claim-swap suite total: 40/40 runs,
  1600 claims, zero failures.
- **Independent harness (Q1):** separate attacker loop, 300 claims
  asserting bytes ∈ {A,B} AND running-file bytes == claimed bytes
  every round, plus 300 FIFO-only claims (all skip in 2.6 ms).
  Zero outside bytes, zero mismatches, zero hangs, zero errors.
- **Code-path analysis:** outside bytes are unreachable, not merely
  unobserved — `openat(O_NOFOLLOW)` fails on the symlink, fstat
  gates regularity, reads come from the pinned fd, and the swapper
  never writes outside bytes to any regular file. The residual
  failure modes (test-side timeout under extreme load, stale
  binary, earlier code iteration) are environmental, not boundary
  defects. The test asserts the correct property (claimed-bytes
  membership + no-hang).
- **Classification:** UNREPRODUCED, non-blocking. The guarded
  invariant (zero outside-byte claims) is demonstrated over 1900+
  adversarial claims across three harnesses plus an impossibility
  argument — this is not "verified because subsequent runs pass".
  Recommendation: keep `-count=10` repetition of both swap tests in
  CI to catch regressions; if the historical log ever resurfaces,
  compare its invariant against the Q1 assertions.

## Regression table (§3)

| Attack | Required outcome | Result |
|---|---|---|
| P7 unpinned source | Refused | REFUSED (test + live Q6b) |
| Context modification | Refused | REFUSED (drift/altered/missing/symlink/subdir) |
| Build-stage race | No attacker bytes execute | HOLDS (copy-then-verify; swap-consistency 10/10) |
| P3 symlink-swap | Zero outside-byte claims | HOLDS (1500+ claims, 3 harnesses) |
| FIFO swap | No wedge | HOLDS (300 claims/2.6 ms, all skip) |
| Oversized file | Bounded refusal | HOLDS (256 MB in 272 µs, queue live) |
| Terminal result + sweep | Preserved | PRESERVED (test + Q4) |
| Fabricated residue | Not trusted | QUARANTINED if unparseable; parseable → UNKNOWN (P5-LOW-1) |
| Result tampering | Detected vs anchor | DETECTED (`verify`); anchor-overwrite itself is P5-HIGH-2 |
| In-repo StateDir/EnvFile | Refused | REFUSED (tests + Q7 production path) |
| Tag mismatch | Refused | REFUSED (`TestRunStaleTagRefused`) |
| Ambiguous result | UNKNOWN | UNKNOWN (timeout/partial/inspect-fail) |

## Test evidence

| Check | Command | Result |
|---|---|---|
| gofmt | `gofmt -l hostexec/ testrunner/ cmd/hostexec/` | clean, exit 0 |
| vet | `go vet ./...` | clean, exit 0 |
| build | `go build ./...` | clean, exit 0 |
| unit broker | `go test -count=1 ./hostexec/` | ok, 52/52 PASS, 0 fail/skip |
| unit runner | `go test -count=1 ./testrunner/` | ok, 9/9 PASS, 0 fail/skip |
| race | `go test -race ./hostexec/ ./testrunner/` | ok, both |
| swap volume | `-run TestClaimSwapLoop -count=30` (+10 race) | 40/40 pass |
| stage volume | `-run TestStageSwapConsistency -count=10` | 10/10 pass |
| full serial | `go test -p 1 -count=1 ./...` | exit 0, 19 ok, 0 FAIL/panic, 3 no-test-files pkgs |
| entrypoint syntax | `sh -n testrunner/entrypoint.sh` | clean |
| Q1 swap/FIFO | `/tmp/p5probe/q14` | 300 claims clean; 300 FIFO skips, no block |
| Q2 oversize | `/tmp/p5probe/q2` | 256 MB refused in 272 µs |
| Q3 anchor | `/tmp/p5probe/q6` | overwrite demonstrated, verify-blind |
| Q4 sweep | `/tmp/p5probe/q6` | terminal intact; ghost → UNKNOWN |
| Q5 tearing | `/tmp/p5probe/q5` | 0/15 untruthful manifests |
| Q6 live stage | `/tmp/p5probe/q17` | 7-file pins ok; plant/symlink refused |
| Q7 placement | `/tmp/p5probe/q17` | in-/tmp state refused |

Scope notes: source-level + fake-Docker only; the `docker` CLI was
never invoked; `hostexec verify` CLI end-to-end needs a
non-`/tmp` non-repo StateDir, unavailable in this sandbox
(environmental — covered at unit level instead).

## Accepted residuals

- Trusted Docker daemon (tag TOCTOU, daemon reads of stage).
- DSN-to-suite visibility within the scratch cluster (owner-accepted
  per prompt4; production unreachable by construction).
- Hollow-green indistinguishability → mandatory diff review (now
  documented + provenance-frozen).
- `network ls` read-only disclosure; root-runner; no run rate limit;
  `cp`/`git` PATH executables (fail closed if absent).
- Torn-snapshot coherence (V5: provenance always truthful).

## Installation readiness

NOT READY — BLOCKED on P5-HIGH-1 (build argv) and P5-HIGH-2 (anchor
first-wins). Both are one-spot, fail-closed-direction fixes with
ready-made re-verification probes (real `docker build` with
`-f Containerfile`; Q3 expecting refusal + conflict preservation).
After those two land plus the P5-MED-2 attestation, no further review
cycle is needed: re-run the §3 table and install.
