-- 000003: schema USAGE for runtime roles. Table and function privileges are
-- useless without USAGE on the containing schema; 000002 omitted it, which
-- wrongly denied all studio-schema access (including the narrow session
-- functions) with 42501. This fix-forward migration restores exactly the
-- intended surface: name resolution only, no data access by itself.

GRANT USAGE ON SCHEMA studio TO studio_app, studio_lookup;
