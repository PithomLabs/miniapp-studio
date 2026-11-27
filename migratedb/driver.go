package migratedb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/jackc/pgx/v5/stdlib"
)

// Compile-time proof the adapter satisfies migrate's driver contract.
var _ database.Driver = (*Driver)(nil)

// NilVersion mirrors migrate's nil version for events outside any migration.
const NilVersion = -1

// Driver is a database.Driver with universal bounded execution. One
// dedicated *sql.Conn serves the whole migration lifecycle; every
// operation runs under an explicit child deadline derived from the
// base context plus configured caps. Methods are called sequentially
// by migrate; Close is safe to call twice.
type Driver struct {
	base   context.Context
	cfg    Config
	db     *sql.DB
	conn   *sql.Conn
	schema string
	table  string
	dbName string
	lockID string

	locked   atomic.Bool
	poisoned atomic.Bool
	closed   atomic.Bool
	version  atomic.Int64
}

// OpenURL satisfies database.Driver for scheme-resolved construction,
// which this adapter deliberately does not support: timeouts, sinks,
// and TLS pinning cannot be expressed through a URL string. Construct
// with Open(ctx, Config) and inject via migrate.NewWithInstance or
// migrate.NewWithDatabaseInstance, neither of which calls this method.
func (d *Driver) Open(url string) (database.Driver, error) {
	return nil, errors.New("migratedb: scheme-resolved Open unsupported; construct with migratedb.Open(ctx, Config)")
}

// Open validates config, dials through the pinned connector,
// reserves and health-checks the dedicated connection, resolves the
// version-table schema, and ensures the version table exists. The
// base context bounds Open itself and parents every later operation.
func Open(ctx context.Context, cfg Config) (*Driver, error) {
	cfg.withDefaults()
	if cfg.Table == "" {
		return nil, errors.New("migratedb: Config.Table required")
	}
	cc := cfg.ConnConfig
	if cc.Config.RuntimeParams == nil {
		cc.Config.RuntimeParams = map[string]string{}
	}
	cc.Config.RuntimeParams["application_name"] = appName
	// Complementary server-side backstops (client deadlines fire
	// first and attribute cleanly; these catch what they miss).
	op := cfg.OpTimeout
	cc.Config.RuntimeParams["statement_timeout"] = formatMs(2 * op)
	cc.Config.RuntimeParams["lock_timeout"] = formatMs(op)

	db := sql.OpenDB(stdlib.GetConnector(cc))
	d := &Driver{base: ctx, cfg: cfg, db: db, table: cfg.Table}
	d.version.Store(NilVersion)

	if err := d.withDeadline("open", NilVersion, op, func(ctx context.Context) error {
		conn, err := db.Conn(ctx)
		if err != nil {
			return err
		}
		d.conn = conn
		// From here on, every failure path must release the
		// reservation: database/sql cannot tear down a connection
		// still checked out, and leaking it stalls Close (observed:
		// 15s test hangs against a blackholed server).
		release := func() {
			_ = conn.Close()
			d.conn = nil
		}
		if err := conn.PingContext(ctx); err != nil {
			release()
			return err
		}
		var schema string
		if err := conn.QueryRowContext(ctx, `SELECT CURRENT_SCHEMA()`).Scan(&schema); err != nil {
			release()
			return err
		}
		if schema == "" {
			release()
			return errors.New("migratedb: empty schema")
		}
		d.schema = schema
		d.dbName = cc.Config.Database
		id, err := database.GenerateAdvisoryLockId(d.dbName, d.schema, d.table)
		if err != nil {
			release()
			return err
		}
		d.lockID = id
		return nil
	}); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := d.ensureVersionTable(); err != nil {
		_ = d.closeOnce(context.Background())
		return nil, err
	}
	return d, nil
}

// withDeadline runs fn under a child context bounded by d (which must
// already account for any parent deadline via base), emits the phase
// event, and poisons the driver on timeout/protocol failure.
func (d *Driver) withDeadline(phase string, version int, bound time.Duration, fn func(context.Context) error) error {
	if d.poisoned.Load() || d.closed.Load() {
		err := errors.New("migratedb: connection invalidated")
		d.cfg.Sink(Event{Version: version, Phase: phase, Deadline: bound, Err: err})
		return err
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(d.base, bound)
	defer cancel()
	err := fn(ctx)
	elapsed := time.Since(start)
	if err != nil && (isTimeout(err) || isConnFailure(err)) {
		d.poisoned.Store(true)
	}
	d.cfg.Sink(Event{Version: version, Phase: phase, Elapsed: elapsed, Deadline: bound, Err: err, Reusable: !d.poisoned.Load()})
	return err
}

func isTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}

