package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestConversationUIEndToEnd(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	rec := doRequest(t, srv, http.MethodGet, "/conversations", "", cookie)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "No conversations yet") {
		t.Fatalf("empty list = %d", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodPost, "/conversations",
		(&url.Values{"opkey": {opkeyForTest(t)}, "title": {"Desk chat"}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("create = %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/conversations/") {
		t.Fatalf("redirect = %q", loc)
	}
	rec = doRequest(t, srv, http.MethodGet, loc, "", cookie)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Desk chat") {
		t.Fatalf("view = %d", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodPost, loc+"/messages",
		(&url.Values{"opkey": {opkeyForTest(t)}, "body": {"Book the venue"}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("send = %d", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet, loc, "", cookie)
	body := rec.Body.String()
	for _, want := range []string{"Book the venue", "Build opened.", "assigned", "EventSource"} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q", want)
		}
	}
}

func TestAnswerFlowViaUI(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	conv, build := setupBuild(t, srv, testTenant)
	code, _ := postCallback(t, srv, signedCallback(t, build, testTenant,
		"ui-q1", "agent.question", map[string]any{"question_id": "ui-1", "body": "Which city?"}))
	if code != http.StatusOK {
		t.Fatalf("question callback = %d", code)
	}
	// The early yield parked on the assigned build (lost-started join);
	// the late started absorbs, and the re-ask supersedes ui-1 with ui-2.
	code, _ = postCallback(t, srv, signedCallback(t, build, testTenant,
		"ui-s0", "agent.started", map[string]any{}))
	if code != http.StatusOK {
		t.Fatalf("started callback = %d", code)
	}
	code, _ = postCallback(t, srv, signedCallback(t, build, testTenant,
		"ui-q2", "agent.question", map[string]any{"question_id": "ui-2", "body": "Which city, really?"}))
	if code != http.StatusOK {
		t.Fatalf("second question = %d", code)
	}
	rec := doRequest(t, srv, http.MethodGet, "/conversations/"+conv, "", cookie)
	if !strings.Contains(rec.Body.String(), "Which city, really?") {
		t.Fatalf("page lacks the question")
	}
	rec = doRequest(t, srv, http.MethodPost, "/questions/ui-2/answer",
		(&url.Values{"opkey": {opkeyForTest(t)}, "answer": {"Leeds"}, "build_id": {build}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("answer = %d", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet, "/conversations/"+conv, "", cookie)
	body := rec.Body.String()
	if !strings.Contains(body, "Leeds") {
		t.Error("page lacks the posted answer")
	}
	if strings.Contains(body, "/questions/ui-2/answer") {
		t.Error("answered question still shows an answer form")
	}
}

func TestAnswerConflict409(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	conv, build := setupBuild(t, srv, testTenant)
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"cf-s", "agent.started", map[string]any{}))
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"cf-q1", "agent.question", map[string]any{"question_id": "cf-1", "body": "First?"}))
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"cf-q2", "agent.question", map[string]any{"question_id": "cf-2", "body": "Second?"}))
	// cf-1 was superseded: answering it 409s and names the current one.
	rec := doRequest(t, srv, http.MethodPost, "/questions/cf-1/answer",
		(&url.Values{"opkey": {opkeyForTest(t)}, "answer": {"x"}, "build_id": {build}}).Encode(), cookie)
	if rec.Code != http.StatusConflict {
		t.Fatalf("stale answer = %d, want 409", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "no longer open") || !strings.Contains(body, "Second?") {
		t.Errorf("conflict page = %q", body)
	}
	if !strings.Contains(body, "/conversations/"+conv) {
		t.Error("conflict page lacks the back link")
	}
}

func TestAnswerAfterTerminalExpiry409(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	_, build := setupBuild(t, srv, testTenant)
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"tx-s", "agent.started", map[string]any{}))
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"tx-q", "agent.question", map[string]any{"question_id": "tx-1", "body": "Late?"}))
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"tx-done", "agent.completed", map[string]any{}))
	// The terminal expired the open question: answering it 409s with
	// the durable cause instead of resurrecting a dead build.
	rec := doRequest(t, srv, http.MethodPost, "/questions/tx-1/answer",
		(&url.Values{"opkey": {opkeyForTest(t)}, "answer": {"x"}, "build_id": {build}}).Encode(), cookie)
	if rec.Code != http.StatusConflict {
		t.Fatalf("post-terminal answer = %d, want 409", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "no longer open") {
		t.Errorf("conflict page = %q", body)
	}
}

