package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PithomLabs/miniapp-studio/dbenv"
	"github.com/PithomLabs/miniapp-studio/store"
)

// Local UI serving (dev serve-init + dev serve): smooth path from a bound
// branch to a running Studio UI for review and testing.
//
// Security shape, non-negotiable:
//   - Secrets are never defaulted, embedded, or printed. First run
//     generates random values into mode-0600 files; later runs read them.
//   - The serve database is a fixed NON-run-pattern name: the test reaper
//     and `dev reset` both refuse non-run databases, so review data can
//     neither be reaped nor force-cleaned by test tooling. A run-pattern
//     name is refused outright.
//   - The app-role password is never rotated implicitly (matrix below).
//   - Serve binds loopback only; Fly/gateway/sender stay unset.
const (
	serveTenant     = "dev"
	serveBinding    = "dev-operator"
	serveCredential = "dev-svc"
)

type servePaths struct {
	metadata, dsn, operators, keys, grants, appPassword, token, marker string
}

func serveFlags(fs *flag.FlagSet) *servePaths {
	p := &servePaths{}
	fs.StringVar(&p.metadata, "metadata-file", "", "branch binding JSON from dev bind (required)")
	fs.StringVar(&p.dsn, "dsn-file", "", "file holding the owner database URL (required)")
	fs.StringVar(&p.operators, "operators-file", "", "operators JSON file to create or update (required)")
	fs.StringVar(&p.keys, "keys-file", "", "Miniapp keys JSON file (required)")
	fs.StringVar(&p.grants, "grants-file", "", "Miniapp grants JSON file (required)")
	fs.StringVar(&p.appPassword, "app-password-file", "", "file holding the studio_app role password (required)")
	fs.StringVar(&p.token, "token-file", "", "file holding the dev login token (required)")
	fs.StringVar(&p.marker, "marker-file", "./dev-serve.json", "init record proving serve-init ran")
	return p
}

func (p *servePaths) check() error {
	missing := []string{}
	add := func(flag, v string) {
		if strings.TrimSpace(v) == "" {
			missing = append(missing, "--"+flag)
		}
	}
	add("metadata-file", p.metadata)
	add("dsn-file", p.dsn)
	add("operators-file", p.operators)
	add("keys-file", p.keys)
	add("grants-file", p.grants)
	add("app-password-file", p.appPassword)
	add("token-file", p.token)
	if len(missing) > 0 {
		return fmt.Errorf("dev serve: missing required flags: %s", strings.Join(missing, ", "))
	}
	return nil
}

// readSecret loads one credential file. Errors name the file, never content.
func readSecret(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	out := strings.TrimSpace(string(raw))
	if out == "" {
		return "", fmt.Errorf("file %s is empty", path)
	}
	return out, nil
}

// writeSecretFile writes Folder 0700 / file 0600, refusing to overwrite.
func writeSecretFile(path string, data []byte) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("dev serve: refusing to overwrite %s", path)
	}
	if err := os.MkdirAll(dirOf(path), 0o700); err != nil {
		return fmt.Errorf("dev serve: mkdir for %s: %w", path, err)
	}
	return os.WriteFile(path, data, 0o600)
}

func dirOf(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[:i]
	}
	return "."
}

