-- 000012: build, turn, and question tables (Phase A, gateway plan §7).
-- One build is one unit of delegated agent work; turns slice it between
-- yields; questions park turns until a user answers. Status/kind CHECKs
-- enforce the §6 state machines at the database layer underneath the
-- tool-level transition tables.

CREATE TABLE studio.builds (
  id               TEXT NOT NULL PRIMARY KEY,
  tenant_id        TEXT NOT NULL,
  conversation_id  TEXT NOT NULL,
  application_id   TEXT NOT NULL DEFAULT '',
  sprite_name      TEXT NOT NULL DEFAULT '',
  gateway_url      TEXT NOT NULL DEFAULT '',
  status           TEXT NOT NULL DEFAULT 'assigned'
    CHECK (status IN ('assigned', 'running', 'waiting_user', 'succeeded', 'failed', 'cancelled')),
  agent_kind       TEXT NOT NULL DEFAULT '',
  agent_session_ref TEXT NOT NULL DEFAULT '',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  FOREIGN KEY (tenant_id, conversation_id)
    REFERENCES studio.conversations (tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX builds_status ON studio.builds (tenant_id, status, updated_at);

CREATE TABLE studio.turns (
  id         TEXT NOT NULL PRIMARY KEY,
  tenant_id  TEXT NOT NULL,
  build_id   TEXT NOT NULL,
  n          INTEGER NOT NULL,
  status     TEXT NOT NULL DEFAULT 'open'
    CHECK (status IN ('open', 'waiting_user', 'answered', 'expired', 'closed')),
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ended_at   TIMESTAMPTZ,
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, build_id, n),
  FOREIGN KEY (tenant_id, build_id)
    REFERENCES studio.builds (tenant_id, id) ON DELETE CASCADE
);

CREATE TABLE studio.questions (
  id          TEXT NOT NULL PRIMARY KEY,
  tenant_id   TEXT NOT NULL,
  build_id    TEXT NOT NULL,
  turn_id     TEXT NOT NULL,
  kind        TEXT NOT NULL
    CHECK (kind IN ('clarification', 'proposal', 'authorization_request')),
  body        TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL DEFAULT 'open'
    CHECK (status IN ('open', 'answered', 'expired', 'withdrawn')),
  answer      TEXT NOT NULL DEFAULT '',
  answered_by TEXT NOT NULL DEFAULT '',
  answered_at TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  FOREIGN KEY (tenant_id, build_id)
    REFERENCES studio.builds (tenant_id, id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, turn_id)
    REFERENCES studio.turns (tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX questions_build ON studio.questions (tenant_id, build_id, status);

CREATE POLICY tenant_isolation ON studio.builds
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation ON studio.turns
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation ON studio.questions
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
ALTER TABLE studio.builds FORCE ROW LEVEL SECURITY;
ALTER TABLE studio.turns FORCE ROW LEVEL SECURITY;
ALTER TABLE studio.questions FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON studio.builds, studio.turns, studio.questions TO studio_app;
