-- 000013: command log and ingested event tables (Phase A, gateway plan §7).
-- commands is Studio's retry source of truth (payload stored, not just its
-- hash); events is the deduped ingest log keyed by (build, event_id).
-- Per-build seq preserves gateway protocol ordering separately from the
-- conversation-stream cursor (migration 000014, decision D1).

CREATE TABLE studio.commands (
  id           TEXT NOT NULL PRIMARY KEY,
  tenant_id    TEXT NOT NULL,
  build_id     TEXT NOT NULL,
  kind         TEXT NOT NULL
    CHECK (kind IN ('assign', 'continue', 'cancel', 'status', 'fetch_artifact')),
  payload      JSONB NOT NULL DEFAULT '{}',
  payload_hash TEXT NOT NULL DEFAULT '',
  state        TEXT NOT NULL DEFAULT 'queued'
    CHECK (state IN ('queued', 'sending', 'acked', 'failed')),
  attempts     INTEGER NOT NULL DEFAULT 0,
  last_error   TEXT NOT NULL DEFAULT '',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  FOREIGN KEY (tenant_id, build_id)
    REFERENCES studio.builds (tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX commands_build ON studio.commands (tenant_id, build_id, created_at);

CREATE TABLE studio.events (
  tenant_id   TEXT NOT NULL,
  build_id    TEXT NOT NULL,
  event_id    TEXT NOT NULL,
  seq         INTEGER NOT NULL DEFAULT 0,
  type        TEXT NOT NULL DEFAULT '',
  payload     JSONB NOT NULL DEFAULT '{}',
  received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, build_id, event_id),
  FOREIGN KEY (tenant_id, build_id)
    REFERENCES studio.builds (tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX events_build_seq ON studio.events (tenant_id, build_id, seq);

CREATE POLICY tenant_isolation ON studio.commands
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation ON studio.events
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
ALTER TABLE studio.commands FORCE ROW LEVEL SECURITY;
ALTER TABLE studio.events FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON studio.commands, studio.events TO studio_app;
