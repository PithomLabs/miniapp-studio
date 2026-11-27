package dbenv

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/migratedb"
)

// TestRoleName is the application-behavior role for the suite.
const TestRoleName = "studio_app_test"

// TestRolePassword is the FIXED test-only credential. It must never
// change: concurrent test processes converge on role creation, and a
// per-run value would reintroduce the a1 invalidation race. Valid only
// on non-production Neon branches holding synthetic data (ADR-015);
// never valid anywhere real. Per-run credential issuance is
// T2-controller work, which retires this residual.
const TestRolePassword = "c0ffee5eed00d5eed00d5eed00d5eed00d5eed00d5eed00d5eed00d5eed00d5eed"

// ErrNoDSN reports a missing bootstrap database URL. Callers map it to
// "skip" (tests) or "fail with setup instructions" (CLI), never to a
// default that could aim at a live database.
var ErrNoDSN = errors.New("TEST_DATABASE_URL unset")

var (
	rxRunDB   = regexp.MustCompile(`^studio_test_run_[0-9]+_[0-9]+_[0-9a-f]+$`)
	rxHexPass = regexp.MustCompile(`^[0-9a-f]{32,}$`)
)

// liveRoleName is the production application role. dbenv references it
// by literal (not by importing store) to avoid an import cycle; the
// literal is pinned against store.AppRoleName by test.
func liveRoleName() string { return "studio_app" }

// IsRunDB reports whether name matches the run-database pattern. Every
// destructive or grant operation re-checks with this predicate
// immediately before acting; the regex is the single source of truth.
func IsRunDB(name string) bool { return rxRunDB.MatchString(name) }

// NewRunDBName generates a unique run-database name. The embedded pid,
// timestamp, and randomness make collisions across concurrent
// processes practically impossible; the name always matches IsRunDB.
func NewRunDBName() (string, error) {
	raw := make([]byte, 4)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	name := fmt.Sprintf("studio_test_run_%d_%d_%s",
		os.Getpid(), time.Now().Unix(), hex.EncodeToString(raw))
	if !rxRunDB.MatchString(name) {
		return "", fmt.Errorf("dbenv: generated run name %q fails the run guard", name)
	}
	return name, nil
}

// CreateRunDB validates the bootstrap URL against the branch policy,
// verifies PG18, creates one run database, and returns the maintainer
// DSN rewritten to it. The bootstrap URL's database component is
// ignored, so a harness can never be pointed at live data by
// misconfiguration (invariant 7).
func CreateRunDB(ctx context.Context, base string, pol EndpointPolicy) (string, error) {
	if base == "" {
		return "", ErrNoDSN
	}
	// Endpoint policy first — an unbound URL is rejected before any
	// connection is opened.
	ep, err := CheckEndpoint(base, pol)
	if err != nil {
		return "", err
	}
	name, err := NewRunDBName()
	if err != nil {
		return "", err
	}
	maint, err := Pool(ctx, base, pol)
	if err != nil {
		return "", err
	}
	defer maint.Close()
	if err := CheckPGMajor(ctx, maint); err != nil {
		return "", err
	}
	if _, err := maint.Exec(ctx, `CREATE DATABASE `+quoteIdent(name)); err != nil {
		return "", fmt.Errorf("dbenv: create run database: %w", err)
	}
	ep.URL.Path = "/" + name
	return ep.URL.String(), nil
}

// EnsureClusterRoles ensures the live role (existence only, for the
// migrate gate) and the fixed test role on the given database. It
// refuses to run against any database outside the run pattern. The
// connection is policy-enforced with pinned TLS verification.
func EnsureClusterRoles(ctx context.Context, dsn string, pol EndpointPolicy) error {
	pool, err := Pool(ctx, dsn, pol)
	if err != nil {
		return err
	}
	defer pool.Close()
	var dbname string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&dbname); err != nil {
		return fmt.Errorf("dbenv: current database: %w", err)
	}
	if !rxRunDB.MatchString(dbname) {
		return fmt.Errorf("dbenv: refusing roles on non-run database %q", dbname)
	}
	if err := ensureRoleExists(ctx, pool, liveRoleName()); err != nil {
		return err
	}
	return ensureTestRole(ctx, pool)
}

