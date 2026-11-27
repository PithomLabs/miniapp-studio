**Task: Prepare Card 002 execution plan — planning only**

Read the current `main` versions of:
- `plans/cards/002-b2-live-proofs.md`
- `docs/gateway-phase-b-live-runbook.md`
- `plans/cards/001-suite-green-neon.md` and its closure evidence
- `plans/miniapp-studio-ultimate-roadmap.md`
- Relevant gateway implementation, tests, and supporting documentation.

Produce **`plans/cards/002_prompt_plan.md`**: a practical, evidence-grounded plan for executing Card 002 and completing its B2 live proofs.

**Requirements**

1. Audit the runbook against the current code, CLI interfaces, tests, infrastructure assumptions, and approved decisions. Do not blindly assume its commands or expectations remain valid.
2. Define a sequenced execution plan covering preflight, pin/tree verification, build, Builder provisioning, privilege and identity proofs, OAuth, provider authentication, Studio callback readiness, supervised no-op turn, sleep/wake/resume, cleanup, and evidence reporting.
3. For every stage specify: prerequisites, responsible party, exact action or command, expected result, evidence to capture, stop conditions, and recovery path.
4. Explicitly resolve the runbook §3 full-suite requirement against Card 001's accepted decision to defer full-suite-green until pre-demo. Propose the precise reconciliation; do not silently waive a gate or claim a focused test gate is equivalent to a full-suite pass.
5. Treat Fly-hosted Studio deployment and callback reachability as an outstanding prerequisite. The repository has no in-tree Studio-on-Fly deployment procedure; do not invent one.
6. Preserve all security and authorization boundaries: owner-held credentials, restricted OAuth grants, separated provider authentication, no secret exposure, N8 re-provisioning precautions, and the N2 prohibition on live gateway upgrades.
7. Identify every owner decision or explicit GO still needed. Card assignment or approval of this plan does not authorize live infrastructure operations.
8. Include the seven-field decision brief, dependencies, evidence/report structure, unresolved blockers, and a final GO/NO-GO recommendation.

**Strict boundaries**

- Planning only. Do not implement code or modify the existing Card 002 brief or runbook.
- Do not execute live Fly, Sprite, Neon, OAuth, provider-authentication, provisioning, deployment, package-installation, or cleanup operations.
- Do not request, print, copy, or store credentials.
- Do not commit or push anything.
- Create only `plans/cards/card002_plan.md`.

Keep the plan actionable and concise. Distinguish verified facts, assumptions, proposed changes, and owner decisions. Cite repository paths and relevant sections wherever they support a conclusion.

When finished, report the file path and summarize the critical blockers and decisions requiring owner attention.