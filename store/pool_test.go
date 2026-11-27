package store

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TestPoolResetsTenantSetting proves F7: a session-level tenant setting left
// on a released connection can never leak into a later checkout, and the
// transaction helper still works without leaking after commit.
func TestPoolResetsTenantSetting(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	insertAppRow(t, env.owner, "tA", "app-a", "a")

	pool, err := NewPool(ctx, env.appDSN, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	// Contaminate one checkout, then release it without cleanup.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `SELECT set_config('app.tenant_id', 'tA', false)`); err != nil {
		t.Fatal(err)
	}
	conn.Release()

	// Later checkouts must see nothing without explicit tenant context.
	for i := 0; i < 4; i++ {
		clean, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var n int
		err = clean.QueryRow(ctx, `SELECT count(*) FROM studio.apps`).Scan(&n)
		clean.Release()
		if err != nil {
			t.Fatalf("bare read: %v", err)
		}
		if n != 0 {
			t.Fatalf("checkout %d saw %d rows: tenant setting leaked across release", i, n)
		}
	}

	// The helper path keeps working and leaves nothing behind.
	var seen int
	if err := InTenantTx(ctx, pool, "tA", func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM studio.apps`).Scan(&seen)
	}); err != nil {
		t.Fatalf("InTenantTx: %v", err)
	}
	if seen != 1 {
		t.Errorf("helper saw %d rows, want 1", seen)
	}
	clean, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer clean.Release()
	var n int
	if err := clean.QueryRow(ctx, `SELECT count(*) FROM studio.apps`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("post-helper checkout saw %d rows, want 0", n)
	}
}

// TestTenantResetHookTimesOut proves the prompt30 fix directly: against a
// peer that completes the handshake and then never answers, the hook must
// return false at its deadline instead of parking forever. Unit-only (an
// in-memory pgwire peer); no database needed.
func TestTenantResetHookTimesOut(t *testing.T) {
	old := tenantResetTimeout
	tenantResetTimeout = 200 * time.Millisecond
	defer func() { tenantResetTimeout = old }()

	server, client := net.Pipe()
	defer server.Close()
	go func() {
		defer server.Close()
		var ln [4]byte
		if _, err := io.ReadFull(server, ln[:]); err != nil {
			return
		}
		rest := make([]byte, binary.BigEndian.Uint32(ln[:])-4)
		if _, err := io.ReadFull(server, rest); err != nil {
			return
		}
		msg := func(typ byte, payload []byte) {
			buf := []byte{typ, 0, 0, 0, 0}
			binary.BigEndian.PutUint32(buf[1:], uint32(len(payload)+4))
			_, _ = server.Write(append(buf, payload...))
		}
		u32 := func(v uint32) []byte {
			b := make([]byte, 4)
			binary.BigEndian.PutUint32(b, v)
			return b
		}
		msg('R', u32(0)) // AuthOk
		msg('S', append(append([]byte("server_version"), 0), append([]byte("18.0"), 0)...))
		msg('K', append(u32(4242), u32(424242)...)) // cancel key, so client cancel works
		_, _ = server.Write([]byte{'Z', 0, 0, 0, 5, 'I'})
		_, _ = io.Copy(io.Discard, server) // blackhole: read, never reply
	}()

	cfg, err := pgx.ParseConfig("postgres://u:x@invalid/db?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	cfg.DialFunc = func(context.Context, string, string) (net.Conn, error) { return client, nil }
	// pgconn resolves before dialing; short-circuit with a dummy address
	// (DialFunc ignores it and returns the pipe).
	cfg.LookupFunc = func(context.Context, string) ([]string, error) { return []string{"127.0.0.1:5432"}, nil }
	conn, err := pgx.ConnectConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	start := time.Now()
	if ok := TenantResetHook()(conn); ok {
		t.Error("hook returned true against a silent server")
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("hook took %s, want bounded near deadline", elapsed)
	}
}

// TestPoolDiscardsUnresettable proves reset failure destroys instead of
// reusing. With MaxConns 1, reusing the broken connection would fail the
// re-acquire; success proves transparent replacement.
func TestPoolDiscardsUnresettable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	pool, err := NewPool(ctx, env.appDSN, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Break the connection underneath the pool, then release: reset must
	// fail and the pool must destroy rather than reuse.
	_ = conn.Conn().Close(context.Background())
	conn.Release()
	fresh, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("re-acquire: %v", err)
	}
	defer fresh.Release()
	var one int
	if err := fresh.QueryRow(ctx, `SELECT 1`).Scan(&one); err != nil || one != 1 {
		t.Fatalf("fresh checkout broken: %v", err)
	}
}
