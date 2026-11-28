package main

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setProdPolicyEnv builds a policy the production guard refuses.
func setProdPolicyEnv(t *testing.T) {
	t.Helper()
	t.Setenv("NEON_PROJECT_ID", "p-test")
	t.Setenv("NEON_BRANCH_ID", "b-test")
	t.Setenv("NEON_ENDPOINT_HOSTS", "db.example.com")
	t.Setenv("NEON_BRANCH_NAME", "production")
	t.Setenv("NEON_BRANCH_PROTECTED", "false")
	t.Setenv("TEST_DATABASE_URL", "")
}

func TestBrokerPoolPolicyFailureNoPanic(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "binding.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	_, _, _, _, err := brokerPool(fs, &brokerFlags{metadataFile: bad})
	if err == nil || !strings.Contains(err.Error(), "binding") {
		t.Fatalf("policy failure err = %v (want binding error, no panic)", err)
	}
}

func TestBrokerPoolGuardRefusalNoPanic(t *testing.T) {
	setProdPolicyEnv(t)
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	_, _, _, _, err := brokerPool(fs, &brokerFlags{})
	if err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("guard refusal err = %v (want refusal, no panic)", err)
	}
}

func TestBrokerPoolDSNMissingNoPanic(t *testing.T) {
	setProdPolicyEnv(t)
	t.Setenv("NEON_BRANCH_NAME", "dev")
	t.Setenv("NEON_BRANCH_PROTECTED", "false")
	// Guard passes for dev; DSN stage must fail with the remedy.
	t.Setenv("NEON_PROJECT_ID", "p-test")
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	_, _, _, _, err := brokerPool(fs, &brokerFlags{})
	if err == nil || !strings.Contains(err.Error(), "TEST_DATABASE_URL") {
		t.Fatalf("missing DSN err = %v (want remedy, no panic)", err)
	}
}

func TestBrokerPoolParseFailureRedacted(t *testing.T) {
	setProdPolicyEnv(t)
	t.Setenv("NEON_BRANCH_NAME", "dev")
	canary := "canary-pw-8f3a"
	t.Setenv("TEST_DATABASE_URL", "://u:"+canary+"@h/db")
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	_, _, _, _, err := brokerPool(fs, &brokerFlags{})
	if err == nil {
		t.Fatal("malformed DSN accepted")
	}
	if strings.Contains(err.Error(), canary) {
		t.Fatalf("parse error leaked DSN: %q", err.Error())
	}
}

// TestF0VerbsNoPanicOnPolicyFailure exercises every F0 database
// verb's error path: with an unreadable binding they must all
// return the policy error, never panic (F35-02 shared root cause).
func TestF0VerbsNoPanicOnPolicyFailure(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "binding.json")
	cases := map[string]func([]string, *bytes.Buffer) error{
		"broker-enroll": func(a []string, b *bytes.Buffer) error { return runDevBrokerEnroll(a, b) },
		"broker-revoke": func(a []string, b *bytes.Buffer) error { return runDevBrokerRevoke(a, b) },
		"broker-leases": func(a []string, b *bytes.Buffer) error { return runDevBrokerLeases(a, b) },
		"invite-create": func(a []string, b *bytes.Buffer) error { return runDevInviteCreate(a, b) },
		"invite-accept": func(a []string, b *bytes.Buffer) error { return runDevInviteAccept(a, b) },
		"invite-revoke": func(a []string, b *bytes.Buffer) error { return runDevInviteRevoke(a, b) },
		"invite-resend": func(a []string, b *bytes.Buffer) error { return runDevInviteResend(a, b) },
		"invite-list":   func(a []string, b *bytes.Buffer) error { return runDevInviteList(a, b) },
		"capability-refresh": func(a []string, b *bytes.Buffer) error {
			return runDevCapabilityRefresh(a, b)
		},
		"operator-offboard": func(a []string, b *bytes.Buffer) error {
			return runDevOperatorOffboard(a, b)
		},
	}
	kek := filepath.Join(t.TempDir(), "kek")
	tok := filepath.Join(t.TempDir(), "tok")
	sess := filepath.Join(t.TempDir(), "sess")
	for _, p := range []string{kek, tok} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	argSets := map[string][]string{
		"broker-enroll":      {"--kek-file", kek, "--tenant", "t", "--provider", "p", "--refresh-file", tok, "--metadata-file", bad},
		"broker-revoke":      {"--tenant", "t", "--provider", "p", "--metadata-file", bad},
		"broker-leases":      {"--tenant", "t", "--metadata-file", bad},
		"invite-create":      {"--tenant", "t", "--binding", "b", "--invited-by", "o", "--token-file", filepath.Join(t.TempDir(), "n1"), "--metadata-file", bad},
		"invite-accept":      {"--token-file", tok, "--session-token-file", sess, "--metadata-file", bad},
		"invite-revoke":      {"--tenant", "t", "--id", "i", "--metadata-file", bad},
		"invite-resend":      {"--tenant", "t", "--id", "i", "--token-file", filepath.Join(t.TempDir(), "n2"), "--metadata-file", bad},
		"invite-list":        {"--tenant", "t", "--metadata-file", bad},
		"capability-refresh": {"--tenant", "t", "--metadata-file", bad},
		"operator-offboard":  {"--tenant", "t", "--binding", "b", "--metadata-file", bad},
	}
	// broker-enroll needs a parseable KEK to reach the pool stage.
	if err := os.WriteFile(kek, []byte(strings.Repeat("ab", 32)), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, verb := range cases {
		var buf bytes.Buffer
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s panicked: %v", name, r)
				}
			}()
			err := verb(argSets[name], &buf)
			if err == nil {
				t.Fatalf("%s: policy failure accepted", name)
			}
		}()
	}
	if len(cases) != 10 {
		t.Fatalf("verb cases = %d, want 10 (a verb escaped the audit)", len(cases))
	}
}

func TestRedactDSNEncodedCanaries(t *testing.T) {
	dsn := "postgres://us%40er:p%40ss%3Aword%2Fx@db.example.com:5432/neondb?sslmode=require"
	variants := []string{
		"dial " + dsn + ": refused",
		"auth failed for us@er with p@ss:word/x",
		"wrapped: " + "postgres://us%40er:p%40ss%3Aword%2Fx@db.example.com:5432/neondb",
	}
	for _, msg := range variants {
		got := redactDSN(errors.New(msg), dsn).Error()
		for _, leak := range []string{"us%40er", "p%40ss%3Aword%2Fx", "us@er", "p@ss:word/x"} {
			if strings.Contains(got, leak) {
				t.Errorf("variant %q leaked %q: %q", msg, leak, got)
			}
		}
		if !strings.Contains(got, "refused") && !strings.Contains(got, "failed") && !strings.Contains(got, "wrapped") {
			t.Errorf("diagnostic context lost: %q", got)
		}
	}
	// Host-only context (no credentials present) passes through.
	hostOnly := errors.New("db.example.com:5432 refused")
	if got := redactDSN(hostOnly, dsn); got != hostOnly {
		t.Errorf("clean error rewritten: %q", got.Error())
	}
}

func TestRedactDSNNestedWraps(t *testing.T) {
	dsn := "postgres://op:s3cret-pw@db:5432/app"
	inner := redactDSN(errors.New("dial "+dsn), dsn)
	outer := redactDSN(errors.New("migrate: "+inner.Error()), dsn)
	if strings.Contains(outer.Error(), "s3cret-pw") {
		t.Fatalf("nested wrap leaked: %q", outer.Error())
	}
}
