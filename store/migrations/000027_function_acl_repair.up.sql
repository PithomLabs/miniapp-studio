-- 000027: repair function ACLs under the owner role.
--
-- Root cause (Card 001, proven live 2026-11-27): ACL statements on
-- studio_lookup-owned functions have no effect when executed by the
-- migrating owner as a mere member (INHERIT FALSE). 000002/000009's
-- REVOKE FROM PUBLIC left explicit PUBLIC EXECUTE on all four
-- session functions (raw proacl {=X/studio_lookup,...} on live Neon),
-- and every GRANT — to studio_app, to the migrating user, and via the
-- test-role mirror — vanished. Clean v26 throughout, zero errors
-- surfaced: the statements are permitted (ownership checks pass via
-- membership) but ineffective. Table grants were unaffected
-- (migrator-owned objects). Only the 000001 SET capability was ever
-- real, and no migration used it — until this file.
--
-- Invariant from here on: function-ACL statements execute ONLY as
-- the function owner. This file SETs ROLE studio_lookup, applies the
-- complete intended function-ACL set, asserts its own postconditions
-- (a future silent-ineffectiveness regression fails THIS migration
-- loudly instead of shipping another hole), and RESETs.
SET ROLE studio_lookup;
REVOKE ALL ON FUNCTION studio.session_lookup(TEXT) FROM PUBLIC;
REVOKE ALL ON FUNCTION studio.session_create(TEXT, TEXT, TEXT, TEXT, TIMESTAMPTZ) FROM PUBLIC;
REVOKE ALL ON FUNCTION studio.session_revoke(TEXT) FROM PUBLIC;
REVOKE ALL ON FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION studio.session_lookup(TEXT) TO studio_app;
GRANT EXECUTE ON FUNCTION studio.session_create(TEXT, TEXT, TEXT, TEXT, TIMESTAMPTZ) TO studio_app;
GRANT EXECUTE ON FUNCTION studio.session_revoke(TEXT) TO studio_app;
GRANT EXECUTE ON FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) TO SESSION_USER;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc p
             WHERE p.pronamespace = 'studio'::regnamespace
             AND has_function_privilege('public', p.oid, 'EXECUTE')) THEN
    RAISE EXCEPTION 'postcondition failed: PUBLIC EXECUTE remains on a studio function';
  END IF;
  IF NOT has_function_privilege('studio_app', 'studio.session_lookup(text)', 'EXECUTE') THEN
    RAISE EXCEPTION 'postcondition failed: studio_app lacks session_lookup EXECUTE';
  END IF;
  IF NOT has_function_privilege('studio_app', 'studio.session_create(text,text,text,text,timestamptz)', 'EXECUTE') THEN
    RAISE EXCEPTION 'postcondition failed: studio_app lacks session_create EXECUTE';
  END IF;
  IF NOT has_function_privilege('studio_app', 'studio.session_revoke(text)', 'EXECUTE') THEN
    RAISE EXCEPTION 'postcondition failed: studio_app lacks session_revoke EXECUTE';
  END IF;
  IF has_function_privilege('studio_app', 'studio.operator_upsert(text,text,boolean)', 'EXECUTE') THEN
    RAISE EXCEPTION 'postcondition failed: studio_app holds operator_upsert EXECUTE';
  END IF;
END $$;
RESET ROLE;
