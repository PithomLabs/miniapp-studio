# Miniapp Studio — Muse Coding Agent Operating Guide

**Status:** Governing working reference for Muse coding agents operating in the Miniapp Studio repository.  
**Last consolidated:** 2026-11-27  
**Purpose:** Make the repository's development environment, local PostgreSQL workflow, sandbox boundaries, test requirements, and owner-authorization rules explicit so every agent can work from the same verified reality.

> **Operating principle:** Reality first. Plan the smallest meaningful increment, implement it, test it, observe what actually happens, learn from the evidence, and adapt. Theory and planning cannot replace contact with a working system. Do not create another planning cycle when a bounded implementation can safely answer the question.

This guide is an operating reference, not permission to perform infrastructure operations. Explicit owner authorization remains required for consequential actions.

> **Stage 1 amendment (2026-11-27, prompt28.3_plan):** Neon-first PostgreSQL is now mandatory (ADR-004). The Docker /
> loopback-PostgreSQL operating procedures in §2.3, §3, §5, §6, and the §10 status snapshot are SUPERSEDED-HISTORY:
> preserved for audit, no longer operable procedure. The endpoint invariant (§3.2) is being re-expressed for Neon
> per ADR-006 (pinned project + branch + TLS + least-privilege role), not loopback. §1, §2.1–§2.2, §4, §8, §9 remain
> in force; §7/§11 remain valid as diagnostic discipline with runner-specific examples treated as historical. Full
> procedure rewrite is owned by the T1 execution plan. Do not follow superseded sections for live work.

---

## 1. Governing principles

### 1.1 First principles and root-cause engineering

When something fails:

1. State the observed failure precisely, including command, execution context, output, and exit code.
2. Separate observed facts from hypotheses, assumptions, and unknowns.
3. Identify the boundary at which the failure occurs: process, filesystem, container, network namespace, host, provider, application, database, or credentials.
4. Construct the smallest experiment that distinguishes competing explanations.
5. Investigate the underlying mechanism before changing configuration or code.
6. Fix the root cause within the authorized scope; do not hide the symptom with retries, skips, fallbacks, broader permissions, or weakened tests.
7. Add regression evidence for the invariant that was previously violated.
8. Report what was and was not verified.

A successful command in one execution context does not prove that another context can perform the same operation. A failure from an isolated shell does not prove that the host service is broken.

### 1.2 Reality-first iteration

Use this loop:

**Observe → Understand → Plan minimally → Implement → Test → Expose to reality → Learn → Adapt.**

- Plan only far enough to make the next meaningful step safe and verifiable.
- Prefer a working, bounded increment over speculative completeness.
- Do not build abstractions for hypothetical future needs without evidence.
- Do not defend an earlier design when actual behavior contradicts it.
- Do not reopen accepted architecture for cosmetic refinements.
- Preserve essential correctness, data safety, authorization, and security even when moving quickly.

### 1.3 Local-first engineering

- Implement and verify capabilities locally before depending on live cloud services.
- Use deterministic fakes and contract tests for local development, but never present fake parity as proof of provider behavior.
- Use the existing local PostgreSQL service for database integration tests through the approved execution path.
- Keep provider-specific effects behind adapters and explicit authorization boundaries.
- Live Fly, GitHub, Neon, deployment, destructive, or other infrastructure operations require the applicable explicit owner GO.

### 1.4 Least privilege and honest evidence

- Do not expand privileges merely because a tool is inconvenient.
- Do not give an agent unrestricted Docker daemon access.
- Do not weaken sandbox isolation or use host networking as an unexamined shortcut.
- Never print, paste, log, commit, or bake credentials into images or artifacts.
- A skipped test is not a passed test. A zero exit code does not make a run complete when required tests skipped.
- Distinguish unit-tested, fake-tested, code-reviewed, integration-tested, and live-provider-verified evidence.

---

## 2. Authority, approvals, and execution boundaries

The Project Owner is the authority for consequential operations. Agents may investigate, recommend, prepare, implement approved local work, test, and report. They cannot authorize themselves.

