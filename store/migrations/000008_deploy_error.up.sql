-- 000008: failure evidence on deployments. Provisioning and reconciliation
-- failures record a short static reason alongside the failed status so the
-- Validate & Deploy screen can explain without re-probing Fly. Reasons are
-- static strings chosen in Go code, never raw provider or database errors.

ALTER TABLE studio.deployments ADD COLUMN error TEXT NOT NULL DEFAULT '';
