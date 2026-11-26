package artifact

import "testing"

// TestCapValues pins the exact size caps: changing them is a
// contract change, never a silent edit.
func TestCapValues(t *testing.T) {
	if MaxFileBytes != 5<<20 {
		t.Errorf("MaxFileBytes = %d, want %d", MaxFileBytes, 5<<20)
	}
	if MaxTotalBytes != 50<<20 {
		t.Errorf("MaxTotalBytes = %d, want %d", MaxTotalBytes, 50<<20)
	}
}

func TestModulePathPattern(t *testing.T) {
	good := []string{
		"studio-artifact/t-demo/invoice-desk",
		"studio-artifact/acme-workspace/app1",
	}
	for _, p := range good {
		if !ValidModulePath(p) {
			t.Errorf("accept %q", p)
		}
	}
	bad := []string{
		"", "studio-artifact//app", "studio-artifact/t/",
		"github.com/other/app", "studio-artifact/t/app/extra",
		"studio-artifact/T-App/app", "studio-artifact/t/app.",
		"./relative", "../escape",
	}
	for _, p := range bad {
		if ValidModulePath(p) {
			t.Errorf("refuse %q: accepted", p)
		}
	}
}

func TestBannedImports(t *testing.T) {
	banned := []string{
		"unsafe", "plugin", "os/exec", "C",
		"net", "net/http", "net/netip", "net/http/httptest",
	}
	for _, p := range banned {
		if !IsBannedImport(p) {
			t.Errorf("ban %q", p)
		}
	}
	allowed := []string{
		"fmt", "encoding/json", "os", "os/signal",
		"github.com/PithomLabs/miniapp/contract",
		"plan9", "network", "exec",
	}
	for _, p := range allowed {
		if IsBannedImport(p) {
			t.Errorf("allow %q: banned", p)
		}
	}
}

func TestIsStdlib(t *testing.T) {
	// Note: "C" is dotless but never reaches this check — the
	// banned-import check runs first and refuses it exactly.
	for _, p := range []string{"fmt", "os/exec", "encoding/json"} {
		if !IsStdlib(p) {
			t.Errorf("stdlib %q", p)
		}
	}
	for _, p := range []string{
		"github.com/x/y", "gopkg.in/a.v1", "example.com/mod",
	} {
		if IsStdlib(p) {
			t.Errorf("non-stdlib %q reported stdlib", p)
		}
	}
}
