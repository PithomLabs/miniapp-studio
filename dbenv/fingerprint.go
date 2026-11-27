package dbenv

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Querier is the minimal read surface the fingerprint collectors
// need. *pgxpool.Pool satisfies it; tests supply fakes. No writes
// are possible through this interface by construction.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Extension pins one installed extension by name and version.
type Extension struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerInfo is the database half of an environment fingerprint. It
// carries server identity and configuration only — never credentials,
// never row data.
type ServerInfo struct {
	Version    string      `json:"version"`
	VersionNum int         `json:"version_num"`
	Encoding   string      `json:"encoding"`
	Collate    string      `json:"collate"`
	Ctype      string      `json:"ctype"`
	Timezone   string      `json:"timezone"`
	Extensions []Extension `json:"extensions"`
}

// CollectServerInfo reads server identity, database encoding/locale,
// timezone, and the installed extension list. Every value comes from
// the live server; nothing is assumed from configuration files.
func CollectServerInfo(ctx context.Context, q Querier) (ServerInfo, error) {
	var info ServerInfo
	if err := q.QueryRow(ctx, `SELECT version()`).Scan(&info.Version); err != nil {
		return ServerInfo{}, fmt.Errorf("dbenv: server version: %w", err)
	}
	if err := q.QueryRow(ctx, `SELECT current_setting('server_version_num')::int`).Scan(&info.VersionNum); err != nil {
		return ServerInfo{}, fmt.Errorf("dbenv: server version number: %w", err)
	}
	if err := q.QueryRow(ctx, `SELECT pg_encoding_to_char(encoding), datcollate, datctype
		FROM pg_database WHERE datname = current_database()`).Scan(&info.Encoding, &info.Collate, &info.Ctype); err != nil {
		return ServerInfo{}, fmt.Errorf("dbenv: database encoding: %w", err)
	}
	if err := q.QueryRow(ctx, `SHOW timezone`).Scan(&info.Timezone); err != nil {
		return ServerInfo{}, fmt.Errorf("dbenv: timezone: %w", err)
	}
	rows, err := q.Query(ctx, `SELECT extname, extversion FROM pg_extension ORDER BY 1`)
	if err != nil {
		return ServerInfo{}, fmt.Errorf("dbenv: extensions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var e Extension
		if err := rows.Scan(&e.Name, &e.Version); err != nil {
			return ServerInfo{}, fmt.Errorf("dbenv: extension row: %w", err)
		}
		info.Extensions = append(info.Extensions, e)
	}
	if err := rows.Err(); err != nil {
		return ServerInfo{}, fmt.Errorf("dbenv: extension rows: %w", err)
	}
	if info.Extensions == nil {
		info.Extensions = []Extension{}
	}
	return info, nil
}
