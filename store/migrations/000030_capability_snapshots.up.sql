-- 000030: capability snapshots. One row per tenant holding the last
-- fully successful refresh: only complete collections publish, so a
-- partial or failed run leaves the previous snapshot (with its
-- recorded refreshed_at, from which readers compute staleness) rather
-- than a misleading complete view. Snapshots are reports only — they
-- grant no permission and gate no execution path.
CREATE TABLE studio.capability_snapshots (
  tenant_id    TEXT NOT NULL PRIMARY KEY,
  snapshot     JSONB NOT NULL,
  provenance   JSONB NOT NULL,
  refreshed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE POLICY tenant_isolation ON studio.capability_snapshots
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
ALTER TABLE studio.capability_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE studio.capability_snapshots FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON studio.capability_snapshots TO studio_app;
