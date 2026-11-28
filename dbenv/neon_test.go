package dbenv

import (
	"crypto/tls"
	"os"
	"strings"
	"testing"
)

func testPolicy() EndpointPolicy {
	return EndpointPolicy{
		ProjectID:    "proj_test_123",
		BranchID:     "br_test_abc",
		ParentBranch: "br_parent_xyz",
		Hosts:        []string{"ep-test-0001.invalid", "ep-test-0001-pooler.invalid"},
	}
}

// TestEndpointPolicy is pure: CheckEndpoint opens no connection, so
// every refusal below proves the harness never dials the endpoint.
// Hosts use .invalid so even an accidental dial cannot succeed.
func TestEndpointPolicy(t *testing.T) {
	pol := testPolicy()
	good := []string{
		"postgres://maint:pw@ep-test-0001.invalid:5432/neondb?sslmode=verify-full",
		"postgresql://maint:pw@ep-test-0001-pooler.invalid:5432/neondb?sslmode=verify-full",
		"postgres://maint:pw@EP-TEST-0001.INVALID:5432/neondb?sslmode=VERIFY-FULL",
	}
	for _, raw := range good {
		ep, err := CheckEndpoint(raw, pol)
		if err != nil {
			t.Errorf("accept %q: %v", redactURL(raw), err)
			continue
		}
		if ep.TLSMode == "" || ep.Policy.BranchID != pol.BranchID {
			t.Errorf("accept %q: binding not echoed", redactURL(raw))
		}
	}
	bad := []string{
		// Loopback and local names: the retired contract, now refused.
		"postgres://maint:pw@127.0.0.1:5434/postgres?sslmode=require",
		"postgres://maint:pw@localhost:5434/postgres?sslmode=require",
		"postgres://maint:pw@localhost.:5434/postgres?sslmode=require",
		"postgresql://maint:pw@[::1]:5434/x?sslmode=require",
		// Private and non-routable addresses.
		"postgres://maint:pw@10.0.0.5:5432/x?sslmode=require",
		"postgres://maint:pw@192.168.1.9:5432/x?sslmode=require",
		"postgres://maint:pw@172.16.0.2:5432/x?sslmode=require",
		"postgres://maint:pw@169.254.8.8:5432/x?sslmode=require",
		// Unknown remote hosts: not bound to the branch.
		"postgres://maint:pw@db.prod.example:5432/postgres?sslmode=require",
		"postgres://maint:pw@ep-other-9999.invalid:5432/x?sslmode=require",
		// Portless with weak TLS still refused (on TLS, not the port).
		"postgres://maint:pw@ep-test-0001.invalid/neondb?sslmode=require",
		// Weak or missing TLS. require encrypts without identity
		// verification (pgx: InsecureSkipVerify); verify-ca skips the
		// hostname. Only verify-full proves chain AND hostname.
		"postgres://maint:pw@ep-test-0001.invalid:5432/neondb",
		"postgres://maint:pw@ep-test-0001.invalid:5432/neondb?sslmode=disable",
		"postgres://maint:pw@ep-test-0001.invalid:5432/neondb?sslmode=prefer",
		"postgres://maint:pw@ep-test-0001.invalid:5432/neondb?sslmode=allow",
		"postgres://maint:pw@ep-test-0001.invalid:5432/neondb?sslmode=require",
		"postgres://maint:pw@ep-test-0001.invalid:5432/neondb?sslmode=verify-ca",
		// Shape violations.
		"https://maint:pw@ep-test-0001.invalid:5432/x?sslmode=require",
		"postgres:///no-host-at-all",
		"://unparseable",
	}
	for _, raw := range bad {
		if _, err := CheckEndpoint(raw, pol); err == nil {
			t.Errorf("refuse %q: accepted", redactURL(raw))
		} else if !strings.Contains(err.Error(), "unapproved endpoint") {
			t.Errorf("refuse %q: error %q lacks the refusal marker", redactURL(raw), err.Error())
		}
	}
}

// TestEndpointDefaultsPort proves a portless DSN (Neon's own shape)
// binds with the driver-default port recorded.
func TestEndpointDefaultsPort(t *testing.T) {
	ep, err := CheckEndpoint(
		"postgres://maint:pw@ep-test-0001.invalid/neondb?sslmode=verify-full", testPolicy())
	if err != nil {
		t.Fatalf("portless DSN refused: %v", err)
	}
	if ep.Port != "5432" {
		t.Errorf("Port = %q, want driver default 5432", ep.Port)
	}
}

// TestEndpointPolicyEmptyFailsClosed proves an unbound policy dials nothing.
func TestEndpointPolicyEmptyFailsClosed(t *testing.T) {
	raw := "postgres://maint:pw@ep-test-0001.invalid:5432/neondb?sslmode=require"
	if _, err := CheckEndpoint(raw, EndpointPolicy{}); err == nil {
		t.Error("empty policy accepted an endpoint")
	}
}

func TestPolicyFromEnv(t *testing.T) {
	t.Setenv("NEON_PROJECT_ID", "proj_test_123")
	t.Setenv("NEON_BRANCH_ID", "br_test_abc")
	t.Setenv("NEON_PARENT_BRANCH", "br_parent_xyz")
	t.Setenv("NEON_ENDPOINT_HOSTS", "ep-test-0001.invalid, ep-test-0001-pooler.invalid")
	t.Setenv("NEON_BRANCH_NAME", "dev")
	t.Setenv("NEON_BRANCH_PROTECTED", "false")
	pol, err := PolicyFromEnv()
	if err != nil {
		t.Fatalf("PolicyFromEnv: %v", err)
	}
	if pol.ProjectID != "proj_test_123" || pol.BranchID != "br_test_abc" || len(pol.Hosts) != 2 {
		t.Errorf("policy mismatch: %+v", pol)
	}
	if pol.BranchName != "dev" || pol.Protected {
		t.Errorf("identity mismatch: %+v", pol)
	}
}

