# Integration-Test Runner (Option 2)

**Status:** Implementation prepared; infrastructure operations NOT yet
authorized. Network creation, PostgreSQL attach, and runner launch each
need an explicit owner GO. Execution mechanism: the host execution
broker (`docs/host-exec-broker.md`) — the owner-side commands below
run as broker jobs, never as manual sequences.

## Architecture

```
agent sandbox (unchanged: no socket, no new network)
  │  results via workspace files only
  ▼
repo/test-results/  ←──  runner container (ephemeral, --rm)
  │                       --network studio-test-internal (--internal)
  │                       no published ports, no socket, no host net
  │                       mounts: repo ro · toolchain ro · modcache ro
  │                               GOCACHE tmpfs · results rw
  │                       env: TEST_DATABASE_URL via 0600 --env-file
  │                       GOPROXY=off · BOUNDARY_* unset (refused if set)
  │                       relay 127.0.0.1:5434 → pg-test:5432
  ▼
miniapp-p0-pg (unchanged bridge + 127.0.0.1:5434 mapping, plus one
second network attachment)
```

The runner makes the existing test suite's loopback assumption true
inside its own namespace: `testdb.CheckTestEndpoint` and `owner.dsn`
work verbatim, zero application-code changes.

The container mounts a per-job owner-side snapshot of the repo
read-only — never the live tree — and executes only the image digest
recorded by a verified, pin-checked build. Toolchain paths arrive via
broker-passed environment, verified as directories at startup.
Acceptance reads the broker's frozen provenance (`disarm.json`,
result anchors via `hostexec verify`), never bare repo files.

## Components (all in `testrunner/`)

- `relay`/`gate`/`netcheck`: one stdlib-only Go program, compiled at
  container start from the mounted repo (source and binary can never
  drift). Relay refuses non-loopback binds in code.
- `entrypoint.sh`: modes `netcheck` (isolation battery) and `run`
  (CI gates in order; container exit = suite exit; results to
  `test-results/`).
- `Containerfile`: minimal base + entrypoint only. No Go, source,
  cache, or secrets baked in. Build context is `testrunner/`.

## Exact operations (owner-side; each needs its GO)

Pre-checks (read-only):
`docker network ls`; `docker inspect miniapp-p0-pg` (networks, port
mapping — confirm internal port 5432, else adjust relay target);
`pg_isready -h 127.0.0.1 -p 5434` baseline.

1. Build: `docker build -t studio-test-runner:1 -f ./testrunner/Containerfile ./testrunner`
   (record the image digest; `-f` is required — Docker defaults to
   `PATH/Dockerfile`, which does not exist).
2. Create: `docker network create --internal
   --label studio.testnet=1 studio-test-internal`.
3. Attach: `docker network connect --alias pg-test
   studio-test-internal miniapp-p0-pg`; then `pg_isready -h
   127.0.0.1 -p 5434` proves the existing mapping undisturbed, and
   `docker network inspect` proves peer lists (bridge unchanged;
   internal holds only PG).
4. Netcheck: `docker run --rm --network studio-test-internal --name
   studio-test-netcheck` + mounts + `--env-file` + image `netcheck`
   → exit 0 required before any run.
5. Run: same shape, `--name studio-test-run-<ts>`, results mount,
   image `run` (wrap in host `timeout`).
6. Read verdict: `test-results/exitcode` must equal the launch
   command's `$?`; `summary.txt` + `gotest.json` carry the detail.

Recovery: `docker rm -f` orphans (`docker ps --filter
name=studio-test-` must return empty after each run);
`docker network disconnect studio-test-internal miniapp-p0-pg`;
`docker network rm studio-test-internal`; `pg_isready` proves PG
healthy after every step. Relay failure aborts before tests via the
`gate` (no false green); the JSON stream is incremental, so partial
results survive a killed run.

## Security assessment

- Agent posture unchanged: still no socket, no new network.
- Runner network: `--internal` blocks external routing, but the
  acceptance battery additionally probes the gateways
  (`--internal` does not automatically isolate the gateway IP):
  own gateway, `172.17.0.1`, external IPs/DNS, daemon socket
  absence, and relay loopback-only (container-IP dial must
  refuse). Any REACHABLE line fails adoption — investigate, never
  waive.
- Mounts verified at launch: repo/toolchain/cache read-only,
  results read-write, no socket, no published ports, no
  `--privileged`/`--network=host` (echoing the banned list the
  suite itself enforces).
- Secrets: DSN travels in a 0600 `--env-file` (path on the command
  line, value never); nothing credential-bearing is baked into the
  image or echoed to results. Relay logs connections, never bytes.
- Rotation guard: entrypoint refuses to start when `BOUNDARY_*` is
  set (the e2e provisioner rotates credentials).
- Residuals: read-only host mounts expose build inputs (verify no
  secrets under toolchain/cache paths at implementation); image
  build needs network (owner-side, fine); maintainer DSN remains
  powerful on the scratch cluster by design (see below).

## Database permission boundary (verified)

No credential redesign: existing restrictions are sufficient.
Executed proof (no PG needed, all green): `TestEndpointAllowlist`,
`TestEndpointRejectionPrecedesDial` (rejection before any dial),
`TestAppDSNRefusesNonRun`, `TestAppDSNShape`,
`TestRunPatternGuard`. Code-verified, DB-execution pending the
runner: bootstrap database component ignored (invariant 7),
`refusing grants on non-run database`, run-DB name-pattern
enforcement, `NOBYPASSRLS` test role. Production is unreachable by
construction (endpoint allowlist), never by convention.

## Verification plan (acceptance order)

1. Netcheck exit 0 with every probe `blocked` and both positive
   controls `ok`.
2. `SELECT version()` = 19.x through the relay (first run only).
3. Run mode: gofmt/vet/build green, full serial suite with exact
   counts; required skips present with reasons (toolchain-image,
   boundary-env); zero PG-connectivity skips.
4. Isolation re-verified post-run (peer lists, mapping intact).
5. Repeat run: identical verdict; orphan check empty.

## Recovery of this doc

If the design changes at implementation, update this file in the
same change. Non-blocking follow-ups: image-digest pinning record,
GOCACHE sizing note after the first timed run.
