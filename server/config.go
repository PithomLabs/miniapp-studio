package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// Operator binds one workspace to its login credential and its server-side
// framework credential. The login token travels only from the operator's
// password manager to the login form; this file holds its SHA-256 hex.
// FrameworkCredential is the tenant's Miniapp service credential, used only
// server-side for runtime.Service invocations — never sent to browsers.
type Operator struct {
	Tenant              string `json:"tenant"`
	Binding             string `json:"binding"`
	LoginTokenSHA256    string `json:"login_token_sha256"`
	FrameworkCredential string `json:"framework_credential"`
}

// Config is the validated server configuration. All secrets arrive from
// files or environment and are never logged.
type Config struct {
	ListenAddr       string
	DatabaseURL      string
	OwnerDatabaseURL string
	Operators        []Operator
	FrameworkKeys    string
	FrameworkGrants  string
	FlyOrg           string
	SpritesTokenFile string
	// GatewayOrgSecret derives per-build gateway HMAC keys. Optional:
	// empty means gateway callbacks stay unconfigured (warned at boot).
	GatewayOrgSecret string
	// SenderEnabled starts the Studio-to-Gateway dispatch loop. Optional:
	// empty/false means command rows accumulate without dispatching.
	SenderEnabled bool
}

var rxHex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Load reads and validates configuration from the environment, collecting
// every problem before failing so operators fix configuration in one pass.
// Required: STUDIO_OPERATORS_FILE, MINIAPP_KEYS_FILE, MINIAPP_GRANTS_FILE,
// database DSNs (or *_FILE variants). Optional: Studio Fly identity as a
// pair (STUDIO_FLY_ORG + STUDIO_SPRITES_TOKEN_FILE).
func Load() (Config, error) {
	var problems []string
	fail := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}
	var cfg Config

	cfg.ListenAddr = envOr("STUDIO_LISTEN", "127.0.0.1:8080")
	if _, _, err := net.SplitHostPort(cfg.ListenAddr); err != nil {
		fail("STUDIO_LISTEN %q is not host:port: %v", cfg.ListenAddr, err)
	}

	dbURL, dbErr := envOrFile("STUDIO_DATABASE_URL")
	cfg.DatabaseURL = mustSecret(dbURL, dbErr, "STUDIO_DATABASE_URL", fail)
	ownerURL, ownerErr := envOrFile("STUDIO_OWNER_DATABASE_URL")
	cfg.OwnerDatabaseURL = mustSecret(ownerURL, ownerErr, "STUDIO_OWNER_DATABASE_URL", fail)
	for _, pair := range [][2]string{{cfg.DatabaseURL, "STUDIO_DATABASE_URL"}, {cfg.OwnerDatabaseURL, "STUDIO_OWNER_DATABASE_URL"}} {
		if pair[0] == "" {
			continue
		}
		u, err := url.Parse(pair[0])
		if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
			fail("%s must be a postgres DSN", pair[1])
		}
	}

	opsPath := strings.TrimSpace(os.Getenv("STUDIO_OPERATORS_FILE"))
	if opsPath == "" {
		fail("STUDIO_OPERATORS_FILE is required")
	} else {
		ops, err := loadOperators(opsPath)
		if err != nil {
			fail("operators file: %v", err)
		} else {
			cfg.Operators = ops
		}
	}

	cfg.FrameworkKeys = strings.TrimSpace(os.Getenv("MINIAPP_KEYS_FILE"))
	cfg.FrameworkGrants = strings.TrimSpace(os.Getenv("MINIAPP_GRANTS_FILE"))
	for _, pair := range [][2]string{{cfg.FrameworkKeys, "MINIAPP_KEYS_FILE"}, {cfg.FrameworkGrants, "MINIAPP_GRANTS_FILE"}} {
		if pair[0] == "" {
			fail("%s is required", pair[1])
		} else if _, err := os.Stat(pair[0]); err != nil {
			fail("%s unreadable: %v", pair[1], err)
		}
	}

	cfg.FlyOrg = strings.TrimSpace(os.Getenv("STUDIO_FLY_ORG"))
	cfg.SpritesTokenFile = strings.TrimSpace(os.Getenv("STUDIO_SPRITES_TOKEN_FILE"))
	orgSecret, orgErr := envOrFile("STUDIO_GATEWAY_ORG_SECRET")
	if orgErr != nil {
		fail("%v", orgErr)
	}
	cfg.GatewayOrgSecret = orgSecret
	switch strings.ToLower(strings.TrimSpace(os.Getenv("STUDIO_SENDER"))) {
	case "1", "true", "yes":
		cfg.SenderEnabled = true
	}
	if (cfg.FlyOrg == "") != (cfg.SpritesTokenFile == "") {
		fail("STUDIO_FLY_ORG and STUDIO_SPRITES_TOKEN_FILE must be set together or not at all")
	}
	if cfg.SpritesTokenFile != "" {
		if _, err := os.Stat(cfg.SpritesTokenFile); err != nil {
			fail("STUDIO_SPRITES_TOKEN_FILE unreadable: %v", err)
		}
	}

	if len(problems) > 0 {
		return Config{}, fmt.Errorf("server: invalid configuration:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return cfg, nil
}

// envOrFile resolves NAME from the environment, NAME_FILE from a file, or
// reports a conflict when both are set. Values are trimmed; absence yields "".
func envOrFile(name string) (value string, err error) {
	direct := strings.TrimSpace(os.Getenv(name))
	path := strings.TrimSpace(os.Getenv(name + "_FILE"))
	if direct != "" && path != "" {
		return "", fmt.Errorf("set exactly one of %s and %s_FILE", name, name)
	}
	if path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s_FILE: %w", name, err)
		}
		return strings.TrimSpace(string(raw)), nil
	}
	return direct, nil
}

func mustSecret(value string, err error, name string, fail func(string, ...any)) string {
	if err != nil {
		fail("%v", err)
		return ""
	}
	if value == "" {
		fail("%s is required", name)
	}
	return value
}

func envOr(name, def string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return def
}

func loadOperators(path string) ([]Operator, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Operators []Operator `json:"operators"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if len(doc.Operators) == 0 {
		return nil, fmt.Errorf("no operators defined")
	}
	tenants := map[string]bool{}
	// F10: two entries sharing one token hash would make login ambiguous
	// (last match wins the loop), so duplicate hashes are a startup error.
	hashes := map[string]bool{}
	for i, op := range doc.Operators {
		where := fmt.Sprintf("operators[%d]", i)
		if op.Tenant == "" || op.Binding == "" {
			return nil, fmt.Errorf("%s: tenant and binding are required", where)
		}
		if tenants[op.Tenant] {
			return nil, fmt.Errorf("%s: duplicate tenant %q", where, op.Tenant)
		}
		tenants[op.Tenant] = true
		if !rxHex64.MatchString(op.LoginTokenSHA256) {
			return nil, fmt.Errorf("%s: login_token_sha256 must be 64 hex chars", where)
		}
		if hashes[strings.ToLower(op.LoginTokenSHA256)] {
			return nil, fmt.Errorf("%s: duplicate login_token_sha256", where)
		}
		hashes[strings.ToLower(op.LoginTokenSHA256)] = true
		if op.FrameworkCredential == "" {
			return nil, fmt.Errorf("%s: framework_credential is required", where)
		}
	}
	return doc.Operators, nil
}
