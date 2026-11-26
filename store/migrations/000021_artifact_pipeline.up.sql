-- 000021: artifact pipeline persistence (Phase D D1).
-- Content-addressed byte store, verification evidence, test-run
-- records, and digest-bound decision columns on approvals. No
-- conversation/Q&A table is touched.
--
-- Association model (contract v1, docs/artifact-contract-v1.md):
-- one decision unit = one bundle-kind artifacts row; member files
-- live as per-file blobs keyed by file digest; the bundle blob
-- (canonical encoding) is keyed by the root digest and referenced
-- from artifacts.storage_ref once verification assembles it.
-- Pre-pipeline claim rows keep storage_ref NULL: never trusted.

CREATE TABLE studio.artifact_blobs (
  tenant_id  TEXT NOT NULL,
  digest     TEXT NOT NULL,
  bytes      BYTEA NOT NULL,
  size_bytes BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, digest),
  CHECK (size_bytes >= 0 AND size_bytes <= 52428800)
);

CREATE TABLE studio.artifact_verifications (
  id                TEXT NOT NULL PRIMARY KEY,
  tenant_id         TEXT NOT NULL,
  artifact_id       TEXT NOT NULL,
  check_set_version TEXT NOT NULL,
  passed            BOOLEAN NOT NULL,
  checks            JSONB NOT NULL DEFAULT '[]',
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, artifact_id, check_set_version),
  FOREIGN KEY (tenant_id, artifact_id)
    REFERENCES studio.artifacts (tenant_id, id) ON DELETE CASCADE
);

CREATE TABLE studio.artifact_test_runs (
  id           TEXT NOT NULL PRIMARY KEY,
  tenant_id    TEXT NOT NULL,
  artifact_id  TEXT NOT NULL,
  status       TEXT NOT NULL
    CHECK (status IN ('queued', 'running', 'passed', 'failed', 'blocked')),
  input_digest TEXT NOT NULL DEFAULT '',
  report       JSONB NOT NULL DEFAULT '{}',
  evidence_id  TEXT NOT NULL DEFAULT '',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at  TIMESTAMPTZ,
  FOREIGN KEY (tenant_id, artifact_id)
    REFERENCES studio.artifacts (tenant_id, id) ON DELETE CASCADE
);

-- storage_ref becomes nullable (pre-pipeline '' backfilled to NULL)
-- and references the blob store. NULL skips the FK check.
UPDATE studio.artifacts SET storage_ref = NULL WHERE storage_ref = '';
ALTER TABLE studio.artifacts ALTER COLUMN storage_ref DROP NOT NULL;
ALTER TABLE studio.artifacts ALTER COLUMN storage_ref DROP DEFAULT;
ALTER TABLE studio.artifacts
  ADD CONSTRAINT artifacts_storage_ref_fkey
  FOREIGN KEY (tenant_id, storage_ref)
  REFERENCES studio.artifact_blobs (tenant_id, digest) ON DELETE CASCADE;

-- Digest-bound decision columns. The one-decision-per-artifact rule
-- is a PARTIAL unique: question-grant rows keep artifact_id = ''
-- and must never collide with artifact decisions.
ALTER TABLE studio.approvals ADD COLUMN artifact_digest TEXT NOT NULL DEFAULT '';
ALTER TABLE studio.approvals ADD COLUMN verification_id TEXT NOT NULL DEFAULT '';
ALTER TABLE studio.approvals ADD COLUMN test_run_id TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX approvals_artifact_unique ON studio.approvals (tenant_id, artifact_id)
  WHERE artifact_id <> '';

CREATE POLICY tenant_isolation ON studio.artifact_blobs
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation ON studio.artifact_verifications
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation ON studio.artifact_test_runs
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
ALTER TABLE studio.artifact_blobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.artifact_verifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.artifact_test_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.artifact_blobs FORCE ROW LEVEL SECURITY;
ALTER TABLE studio.artifact_verifications FORCE ROW LEVEL SECURITY;
ALTER TABLE studio.artifact_test_runs FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON
  studio.artifact_blobs, studio.artifact_verifications, studio.artifact_test_runs
  TO studio_app;
