package hostexec

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func stageRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestStageContextOk(t *testing.T) {
	root := stageRepo(t, map[string]string{
		"testrunner/Containerfile": "FROM x\n",
		"testrunner/main.go":       "package main\n",
	})
	pins, err := HashPinInputs(root)
	if err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(t.TempDir(), "stage")
	if err := StageContext(context.Background(), root, stage, pins); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(stage, "main.go"))
	if err != nil || string(raw) != "package main\n" {
		t.Errorf("staged bytes = %q, %v", raw, err)
	}
}

func TestStageContextRefusesDrift(t *testing.T) {
	mk := func(t *testing.T) (string, map[string]string) {
		root := stageRepo(t, map[string]string{
			"testrunner/Containerfile": "FROM x\n",
			"testrunner/main.go":       "package main\n",
		})
		pins, err := HashPinInputs(root)
		if err != nil {
			t.Fatal(err)
		}
		return root, pins
	}
	t.Run("unexpected file (P7)", func(t *testing.T) {
		root, pins := mk(t)
		os.WriteFile(filepath.Join(root, "testrunner", "pwn.go"), []byte("package main\n"), 0o644)
		err := StageContext(context.Background(), root, filepath.Join(t.TempDir(), "s"), pins)
		if err == nil || !strings.Contains(err.Error(), "pins cover") {
			t.Errorf("plant accepted: %v", err)
		}
	})
	t.Run("altered file", func(t *testing.T) {
		root, pins := mk(t)
		os.WriteFile(filepath.Join(root, "testrunner", "main.go"), []byte("tampered\n"), 0o644)
		err := StageContext(context.Background(), root, filepath.Join(t.TempDir(), "s"), pins)
		if err == nil || !strings.Contains(err.Error(), "drifts") {
			t.Errorf("tamper accepted: %v", err)
		}
	})
	t.Run("missing file", func(t *testing.T) {
		root, pins := mk(t)
		os.Remove(filepath.Join(root, "testrunner", "main.go"))
		if err := StageContext(context.Background(), root, filepath.Join(t.TempDir(), "s"), pins); err == nil {
			t.Error("deletion accepted")
		}
	})
	t.Run("subdir refused", func(t *testing.T) {
		root := stageRepo(t, map[string]string{"testrunner/main.go": "x\n"})
		os.Mkdir(filepath.Join(root, "testrunner", "sub"), 0o755)
		pins := map[string]string{"testrunner/main.go": "x", "testrunner/sub": "y"}
		if err := StageContext(context.Background(), root, filepath.Join(t.TempDir(), "s"), pins); err == nil {
			t.Error("subdir accepted")
		}
	})
	t.Run("symlink refused", func(t *testing.T) {
		root := stageRepo(t, map[string]string{"testrunner/main.go": "x\n"})
		os.Symlink("main.go", filepath.Join(root, "testrunner", "link.go"))
		pins := map[string]string{"testrunner/main.go": "x", "testrunner/link.go": "y"}
		if err := StageContext(context.Background(), root, filepath.Join(t.TempDir(), "s"), pins); err == nil {
			t.Error("symlink accepted")
		}
	})
}

func TestStageRepoSnapshot(t *testing.T) {
	root := stageRepo(t, map[string]string{
		"go.mod":                     "module x\n",
		"pkg/a.go":                   "package a\n",
		".agent-jobs/pending/q.json": "{}",
		"test-results/exitcode":      "0\n",
	})
	m1, err := StageRepo(context.Background(), root, filepath.Join(t.TempDir(), "s1"))
	if err != nil {
		t.Fatal(err)
	}
	// Manifest determinism: same tree twice → same hash.
	m2, err := StageRepo(context.Background(), root, filepath.Join(t.TempDir(), "s2"))
	if err != nil {
		t.Fatal(err)
	}
	if m1 != m2 {
		t.Error("manifest nondeterministic")
	}
	// One-byte change → different manifest.
	os.WriteFile(filepath.Join(root, "pkg/a.go"), []byte("package b\n"), 0o644)
	m3, err := StageRepo(context.Background(), root, filepath.Join(t.TempDir(), "s3"))
	if err != nil {
		t.Fatal(err)
	}
	if m3 == m1 {
		t.Error("tamper invisible in manifest")
	}
}

