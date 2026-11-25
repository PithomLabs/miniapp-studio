package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func validEnv(t *testing.T) {
	t.Helper()
	ops := writeTemp(t, "ops.json", `{"operators":[{"tenant":"t1","binding":"op1",`+
		`"login_token_sha256":"`+strings.Repeat("a", 64)+`","framework_credential":"cred-1"}]}`)
	keys := writeTemp(t, "keys.json", `{}`)
	grants := writeTemp(t, "grants.json", `{}`)
	t.Setenv("STUDIO_LISTEN", "127.0.0.1:8080")
	t.Setenv("STUDIO_DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("STUDIO_OWNER_DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("STUDIO_OPERATORS_FILE", ops)
	t.Setenv("MINIAPP_KEYS_FILE", keys)
	t.Setenv("MINIAPP_GRANTS_FILE", grants)
	t.Setenv("STUDIO_FLY_ORG", "")
	t.Setenv("STUDIO_SPRITES_TOKEN_FILE", "")
	t.Setenv("STUDIO_DATABASE_URL_FILE", "")
	t.Setenv("STUDIO_OWNER_DATABASE_URL_FILE", "")
}

func TestLoadValid(t *testing.T) {
	validEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:8080" || len(cfg.Operators) != 1 {
		t.Errorf("config = %+v", cfg)
	}
}

func TestLoadCollectsProblems(t *testing.T) {
	t.Setenv("STUDIO_LISTEN", "not-a-listen-addr")
	t.Setenv("STUDIO_DATABASE_URL", "")
	t.Setenv("STUDIO_OWNER_DATABASE_URL", "")
	t.Setenv("STUDIO_OPERATORS_FILE", "")
	t.Setenv("MINIAPP_KEYS_FILE", "")
	t.Setenv("MINIAPP_GRANTS_FILE", "")
	t.Setenv("STUDIO_FLY_ORG", "personal")
	t.Setenv("STUDIO_SPRITES_TOKEN_FILE", "")
	_, err := Load()
	if err == nil {
		t.Fatal("invalid config accepted")
	}
	for _, want := range []string{"STUDIO_LISTEN", "STUDIO_DATABASE_URL", "STUDIO_OWNER_DATABASE_URL",
		"STUDIO_OPERATORS_FILE", "MINIAPP_KEYS_FILE", "MINIAPP_GRANTS_FILE", "STUDIO_FLY_ORG"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q omits %s", err, want)
		}
	}
}

func TestLoadRejectsBothSecretSources(t *testing.T) {
	validEnv(t)
	secret := writeTemp(t, "dsn", "postgres://u:p@localhost:5432/db")
	t.Setenv("STUDIO_DATABASE_URL_FILE", secret)
	if _, err := Load(); err == nil {
		t.Error("both DSN sources accepted, want refusal")
	}
}

func TestLoadOperatorsValidation(t *testing.T) {
	cases := map[string]string{
		"malformed":     `{oops`,
		"empty":         `{"operators":[]}`,
		"missing field": `{"operators":[{"tenant":"t1"}]}`,
		"bad hash":      `{"operators":[{"tenant":"t1","binding":"b","login_token_sha256":"xyz","framework_credential":"c"}]}`,
		"dup tenant": `{"operators":[{"tenant":"t1","binding":"a","login_token_sha256":"` +
			strings.Repeat("a", 64) + `","framework_credential":"c"},` +
			`{"tenant":"t1","binding":"b","login_token_sha256":"` +
			strings.Repeat("b", 64) + `","framework_credential":"c"}]}`,
		// F10: one token hash shared across two tenants is ambiguous at login.
		"dup hash": `{"operators":[{"tenant":"t1","binding":"a","login_token_sha256":"` +
			strings.Repeat("c", 64) + `","framework_credential":"c"},` +
			`{"tenant":"t2","binding":"b","login_token_sha256":"` +
			strings.Repeat("c", 64) + `","framework_credential":"c"}]}`,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			validEnv(t)
			t.Setenv("STUDIO_OPERATORS_FILE", writeTemp(t, "ops.json", doc))
			if _, err := Load(); err == nil {
				t.Error("invalid operators accepted, want refusal")
			}
		})
	}
}

// The table above proves refusal; this proves the refusal names the rule.
func TestLoadOperatorsDuplicateHashMessage(t *testing.T) {
	doc := `{"operators":[{"tenant":"t1","binding":"a","login_token_sha256":"` +
		strings.Repeat("c", 64) + `","framework_credential":"c"},` +
		`{"tenant":"t2","binding":"b","login_token_sha256":"` +
		strings.Repeat("c", 64) + `","framework_credential":"c"}]}`
	validEnv(t)
	t.Setenv("STUDIO_OPERATORS_FILE", writeTemp(t, "ops.json", doc))
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "duplicate login_token_sha256") {
		t.Fatalf("err = %v, want duplicate login_token_sha256", err)
	}
}

func TestLoadGatewayOrgSecret(t *testing.T) {
	validEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GatewayOrgSecret != "" {
		t.Errorf("default org secret = %q, want empty (unconfigured)", cfg.GatewayOrgSecret)
	}
	t.Setenv("STUDIO_GATEWAY_ORG_SECRET", "  s3cret  ")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GatewayOrgSecret != "s3cret" {
		t.Errorf("org secret = %q, want trimmed", cfg.GatewayOrgSecret)
	}
	t.Setenv("STUDIO_GATEWAY_ORG_SECRET_FILE", writeTemp(t, "orgsecret", "file-secret\n"))
	if _, err := Load(); err == nil {
		t.Error("both org-secret sources accepted, want refusal")
	}
}

func TestLoadDSNScheme(t *testing.T) {
	validEnv(t)
	t.Setenv("STUDIO_DATABASE_URL", "mysql://u:p@localhost/db")
	if _, err := Load(); err == nil {
		t.Error("non-postgres DSN accepted, want refusal")
	}
}
