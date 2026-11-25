-- 000007: composite tenant foreign keys (F3). Child rows referenced apps
-- by global id alone, so a tenant-A row could reference a tenant-B app and a
-- legitimate delete could cascade into another tenant's rows. This migration
-- refuses to run while any cross-tenant reference exists (failing with a
-- clear, identifier-free error and no partial state), then replaces the three
-- foreign keys with composite (tenant_id, app_id) references. RLS remains
-- the access layer; the composite FK is the integrity layer.

DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM studio.app_specs AS c JOIN studio.apps AS p ON p.id = c.app_id
    WHERE p.tenant_id IS DISTINCT FROM c.tenant_id
    UNION ALL
    SELECT 1 FROM studio.deployments AS c JOIN studio.apps AS p ON p.id = c.app_id
    WHERE p.tenant_id IS DISTINCT FROM c.tenant_id
    UNION ALL
    SELECT 1 FROM studio.runs AS c JOIN studio.apps AS p ON p.id = c.app_id
    WHERE p.tenant_id IS DISTINCT FROM c.tenant_id
  ) THEN
    RAISE EXCEPTION 'refusing migration: cross-tenant app references exist (resolve data first)';
  END IF;
END $$;

ALTER TABLE studio.apps ADD CONSTRAINT apps_tenant_id_unique UNIQUE (tenant_id, id);

ALTER TABLE studio.app_specs
  DROP CONSTRAINT app_specs_app_id_fkey,
  ADD CONSTRAINT app_specs_tenant_app_fkey
    FOREIGN KEY (tenant_id, app_id) REFERENCES studio.apps (tenant_id, id)
    ON DELETE CASCADE;

ALTER TABLE studio.deployments
  DROP CONSTRAINT deployments_app_id_fkey,
  ADD CONSTRAINT deployments_tenant_app_fkey
    FOREIGN KEY (tenant_id, app_id) REFERENCES studio.apps (tenant_id, id)
    ON DELETE CASCADE;

ALTER TABLE studio.runs
  DROP CONSTRAINT runs_app_id_fkey,
  ADD CONSTRAINT runs_tenant_app_fkey
    FOREIGN KEY (tenant_id, app_id) REFERENCES studio.apps (tenant_id, id)
    ON DELETE CASCADE;