func TestPolicyFromEnvRequiresIdentity(t *testing.T) {
	base := map[string]string{
		"NEON_PROJECT_ID": "p", "NEON_BRANCH_ID": "b",
		"NEON_ENDPOINT_HOSTS": "h.invalid",
		"NEON_BRANCH_NAME":    "dev", "NEON_BRANCH_PROTECTED": "false",
	}
	setAll := func(t *testing.T, skip string, protected string) {
		t.Helper()
		for k, v := range base {
			if k == skip {
				t.Setenv(k, "")
			} else if k == "NEON_BRANCH_PROTECTED" && protected != "" {
				t.Setenv(k, protected)
			} else {
				t.Setenv(k, v)
			}
		}
	}
	setAll(t, "NEON_BRANCH_NAME", "")
	if _, err := PolicyFromEnv(); err == nil {
		t.Error("missing branch name accepted")
	}
	setAll(t, "NEON_BRANCH_PROTECTED", "")
	if _, err := PolicyFromEnv(); err == nil {
		t.Error("missing protection marker accepted")
	}
	// Truly-unset (not set-but-empty) exercises the !protectedSet
	// branch; t.Setenv cannot unset, so do it manually with restore.
	t.Setenv("NEON_PROJECT_ID", "p")
	t.Setenv("NEON_BRANCH_ID", "b")
	t.Setenv("NEON_ENDPOINT_HOSTS", "h.invalid")
	t.Setenv("NEON_BRANCH_NAME", "dev")
	old, hadOld := os.LookupEnv("NEON_BRANCH_PROTECTED")
	os.Unsetenv("NEON_BRANCH_PROTECTED")
	t.Cleanup(func() {
		if hadOld {
			os.Setenv("NEON_BRANCH_PROTECTED", old)
		}
	})
	if _, err := PolicyFromEnv(); err == nil {
		t.Error("unset protection marker accepted")
	}
	setAll(t, "", "maybe")
	if _, err := PolicyFromEnv(); err == nil {
		t.Error("unparsable protection marker accepted")
	}
}

func TestPolicyFromEnvMissing(t *testing.T) {
	t.Setenv("NEON_PROJECT_ID", "")
	t.Setenv("NEON_BRANCH_ID", "")
	t.Setenv("NEON_ENDPOINT_HOSTS", "")
	if _, err := PolicyFromEnv(); err == nil {
		t.Error("missing metadata accepted")
	}
}

// TestTLSConfigPinsVerification proves the pinned config verifies
// chain and hostname: system roots, ServerName bound to the endpoint
// host, TLS 1.2 floor, default verification (no bypass flags, no
// custom verifier that could skip the hostname check).
func TestTLSConfigPinsVerification(t *testing.T) {
	ep, err := CheckEndpoint(
		"postgres://maint:pw@ep-test-0001.invalid:5432/neondb?sslmode=verify-full", testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := TLSConfigFor(ep)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InsecureSkipVerify {
		t.Error("pinned config skips verification")
	}
	if cfg.VerifyPeerCertificate != nil {
		t.Error("pinned config uses a custom verifier instead of default chain+hostname verification")
	}
	if cfg.ServerName != "ep-test-0001.invalid" {
		t.Errorf("ServerName = %q, want the endpoint host", cfg.ServerName)
	}
	if cfg.RootCAs == nil {
		t.Error("pinned config has no root pool")
	}
	if cfg.MinVersion < tls.VersionTLS12 {
		t.Errorf("MinVersion = %x, want at least TLS 1.2", cfg.MinVersion)
	}
}

// TestPoolConfigPinsTLS proves PoolConfig returns a pool whose
// TLSConfig is the pinned verifying config: enforcement lives in
// code, not in the DSN string. It opens no connection.
func TestPoolConfigPinsTLS(t *testing.T) {
	cfg, err := PoolConfig(
		"postgres://maint:pw@ep-test-0001.invalid:5432/neondb?sslmode=verify-full", testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	tc := cfg.ConnConfig.TLSConfig
	if tc == nil {
		t.Fatal("pool config has no TLSConfig")
	}
	if tc.InsecureSkipVerify || tc.ServerName != "ep-test-0001.invalid" || tc.RootCAs == nil {
		t.Errorf("pool TLSConfig not pinned to verifying: %+v", tc)
	}
}

// TestPoolConfigRejectsUnbound proves PoolConfig enforces the endpoint
// policy before building anything dialable.
func TestPoolConfigRejectsUnbound(t *testing.T) {
	raw := "postgres://maint:pw@127.0.0.1:5434/postgres?sslmode=verify-full"
	if _, err := PoolConfig(raw, testPolicy()); err == nil {
		t.Error("unbound endpoint accepted")
	} else if !strings.Contains(err.Error(), "unapproved endpoint") {
		t.Errorf("error %q lacks the refusal marker", err.Error())
	}
}

// TestCheckEndpointRedactsParseErrors (F35-03): malformed DSNs fail
// closed at the source without echoing credential material.
func TestCheckEndpointRedactsParseErrors(t *testing.T) {
	pol := testPolicy()
	canary := "canary-pw-f35-03"
	for _, raw := range []string{
		"://u:" + canary + "@h/db",
		"postgres://u:p%40" + canary + "@h/db\x7f",
	} {
		_, err := CheckEndpoint(raw, pol)
		if err == nil {
			t.Fatalf("malformed DSN accepted: %q", raw)
		}
		if strings.Contains(err.Error(), canary) {
			t.Errorf("parse error leaked DSN: %q", err.Error())
		}
	}
}
