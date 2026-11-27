# Host Execution Broker

**Status:** Remediated per `plans/prompt4.md` (adversarial findings
H1/H2/M1–M5/L1–L6 + digest binding). Installation, service startup,
and all Docker operations are NOT authorized yet.

## Privilege boundary (read this first)

The broker is a **privileged security boundary**: it runs as the
owner with Docker access and executes operations the agent must
never perform directly. Its narrowness is load-bearing:

- The agent holds NO Docker access, before or after installation.
  It writes strictly validated job files and reads result files;
  both cross the boundary as inert workspace data.
- The broker executes ONLY the nine typed operations in
  `hostexec/policy.go`, with fixed arguments it constructs itself.
  There is no generic command execution, no shell, no
  job-supplied paths, and no secret values in jobs.
- Compromise impact if the agent side is subverted: at most the
  allowlisted deployment/recovery operations, each bounded,
  logged, and auditable in `done/`. The queue cannot widen what
  the broker will do — that requires owner-side code or config
  changes under review.
- The systemd unit further confines the broker process itself
  (see `hostexec/hostexec.service`: no new privileges, strict
  filesystem, unix-socket-only networking).

## Design

- Queue: `<repo>/.agent-jobs/{pending,running,done,rejected}/`
  (gitignored). Agent writes `pending/<id>.tmp.*`, renames to
  `pending/<id>.json`; broker claims (handle-based: dirfd +
  `O_NOFOLLOW`, fstat, capped read, dev/ino re-verify), executes,
  publishes `done/<id>.json` (temp + no-clobber link; squats land
  under conflict names, never overwrite).
- Validation: 64KB cap enforced during the read, id charset
  `[a-z0-9-]{1,64}`, strict JSON (unknown fields, trailing data,
  and duplicate keys refused), enum-pinned ops/params, banned-token
  backstop, duplicate ids quarantined under unique names.
- Two snapshot domains (H1/H2 closure): INFRA inputs (the flat
  `testrunner/` context) carry install-time pins over the FULL
  walked set — unexpected, missing, or altered files refuse the
  build; the verified bytes are copied to an owner-side stage,
  re-hashed, and the image builds from the stage only. RUN inputs
  (the whole repo) are snapshotted per job (minus queue/results,
  symlinks refused) and the container mounts the snapshot
  read-only — never the live tree. Mid-copy replacement is detected
  and refused loudly (never silently half-copied), so repo mutation
  can delay a run but never corrupt one; post-hash tamper cannot
  affect a build.
- Digest binding: each verified build records its image ID
  owner-side; every run resolves the tag and refuses unless it
  matches (stale tags and missing builds fail loudly). The
  image-build argv names its Containerfile explicitly (`-f`,
  absolute stage path — Docker defaults to `PATH/Dockerfile`).
  Resolve-then-run tag TOCTOU is accepted under the trusted-daemon
  model (only a Docker-capable actor could exploit it).
- Evidence: every result is anchored owner-side (self-verifying
  single file: hash line + payload, O_EXCL first-wins — never
  overwritten, never a hardlink) before the repo copy publishes.
  Claims and sweeps consult anchors, so deleting a repo copy can
  neither free its id nor erase history; collisions preserve
  under conflict names, loudly. Repo copies are untrusted
  projections; `hostexec verify --config` enumerates anchors and
  reports mismatch/missing/conflict/corrupt.
- Disarm provenance: the first green suite run persists
  `disarm.json` (job, git HEAD/status read from the executed
  snapshot, image ID, run manifest, package counts, and sha256 of
  `gotest.json`/`summary.txt`/`exitcode` as observed) and disarms
  deploy-only ops (`net.create/connect`, `image.build`,
  `net.disconnect/remove`). Hashes establish integrity FROM
  capture, never authenticity before it; gaps (missing/unreadable/
  oversized) are recorded and must block install acceptance.
  Steady-state ops stay armed for repeatable runs. Re-arm is
  owner-side only. A green exit plus frozen provenance still
  requires owner diff review: test-gutting is indistinguishable
  from legitimate dev by inspection of a dirty tree. The run
  manifest describes executed bytes; quiesce the tree for a
  meaningful run.
