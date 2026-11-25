package gateway

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestAdapterInboxSecureChain (B1): every adapter-created handshake path
// carries the permission model — .studio group-traversable, inbox
// group-writable + sticky, both owned by the inbox group.
func TestAdapterInboxSecureChain(t *testing.T) {
	runner := &ScriptedRunner{Scripts: []Script{{Stdout: "{\"type\":\"text\",\"timestamp\":1,\"sessionID\":\"ses-i-1\",\"part\":{\"type\":\"text\",\"text\":\"hi\"}}\n"}}}
	adapter, _, work := testAdapter(t, runner)
	if _, ch, err := adapter.Start(context.Background(), BuildAssignment{BuildID: "b-inb", WorkDir: work, Prompt: "p"}); err != nil {
		t.Fatal(err)
	} else {
		_ = drain(t, ch)
	}
	studio := filepath.Join(work, ".studio")
	inbox := filepath.Join(work, inboxDir)
	assertDirMode(t, studio, 0o750, true, false)
	assertDirMode(t, inbox, 0o730, true, true)
	if gidOf(t, inbox) != os.Getegid() {
		t.Errorf("inbox gid = %d, want egid %d", gidOf(t, inbox), os.Getegid())
	}
}

// assertDirMode checks permission bits plus the setgid/sticky bits
// (FileMode.Perm drops the special bits, so they are asserted separately).
func assertDirMode(t *testing.T, path string, perm os.FileMode, setgid, sticky bool) {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != perm {
		t.Errorf("%s perm = %o, want %o", path, fi.Mode().Perm(), perm)
	}
	if setgid && fi.Mode()&os.ModeSetgid == 0 {
		t.Errorf("%s lacks setgid", path)
	}
	if sticky && fi.Mode()&os.ModeSticky == 0 {
		t.Errorf("%s lacks sticky bit", path)
	}
}

func gidOf(t *testing.T, path string) int {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skip("no stat_t; cannot assert gid")
	}
	return int(st.Gid)
}

// TestAdapterShutdownCancelsRuns (M3): gateway shutdown signals tracked
// agent runs instead of orphaning them until the next boot.
func TestAdapterShutdownCancelsRuns(t *testing.T) {
	runner := &ScriptedRunner{Scripts: []Script{{
		Stdout: "{\"type\":\"text\",\"timestamp\":1,\"sessionID\":\"ses-sd-1\",\"part\":{\"type\":\"text\",\"text\":\"hi\"}}\n",
		Hang:   true,
	}}}
	adapter, _, work := testAdapter(t, runner)
	if _, _, err := adapter.Start(context.Background(), BuildAssignment{BuildID: "b-sd", WorkDir: work, Prompt: "p"}); err != nil {
		t.Fatal(err)
	}
	adapter.mu.Lock()
	live := len(adapter.runs)
	adapter.mu.Unlock()
	if live != 1 {
		t.Fatalf("live runs = %d, want 1", live)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	adapter.Shutdown(ctx)
	deadline := time.Now().Add(5 * time.Second)
	for {
		adapter.mu.Lock()
		live = len(adapter.runs)
		adapter.mu.Unlock()
		if live == 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if live != 0 {
		t.Errorf("live runs after Shutdown = %d, want 0", live)
	}
}
