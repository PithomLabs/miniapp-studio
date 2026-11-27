package main

// Command studio dev: the canonical development-database interface.
//
//	studio dev diag-roles  # role-capability booleans for studio_lookup (secret-free diagnostic)
//	studio dev setup       # Neon preflight: metadata, endpoint binding, TLS, live version
//	studio dev doctor      # diagnose toolchain, endpoint, database, migrations
//	studio dev init        # ensure the shared dev database (idempotent;
//	                         explicit --db-name provisions scratch instead)
//	studio dev reset       # drop stale run databases (scoped, explicit)
//	studio dev test        # run the test suite through one entry point
//	studio dev fingerprint # machine-readable environment record
//
// Every verb reuses the dbenv + store packages shared with the test
// harness — the CLI never reimplements endpoint policy, provisioning,
// grants, or migration logic. DSNs arrive via TEST_DATABASE_URL or
// --dsn-file and are never printed; errors pass through redactDSN.
// Branch binding arrives via NEON_PROJECT_ID, NEON_BRANCH_ID,
// NEON_PARENT_BRANCH (optional), NEON_ENDPOINT_HOSTS, NEON_BRANCH_NAME,
// and NEON_BRANCH_PROTECTED. Every verb refuses production bindings.

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/PithomLabs/miniapp-studio/dbenv"
	"github.com/PithomLabs/miniapp-studio/store"
)

func runDev(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: studio dev <bind|setup|doctor|init|reset|test|fingerprint|diag-roles|serve-init|serve|secrets-validate|secrets-plan|broker-init-kek|broker-enroll|broker-revoke|broker-leases|invite-create|invite-accept|invite-revoke|invite-resend|invite-list|capability-refresh> [flags]")
	}
	switch args[0] {
	case "bind":
		return runDevBind(args[1:])
	case "diag-roles":
		return runDevDiagRoles(args[1:])
	case "setup":
		return runDevSetup(args[1:])
	case "doctor":
		return runDevDoctor(args[1:])
	case "init":
		return runDevInit(args[1:])
	case "reset":
		return runDevReset(args[1:])
	case "test":
		return runDevTest(args[1:])
	case "fingerprint":
		return runDevFingerprint(args[1:])
	case "serve-init":
		return runDevServeInit(args[1:])
	case "serve":
		return runDevServe(args[1:])
	case "secrets-validate":
		return runDevSecretsValidate(args[1:], os.Stdout)
	case "secrets-plan":
		return runDevSecretsPlan(args[1:], os.Stdout)
	case "broker-init-kek":
		return runDevBrokerInitKEK(args[1:], os.Stdout)
	case "broker-enroll":
		return runDevBrokerEnroll(args[1:], os.Stdout)
	case "broker-revoke":
		return runDevBrokerRevoke(args[1:], os.Stdout)
	case "broker-leases":
		return runDevBrokerLeases(args[1:], os.Stdout)
	case "invite-create":
		return runDevInviteCreate(args[1:], os.Stdout)
	case "invite-accept":
		return runDevInviteAccept(args[1:], os.Stdout)
	case "invite-revoke":
		return runDevInviteRevoke(args[1:], os.Stdout)
	case "invite-resend":
		return runDevInviteResend(args[1:], os.Stdout)
	case "invite-list":
		return runDevInviteList(args[1:], os.Stdout)
	case "capability-refresh":
		return runDevCapabilityRefresh(args[1:], os.Stdout)
	default:
		return fmt.Errorf("unknown dev command %q (want bind|setup|doctor|init|reset|test|fingerprint|diag-roles|serve-init|serve|secrets-validate|secrets-plan|broker-init-kek|broker-enroll|broker-revoke|broker-leases|invite-create|invite-accept|invite-revoke|invite-resend|invite-list|capability-refresh)", args[0])
	}
}

// devDSN resolves the bootstrap database URL for dev verbs: explicit
// --dsn-file wins, TEST_DATABASE_URL is the default. Callers must
// never print the returned value.
func devDSN(dsnFile string) (string, error) {
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsnFile != "" {
		raw, err := os.ReadFile(dsnFile)
		if err != nil {
			return "", fmt.Errorf("read dsn file: %w", err)
		}
		dsn = strings.TrimSpace(string(raw))
	}
	if dsn == "" {
		return "", fmt.Errorf("set TEST_DATABASE_URL or --dsn-file")
	}
	return dsn, nil
}

