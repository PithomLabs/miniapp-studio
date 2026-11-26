package artifact

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestCheckToolchain(t *testing.T) {
	if err := CheckToolchain(Toolchain{Go: PinnedGoVersion, Miniapp: PinnedMiniapp}); err != nil {
		t.Errorf("pinned toolchain refused: %v", err)
	}
	bad := []Toolchain{
		{}, {Go: PinnedGoVersion}, {Miniapp: PinnedMiniapp},
		{Go: "9.9.9", Miniapp: PinnedMiniapp},
		{Go: PinnedGoVersion, Miniapp: "v0.0.0-other"},
		{Go: PinnedGoVersion + " ", Miniapp: PinnedMiniapp},
	}
	for _, decl := range bad {
		if err := CheckToolchain(decl); err == nil {
			t.Errorf("refuse %+v: accepted", decl)
		} else if err.Code != EDeps {
			t.Errorf("refuse %+v: code = %s, want EDeps", decl, err.Code)
		}
	}
}

// TestGoSumLinesMatchRepo fails the suite if the pinned go.sum lines
// drift from the repository's own go.sum.
func TestGoSumLinesMatchRepo(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "..", "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, line := range []string{MiniappGoSumZip, MiniappGoSumMod} {
		if !strings.Contains(body, line+"\n") {
			t.Errorf("go.sum lacks pinned line %q", line)
		}
	}
}

// TestToolchainPinMatchesGoMod fails the suite if the frozen pins
// drift from the repository's own go.mod. Pins are enforced by test,
// never by documentation alone.
func TestToolchainPinMatchesGoMod(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	goLine := regexp.MustCompile(`(?m)^go ([0-9.]+)$`).FindSubmatch(raw)
	if goLine == nil || string(goLine[1]) != PinnedGoVersion {
		t.Errorf("go.mod go directive = %q, want pinned %q", goLine, PinnedGoVersion)
	}
	modLine := regexp.MustCompile(`(?m)^\s*github\.com/PithomLabs/miniapp (\S+)$`).FindSubmatch(raw)
	if modLine == nil || string(modLine[1]) != PinnedMiniapp {
		t.Errorf("go.mod miniapp = %q, want pinned %q", modLine, PinnedMiniapp)
	}
}
