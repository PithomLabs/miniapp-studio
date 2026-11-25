package gateway

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func readAllString(t *testing.T, r io.Reader) string {
	t.Helper()
	raw, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestSupervisorRequestCancelSparesAgent (M3): agent lifetime must not be
// tied to the HTTP request context — cancelling the caller's context
// after spawn must leave the child alive; only the explicit Cancel path
// stops it.
func TestSupervisorRequestCancelSparesAgent(t *testing.T) {
	s := testSupervisor(t)
	ctx, cancel := context.WithCancel(context.Background())
	p, err := s.Start(ctx, "m3", t.TempDir(), "sh", []string{"-c", "sleep 30"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Kill() }()
	cancel()
	time.Sleep(200 * time.Millisecond)
	if err := p.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("agent died with request context: %v", err)
	}
}

// TestSupervisorKillReapsGrandchildren (M4): cancel/reap must reach the
// whole process group — a grandchild sleeper must not survive Kill.
func TestSupervisorKillReapsGrandchildren(t *testing.T) {
	s := testSupervisor(t)
	marker := "31337"
	p, err := s.Start(context.Background(), "m4", t.TempDir(), "sh",
		[]string{"-c", "sleep " + marker + " & wait"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = exec.Command("pkill", "-f", "sleep "+marker).Run()
	})
	time.Sleep(300 * time.Millisecond) // let the grandchild appear
	if !procWithMarkerExists(t, marker) {
		t.Skip("grandchild never appeared; cannot test group kill")
	}
	if err := p.Kill(); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Wait(); err == nil {
		t.Error("killed wait returned no error")
	}
	deadline := time.Now().Add(3 * time.Second)
	for procWithMarkerExists(t, marker) && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if procWithMarkerExists(t, marker) {
		t.Error("grandchild sleeper survived Kill (no process group)")
	}
}

// TestKillProcessGroupNonLeader (M4): a pid that is not a group leader
// (never true for supervisor children) still dies via the direct-kill
// fallback instead of surviving with ESRCH.
func TestKillProcessGroupNonLeader(t *testing.T) {
	cmd := exec.Command("sleep", "300")
	if err := cmd.Start(); err != nil {
		t.Skip("no sleep binary")
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	if err := killProcessGroup(cmd.Process.Pid, os.Kill); err != nil {
		t.Fatalf("non-leader kill: %v", err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("non-leader survived group kill")
	}
}

func procWithMarkerExists(t *testing.T, marker string) bool {
	t.Helper()
	entries, err := os.ReadDir("/proc")
	if err != nil {
		t.Skip("no /proc; cannot verify group kill")
	}
	for _, e := range entries {
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if err != nil {
			continue
		}
		if strings.Contains(string(raw), "sleep "+marker) {
			return true
		}
	}
	return false
}

// TestSupervisorStderrDrainedAndBounded (M5): a runaway stderr must
// neither wedge the agent nor grow memory without bound — the supervisor
// drains it and keeps only a bounded tail.
func TestSupervisorStderrDrainedAndBounded(t *testing.T) {
	s := testSupervisor(t)
	p, err := s.Start(context.Background(), "m5", t.TempDir(), "sh",
		[]string{"-c", "seq 1 200000 >&2; echo done"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _, _ = p.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		_ = p.Kill()
		t.Fatal("agent wedged on undrained stderr")
	}
	tail := readAllString(t, p.Stderr())
	if len(tail) > 64*1024 {
		t.Errorf("stderr tail = %d bytes, want bounded ≤64KB", len(tail))
	}
	if !strings.Contains(tail, "200000") {
		t.Error("stderr tail missing the final lines")
	}
}
