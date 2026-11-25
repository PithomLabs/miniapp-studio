-- Down reverses 000007. Reverting re-opens the cross-tenant reference
-- hole by design of the older schema; use only to roll back a failed
-- deploy, then re-apply forward immediately.

ALTER TABLE studio.runs
  DROP CONSTRAINT runs_tenant_app_fkey,
  ADD CONSTRAINT runs_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES studio.apps (id) ON DELETE CASCADE;

ALTER TABLE studio.deployments
  DROP CONSTRAINT deployments_tenant_app_fkey,
  ADD CONSTRAINT deployments_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES studio.apps (id) ON DELETE CASCADE;

ALTER TABLE studio.app_specs
  DROP CONSTRAINT app_specs_tenant_app_fkey,
  ADD CONSTRAINT app_specs_app_id_fkey
    FOREIGN KEY (app_id) REFERENCES studio.apps (id) ON DELETE CASCADE;

ALTER TABLE studio.apps DROP CONSTRAINT apps_tenant_id_unique;
