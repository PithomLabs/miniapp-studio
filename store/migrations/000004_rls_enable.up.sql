-- 000004: enable RLS and complete lookup privileges. FORCE ROW LEVEL
-- SECURITY alone does not enable enforcement; without ENABLE the policies in
-- 000002 never filtered. This migration enables RLS on every Studio table
-- (FORCE stays) and grants the lookup role the operator-table writes its
-- provisioning function performs.

ALTER TABLE studio.operators ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.apps ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.app_specs ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.deployments ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.runs ENABLE ROW LEVEL SECURITY;

GRANT INSERT, UPDATE, DELETE ON studio.operators TO studio_lookup;
