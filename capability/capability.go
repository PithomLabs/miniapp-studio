// Package capability implements framework-native capability refresh:
// a report of what the platform can actually do, derived only from
// authoritative sources — the registered Studio tool manifest crossed
// against bound handlers, and the live connector list. The snapshot
// grants nothing: execution keeps using the existing authorization,
// approval, effect-ledger, intent-hash, and broker-lease paths, and a
// source-scan test pins that this package holds no call edges into
// them.
//
// Supported states (honest subset — "planned" has no declaration
// source in the existing contract and is never emitted):
//   - implemented: tool with a bound handler, or connection attested
//     by the live list.
//   - unavailable: connector manager unconfigured (snapshot-level
//     marker; no connection data exists to report).
//   - failed: manifest/implementation drift (declared tool, no bound
//     handler). Entry-level data, not a publish blocker.
//   - stale: computed at read time from refreshed_at (no writer).
//
// Collection errors (connector list failure) block publish: the
// previous snapshot stands rather than a partial view.
package capability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/fly"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/tools"
)

// States.
const (
	StateImplemented = "implemented"
	StateUnavailable = "unavailable"
	StateFailed      = "failed"
	StateStale       = "stale"
)

// ErrNoSnapshot reports a tenant with no published refresh yet.
var ErrNoSnapshot = errors.New("capability: no snapshot published")

// Entry is one refreshed capability in CapabilityRef shape
// (provider/operation/version). Connection entries carry no version
// — the connector contract has none — and say so with "".
type Entry struct {
	Kind      string `json:"kind"` // "tool" | "connection"
	Provider  string `json:"provider"`
	Operation string `json:"operation"`
	Version   string `json:"version"`
	State     string `json:"state"`
	Detail    string `json:"detail,omitempty"`
}

// Snapshot is the published per-tenant report plus read-time state.
type Snapshot struct {
	ToolsVersion         string  `json:"tools_version"`
	Build                string  `json:"build"`
	Entries              []Entry `json:"entries"`
	ConnectorsConfigured bool    `json:"connectors_configured"`
	RefreshedAt          time.Time
	Stale                bool
}

// SourceProvenance records one collection source for a publish.
type SourceProvenance struct {
	Source    string    `json:"source"`
	CheckedAt time.Time `json:"checked_at"`
	Count     int       `json:"count"`
	Note      string    `json:"note,omitempty"`
}

// ToolLister enumerates manifest tools and reports which have bound
// handlers. The default reads the live registry; tests inject
// doubles (including drifted ones).
type ToolLister func() (manifestVersion, build string, tools []DeclaredTool, err error)

// DeclaredTool is one manifest tool and its binding state.
type DeclaredTool struct {
	Name    string
	Bound   bool
	Version string
}

// DefaultToolLister crosses the Studio manifest against the bound
// implementation: every manifest tool must have a handler, and the
// cross-check itself is the drift detector.
func DefaultToolLister() (string, string, []DeclaredTool, error) {
	m := tools.Manifest()
	impl := tools.NewImplementation(tools.Deps{})
	out := make([]DeclaredTool, 0, len(m.Tools))
	for _, t := range m.Tools {
		_, bound := impl.ClassATools[string(t.Name)]
		out = append(out, DeclaredTool{Name: string(t.Name), Bound: bound, Version: string(m.Version)})
	}
	return string(m.Version), tools.Build, out, nil
}

// Service collects and publishes capability snapshots.
type Service struct {
	db    *pgxpool.Pool
	tools ToolLister
	conns fly.ConnectorManager // nil means unconfigured (honest marker, not error)
}

// New wires the service. A nil ConnectorManager is allowed and
// reports unavailable connector state; tools defaults to the live
// registry when nil.
func New(db *pgxpool.Pool, lister ToolLister, conns fly.ConnectorManager) (*Service, error) {
	if db == nil {
		return nil, fmt.Errorf("capability: pool is required")
	}
	if lister == nil {
		lister = DefaultToolLister
	}
	return &Service{db: db, tools: lister, conns: conns}, nil
}

