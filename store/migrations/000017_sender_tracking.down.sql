-- Restore the pre-000017 builds shape exactly.
ALTER TABLE studio.builds DROP COLUMN gateway_turn;
ALTER TABLE studio.builds DROP COLUMN model;
