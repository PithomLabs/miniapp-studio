package store

import (
	"context"
	"testing"

	"github.com/PithomLabs/miniapp-studio/testdb"
)

// TestMigrationDownUpBounded is the decisive prompt30-bug experiment:
// full Up → Down to v15 → Up replay on a fresh scratch database — the
// exact sequence that stalled at SetVersion(17) in TestRLSEnable000016.
// Every leg runs through the bounded adapter with phase instrumentation,
// so any outcome here is decisive: pass closes the incident mechanism,
// failure arrives with the operation, version, and dirty state attached,
// and a stall becomes a go-test-timeout panic with a locating stack.
// Skips honestly without a bound Neon branch (via scratchDB/Policy).
func TestMigrationDownUpBounded(t *testing.T) {
	dsn := scratchDB(t, "studio_bounded_cycle")
	ctx := context.Background()
	pol := testdb.Policy(t)
	testdb.FrameworkBaseline(t, dsn)
	owner, err := NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Registered after scratchDB, so LIFO runs it BEFORE the drop:
	// owned backends close cleanly instead of being FORCE-killed.
	t.Cleanup(owner.Close)
	latest, err := LatestVersion()
	if err != nil {
		t.Fatal(err)
	}
	assertVersion := func(want uint, what string) {
		t.Helper()
		ver, dirty, err := CurrentVersion(ctx, owner)
		if err != nil {
			t.Fatalf("%s: version read: %v", what, err)
		}
		t.Logf("%s: v%d dirty=%v", what, ver, dirty)
		if ver != want || dirty {
			t.Fatalf("%s: v%d dirty=%v, want v%d clean", what, ver, dirty, want)
		}
	}

	t.Log("leg 1/3: full up")
	if err := Migrate(ctx, dsn, pol); err != nil {
		t.Fatalf("full up: %v", err)
	}
	assertVersion(latest, "after full up")

	t.Log("leg 2/3: down to v15")
	v15 := uint(15)
	if err := migrateTo(ctx, dsn, pol, &v15); err != nil {
		t.Fatalf("down to v15: %v", err)
	}
	assertVersion(v15, "after down")

	t.Log("leg 3/3: replay up")
	if err := Migrate(ctx, dsn, pol); err != nil {
		t.Fatalf("replay up: %v", err)
	}
	assertVersion(latest, "after replay")
	t.Log("done: full cycle clean")
}
