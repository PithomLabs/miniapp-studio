package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func runningBuild(t *testing.T, h harness, prefix, cred string) (conv, buildID string) {
	t.Helper()
	conv = createConversation(t, h, prefix+"-c", cred)
	buildID, _ = createBuild(t, h, conv, prefix+"-b", cred)
	got := ingest(t, h, buildID, prefix+"-started", evtStarted,
		map[string]any{"agent_session_id": "sess-" + prefix}, prefix+"-s", cred)
	if got["build_status"] != buildRunning {
		t.Fatalf("setup: build = %v, want running", got)
	}
	return conv, buildID
}

func streamLen(t *testing.T, h harness, conv, key, cred string) int {
	t.Helper()
	page := invoke(t, h, ToolConversationStream,
		map[string]any{"conversation_id": conv}, key, cred)
	items, _ := page["items"].([]any)
	return len(items)
}

func TestQuestionAnswerRound(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "qa", toolCredA)
	q := ingest(t, h, buildID, "qa-q1", evtQuestion,
		map[string]any{"question_id": "q-1", "body": "Which region?"}, "qa-q", toolCredA)
	if q["transitioned"] != true || q["build_status"] != buildWaitingUser {
		t.Fatalf("question = %v", q)
	}
	st := invoke(t, h, ToolBuildStatus,
		map[string]any{"build_id": buildID}, "qa-st", toolCredA)
	qs, _ := st["open_questions"].([]any)
	if len(qs) != 1 {
		t.Fatalf("open questions = %d, want 1", len(qs))
	}
	first, _ := qs[0].(map[string]any)
	if first["id"] != "q-1" || first["kind"] != "clarification" {
		t.Fatalf("question = %v", first)
	}
	ans := invoke(t, h, ToolQuestionAnswer,
		map[string]any{"question_id": "q-1", "answer": "eu-west"}, "qa-a", toolCredA)
	if ans["build_status"] != buildRunning {
		t.Fatalf("answer = %v", ans)
	}
	// The answer is also a transcript message.
	if ans["message_id"] == nil || ans["seq"] == nil {
		t.Errorf("answer lacks message/seq: %v", ans)
	}
	// Answering twice fails: the question is no longer open.
	err := invokeErr(t, h, ToolQuestionAnswer,
		mustJSON(t, map[string]any{"question_id": "q-1", "answer": "x"}), "qa-a2", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "no longer open") {
		t.Errorf("second answer = %v, want no-longer-open", err)
	}
	// Second question closes turn 1 and parks turn 2.
	q2 := ingest(t, h, buildID, "qa-q2", evtQuestion,
		map[string]any{"question_id": "q-2", "body": "Size?"}, "qa-qb", toolCredA)
	if q2["build_status"] != buildWaitingUser {
		t.Fatalf("second question = %v", q2)
	}
	st = invoke(t, h, ToolBuildStatus,
		map[string]any{"build_id": buildID}, "qa-st2", toolCredA)
	turn, _ := st["turn"].(map[string]any)
	if turn["n"] != float64(2) || turn["status"] != turnWaitingUser {
		t.Fatalf("turn = %v, want n=2 waiting", turn)
	}
	// Complete the build after answering.
	invoke(t, h, ToolQuestionAnswer,
		map[string]any{"question_id": "q-2", "answer": "large"}, "qa-a3", toolCredA)
	done := ingest(t, h, buildID, "qa-done", evtCompleted,
		map[string]any{}, "qa-d", toolCredA)
	if done["transitioned"] != true || done["build_status"] != buildSucceeded {
		t.Fatalf("completed = %v", done)
	}
	st = invoke(t, h, ToolBuildStatus,
		map[string]any{"build_id": buildID}, "qa-st3", toolCredA)
	if turn, _ := st["turn"].(map[string]any); turn["status"] != turnClosed {
		t.Errorf("final turn = %v, want closed", turn)
	}
}

