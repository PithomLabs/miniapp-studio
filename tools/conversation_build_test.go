package tools

import (
	"strings"
	"testing"
)

func createConversation(t *testing.T, h harness, key, cred string) string {
	t.Helper()
	res := invoke(t, h, ToolConversationCreate, map[string]any{"title": "t-" + key}, key, cred)
	id, _ := res["conversation_id"].(string)
	if id == "" {
		t.Fatalf("conversation.create = %v", res)
	}
	return id
}

func sendMsg(t *testing.T, h harness, conv, body, key, cred string) (string, float64, string) {
	t.Helper()
	res := invoke(t, h, ToolMessageSend,
		map[string]any{"conversation_id": conv, "body": body}, key, cred)
	id, _ := res["message_id"].(string)
	seq, _ := res["seq"].(float64)
	build, _ := res["build_id"].(string)
	if id == "" || seq == 0 || build == "" {
		t.Fatalf("message.send = %v", res)
	}
	return id, seq, build
}

func createBuild(t *testing.T, h harness, conv, key, cred string) (buildID, turnID string) {
	t.Helper()
	res := invoke(t, h, ToolBuildCreate,
		map[string]any{"conversation_id": conv}, key, cred)
	buildID, _ = res["build_id"].(string)
	turnID, _ = res["turn_id"].(string)
	if buildID == "" || turnID == "" {
		t.Fatalf("build.create = %v", res)
	}
	if res["status"] != buildAssigned {
		t.Fatalf("status = %v, want assigned", res["status"])
	}
	return buildID, turnID
}

func ingest(t *testing.T, h harness, buildID, eventID, typ string, payload map[string]any, key, cred string) map[string]any {
	t.Helper()
	return invoke(t, h, ToolBuildIngestEvent, map[string]any{
		"build_id": buildID, "event_id": eventID, "type": typ, "payload": payload,
	}, key, cred)
}

func TestConversationFlow(t *testing.T) {
	h := setupFakes(t)
	conv := createConversation(t, h, "c-1", toolCredA)
	_, s1, build1 := sendMsg(t, h, conv, "hello", "c-2", toolCredA)
	_, s2, build2 := sendMsg(t, h, conv, "world", "c-3", toolCredA)
	if build1 != build2 {
		t.Errorf("second send opened %q, want reuse of %q", build2, build1)
	}
	// user, system(build opened), user — the ensure step is stream-visible.
	if s1 != 1 || s2 != 3 {
		t.Errorf("seqs = %v, %v, want 1 and 3", s1, s2)
	}
	page := invoke(t, h, ToolConversationStream,
		map[string]any{"conversation_id": conv}, "c-4", toolCredA)
	items, _ := page["items"].([]any)
	if len(items) != 3 {
		t.Fatalf("stream items = %d, want 3", len(items))
	}
	first, _ := items[0].(map[string]any)
	if first["kind"] != "message" || first["body"] != "hello" {
		t.Errorf("first item = %v", first)
	}
	// Cursor paging resumes after the cursor.
	page2 := invoke(t, h, ToolConversationStream,
		map[string]any{"conversation_id": conv, "after_seq": s1}, "c-5", toolCredA)
	items2, _ := page2["items"].([]any)
	if len(items2) != 2 {
		t.Fatalf("paged items = %d, want 2", len(items2))
	}
	if page2["latest_seq"] != s2 {
		t.Errorf("latest_seq = %v, want %v", page2["latest_seq"], s2)
	}
	list := invoke(t, h, ToolConversationList, map[string]any{}, "c-6", toolCredA)
	convs, _ := list["conversations"].([]any)
	if len(convs) != 1 {
		t.Errorf("conversations = %d, want 1", len(convs))
	}
	got := invoke(t, h, ToolConversationGet,
		map[string]any{"conversation_id": conv}, "c-7", toolCredA)
	conversation, _ := got["conversation"].(map[string]any)
	if conversation["id"] != conv {
		t.Errorf("conversation.get = %v", got)
	}
	if builds, _ := got["builds"].([]any); len(builds) != 1 {
		t.Errorf("builds = %d, want 1 (ensured by send)", len(builds))
	}
}

func TestConversationCrossTenant(t *testing.T) {
	h := setupFakes(t)
	conv := createConversation(t, h, "x-1", toolCredA)
	// Tenant B sees nothing: no list entry, no stream, no append.
	list := invoke(t, h, ToolConversationList, map[string]any{}, "x-2", toolCredB)
	if convs, _ := list["conversations"].([]any); len(convs) != 0 {
		t.Errorf("tenant B lists %d conversations", len(convs))
	}
	err := invokeErr(t, h, ToolConversationStream,
		mustJSON(t, map[string]any{"conversation_id": conv}), "x-3", toolCredB)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("cross-tenant stream = %v, want not found", err)
	}
	err = invokeErr(t, h, ToolMessageSend,
		mustJSON(t, map[string]any{"conversation_id": conv, "body": "hi"}), "x-4", toolCredB)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("cross-tenant append = %v, want not found", err)
	}
}

func TestBuildLifecycle(t *testing.T) {
	h := setupFakes(t)
	conv := createConversation(t, h, "b-1", toolCredA)
	buildID, _ := createBuild(t, h, conv, "b-2", toolCredA)
	st := invoke(t, h, ToolBuildStatus,
		map[string]any{"build_id": buildID}, "b-3", toolCredA)
	if st["status"] != buildAssigned {
		t.Fatalf("status = %v", st["status"])
	}
	turn, _ := st["turn"].(map[string]any)
	if turn["n"] != float64(1) || turn["status"] != turnOpen {
		t.Fatalf("turn = %v, want n=1 open", turn)
	}
	if qs, _ := st["open_questions"].([]any); len(qs) != 0 {
		t.Fatalf("open questions = %d", len(qs))
	}
	// started moves assigned → running and records the session ref.
	got := ingest(t, h, buildID, "e-started", evtStarted,
		map[string]any{"agent_session_id": "sess-1"}, "b-4", toolCredA)
	if got["transitioned"] != true || got["build_status"] != buildRunning {
		t.Fatalf("started = %v", got)
	}
	st = invoke(t, h, ToolBuildStatus,
		map[string]any{"build_id": buildID}, "b-5", toolCredA)
	if st["agent_session_ref"] != "sess-1" {
		t.Errorf("session ref = %v", st["agent_session_ref"])
	}
	// Cancel from running closes everything honestly.
	out := invoke(t, h, ToolBuildCancel,
		map[string]any{"build_id": buildID}, "b-6", toolCredA)
	if out["status"] != buildCancelled {
		t.Fatalf("cancel = %v", out)
	}
	st = invoke(t, h, ToolBuildStatus,
		map[string]any{"build_id": buildID}, "b-7", toolCredA)
	if st["status"] != buildCancelled {
		t.Errorf("status = %v, want cancelled", st["status"])
	}
	// Cancelling twice is a deterministic refusal, not a silent no-op.
	err := invokeErr(t, h, ToolBuildCancel,
		mustJSON(t, map[string]any{"build_id": buildID}), "b-8", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "cannot move from cancelled") {
		t.Errorf("second cancel = %v, want transition refusal", err)
	}
}
