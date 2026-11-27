-- 000029: tenant invitations. One row per invitation, keyed by
-- (tenant_id, id); the raw token is never stored — only its sha256
-- hash (see store.HashToken), looked up globally by the accept
-- function. Creation/revocation/listing run as the app role under
-- tenant RLS; acceptance is an unauthenticated token capability, so
-- it flows through the fixed-body SECURITY DEFINER function below
-- (session_create precedent): guarded consume + operator upsert in
-- one atomic UPDATE, exactly-once under concurrency via the row
-- lock (second acceptor finds consumed_at set and gets no row).
CREATE TABLE studio.invitations (
  tenant_id  TEXT NOT NULL,
  id         TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  binding_id TEXT NOT NULL CHECK (binding_id <> ''),
  invited_by TEXT NOT NULL CHECK (invited_by <> ''),
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (expires_at > created_at),
  PRIMARY KEY (tenant_id, id)
);
CREATE INDEX invitations_token_hash ON studio.invitations (token_hash);
CREATE POLICY tenant_isolation ON studio.invitations
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
-- Lookup-role access for the accept function only: find by hash,
-- mark consumed. No broader invitation access for studio_lookup.
CREATE POLICY invitation_accept_lookup ON studio.invitations
  FOR SELECT TO studio_lookup USING (true);
CREATE POLICY invitation_accept_consume ON studio.invitations
  FOR UPDATE TO studio_lookup USING (true) WITH CHECK (true);
ALTER TABLE studio.invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.invitations FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON studio.invitations TO studio_app;
GRANT SELECT ON studio.invitations TO studio_lookup;
GRANT UPDATE (consumed_at) ON studio.invitations TO studio_lookup;

CREATE OR REPLACE FUNCTION studio.invitation_accept(p_token_hash TEXT)
RETURNS TABLE (o_tenant_id TEXT, o_binding_id TEXT)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = studio
AS $$
DECLARE
  v_tenant_id TEXT;
  v_binding_id TEXT;
BEGIN
  UPDATE studio.invitations AS i
  SET consumed_at = now()
  WHERE i.token_hash = p_token_hash
    AND i.consumed_at IS NULL
    AND i.revoked_at IS NULL
    AND i.expires_at > now()
  RETURNING i.tenant_id, i.binding_id INTO v_tenant_id, v_binding_id;
  IF NOT FOUND THEN
    RETURN;
  END IF;
  INSERT INTO studio.operators AS o (tenant_id, binding_id, active)
  VALUES (v_tenant_id, v_binding_id, true)
  ON CONFLICT (tenant_id, binding_id) DO UPDATE SET active = true;
  o_tenant_id := v_tenant_id;
  o_binding_id := v_binding_id;
  RETURN NEXT;
END;
$$;
ALTER FUNCTION studio.invitation_accept(TEXT) OWNER TO studio_lookup;
REVOKE ALL ON FUNCTION studio.invitation_accept(TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION studio.invitation_accept(TEXT) TO studio_app;
