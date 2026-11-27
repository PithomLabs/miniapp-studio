-- B2-01B: durable sprite lifecycle receipts. One row per completed
-- lifecycle transition, keyed by the client-supplied idempotency key:
-- the composite primary key is the cross-restart replay guard.
-- Build-less by design (infra operations carry no build); current
-- position per sprite derives as the latest to_state. Transition and
-- state vocabularies mirror the fly lifecycle contract. Receipts carry
-- identity and scope only — never credentials or transcripts.
CREATE TABLE studio.sprite_lifecycle_receipts (
  id          TEXT NOT NULL,
  tenant_id   TEXT NOT NULL,
  sprite      TEXT NOT NULL,
  transition  TEXT NOT NULL
    CHECK (transition IN ('provision', 'wake', 'sleep', 'connect', 'disconnect', 'destroy')),
  from_state  TEXT NOT NULL
    CHECK (from_state IN ('absent', 'asleep', 'awake', 'connected')),
  to_state    TEXT NOT NULL
    CHECK (to_state IN ('absent', 'asleep', 'awake', 'connected')),
  at          TIMESTAMPTZ NOT NULL,
  approver    TEXT NOT NULL DEFAULT '',
  auth_ref    TEXT NOT NULL DEFAULT '',
  provider    TEXT NOT NULL DEFAULT '',
  latency_ms  BIGINT NOT NULL DEFAULT 0 CHECK (latency_ms >= 0),
  PRIMARY KEY (tenant_id, id)
);
CREATE INDEX sprite_lifecycle_latest ON studio.sprite_lifecycle_receipts (tenant_id, sprite, at DESC);
CREATE POLICY tenant_isolation ON studio.sprite_lifecycle_receipts
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
ALTER TABLE studio.sprite_lifecycle_receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.sprite_lifecycle_receipts FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON studio.sprite_lifecycle_receipts TO studio_app;
