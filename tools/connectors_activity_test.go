package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/runtime"

	"github.com/PithomLabs/miniapp-studio/fly"
)

func TestConnectorsList(t *testing.T) {
	h := setupFakes(t)
	h.conns.Connections = []fly.Connection{
		{ID: "c1", Provider: "box", ConnectionType: "oauth"},
		{ID: "c2", Provider: "slack", ConnectionType: "oauth"},
	}
	res := invoke(t, h, ToolConnectorsList, map[string]any{}, "n-1", toolCredA)
	if res["status"] != "ok" {
		t.Errorf("list = %v", res)
	}
	if conns, _ := res["connections"].([]any); len(conns) != 2 {
		t.Errorf("connections = %d, want 2", len(conns))
	}
	filtered := invoke(t, h, ToolConnectorsList, map[string]any{"provider": "box"}, "n-2", toolCredA)
	if conns, _ := filtered["connections"].([]any); len(conns) != 1 {
		t.Errorf("filtered = %d, want 1", len(conns))
	}
	if n := auditCount(t, h, toolTenantA, ToolConnectorsList); n == 0 {
		t.Error("no audit evidence for connectors.list")
	}
}

func TestConnectorsListUnconfigured(t *testing.T) {
	h := setupTools(t, fly.NewFakeSpriteManager(), nil)
	res := invoke(t, h, ToolConnectorsList, map[string]any{}, "u-1", toolCredA)
	if res["status"] != "unconfigured" {
		t.Errorf("list = %v, want honest unconfigured state", res)
	}
}

func TestCustomAttest(t *testing.T) {
	h := setupFakes(t)
	h.conns.Connections = []fly.Connection{
		{ID: "conn_acme", Provider: "custom_api", ConnectionType: "api_key"},
	}
	res := invoke(t, h, ToolConnectorsAttest, map[string]any{
		"id": "conn_acme", "provider": "custom_api", "type": "api_key",
	}, "a-1", toolCredA)
	if res["recorded"] != true || res["id"] != "conn_acme" {
		t.Errorf("attest = %v", res)
	}
	invokeErr(t, h, ToolConnectorsAttest, mustJSON(t, map[string]any{
		"id": "conn_ghost", "provider": "custom_api", "type": "api_key",
	}), "a-2", toolCredA)
	invokeErr(t, h, ToolConnectorsAttest, mustJSON(t, map[string]any{
		"id": "conn_acme", "provider": "box", "type": "api_key",
	}), "a-3", toolCredA)
	// Smuggled credential fields are rejected by strict decoding.
	invokeErr(t, h, ToolConnectorsAttest, mustJSON(t, map[string]any{
		"id": "conn_acme", "provider": "custom_api", "type": "api_key",
		"access_token": "sk-marker-should-never-persist",
	}), "a-4", toolCredA)
	if n := auditCount(t, h, toolTenantA, ToolConnectorsAttest); n == 0 {
		t.Error("no audit evidence for connectors.custom_attest")
	}
}

func TestCustomAttestUnconfigured(t *testing.T) {
	h := setupTools(t, fly.NewFakeSpriteManager(), nil)
	invokeErr(t, h, ToolConnectorsAttest, mustJSON(t, map[string]any{
		"id": "conn_acme", "provider": "custom_api", "type": "api_key",
	}), "a-1", toolCredA)
}

func TestActivityList(t *testing.T) {
	h := setupFakes(t)
	createApp(t, h, toolCredA, toolTenantA, "shop", "e-1")
	createApp(t, h, toolCredB, toolTenantB, "other", "e-2")
	mine := invoke(t, h, ToolActivityList, map[string]any{"limit": 50}, "e-3", toolCredA)
	events, _ := mine["events"].([]any)
	if len(events) == 0 {
		t.Fatal("no events for tenant with activity")
	}
	theirs := invoke(t, h, ToolActivityList, map[string]any{"limit": 50}, "e-4", toolCredB)
	if evs, _ := theirs["events"].([]any); len(evs) == 0 {
		t.Fatal("no events for second tenant")
	}
	// Tenant scoping: neither side sees the other's agent.
	seen := map[string]bool{}
	for _, e := range events {
		seen[e.(map[string]any)["agent"].(string)] = true
	}
	if seen["agent-b"] {
		t.Error("tenant A sees tenant B activity")
	}
	// Pagination: cursor pages forward without overlap.
	first := invoke(t, h, ToolActivityList, map[string]any{"limit": 1}, "e-5", toolCredA)
	ev1, _ := first["events"].([]any)
	cursor, _ := first["next_cursor"].(string)
	if len(ev1) != 1 || cursor == "" {
		t.Fatalf("page 1 = %v", first)
	}
	second := invoke(t, h, ToolActivityList, map[string]any{"limit": 1, "cursor": cursor}, "e-6", toolCredA)
	ev2, _ := second["events"].([]any)
	if len(ev2) != 1 {
		t.Fatalf("page 2 = %v", second)
	}
	if ev1[0].(map[string]any)["id"] == ev2[0].(map[string]any)["id"] {
		t.Error("pagination overlapped")
	}
	invokeErr(t, h, ToolActivityList, mustJSON(t, map[string]any{"limit": 500}), "e-7", toolCredA)
	invokeErr(t, h, ToolActivityList, mustJSON(t, map[string]any{"cursor": "nope"}), "e-8", toolCredA)
}

