package gateway

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestShouldReap(t *testing.T) {
	root := t.TempDir()
	write := func(pid, comm, cmdline string, start int64) {
		t.Helper()
		dir := filepath.Join(root, pid)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(cmdline), 0o600); err != nil {
			t.Fatal(err)
		}
		stat := fmt.Sprintf("%s (%s) R 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 %d 0 0\n",
			pid, comm, start)
		if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(stat), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	const selfTicks = 200
	write("100", "opencode", "opencode\x00run\x00--dir\x00/w\x00", 100)
	write("200", "sleep", "sleep\x0030\x00", 100)
	write("300", "x", "", 100)
	write("500", "opencode", "/usr/local/bin/opencode\x00run\x00", 100)
	write("600", "opencode", "opencode\x00run\x00", 300)
	write("700", "vim", "vim\x00opencode-notes.txt\x00", 100)
	write("800", "my (odd) opencode", "opencode\x00run\x00", 100)
	if !shouldReap(root, 100, selfTicks) {
		t.Error("opencode orphan predating boot not selected")
	}
	if shouldReap(root, 200, selfTicks) {
		t.Error("foreign process selected for kill")
	}
	if shouldReap(root, 300, selfTicks) {
		t.Error("empty cmdline selected for kill")
	}
	if shouldReap(root, 404, selfTicks) {
		t.Error("missing pid selected for kill")
	}
	if shouldReap(root, 0, selfTicks) || shouldReap(root, -1, selfTicks) {
		t.Error("non-pid selected for kill")
	}
	if !shouldReap(root, 500, selfTicks) {
		t.Error("pathed argv[0] with opencode basename not selected")
	}
	if shouldReap(root, 600, selfTicks) {
		t.Error("post-boot pid selected for kill (PID reuse guard)")
	}
	if shouldReap(root, 700, selfTicks) {
		t.Error("decoy argv mentioning opencode selected for kill")
	}
	if !shouldReap(root, 800, selfTicks) {
		t.Error("comm-with-parens orphan not selected (stat parser)")
	}
	// Unknown self ticks skips only the time gate: argv[0]-exact
	// still applies.
	if !shouldReap(root, 100, 0) {
		t.Error("orphan refused without self ticks (argv0 should suffice)")
	}
	if shouldReap(root, 700, 0) {
		t.Error("decoy selected without self ticks")
	}
}

func TestRecoverSessions(t *testing.T) {
	srv, _ := testServer(t)
	ctx := context.Background()
	// A stale running record with a dead pid: marked failed + callback queued.
	if err := srv.Sessions.Create("b-rec", SessionRecord{
		SessionRef: "sess-rec", BuildID: "b-rec", Workspace: "w-1",
		WorkDir: t.TempDir(), PID: 1<<30 - 7, State: SessionRunning,
	}); err != nil {
		t.Fatal(err)
	}
	// A finished record is left alone.
	if err := srv.Sessions.Create("b-done", SessionRecord{
		SessionRef: "sess-done", BuildID: "b-done", State: SessionDone,
	}); err != nil {
		t.Fatal(err)
	}
	srv.RecoverSessions(ctx)
	rec, err := srv.Sessions.Get("b-rec")
	if err != nil {
		t.Fatal(err)
	}
	if rec.State != SessionFailed {
		t.Errorf("stale state = %q, want failed", rec.State)
	}
	done, err := srv.Sessions.Get("b-done")
	if err != nil {
		t.Fatal(err)
	}
	if done.State != SessionDone {
		t.Errorf("finished state = %q, want untouched done", done.State)
	}
	if q, _ := srv.Outbox.Depth(); q != 1 {
		t.Fatalf("outbox depth = %d, want 1 failed callback", q)
	}
	// A live foreign process is never reaped: mark-only.
	procRoot := t.TempDir()
	pid := os.Getpid()
	dir := fmt.Sprintf("%s/%d", procRoot, pid)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Our own test binary is definitively not opencode.
	if err := os.WriteFile(dir+"/cmdline", []byte("gateway.test\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv.ProcRoot = procRoot
	if err := srv.Sessions.Create("b-live", SessionRecord{
		SessionRef: "sess-live", BuildID: "b-live", Workspace: "w-1", PID: pid, State: SessionRunning,
	}); err != nil {
		t.Fatal(err)
	}
	srv.RecoverSessions(ctx)
	live, err := srv.Sessions.Get("b-live")
	if err != nil {
		t.Fatal(err)
	}
	if live.State != SessionFailed {
		t.Errorf("live-foreign state = %q, want failed", live.State)
	}
	// Signal 0 proves our own process survived recovery (FindProcess alone
	// never fails on Linux, so it would prove nothing).
	if proc, err := os.FindProcess(pid); err != nil {
		t.Fatal(err)
	} else if err := proc.Signal(syscall.Signal(0)); err != nil {
		t.Errorf("test process did not survive recovery: %v", err)
	}
}
