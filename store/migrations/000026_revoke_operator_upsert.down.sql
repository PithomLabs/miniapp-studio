-- 000026 down: restore pre-migration grant state exactly (down/up
-- cycling must round-trip). Re-grants app EXECUTE (the insecure state
-- under test by design) and removes the owner grant.
GRANT EXECUTE ON FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) TO studio_app;

DO $$ BEGIN
  EXECUTE format('REVOKE EXECUTE ON FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) FROM %I', session_user);
END $$;
