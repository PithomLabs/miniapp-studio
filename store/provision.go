package store

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var rxHexPassword = regexp.MustCompile(`^[0-9a-f]{32,}$`)

// EnsureAppRole creates the application role when missing and sets its
// password in all cases. Production callers pass an owner-generated secret
// from a file; tests pass per-run random values that are never emitted.
// The role can log in but can never bypass RLS. Passwords must be lowercase
// hex (generate with `openssl rand -hex 32`) so role DDL stays injectable-free.
func EnsureAppRole(ctx context.Context, owner *pgxpool.Pool, password string) error {
	if !rxHexPassword.MatchString(password) {
		return fmt.Errorf("store: application role password must be lowercase hex, 32+ chars")
	}
	tx, err := owner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: provision begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var exists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, AppRoleName,
	).Scan(&exists); err != nil {
		return fmt.Errorf("store: provision role check: %w", err)
	}
	if !exists {
		if _, err := tx.Exec(ctx, `CREATE ROLE `+AppRoleName+` WITH LOGIN NOBYPASSRLS`); err != nil {
			return fmt.Errorf("store: provision create role: %w", err)
		}
		var dbname string
		if err := tx.QueryRow(ctx, `SELECT current_database()`).Scan(&dbname); err != nil {
			return fmt.Errorf("store: provision database: %w", err)
		}
		grant := `GRANT CONNECT ON DATABASE ` + pgx.Identifier{dbname}.Sanitize() + ` TO ` + AppRoleName
		if _, err := tx.Exec(ctx, grant); err != nil {
			return fmt.Errorf("store: provision connect: %w", err)
		}
	}
	// Hex-validated above, safe to interpolate: DDL takes no bind parameters.
	if _, err := tx.Exec(ctx, `ALTER ROLE `+AppRoleName+` WITH PASSWORD '`+password+`'`); err != nil {
		return fmt.Errorf("store: provision password: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: provision commit: %w", err)
	}
	return nil
}

// UpsertOperator provisions or updates one (tenant, binding) operator row.
// It runs on an owner connection; the SECURITY DEFINER function constrains
// the write to the operators table.
func UpsertOperator(ctx context.Context, owner *pgxpool.Pool, tenantID, bindingID string, active bool) error {
	if tenantID == "" || bindingID == "" {
		return fmt.Errorf("store: tenant and binding are required")
	}
	if _, err := owner.Exec(ctx, `SELECT studio.operator_upsert($1, $2, $3)`,
		tenantID, bindingID, active); err != nil {
		return fmt.Errorf("store: upsert operator: %w", err)
	}
	return nil
}
