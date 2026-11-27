package capability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRefreshGrantsNothing pins the key increment boundary: refresh
// reports what the platform can do and must hold no call edges into
// anything that authorizes action — broker minting, invitation
// acceptance, or session issuance. Production files in this package
// must not reference those surfaces; the needles assemble from parts
// so this scanner never matches its own source.
func TestRefreshGrantsNothing(t *testing.T) {
	needles := []string{
		"bro" + "ker.",
		"invi" + "te.",
		"sess" + "ion_create",
		"invita" + "tion_accept",
		"Mint" + "(",
		"Cons" + "ume(",
		"Enr" + "oll(",
		"Appr" + "ove(",
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(".", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range needles {
			if strings.Contains(string(raw), n) {
				t.Errorf("%s references forbidden authorization surface %q", e.Name(), n)
			}
		}
		// The handler map may only be probed for key presence (the
		// drift check); any other access form could invoke a tool.
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.Contains(line, "ClassATools") && !strings.Contains(line, "_, bound :=") {
				t.Errorf("%s: non-presence ClassATools access: %q", e.Name(), strings.TrimSpace(line))
			}
		}
	}
}

// TestDefaultToolListerReadsLiveRegistry proves the default source
// reflects the actual tool registration (not a hard-coded list): it
// must return the manifest's tools with handlers bound.
func TestDefaultToolListerReadsLiveRegistry(t *testing.T) {
	version, build, declared, err := DefaultToolLister()
	if err != nil {
		t.Fatal(err)
	}
	if version == "" || build == "" || len(declared) == 0 {
		t.Fatalf("lister = (%q, %q, %d tools)", version, build, len(declared))
	}
	for _, d := range declared {
		if !d.Bound {
			t.Errorf("manifest tool %q has no bound handler (registry drift)", d.Name)
		}
	}
}
