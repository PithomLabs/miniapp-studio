package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDevDispatch(t *testing.T) {
	if err := runDev(nil); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Errorf("empty args: %v", err)
	}
	if err := runDev([]string{"frobnicate"}); err == nil || !strings.Contains(err.Error(), "unknown dev command") {
		t.Errorf("unknown verb: %v", err)
	}
}

func TestGoToolchainOK(t *testing.T) {
	cases := []struct {
		runtime, mod string
		want         bool
	}{
		{"go1.27.3", "module x\n\ngo 1.27\n", true},
		{"go1.27.3", "module x\n\ngo 1.27.3\n", true},
		{"go1.28.0", "module x\n\ngo 1.27\n", true},
		{"go1.26.0", "module x\n\ngo 1.27\n", false},
		{"go1.27.3", "module x\n", false},
		{"bogus", "module x\n\ngo 1.27\n", false},
	}
	for _, c := range cases {
		if got, _ := goToolchainOK(c.runtime, c.mod); got != c.want {
			t.Errorf("goToolchainOK(%q) = %v, want %v", c.runtime, got, c.want)
		}
	}
}

func TestFingerprintShape(t *testing.T) {
	fp := Fingerprint{
		GitCommit: "abc", GoVersion: "go1.27.3", GoModFloor: "1.27",
		Endpoint: NeonEndpoint{
			ProjectID: "proj_test_123", BranchID: "br_test_abc", ParentBranch: "br_parent_xyz",
			Host: "ep-test-0001.invalid", Port: "5432", TLSMode: "verify-full",
		},
		Env:    map[string]string{"TEST_DATABASE_URL": "set"},
		Checks: []FingerprintCheck{{Name: "endpoint", Status: "pass"}},
	}
	raw, err := json.Marshal(fp)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, key := range []string{"git_commit", "go_version", "endpoint", "server", "studio_migrations", "framework_migrations", "env", "checks",
		"project_id", "branch_id", "parent_branch", "tls_mode"} {
		if !strings.Contains(s, `"`+key+`"`) {
			t.Errorf("fingerprint JSON lacks %q: %s", key, s)
		}
	}
	// Credentials must never appear in the record, by construction.
	for _, banned := range []string{"postgres://", "password", "passwd", "secret"} {
		if strings.Contains(strings.ToLower(s), banned) {
			t.Errorf("fingerprint leaks %q", banned)
		}
	}
}

func TestDbTestPackagesExist(t *testing.T) {
	if len(dbTestPackages) == 0 {
		t.Fatal("empty package list")
	}
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	for _, p := range dbTestPackages {
		dir := filepath.Join(repo, strings.TrimSuffix(strings.TrimPrefix(p, "./"), "/..."))
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("package dir %s missing (list rotted): %v", dir, err)
		}
		matches, _ := filepath.Glob(filepath.Join(dir, "*_test.go"))
		if len(matches) == 0 {
			t.Errorf("package %s has no tests", dir)
		}
	}
}

func TestDevResetRequiresConfirm(t *testing.T) {
	if err := runDevReset(nil); err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Errorf("reset without --yes: %v", err)
	}
}

func TestDevTestArgv(t *testing.T) {
	got := devTestArgv("TestX", "1", true, "5m", []string{"./store/..."}, []string{"-shuffle", "on"})
	want := []string{"test", "-p", "1", "-count", "1", "-run", "TestX", "-v", "-timeout", "5m", "./store/...", "-shuffle", "on"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("argv = %q, want %q", got, want)
	}
	plain := devTestArgv("", "2", false, "", []string{"./..."}, nil)
	if strings.Join(plain, " ") != "test -p 1 -count 2 ./..." {
		t.Errorf("plain argv = %q", plain)
	}
}

func TestDevDSNMissing(t *testing.T) {
	t.Setenv("TEST_DATABASE_URL", "")
	if _, err := devDSN(""); err == nil {
		t.Error("missing DSN accepted")
	}
}

func TestDevPolicyPrecedence(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "b.json")
	content := `{"project_id":"file-proj","branch_id":"file-br","endpoint_hosts":["ep-file.invalid"]}`
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	// Explicit file wins over (absent) env.
	t.Setenv("NEON_PROJECT_ID", "")
	t.Setenv("NEON_BRANCH_ID", "")
	t.Setenv("NEON_ENDPOINT_HOSTS", "")
	pol, err := devPolicy(file)
	if err != nil || pol.ProjectID != "file-proj" {
		t.Errorf("file policy: %+v %v", pol, err)
	}
	// Without a file, env applies.
	t.Setenv("NEON_PROJECT_ID", "env-proj")
	t.Setenv("NEON_BRANCH_ID", "env-br")
	t.Setenv("NEON_ENDPOINT_HOSTS", "ep-env.invalid")
	pol, err = devPolicy("")
	if err != nil || pol.ProjectID != "env-proj" {
		t.Errorf("env policy: %+v %v", pol, err)
	}
	// A bad file fails instead of falling back to env.
	if _, err := devPolicy(filepath.Join(dir, "absent.json")); err == nil {
		t.Error("absent file fell back to env")
	}
}
