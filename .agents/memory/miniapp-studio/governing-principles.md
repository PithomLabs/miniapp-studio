---
type: project
description: PRINCIPLES.md locked as governing
---

# Governing principles: PRINCIPLES.md (owner-ordered 2026-11-26)

Authority: `/home/chaschel/Documents/go/miniapp-studio/PRINCIPLES.md` — First-Principles, Local-First, Reality-First Engineering. Governs all failure investigation, technical decisions, and system changes. Objective: understand the system, solve the underlying problem at the narrowest correct boundary, and produce evidence — not make errors disappear.

## Binding rules I must follow every time

1. **Facts ≠ inferences ≠ unknowns.** Every material diagnosis records Observed (exact command/output), Inferred (and why), Unknown (what this context cannot establish), Next discriminating test. Never present inference as fact. Never use "environmental", "sandbox issue", "flaky" as a final diagnosis without the identified boundary + mechanism.
2. **Reconstruct execution topology first:** agent process → sandbox/container → namespace/policy boundary → socket/route/bridge → service endpoint → service process. Never assume shared host, loopback, filesystem, daemon, or credentials.
3. **Test competing explanations** outside-in with smallest safe discriminating checks (service down? not listening? mapping/context wrong? namespace/route? socket perms/policy? auth? wrong harness URL?). Owner-terminal results prove nothing about the agent shell and vice versa; record which context each check ran in.
4. **Preserve local-first:** host-managed PG stays the local DB even if the sandbox can't reach it; the task is a supported test-execution path, not DB duplication or pretending skips passed.
5. **Smallest durable correction at the correct layer;** never reinstall-what-works, open DB/daemon broadly, mount the Docker socket without security review + authorization, hard-code topology-specific addresses, or swap integration tests for mocks.
6. **Evidence, not decoration:** report executed-passed / executed-failed / skipped-with-counts-and-reasons / not-run-and-why / fake-vs-real. Never "full suite passed" with DB skips or unexecuted SQL. Unexecuted migration = unverified, always.
7. **Report format:** Finding / Evidence (with context) / Root cause (only when demonstrated, else leading hypotheses) / Correction / Verification (exact commands, counts, fake-vs-real) / Remaining gate (precise owner action or environment).
8. **Loop:** Observe → Map boundary → Hypothesize → Discriminating tests → Root cause → Smallest correction → Verify invariant → Record evidence → Iterate.
