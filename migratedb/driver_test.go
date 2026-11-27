package migratedb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
)

// mustReturn fails the test unless fn returns within the watchdog
// bound. A hang regression must fail in seconds, never stall the
// suite until go test's own timeout.
func mustReturn(t *testing.T, bound time.Duration, fn func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		return err
	case <-time.After(bound):
		t.Fatalf("operation did not return within %s (hang regression)", bound)
		return nil
	}
}

type eventLog struct {
	mu     sync.Mutex
	events []Event
}

func (l *eventLog) sink(e Event) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, e)
}

func testConnConfig(t *testing.T, addr string) pgx.ConnConfig {
	t.Helper()
	// Plaintext local stub (TLS pinning is asserted separately on
	// construction; the combination is proven on live Neon). Built via
	// ParseConfig because stdlib.GetConnector requires it — same as the
	// production path through dbenv.PoolConfig.
	cfg, err := pgx.ParseConfig("postgres://stub@" + addr + "/stubdb?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	return *cfg
}

func testConfig(t *testing.T, addr string, log *eventLog) Config {
	return Config{
		ConnConfig:     testConnConfig(t, addr),
		Table:          "stub_migrations",
		OpTimeout:      300 * time.Millisecond,
		LockTimeout:    600 * time.Millisecond,
		CleanupTimeout: 300 * time.Millisecond,
		Sink:           log.sink,
	}
}

// openDriver manually wires a Driver over the stub WITHOUT calling
// Open: db.Conn is lazy (no I/O), so this works even though the stub
// never answers. Used to exercise post-Open operations against a
// stalled transport — the faithful shape of the production hang,
// where the connection died mid-run.
func openDriver(t *testing.T, addr string, log *eventLog) *Driver {
	t.Helper()
	cfg := testConfig(t, addr, log)
	db := sql.OpenDB(stdlib.GetConnector(cfg.ConnConfig))
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	lockID, err := database.GenerateAdvisoryLockId("stubdb", "public", cfg.Table)
	if err != nil {
		t.Fatal(err)
	}
	return &Driver{base: context.Background(), cfg: cfg, db: db, conn: conn,
		schema: "public", table: cfg.Table, dbName: "stubdb", lockID: lockID}
}

func TestOpenBlackholeBounded(t *testing.T) {
	stub := newPGStub(t, modeBlackhole)
	var log eventLog
	start := time.Now()
	err := mustReturn(t, 30*time.Second, func() error {
		_, err := Open(context.Background(), testConfig(t, stub.addr(), &log))
		return err
	})
	if err == nil {
		t.Fatal("Open against a blackholed server succeeded")
	}
	if elapsed := time.Since(start); elapsed > 15*time.Second {
		t.Errorf("Open took %s, want bounded near op timeout", elapsed)
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, driver.ErrBadConn) {
		t.Errorf("Open error = %v, want deadline or bad-connection", err)
	}
}

func TestOpenErrorPropagates(t *testing.T) {
	stub := newPGStub(t, modeError)
	var log eventLog
	start := time.Now()
	err := mustReturn(t, 30*time.Second, func() error {
		_, err := Open(context.Background(), testConfig(t, stub.addr(), &log))
		return err
	})
	if err == nil {
		t.Fatal("Open against an erroring server succeeded")
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("Open error took %s, want fast failure", elapsed)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Open error = %v, want fast SQL error not deadline", err)
	}
}

func TestSetVersionBeginBounded(t *testing.T) {
	// Deterministic repro of the production hang: SetVersion's
	// transaction-begin against a transport that never answers must
	// return within its deadline and invalidate the connection.
	stub := newPGStub(t, modeBlackhole)
	var log eventLog
	d := openDriver(t, stub.addr(), &log)
	start := time.Now()
	err := mustReturn(t, 30*time.Second, func() error { return d.SetVersion(17, false) })
	if err == nil {
		t.Fatal("SetVersion against blackhole succeeded")
	}
	if elapsed := time.Since(start); elapsed > 15*time.Second {
		t.Errorf("SetVersion took %s, want bounded near op timeout", elapsed)
	}
	if !d.poisoned.Load() {
		t.Error("timed-out connection not poisoned")
	}
	if _, _, err := d.Version(); err == nil {
		t.Error("operation after poisoning succeeded")
	} else if got := err.Error(); !strings.Contains(got, "invalidated") {
		t.Errorf("post-poison error = %q, want invalidation refusal", got)
	}
	_ = d.Close()
}

func TestLockBounded(t *testing.T) {
	stub := newPGStub(t, modeBlackhole)
	var log eventLog
	d := openDriver(t, stub.addr(), &log)
	start := time.Now()
	err := mustReturn(t, 30*time.Second, func() error { return d.Lock() })
	if err == nil {
		t.Fatal("Lock against blackhole succeeded")
	}
	// Each try dies at the op bound; the loop must still respect the
	// overall lock timeout instead of retrying forever.
	if elapsed := time.Since(start); elapsed > 15*time.Second {
		t.Errorf("Lock took %s, want bounded near lock timeout", elapsed)
	}
	_ = d.Close()
}

func TestVersionRunDropBounded(t *testing.T) {
	stub := newPGStub(t, modeBlackhole)
	var log eventLog
	d := openDriver(t, stub.addr(), &log)
	if _, _, err := d.Version(); err == nil {
		t.Error("Version against blackhole succeeded")
	}
	if err := d.Run(strings.NewReader("SELECT 1")); err == nil {
		t.Error("Run against blackhole succeeded")
	}
	// Fresh driver: the first failure poisoned the previous one, and
	// poison-once-fail-fast is itself asserted by TestSetVersionBeginBounded.
	d2 := openDriver(t, stub.addr(), &log)
	if err := d2.Drop(); err == nil {
		t.Error("Drop against blackhole succeeded")
	}
	_ = d.Close()
	_ = d2.Close()
}

func TestIsConnFailure(t *testing.T) {
	sqlState := func(code string) error { return &pgconn.PgError{Code: code} }
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"connection refused", sqlState("08006"), true},
		{"operator intervention", sqlState("57P01"), true},
		{"undefined table", sqlState("42P01"), false},
		{"insufficient privilege", sqlState("42501"), false},
		{"no rows", sql.ErrNoRows, false},
		{"deadline", context.DeadlineExceeded, true},
		{"canceled", context.Canceled, true},
		{"generic", errors.New("boom"), true},
		{"wrapped sql error", &database.Error{OrigErr: &pgconn.PgError{Code: "42601"}}, false},
		{"wrapped conn error", &database.Error{OrigErr: &pgconn.PgError{Code: "08006"}}, true},
		{"wrapped no rows", &database.Error{OrigErr: sql.ErrNoRows}, false},
	}
	for _, c := range cases {
		if got := isConnFailure(c.err); got != c.want {
			t.Errorf("%s: isConnFailure = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestCloseIdempotentAndFailFast(t *testing.T) {
	stub := newPGStub(t, modeBlackhole)
	var log eventLog
	d := openDriver(t, stub.addr(), &log)
	err := mustReturn(t, 30*time.Second, func() error { return d.Close() })
	if err != nil {
		t.Logf("close over blackhole: %v (recorded, tolerated)", err)
	}
	if err := d.Close(); err != nil {
		t.Errorf("second Close: %v, want nil", err)
	}
	if err := d.Lock(); err == nil {
		t.Error("Lock after Close succeeded")
	}
}

func TestEventsSecretFree(t *testing.T) {
	stub := newPGStub(t, modeBlackhole)
	var log eventLog
	d := openDriver(t, stub.addr(), &log)
	_ = d.SetVersion(3, false)
	_ = d.Close()
	raw, err := json.Marshal(log.events)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if strings.Contains(s, "127.0.0.1") {
		t.Errorf("events leak endpoint material: %s", s)
	}
	if len(log.events) == 0 {
		t.Fatal("no events emitted")
	}
	for _, e := range log.events {
		if e.Phase == "" {
			t.Errorf("event without phase: %+v", e)
		}
	}
}

func TestWithDeadlineMechanics(t *testing.T) {
	// Direct mechanism proof: a fn that only returns on cancellation
	// must surface a deadline, poison the driver, and emit one event.
	var log eventLog
	d := &Driver{base: context.Background(), cfg: Config{OpTimeout: time.Second, Sink: log.sink}}
	block := func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}
	start := time.Now()
	err := d.withDeadline("probe", NilVersion, 150*time.Millisecond, block)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("took %s, want bounded", elapsed)
	}
	if !d.poisoned.Load() {
		t.Error("not poisoned after deadline")
	}
	if len(log.events) != 1 || log.events[0].Phase != "probe" || log.events[0].Reusable {
		t.Errorf("events = %+v", log.events)
	}
	// Parent cancellation (not just timeout) propagates through base.
	pctx, cancel := context.WithCancel(context.Background())
	d2 := &Driver{base: pctx, cfg: Config{OpTimeout: time.Minute, Sink: log.sink}}
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	if err := d2.withDeadline("probe-cancel", NilVersion, time.Minute, block); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want Canceled", err)
	}
}

