package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func insertApp(t *testing.T, env testEnv, tenant, id, slug string) {
	t.Helper()
	if _, err := env.owner.Exec(context.Background(),
		`INSERT INTO studio.apps (id, tenant_id, slug, name, owner_identity)
		 VALUES ($1, $2, $3, $4, $5)`, id, tenant, slug, slug, "op"); err != nil {
		t.Fatalf("insert app: %v", err)
	}
}

func countApps(t *testing.T, env testEnv, tenant string) int {
	t.Helper()
	var n int
	err := InTenantTx(context.Background(), env.app, tenant, func(tx pgx.Tx) error {
		if got := TenantOf(context.Background(), tx); got != tenant {
			t.Errorf("armed tenant = %q, want %q", got, tenant)
		}
		return tx.QueryRow(context.Background(), `SELECT count(*) FROM studio.apps`).Scan(&n)
	})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func TestTenantIsolation(t *testing.T) {
	env := setupTestEnv(t)
	insertApp(t, env, "t1", "app-1", "one")
	insertApp(t, env, "t2", "app-2", "two")
	if n := countApps(t, env, "t1"); n != 1 {
		t.Errorf("t1 sees %d apps, want 1", n)
	}
	if n := countApps(t, env, "t2"); n != 1 {
		t.Errorf("t2 sees %d apps, want 1", n)
	}
}

func TestTenantContextRequired(t *testing.T) {
	env := setupTestEnv(t)
	insertApp(t, env, "t1", "app-1", "one")
	var n int
	// No tenant armed: RLS shows nothing rather than everything.
	if err := env.app.QueryRow(context.Background(),
		`SELECT count(*) FROM studio.apps`).Scan(&n); err != nil {
		t.Fatalf("unscoped read: %v", err)
	}
	if n != 0 {
		t.Errorf("unscoped read saw %d rows, want 0", n)
	}
}

func TestTenantWriteChecked(t *testing.T) {
	env := setupTestEnv(t)
	err := InTenantTx(context.Background(), env.app, "t1", func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(),
			`INSERT INTO studio.apps (id, tenant_id, slug, name, owner_identity)
			 VALUES ('app-x', 't2', 'x', 'x', 'op')`)
		return err
	})
	if err == nil {
		t.Fatal("cross-tenant write accepted, want RLS denial")
	}
}

func TestInTenantTxRollback(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	err := InTenantTx(ctx, env.app, "t1", func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`INSERT INTO studio.apps (id, tenant_id, slug, name, owner_identity)
			 VALUES ('app-r', 't1', 'r', 'r', 'op')`); err != nil {
			return err
		}
		return context.DeadlineExceeded
	})
	if err != context.DeadlineExceeded {
		t.Fatalf("error = %v", err)
	}
	if n := countApps(t, env, "t1"); n != 0 {
		t.Errorf("rolled-back row persisted (%d rows)", n)
	}
}

func TestProvisionIdempotent(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	if err := UpsertOperator(ctx, env.owner, "t1", "op1", true); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := UpsertOperator(ctx, env.owner, "t1", "op1", false); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	if _, _, err := env.store.Create(ctx, "t1", "op1", time.Now()); err == nil {
		t.Error("session for deactivated operator accepted, want refusal")
	}
	if err := UpsertOperator(ctx, env.owner, "t1", "op1", true); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if _, _, err := env.store.Create(ctx, "t1", "op1", time.Now()); err != nil {
		t.Errorf("session after reactivate refused: %v", err)
	}
	// Role creation is idempotent; rotation takes effect for new connections.
	fresh := randomHexForTest(t, 32)
	if err := EnsureAppRole(ctx, env.owner, fresh); err != nil {
		t.Fatalf("EnsureAppRole again: %v", err)
	}
	rotated, err := NewPool(ctx, dsnForUser(t, os.Getenv("TEST_DATABASE_URL"), AppRoleName, fresh), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer rotated.Close()
	if err := rotated.Ping(ctx); err != nil {
		t.Errorf("rotated credential cannot connect: %v", err)
	}
	if err := EnsureAppRole(ctx, env.owner, "not-hex!"); err == nil {
		t.Error("non-hex password accepted, want refusal")
	}
}
