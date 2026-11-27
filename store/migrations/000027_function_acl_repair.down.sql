-- 000027 down: remove the restored function grants. PUBLIC stays
-- revoked: a down migration must never reopen a security hole.
-- SET ROLE is required here for the same reason as the up leg —
-- function-ACL statements execute only as the function owner.
SET ROLE studio_lookup;
REVOKE EXECUTE ON FUNCTION studio.session_lookup(TEXT) FROM studio_app;
REVOKE EXECUTE ON FUNCTION studio.session_create(TEXT, TEXT, TEXT, TEXT, TIMESTAMPTZ) FROM studio_app;
REVOKE EXECUTE ON FUNCTION studio.session_revoke(TEXT) FROM studio_app;
REVOKE EXECUTE ON FUNCTION studio.operator_upsert(TEXT, TEXT, BOOLEAN) FROM SESSION_USER;
RESET ROLE;
