-- 000005: USAGE ON SCHEMA app for the application role (F2). The table
-- grants in 000002 were ineffective without schema USAGE, so every Service
-- invocation and framework-owned persistence operation through the
-- application pool failed closed with 42501. This grants name resolution
-- only; table access stays exactly the 000002 grant set, and framework RLS
-- policies continue to scope every row.

GRANT USAGE ON SCHEMA app TO studio_app;
