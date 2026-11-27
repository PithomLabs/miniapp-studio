package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// tenantResetTimeout bounds the AfterRelease RESET. The statement is
// trivial and local; anything slower is a dead transport masquerading
// as a live connection, and waiting on it once hung pool teardown and
// acquisition across two live packages (prompt30 S7). Failure destroys
// the connection — never waits, never reuses. A var (not const) so
// tests can shorten it; production must never reassign it.
var tenantResetTimeout = 10 * time.Second

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
// AfterRelease provides no context, so the hook derives its own short
// deadline (an unbounded RESET hung teardown live); any error or timeout
// destroys the connection, and the failure is logged without secrets.
//
// This hook is narrow by design: it resets exactly app.tenant_id. It does
// not DISCARD ALL and must never grow into a general sanitizer —
// transaction-scoped SET LOCAL (see InTenantTx) remains the only supported
// way to establish tenant context.
func TenantResetHook() func(*pgx.Conn) bool {
	return func(c *pgx.Conn) bool {
		ctx, cancel := context.WithTimeout(context.Background(), tenantResetTimeout)
		defer cancel()
		if _, err := c.Exec(ctx, `RESET app.tenant_id`); err != nil {
			slog.Warn("tenant reset failed; destroying connection",
				"timeout", errors.Is(err, context.DeadlineExceeded))
			return false
		}
		return true
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