A direct owner instruction authorizes only the exact operation it names. Planning approval is not implementation approval; implementation approval is not cloud-operation approval.

### 2.1 Owner approval is required for

- Creating, attaching, detaching, or deleting Docker networks or containers when this changes shared infrastructure.
- Launching the approved integration-test runner against the shared PostgreSQL service, unless that exact invocation has already been authorized.
- Changing sandbox networking, filesystem policy, permissions, or execution mode.
- Provisioning, destroying, or changing Fly Sprites or Machines.
- Granting or revoking Connectors, GitHub Apps, OAuth permissions, or other external access.
- GitHub writes, including Issues, Discussions, branches, pull requests, merges, and workflow changes.
- Database schema changes outside an explicitly authorized implementation slice, database replacement, destructive operations, or changes to test endpoint/credential policy.
- Publicly exposing a service or URL.

### 2.2 Fail-closed behavior

If an operation is outside the current approval, stop and request the exact approval. Do not interpret a proposed plan, a Project status field, a prior approval for a different action, or the agent's own recommendation as authorization.

### 2.3 Docker authority

The coding agent's ordinary sandbox must not receive the Docker socket. Docker daemon operations are performed by the owner in the host terminal using the reviewed and authorized commands. The test runner itself receives neither the Docker socket nor privileged mode.

---

## 3. Miniapp Studio local environment: verified facts

The following facts were established from owner-terminal observations and repository investigation. Re-check volatile state when necessary; do not assume a container remains healthy forever.

### 3.1 PostgreSQL containers

There are two distinct local PostgreSQL containers. Do not confuse them.

| Container | Host mapping | Role in this workflow |
|---|---|---|
| `miniapp-p0-pg` | `127.0.0.1:5434 → container:5432` | Relevant local PG19 scratch/test cluster |
| `miniapp-dev-db-1` | `127.0.0.1:5433 → container:5432` | Separate development database; leave untouched unless specifically authorized |

Verified host observations on 2026-11-27:

- `miniapp-p0-pg` was running.
- `pg_isready -h 127.0.0.1 -p 5434` reported accepting connections.
- `SHOW server_version;` returned PostgreSQL `19.0` (`19.0 (Debian 19.0-1.pgdg13+1)`).
- `miniapp-p0-pg` was attached to Docker's default `bridge` network, at container address `172.17.0.2`, gateway `172.17.0.1`.
- At inspection time it was the only container attached to that bridge.
- Its host port mapping was `127.0.0.1:5434→5432`.

These host facts do not imply that the Muse agent's sandbox can reach host loopback or the Docker socket.

### 3.2 PostgreSQL endpoint invariant

The test database harness `testdb.CheckTestEndpoint` fail-closes to:

- Host: `127.0.0.1`, `localhost`, or `::1`.
- Port: `5434`.
- Server: PostgreSQL 19.

The endpoint restriction is intentional. Do not change the allowlist, port, server-version policy, DSN semantics, or test database architecture to accommodate a sandbox limitation without explicit owner re-approval and regression tests.

The existing owner DSN is conventionally available at:

`/tmp/boundary-b/owner.dsn`

Read it only when the approved test workflow requires it. Never print its contents, include it in command output, paste it into chat, or commit it. Do not put the DSN directly on a command line. Use the approved mode-0600 environment file mechanism in the runner workflow.

### 3.3 Existing test database harness

The repository's `testdb` harness already provides the intended test isolation and least-privilege roles, including run-specific databases and a restricted test application role. Reuse it. Do not invent a new credential scheme or substitute production credentials.

The `BOUNDARY_*` environment variables must remain unset for the integration-test runner. The provisioner tests can rotate credentials when those variables are present. The runner entrypoint must fail closed if any `BOUNDARY_*` variable is set.

The endpoint allowlist and rejection-before-dial behavior are intentional safety controls. Never bypass them by changing application code to accept arbitrary database hosts.

---

## 4. Muse sandbox: known execution limitations

The agent shell has been observed to run in a restricted environment that differs from the owner's host environment.

Observed barriers include:

