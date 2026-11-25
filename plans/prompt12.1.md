This prompt authorizes a **bounded residual-closure pass** based on `plans/adv_review_prompt12.md`. It addresses N1 and N4, restores PostgreSQL 19 verification, verifies the upstream OpenCode hash, and tracks N2–N10 without turning the work into another broad review.&#x20;

adv_review_prompt12.md



# Implementation Directive — Phase B Part 1 Residual Closure (Prompt 12)

## 1. Role and objective

Act as the Senior Go Systems Architect and implementation agent for Miniapp Studio.

Execute a strictly bounded residual-closure pass following the adversarial review:

`plans/adv_review_prompt12.md`

The review verdict is **ACCEPTED WITH BOUNDED RESIDUALS**. Phase B Part 1 is accepted at that level. Do not reopen its architecture or repeat the entire adversarial review.

Your objective is to close the identified prerequisites for a future Part 2 authorization decision:

1. Correct N1: make the Gateway service privilege requirement explicit, make the filesystem ownership specification operationally consistent, and add a fail-closed boot-time privilege-drop self-test.
2. Correct N4: resolve the M5 identifier collision and explicitly mark capability matrix v1 as superseded.
3. Restore and independently execute the PostgreSQL 19 Docker verification blocked during prompt12 review.
4. Resolve R-NET: independently verify the vendored OpenCode `run.ts` provenance and hashes.
5. Explicitly track N2–N10 without expanding this pass into their implementation.

Produce a durable residual-closure report and update the governing evidence and live runbook.

Stop when these tasks and their evidence are complete. Do not proceed to Part 2.

## 2. Governing engineering principles

The locked **Miniapp Studio Implementation Agent Guidelines** apply.

In particular:

- Investigate root causes and fix them at the component that owns the invariant.
- Do not patch only the visible symptom.
- Preserve security boundaries and fail-closed behavior.
- Test invariants, including negative and adversarial cases.
- Distinguish code-verified, test-verified, database-verified, and live-verified evidence.
- Never claim that a test, live operation, or verification succeeded when it did not.
- Keep scope bounded. Do not reopen accepted architecture without concrete evidence of a violated invariant.

The existing accepted Agent Gateway design, Phase A acceptance, Phase B Part 1 corrections, Miniapp application contract, and frozen plans remain authoritative.

## 3. Authorization and forbidden actions

This directive authorizes only local, bounded implementation, documentation, and verification work described below.

**Part 2 remains unauthorized.**

Do not:

- Create, destroy, provision, or modify Fly Sprites.
- Perform Fly API mutations.
- Start OAuth ceremonies.
- Install or run OpenCode on live infrastructure.
- Execute any section of the owner-run Phase B live runbook.
- Use production credentials or request credentials in chat.
- Modify Miniapp or its repository.
- Change F1, G-01, P5, or frozen plans.
- Add adapters, frameworks, workflow engines, or unrelated tools.
- Implement N2–N10 as part of this pass.
- Commit, push, stage, or alter Git history.

No live Gateway upgrade is authorized. N2 must be closed before an actual live upgrade can be performed.

Preserve the existing all-untracked, zero-commit working-tree boundary.

## 4. N1 — Resolve Gateway service privilege and identity specification

### 4.1 Root cause to address

The Gateway uses `SysProcAttr.Credential` to drop agent processes to the configured `studio-agent` identity.

That mechanism requires the Gateway service process to possess sufficient operating-system privileges to change UID, GID, and supplementary groups.

The implementation currently does not make this service privilege requirement explicit or independently verify it at startup.

The correction report and runbook also disagree about file ownership: the runbook expects `studio-gw`-owned files, while the actual file-creation path applies group and mode changes without changing the file owner. Under a privileged Gateway service, the owner is the Gateway service's effective UID, not necessarily `studio-gw`.

This must be corrected at the architectural and operational boundary. Do not weaken the permission model to make the old runbook assertions pass.

### 4.2 Define the actual service privilege contract

Update the provisioning specification, Gateway documentation, and runbook to state explicitly:

