package gateway

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSupervisorRunAndExitCode(t *testing.T) {
	s := testSupervisor(t)
	ctx := context.Background()
	p, err := s.Start(ctx, "p1", t.TempDir(), "sh", []string{"-c", "echo hello; exit 3"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Pid() <= 0 {
		t.Errorf("pid = %d", p.Pid())
	}
	code, err := p.Wait()
	if code != 3 || err == nil {
		t.Errorf("wait = (%d, %v), want (3, error)", code, err)
	}
	if _, ok := s.Lookup("p1"); !ok {
		t.Error("process missing from registry")
	}
	// Duplicate ids refused.
	if _, err := s.Start(ctx, "p1", t.TempDir(), "sh", []string{"-c", "true"}, nil); err == nil {
		t.Error("duplicate id accepted, want refusal")
	}
	// Missing binary fails fast.
	if _, err := s.Start(ctx, "p2", t.TempDir(), "gateway-no-such-binary-xyz", nil, nil); err == nil {
		t.Error("missing binary accepted, want error")
	}
}

func TestSupervisorKill(t *testing.T) {
	s := testSupervisor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, err := s.Start(ctx, "k1", t.TempDir(), "sh", []string{"-c", "sleep 30"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Kill(); err != nil {
		t.Fatal(err)
	}
	code, err := p.Wait()
	if code != -1 || err == nil {
		t.Errorf("killed wait = (%d, %v), want (-1, error)", code, err)
	}
	// Second wait returns the same outcome.
	if code2, _ := p.Wait(); code2 != code {
		t.Errorf("second wait = %d, want %d", code2, code)
	}
}

func TestSupervisorStdoutStreams(t *testing.T) {
	s := testSupervisor(t)
	ctx := context.Background()
	p, err := s.Start(ctx, "o1", t.TempDir(), "sh",
		[]string{"-c", "printf '{\"a\":1}\\n{\"b\":2}\\n'"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	buf := make([]byte, 64)
	for {
		n, rerr := p.Stdout().Read(buf)
		sb.Write(buf[:n])
		if rerr != nil {
			break
		}
	}
	if _, err := p.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	if got := sb.String(); got != "{\"a\":1}\n{\"b\":2}\n" {
		t.Errorf("stdout = %q", got)
	}
}
