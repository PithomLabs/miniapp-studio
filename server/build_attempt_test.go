package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// attemptBuild opens one conversation + build through the real tools.
func attemptBuild(t *testing.T, srv *Server, tenant, key string) (conv, build string) {
	t.Helper()
	ctx := context.Background()
	out, err := srv.RunTool(ctx, tenant, "conversation.create",
		map[string]any{"title": "B0 task"}, key+"-c")
	if err != nil {
		t.Fatal(err)
	}
	conv, _ = out["conversation_id"].(string)
	out, err = srv.RunTool(ctx, tenant, "build.create",
		map[string]any{"conversation_id": conv}, key+"-b")
	if err != nil {
		t.Fatal(err)
	}
	build, _ = out["build_id"].(string)
	return conv, build
}

func attemptInput() AttemptInput {
	return AttemptInput{
		Branch: "try/login", StatusDigest: "sha256:abc",
		Commits:      []string{"abcdef1234567890"},
		DirtyNote:    "uncommitted prompt tweak",
		TestsPassed:  false,
		TestsSummary: "2 fail on redirect",
		Report:       "tried X, redirect still drops next",
		Verdict:      "reject",
		VerdictNote:  "needs another attempt",
	}
}

// TestRecordAttemptRoundTrip pins the B0 record: attempt 1 lands as
// an event + stream row, visible through the public stream.
func TestRecordAttemptRoundTrip(t *testing.T) {
	srv, _ := testServer(t)
	conv, build := attemptBuild(t, srv, testTenant, "ar")
	out, code, err := srv.RecordAttempt(context.Background(), testTenant, testBinding,
		build, attemptInput(), opkeyForTest(t))
	if err != nil || code != 0 {
		t.Fatalf("record = %v/%d/%v", out, code, err)
	}
	if out.N != 1 || out.ParentN != 0 || out.Replay {
		t.Fatalf("outcome = %+v", out)
	}
	page, err := srv.RunTool(context.Background(), testTenant, "conversation.stream",
		map[string]any{"conversation_id": conv}, "ar-s")
	if err != nil {
		t.Fatal(err)
	}
	items, _ := page["items"].([]any)
	found := false
	for _, it := range items {
		m, _ := it.(map[string]any)
		pm, _ := m["payload"].(map[string]any)
		text, _ := pm["text"].(string)
		if strings.Contains(text, "Attempt 1 rejected") {
			found = true
		}
	}
	if !found {
		t.Fatalf("stream lacks the attempt card: %v", page)
	}
}

// TestRecordAttemptLinksParent proves the retry chain: attempt 2
// names attempt 1 as its parent.
func TestRecordAttemptLinksParent(t *testing.T) {
	srv, _ := testServer(t)
	_, build := attemptBuild(t, srv, testTenant, "al")
	ctx := context.Background()
	if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build,
		attemptInput(), opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	second := attemptInput()
	second.Verdict = "accept"
	second.TestsPassed = true
	out, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build,
		second, opkeyForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	if out.N != 2 || out.ParentN != 1 || out.Replay {
		t.Fatalf("outcome = %+v, want n=2 parent=1", out)
	}
	if out.BuildStatus != "succeeded" {
		t.Fatalf("status = %q, want succeeded", out.BuildStatus)
	}
}

// TestRecordAttemptReplay proves same-opkey idempotency: the retried
// POST replays instead of recording attempt 2.
func TestRecordAttemptReplay(t *testing.T) {
	srv, _ := testServer(t)
	conv, build := attemptBuild(t, srv, testTenant, "arp")
	ctx := context.Background()
	key := opkeyForTest(t)
	first, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build,
		attemptInput(), key)
	if err != nil || first.Replay {
		t.Fatalf("first = %+v/%v", first, err)
	}
	second, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build,
		attemptInput(), key)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replay || second.N != 1 {
		t.Fatalf("replay = %+v, want n=1 replay", second)
	}
	page, err := srv.RunTool(ctx, testTenant, "conversation.stream",
		map[string]any{"conversation_id": conv}, "arp-s")
	if err != nil {
		t.Fatal(err)
	}
	items, _ := page["items"].([]any)
	n := 0
	for _, it := range items {
		m, _ := it.(map[string]any)
		pm, _ := m["payload"].(map[string]any)
		if text, _ := pm["text"].(string); strings.Contains(text, "Attempt 1") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("attempt cards = %d, want 1", n)
	}
}

// TestRecordAttemptAcceptCloses proves acceptance ends the manual
// build through the legal edges; anything after refuses.
func TestRecordAttemptAcceptCloses(t *testing.T) {
	srv, _ := testServer(t)
	conv, build := attemptBuild(t, srv, testTenant, "aac")
	ctx := context.Background()
	in := attemptInput()
	in.Verdict = "accept"
	in.TestsPassed = true
	if _, _, err := srv.RecordAttempt(ctx, testTenant, testBinding, build, in, opkeyForTest(t)); err != nil {
		t.Fatal(err)
	}
	got, err := srv.RunTool(ctx, testTenant, "conversation.get",
		map[string]any{"conversation_id": conv}, "aac-g")
	if err != nil {
		t.Fatal(err)
	}
	builds, _ := got["builds"].([]any)
	b := builds[len(builds)-1].(map[string]any)
	if b["status"] != "succeeded" {
		t.Fatalf("status = %v, want succeeded", b["status"])
	}
	if turn, _ := b["turn"].(map[string]any); turn["status"] != "closed" {
		t.Fatalf("turn = %v, want closed", turn)
	}
	if _, code, err := srv.RecordAttempt(ctx, testTenant, testBinding, build,
		attemptInput(), opkeyForTest(t)); err == nil || code != http.StatusBadRequest {
		t.Fatalf("post-accept record = %d/%v, want 400", code, err)
	}
}

