# Boundary B U5 acceptance harness

Reviewable, resumable browser acceptance for the Focus Console.
Replaces the monolithic `/tmp` matrix script: each phase runs
standalone against existing state, asserts its prerequisites
first, and records verified outputs in a run-scoped ledger.

## Layout

- `run.py` — orchestrator: `--init` provisions a run,
  `--phase N` runs one phase, `--all` runs all six in order.
- `phases/phaseN_*.py` — one independently runnable phase each.
- `lib/harness.py` — CDP client, login, psql, callback signer,
  Chrome launcher, screenshots.
- `lib/ledger.py` — ledger schema, atomic read/write, validation
  (`python3 lib/ledger.py` runs its self-test).
- `lib/phase.py` — per-phase context, prerequisite helpers,
  derived timing constants, attempt nonces.
- `lib/serverctl.py` — studio + stub lifecycle for runs.
- `stub/gateway.py` — stub gateway (records commands, mimics
  200/409/404 shapes; never an agent).
- `scripts/provision_run.sh` — database, secrets, policy files
  for one run (invoked by `run.py --init`).
- `provision_test.go` — committed run provisioner (schema +
  operators), replacing the old throwaway setup test. Skips
  without `BOUNDARY_DSN_FILE`/`BOUNDARY_DIR`.

Runtime state (ledgers, browser profiles, screenshots, logs)
lives under `/tmp/boundary-b/runs/<run-id>/`, never here.

## Environment

- `STUDIO_BASE` (default `http://127.0.0.1:18082`),
  `BOUNDARY_RUN_DIR` (default `/tmp/boundary-b`).
- `BOUNDARY_HOME/owner.dsn` (default `/tmp/boundary-b/owner.dsn`)
  must exist before `--init`: operator-bootstrapped once per
  cluster from `docker inspect`, mode 0600.
- Operator tokens and DSNs come from the run directory's 0600
  files, read by phases at runtime — never committed, never
  logged, never stored in the ledger.

## Rules (from the U5 runbook)

- Prerequisites before work; `PREREQ:` failures, never mystery
  DOM asserts against a foreign environment.
- Ledger-recorded IDs and relative assertions only.
- A failed phase destroys nothing; rerun that phase alone.
- Clean-slate acceptance = two consecutive full green runs on
  independently provisioned databases.

## Canonical commands

```bash
# New run (fresh DB + secrets + servers + ledger):
python3 run.py --init --ledger /tmp/boundary-b/runs/<run-id>/ledger.json --db studio_boundary_bN

# One phase against existing state:
python3 run.py --phase 3 --ledger /tmp/boundary-b/runs/<run-id>/ledger.json --db studio_boundary_bN

# All phases in order (stops at first failure; dependents block):
python3 run.py --all --ledger /tmp/boundary-b/runs/<run-id>/ledger.json --db studio_boundary_bN
```
