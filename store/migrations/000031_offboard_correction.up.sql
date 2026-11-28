-- 000031: F35-01 invitation/offboarding correction + F35-14 DELETE
-- tightening on the increment-3 tables.
--
-- The 000029 accept function unconditionally reactivated operators,
-- so a stale invite defeated offboarding. Corrected behavior:
--   1. Accept refuses when the intended binding exists inactive.
--   2. Accept never flips active=false to true (ON CONFLICT DO
--      NOTHING; an existing row here passed the active gate).
--   3. Offboarding revokes outstanding invites (revoke-FIRST, then
--      deactivate — see lock note below).
--   4. A reactivation trigger revokes outstanding invites on every
--      false->true path, function or raw SQL, so no pre-offboard
--      invite can revive after reactivation on any path.
-- Lock note: no explicit row locks are taken anywhere in these
-- paths. Accept's consume UPDATE re-checks all guards atomically;
-- the single-transaction Go Accept rolls back on any failure; and
-- session_create's operator-active guard backstops the raw-SQL
-- offboard race (rollback leaves the invite consumable). An
-- explicit SELECT FOR UPDATE design was rejected: when the
-- operator row is absent it acquires no lock, which admits a
-- deadlock cycle against offboarding's operator-first order.
--
-- Ownership note (live 42501, fixed before first success): function
-- DDL/ACL statements execute ONLY as studio_lookup (000027
-- precedent — the migrator-as-member cannot replace lookup-owned
-- functions). Table ACLs and CREATE TRIGGER stay with the migrator
-- (table owner; studio_lookup holds no TRIGGER privilege).
SET ROLE studio_lookup;
CREATE OR REPLACE FUNCTION studio.invitation_accept(p_token_hash TEXT)
RETURNS TABLE (o_tenant_id TEXT, o_binding_id TEXT)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = studio
AS $$
DECLARE
  v_tenant_id TEXT;
  v_binding_id TEXT;
  v_active BOOLEAN;
BEGIN
  SELECT i.tenant_id, i.binding_id INTO v_tenant_id, v_binding_id
  FROM studio.invitations AS i
  WHERE i.token_hash = p_token_hash
    AND i.consumed_at IS NULL
    AND i.revoked_at IS NULL
    AND i.expires_at > now();
  IF NOT FOUND THEN
    RETURN;
  END IF;
  SELECT o.active INTO v_active
  FROM studio.operators AS o
  WHERE o.tenant_id = v_tenant_id AND o.binding_id = v_binding_id;
  IF FOUND AND NOT v_active THEN
    RETURN;
  END IF;
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
  ON CONFLICT (tenant_id, binding_id) DO NOTHING;
  o_tenant_id := v_tenant_id;
  o_binding_id := v_binding_id;
  RETURN NEXT;
END;
$$;
ALTER FUNCTION studio.invitation_accept(TEXT) OWNER TO studio_lookup;
REVOKE ALL ON FUNCTION studio.invitation_accept(TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION studio.invitation_accept(TEXT) TO studio_app;

-- Owner-only offboarding: revoke outstanding invites FIRST, then
-- deactivate, in one call. EXECUTE goes to the migration user (the
-- owner identity), never to the application role — same pattern as
-- operator_upsert: the app must not offboard operators by itself.
-- SET ROLE does not change session_user, so the dynamic grant below
-- still resolves the migrating owner, not studio_lookup.
CREATE OR REPLACE FUNCTION studio.operator_offboard(p_tenant_id TEXT, p_binding_id TEXT)
RETURNS VOID
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = studio
AS $$
BEGIN
  UPDATE studio.invitations AS i
  SET revoked_at = now()
  WHERE i.tenant_id = p_tenant_id AND i.binding_id = p_binding_id
    AND i.revoked_at IS NULL AND i.consumed_at IS NULL;
  INSERT INTO studio.operators AS o (tenant_id, binding_id, active)
  VALUES (p_tenant_id, p_binding_id, false)
  ON CONFLICT (tenant_id, binding_id) DO UPDATE SET active = false;
END;
$$;
ALTER FUNCTION studio.operator_offboard(TEXT, TEXT) OWNER TO studio_lookup;
REVOKE ALL ON FUNCTION studio.operator_offboard(TEXT, TEXT) FROM PUBLIC;
DO $$ BEGIN
  EXECUTE format('GRANT EXECUTE ON FUNCTION studio.operator_offboard(TEXT, TEXT) TO %I', session_user);
END $$;

-- Reactivation safeguard: any false->true transition revokes the
-- binding's outstanding invites, on every path (functions and raw
-- SQL alike — triggers fire regardless of caller). Runs as
-- studio_lookup under the existing invitation UPDATE policy.
CREATE OR REPLACE FUNCTION studio.invitation_reactivation_revoke()
RETURNS TRIGGER
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = studio
AS $$
BEGIN
  IF OLD.active = false AND NEW.active = true THEN
    UPDATE studio.invitations AS i
    SET revoked_at = now()
    WHERE i.tenant_id = NEW.tenant_id AND i.binding_id = NEW.binding_id
      AND i.revoked_at IS NULL AND i.consumed_at IS NULL;
  END IF;
  RETURN NEW;
END;
$$;
ALTER FUNCTION studio.invitation_reactivation_revoke() OWNER TO studio_lookup;
RESET ROLE;

-- Table-owner operations (migrator): trigger creation needs the
-- TRIGGER privilege studio_lookup lacks, and table ACLs live on
-- migrator-owned objects.
DROP TRIGGER IF EXISTS operators_reactivation_revoke ON studio.operators;
CREATE TRIGGER operators_reactivation_revoke
AFTER UPDATE OF active ON studio.operators
FOR EACH ROW EXECUTE FUNCTION studio.invitation_reactivation_revoke();

-- The offboard path sets revoked_at as studio_lookup: extend the
-- column grant (consume path keeps using consumed_at only).
GRANT UPDATE (revoked_at) ON studio.invitations TO studio_lookup;

-- F35-14: no code path deletes from the increment-3 tables.
REVOKE DELETE ON studio.invitations FROM studio_app;
REVOKE DELETE ON studio.capability_snapshots FROM studio_app;