// ensureTestRole creates the test role once with its fixed password.
// An existing role is untouched — never rotated (invariant 3).
// Concurrent creators converge on duplicate_object like the live role.
func ensureTestRole(ctx context.Context, owner *pgxpool.Pool) error {
	if !rxHexPass.MatchString(TestRolePassword) {
		return fmt.Errorf("dbenv: fixed test password violates the hex rule")
	}
	return ensureRoleWithPassword(ctx, owner, TestRoleName, TestRolePassword)
}

// ensureRoleExists creates the live role when the cluster lacks it, so
// the Studio migrate gate (which requires existence) passes on fresh
// test clusters. Creation-only: an existing password is never touched,
// and the generated value is discarded immediately — nothing
// authenticates as the live role.
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
		return fmt.Errorf("dbenv: refusing unmanaged role %q", role)
	}
	if !rxHexPass.MatchString(password) {
		return fmt.Errorf("dbenv: password violates the hex rule")
	}
	var exists bool
	if err := owner.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, role,
	).Scan(&exists); err != nil {
		return fmt.Errorf("dbenv: role check: %w", err)
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
		return fmt.Errorf("dbenv: create role: %w", err)
	}
	return nil
}

// studioTables mirrors the studio_app DML grant set from migrations
// 000002 (apps, app_specs, deployments, runs), 000011 (conversations,
// messages), 000012 (builds, turns, questions), 000013 (commands,
// events), 000014 (conversation_stream, artifacts, approvals),
// 000021 (artifact_blobs, artifact_verifications, artifact_test_runs),
// 000024 (checkpoints), 000025 (sprite_lifecycle_receipts).
// Deliberately absent: sessions, operators (no table access by design).
var studioTables = []string{
	"apps", "app_specs", "deployments", "runs",
	"conversations", "messages", "builds", "turns", "questions",
	"commands", "events", "conversation_stream", "artifacts", "approvals",
	"artifact_blobs", "artifact_verifications", "artifact_test_runs",
	"checkpoints", "sprite_lifecycle_receipts",
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
		return fmt.Errorf("dbenv: current database: %w", err)
	}
	if !rxRunDB.MatchString(dbname) {
		return fmt.Errorf("dbenv: refusing grants on non-run database %q", dbname)
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
	}
	for _, s := range stmts {
		if _, err := pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("dbenv: grant: %w", err)
		}
	}
	// Function grants ride a dedicated owner-role connection (Card
	// 001): grants on studio_lookup-owned functions issued by the
	// migrator-as-member take no effect, so they run AS studio_lookup.
	return grantTestFunctionExec(ctx, pool, role)
}

// grantTestFunctionExec mirrors the session-function EXECUTE set onto
// the test role AS studio_lookup over one dedicated connection, with
// RESET guaranteed before the connection returns to the pool. A
// pooled SET ROLE without guaranteed RESET would leak the owner role
// into the next checkout; the double RESET (explicit + deferred
// backstop) closes that.
func grantTestFunctionExec(ctx context.Context, pool *pgxpool.Pool, role string) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("dbenv: acquire for function grants: %w", err)
	}
	defer conn.Release()
	defer func() {
		_, _ = conn.Exec(context.Background(), `RESET ROLE`)
	}()
	if _, err := conn.Exec(ctx, `SET ROLE studio_lookup`); err != nil {
		return fmt.Errorf("dbenv: set role for function grants: %w", err)
	}
	for _, s := range []string{
		`GRANT EXECUTE ON FUNCTION studio.session_lookup(TEXT) TO ` + role,
		`GRANT EXECUTE ON FUNCTION studio.session_create(TEXT, TEXT, TEXT, TEXT, TIMESTAMPTZ) TO ` + role,
		`GRANT EXECUTE ON FUNCTION studio.session_revoke(TEXT) TO ` + role,
	} {
		if _, err := conn.Exec(ctx, s); err != nil {
			return fmt.Errorf("dbenv: grant: %w", err)
		}
	}
	if _, err := conn.Exec(ctx, `RESET ROLE`); err != nil {
		return fmt.Errorf("dbenv: reset role after function grants: %w", err)
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

// miniappModule is the pinned framework module whose migrations/ tree
// is the framework baseline. Resolved via the Go toolchain so the
// version always tracks go.mod, never a hardcoded path.
const miniappModule = "github.com/PithomLabs/miniapp"

// FrameworkModuleDir resolves the on-disk framework module directory
// using the Go toolchain. It must run anywhere inside the checkout.
func FrameworkModuleDir(ctx context.Context, repoRoot string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-f", "{{.Dir}}", miniappModule)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("dbenv: resolve framework module: %w", err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return "", fmt.Errorf("dbenv: framework module dir empty")
	}
	return dir, nil
}

