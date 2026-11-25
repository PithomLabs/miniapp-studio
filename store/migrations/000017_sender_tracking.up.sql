-- 000017: sender-loop tracking columns on studio.builds (Phase C1).
-- model: optional agent model override, set at build.create, passed to
--   the gateway assign command verbatim; empty means gateway default.
-- gateway_turn: last turn number the gateway acknowledged executing for
--   this build (assign completes turn 1). NULL means no confirmed
--   execution yet. The sender derives the next continue turn as
--   gateway_turn + 1; Studio turn numbers are NOT used because yield
--   parking and execution count diverge on failure and supersede paths.
ALTER TABLE studio.builds ADD COLUMN model TEXT NOT NULL DEFAULT '';
ALTER TABLE studio.builds ADD COLUMN gateway_turn INTEGER;
