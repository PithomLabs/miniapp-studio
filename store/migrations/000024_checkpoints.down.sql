-- Down migration for 000024: remove checkpoint persistence.
-- Destroys B1 recovery points by design (down is for version
-- staging in tests, never production rollback — attempt events
-- and their records survive).
DROP TABLE studio.checkpoints;
