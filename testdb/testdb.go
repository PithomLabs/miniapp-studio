// Package testdb provisions isolated, run-specific PostgreSQL databases
// for the Go test suite (U5.12). It is TEST-ONLY: no non-test source
// file may import it (enforced by TestNoProdTestDBImport), and it
// creates no production migration.
//
// The shared, test-independent primitives (endpoint allowlist, run
// names, roles, grants, provisioning) live in dbenv so the `studio
// dev` command interface reuses the identical logic instead of
// reimplementing it; this package keeps only the *testing.T-bound
// orchestration (singletons, skips, fatals, cleanup).
//
// Model (Neon-first, ADR-004/006; the loopback cluster model was
// retired by the T1 C4 redesign):
//   - TEST_DATABASE_URL names the branch endpoint and maintainer
//     credentials; NEON_PROJECT_ID, NEON_BRANCH_ID, and
//     NEON_ENDPOINT_HOSTS bind it to one known branch. The URL's
//     database component is IGNORED, so a harness can never be
//     pointed at live data by misconfiguration (invariant 7).
//   - Each test process provisions exactly one run database,
//     studio_test_run_<pid>_<unixts>_<rand> (invariant 4). Tests in
//     the process share it serially under the existing truncate
//     discipline; concurrent processes never share a name.
//   - The suite exercises application behavior as studio_app_test, a
//     LOGIN NOBYPASSRLS role with a FIXED test-only password. Fixed
//     is load-bearing: concurrent creators converging on CREATE ROLE
//     must agree on the credential (invariant 3). The password is
//     valid only on non-production Neon branches holding synthetic
//     data; nothing outside dbenv may rotate it, and dbenv exposes
//     no rotation API.
//   - studio_app (the live role) is never authenticated by tests and
//     never rotated by tests (invariants 1, 6). Its existence is
//     ensured solely to satisfy the migrate gate.
//   - The test role receives the exact studio_app privilege mirror on
//     run databases only (invariant 5); grants refuse any database
//     outside the run pattern (invariant 7).
package testdb

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/dbenv"
)

var (
	runOnce sync.Once
	runDSN  string
	runErr  error
)

// RunDSN provisions (once per test process) the isolated run database
// and returns its maintainer DSN. It skips the test when
// TEST_DATABASE_URL is unset. The URL's database component is ignored;
// the run name is always generated. The database carries the framework
// baseline and both cluster roles; the caller applies Studio migrations
// and GrantTestPrivileges (in that order) before use.
func RunDSN(t *testing.T) string {
	t.Helper()
	runOnce.Do(func() { runDSN, runErr = provisionRunDB(t) })
	if runErr != nil {
		if errors.Is(runErr, dbenv.ErrNoDSN) {
			t.Skip("TEST_DATABASE_URL unset")
		}
		if errors.Is(runErr, dbenv.ErrNoPolicy) {
			t.Skipf("live Neon proof pending: %v", runErr)
		}
		t.Fatalf("testdb provision: %v", runErr)
	}
	// Guardrail: from here on, every environment-derived connection in
	// this process (testkit.NewDB, stray helpers, future code) lands on
	// the run database — never the bootstrap database (invariant 7).
	// The bootstrap URL is consumed exactly once, above, under sync.Once.
	_ = os.Setenv("TEST_DATABASE_URL", runDSN)
	return runDSN
}

func provisionRunDB(t *testing.T) (string, error) {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	pol, err := dbenv.PolicyFromEnv()
	if err != nil {
		return "", err
	}
	dsn, err := dbenv.CreateRunDB(context.Background(), base, pol)
	if err != nil {
		return "", err
	}
	// Framework baseline first (the Studio migrate gate requires it).
	FrameworkBaseline(t, dsn)
	if err := dbenv.EnsureClusterRoles(context.Background(), dsn, pol); err != nil {
		return "", err
	}
	maint, err := dbenv.Pool(context.Background(), base, pol)
	if err != nil {
		return "", err
	}
	defer maint.Close()
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	reapOldRunDBs(context.Background(), maint, strings.TrimPrefix(parsed.Path, "/"))
	return dsn, nil
}