- The shell's `127.0.0.1` is its own network-namespace loopback, not the owner's host loopback.
- The shell may have no usable route to host or external addresses.
- Access to `/var/run/docker.sock` is denied, even when the visible Unix group membership appears to include the socket's group.
- The settings file under `~/.config/muse/` may be unreadable from the nested execution context.
- A nested `muse exec` may fail before the command runs because its session lock cannot be created on a read-only filesystem.

Interpret these observations correctly:

- `ECONNREFUSED` on the agent's `127.0.0.1:5434` does not prove PostgreSQL is stopped on the host.
- Docker socket permission denial does not prove Docker is absent or broken.
- `ENETUNREACH` does not prove the database container is unhealthy.
- A nested-session lock failure is a launch failure, not a network-connectivity result.
- A host-side `docker ps` or `pg_isready` result establishes host state, not sandbox reachability.

Do not repeatedly probe a known policy barrier, request broader privileges, or relabel a caller-side restriction as database failure. Identify which context executed the command and what the result actually proves.

### 4.1 Sandbox network experiment history

One owner-authorized invocation of `muse exec --sandbox-network enabled ...` failed before the network command ran. The nested process could not read the Muse settings file and could not create its session lock directory because the filesystem was read-only.

That result proves neither success nor failure of sandbox network connectivity. Do not retry, escalate, or adopt `--sandbox-network enabled` as the standard setup without separate authorization and a clear security assessment.

---

## 5. Approved PostgreSQL integration-test runner design

**Design status:** Approved in principle.  
**Repository implementation:** Prepared locally according to the latest implementation report.  
**Infrastructure status:** Not yet authorized or deployed.  
**Important:** Approval of the architecture does not authorize Docker network creation, PostgreSQL attachment, or runner launch.

The runner exists to make the existing loopback assumption true inside an isolated, disposable runner namespace without changing application code, the database endpoint allowlist, or the agent sandbox's security posture.

### 5.1 Architecture

```text
Muse agent sandbox (unchanged)
  - no Docker socket
  - no newly granted network access
  - reads test results through workspace files

Owner-launched ephemeral integration-test runner
  - attached to dedicated Docker --internal network
  - no published ports
  - no host networking
  - no Docker socket
  - repository mounted read-only
  - Go toolchain mounted read-only
  - module cache mounted read-only
  - GOCACHE on tmpfs
  - results directory mounted read-write
  - DSN supplied through a mode-0600 env file
  - BOUNDARY_* unset
  - GOPROXY=off
  - loopback relay: 127.0.0.1:5434 → pg-test:5432

Existing miniapp-p0-pg
  - retains original bridge network and 127.0.0.1:5434 host mapping
  - receives one additional attachment to the dedicated internal network
```

The runner-local relay listens only on `127.0.0.1:5434`. It forwards TCP to the PostgreSQL service alias `pg-test:5432`. The Go test process continues to use the existing DSN and endpoint allowlist unchanged.

### 5.2 Repository components

The prepared implementation report describes these repo-owned components:

- `testrunner/`: standard-library-only Go program with `relay`, `gate`, and `netcheck` subcommands.
- Relay: refuses non-loopback binds in code; forwards bytes without inspecting or rewriting PostgreSQL/TLS protocol data.
- Gate: readiness check that prevents the test suite from starting when the relay/database is unavailable.
- Netcheck: positive and negative network-isolation probes, including gateway checks, external IP/DNS, daemon socket absence, and loopback-only relay behavior.
- `entrypoint.sh`: `netcheck` and `run` modes; `run` executes canonical gates in CI order and writes machine-readable and human-readable results.
- `Containerfile`: minimal base and entrypoint. It does not bake in Go source, a toolchain, module cache, or secrets. The runner compiles the mounted source at execution time to avoid source/binary drift.
- `test-results/`: gitignored result directory.
- `docs/test-runner.md`: permanent implementation, security, owner-operation, validation, and recovery record.

The implementation was reported to have relay round-trip, concurrency, refusal, and netcheck unit tests passing with and without race detection. A relay shutdown hang was found and fixed. Treat these as reported repo-level results; do not represent them as live Docker or PostgreSQL runner verification.

