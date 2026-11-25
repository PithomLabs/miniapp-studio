package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/PithomLabs/miniapp-studio/gateway"
	"github.com/PithomLabs/miniapp-studio/tools"
)

func TestConsoleRequiresLogin(t *testing.T) {
	srv, _ := testServer(t)
	rec := doRequest(t, srv, http.MethodGet, "/console", "")
	if rec.Code != http.StatusFound {
		t.Fatalf("anonymous console = %d, want 302 to login", rec.Code)
	}
	if loc := rec.Header().Get("Location"); !strings.HasPrefix(loc, "/login") {
		t.Errorf("redirect = %q, want /login", loc)
	}
	rec = doRequest(t, srv, http.MethodGet, "/static/focus/console.js", "")
	if rec.Code != http.StatusFound {
		t.Errorf("anonymous console.js = %d, want 302", rec.Code)
	}
}

func TestConsoleGalleryEmptyAndCreate(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	rec := doRequest(t, srv, http.MethodGet, "/console?scene=gallery", "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("gallery = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"No conversations yet", "New conversation", `id="spine"`, `id="pulse"`, `id="lineForm"`, "console.js", `data-persona="ops"`} {
		if !strings.Contains(body, want) {
			t.Errorf("gallery lacks %q", want)
		}
	}
	if strings.Contains(body, `class="sph" data-s="proven"`) || strings.Contains(body, `class="sph" data-s="active"`) {
		t.Errorf("empty gallery shows non-gray pills")
	}
	rec = doRequest(t, srv, http.MethodPost, "/console/conversations",
		(&url.Values{"opkey": {opkeyForTest(t)}, "title": {"Desk"}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("create = %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/console?scene=run&conversation=") {
		t.Fatalf("redirect = %q, want focused run scene", loc)
	}
	rec = doRequest(t, srv, http.MethodGet, "/console?scene=gallery", "", cookie)
	if !strings.Contains(rec.Body.String(), "Desk") {
		t.Errorf("gallery lacks the new conversation")
	}
}

func TestConsoleLineOpensBuildAndSpine(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	conv := consoleConversation(t, srv, cookie, "Shop")
	rec := doRequest(t, srv, http.MethodPost, "/console/line",
		(&url.Values{"opkey": {opkeyForTest(t)}, "conversation_id": {conv}, "body": {"Track my orders"}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("line post = %d", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet, "/console?scene=run&conversation="+conv, "", cookie)
	body := rec.Body.String()
	for _, want := range []string{"Track my orders", "Build opened.", `class="sph" data-s="proven"`, "Describe"} {
		if !strings.Contains(body, want) {
			t.Errorf("run scene lacks %q", want)
		}
	}
	// Guardrail: Run never goes green from builder state alone —
	// exactly one proven pill (Describe) on a fresh assigned build.
	if n := strings.Count(body, `class="sph" data-s="proven"`); n != 1 {
		t.Errorf("proven pills = %d, want 1 (Describe only)", n)
	}
	if !strings.Contains(body, `class="sph" data-s="active"`) {
		t.Errorf("no active pill; Run should be active on an assigned build")
	}
}

func TestConsoleTenantScenesNeedNoFocus(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	for scene, want := range map[string]string{
		"gallery": "New conversation", "evidence": "Gate ledger lands in Phase D",
		"settings": "Stop everything",
	} {
		rec := doRequest(t, srv, http.MethodGet, "/console?scene="+scene, "", cookie)
		if rec.Code != http.StatusOK {
			t.Errorf("%s unfocused = %d, want 200", scene, rec.Code)
			continue
		}
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("%s lacks %q", scene, want)
		}
	}
}

func TestConsoleAllScenesRenderFocused(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	conv := consoleConversation(t, srv, cookie, "Shop")
	// Every scene in the dispatch map must render focused. A wrong
	// template filename 500s instead — this test pins the map to the
	// files on disk.
	for scene := range focusScenes {
		rec := doRequest(t, srv, http.MethodGet,
			"/console?scene="+scene+"&conversation="+conv, "", cookie)
		if rec.Code != http.StatusOK {
			t.Errorf("scene %s focused = %d, want 200", scene, rec.Code)
		}
	}
}

func TestConsoleSceneRouting(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	rec := doRequest(t, srv, http.MethodGet, "/console?scene=nope", "", cookie)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown scene = %d, want 404", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet, "/console?scene=run", "", cookie)
	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "scene=gallery") {
		t.Errorf("sceneless run = %d %q, want redirect to gallery", rec.Code, rec.Header().Get("Location"))
	}
	rec = doRequest(t, srv, http.MethodGet, "/console?scene=run&conversation=missing", "", cookie)
	if rec.Code != http.StatusNotFound {
		t.Errorf("missing conversation = %d, want 404", rec.Code)
	}
}

func TestConsoleCrossTenantIsolation(t *testing.T) {
	srv, _ := testServer(t)
	cookieA := login(t, srv, testToken)
	conv := consoleConversation(t, srv, cookieA, "Private")
	cookieB := login(t, srv, testTokenB)
	rec := doRequest(t, srv, http.MethodGet, "/console?scene=run&conversation="+conv, "", cookieB)
	if rec.Code != http.StatusNotFound {
		t.Errorf("cross-tenant console = %d, want 404", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet, "/console?scene=gallery", "", cookieB)
	if strings.Contains(rec.Body.String(), "Private") {
		t.Errorf("tenant B gallery leaks tenant A conversation")
	}
}

func TestConsoleStaticJS(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	rec := doRequest(t, srv, http.MethodGet, "/static/focus/console.js", "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("console.js = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("content-type = %q", ct)
	}
	for _, want := range []string{"EventSource", "dataset.persona", "Last-Event-ID", "replyQID", "killForm", "HOLD_MS", "data-reply-qid"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("console.js lacks %q", want)
		}
	}
}

func TestConsoleLineReplyAnswersQuestion(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	conv, build := setupBuild(t, srv, testTenant)
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"rp-s", "agent.started", map[string]any{}))
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"rp-q", "agent.question", map[string]any{"question_id": "rp-1", "body": "Which region?"}))
	rec := doRequest(t, srv, http.MethodPost, "/console/line",
		(&url.Values{"opkey": {opkeyForTest(t)}, "conversation_id": {conv},
			"question_id": {"rp-1"}, "body": {"eu-west"}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("line reply = %d", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet, "/console?scene=run&conversation="+conv, "", cookie)
	body := rec.Body.String()
	if !strings.Contains(body, "eu-west") {
		t.Errorf("run scene lacks the posted answer")
	}
	if strings.Contains(body, "Decisions") {
		t.Errorf("clarification answer created a decision entry")
	}
}

func TestConsoleLineReplyStale409(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	conv, build := setupBuild(t, srv, testTenant)
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"rs-s", "agent.started", map[string]any{}))
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"rs-q1", "agent.question", map[string]any{"question_id": "rs-1", "body": "First?"}))
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"rs-q2", "agent.question", map[string]any{"question_id": "rs-2", "body": "Second?"}))
	rec := doRequest(t, srv, http.MethodPost, "/console/line",
		(&url.Values{"opkey": {opkeyForTest(t)}, "conversation_id": {conv},
			"question_id": {"rs-1"}, "body": {"x"}}).Encode(), cookie)
	if rec.Code != http.StatusConflict {
		t.Fatalf("stale reply = %d, want 409", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "no longer open") || !strings.Contains(body, "rs-1") {
		t.Errorf("conflict page lacks the asked question:\n%s", body)
	}
	if !strings.Contains(body, "Second?") {
		t.Errorf("conflict page lacks the current open question")
	}
}

