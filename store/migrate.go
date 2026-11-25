package store

import (
	"context"
	"embed"
	"fmt"
	"net/url"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// MigrationsTable isolates Studio's version tracking from the framework's
// schema_migrations table in the shared database.
const MigrationsTable = "studio_schema_migrations"

// MinFrameworkVersion is the oldest framework schema the Studio migrations
// accept. Studio tables reference framework concepts by tenant convention,
// so the framework baseline must be present first.
const MinFrameworkVersion = 6

// AppRoleName is the provisioned application role. It is created out of band
// (see EnsureAppRole) before the migrations that grant to it.
const AppRoleName = "studio_app"

// Migrate applies pending Studio migrations over ownerDSN. It refuses to run
// unless the framework schema is present at MinFrameworkVersion (clean) and
// the application role exists. Migration state lives in MigrationsTable, so
// framework and Studio versioning never collide.
func Migrate(ctx context.Context, ownerDSN string) error {
	pool, err := pgxpool.New(ctx, ownerDSN)
	if err != nil {
		return fmt.Errorf("store: migrate connect: %w", err)
	}
	defer pool.Close()
	version, dirty, err := frameworkVersion(ctx, pool)
	if err != nil {
		return fmt.Errorf("store: migrate: framework schema check: %w", err)
	}
	if dirty {
		return fmt.Errorf("store: migrate: framework schema is dirty")
	}
	if version < MinFrameworkVersion {
		return fmt.Errorf("store: migrate: framework schema at v%d, need >= v%d", version, MinFrameworkVersion)
	}
	var hasAppRole bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, AppRoleName,
	).Scan(&hasAppRole); err != nil {
		return fmt.Errorf("store: migrate: role check: %w", err)
	}
	if !hasAppRole {
		return fmt.Errorf("store: migrate: role %q missing (run provision first)", AppRoleName)
	}
	return migrateTo(ownerDSN, nil)
}

// migrateTo migrates to a target version (nil means latest). Tests use it to
// stage databases at older versions; production always migrates to latest.
func migrateTo(ownerDSN string, target *uint) error {
	src, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("store: migrate source: %w", err)
	}
	dsn, err := withMigrationsTable(ownerDSN)
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return fmt.Errorf("store: migrate instance: %w", err)
	}
	defer func() { _, _ = m.Close() }()
	if target == nil {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("store: migrate up: %w", err)
		}
	} else if err := m.Migrate(*target); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("store: migrate to v%d: %w", *target, err)
	}
	if _, _, err := m.Version(); err != nil {
		return fmt.Errorf("store: migrate version: %w", err)
	}
	return nil
}

func frameworkVersion(ctx context.Context, pool *pgxpool.Pool) (uint, bool, error) {
	var version uint
	var dirty bool
	err := pool.QueryRow(ctx, `SELECT version, dirty FROM public.schema_migrations`).Scan(&version, &dirty)
	if err != nil {
		return 0, false, fmt.Errorf("read framework schema version: %w", err)
	}
	return version, dirty, nil
}

func withMigrationsTable(dsn string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("store: parse DSN: %w", err)
	}
	query := parsed.Query()
	query.Set("x-migrations-table", MigrationsTable)
	parsed.RawQuery = query.Encode()
	// pgx/v5 migrate driver expects a postgres-compatible URL scheme.
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return "", fmt.Errorf("store: DSN scheme must be postgres")
	}
	return strings.Replace(parsed.String(), "postgres://", "pgx5://", 1), nil
}
