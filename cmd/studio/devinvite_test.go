package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevInviteVerbsValidateArgs(t *testing.T) {
	var buf bytes.Buffer
	if err := runDevInviteCreate(nil, &buf); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("create without args err = %v", err)
	}
	if err := runDevInviteAccept(nil, &buf); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("accept without args err = %v", err)
	}
	if err := runDevInviteRevoke([]string{"--tenant", "t"}, &buf); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("revoke without id err = %v", err)
	}
	if err := runDevInviteResend(nil, &buf); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("resend without args err = %v", err)
	}
	if err := runDevInviteList(nil, &buf); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("list without tenant err = %v", err)
	}
	// Accept audits the token file before touching the network.
	var buf2 bytes.Buffer
	loose := filepath.Join(t.TempDir(), "tok")
	if err := os.WriteFile(loose, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runDevInviteAccept([]string{"--token-file", loose, "--session-token-file", filepath.Join(t.TempDir(), "s")}, &buf2)
	if err == nil || !strings.Contains(err.Error(), "token file") {
		t.Fatalf("accept with 0644 token err = %v", err)
	}
}

func TestReserveFillTokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tok")
	resv, err := reserveTokenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil || fi.Mode().Perm() != 0o600 {
		resv.Close()
		t.Fatalf("mode = %v, %v", fi, err)
	}
	// Second reservation of the same path refuses.
	if resv2, err := reserveTokenFile(path); err == nil {
		resv2.Close()
		t.Fatal("double reservation accepted")
	}
	if err := fillTokenFile(resv, "secret-1"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if strings.TrimSpace(string(raw)) != "secret-1" {
		t.Fatal("filled content mismatch")
	}
}
