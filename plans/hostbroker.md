# Coding-Agent Prompt — Broker Bootstrap + Deployment (hands-off)

Use in a fresh session running under a profile that denies the agent
writes to the broker StateDir/EnvFile. If that precondition does not
hold, the agent must stop before installing anything.

## Role and objective

You are the coding agent. Establish the host execution broker and run
the PostgreSQL test-runner deployment through it, with zero owner
command sequences. Follow repo `PRINCIPLES.md`, `SKILLS.md`, and
`AGENTS.md`. Governing design docs: `docs/host-exec-broker.md`,
`docs/test-runner.md`.

## Preconditions (verify, do not assume)

- `StateDir`/`EnvFile` paths are known (from the owner attestation).
- Git tree state recorded (`git status --porcelain`, HEAD commit).

## Phase 0 — Attestation re-check (fail closed)

Attempt to create a file under StateDir and EnvFile's directory.
Both attempts MUST be refused. If either write succeeds: STOP
immediately, report the profile gap, do not install, do not proceed.
Also confirm `extra_agent_roots` covers the sandbox temp roots.

## Phase 1 — Install (all by you)

1. `go build -o ~/.local/bin/hostexec ./cmd/hostexec` from the repo.
2. Write `~/.config/hostexec/config.json`: `repo_root`, `queue_dir`
   (`<repo>/.agent-jobs`), `state_dir`, `env_file` (0600, DSN copied
   by path only — never print or log its value), `mod_cache`,
   `toolchain`, `docker_bin`, `git_bin`, `pins`, `extra_agent_roots`.
3. Record `pins` ONLY from a reviewed tree: `git status` clean (or
   diff fully reviewed), commit recorded, then
   `hostexec pin --repo <repo>` output into config.
4. Install the unit (adjust the two marked paths):
   `cp hostexec/hostexec.service ~/.config/systemd/user/ &&
   systemctl --user daemon-reload &&
   systemctl --user enable --now hostexec`.
5. Confirm active: `systemctl --user is-active hostexec`.

## Phase 2 — Channel proof (before any mutating job)

Submit queue jobs as `<repo>/.agent-jobs/pending/<id>.json`
(`[a-z0-9-]`, unique per job), poll `done/<id>.json` with a timeout:

1. `{"id":"ch-version","op":"version"}` → expect ok + server version.
2. Malformed job (bad JSON) and banned job (unknown op) → both must
   land in `rejected/` with reasons; nothing executes.
3. Absence of evidence is failure: missing result past timeout means
   STOP and diagnose (broker logs via `systemctl --user status`).

## Phase 3 — Deployment gates (in order, verify each)

Submit one job, verify its result, then advance. Any unexpected
output, nonzero exit, UNKNOWN status, or conflict file means STOP.

1. `image.build` → ok + recorded image ID. Verify pins matched
   (no drift error).
2. `net.create`, then `net.connect` → ok.
3. `container.run {"mode":"netcheck"}` → ok with every probe
   blocked. ANY reachable gateway/external/DNS/socket means STOP.
4. `container.run {"mode":"run"}` → full suite. Require zero
   PostgreSQL-connectivity skips; record exact counts.
5. Owner-side `hostexec verify --config` must be clean (owner runs
   this one command: anchors are unreadable from the sandbox by
   design). Then review `disarm.json` (frozen git state, image ID,
   manifest, package counts, results hashes) + the repo diff.
6. Post-run: peer lists, mapping intact, orphans empty, repeat run
   identical, deploy ops disarmed (prove by submitting `net.create`
   and observing rejection).

## Standing rules

- Never print, log, or persist credential values; DSN travels by
  file path only.
- Never weaken checks, skip failing tests, or reinterpret a red
  gate as green. Unknown = stopped, not passed.
- `BOUNDARY_*` stays unset everywhere in this flow.
- On ANY surprise: stop, preserve all evidence (job files,
  results, broker status output), report exactly what diverged.
- Final report: commands run, per-gate results with exit codes and
  counts, `verify` outcome, disarm record summary, residuals, and
  an explicit installation-readiness statement.
