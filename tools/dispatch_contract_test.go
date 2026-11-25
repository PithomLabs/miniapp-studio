package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/store"
)

// queryRow runs one scan inside tenant context (FORCE RLS filters
// everything without it, including owner connections).
func queryRow(t *testing.T, h harness, tenant, query string, args []any, dest ...any) {
	t.Helper()
	err := store.InTenantTx(context.Background(), h.owner, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), query, args...).Scan(dest...)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestBuildCreateStoresModel pins the sender contract: an explicit model
// persists on the build row and an assign command is queued for it.
func TestBuildCreateStoresModel(t *testing.T) {
	h := setupFakes(t)
	conv := createConversation(t, h, "dm-c", toolCredA)
	res := invoke(t, h, ToolBuildCreate,
		map[string]any{"conversation_id": conv, "model": "openrouter/deepseek/deepseek-v4.1-flash"}, "dm-p", toolCredA)
	buildID, _ := res["build_id"].(string)
	var model string
	var turnSet bool
	queryRow(t, h, toolTenantA,
		`SELECT model, gateway_turn IS NOT NULL FROM studio.builds WHERE id = $1`,
		[]any{buildID}, &model, &turnSet)
	if model != "openrouter/deepseek/deepseek-v4.1-flash" {
		t.Errorf("builds.model = %q, want the requested model", model)
	}
	if turnSet {
		t.Error("gateway_turn set before any dispatch, want NULL (unknown)")
	}
	var kind string
	queryRow(t, h, toolTenantA,
		`SELECT kind FROM studio.commands WHERE build_id = $1`,
		[]any{buildID}, &kind)
	if kind != "assign" {
		t.Errorf("queued command = %q, want assign", kind)
	}
}

// TestBuildCreateDefaultModelEmpty pins the default path: no model in,
// empty model stored, gateway default applies at dispatch.
func TestBuildCreateDefaultModelEmpty(t *testing.T) {
	h := setupFakes(t)
	conv := createConversation(t, h, "dm-c2", toolCredA)
	buildID, _ := createBuild(t, h, conv, "dm-p2", toolCredA)
	var model string
	queryRow(t, h, toolTenantA,
		`SELECT model FROM studio.builds WHERE id = $1`,
		[]any{buildID}, &model)
	if model != "" {
		t.Errorf("builds.model = %q, want empty default", model)
	}
}

// TestBuildCreateRejectsBadModel pins fail-fast model validation: a
// malformed id fails here, never as a cloud-side failure.
func TestBuildCreateRejectsBadModel(t *testing.T) {
	h := setupFakes(t)
	conv := createConversation(t, h, "dm-c4", toolCredA)
	for i, bad := range []string{"not-a-model-id", "/leading", "trailing/", "double//slash", "/"} {
		err := invokeErr(t, h, ToolBuildCreate,
			mustJSON(t, map[string]any{"conversation_id": conv, "model": bad}),
			"dm-p4-"+string(rune('a'+i)), toolCredA)
		if err == nil {
			t.Errorf("model %q accepted, want refusal", bad)
		}
	}
}

// TestMessageSendAssignCarriesMessageID pins the prompt-resolution
// contract: the auto-assign names the triggering user message so the
// sender never guesses which message is the brief.
func TestMessageSendAssignCarriesMessageID(t *testing.T) {
	h := setupFakes(t)
	conv := createConversation(t, h, "dm-c3", toolCredA)
	msgID, _, build := sendMsg(t, h, conv, "review my invoices", "dm-p3", toolCredA)
	var raw json.RawMessage
	queryRow(t, h, toolTenantA,
		`SELECT payload FROM studio.commands WHERE build_id = $1 AND kind = 'assign'`,
		[]any{build}, &raw)
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["message_id"] != msgID {
		t.Errorf("assign payload message_id = %v, want %q", payload["message_id"], msgID)
	}
}
