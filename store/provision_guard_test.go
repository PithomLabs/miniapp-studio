package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoHarnessRotation scans every test file for the explicit rotation
// entry point. RotateAppRole mutates the persistent live credential, so
// exactly one test file may reference it: the Boundary B run
// provisioner, where rotation is the documented administrative
// operation on a serial run. Any other reference fails the suite —
// this is the structural half of U5.12 invariant 2 (the behavioral
// half is the rotation-mechanism test on scratch roles).
//
// The needles are assembled from parts so this scanner never matches
// its own source.
func TestNoHarnessRotation(t *testing.T) {
	needle := "Rotate" + "AppRole("
	allowed := filepath.Join("e2e", "boundary_b", "provision_test.go")
	root := repoRoot(t)
	var offenders []string
	walkGo(t, root, func(path string, isTest bool, body string) {
		if !isTest || !strings.Contains(body, needle) {
			return
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		if rel != allowed {
			offenders = append(offenders, rel)
		}
	})
	if len(offenders) > 0 {
		t.Errorf("rotation entry point referenced by test files (want only %s):\n  - %s",
			allowed, strings.Join(offenders, "\n  - "))
	}
}

// TestNoProdTestDBImport proves the test-only provisioning package
// stays out of production code: no non-test Go file may import it.
func TestNoProdTestDBImport(t *testing.T) {
	needle := "miniapp-studio/testdb\""
	root := repoRoot(t)
	var offenders []string
	walkGo(t, root, func(path string, isTest bool, body string) {
		if isTest || !strings.Contains(body, needle) {
			return
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		offenders = append(offenders, rel)
	})
	if len(offenders) > 0 {
		t.Errorf("testdb imported by production files:\n  - %s", strings.Join(offenders, "\n  - "))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// Tests execute with CWD = the package directory (<root>/store).
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(wd)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(abs)
}

func walkGo(t *testing.T, root string, visit func(path string, isTest bool, body string)) {
	t.Helper()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", ".hg", "node_modules", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		visit(path, strings.HasSuffix(info.Name(), "_test.go"), string(raw))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
