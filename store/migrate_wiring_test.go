package store

import (
	"context"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp-studio/dbenv"
)

// TestMigrateEnforcesPolicyFirst pins the wiring: migrateTo validates
// the DSN against the branch policy BEFORE constructing anything that
// dials. Pure (no connection attempted); the refusal marker proves it.
func TestMigrateEnforcesPolicyFirst(t *testing.T) {
	pol := dbenv.EndpointPolicy{
		ProjectID: "proj_test", BranchID: "br_test",
		Hosts: []string{"ep-test-0001.invalid"},
	}
	err := migrateTo(context.Background(),
		"postgres://maint:pw@127.0.0.1:5434/postgres?sslmode=verify-full", pol, nil)
	if err == nil {
		t.Fatal("unbound DSN accepted by migrateTo")
	}
	if !strings.Contains(err.Error(), "unapproved endpoint") {
		t.Errorf("error = %q, want policy refusal before any migrate work", err.Error())
	}
}
