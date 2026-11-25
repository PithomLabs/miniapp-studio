-- 000002: Studio schema, tenant RLS, session functions, and grants.
--
-- Prerequisites (checked by the migrate command, not here):
--   1. Framework migrations applied (app.* tables at version >= 6).
--   2. Application role exists: CREATE ROLE studio_app WITH LOGIN NOBYPASSRLS.
--
-- Threat model: the application role reaches tenant rows only; it has no
-- privileges at all on studio.sessions and studio.operators. Session access
-- flows exclusively through the three SECURITY DEFINER functions below,
-- owned by the non-login studio_lookup role. FORCE ROW LEVEL SECURITY keeps
-- even table owners inside the policy boundary.

CREATE SCHEMA IF NOT EXISTS studio;

-- Pre-provisioned operator identities: exactly the (tenant, binding) pairs
-- allowed to hold sessions. Rows are written by the owner-only provision
-- command, never by the application role.
CREATE TABLE IF NOT EXISTS studio.operators (
  tenant_id  TEXT NOT NULL,
  binding_id TEXT NOT NULL,
  active     BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, binding_id)
);

CREATE TABLE IF NOT EXISTS studio.sessions (
  id          TEXT NOT NULL PRIMARY KEY,
  tenant_id   TEXT NOT NULL,
  binding_id  TEXT NOT NULL,
  token_hash  TEXT NOT NULL UNIQUE,
  expires_at  TIMESTAMPTZ NOT NULL,
  revoked_at  TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS sessions_token_hash ON studio.sessions (token_hash);
CREATE INDEX IF NOT EXISTS sessions_tenant ON studio.sessions (tenant_id);

CREATE TABLE IF NOT EXISTS studio.apps (
  id             TEXT NOT NULL PRIMARY KEY,
  tenant_id      TEXT NOT NULL,
  slug           TEXT NOT NULL,
  name           TEXT NOT NULL,
  description    TEXT NOT NULL DEFAULT '',
  owner_identity TEXT NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, slug)
);
CREATE INDEX IF NOT EXISTS apps_tenant ON studio.apps (tenant_id);

CREATE TABLE IF NOT EXISTS studio.app_specs (
  id         TEXT NOT NULL PRIMARY KEY,
  tenant_id  TEXT NOT NULL,
  app_id     TEXT NOT NULL REFERENCES studio.apps (id) ON DELETE CASCADE,
  version    INTEGER NOT NULL,
  spec_hash  TEXT NOT NULL,
  document   JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, app_id, version)
);
CREATE INDEX IF NOT EXISTS app_specs_app ON studio.app_specs (tenant_id, app_id);

CREATE TABLE IF NOT EXISTS studio.deployments (
  id            TEXT NOT NULL PRIMARY KEY,
  tenant_id     TEXT NOT NULL,
  app_id        TEXT NOT NULL REFERENCES studio.apps (id) ON DELETE CASCADE,
  slug          TEXT NOT NULL,
  resource_kind TEXT NOT NULL,
  resource_name TEXT NOT NULL,
  resource_id   TEXT NOT NULL DEFAULT '',
  url           TEXT NOT NULL DEFAULT '',
  status        TEXT NOT NULL,
  spec_hash     TEXT NOT NULL DEFAULT '',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, app_id)
);
CREATE INDEX IF NOT EXISTS deployments_slug ON studio.deployments (tenant_id, slug);

CREATE TABLE IF NOT EXISTS studio.runs (
  id          TEXT NOT NULL PRIMARY KEY,
  tenant_id   TEXT NOT NULL,
  app_id      TEXT NOT NULL REFERENCES studio.apps (id) ON DELETE CASCADE,
  spec_hash   TEXT NOT NULL,
  workflow_id TEXT NOT NULL,
  status      TEXT NOT NULL,
  result      JSONB,
  error       TEXT NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS runs_app ON studio.runs (tenant_id, app_id, created_at);

-- Tenant isolation on application tables. The USING/WITH CHECK expression
-- reads the transaction-local setting the server arms per request.
CREATE POLICY tenant_isolation ON studio.apps
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation ON studio.app_specs
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation ON studio.deployments
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation ON studio.runs
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));

-- The lookup role's own narrow policies: exactly the rows its functions
-- touch. No policy grants the application role any session/operator access.
CREATE POLICY session_lookup_only ON studio.sessions
  FOR SELECT TO studio_lookup USING (true);
CREATE POLICY session_insert_only ON studio.sessions
  FOR INSERT TO studio_lookup WITH CHECK (true);
CREATE POLICY session_revoke_only ON studio.sessions
  FOR UPDATE TO studio_lookup USING (true) WITH CHECK (true);
CREATE POLICY operator_lookup_only ON studio.operators
  FOR SELECT TO studio_lookup USING (true);
-- Lookup-role management policy for the provisioning function below.
CREATE POLICY operator_manage ON studio.operators
  FOR ALL TO studio_lookup USING (true) WITH CHECK (true);