// devPolicy loads the endpoint policy: an explicit --metadata-file
// wins (owner-terminal file handoff); otherwise the NEON_*
// environment (CI/T2 non-interactive path), which must carry the same
// branch identity (NEON_BRANCH_NAME + NEON_BRANCH_PROTECTED). No
// merging: a binding half from file and half from env would name no
// real branch.
func devPolicy(metadataFile string) (dbenv.EndpointPolicy, error) {
	if metadataFile != "" {
		return dbenv.PolicyFromFile(metadataFile)
	}
	return dbenv.PolicyFromEnv()
}

// devDatabaseName is the one shared development database. Deliberately
// NOT run-pattern: test tooling (reaper, reset) refuses non-run
// databases, so review data is never force-cleaned; per-process test
// isolation still uses fresh run databases via testdb (untouched).
const devDatabaseName = "studio_dev_ui"

// resolveInitDB maps --db-name to (name, stable): empty means the
// shared dev database (idempotent); an explicit name must match the
// run pattern and provisions scratch (refusing existing names).
func resolveInitDB(explicit string) (name string, stable bool, err error) {
	if explicit == "" {
		return devDatabaseName, true, nil
	}
	if !dbenv.IsRunDB(explicit) {
		return "", false, fmt.Errorf("dev init: refusing non-run database name %q (explicit names are scratch run databases; omit --db-name for the shared dev database)", explicit)
	}
	return explicit, false, nil
}

// ensureDevDatabase provisions a dev database, then baselines, migrates,
// and version-verifies it. Stable mode (the shared dev database) reuses
// an existing database and skips test grants; scratch mode (explicit
// run-pattern names) refuses existing names and mirrors test grants,
// preserving the legacy init behavior exactly.
func ensureDevDatabase(ctx context.Context, base string, pol dbenv.EndpointPolicy, dbName string, stable bool) (string, error) {
	// Defense in depth: both callers guard first, but this helper
	// performs every mutation, so it refuses production itself.
	if err := dbenv.RefuseProductionBranch(pol); err != nil {
		return "", fmt.Errorf("dev init: %w", err)
	}
	maint, err := dbenv.Pool(ctx, base, pol)
	if err != nil {
		return "", redactDSN(fmt.Errorf("dev init: %w", err), base)
	}
	defer maint.Close()
	if err := dbenv.CheckPGMajor(ctx, maint); err != nil {
		return "", err
	}
	var exists bool
	if err := maint.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, dbName).Scan(&exists); err != nil {
		return "", fmt.Errorf("dev init: existence check: %w", err)
	}
	if exists && !stable {
		return "", fmt.Errorf("dev init: database %q already exists; pick another name or reset it first", dbName)
	}
	parsed, _ := url.Parse(base)
	parsed.Path = "/" + dbName
	dsn := parsed.String()
	if !exists {
		if _, err := maint.Exec(ctx, `CREATE DATABASE `+dbenv.QuoteIdent(dbName)); err != nil {
			return "", fmt.Errorf("dev init: create database: %w", err)
		}
		fmt.Printf("dev init: created database %s\n", dbName)
	}
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	modDir, err := dbenv.FrameworkModuleDir(ctx, root)
	if err != nil {
		return "", err
	}
	if err := dbenv.ApplyFrameworkBaseline(ctx, dsn, modDir, pol); err != nil {
		return "", redactDSN(err, dsn)
	}
	if stable {
		// Live-role existence only: no run gate (non-run database
		// by design), no fixed test credential on a persistent DB.
		pool, err := dbenv.Pool(ctx, base, pol)
		if err != nil {
			return "", redactDSN(fmt.Errorf("dev init: %w", err), base)
		}
		roleErr := dbenv.EnsureLiveRole(ctx, pool)
		pool.Close()
		if roleErr != nil {
			return "", redactDSN(fmt.Errorf("dev init: %w", roleErr), base)
		}
	} else if err := dbenv.EnsureClusterRoles(ctx, dsn, pol); err != nil {
		return "", redactDSN(fmt.Errorf("dev init: %w", err), dsn)
	}
	if err := store.Migrate(ctx, dsn, pol); err != nil {
		return "", redactDSN(err, dsn)
	}
	owner, err := dbenv.Pool(ctx, dsn, pol)
	if err != nil {
		return "", redactDSN(fmt.Errorf("dev init: %w", err), dsn)
	}
	defer owner.Close()
	if !stable {
		if err := dbenv.GrantTestPrivileges(ctx, owner); err != nil {
			return "", redactDSN(err, dsn)
		}
	}
	applied, dirty, err := store.CurrentVersion(ctx, owner)
	if err != nil {
		return "", redactDSN(err, dsn)
	}
	latest, err := store.LatestVersion()
	if err != nil {
		return "", err
	}
	if dirty || applied != latest {
		return "", fmt.Errorf("dev init: studio schema at v%d/%d dirty=%v, want clean v%d", applied, latest, dirty, latest)
	}
	if exists {
		fmt.Printf("dev init: reusing database %s (re-verified clean: studio v%d)\n", dbName, applied)
	} else if stable {
		fmt.Printf("dev init: initialized %s (studio v%d, framework baseline applied, roles ensured)\n", dbName, applied)
	} else {
		fmt.Printf("dev init: initialized %s (studio v%d, framework baseline applied, roles + grants ensured)\n", dbName, applied)
	}
	return dsn, nil
}

