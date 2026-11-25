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

	"github.com/PithomLabs/miniapp-studio/store"
)

// The literal live-role name must track the production constant;
// importing store here is safe (store never imports testdb).
func TestLiveRoleNameMatches(t *testing.T) {
	if liveRoleName() != store.AppRoleName {
		t.Errorf("testdb live role %q drifts from store %q", liveRoleName(), store.AppRoleName)
	}
}

func TestFixedPasswordHexRule(t *testing.T) {
	if !rxHexPass.MatchString(TestRolePassword) {
		t.Error("fixed test password violates the hex rule")
	}
	if TestRoleName == liveRoleName() {
		t.Error("test role collides with the live role name")
	}
}

func TestRunPatternGuard(t *testing.T) {
	good := []string{
		"studio_test_run_1234_1764000000_ab12cd34",
		"studio_test_run_1_1_00",
	}
	for _, name := range good {
		if !rxRunDB.MatchString(name) {
			t.Errorf("%q rejected, want accept", name)
		}
	}
	bad := []string{
		"studio_boundary_b", "studio_boundary_b_b1", "postgres",
		"studio_test_run_", "studio_test_run_x_y_z",
		"studio_test_run_1_2_zz; DROP", "STUDIO_TEST_RUN_1_2_AB",
		"studio_test_run_1_2_ab;--",
	}
	for _, name := range bad {
		if rxRunDB.MatchString(name) {
			t.Errorf("%q accepted, want refusal", name)
		}
	}
}

func TestAppDSNRefusesNonRun(t *testing.T) {
	// AppDSN fatals on refusal, so a refusal here would kill the test.
	// Instead assert the guard predicate directly on hostile inputs.
	for _, db := range []string{"postgres", "studio_boundary_b", "app", "studio_test_run_x"} {
		u, err := url.Parse("postgres://maint@127.0.0.1:5434/" + db)
		if err != nil {
			t.Fatal(err)
		}
		if rxRunDB.MatchString(strings.TrimPrefix(u.Path, "/")) {
			t.Errorf("guard admits database %q", db)
		}
	}
}

func TestAppDSNShape(t *testing.T) {
	dsn := "postgres://maint@127.0.0.1:5434/studio_test_run_9_9_abcdef01"
	got := AppDSN(t, dsn)
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.User.Username() != TestRoleName {
		t.Error("app DSN user is not the test role")
	}
	if pw, ok := parsed.User.Password(); !ok || pw != TestRolePassword {
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
	if err := GrantTestPrivileges(ctx, pool); err == nil {
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

// TestEndpointAllowlist is pure: CheckTestEndpoint opens no connection,
// so every refusal below proves the harness never dials the endpoint.
func TestEndpointAllowlist(t *testing.T) {
	good := []string{
		"postgres://maint:pw@127.0.0.1:5434/postgres",
		"postgres://maint:pw@localhost:5434/anything",
		"postgresql://maint:pw@[::1]:5434/x",
	}
	for _, raw := range good {
		if _, err := CheckTestEndpoint(raw); err != nil {
			t.Errorf("accept %q: %v", redactURL(raw), err)
		}
	}
	bad := []string{
		"postgres://maint:pw@db.prod.example:5434/postgres",
		"postgres://maint:pw@10.0.0.5:5434/postgres",
		"postgres://maint:pw@127.0.0.1:5433/postgres",
		"postgres://maint:pw@127.0.0.1:5432/postgres",
		"postgres://maint:pw@127.0.0.1/postgres",
		"postgres://maint:pw@localhost./postgres",
		"postgres:///no-host-at-all",
		"https://maint:pw@127.0.0.1:5434/postgres",
		"://unparseable",
	}
	for _, raw := range bad {
		if _, err := CheckTestEndpoint(raw); err == nil {
			t.Errorf("refuse %q: accepted", redactURL(raw))
		} else if !strings.Contains(err.Error(), "unapproved endpoint") {
			t.Errorf("refuse %q: error %q lacks the refusal marker", redactURL(raw), err.Error())
		}
	}
}

func redactURL(raw string) string {
	if u, err := url.Parse(raw); err == nil {
		if _, ok := u.User.Password(); ok {
			u.User = url.User(u.User.Username())
			return u.String()
		}
	}
	return "unparseable-url"
}

// TestRunDSNRejectsUnapprovedChild exists only as the subprocess target
// for TestEndpointRejectionPrecedesDial: in a normal suite run it
// provisions (or reuses) the run database and passes.
func TestRunDSNRejectsUnapprovedChild(t *testing.T) {
	_ = RunDSN(t)
}

// TestEndpointRejectionPrecedesDial proves end to end that a
// mispointed TEST_DATABASE_URL is refused before any connection is
// attempted: the child process gets an unresolvable host, so any dial
// would fail with a DNS error instead of the allowlist refusal.
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
		"TEST_DATABASE_URL=postgres://maint:pw@u512-unapproved.invalid:5434/postgres")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("child with unapproved endpoint succeeded:\n%s", out)
	}
	if !strings.Contains(string(out), "unapproved endpoint") {
		t.Fatalf("child failed without the allowlist refusal:\n%s", out)
	}
	if strings.Contains(string(out), "no such host") ||
		strings.Contains(string(out), "connection refused") {
		t.Fatalf("child attempted a dial before refusing:\n%s", out)
	}
}
