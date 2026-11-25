# P0 evidence record (2026-11-24)

## Studio skeleton — GREEN in this environment

- `gofmt -l .` → no output (exit 0)
- `go vet ./...` → pass
- `go build ./...` → pass
- `go test ./...` → pass (`[no test files]`)
- Toolchain: go1.27.3, module `go 1.26.0`

## Framework baseline — partial (PG-blocked steps outstanding)

Baseline: `73351a801fc54e19ee7254e6fe3ea7fbed859378` (= `origin/main`).

Passed here (read-only, miniapp tree untouched):

- `gofmt -l .` → no output (exit 0)
- `go vet ./...` → pass
- `go build ./...` → pass (verified in earlier recon)

Blocked here — no Postgres available (`TEST_DATABASE_URL` unset, no local
server; docker socket present but permission-denied in this sandbox).

## Docker diagnosis (2026-11-24, read-only, nothing changed)

- User `chaschel` (uid 1000), groups `chaschel` + `nogroup`.
- Socket `/var/run/docker.sock`: `srw-rw---- nobody:nogroup` — the user IS in
  the owning group, so host file permissions already allow access.
- Client: Docker Engine Community v29.5.2, context `default` →
  `unix:///var/run/docker.sock`, `DOCKER_HOST` unset (correct endpoint).
- `docker ps` / raw socket connect: permission denied from this sandboxed shell;
  no daemon process visible in this shell's process view.
- Verdict: restriction is the sandbox boundary, not host misconfiguration. No
  permission changes made or needed (prompt4 §A1 prohibitions respected).

Precise action required: run the PG19 gate commands below from the owner's
unsandboxed shell (where group membership already permits daemon access), or
grant this agent's shell docker API access. P0 stays red until then; no
framework-dependent P2 work starts.

Run in an authorized environment with PG19 (see miniapp `compose.yaml` /
`.github/workflows/ci.yml` for the pinned image and service shape):

```bash
export TEST_DATABASE_URL=postgres://miniapp:miniapp-ci@localhost:5432/miniapp_dev
export TEST_DBOS_DATABASE_URL=$TEST_DATABASE_URL
export MINIAPP_POLICY=strict CI=true
go test -p 1 ./...
go test ./executor/dbos/ -run 'TestSleepAcrossRestart' -count=1 -v  # must execute, not skip
go test ./probe/classd/ -run 'TestProbeRestart' -count=1 -v         # must execute, not skip
go test -race ./contract/ ./internal/... ./spikes/
```

## Still outstanding (hard gate before any reliance on the pin)

- PG-backed framework suite above, all green.
- Studio import spike: add `require github.com/PithomLabs/miniapp` at the exact
  passing SHA, run one Class A invoke + one approval park/decide round-trip.
- Until both pass, the pin stays provisional and no P1/P3 feature work may rely
  on the framework.

## P0 completion record (2026-11-24, authorized unsandboxed shell)

- Framework SHA: `73351a801fc54e19ee7254e6fe3ea7fbed859378` (= `origin/main`).
- Postgres: dedicated `miniapp-p0-pg` container, image digest
  `sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60`
  (exact compose pin), server `19.0 (Debian 19.0-1.pgdg13+1)` on
  127.0.0.1:5434. Owner's `miniapp-dev-db-1` was never touched. DB credential
  was generated to a 600 file and never emitted; the one CLI-exposed value was
  rotated before any test ran.
- `gofmt -l .`: clean. `go vet ./...`: pass.
- `go test -p 1 ./...` (MINIAPP_POLICY=strict, CI=true): exit 0, 22 packages
  ok, zero failures.
- `TestSleepAcrossRestart`: EXECUTED and PASS (15.2s).
- `TestProbeRestart`: EXECUTED and PASS (15.3s).
- `go test -race ./contract/ ./internal/... ./spikes/`: all ok.
- Studio import spike (`spike/`, require pinned at
  `v0.0.0-20261123181255-73351a801fc5`): `TestImportClassAInvoke`,
  `TestApprovalParkDecideResume` (park → decide → same-key re-Invoke executes
  exactly once; `Status` observation-only; terminal replay and key-reuse
  conflict verified), `TestApprovalDenySettlesTerminal` — all PASS. One
  test-authoring correction during the spike: replayed results compare
  semantically (jsonb text form), not byte-wise.
- Studio CI gained the PG19 service so the spike executes (never skips) there.

**P0 exit decision: PASS.** The pin `73351a8` is finalized for Studio reliance.
B2+ framework-dependent work is unblocked. External writes remain disabled
until G-01 (unchanged).
