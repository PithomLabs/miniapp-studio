package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp-studio/dbenv"
)

func TestServeDSNs(t *testing.T) {
	owner, app, err := serveDSNs("postgres://neon_owner:pw@ep-1/x.neon.tech:5432/neondb?sslmode=verify-full", "studio_dev_ui", "apw")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(owner, "/studio_dev_ui?") || !strings.Contains(owner, "neon_owner") {
		t.Errorf("owner DSN = %q", redactForTest(owner))
	}
	if !strings.Contains(app, "/studio_dev_ui?") || !strings.Contains(app, "studio_app") {
		t.Errorf("app DSN user/db wrong: %q", redactForTest(app))
	}
	if _, _, err := serveDSNs("://bad", "studio_dev_ui", "x"); err == nil {
		t.Error("bad base DSN accepted")
	}
}

func redactForTest(dsn string) string {
	if i := strings.Index(dsn, "@"); i >= 0 {
		if j := strings.LastIndex(dsn[:i], ":"); j >= 0 {
			return dsn[:j] + ":***" + dsn[i:]
		}
	}
	return dsn
}

func TestResolveInitDB(t *testing.T) {
	name, stable, err := resolveInitDB("")
	if err != nil || !stable || name != devDatabaseName {
		t.Errorf("default = (%q,%v,%v)", name, stable, err)
	}
	name, stable, err = resolveInitDB("studio_test_run_1_2_abcdef")
	if err != nil || stable || name != "studio_test_run_1_2_abcdef" {
		t.Errorf("explicit run name = (%q,%v,%v)", name, stable, err)
	}
	if _, _, err := resolveInitDB("nope"); err == nil {
		t.Error("non-run explicit name accepted")
	}
}

func TestPasswordPlan(t *testing.T) {
	cases := []struct {
		role, file, rotate bool
		want               passwordAction
		wantErr            bool
	}{
		{false, false, false, pwGenerateCreate, false},
		{false, false, true, pwGenerateCreate, false}, // flag irrelevant: nothing to rotate
		{false, true, false, pwCreate, false},
		{false, true, true, pwCreate, false},
		{true, false, false, pwRefuse, true}, // the fail-closed cell
		{true, false, true, pwGenerateRotate, false},
		{true, true, false, pwProbe, false},
		{true, true, true, pwRotate, false},
	}
	for _, tc := range cases {
		got, err := passwordPlan(tc.role, tc.file, tc.rotate)
		if (err != nil) != tc.wantErr || got != tc.want {
			t.Errorf("plan(%v,%v,%v) = (%v,%v), want (%v,err=%v)",
				tc.role, tc.file, tc.rotate, got, err, tc.want, tc.wantErr)
		}
	}
	if _, err := passwordPlan(true, false, false); err == nil ||
		!strings.Contains(err.Error(), "--rotate-app-password") {
		t.Errorf("refusal must name the explicit flag: %v", err)
	}
}

func TestServeRunPatternRefused(t *testing.T) {
	if !dbenv.IsRunDB("studio_test_run_1_2_abcdef") {
		t.Fatal("fixture name must match the run guard for this test to mean anything")
	}
	if dbenv.IsRunDB(devDatabaseName) {
		t.Fatalf("serve database name %q matches the run guard (reaper/reset would claim it)", devDatabaseName)
	}
	err := runDevServeInit([]string{
		"--metadata-file", "nope.json", "--dsn-file", "nope.dsn",
		"--operators-file", "o", "--keys-file", "k", "--grants-file", "g",
		"--app-password-file", "p", "--token-file", "t",
		"--db-name", "studio_test_run_1_2_abcdef",
	})
	if err == nil || !strings.Contains(err.Error(), "run-pattern") {
		t.Fatalf("run-pattern db-name accepted: %v", err)
	}
}