// ApplyFrameworkBaseline applies the framework migration chain to dsn
// through the bounded migratedb adapter (same migration files as
// testkit.MigrateUp, "no change" tolerated) without the *testing.T
// dependency, so CLI-driven bootstrap follows the identical path as
// the test harness. The stock pgx driver path is retired: all
// framework migrations run bounded with pinned TLS.
func ApplyFrameworkBaseline(ctx context.Context, dsn, moduleDir string, pol EndpointPolicy) error {
	poolCfg, err := PoolConfig(dsn, pol)
	if err != nil {
		return err
	}
	sink, phases := migratedb.CaptureSink(12)
	drv, err := migratedb.Open(ctx, migratedb.Config{
		ConnConfig: *poolCfg.ConnConfig,
		Table:      "schema_migrations",
		Sink:       sink,
	})
	if err != nil {
		return fmt.Errorf("dbenv: framework migrate driver: %w (phases: %s)", err, phases())
	}
	migrations := filepath.Join(moduleDir, "migrations")
	m, err := migrate.NewWithDatabaseInstance("file://"+migrations, "migratedb", drv)
	if err != nil {
		_ = drv.Close()
		return fmt.Errorf("dbenv: framework migrate setup: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err.Error() != "no change" {
		return fmt.Errorf("dbenv: framework migrate up: %w (phases: %s)", err, phases())
	}
	return nil
}

// RunDB describes one run-pattern database for inspection and safe
// cleanup. CreatedUnix comes from the embedded timestamp in the name.
type RunDB struct {
	Name        string
	CreatedUnix int64
	Backends    int
}

// ListRunDBs returns all run-pattern databases with backend counts.
// Non-matching names are never returned, however they were created.
func ListRunDBs(ctx context.Context, pool *pgxpool.Pool) ([]RunDB, error) {
	rows, err := pool.Query(ctx, `SELECT d.datname, COALESCE(
		(SELECT count(*) FROM pg_stat_activity a WHERE a.datname = d.datname), 0)
		FROM pg_database d WHERE d.datname LIKE 'studio_test_run\_%' ESCAPE '\'`)
	if err != nil {
		return nil, fmt.Errorf("dbenv: list run databases: %w", err)
	}
	defer rows.Close()
	var out []RunDB
	for rows.Next() {
		var name string
		var backends int
		if err := rows.Scan(&name, &backends); err != nil {
			return nil, fmt.Errorf("dbenv: scan run databases: %w", err)
		}
		if !rxRunDB.MatchString(name) {
			continue
		}
		parts := strings.Split(name, "_")
		ts, err := strconv.ParseInt(parts[4], 10, 64)
		if err != nil {
			continue
		}
		out = append(out, RunDB{Name: name, CreatedUnix: ts, Backends: backends})
	}
	return out, rows.Err()
}

// StaleRunDBs filters run databases to those older than maxAge with
// zero connected backends. Pure: the drop decision is unit-testable
// without a database.
func StaleRunDBs(dbs []RunDB, now time.Time, maxAge time.Duration) []string {
	var out []string
	for _, db := range dbs {
		if db.Backends > 0 {
			continue
		}
		if now.Unix()-db.CreatedUnix < int64(maxAge.Seconds()) {
			continue
		}
		out = append(out, db.Name)
	}
	return out
}

// QuoteIdent quotes an identifier for safe interpolation. Exported so
// reset paths share one implementation with the grant path.
func QuoteIdent(s string) string { return quoteIdent(s) }
