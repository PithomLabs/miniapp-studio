# Taskfile reference — `task dev:*` commands

The complete operator reference for the project's Taskfile layer. Two files:
`Taskfile.yml` (root; includes the `dev` namespace) and `Taskfile.dev.yml`
(all tasks). Tasks orchestrate the canonical `studio dev` Go verbs — they
never reimplement application logic. Every task runs `go run ./cmd/studio`
from the repo root, so no build step or install is needed.

Related contracts: [dev-contract](dev-contract.md) (database lifecycle rules),
[dev-serve](dev-serve.md) (local UI serving in depth).

## Concepts

**File handoff, never transcription.** `dev:bind` writes two files: the branch
binding (`./dev-neon.json`, repo-local, gitignored, identities only) and the
DSN (`~/.config/miniapp-studio/dev.dsn`, outside the repo, mode 0600).
Every other task consumes both by path. Credentials never appear in task
output, logs, or chat.

**Overridable vars.** Every path and name is a Taskfile var; override per run:

```
task dev:setup DSN_FILE=/elsewhere/dev.dsn
```

| Var | Default | Purpose |
|---|---|---|
| `METADATA_FILE` | `./dev-neon.json` | Branch binding from `dev:bind` |
| `DSN_FILE` | `$HOME/.config/miniapp-studio/dev.dsn` | Owner database URL (0600) |
| `NEON_PROJECT` / `NEON_BRANCH` | `miniapp-studio-dev` / `dev` | Bind targets |
| `SERVE_LISTEN` | `127.0.0.1:8080` | Local UI listen address |
| `SERVE_OPERATORS_FILE` … `SERVE_TOKEN_FILE` | `$HOME/.config/miniapp-studio/…` | Serve credential files |
| `SERVE_MARKER_FILE` | `./dev-serve.json` | Init-done record (gitignored) |

**Production guard.** Every `dev:*` command refuses platform-protected
branches and `production`/`prod` names before touching Neon; bindings
predating branch identity fail closed with a re-bind instruction. Mark
production branches Protected in the Neon dashboard.

**Two database classes.** The shared dev database `studio_dev_ui`
(idempotent, reused, never reaped) versus disposable run-pattern
databases `studio_test_run_*` (fresh per test process, reaped when
stale). Tests never touch the shared database.

## Setup tasks

### `task dev:bind`
Binds a Neon project/branch by name. Resolves IDs via the local `neon`
CLI, records branch identity (name + platform protection marker), refuses
production targets, streams the connection string directly into the 0600
DSN file (refuses to overwrite — delete explicitly to re-fetch), and
writes the binding file. Re-run after toggling branch protection or when
told a binding predates branch identity. If the DSN step fails, the CLI's
sanitized diagnostic is shown (DSN-shaped text is redacted).

### `task dev:setup`
Neon preflight: prints bound project/branch, then verifies endpoint
binding, verify-full TLS, and the live server version. Provisions
nothing. First check when anything downstream misbehaves.

### `task dev:doctor`
Full diagnosis: toolchain, endpoint, database, and migrations, ending in
an explicit pass/fail per check (including the production-guard check).
Fails loudly with causes, never silently.

### `task dev:diag-roles`
Records `studio_lookup` capability booleans (member/set/admin/inherit) to
`./dev-diag-roles.json`. Secret-free; used to diagnose 42501-class
permission failures with evidence instead of guesses.

## Database tasks

### `task dev:init`
Ensures the shared dev database (**idempotent**): creates `studio_dev_ui`
if missing, then always re-verifies (framework baseline, Studio
migrations, roles, clean version match). Re-running prints
`reusing database studio_dev_ui (re-verified clean: …)` and changes
nothing. With `--db-name <run-pattern-name>` it provisions one scratch
run database instead (refuses existing names).

### `task dev:migrate`
Runs pending Studio migrations on the **bound branch** through the
bounded adapter (per-operation deadlines, lock + version protocol). The
binding is required precisely so the production guard engages; raw
migrate-against-anything lives outside this task on purpose.

### `task dev:reset`
Drops stale run-pattern databases on the bound branch — and only those
(anything else is refused by name, so `studio_dev_ui` is immune).
Always preview first: `task dev:reset -- --dry-run`, then execute with
`task dev:reset -- --yes`. Refuses to run with neither flag.

## Test tasks

### `task dev:test`
Runs the database-touching packages serially (`-p 1`) through the bound
branch, handing the DSN to the test binary as child-only environment
(never logged). Extra flags pass through after `--`:

```
task dev:test -- -- -run 'TestFoo|TestBar' -timeout 15m -v -count=1 2>&1 | tee test-run-foo.log
```

The double `--` is load-bearing (task separator + passthrough
terminator); `-count=1` defeats result caching; `set -o pipefail` first
so a failing suite can't hide behind `tee`'s exit code. Tests that need
live Neon skip honestly without it.

### `task dev:test-full`
The complete suite (`./...`) serially through the bound branch. Slow
(full live coverage); prefer scoped `-run` gates for iteration and
reserve this for pre-demo verification.

### `task dev:bootstrap`
The fresh-environment chain: `setup` → `doctor` → `init` → `test`, in
order, stopping at the first failure.

## Fingerprint tasks

### `task dev:fingerprint`
Prints the machine-readable environment record (JSON, R312 fields: toolchain,
project/branch identity, TLS posture, migration versions). No credentials.

