package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/PithomLabs/miniapp/testkit"
)

func tableExists(t *testing.T, dsn, table string) bool {
	t.Helper()
	ctx := context.Background()
	pool, err := NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var exists bool
	err = pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM studio.`+table+` LIMIT 1)`, //nolint:gosec // test-constant table name
	).Scan(&exists)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
			return false
		}
		t.Fatalf("probe %s: %v", table, err)
	}
	return true
}

// The Phase A down files must reverse cleanly: down to v10 removes exactly
// the four new table groups while older tables survive, and migrating back
// up restores v14 clean.
// phaseATables are the ten tables migrations 000011-000014 created. Do NOT
// extend this list for later migrations: it stages version-pinned
// assertions (v10/v15/v16) where newer tables cannot exist. Tables
// from 000015 onward get catalog coverage from the live table lists
// in store_test.go (RLS flags) and privilege_parity_test.go.
var phaseATables = []string{"conversations", "messages", "builds", "turns",
	"questions", "commands", "events", "conversation_stream", "artifacts", "approvals"}

// rlsFlags reads both RLS catalog flags for one studio table.
func rlsFlags(t *testing.T, dsn, table string) (enabled, forced bool) {
	t.Helper()
	ctx := context.Background()
	pool, err := NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.QueryRow(ctx,
		`SELECT relrowsecurity, relforcerowsecurity FROM pg_class
		 WHERE relnamespace = 'studio'::regnamespace AND relname = $1`,
		table).Scan(&enabled, &forced); err != nil {
		t.Fatalf("catalog %s: %v", table, err)
	}
	return enabled, forced
}

// TestRLSEnable000016 proves the closure item against staged schema
// versions on a scratch database: at v15 the Phase A tables carry FORCE
// without ENABLE (the shipped bug — policies inert); at v16 both flags
// hold on all ten tables.
func TestRLSEnable000016(t *testing.T) {
	dsn := scratchDB(t, "studio_rls_flags")
	testkit.MigrateUp(t, dsn)
	ctx := context.Background()
	owner, err := NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	if err := Migrate(ctx, dsn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	v15 := uint(15)
	if err := migrateTo(dsn, &v15); err != nil {
		t.Fatalf("down to v15: %v", err)
	}
	for _, table := range phaseATables {
		enabled, forced := rlsFlags(t, dsn, table)
		if enabled || !forced {
			t.Errorf("v15 %s: enabled=%v forced=%v, want the shipped inert state (false/true)", table, enabled, forced)
		}
	}
	if err := Migrate(ctx, dsn); err != nil {
		t.Fatalf("up to v16: %v", err)
	}
	for _, table := range phaseATables {
		enabled, forced := rlsFlags(t, dsn, table)
		if !enabled || !forced {
			t.Errorf("v16 %s: enabled=%v forced=%v, want true/true", table, enabled, forced)
		}
	}
}

func TestMigrateDownRoundTrip(t *testing.T) {
	dsn := scratchDB(t, "studio_migrate_down")
	testkit.MigrateUp(t, dsn)
	ctx := context.Background()
	owner, err := NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	if err := Migrate(ctx, dsn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	latest, err := LatestVersion()
	if err != nil {
		t.Fatal(err)
	}
	if latest < 14 {
		t.Fatalf("LatestVersion = %d, want >= 14", latest)
	}
	v10 := uint(10)
	if err := migrateTo(dsn, &v10); err != nil {
		t.Fatalf("down to v10: %v", err)
	}
	for _, gone := range []string{"conversations", "messages", "builds", "turns",
		"questions", "commands", "events", "conversation_stream", "artifacts", "approvals"} {
		if tableExists(t, dsn, gone) {
			t.Errorf("studio.%s survived down to v10", gone)
		}
	}
	for _, kept := range []string{"apps", "app_specs", "deployments", "sessions", "operators"} {
		if !tableExists(t, dsn, kept) {
			t.Errorf("studio.%s lost in down to v10", kept)
		}
	}
	version, dirty, err := CurrentVersion(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if version != 10 || dirty {
		t.Fatalf("after down: v%d dirty=%v, want v10 clean", version, dirty)
	}
	if err := Migrate(ctx, dsn); err != nil {
		t.Fatalf("back up: %v", err)
	}
	version, dirty, err = CurrentVersion(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if version != latest || dirty {
		t.Errorf("after re-up: v%d dirty=%v, want v%d clean", version, dirty, latest)
	}
}