func TestActivityListExclude(t *testing.T) {
	h := setupFakes(t)
	createApp(t, h, toolCredA, toolTenantA, "shop", "x-1")
	full := invoke(t, h, ToolActivityList, map[string]any{"limit": 50}, "x-2", toolCredA)
	all, _ := full["events"].([]any)
	if len(all) == 0 {
		t.Fatal("no events without exclusion")
	}
	filtered := invoke(t, h, ToolActivityList,
		map[string]any{"limit": 50, "exclude": []string{"app.create"}}, "x-3", toolCredA)
	kept, _ := filtered["events"].([]any)
	for _, e := range kept {
		if e.(map[string]any)["tool"] == "app.create" {
			t.Fatalf("excluded tool present: %v", e)
		}
	}
	// Unknown tools exclude nothing; oversized lists refuse.
	invoke(t, h, ToolActivityList, map[string]any{"limit": 50, "exclude": []string{"nope"}}, "x-4", toolCredA)
	invokeErr(t, h, ToolActivityList, mustJSON(t, map[string]any{"exclude": []string{""}}), "x-5", toolCredA)
	big := make([]string, 17)
	for i := range big {
		big[i] = "x"
	}
	invokeErr(t, h, ToolActivityList, mustJSON(t, map[string]any{"exclude": big}), "x-6", toolCredA)
}

func TestActivityListSurvivesStreamPollFlood(t *testing.T) {
	// Guardrail 2: transport polling must not displace business
	// receipts. Every audit row below is written through the real
	// Miniapp runtime invocation path (h.invoke), never inserted
	// directly — this is the complete behavior being claimed.
	h := setupFakes(t)
	conv := createConversation(t, h, "f-1", toolCredA)
	for i := 0; i < 40; i++ {
		invoke(t, h, ToolConversationStream,
			map[string]any{"conversation_id": conv, "after_seq": 0},
			"f-poll-"+string(rune('a'+i/26))+string(rune('a'+i%26)), toolCredA)
	}
	// 40 polls x 2 rows each = 80 stream rows; the unfiltered page
	// of 50 physically cannot contain the conversation.create
	// receipt anymore.
	plain := invoke(t, h, ToolActivityList, map[string]any{"limit": 50}, "f-2", toolCredA)
	for _, e := range plain["events"].([]any) {
		if e.(map[string]any)["tool"] == "conversation.create" {
			t.Fatal("unfiltered page unexpectedly still holds the receipt; test proves nothing")
		}
	}
	filtered := invoke(t, h, ToolActivityList,
		map[string]any{"limit": 50, "exclude": []string{"conversation.stream"}}, "f-3", toolCredA)
	found := false
	for _, e := range filtered["events"].([]any) {
		if e.(map[string]any)["tool"] == "conversation.create" {
			found = true
		}
		if e.(map[string]any)["tool"] == "conversation.stream" {
			t.Fatalf("excluded tool present: %v", e)
		}
	}
	if !found {
		t.Fatal("business receipt missing from filtered page under poll flood")
	}
}

func TestActivityScrubbing(t *testing.T) {
	h := setupFakes(t)
	ctx := context.Background()
	details := `{"ok":1,"access_token":"sk-marker-should-never-surface",` +
		`"nested":{"password":"hunter2"},"list":[{"secret":"x"}]}`
	if _, err := h.owner.Exec(ctx,
		`INSERT INTO app.audit_events (tenant_id, agent_id, authority_kind, authority_id,
		  miniapp_id, miniapp_version, tool, action, outcome, details)
		 VALUES ($1,'a','user','a','m','1','t','act','ok',$2)`, toolTenantA, details); err != nil {
		t.Fatalf("seed audit: %v", err)
	}
	out, err := h.svc.Invoke(ctx, runtime.InvokeRequest{
		MiniappID: MiniappID, Version: Version, Tool: ToolActivityList,
		Input: json.RawMessage(`{"limit":50}`), IdempotencyKey: "scrub-1",
		Credential: toolCredA, RequestedTrust: contract.TrustAutonomous,
	})
	if err != nil {
		t.Fatalf("activity.list: %v", err)
	}
	raw := string(out.Result)
	for _, marker := range []string{"sk-marker-should-never-surface", "hunter2", `"secret":"x"`} {
		if strings.Contains(raw, marker) {
			t.Errorf("secret marker %q present in response", marker)
		}
	}
	if !strings.Contains(raw, "[REDACTED]") {
		t.Error("no redaction markers in response")
	}
	if !strings.Contains(raw, `"ok":1`) {
		t.Error("safe payload lost during scrubbing")
	}
}
