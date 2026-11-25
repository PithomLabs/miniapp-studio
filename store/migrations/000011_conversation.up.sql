-- 000011: conversation and message tables (Phase A, gateway plan §7).
-- Conversations are the permanent business record; messages hold user and
-- assistant text. Ordering for SSE replay lives in conversation_stream
-- (migration 000014, decision D1), so messages carry no sequence of their
-- own: one cursor, one sequence space, no cross-table coordination.

CREATE TABLE studio.conversations (
  id             TEXT NOT NULL PRIMARY KEY,
  tenant_id      TEXT NOT NULL,
  application_id TEXT NOT NULL DEFAULT '',
  title          TEXT NOT NULL DEFAULT '',
  status         TEXT NOT NULL DEFAULT 'open'
    CHECK (status IN ('open', 'closed')),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id)
);
CREATE INDEX conversations_app ON studio.conversations (tenant_id, application_id, updated_at);

CREATE TABLE studio.messages (
  id              TEXT NOT NULL PRIMARY KEY,
  tenant_id       TEXT NOT NULL,
  conversation_id TEXT NOT NULL,
  role            TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
  body            TEXT NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  FOREIGN KEY (tenant_id, conversation_id)
    REFERENCES studio.conversations (tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX messages_conversation ON studio.messages (tenant_id, conversation_id, created_at);

CREATE POLICY tenant_isolation ON studio.conversations
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation ON studio.messages
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
ALTER TABLE studio.conversations FORCE ROW LEVEL SECURITY;
ALTER TABLE studio.messages FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON studio.conversations, studio.messages TO studio_app;
