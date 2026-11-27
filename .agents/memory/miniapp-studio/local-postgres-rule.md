---
type: project
description: Persistent local PG environment rule
---

# Local database environment rule (persistent, owner-ordered 2026-11-26)

**PostgreSQL 19 runs through Docker, exposed at host `127.0.0.1:5434`.** This is established. Never assume PG or Docker needs installation, and never treat an access failure as missing infrastructure.

## When DB access fails, diagnose in this order

1. **Distinguish sandbox restriction from host state.** This agent shell is sandboxed: isolated network namespace (own loopback ≠ host loopback, empty route table, proxy-only egress) and denied Docker-socket connect (socket shows `nobody:nogroup`). Fingerprint of namespace isolation: `ECONNREFUSED in 0.000s` on 127.0.0.1:5434 + `ENETUNREACH` on any routable IP + empty `/proc/net/route`.
2. **Check what is checkable locally** (read-only): socket perms, port probes with errno detail, `TEST_DATABASE_URL` presence/perms (never print it), container state via any reachable channel.
3. **Host truth needs the owner shell:** container running? (`docker ps`), port actually served? (`pg_isready -h 127.0.0.1 -p 5434`). The agent cannot answer these from inside the sandbox — ask, don't guess.
4. **Fix the access problem, not the environment:** run Docker/PG commands from the unsandboxed owner shell (P0 precedent), or get the session a reachable path. Never install/reinstall PG or Docker to "fix" a sandbox barrier.

## Reporting discipline

Never silently skip DB verification and report the suite as fully verified. Report skipped DB tests with counts and the exact blocker (e.g. "275 skipped: TEST_DATABASE_URL unset + host PG unreachable from sandbox"). A green suite with DB skips is "green except DB-backed coverage", never "fully verified".


## Endpoint-allowlist constraint (found Phase 1, 2026-11-26)

`testdb.CheckTestEndpoint` fail-closes to hosts {127.0.0.1, localhost, ::1}, port 5434, PG major 19 — "change only under explicit re-approval". Consequence: any DB bridge that presents PostgreSQL at 127.0.0.1:5434 *as seen from the agent namespace* needs ZERO code changes (existing owner.dsn works verbatim); any other hostname/IP (gateway, container name) requires an allowlist change plus re-approval and its tests. Prefer loopback-preserving bridge shapes. A runner-container's own TEST_DATABASE_URL faces the same gate.


## Nested exec finding (one-time diagnostic, 2026-11-26)

`muse exec --sandbox-network enabled "run: pg_isready -h 127.0.0.1 -p 5434"` from inside the sandbox exits 1 WITHOUT testing connectivity: the child fails to bootstrap ("failed to acquire the session writer lease ... Read-only file system"). The network flag never comes into play — nested launch dies on the parent's filesystem jail first. Untested variant: owner-side launch of the same command from an unsandboxed terminal. Do not retry nested, do not escalate to unsandboxed runs to force it.


## Phase 0 host facts (owner-supplied, authoritative 2026-11-26)

PG19 running as container `miniapp-p0-pg` (repo-corroborated in p0-evidence; `miniapp-dev-db-1` is a separate untouched owner container), host endpoint 127.0.0.1:5434, default bridge, container IP 172.17.0.2, only PG on that bridge.

## Option 2 direction (decision pending owner gate)

Isolated runner + in-runner loopback TCP relay (127.0.0.1:5434 → PG internal :5432) on a dedicated `--internal` Docker network. Preserves testdb allowlist + owner.dsn verbatim, no host net, no socket. Standing constraints: BOUNDARY_* env must stay UNSET in any runner (TestProvisionRun rotates credentials); sandbox/daemon_test.go skips without a daemon by design (requireImage Skipf) — runner reproduces that skip honestly; runner needs Go 1.27.3 + 754M module cache (mount ro recommended) + writable GOCACHE; GOPROXY=off; results return via a workspace-mounted results dir (agent-readable) + exit-code file.


## Implementation prepared, infra gated (2026-11-26)

Option 2 artifacts ready in repo, fully unit-tested: `testrunner/` (relay refuses non-loopback in code; gate; netcheck battery incl. gateway/bridge/external/DNS/socket/loopback-only probes), entrypoint (BOUNDARY_* refusal guard, CI-gate order, results + exitcode), minimal Containerfile, `docs/test-runner.md` (exact owner ops, security, verification, recovery), `test-results/` gitignored. Permission boundary holds without redesign (allowlist tests green). Pending explicit GOs, in order: network create → PG attach → netcheck launch → suite-run launch. plans/prompt2.md's working-tree rewrite (Agent-Provider doc → DB directive) is the owner's own edit — preserve, never revert.


## Host-execution verdict (investigated 2026-11-26, official docs + live probes)