func TestConversationCrossTenantUI(t *testing.T) {
	srv, _ := testServer(t)
	cookieA := login(t, srv, testToken)
	rec := doRequest(t, srv, http.MethodPost, "/conversations",
		(&url.Values{"opkey": {opkeyForTest(t)}, "title": {"A"}}).Encode(), cookieA)
	loc := rec.Header().Get("Location")
	cookieB := login(t, srv, testTokenB)
	rec = doRequest(t, srv, http.MethodGet, loc, "", cookieB)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-tenant view = %d, want 404", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodPost, loc+"/messages",
		(&url.Values{"opkey": {opkeyForTest(t)}, "body": {"hi"}}).Encode(), cookieB)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-tenant send = %d, want 404", rec.Code)
	}
}

func TestUntrustedTextEscaped(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	rec := doRequest(t, srv, http.MethodPost, "/conversations",
		(&url.Values{"opkey": {opkeyForTest(t)}, "title": {"X"}}).Encode(), cookie)
	loc := rec.Header().Get("Location")
	payload := `<script>alert(1)</script>`
	rec = doRequest(t, srv, http.MethodPost, loc+"/messages",
		(&url.Values{"opkey": {opkeyForTest(t)}, "body": {payload}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("send = %d", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet, loc, "", cookie)
	body := rec.Body.String()
	if strings.Contains(body, payload) {
		t.Error("raw script tag rendered")
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Error("escaped script tag missing")
	}
}

func TestAgentCanaryEndToEnd(t *testing.T) {
	srv := callbackServer(t)
	srv.ssePoll = 10 * time.Millisecond
	cookie := login(t, srv, testToken)
	conv, build := setupBuild(t, srv, testTenant)
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"can-s", "agent.started", map[string]any{}))
	canary := "sk-live-CANARY0000000000000002"
	code, _ := postCallback(t, srv, signedCallback(t, build, testTenant,
		"can-q", "agent.question", map[string]any{
			"question_id": "can-1", "body": "paste " + canary + " to proceed",
		}))
	if code != http.StatusOK {
		t.Fatalf("question callback = %d", code)
	}
	// Database persistence sanitized, prose preserved.
	owner := ownerPool(t)
	var stored string
	if err := owner.QueryRow(context.Background(),
		`SELECT body FROM studio.questions WHERE tenant_id = $1 AND id = $2`,
		testTenant, "can-1").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, canary) {
		t.Errorf("questions.body persists the canary")
	}
	if !strings.Contains(stored, "paste ") || !strings.Contains(stored, " to proceed") {
		t.Errorf("prose lost: %q", stored)
	}
	// Rendered page clean.
	rec := doRequest(t, srv, http.MethodGet, "/conversations/"+conv, "", cookie)
	if strings.Contains(rec.Body.String(), canary) {
		t.Error("rendered page leaks the canary")
	}
	// SSE bytes clean.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	srec, req := streamRequest(t, srv, conv, "", ctx)
	srv.Handler().ServeHTTP(srec, req)
	if strings.Contains(srec.Body.String(), canary) {
		t.Error("SSE stream leaks the canary")
	}
}

func TestAuthorizationButtons(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	conv, build := setupBuild(t, srv, testTenant)
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"ab-s", "agent.started", map[string]any{}))
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"ab-q", "agent.authorization_request",
		map[string]any{"question_id": "ab-1", "body": "Ship it?"}))
	rec := doRequest(t, srv, http.MethodGet, "/conversations/"+conv, "", cookie)
	body := rec.Body.String()
	if !strings.Contains(body, `value="approved"`) || !strings.Contains(body, `value="rejected"`) {
		t.Fatalf("authorization buttons missing")
	}
	rec = doRequest(t, srv, http.MethodPost, "/questions/ab-1/answer",
		(&url.Values{"opkey": {opkeyForTest(t)}, "answer": {"maybe"}, "build_id": {build}}).Encode(), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("vague authorization = %d, want 400", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodPost, "/questions/ab-1/answer",
		(&url.Values{"opkey": {opkeyForTest(t)}, "answer": {"approved"}, "build_id": {build}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Errorf("approve = %d, want 302", rec.Code)
	}
}
