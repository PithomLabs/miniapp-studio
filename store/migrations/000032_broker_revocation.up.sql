-- 000032: F35-05 versioned revocation + F35-14 broker DELETE
-- tightening.
--
-- broker_credentials.version generations the credential: bumped on
-- every enroll and revoke. broker_leases.credential_version binds
-- each lease to the generation that issued it, so Mint finalization
-- can detect revocation-or-reenrollment that landed mid-exchange
-- and Consume can retire leases from dead generations.
-- Backfill is coherent by construction: pre-migration rows read
-- version 1 on both sides.
ALTER TABLE studio.broker_credentials ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE studio.broker_leases ADD COLUMN credential_version BIGINT NOT NULL DEFAULT 1;

-- F35-14: no code path deletes from the broker tables.
REVOKE DELETE ON studio.broker_credentials FROM studio_app;
REVOKE DELETE ON studio.broker_leases FROM studio_app;