// ScratchDSN creates a second, BARE run-pattern database for tests that
// stage their own schema versions (migrate up/down, schema gates). The
// caller migrates it; testdb grants nothing until asked. Force-dropped
// during test cleanup. Unique per call, so concurrent test processes
// never collide on fixed scratch names.
func ScratchDSN(t *testing.T) string {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	// Same endpoint gate as run provisioning.
	pol, err := dbenv.PolicyFromEnv()
	if err != nil {
		t.Skipf("live Neon proof pending: %v", err)
	}
	ep, err := dbenv.CheckEndpoint(base, pol)
	if err != nil {
		t.Fatal(err)
	}
	parsed := ep.URL
	name, err := dbenv.NewRunDBName()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	admin, err := dbenv.Pool(ctx, base, pol)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if err := dbenv.CheckPGMajor(ctx, admin); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		t.Fatalf("create scratch db: %v", err)
	}
	t.Cleanup(func() {
		p, err := dbenv.Pool(context.Background(), base, pol)
		if err != nil {
			return
		}
		defer p.Close()
		_, _ = p.Exec(context.Background(), `DROP DATABASE IF EXISTS `+dbenv.QuoteIdent(name)+` WITH (FORCE)`)
	})
	parsed.Path = "/" + name
	return parsed.String()
}

// AppDSN rewrites a maintainer run DSN to the test-role credential.
// It refuses URLs whose database is outside the run pattern, so even a
// hand-built DSN cannot aim the test role at a live database.
func AppDSN(t *testing.T, dsn string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if !dbenv.IsRunDB(strings.TrimPrefix(parsed.Path, "/")) {
		t.Fatalf("testdb: refusing app DSN for non-run database %q", parsed.Path)
	}
	parsed.User = url.UserPassword(dbenv.TestRoleName, dbenv.TestRolePassword)
	return parsed.String()
}

// MustTestDB aborts when name is outside the run-database pattern.
func MustTestDB(t *testing.T, name string) {
	t.Helper()
	if !dbenv.IsRunDB(name) {
		t.Fatalf("testdb: %q is not a run database", name)
	}
}

// Policy loads the branch endpoint policy for migrate callers. It
// skips (never fails) when Neon metadata is absent: without a bound
// branch there is nothing to migrate against.
func Policy(t *testing.T) dbenv.EndpointPolicy {
	t.Helper()
	pol, err := dbenv.PolicyFromEnv()
	if err != nil {
		t.Skipf("live Neon proof pending: %v", err)
	}
	return pol
}

// FrameworkBaseline applies the framework migration chain to dsn
// through the bounded adapter. It replaces testkit.MigrateUp at every
// Studio call site: identical migration files, but policy-bound,
// TLS-pinned, deadline-enforced execution instead of the stock driver.
func FrameworkBaseline(t *testing.T, dsn string) {
	t.Helper()
	pol := Policy(t)
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("testdb: cannot locate package source")
	}
	// go list -m resolves from anywhere inside the module.
	modDir, err := dbenv.FrameworkModuleDir(context.Background(), filepath.Dir(thisFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := dbenv.ApplyFrameworkBaseline(context.Background(), dsn, modDir, pol); err != nil {
		t.Fatalf("testdb: framework baseline: %v", err)
	}
}

// reapOldRunDBs drops abandoned run databases: name-pattern match,
// embedded timestamp older than 30 minutes, zero connected backends,
// and never the caller's own. Best effort; failures are ignored so a
// crowded cluster never fails provisioning.
func reapOldRunDBs(ctx context.Context, maint *pgxpool.Pool, self string) {
	rows, err := maint.Query(ctx, `SELECT datname FROM pg_database WHERE datname LIKE 'studio_test_run\_%' ESCAPE '\'`)
	if err != nil {
		return
	}
	names, err := pgxCollectStrings(rows)
	if err != nil {
		return
	}
	for _, name := range names {
		if name == self || !dbenv.IsRunDB(name) {
			continue
		}
		parts := strings.Split(name, "_")
		ts, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil || time.Since(time.Unix(ts, 0)) < 30*time.Minute {
			continue
		}
		var backends int
		if err := maint.QueryRow(ctx,
			`SELECT count(*) FROM pg_stat_activity WHERE datname = $1`, name,
		).Scan(&backends); err != nil || backends > 0 {
			continue
		}
		_, _ = maint.Exec(ctx, `DROP DATABASE `+dbenv.QuoteIdent(name))
	}
}

func pgxCollectStrings(rows pgx.Rows) ([]string, error) {
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
