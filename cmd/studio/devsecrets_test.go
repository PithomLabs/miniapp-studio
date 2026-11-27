package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDevSecretsVerbsNeverPrintValues is the canary proof: fixture
// secrets flow through both verbs and must never appear in output.
func TestDevSecretsVerbsNeverPrintValues(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, "test.env")
	canary := "canary-hunter2-value"
	body := "STUDIO_DATABASE_URL=postgres://u:" + canary + "@h:5432/db\nSTUDIO_LISTEN=127.0.0.1:9090\n"
	if err := os.WriteFile(env, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, verb := range map[string]func([]string, *bytes.Buffer) error{
		"validate": func(a []string, b *bytes.Buffer) error { return runDevSecretsValidate(a, b) },
		"plan":     func(a []string, b *bytes.Buffer) error { return runDevSecretsPlan(append(a, "--env", "dev"), b) },
	} {
		var buf bytes.Buffer
		if err := verb([]string{"--env-file", env}, &buf); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if strings.Contains(buf.String(), canary) {
			t.Fatalf("%s leaked the secret value:\n%s", name, buf.String())
		}
		if !strings.Contains(buf.String(), "STUDIO_DATABASE_URL") {
			t.Errorf("%s omits the key name:\n%s", name, buf.String())
		}
	}
}

func TestDevSecretsValidateFailsClosed(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.env")
	if err := os.WriteFile(bad, []byte("UNKNOWN_KEY=x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := runDevSecretsValidate([]string{"--env-file", bad}, &buf); err == nil {
		t.Fatal("unknown key accepted")
	}
	loose := filepath.Join(dir, "loose")
	if err := os.WriteFile(loose, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok := filepath.Join(dir, "ok.env")
	if err := os.WriteFile(ok, []byte("STUDIO_LISTEN=127.0.0.1:9090\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := runDevSecretsValidate([]string{"--env-file", ok, "--check-files", loose}, &buf); err == nil {
		t.Fatal("0644 check-file accepted")
	}
	var buf2 bytes.Buffer
	if err := runDevSecretsPlan([]string{"--env-file", ok, "--env", "nope"}, &buf2); err == nil {
		t.Fatal("invalid env accepted")
	}
}