func TestConsoleDecideApprovalRoundTrip(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	conv, build := setupBuild(t, srv, testTenant)
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"da-s", "agent.started", map[string]any{}))
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"da-q", "agent.authorization_request", map[string]any{"question_id": "da-1", "body": "Ship it?"}))
	rec := doRequest(t, srv, http.MethodGet, "/console?scene=run&conversation="+conv, "", cookie)
	if body := rec.Body.String(); !strings.Contains(body, "Approval required") || !strings.Contains(body, "Ship it?") {
		t.Fatalf("pulse lacks the approval card")
	}
	if !strings.Contains(rec.Body.String(), "agent only receives answers submitted to its open question") {
		t.Errorf("line lacks the F8 mid-run hint on a waiting build")
	}
	rec = doRequest(t, srv, http.MethodPost, "/console/decide",
		(&url.Values{"opkey": {opkeyForTest(t)}, "question_id": {"da-1"}, "decision": {"approved"},
			"build_id": {build}, "conversation_id": {conv}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("decide = %d", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet, "/console?scene=run&conversation="+conv, "", cookie)
	body := rec.Body.String()
	if !strings.Contains(body, "Decisions") || !strings.Contains(body, "approved") || !strings.Contains(body, "da-1") {
		t.Errorf("run scene lacks the recorded decision:\n%s", body)
	}
	if strings.Contains(body, "Approval required") {
		t.Errorf("decided approval still shows as pending")
	}
}

func TestConsoleDecideRefusesClarification(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	conv, build := setupBuild(t, srv, testTenant)
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"dc-s", "agent.started", map[string]any{}))
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"dc-q", "agent.question", map[string]any{"question_id": "dc-1", "body": "Which?"}))
	rec := doRequest(t, srv, http.MethodPost, "/console/decide",
		(&url.Values{"opkey": {opkeyForTest(t)}, "question_id": {"dc-1"}, "decision": {"approved"},
			"build_id": {build}, "conversation_id": {conv}}).Encode(), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("decide on clarification = %d, want 400", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodPost, "/console/decide",
		(&url.Values{"opkey": {opkeyForTest(t)}, "question_id": {"dc-1"}, "decision": {"maybe"},
			"build_id": {build}, "conversation_id": {conv}}).Encode(), cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("vague decision = %d, want 400", rec.Code)
	}
}

