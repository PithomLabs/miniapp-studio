-- AMENDMENT (prompt36 re-review): DROP FUNCTION executes AS
-- studio_lookup. Dropping a lookup-owned function as the
-- migrator-as-member 42501s (proven live on the 000031 up-leg,
-- same ownership rule). Per the 000026 Card-001 precedent this
-- amendment is safe: down-files re-run on every down-migration
-- and this down never ran on any persistent database (downs run
-- only on disposable scratch DBs), and the runner verifies
-- dirty-flag only, never checksums — so it changes future
-- down-runs, not applied history. Table drop stays with the
-- migrator (table owner).
SET ROLE studio_lookup;
DROP FUNCTION IF EXISTS studio.invitation_accept(TEXT);
RESET ROLE;
DROP TABLE IF EXISTS studio.invitations;
