# OpenCode run.ts @ v1.19.6 — vendored audit record

Provenance: `anomalyco/opencode`, tag `v1.19.6`,
path `packages/opencode/src/cli/cmd/run.ts` (31,940 bytes).
Git blob SHA-1 `944aec17abd75cc3272688e89a140b56cd2bb2d3`
(matches the tag tree entry — authenticity verified, not assumed).
SHA-256 `e040e90d4c81cdd59b9568ee90a53c0209cd9ddf1b8b207d1a8685b2e26d57d0`.
Fetched 2026-11-25 via the GitHub API tree + raw blob (hash-verified).

Only the excerpts below were copied; line numbers refer to the blob.
Full-file re-fetch + hash comparison re-verifies at any time.

## Verified claims (with evidence)

**1. The `question` tool is hard-denied in non-interactive runs
(lines 430–445).** The run command builds a permission ruleset that is
empty when interactive and denies `question`, `plan_enter`, and
`plan_exit` otherwise. There is no question-reply path in the run
loop (the `permission.asked` handler auto-replies once/reject for
*permission* prompts only). Consequence: Studio↔agent clarification
cannot use the native tool under `opencode run`; the file-drop
handshake is the required mechanism.

```ts
const rules: PermissionV1.Ruleset = interactive
  ? []
  : [
      { permission: "question", action: "deny", pattern: "*" },
      { permission: "plan_enter", action: "deny", pattern: "*" },
      { permission: "plan_exit",  action: "deny", pattern: "*" },
    ]
```

**2. `--format json` emits `{type, timestamp, sessionID, ...data}`
lines (lines 668–679).** Timestamp is `Date.now()` (millis); the
session id is the active run's.

```ts
function emit(type: string, data: Record<string, unknown>) {
  if (args.format === "json") {
    process.stdout.write(
      JSON.stringify({ type, timestamp: Date.now(), sessionID, ...data }) + EOL,
    )
    return true
  }
  return false
}
```

**3. The six NDJSON types the translator handles are exactly the
emitted set.** Observed `emit(...)` call sites: `text` (completed text
parts only — gated on `part.time?.end`), `tool_use`, `step_start`,
`step_finish`, `reasoning` (gated on completed + thinking flag), and
`error`. Foreign-session lines are skipped by the consumer (`part` /
`properties` session guards appear throughout the loop).

**4. Error lines are mid-run; the exit code terminalizes (lines
771–779, 830–832).** `session.error` events emit `error` lines and
accumulate into a local; the loop keeps streaming until idle; only
`finish()` maps accumulated error to `exitCode = 1`. Prompt/command
failures emit `error` and set exit 1 immediately. In this version
every error line is therefore followed by exit 1 — the F5 rule
(terminality from the exit code, error lines as diagnostic progress)
coincides here and stays honest if that coupling ever loosens.

**5. Exit semantics: 0 when idle without session errors, 1 on any
error.** Every failure path sets `process.exitCode = 1` or calls
`process.exit(1)`; success falls through to exit 0.

**6. `--session <id>` resumes a specific session `--dir`, `--model`,
`--agent`, `--continue`, `--format` all exist as options (lines
147–210).** Unknown `--session` ids fail honestly ("Session not
found", exit 1, no session lines) — the adapter's identity-less
stream refusal matches this shape.

## Live-confirmation remainder (P-G0/P-G1)

Vendored source proves the *code*; only live execution proves the
*behavior* on the builder (part internals beyond the guarded fields,
exact stdout interleaving, wake/exit timing). P-G0 (captured
transcript vs translator) and P-G1 (forced-error + clean exit codes)
remain the required live proofs.