### `task dev:parity-capture`
Captures the fingerprint to `./dev-parity.local.json` (gitignored) for
cross-environment comparison (local vs Sprite vs CI).

## Secrets tasks

### `task dev:secrets-validate`
Validates a dotenv file against the secrets registry (strict
`KEY=VALUE` subset; unknown keys, duplicates, empties, and `export`
rejected) and reports every registered variable's status (names
only — values never print). Extra secret files audit via
`-- --check-files a,b` (must be owner-only; see
[secrets](secrets.md)). `ENV_FILE` overrides the path (default
`./dev.env`).

### `task dev:secrets-plan`
Shows the sync plan for one environment (`ENV`, default `dev`):
registered keys in scope with sensitivity and rotation, and whether
each value is present in the dotenv file. Local planning only —
performs no Fly/network operations.

## Broker tasks (tenant credentials, option A)

### `task dev:broker-init-kek`
Generates the broker KEK (32 random bytes, 64 hex chars) into a
0600 file. Refuses to overwrite; `KEK_FILE` overridable (default
`$HOME/.config/miniapp-studio/broker-kek`). Run once per
environment, from the owner terminal.

### `task dev:broker-enroll`
Seals a tenant/provider refresh token into `studio_dev_ui`
(`TENANT`, `PROVIDER`, `REFRESH_FILE` — no defaults; the refresh
file must be 0600 and is never a flag). Prints names only.
Re-enrollment overwrites and clears revocation (re-consent path).

### `task dev:broker-revoke`
Marks a credential revoked; future mints fail. Missing credentials
are a typed error, not silent success.

### `task dev:broker-leases`
Lists lease metadata for a tenant (scope + lifecycle only;
`-- --provider P` filters). Tokens are never stored, so there is
nothing to leak. There is deliberately no mint verb: minting
happens only inside broker process memory (see
[broker](broker.md)).

## Invitation tasks (tenant onboarding)

### `task dev:invite-create`
Creates an invitation for `BINDING`, invited by `INVITED_BY`, in
`TENANT` (no defaults). The raw token goes to a new 0600
`TOKEN_FILE` (never printed); deliver it out-of-band, then delete
the file. TTL via `-- --ttl-hours N` (default 168, max 720).

### `task dev:invite-accept`
Accepts the token in `TOKEN_FILE` (0600) into a session in one
atomic transaction; the session token goes to a new 0600
`SESSION_TOKEN_FILE`. Failures are uniform (no state oracle).

### `task dev:invite-revoke`
Revokes invitation `INVITE_ID` in `TENANT` immediately.

### `task dev:invite-resend`
Rotates an invitation: revokes the old token, issues a fresh one
to `TOKEN_FILE` for the same binding.

### `task dev:invite-list`
Lists invitation metadata with live/consumed/revoked state. See
[invitations](invitations.md).

## Capability tasks

### `task dev:capability-refresh`
Refreshes the tenant's capability snapshot (tool registry +
connector state) and prints the summary with staleness. Connector
state honestly reports unconfigured from this verb (no manager
attached). See [capabilities](capabilities.md).

## Serve tasks (local UI)

### `task dev:serve-init`
One-time setup for local UI review. Ensures the shared dev database,
resolves the app-role password through an explicit matrix (generates
0600 files when missing; an existing role with a missing file is a hard
stop unless re-run with `-- --rotate-app-password`, which rotates
loudly), provisions the dev operator, writes keys/grants in tested
shapes (verifying — never editing — existing files), and records the
`./dev-serve.json` marker. Prints the login-token **path** (never the
value) and the next command. Idempotent: safe to re-run.

### `task dev:serve`
Serves the Studio UI on `SERVE_LISTEN` (default
`http://127.0.0.1:8080`; loopback IPs only — anything else is refused).
Requires the serve-init marker. Assembles all secrets in-process into
child-only environment and execs `studio serve`; stop with Ctrl-C. Fly,
gateway, and sender integrations stay unset, so cloud-backed UI honestly
reports "not configured".

## Common workflows

**First day on a machine:** `dev:bind` → `dev:setup` → `dev:init` →
`dev:test -- -- -run <gate>` for whatever you touch.

**Review the UI:** `dev:serve-init` (once; add `-- --rotate-app-password`
if it stops on a pre-existing role) → `dev:serve` → open the printed URL
→ sign in with the token from the printed file path.

**After pulling upstream:** `dev:setup` (confirms binding + versions) →
`dev:init` (applies any new migrations to the shared DB, no-op otherwise).

**Something broke:** `dev:doctor` first (it names the violated invariant),
`dev:diag-roles` for permission errors, `dev:reset -- --dry-run` if stale
run databases are suspected.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `predates branch-identity metadata` | Binding written before the guard existed | `rm $DSN_FILE && task dev:bind` (re-fetches same credential) |
| `refusing to overwrite …dev.dsn` | Rotation protection | Delete explicitly, then re-run (see above) |
| `refusing protected/denylisted branch` | Prod guard engaged | Bind a development branch instead |
| `neon connection-string: exit status 1: …` | Neon CLI/API-side failure | The sanitized CLI diagnostic follows the colon; fix that cause and re-run |
| `role exists but password file is missing` | App role predates filed credentials | Re-run `serve-init` with `-- --rotate-app-password` |
| Serve boots but Deploy says "not configured" | Expected | Fly credentials are unset by design for local review |
| `missing ./dev-serve.json` | Serve without init | Run `task dev:serve-init` first |
