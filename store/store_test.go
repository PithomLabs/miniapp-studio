package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/testkit"
)

// testEnv wires owner + application pools with fresh migrations. Tests share
// one database serially (the module runs -p 1); each test truncates Studio
// tables and provisions its own operators.
type testEnv struct {
	owner  *pgxpool.Pool
	app    *pgxpool.Pool
	appDSN string
	store  *Store
}

func setupTestEnv(t *testing.T) testEnv {
	t.Helper()
	ctx := context.Background()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	owner, err := NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatalf("owner pool: %v", err)
	}
	t.Cleanup(owner.Close)
	testkit.MigrateUp(t, dsn)
	pass := randomHexForTest(t, 32)
	if err := EnsureAppRole(ctx, owner, pass); err != nil {
		t.Fatalf("EnsureAppRole: %v", err)
	}
	appDSN := dsnForUser(t, dsn, AppRoleName, pass)
	if err := Migrate(ctx, dsn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	app, err := NewPool(ctx, appDSN, 0)
	if err != nil {
		t.Fatalf("app pool: %v", err)
	}
	t.Cleanup(app.Close)
	if _, err := owner.Exec(ctx, `TRUNCATE studio.sessions, studio.operators, studio.apps,
		studio.app_specs, studio.deployments, studio.runs`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return testEnv{owner: owner, app: app, appDSN: appDSN, store: New(app)}
}

func randomHexForTest(t *testing.T, n int) string {
	t.Helper()
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(raw)
}

// dsnForUser swaps credentials without ever emitting the secret: the parsed
// URL round-trips through encoding, and only connection behavior is asserted.
func dsnForUser(t *testing.T, dsn, user, pass string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	parsed.User = url.UserPassword(user, pass)
	return parsed.String()
}

func mustUpsertOperator(t *testing.T, env testEnv, tenant, binding string) {
	t.Helper()
	if err := UpsertOperator(context.Background(), env.owner, tenant, binding, true); err != nil {
		t.Fatalf("UpsertOperator: %v", err)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	env := setupTestEnv(t)
	if err := Migrate(context.Background(), os.Getenv("TEST_DATABASE_URL")); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	var version uint
	var dirty bool
	if err := env.owner.QueryRow(context.Background(),
		`SELECT version, dirty FROM studio_schema_migrations`).Scan(&version, &dirty); err != nil {
		t.Fatalf("studio version: %v", err)
	}
	// The expectation derives from the embedded migrations, so adding a
	// migration can never leave a stale hardcoded version behind.
	latest, err := LatestVersion()
	if err != nil {
		t.Fatal(err)
	}
	if version != latest || dirty {
		t.Errorf("studio schema at v%d dirty=%v, want v%d clean", version, dirty, latest)
	}
}

func TestMigrateRequiresFramework(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	ctx := context.Background()
	owner, err := NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	dbname := "studio_migcheck_" + randomHexForTest(t, 4)
	if _, err := owner.Exec(ctx, `CREATE DATABASE `+dbname); err != nil {
		t.Fatalf("create scratch db: %v", err)
	}
	defer func() { _, _ = owner.Exec(ctx, `DROP DATABASE IF EXISTS `+dbname) }()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + dbname
	if err := Migrate(ctx, parsed.String()); err == nil {
		t.Error("migrate without framework schema accepted, want refusal")
	} else if !strings.Contains(err.Error(), "framework schema") {
		t.Errorf("error = %v, want framework-schema refusal", err)
	}
}

func TestSessionLifecycle(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	mustUpsertOperator(t, env, "t1", "op1")
	token, sess, err := env.store.Create(ctx, "t1", "op1", time.Now())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token == "" || sess.TenantID != "t1" || sess.BindingID != "op1" {
		t.Errorf("session = %+v", sess)
	}
	got, err := env.store.Lookup(ctx, HashToken(token))
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got.TenantID != "t1" || got.BindingID != "op1" {
		t.Errorf("lookup = %+v", got)
	}
	// F9: the lookup returns the stable session id, matching what Create minted.
	if sess.ID == "" || got.ID != sess.ID {
		t.Errorf("lookup id = %q, create id = %q, want equal non-empty", got.ID, sess.ID)
	}
	if err := env.store.Revoke(ctx, HashToken(token)); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err := env.store.Lookup(ctx, HashToken(token)); err != ErrNoSession {
		t.Errorf("revoked lookup = %v, want ErrNoSession", err)
	}
	if err := env.store.Revoke(ctx, HashToken(token)); err != nil {
		t.Errorf("second revoke: %v", err)
	}
}

func TestSessionRejectsUnprovisioned(t *testing.T) {
	env := setupTestEnv(t)
	_, _, err := env.store.Create(context.Background(), "ghost", "op", time.Now())
	if err == nil {
		t.Error("session for unprovisioned operator accepted, want refusal")
	} else if !strings.Contains(err.Error(), "refused") {
		t.Errorf("error = %v, want operator-refusal (not a privilege failure)", err)
	}
}

func TestSessionExpiryBounds(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	mustUpsertOperator(t, env, "t1", "op1")
	now := time.Now()
	if _, err := env.store.CreateWithToken(ctx, "tok-past", "t1", "op1", now.Add(-time.Hour)); err == nil {
		t.Error("past-expiry session accepted, want refusal")
	}
	if _, err := env.store.CreateWithToken(ctx, "tok-far", "t1", "op1", now.Add(31*24*time.Hour)); err == nil {
		t.Error("over-limit expiry accepted, want refusal")
	}
	if _, err := env.store.CreateWithToken(ctx, "tok-ok", "t1", "op1", now.Add(time.Hour)); err != nil {
		t.Errorf("valid bounded session refused: %v", err)
	}
	// Expired sessions vanish from lookup even when the row lingers.
	if _, err := env.owner.Exec(ctx,
		`UPDATE studio.sessions SET expires_at = now() - INTERVAL '1 hour' WHERE token_hash = $1`,
		HashToken("tok-ok")); err != nil {
		t.Fatal(err)
	}
	if _, err := env.store.Lookup(ctx, HashToken("tok-ok")); err != ErrNoSession {
		t.Errorf("expired lookup = %v, want ErrNoSession", err)
	}
}

func TestLookupNoResultCases(t *testing.T) {
	env := setupTestEnv(t)
	for name, hash := range map[string]string{
		"forged": strings.Repeat("ab", 32),
		"empty":  "",
	} {
		if _, err := env.store.Lookup(context.Background(), hash); err != ErrNoSession {
			t.Errorf("%s token: lookup = %v, want ErrNoSession", name, err)
		}
	}
}

func TestDirectSessionReadDenied(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	_, err := env.app.Exec(ctx, `SELECT count(*) FROM studio.sessions`)
	if err == nil {
		t.Fatal("app role read sessions directly, want permission denied")
	}
	if !strings.Contains(err.Error(), "42501") || !strings.Contains(err.Error(), "sessions") {
		t.Errorf("error = %v, want table-level 42501 on sessions", err)
	}
	if _, err := env.app.Exec(ctx, `SELECT count(*) FROM studio.operators`); err == nil {
		t.Fatal("app role read operators directly, want permission denied")
	} else if !strings.Contains(err.Error(), "42501") || !strings.Contains(err.Error(), "operators") {
		t.Errorf("error = %v, want table-level 42501 on operators", err)
	}
}

func TestRoleAttributes(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	check := func(query string, want bool, what string) {
		t.Helper()
		var got bool
		if err := env.owner.QueryRow(ctx, query).Scan(&got); err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		if got != want {
			t.Errorf("%s = %v, want %v", what, got, want)
		}
	}
	check(`SELECT NOT rolbypassrls FROM pg_roles WHERE rolname = 'studio_app'`, true, "app NOBYPASSRLS")
	check(`SELECT NOT rolbypassrls FROM pg_roles WHERE rolname = 'studio_lookup'`, true, "lookup NOBYPASSRLS")
	check(`SELECT rolcanlogin FROM pg_roles WHERE rolname = 'studio_app'`, true, "app LOGIN")
	check(`SELECT NOT rolcanlogin FROM pg_roles WHERE rolname = 'studio_lookup'`, true, "lookup NOLOGIN")
	check(`SELECT NOT EXISTS (SELECT 1 FROM pg_auth_members m JOIN pg_roles r ON r.oid = m.roleid
		WHERE m.member IN (SELECT oid FROM pg_roles WHERE rolname IN ('studio_app','studio_lookup')))`,
		true, "no role memberships for runtime roles")
	check(`SELECT NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_roles r ON r.oid = c.relowner
		WHERE c.relnamespace = 'studio'::regnamespace
		AND r.rolname = 'studio_app')`, true, "app owns no studio objects")
	check(`SELECT count(*) = 0 FROM pg_proc p
		WHERE p.pronamespace = 'studio'::regnamespace
		AND has_function_privilege('public', p.oid, 'EXECUTE')`, true, "no PUBLIC execute")
	check(`SELECT has_function_privilege('studio_app', 'studio.session_lookup(text)', 'EXECUTE')`, true, "app executes lookup")
	check(`SELECT NOT has_function_privilege('studio_app', 'studio.operator_upsert(text,text,boolean)', 'EXECUTE')`,
		true, "app cannot execute operator_upsert")
	for _, table := range []string{"operators", "sessions", "apps", "app_specs", "deployments", "runs",
		"conversations", "messages", "builds", "turns", "questions",
		"commands", "events", "conversation_stream", "artifacts", "approvals"} {
		check(`SELECT relforcerowsecurity FROM pg_class WHERE relnamespace = 'studio'::regnamespace AND relname = '`+table+`'`,
			true, "FORCE RLS on "+table)
		check(`SELECT relrowsecurity FROM pg_class WHERE relnamespace = 'studio'::regnamespace AND relname = '`+table+`'`,
			true, "RLS enabled on "+table)
	}
}

// TestFrameworkTablesReachableAsApp proves the F2 correction: the granted
// framework tables are readable through the application role (RLS-filtered,
// never 42501). Fails before 000005 with permission denied for schema app.
func TestFrameworkTablesReachableAsApp(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	tables := []string{"invocations", "approvals", "audit_events", "effect_records", "kill_switches"}
	for _, table := range tables {
		var n int
		err := InTenantTx(ctx, env.app, "t-f2", func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, `SELECT count(*) FROM app.`+table).Scan(&n)
		})
		if err != nil {
			t.Errorf("%s: %v", table, err)
		}
	}
}
