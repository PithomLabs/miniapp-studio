// Package testdb provisions isolated, run-specific PostgreSQL databases
// for the Go test suite (U5.12). It is TEST-ONLY: no non-test source
// file may import it (enforced by TestNoProdTestDBImport), and it
// creates no production migration.
//
// Model:
//   - TEST_DATABASE_URL names the cluster and maintainer credentials;
//     its database component is IGNORED, so a harness can never be
//     pointed at live data by misconfiguration (invariant 7).
//   - Each test process provisions exactly one run database,
//     studio_test_run_<pid>_<unixts>_<rand> (invariant 4). Tests in
//     the process share it serially under the existing truncate
//     discipline; concurrent processes never share a name.
//   - The suite exercises application behavior as studio_app_test, a
//     LOGIN NOBYPASSRLS role with a FIXED test-only password. Fixed
//     is load-bearing: concurrent creators converging on CREATE ROLE
//     must agree on the credential (invariant 3). The password is
//     valid only on localhost scratch clusters holding synthetic
//     data; nothing outside this package may rotate it, and this
//     package exposes no rotation API.
//   - studio_app (the live role) is never authenticated by tests and
//     never rotated by tests (invariants 1, 6). Its existence is
//     ensured solely to satisfy the migrate gate.
//   - The test role receives the exact studio_app privilege mirror on
//     run databases only (invariant 5); grants refuse any database
//     outside the run pattern (invariant 7).
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/testkit"
)

// TestRoleName is the application-behavior role for the suite.
const TestRoleName = "studio_app_test"

// TestRolePassword is the FIXED test-only credential. It must never
// change: concurrent test processes converge on role creation, and a
// per-run value would reintroduce the a1 invalidation race. Test-only,
// localhost scratch clusters, synthetic data; never valid anywhere real.
const TestRolePassword = "c0ffee5eed00d5eed00d5eed00d5eed00d5eed00d5eed00d5eed00d5eed00d5eed"

var (
	rxRunDB   = regexp.MustCompile(`^studio_test_run_[0-9]+_[0-9]+_[0-9a-f]+$`)
	rxHexPass = regexp.MustCompile(`^[0-9a-f]{32,}$`)
)

// U5.12.1: the harness provisions test databases on exactly one
// approved endpoint — the local PostgreSQL 19 test cluster. Any
// other host or port is rejected BEFORE any connection is made, so a
// mispointed TEST_DATABASE_URL can never create a studio_test_run_*
// database on a remote or production cluster. Fail-closed by
// allowlist; change these constants only under explicit re-approval.
const allowedTestPort = "5434"

var allowedTestHosts = map[string]bool{
	"127.0.0.1": true,
	"localhost": true,
	"::1":       true,
}

// wantPGMajor is the only accepted server major version.
const wantPGMajor = 19

// CheckTestEndpoint parses a bootstrap database URL and enforces the
// endpoint allowlist. Pure: it opens no connection, so rejection
// tests prove the harness never dials an unapproved endpoint.
func CheckTestEndpoint(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("testdb: unapproved endpoint: parse: %w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return nil, fmt.Errorf("testdb: unapproved endpoint: scheme must be postgres")
	}
	if !allowedTestHosts[parsed.Hostname()] {
		return nil, fmt.Errorf("testdb: unapproved endpoint host %q (want local test cluster)", parsed.Hostname())
	}
	if parsed.Port() != allowedTestPort {
		return nil, fmt.Errorf("testdb: unapproved endpoint port %q (want %s)", parsed.Port(), allowedTestPort)
	}
	return parsed, nil
}

