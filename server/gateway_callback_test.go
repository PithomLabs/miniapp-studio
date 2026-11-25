package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PithomLabs/miniapp-studio/gateway"
	"github.com/PithomLabs/miniapp-studio/tools"
)

// callbackServer builds a server with the test-only static callback keys.
func callbackServer(t *testing.T) *Server {
	t.Helper()
	srv, _ := testServer(t)
	srv.gwKeys = gateway.StaticProvider{}
	return srv
}

func setupBuild(t *testing.T, srv *Server, tenant string) (conv, build string) {
	t.Helper()
	ctx := context.Background()
	out, err := srv.invokeToolTenant(ctx, tenant, tools.ToolConversationCreate,
		map[string]any{"title": "cb"}, "cb-conv-"+tenant)
	if err != nil {
		t.Fatalf("conversation.create: %v", err)
	}
	conv, _ = out["conversation_id"].(string)
	out, err = srv.invokeToolTenant(ctx, tenant, tools.ToolBuildCreate,
		map[string]any{"conversation_id": conv}, "cb-build-"+tenant)
	if err != nil {
		t.Fatalf("build.create: %v", err)
	}
	build, _ = out["build_id"].(string)
	return conv, build
}

func postCallback(t *testing.T, srv *Server, body []byte) (int, map[string]any) {
	t.Helper()
	rec := doRequest(t, srv, http.MethodPost, "/gw/v1/callback", string(body))
	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode callback response: %v", err)
	}
	return rec.Code, decoded
}

func signedCallback(t *testing.T, build, workspace, msgID, name string, payload map[string]any) []byte {
	t.Helper()
	raw, err := gateway.Sign(gateway.StaticTestKey(), build, workspace, msgID, name, payload)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestCallbackHappyPath(t *testing.T) {
	srv := callbackServer(t)
	_, build := setupBuild(t, srv, testTenant)
	code, ack := postCallback(t, srv, signedCallback(t, build, testTenant,
		"cb-m1", "agent.started", map[string]any{"agent_session_id": "s-1"}))
	if code != http.StatusOK || ack["ack"] != true {
		t.Fatalf("started = %d %v", code, ack)
	}
	if ack["transitioned"] != true || ack["build_status"] != "running" {
		t.Errorf("started ack = %v", ack)
	}
	code, ack = postCallback(t, srv, signedCallback(t, build, testTenant,
		"cb-m2", "agent.question",
		map[string]any{"question_id": "q-1", "body": "Which?"}))
	if code != http.StatusOK || ack["build_status"] != "waiting_user" {
		t.Fatalf("question = %d %v", code, ack)
	}
	// Exact redelivery re-executes under a fresh attempt key and hits the
	// ingest dedup: acknowledged as duplicate with no reapplied effect.
	code, ack = postCallback(t, srv, signedCallback(t, build, testTenant,
		"cb-m2", "agent.question",
		map[string]any{"question_id": "q-1", "body": "Which?"}))
	if code != http.StatusOK || ack["duplicate"] != true {
		t.Fatalf("redelivery = %d %v, want duplicate ack", code, ack)
	}
	st, err := srv.invokeToolTenant(context.Background(), testTenant,
		tools.ToolBuildStatus, map[string]any{"build_id": build}, "cb-st-dup")
	if err != nil {
		t.Fatal(err)
	}
	if qs, _ := st["open_questions"].([]any); len(qs) != 1 {
		t.Errorf("open questions after redelivery = %d, want 1", len(qs))
	}
}

func TestCallbackRejects(t *testing.T) {
	srv := callbackServer(t)
	_, build := setupBuild(t, srv, testTenant)
	good := signedCallback(t, build, testTenant, "cb-r1", "agent.progress",
		map[string]any{"text": "hi"})
	tampered := func() []byte {
		var m map[string]any
		if err := json.Unmarshal(good, &m); err != nil {
			t.Fatal(err)
		}
		m["payload"] = map[string]any{"text": "evil"}
		out, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}()
	cases := map[string]struct {
		body []byte
		want int
	}{
		"malformed":     {[]byte("{oops"), http.StatusBadRequest},
		"tampered body": {tampered, http.StatusUnauthorized},
		"unknown build": {signedCallback(t, "no-such-build", testTenant, "cb-r2", "agent.progress", map[string]any{}), http.StatusNotFound},
		"cross tenant":  {signedCallback(t, build, testTenantB, "cb-r3", "agent.progress", map[string]any{}), http.StatusNotFound},
		"oversized":     {make([]byte, 2<<20), http.StatusBadRequest},
		"empty":         {[]byte{}, http.StatusBadRequest},
		"wrong key":     {mustResign(t, good), http.StatusUnauthorized},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			code, _ := postCallback(t, srv, tc.body)
			if code != tc.want {
				t.Errorf("status = %d, want %d", code, tc.want)
			}
		})
	}
}

