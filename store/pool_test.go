package store

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestPoolResetsTenantSetting proves F7: a session-level tenant setting left
// on a released connection can never leak into a later checkout, and the
// transaction helper still works without leaking after commit.
func TestPoolResetsTenantSetting(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	insertAppRow(t, env.owner, "tA", "app-a", "a")

	pool, err := NewPool(ctx, env.appDSN, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	// Contaminate one checkout, then release it without cleanup.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `SELECT set_config('app.tenant_id', 'tA', false)`); err != nil {
		t.Fatal(err)
	}
	conn.Release()

	// Later checkouts must see nothing without explicit tenant context.
	for i := 0; i < 4; i++ {
		clean, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var n int
		err = clean.QueryRow(ctx, `SELECT count(*) FROM studio.apps`).Scan(&n)
		clean.Release()
		if err != nil {
			t.Fatalf("bare read: %v", err)
		}
		if n != 0 {
			t.Fatalf("checkout %d saw %d rows: tenant setting leaked across release", i, n)
		}
	}

	// The helper path keeps working and leaves nothing behind.
	var seen int
	if err := InTenantTx(ctx, pool, "tA", func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM studio.apps`).Scan(&seen)
	}); err != nil {
		t.Fatalf("InTenantTx: %v", err)
	}
	if seen != 1 {
		t.Errorf("helper saw %d rows, want 1", seen)
	}
	clean, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer clean.Release()
	var n int
	if err := clean.QueryRow(ctx, `SELECT count(*) FROM studio.apps`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("post-helper checkout saw %d rows, want 0", n)
	}
}

// TestPoolDiscardsUnresettable proves reset failure destroys instead of
// reusing. With MaxConns 1, reusing the broken connection would fail the
// re-acquire; success proves transparent replacement.
func TestPoolDiscardsUnresettable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	pool, err := NewPool(ctx, env.appDSN, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Break the connection underneath the pool, then release: reset must
	// fail and the pool must destroy rather than reuse.
	_ = conn.Conn().Close(context.Background())
	conn.Release()
	fresh, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("re-acquire: %v", err)
	}
	defer fresh.Release()
	var one int
	if err := fresh.QueryRow(ctx, `SELECT 1`).Scan(&one); err != nil || one != 1 {
		t.Fatalf("fresh checkout broken: %v", err)
	}
}
