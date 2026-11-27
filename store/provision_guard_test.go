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

// TestAppRoleNameLiteral pins the production application-role name.
// Twin of TestLiveRoleNameLiteral in dbenv (a direct cross-package
// comparison would cycle the test binary now that store imports dbenv).
func TestAppRoleNameLiteral(t *testing.T) {
	if AppRoleName != "studio_app" {
		t.Errorf("AppRoleName %q drifts from studio_app", AppRoleName)
	}
}

// TestNoProdDbenvImport proves the shared environment contract stays
// out of production request paths. Only the test harness itself, the
// `studio dev` command surface (dev.go, bind.go, devserve.go,
// devbroker.go), the migrate boundary-enforcement point (migrate.go),
// and tests may import dbenv: it carries the fixed test credential
// and branch provisioning primitives.
func TestNoProdDbenvImport(t *testing.T) {
	needle := "miniapp-studio/dbenv\""
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
		if strings.HasPrefix(rel, "testdb"+string(filepath.Separator)) {
			return
		}
		if rel == filepath.Join("cmd", "studio", "dev.go") {
			return
		}
		// bind.go is the same `studio dev` command surface (owner-terminal
		// branch binding); identical justification as dev.go.
		if rel == filepath.Join("cmd", "studio", "bind.go") {
			return
		}
		// devserve.go is the same `studio dev` command surface
		// (owner-terminal serve-init/serve verbs for local UI review);
		// identical justification as dev.go. It is a main-package file,
		// so production request paths cannot import it by construction.
		if rel == filepath.Join("cmd", "studio", "devserve.go") {
			return
		}
		// migrate.go imports dbenv solely to enforce the production
		// guard on a binding it already requires: raw
		// migrate-against-prod must fail here so future promotion
		// machinery stays the only path. Main package, unimportable
		// from request paths, same as above.
		if rel == filepath.Join("cmd", "studio", "migrate.go") {
			return
		}
		// devbroker.go is the same `studio dev` command surface
		// (owner-terminal broker-init-kek/enroll/revoke/leases);
		// identical justification as devserve.go. It touches only
		// policy, production-guard, and pinned-pool symbols — no
		// test credential, no provisioning.
		if rel == filepath.Join("cmd", "studio", "devbroker.go") {
			return
		}
		// store/migrate.go: endpoint policy + pinned pools only, enforced
		// by TestMigrateDbenvSymbolRestriction below.
		if rel == filepath.Join("store", "migrate.go") {
			return
		}
		offenders = append(offenders, rel)
	})
	if len(offenders) > 0 {
		t.Errorf("dbenv imported by production files:\n  - %s", strings.Join(offenders, "\n  - "))
	}
}

// TestMigrateDbenvSymbolRestriction confines what the migrate runner
// may touch inside dbenv: endpoint policy, policy loaders, and pinned
// pool construction. Test credentials, role provisioning, grants, and
// run-database management stay unreachable from this file.
func TestMigrateDbenvSymbolRestriction(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "store", "migrate.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, banned := range []string{
		"TestRoleName", "TestRolePassword", "EnsureClusterRoles",
		"GrantTestPrivileges", "CreateRunDB", "NewRunDBName", "IsRunDB",
	} {
		if strings.Contains(body, "dbenv."+banned) {
			t.Errorf("store/migrate.go reaches dbenv.%s (test-only surface)", banned)
		}
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
