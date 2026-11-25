package store

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	// U5.12 invariant 1: re-ensuring the live role never changes its
	// password. The old password keeps working; the ignored value does
	// not. (The Boundary B a1 abort is the red-before for the old
	// rotate-on-boot behavior; this assertion is the green-after.)
	decoy := randomHexForTest(t, 32)
	if err := EnsureAppRole(ctx, env.owner, decoy); err != nil {
		t.Fatalf("EnsureAppRole again: %v", err)
	}
	decoyPool, err := NewPool(ctx, dsnForUser(t, env.dsn, AppRoleName, decoy), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer decoyPool.Close()
	if err := decoyPool.Ping(ctx); err == nil {
		t.Error("ignored decoy password authenticates: re-ensure rotated the live credential")
	}
	// The rotation MECHANISM is proven against a scratch role so the
	// suite never mutates a persistent credential (invariant 2 Scope).
	probe := "studio_u512_probe_" + randomHexForTest(t, 4)
	passA, passB := randomHexForTest(t, 32), randomHexForTest(t, 32)
	if _, err := env.owner.Exec(ctx, `CREATE ROLE `+probe+` WITH LOGIN NOBYPASSRLS PASSWORD '`+passA+`'`); err != nil {
		t.Fatalf("create probe role: %v", err)
	}
	defer func() { _, _ = env.owner.Exec(ctx, `DROP ROLE IF EXISTS `+probe) }()
	grantConnectDB(t, env.owner, probe)
	if err := rotateRolePassword(ctx, env.owner, probe, passB); err != nil {
		t.Fatalf("rotate probe: %v", err)
	}
	rotated, err := NewPool(ctx, dsnForUser(t, env.dsn, probe, passB), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer rotated.Close()
	if err := rotated.Ping(ctx); err != nil {
		t.Errorf("rotated probe credential cannot connect: %v", err)
	}
	stale, err := NewPool(ctx, dsnForUser(t, env.dsn, probe, passA), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer stale.Close()
	if err := stale.Ping(ctx); err == nil {
		t.Error("pre-rotation probe credential still connects, want rejection")
	}
	if err := rotateRolePassword(ctx, env.owner, probe, "not-hex!"); err == nil {
		t.Error("non-hex rotation accepted, want refusal")
	}
	if err := rotateRolePassword(ctx, env.owner, "studio_u512_probe_missing", passB); err == nil {
		t.Error("rotation of a missing role accepted, want refusal")
	}
	if err := EnsureAppRole(ctx, env.owner, "not-hex!"); err == nil {
		t.Error("non-hex password accepted, want refusal")
	}
}

// grantConnectDB gives a scratch login role CONNECT on the current
// database so dial assertions test the password, not the grant.
func grantConnectDB(t *testing.T, owner *pgxpool.Pool, role string) {
	t.Helper()
	var dbname string
	if err := owner.QueryRow(context.Background(), `SELECT current_database()`).Scan(&dbname); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(context.Background(),
		`GRANT CONNECT ON DATABASE `+pgx.Identifier{dbname}.Sanitize()+` TO `+role); err != nil {
		t.Fatalf("connect for %s: %v", role, err)
	}
}

// TestCreateOnceNeverReplacesPassword pins invariant 1 on the exact
// primitive EnsureAppRole uses: the first creation wins, and every
// later call with a different password is a no-op for the credential.
func TestCreateOnceNeverReplacesPassword(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	role := "studio_u512_once_" + randomHexForTest(t, 4)
	passA, passB := randomHexForTest(t, 32), randomHexForTest(t, 32)
	defer func() { _, _ = env.owner.Exec(ctx, `DROP ROLE IF EXISTS `+role) }()
	created, err := createRoleOnce(ctx, env.owner, role, passA)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("first create reported no creation")
	}
	grantConnectDB(t, env.owner, role)
	created, err = createRoleOnce(ctx, env.owner, role, passB)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Error("second create reported creation, want convergence")
	}
	dial := func(pass string) error {
		p, err := NewPool(ctx, dsnForUser(t, env.dsn, role, pass), 0)
		if err != nil {
			return err
		}
		defer p.Close()
		return p.Ping(ctx)
	}
	if err := dial(passA); err != nil {
		t.Errorf("original password rejected after re-ensure: %v", err)
	}
	if err := dial(passB); err == nil {
		t.Error("second password authenticates: re-ensure replaced the credential")
	}
}
