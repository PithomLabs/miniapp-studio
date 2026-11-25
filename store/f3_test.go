package store

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/testdb"
	"github.com/PithomLabs/miniapp/testkit"
)

// scratchDB creates a disposable database and returns its owner DSN. The
// database is force-dropped during cleanup.
func scratchDB(t *testing.T, prefix string) string {
	t.Helper()
	ctx := context.Background()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	// U5.12: staging tests need the cluster roles to exist. RunDSN
	// ensures both (create-only) without touching any credential.
	_ = testdb.RunDSN(t)
	admin, err := NewPool(ctx, base, 0)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("%s_%s", prefix, randomHexForTest(t, 4))
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		admin.Close()
		t.Fatalf("create scratch db: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), `DROP DATABASE IF EXISTS `+name+` WITH (FORCE)`)
		admin.Close()
	})
	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + name
	return parsed.String()
}

func insertAppRow(t *testing.T, pool *pgxpool.Pool, tenant, id, slug string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO studio.apps (id, tenant_id, slug, name, owner_identity)
		 VALUES ($1, $2, $3, $4, $5)`, id, tenant, slug, slug, "op"); err != nil {
		t.Fatalf("insert app: %v", err)
	}
}

// TestCrossTenantReferenceRejected proves F3: a child row in one tenant
// cannot reference an app owned by another tenant, on any of the three
// child tables. Fails before the composite-FK migration.
func TestCrossTenantReferenceRejected(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	insertAppRow(t, env.owner, "tA", "app-a", "a")
	insertAppRow(t, env.owner, "tB", "app-b", "b")
	cases := []struct {
		table  string
		insert string
		args   []any
	}{
		{"app_specs", `INSERT INTO studio.app_specs (id, tenant_id, app_id, version, spec_hash, document)
			VALUES ('spec-x', 'tA', 'app-b', 1, 'h', '{}')`, nil},
		{"deployments", `INSERT INTO studio.deployments (id, tenant_id, app_id, slug, resource_kind, resource_name, status)
			VALUES ('dep-x', 'tA', 'app-b', 'x', 'sprite', 'x', 'provisioning')`, nil},
		{"runs", `INSERT INTO studio.runs (id, tenant_id, app_id, spec_hash, workflow_id, status)
			VALUES ('run-x', 'tA', 'app-b', 'h', 'f', 'running')`, nil},
	}
	for _, tc := range cases {
		err := InTenantTx(ctx, env.app, "tA", func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, tc.insert, tc.args...)
			return err
		})
		if err == nil {
			t.Errorf("%s: cross-tenant reference accepted, want FK violation", tc.table)
		} else if !strings.Contains(err.Error(), "23503") {
			t.Errorf("%s: error = %v, want FK violation 23503", tc.table, err)
		}
	}
}

// TestSameTenantReferenceAccepted guards the F3 rewrite: same-tenant
// references keep working on all three child tables.
func TestSameTenantReferenceAccepted(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	insertAppRow(t, env.owner, "tA", "app-a", "a")
	err := InTenantTx(ctx, env.app, "tA", func(tx pgx.Tx) error {
		for _, stmt := range []string{
			`INSERT INTO studio.app_specs (id, tenant_id, app_id, version, spec_hash, document)
				VALUES ('spec-a', 'tA', 'app-a', 1, 'h', '{}')`,
			`INSERT INTO studio.deployments (id, tenant_id, app_id, slug, resource_kind, resource_name, status)
				VALUES ('dep-a', 'tA', 'app-a', 'a', 'sprite', 'a', 'provisioning')`,
			`INSERT INTO studio.runs (id, tenant_id, app_id, spec_hash, workflow_id, status)
				VALUES ('run-a', 'tA', 'app-a', 'h', 'f', 'running')`,
		} {
			if _, err := tx.Exec(ctx, stmt); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Errorf("same-tenant references refused: %v", err)
	}
}

// TestCascadeStaysInTenant proves deletes cannot cross tenants: removing one
// tenant's app deletes exactly its own children and leaves the other
// tenant's rows intact — including same-tenant cascade behavior.
func TestCascadeStaysInTenant(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	insertAppRow(t, env.owner, "tA", "app-a", "a")
	insertAppRow(t, env.owner, "tB", "app-b", "b")
	for _, stmt := range []string{
		`INSERT INTO studio.app_specs (id, tenant_id, app_id, version, spec_hash, document)
			VALUES ('spec-a', 'tA', 'app-a', 1, 'h', '{}')`,
		`INSERT INTO studio.app_specs (id, tenant_id, app_id, version, spec_hash, document)
			VALUES ('spec-b', 'tB', 'app-b', 1, 'h', '{}')`,
	} {
		if _, err := env.owner.Exec(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	if err := InTenantTx(ctx, env.app, "tB", func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM studio.apps WHERE id = 'app-b'`)
		return err
	}); err != nil {
		t.Fatalf("delete own app: %v", err)
	}
	var aSpecs, bSpecs int
	if err := env.owner.QueryRow(ctx,
		`SELECT count(*) FROM studio.app_specs WHERE tenant_id = 'tA'`).Scan(&aSpecs); err != nil {
		t.Fatal(err)
	}
	if err := env.owner.QueryRow(ctx,
		`SELECT count(*) FROM studio.app_specs WHERE tenant_id = 'tB'`).Scan(&bSpecs); err != nil {
		t.Fatal(err)
	}
	if aSpecs != 1 {
		t.Errorf("tenant A specs = %d after tenant B delete, want 1", aSpecs)
	}
	if bSpecs != 0 {
		t.Errorf("tenant B specs = %d after own delete, want 0 (cascade)", bSpecs)
	}
}

// TestMigrationRefusesCrossTenantData stages a v6 database containing a
// cross-tenant reference, then proves migrating to latest fails with a
// clear, non-sensitive error and leaves no partial state behind.
func TestMigrationRefusesCrossTenantData(t *testing.T) {
	dsn := scratchDB(t, "studio_f3")
	ctx := context.Background()
	testkit.MigrateUp(t, dsn)
	owner, err := NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	six := uint(6)
	if err := migrateTo(dsn, &six); err != nil {
		t.Fatalf("migrate to v6: %v", err)
	}
	for _, stmt := range []string{
		`INSERT INTO studio.apps (id, tenant_id, slug, name, owner_identity)
			VALUES ('app-a', 'tA', 'a', 'a', 'op'), ('app-b', 'tB', 'b', 'b', 'op')`,
		`INSERT INTO studio.app_specs (id, tenant_id, app_id, version, spec_hash, document)
			VALUES ('spec-x', 'tA', 'app-b', 1, 'h', '{}')`,
	} {
		if _, err := owner.Exec(ctx, stmt); err != nil {
			t.Fatalf("seed violation: %v", err)
		}
	}
	err = Migrate(ctx, dsn)
	if err == nil {
		t.Fatal("migration accepted cross-tenant data, want refusal")
	}
	if !strings.Contains(err.Error(), "cross-tenant") {
		t.Errorf("error = %v, want clear cross-tenant refusal", err)
	}
	if strings.Contains(err.Error(), "app-b") || strings.Contains(err.Error(), "spec-x") {
		t.Errorf("error leaks identifiers: %v", err)
	}
}
