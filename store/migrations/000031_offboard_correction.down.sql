-- Down of 000031: restore the 000029 accept body exactly, drop the
-- offboard path and trigger, restore DELETE grants. Function DDL
-- runs as studio_lookup (000027 precedent); trigger and table ACLs
-- run as the migrator (table owner).
DROP TRIGGER IF EXISTS operators_reactivation_revoke ON studio.operators;

SET ROLE studio_lookup;
DROP FUNCTION IF EXISTS studio.invitation_reactivation_revoke();
DROP FUNCTION IF EXISTS studio.operator_offboard(TEXT, TEXT);

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
RESET ROLE;

REVOKE UPDATE (revoked_at) ON studio.invitations FROM studio_lookup;
GRANT DELETE ON studio.invitations TO studio_app;
GRANT DELETE ON studio.capability_snapshots TO studio_app;
