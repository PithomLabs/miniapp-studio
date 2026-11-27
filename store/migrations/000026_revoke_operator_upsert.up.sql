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
--
-- AMENDMENT (Card 001): run the ACL statements AS studio_lookup.
-- Function-ACL writes by the migrator-as-member take no effect (and,
-- once 000027 writes real entries, 42501 against them — proven live
-- on the down-leg). This text runs on future fresh-DB applications
-- (same end state: absent target warns either way) and on replay-UP
-- legs (where the REVOKE must remove a real entry) — strictly safer
-- in every direction. The runner verifies no checksums (dirty-flag
-- only), so amending changes future runs, not recorded history.
SET ROLE studio_lookup;
REVOKE EXECUTE ON FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) FROM studio_app;

DO $$ BEGIN
  IF NOT has_function_privilege(session_user, 'studio.operator_upsert(TEXT, TEXT, BOOLEAN)', 'EXECUTE') THEN
    EXECUTE format('GRANT EXECUTE ON FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) TO %I', session_user);
  END IF;
END $$;
RESET ROLE;
