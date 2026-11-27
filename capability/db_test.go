package capability

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/dbenv"
	"github.com/PithomLabs/miniapp-studio/fly"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/testdb"
)

func testService(t *testing.T, lister ToolLister, conns fly.ConnectorManager) (*Service, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	dsn := testdb.RunDSN(t)
	owner, err := store.NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatalf("owner pool: %v", err)
	}
	t.Cleanup(owner.Close)
	if err := store.Migrate(ctx, dsn, testdb.Policy(t)); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := dbenv.GrantTestPrivileges(ctx, owner); err != nil {
		t.Fatalf("GrantTestPrivileges: %v", err)
	}
	app, err := store.NewPool(ctx, testdb.AppDSN(t, dsn), 0)
	if err != nil {
		t.Fatalf("app pool: %v", err)
	}
	t.Cleanup(app.Close)
	svc, err := New(app, lister, conns)
	if err != nil {
		t.Fatal(err)
	}
	return svc, owner
}

func scriptedLister(tools ...DeclaredTool) ToolLister {
	return func() (string, string, []DeclaredTool, error) {
		return "9.9.9", "test-build", tools, nil
	}
}

func TestCapabilityRoundTrip(t *testing.T) {
	fake := fly.NewFakeConnectorManager()
	fake.Connections = []fly.Connection{{ID: "c1", Provider: "google", ConnectionType: "oauth"}}
	svc, _ := testService(t, scriptedLister(
		DeclaredTool{Name: "app.create", Bound: true, Version: "9.9.9"},
		DeclaredTool{Name: "app.list", Bound: true, Version: "9.9.9"},
	), fake)
	ctx := context.Background()
	now := time.Now()
	if err := svc.Refresh(ctx, "tenant-cap-1", now); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	snap, prov, err := svc.Read(ctx, "tenant-cap-1", time.Hour, now)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(snap.Entries) != 3 || !snap.ConnectorsConfigured || snap.Stale {
		t.Fatalf("snapshot = %+v", snap)
	}
	for _, e := range snap.Entries {
		if e.State != StateImplemented {
			t.Fatalf("entry %+v not implemented", e)
		}
	}
	if len(prov) != 2 {
		t.Fatalf("provenance = %+v", prov)
	}
}

func TestCapabilityFailureRetainsSnapshot(t *testing.T) {
	fake := fly.NewFakeConnectorManager()
	svc, _ := testService(t, scriptedLister(DeclaredTool{Name: "t", Bound: true}), fake)
	ctx := context.Background()
	if err := svc.Refresh(ctx, "tenant-cap-2", time.Now()); err != nil {
		t.Fatal(err)
	}
	before, _, err := svc.Read(ctx, "tenant-cap-2", 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	// Connector failure blocks publish; the old snapshot stands.
	fake.Fail["ListConnections"] = errors.New("provider down")
	if err := svc.Refresh(ctx, "tenant-cap-2", time.Now()); err == nil {
		t.Fatal("failed refresh published")
	}
	after, _, err := svc.Read(ctx, "tenant-cap-2", 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !after.RefreshedAt.Equal(before.RefreshedAt) {
		t.Fatal("failed refresh moved the snapshot")
	}
}

func TestCapabilityUnconfiguredManager(t *testing.T) {
	svc, _ := testService(t, scriptedLister(DeclaredTool{Name: "t", Bound: true}), nil)
	ctx := context.Background()
	if err := svc.Refresh(ctx, "tenant-cap-3", time.Now()); err != nil {
		t.Fatalf("unconfigured refresh: %v", err)
	}
	snap, prov, err := svc.Read(ctx, "tenant-cap-3", 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if snap.ConnectorsConfigured || len(snap.Entries) != 1 {
		t.Fatalf("snapshot = %+v", snap)
	}
	if len(prov) != 2 || prov[1].Note == "" {
		t.Fatalf("provenance = %+v", prov)
	}
}

func TestCapabilityDriftIsDataNotBlocker(t *testing.T) {
	svc, _ := testService(t, scriptedLister(
		DeclaredTool{Name: "ok.tool", Bound: true},
		DeclaredTool{Name: "ghost.tool", Bound: false},
	), nil)
	ctx := context.Background()
	if err := svc.Refresh(ctx, "tenant-cap-4", time.Now()); err != nil {
		t.Fatalf("drifted refresh: %v", err)
	}
	snap, _, err := svc.Read(ctx, "tenant-cap-4", 0, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var failed *Entry
	for i, e := range snap.Entries {
		if e.Operation == "ghost.tool" {
			failed = &snap.Entries[i]
		}
	}
	if failed == nil || failed.State != StateFailed || failed.Detail == "" {
		t.Fatalf("drift entry = %+v", snap.Entries)
	}
}

func TestCapabilityTenantIsolationAndStale(t *testing.T) {
	svc, owner := testService(t, scriptedLister(DeclaredTool{Name: "t", Bound: true}), nil)
	ctx := context.Background()
	if err := svc.Refresh(ctx, "tenant-cap-5", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Read(ctx, "tenant-cap-other", 0, time.Now()); !errors.Is(err, ErrNoSnapshot) {
		t.Fatalf("foreign read err = %v, want ErrNoSnapshot", err)
	}
	if _, err := owner.Exec(ctx, `UPDATE studio.capability_snapshots SET refreshed_at = now() - interval '2 hours' WHERE tenant_id = 'tenant-cap-5'`); err != nil {
		t.Fatal(err)
	}
	snap, _, err := svc.Read(ctx, "tenant-cap-5", time.Hour, time.Now())
	if err != nil || !snap.Stale {
		t.Fatalf("stale read = %+v, %v", snap, err)
	}
	fresh, _, err := svc.Read(ctx, "tenant-cap-5", 3*time.Hour, time.Now())
	if err != nil || fresh.Stale {
		t.Fatalf("fresh-threshold read = %+v, %v", fresh, err)
	}
}

func TestCapabilityConcurrentRefresh(t *testing.T) {
	svc, _ := testService(t, scriptedLister(DeclaredTool{Name: "t", Bound: true}), nil)
	ctx := context.Background()
	const racers = 8
	var wg sync.WaitGroup
	errs := make(chan error, racers)
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- svc.Refresh(ctx, "tenant-cap-6", time.Now())
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent refresh: %v", err)
		}
	}
	snap, _, err := svc.Read(ctx, "tenant-cap-6", 0, time.Now())
	if err != nil || len(snap.Entries) != 1 {
		t.Fatalf("post-race snapshot = %+v, %v", snap, err)
	}
}
