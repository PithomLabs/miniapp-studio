package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// packetAttempt inserts one attempt_result event the way RecordAttempt
// persists it, so packet tests read realistic rows.
func packetAttempt(t *testing.T, h harness, buildID, eventID string, n int, branch, report string) {
	t.Helper()
	payload := fmt.Sprintf(`{"n":%d,"parent_n":%d,"verdict":"rejected","git":{"branch":%q},"tests":{"passed":false},"report":%q}`,
		n, n-1, branch, report)
	if _, err := h.owner.Exec(context.Background(),
		`INSERT INTO studio.events (tenant_id, build_id, event_id, seq, type, payload)
		 VALUES ($1, $2, $3, $4, 'attempt_result', $5)`,
		toolTenantA, buildID, eventID, n, payload); err != nil {
		t.Fatal(err)
	}
}

// TestBuildPacketRoundTrip pins the v0 packet: version, ids,
// workspace, objective, task messages, branch hint, attempts with
// excerpts (never full reports), and evidence references.
func TestBuildPacketRoundTrip(t *testing.T) {
	h := setupFakes(t)
	conv, buildID := runningBuild(t, h, "pkt", toolCredA)
	sendMsg(t, h, conv, "fix the login redirect", "pkt-m1", toolCredA)
	sendMsg(t, h, conv, "it must preserve the next parameter", "pkt-m2", toolCredA)
	packetAttempt(t, h, buildID, "pkt-e1", 1, "fix/login", "tried X")
	got := invoke(t, h, ToolBuildPacket,
		map[string]any{"build_id": buildID}, "pkt-k", toolCredA)
	if got["packet_version"] != "v0" {
		t.Fatalf("version = %v, want v0", got["packet_version"])
	}
	if got["build_id"] != buildID || got["conversation_id"] != conv {
		t.Fatalf("ids = %v/%v", got["build_id"], got["conversation_id"])
	}
	ws, _ := got["workspace"].(map[string]any)
	if ws["id"] != "local-desktop" || ws["location"] != "local_desktop" {
		t.Fatalf("workspace = %v", ws)
	}
	msgs, _ := got["task_messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("messages = %d, want 2", len(msgs))
	}
	if m := msgs[0].(map[string]any); m["body"] != "fix the login redirect" || m["truncated"] == true {
		t.Fatalf("message 0 = %v", m)
	}
	if got["branch_hint"] != "fix/login" {
		t.Fatalf("branch_hint = %v, want fix/login", got["branch_hint"])
	}
	atts, _ := got["prior_attempts"].([]any)
	if len(atts) != 1 {
		t.Fatalf("attempts = %d, want 1", len(atts))
	}
	a := atts[0].(map[string]any)
	if a["n"] != 1.0 || a["verdict"] != "rejected" || a["event_id"] != "pkt-e1" {
		t.Fatalf("attempt = %v", a)
	}
	if _, ok := a["report"]; ok {
		t.Fatalf("attempt carries a full report: %v", a)
	}
	if a["report_excerpt"] != "tried X" {
		t.Fatalf("excerpt = %v", a["report_excerpt"])
	}
	refs, _ := got["evidence_refs"].(map[string]any)
	if _, ok := refs["artifacts"]; !ok {
		t.Fatalf("evidence_refs = %v", refs)
	}
	if br, _ := got["bridge"].(map[string]any); br["mode"] != "manual" {
		t.Fatalf("bridge = %v", br)
	}
}

// TestBuildPacketTruncation proves bounds are explicit: long bodies
// cut with a marker instead of silently.
func TestBuildPacketTruncation(t *testing.T) {
	h := setupFakes(t)
	conv, buildID := runningBuild(t, h, "ptt", toolCredA)
	sendMsg(t, h, conv, strings.Repeat("w", 5000), "ptt-m1", toolCredA)
	got := invoke(t, h, ToolBuildPacket,
		map[string]any{"build_id": buildID}, "ptt-k", toolCredA)
	msgs, _ := got["task_messages"].([]any)
	m := msgs[0].(map[string]any)
	if m["truncated"] != true {
		t.Fatalf("long body not flagged: %v", len(m["body"].(string)))
	}
	if !strings.HasSuffix(m["body"].(string), "…(truncated)") {
		t.Fatalf("long body lacks the marker")
	}
}

func TestBuildPacketIsolation(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "pti", toolCredA)
	err := invokeErr(t, h, ToolBuildPacket,
		mustJSON(t, map[string]any{"build_id": buildID}), "pti-k", toolCredB)
	if err == nil {
		t.Fatal("cross-tenant packet read allowed")
	}
}

func TestBuildPacketUnknownBuild(t *testing.T) {
	h := setupFakes(t)
	err := invokeErr(t, h, ToolBuildPacket,
		mustJSON(t, map[string]any{"build_id": "nope"}), "ptu-k", toolCredA)
	if err == nil {
		t.Fatal("unknown build packet allowed")
	}
}
