Broker implementation complete. No installation, no service, no Docker operations — stopped as instructed.

  Built: hostexec/ library + cmd/hostexec daemon covering all four binding requirements — (1) nine typed ops with
  broker-constructed fixed argv, zero shell, banned-token backstop; (2) queue treated as hostile: symlinks never followed,
  traversal impossible by construction, malformed/oversized/duplicate/unknown-field jobs quarantined with reasons, atomic
  rename transitions, crash leftovers swept to honest UNKNOWN; (3) privilege boundary documented in docs/host-exec-broker.md,
  systemd unit hardened (no new privs, strict filesystem, unix-socket-only networking, single-instance lock, docker
  preflight); (4) deploy-only ops auto-disarm after the first green suite run, persisted across restarts, re-arm owner-only —
  while version/run/sweep/inspect stay armed for repeatable runs. Build inputs pinned by hash with drift refusal, plus a pin
  helper for install-time recording.

  Verification: 28 broker tests green with and without -race (including a fake-docker end-to-end of the disarm flow and
  pin-tamper refusal); full serial suite exit 0; two genuine bugs caught by the tests during development (shutdown hang fixed
  in relay; a ghost-id false positive in the banned-token list, fixed by narrowing it).

  Pending: your evidence review, then the one-time setup (build binary, write config+pins, install unit, verify via a
  read-only version job).

◆ Worked for 10m 06s · 1:45 AM · 27/Nov

◆ Recap: Permanent test-runner execution now routes through the host broker, implemented and fully tested with auto-disarming
         deploy permissions. Next action is the owner's one-time broker installation and verification.

do we need adversarial code review now per above or later?

**Now, before installation.**

The broker is a privileged host-side security boundary. An implementation bug could give an agent unintended host or Docker capabilities. We should adversarially review it before installing the daemon.

Keep the review tightly scoped:

1. **Privilege escalation:** Can an agent construct a job that executes arbitrary commands or escapes the allowlist?
2. **Queue security:** Symlink attacks, path traversal, TOCTOU races, duplicate jobs and concurrent execution.
3. **Authorization:** Can deployment permissions be replayed, bypassed or re-enabled by the agent?
4. **Docker boundary:** Can operations access the socket, host networking, arbitrary mounts or unrelated containers?
5. **Secrets:** Can DSNs, environment variables or credentials leak into logs or results?
6. **Crash recovery:** Can uncertain operations be mistaken for success or executed twice?
7. **Systemd hardening:** Does the actual running service enforce the documented restrictions?

**My recommendation:** independent adversarial review, followed by one remediation pass for genuine security or structural defects.

Then install the broker, verify it with a read-only version job, and proceed with the already-approved runner deployment gates.

No more architecture review cycles unless the review discovers a foundational vulnerability.

**Reality-first: review the security boundary, then put it to work.**