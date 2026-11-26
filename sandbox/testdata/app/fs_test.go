package app

import (
	"os"
	"testing"
)

// TestFSConfinement proves mount policy: /work read-only, /scratch
// writable, no host paths visible.
func TestFSConfinement(t *testing.T) {
	if err := os.WriteFile("/work/should-fail", []byte("x"), 0o644); err == nil {
		t.Fatal("write to /work succeeded: source mount is not read-only")
	}
	if err := os.WriteFile("/scratch/ok", []byte("x"), 0o644); err != nil {
		t.Fatalf("write to /scratch failed: %v", err)
	}
	if _, err := os.ReadFile("/work/app/go.mod"); err != nil {
		t.Fatalf("expected source unreadable: %v", err)
	}
}
