package store

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LatestVersion returns the highest migration version embedded in this
// binary. The server refuses to start (F12) unless the database matches it.
func LatestVersion() (uint, error) {
	entries, err := fs.Glob(migrationFiles, "migrations/*.up.sql")
	if err != nil {
		return 0, fmt.Errorf("store: list migrations: %w", err)
	}
	var latest uint
	for _, name := range entries {
		base := strings.TrimSuffix(strings.TrimPrefix(name, "migrations/"), ".up.sql")
		num := base
		if i := strings.IndexByte(base, '_'); i >= 0 {
			num = base[:i]
		}
		v, err := strconv.ParseUint(num, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("store: migration %q has no numeric prefix: %w", name, err)
		}
		if uint(v) > latest {
			latest = uint(v)
		}
	}
	if latest == 0 {
		return 0, fmt.Errorf("store: no embedded migrations found")
	}
	return latest, nil
}

// CurrentVersion reads the database's Studio schema version. It reports
// version 0 (not an error) when the migrations table does not exist yet —
// that database was never migrated.
func CurrentVersion(ctx context.Context, pool *pgxpool.Pool) (version uint, dirty bool, err error) {
	err = pool.QueryRow(ctx,
		`SELECT version, dirty FROM `+MigrationsTable,
	).Scan(&version, &dirty)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
			return 0, false, nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("store: schema version: %w", err)
	}
	return version, dirty, nil
}
