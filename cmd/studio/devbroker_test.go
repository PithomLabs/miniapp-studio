package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevBrokerInitKEKWrites0600Once(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kek")
	var buf bytes.Buffer
	if err := runDevBrokerInitKEK([]string{"--kek-file", path}, &buf); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Fatalf("KEK mode = %04o, want 0600", perm)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// 64 hex chars + newline; the value itself must not echo.
	if len(strings.TrimSpace(string(raw))) != 64 {
		t.Fatalf("KEK file holds %d chars, want 64", len(strings.TrimSpace(string(raw))))
	}
	if strings.Contains(buf.String(), strings.TrimSpace(string(raw))) {
		t.Fatal("init-kek printed the key material")
	}
	// Second run refuses to overwrite.
	var buf2 bytes.Buffer
	if err := runDevBrokerInitKEK([]string{"--kek-file", path}, &buf2); err == nil {
		t.Fatal("overwrite accepted")
	}
}

func TestDevBrokerVerbsValidateArgs(t *testing.T) {
	var buf bytes.Buffer
	if err := runDevBrokerEnroll(nil, &buf); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("enroll without args err = %v", err)
	}
	if err := runDevBrokerRevoke([]string{"--tenant", "t"}, &buf); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("revoke without provider err = %v", err)
	}
	if err := runDevBrokerLeases(nil, &buf); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("leases without tenant err = %v", err)
	}
	// Enroll audits the KEK file before touching the network.
	var buf2 bytes.Buffer
	err := runDevBrokerEnroll([]string{"--kek-file", "/nonexistent", "--tenant", "t", "--provider", "p", "--refresh-file", "/nonexistent"}, &buf2)
	if err == nil || !strings.Contains(err.Error(), "KEK") {
		t.Fatalf("enroll with missing KEK err = %v", err)
	}
}
