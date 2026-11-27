# prompt4_plan — Broker Remediation (H1/H2/M1–M5/L1–L6 + digest binding)

## Goal

Execute the single authorized repository-only remediation pass from
`plans/prompt4.md`: close the adversarial findings against the host
execution broker (H1/H2 blockers, M1–M5, L1–L6) and bind container
execution to the recorded image digest — with a regression test per
exploit, especially P3 (queue swap) and P7 (unpinned build input).
No install, start, Docker, PostgreSQL, systemd, credential, or host
changes; no owner diagnostic commands.

## Success Criteria

- Every finding H1/H2/M1–M5/L1–L6 has a fix plus a failing-before/
  passing-after regression test; the P3 swap loop yields zero
  outside-byte claims and the P7 plant is refused.
- Container `run` executes only the digest recorded by a verified
  build; stale tags and missing builds refuse loudly.
- `gofmt`/`vet`/`build` clean; full serial suite green with exact
  pass/fail/skip counts; race clean on touched packages.
- Report delivers: files changed, finding-to-fix mapping,
  reproduction evidence, counts, classified remaining risks, and an
  installation-readiness verdict.

## Approach

Fix at the owning boundary per finding, in prompt4's priority order.
Two design judgments carry the whole pass:

**D1 — Two snapshot domains (entailed by H1a vs H1b).** The review's
"stage verified bytes" fix closes H1a only as written: the image
build bakes just `entrypoint.sh`, while run mode compiles and tests
the whole repo from the mount — so a tampered `relay_test.go` (H1b)
executes no matter what the image pins say. The only coherent
closure: (a) INFRA inputs (`testrunner/` context → image): full-walk
install-time pins, unexpected files refused, staged immutable build;
(b) RUN inputs (whole repo for gofmt/vet/build/test): per-job
owner-side snapshot mounted read-only instead of the live repo, with
frozen provenance (tree hash, git HEAD/status, package counts). The
broker cannot distinguish test-gutting from legitimate dev (both are
dirty trees), so H1b/L5 close as snapshot-consistency plus frozen
provenance plus mandatory owner diff review — stated plainly, not as
a refusal that cannot exist.

**D2 — Bind by resolve-and-compare, not by ID reference.** Before
each run, the broker resolves the tag to its image ID and refuses
unless it equals the ID recorded at the verified build. This closes
the real hole (stale/mismatched image under a reused tag) using
only already-used operations, with no new Docker-behavior
dependency. Daemon-side TOCTOU is out of the threat model (daemon
trusted) and recorded as such.

Supporting decisions: fd-based capped claim with dev/ino re-verify
and O_EXCL sinks (M1/M2/L3); sweep validates residue and never
overwrites terminal results (M4); results anchored owner-side with
a `verify` subcommand, repo copies explicitly untrusted (M3);
`StatusUnknown` for every indeterminate outcome (L4); config
placement/symlink/ownership guards incl. QueueDir resolution (M5);
strict decoder minus the `\n` false positive (L1/L2); disarm
provenance record (L5); broker-passed toolchain paths (L6).
I-residuals stay deferred except I6 (install procedure) and I4
(accepted doctrine per prompt4); I5 is superseded by D1 and
recorded as such.

## Steps

### U1 — H1/H2 + digest binding (build integrity)

- Replace `PinInputs` with a full flat-context walk: every regular
  file pinned, symlinks/subdirs/unexpected names refused.
- Stage to `StateDir/stage/<jobid>/`: copy, re-hash, compare; build
  the image from the stage only. Per-run repo snapshot (minus
  `.agent-jobs/`, `test-results/`, symlink scan refuses) mounted
  read-only; tree hash recorded.
- Record image ID at build; run resolves tag→ID and refuses on
  mismatch or missing build.
- Tests: P7 plant refused; unexpected-file refusal; snapshot
  independence after repo tamper; bounded swap loop always
  self-consistent; resolve-mismatch refusal.

