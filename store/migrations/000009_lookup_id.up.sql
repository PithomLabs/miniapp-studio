-- 000009: return the session id from session_lookup (F9). Callers need a
-- stable handle for logout auditing and whoami responses; previously only
-- tenant/binding/expiry came back. Return-type changes require drop and
-- recreate; ownership, grants, and revocation are re-applied identically.

DROP FUNCTION IF EXISTS studio.session_lookup(TEXT);

CREATE FUNCTION studio.session_lookup(p_token_hash TEXT)
RETURNS TABLE (id TEXT, tenant_id TEXT, binding_id TEXT, expires_at TIMESTAMPTZ)
LANGUAGE sql STABLE SECURITY DEFINER
SET search_path = studio
AS $$
  SELECT s.id, s.tenant_id, s.binding_id, s.expires_at
  FROM studio.sessions AS s
  WHERE s.token_hash = p_token_hash
    AND s.revoked_at IS NULL
    AND s.expires_at > now()
$$;
ALTER FUNCTION studio.session_lookup(TEXT) OWNER TO studio_lookup;
REVOKE ALL ON FUNCTION studio.session_lookup(TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION studio.session_lookup(TEXT) TO studio_app;