// isConnFailure reports transport-level failures that taint the
// connection (as opposed to SQL errors, after which it stays usable).
// Conservative: unknown error shapes poison, because a suspect
// connection must never be trusted again.
func isConnFailure(err error) bool {
	if err == nil {
		return false
	}
	// No-rows is a normal control-flow outcome (empty version table),
	// never transport failure.
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	// database.Error does not implement Unwrap: look through it
	// explicitly or every wrapped SQL error poisons (observed live:
	// a 42601 body failure wrongly invalidated the connection).
	var dbErr *database.Error
	if errors.As(err, &dbErr) {
		return isConnFailure(dbErr.OrigErr)
	}
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		// Class 08 (connection exception) and 57P0x (operator
		// intervention) taint the connection; SQL errors do not.
		s := pgErr.SQLState()
		return strings.HasPrefix(s, "08") || strings.HasPrefix(s, "57P")
	}
	return true
}

// Lock acquires the migrator advisory lock with a bounded try-loop
// (pg_advisory_lock itself waits forever). Same lock id inputs as the
// stock driver, so mutual exclusion interoperates across versions.
func (d *Driver) Lock() error {
	deadline := time.Now().Add(d.cfg.LockTimeout)
	for {
		acquired := false
		err := d.withDeadline("lock", int(d.version.Load()), d.cfg.OpTimeout, func(ctx context.Context) error {
			e := d.conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock($1)`, d.lockID).Scan(&acquired)
			return e
		})
		if err != nil {
			return &database.Error{OrigErr: err, Err: "try lock failed"}
		}
		if acquired {
			d.locked.Store(true)
			return nil
		}
		if time.Until(deadline) <= 0 {
			err := fmt.Errorf("migratedb: lock acquisition timed out after %s", d.cfg.LockTimeout)
			d.cfg.Sink(Event{Version: int(d.version.Load()), Phase: "lock", Deadline: d.cfg.LockTimeout, Err: err})
			return &database.Error{OrigErr: err, Err: "try lock failed"}
		}
		select {
		case <-d.base.Done():
			return &database.Error{OrigErr: d.base.Err(), Err: "try lock failed"}
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// Unlock releases the advisory lock. Best effort under a bound; the
// Close path records (rather than hides) a failure to release.
func (d *Driver) Unlock() error {
	if !d.locked.Load() {
		return nil
	}
	err := d.withDeadline("unlock", int(d.version.Load()), d.cfg.OpTimeout, func(ctx context.Context) error {
		return d.conn.QueryRowContext(ctx, `SELECT pg_advisory_unlock($1)`, d.lockID).Scan(new(bool))
	})
	if err == nil {
		d.locked.Store(false)
		return nil
	}
	return &database.Error{OrigErr: err}
}

// Close releases the advisory lock on a separately bounded cleanup
// context (never the possibly-cancelled base), then closes the
// dedicated connection and pool. Safe to call twice; post-Close
// operations fail fast.
func (d *Driver) Close() error {
	return d.closeOnce(context.Background())
}

func (d *Driver) closeOnce(parent context.Context) error {
	if d.closed.Swap(true) {
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, d.cfg.CleanupTimeout)
	defer cancel()
	start := time.Now()
	var err error
	if d.locked.Load() && d.conn != nil {
		var ok bool
		if uerr := d.conn.QueryRowContext(ctx, `SELECT pg_advisory_unlock($1)`, d.lockID).Scan(&ok); uerr != nil {
			err = uerr
		} else {
			d.locked.Store(false)
		}
	}
	if d.conn != nil {
		if cerr := d.conn.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	if cerr := d.db.Close(); cerr != nil && err == nil {
		err = cerr
	}
	d.poisoned.Store(true)
	d.cfg.Sink(Event{Version: int(d.version.Load()), Phase: "close",
		Elapsed: time.Since(start), Deadline: d.cfg.CleanupTimeout, Err: err})
	return err
}

// quoteIdentifier mirrors the stock driver's lib/pq-derived quoting.
func quoteIdentifier(name string) string {
	if end := strings.IndexRune(name, 0); end > -1 {
		name = name[:end]
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func formatMs(d time.Duration) string {
	return fmt.Sprintf("%d", d.Milliseconds())
}
