# C1 Command-Row Upgrade Restriction (F6)

## Rule

Any database carrying `studio.commands` rows written before the C1
sender (Boundary A) must start fresh or backfill before the sender
runs. There is no silent upgrade path for legacy rows.

## Why

Pre-C1 `question.answer` logged `continue` rows without a frozen
`turn`. On first claim the sender terminal-fails such rows
("continue row lacks a frozen turn"), and the question is already
`answered`, so it can never be re-answered: the authorized work
strands visibly but unrecoverably. Pre-C1 `assign` rows are safe
(the sender resolves message-less prompts and now freezes them
per F2).

## Allowed procedures

- **Fresh database** (demo path): migrate an empty database to the
  current version. All command rows are then C1-shaped.
- **Backfill** (only if legacy rows must survive): before starting
  the sender, set a frozen turn on every `queued` continue row
  lacking one, derived from the owning build's tracked
  `gateway_turn` plus one, in ascending `created_at` order per
  build. Verify every row carries `turn` before the first poll;
  the sender still fails any row the backfill missed rather than
  guessing.

## Never

- Never run the C1 sender against legacy rows expecting silent
  recovery; the fail-fast is intentional.
- Never re-answer an `answered` question to "fix" a stranded
  continue; the tool refuses, by design.
