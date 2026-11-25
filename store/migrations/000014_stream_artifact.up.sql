-- 000014: conversation stream, artifacts, and approvals (Phase A).
-- conversation_stream is decision D1: one row per renderable message or
-- agent event with a monotonic per-conversation sequence allocated
-- transactionally with the source insert (advisory-lock MAX+1 in the
-- tools). Aborted transactions can leave gaps; cursor replay needs
-- greater-than only, never contiguity. Payloads stay in messages/events;
-- the stream stores references.
-- SSE replay uses seq as Last-Event-ID. Artifacts are immutable verified
-- review bundles; approvals record artifact decisions and action grants
-- (artifact_id NULL means a grant for an authorization_request question).

CREATE TABLE studio.conversation_stream (
  tenant_id       TEXT NOT NULL,
  conversation_id TEXT NOT NULL,
  seq             BIGINT NOT NULL,
  kind            TEXT NOT NULL CHECK (kind IN ('message', 'event')),
  ref_id          TEXT NOT NULL,
  build_id        TEXT NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, conversation_id, seq),
  FOREIGN KEY (tenant_id, conversation_id)
    REFERENCES studio.conversations (tenant_id, id) ON DELETE CASCADE
);

CREATE TABLE studio.artifacts (
  id          TEXT NOT NULL PRIMARY KEY,
  tenant_id   TEXT NOT NULL,
  build_id    TEXT NOT NULL,
  kind        TEXT NOT NULL CHECK (kind IN ('diff', 'spec', 'test_report', 'bundle')),
  digest      TEXT NOT NULL,
  size_bytes  BIGINT NOT NULL DEFAULT 0,
  storage_ref TEXT NOT NULL DEFAULT '',
  manifest    JSONB NOT NULL DEFAULT '{}',
  verified_at TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, build_id, digest),
  FOREIGN KEY (tenant_id, build_id)
    REFERENCES studio.builds (tenant_id, id) ON DELETE CASCADE
);

CREATE TABLE studio.approvals (
  id          TEXT NOT NULL PRIMARY KEY,
  tenant_id   TEXT NOT NULL,
  build_id    TEXT NOT NULL,
  question_id TEXT NOT NULL DEFAULT '',
  artifact_id TEXT NOT NULL DEFAULT '',
  decision    TEXT NOT NULL CHECK (decision IN ('approved', 'rejected')),
  decided_by  TEXT NOT NULL DEFAULT '',
  decided_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  note        TEXT NOT NULL DEFAULT '',
  UNIQUE (tenant_id, id),
  FOREIGN KEY (tenant_id, build_id)
    REFERENCES studio.builds (tenant_id, id) ON DELETE CASCADE
);

CREATE POLICY tenant_isolation ON studio.conversation_stream
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation ON studio.artifacts
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation ON studio.approvals
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
ALTER TABLE studio.conversation_stream FORCE ROW LEVEL SECURITY;
ALTER TABLE studio.artifacts FORCE ROW LEVEL SECURITY;
ALTER TABLE studio.approvals FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON studio.conversation_stream, studio.artifacts, studio.approvals TO studio_app;
