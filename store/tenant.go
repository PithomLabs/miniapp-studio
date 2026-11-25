package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InTenantTx runs fn inside a transaction with the tenant context armed.
// RLS policies read app.tenant_id from this transaction-local setting.
//
// INVIOLABLE RULE: SET LOCAL inside this helper is the only supported way to
// establish tenant context. Never issue session-level SET app.tenant_id in
// production code: pgxpool does not reset session state by default, and a
// forgotten setting would leak across checkouts. TenantResetHook destroys
// connections it cannot sanitize, but it is a backstop, not a license.
func InTenantTx(ctx context.Context, pool *pgxpool.Pool, tenantID string, fn func(pgx.Tx) error) error {
	if tenantID == "" {
		return fmt.Errorf("store: tenant is required")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, tenantID); err != nil {
		return fmt.Errorf("store: arm tenant: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: commit: %w", err)
	}
	return nil
}

// TenantOf reports the tenant context armed on a transaction, or "" when the
// transaction carries none. Tests use it to prove context arming; production
// paths never branch on it.
func TenantOf(ctx context.Context, tx pgx.Tx) string {
	var tenant string
	if err := tx.QueryRow(ctx, `SELECT current_setting('app.tenant_id', true)`).Scan(&tenant); err != nil {
		return ""
	}
	return tenant
}
