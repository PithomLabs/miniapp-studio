package testdb

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/dbenv"
)

// Pure contract tests (endpoint allowlist, run pattern, fixed password,
// live-role tracking) moved to dbenv with the code they cover.

func TestAppDSNRefusesNonRun(t *testing.T) {
	// AppDSN fatals on refusal, so a refusal here would kill the test.
	// Instead assert the guard predicate directly on hostile inputs.
	for _, db := range []string{"postgres", "studio_boundary_b", "app", "studio_test_run_x"} {
		u, err := url.Parse("postgres://maint@ep-test-0001.invalid:5432/" + db + "?sslmode=verify-full")
		if err != nil {
			t.Fatal(err)
		}
		if dbenv.IsRunDB(strings.TrimPrefix(u.Path, "/")) {
			t.Errorf("guard admits database %q", db)
		}
	}
}

func TestAppDSNShape(t *testing.T) {
	dsn := "postgres://maint@ep-test-0001.invalid:5432/studio_test_run_9_9_abcdef01?sslmode=verify-full"
	got := AppDSN(t, dsn)
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.User.Username() != dbenv.TestRoleName {
		t.Error("app DSN user is not the test role")
	}
	if pw, ok := parsed.User.Password(); !ok || pw != dbenv.TestRolePassword {
		t.Error("app DSN credential is not the fixed test password")
	}
	if parsed.Path != "/studio_test_run_9_9_abcdef01" {
		t.Errorf("app DSN database moved: %q", parsed.Path)
	}
}

// TestGrantRefusesNonRunDB proves invariant 7 at the boundary: grants
// aimed at a non-run database are refused before any GRANT executes.
// It derives maint credentials from the run DSN and retargets the
// always-present postgres database, so it is independent of test order
// and of the TEST_DATABASE_URL guardrail swap.
func TestGrantRefusesNonRunDB(t *testing.T) {
	run := RunDSN(t)
	parsed, err := url.Parse(run)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/postgres"
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := dbenv.GrantTestPrivileges(ctx, pool); err == nil {
		t.Error("grants on a non-run database accepted, want refusal")
	} else if !strings.Contains(err.Error(), "non-run") {
		t.Errorf("error = %v, want non-run refusal", err)
	}
}

// TestRunDSNSingleton provisions the process run database once and pins
// the singleton: repeated calls return the identical run-pattern DSN.
func TestRunDSNSingleton(t *testing.T) {
	first := RunDSN(t)
	second := RunDSN(t)
	if first != second {
		t.Error("RunDSN returned distinct databases within one process")
	}
	parsed, err := url.Parse(first)
	if err != nil {
		t.Fatal(err)
	}
	MustTestDB(t, strings.TrimPrefix(parsed.Path, "/"))
}

// TestRunDSNRejectsUnapprovedChild exists only as the subprocess target
// for TestEndpointRejectionPrecedesDial: in a normal suite run it
// provisions (or reuses) the run database and passes.
func TestRunDSNRejectsUnapprovedChild(t *testing.T) {
	_ = RunDSN(t)
}

// TestEndpointRejectionPrecedesDial proves end to end that a
// mispointed TEST_DATABASE_URL is refused before any connection is
// attempted: the child process gets a policy bound to one branch but
// a URL aimed elsewhere at an unresolvable host, so any dial would
// fail with a DNS error instead of the policy refusal.
func TestEndpointRejectionPrecedesDial(t *testing.T) {
	if os.Getenv("U512_ENDPOINT_CHILD") != "" {
		t.Skip("child process")
	}
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test",
		"-run", "TestRunDSNRejectsUnapprovedChild", "-count=1", ".")
	cmd.Dir = filepath.Dir(thisFile)
	env := os.Environ()
	env = append(env, "U512_ENDPOINT_CHILD=1",
		"NEON_PROJECT_ID=proj_test_123",
		"NEON_BRANCH_ID=br_test_abc",
		"NEON_ENDPOINT_HOSTS=ep-test-0001.invalid",
		"TEST_DATABASE_URL=postgres://maint:pw@u512-unapproved.invalid:5432/postgres?sslmode=verify-full")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("child with unapproved endpoint succeeded:\n%s", out)
	}
	if !strings.Contains(string(out), "unapproved endpoint") {
		t.Fatalf("child failed without the policy refusal:\n%s", out)
	}
	if strings.Contains(string(out), "no such host") ||
		strings.Contains(string(out), "connection refused") {
		t.Fatalf("child attempted a dial before refusing:\n%s", out)
	}
}
