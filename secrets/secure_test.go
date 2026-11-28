package secrets

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestOpenSecureRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if f, err := OpenSecure(link); err == nil {
		f.Close()
		t.Fatal("symlink to 0600 file accepted")
	}
	if _, _, err := ReadSecureFile(link, MaxValueBytes); err == nil {
		t.Fatal("ReadSecureFile followed symlink")
	}
	if err := CheckFileMode(link); err == nil {
		t.Fatal("CheckFileMode passed a symlink")
	}
}

func TestOpenSecureRefusesNonRegular(t *testing.T) {
	dir := t.TempDir()
	if f, err := OpenSecure(dir); err == nil {
		f.Close()
		t.Fatal("directory accepted")
	}
}

// TestOpenSecureReadsAuditedBytes is the deterministic TOCTOU
// proof: the descriptor is opened (audited), the path is then
// swapped underneath, and the read still returns exactly the
// audited file's bytes — the audit and the read cannot diverge.
func TestOpenSecureReadsAuditedBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "live")
	fileA := filepath.Join(dir, "a")
	if err := os.WriteFile(fileA, []byte("BYTES-A"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("BYTES-A"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := OpenSecure(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	// Swap the path to different content after the audit.
	swap := filepath.Join(dir, "b")
	if err := os.WriteFile(swap, []byte("BYTES-B-OUTSIDE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(swap, path); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "BYTES-A" {
		t.Fatalf("read after swap = %q, want audited bytes", got)
	}
	after, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("descriptor identity changed under read")
	}
}

// TestReadSecureFileSwapLoop hammers rename-swaps concurrently with
// reads: every successful read returns exactly one generation's
// complete bytes — never mixed, partial, or unaudited content.
func TestReadSecureFileSwapLoop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "live")
	genA, genB := strings.Repeat("A", 4096), strings.Repeat("B", 4096)
	if err := os.WriteFile(path, []byte(genA), 0o600); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		flip := false
		for {
			select {
			case <-stop:
				return
			default:
			}
			tmp := filepath.Join(dir, "swap")
			content := genA
			if flip {
				content = genB
			}
			flip = !flip
			if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
				return
			}
			_ = os.Rename(tmp, path)
		}
	}()
	defer func() { close(stop); wg.Wait() }()
	for i := 0; i < 300; i++ {
		raw, _, err := ReadSecureFile(path, MaxValueBytes)
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		s := string(raw)
		if s != genA && s != genB {
			t.Fatalf("read %d returned %d mixed bytes", i, len(s))
		}
	}
}

func TestReadSecureFileBounds(t *testing.T) {
	p := filepath.Join(t.TempDir(), "big")
	if err := os.WriteFile(p, make([]byte, MaxValueBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadSecureFile(p, MaxValueBytes); err == nil {
		t.Fatal("oversize file accepted")
	}
	if _, _, err := ReadSecureFile(filepath.Join(t.TempDir(), "missing"), MaxValueBytes); err == nil {
		t.Fatal("missing file accepted")
	}
	loose := filepath.Join(t.TempDir(), "loose")
	if err := os.WriteFile(loose, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadSecureFile(loose, MaxValueBytes); err == nil {
		t.Fatal("0644 file accepted")
	}
}