### 5.3 Network isolation requirements

The dedicated network must be Docker `--internal`. It should contain only the existing PostgreSQL container and the current ephemeral runner during a run. The PostgreSQL container remains attached to its existing bridge network; do not replace or disturb its existing host port mapping.

An internal network is not automatically proof that every host/gateway path is inaccessible. The `netcheck` battery must explicitly test the runner's own gateway and the known host bridge gateway `172.17.0.1`, as well as external IP and DNS reachability.

**Any unexpected successful reachability probe is a failure.** Do not waive a reachable result. Stop, preserve evidence, investigate the route, and obtain owner review before changing the design.

Required negative checks include:

- Runner cannot reach its own gateway where the approved design expects it blocked.
- Runner cannot reach `172.17.0.1` (the host gateway observed for the existing bridge).
- Runner cannot reach external IPs or resolve/reach external DNS.
- Docker daemon socket is absent and inaccessible.
- Relay refuses a connection to the runner's container IP; only runner loopback is listening.
- No published ports, `--network=host`, `--privileged`, or Docker socket mounts exist.

Required positive checks include:

- Runner can reach `pg-test:5432` through the relay.
- `pg_isready -h 127.0.0.1 -p 5434` accepts connections from inside the runner.
- SQL reports PostgreSQL 19.x.

### 5.4 Mount and secret boundaries

At launch, verify actual mount modes rather than relying on command intent:

| Mount | Required mode |
|---|---|
| Repository source | Read-only |
| Go toolchain | Read-only |
| Module cache | Read-only |
| `GOCACHE` | tmpfs / disposable writable cache |
| `test-results/` | Read-write, limited to results |
| Docker socket | Not mounted |

Inspect the toolchain and module-cache paths for credentials or unrelated sensitive files before mounting. If secrets are present, stop and redesign the mount scope; do not silently expose them read-only.

The DSN must be supplied only through the approved mode-0600 env file. The DSN value must never appear in shell history, command arguments, container image layers, stdout/stderr, result files, or logs. Relay logs may report connection lifecycle/metadata but never stream contents.

Do not bake credentials into the Containerfile or runner image. Do not print environment variables for debugging.

### 5.5 Toolchain and test environment

Verified agent-side toolchain facts used by the runner design:

- `go1.27.3`
- `linux/amd64`
- `GOROOT=/usr/local/go`
- Module cache approximately 754 MB
- `GOFLAGS=-buildvcs=false`
- No vendor directory
- Default GOPROXY in the normal environment

The runner must use the exact matching Go toolchain and existing module cache, mounted read-only. Set `GOPROXY=off` so a missing dependency fails visibly instead of silently downloading from the network. The module cache must be checked for completeness before the run.

Use `GOCACHE` on tmpfs. Do not allow the runner to download or update dependencies. Do not change `go.mod`, `go.sum`, or dependency versions merely to make the runner work.

### 5.6 Test suite behavior

The canonical CI gates run in this order:

1. `gofmt -l .` — output must be empty.
2. `go vet ./...`
3. `go build ./...`
4. `go test -p 1 ./...`

The runner must preserve the actual exit code of the suite and write:

- `gotest.json` — machine-readable test events/results.
- `summary.txt` — human-readable exact pass/fail/skip counts and explanations.
- `exitcode` — exact suite exit code.

The launch command's `$?` must match the result file's `exitcode`. Preserve partial results if the run is interrupted.

Expected skips must be explained, not hidden. The existing `sandbox/daemon_test.go` skip when the optional toolchain image is absent is a designed skip. The `boundary-env` skip may also be expected under the runner's explicit environment contract. Confirm their actual reasons from the test output. There must be **zero PostgreSQL-connectivity skips**. A zero exit code with required DB tests skipped is incomplete verification.

The runner must not set any `BOUNDARY_*` variables. Its entrypoint must refuse to run if they are set, because the provisioner test path can rotate credentials.

### 5.7 Database permission boundary

The current endpoint allowlist, run-database naming checks, test role design, and provisioning guards are intentional. Existing non-DB-dependent tests reported green include:

