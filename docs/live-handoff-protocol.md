# Live handoff protocol (T1)

**Rule zero: chat is untrusted transport.** No identifier, hostname, DSN,
password, or API key crosses chat, clipboards, scrollback-paste, or screenshots —
secret or otherwise. All owner↔agent handoff moves through files defined here.
The owner acknowledges completion with the word "done"; the agent verifies from
files, never from pasted values.

## Files

| Path | Contents | May agent read? | Committed? |
|---|---|---|---|
| `./dev-neon.json` (repo root) | Branch binding: project/branch IDs, endpoint hostnames. Identities only, never credentials | YES — validated on every load (strict schema, exact-host policy) | NO (gitignored) |
| `~/.config/miniapp-studio/dev.dsn` | Maintainer DSN for the bound branch, mode-0600 | NEVER — outside the repo; owner-terminal processes only | NO (outside repo) |
| `./dev-parity.local.json` | Fingerprint evidence (redacted by construction, test-enforced) | YES | NO (gitignored) |

## Owner sequences (exact, zero transcription)

First bind (project and branch already exist in the dashboard):

```
task dev:bind
```

This resolves `miniapp-studio-dev`/`dev` by name, writes both files, and
enforces `sslmode=verify-full`. It never overwrites an existing DSN file.
Override names only if yours differ: `task dev:bind NEON_PROJECT=<name> NEON_BRANCH=<name>`.

Live T1 proofs (after ADR-004/005/006/010 acceptance + explicit live GO):

```
task dev:bootstrap
task dev:parity-capture
```

Then reply "done". The agent reads `./dev-neon.json` (binding sanity) and
`./dev-parity.local.json` (results) and reports verified vs pending.

Rotation: delete `~/.config/miniapp-studio/dev.dsn` explicitly, then re-run
`task dev:bind`. There is no overwrite path by design.

## Fail-closed points

- Binding loader rejects unknown fields, missing fields, non-hostname
  hosts, and loopback/private hosts — a tampered file cannot bind.
- Every verb re-validates the binding against the DSN (exact-host +
  verify-full) before connecting; file/DSN mismatch aborts.
- `dev bind` fails (never guesses) on missing/ambiguous project or
  branch names, missing endpoint data, or an unparseable DSN.
- DSN bytes are streamed file-to-file, scrubbed after use, and never
  logged; only the endpoint hostname is ever retained or printed.
- Evidence JSON cannot carry credentials (no such fields exist; the
  leak scan test enforces this on every run).
