package gateway

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFsyncDir pins the M1 helper: real dirs sync, unopenable dirs
// error (callers propagate; a failed dirsync must never read as a
// durable commit).
func TestFsyncDir(t *testing.T) {
	dir := t.TempDir()
	if err := fsyncDir(dir); err != nil {
		t.Fatalf("fsyncDir(temp) = %v", err)
	}
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(locked, 0o755) }()
	if os.Geteuid() == 0 {
		t.Log("running as root: 000 dirs stay openable; error path covered by the missing-dir case")
	} else if err := fsyncDir(locked); err == nil {
		t.Error("fsyncDir(000) accepted, want error")
	}
	if err := fsyncDir(filepath.Join(dir, "missing")); err == nil {
		t.Error("fsyncDir(missing) accepted, want error")
	}
}