// checkPG19 verifies the connected server is PostgreSQL 19. Called
// immediately after connecting to an allowlisted endpoint and before
// any database is provisioned on it.
func checkPG19(ctx context.Context, pool *pgxpool.Pool) error {
	var num int
	if err := pool.QueryRow(ctx, `SELECT current_setting('server_version_num')::int`).Scan(&num); err != nil {
		return fmt.Errorf("testdb: server version check: %w", err)
	}
	if num/10000 != wantPGMajor {
		return fmt.Errorf("testdb: unapproved endpoint: requires PostgreSQL %d, server reports major %d",
			wantPGMajor, num/10000)
	}
	return nil
}

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
		if errors.Is(runErr, errNoDSN) {
			t.Skip("TEST_DATABASE_URL unset")
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

var errNoDSN = errors.New("TEST_DATABASE_URL unset")

func provisionRunDB(t *testing.T) (string, error) {
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		return "", errNoDSN
	}
	// U5.12.1: endpoint allowlist first — an unapproved host or
	// port is rejected before any connection is opened.
	parsed, err := CheckTestEndpoint(base)
	if err != nil {
		return "", err
	}
	raw := make([]byte, 4)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	name := fmt.Sprintf("studio_test_run_%d_%d_%s",
		os.Getpid(), time.Now().Unix(), hex.EncodeToString(raw))
	if !rxRunDB.MatchString(name) {
		return "", fmt.Errorf("generated run name %q fails the run guard", name)
	}
	ctx := context.Background()
	maint, err := pgxpool.New(ctx, base)
	if err != nil {
		return "", fmt.Errorf("maint pool: %w", err)
	}
	defer maint.Close()
	if err := checkPG19(ctx, maint); err != nil {
		return "", err
	}
	if _, err := maint.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		return "", fmt.Errorf("create run database: %w", err)
	}
	parsed.Path = "/" + name
	dsn := parsed.String()
	// Framework baseline first (the Studio migrate gate requires it).
	testkit.MigrateUp(t, dsn)
	// Both cluster roles are ensured create-only (passwords never
	// touched when the roles exist). Studio migrations and the test
	// privilege mirror are applied by the caller: testdb must not
	// import the store package (import cycle in store's own tests).
	owner, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return "", fmt.Errorf("run owner pool: %w", err)
	}
	defer owner.Close()
	if err := ensureRoleExists(ctx, owner, liveRoleName()); err != nil {
		return "", err
	}
	if err := ensureTestRole(ctx, owner); err != nil {
		return "", err
	}
	reapOldRunDBs(ctx, maint, name)
	return dsn, nil
}

// liveRoleName is the production application role. testdb references it
// by literal (not by importing store) to avoid an import cycle; a typo
// fails loudly because the Studio migrate gate requires the live role
// to exist on every run database.
func liveRoleName() string { return "studio_app" }

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
	// U5.12.1: same endpoint gate as run provisioning.
	parsed, err := CheckTestEndpoint(base)
	if err != nil {
		t.Fatal(err)
	}
	raw := make([]byte, 4)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("studio_test_run_%d_%d_%s",
		os.Getpid(), time.Now().Unix(), hex.EncodeToString(raw))
	if !rxRunDB.MatchString(name) {
		t.Fatalf("generated scratch name %q fails the run guard", name)
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if err := checkPG19(ctx, admin); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		t.Fatalf("create scratch db: %v", err)
	}
	t.Cleanup(func() {
		p, err := pgxpool.New(context.Background(), base)
		if err != nil {
			return
		}
		defer p.Close()
		_, _ = p.Exec(context.Background(), `DROP DATABASE IF EXISTS `+quoteIdent(name)+` WITH (FORCE)`)
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
	if !rxRunDB.MatchString(strings.TrimPrefix(parsed.Path, "/")) {
		t.Fatalf("testdb: refusing app DSN for non-run database %q", parsed.Path)
	}
	parsed.User = url.UserPassword(TestRoleName, TestRolePassword)
	return parsed.String()
}

// MustTestDB aborts when name is outside the run-database pattern.
func MustTestDB(t *testing.T, name string) {
	t.Helper()
	if !rxRunDB.MatchString(name) {
		t.Fatalf("testdb: %q is not a run database", name)
	}
}

// studioTables mirrors the studio_app DML grant set from migrations
// 000002 (apps, app_specs, deployments, runs), 000011 (conversations,
// messages), 000012 (builds, turns, questions), 000013 (commands,
// events), 000014 (conversation_stream, artifacts, approvals).
// Deliberately absent: sessions, operators (no table access by design).
var studioTables = []string{
	"apps", "app_specs", "deployments", "runs",
	"conversations", "messages", "builds", "turns", "questions",
	"commands", "events", "conversation_stream", "artifacts", "approvals",
}

// frameworkTables mirrors the 000002 app.* DML grant set.
var frameworkTables = []string{
	"invocations", "approvals", "audit_events", "effect_records", "kill_switches",
}

// GrantTestPrivileges applies the exact studio_app privilege mirror to
// the test role on the current database. pool must be a maintainer
// connection ON a run database; anything else is refused (invariant 7).
// Every grant below names a fixed migration-derived object; the
// privilege-parity test in the store package fails the suite if the
// mirror ever drifts from the live role.
func GrantTestPrivileges(ctx context.Context, pool *pgxpool.Pool) error {
	var dbname string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&dbname); err != nil {
		return fmt.Errorf("testdb: current database: %w", err)
	}
	if !rxRunDB.MatchString(dbname) {
		return fmt.Errorf("testdb: refusing grants on non-run database %q", dbname)
	}
	role := TestRoleName
	stmts := []string{
		`GRANT CONNECT ON DATABASE ` + quoteIdent(dbname) + ` TO ` + role,
		`GRANT USAGE ON SCHEMA studio, app TO ` + role,
		`GRANT SELECT, INSERT, UPDATE, DELETE ON ` + studioObjectList() + ` TO ` + role,
		`GRANT SELECT, INSERT, UPDATE, DELETE ON ` + frameworkObjectList() + ` TO ` + role,
		// 000010 grants the version table unqualified; resolve identically.
		`GRANT SELECT ON studio_schema_migrations TO ` + role,
		`GRANT USAGE, SELECT ON SEQUENCE app.audit_events_id_seq TO ` + role,
		`GRANT EXECUTE ON FUNCTION studio.session_lookup(TEXT) TO ` + role,
		`GRANT EXECUTE ON FUNCTION studio.session_create(TEXT, TEXT, TEXT, TEXT, TIMESTAMPTZ) TO ` + role,
		`GRANT EXECUTE ON FUNCTION studio.session_revoke(TEXT) TO ` + role,
	}
	for _, s := range stmts {
		if _, err := pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("testdb: grant: %w", err)
		}
	}
	return nil
}

