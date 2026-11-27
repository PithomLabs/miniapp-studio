# Development database contract

The canonical interface for database-backed development and testing,
Neon-first (ADR-004/006, implemented T1 repo-local 2026-11-27).
One contract, every environment (dev branch, Sprite execution, CI
branch): same commands, same migrations, same suite. Environment
differences live in configuration (which Neon branch binding),
never in forked logic. No self-hosted PostgreSQL exists in this
contract; loopback and private hosts are refused outright.

## Entry points

`task dev:<name>` (see `Taskfile.dev.yml`) — each task invokes a
`studio dev` subcommand implemented in Go and unit-tested:

| Task | Command | Purpose |
|---|---|---|
| `dev:setup` | `studio dev setup` | Neon preflight: metadata, endpoint binding, TLS, live version (provisions nothing) |
| `dev:doctor` | `studio dev doctor` | Diagnose toolchain, endpoint, database, migrations; fails loudly on any violated invariant |
| `dev:init` | `studio dev init` | Ensure the shared dev database, idempotent (explicit `--db-name` provisions scratch instead) |
| `dev:serve-init` / `dev:serve` | `studio dev serve-init` / `serve` | One-time local UI setup, then loopback serving (see `docs/dev-serve.md`) |
| `dev:reset` | `studio dev reset` | Drop stale run databases (`--dry-run` previews, `--yes` executes) |
| `dev:migrate` | `studio migrate` | Migrations on the bound branch (binding required; refuses production) |
| `dev:test` | `studio dev test --db-only` | Database-touching test packages, serially |
| `dev:test-full` | `studio dev test` | Complete suite, exactly as CI runs it |
| `dev:fingerprint` | `studio dev fingerprint` | Machine-readable environment record (JSON, R312 fields) |
| `dev:bootstrap` | chain | setup → doctor → init → test for a fresh environment |
| `dev:parity-capture` | `studio dev fingerprint` | Capture the fingerprint for cross-environment comparison |

Branch binding arrives via `NEON_PROJECT_ID`, `NEON_BRANCH_ID`,
`NEON_PARENT_BRANCH` (optional), `NEON_ENDPOINT_HOSTS`, `NEON_BRANCH_NAME`,
and `NEON_BRANCH_PROTECTED`. The dev branch is owner-provisioned until
the T2 controller exists.

## Rules the interface enforces

- Endpoint policy (exact branch-endpoint binding + verify-full TLS)
  and the PG18 gate run before any connection is used for
  provisioning; violations fail with the exact cause, never silent
  fallback. Loopback, private, and unbound hosts are all refused,
  as is any sslmode weaker than verify-full (chain + hostname).
  Pools are built only via `dbenv.Pool`, which pins the verifying
  TLS config in code rather than trusting the DSN string.
- Only run-pattern databases (`studio_test_run_*`) are created,
  granted, reset, or dropped by dev tooling — plus the one shared dev
  database `studio_dev_ui`, which is explicitly exempt: test tooling
  (`dev reset`, run reaping) refuses non-run databases, so review data
  is never force-cleaned. Anything else is refused by name.
- Production guard: every dev command refuses platform-protected
  branches and `production`/`prod` names before touching Neon; bindings
  predating branch identity fail closed with a re-bind instruction.
  Re-bind after toggling protection (bindings attest point-in-time).
- `studio provision` and the e2e boundary provisioner take explicit
  operator-chosen DSNs and are intentionally unguarded: they are the
  operator's finger, not the dev path. Production promotion is
  manual-out-of-band until a governed path exists; raw
  `studio migrate` against production is refused (it always resolves
  a binding).
- Credentials arrive via `TEST_DATABASE_URL` (environment) or
  `--dsn-file` (file). They are never printed, never logged, and
  never appear in fingerprints, errors (redacted), or task output.
  Neon management API keys never enter this interface at all.
- `reset` requires `--yes`, skips databases younger than the
  threshold (default 30m) and any with connected backends, and
  reports exactly what it dropped and kept.
- The fingerprint carries versions, counts, Neon endpoint identity
  (project/branch/parent/host/port/TLS), and migration states —
  evidence for `docs/dev-parity-report.md`.

## Per-environment notes

- Dev branch: persistent Neon branch, owner-provisioned; same verbs
  against it from any shell with the binding + DSN.
- Sprite execution: same verbs run in the Sprite against its bound
  Neon branch over TLS; the Sprite runs no PostgreSQL server.
- CI branch: same verbs against the per-run ephemeral branch; see
  the T2 track plan for the controller wiring.
