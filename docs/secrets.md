# Secrets contract (F0)

One registry, one parser, one loader, one redactor. The `secrets`
package owns how Miniapp Studio names, reads, and scrubs
configuration; every surface (server boot, Taskfile verbs, future
Fly sync, future tenant broker) goes through it.

## Inventory

The canonical variable set lives in `secrets.StudioRegistry()`.
Each entry carries scope (`deployment` | `tenant`), sensitivity
(`secret` | `config`), rotation (`static` | `rotating` | `session`),
consumers, and environments. Tenant-scoped entries arrive with the
credential broker; the scope exists now so ambient tenant values
can't slip in later.

| Variable | Sensitivity | Rotation | Notes |
|---|---|---|---|
| `STUDIO_DATABASE_URL` | secret | static | App-role DSN |
| `STUDIO_OWNER_DATABASE_URL` | secret | static | Migration-role DSN |
| `STUDIO_GATEWAY_ORG_SECRET` | secret | rotating | Per-build HMAC root |
| `STUDIO_OPERATORS_FILE` | config | static | Path; file is audited |
| `MINIAPP_KEYS_FILE` | config | static | Path; file is audited |
| `MINIAPP_GRANTS_FILE` | config | static | Path; file is audited |
| `STUDIO_SPRITES_TOKEN_FILE` | config | rotating | Path; file is audited |
| `STUDIO_FLY_ORG` | config | static | |
| `STUDIO_LISTEN` | config | static | Default `127.0.0.1:8080` |
| `STUDIO_SENDER` | config | static | |
| `STUDIO_WORKSPACE_ROOT` | config | static | |

Registration rejects duplicate names, malformed names, unknown
enums, and — structurally — any secret carrying a default.

## Precedence and conflicts

`Registry.Load` resolves flag, then process env, then `_FILE`,
then dotenv, then the compiled default (config only). The same key
from more than one source fails closed naming every set source.
Secrets reject CLI flags outright (process listings are public).

## Dotenv subset

`ParseDotEnv` accepts blank lines, full-line `#` comments, and
`KEY=value` (value = everything after the first `=`). Rejected:
`export` prefixes, missing `=`, malformed keys, duplicates, empty
values, unknown keys (against the registry), oversize files
(64 KiB) and oversize values (16 KiB). No interpolation, quoting,
or multi-line values — use `_FILE` references for large blobs.

## Permission audit

`CheckFileMode` refuses any secret-bearing file with group/other
permission bits (0600 or stricter, e.g. 0400, passes). The server
audits every credential file it touches (DSN files, operators,
keys, grants, Sprites token). All Studio tooling already writes
0600; a legacy looser file now fails boot loudly instead of
leaking silently. Message names the path and mode — never content.

## Redaction (F35-09 disposition)

The former `Registry.Redact` helper was removed: it had zero
callers and demonstrably narrow coverage (JSON, bare values,
`export` lines, lowercase names, continuations all bypassed it),
so wiring it would have granted false confidence. The actual
controls are structural prevention (0600 file handoff,
identities-only output, no `--show` paths anywhere) plus the two
scoped redactors that remain: `redactDSN` (DSN credential spans in
CLI errors) and `RedactSpans` (credential-shaped spans in
tool/sandbox output). Each documents its own scope; neither claims
general secret detection.

## Verbs

- `task dev:secrets-validate` — parse + registry + statuses
  (names only) + optional `--check-files` audit.
- `task dev:secrets-plan` — per-environment key scope from a
  dotenv file. Local only; no Fly calls.

## Rotation

`rotating` marks values with a defined rotation path (gateway org
secret, Sprites token); `session` is reserved for broker-issued
tenant credentials. Rotation procedures live with their owners
(`docs/dev-serve.md` for the app role); this registry only declares
which keys rotate.

## Tenant broker (decided: option A, server-side mint)

Tenant variables (provider OAuth/API keys) will register with
`ScopeTenant` and resolve exclusively through the broker — never
env, never dotenv. No tenant-scoped entries exist yet; the scope
exists so ambient tenant values can't slip in later. The broker
shape decision closed in increment 2 (owner decision A):
refresh tokens sealed in Neon under a deployment KEK, short-lived
execution leases minted server-side; see `docs/broker.md`. The
KEK path travels as an explicit `--kek-file` verb flag (never
ambient, never a registry read); no `*_KEK_FILE` registry entry
exists (F35-10: the unused entry was removed).