func TestAuthorizationAnswerDecides(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "az", toolCredA)
	ingest(t, h, buildID, "az-q", evtAuthorizationRequest,
		map[string]any{"question_id": "az-1", "body": "Deploy?"}, "az-e", toolCredA)
	// A vague answer is not a decision.
	err := invokeErr(t, h, ToolQuestionAnswer,
		mustJSON(t, map[string]any{"question_id": "az-1", "answer": "maybe"}), "az-v", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "approved or rejected") {
		t.Fatalf("vague answer = %v, want decision refusal", err)
	}
	ans := invoke(t, h, ToolQuestionAnswer,
		map[string]any{"question_id": "az-1", "answer": "approved"}, "az-a", toolCredA)
	if ans["build_status"] != buildRunning {
		t.Fatalf("answer = %v", ans)
	}
	var decision, by string
	err = h.owner.QueryRow(context.Background(),
		`SELECT decision, decided_by FROM studio.approvals
		 WHERE tenant_id = $1 AND question_id = $2`, toolTenantA, "az-1").Scan(&decision, &by)
	if err != nil {
		t.Fatalf("approvals row: %v", err)
	}
	if decision != "approved" || by != "agent-a" {
		t.Errorf("approval = %q by %q", decision, by)
	}
}

func TestYieldSupersedes(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "sup", toolCredA)
	ingest(t, h, buildID, "sup-q1", evtQuestion,
		map[string]any{"question_id": "sup-1", "body": "First?"}, "sup-e1", toolCredA)
	ingest(t, h, buildID, "sup-q2", evtQuestion,
		map[string]any{"question_id": "sup-2", "body": "Second?"}, "sup-e2", toolCredA)
	// The first question expired; answering it fails.
	err := invokeErr(t, h, ToolQuestionAnswer,
		mustJSON(t, map[string]any{"question_id": "sup-1", "answer": "x"}), "sup-a1", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "no longer open") {
		t.Errorf("superseded answer = %v, want no-longer-open", err)
	}
	st := invoke(t, h, ToolBuildStatus,
		map[string]any{"build_id": buildID}, "sup-st", toolCredA)
	qs, _ := st["open_questions"].([]any)
	if len(qs) != 1 {
		t.Fatalf("open = %d, want 1", len(qs))
	}
	if first, _ := qs[0].(map[string]any); first["id"] != "sup-2" {
		t.Errorf("open question = %v, want sup-2", first)
	}
	if turn, _ := st["turn"].(map[string]any); turn["n"] != float64(2) {
		t.Errorf("turn = %v, want n=2", turn)
	}
}

func TestIngestDuplicatesAndTerminalAbsorb(t *testing.T) {
	h := setupFakes(t)
	conv, buildID := runningBuild(t, h, "dup", toolCredA)
	before := streamLen(t, h, conv, "dup-s0", toolCredA)
	first := ingest(t, h, buildID, "dup-e1", evtProgress,
		map[string]any{"text": "working"}, "dup-k1", toolCredA)
	if first["duplicate"] == true {
		t.Fatalf("first delivery marked duplicate: %v", first)
	}
	again := ingest(t, h, buildID, "dup-e1", evtProgress,
		map[string]any{"text": "working"}, "dup-k2", toolCredA)
	if again["duplicate"] != true {
		t.Fatalf("redelivery = %v, want duplicate ack", again)
	}
	if n := streamLen(t, h, conv, "dup-s1", toolCredA); n != before+1 {
		t.Errorf("stream grew by %d, want exactly 1", n-before)
	}
	// Terminal builds absorb late events as history without transitioning.
	invoke(t, h, ToolBuildCancel,
		map[string]any{"build_id": buildID}, "dup-c", toolCredA)
	late := ingest(t, h, buildID, "dup-late", evtCompleted,
		map[string]any{}, "dup-k3", toolCredA)
	if late["transitioned"] == true || late["build_status"] != buildCancelled {
		t.Errorf("late completed = %v, want absorbed without transition", late)
	}
	if n := streamLen(t, h, conv, "dup-s2", toolCredA); n != before+3 {
		t.Errorf("stream = %d, want progress + cancel-note + late event", n)
	}
}

func TestIngestUnknownAndArtifacts(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "unk", toolCredA)
	// Unknown future types persist opaquely without transitions or crashes.
	got := ingest(t, h, buildID, "unk-e1", "agent.future_thing",
		map[string]any{"x": 1}, "unk-k1", toolCredA)
	if got["transitioned"] == true || got["build_status"] != buildRunning {
		t.Errorf("unknown = %v, want history-only", got)
	}
	// Artifacts persist unverified; malformed entries skip; digests dedup.
	done := ingest(t, h, buildID, "unk-done", evtCompleted, map[string]any{
		"artifacts": []any{
			map[string]any{"kind": "diff", "digest": "d1", "size_bytes": float64(10)},
			map[string]any{"kind": "bogus", "digest": "dx"},
			map[string]any{"kind": "diff"},
			"not-an-object",
			map[string]any{"kind": "diff", "digest": "d1"},
		},
	}, "unk-k2", toolCredA)
	if done["build_status"] != buildSucceeded {
		t.Fatalf("completed = %v", done)
	}
	var n int
	var verified bool
	err := h.owner.QueryRow(context.Background(),
		`SELECT count(*), bool_and(verified_at IS NULL) FROM studio.artifacts
		 WHERE tenant_id = $1 AND build_id = $2`, toolTenantA, buildID).Scan(&n, &verified)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || !verified {
		t.Errorf("artifacts = %d rows, all-unverified=%v, want 1 unverified", n, verified)
	}
}

