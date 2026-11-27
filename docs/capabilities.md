# Capability refresh (F0)

Refresh reports what the platform can actually do. It derives only
from authoritative sources — the registered Studio tool manifest
crossed against bound handlers, and the live connector list — and
grants nothing: execution keeps using the existing authorization,
approval, effect-ledger, intent-hash, and broker-lease paths. A
source-scan test (`TestRefreshGrantsNothing`) pins the absence of
call edges from refresh into those surfaces.

## Sources and states

- **Tools**: each manifest tool with a bound handler =
  `implemented`; declared-but-unbound = `failed` (drift data, not
  a publish blocker).
- **Connectors**: each live-list connection = `implemented`;
  unconfigured manager = snapshot-level `unavailable` marker;
  list failure = refresh error, previous snapshot retained.
- **Stale**: computed at read time from `refreshed_at`.
- **Planned**: unsupported by the existing contract (no
  declaration source exists) — never emitted, documented here.

## Storage and publish

`studio.capability_snapshots`, one row per tenant
`(snapshot, provenance, refreshed_at)`, replaced in a single
transaction only after a fully successful collection. Partial or
failed runs publish nothing. Concurrent refreshes are safe (each
publish is complete; last writer wins).

## Operator use

`task dev:capability-refresh TENANT=<t>` refreshes and prints the
summary. The verb carries no connector manager, so connector
state honestly reports unconfigured while tool registration
refreshes genuinely.