func genHex(n int) (string, error) {
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("dev serve: random: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

// serveDSNs builds owner + app DSNs for the serve database out of the
// bound branch DSN. Pure string surgery; the only secret input is the
// app password, which must already be in hand (file or generated).
func serveDSNs(baseDSN, dbName, appPassword string) (owner, app string, err error) {
	parsed, err := url.Parse(baseDSN)
	if err != nil {
		return "", "", fmt.Errorf("dev serve: parse branch DSN: %w", err)
	}
	parsed.Path = "/" + dbName
	owner = parsed.String()
	parsed.User = url.UserPassword(store.AppRoleName, appPassword)
	app = parsed.String()
	return owner, app, nil
}

type serveMarker struct {
	DB         string `json:"db"`
	Tenant     string `json:"tenant"`
	Binding    string `json:"binding"`
	Credential string `json:"credential"`
	CreatedAt  string `json:"created_at"`
}

// passwordAction is one cell of the app-role password matrix, decided
// BEFORE any Neon mutation so ambiguous states fail fast. Pure and
// unit-tested: the live flow must implement exactly this table.
type passwordAction int

const (
	pwGenerateCreate passwordAction = iota // role missing, file missing
	pwCreate                               // role missing, file present
	pwRefuse                               // role exists, file missing, no flag
	pwGenerateRotate                       // role exists, file missing, flag set
	pwProbe                                // role exists, file present, no flag
	pwRotate                               // role exists, file present, flag set
)

func passwordPlan(roleExists, fileExists, rotateFlag bool) (passwordAction, error) {
	switch {
	case !roleExists && !fileExists:
		return pwGenerateCreate, nil
	case !roleExists && fileExists:
		return pwCreate, nil
	case roleExists && !fileExists && !rotateFlag:
		return pwRefuse, fmt.Errorf("role %q exists but its password file is missing; re-run with --rotate-app-password to deliberately set it (invalidates any other setup using the old password), or supply the file", store.AppRoleName)
	case roleExists && !fileExists && rotateFlag:
		return pwGenerateRotate, nil
	case roleExists && fileExists && !rotateFlag:
		return pwProbe, nil
	default: // roleExists && fileExists && rotateFlag
		return pwRotate, nil
	}
}

func runDevServeInit(args []string) error {
	fs := flag.NewFlagSet("dev serve-init", flag.ContinueOnError)
	p := serveFlags(fs)
	dbName := fs.String("db-name", devDatabaseName, "serve database name (must NOT match the run pattern)")
	tenant := fs.String("tenant", serveTenant, "dev tenant id")
	binding := fs.String("binding", serveBinding, "dev operator binding id")
	credential := fs.String("credential", serveCredential, "Miniapp service credential name")
	rotate := fs.Bool("rotate-app-password", false, "EXPLICIT: set the studio_app password from --app-password-file (generating it first if missing)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := p.check(); err != nil {
		return err
	}
	if dbenv.IsRunDB(*dbName) {
		return fmt.Errorf("dev serve-init: refusing run-pattern database name %q (test tooling would reap it); use a stable non-run name", *dbName)
	}
	if strings.TrimSpace(*tenant) == "" || strings.TrimSpace(*binding) == "" || strings.TrimSpace(*credential) == "" {
		return fmt.Errorf("dev serve-init: tenant, binding, and credential must be non-empty")
	}

	pol, err := dbenv.PolicyFromFile(p.metadata)
	if err != nil {
		return err
	}
	base, err := readSecret(p.dsn)
	if err != nil {
		return fmt.Errorf("dev serve-init: owner DSN: %w", err)
	}
	if _, err := dbenv.CheckEndpoint(base, pol); err != nil {
		return err
	}
	if err := dbenv.RefuseProductionBranch(pol); err != nil {
		return fmt.Errorf("dev serve-init: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	maint, err := dbenv.Pool(ctx, base, pol)
	if err != nil {
		return redactDSN(fmt.Errorf("dev serve-init: %w", err), base)
	}
	defer maint.Close()

	// 1. Password matrix FIRST: decide before any Neon mutation, so
	// ambiguous states fail fast. The role must also exist before
	// migrations run, because 000002 grants to it.
	password, pwErr := readSecret(p.appPassword)
	var roleExists bool
	if err := maint.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, store.AppRoleName).Scan(&roleExists); err != nil {
		return fmt.Errorf("dev serve-init: role check: %w", err)
	}
	action, err := passwordPlan(roleExists, pwErr == nil, *rotate)
	if err != nil {
		return fmt.Errorf("dev serve-init: %w", err)
	}
	created := false
	switch action {
	case pwGenerateCreate, pwGenerateRotate:
		password, err = genHex(32)
		if err != nil {
			return err
		}
		if err := writeSecretFile(p.appPassword, []byte(password+"\n")); err != nil {
			return err
		}
		fmt.Printf("dev serve-init: generated app-role password file %s (0600)\n", p.appPassword)
		if action == pwGenerateCreate {
			if err := store.EnsureAppRole(ctx, maint, password); err != nil {
				return fmt.Errorf("dev serve-init: create app role: %w", err)
			}
			created = true
		}
	case pwCreate:
		if err := store.EnsureAppRole(ctx, maint, password); err != nil {
			return fmt.Errorf("dev serve-init: create app role: %w", err)
		}
		created = true
	case pwRotate:
		if err := store.RotateAppRole(ctx, maint, password); err != nil {
			return fmt.Errorf("dev serve-init: rotate app role: %w", err)
		}
		fmt.Printf("dev serve-init: WARNING: rotated role %q to the password file (update any other setup using the old password)\n", store.AppRoleName)
	case pwProbe:
		// Proven after migrate, when the serve pool exists.
	}
	rotateAfter := action == pwGenerateRotate
	if rotateAfter {
		if err := store.RotateAppRole(ctx, maint, password); err != nil {
			return fmt.Errorf("dev serve-init: rotate app role: %w", err)
		}
		fmt.Printf("dev serve-init: WARNING: rotated role %q to the generated password file (update any other setup using the old password)\n", store.AppRoleName)
	}

	// 2. Serve database via the shared helper (same path as dev init:
	// create if missing, baseline, migrate, version-verify).
	ownerDSN, err := ensureDevDatabase(ctx, base, pol, *dbName, true)
	if err != nil {
		return err
	}
	owner, err := dbenv.Pool(ctx, ownerDSN, pol)
	if err != nil {
		return redactDSN(fmt.Errorf("dev serve-init: %w", err), ownerDSN)
	}
	defer owner.Close()

	// 3. Prove file and role agree (skipped only when this run just
	// generated the password — nothing to disagree with yet; a rotation
	// to a pre-existing file still probes as proof it took), then
	// ensure CONNECT on the serve database.
	if !created && !rotateAfter {
		if err := probeAppLogin(ctx, base, pol, *dbName, password); err != nil {
			return err
		}
	}
	if err := store.EnsureAppRoleConnect(ctx, owner); err != nil {
		return fmt.Errorf("dev serve-init: app role connect: %w", err)
	}

	// 4. Operator row + token file + operators file.
	token, err := readSecret(p.token)
	if err != nil {
		token, err = genHex(32)
		if err != nil {
			return err
		}
		if err := writeSecretFile(p.token, []byte(token+"\n")); err != nil {
			return err
		}
		fmt.Printf("dev serve-init: generated login token file %s (0600)\n", p.token)
	}
	if err := store.UpsertOperator(ctx, owner, *tenant, *binding, true); err != nil {
		return fmt.Errorf("dev serve-init: operator row: %w", err)
	}
	sum := sha256.Sum256([]byte(token))
	if err := upsertOperatorsFile(p.operators, operatorEntry{
		Tenant: *tenant, Binding: *binding,
		LoginTokenSHA256: hex.EncodeToString(sum[:]), FrameworkCredential: *credential,
	}); err != nil {
		return fmt.Errorf("dev serve-init: operators file: %w", err)
	}

	// 5. Keys/grants files: create from the test-proven shapes when
	// missing; when present, verify (never silently edit).
	if err := ensureKeysFile(p.keys, *credential, *tenant); err != nil {
		return err
	}
	if err := ensureGrantsFile(p.grants, *tenant); err != nil {
		return err
	}

	// 6. Marker + next steps. Values never printed.
	marker := serveMarker{DB: *dbName, Tenant: *tenant, Binding: *binding,
		Credential: *credential, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	raw, _ := json.MarshalIndent(marker, "", "  ")
	if err := os.WriteFile(p.marker, append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("dev serve-init: marker: %w", err)
	}
	fmt.Printf("dev serve-init: ready (db %s, tenant %q). Next:\n", *dbName, *tenant)
	fmt.Printf("  1. task dev:serve\n")
	fmt.Printf("  2. open the printed URL; sign in with the token in %s\n", p.token)
	return nil
}

// probeAppLogin verifies the password file agrees with the live role.
func probeAppLogin(ctx context.Context, base string, pol dbenv.EndpointPolicy, dbName, password string) error {
	_, appDSN, err := serveDSNs(base, dbName, password)
	if err != nil {
		return err
	}
	if _, err := dbenv.CheckEndpoint(appDSN, pol); err != nil {
		return fmt.Errorf("dev serve-init: app endpoint rejected: %w", err)
	}
	pool, err := dbenv.Pool(ctx, appDSN, pol)
	if err != nil {
		return fmt.Errorf("dev serve-init: password file does not match role %q (re-run with --rotate-app-password to set it deliberately)", store.AppRoleName)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("dev serve-init: password file does not match role %q (re-run with --rotate-app-password to set it deliberately)", store.AppRoleName)
	}
	return nil
}

func ensureKeysFile(path, credential, tenant string) error {
	want := map[string]any{
		credential: map[string]any{"agent": "svc-agent", "kind": "service", "id": "svc", "tenant": tenant},
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("dev serve-init: keys file: %w", err)
		}
		out, _ := json.MarshalIndent(want, "", "  ")
		if err := writeSecretFile(path, append(out, '\n')); err != nil {
			return err
		}
		fmt.Printf("dev serve-init: wrote keys file %s (0600)\n", path)
		return nil
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("dev serve-init: keys file %s is not JSON: %w", path, err)
	}
	if _, ok := doc[credential]; !ok {
		return fmt.Errorf("dev serve-init: keys file %s lacks credential %q; add it (same shape as a fresh file) or point --keys-file elsewhere", path, credential)
	}
	return nil
}

func ensureGrantsFile(path, tenant string) error {
	want := map[string]any{
		tenant: map[string]any{"studio-runner": "autonomous", "studio": "autonomous"},
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("dev serve-init: grants file: %w", err)
		}
		out, _ := json.MarshalIndent(want, "", "  ")
		if err := writeSecretFile(path, append(out, '\n')); err != nil {
			return err
		}
		fmt.Printf("dev serve-init: wrote grants file %s (0600)\n", path)
		return nil
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("dev serve-init: grants file %s is not JSON: %w", path, err)
	}
	if _, ok := doc[tenant]; !ok {
		return fmt.Errorf("dev serve-init: grants file %s lacks tenant %q; add it (same shape as a fresh file) or point --grants-file elsewhere", path, tenant)
	}
	return nil
}

func runDevServe(args []string) error {
	fs := flag.NewFlagSet("dev serve", flag.ContinueOnError)
	p := serveFlags(fs)
	listen := fs.String("listen", "127.0.0.1:8080", "studio listen address (loopback IPs only: 127.0.0.1 or ::1; hostnames refused)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := p.check(); err != nil {
		return err
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(*listen))
	if err != nil {
		return fmt.Errorf("dev serve: bad listen address %q: %w (local review only)", *listen, err)
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("dev serve: refusing non-loopback listen address %q (local review only)", *listen)
	}
	raw, err := os.ReadFile(p.marker)
	if err != nil {
		return fmt.Errorf("dev serve: missing marker %s — run task dev:serve-init first: %w", p.marker, err)
	}
	var marker serveMarker
	if err := json.Unmarshal(raw, &marker); err != nil || marker.DB == "" {
		return fmt.Errorf("dev serve: marker %s is invalid — re-run task dev:serve-init", p.marker)
	}
	pol, err := dbenv.PolicyFromFile(p.metadata)
	if err != nil {
		return err
	}
	base, err := readSecret(p.dsn)
	if err != nil {
		return fmt.Errorf("dev serve: owner DSN: %w", err)
	}
	password, err := readSecret(p.appPassword)
	if err != nil {
		return fmt.Errorf("dev serve: app password: %w (run task dev:serve-init)", err)
	}
	ownerDSN, appDSN, err := serveDSNs(base, marker.DB, password)
	if err != nil {
		return err
	}
	for name, dsn := range map[string]string{"owner": ownerDSN, "app": appDSN} {
		if _, err := dbenv.CheckEndpoint(dsn, pol); err != nil {
			return fmt.Errorf("dev serve: %s endpoint rejected: %w", name, err)
		}
	}
	if err := dbenv.RefuseProductionBranch(pol); err != nil {
		return fmt.Errorf("dev serve: %w", err)
	}
	root, err := repoRoot()
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "run", "./cmd/studio", "serve")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"STUDIO_LISTEN="+strings.TrimSpace(*listen),
		"STUDIO_DATABASE_URL="+appDSN,
		"STUDIO_OWNER_DATABASE_URL="+ownerDSN,
		"STUDIO_OPERATORS_FILE="+p.operators,
		"MINIAPP_KEYS_FILE="+p.keys,
		"MINIAPP_GRANTS_FILE="+p.grants,
	)
	fmt.Printf("dev serve: Studio on http://%s (tenant %q; stop with Ctrl-C)\n", strings.TrimSpace(*listen), marker.Tenant)
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			os.Exit(exit.ExitCode())
		}
		return err
	}
	return nil
}
