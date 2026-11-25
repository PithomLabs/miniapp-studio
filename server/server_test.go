package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp-studio/fly"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp/testkit"
)

const (
	testTenant   = "t-web"
	testTenantB  = "t-web-b"
	testBinding  = "web-op"
	testBindingB = "web-op-b"
	testToken    = "web-operator-token-for-tests-only"
	testTokenB   = "web-operator-token-b-for-tests-only"
	testSvcCred  = "test-svc-cred"
	testSvcCredB = "test-svc-cred-b"
)

// testServer builds a full server against the test database with one
// provisioned operator. It fails the test when TEST_DATABASE_URL is unset.
func testServer(t *testing.T) (*Server, Config) {
	return testServerWith(t, nil, nil)
}

// testServerWith additionally injects Fly managers (fakes in tests).
func testServerWith(t *testing.T, sprites fly.SpriteManager, conns fly.ConnectorManager) (*Server, Config) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	return testServerOn(t, dsn, sprites, conns)
}

// testServerOn builds the same server against an explicit database. The F12
// gate tests use it with a scratch database.
func testServerOn(t *testing.T, dsn string, sprites fly.SpriteManager, conns fly.ConnectorManager) (*Server, Config) {
	t.Helper()
	return bootServerOn(t, dsn, sprites, conns, true, nil)
}

