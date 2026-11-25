-- Down reverses 000002. Destroys Studio data. The application role is
-- operator-owned and is never dropped here.

DROP FUNCTION IF EXISTS studio.operator_upsert(TEXT, TEXT, BOOLEAN);
DROP FUNCTION IF EXISTS studio.session_revoke(TEXT);
DROP FUNCTION IF EXISTS studio.session_create(TEXT, TEXT, TEXT, TEXT, TIMESTAMPTZ);
DROP FUNCTION IF EXISTS studio.session_lookup(TEXT);
DROP TABLE IF EXISTS studio.runs;
DROP TABLE IF EXISTS studio.deployments;
DROP TABLE IF EXISTS studio.app_specs;
DROP TABLE IF EXISTS studio.apps;
DROP TABLE IF EXISTS studio.sessions;
DROP TABLE IF EXISTS studio.operators;
DROP SCHEMA IF EXISTS studio;
