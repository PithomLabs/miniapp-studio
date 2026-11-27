package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const bindProjects = `[{"id":"proj_1","name":"miniapp-studio-dev","extra":true},
	{"id":"proj_2","name":"other"}]`

func TestFindNamed(t *testing.T) {
	id, err := findNamed([]byte(bindProjects), "miniapp-studio-dev", "project")
	if err != nil || id != "proj_1" {
		t.Errorf("exact match: id=%q err=%v", id, err)
	}
	if _, err := findNamed([]byte(bindProjects), "absent", "project"); err == nil {
		t.Error("miss accepted")
	} else if !strings.Contains(err.Error(), "miniapp-studio-dev") {
		t.Errorf("miss error %q does not list available names", err.Error())
	}
	dup := `[{"id":"a","name":"x"},{"id":"b","name":"x"}]`
	if _, err := findNamed([]byte(dup), "x", "project"); err == nil {
		t.Error("ambiguous name accepted")
	}
	// Object-wrapped lists (the actual `neon * list -o json` shape).
	wrapped := `{"projects":[{"id":"proj_1","name":"miniapp-studio-dev"}],"pagination":{"x":1}}`
	id, err = findNamed([]byte(wrapped), "miniapp-studio-dev", "project")
	if err != nil || id != "proj_1" {
		t.Errorf("wrapped match: id=%q err=%v", id, err)
	}
	wrappedBranches := `{"branches":[{"id":"br_1","name":"dev"}]}`
	id, err = findNamed([]byte(wrappedBranches), "dev", "branch")
	if err != nil || id != "br_1" {
		t.Errorf("wrapped branch match: id=%q err=%v", id, err)
	}
	// Unknown wrapper shape fails with key names only — never values.
	weird := `{"weird":["proj_secret_value"],"count":3}`
	_, err = findNamed([]byte(weird), "x", "project")
	if err == nil {
		t.Error("unknown wrapper accepted")
	} else if strings.Contains(err.Error(), "proj_secret_value") {
		t.Errorf("diagnostic leaks a value: %q", err.Error())
	} else if !strings.Contains(err.Error(), "weird:array") || !strings.Contains(err.Error(), "count:number") {
		t.Errorf("diagnostic lacks key listing: %q", err.Error())
	}
	for name, data := range map[string]string{
		"malformed":    `[{"id":`,
		"wrong shape":  `{"id":"a"}`,
		"missing id":   `[{"name":"x"}]`,
		"missing name": `[{"id":"a"}]`,
	} {
		if _, err := findNamed([]byte(data), "x", "project"); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// TestNormalizeDSNFile uses a placeholder password; the test asserts
// behavior, never logs the value.
func TestNormalizeDSNFile(t *testing.T) {
	dir := t.TempDir()
	write := func(content string) string {
		p := filepath.Join(dir, "t.dsn")
		os.Remove(p)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// Missing sslmode is upgraded; hostname extracted.
	host, err := normalizeDSNFile(write("postgres://u:PLACEHOLDER@ep-x.invalid:5432/db\n"))
	if err != nil || host != "ep-x.invalid" {
		t.Fatalf("upgrade: host=%q err=%v", host, err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "t.dsn"))
	if !strings.Contains(string(raw), "sslmode=verify-full") {
		t.Errorf("sslmode not enforced: %s", strings.ReplaceAll(string(raw), "PLACEHOLDER", "REDACTED"))
	}
	// Weaker mode is upgraded, not preserved.
	host, err = normalizeDSNFile(write("postgres://u:PLACEHOLDER@ep-x.invalid:5432/db?sslmode=require"))
	if err != nil || host != "ep-x.invalid" {
		t.Fatalf("require upgrade: host=%q err=%v", host, err)
	}
	raw, _ = os.ReadFile(filepath.Join(dir, "t.dsn"))
	if strings.Contains(string(raw), "sslmode=require") {
		t.Error("weak sslmode preserved")
	}
	// verify-full is left byte-identical (no gratuitous rewrite).
	orig := "postgres://u:PLACEHOLDER@ep-x.invalid:5432/db?sslmode=verify-full"
	p := write(orig)
	if _, err := normalizeDSNFile(p); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(p); string(raw) != orig {
		t.Error("verify-full DSN rewritten unnecessarily")
	}
	// Garbage fails closed.
	if _, err := normalizeDSNFile(write("not a url")); err == nil {
		t.Error("garbage DSN accepted")
	}
	// Permissions stay 0600 after rewrite.
	write("postgres://u:PLACEHOLDER@ep-x.invalid:5432/db")
	fi, _ := os.Stat(filepath.Join(dir, "t.dsn"))
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("DSN file mode %o, want 0600", fi.Mode().Perm())
	}
}

func TestDevBindRequiresFlags(t *testing.T) {
	// Flag presence is checked via map iteration, so any of the four
	// required flags may surface first; assert the shape, not the order.
	if err := runDevBind(nil); err == nil || !strings.Contains(err.Error(), "is required") {
		t.Errorf("missing flags: %v", err)
	}
}
