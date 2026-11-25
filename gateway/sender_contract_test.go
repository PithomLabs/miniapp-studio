package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestStatusReportsAcceptedTurn pins the reconcile primitive Studio's
// sender depends on: status.turn is the last turn the gateway accepted,
// advancing only on accepted commands.
func TestStatusReportsAcceptedTurn(t *testing.T) {
	srv, _ := testServer(t)
	code, out := postCommand(t, srv, signedCommand(t, "b-turn", "a-1", CmdAssign,
		map[string]any{"prompt": "p", "workspace_id": "w-1"}))
	if code != http.StatusOK {
		t.Fatalf("assign = %d %v", code, out)
	}
	_, out = postCommand(t, srv, signedCommand(t, "b-turn", "s-1", CmdStatus, nil))
	if out["turn"] != float64(1) {
		t.Fatalf("status after assign = %v, want turn 1", out)
	}
	code, out = postCommand(t, srv, signedCommand(t, "b-turn", "c-1", CmdContinue,
		map[string]any{"question_id": "q", "kind": "k", "answer": "a", "turn": float64(2)}))
	if code != http.StatusOK {
		t.Fatalf("continue = %d %v", code, out)
	}
	_, out = postCommand(t, srv, signedCommand(t, "b-turn", "s-2", CmdStatus, nil))
	if out["turn"] != float64(2) {
		t.Fatalf("status after continue = %v, want turn 2", out)
	}
	// A refused continue (wrong turn) must not move the reported turn.
	code, _ = postCommand(t, srv, signedCommand(t, "b-turn", "c-bad", CmdContinue,
		map[string]any{"question_id": "q", "kind": "k", "answer": "a", "turn": float64(9)}))
	if code != http.StatusConflict {
		t.Fatalf("wrong-turn continue = %d, want 409", code)
	}
	_, out = postCommand(t, srv, signedCommand(t, "b-turn", "s-3", CmdStatus, nil))
	if out["turn"] != float64(2) {
		t.Fatalf("status after refused continue = %v, want turn 2", out)
	}
}

// TestCommandConflictCodes pins the machine-readable taxonomy Studio's
// sender classifies on: every deterministic 409 carries a stable code.
func TestCommandConflictCodes(t *testing.T) {
	srv, adapter := testServer(t)
	cases := []struct {
		name  string
		build string
		setup func()
		id    string
		cmd   string
		body  map[string]any
		code  string
	}{
		{"already assigned", "b-dup",
			func() {
				postCommand(t, srv, signedCommand(t, "b-dup", "a-1", CmdAssign,
					map[string]any{"prompt": "p", "workspace_id": "w-1"}))
			},
			"a-2", CmdAssign, map[string]any{"prompt": "p2", "workspace_id": "w-1"},
			"already_assigned"},
		{"stale turn", "b-stale",
			func() {
				postCommand(t, srv, signedCommand(t, "b-stale", "a-1", CmdAssign,
					map[string]any{"prompt": "p", "workspace_id": "w-1"}))
			},
			"c-1", CmdContinue, map[string]any{"question_id": "q", "kind": "k", "answer": "a", "turn": float64(9)},
			"stale_turn"},
		{"terminal session", "b-term",
			func() {
				postCommand(t, srv, signedCommand(t, "b-term", "a-1", CmdAssign,
					map[string]any{"prompt": "p", "workspace_id": "w-1"}))
				postCommand(t, srv, signedCommand(t, "b-term", "x-1", CmdCancel, nil))
			},
			"c-1", CmdContinue, map[string]any{"question_id": "q", "kind": "k", "answer": "a", "turn": float64(2)},
			"session_terminal"},
		{"busy session", "b-busy",
			func() {
				postCommand(t, srv, signedCommand(t, "b-busy", "a-1", CmdAssign,
					map[string]any{"prompt": "p", "workspace_id": "w-1"}))
				adapter.Fail["Continue"] = ErrSessionBusy
			},
			"c-1", CmdContinue, map[string]any{"question_id": "q", "kind": "k", "answer": "a", "turn": float64(2)},
			"session_busy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter.Fail = map[string]error{}
			tc.setup()
			code, out := postCommand(t, srv, signedCommand(t, tc.build, tc.id, tc.cmd, tc.body))
			if code != http.StatusConflict {
				t.Fatalf("%s = %d %v, want 409", tc.cmd, code, out)
			}
			if out["code"] != tc.code {
				t.Fatalf("code = %v, want %q (body %v)", out["code"], tc.code, out)
			}
		})
	}
}

// TestIndeterminateCarriesCode pins the poison marker: an ambiguous
// dispatch reports 500 with code indeterminate, and resending the same
// id replays indeterminate (never a fresh dispatch).
func TestIndeterminateCarriesCode(t *testing.T) {
	srv, adapter := testServer(t)
	adapter.Fail["Start"] = ErrAgentStray
	payload := map[string]any{"prompt": "p", "workspace_id": "w-1"}
	code, out := postCommand(t, srv, signedCommand(t, "b-ind", "a-1", CmdAssign, payload))
	if code != http.StatusInternalServerError || out["code"] != "indeterminate" {
		t.Fatalf("stray start = %d %v, want 500 + indeterminate", code, out)
	}
	// Clearing the fault must not let the poisoned id dispatch again:
	// the 409 below comes from the claim layer, not a fresh start.
	adapter.Fail = map[string]error{}
	code, out = postCommand(t, srv, signedCommand(t, "b-ind", "a-1", CmdAssign, payload))
	if code != http.StatusConflict || out["code"] != "indeterminate" {
		t.Fatalf("poisoned resend = %d %v, want 409 + indeterminate", code, out)
	}
	if len(adapter.Starts) != 0 {
		t.Fatalf("adapter started %d times, want 0 (poisoned id never dispatches)", len(adapter.Starts))
	}
}

// TestSendWithIDCollapsesAcrossClients pins the crash-recovery primitive
// the sender relies on: two client instances (Studio before and after a
// restart) sending the same stable id execute once and observe the same
// recorded response.
func TestSendWithIDCollapsesAcrossClients(t *testing.T) {
	srv, adapter := testServer(t)
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()
	ctx := context.Background()
	key := StaticTestKey()
	payload := map[string]any{"prompt": "hello", "workspace_id": "w-1"}
	first := &Client{Timeout: 5 * time.Second, Retries: 0}
	out1, err := first.SendWithID(ctx, httpSrv.URL, key, "b-sid", "stable-row-id", CmdAssign, payload)
	if err != nil {
		t.Fatalf("first send: %v", err)
	}
	second := &Client{Timeout: 5 * time.Second, Retries: 0}
	out2, err := second.SendWithID(ctx, httpSrv.URL, key, "b-sid", "stable-row-id", CmdAssign, payload)
	if err != nil {
		t.Fatalf("second send: %v", err)
	}
	if out1["session_ref"] == nil || out1["session_ref"] != out2["session_ref"] {
		t.Fatalf("responses differ: %v vs %v", out1, out2)
	}
	if len(adapter.Starts) != 1 {
		t.Fatalf("adapter started %d times, want 1", len(adapter.Starts))
	}
	if _, err := first.SendWithID(ctx, httpSrv.URL, key, "b-sid", "", CmdStatus, nil); err == nil {
		t.Error("empty message id accepted, want refusal")
	}
}
