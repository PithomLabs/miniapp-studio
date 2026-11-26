-- Down migration for 000021: remove the artifact pipeline
-- persistence. Destroys review evidence by design (down is for
-- version staging in tests, never production rollback of
-- decisions — approvals rows survive, minus the D1 columns).

DROP INDEX studio.approvals_artifact_unique;
ALTER TABLE studio.approvals DROP COLUMN test_run_id;
ALTER TABLE studio.approvals DROP COLUMN verification_id;
ALTER TABLE studio.approvals DROP COLUMN artifact_digest;
ALTER TABLE studio.artifacts DROP CONSTRAINT artifacts_storage_ref_fkey;
UPDATE studio.artifacts SET storage_ref = '' WHERE storage_ref IS NULL;
ALTER TABLE studio.artifacts ALTER COLUMN storage_ref SET NOT NULL;
ALTER TABLE studio.artifacts ALTER COLUMN storage_ref SET DEFAULT '';
DROP TABLE studio.artifact_test_runs;
DROP TABLE studio.artifact_verifications;
DROP TABLE studio.artifact_blobs;
