-- RPA F0 broker (option A: server-side mint). Refresh tokens rest here
-- encrypted under a deployment-held KEK (AES-256-GCM, AAD binds
-- tenant+provider); plaintext refresh NEVER leaves broker memory.
-- Leases record issue + use WITHOUT token material: minted access
-- tokens exist only in memory between Mint and the provider call.
-- Access rows name identities only — no secret bytes, ever.
CREATE TABLE studio.broker_credentials (
  tenant_id  TEXT NOT NULL,
  provider   TEXT NOT NULL CHECK (provider <> ''),
  refresh_ct BYTEA NOT NULL,
  kek_id     TEXT NOT NULL CHECK (kek_id <> ''),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  revoked_at TIMESTAMPTZ,
  PRIMARY KEY (tenant_id, provider)
);
CREATE POLICY tenant_isolation ON studio.broker_credentials
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
ALTER TABLE studio.broker_credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.broker_credentials FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON studio.broker_credentials TO studio_app;

CREATE TABLE studio.broker_leases (
  id          TEXT NOT NULL,
  tenant_id   TEXT NOT NULL,
  provider    TEXT NOT NULL CHECK (provider <> ''),
  capability  TEXT NOT NULL CHECK (capability <> ''),
  intent_hash TEXT NOT NULL CHECK (intent_hash <> ''),
  single_use  BOOLEAN NOT NULL,
  issued_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at  TIMESTAMPTZ NOT NULL,
  used_at     TIMESTAMPTZ,
  CHECK (expires_at > issued_at),
  PRIMARY KEY (tenant_id, id)
);
CREATE INDEX broker_leases_expiry ON studio.broker_leases (tenant_id, expires_at);
CREATE POLICY tenant_isolation ON studio.broker_leases
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
ALTER TABLE studio.broker_leases ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.broker_leases FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON studio.broker_leases TO studio_app;
