package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PithomLabs/miniapp-studio/tools"
)

// streamRequest builds an SSE request carrying a session cookie, an
// optional Last-Event-ID, and a cancellable context (client disconnect).
func streamRequest(t *testing.T, srv *Server, conv, lastEventID string, ctx context.Context) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	cookie := login(t, srv, testToken)
	req := httptest.NewRequest(http.MethodGet, "/conversations/"+conv+"/events", nil).WithContext(ctx)
	req.AddCookie(cookie)
	if lastEventID != "" {
		req.Header.Set("Last-Event-ID", lastEventID)
	}
	return httptest.NewRecorder(), req
}

func seedMessages(t *testing.T, srv *Server, conv string, bodies ...string) {
	t.Helper()
	ctx := context.Background()
	for i, body := range bodies {
		key, err := newOpKey()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := srv.invokeToolTenant(ctx, testTenant, tools.ToolMessageSend,
			map[string]any{"conversation_id": conv, "body": body}, key); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
}

func TestSSEInitialBurstAndReplay(t *testing.T) {
	srv := callbackServer(t)
	srv.ssePoll = 10 * time.Millisecond
	out, err := srv.invokeToolTenant(context.Background(), testTenant,
		tools.ToolConversationCreate, map[string]any{"title": "s"}, "sse-conv-1")
	if err != nil {
		t.Fatal(err)
	}
	conv, _ := out["conversation_id"].(string)
	seedMessages(t, srv, conv, "one", "two", "three")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	rec, req := streamRequest(t, srv, conv, "", ctx)
	srv.Handler().ServeHTTP(rec, req)
	body := rec.Body.String()
	// Initial burst renders every persisted row in order with ids: three
	// user messages plus the build-ensured system note at seq 2.
	for _, want := range []string{"id: 1", "id: 2", "id: 3", "id: 4", `"body":"one"`, `"body":"three"`, "event: update"} {
		if !strings.Contains(body, want) {
			t.Errorf("burst lacks %q:\n%s", want, body)
		}
	}
	if i1, i2 := strings.Index(body, `"body":"one"`), strings.Index(body, `"body":"two"`); i1 > i2 || i1 < 0 {
		t.Errorf("items out of order:\n%s", body)
	}
	// A resumed cursor replays only what follows it (restart recovery is a
	// fresh handler invocation from the same cursor — exactly this).
	ctx2, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel2()
	rec2, req2 := streamRequest(t, srv, conv, "3", ctx2)
	srv.Handler().ServeHTTP(rec2, req2)
	replay := rec2.Body.String()
	if strings.Contains(replay, `"body":"two"`) || !strings.Contains(replay, `"body":"three"`) {
		t.Errorf("replay after 3 wrong:\n%s", replay)
	}
}

func TestSSELiveUpdateMixedTraffic(t *testing.T) {
	srv := callbackServer(t)
	srv.ssePoll = 10 * time.Millisecond
	conv, build := setupBuild(t, srv, testTenant)
	seedMessages(t, srv, conv, "user-words")
	code, _ := postCallback(t, srv, signedCallback(t, build, testTenant,
		"mix-e1", "agent.progress", map[string]any{"text": "agent-words"}))
	if code != http.StatusOK {
		t.Fatalf("callback = %d", code)
	}
	seedMessages(t, srv, conv, "more-user-words")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	rec, req := streamRequest(t, srv, conv, "", ctx)
	srv.Handler().ServeHTTP(rec, req)
	body := rec.Body.String()
	// System note, user message, agent event, user message — one cursor.
	marks := []string{"Build opened.", "user-words", "agent-words", "more-user-words"}
	last := -1
	for _, mark := range marks {
		at := strings.Index(body, mark)
		if at <= last {
			t.Fatalf("mark %q out of order in:\n%s", mark, body)
		}
		last = at
	}
	for _, id := range []string{"id: 1", "id: 2", "id: 3", "id: 4"} {
		if !strings.Contains(body, id) {
			t.Errorf("burst lacks %q:\n%s", id, body)
		}
	}
}

func TestSSELiveUpdate(t *testing.T) {
	srv := callbackServer(t)
	srv.ssePoll = 10 * time.Millisecond
	conv, _ := setupBuild(t, srv, testTenant)
	seedMessages(t, srv, conv, "before")
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()
	rec, req := streamRequest(t, srv, conv, "", ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.Handler().ServeHTTP(rec, req)
	}()
	// A row written mid-stream must appear without reconnecting.
	time.Sleep(80 * time.Millisecond)
	seedMessages(t, srv, conv, "during")
	<-done
	body := rec.Body.String()
	if !strings.Contains(body, `"body":"before"`) || !strings.Contains(body, `"body":"during"`) {
		t.Errorf("live stream lacks rows:\n%s", body)
	}
	if ib, id := strings.Index(body, `"body":"before"`), strings.Index(body, `"body":"during"`); ib > id || ib < 0 {
		t.Errorf("live rows out of order:\n%s", body)
	}
}

func TestSSEAuthAndErrors(t *testing.T) {
	srv := callbackServer(t)
	conv, _ := setupBuild(t, srv, testTenant)
	// No session → login redirect (GET page behavior).
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/conversations/"+conv+"/events", nil))
	if rec.Code != http.StatusFound {
		t.Errorf("unauthenticated = %d, want 302 to login", rec.Code)
	}
	// Cross-tenant conversation → 404.
	cookieB := login(t, srv, testTokenB)
	req := httptest.NewRequest(http.MethodGet, "/conversations/"+conv+"/events", nil)
	req.AddCookie(cookieB)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-tenant = %d, want 404", rec.Code)
	}
	// Malformed cursor fails closed.
	cookie := login(t, srv, testToken)
	req = httptest.NewRequest(http.MethodGet, "/conversations/"+conv+"/events?after=nope", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad cursor = %d, want 400", rec.Code)
	}
}

func TestShapeStreamItem(t *testing.T) {
	data, seq, err := shapeStreamItem(map[string]any{
		"seq": float64(7), "kind": "event", "ref_id": "e1", "build_id": "b1",
		"role": nil, "body": nil, "type": "agent.progress",
		"payload": []byte(`{"text":"hi"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq != 7 {
		t.Errorf("seq = %d", seq)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	payload, _ := decoded["payload"].(map[string]any)
	if payload["text"] != "hi" {
		t.Errorf("payload embedded as %v, want parsed JSON", decoded["payload"])
	}
	if _, _, err := shapeStreamItem(map[string]any{
		"seq": float64(1), "payload": []byte("{oops"),
	}); err == nil {
		t.Error("corrupt payload accepted, want error")
	}
}
