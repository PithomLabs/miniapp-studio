package boundaryb

// Committed acceptance-run provisioner. Replaces the throwaway ZZ temp
// test: same tested helpers (testkit.MigrateUp, store.Migrate,
// EnsureAppRole, UpsertOperator), reviewable, repeatable. Skips without
// its env so ordinary `go test ./...` never touches it.
//
// Run once per clean database, BEFORE any live pass:
//
//	BOUNDARY_DSN_FILE=<owner-dsn> BOUNDARY_DIR=<run-dir> \
//	  go test ./e2e/boundary_b/ -run TestProvisionRun -count=1 -v
//
// BOUNDARY_DIR must already hold approle (hex), tokenA, tokenB.
// Operator logins/tokens are provisioned separately via the studio
// provision CLI (see scripts/provision_run.sh), which owns the
// operators file. This test never starts servers or browsers.

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp/testkit"
)

func TestProvisionRun(t *testing.T) {
	dsnFile := os.Getenv("BOUNDARY_DSN_FILE")
	dir := os.Getenv("BOUNDARY_DIR")
	if dsnFile == "" || dir == "" {
		t.Skip("boundary env unset")
	}
	raw, err := os.ReadFile(dsnFile)
	if err != nil {
		t.Fatal(err)
	}
	dsn := strings.TrimSpace(string(raw))
	testkit.MigrateUp(t, dsn)
	ctx := context.Background()
	owner, err := store.NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	pwRaw, err := os.ReadFile(dir + "/approle")
	if err != nil {
		t.Fatal(err)
	}
	// Fixed run password from file, applied by EXPLICIT rotation: run
	// provisioning is an administrative operation (U5.12 invariant 2),
	// runs are serial, and the run owns this credential via app.dsn.
	// Ordinary test harnesses must never call RotateAppRole.
	if err := store.RotateAppRole(ctx, owner, strings.TrimSpace(string(pwRaw))); err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx, dsn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := store.UpsertOperator(ctx, owner, "t-demo", "demo-op", true); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertOperator(ctx, owner, "t-demo-b", "demo-op-b", true); err != nil {
		t.Fatal(err)
	}
	t.Log("provisioned: framework + studio schema, app role, demo operators")
}