### U2 — M1/M2/L3 (claim rewrite)

- `openat(O_RDONLY|O_NOFOLLOW|O_NONBLOCK)` → fstat regular-only →
  `LimitReader` cap → O_EXCL `running/` create → dev/ino re-verify
  → unlink; skip-and-continue past poison (no head-blocking);
  best-effort quarantine with unique nanos names.
- Tests: P3 swap loop, zero outside-byte hits; FIFO swap never
  hangs; oversize refused with queue live; symlink never followed.

### U3 — M4/L4 (sweep + status vocabulary)

- Sweep skips `done/`-terminal ids with warning; parses residue
  (op+size into UNKNOWN) or quarantines it loudly.
- `unknown` for timeouts, partial sweeps, build-ok/inspect-fail;
  `error` reserved for determinate failure.
- Tests: three sweep cases; timeout/partial/inspect-fail → unknown.

### U4 — M3/L5 (anchors + provenance)

- `Finish` writes the state-side anchor first, then repo `done/`
  via no-clobber link (EEXIST preserves owner-side + warns).
- New `verify` subcommand compares repo copies against anchors.
- Disarm writes `disarm.json` (git HEAD/status, package counts
  best-effort, image ID, timestamp); absence recorded, never fatal.
- Tests: anchor equality; tampered copy detected by `verify`;
  disarm record incl. git-absent branch.

### U5 — M5 (config integrity)

- Refuse StateDir/EnvFile under (or symlinked into) RepoRoot in
  both directions; EvalSymlinks on all three dirs; EnvFile
  euid-owned, no `.git` ancestor; StateDir not world-writable.
- Tests: each refusal plus the positive case.

### U6 — L1/L2 (decoder)

- Trailing-data EOF check; top-level duplicate-key rejection via
  token scan; drop `"\n"` from the banned list.
- Tests: trailing/dup rejected; pretty-printed + trailing newline
  accepted.

### U7 — L6 (entrypoint portability)

- Broker passes `GOMODCACHE`/`GOROOT` via `-e` from config;
  entrypoint requires and verifies them; `TestArgvFixed` updated.
- Validation: `sh -n`, focused pattern test (documented as the
  limit of shell coverage without a harness).

### U8 — Docs + gates + report

- Update `docs/host-exec-broker.md` (snapshot design, provenance
  acceptance, install controls) and `docs/test-runner.md`.
- Run gofmt/vet/build, full serial suite with exact counts, race
  on `hostexec`/`testrunner`; deliver the six required report
  items (files, mapping, repro evidence, counts, risks, verdict).

## Validation Plan

- Per unit: focused `go test ./hostexec/ -run <Test> -count=1`
  (fake-docker fixtures; no daemon needed) — each new test fails
  on the pre-fix code path by construction (swap/plant/tamper
  fixtures).
- `sh -n testrunner/entrypoint.sh` for U7.
- Final: `gofmt -l .`, `go vet ./...`, `go build ./...`,
  `go test -p 1 ./...` with exact counts (baseline today: 36/36
  in `hostexec`+`testrunner`, full suite exit 0),
  `go test -race ./hostexec/ ./testrunner/`.
- Highest-risk validation: the U1 snapshot-consistency loop and
  U2 P3-zero-hit loop (concurrency proofs, not single shots).

## Risks / Open Questions

- Snapshot/provenance adds seconds per run (repo ~10MB); measured
  at verification, bounded by design.
- Git-absent environments degrade provenance to recorded-absence;
  never fatal, always explicit.
- Resolve-compare TOCTOU vs a malicious daemon is out of threat
  model (daemon trusted); stated, not solved.
- H1b/L5 ultimately rest on owner diff review of frozen
  provenance — a technical refusal is impossible and not claimed.
- Open questions: None. Every design input verified locally
  (code regions, baseline counts, repo size, symlink absence).