func TestIngestCrossTenant(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "ct", toolCredA)
	err := invokeErr(t, h, ToolBuildIngestEvent,
		mustJSON(t, map[string]any{"build_id": buildID, "event_id": "x",
			"type": evtProgress, "payload": map[string]any{}}), "ct-k", toolCredB)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("cross-tenant ingest = %v, want not found", err)
	}
	err = invokeErr(t, h, ToolBuildCancel,
		mustJSON(t, map[string]any{"build_id": buildID}), "ct-c", toolCredB)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("cross-tenant cancel = %v, want not found", err)
	}
}

func TestIngestYieldOnAssignedParks(t *testing.T) {
	h := setupFakes(t)
	conv := createConversation(t, h, "nr-c", toolCredA)
	buildID, _ := createBuild(t, h, conv, "nr-b", toolCredA)
	// CONTRACT CHANGE (prompt18 U2, needs owner ratification): a yield on
	// an assigned build used to absorb (history only). Absorbing strands
	// the agent — it waits for an answer to a question Studio never
	// parks, and the later completed/failed absorbs too, wedging the
	// build assigned forever with no recovery path. The authenticated
	// yield proves a live run whose started was lost or reordered, so it
	// parks: assigned → running → waiting_user with the question open.
	got := ingest(t, h, buildID, "nr-q", evtQuestion,
		map[string]any{"question_id": "nr-1", "body": "Early?"}, "nr-k", toolCredA)
	if got["transitioned"] != true || got["build_status"] != buildWaitingUser {
		t.Fatalf("early yield = %v, want parked/waiting_user", got)
	}
	st := invoke(t, h, ToolBuildStatus,
		map[string]any{"build_id": buildID}, "nr-st", toolCredA)
	qs, _ := st["open_questions"].([]any)
	if len(qs) != 1 {
		t.Fatalf("open questions = %d, want 1", len(qs))
	}
	if first, _ := qs[0].(map[string]any); first["id"] != "nr-1" {
		t.Errorf("open question = %v, want nr-1", first)
	}
	if turn, _ := st["turn"].(map[string]any); turn["status"] != turnWaitingUser {
		t.Errorf("turn = %v, want waiting_user", turn)
	}
}

func TestIngestDefensiveTurnBranch(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "def", toolCredA)
	// Stage the unreachable-through-tools state directly: latest turn
	// closed while the build still runs. The yield must park a fresh turn
	// rather than fail a legal event.
	if _, err := h.owner.Exec(context.Background(),
		`UPDATE studio.turns SET status = 'closed', ended_at = now()
		 WHERE tenant_id = $1 AND build_id = $2`, toolTenantA, buildID); err != nil {
		t.Fatal(err)
	}
	got := ingest(t, h, buildID, "def-q", evtQuestion,
		map[string]any{"question_id": "def-1", "body": "Q?"}, "def-k", toolCredA)
	if got["transitioned"] != true || got["build_status"] != buildWaitingUser {
		t.Fatalf("yield = %v", got)
	}
	st := invoke(t, h, ToolBuildStatus,
		map[string]any{"build_id": buildID}, "def-st", toolCredA)
	if turn, _ := st["turn"].(map[string]any); turn["n"] != float64(2) {
		t.Errorf("turn = %v, want fresh n=2", turn)
	}
}