func TestConsoleKillStopsTenantBuilds(t *testing.T) {
	srv, _ := testServer(t)
	cookieA := login(t, srv, testToken)
	convA1 := consoleConversation(t, srv, cookieA, "A1")
	doRequest(t, srv, http.MethodPost, "/console/line",
		(&url.Values{"opkey": {opkeyForTest(t)}, "conversation_id": {convA1}, "body": {"hi"}}).Encode(), cookieA)
	convA2 := consoleConversation(t, srv, cookieA, "A2")
	doRequest(t, srv, http.MethodPost, "/console/line",
		(&url.Values{"opkey": {opkeyForTest(t)}, "conversation_id": {convA2}, "body": {"hi"}}).Encode(), cookieA)
	cookieB := login(t, srv, testTokenB)
	convB := consoleConversation(t, srv, cookieB, "B1")
	doRequest(t, srv, http.MethodPost, "/console/line",
		(&url.Values{"opkey": {opkeyForTest(t)}, "conversation_id": {convB}, "body": {"hi"}}).Encode(), cookieB)

	rec := doRequest(t, srv, http.MethodPost, "/console/kill",
		(&url.Values{"opkey": {opkeyForTest(t)}}).Encode(), cookieA)
	if rec.Code != http.StatusFound {
		t.Fatalf("kill = %d", rec.Code)
	}
	loc, _ := url.Parse(rec.Header().Get("Location"))
	if notice := loc.Query().Get("notice"); !strings.Contains(notice, "Stopped 2 builds") {
		t.Errorf("notice = %q, want Stopped 2 builds", notice)
	}
	// The unfocused gallery landing must render the notice, not drop it.
	rec = doRequest(t, srv, http.MethodGet, loc.RequestURI(), "", cookieA)
	if !strings.Contains(rec.Body.String(), "Stopped 2 builds") {
		t.Errorf("gallery landing lacks the kill notice")
	}
	for _, conv := range []string{convA1, convA2} {
		rec = doRequest(t, srv, http.MethodGet, "/console?scene=run&conversation="+conv, "", cookieA)
		if !strings.Contains(rec.Body.String(), "cancelled") {
			t.Errorf("tenant A build in %s not cancelled", conv)
		}
	}
	rec = doRequest(t, srv, http.MethodGet, "/console?scene=run&conversation="+convB, "", cookieB)
	if strings.Contains(rec.Body.String(), "cancelled") {
		t.Errorf("tenant B build was cancelled by tenant A kill")
	}
	// Repeat converges to zero with no error: natural idempotency.
	rec = doRequest(t, srv, http.MethodPost, "/console/kill",
		(&url.Values{"opkey": {opkeyForTest(t)}}).Encode(), cookieA)
	loc, _ = url.Parse(rec.Header().Get("Location"))
	if notice := loc.Query().Get("notice"); !strings.Contains(notice, "Stopped 0 builds") {
		t.Errorf("repeat notice = %q, want Stopped 0 builds", notice)
	}
}