- Timeouts, partial sweeps, and indeterminate outcomes report
  `UNKNOWN` (reconcile before retry); `error` is reserved for
  determinate failure. Crashes convert `running/` residue to
  UNKNOWN only after validating it and confirming no terminal
  verdict exists.

## Install (one-time, owner — NOT yet authorized)

0. Attest the trust geography (P5-MED-2, fail closed): record the
   exact StateDir/EnvFile, demonstrate agent-side writes there are
   refused, and confirm `extra_agent_roots` covers the sandbox's
   temp roots. No attestation → no install. If the write-refusal
   demonstration FAILS (writes succeed): the session profile does
   not confine the agent away from those paths — do not install.
   Resolve by running sessions under a profile that denies them,
   restart, and re-attest. Placement alone is not a substitute:
   demonstrated 2026-11-26 that home paths are agent-writable
   under the default profile.
1. Review `hostexec/` source and this doc.
2. `cd <repo> && go build -o ~/.local/bin/hostexec ./cmd/hostexec`
3. `mkdir -p ~/.config/hostexec` and write `config.json`:
   `repo_root`, `queue_dir` (`<repo>/.agent-jobs`), `state_dir`,
   `env_file` (0600 DSN file), `mod_cache`, `toolchain`,
   `docker_bin`, `git_bin`, `pins`, `extra_agent_roots`.
   Rules enforced at startup (fail closed): state/env outside the
   repo AND outside every agent root (`/tmp`, `/dev/shm`, declared
   extras), symlinks resolved before comparison, env file
   owner-owned/0600 and outside any checkout, state not
   world-writable.
4. Record pins ONLY from a reviewed tree: `git status` clean (or
   diff fully reviewed), known commit recorded, then
   `hostexec pin --repo <repo>` output into config. Never bless
   an unreviewed tree — pins are the trust root for builds.
5. Install the unit (adjust the two marked paths):
   `cp hostexec/hostexec.service ~/.config/systemd/user/ &&
   systemctl --user daemon-reload &&
   systemctl --user enable --now hostexec`.
6. Verify the channel with a read-only `version` job before any
   mutating job; confirm broker self-test behavior (malformed and
   banned jobs rejected) from `rejected/`.

## Operations (all via queue jobs; deployment sequence needs its GO)

Each gate is one job file; the agent polls `done/<id>.json` with a
timeout, verifies the result, and submits the next job or stops.
Exact argv per op is pinned by `TestArgvFixed`; results carry exit
codes, bounded scrubbed streams, image ID, and run manifest.

## Verification (acceptance order, at deployment)

1. Broker self-tests: malformed/banned/oversized/duplicate jobs
   rejected; unknown field rejected; second instance refused
   (unit-covered; re-verify live at install).
2. `version` job ok (channel proof, zero infra change).
3. Deployment gates through the broker: build (pins verified,
   digest recorded) → net.create → net.connect → netcheck (all
   probes blocked) → suite run (zero PG-connectivity skips).
4. `hostexec verify --config` clean: every repo verdict matches
   its anchor; conflicts/absences fail the acceptance.
5. Disarm record reviewed: frozen git state, diff inspected,
   package counts sane — THEN the green is trusted.
6. Post-run: peer lists, mapping intact, orphans empty, repeat
   run identical, deploy ops disarmed (prove by submitting
   `net.create` and observing rejection).

## Recovery

- Broker crash: systemd restarts; startup sweep marks in-flight
  UNKNOWN; agent reconciles before retry.
- Job timeout: broker reaps `studio-test-*` orphans best-effort
  and records timeout+reconcile guidance.
- Full rollback: `container.rm`, `net.disconnect`, `net.remove`
  jobs (while armed) or equivalent direct owner commands;
  `pg_isready` proves PG healthy after every step.
- Re-arm (owner only): `echo deploy >
  <stateDir>/armed` (or restart after editing); record why.