// Refresh collects from every source and publishes the snapshot in
// one transaction — only on full success. A connector-list failure
// aborts before any write, leaving the previous snapshot standing.
func (s *Service) Refresh(ctx context.Context, tenant string, now time.Time) error {
	if tenant == "" {
		return fmt.Errorf("capability: refresh requires tenant")
	}
	version, build, declared, err := s.tools()
	if err != nil {
		return fmt.Errorf("capability: tool manifest: %w", err)
	}
	entries := []Entry{}
	for _, d := range declared {
		e := Entry{Kind: "tool", Provider: "studio", Operation: d.Name, Version: d.Version, State: StateImplemented}
		if !d.Bound {
			e.State, e.Detail = StateFailed, "declared in manifest, no handler bound"
		}
		entries = append(entries, e)
	}
	prov := []SourceProvenance{{Source: "tool-manifest", CheckedAt: now, Count: len(declared)}}
	configured := s.conns != nil
	if s.conns != nil {
		conns, err := s.conns.ListConnections(ctx, "")
		if err != nil {
			return fmt.Errorf("capability: connector list: %w", err)
		}
		for _, c := range conns {
			entries = append(entries, Entry{Kind: "connection", Provider: c.Provider, Operation: c.ConnectionType, State: StateImplemented})
		}
		prov = append(prov, SourceProvenance{Source: "connector-manager", CheckedAt: now, Count: len(conns)})
	} else {
		prov = append(prov, SourceProvenance{Source: "connector-manager", CheckedAt: now, Note: "manager unconfigured: connection state unavailable"})
	}
	snap := map[string]any{
		"tools_version": version, "build": build, "entries": entries,
		"connectors_configured": configured,
	}
	snapRaw, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("capability: encode snapshot: %w", err)
	}
	provRaw, err := json.Marshal(prov)
	if err != nil {
		return fmt.Errorf("capability: encode provenance: %w", err)
	}
	return store.InTenantTx(ctx, s.db, tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
INSERT INTO studio.capability_snapshots (tenant_id, snapshot, provenance, refreshed_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id) DO UPDATE
SET snapshot = EXCLUDED.snapshot, provenance = EXCLUDED.provenance, refreshed_at = EXCLUDED.refreshed_at`,
			tenant, snapRaw, provRaw, now)
		if err != nil {
			return fmt.Errorf("capability: publish: %w", err)
		}
		return nil
	})
}

// Read returns a tenant's snapshot, computing staleness from
// refreshed_at against staleAfter. Unknown tenants and tenants
// never refreshed get ErrNoSnapshot (no cross-tenant leakage: RLS
// confines the read and absence is uniform).
func (s *Service) Read(ctx context.Context, tenant string, staleAfter time.Duration, now time.Time) (Snapshot, []SourceProvenance, error) {
	if tenant == "" {
		return Snapshot{}, nil, fmt.Errorf("capability: read requires tenant")
	}
	var snap Snapshot
	var prov []SourceProvenance
	err := store.InTenantTx(ctx, s.db, tenant, func(tx pgx.Tx) error {
		var snapRaw, provRaw []byte
		var at time.Time
		if err := tx.QueryRow(ctx, `
SELECT snapshot, provenance, refreshed_at FROM studio.capability_snapshots WHERE tenant_id = $1`,
			tenant).Scan(&snapRaw, &provRaw, &at); err != nil {
			return err
		}
		if err := json.Unmarshal(snapRaw, &snap); err != nil {
			return fmt.Errorf("decode snapshot: %w", err)
		}
		if err := json.Unmarshal(provRaw, &prov); err != nil {
			return fmt.Errorf("decode provenance: %w", err)
		}
		snap.RefreshedAt = at
		return nil
	})
	if err != nil {
		// Absence (including RLS-confined foreign reads) is uniform;
		// corruption and transport failures stay distinguishable so a
		// broken snapshot never masquerades as a missing one.
		if errors.Is(err, pgx.ErrNoRows) {
			return Snapshot{}, nil, ErrNoSnapshot
		}
		return Snapshot{}, nil, fmt.Errorf("capability: read: %w", err)
	}
	snap.Stale = staleAfter > 0 && now.Sub(snap.RefreshedAt) > staleAfter
	return snap, prov, nil
}