func TestConsoleDoubleAnswerOneWins409(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	conv, build := setupBuild(t, srv, testTenant)
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"da2-s", "agent.started", map[string]any{}))
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"da2-q", "agent.question", map[string]any{"question_id": "da2-1", "body": "Pick?"}))
	post := func(answer string) int {
		rec := doRequest(t, srv, http.MethodPost, "/console/line",
			(&url.Values{"opkey": {opkeyForTest(t)}, "conversation_id": {conv},
				"question_id": {"da2-1"}, "body": {answer}}).Encode(), cookie)
		return rec.Code
	}
	// Sequential double-submit first: deterministic 302 then 409.
	if code := post("first"); code != http.StatusFound {
		t.Fatalf("first answer = %d, want 302", code)
	}
	if code := post("second"); code != http.StatusConflict {
		t.Fatalf("second answer = %d, want 409", code)
	}
}

func TestConsoleKillVsAnswerRace(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	conv, build := setupBuild(t, srv, testTenant)
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"ka-s", "agent.started", map[string]any{}))
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"ka-q", "agent.question", map[string]any{"question_id": "ka-1", "body": "Race?"}))
	var wg sync.WaitGroup
	codes := make([]int, 2)
	answerForm := (&url.Values{"opkey": {opkeyForTest(t)}, "conversation_id": {conv},
		"question_id": {"ka-1"}, "body": {"x"}}).Encode()
	killForm := (&url.Values{"opkey": {opkeyForTest(t)}}).Encode()
	wg.Add(2)
	go func() {
		defer wg.Done()
		rec := doRequest(t, srv, http.MethodPost, "/console/line", answerForm, cookie)
		codes[0] = rec.Code
	}()
	go func() {
		defer wg.Done()
		rec := doRequest(t, srv, http.MethodPost, "/console/kill", killForm, cookie)
		codes[1] = rec.Code
	}()
	wg.Wait()
	// Either the answer wins (302, then kill cancels the running
	// build) or the kill wins (answer 409s). No 500s, no wedges.
	if codes[0] != http.StatusFound && codes[0] != http.StatusConflict {
		t.Errorf("answer race code = %d", codes[0])
	}
	if codes[1] != http.StatusFound {
		t.Errorf("kill race code = %d, want 302", codes[1])
	}
	rec := doRequest(t, srv, http.MethodGet, "/console?scene=run&conversation="+conv, "", cookie)
	// Whoever wins, the kill cancels the build last: the only
	// coherent end state is cancelled.
	if body := rec.Body.String(); !strings.Contains(body, "cancelled") {
		t.Errorf("end state incoherent:\n%s", body)
	}
	_ = build
}

func TestConsoleKillRequiresSessionAndOrigin(t *testing.T) {
	srv, _ := testServer(t)
	rec := doRequest(t, srv, http.MethodPost, "/console/kill",
		(&url.Values{"opkey": {opkeyForTest(t)}}).Encode())
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous kill = %d, want 401", rec.Code)
	}
	cookie := login(t, srv, testToken)
	req := httptest.NewRequest(http.MethodPost, "/console/kill",
		strings.NewReader((&url.Values{"opkey": {opkeyForTest(t)}}).Encode()))
	req.AddCookie(cookie)
	// No Origin header: forged/cross-site posts fail closed.
	rec2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec2, req)
	if rec2.Code != http.StatusForbidden {
		t.Errorf("kill without Origin = %d, want 403", rec2.Code)
	}
}