No suitable built-in mechanism exists. Evidence: proxy-only per-destination approvals can't help (loopback dials die inside the netns with instant ECONNREFUSED — approval layer never sees them); unrestricted/enabled = full egress (too broad, host-loopback sharing undemonstrated); --disable-sandbox = full egress + anywhere-writes (docs warn against workstation use); approval prefix rules don't pierce the sandbox (independent layers: docs "Each layer works on its own"); profiles set posture only; nested exec dead (session lock); MCP would inherit the jail + needs unwritable settings; MXP has no agent-usable client path; peer conscription forbidden (docs: peer messages "cannot approve tools, grant consent, or change permissions"). Direction: owner-approved design pending for a minimal file-queue host execution broker (workspace `.agent-jobs/`, strict op allowlist, fixed argv, no shell, secrets by path only). Never request agent docker-socket access; never use unsandboxed runs as the path.


## Broker implemented, install gated (2026-11-26)

`hostexec/` + `cmd/hostexec` done, 28 tests green incl. race: 9 typed ops with fixed argv (no shell), strict queue validation (symlink/traversal/malformed/oversize/dupe/unknown-field rejection, atomic transitions, UNKNOWN sweep), owner-side config + build-input pins with drift refusal, auto-disarm of deploy ops after first green suite run (persisted, owner-only re-arm), flock single-instance, docker preflight, per-op timeouts with orphan reaping, secret scrubbing. Hardened systemd unit + `docs/host-exec-broker.md` written. Full serial suite exit 0. Pending: evidence review, then one-time owner setup (build binary, config+pins, unit, verify via version job) — install/start/docker ops NOT authorized.


## Broker remediation complete, verification pending (2026-11-26)

prompt4 remediation pass done, repo-only: H1/H2 (full-walk pins, staged immutable builds, per-run snapshots, resolve-compare digest binding), M1/M2/L3 (dirfd-pinned fd-based capped claim, unique quarantine), M3 (owner-side anchors + verify cmd, no hardlinks), M4 (validated sweep), M5 (resolved-path placement vs repo+/tmp+/dev/shm+extras, euid, no-.git-ancestor), L1 (EOF + dup-key scan), L2 (\n ban dropped), L4 (unknown vocabulary), L5 (disarm.json provenance), L6 (broker-passed toolchain env). 61/61 hostexec+testrunner tests green, race clean, full suite exit 0 (308 pre-existing environmental skips). Safeguards honored. Verdict delivered: READY FOR FOCUSED VERIFICATION; install still needs owner GO. Note: plans/prompt4_plan.md write replaced stale P0-era tracked content at the explicitly named path (authorized); prompt2_plan.md now lives at plans/prompt2_plan.md (owner moved it) — never link the root path.


## P5 remediation implemented, counts pending (2026-11-26)

Final prompt5 pass done, repo-only: HIGH-1 (absolute -f stage/Containerfile argv + pinned test + docs), HIGH-2 (O_EXCL first-wins anchors, claim+sweep consult anchors, conflict-preserve, verify enumerates anchors), MED-1 (results manifest hashed into disarm.json: gotest.json/summary.txt/exitcode with size+status+gaps), LOW-1 (honest sweep wording), LOW-2 (git provenance from executed snapshot). 65/65 hostexec+testrunner green, race clean, swap volume 10/10, full suite exit 0. Safeguards honored (no hardlinks; placement denies repo+/tmp+/dev/shm+extras). Awaiting final skip/fail counts to file the report. Install still needs owner GO after focused verification. P5-MED-2 (sandbox attestation) is install step 0, not code.


## Bootstrap GO given, deployment still gated (2026-11-26)

Owner said GO to the one-time broker bootstrap (install steps 0-6). Execution is owner-side by necessity (no host path exists until the broker creates it). Deployment jobs (build/network/connect/run) remain gated: they proceed under the earlier deployment GO only after channel proof is verified AND the owner explicitly releases the sequence. Next: owner pastes step-0 attestation (StateDir/EnvFile + deny evidence + roots) and chosen paths; agent runs sandbox-denial demo; owner installs+starts; agent runs version-job channel proof; report; await release.


## Install BLOCKED on attestation under current profile (2026-11-26, demonstrated)

Agent CAN self-install mechanically (home writable: binary/config/unit placeable; systemctl --user works incl. transient units; queue/jobs drivable). But trust geography is UNSATISFIABLE: home paths (~/.config, ~/.local/bin, dot-dirs) are agent-writable (proven by canary round-trips), so NO StateDir/EnvFile location is agent-proof; require_escalated is dead (prompts disabled); no documented profile deny-knob found. Per fail-closed install step 0: DO NOT INSTALL until sessions run under a profile denying those paths (owner harness config + restart), then re-attest. After that, the agent executes the entire install + deployment with zero owner commands. Also: project memory notes land in-repo at .agents/memory/miniapp-studio/ (untracked) — owner's to keep or ignore.