- The Gateway service must run with effective privileges sufficient to change to the configured agent UID/GID and supplementary groups.
- The reference deployment configuration uses a privileged service identity.
- A non-root configuration is acceptable only when its effective operating-system capabilities are independently demonstrated to permit the complete credential transition.
- A service running as `studio-gw` without the necessary privilege must not be assumed capable of launching `studio-agent`.
- The production Gateway must never fall back to launching agents under its own identity.
- Any unavailable or insufficient privilege causes an honest fail-closed refusal.

Do not invent a Fly Sprite service `user` field. The Sprite service API surface remains unchanged.

Document the actual distinction between:

1. Gateway service identity.
2. Agent process identity.
3. Filesystem owner.
4. Shared I/O group.

These identities must not be treated as interchangeable.

### 4.3 Add a boot-time credential-drop self-test

Add a small, deterministic startup verification to the production Gateway service startup path.

Before the service reports itself ready:

1. Resolve and validate the configured agent UID, GID, and required supplementary groups.
2. Confirm the service identity is distinct from the agent identity.
3. Attempt to launch a harmless fixed executable such as `true` using the exact credential-drop configuration used for real agent processes.
4. Wait for the child and verify successful completion.
5. If the credential transition or process launch fails, refuse Gateway startup and report a useful, non-secret diagnostic.
6. Do not continue in a degraded same-UID mode.

The self-test must exercise the actual production credential construction path, not a parallel approximation.

Keep the self-test narrow. Do not access or expose signing keys as part of it.

Add test seams where necessary so unit tests can verify:

- Successful startup when the credential-drop self-test succeeds.
- Startup refusal when UID/GID/group resolution fails.
- Startup refusal on credential-drop failure, including `EPERM`.
- No production readiness path bypasses the self-test.
- There is no same-UID fallback.
- The exact agent identity and supplementary groups are passed to the child.

Retain the existing local-process identity and credential tests. Do not claim that these replace the real Sprite EACCES proof.

### 4.4 Correct the filesystem permission matrix

Update the correction report and live runbook to make filesystem ownership identity-aware.

Do not hardcode `studio-gw` as the expected owner of Gateway-created files unless the actual service process runs under that UID.

Instead:

- Determine the Gateway process's actual effective UID.
- Determine the numeric `studio-io` GID.
- Verify ownership of Gateway-created directories and answer files against the Gateway effective UID.
- Verify the expected shared group against `studio-io`.
- Verify exact permission bits and sticky/setgid behavior.
- Verify the agent's intended read/write/delete permissions through actual operations.

Preserve the intended permission semantics:

| Resource                     | Required behavior                                        |
| ---------------------------- | -------------------------------------------------------- |
| Gateway keys directory/files | Gateway-only access; agent denied                        |
| Gateway state                | Gateway-only access                                      |
| `.studio` directory          | Gateway-owned; `studio-io` group; mode 2750              |
| Inbox directory              | Gateway-owned; `studio-io` group; mode 2730              |
| Agent question drop          | Agent-owned; group-readable by `studio-io`; bounded size |
| Gateway answer               | Gateway-owned; `studio-io` group; mode 0640              |
| Agent answer access          | Read permitted; overwrite/delete denied                  |
| Agent workdir                | Restricted to the application builder's intended scope   |

The live runbook must derive the actual Gateway service EUID from the running process rather than assuming it is `studio-gw`.

For example, use the process's `/proc/<pid>/status` effective UID and compare it with the numeric owner reported by `stat`. Keep the commands executable in the Sprite shell.

Preserve the existing agent-side permission-denial checks. The actual different-UID/key-read-denied demonstration remains a Part 2 live gate.

### 4.5 N1 acceptance

N1 is closed only when:

- The service privilege contract is explicit and consistent across documentation.
- Production startup performs the fail-closed credential-drop self-test.
- Tests prove successful and failed startup paths.
- The permission matrix no longer assumes a service owner that the implementation does not establish.
- Runbook §7 and §7b use the actual process identity.
- The correction report clearly distinguishes local proof from future live EACCES proof.

Do not claim the live identity boundary has been verified. That remains a Part 2 obligation.

