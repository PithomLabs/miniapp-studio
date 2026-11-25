package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp-studio/store"
)

// scratchDatabase creates an empty database and returns a DSN for it. The
// name is always a test constant, never operator input. The database is
// dropped when the test finishes.
func scratchDatabase(t *testing.T, name string) string {
	t.Helper()
	ctx := context.Background()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	admin, err := store.NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Exec(ctx, `DROP DATABASE IF EXISTS "`+name+`"`); err != nil {
		t.Fatalf("drop scratch: %v", err)
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE "`+name+`"`); err != nil {
		t.Fatalf("create scratch: %v", err)
	}
	t.Cleanup(func() {
		p, err := store.NewPool(ctx, dsn, 0)
		if err != nil {
			return
		}
		defer p.Close()
		_, _ = p.Exec(ctx, `DROP DATABASE IF EXISTS "`+name+`" WITH (FORCE)`)
	})
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	return u.String()
}

// F12: New refuses a database that never ran Studio migrations.
func TestNewRefusesUnmigratedDatabase(t *testing.T) {
	scratch := scratchDatabase(t, "studio_gate")
	_, err := New(context.Background(), Config{DatabaseURL: scratch}, Options{})
	if err == nil || !strings.Contains(err.Error(), "schema at v0") {
		t.Fatalf("err = %v, want schema-at-v0 refusal", err)
	}
}

// F12: readyz reports 503 against an unmigrated database — and the full
// server (whose New runs the same gate) builds fine once it is migrated.
func TestReadyzSchemaGate(t *testing.T) {
	scratch := scratchDatabase(t, "studio_gate_readyz")
	ctx := context.Background()
	pool, err := store.NewPool(ctx, scratch, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	bare := &Server{pool: pool, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	rec := httptest.NewRecorder()
	bare.handleReadyz(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unmigrated readyz = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "unavailable") {
		t.Errorf("unmigrated readyz body = %q", rec.Body.String())
	}
	srv, _ := testServerOn(t, scratch, nil, nil)
	got := doRequest(t, srv, http.MethodGet, "/readyz", "")
	if got.Code != http.StatusOK || !strings.Contains(got.Body.String(), "ready") {
		t.Errorf("migrated readyz = %d %q, want 200 ready", got.Code, got.Body.String())
	}
}

// F12: a dirty version row (failed migration) also reports unready, on a
// scratch database so the shared test database is never touched.
func TestReadyzDirtySchema(t *testing.T) {
	scratch := scratchDatabase(t, "studio_gate_dirty")
	srv, _ := testServerOn(t, scratch, nil, nil)
	ctx := context.Background()
	admin, err := store.NewPool(ctx, scratch, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Exec(ctx, `UPDATE studio_schema_migrations SET dirty = true`); err != nil {
		t.Fatalf("mark dirty: %v", err)
	}
	got := doRequest(t, srv, http.MethodGet, "/readyz", "")
	if got.Code != http.StatusServiceUnavailable {
		t.Errorf("dirty readyz = %d, want 503", got.Code)
	}
}