func TestCaptureSinkSummary(t *testing.T) {
	sink, summary := CaptureSink(3)
	sink(Event{Phase: "open"})
	sink(Event{Phase: "lock"})
	sink(Event{Phase: "run", Err: context.DeadlineExceeded})
	sink(Event{Phase: "close"})
	got := summary()
	// Last 3 only, in order, outcome-marked, no secret-shaped content.
	want := "lock✓→run✗(deadline)→close✓"
	if got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}
}

func TestNoGoroutineLeakOnFailure(t *testing.T) {
	stub := newPGStub(t, modeBlackhole)
	cycle := func() {
		var log eventLog
		_, _ = Open(context.Background(), testConfig(t, stub.addr(), &log))
	}
	cycle() // warm caches, pools, first-time inits
	baseline := runtime.NumGoroutine()
	for i := 0; i < 3; i++ {
		cycle()
	}
	// Release server-side conns so client close-drains (which await
	// server FIN by pgx design) can finish; then account.
	stub.dropConns()
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > baseline && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if got := runtime.NumGoroutine(); got > baseline {
		t.Errorf("goroutines grew %d -> %d across failed opens", baseline, got)
	}
}

// TestReadWholeBodyPinsNoSplitting guards the single-Exec decision:
// migration bytes with dollar-quoted semicolons, quoted semicolons,
// and comment semicolons must pass through byte-identical for the
// SERVER to split. Client-side splitting mangled a real framework
// migration on live Neon (42601); this test forbids its return.
func TestReadWholeBodyPinsNoSplitting(t *testing.T) {
	in := "-- comment; with semicolon\n" +
		"CREATE FUNCTION f() RETURNS void LANGUAGE sql AS $$\n" +
		"  SELECT ';not-a-splitter';\n" +
		"$$;\n" +
		"SELECT 'it''s; fine';\n"
	got, err := readWholeBody(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != in {
		t.Errorf("body altered:\n%q", got)
	}
}

func TestConfigDefaults(t *testing.T) {
	var c Config
	c.withDefaults()
	if c.OpTimeout != DefaultOpTimeout || c.LockTimeout != DefaultLockTimeout || c.CleanupTimeout != DefaultCleanupTimeout {
		t.Errorf("defaults not applied: %+v", c)
	}
	if c.Sink == nil {
		t.Error("default sink nil")
	}
}
