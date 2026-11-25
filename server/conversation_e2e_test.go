package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/gateway"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/testdb"
)

func ownerPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := store.NewPool(context.Background(), testdb.RunDSN(t), 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// TestPhaseAEndToEnd executes the directive's 15-step functional proof in
// order against the fake-signed callback path: one conversation from
// creation through completed build, observed by the browser throughout.
func TestPhaseAEndToEnd(t *testing.T) {
	srv := callbackServer(t)
	srv.ssePoll = 10 * time.Millisecond
	cookie := login(t, srv, testToken)
	owner := ownerPool(t)
	ctx := context.Background()

	// 1. The user creates a conversation.
	rec := doRequest(t, srv, http.MethodPost, "/conversations",
		(&url.Values{"opkey": {opkeyForTest(t)}, "title": {"Venue booking"}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("1 create = %d", rec.Code)
	}
	loc := rec.Header().Get("Location")

	// 2-3. The user submits a requirement; Studio persists it and opens a build.
	rec = doRequest(t, srv, http.MethodPost, loc+"/messages",
		(&url.Values{"opkey": {opkeyForTest(t)}, "body": {"Book the venue"}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("2 send = %d", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet, loc, "", cookie)
	if !strings.Contains(rec.Body.String(), "assigned") {
		t.Fatalf("3 page lacks the assigned build")
	}
	var buildID string
	if err := owner.QueryRow(ctx,
		`SELECT id FROM studio.builds WHERE tenant_id = $1`, testTenant).Scan(&buildID); err != nil {
		t.Fatalf("3 build row: %v", err)
	}

	postEvent := func(msgID, typ string, payload map[string]any) map[string]any {
		t.Helper()
		code, ack := postCallback(t, srv, signedCallback(t, buildID, testTenant, msgID, typ, payload))
		if code != http.StatusOK {
			t.Fatalf("callback %s = %d", typ, code)
		}
		return ack
	}

	// 4-5. The fake gateway sends a signed progress event; Studio acks it.
	ack := postEvent("e2e-p1", "agent.progress", map[string]any{"text": "Reading requirements"})
	if ack["ack"] != true || ack["duplicate"] == true {
		t.Fatalf("5 ack = %v", ack)
	}

	// 6. The browser receives the event through SSE.
	sctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	srec, sreq := streamRequest(t, srv, strings.TrimPrefix(loc, "/conversations/"), "", sctx)
	srv.Handler().ServeHTTP(srec, sreq)
	if !strings.Contains(srec.Body.String(), "Reading requirements") {
		t.Fatalf("6 SSE lacks the progress event")
	}

	// Started precedes the question: the state machine parks yields only on
	// running builds, so the fake starts the agent first. (Progress above
	// absorbed as history while assigned — arrival-ordered transitions.)
	postEvent("e2e-s", "agent.started", map[string]any{"agent_session_id": "e2e-sess"})

	// 7-8. The fake gateway asks a signed question; Studio parks the turn.
	ack = postEvent("e2e-q", "agent.question",
		map[string]any{"question_id": "e2e-q1", "body": "Which city?"})
	if ack["build_status"] != "waiting_user" {
		t.Fatalf("8 park = %v", ack)
	}

	// 9. The browser displays the question in the original conversation.
	rec = doRequest(t, srv, http.MethodGet, loc, "", cookie)
	if !strings.Contains(rec.Body.String(), "Which city?") {
		t.Fatalf("9 page lacks the question")
	}

	// 10-11. The user answers; Studio validates, persists, and resumes.
	rec = doRequest(t, srv, http.MethodPost, "/questions/e2e-q1/answer",
		(&url.Values{"opkey": {opkeyForTest(t)}, "answer": {"Leeds"}, "build_id": {buildID}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("10 answer = %d", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet, loc, "", cookie)
	if body := rec.Body.String(); !strings.Contains(body, "Leeds") || !strings.Contains(body, "running") {
		t.Fatalf("11 page lacks answer/running state")
	}

	// 12. The continuation command is logged for the gateway.
	var continueKind string
	if err := owner.QueryRow(ctx,
		`SELECT kind FROM studio.commands WHERE tenant_id = $1 AND build_id = $2 AND kind = 'continue'`,
		testTenant, buildID).Scan(&continueKind); err != nil {
		t.Fatalf("12 continue command: %v", err)
	}

	// 13-14. The fake gateway completes with artifact metadata; Studio persists.
	ack = postEvent("e2e-done", "agent.completed", map[string]any{
		"artifacts": []any{map[string]any{"kind": "spec", "digest": "e2e-digest-1"}},
	})
	if ack["build_status"] != "succeeded" {
		t.Fatalf("14 completion = %v", ack)
	}

	// 15. The browser shows the completed history and the artifact record.
	rec = doRequest(t, srv, http.MethodGet, loc, "", cookie)
	body := rec.Body.String()
	if !strings.Contains(body, "succeeded") || !strings.Contains(body, "agent.completed") {
		t.Fatalf("15 page lacks completion")
	}
	var artifacts int
	if err := owner.QueryRow(ctx,
		`SELECT count(*) FROM studio.artifacts WHERE tenant_id = $1 AND build_id = $2`,
		testTenant, buildID).Scan(&artifacts); err != nil || artifacts != 1 {
		t.Fatalf("15 artifacts = %d, %v", artifacts, err)
	}
}

// TestPhaseARestartRecovery proves the ordered shutdown → cold boot
// lifecycle against one persistent database: sessions, parked questions,
// SSE replay, and duplicate handling all survive, because every one of
// them is database rows, not process memory.
func TestPhaseARestartRecovery(t *testing.T) {
	srv1, _ := testServer(t)
	srv1.gwKeys = gateway.StaticProvider{}
	cookie := login(t, srv1, testToken)
	conv, build := setupBuild(t, srv1, testTenant)
	postCallback(t, srv1, signedCallback(t, build, testTenant,
		"rs-s", "agent.started", map[string]any{}))
	postCallback(t, srv1, signedCallback(t, build, testTenant,
		"rs-q", "agent.question", map[string]any{"question_id": "rs-1", "body": "Still there?"}))

	// Ordered shutdown first: the error is surfaced, not masked, and
	// completion is proven observably (the pool refuses new work) before
	// the replacement boots. The cleanup re-close then proves idempotency.
	if err := srv1.Close(); err != nil {
		t.Fatalf("srv1 shutdown: %v", err)
	}
	if conn, err := srv1.pool.Acquire(context.Background()); err == nil {
		conn.Release()
		t.Fatal("srv1 pool still usable after Close")
	}
	if err := srv1.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}

	// Fresh server, same run database, no truncate: the restart.
	srv2, _ := bootServerOn(t, testdb.RunDSN(t), nil, nil, false, nil)
	srv2.gwKeys = gateway.StaticProvider{}
	srv2.ssePoll = 10 * time.Millisecond

	// The pre-restart session cookie still authenticates the fresh process.
	rec := doRequest(t, srv2, http.MethodGet, "/conversations/"+conv, "", cookie)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Still there?") {
		t.Fatalf("restarted view = %d", rec.Code)
	}
	// The parked answer still applies after the restart.
	rec = doRequest(t, srv2, http.MethodPost, "/questions/rs-1/answer",
		(&url.Values{"opkey": {opkeyForTest(t)}, "answer": {"yes"}, "build_id": {build}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("post-restart answer = %d", rec.Code)
	}
	// SSE replays the full persisted history from the head cursor.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	srec, req := streamRequest(t, srv2, conv, "", ctx)
	srv2.Handler().ServeHTTP(srec, req)
	replay := srec.Body.String()
	for _, want := range []string{"Still there?", "yes", "Build opened."} {
		if !strings.Contains(replay, want) {
			t.Errorf("replay lacks %q", want)
		}
	}
	// Duplicate callbacks acknowledge without reapplying after restart.
	code, ack := postCallback(t, srv2, signedCallback(t, build, testTenant,
		"rs-q", "agent.question", map[string]any{"question_id": "rs-1", "body": "Still there?"}))
	if code != http.StatusOK || ack["duplicate"] != true {
		t.Errorf("post-restart redelivery = %d %v, want duplicate ack", code, ack)
	}
}
