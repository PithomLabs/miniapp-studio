package store

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/dbenv"
	"github.com/PithomLabs/miniapp-studio/migratedb"
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
// framework and Studio versioning never collide. Execution runs through the
// bounded migratedb adapter (never the stock driver); ctx is the overall
// deadline parent (dev init passes 5m; deadline-less callers get adapter
// defaults), and pol binds the endpoint + pins TLS.
func Migrate(ctx context.Context, ownerDSN string, pol dbenv.EndpointPolicy) error {
	pool, err := dbenv.Pool(ctx, ownerDSN, pol)
	if err != nil {
		return fmt.Errorf("store: migrate connect: %w", err)
	}
	defer pool.Close()
	version, dirty, err := FrameworkVersion(ctx, pool)
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
	return migrateTo(ctx, ownerDSN, pol, nil)
}

// migrateTo migrates to a target version (nil means latest). Tests use it to
// stage databases at older versions; production always migrates to latest.
// The database driver is the bounded migratedb adapter, constructed here
// (never scheme-resolved stock), so every operation inherits deadlines,
// pinned TLS, and instrumentation.
func migrateTo(ctx context.Context, ownerDSN string, pol dbenv.EndpointPolicy, target *uint) error {
	src, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("store: migrate source: %w", err)
	}
	poolCfg, err := dbenv.PoolConfig(ownerDSN, pol)
	if err != nil {
		return err
	}
	sink, phases := migratedb.CaptureSink(12)
	drv, err := migratedb.Open(ctx, migratedb.Config{
		ConnConfig: *poolCfg.ConnConfig,
		Table:      MigrationsTable,
		Sink:       sink,
	})
	if err != nil {
		return fmt.Errorf("store: migrate driver: %w (phases: %s)", err, phases())
	}
	m, err := migrate.NewWithInstance("iofs", src, "migratedb", drv)
	if err != nil {
		_ = drv.Close()
		return fmt.Errorf("store: migrate instance: %w", err)
	}
	defer func() { _, _ = m.Close() }()
	if target == nil {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("%w (phases: %s)", diagnoseMigrateErr(m, "up", err), phases())
		}
	} else if err := m.Migrate(*target); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("%w (phases: %s)", diagnoseMigrateErr(m, fmt.Sprintf("to v%d", *target), err), phases())
	}
	if _, _, err := m.Version(); err != nil {
		return fmt.Errorf("store: migrate version: %w", err)
	}
	return nil
}

// FrameworkVersion reads the applied framework schema version. It
// reports version 0 (not an error) when the framework table does not
// exist yet — that database was never migrated.
func FrameworkVersion(ctx context.Context, pool *pgxpool.Pool) (uint, bool, error) {
	var version uint
	var dirty bool
	err := pool.QueryRow(ctx, `SELECT version, dirty FROM public.schema_migrations`).Scan(&version, &dirty)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
			return 0, false, nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("read framework schema version: %w", err)
	}
	return version, dirty, nil
}

// NOTE (prompt30): withMigrationsTable retired with the stock-driver path.
// migrateTo now constructs the bounded migratedb adapter in Go (table name
// via Config, no URL scheme mapping). The scheme-rewrite regression moved to
// dbenv.pgx5URL coverage, which still serves file-source callers in transition
// (removed entirely once ApplyFrameworkBaseline migrates below).
