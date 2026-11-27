// Package migratedb is a bounded-execution database.Driver for
// golang-migrate, built for Neon-backed Studio migrations.
//
// Structural problem it solves: migrate's database.Driver interface
// takes no context anywhere, and the stock pgx driver uses
// context.Background for every database call except Run bodies under
// x-statement-timeout. A stalled SetVersion BeginTx therefore hangs
// forever (proven live: TestRLSEnable000016, 10m go-test panic).
// Timeouts cannot flow through the interface, so they live INSIDE this
// driver, configured at construction:
//
//   - Open takes the caller's context as the BASE context. Every
//     operation derives explicit child deadlines from it, so an
//     overall deadline (dev init's 5m, a test timeout) genuinely
//     bounds migration work — verified by test, not claimed.
//   - One dedicated *sql.Conn serves the whole migration lifecycle
//     (advisory lock is session-held by design). It is Ping-checked
//     at Open and invalidated on timeout/protocol failure; a
//     poisoned connection is never reused.
//   - Advisory-lock SQL, version-table protocol, dirty-marker
//     semantics, and statement splitting mirror the stock driver
//     exactly (same lock id inputs, same multistmt parser) —
//     bounded execution without behavior drift.
//
// What this does NOT promise: instant cancellation of in-flight
// server work. Cancellation cooperates at operation boundaries; every
// operation is deadline-bounded, so the worst case is one op timeout,
// never infinity.
package migratedb

import (
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	// DefaultOpTimeout bounds any single migration database operation
	// when the caller supplies no tighter deadline.
	DefaultOpTimeout = 60 * time.Second
	// DefaultLockTimeout mirrors migrate.DefaultLockTimeout for
	// advisory acquisition (implemented as a bounded try-loop here,
	// since pg_advisory_lock itself waits forever).
	DefaultLockTimeout = 15 * time.Second
	// DefaultCleanupTimeout bounds the Close path (unlock attempt +
	// connection teardown), which must never depend on an
	// already-cancelled operation context.
	DefaultCleanupTimeout = 10 * time.Second
	// appName identifies migration sessions in pg_stat_activity.
	appName = "studio-migrator"
)

// Config constructs the driver. ConnConfig carries the pinned TLS
// configuration (callers pass dbenv-derived config); Table names the
// version table; timeouts bound every phase (non-positive values take
// the defaults above). Sink receives structured phase events; nil
// selects the stdlib-slog default. None of these fields may carry
// credentials beyond what ConnConfig already holds for dialing, and
// the driver never logs the config whole.
type Config struct {
	ConnConfig     pgx.ConnConfig
	Table          string
	OpTimeout      time.Duration
	LockTimeout    time.Duration
	CleanupTimeout time.Duration
	Sink           EventSink
}

func (c *Config) withDefaults() {
	if c.OpTimeout <= 0 {
		c.OpTimeout = DefaultOpTimeout
	}
	if c.LockTimeout <= 0 {
		c.LockTimeout = DefaultLockTimeout
	}
	if c.CleanupTimeout <= 0 {
		c.CleanupTimeout = DefaultCleanupTimeout
	}
	if c.Sink == nil {
		c.Sink = SlogSink(nil)
	}
}
