package dbenv

import (
	"net/url"
	"testing"
)

// The literal live-role name is pinned here (dbenv must not import
// store: store now imports dbenv for the endpoint policy, so a direct
// comparison would cycle the test binary). The twin pin lives in
// TestAppRoleNameLiteral (store package); drift on either side fails.
func TestLiveRoleNameLiteral(t *testing.T) {
	if liveRoleName() != "studio_app" {
		t.Errorf("dbenv live role %q drifts from studio_app", liveRoleName())
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
		if !IsRunDB(name) {
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
		if IsRunDB(name) {
			t.Errorf("%q accepted, want refusal", name)
		}
	}
}

func TestNewRunDBName(t *testing.T) {
	a, err := NewRunDBName()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewRunDBName()
	if err != nil {
		t.Fatal(err)
	}
	if !IsRunDB(a) || !IsRunDB(b) {
		t.Errorf("generated names fail the run guard: %q %q", a, b)
	}
	if a == b {
		t.Error("generated names collide")
	}
}

// NOTE (T1 C4 redesign): the loopback TestEndpointAllowlist was retired
// with CheckTestEndpoint. Its coverage moved to TestEndpointPolicy in
// neon_test.go, where the old loopback accepts are now refusals by
// approved architecture change (ADR-004/006) — not by test weakening.

func redactURL(raw string) string {
	if u, err := url.Parse(raw); err == nil {
		if _, ok := u.User.Password(); ok {
			u.User = url.User(u.User.Username())
			return u.String()
		}
	}
	return "unparseable-url"
}
