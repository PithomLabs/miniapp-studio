package gateway

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPumpFailureMarksDegraded (F8): when the outbox cannot persist,
// the pump must not drop events silently — it marks the session
// degraded (durably, surfaced via status) and still drains the run to
// completion instead of wedging the build forever.
func TestPumpFailureMarksDegraded(t *testing.T) {
	dir := t.TempDir()
	adapter := NewFakeAdapter()
	adapter.RunEvents = [][]AgentEvent{{{Kind: EventMessage, Text: "hi"}, {Kind: EventDone}}}
	srv := f4Server(t, dir, adapter)
	adapter.Sessions = srv.Sessions
	f4Record(t, srv, "b-deg")
	// Break only the outbox file: sessions stay writable so the
	// degraded mark can land.
	_ = os.Remove(filepath.Join(dir, "outbox.json"))
	if err := os.Mkdir(filepath.Join(dir, "outbox.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The direct resumed-emit fails too (indeterminate 500); the pump's
	// message event must still degrade-mark instead of vanishing.
	code, _ := postCommand(t, srv, signedCommand(t, "b-deg", "deg-1", CmdContinue,
		map[string]any{"question_id": "q", "kind": "clarification", "answer": "a", "turn": 1}))
	if code != http.StatusInternalServerError {
		t.Fatalf("continue = %d, want 500 (direct emit failed honestly)", code)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		rec, _ := srv.Sessions.Get("b-deg")
		if rec.Degraded || time.Now().After(deadline) {
			if !rec.Degraded {
				t.Fatal("session never marked degraded after pump failures")
			}
			if rec.DegradedDetail == "" {
				t.Error("degraded flag set without detail")
			}
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	// The run still drained: no wedged build, no leaked pump.
	code, out := postCommand(t, srv, signedCommand(t, "b-deg", "deg-2", CmdStatus, nil))
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	if out["degraded"] != true {
		t.Errorf("status degraded = %v, want true (operator-visible)", out["degraded"])
	}
}
