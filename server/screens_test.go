package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/fly"
)

func postForm(t *testing.T, srv *Server, target string, values url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return doRequest(t, srv, http.MethodPost, target, values.Encode(), cookies...)
}

func validSpecForTenant(t *testing.T, tenant string) string {
	t.Helper()
	raw, err := os.ReadFile("../spec/testdata/valid_invoice_review.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc["app"].(map[string]any)["owner"] = tenant
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func opkeyForTest(t *testing.T) string {
	t.Helper()
	key, err := newOpKey()
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func createAppViaRoute(t *testing.T, srv *Server, cookie *http.Cookie, slug, key string) string {
	t.Helper()
	rec := postForm(t, srv, "/apps", url.Values{
		"opkey": {key}, "slug": {slug}, "name": {slug}, "description": {"d"},
	}, cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("create status = %d, body %.200q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	id := strings.TrimPrefix(loc, "/apps/")
	if id == "" || id == loc {
		t.Fatalf("location = %q", loc)
	}
	return id
}

func TestHomeStates(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	empty := doRequest(t, srv, http.MethodGet, "/", "", cookie)
	if empty.Code != 200 || !strings.Contains(empty.Body.String(), "No applications yet") {
		t.Errorf("empty home missing empty state: %d", empty.Code)
	}
	createAppViaRoute(t, srv, cookie, "shop", opkeyForTest(t))
	full := doRequest(t, srv, http.MethodGet, "/", "", cookie)
	if !strings.Contains(full.Body.String(), "shop") {
		t.Error("home missing created app")
	}
}

func TestCreateValidatePreview(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	form := doRequest(t, srv, http.MethodGet, "/apps/new", "", cookie)
	if form.Code != 200 || !strings.Contains(form.Body.String(), "opkey") {
		t.Fatalf("create form = %d", form.Code)
	}
	doc := validSpecForTenant(t, testTenant)
	valid := postForm(t, srv, "/apps/new/validate", url.Values{
		"opkey": {opkeyForTest(t)}, "spec": {doc},
	}, cookie)
	if valid.Code != 200 || !strings.Contains(valid.Body.String(), "Content hash:") {
		t.Errorf("validate valid spec: %d", valid.Code)
	}
	invalid := postForm(t, srv, "/apps/new/validate", url.Values{
		"opkey": {opkeyForTest(t)}, "spec": {`{"nope":true}`},
	}, cookie)
	if invalid.Code != 200 || !strings.Contains(invalid.Body.String(), "E_SCHEMA") {
		t.Errorf("validate invalid spec lacks error code: %d", invalid.Code)
	}
	preview := postForm(t, srv, "/apps/new/preview", url.Values{
		"opkey": {opkeyForTest(t)}, "spec": {doc},
		"workflow": {"prepare_review"}, "fixture": {"one-pending-invoice"},
	}, cookie)
	if preview.Code != 200 || !strings.Contains(preview.Body.String(), "comparison") {
		t.Errorf("preview: %d %.300q", preview.Code, preview.Body.String())
	}
}

func TestAppsRoutesIdempotencyAndIsolation(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	cookieB := login(t, srv, testTokenB)
	key := opkeyForTest(t)
	id1 := createAppViaRoute(t, srv, cookie, "shop", key)
	// Same key resubmitted with the identical payload attaches.
	rec := postForm(t, srv, "/apps", url.Values{
		"opkey": {key}, "slug": {"shop"}, "name": {"shop"}, "description": {"d"},
	}, cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("resubmit status = %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/apps/"+id1 {
		t.Errorf("resubmit location = %q, want attach to %s", loc, id1)
	}
	// Fresh key with the identical payload is an independent operation
	// that fails honestly on the taken slug.
	rec = postForm(t, srv, "/apps", url.Values{
		"opkey": {opkeyForTest(t)}, "slug": {"shop"}, "name": {"shop"},
	}, cookie)
	if rec.Code == http.StatusFound {
		t.Error("duplicate slug accepted on fresh key")
	}
	// Cross-tenant reads are uniform 404s.
	if rec := doRequest(t, srv, http.MethodGet, "/apps/"+id1, "", cookieB); rec.Code != 404 {
		t.Errorf("cross-tenant detail = %d, want 404", rec.Code)
	}
	// Unknown apps are the same 404.
	if rec := doRequest(t, srv, http.MethodGet, "/apps/does-not-exist", "", cookie); rec.Code != 404 {
		t.Errorf("unknown app = %d, want 404", rec.Code)
	}
	// Spec versions save and show.
	doc := validSpecForTenant(t, testTenant)
	saved := postForm(t, srv, "/apps/"+id1+"/specs", url.Values{
		"opkey": {opkeyForTest(t)}, "spec": {doc},
	}, cookie)
	if saved.Code != http.StatusFound {
		t.Fatalf("save spec = %d", saved.Code)
	}
	detail := doRequest(t, srv, http.MethodGet, "/apps/"+id1, "", cookie)
	if !strings.Contains(detail.Body.String(), "v1") && !strings.Contains(detail.Body.String(), "version 1") {
		t.Errorf("detail lacks version history: %.300q", detail.Body.String())
	}
	// Missing opkey fails closed.
	bad := postForm(t, srv, "/apps", url.Values{"slug": {"x"}, "name": {"x"}}, cookie)
	if bad.Code != http.StatusBadRequest {
		t.Errorf("missing opkey = %d, want 400", bad.Code)
	}
}

func TestDeployFlow(t *testing.T) {
	sprites := fly.NewFakeSpriteManager()
	conns := fly.NewFakeConnectorManager()
	srv, _ := testServerWith(t, sprites, conns)
	cookie := login(t, srv, testToken)
	id := createAppViaRoute(t, srv, cookie, "shop", opkeyForTest(t))
	postForm(t, srv, "/apps/"+id+"/specs", url.Values{
		"opkey": {opkeyForTest(t)}, "spec": {validSpecForTenant(t, testTenant)},
	}, cookie)
	page := doRequest(t, srv, http.MethodGet, "/apps/"+id+"/deploy", "", cookie)
	if page.Code != 200 || !strings.Contains(page.Body.String(), "Provision") {
		t.Fatalf("deploy page = %d", page.Code)
	}
	prov := postForm(t, srv, "/apps/"+id+"/deploy", url.Values{
		"opkey": {opkeyForTest(t)}, "confirm": {"yes"},
	}, cookie)
	if prov.Code != 200 || !strings.Contains(prov.Body.String(), "ready") {
		t.Errorf("provision = %d %.300q", prov.Code, prov.Body.String())
	}
	status := doRequest(t, srv, http.MethodGet, "/apps/"+id+"/deploy", "", cookie)
	if !strings.Contains(status.Body.String(), "https://") {
		t.Error("deployed URL not shown")
	}
	retire := postForm(t, srv, "/apps/"+id+"/retire", url.Values{
		"opkey": {opkeyForTest(t)}, "confirm": {"yes"},
	}, cookie)
	if retire.Code != http.StatusFound {
		t.Errorf("retire = %d", retire.Code)
	}
	if len(sprites.Sprites) != 0 {
		t.Errorf("sprites remain: %d", len(sprites.Sprites))
	}
	// Confirmation is mandatory.
	id2 := createAppViaRoute(t, srv, cookie, "other", opkeyForTest(t))
	postForm(t, srv, "/apps/"+id2+"/specs", url.Values{
		"opkey": {opkeyForTest(t)}, "spec": {validSpecForTenant(t, testTenant)},
	}, cookie)
	noConfirm := postForm(t, srv, "/apps/"+id2+"/deploy", url.Values{
		"opkey": {opkeyForTest(t)},
	}, cookie)
	if noConfirm.Code != http.StatusBadRequest {
		t.Errorf("unconfirmed provision = %d, want 400", noConfirm.Code)
	}
}

func TestDeployUnconfiguredHonest(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	id := createAppViaRoute(t, srv, cookie, "shop", opkeyForTest(t))
	page := doRequest(t, srv, http.MethodGet, "/apps/"+id+"/deploy", "", cookie)
	if !strings.Contains(page.Body.String(), "not configured") {
		t.Error("unconfigured state not disclosed")
	}
	prov := postForm(t, srv, "/apps/"+id+"/deploy", url.Values{
		"opkey": {opkeyForTest(t)}, "confirm": {"yes"},
	}, cookie)
	if prov.Code == http.StatusFound {
		t.Error("unconfigured provision redirected as if successful")
	}
}

// gatedSprites blocks sprite creation until released, proving the
// interstitial path against genuinely slow backends.
type gatedSprites struct {
	fly.SpriteManager
	gate chan struct{}
}

func (g *gatedSprites) CreateSprite(ctx context.Context, name string, opts fly.SpriteOptions) (fly.Sprite, error) {
	<-g.gate
	mgr, _ := g.SpriteManager.(*fly.FakeSpriteManager)
	return mgr.CreateSprite(ctx, name, opts)
}

func TestDeployInterstitial(t *testing.T) {
	gate := make(chan struct{})
	inner := fly.NewFakeSpriteManager()
	srv, _ := testServerWith(t, &gatedSprites{SpriteManager: inner, gate: gate}, fly.NewFakeConnectorManager())
	cookie := login(t, srv, testToken)
	id := createAppViaRoute(t, srv, cookie, "shop", opkeyForTest(t))
	postForm(t, srv, "/apps/"+id+"/specs", url.Values{
		"opkey": {opkeyForTest(t)}, "spec": {validSpecForTenant(t, testTenant)},
	}, cookie)
	type result struct {
		rec *httptest.ResponseRecorder
	}
	done := make(chan result, 1)
	go func() {
		done <- result{postForm(t, srv, "/apps/"+id+"/deploy", url.Values{
			"opkey": {opkeyForTest(t)}, "confirm": {"yes"},
		}, cookie)}
	}()
	// The POST must return the interstitial while the backend is stuck...
	select {
	case res := <-done:
		if res.rec.Code != http.StatusAccepted {
			t.Fatalf("slow provision = %d, want 202 interstitial", res.rec.Code)
		}
		if body := res.rec.Body.String(); !strings.Contains(body, "refresh") {
			t.Errorf("interstitial lacks refresh: %.200q", body)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("slow provision never responded")
	}
	// ...and the deployment completes once the backend proceeds.
	close(gate)
	deadline := time.Now().Add(10 * time.Second)
	for {
		page := doRequest(t, srv, http.MethodGet, "/apps/"+id+"/deploy", "", cookie)
		if strings.Contains(page.Body.String(), "ready") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("deployment never ready: %.300q", page.Body.String())
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestConnectorsScreens(t *testing.T) {
	conns := fly.NewFakeConnectorManager()
	conns.Connections = []fly.Connection{{ID: "c1", Provider: "box", ConnectionType: "oauth"}}
	srv, _ := testServerWith(t, fly.NewFakeSpriteManager(), conns)
	cookie := login(t, srv, testToken)
	page := doRequest(t, srv, http.MethodGet, "/connectors", "", cookie)
	if !strings.Contains(page.Body.String(), "c1") {
		t.Errorf("explorer lacks connection: %.300q", page.Body.String())
	}
	wizard := doRequest(t, srv, http.MethodGet, "/connectors/custom/new", "", cookie)
	if wizard.Code != 200 {
		t.Fatalf("wizard = %d", wizard.Code)
	}
	marker := "sk-test-marker-" + opkeyForTest(t)
	tested := postForm(t, srv, "/connectors/custom/test", url.Values{
		"opkey": {opkeyForTest(t)}, "name": {"acme"},
		"base_url": {"https://api.example"}, "auth_method": {"header"},
		"credential": {marker},
	}, cookie)
	if tested.Code != 200 {
		t.Fatalf("wizard test = %d", tested.Code)
	}
	if strings.Contains(tested.Body.String(), marker) {
		t.Error("credential echoed in test response")
	}
	created := postForm(t, srv, "/connectors/custom/create", url.Values{
		"opkey": {opkeyForTest(t)}, "name": {"acme"},
		"base_url": {"https://api.example"}, "auth_method": {"header"},
		"credential": {marker},
	}, cookie)
	if created.Code != http.StatusFound {
		t.Fatalf("wizard create = %d", created.Code)
	}
	assertMarkerAbsent(t, marker)
}

func TestConnectorsUnconfigured(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	page := doRequest(t, srv, http.MethodGet, "/connectors", "", cookie)
	if !strings.Contains(page.Body.String(), "not configured") {
		t.Error("unconfigured connectors state not disclosed")
	}
}

func TestActivityScreen(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	truncateAudit(t)
	empty := doRequest(t, srv, http.MethodGet, "/activity", "", cookie)
	if !strings.Contains(empty.Body.String(), "No activity yet") {
		t.Errorf("empty activity state missing: %.200q", empty.Body.String())
	}
	createAppViaRoute(t, srv, cookie, "shop", opkeyForTest(t))
	full := doRequest(t, srv, http.MethodGet, "/activity", "", cookie)
	if !strings.Contains(full.Body.String(), "app.create") {
		t.Errorf("activity lacks tool event: %.300q", full.Body.String())
	}
}

func TestCSRFEnforced(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	bare := httptest.NewRequest(http.MethodPost, "/apps", strings.NewReader("opkey=x"))
	bare.AddCookie(cookie)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, bare)
	if rec.Code != http.StatusForbidden {
		t.Errorf("missing Origin = %d, want 403", rec.Code)
	}
	forged := httptest.NewRequest(http.MethodPost, "/apps", strings.NewReader("opkey=x"))
	forged.AddCookie(cookie)
	forged.Header.Set("Origin", "https://evil.test")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, forged)
	if rec.Code != http.StatusForbidden {
		t.Errorf("forged Origin = %d, want 403", rec.Code)
	}
	lreq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("token=x"))
	lrec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(lrec, lreq)
	if lrec.Code != http.StatusForbidden {
		t.Errorf("login without Origin = %d, want 403", lrec.Code)
	}
}

func TestMutationsRequireAuth(t *testing.T) {
	srv, _ := testServer(t)
	for _, target := range []string{"/apps", "/apps/x/specs", "/apps/x/deploy", "/connectors/custom/test"} {
		req := httptest.NewRequest(http.MethodPost, target, strings.NewReader("opkey=x"))
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("POST %s unauthenticated = %d, want 401", target, rec.Code)
		}
	}
}

// assertMarkerAbsent scans every persisted request/response surface for a
// credential marker: invocation inputs and results plus audit details.
func assertMarkerAbsent(t *testing.T, marker string) {
	t.Helper()
	ctx := context.Background()
	owner, err := pgxpool.New(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	var invocations, audit int
	if err := owner.QueryRow(ctx,
		`SELECT count(*) FROM app.invocations
		 WHERE input::TEXT LIKE '%' || $1 || '%' OR result::TEXT LIKE '%' || $1 || '%'`,
		marker).Scan(&invocations); err != nil {
		t.Fatalf("scan invocations: %v", err)
	}
	if err := owner.QueryRow(ctx,
		`SELECT count(*) FROM app.audit_events WHERE details::TEXT LIKE '%' || $1 || '%'`,
		marker).Scan(&audit); err != nil {
		t.Fatalf("scan audit: %v", err)
	}
	if invocations != 0 || audit != 0 {
		t.Errorf("credential marker persisted (invocations=%d audit=%d)", invocations, audit)
	}
}

func truncateAudit(t *testing.T) {
	t.Helper()
	owner, err := pgxpool.New(context.Background(), os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	if _, err := owner.Exec(context.Background(), `TRUNCATE app.audit_events`); err != nil {
		t.Fatalf("truncate audit: %v", err)
	}
}

func TestUnknownRoute404(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	rec := doRequest(t, srv, http.MethodGet, "/no-such-route", "", cookie)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown route = %d, want 404", rec.Code)
	}
}
