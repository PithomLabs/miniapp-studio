package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantResetHook returns a pgxpool AfterRelease hook that clears
// transaction-external tenant state before a connection rejoins the idle
// pool. A connection that cannot be sanitized is destroyed, never reused.
//
// Mechanism rationale (pgx v5.13.0): AfterRelease runs before the connection
// returns to idle, so the pool invariant is observable — idle connections
// are always clean — and a false result cannot fail any user query (the
// borrower already released). PrepareConn was considered and rejected: its
// richer semantics add nothing here, and its failure modes surface inside
// user queries. BeforeAcquire is deprecated upstream and excluded.
// AfterRelease provides no context; RESET is a trivial local statement, so
// a background context is correct, and any error destroys the connection.
//
// This hook is narrow by design: it resets exactly app.tenant_id. It does
// not DISCARD ALL and must never grow into a general sanitizer —
// transaction-scoped SET LOCAL (see InTenantTx) remains the only supported
// way to establish tenant context.
func TenantResetHook() func(*pgx.Conn) bool {
	return func(c *pgx.Conn) bool {
		_, err := c.Exec(context.Background(), `RESET app.tenant_id`)
		return err == nil
	}
}

// NewPool opens a pool with tenant-hygiene protection. maxConns bounds the
// pool; values <= 0 select a small default. All Studio-constructed pools —
// server and tests — must come through here (or set TenantResetHook
// explicitly) so no checkout can inherit another request's tenant.
func NewPool(ctx context.Context, dsn string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("store: parse DSN: %w", err)
	}
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	} else {
		cfg.MaxConns = 4
	}
	cfg.AfterRelease = TenantResetHook()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: pool: %w", err)
	}
	return pool, nil
}
