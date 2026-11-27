# RPA increment 1 (F0 secrets) — evidence

Scope: `plans/mom-n-pop-rpa-platform-plan.md` §9 increment 1 —
unified secrets inventory/loader/redaction, Taskfile verbs, server
adoption. No broker, no Fly sync, no tenant credentials.

## Scoping correction (verified before building)

Plan §5 assumed approvals, effects, and intent-hash lifecycles
were Studio gaps. Inspection proved all three already exist:
`store/` approval queue + `runtime.Service` resume (`server/`),
effect ledger (`app.effect_records` + `runtime.Service`), trust
tiers (`tools/`), intent-hash audit (`gateway.CallbackClaims` +
`build.intent_hash`). Increment 1 was re-scoped to the one
genuinely missing F0 surface: secrets management. No duplicate
machinery was built.

## Files

New: `secrets/registry.go`, `secrets/dotenv.go`,
`secrets/load.go`, `secrets/redact.go`,
`secrets/secrets_test.go`, `cmd/studio/devsecrets.go`,
`cmd/studio/devsecrets_test.go`, `docs/secrets.md`.
Modified: `cmd/studio/dev.go` (dispatch + usage),
`server/config.go` (loader adoption), `server/config_test.go`
(+2 tests), `Taskfile.dev.yml` (+2 tasks), `docs/taskfile.md`.

## Contract (behavioral, all test-pinned)

- Registry: 11 variables; rejects dupes, bad names, bad enums,
  and secrets-with-defaults structurally.
- Precedence flag > env > _FILE > dotenv > default(config
  only); multi-source sets fail naming every source; secrets
  reject CLI flags.
- Dotenv: strict KEY=VALUE subset (no export/interpolation/
  multiline/quoting); unknown keys, dupes, empties rejected;
  64 KiB file / 16 KiB value bounds.
- Permission audit: group/other bits refuse (0600/0400 pass);
  server audits DSN, operators, keys, grants, Sprites-token files.
- Redaction: per-line secret assignment + DSN-shape scrubbing;
  identities/config pass through.
- Verbs: `dev secrets-validate` (parse/audit/status, names
  only), `dev secrets-plan` (per-env key scope, local only).
  Canary test proves values never print. No `--show` path exists.

## Behavior deltas on server adoption (deliberate, tested)

1. Credential files must now be owner-only (all Studio tooling
   already writes 0600; a legacy looser file fails boot loudly).
2. `_FILE` siblings of previously-direct vars (LISTEN, SENDER,
   paths) now conflict-fail instead of being silently ignored.
3. Valid configs: zero behavior change — the full pre-existing
   server suite passes unmodified.

## Gates (this session, local)

- `gofmt -l .`: clean. `go build ./...`, `go vet ./...`: pass.
- `go test ./... -count=1`: **460 pass / 311 skip / 0 fail**
  (baseline 448/311/0; +12 = 8 secrets + 2 verb + 2 server).
- Taskfile smoke: `dev:secrets-validate` and `dev:secrets-plan`
  exercised with fixture dotenv (deleted after); failures naming
  keys verified for unknown-key and 0644 paths.
- DB-backed tests skip honestly without Neon (unchanged
  policy); no live runs in this increment.

## Open decision (owner): tenant credential broker shape

Increment 2 needs tenant provider credentials (Google/Microsoft
OAuth, etc.). Two shapes:

- **(a) Server-side mint:** broker holds refresh tokens,
  mints short-lived access tokens per execution. Smaller blast
  radius per execution; broker is a high-value service.
- **(b) DB-held retrieval:** broker retrieves stored
  credentials at execution time under RLS. Simpler; credentials
  at rest need envelope encryption + rotation story.

Recommendation: (a) — matches the existing server-side
`framework_credential` pattern and keeps tenant secrets out of
ambient process state entirely. Awaiting owner decision before
increment 2.