func TestConsoleAgentSecretsNeverRender(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	conv, build := setupBuild(t, srv, testTenant)
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"sc-s", "agent.started", map[string]any{}))
	canary := "sk-test-AAAAAAAAAAAAAAAAAAAA"
	postCallback(t, srv, signedCallback(t, build, testTenant,
		"sc-q", "agent.question", map[string]any{"question_id": "sc-1", "body": "Use key " + canary + " now"}))
	rec := doRequest(t, srv, http.MethodGet, "/console?scene=run&conversation="+conv, "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("run scene = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), canary) {
		t.Errorf("console renders an agent-supplied secret span verbatim")
	}
	if !strings.Contains(rec.Body.String(), "sc-1") {
		t.Errorf("scrubbed question itself went missing")
	}
}

func TestConsoleA11yWiring(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	rec := doRequest(t, srv, http.MethodGet, "/console?scene=gallery", "", cookie)
	body := rec.Body.String()
	for _, want := range []string{`aria-live="polite"`, `aria-label="Pulse"`, `aria-label="Persona"`,
		`aria-label="Hold to stop every non-terminal build in this tenant"`, `aria-label="Send"`,
		`aria-label="Describe what you want"`, "prefers-reduced-motion", `data-persona="ops"`} {
		if !strings.Contains(body, want) {
			t.Errorf("console shell lacks %q", want)
		}
	}
	rec = doRequest(t, srv, http.MethodGet, "/static/focus/console.js", "", cookie)
	js := rec.Body.String()
	for _, want := range []string{"lastFocused", "'Escape'", "keydown", "keyup", "localStorage", "HOLD_MS"} {
		if !strings.Contains(js, want) {
			t.Errorf("console.js lacks %q", want)
		}
	}
}

func TestConsolePulseShowsDispatchOutcomes(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	cookie := login(t, w.srv, testToken)
	conv, build := w.openBuild(t, testTenant, "dsp-ui")
	w.snd.PollOnce(ctx) // assign acks → dispatch receipt streams
	w.adapter.Fail["Continue"] = errors.New("boom")
	contID := w.insertCommand(t, testTenant, build, gateway.CmdContinue, map[string]any{
		"question_id": "q", "kind": "clarification", "answer": "a", "turn": float64(2),
	}, "queued", 0)
	for i := 0; i < testSenderCfg().MaxAttempts; i++ {
		w.resetDue(t, testTenant, contID)
		w.snd.PollOnce(ctx)
	}
	rec := doRequest(t, w.srv, http.MethodGet, "/console?scene=run&conversation="+conv, "", cookie)
	body := rec.Body.String()
	if !strings.Contains(body, "dispatch assign acked") {
		t.Errorf("pulse lacks the ack receipt")
	}
	if !strings.Contains(body, "dispatch continue failed") || !strings.Contains(body, "exhausted 5 attempts") {
		t.Errorf("pulse lacks the exhaustion alert")
	}
	if !strings.Contains(body, "acknowledged") || !strings.Contains(body, "exhausted") {
		t.Errorf("run dispatch table lacks acked/exhausted states")
	}
}

func TestConsolePlaceholderScenes(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	conv := consoleConversation(t, srv, cookie, "Shop")
	for scene, phase := range map[string]string{"spec": "Phase D", "validate": "Phase D", "connect": "Phase D", "deploy": "Phase E"} {
		rec := doRequest(t, srv, http.MethodGet, "/console?scene="+scene+"&conversation="+conv, "", cookie)
		if rec.Code != http.StatusOK {
			t.Errorf("%s = %d", scene, rec.Code)
			continue
		}
		body := rec.Body.String()
		if !strings.Contains(body, "lands in "+phase) || !strings.Contains(body, "Not built yet") {
			t.Errorf("%s lacks the honest Phase note", scene)
		}
		// Only Describe may be active (focused conversation, no builds
		// yet); the future phase itself must stay gray.
		if strings.Contains(body, `class="sph" data-s="proven"`) {
			t.Errorf("%s shows a proven pill", scene)
		}
		if n := strings.Count(body, `class="sph" data-s="active"`); n != 1 {
			t.Errorf("%s shows %d active pills, want 1 (Describe)", scene, n)
		}
	}
}