## 5. N4 — Resolve M5 naming and evidence matrix history

### 5.1 Preserve the original finding lineage

The original `plans/prompt10.md` defines M5 as the SSE stream-lifetime item.

The subsequent adversarial review reused M5 for bounded stderr handling.

Both meanings now appear in project documentation. The correction pass implemented bounded stderr handling, but the original SSE stream-lifetime item remains deferred.

Resolve this without rewriting historical review decisions or pretending the SSE item was implemented.

Use explicit identifiers consistently:

- **M5-STDERR:** bounded Gateway stderr buffering, implemented and tested during prompt11 correction.
- **M5-SSE:** original SSE stream-lifetime and concurrent-stream-budget operational item, still deferred.

Add a clear mapping note explaining why the later stderr finding had a colliding identifier.

Update all current references in:

- `docs/prompt11-correction-report.md`
- `docs/gateway-phase-b-evidence.md`
- `docs/gateway-phase-b-live-runbook.md`
- `plans/prompt11_plan.md`
- Any other current-phase document that uses the ambiguous identifier.

Preserve original historical prompt documents as historical records. Do not rewrite their original finding definitions.

### 5.2 Mark capability matrix v1 as superseded

In `docs/gateway-phase-b-evidence.md`, put an unmistakable banner immediately before capability matrix v1.

The banner must state:

- Matrix v1 is historical and superseded.
- It describes the initial Part 1 implementation before the prompt11 correction pass.
- Matrix v2 and the correction report are the governing current evidence.
- Present-tense claims in v1 must not be interpreted as current verification.

Where necessary, change misleading historical statements to explicitly past-tense statements.

Do not delete v1. Preserve the audit trail.

### 5.3 Clarify correction counts

Where documentation says "M1–M16 all fixed," clarify that this means the prompt11 correction bundle's findings, including M5-STDERR.

Explicitly state that M5-SSE remains deferred and is not part of that completed correction bundle.

Do not imply the SSE lifetime/concurrent-stream-budget work was implemented.

### 5.4 N4 acceptance

N4 is closed only when:

- The two M5 lineages are unambiguous across current documents.
- Historical prompt references remain traceable.
- Matrix v1 is visibly superseded.
- Matrix v2 remains the current capability record.
- The SSE operational backlog remains honestly deferred.
- The correction completion language cannot be read as claiming SSE lifetime work is complete.

## 6. R-DB — Restore PostgreSQL 19 Docker verification

### 6.1 Mandatory database baseline

Use only the user's existing PostgreSQL 19 Docker environment:

`127.0.0.1:5434`

PostgreSQL 19 is the exclusive database-testing baseline for this project.

Do not substitute PostgreSQL 16, 17, or another version.

Do not silently skip database-backed tests.

### 6.2 Restore access safely

The prompt12 review could not connect because:

- `127.0.0.1:5434` refused connections.
- Docker socket access was denied.
- No test credentials were available in that review environment.

Investigate the current environment and use the established authorized access path if available.

Do not weaken Docker permissions, add users to privileged groups, alter Docker configuration, or delete/recreate the user's database container as a workaround.

If the existing PG19 container is unavailable, report the exact blocker and stop database-backed execution. Do not invent credentials or request secrets in chat.

### 6.3 Verify before testing

Before any database-backed command:

1. Confirm that the target endpoint is `127.0.0.1:5434`.
2. Confirm the actual database connection succeeds.
3. Execute `SHOW server_version`.
4. Execute `SELECT version()`.
5. Require PostgreSQL 19.x.
6. Verify the target database/schema is isolated and safe for tests.
7. Confirm test credentials are obtained from the existing protected environment without printing them.

Never expose a DSN, password, token, or credential in logs or the final report.

If the target is not PG19 or isolation cannot be established, stop.

### 6.4 Required database gates

Run the full suite serially:

```
go test -p 1 -count=1 ./...
```

Run the six-package race suite against the same PG19 baseline:

```
go test -race -p 1 -count=1 ./gateway ./store ./tools ./server ./cmd/studio ./cmd/gateway
```

Independently identify and execute the specific DB-backed tests for:

