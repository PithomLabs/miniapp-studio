-- B0: distinguish manual-local builds from gateway-driven ones.
-- local_desktop builds run through the supervised manual bridge
-- (docs/local-agent-bridge.md), not the sender loop; remote_sprite
-- arrives with the B2 handoff. No grant/RLS change: column-only.
ALTER TABLE studio.builds
  ADD COLUMN execution_location TEXT NOT NULL DEFAULT 'local_desktop'
  CHECK (execution_location IN ('local_desktop', 'remote_sprite'));
