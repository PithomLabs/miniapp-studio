-- Restore the pre-000018 commands shape exactly.
DROP INDEX studio.commands_due;
ALTER TABLE studio.commands DROP COLUMN next_attempt_at;
