# Adversarial Security Review — Host Execution Broker (prompt3)

## Executive verdict: BLOCKED

The broker's **Docker-operation confinement holds**: every `docker` argv is
fixed at the broker side, no job-supplied value reaches an argument vector,
and the nine-operation allowlist cannot be widened from the queue. But the
**execution-integrity boundary is broken in two demonstrated ways**: the
build-pin set does not cover the full build context (H1, deterministic,
no race needed), and pin verification races the build over an
agent-writable path (H2). Either lets an untrusted agent decide what code
runs inside the authorized `image.build` / `container.run` operations —
forging a green suite, exfiltrating the test DSN into agent-readable
results, and flipping the one-way auto-disarm. Those two findings alone
block installation. The queue TOCTOU/unbounded-read pair (M1/M2) adds a
trivial persistent-DoS chain, and the evidence-integrity findings (M3/M4)
mean owner acceptance would be read from forgeable files.

Direct answer to §8: an adversarial agent **cannot** make the broker run
an unlisted Docker operation, but it **can** corrupt what executes inside
the listed operations and forge the resulting evidence — so installation
is BLOCKED until H1/H2 (and, for a trustworthy acceptance, M3/M4) are fixed.

No remediation was performed: the session instruction ("do not modify any
codebase") overrides prompt3 §6. Smallest durable corrections are given
per finding instead. No broker installation, service start, or Docker
infrastructure operation was performed.

## Scope reviewed

- `hostexec/policy.go`, `hostexec/queue.go`, `hostexec/exec.go`,
  `hostexec/serve.go`, `hostexec/hostexec.service`,
  `hostexec/{policy,queue,exec,serve}_test.go`
- `cmd/hostexec/main.go`
- `testrunner/{Containerfile,entrypoint.sh,main.go,relay.go,netcheck.go,relay_test.go}`
- `docs/host-exec-broker.md`, `docs/test-runner.md`, `testdb/testdb.go`
  (DSN-echo check only), `.gitignore`
- Out of scope / untouched: live Docker daemon, PG19, systemd install.

## Confirmed findings

### H1 — HIGH: pin set incomplete; unpinned files in the build context are compiled in

- **Files:** `hostexec/policy.go:206-214` (`PinInputs`, 5 files);
  `hostexec/serve.go:150-163` (pin gate); `testrunner/entrypoint.sh:34,67`
  (`go build` / `go test ./...`); `hostexec/exec.go:117` (context =
  `<repo>/testrunner`).
- **Preconditions:** workspace write (granted). No race, no timing.
- **Attack:** (a) add `testrunner/pwn.go` (`package main`, `init()` posts
  `TEST_DATABASE_URL` to `/results/` and/or forces exit 0); submit
  `image.build` — the 5 pinned hashes still match, build succeeds,
  malicious code is in the image. (b) Tamper with the already-present
  unpinned `testrunner/relay_test.go`: it executes under `go test ./...`
  in run mode with the DSN in env.
- **Impact:** forged green suite → premature auto-disarm (privileged
  state flip + deployment DoS); test-DSN exfil into agent-readable
  `test-results/`; arbitrary code in the runner container.
- **Evidence (demonstrated):** P7 probe — added a marker `pwn.go` to a
  copy of the runner sources, `go build`, marker present in binary
  (`strings | grep -c` = 1). `relay_test.go` exists in-context and is
  absent from `PinInputs()` by inspection; it runs (`testrunner` suite
  passes, 8/8).
- **Root cause:** pin coverage was enumerated as "the 5 known files"
  instead of "the whole build context".
- **Correction:** hash the entire context: walk `<repo>/testrunner`,
  pin every regular file (fail on unexpected names), or stage verified
  bytes to owner-side `StateDir` and build from the stage (also fixes H2).

### H2 — HIGH: pin check races the build over an agent-writable path (TOCTOU)

- **Files:** `hostexec/serve.go:151` (`HashPinInputs(RepoRoot)`) then
  `hostexec/exec.go:117` (`docker build … <repo>/testrunner`).
- **Preconditions:** workspace write. The window between hash and the
  daemon reading the context is milliseconds-to-seconds and re-arms on
  every `image.build` job, so a swap loop wins reliably.
- **Attack:** loop `rename()` between a pristine and a backdoored
  `testrunner/entrypoint.sh`; submit `image.build` repeatedly. Any pass
  of the hash followed by the daemon reading the backdoored bytes bakes
  the attacker's entrypoint into `studio-test-runner:1`.
- **Impact:** same as H1 (image content fully attacker-chosen).
- **Evidence:** code-level (two separate operations, no lock/snapshot,
  agent-writable path throughout). The P3 probe proves the sibling
  queue race is winnable at 12%/attempt with a naive loop; this window
  is orders of magnitude wider.
- **Root cause:** verify-then-use on untrusted storage without
  snapshotting.
- **Correction:** after hashing, copy the pinned tree to an owner-side
  staging dir under `StateDir`, re-hash the copies, and `docker build`
  the stage. One fix covers H1+H2 if the stage is populated by an
  allowlisted full-context walk.

### M1 — MEDIUM: `Claim` Lstat→ReadFile TOCTOU; symlink/FIFO swap wins

- **Files:** `hostexec/queue.go:78-108` (Lstat guard, then
  `os.ReadFile(src)` + `os.Rename` on the path, not the handle).
- **Preconditions:** workspace write; naive rename loop.
- **Attack:** alternate `pending/v.json` between a regular job and a
  symlink (or FIFO) while the broker claims. Lstat passes on the
  regular file; `ReadFile` follows the swapped-in symlink, or blocks
  forever on a FIFO (single-threaded broker, no read timeout — full
  wedge until owner intervenes).
- **Impact:** out-of-queue file content read into job bytes (limited
  oracle via reject reasons); reliable broker wedge via FIFO (DoS);
  executed bytes diverge from the `running/` file (audit mismatch).
- **Evidence (demonstrated):** P3 probe, 3000 claims under swapping:
  `outside-byte hits=370 regular=1415 empty=1215`. Every hit is a
  claim whose bytes came from the symlink target after the Lstat guard
  passed — the guard is bypassable at will.
- **Root cause:** path-based check-then-use instead of handle-based.
- **Correction:** `open(O_NOFOLLOW)` → `fstat` (regular only) →
  size-capped read from the fd → `rename`. One change also fixes M2.

### M2 — MEDIUM: unbounded job read before the size check; OOM crash loop

- **Files:** `hostexec/queue.go:105-119` (`os.ReadFile` then
  `len(raw) > MaxJobBytes`).
- **Preconditions:** workspace write. No race needed.
- **Attack:** write one large regular `pending/big.json` (e.g. 1 GiB).
  The broker reads it fully; against `MemoryMax=512M` it is OOM-killed,
  systemd restarts it, it re-reads the same file — persistent crash
  loop until the owner deletes the file.
- **Impact:** durable broker DoS with one file write.
- **Evidence (demonstrated):** P2 probe — a 32 MiB file is read in
  full; the quarantine error echoes the exact size (`oversized:
  33554432 bytes`), proving no capped reader is in the path.
- **Root cause:** size enforced after allocation instead of during read.
- **Correction:** `io.LimitReader(fd, MaxJobBytes+1)`; refuse when the
  cap is exceeded without ever allocating the excess.

### M3 — MEDIUM: verdicts live in agent-writable space; evidence forgeable post-hoc

- **Files:** `hostexec/queue.go:127-154` (`done/` under repo);
  `testrunner/entrypoint.sh:58-75` (`test-results/` under repo);
  `docs/host-exec-broker.md:87-89` (acceptance reads those files).
- **Preconditions:** workspace write (same uid).
- **Attack:** after (or without) any run, rewrite `done/<id>.json` and
  `test-results/{exitcode,summary.txt,gotest.json}` to taste. The
  broker never re-reads them for state (disarm uses the in-memory exit
  code — that part is safe), but every owner acceptance step ("repeat
  run identical", "exact counts") reads forgeable bytes. Bonus: squat
  `done/<future-id>.json` to force duplicate-rejection of that id
  (targeted job DoS via `queue.go:92-103`).
- **Impact:** owner verification and audit trail untrustworthy.
- **Evidence:** code + layout (both paths under `RepoRoot`, broker
  same-uid; `.gitignore` confirms runtime-state status, not protection).
- **Root cause:** no owner-side copy or integrity anchor for verdicts.
- **Correction:** on `Finish`, also write the result (or its sha256)
  to owner-side `StateDir`; acceptance compares repo copies against it.

### M4 — MEDIUM: `SweepStartup` trusts agent-writable `running/`; rewrites finished verdicts

- **Files:** `hostexec/queue.go:171-194`.
- **Preconditions:** workspace write + any broker restart (threat model
  grants host restart).
- **Attack:** plant `running/w.json` for an id whose `done/w.json` is
  already `ok`; wait for a restart. The sweep overwrites the finished
  verdict with `UNKNOWN`. Planting an id that never ran fabricates a
  verdict from nothing — no `ParseJob`, no op recorded (`"op": "?"`).
- **Impact:** downgrade/erase of finished results; fabricated verdicts.
- **Evidence (demonstrated):** P4 probe — pre-existing
  `done/w.json={"status":"ok"}` became full `UNKNOWN` record after
  `SweepStartup`; planted `running/ghost1.json` produced
  `done/ghost1.json` (`UNKNOWN`) for a job that never executed.
- **Root cause:** sweep never checks `done/` for a terminal result and
  never validates the residue.
- **Correction:** skip (with a warning) any residue whose `done/<id>.json`
  already holds a terminal result; record the residue's parsed op and
  size in the UNKNOWN record; ignore unparseable residues loudly.

### M5 — MEDIUM: `LoadConfig` accepts owner-side paths inside the agent-writable repo

- **Files:** `hostexec/exec.go:41-82` (checks `QueueDir` under repo,
  never checks `StateDir`/`EnvFile` are outside it).
- **Preconditions:** one misconfigured install (no warning issued).
- **Attack:** with `state_dir` under the repo, the agent edits `armed`
  (re-arm at will), and with `env_file` under the repo the DSN file
  sits in committable, agent-readable space; both silently void the
  "owner-side" trust the code comments assume.
- **Impact:** authorization bypass (re-arm) + secret mishandling.
- **Evidence (demonstrated):** P6 probe — config with `state_dir` and
  `env_file` inside the repo: `LoadConfig` accepted, `err=<nil>`.
- **Root cause:** one-sided placement check.
- **Correction:** refuse `StateDir`, `EnvFile` (and lock/state files)
  under `RepoRoot`, mirroring the existing `QueueDir` check; refuse
  `EnvFile` unless 0600 (already done) and outside version control.

### L1 — LOW: decoder accepts trailing garbage and duplicate keys

- **Files:** `hostexec/policy.go:122-126` (single `Decode`, no EOF check).
- **Evidence (demonstrated):** P1 probe — `{"id":"a1","op":"version"}
  TRAILING` ACCEPT; second-JSON-object trailing ACCEPT; duplicate
  `"op"` (last-wins → `net.create`) ACCEPT; duplicate `"id"`
  (last-wins) ACCEPT. Oversize/off-schema cases still reject.
- **Impact:** strictness-claim violation; audit confusion (executed op
  differs from a naive first-key reading). No privilege gain: the
  surviving value is still allowlisted.
- **Correction:** after the first `Decode`, a second decode must return
  `io.EOF`; reject duplicate keys (decode into `map[string]any` first
  and compare, or token-scan).

### L2 — LOW: banned `\n` rejects ordinary job files (trailing newline, pretty print)

- **Files:** `hostexec/policy.go:88-92,184-192`.
- **Evidence (demonstrated):** P1 probe — valid job + trailing newlines
  REJECT (`banned token "\n"`). Any `echo '{…}' > pending/x.json` or
  editor-saved file fails; undocumented in `docs/host-exec-broker.md`.
- **Impact:** interop/robustness (fail-closed, but hostile to normal use).
- **Correction:** drop `"\n"` from the banned list — newlines cannot
  reach argv since values are enum/charset-pinned — or trim trailing
  whitespace before the check and document single-line-canonical form.

### L3 — LOW: duplicate quarantine is last-write-wins in `rejected/`

- **Files:** `hostexec/queue.go:92-103` (`os.Rename` overwrites).
- **Evidence (demonstrated):** P5 probe — two sequential duplicates;
  `rejected/a1.json` holds the second body, the first is gone.
- **Correction:** quarantine to `a1.json.<nanos>` or refuse to
  overwrite an existing quarantine entry.

### L4 — LOW: ambiguous outcomes report `error`, never `unknown`

- **Files:** `hostexec/serve.go:179-180,193-197,221,236`
  (only `SweepStartup` uses `StatusUnknown`).
- **Impact:** §F asks ambiguous outcomes to read `UNKNOWN`. Timeouts
  after partial Docker success, partial sweep removals, and
  "build ok but image inspect failed" are honestly non-OK but use the
  wrong vocabulary, and agents keying on `unknown` for reconcile will
  mis-handle them. The reconcile guidance text is present — this is a
  contract-consistency fix.
- **Correction:** use `StatusUnknown` for timeout/partial/indeterminate
  outcomes; reserve `error` for determinate failure.

### L5 — LOW: disarm equates container exit 0 with a "genuinely successful full suite"

- **Files:** `hostexec/serve.go:137-145`; suite content = agent-writable
  repo, tests unpinned.
- **Impact:** an agent can gut/bypass tests to force exit 0 → hollow
  green → premature disarm + misleading evidence. The only control is
  owner review of the repo diff, and the broker records nothing at
  disarm time (one log line) to support that review.
- **Correction:** at disarm, persist `git HEAD`, `git status --porcelain`,
  and `gotest.json` package pass counts into `StateDir`; make them
  acceptance inputs in `docs/host-exec-broker.md`.

### L6 — LOW: entrypoint hardcodes a one-machine module-cache path

- **Files:** `testrunner/entrypoint.sh:27`
  (`GOMODCACHE=/home/chaschel/go/pkg/mod`) vs broker-mounted
  `c.ModCache` (`hostexec/exec.go:133`).
- **Impact:** on any other machine/user the mounted cache is ignored
  and the hermetic build (`GOPROXY=off`) fails — fail-closed, but a
  portability/reliability defect in the deployment path.
- **Correction:** probe a short candidate list (mount path-derived)
  and use the first existing dir, or have the broker pass the path via
  a broker-written file in the results mount… simplest is entrypoint
  probing with loud failure listing candidates.

## Informational / accepted residuals

- **I1** `inspect.readonly:net.ls` discloses unrelated network names to
  the agent (read-only, needed for reconcile; accept + note).
- **I2** Runner runs as root, no `--user`; results mount rw → root-owned
  files in the repo. Consider `--user`, tmpfs `GOCACHE`.
- **I3** No rate limit on steady-state `container.run` — run-spam is a
  local resource DoS. Accept for owner-gated local use.
- **I4** Full container output lands unscrubbed in `test-results/` while
  broker tails are scrubbed (`Scrub`, `exec.go:172`). Benign suite
  verified not to echo credentials (`testdb` errors name host/port/db
  only). A malicious agent-authored test printing the DSN is inherent
  to handing the DSN to agent code — the real boundary is cluster
  scoping + rotation; document it as accepted.
- **I5** `go.mod`/`go.sum` unpinned — analyzed fail-closed
  (`GOPROXY=off`, `--internal` net); still, add to the pin set with H1.
- **I6** `hostexec pin` blesses whatever tree it reads — procedural
  control; require git-clean + known-commit + review before recording.
- **I7** `container.run` uses the image tag, not the digest recorded at
  build; re-pointing needs a host-side Docker actor (trusted). Note.
- **I8** Unit is a solid baseline (`NoNewPrivileges`, `ProtectSystem=strict`,
  `AF_UNIX|AF_NETLINK`-only, `MemoryMax`). Optional: `ProtectKernelTunables`,
  `ProtectControlGroups`, `PrivateDevices`, `SystemCallFilter`.
- **I9** Checked, not a finding: `\uXXXX`-escaped banned tokens bypass
  the raw-bytes backstop but decoded values are charset/enum-pinned
  (P1 `semicolon-u-escape` still rejected) — harmless as designed.
- **I10** Stated assumption: config/state/pin integrity ultimately rests
  on the agent sandbox confining writes to the workspace (same uid).
  Correct install per docs satisfies this; M5 is the guardrail gap.

## What held under attack (§4 mapping)

1. Command injection — argv fully broker-constructed, pinned by
   `TestArgvFixed`; no shell anywhere. HELD.
2. Unknown ops/fields — `DisallowUnknownFields` + allowlists. HELD
   (modulo L1 trailing/dup laxity).
3. Mounts/network modes — constants only; no `--privileged`, host net,
   socket, or published ports in any argv. HELD.
4. Symlink/traversal (static) — non-regular skipped, names never leave
   the scanned dir. HELD statically; M1 breaks it dynamically.
5. Duplicates/replays — quarantined, never re-executed. HELD (modulo L3).
6. Single-flight — single thread + `flock` + `TestServeSecondInstanceRefused`. HELD.
7. Forged disarm via queue — impossible; disarm reads the real exit
   code. HELD as a queue property; H1/H2/L5 attack the exit code's
   meaning instead.
8. Re-arm — no queue path; filesystem-only. HELD given correct install
   (modulo M5).
9. Pin tamper (5 files) — refused, tested (`TestBuildPinDrift`). HELD
   narrowly; H1/H2 bypass outside the 5.
10. Crash → UNKNOWN via sweep. HELD (modulo M4 planting, L4 vocabulary).
11. Broker results scrubbed + bounded (`Scrub`+`Tail`, tested). HELD
    (I4 covers the container-side half).
12. Unit confinement — reviewed, correct for CLI-over-socket (I8).

## Remediation performed

None — forbidden by the session's do-not-modify constraint, which
overrides prompt3 §6. Corrections above are specified to be directly
implementable; H1+H2 share one fix (owner-side staged build from a
fully-pinned context walk), M1+M2 share one fix (handle-based capped
claim). Predicted re-review surface after fixes: `queue.go` claim path,
pin/build path, two regression probes (P3→0 hits, P7→refusal).

## Owner decisions required

1. Accept BLOCKED and schedule the H1/H2 fix before any install GO.
2. Decide M3's evidence anchor (owner-side result copies/hashes) as a
   precondition for trusting acceptance, or accept forgeable evidence.
3. Confirm I4 (DSN-to-suite is inherently agent-visible; boundary =
   scratch-cluster scoping + rotation) as accepted doctrine.
4. Confirm correct-install values so M5's guardrail has a defined
   target (state/env strictly outside the repo).

## Test evidence

| Check | Command | Result |
|---|---|---|
| gofmt | `gofmt -l hostexec/ testrunner/ cmd/hostexec/` | clean (no output), exit 0 |
| vet | `go vet ./hostexec/ ./testrunner/ ./cmd/hostexec/` | clean, exit 0 |
| build | `go build ./...` | clean, exit 0 |
| unit (broker) | `go test -count=1 ./hostexec/` | ok, 28/28 `--- PASS`, 0 fail/skip |
| unit (runner) | `go test -count=1 ./testrunner/` | ok, 8/8 `--- PASS`, 0 fail/skip |
| race | `go test -race -count=1 ./hostexec/` | ok |
| full serial | `go test -p 1 -count=1 ./...` | exit 0, 19 `ok`, 0 FAIL/panic, 3 no-test-files pkgs |
| P1 decoder | `/tmp/hxprobe2/p1.go` | trailing/dup ACCEPT (L1); newline REJECT (L2) |
| P2 oversize | `/tmp/hxprobe2/p245/` | full 32 MiB read, exact-size error (M2) |
| P3 swap race | `/tmp/hxprobe2/p3/` | 370/3000 outside-byte claims (M1) |
| P4 sweep | `/tmp/hxprobe2/p245/` | OK→UNKNOWN overwrite; ghost verdict (M4) |
| P5 dup | `/tmp/hxprobe2/p245/` | last-write-wins (L3) |
| P6 config | `/tmp/hxprobe2/p6/` | in-repo state/env accepted (M5) |
| P7 unpinned | `/tmp/p7mod/` | marker compiled into binary (H1) |

Source-level + fake-Docker (`TestServe*`, fixture script) verification
only. No broker install, no live-Docker, no PG-backed run was performed
or authorized. Probes live outside the repo (`/tmp/hxprobe2/*`,
`/tmp/p7mod`) and are re-runnable; the repo tree was not modified
(`git status` shows only pre-existing plan deletions plus `.gitignore`).

## Installation readiness

NOT READY. Do not install, start, or GO any deployment job until H1/H2
are fixed and P3/P7-style regression probes are green, and do not treat
repo-side verdicts as authoritative until M3/M4 are addressed.