- F7 provisioning.
- F7 upgrade.
- F7 cross-session advisory-lock serialization.
- M14 provision/upgrade digest reporting.
- M15 KDF server wiring and wrong-secret rejection.
- Migration and RLS verification suites.

Discover the actual test names from source. Do not guess test names or count a skipped test as a pass.

These targeted tests supplement the full suite; they do not replace it.

Record:

- PostgreSQL version and endpoint.
- Safe database identity (no credentials).
- Exact commands.
- Exit codes.
- Per-package results.
- Skipped tests and reasons.
- Results of the named F7/M14/M15 tests.

If any database gate fails, investigate whether the failure is environmental or code-related. Do not change code merely to suppress a database failure.

### 6.5 R-DB acceptance

R-DB is closed only when all required database-backed gates have actually executed and passed against PostgreSQL 19 Docker.

Source inspection, reading test assertions, or a previous test report is not a substitute for execution.

If the environment remains blocked, complete the other authorized work but leave R-DB explicitly blocked and the overall residual-closure pass incomplete.

## 7. R-NET — Independently verify OpenCode source provenance

Verify the vendored source record:

`docs/opencode-run-v1.19.6.md`

Expected source:

- Repository: `anomalyco/opencode`
- Tag: `v1.19.6`
- Path: `packages/opencode/src/cli/cmd/run.ts`
- Expected Git blob SHA-1: `944aec17abd75cc3272688e89a140b56cd2bb2d3`
- Expected source SHA-256: `e040e90d4c81cdd59b9568ee90a53c0209cd9ddf1b8b207d1a8685b2e26d57d0`

Independently verify:

1. The tag identity and its resolved commit.
2. The tag tree's entry for the exact source path.
3. The exact downloaded source bytes.
4. The Git blob SHA-1, including the Git blob header semantics.
5. The source SHA-256.
6. That both hashes match the vendored audit record.

Use the official upstream repository and tag. Do not trust the existing document's hashes without rechecking them.

No upstream code should be copied into the implementation. This is a provenance verification only.

If network access is unavailable, report R-NET as blocked. Do not claim verification from the existing document alone.

Record the verification command, upstream reference, tag commit, tree-entry identity, both calculated hashes, and result.

## 8. N2–N10 — Track, do not expand

Do not implement the following findings in this pass.

Add or update a concise residual register in the new closure report and evidence documentation.

| Finding | Residual                                                         | Required handling                                                                                                     |
| ------- | ---------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| N2      | Upgrade idle-check TOCTOU and empty-URL skip                     | Must be fixed and tested before any live Gateway upgrade. Update the runbook to prohibit invoking upgrade until then. |
| N3      | Recovery guidance text misdirects operators                      | Track for the next bounded server-message correction.                                                                 |
| N4      | M5 identifier collision and unmarked v1 matrix                   | Close in this pass as specified above.                                                                                |
| N5      | PID recording occurs after launch capture                        | Track. Before relying on live orphan recovery, record pre-restart agent PIDs and verify their absence after recovery. |
| N6      | Launch identity capture lacks a timeout                          | Track. Do not claim bounded startup recovery for a hung agent binary.                                                 |
| N7      | Deliverer transition errors are silently ignored                 | Track for deliverer observability hardening.                                                                          |
| N8      | Staging key-file creation has a temporary permissive-mode window | Track. Do not re-provision while an untrusted sibling agent may exploit the staging window.                           |
| N9      | Adapter Continue retains an internal check-then-act window       | Track for adapter-level hardening; current server serialization remains the relied-upon boundary.                     |
| N10     | Silent IOGID fallback                                            | Track for explicit misconfiguration refusal.                                                                          |

For every residual, record:

- Root issue.
- Affected component.
- Security/correctness impact.
- Whether it affects Part 2 readiness or a narrower operation.
- Required durable correction and regression test.
- Whether owner risk acceptance is required.

Specific operational constraints:

- **N2:** No live `gateway upgrade` operation until N2 is closed. If binary drift is detected during a future live run, stop and report rather than bypassing the guard.
- **N5/N6:** Explicitly disclose their implications for orphan reaping and launch hangs before depending on automated recovery during real process failures.
- Do not silently promote these items to fixed or dismiss them as irrelevant.

