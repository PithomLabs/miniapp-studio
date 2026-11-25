package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestCollect(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.diff"), []byte("diff-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "spec.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Collect(dir, Manifest{Entries: []ManifestEntry{
		{Path: "app.diff", Kind: "diff"},
		{Path: "sub/spec.json", Kind: "spec"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("artifacts = %d", len(got))
	}
	sum := sha256.Sum256([]byte("diff-bytes"))
	if got[0].Digest != "sha256:"+hex.EncodeToString(sum[:]) || got[0].Size != 10 {
		t.Errorf("artifact = %+v", got[0])
	}
	if string(got[1].Data) != "{}" || got[1].Kind != "spec" {
		t.Errorf("artifact = %+v", got[1])
	}
}

func TestCollectRefuses(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	big := make([]byte, MaxArtifactBytes+1)
	if err := os.WriteFile(filepath.Join(dir, "big.bin"), big, 0o600); err != nil {
		t.Fatal(err)
	}
	cases := map[string]ManifestEntry{
		"absolute":   {Path: "/etc/passwd", Kind: "bundle"},
		"traversal":  {Path: "../secret", Kind: "bundle"},
		"nested":     {Path: "d/../../secret", Kind: "bundle"},
		"symlink":    {Path: "link", Kind: "bundle"},
		"directory":  {Path: "d", Kind: "bundle"},
		"oversize":   {Path: "big.bin", Kind: "bundle"},
		"bad kind":   {Path: "big.bin", Kind: "binary"},
		"missing":    {Path: "nope.txt", Kind: "diff"},
		"empty path": {Path: "", Kind: "diff"},
	}
	for name, entry := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Collect(dir, Manifest{Entries: []ManifestEntry{entry}}); err == nil {
				t.Error("collect accepted, want refusal")
			}
		})
	}
}

// TestCollectRefusesInsideSymlink (M8 RED): even a symlink resolving
// inside the workdir must be refused — the manifest names real files,
// and descriptor-pinned reads cannot traverse links at all.
func TestCollectRefusesInsideSymlink(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "real.txt"), []byte("inside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "real.txt"), filepath.Join(dir, "alias")); err != nil {
		t.Fatal(err)
	}
	if _, err := Collect(dir, Manifest{Entries: []ManifestEntry{{Path: "alias", Kind: "diff"}}}); err == nil {
		t.Error("inside symlink accepted, want refusal")
	}
}

// TestCollectSwapNeverEscapes (M8): while a swapper alternates the
// entry between a real file and an outside symlink, collection serves
// only inside bytes or errors — never outside bytes (fd-pinned reads
// are immune to path substitution after open).
func TestCollectSwapNeverEscapes(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("OUTSIDE-BYTES"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "swap")
	stage := filepath.Join(dir, ".swap-stage")
	_ = os.WriteFile(stage, []byte("inside"), 0o600)
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = os.Remove(target)
			if i%2 == 0 {
				// Atomic file swap (hardlink): readers see a
				// complete inode or absence — never a tear.
				_ = os.Link(stage, target)
			} else {
				_ = os.Symlink(filepath.Join(outside, "secret"), target)
			}
			i++
		}
	}()
	defer func() { close(stop); wg.Wait() }()
	for i := 0; i < 200; i++ {
		arts, err := Collect(dir, Manifest{Entries: []ManifestEntry{{Path: "swap", Kind: "diff"}}})
		if err != nil {
			continue
		}
		if len(arts) != 1 || string(arts[0].Data) != "inside" {
			t.Fatalf("collect served %q, want inside bytes or error", arts[0].Data)
		}
	}
}
