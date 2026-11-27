# Local UI serving

Two commands, first run to reviewing:

```
task dev:serve-init   # once: serve DB, app role, operator, keys/grants
task dev:serve        # every time: serves on http://127.0.0.1:8080 (Ctrl-C stops)
```

Sign in with the login token stored in the file path printed by `serve-init`
(default `$HOME/.config/miniapp-studio/dev-login-token.txt`).

## What it does

- Uses the bound Neon branch (`dev bind` first) and the shared dev database,
  `studio_dev_ui` — the same database `task dev:init` ensures (one database,
  one code path). Non-run-pattern is deliberate: test tooling (`dev reset`,
  run reaping) refuses non-run databases, so review data is never
  force-cleaned. Tests never use it: every test process provisions its own
  disposable run database.
- Every dev command refuses production bindings: platform-protected
  branches and `production`/`prod` names fail closed (bind refuses
  production names before any Neon call; the platform marker is checked
  after branch resolution but before the connection string is fetched or
  anything is written). Mark production branches Protected in the Neon
  dashboard; bindings predating branch identity must re-bind once.
- Applies the framework baseline + Studio migrations, ensures the app role
  and one dev operator, and writes keys/grants in the tested shapes.
- Serves loopback-only with Fly/gateway/sender unset. Deploy pages report
  "not configured" honestly; no cloud mutations are possible from this path.

## Safety rules (also enforced in code + unit tests)

- Secrets are generated (crypto-random, 0600) on first run, never defaulted,
  never printed, never overwritten. Re-running init is idempotent.
- An existing app role with a missing password file is a hard STOP, never
  an implicit rotation. Re-run with `task dev:serve-init -- --rotate-app-password`
  to deliberately generate-and-set it (a loud warning names what breaks);
  anything else using the old password must be updated by hand.
- Existing keys/grants/operators files are merged or verified, never
  clobbered; foreign content fails with instructions, not silent edits.
- Only `dev:bind` files + this page's conventional paths are read; override
  any path per run (`task dev:serve SERVE_LISTEN=127.0.0.1:8081`).

Local review only — nothing here provisions shared, staging, or production
resources. Tear down the serve database from the Neon dashboard when done
with it (`dev reset` will not touch it by design).
