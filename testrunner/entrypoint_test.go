package main

import (
	"os"
	"strings"
	"testing"
)

// TestEntrypointToolchainEnv pins the L6 contract: toolchain paths
// arrive via broker-passed environment, are required, and are verified
// as directories — never hardcoded. A pattern test is the honest limit
// of shell coverage without a shell harness; `sh -n` (CI/manual gate)
// covers syntax.
func TestEntrypointToolchainEnv(t *testing.T) {
	raw, err := os.ReadFile("entrypoint.sh")
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{
		`${GOROOT:?broker must pass GOROOT}`,
		`${GOMODCACHE:?broker must pass GOMODCACHE}`,
		`[ -d "$GOROOT" ]`,
		`[ -d "$GOMODCACHE" ]`,
		`PATH="$GOROOT/bin:${PATH}"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("entrypoint lacks %q", want)
		}
	}
	for _, banned := range []string{
		"/usr/local/go",
		"/home/chaschel/go/pkg/mod",
		"GOMODCACHE:-",
	} {
		if strings.Contains(body, banned) {
			t.Errorf("entrypoint hardcodes %q", banned)
		}
	}
}
