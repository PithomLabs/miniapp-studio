package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PithomLabs/miniapp-studio/server"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp/testkit"
)

// provisionDB prepares the shared test database idempotently and reserves
// the provision-test tenants, removing their rows before and after.
func provisionDB(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	testkit.MigrateUp(t, dsn)
	ctx := context.Background()
	owner, err := store.NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	pass, err := store.RandomHex(32)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureAppRole(ctx, owner, pass); err != nil {
		t.Fatalf("EnsureAppRole: %v", err)
	}
	if err := store.Migrate(ctx, dsn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	clean := func() {
		_, _ = owner.Exec(ctx, `DELETE FROM studio.operators WHERE tenant_id IN ('t-prov','t-prov-b')`)
	}
	clean()
	t.Cleanup(func() { clean(); owner.Close() })
	return dsn
}

func writeSecret(t *testing.T, name, value string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func operatorActive(t *testing.T, dsn, tenant string) bool {
	t.Helper()
	ctx := context.Background()
	owner, err := store.NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	var active bool
	if err := owner.QueryRow(ctx,
		`SELECT active FROM studio.operators WHERE tenant_id = $1`, tenant,
	).Scan(&active); err != nil {
		t.Fatalf("operator row: %v", err)
	}
	return active
}

func readOperators(t *testing.T, path string) []operatorEntry {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Operators []operatorEntry `json:"operators"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Operators
}

// F13: fresh provision creates the operator row and a 0600 operators file.
// The newline-terminated token file also proves the F8 trim: the stored
// hash is over the bare token.
func TestProvisionFresh(t *testing.T) {
	dsn := provisionDB(t)
	dir := t.TempDir()
	tokenFile := writeSecret(t, "token", "prov-token-1\n")
	passFile := writeSecret(t, "pass", mustHex(t)+"\n")
	dsnFile := writeSecret(t, "dsn", dsn+"\n")
	opsFile := filepath.Join(dir, "operators.json")
	err := runProvision([]string{
		"--tenant", "t-prov", "--binding", "prov-op",
		"--framework-credential", "prov-svc",
		"--login-token-file", tokenFile,
		"--app-role-password-file", passFile,
		"--operators-file", opsFile,
		"--dsn-file", dsnFile,
	})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	if !operatorActive(t, dsn, "t-prov") {
		t.Error("operator row missing or inactive")
	}
	fi, err := os.Stat(opsFile)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("operators file mode = %o, want 600", fi.Mode().Perm())
	}
	ops := readOperators(t, opsFile)
	if len(ops) != 1 {
		t.Fatalf("operators = %d entries, want 1", len(ops))
	}
	sum := sha256.Sum256([]byte("prov-token-1"))
	if ops[0].LoginTokenSHA256 != hex.EncodeToString(sum[:]) {
		t.Error("stored hash is not over the trimmed token")
	}
	if ops[0].FrameworkCredential != "prov-svc" || ops[0].Binding != "prov-op" {
		t.Errorf("entry = %+v", ops[0])
	}
}

// F13: re-provisioning with a new token rotates the hash; re-running with
// the same token is byte-identical; other tenants are preserved.
func TestProvisionRotationAndIdempotency(t *testing.T) {
	dsn := provisionDB(t)
	dir := t.TempDir()
	opsFile := filepath.Join(dir, "operators.json")
	prov := func(tenant, binding, token string) {
		t.Helper()
		tokenFile := writeSecret(t, "token", token+"\n")
		passFile := writeSecret(t, "pass", mustHex(t)+"\n")
		if err := runProvision([]string{
			"--tenant", tenant, "--binding", binding,
			"--framework-credential", "svc-" + tenant,
			"--login-token-file", tokenFile,
			"--app-role-password-file", passFile,
			"--operators-file", opsFile,
			"--dsn-file", writeSecret(t, "dsn", dsn),
		}); err != nil {
			t.Fatalf("provision %s: %v", tenant, err)
		}
	}
	prov("t-prov", "prov-op", "tok-one")
	before, _ := os.ReadFile(opsFile)
	prov("t-prov", "prov-op", "tok-two")
	rotated, _ := os.ReadFile(opsFile)
	sum := sha256.Sum256([]byte("tok-two"))
	ops := readOperators(t, opsFile)
	if len(ops) != 1 || ops[0].LoginTokenSHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("rotation did not replace the hash: %+v", ops)
	}
	if string(before) == string(rotated) {
		t.Fatal("rotated file identical to pre-rotation bytes")
	}
	prov("t-prov", "prov-op", "tok-two")
	again, _ := os.ReadFile(opsFile)
	if string(again) != string(rotated) {
		t.Error("same-input re-provision is not byte-identical")
	}
	prov("t-prov-b", "prov-op-b", "tok-b")
	ops = readOperators(t, opsFile)
	if len(ops) != 2 {
		t.Fatalf("operators = %d entries, want 2 (preservation)", len(ops))
	}
	if !operatorActive(t, dsn, "t-prov") || !operatorActive(t, dsn, "t-prov-b") {
		t.Error("operator rows missing after rotation")
	}
}

func mustHex(t *testing.T) string {
	t.Helper()
	hexPass, err := store.RandomHex(32)
	if err != nil {
		t.Fatal(err)
	}
	return hexPass
}

// F13: redaction removes the DSN password and preserves clean errors
// untouched, chain included.
func TestRedactDSN(t *testing.T) {
	sentinel := errors.New("boom")
	if got := redactDSN(sentinel, "postgres://u:pw@h/db"); got != sentinel {
		t.Error("clean error was rewritten")
	}
	leaky := errors.New(`connect postgres://op:Xk9-redact-me-zz42@db:5432/app failed`)
	got := redactDSN(leaky, "postgres://op:Xk9-redact-me-zz42@db:5432/app")
	if strings.Contains(got.Error(), "Xk9-redact-me-zz42") {
		t.Errorf("password survived: %q", got.Error())
	}
	unparseable := errors.New(`bad :://oops-Xk9-redact-me-zz42`)
	got = redactDSN(unparseable, "://oops-Xk9-redact-me-zz42")
	if strings.Contains(got.Error(), "Xk9-redact-me-zz42") {
		t.Errorf("password survived unparseable path: %q", got.Error())
	}
}

// F13: migrate failures against a real (unreachable) DSN never echo its
// password.
func TestMigrateRedactsPassword(t *testing.T) {
	u, err := url.Parse(os.Getenv("TEST_DATABASE_URL"))
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("nobody", "Xk9-redact-me-zz42")
	u.Host = "127.0.0.1:1"
	err = runMigrate([]string{"--dsn-file", writeSecret(t, "dsn", u.String())})
	if err == nil {
		t.Fatal("migrate against a dead port succeeded")
	}
	if strings.Contains(err.Error(), "Xk9-redact-me-zz42") {
		t.Errorf("password leaked: %q", err.Error())
	}
}

// F13: the serve happy path launches the real server and answers healthz.
func TestServeHappyPath(t *testing.T) {
	dsn := provisionDB(t)
	ctx := context.Background()
	owner, err := store.NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	pass, err := store.RandomHex(32)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureAppRole(ctx, owner, pass); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertOperator(ctx, owner, "t-prov", "prov-op", true); err != nil {
		t.Fatal(err)
	}
	appDSN := dsn
	if parsed, perr := url.Parse(dsn); perr == nil {
		parsed.User = url.UserPassword(store.AppRoleName, pass)
		appDSN = parsed.String()
	}
	sum := sha256.Sum256([]byte("serve-token"))
	cfg := server.Config{
		ListenAddr: "127.0.0.1:0", DatabaseURL: appDSN,
		FrameworkKeys:   writeSecret(t, "keys", `{"prov-svc":{"agent":"a","kind":"service","id":"s","tenant":"t-prov"}}`),
		FrameworkGrants: writeSecret(t, "grants", `{"t-prov":{"studio-runner":"autonomous","studio":"autonomous"}}`),
		Operators: []server.Operator{{
			Tenant: "t-prov", Binding: "prov-op",
			LoginTokenSHA256: hex.EncodeToString(sum[:]), FrameworkCredential: "prov-svc",
		}},
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- serveOn(runCtx, cfg, ln) }()
	base := "http://" + ln.Addr().String()
	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := http.Get(base + "/healthz")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("healthz never answered: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serveOn: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("server did not shut down")
	}
}
