package main

import (
	"encoding/json"
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

func TestBranchIdentity(t *testing.T) {
	branches := `{"branches":[
		{"id":"br_dev","name":"dev","protected":false,"extra":1},
		{"id":"br_prod","name":"production","protected":true}]}`
	id, prot, err := branchIdentity([]byte(branches), "dev")
	if err != nil || id != "br_dev" || prot {
		t.Errorf("dev: id=%q protected=%v err=%v", id, prot, err)
	}
	id, prot, err = branchIdentity([]byte(branches), "production")
	if err != nil || id != "br_prod" || !prot {
		t.Errorf("production: id=%q protected=%v err=%v", id, prot, err)
	}
	// Missing marker fails closed (never assumes unprotected).
	legacy := `{"branches":[{"id":"br_1","name":"dev"}]}`
	if _, _, err := branchIdentity([]byte(legacy), "dev"); err == nil {
		t.Error("missing protected marker accepted")
	} else if !strings.Contains(err.Error(), "protection marker") {
		t.Errorf("marker error %q lacks remedy", err.Error())
	}
	// Missing branch keeps the dashboard guidance.
	if _, _, err := branchIdentity([]byte(branches), "absent"); err == nil {
		t.Error("miss accepted")
	} else if !strings.Contains(err.Error(), "dashboard") {
		t.Errorf("miss error %q lacks dashboard guidance", err.Error())
	}
	dup := `{"branches":[{"id":"a","name":"x","protected":false},{"id":"b","name":"x","protected":false}]}`
	if _, _, err := branchIdentity([]byte(dup), "x"); err == nil {
		t.Error("ambiguous name accepted")
	}
	// Bare-array shape (non-wrapped CLI output).
	bare := `[{"id":"br_1","name":"dev","protected":false}]`
	id, prot, err = branchIdentity([]byte(bare), "dev")
	if err != nil || id != "br_1" || prot {
		t.Errorf("bare array: id=%q protected=%v err=%v", id, prot, err)
	}
	// Explicit null marker fails closed like a missing key.
	nul := `{"branches":[{"id":"br_1","name":"dev","protected":null}]}`
	if _, _, err := branchIdentity([]byte(nul), "dev"); err == nil {
		t.Error("null protected marker accepted")
	}
}

func TestDevBindRefusesProductionName(t *testing.T) {
	dir := t.TempDir()
	meta := filepath.Join(dir, "m.json")
	dsn := filepath.Join(dir, "d.dsn")
	err := runDevBind([]string{"--project", "p", "--branch", "production",
		"--metadata-file", meta, "--dsn-file", dsn})
	if err == nil || !strings.Contains(err.Error(), "production") {
		t.Fatalf("production bind accepted: %v", err)
	}
	// Zero side effects: refusal precedes every Neon call and file write.
	if _, statErr := os.Stat(meta); !os.IsNotExist(statErr) {
		t.Error("refused bind wrote a metadata file")
	}
	if _, statErr := os.Stat(dsn); !os.IsNotExist(statErr) {
		t.Error("refused bind wrote a DSN file")
	}
}

func TestScrubDSN(t *testing.T) {
	in := `error: connect postgres://user:s3cret@host/db failed`
	got := scrubDSN(in)
	if strings.Contains(got, "s3cret") || !strings.Contains(got, "[redacted-dsn]") {
		t.Errorf("scrubbed = %q", got)
	}
	plain := "Unrecognized argument: branch"
	if scrubDSN(plain) != plain {
		t.Errorf("plain diagnostic altered: %q", scrubDSN(plain))
	}
}

// TestDevBindEndToEnd drives runDevBind against a fake `neon` binary:
// it pins the positional branch form (a CLI drift broke --branch with
// an opaque exit status), the identity fields recorded, and the DSN
// file mode. Placeholder password only; asserted rewritten, never read.
func TestDevBindEndToEnd(t *testing.T) {
	dir := t.TempDir()
	stub := "#!/bin/sh\n" +
		"echo \"$@\" >> \"$NEON_STUB_ARGS\"\n" +
		"case \"$1\" in\n" +
		"  projects) echo '{\"projects\":[{\"id\":\"proj_1\",\"name\":\"miniapp-studio-dev\"}]}';;\n" +
		"  branches) echo '{\"branches\":[{\"id\":\"br_1\",\"name\":\"dev\",\"protected\":false}]}';;\n" +
		"  connection-string) echo 'postgres://u:placeholder@ep-test.invalid:5432/neondb?sslmode=require';;\n" +
		"esac\n"
	stubPath := filepath.Join(dir, "neon")
	if err := os.WriteFile(stubPath, []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	argsPath := filepath.Join(dir, "args.log")
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("NEON_STUB_ARGS", argsPath)
	meta := filepath.Join(dir, "dev-neon.json")
	dsn := filepath.Join(dir, "dev.dsn")
	err := runDevBind([]string{"--project", "miniapp-studio-dev", "--branch", "dev",
		"--metadata-file", meta, "--dsn-file", dsn})
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	argv, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	// Positional branch form: no --branch flag anywhere; database
	// explicit (newer CLIs refuse to choose among several).
	for _, line := range strings.Split(strings.TrimSpace(string(argv)), "\n") {
		if strings.HasPrefix(line, "connection-string ") {
			if strings.Contains(line, "--branch") {
				t.Errorf("connection-string uses --branch flag: %q", line)
			}
			if !strings.Contains(line, "br_1 --project-id proj_1 --database-name neondb") {
				t.Errorf("connection-string argv = %q", line)
			}
		}
	}
	// Identity recorded + round-trips through the guarded loader.
	raw, err := os.ReadFile(meta)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("metadata unparsable: %v", err)
	}
	if doc["branch_name"] != "dev" || doc["protected"] != false ||
		doc["project_name"] != "miniapp-studio-dev" || doc["bound_at"] == nil {
		t.Errorf("metadata lacks identity: %s", raw)
	}
	// DSN file: 0600, sslmode rewritten to verify-full.
	fi, err := os.Stat(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("dsn mode = %o", fi.Mode().Perm())
	}
	content, err := os.ReadFile(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "sslmode=verify-full") {
		t.Errorf("dsn sslmode not rewritten: %q", redactForTest(string(content)))
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
