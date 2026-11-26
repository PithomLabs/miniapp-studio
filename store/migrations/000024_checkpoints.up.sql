-- B1: durable checkpoints for local interruption recovery.
-- One row per captured recovery point; ordering per build via seq
-- assigned under the locked build row. Contents are references and
-- digests only — never source blobs, credentials, or transcripts.
CREATE TABLE studio.checkpoints (
  id            TEXT NOT NULL PRIMARY KEY,
  tenant_id     TEXT NOT NULL,
  build_id      TEXT NOT NULL,
  attempt_n     INTEGER NOT NULL CHECK (attempt_n >= 1),
  turn_id       TEXT NOT NULL,
  version       TEXT NOT NULL DEFAULT 'v1' CHECK (version IN ('v1')),
  complete      BOOLEAN NOT NULL,
  seq           INTEGER NOT NULL CHECK (seq >= 1),
  repo_path     TEXT NOT NULL DEFAULT '',
  git_branch    TEXT NOT NULL DEFAULT '',
  git_head      TEXT NOT NULL DEFAULT '',
  git_clean     BOOLEAN NOT NULL DEFAULT false,
  manifest      JSONB NOT NULL DEFAULT '{}',
  context       JSONB NOT NULL DEFAULT '{}',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, build_id, seq),
  FOREIGN KEY (tenant_id, build_id)
    REFERENCES studio.builds (tenant_id, id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, turn_id)
    REFERENCES studio.turns (tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX checkpoints_latest ON studio.checkpoints (tenant_id, build_id, complete, seq DESC);
CREATE POLICY tenant_isolation ON studio.checkpoints
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
ALTER TABLE studio.checkpoints ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.checkpoints FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON studio.checkpoints TO studio_app;
