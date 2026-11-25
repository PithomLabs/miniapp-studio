---
type: reference
description: AGENTS.md standing guidelines lock-in
---

# Standing guidelines lock-in (2026-11-25)

`/home/chaschel/Documents/go/miniapp-studio/AGENTS.md` is the binding
operating standard for all Miniapp Studio implementation work (Phases B–E).
Key enforcements already aligned with session practice:

- Root-cause discipline: red reproduction on baseline, invariant tracing,
  fix at owning component, permanent regression tests, residual-risk notes.
- Phase discipline: authorized scope only; no early later-phase work;
  no commits/deploys/live ops without explicit authorization.
- Evidence honesty: label every claim code-verified / unit-tested /
  integration-tested / live-tested / specified-unverified; never present
  fakes or inspection as live proof.
- PG19 Docker `127.0.0.1:5434` is the sole DB baseline; verify version
  first; stop and report if unavailable or wrong major.
- Security: threat model + actor×resource×operation matrix; enforce
  identity/permissions at runtime; fail closed; no same-user fallback;
  OS-identity proof needs live EACCES checks (fakes = unit evidence only).
- Filesystem: OpenRoot + O_NOFOLLOW/O_EXCL, fstat-on-fd checks, size
  bounds, atomic replacement with ownership-defined EEXIST handling.
- Concurrency: durable claims before dispatch; stale claims are
  indeterminate unless no-child is provable (never auto-drop); new fenced
  turn for recovery; at-most-once dispatch framing.
- Freshness: re-sign per attempt with stable logical id; never weaken
  receiver skew/signature checks; prove stale-forged still rejected.
- Provisioning: create-if-absent, unhealthy fails closed, explicit
  guarded upgrades, cross-process coordination proof.
- Sign-off checklist (§11) must be walked before reporting completion.