// repoRoot resolves the checkout root. Dev verbs run from anywhere
// inside the checkout; anything else fails with instructions.
func repoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not inside the repository checkout (git rev-parse failed): %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

var rxGoDirective = regexp.MustCompile(`(?m)^go (\d+)\.(\d+)`)

// goToolchainOK compares the running toolchain against the go.mod
// floor. Pure: takes version strings, returns pass/fail + detail.
func goToolchainOK(runtimeVer, goMod string) (bool, string) {
	m := rxGoDirective.FindStringSubmatch(goMod)
	if m == nil {
		return false, "go.mod has no go directive"
	}
	var wantMajor, wantMinor int
	fmt.Sscanf(m[1]+"."+m[2], "%d.%d", &wantMajor, &wantMinor)
	ver := strings.TrimPrefix(runtimeVer, "go")
	var gotMajor, gotMinor int
	if _, err := fmt.Sscanf(ver, "%d.%d", &gotMajor, &gotMinor); err != nil {
		return false, fmt.Sprintf("unparseable runtime version %q", runtimeVer)
	}
	if gotMajor != wantMajor || gotMinor < wantMinor {
		return false, fmt.Sprintf("runtime go%d.%d below go.mod floor go%d.%d", gotMajor, gotMinor, wantMajor, wantMinor)
	}
	return true, fmt.Sprintf("runtime %s meets go.mod floor %s.%s", runtimeVer, m[1], m[2])
}

// Fingerprint is the machine-readable environment record. It carries
// versions, counts, and Neon endpoint identity — never credentials,
// DSNs, or row data.
type Fingerprint struct {
	At         string             `json:"at_utc"`
	GitCommit  string             `json:"git_commit"`
	GoVersion  string             `json:"go_version"`
	GoModFloor string             `json:"go_mod_floor"`
	Endpoint   NeonEndpoint       `json:"endpoint"`
	Server     dbenv.ServerInfo   `json:"server"`
	Studio     MigrationState     `json:"studio_migrations"`
	Framework  MigrationState     `json:"framework_migrations"`
	Env        map[string]string  `json:"env"`
	Checks     []FingerprintCheck `json:"checks"`
}

// NeonEndpoint is the R312 endpoint identity: project, branch, and
// parent branch from the endpoint policy plus the validated host,
// port, and TLS mode. No credentials, no DSN.
type NeonEndpoint struct {
	ProjectID    string `json:"project_id"`
	BranchID     string `json:"branch_id"`
	ParentBranch string `json:"parent_branch,omitempty"`
	Host         string `json:"host"`
	Port         string `json:"port"`
	TLSMode      string `json:"tls_mode"`
}

