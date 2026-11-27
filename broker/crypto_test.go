package broker

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testKEK(t *testing.T) []byte {
	t.Helper()
	kek := make([]byte, kekBytes)
	if _, err := rand.Read(kek); err != nil {
		t.Fatal(err)
	}
	return kek
}

func TestSealOpenRoundTrip(t *testing.T) {
	kek := testKEK(t)
	ct, err := Seal(kek, "tenant-a", "google", []byte("refresh-xyz"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ct, []byte("refresh-xyz")) {
		t.Fatal("ciphertext contains plaintext")
	}
	plain, err := Open(kek, "tenant-a", "google", ct)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != "refresh-xyz" {
		t.Fatalf("round trip = %q", plain)
	}
	// Random nonces: same input seals differently.
	ct2, _ := Seal(kek, "tenant-a", "google", []byte("refresh-xyz"))
	if bytes.Equal(ct, ct2) {
		t.Fatal("deterministic seal: nonce reuse suspected")
	}
}

func TestSealBindsTenantAndProvider(t *testing.T) {
	kek := testKEK(t)
	ct, err := Seal(kek, "tenant-a", "google", []byte("s3cret"))
	if err != nil {
		t.Fatal(err)
	}
	// A row confused across tenants or providers must fail closed,
	// never mint for the wrong owner.
	if _, err := Open(kek, "tenant-b", "google", ct); err == nil {
		t.Fatal("cross-tenant open accepted")
	}
	if _, err := Open(kek, "tenant-a", "microsoft", ct); err == nil {
		t.Fatal("cross-provider open accepted")
	}
	if _, err := Open(testKEK(t), "tenant-a", "google", ct); err == nil {
		t.Fatal("wrong-KEK open accepted")
	}
	// Tamper with version, nonce, and body independently.
	for name, mutate := range map[string]func([]byte){
		"version": func(b []byte) { b[0] ^= 0xff },
		"nonce":   func(b []byte) { b[3] ^= 0x01 },
		"body":    func(b []byte) { b[len(b)-1] ^= 0x01 },
		"trunc":   func(b []byte) {},
	} {
		cp := bytes.Clone(ct)
		mutate(cp)
		if name == "trunc" {
			cp = cp[:len(cp)/2]
		}
		if _, err := Open(kek, "tenant-a", "google", cp); err == nil {
			t.Fatalf("tampered blob (%s) accepted", name)
		}
	}
}

func TestSealRejectsBadInputs(t *testing.T) {
	kek := testKEK(t)
	if _, err := Seal(make([]byte, 16), "t", "p", []byte("x")); err == nil {
		t.Error("short KEK accepted")
	}
	if _, err := Seal(kek, "", "p", []byte("x")); err == nil {
		t.Error("empty tenant accepted")
	}
	if _, err := Seal(kek, "t", "p", nil); err == nil {
		t.Error("empty plaintext accepted")
	}
	if _, err := Seal(kek, "t", "p", make([]byte, maxPlaintext+1)); err == nil {
		t.Error("oversize plaintext accepted")
	}
}

func TestLoadKEK(t *testing.T) {
	kek := testKEK(t)
	hexed := []byte(hex.EncodeToString(kek))
	strict := filepath.Join(t.TempDir(), "kek")
	if err := os.WriteFile(strict, append(hexed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	got, id, err := LoadKEK(strict)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, kek) || id == "" || id != KeyID(kek) {
		t.Fatalf("KEK mismatch: id=%q", id)
	}
	loose := filepath.Join(t.TempDir(), "kek-loose")
	if err := os.WriteFile(loose, hexed, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadKEK(loose); err == nil {
		t.Fatal("0644 KEK accepted")
	}
	for name, body := range map[string]string{
		"short":   "abcd",
		"nonhex":  strings.Repeat("zz", 32),
		"longhex": strings.Repeat("ab", 33),
		"empty":   "",
	} {
		p := filepath.Join(t.TempDir(), "kek-"+name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := LoadKEK(p); err == nil {
			t.Fatalf("malformed KEK (%s) accepted", name)
		}
	}
}

func TestNewValidatesKEK(t *testing.T) {
	if _, err := New(make([]byte, 16), &pgxpool.Pool{}, nil); err == nil {
		t.Fatal("short KEK accepted")
	}
	if _, err := New(testKEK(t), nil, nil); err == nil {
		t.Fatal("nil pool accepted")
	}
	// Nil exchanger is allowed (non-mint verbs), but Mint fails
	// closed before touching the database or the credential.
	b, err := New(testKEK(t), &pgxpool.Pool{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := b.Mint(context.Background(), "t", "p", "c", strings.Repeat("a", 64), time.Minute, true); err == nil || !strings.Contains(err.Error(), "exchanger") {
		t.Fatalf("mint without exchanger err = %v", err)
	}
	// Nil KEK is allowed (non-crypto verbs); Enroll fails closed.
	b2, err := New(nil, &pgxpool.Pool{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := b2.Enroll(context.Background(), "t", "p", []byte("r")); err == nil {
		t.Fatal("enroll without KEK accepted")
	}
}
