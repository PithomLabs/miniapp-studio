package hostexec

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Staging snapshots agent-writable trees into owner-side StateDir. The
// rule is copy-then-verify: bytes are copied first and the STAGED copy
// is hashed and compared, so a swap mid-copy breaks verification
// instead of smuggling bytes into the build. Verification reads only
// staged (agent-unreachable) bytes; the live tree is never trusted
// after the copy starts.

// Runtime dirs excluded from run snapshots. Fixed names, documented:
// the queue and prior results are broker/agent runtime state, not
// build inputs. Everything else — including .git, which the
// provenance record reads — is snapshotted.
var stageExcludes = []string{".agent-jobs", "test-results"}

// StageContext snapshots the image build context and verifies it
// against install-time pins: exact file set, exact hashes. Any drift
// (new, missing, or altered file) refuses the build.
func StageContext(ctx context.Context, repoRoot, stageDir string, pins map[string]string) error {
	if err := freshDir(stageDir); err != nil {
		return err
	}
	src := filepath.Join(repoRoot, RunnerContext)
	if exit, _, errb, _, _ := Run(ctx, []string{"cp", "-a", src + string(filepath.Separator) + ".", stageDir}, 5*time.Minute); exit != 0 {
		return fmt.Errorf("stage copy: %s", errb)
	}
	// Verify the STAGED copy: flat, regular-only, exact pin match.
	entries, err := os.ReadDir(stageDir)
	if err != nil {
		return fmt.Errorf("stage read: %w", err)
	}
	got := map[string]string{}
	for _, e := range entries {
		if e.IsDir() {
			return fmt.Errorf("staged subdir %q refused", e.Name())
		}
		fi, err := os.Lstat(filepath.Join(stageDir, e.Name()))
		if err != nil || !fi.Mode().IsRegular() {
			return fmt.Errorf("staged entry %q is not a regular file", e.Name())
		}
		rel := RunnerContext + "/" + e.Name()
		got[rel] = ""
	}
	if len(got) != len(pins) {
		return fmt.Errorf("staged %d files, pins cover %d (re-pin under owner review)", len(got), len(pins))
	}
	for rel := range got {
		base := strings.TrimPrefix(rel, RunnerContext+"/")
		raw, err := os.ReadFile(filepath.Join(stageDir, base))
		if err != nil {
			return fmt.Errorf("stage read %s: %w", base, err)
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != pins[rel] {
			return fmt.Errorf("staged %s drifts from pins (re-pin under owner review)", rel)
		}
	}
	return nil
}

// StageRepo snapshots the whole repo for one run and returns the tree
// manifest hash. The snapshot — never the live tree — is what the
// container mounts. Mid-run repo mutation cannot affect the run;
// post-hoc tamper cannot rewrite the frozen manifest.
func StageRepo(ctx context.Context, repoRoot, stageDir string) (string, error) {
	if err := freshDir(stageDir); err != nil {
		return "", err
	}
	if exit, _, errb, _, _ := Run(ctx, []string{"cp", "-a", repoRoot + string(filepath.Separator) + ".", stageDir}, 10*time.Minute); exit != 0 {
		return "", fmt.Errorf("stage copy: %s", errb)
	}
	for _, ex := range stageExcludes {
		if err := os.RemoveAll(filepath.Join(stageDir, ex)); err != nil {
			return "", fmt.Errorf("stage exclude %s: %w", ex, err)
		}
	}
	// Fail closed on anything but files and dirs (symlinks especially:
	// the snapshot must be self-contained bytes, never pointers out).
	var files []string
	err := filepath.WalkDir(stageDir, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			return nil
		}
		fi, err := os.Lstat(p)
		if err != nil || !fi.Mode().IsRegular() {
			rel, _ := filepath.Rel(stageDir, p)
			return fmt.Errorf("staged non-file %q refused", rel)
		}
		rel, _ := filepath.Rel(stageDir, p)
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return "", err
	}
	return TreeManifest(stageDir, files)
}

// TreeManifest hashes sorted (path, mode, size, bytes) over files.
func TreeManifest(root string, files []string) (string, error) {
	sort.Strings(files)
	h := sha256.New()
	for _, rel := range files {
		p := filepath.Join(root, rel)
		fi, err := os.Stat(p)
		if err != nil {
			return "", fmt.Errorf("manifest stat %s: %w", rel, err)
		}
		fmt.Fprintf(h, "%s\n%d\n%d\n", rel, fi.Mode().Perm(), fi.Size())
		raw, err := os.ReadFile(p)
		if err != nil {
			return "", fmt.Errorf("manifest read %s: %w", rel, err)
		}
		h.Write(raw)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func freshDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("stage clean: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("stage mkdir: %w", err)
	}
	return nil
}
