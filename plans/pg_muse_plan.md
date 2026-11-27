# Verdict: no suitable built-in mechanism — broker proposal follows

I exhausted the supported surfaces, grounding each in official docs ([permissions](https://dev.meta.ai/docs/muse-code/permissions)) plus live probes:

| Mechanism | Verdict |
|---|---|
| Proxy-only per-destination approval | Cannot work: loopback dials die inside the netns (instant `ECONNREFUSED`) before the approval layer ever sees them |
| `unrestricted`/`enabled` network | Too broad (full egress); host-loopback sharing undemonstrated |
| `--disable-sandbox` / `--yolo` | Full egress + anywhere-writes; docs forbid on credential-bearing workstations |
| Approval prefix rules / profiles | Don't pierce the sandbox — independent layers ("each layer works on its own"); posture only |
| Nested `muse exec` | Demonstrated dead (session lock); forbidden to retry |
| MCP / MXP / peer session | Inherits jail / no agent-usable client path / docs-forbidden as consent path |

Per-command unsandboxed runs would also mean manual approval per gate — exactly what you ruled out. So per your fallback instruction, here is the proposal.

# Proposed: minimal host execution broker

**Shape.** Owner-side daemon (you install once; then hands-off permanently) watching a gitignored workspace queue `.agent-jobs/{pending,running,done}/` — the workspace is the proven shared surface. I write a job file, the broker validates, executes, and writes a result file; I poll with a timeout. No network surface on the broker; no socket for me, ever.

**Narrowness (the core property).** Jobs are strict-schema JSON naming one allowlisted op — `net.create|net.connect|image.build|container.run|net.disconnect|net.remove|container.rm|inspect.readonly` — with fixed constants (network name, container, alias, image, mounts, env-file path). The broker constructs argv arrays itself: no shell, no interpolation, unknown op/field rejected, banned substrings (`--privileged`, host-net, socket paths, literal secrets) rejected. Single-flight via lockfile. Secrets travel by path only (the 0600 env-file); results scrubbed for DSN patterns.

**Honesty machinery.** Startup sweep moves crashed `running/` jobs to `done/UNKNOWN` (never success); my standing rule reconciles via read-only inspect ops before any retry. Every job+result persists as the audit trail.

**Setup (one-time, owner).** Install script + systemd unit + start + verify with a read-only `version` job (proves the channel with zero infra change). Broker self-tests first: malformed/banned/oversized/unknown-field jobs all rejected; a failing op records nonzero honestly.

**Then hands-off.** The five deployment gates run through the broker with my per-gate verification from result files intact — netcheck must pass before any suite run, and I still stop on any surprise. Recovery: `rm -f` orphans, disconnect, remove network, `pg_isready` proves PG healthy — executable through the broker itself.

**To proceed** I need design approval; then I'll implement the broker code, queue convention, and docs as repo files (no infra), ready for your one-time setup.
