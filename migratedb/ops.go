package migratedb

import (
	"context"
	"database/sql"
	"errors"
	"io"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/jackc/pgx/v5/pgconn"
)

// sqlStateUndefinedTable mirrors store's string-literal convention.
const sqlStateUndefinedTable = "42P01"

// Run executes a migration body on the dedicated connection. The body
// travels as ONE statement string and the SERVER splits it — exactly
// like the stock driver default (x-multi-statement off). Client-side
// splitting is banned: migrate's splitter mangled a real framework
// migration (dollar-quoted bodies split mid-statement → 42601 on live
// Neon), while server-side parsing handles every construct. Any
// failure aborts the migration like stock; SQL errors do not poison.
func (d *Driver) Run(migration io.Reader) error {
	return d.withDeadline("run", int(d.version.Load()), d.cfg.OpTimeout, func(ctx context.Context) error {
		body, err := readWholeBody(migration)
		if err != nil {
			return &database.Error{OrigErr: err}
		}
		if _, err := d.conn.ExecContext(ctx, string(body)); err != nil {
			return &database.Error{OrigErr: err}
		}
		return nil
	})
}

// readWholeBody returns the migration untouched for single-Exec
// delivery. It exists as a named function so tests pin the
// no-client-side-splitting decision against future cleverness.
func readWholeBody(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

// SetVersion records version+dirty in a single transaction sharing ONE
// child context for Begin, statements, and Commit/Rollback, so the
// whole marker write is bounded as a unit (the original hang was an
// unbounded BeginTx here). Same TRUNCATE+INSERT protocol as stock,
// including the issue-330 nil-version rewrite rule.
func (d *Driver) SetVersion(version int, dirty bool) error {
	phase := "mark_clean"
	if dirty {
		phase = "mark_dirty"
	}
	err := d.withDeadline(phase, version, d.cfg.OpTimeout, func(ctx context.Context) error {
		tx, err := d.conn.BeginTx(ctx, nil)
		if err != nil {
			return &database.Error{OrigErr: err, Err: "transaction start failed"}
		}
		query := `TRUNCATE ` + quoteIdentifier(d.schema) + `.` + quoteIdentifier(d.table)
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return rollbackJoin(tx, &database.Error{OrigErr: err, Query: []byte(query)})
		}
		if version >= 0 || (version == database.NilVersion && dirty) {
			query = `INSERT INTO ` + quoteIdentifier(d.schema) + `.` + quoteIdentifier(d.table) + ` (version, dirty) VALUES ($1, $2)`
			if _, err := tx.ExecContext(ctx, query, version, dirty); err != nil {
				return rollbackJoin(tx, &database.Error{OrigErr: err, Query: []byte(query)})
			}
		}
		if err := tx.Commit(); err != nil {
			return &database.Error{OrigErr: err, Err: "transaction commit failed"}
		}
		return nil
	})
	if err == nil {
		d.version.Store(int64(version))
	}
	return err
}

func rollbackJoin(tx *sql.Tx, err error) error {
	if rerr := tx.Rollback(); rerr != nil {
		return errors.Join(err, rerr)
	}
	return err
}

// Version reads the current version marker. Nil-version and
// missing-table map exactly like stock, as successful outcomes (an
// empty version table is normal on fresh databases, not a failure,
// and must neither poison the connection nor emit error events).
func (d *Driver) Version() (version int, dirty bool, err error) {
	query := `SELECT version, dirty FROM ` + quoteIdentifier(d.schema) + `.` + quoteIdentifier(d.table) + ` LIMIT 1`
	var empty, missing bool
	if werr := d.withDeadline("version", int(d.version.Load()), d.cfg.OpTimeout, func(ctx context.Context) error {
		err := d.conn.QueryRowContext(ctx, query).Scan(&version, &dirty)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, sql.ErrNoRows):
			empty = true
			return nil
		default:
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == sqlStateUndefinedTable {
				missing = true
				return nil
			}
			return err
		}
	}); werr != nil {
		return 0, false, &database.Error{OrigErr: werr, Query: []byte(query)}
	}
	if empty || missing {
		return database.NilVersion, false, nil
	}
	d.version.Store(int64(version))
	return version, dirty, nil
}

// Drop removes all tables in the current schema, mirroring stock.
// Each statement is bounded; the driver is left poisoned afterwards
// only if a transport failure occurred.
func (d *Driver) Drop() (err error) {
	return d.withDeadline("drop", int(d.version.Load()), d.cfg.OpTimeout, func(ctx context.Context) error {
		query := `SELECT table_name FROM information_schema.tables WHERE table_schema=(SELECT current_schema()) AND table_type='BASE TABLE'`
		rows, err := d.conn.QueryContext(ctx, query)
		if err != nil {
			return &database.Error{OrigErr: err, Query: []byte(query)}
		}
		defer func() {
			if cerr := rows.Close(); cerr != nil {
				err = errors.Join(err, cerr)
			}
		}()
		var tables []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return err
			}
			if name != "" {
				tables = append(tables, name)
			}
		}
		if err := rows.Err(); err != nil {
			return &database.Error{OrigErr: err, Query: []byte(query)}
		}
		for _, t := range tables {
			query = `DROP TABLE IF EXISTS ` + quoteIdentifier(t) + ` CASCADE`
			if _, err := d.conn.ExecContext(ctx, query); err != nil {
				return &database.Error{OrigErr: err, Query: []byte(query)}
			}
		}
		return nil
	})
}

// ensureVersionTable creates the version table when absent, holding
// the advisory lock like stock (deviation from caller-locks noted in
// the original source is preserved).
func (d *Driver) ensureVersionTable() error {
	if err := d.Lock(); err != nil {
		return err
	}
	defer func() {
		_ = d.Unlock()
	}()
	query := `SELECT COUNT(1) FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2 LIMIT 1`
	var err error
	werr := d.withDeadline("ensure_table", NilVersion, d.cfg.OpTimeout, func(ctx context.Context) error {
		var count int
		rowErr := d.conn.QueryRowContext(ctx, query, d.schema, d.table).Scan(&count)
		if rowErr != nil {
			err = &database.Error{OrigErr: rowErr, Query: []byte(query)}
			return err
		}
		if count == 1 {
			return nil
		}
		create := `CREATE TABLE IF NOT EXISTS ` + quoteIdentifier(d.schema) + `.` + quoteIdentifier(d.table) + ` (version bigint not null primary key, dirty boolean not null)`
		if _, rowErr = d.conn.ExecContext(ctx, create); rowErr != nil {
			err = &database.Error{OrigErr: rowErr, Query: []byte(create)}
			return err
		}
		return nil
	})
	if werr != nil {
		return werr
	}
	return err
}