func TestRecordAttemptIsolation(t *testing.T) {
	srv, _ := testServer(t)
	_, build := attemptBuild(t, srv, testTenant, "ais")
	if _, code, err := srv.RecordAttempt(context.Background(), testTenantB, testBindingB,
		build, attemptInput(), opkeyForTest(t)); err == nil || code != http.StatusNotFound {
		t.Fatalf("cross-tenant record = %d/%v, want 404", code, err)
	}
}

// TestConsoleAttemptFormRoundTrip drives the whole B0 loop through
// HTTP: Line message opens the build, the run scene shows the
// packet, the form records the attempt, the context bar names the
// branch.
func TestConsoleAttemptFormRoundTrip(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	conv := consoleConversation(t, srv, cookie, "B0 demo task")
	rec := doRequest(t, srv, http.MethodPost, "/console/line",
		(&url.Values{"opkey": {opkeyForTest(t)}, "conversation_id": {conv},
			"body": {"add a health endpoint"}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("line = %d, want 302", rec.Code)
	}
	got, err := srv.RunTool(context.Background(), testTenant, "conversation.get",
		map[string]any{"conversation_id": conv}, "afr-g")
	if err != nil {
		t.Fatal(err)
	}
	builds, _ := got["builds"].([]any)
	build, _ := builds[len(builds)-1].(map[string]any)["build_id"].(string)
	rec = doRequest(t, srv, http.MethodGet,
		"/console?scene=run&conversation="+conv, "", cookie)
	body := rec.Body.String()
	for _, want := range []string{"Execution packet", "v0", "B0 demo task", "Record attempt result", `id="ctxbar"`} {
		if !strings.Contains(body, want) {
			t.Errorf("run scene lacks %q", want)
		}
	}
	form := url.Values{"opkey": {opkeyForTest(t)}, "conversation_id": {conv}, "build_id": {build},
		"branch": {"try/health"}, "tests_summary": {"go test ./... green"},
		"tests_passed": {"on"}, "report": {"added /healthz, all green"},
		"verdict": {"accept"}, "verdict_note": {"ship it"}}
	rec = doRequest(t, srv, http.MethodPost, "/console/attempt-result", form.Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("record = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "scene=run") {
		t.Fatalf("redirect = %q, want run scene", loc)
	}
	rec = doRequest(t, srv, http.MethodGet,
		"/console?scene=run&conversation="+conv, "", cookie)
	body = rec.Body.String()
	for _, want := range []string{"Attempt 1", "try/health", "accept", "succeeded"} {
		if !strings.Contains(body, want) {
			t.Errorf("run scene lacks %q after record", want)
		}
	}
	if strings.Contains(body, "Record attempt result") {
		t.Errorf("result form still shown on a succeeded build")
	}
}

// TestConsoleAttemptValidation pins the form contract: missing key,
// bad verdict, and missing fields fail closed.
func TestConsoleAttemptValidation(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	conv := consoleConversation(t, srv, cookie, "B0 validation")
	rec := doRequest(t, srv, http.MethodPost, "/console/line",
		(&url.Values{"opkey": {opkeyForTest(t)}, "conversation_id": {conv},
			"body": {"task"}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("line = %d", rec.Code)
	}
	got, err := srv.RunTool(context.Background(), testTenant, "conversation.get",
		map[string]any{"conversation_id": conv}, "av-g")
	if err != nil {
		t.Fatal(err)
	}
	builds, _ := got["builds"].([]any)
	build, _ := builds[len(builds)-1].(map[string]any)["build_id"].(string)
	good := func() url.Values {
		return url.Values{"opkey": {opkeyForTest(t)}, "conversation_id": {conv}, "build_id": {build},
			"branch": {"b"}, "tests_summary": {"s"}, "report": {"r"}, "verdict": {"rework"}}
	}
	cases := map[string]url.Values{
		"missing opkey":  {"conversation_id": {conv}, "build_id": {build}},
		"missing branch": func() url.Values { v := good(); v.Del("branch"); return v }(),
		"missing report": func() url.Values { v := good(); v.Del("report"); return v }(),
		"bad verdict":    func() url.Values { v := good(); v.Set("verdict", "maybe"); return v }(),
		"bad sha":        func() url.Values { v := good(); v.Set("commits", "not-a-sha!!"); return v }(),
		"unknown build":  func() url.Values { v := good(); v.Set("build_id", "nope"); return v }(),
	}
	for name, form := range cases {
		rec := doRequest(t, srv, http.MethodPost, "/console/attempt-result", form.Encode(), cookie)
		if rec.Code != http.StatusBadRequest && !(name == "unknown build" && rec.Code == http.StatusNotFound) {
			t.Errorf("%s = %d, want 400/404", name, rec.Code)
		}
	}
	// No session at all: 401, never a record.
	rec = doRequest(t, srv, http.MethodPost, "/console/attempt-result", good().Encode())
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous = %d, want 401", rec.Code)
	}
}