func TestConsoleEvidenceReceipts(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	conv := consoleConversation(t, srv, cookie, "Shop")
	rec := doRequest(t, srv, http.MethodGet, "/console?scene=evidence&conversation="+conv, "", cookie)
	body := rec.Body.String()
	if !strings.Contains(body, "conversation.create") || !strings.Contains(body, "succeeded") {
		t.Errorf("evidence lacks live receipts:\n%s", body)
	}
	if !strings.Contains(body, "Gate ledger lands in Phase D") {
		t.Errorf("evidence lacks the gate-ledger note")
	}
}

func TestConsoleEvidenceHidesStreamPolls(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	conv := consoleConversation(t, srv, cookie, "Shop")
	// Per-second SSE polls audit as conversation.stream invocations;
	// the receipts ledger hides that transport noise (display only).
	for i := 0; i < 3; i++ {
		key, err := newOpKey()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := srv.invokeToolTenant(context.Background(), testTenant, tools.ToolConversationStream,
			map[string]any{"conversation_id": conv, "after_seq": 0}, key); err != nil {
			t.Fatal(err)
		}
	}
	rec := doRequest(t, srv, http.MethodGet, "/console?scene=evidence&conversation="+conv, "", cookie)
	body := rec.Body.String()
	if strings.Contains(body, "conversation.stream") {
		t.Errorf("evidence shows stream poll records")
	}
	if !strings.Contains(body, "conversation.create") {
		t.Errorf("evidence lost real receipts while filtering polls")
	}
	if !strings.Contains(body, "hidden as transport noise") {
		t.Errorf("evidence lacks the filter disclosure")
	}
}

func TestBuildSpine(t *testing.T) {
	b := func(status string) []any { return []any{map[string]any{"status": status}} }
	cases := []struct {
		name       string
		conv       string
		builds     []any
		wantStates map[string]string
	}{
		{"no focus", "", nil, map[string]string{"Describe": "idle", "Run": "idle", "Evidence": "idle"}},
		{"no builds", "c", nil, map[string]string{"Describe": "active", "Run": "idle"}},
		{"assigned", "c", b("assigned"), map[string]string{"Describe": "proven", "Run": "active"}},
		{"running", "c", b("running"), map[string]string{"Describe": "proven", "Run": "active"}},
		{"waiting", "c", b("waiting_user"), map[string]string{"Run": "active"}},
		{"failed", "c", b("failed"), map[string]string{"Run": "refused"}},
		{"cancelled", "c", b("cancelled"), map[string]string{"Run": "refused"}},
		{"succeeded", "c", b("succeeded"), map[string]string{"Run": "idle", "Describe": "proven"}},
	}
	for _, tc := range cases {
		pills := buildSpine(tc.conv, tc.builds)
		if len(pills) != 7 {
			t.Fatalf("%s: pills = %d, want 7", tc.name, len(pills))
		}
		got := map[string]string{}
		for _, p := range pills {
			got[p.Label] = p.State
			if p.Href == "" || p.Title == "" {
				t.Errorf("%s: pill %q lacks href/title", tc.name, p.Label)
			}
		}
		for label, want := range tc.wantStates {
			if got[label] != want {
				t.Errorf("%s: %s = %q, want %q", tc.name, label, got[label], want)
			}
		}
		// Guardrail: nothing in C6 may mark product Run proven.
		if got["Run"] == "proven" {
			t.Errorf("%s: Run proven from builder state", tc.name)
		}
		if got["Spec"] != "idle" || got["Validate"] != "idle" || got["Connect"] != "idle" || got["Deploy"] != "idle" || got["Evidence"] != "idle" {
			t.Errorf("%s: future phase not gray", tc.name)
		}
	}
}

// consoleConversation opens one conversation through the console route
// and returns its id.
func consoleConversation(t *testing.T, srv *Server, cookie *http.Cookie, title string) string {
	t.Helper()
	rec := doRequest(t, srv, http.MethodPost, "/console/conversations",
		(&url.Values{"opkey": {opkeyForTest(t)}, "title": {title}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("create conversation = %d", rec.Code)
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	return loc.Query().Get("conversation")
}