- `TestEndpointAllowlist`
- `TestEndpointRejectionPrecedesDial`
- `TestAppDSNRefusesNonRun`
- `TestAppDSNShape`
- `TestRunPatternGuard`

These code-level checks do not replace DB-backed execution. The runner must execute migration `000025`, lifecycle receipt-store tests, RLS/grant tests, and privilege-parity tests against the actual local PG19 service.

Do not introduce production data or production credentials. The endpoint allowlist is part of the production-isolation boundary; do not broaden it.

---

## 6. Owner-controlled runner lifecycle

Every shared-infrastructure operation remains owner-controlled and individually gated.

The reviewed operation sequence in `docs/test-runner.md` is:

1. Read-only pre-checks: inspect Docker networks and `miniapp-p0-pg`; confirm its internal port is `5432`; verify the host-side PG baseline.
2. Build the minimal runner image from `testrunner/`; record its digest.
3. Create the dedicated `--internal` network `studio-test-internal`.
4. Attach `miniapp-p0-pg` to that network with alias `pg-test`, preserving its existing bridge attachment and host mapping.
5. Verify peer membership and confirm the host-side `pg_isready` still succeeds.
6. Launch the runner's `netcheck` mode. It must exit 0 with every required negative probe blocked and positive control successful.
7. Only after netcheck is accepted, launch `run` mode.
8. Compare runner exit code with `test-results/exitcode`; inspect summary and JSON results.
9. Repeat the suite run to establish repeatability; verify no orphan runner remains.
10. Disconnect PostgreSQL from the dedicated network and remove the network when the runner facility is not intended to remain provisioned. Verify PG remains healthy.

**Approval granularity:** network creation, PostgreSQL attachment, netcheck launch, and suite-run launch are separate owner approval boundaries unless the owner explicitly authorizes a precisely enumerated sequence together. Approval for one does not imply approval for the next.

The agent may prepare exact commands, inspect results, and update repo documentation within its authorized scope. It must not run Docker lifecycle commands itself from the restricted sandbox.

---

## 7. Failure diagnosis and recovery

### 7.1 Diagnose the correct layer

| Symptom | What it establishes | Next investigation |
|---|---|---|
| Host `pg_isready` succeeds; agent `127.0.0.1:5434` refuses | Host DB is available; agent loopback is separate or isolated | Use approved runner path; do not restart PG |
| Agent `docker ps` gets socket permission denied | Agent lacks Docker API access | Preserve the boundary; owner operates Docker |
| `ENETUNREACH` from agent | No route from that namespace to the target | Inspect allowed network path; do not change DSN automatically |
| Nested `muse exec` fails creating session lock | Nested launch/filesystem failure | Treat as no network evidence |
| Runner gate cannot reach PG | Relay, alias, internal network, or PG attachment issue | Stop before running tests; inspect exact endpoint and peers |
| External/gateway netcheck is reachable | Isolation invariant is violated | Stop adoption; preserve evidence; investigate and request owner decision |
| `GOPROXY=off` reports missing module | Mounted module cache is incomplete | Reconcile dependency cache in an authorized, networked preparation context; do not let runner download |
| Database tests skip | DB verification did not occur | Report incomplete; diagnose runner/relay/DSN path |
| Runner hangs | Run may be incomplete | Owner removes the exact orphan container; retain incremental results |

### 7.2 Recovery rules

- Relay failure must fail the readiness gate before tests begin.
- Do not report a green test run if the relay or database was unavailable.
- For a hung runner, the owner may remove the exact named runner container using the approved recovery procedure. Preserve partial result files.
- Check for orphan containers after each run.
- Disconnect only the runner network attachment when rolling back; do not remove the original bridge attachment or alter the host port mapping.
- Remove the dedicated network only after confirming it is no longer in use.
- After every cleanup step, verify `miniapp-p0-pg` remains healthy with host-side `pg_isready`.
- If any cleanup operation has an unexpected effect, stop and request owner direction.

---

## 8. Standard reporting format

Every implementation or test report must include:

