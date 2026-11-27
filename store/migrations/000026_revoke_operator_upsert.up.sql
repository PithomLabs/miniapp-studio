-- 000026: remove application EXECUTE on operator_upsert (privilege fix).
--
-- studio.operator_upsert is SECURITY DEFINER (owner studio_lookup) and
-- upserts operator rows INCLUDING the admin flag. The 000002 grant of
-- EXECUTE to studio_app therefore lets any app session mint itself an
-- admin operator — a live privilege-escalation hole. Both privilege
-- tests (RoleAttributes, PrivilegeParity) forbid this grant, and no app
-- code calls the function (sole caller is the owner-side UpsertOperator
-- provisioner). The tests encode the secure design; this migration
-- makes reality match.
--
-- The migrating owner receives EXECUTE instead (dynamic: owner name
-- varies per project), keeping the owner-side provisioner working.
-- Forward migration (not a 000002 edit): 000002 has applied on live
-- branches, so history is immutable from here on.
REVOKE EXECUTE ON FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) FROM studio_app;

DO $$ BEGIN
  IF NOT has_function_privilege(session_user, 'studio.operator_upsert(TEXT, TEXT, BOOLEAN)', 'EXECUTE') THEN
    EXECUTE format('GRANT EXECUTE ON FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) TO %I', session_user);
  END IF;
END $$;