ALTER TABLE studio.operators FORCE ROW LEVEL SECURITY;
ALTER TABLE studio.sessions FORCE ROW LEVEL SECURITY;
ALTER TABLE studio.apps FORCE ROW LEVEL SECURITY;
ALTER TABLE studio.app_specs FORCE ROW LEVEL SECURITY;
ALTER TABLE studio.deployments FORCE ROW LEVEL SECURITY;
ALTER TABLE studio.runs FORCE ROW LEVEL SECURITY;

-- Privileges: the application role receives data-table DML only. It gets no
-- privileges on studio.sessions or studio.operators, and no table ownership.
GRANT SELECT, INSERT, UPDATE, DELETE ON
  studio.apps, studio.app_specs, studio.deployments, studio.runs
  TO studio_app;
GRANT SELECT, INSERT ON studio.sessions TO studio_lookup;
GRANT UPDATE (revoked_at) ON studio.sessions TO studio_lookup;
GRANT SELECT ON studio.operators TO studio_lookup;

-- Session functions. Fixed SQL, fully qualified names, hardened search_path.
-- Owned by the lookup role; PUBLIC execution revoked; app role executes.

CREATE OR REPLACE FUNCTION studio.session_lookup(p_token_hash TEXT)
RETURNS TABLE (tenant_id TEXT, binding_id TEXT, expires_at TIMESTAMPTZ)
LANGUAGE sql STABLE SECURITY DEFINER
SET search_path = studio
AS $$
  SELECT s.tenant_id, s.binding_id, s.expires_at
  FROM studio.sessions AS s
  WHERE s.token_hash = p_token_hash
    AND s.revoked_at IS NULL
    AND s.expires_at > now()
$$;
ALTER FUNCTION studio.session_lookup(TEXT) OWNER TO studio_lookup;
REVOKE ALL ON FUNCTION studio.session_lookup(TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION studio.session_lookup(TEXT) TO studio_app;

CREATE OR REPLACE FUNCTION studio.session_create(
  p_id TEXT, p_token_hash TEXT, p_tenant_id TEXT, p_binding_id TEXT, p_expires_at TIMESTAMPTZ)
RETURNS TEXT
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path = studio
AS $$
  INSERT INTO studio.sessions AS s (id, tenant_id, binding_id, token_hash, expires_at)
  SELECT p_id, p_tenant_id, p_binding_id, p_token_hash, p_expires_at
  WHERE EXISTS (
    SELECT 1 FROM studio.operators AS o
    WHERE o.tenant_id = p_tenant_id AND o.binding_id = p_binding_id AND o.active
  )
  AND p_expires_at > now()
  AND p_expires_at <= now() + INTERVAL '30 days'
  RETURNING s.id
$$;
ALTER FUNCTION studio.session_create(TEXT, TEXT, TEXT, TEXT, TIMESTAMPTZ) OWNER TO studio_lookup;
REVOKE ALL ON FUNCTION studio.session_create(TEXT, TEXT, TEXT, TEXT, TIMESTAMPTZ) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION studio.session_create(TEXT, TEXT, TEXT, TEXT, TIMESTAMPTZ) TO studio_app;

CREATE OR REPLACE FUNCTION studio.session_revoke(p_token_hash TEXT)
RETURNS BOOLEAN
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path = studio
AS $$
  UPDATE studio.sessions AS s SET revoked_at = now()
  WHERE s.token_hash = p_token_hash AND s.revoked_at IS NULL
  RETURNING true
$$;
ALTER FUNCTION studio.session_revoke(TEXT) OWNER TO studio_lookup;
REVOKE ALL ON FUNCTION studio.session_revoke(TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION studio.session_revoke(TEXT) TO studio_app;

-- Owner-only operator provisioning. EXECUTE goes to the migration user (the
-- owner/provisioning identity resolved at migrate time), never to the
-- application role: the app must not manufacture tenants for itself. If the
-- owner role ever rotates, re-grant EXECUTE to the new owner explicitly.
CREATE OR REPLACE FUNCTION studio.operator_upsert(p_tenant_id TEXT, p_binding_id TEXT, p_active BOOLEAN)
RETURNS VOID
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path = studio
AS $$
  INSERT INTO studio.operators AS o (tenant_id, binding_id, active)
  VALUES (p_tenant_id, p_binding_id, p_active)
  ON CONFLICT (tenant_id, binding_id) DO UPDATE SET active = p_active
$$;
ALTER FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) OWNER TO studio_lookup;
REVOKE ALL ON FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) FROM PUBLIC;
DO $$ BEGIN
  EXECUTE format('GRANT EXECUTE ON FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) TO %I', session_user);
END $$;

-- Framework tables the Studio runtime touches through runtime.Service.
GRANT SELECT, INSERT, UPDATE, DELETE ON
  app.invocations, app.approvals, app.audit_events, app.effect_records, app.kill_switches
  TO studio_app;