func TestAgentContentScrubbedOnRead(t *testing.T) {
	h := setupFakes(t)
	conv, buildID := runningBuild(t, h, "scr", toolCredA)
	ingest(t, h, buildID, "scr-e1", evtProgress, map[string]any{
		"text": "hi", "api_key": "sk-live-secret",
		"nested": map[string]any{"token": "abc"},
	}, "scr-k1", toolCredA)
	ingest(t, h, buildID, "scr-q", evtProposal, map[string]any{
		"question_id": "scr-1", "body": "Pick one",
		"options": []any{"a", map[string]any{"api_key": "sk-opt-secret"}},
	}, "scr-k2", toolCredA)
	page := invoke(t, h, ToolConversationStream,
		map[string]any{"conversation_id": conv}, "scr-s", toolCredA)
	raw, _ := json.Marshal(page)
	if strings.Contains(string(raw), "sk-live-secret") {
		t.Errorf("stream leaks secret: %s", raw)
	}
	if !strings.Contains(string(raw), "[REDACTED]") || !strings.Contains(string(raw), `"text":"hi"`) {
		t.Errorf("stream scrub wrong: %s", raw)
	}
	st := invoke(t, h, ToolBuildStatus,
		map[string]any{"build_id": buildID}, "scr-st", toolCredA)
	raw, _ = json.Marshal(st)
	if strings.Contains(string(raw), "sk-opt-secret") {
		t.Errorf("status leaks secret: %s", raw)
	}
	// User transcript text stays verbatim (it is the operator's own words).
	sendMsg(t, h, conv, "my token is abc", "scr-m", toolCredA)
	page = invoke(t, h, ToolConversationStream,
		map[string]any{"conversation_id": conv}, "scr-s2", toolCredA)
	raw, _ = json.Marshal(page)
	if !strings.Contains(string(raw), "my token is abc") {
		t.Errorf("user text mangled: %s", raw)
	}
}

// TestWritePathCanariesAreSanitized is the closure item 1 regression test:
// embedded secret canaries driven through the real ingest path must be
// sanitized in Studio-owned persistence AND on every read surface, with
// surrounding prose preserved. This is the previously failing scenario
// (question bodies returned verbatim).
func TestWritePathCanariesAreSanitized(t *testing.T) {
	h := setupFakes(t)
	conv, buildID := runningBuild(t, h, "can", toolCredA)
	canary := "sk-live-CANARY0000000000000001"
	body := "paste " + canary + " to proceed"
	ingest(t, h, buildID, "can-q", evtQuestion,
		map[string]any{"question_id": "can-1", "body": body}, "can-k1", toolCredA)
	ctx := context.Background()
	// Database persistence: questions.body and events.payload sanitized.
	var storedBody, storedPayload string
	if err := h.owner.QueryRow(ctx,
		`SELECT body FROM studio.questions WHERE tenant_id = $1 AND id = $2`,
		toolTenantA, "can-1").Scan(&storedBody); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(storedBody, canary) {
		t.Errorf("questions.body persists the canary: %q", storedBody)
	}
	if !strings.Contains(storedBody, "paste ") || !strings.Contains(storedBody, " to proceed") {
		t.Errorf("surrounding prose lost: %q", storedBody)
	}
	if err := h.owner.QueryRow(ctx,
		`SELECT payload::text FROM studio.events WHERE tenant_id = $1 AND event_id = $2`,
		toolTenantA, "can-q").Scan(&storedPayload); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(storedPayload, canary) {
		t.Errorf("events.payload persists the canary")
	}
	// Every read surface: build.status, conversation.get, stream.
	for _, tc := range []struct {
		tool  string
		input map[string]any
		key   string
	}{
		{ToolBuildStatus, map[string]any{"build_id": buildID}, "can-r1"},
		{ToolConversationGet, map[string]any{"conversation_id": conv}, "can-r2"},
		{ToolConversationStream, map[string]any{"conversation_id": conv}, "can-r3"},
	} {
		out := invoke(t, h, tc.tool, tc.input, tc.key, toolCredA)
		raw, _ := json.Marshal(out)
		if strings.Contains(string(raw), canary) {
			t.Errorf("%s leaks the canary", tc.tool)
		}
	}
}

func TestConversationToolsAudited(t *testing.T) {
	h := setupFakes(t)
	conv := createConversation(t, h, "au-1", toolCredA)
	sendMsg(t, h, conv, "hi", "au-2", toolCredA)
	for _, tool := range []string{ToolConversationCreate, ToolMessageSend} {
		if n := auditCount(t, h, toolTenantA, tool); n == 0 {
			t.Errorf("no audit evidence for %s", tool)
		}
	}
}

func TestCommandsLogged(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "cmd", toolCredA)
	var n int
	err := h.owner.QueryRow(context.Background(),
		`SELECT count(*) FROM studio.commands WHERE tenant_id = $1 AND build_id = $2 AND state = 'queued'`,
		toolTenantA, buildID).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("queued commands = %d, want 1 (assign)", n)
	}
}