func TestServeFlagsRequired(t *testing.T) {
	if err := runDevServeInit([]string{}); err == nil || !strings.Contains(err.Error(), "missing required flags") {
		t.Fatalf("init without flags: %v", err)
	}
	if err := runDevServe([]string{}); err == nil || !strings.Contains(err.Error(), "missing required flags") {
		t.Fatalf("serve without flags: %v", err)
	}
}

func TestServeRefusesWithoutInit(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "dev-serve.json")
	err := runDevServe([]string{
		"--metadata-file", "m", "--dsn-file", "d", "--operators-file", "o",
		"--keys-file", "k", "--grants-file", "g", "--app-password-file", "p",
		"--token-file", "t", "--marker-file", marker,
	})
	if err == nil || !strings.Contains(err.Error(), "dev:serve-init") {
		t.Fatalf("serve without marker: %v", err)
	}
	if err := os.WriteFile(marker, []byte("{bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{
		"--metadata-file", "m", "--dsn-file", "d", "--operators-file", "o",
		"--keys-file", "k", "--grants-file", "g", "--app-password-file", "p",
		"--token-file", "t", "--marker-file", marker,
	}
	if err := runDevServe(args); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("serve with corrupt marker: %v", err)
	}
}

func TestServeRefusesNonLoopback(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "dev-serve.json")
	raw, _ := json.Marshal(serveMarker{DB: devDatabaseName, Tenant: "dev"})
	if err := os.WriteFile(marker, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	base := []string{
		"--metadata-file", "m", "--dsn-file", "d", "--operators-file", "o",
		"--keys-file", "k", "--grants-file", "g", "--app-password-file", "p",
		"--token-file", "t", "--marker-file", marker,
	}
	for _, listen := range []string{"0.0.0.0:8080", "10.0.0.5:8080"} {
		args := append(append([]string{}, base...), "--listen", listen)
		if err := runDevServe(args); err == nil || !strings.Contains(err.Error(), "non-loopback") {
			t.Errorf("listen %s accepted: %v", listen, err)
		}
	}
}

func TestWriteSecretFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "pw")
	if err := writeSecretFile(p, []byte("s3cret\n")); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 600", fi.Mode().Perm())
	}
	if err := writeSecretFile(p, []byte("other")); err == nil {
		t.Error("overwrite accepted")
	}
}

func TestGenHex(t *testing.T) {
	a, err := genHex(32)
	if err != nil {
		t.Fatal(err)
	}
	b, err := genHex(32)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 64 || a == b {
		t.Errorf("weak generator: len=%d equal=%v", len(a), a == b)
	}
	for _, c := range a {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Fatalf("non-hex char %q", c)
		}
	}
}

func TestEnsureKeysGrantsFiles(t *testing.T) {
	dir := t.TempDir()
	kp, gp := filepath.Join(dir, "k.json"), filepath.Join(dir, "g.json")
	if err := ensureKeysFile(kp, "dev-svc", "dev"); err != nil {
		t.Fatal(err)
	}
	if err := ensureGrantsFile(gp, "dev"); err != nil {
		t.Fatal(err)
	}
	// Idempotent re-run against what we wrote.
	if err := ensureKeysFile(kp, "dev-svc", "dev"); err != nil {
		t.Errorf("keys re-check: %v", err)
	}
	if err := ensureGrantsFile(gp, "dev"); err != nil {
		t.Errorf("grants re-check: %v", err)
	}
	// Foreign files are never edited or silently accepted.
	other := filepath.Join(dir, "other.json")
	if err := os.WriteFile(other, []byte(`{"someone-else": {}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureKeysFile(other, "dev-svc", "dev"); err == nil {
		t.Error("keys file lacking the credential accepted")
	}
	if err := ensureGrantsFile(other, "dev"); err == nil {
		t.Error("grants file lacking the tenant accepted")
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureKeysFile(bad, "dev-svc", "dev"); err == nil {
		t.Error("malformed keys file accepted")
	}
}
