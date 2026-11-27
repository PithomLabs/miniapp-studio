package dbenv

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// fakeRow returns canned scan values in order.
type fakeRow struct {
	vals []any
	err  error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.vals[i].(string)
		case *int:
			*d = r.vals[i].(int)
		case *bool:
			*d = r.vals[i].(bool)
		default:
			panic("fakeRow: unhandled type")
		}
	}
	return nil
}

// fakeRows returns canned rows for the extension query. The embedded
// pgx.Rows satisfies the rest of the interface; production code only
// touches Next/Scan/Err/Close.
type fakeRows struct {
	pgx.Rows
	rows [][]any
	i    int
}

func (r *fakeRows) Next() bool { return r.i < len(r.rows) }
func (r *fakeRows) Scan(dest ...any) error {
	for i := range dest {
		*(dest[i].(*string)) = r.rows[r.i][i].(string)
	}
	r.i++
	return nil
}
func (r *fakeRows) Err() error { return nil }
func (r *fakeRows) Close()     {}

type fakeQuerier struct {
	version    string
	versionNum int
	enc        string
	collate    string
	ctype      string
	tz         string
	exts       [][]any
}

func (f fakeQuerier) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	switch {
	case strings.Contains(sql, "version()"):
		return fakeRow{vals: []any{f.version}}
	case strings.Contains(sql, "server_version_num"):
		return fakeRow{vals: []any{f.versionNum}}
	case strings.Contains(sql, "pg_encoding_to_char"):
		return fakeRow{vals: []any{f.enc, f.collate, f.ctype}}
	default:
		return fakeRow{vals: []any{f.tz}}
	}
}

func (f fakeQuerier) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return &fakeRows{rows: f.exts}, nil
}

func TestCollectServerInfo(t *testing.T) {
	fq := fakeQuerier{
		version: "PostgreSQL 18.0", versionNum: 180000,
		enc: "UTF8", collate: "C", ctype: "C", tz: "Etc/UTC",
		exts: [][]any{{"plpgsql", "1.0"}},
	}
	info, err := CollectServerInfo(context.Background(), fq)
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "PostgreSQL 18.0" || info.VersionNum != 180000 {
		t.Errorf("version = %+v", info)
	}
	if info.Encoding != "UTF8" || info.Collate != "C" || info.Ctype != "C" || info.Timezone != "Etc/UTC" {
		t.Errorf("locale = %+v", info)
	}
	if len(info.Extensions) != 1 || info.Extensions[0].Name != "plpgsql" {
		t.Errorf("extensions = %+v", info.Extensions)
	}
}

func TestCollectServerInfoEmptyExtensions(t *testing.T) {
	fq := fakeQuerier{version: "v", versionNum: 190000}
	info, err := CollectServerInfo(context.Background(), fq)
	if err != nil {
		t.Fatal(err)
	}
	if info.Extensions == nil {
		t.Error("extensions must be non-nil (JSON [])")
	}
}

func TestStaleRunDBs(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	dbs := []RunDB{
		{Name: "studio_test_run_1_1799990000_ab", CreatedUnix: 1799990000, Backends: 0}, // old, idle
		{Name: "studio_test_run_1_1799999900_ab", CreatedUnix: 1799999900, Backends: 0}, // young
		{Name: "studio_test_run_1_1799990000_cd", CreatedUnix: 1799990000, Backends: 2}, // old, busy
	}
	got := StaleRunDBs(dbs, now, 30*time.Minute)
	if len(got) != 1 || got[0] != "studio_test_run_1_1799990000_ab" {
		t.Errorf("stale = %v, want only the old idle one", got)
	}
	got = StaleRunDBs(dbs, now, 0)
	if len(got) != 2 {
		t.Errorf("zero threshold should qualify both idle DBs: %v", got)
	}
	for _, name := range got {
		if strings.HasSuffix(name, "_cd") {
			t.Errorf("zero threshold must still spare busy DBs: %v", got)
		}
	}
}

func TestFrameworkModuleDir(t *testing.T) {
	dir, err := FrameworkModuleDir(context.Background(), ".")
	if err != nil {
		t.Skipf("module resolution unavailable: %v", err)
	}
	if dir == "" {
		t.Error("empty module dir")
	}
}