## 9. Documentation deliverables

Create:

`docs/prompt12-residual-closure-report.md`

The report must contain:

1. Executive summary and current acceptance state.
2. Exact scope and tree state.
3. N1 root cause, correction, and code/test evidence.
4. Updated service privilege contract.
5. Corrected filesystem permission matrix.
6. N4 identifier mapping and matrix v1 supersession record.
7. R-DB exact environment and gate results.
8. R-NET upstream provenance verification.
9. N2–N10 residual register and operational restrictions.
10. Exact list of checks not run and why.
11. Explicit statement that Part 2 was not executed or authorized.
12. Final recommendation for a separate Part 2 authorization decision.

Update, as appropriate:

- `docs/prompt11-correction-report.md`
- `docs/gateway-phase-b-evidence.md`
- `docs/gateway-phase-b-live-runbook.md`
- `plans/prompt11_plan.md`

Do not modify `plans/adv_review_prompt12.md`; preserve it as the independent review record.

Ensure all documents agree on service privilege, filesystem ownership, M5 naming, superseded matrix status, PG19 results, and remaining live gates.

## 10. Verification of changed code and documents

For N1 implementation changes, run the relevant focused unit and local-process tests, then the normal Go gates:

```
gofmt -l .
go vet ./...
go build ./...
```

After confirming PostgreSQL 19 availability and isolation, run all database gates in §6.

For documentation changes, inspect every cross-reference and search the affected current documents for:

- `M5`
- `studio-gw`
- `service uid`
- `privileged`
- `superseded`
- `PG19`
- `upgrade`

Ensure there are no contradictory ownership expectations or ambiguous M5 references left in current documentation.

Do not rerun a general adversarial audit of all Phase B corrections. Validate the exact changes and the named residual closures only.

## 11. Acceptance criteria

The residual-closure pass is complete only when:

### N1

- Service privilege requirement is explicit.
- Boot-time credential-drop self-test is implemented and tested.
- Startup fails closed if the required privilege is absent.
- Permission matrix and runbook are based on the actual Gateway service identity.
- Live EACCES proof remains honestly deferred.

### N4

- M5-STDERR and M5-SSE are distinct and traceable.
- Matrix v1 has an unmistakable superseded banner.
- Matrix v2 is clearly authoritative.
- SSE lifetime work remains classified as deferred.

### R-DB

- PostgreSQL 19 Docker has been verified.
- All mandatory database and race gates have executed.
- F7, M14, and M15 targeted tests passed.
- No substitute PostgreSQL version was used.

### R-NET

- Both upstream hashes have been independently verified against the pinned tag and exact file.

### N2–N10

- Every residual is recorded accurately.
- N2 prohibits live upgrade until fixed.
- N5/N6 implications are explicit.
- No unapproved implementation expansion occurred.

### Governance

- Existing accepted architecture remains unchanged.
- No Part 2 activity occurred.
- No code or documentation changes outside this directive occurred.
- No commits or pushes occurred.

## 12. Stop condition and final decision

When the bounded work is complete, stop.

Do not provision a Sprite, install OpenCode, perform OAuth, modify Fly infrastructure, or execute any live runbook step.

Conclude using exactly one status:

**READY FOR SEPARATE PART 2 AUTHORIZATION**

Use this only if N1, N4, R-DB, and R-NET are independently closed and all other residuals are explicitly tracked with safe operational restrictions.

**NOT READY — BOUNDED RESIDUALS REMAIN**

Use this if any required prerequisite is blocked or incomplete. Identify the exact remaining item and evidence needed. In particular, do not claim readiness if PostgreSQL 19 database gates remain unexecuted.

This status is a recommendation only. It does not authorize Part 2.

No commits. No pushes. No live infrastructure operations.

The key safeguard is that **PG19 verification is a hard prerequisite**, not something the agent can replace with source inspection or an older PostgreSQL version. Likewise, N2 remains an explicit prohibition on live upgrades until its TOCTOU issue is addressed.