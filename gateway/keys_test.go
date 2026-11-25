package gateway

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileKeys(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "keys.json")
	if err := os.WriteFile(good, []byte(`{"b-1":"00112233445566778899aabbccddeeff"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	keys, err := LoadKeys(good)
	if err != nil {
		t.Fatal(err)
	}
	k, err := keys.KeyFor("b-1")
	if err != nil || len(k) != 16 {
		t.Errorf("key = %x, %v", k, err)
	}
	if _, err := keys.KeyFor("nope"); err == nil {
		t.Error("unknown build accepted")
	}
	loose := filepath.Join(dir, "loose.json")
	if err := os.WriteFile(loose, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKeys(loose); err == nil {
		t.Error("world-readable keys accepted, want refusal")
	}
	short := filepath.Join(dir, "short.json")
	if err := os.WriteFile(short, []byte(`{"b":"abcd"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKeys(short); err == nil {
		t.Error("short key accepted, want refusal")
	}
	if _, err := LoadKeys(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("missing file accepted, want refusal")
	}
}

func TestLoadKeysDir(t *testing.T) {
	dir := t.TempDir()
	kdir := filepath.Join(dir, "keys")
	if err := os.Mkdir(kdir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kdir, "a.json"), []byte(`{"b-a":"00112233445566778899aabbccddeeff"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kdir, "b.json"), []byte(`{"b-b":"ffeeddccbbaa99887766554433221100"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kdir, "note.txt"), []byte("ignored"), 0o600); err != nil {
		t.Fatal(err)
	}
	keys, err := LoadKeysDir(kdir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := keys.KeyFor("b-a"); err != nil {
		t.Errorf("b-a: %v", err)
	}
	if _, err := keys.KeyFor("b-b"); err != nil {
		t.Errorf("b-b: %v", err)
	}
	if _, err := keys.KeyFor("nope"); err == nil {
		t.Error("unknown build accepted")
	}
	loose := filepath.Join(dir, "loose")
	if err := os.Mkdir(loose, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(loose, "x.json"), []byte(`{"b":"00112233445566778899aabbccddeeff"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKeysDir(loose); err == nil {
		t.Error("loose-permission member accepted, want refusal")
	}
}

func TestDeriveKey(t *testing.T) {
	secret := []byte("org-secret-0123456789abcdef")
	a1 := DeriveKey(secret, "build-1")
	a2 := DeriveKey(secret, "build-1")
	b := DeriveKey(secret, "build-2")
	other := DeriveKey([]byte("different-secret-0123456789"), "build-1")
	if len(a1) != 32 || string(a1) != string(a2) {
		t.Errorf("not deterministic 32-byte: %x", a1)
	}
	if string(a1) == string(b) || string(a1) == string(other) {
		t.Error("derived keys collide across builds/secrets")
	}
	var p KDFProvider
	if _, err := p.KeyFor("build-1"); err == nil {
		t.Error("empty secret accepted")
	}
	p.Secret = secret
	if _, err := p.KeyFor(""); err == nil {
		t.Error("empty build accepted")
	}
	if _, err := p.KeyFor("build-1"); err != nil {
		t.Errorf("KeyFor: %v", err)
	}
}

// TestReloadingKeysPicksUpNewFiles (F7): provisioned key files land
// without a gateway restart — every lookup re-reads the dir.
func TestReloadingKeysPicksUpNewFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keys-a.json"), []byte(`{"b-a":"00112233445566778899aabbccddeeff"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	keys := ReloadingKeys{Dir: dir}
	if _, err := keys.KeyFor("b-a"); err != nil {
		t.Fatalf("KeyFor(a) = %v", err)
	}
	if _, err := keys.KeyFor("b-b"); err == nil {
		t.Fatal("unknown build accepted")
	}
	// A provision lands mid-run: visible immediately, no restart.
	if err := os.WriteFile(filepath.Join(dir, "keys-b.json"), []byte(`{"b-b":"ffeeddccbbaa99887766554433221100"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := keys.KeyFor("b-b"); err != nil {
		t.Fatalf("KeyFor(b) after refresh = %v", err)
	}
	// A malformed dir errors the lookup instead of serving stale keys.
	if err := os.WriteFile(filepath.Join(dir, "keys-bad.json"), []byte(`{oops`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := keys.KeyFor("b-a"); err == nil {
		t.Error("lookup over malformed dir accepted, want error")
	}
}