func studioObjectList() string {
	qualified := make([]string, len(studioTables))
	for i, t := range studioTables {
		qualified[i] = "studio." + t
	}
	return strings.Join(qualified, ", ")
}

func frameworkObjectList() string {
	qualified := make([]string, len(frameworkTables))
	for i, t := range frameworkTables {
		qualified[i] = "app." + t
	}
	return strings.Join(qualified, ", ")
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// ensureTestRole creates the test role once with its fixed password.
// An existing role is untouched — never rotated (invariant 3).
// Concurrent creators converge on duplicate_object like the live role.
func ensureTestRole(ctx context.Context, owner *pgxpool.Pool) error {
	if !rxHexPass.MatchString(TestRolePassword) {
		return fmt.Errorf("testdb: fixed test password violates the hex rule")
	}
	return ensureRoleWithPassword(ctx, owner, TestRoleName, TestRolePassword)
}

// ensureRoleExists creates the live role when the cluster lacks it, so
// the Studio migrate gate (which requires existence) passes on fresh
// test clusters. Creation-only: an existing password is never touched,
// and the generated value is discarded immediately — nothing in the
// suite authenticates as the live role.
func ensureRoleExists(ctx context.Context, owner *pgxpool.Pool, role string) error {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	return ensureRoleWithPassword(ctx, owner, role, hex.EncodeToString(random))
}

// ensureRoleWithPassword is the test-side create-once twin of the
// production primitive: validated fixed names only, duplicate_object
// convergence, never a password change on an existing role.
func ensureRoleWithPassword(ctx context.Context, owner *pgxpool.Pool, role, password string) error {
	if role != TestRoleName && role != liveRoleName() {
		return fmt.Errorf("testdb: refusing unmanaged role %q", role)
	}
	if !rxHexPass.MatchString(password) {
		return fmt.Errorf("testdb: password violates the hex rule")
	}
	var exists bool
	if err := owner.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, role,
	).Scan(&exists); err != nil {
		return fmt.Errorf("testdb: role check: %w", err)
	}
	if exists {
		return nil
	}
	if _, err := owner.Exec(ctx, `CREATE ROLE `+role+
		` WITH LOGIN NOBYPASSRLS PASSWORD '`+password+`'`); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42710" {
			return nil
		}
		return fmt.Errorf("testdb: create role: %w", err)
	}
	return nil
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
		if name == self || !rxRunDB.MatchString(name) {
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
		_, _ = maint.Exec(ctx, `DROP DATABASE `+quoteIdent(name))
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