// bootServerOn boots a server, truncating first unless told not to. The
// Phase A restart test boots a second server without truncating to prove a
// fresh process serves pre-existing rows. mutate adjusts the config before
// New for tests covering config-driven wiring.
func bootServerOn(t *testing.T, dsn string, sprites fly.SpriteManager, conns fly.ConnectorManager, truncate bool, mutate func(*Config)) (*Server, Config) {
	t.Helper()
	ctx := context.Background()
	testkit.MigrateUp(t, dsn)
	owner, err := store.NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(owner.Close)
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
	if truncate {
		if _, err := owner.Exec(ctx, `TRUNCATE studio.sessions, studio.operators, studio.apps,
		studio.app_specs, studio.deployments, studio.runs`); err != nil {
			t.Fatalf("truncate: %v", err)
		}
		if _, err := owner.Exec(ctx, `TRUNCATE studio.conversations, studio.messages, studio.builds,
		studio.turns, studio.questions, studio.commands, studio.events,
		studio.conversation_stream, studio.artifacts, studio.approvals`); err != nil {
			t.Fatalf("truncate conversations: %v", err)
		}
		// Framework invocation records carry idempotency outcomes (including
		// recorded failures) across tests; each test starts hermetic.
		if _, err := owner.Exec(ctx, `TRUNCATE app.invocations, app.audit_events`); err != nil {
			t.Fatalf("truncate invocations: %v", err)
		}
	}
	if err := store.UpsertOperator(ctx, owner, testTenant, testBinding, true); err != nil {
		t.Fatalf("UpsertOperator: %v", err)
	}
	if err := store.UpsertOperator(ctx, owner, testTenantB, testBindingB, true); err != nil {
		t.Fatalf("UpsertOperator B: %v", err)
	}
	sum := sha256.Sum256([]byte(testToken))
	sumB := sha256.Sum256([]byte(testTokenB))
	ops := map[string]any{"operators": []any{
		map[string]any{
			"tenant": testTenant, "binding": testBinding,
			"login_token_sha256":   hex.EncodeToString(sum[:]),
			"framework_credential": testSvcCred,
		},
		map[string]any{
			"tenant": testTenantB, "binding": testBindingB,
			"login_token_sha256":   hex.EncodeToString(sumB[:]),
			"framework_credential": testSvcCredB,
		},
	}}
	opsRaw, _ := json.Marshal(ops)
	dir := t.TempDir()
	opsFile := filepath.Join(dir, "ops.json")
	keysFile := filepath.Join(dir, "keys.json")
	grantsFile := filepath.Join(dir, "grants.json")
	if err := os.WriteFile(opsFile, opsRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	keys := map[string]any{
		testSvcCred:  map[string]any{"agent": "svc-agent", "kind": "service", "id": "svc", "tenant": testTenant},
		testSvcCredB: map[string]any{"agent": "svc-agent-b", "kind": "service", "id": "svc-b", "tenant": testTenantB},
	}
	keysRaw, _ := json.Marshal(keys)
	if err := os.WriteFile(keysFile, keysRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	grants := map[string]any{
		testTenant:  map[string]any{"studio-runner": "autonomous", "studio": "autonomous"},
		testTenantB: map[string]any{"studio-runner": "autonomous", "studio": "autonomous"},
	}
	grantsRaw, _ := json.Marshal(grants)
	if err := os.WriteFile(grantsFile, grantsRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	appDSN := dsn
	if parsed, err := url.Parse(dsn); err == nil {
		parsed.User = url.UserPassword(store.AppRoleName, pass)
		appDSN = parsed.String()
	}
	cfg := Config{
		ListenAddr: "127.0.0.1:0", DatabaseURL: appDSN, OwnerDatabaseURL: dsn,
		FrameworkKeys: keysFile, FrameworkGrants: grantsFile,
		Operators: []Operator{
			{Tenant: testTenant, Binding: testBinding,
				LoginTokenSHA256: hex.EncodeToString(sum[:]), FrameworkCredential: testSvcCred},
			{Tenant: testTenantB, Binding: testBindingB,
				LoginTokenSHA256: hex.EncodeToString(sumB[:]), FrameworkCredential: testSvcCredB},
		},
	}
	if mutate != nil {
		mutate(&cfg)
	}
	srv, err := New(ctx, cfg, Options{Sprites: sprites, Connectors: conns})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := srv.Close(); err != nil {
			t.Errorf("server close: %v", err)
		}
	})
	return srv, cfg
}

func doRequest(t *testing.T, srv *Server, method, target, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if method == http.MethodPost {
		req.Header.Set("Origin", "http://example.com")
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func login(t *testing.T, srv *Server, token string) *http.Cookie {
	t.Helper()
	rec := doRequest(t, srv, http.MethodPost, "/login",
		(&url.Values{"token": {token}, "next": {"/"}}).Encode())
	if rec.Code != http.StatusFound {
		t.Fatalf("login status = %d, want 302", rec.Code)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookie {
			if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
				t.Fatalf("cookie flags = %+v", c)
			}
			return c
		}
	}
	t.Fatal("no session cookie set")
	return nil
}

func TestHealthzReadyz(t *testing.T) {
	srv, _ := testServer(t)
	if rec := doRequest(t, srv, http.MethodGet, "/healthz", ""); rec.Code != 200 {
		t.Errorf("healthz = %d", rec.Code)
	}
	if rec := doRequest(t, srv, http.MethodGet, "/readyz", ""); rec.Code != 200 {
		t.Errorf("readyz = %d", rec.Code)
	}
}

func TestUnauthenticatedRedirects(t *testing.T) {
	srv, _ := testServer(t)
	rec := doRequest(t, srv, http.MethodGet, "/", "")
	if rec.Code != http.StatusFound {
		t.Fatalf("/ status = %d, want 302 to login", rec.Code)
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if loc.Path != "/login" || loc.Query().Get("next") != "/" {
		t.Errorf("redirect = %q", rec.Header().Get("Location"))
	}
	rec = doRequest(t, srv, http.MethodGet, "/api/whoami", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("api status = %d, want 401", rec.Code)
	}
}

func TestLoginRejectsBadToken(t *testing.T) {
	srv, _ := testServer(t)
	rec := doRequest(t, srv, http.MethodPost, "/login",
		(&url.Values{"token": {"wrong"}, "next": {"/"}}).Encode())
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookie && c.Value != "" {
			t.Error("session cookie set on failed login")
		}
	}
}

// F5: the token matches the operators file but the operator row is
// deactivated — login must render the identical 401 page, not a 500.
func TestLoginUnprovisionedOperator401(t *testing.T) {
	srv, _ := testServer(t)
	ctx := context.Background()
	owner, err := store.NewPool(ctx, os.Getenv("TEST_DATABASE_URL"), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	if err := store.UpsertOperator(ctx, owner, testTenant, testBinding, false); err != nil {
		t.Fatal(err)
	}
	rec := doRequest(t, srv, http.MethodPost, "/login",
		(&url.Values{"token": {testToken}, "next": {"/"}}).Encode())
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Invalid credentials.") {
		t.Error("unprovisioned login lacks the identical 401 page")
	}
	if strings.Contains(body, "Something went wrong") {
		t.Error("unprovisioned login rendered the 500 page")
	}
}

func TestLoginFlow(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	rec := doRequest(t, srv, http.MethodGet, "/", "", cookie)
	if rec.Code != 200 {
		t.Fatalf("/ status = %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, testTenant) {
		t.Errorf("home lacks tenant %q", testTenant)
	}
	rec = doRequest(t, srv, http.MethodGet, "/api/whoami", "", cookie)
	var whoami map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&whoami); err != nil {
		t.Fatal(err)
	}
	if whoami["tenant"] != testTenant || whoami["binding"] != testBinding {
		t.Errorf("whoami = %v", whoami)
	}
}

func TestLoginNextValidation(t *testing.T) {
	srv, _ := testServer(t)
	rec := doRequest(t, srv, http.MethodPost, "/login",
		(&url.Values{"token": {testToken}, "next": {"https://evil.test/"}}).Encode())
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("redirect = %q, want /", loc)
	}
}

func TestLogoutRevokes(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	rec := doRequest(t, srv, http.MethodPost, "/logout", "", cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("logout status = %d", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet, "/", "", cookie)
	if rec.Code != http.StatusFound {
		t.Errorf("revoked session status = %d, want 302", rec.Code)
	}
}

func TestRevokedSessionDenied(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	if err := srv.Sessions().Revoke(context.Background(), store.HashToken(cookie.Value)); err != nil {
		t.Fatal(err)
	}
	if rec := doRequest(t, srv, http.MethodGet, "/", "", cookie); rec.Code != http.StatusFound {
		t.Errorf("revoked session status = %d, want 302", rec.Code)
	}
}

func TestForgedSessionDenied(t *testing.T) {
	srv, _ := testServer(t)
	forged := &http.Cookie{Name: SessionCookie, Value: "forged-token-value"}
	if rec := doRequest(t, srv, http.MethodGet, "/", "", forged); rec.Code != http.StatusFound {
		t.Errorf("forged session status = %d, want 302", rec.Code)
	}
}