1. **Scope:** exact task and files changed.
2. **Execution context:** Muse sandbox, host terminal, runner container, or another named context.
3. **Authorization:** exact owner instruction permitting the operation.
4. **Observed facts:** commands, relevant output, and exit codes.
5. **Tests:** exact commands, pass/fail/skip counts, and skip reasons.
6. **Evidence tier:** unit/fake, code-reviewed, local integration, runner integration, or live provider.
7. **Security checks:** relevant positive and negative probes.
8. **Unknowns:** what was not tested or could not be observed.
9. **Recovery:** resources created, retained, removed, and remaining.
10. **Next approval boundary:** the exact next operation requiring owner GO.

Never state “all tests passed” without exact counts and skip disclosure. Never label a restriction as environmental breakage without evidence. Never claim provider behavior based only on a fake, scripted HTTP double, or vendor documentation.

---

## 9. Repository workflow and change discipline

- Read the applicable repository charter and instructions before changing code: `SKILLS.md`, `PRINCIPLES.md`, `AGENTS.md`, and relevant architecture/runbook documentation.
- Treat this file and `docs/test-runner.md` as setup guidance, not permission to override the repository's governing instructions.
- If instructions conflict, apply the newest explicit owner decision where it clearly resolves the conflict. If an unresolved conflict changes scope, security, runtime architecture, or data safety, stop and ask the owner. Do not silently disregard `AGENTS.md` or any canonical instruction.
- Inspect the working tree before editing. Preserve unrelated modifications and user work.
- Make the smallest change that satisfies the authorized task. Do not reformat or rewrite unrelated files.
- Add regression tests for the underlying invariant, not merely the observed symptom.
- Run the narrowest relevant tests first, then the required broader gates.
- Do not weaken tests, increase skip conditions, widen allowlists, or modify security checks to manufacture a green result.
- Do not commit, push, create GitHub records, or perform deployment operations unless separately authorized.
- Keep generated results under the designated gitignored `test-results/` directory; do not commit DSNs, logs containing secrets, or transient runner artifacts.

---

## 10. Current status snapshot

This section records the latest reported state at the time this guide was consolidated. Re-check volatile state before acting.

| Item | Status |
|---|---|
| Host PostgreSQL 19 | Verified running at `127.0.0.1:5434` on 2026-11-27 |
| `miniapp-p0-pg` | Existing container; default bridge retained |
| `miniapp-dev-db-1` | Separate container on port 5433; untouched |
| Agent direct host-DB access | Blocked by sandbox network namespace |
| Agent Docker socket access | Denied; preserve this boundary |
| `muse exec --sandbox-network enabled` test | Failed before network command execution; no connectivity conclusion |
| Option 2 runner architecture | Approved in principle |
| `testrunner/` implementation | Reported prepared; local unit tests and full serial suite reported green |
| `docs/test-runner.md` | Prepared as permanent runner operation/security reference |
| Docker internal network | Not yet created |
| PostgreSQL second network attachment | Not yet made |
| Runner netcheck | Not yet run against real Docker network |
| DB-backed runner suite | Not yet run through runner |
| Infrastructure authorization | Pending separate owner GOs for each operation |

Do not upgrade a “prepared” or “unit-tested” state to “deployed” or “integration-verified” without the corresponding evidence.

---

## 11. Quick decision rules

- **Host DB works, agent shell cannot reach it:** solve the execution path; do not reinstall or restart PostgreSQL.
- **Docker socket denied in agent shell:** preserve that restriction; use owner-controlled Docker operations.
- **A runner network probe reaches an unexpected gateway or external endpoint:** stop; do not waive the result.
- **A DB-backed test skips:** verification is incomplete.
- **A proposed fix requires broader security privileges:** investigate a narrower alternative first and request owner approval.
- **A plan has enough information for a safe, bounded local implementation:** implement it rather than commissioning endless planning reviews.
- **A real provider capability differs from the design:** trust measured behavior, preserve evidence, and adapt the smallest necessary layer.

**The standard is not a perfect plan. The standard is a safe, reproducible development loop that survives contact with reality.**