// MigrationState reports applied vs available schema versions.
type MigrationState struct {
	Applied uint `json:"applied"`
	Latest  uint `json:"latest"`
	Dirty   bool `json:"dirty"`
}

// FingerprintCheck is one named gate outcome.
type FingerprintCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"` // pass | fail | skip
	Detail string `json:"detail"`
}

// runDevSetup is the Neon preflight behind `task dev:setup`. It
// verifies branch metadata, endpoint binding, TLS, and live server
// version. It provisions nothing: branch creation stays an explicit
// owner operation until the T2 controller exists.
func runDevSetup(args []string) error {
	fs := flag.NewFlagSet("dev setup", flag.ContinueOnError)
	dsnFile := fs.String("dsn-file", "", "file holding TEST_DATABASE_URL (or set the env var)")
	metadataFile := fs.String("metadata-file", "", "branch binding JSON from dev bind (or NEON_* env)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	pol, err := devPolicy(*metadataFile)
	if err != nil {
		return fmt.Errorf("dev setup: %v (run dev bind, then retry)", err)
	}
	if err := dbenv.RefuseProductionBranch(pol); err != nil {
		return fmt.Errorf("dev setup: %w", err)
	}
	fmt.Printf("project: %s (%s)\nbranch:  %s (%s)\n", pol.ProjectName, pol.ProjectID, pol.BranchName, pol.BranchID)
	if pol.ParentBranch != "" {
		fmt.Printf("parent:   %s\n", pol.ParentBranch)
	}
	dsn, err := devDSN(*dsnFile)
	if err != nil {
		return fmt.Errorf("dev setup: %v", err)
	}
	ep, err := dbenv.CheckEndpoint(dsn, pol)
	if err != nil {
		return fmt.Errorf("dev setup: %v", err)
	}
	fmt.Printf("endpoint: %s:%s (%s, bound)\n", ep.Host, ep.Port, ep.TLSMode)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := dbenv.Pool(ctx, dsn, pol)
	if err != nil {
		return redactDSN(fmt.Errorf("dev setup: %w", err), dsn)
	}
	defer pool.Close()
	if err := dbenv.CheckPGMajor(ctx, pool); err != nil {
		return fmt.Errorf("dev setup: %v", err)
	}
	info, err := dbenv.CollectServerInfo(ctx, pool)
	if err != nil {
		return redactDSN(fmt.Errorf("dev setup: %w", err), dsn)
	}
	fmt.Printf("server:   %s (num %d, %d extensions)\n", info.Version, info.VersionNum, len(info.Extensions))
	fmt.Println("dev setup: preflight passes — next: studio dev doctor, then studio dev init")
	return nil
}

// runDevDiagRoles prints the six-boolean role-capability report for
// studio_lookup as seen by the migration session. Output carries
// booleans plus the already-known role name only — no credentials,
// DSNs, hostnames, or IDs beyond the fixed role name.
func runDevDiagRoles(args []string) error {
	fs := flag.NewFlagSet("dev diag-roles", flag.ContinueOnError)
	dsnFile := fs.String("dsn-file", "", "file holding TEST_DATABASE_URL (or set the env var)")
	metadataFile := fs.String("metadata-file", "", "branch binding JSON from dev bind (or NEON_* env)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	pol, err := devPolicy(*metadataFile)
	if err != nil {
		return err
	}
	if err := dbenv.RefuseProductionBranch(pol); err != nil {
		return fmt.Errorf("dev diag-roles: %w", err)
	}
	dsn, err := devDSN(*dsnFile)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := dbenv.Pool(ctx, dsn, pol)
	if err != nil {
		return redactDSN(err, dsn)
	}
	defer pool.Close()
	rep, err := dbenv.CollectRoleReport(ctx, pool, "studio_lookup")
	if err != nil {
		return redactDSN(err, dsn)
	}
	raw, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(raw))
	return nil
}

func runDevDoctor(args []string) error {
	fs := flag.NewFlagSet("dev doctor", flag.ContinueOnError)
	dsnFile := fs.String("dsn-file", "", "file holding TEST_DATABASE_URL (or set the env var)")
	metadataFile := fs.String("metadata-file", "", "branch binding JSON from dev bind (or NEON_* env)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fp, err := collectFingerprint(*dsnFile, *metadataFile)
	failed := err != nil
	for _, c := range fp.Checks {
		fmt.Printf("%-12s %s\n", c.Status, c.Name)
		if c.Detail != "" {
			fmt.Printf("             %s\n", c.Detail)
		}
		if c.Status == "fail" {
			failed = true
		}
	}
	if failed {
		if err != nil {
			return fmt.Errorf("dev doctor: %v", err)
		}
		return fmt.Errorf("dev doctor: environment does not satisfy the contract (see FAIL lines above)")
	}
	fmt.Println("dev doctor: all checks pass")
	return nil
}

func runDevFingerprint(args []string) error {
	fs := flag.NewFlagSet("dev fingerprint", flag.ContinueOnError)
	dsnFile := fs.String("dsn-file", "", "file holding TEST_DATABASE_URL (or set the env var)")
	metadataFile := fs.String("metadata-file", "", "branch binding JSON from dev bind (or NEON_* env)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fp, err := collectFingerprint(*dsnFile, *metadataFile)
	if err != nil {
		return fmt.Errorf("dev fingerprint: %w", err)
	}
	raw, err := json.MarshalIndent(fp, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(raw))
	return nil
}

// collectFingerprint gathers every contract input. It fails fast on
// the first violated gate (toolchain, endpoint, live server); version
// skew and dirty schemas fail loudly with the exact mismatch stated.
func collectFingerprint(dsnFile, metadataFile string) (Fingerprint, error) {
	var fp Fingerprint
	fp.At = time.Now().UTC().Format(time.RFC3339)
	check := func(name, status, detail string) {
		fp.Checks = append(fp.Checks, FingerprintCheck{Name: name, Status: status, Detail: detail})
	}

	root, err := repoRoot()
	if err != nil {
		return fp, err
	}
	if out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output(); err != nil {
		return fp, fmt.Errorf("git commit: %w", err)
	} else {
		fp.GitCommit = strings.TrimSpace(string(out))
	}
	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return fp, fmt.Errorf("read go.mod: %w", err)
	}
	fp.GoVersion = runtime.Version()
	if m := rxGoDirective.FindStringSubmatch(string(goMod)); m != nil {
		fp.GoModFloor = m[1] + "." + m[2]
	}
	ok, detail := goToolchainOK(runtime.Version(), string(goMod))
	if !ok {
		check("go-toolchain", "fail", detail)
		return fp, fmt.Errorf("%s", detail)
	}
	check("go-toolchain", "pass", detail)

	pol, err := devPolicy(metadataFile)
	if err != nil {
		check("endpoint", "fail", err.Error())
		return fp, err
	}
	if err := dbenv.RefuseProductionBranch(pol); err != nil {
		check("production-guard", "fail", err.Error())
		return fp, err
	}
	check("production-guard", "pass", "branch "+pol.BranchName)
	dsn, err := devDSN(dsnFile)
	if err != nil {
		check("endpoint", "fail", err.Error())
		return fp, err
	}
	ep, err := dbenv.CheckEndpoint(dsn, pol)
	if err != nil {
		check("endpoint", "fail", err.Error())
		return fp, err
	}
	fp.Endpoint = NeonEndpoint{
		ProjectID: pol.ProjectID, BranchID: pol.BranchID, ParentBranch: pol.ParentBranch,
		Host: ep.Host, Port: ep.Port, TLSMode: ep.TLSMode,
	}
	check("endpoint", "pass", fmt.Sprintf("branch %s at %s:%s (%s)", pol.BranchID, ep.Host, ep.Port, ep.TLSMode))
	fp.Env = map[string]string{"TEST_DATABASE_URL": "set"}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := dbenv.Pool(ctx, dsn, pol)
	if err != nil {
		return fp, redactDSN(err, dsn)
	}
	defer pool.Close()
	if err := dbenv.CheckPGMajor(ctx, pool); err != nil {
		check("postgres-major", "fail", err.Error())
		return fp, err
	}
	info, err := dbenv.CollectServerInfo(ctx, pool)
	if err != nil {
		return fp, redactDSN(err, dsn)
	}
	fp.Server = info
	check("postgres-major", "pass", fmt.Sprintf("major %d (%s)", info.VersionNum/10000, info.Version))

	studioApplied, studioDirty, err := store.CurrentVersion(ctx, pool)
	if err != nil {
		return fp, redactDSN(err, dsn)
	}
	studioLatest, err := store.LatestVersion()
	if err != nil {
		return fp, err
	}
	fp.Studio = MigrationState{Applied: studioApplied, Latest: studioLatest, Dirty: studioDirty}
	fwApplied, fwDirty, err := store.FrameworkVersion(ctx, pool)
	if err != nil {
		return fp, redactDSN(err, dsn)
	}
	fp.Framework = MigrationState{Applied: fwApplied, Latest: 0, Dirty: fwDirty}
	if studioDirty || fwDirty {
		check("migrations-clean", "fail", "dirty migration state on bootstrap database")
		return fp, fmt.Errorf("dirty migration state")
	}
	check("migrations-clean", "pass", fmt.Sprintf("studio v%d/%d, framework v%d", studioApplied, studioLatest, fwApplied))
	return fp, nil
}

func runDevInit(args []string) error {
	fs := flag.NewFlagSet("dev init", flag.ContinueOnError)
	dsnFile := fs.String("dsn-file", "", "file holding TEST_DATABASE_URL (or set the env var)")
	metadataFile := fs.String("metadata-file", "", "branch binding JSON from dev bind (or NEON_* env)")
	dbName := fs.String("db-name", "", "explicit scratch run database name (must match the run pattern; default ensures the shared dev database)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	base, err := devDSN(*dsnFile)
	if err != nil {
		return err
	}
	pol, err := devPolicy(*metadataFile)
	if err != nil {
		return err
	}
	if _, err := dbenv.CheckEndpoint(base, pol); err != nil {
		return err
	}
	if err := dbenv.RefuseProductionBranch(pol); err != nil {
		return fmt.Errorf("dev init: %w", err)
	}
	name, stable, err := resolveInitDB(*dbName)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	_, err = ensureDevDatabase(ctx, base, pol, name, stable)
	return err
}

func runDevReset(args []string) error {
	fs := flag.NewFlagSet("dev reset", flag.ContinueOnError)
	dsnFile := fs.String("dsn-file", "", "file holding TEST_DATABASE_URL (or set the env var)")
	metadataFile := fs.String("metadata-file", "", "branch binding JSON from dev bind (or NEON_* env)")
	confirm := fs.Bool("yes", false, "actually drop (required; without it, or with --dry-run, only lists)")
	dryRun := fs.Bool("dry-run", false, "list what would be dropped without dropping")
	olderThan := fs.Duration("older-than", 30*time.Minute, "only consider databases older than this")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*confirm && !*dryRun {
		return fmt.Errorf("dev reset: refusing without --yes (or preview with --dry-run)")
	}
	base, err := devDSN(*dsnFile)
	if err != nil {
		return err
	}
	pol, err := devPolicy(*metadataFile)
	if err != nil {
		return err
	}
	if _, err := dbenv.CheckEndpoint(base, pol); err != nil {
		return err
	}
	if err := dbenv.RefuseProductionBranch(pol); err != nil {
		return fmt.Errorf("dev reset: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pool, err := dbenv.Pool(ctx, base, pol)
	if err != nil {
		return redactDSN(fmt.Errorf("dev reset: %w", err), base)
	}
	defer pool.Close()
	if err := dbenv.CheckPGMajor(ctx, pool); err != nil {
		return err
	}
	dbs, err := dbenv.ListRunDBs(ctx, pool)
	if err != nil {
		return err
	}
	drop := dbenv.StaleRunDBs(dbs, time.Now(), *olderThan)
	kept := len(dbs) - len(drop)
	if *dryRun || !*confirm {
		fmt.Printf("would drop %d run database(s), keep %d (young or busy):\n", len(drop), kept)
		for _, name := range drop {
			fmt.Printf("  %s\n", name)
		}
		return nil
	}
	for _, name := range drop {
		if !dbenv.IsRunDB(name) {
			return fmt.Errorf("dev reset: refusing non-run database %q", name)
		}
		if _, err := pool.Exec(ctx, `DROP DATABASE `+dbenv.QuoteIdent(name)); err != nil {
			return fmt.Errorf("dev reset: drop %s: %w", name, err)
		}
		fmt.Printf("dropped %s\n", name)
	}
	fmt.Printf("reset complete: dropped %d, kept %d\n", len(drop), kept)
	return nil
}

// devTestArgv builds the go test invocation. Common flags are
// first-class (so `task dev:test -- -v` works without remembering an
// escape hatch); anything after a `--` separator passes through
// verbatim for the long tail of go test flags.
func devTestArgv(runPattern, count string, verbose bool, timeout string, pkgs, extra []string) []string {
	argv := []string{"test", "-p", "1", "-count", count}
	if runPattern != "" {
		argv = append(argv, "-run", runPattern)
	}
	if verbose {
		argv = append(argv, "-v")
	}
	if timeout != "" {
		argv = append(argv, "-timeout", timeout)
	}
	argv = append(argv, pkgs...)
	argv = append(argv, extra...)
	return argv
}

// dbTestPackages pins the packages whose tests touch the database,
// determined by testdb/store imports. The unit test below asserts
// every entry exists so the list cannot rot silently.
var dbTestPackages = []string{
	"./store/...",
	"./server/...",
	"./tools/...",
	"./testdb/...",
	"./runner/...",
	"./spike/...",
	"./sandbox/...",
	"./cmd/studio/...",
}

func runDevTest(args []string) error {
	fs := flag.NewFlagSet("dev test", flag.ContinueOnError)
	dbOnly := fs.Bool("db-only", false, "run only database-touching packages")
	runPattern := fs.String("run", "", "pass -run to go test")
	count := fs.String("count", "1", "pass -count to go test")
	verbose := fs.Bool("v", false, "pass -v to go test (streaming per-test progress)")
	timeout := fs.String("timeout", "", "pass -timeout to go test (e.g. 5m)")
	dsnFile := fs.String("dsn-file", "", "DSN file from dev bind (with --metadata-file; child env only, never logged)")
	metadataFile := fs.String("metadata-file", "", "branch binding JSON from dev bind (with --dsn-file)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	pkgs := []string{"./..."}
	if *dbOnly {
		pkgs = dbTestPackages
	}
	argv := devTestArgv(*runPattern, *count, *verbose, *timeout, pkgs, fs.Args())
	root, err := repoRoot()
	if err != nil {
		return err
	}
	cmd := exec.Command("go", argv...)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// File handoff: validate the binding, then hand files to the test
	// subprocess as environment. The DSN bytes stay in-process (owner
	// terminal only) and are never printed. Without flags, the current
	// environment passes through unchanged (CI/T2 path).
	if *metadataFile != "" || *dsnFile != "" {
		if *metadataFile == "" || *dsnFile == "" {
			return fmt.Errorf("dev test: --metadata-file and --dsn-file go together")
		}
		pol, err := dbenv.PolicyFromFile(*metadataFile)
		if err != nil {
			return err
		}
		if err := dbenv.RefuseProductionBranch(pol); err != nil {
			return fmt.Errorf("dev test: %w", err)
		}
		raw, err := os.ReadFile(*dsnFile)
		if err != nil {
			return fmt.Errorf("dev test: read DSN file: %v", err)
		}
		dsn := strings.TrimSpace(string(raw))
		for i := range raw {
			raw[i] = 0
		}
		if _, err := dbenv.CheckEndpoint(dsn, pol); err != nil {
			return err
		}
		protected := "false"
		if pol.Protected {
			protected = "true"
		}
		cmd.Env = append(os.Environ(),
			"TEST_DATABASE_URL="+dsn,
			"NEON_PROJECT_ID="+pol.ProjectID,
			"NEON_BRANCH_ID="+pol.BranchID,
			"NEON_PARENT_BRANCH="+pol.ParentBranch,
			"NEON_ENDPOINT_HOSTS="+strings.Join(pol.Hosts, ","),
			"NEON_PROJECT_NAME="+pol.ProjectName,
			"NEON_BRANCH_NAME="+pol.BranchName,
			"NEON_BRANCH_PROTECTED="+protected,
		)
	}
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			os.Exit(exit.ExitCode())
		}
		return err
	}
	return nil
}