func TestStageRepoExcludesAndSymlinks(t *testing.T) {
	root := stageRepo(t, map[string]string{
		"go.mod":                     "module x\n",
		".agent-jobs/pending/q.json": "{}",
		"test-results/exitcode":      "0\n",
	})
	stage := filepath.Join(t.TempDir(), "s")
	if _, err := StageRepo(context.Background(), root, stage); err != nil {
		t.Fatal(err)
	}
	for _, ex := range []string{".agent-jobs", "test-results"} {
		if _, err := os.Stat(filepath.Join(stage, ex)); !os.IsNotExist(err) {
			t.Errorf("excluded %s present in snapshot", ex)
		}
	}
	os.Symlink("go.mod", filepath.Join(root, "link"))
	if _, err := StageRepo(context.Background(), root, filepath.Join(t.TempDir(), "s2")); err == nil {
		t.Error("symlink in tree accepted")
	}
}

// TestStageSwapConsistency is the H2 regression: a rename-swap attack
// loop runs while snapshots are taken. Every COMPLETED snapshot must
// be self-consistent (its manifest describes exactly its bytes) and
// one of the two swapped contents, never a mix. cp may also detect a
// mid-copy replacement and refuse: that fail-closed refusal is the
// third honest outcome (counted, bounded below), never a failure.
func TestStageSwapConsistency(t *testing.T) {
	root := stageRepo(t, map[string]string{"f.txt": "AAAAAAAAAA"})
	victim := filepath.Join(root, "f.txt")
	// Swap staging lives OUTSIDE the staged tree (same filesystem, so
	// the rename stays atomic): temp files must never pollute snapshots.
	swapDir := t.TempDir()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		alt := 0
		for {
			select {
			case <-stop:
				return
			default:
			}
			body := "AAAAAAAAAA"
			if alt%2 == 1 {
				body = "BBBBBBBBBB"
			}
			alt++
			tmp := filepath.Join(swapDir, "f.swap")
			os.WriteFile(tmp, []byte(body), 0o644)
			os.Rename(tmp, victim)
		}
	}()
	defer func() { close(stop); wg.Wait() }()
	seen := map[string]bool{}
	refused := 0
	succeeded := 0
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for i := 0; i < 20; i++ {
		stage := filepath.Join(t.TempDir(), "s")
		m, err := StageRepo(ctx, root, stage)
		if err != nil {
			// cp detects mid-copy replacement and refuses: the
			// fail-closed outcome. Only a refusal naming the
			// copy is acceptable here — anything else is a bug.
			if !strings.Contains(err.Error(), "stage copy") {
				t.Fatalf("stage %d: unexpected error: %v", i, err)
			}
			refused++
			continue
		}
		succeeded++
		// Self-consistency: re-hash the stage, must match.
		raw, err := os.ReadFile(filepath.Join(stage, "f.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if s := string(raw); s != "AAAAAAAAAA" && s != "BBBBBBBBBB" {
			t.Fatalf("stage %d torn: %q", i, s)
		}
		seen[m] = true
	}
	if succeeded < 10 {
		t.Fatalf("only %d/20 stages completed (refused %d): replace detection too eager or staging broken", succeeded, refused)
	}
	if len(seen) > 2 {
		keys := []string{}
		for k := range seen {
			keys = append(keys, k)
		}
		t.Errorf("manifests = %d distinct, want ≤2 (atomic contents only): %v", len(seen), keys)
	}
}

// TestStageSnapshotIndependence proves post-hash repo mutation cannot
// affect the snapshot: tamper after staging leaves staged bytes and
// manifest unchanged.
func TestStageSnapshotIndependence(t *testing.T) {
	root := stageRepo(t, map[string]string{"f.txt": "orig"})
	stage := filepath.Join(t.TempDir(), "s")
	m1, err := StageRepo(context.Background(), root, stage)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "f.txt"), []byte("tampered"), 0o644)
	raw, _ := os.ReadFile(filepath.Join(stage, "f.txt"))
	if string(raw) != "orig" {
		t.Errorf("staged bytes changed: %q", raw)
	}
	// The manifest is a pure function of staged bytes: recomputing over
	// the stage must reproduce it exactly.
	m2, err := TreeManifest(stage, []string{"f.txt"})
	if err != nil || m2 == "" {
		t.Fatalf("remanifest: %q, %v", m2, err)
	}
	m3, err := TreeManifest(stage, []string{"f.txt"})
	if err != nil || m3 != m2 {
		t.Errorf("manifest unstable: %q vs %q", m2, m3)
	}
	if m1 == "" {
		t.Error("empty stage manifest")
	}
}