// mustResign re-signs the same envelope bytes with a wrong key.
func mustResign(t *testing.T, raw []byte) []byte {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	m["signature"] = strings.Repeat("0", 64)
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestCallbackViaGatewayOutbox proves both sides interoperate: a signed
// envelope travels outbox → HTTP → the real callback route → ingest.
func TestCallbackViaGatewayOutbox(t *testing.T) {
	srv := callbackServer(t)
	_, build := setupBuild(t, srv, testTenant)
	httpSrv := httptest.NewServer(srv.Handler())
	defer httpSrv.Close()
	o, err := gateway.OpenOutbox(t.TempDir() + "/outbox.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := gateway.Sign(gateway.StaticTestKey(), build, testTenant,
		"joint-m1", "agent.started", map[string]any{"agent_session_id": "joint-s"})
	if err != nil {
		t.Fatal(err)
	}
	if err := o.Enqueue(gateway.OutboxItem{
		ID: "joint-m1", BuildID: build, MessageID: "joint-m1", Envelope: raw,
	}); err != nil {
		t.Fatal(err)
	}
	d := &gateway.Deliverer{Outbox: o, Studio: httpSrv.URL, Keys: gateway.StaticProvider{},
		Tick: 5 * time.Millisecond, BaseWait: time.Millisecond, MaxWait: 5 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	d.Run(ctx)
	if q, f := o.Depth(); q+f != 0 {
		t.Fatalf("outbox depth = %d/%d, want drained", q, f)
	}
	st, err := srv.invokeToolTenant(context.Background(), testTenant,
		tools.ToolBuildStatus, map[string]any{"build_id": build}, "joint-st")
	if err != nil {
		t.Fatal(err)
	}
	if st["status"] != "running" || st["agent_session_ref"] != "joint-s" {
		t.Errorf("build = %v, want running/joint-s", st)
	}
}

// TestCallbackKDFWiring proves New builds the KDF provider from config:
// a callback signed with the derived key verifies with no explicit
// provider injected.
func TestCallbackKDFWiring(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	secret := "kdf-org-secret-0123456789abcdef"
	srv, _ := bootServerOn(t, dsn, nil, nil, true, func(c *Config) {
		c.GatewayOrgSecret = secret
	})
	_, build := setupBuild(t, srv, testTenant)
	key := gateway.DeriveKey([]byte(secret), build)
	raw, err := gateway.Sign(key, build, testTenant, "kdf-m1", "agent.progress",
		map[string]any{"text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	code, ack := postCallback(t, srv, raw)
	if code != http.StatusOK || ack["ack"] != true {
		t.Fatalf("kdf callback = %d %v", code, ack)
	}
}

// TestCallbackKDFRejectsWrongSecret (M15): the production KDF path
// enforces its keys — a callback signed under a different org secret
// is unauthorized, proving verification actually gates on derivation.
func TestCallbackKDFRejectsWrongSecret(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	secret := "kdf-org-secret-0123456789abcdef"
	srv, _ := bootServerOn(t, dsn, nil, nil, true, func(c *Config) {
		c.GatewayOrgSecret = secret
	})
	_, build := setupBuild(t, srv, testTenant)
	wrong := gateway.DeriveKey([]byte("different-org-secret-abcdef0123456789"), build)
	raw, err := gateway.Sign(wrong, build, testTenant, "kdf-w1", "agent.progress",
		map[string]any{"text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	code, _ := postCallback(t, srv, raw)
	if code != http.StatusUnauthorized {
		t.Fatalf("wrong-secret callback = %d, want 401", code)
	}
}

func TestCallbackUnconfigured(t *testing.T) {
	srv, _ := testServer(t)
	srv.gwKeys = nil
	code, _ := postCallback(t, srv, []byte(`{}`))
	if code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", code)
	}
}

// TestCallbackDuplicateQuestionIDFailsTerminally pins handoff honesty: a
// question id reused under a fresh event id is a deterministic gateway
// bug, so it fails 400 (terminal) instead of 500 (retry poison).
func TestCallbackDuplicateQuestionIDFailsTerminally(t *testing.T) {
	srv := callbackServer(t)
	_, build := setupBuild(t, srv, testTenant)
	code, _ := postCallback(t, srv, signedCallback(t, build, testTenant,
		"dup-e1", "agent.question", map[string]any{"question_id": "dup-1", "body": "Q?"}))
	if code != http.StatusOK {
		t.Fatalf("first question = %d", code)
	}
	code, ack := postCallback(t, srv, signedCallback(t, build, testTenant,
		"dup-e2", "agent.question", map[string]any{"question_id": "dup-1", "body": "Q again?"}))
	if code != http.StatusBadRequest {
		t.Fatalf("reused question id = %d %v, want 400", code, ack)
	}
}
