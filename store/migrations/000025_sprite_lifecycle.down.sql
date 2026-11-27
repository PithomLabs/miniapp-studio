-- Down migration for 000025: remove lifecycle receipt persistence.
-- Destroys recorded transition receipts by design (down is for version
-- staging in tests, never production rollback).
DROP TABLE studio.sprite_lifecycle_receipts;
