# Task Cards — file convention

GitHub Project #3 is authoritative for ownership, priority, and status. This directory holds the
canonical detailed brief + evidence references for each card. The owner mirrors status changes to the
Project; the agent never moves board items directly.

- One file per card: `NNN-short-slug.md`, zero-padded sequence, never reused.
- Copy `TEMPLATE.md`. Fill every section; no placeholders at authorization time.
- Status values: `draft` → `authorized` → `in-progress` → `in-review` → `done` | `blocked`.
  Only the owner sets `authorized`; the agent advances the rest by editing the file.
- Seven-field brief is mandatory: execution location, application target, database target,
  infrastructure actions, authorization class, verification/evidence, recovery/rollback.
- Evidence: link gitignored log files, test counts, and exact commands. Board status must be
  reproducible from the file alone.
- Single active card by default; parallel cards require isolated worktrees + isolated DB state,
  stated on the card.
