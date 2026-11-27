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

-- Canonical SET-capability repair (SQLSTATE 42501 root-cause fix).
-- PostgreSQL 16+ grants a CREATEROLE creator an automatic membership in
-- each role it creates with ADMIN TRUE, SET FALSE, INHERIT FALSE — so a
-- membership row EXISTS while SET ROLE stays prohibited. The earlier
-- existence-gated GRANT skipped itself into exactly this trap (proven
-- live: MEMBER true, SET false, ADMIN true). Membership existence is
-- NOT the invariant; effective SET capability is.
--
-- The explicit grant below establishes SET TRUE without inheritance and
-- without admin on the new membership. All three options are explicit
-- because omitted options retain existing values on modification. The
-- bootstrap-superuser auto-membership is left untouched; this
-- additional membership is the documented mechanism. The migrating
-- owner name varies per project, hence SESSION_USER (keyword, valid
-- grantee). studio_lookup itself gains nothing; runtime roles gain
-- nothing; the no-memberships-for-runtime-roles invariant is untouched.
--
-- Applied-migration rule: editing this file does not re-run it where
-- already applied. Verification recreates disposable run databases
-- (migrations run only on fresh run DBs in this program). A persistent
-- database with 000001 already recorded would need a separately
-- controlled repair — none exists; do not assume one.
-- NOTE (T1 live finding): edited pre-first-live-application. The chain
-- had applied 0 migrations on Neon when membership handling was added;
-- loopback applications are superseded-history.
GRANT studio_lookup TO SESSION_USER
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;

-- Postcondition: abort loudly unless the migration principal can
-- actually SET ROLE. This check, not row existence, is the invariant.
DO $$ BEGIN
  IF NOT pg_has_role(session_user, 'studio_lookup', 'SET') THEN
    RAISE EXCEPTION 'postcondition failed: migration principal cannot SET ROLE studio_lookup';
  END IF;
END $$;
