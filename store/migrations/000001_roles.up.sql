-- 000001: Studio roles. The session-lookup role owns the SECURITY DEFINER
-- session functions (created in 000002) and nothing else. It cannot log in
-- and cannot bypass RLS. The application role is provisioned out of band by
-- the operator (see README deploy section) BEFORE later migrations run:
--
--   CREATE ROLE studio_app WITH LOGIN PASSWORD '<owner-generated>' NOBYPASSRLS;
--
-- Migration 000002 grants it table and function privileges.

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'studio_lookup') THEN
    CREATE ROLE studio_lookup WITH NOLOGIN NOBYPASSRLS;
  END IF;
END $$;
