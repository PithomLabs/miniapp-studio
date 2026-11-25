# Boundary B U5 Acceptance Runbook

> **CREDENTIAL MODEL (U5.12, permanent).** The interim
> credential-mutation restriction retired with role separation: the
> Go suite provisions isolated run databases and exercises
> application behavior as `studio_app_test`, and no test path
> rotates the live `studio_app` credential (proven by concurrent
> suite runs against a live server; see `docs/u5.12-report.md`).
> The one remaining serial operation is acceptance-run
> provisioning itself, which applies an explicit administrative
> rotation for the run's own credential: never provision two runs
> concurrently. If a live pass loses database connectivity, still
> abort, verify role configuration and connectivity, restart or
> reprovision, and begin a new pass.

## Environment

- PG19 cluster; each acceptance run uses its own isolated database
  (`studio_boundary_bN`, migrated + operators provisioned per run).
- Studio server binary (`go build ./cmd/studio`) on 127.0.0.1:18082
  with `STUDIO_SENDER=1`; stub gateway on 127.0.0.1:18093.
- Secrets live only in the run directory (`/tmp/boundary-b/`,
  mode 0700, files 0600) and process environment. Never in the
  repo, the ledger, logs, or the report.

## Harness layout (`e2e/boundary_b/`)

- `run.py` — orchestrator. Runs one phase
  (`--phase N --ledger <path>`) or all phases in order.
- `phases/` — one script per acceptance phase; each is
  independently executable against existing state.
- `lib/` — shared CDP client, login, psql helper, callback
  signer caller, stub-gateway manager, screenshot helper,
  ledger read/write helpers.
- Run-scoped runtime state (ledger, browser profiles,
  screenshots staging, logs) lives under
  `/tmp/boundary-b/runs/<run-id>/`, never in the repo.

## Phase order

1. Environment and authentication.
2. Conversation and dispatch.
3. Clarification lifecycle.
4. Approvals and Evidence.
5. Console behavior.
6. Recovery and isolation.
7. Final clean-slate acceptance (orchestrator entry; not a phase
   body — it provisions a fresh database and runs phases 1–6).

## Rules

- Every phase asserts its prerequisites first (`PREREQ:` failure,
  no DOM asserts before them).
- Assertions reference ledger-recorded IDs and relative changes
  only — no "latest row", no absolute counts, no epoch-second tags.
- A failed phase never destroys state; rerun that phase alone.
- Dependent phases report blocked, not failed, when a
  prerequisite phase has not passed.
- Clean-slate acceptance = two consecutive complete green runs,
  each on an independently provisioned database. Any failure
  restarts the count at zero.
