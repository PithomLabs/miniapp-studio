-- 000026 down: restore pre-migration grant state exactly (down/up
-- cycling must round-trip). Re-grants app EXECUTE (the insecure state
-- under test by design) and removes the owner grant.
--
-- AMENDMENT (Card 001): down-files re-run on every down-migration and
-- this down never ran on any persistent database (downs run only on
-- disposable scratch DBs), so amending it changes future down-runs,
-- not applied history. SET ROLE is required for the same reason as
-- the up leg: the GRANT 42501s as a mere member once 000027's
-- explicit entries exist (proven live on the v27→v15 down-leg).
SET ROLE studio_lookup;
GRANT EXECUTE ON FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) TO studio_app;

DO $$ BEGIN
  EXECUTE format('REVOKE EXECUTE ON FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) FROM %I', session_user);
END $$;
RESET ROLE;
