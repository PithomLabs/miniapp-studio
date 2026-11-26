package app

import (
	"os"
	"testing"
)

// TestEnvScrub proves the container never inherits host environment:
// the suite sets STUDIO_SENTINEL in its own process; the fixture
// fails if the value leaks across.
func TestEnvScrub(t *testing.T) {
	if v := os.Getenv("STUDIO_SENTINEL"); v != "" {
		t.Fatalf("host env leaked into the container: %q", v)
	}
}
