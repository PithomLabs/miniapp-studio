package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testRegistry(t *testing.T) *Registry {
	t.Helper()
	r, err := NewRegistry([]Variable{
		{Name: "T_SECRET", Scope: ScopeDeployment, Sensitivity: SensitivitySecret, Rotation: "rotating"},
		{Name: "T_CONFIG", Scope: ScopeDeployment, Sensitivity: SensitivityConfig, Rotation: "static", Default: "dflt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRegistryRejectsSecretsWithDefaults(t *testing.T) {
	if _, err := NewRegistry([]Variable{{Name: "S", Scope: ScopeDeployment, Sensitivity: SensitivitySecret, Rotation: "static", Default: "x"}}); err == nil {
		t.Fatal("secret with default accepted")
	}
}

func TestRegistryRejectsDuplicatesAndBadNames(t *testing.T) {
	base := Variable{Name: "S", Scope: ScopeDeployment, Sensitivity: SensitivityConfig, Rotation: "static"}
	if _, err := NewRegistry([]Variable{base, base}); err == nil {
		t.Fatal("duplicate name accepted")
	}
	for _, name := range []string{"", "lower", "HAS-DASH", "HAS SPACE"} {
		bad := base
		bad.Name = name
		if _, err := NewRegistry([]Variable{bad}); err == nil {
			t.Fatalf("invalid name %q accepted", name)
		}
	}
}

func TestParseDotEnvStrictSubset(t *testing.T) {
	reg := testRegistry(t)
	got, err := ParseDotEnv([]byte("# comment\n\nT_SECRET = s3cret value \nT_CONFIG=cfg\n"), reg)
	if err != nil {
		t.Fatalf("valid dotenv rejected: %v", err)
	}
	if got["T_SECRET"] != " s3cret value" || got["T_CONFIG"] != "cfg" {
		t.Fatalf("values = %q", got)
	}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"export", "export T_CONFIG=x\n", "export"},
		{"no-equals", "T_CONFIG\n", "KEY=VALUE"},
		{"bad-key", "oops=x\n", "invalid key"},
		{"dup", "T_CONFIG=a\nT_CONFIG=b\n", "duplicate key"},
		{"empty", "T_CONFIG=   \n", "empty value"},
		{"unknown", "T_CONFIG=a\nNOPE=b\n", "unknown key"},
	} {
		if _, err := ParseDotEnv([]byte(tc.body), reg); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want substring %q", tc.name, err, tc.want)
		}
	}
	if _, err := ParseDotEnv(make([]byte, MaxDotEnvBytes+1), reg); err == nil {
		t.Error("oversize file accepted")
	}
}

func writeSecretFile(t *testing.T, mode os.FileMode, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "s")
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadPrecedenceAndConflict(t *testing.T) {
	reg := testRegistry(t)
	t.Setenv("T_SECRET", "from-env")
	t.Setenv("T_SECRET_FILE", "")
	if got, err := reg.Load("T_SECRET", LoadOpts{}); err != nil || got != "from-env" {
		t.Fatalf("env load = %q, %v", got, err)
	}
	f := writeSecretFile(t, 0o600, "from-file\n")
	t.Setenv("T_SECRET_FILE", f)
	if _, err := reg.Load("T_SECRET", LoadOpts{}); err == nil || !strings.Contains(err.Error(), "multiple sources") {
		t.Fatalf("env+file conflict err = %v", err)
	}
	t.Setenv("T_SECRET", "")
	if got, err := reg.Load("T_SECRET", LoadOpts{}); err != nil || got != "from-file" {
		t.Fatalf("file load = %q, %v", got, err)
	}
	t.Setenv("T_SECRET_FILE", "")
	if got, err := reg.Load("T_SECRET", LoadOpts{DotEnv: map[string]string{"T_SECRET": "from-dotenv"}}); err != nil || got != "from-dotenv" {
		t.Fatalf("dotenv load = %q, %v", got, err)
	}
	if got, err := reg.Load("T_CONFIG", LoadOpts{}); err != nil || got != "dflt" {
		t.Fatalf("default load = %q, %v", got, err)
	}
}

func TestLoadConfigFlagButNeverSecretFlag(t *testing.T) {
	reg := testRegistry(t)
	if got, err := reg.Load("T_CONFIG", LoadOpts{FlagValue: "fv", FlagSet: true}); err != nil || got != "fv" {
		t.Fatalf("config flag = %q, %v", got, err)
	}
	if _, err := reg.Load("T_SECRET", LoadOpts{FlagValue: "fv", FlagSet: true}); err == nil || !strings.Contains(err.Error(), "never arrive via a CLI flag") {
		t.Fatalf("secret flag err = %v", err)
	}
	if _, err := reg.Load("UNKNOWN_VAR", LoadOpts{}); err == nil {
		t.Fatal("unknown variable accepted")
	}
}

func TestLoadRefusesWorldReadableFile(t *testing.T) {
	reg := testRegistry(t)
	f := writeSecretFile(t, 0o644, "shh")
	t.Setenv("T_SECRET", "")
	t.Setenv("T_SECRET_FILE", f)
	if _, err := reg.Load("T_SECRET", LoadOpts{}); err == nil || !strings.Contains(err.Error(), "owner-only") {
		t.Fatalf("0644 file err = %v", err)
	}
	if err := CheckFileMode(f); err == nil {
		t.Fatal("CheckFileMode accepted 0644")
	}
	strict := writeSecretFile(t, 0o600, "shh")
	if err := CheckFileMode(strict); err != nil {
		t.Fatalf("0600 rejected: %v", err)
	}
	readonly := writeSecretFile(t, 0o400, "shh")
	if err := CheckFileMode(readonly); err != nil {
		t.Fatalf("0400 rejected: %v", err)
	}
}

func TestStudioRegistryLoads(t *testing.T) {
	reg := StudioRegistry()
	if _, ok := reg.Lookup("STUDIO_DATABASE_URL"); !ok {
		t.Fatal("STUDIO_DATABASE_URL missing")
	}
	if n := len(reg.Names()); n != 11 {
		t.Fatalf("registry size = %d, want 11", n)
	}
}
