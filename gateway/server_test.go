package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testServer(t *testing.T) (*Server, *FakeAdapter) {
	t.Helper()
	dir := t.TempDir()
	sessions, err := NewSessionManager(filepath.Join(dir, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	outbox, err := OpenOutbox(filepath.Join(dir, "outbox.json"))
	if err != nil {
		t.Fatal(err)
	}
	dedup, err := OpenDedupStore(filepath.Join(dir, "dedup.json"))
	if err != nil {
		t.Fatal(err)
	}
	adapter := NewFakeAdapter()
	adapter.Sessions = sessions
	return &Server{
		Keys: StaticProvider{}, Adapter: adapter, Sessions: sessions,
		Outbox: outbox, Dedup: dedup, WorkRoot: filepath.Join(dir, "work"),
	}, adapter
}

func postCommand(t *testing.T, srv *Server, body []byte) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/gw/v1/command", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return rec.Code, decoded
}

func signedCommand(t *testing.T, build, msgID, name string, payload map[string]any) []byte {
	t.Helper()
	raw, err := SignCommand(StaticTestKey(), build, msgID, name, payload)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestCommandAssignFlow(t *testing.T) {
	srv, adapter := testServer(t)
	code, out := postCommand(t, srv, signedCommand(t, "b-1", "c-1", CmdAssign,
		map[string]any{"prompt": "build it", "model": "m", "agent": "a", "workspace_id": "w-1"}))
	if code != http.StatusOK || out["session_ref"] == nil {
		t.Fatalf("assign = %d %v", code, out)
	}
	if len(adapter.Starts) != 1 || adapter.Starts[0].Prompt != "build it" {
		t.Fatalf("adapter starts = %+v", adapter.Starts)
	}
	if adapter.Starts[0].Turn != 1 {
		t.Errorf("turn = %d, want 1", adapter.Starts[0].Turn)
	}
	// The started event waits in the outbox for the deliverer.
	if q, _ := srv.Outbox.Depth(); q != 1 {
		t.Fatalf("outbox depth = %d, want 1", q)
	}
	due := srv.Outbox.Due(time.Now())
	env, err := Parse(due[0].Envelope)
	if err != nil {
		t.Fatal(err)
	}
	if env.Name != "agent.started" || env.BuildID != "b-1" {
		t.Errorf("outbox event = %s/%s", env.Name, env.BuildID)
	}
	if err := Verify(StaticTestKey(), env); err != nil {
		t.Errorf("outbox signature: %v", err)
	}
	// Exact redelivery replays the stored response without re-executing.
	code, out2 := postCommand(t, srv, signedCommand(t, "b-1", "c-1", CmdAssign,
		map[string]any{"prompt": "build it", "workspace_id": "w-1"}))
	if code != http.StatusOK || out2["session_ref"] != out["session_ref"] {
		t.Fatalf("redelivery = %d %v", code, out2)
	}
	if len(adapter.Starts) != 1 {
		t.Errorf("adapter started %d times", len(adapter.Starts))
	}
	// A second distinct assignment for the same build conflicts.
	code, _ = postCommand(t, srv, signedCommand(t, "b-1", "c-2", CmdAssign,
		map[string]any{"prompt": "again", "workspace_id": "w-1"}))
	if code != http.StatusConflict {
		t.Errorf("reassign = %d, want 409", code)
	}
}

func TestCommandContinueCancelStatus(t *testing.T) {
	srv, adapter := testServer(t)
	postCommand(t, srv, signedCommand(t, "b-2", "c-1", CmdAssign,
		map[string]any{"prompt": "p", "workspace_id": "w-1"}))
	code, out := postCommand(t, srv, signedCommand(t, "b-2", "c-2", CmdContinue,
		map[string]any{"question_id": "q", "kind": "clarification", "answer": "a", "turn": float64(2)}))
	if code != http.StatusOK || out["resumed"] != true {
		t.Fatalf("continue = %d %v", code, out)
	}
	if len(adapter.Continues) != 1 || adapter.Continues[0].Input.Answer != "a" {
		t.Fatalf("continues = %+v", adapter.Continues)
	}
	code, out = postCommand(t, srv, signedCommand(t, "b-2", "c-3", CmdStatus, nil))
	if code != http.StatusOK || out["state"] != SessionRunning {
		t.Fatalf("status = %d %v", code, out)
	}
	code, out = postCommand(t, srv, signedCommand(t, "b-2", "c-4", CmdCancel, nil))
	if code != http.StatusOK || out["cancelled"] != true {
		t.Fatalf("cancel = %d %v", code, out)
	}
	if len(adapter.Cancels) != 1 {
		t.Fatalf("cancels = %v", adapter.Cancels)
	}
	// Cancel is idempotent; continue after cancel conflicts.
	code, _ = postCommand(t, srv, signedCommand(t, "b-2", "c-5", CmdCancel, nil))
	if code != http.StatusOK {
		t.Errorf("second cancel = %d, want 200", code)
	}
	code, _ = postCommand(t, srv, signedCommand(t, "b-2", "c-6", CmdContinue,
		map[string]any{"question_id": "q", "answer": "a"}))
	if code != http.StatusConflict {
		t.Errorf("continue-after-cancel = %d, want 409", code)
	}
}

func TestCommandFetchArtifact(t *testing.T) {
	srv, _ := testServer(t)
	postCommand(t, srv, signedCommand(t, "b-3", "c-1", CmdAssign,
		map[string]any{"prompt": "p", "workspace_id": "w-1"}))
	code, out := postCommand(t, srv, signedCommand(t, "b-3", "c-2", CmdFetchArtifact,
		map[string]any{"manifest": map[string]any{"entries": []any{
			map[string]any{"path": "a.diff", "kind": "diff"},
		}}}))
	if code != http.StatusOK {
		t.Fatalf("fetch = %d %v", code, out)
	}
	arts, _ := out["artifacts"].([]any)
	if len(arts) != 1 {
		t.Fatalf("artifacts = %v", out)
	}
	first, _ := arts[0].(map[string]any)
	if first["digest"] != "sha256:fake" || first["data_b64"] != "ZmFrZQ==" {
		t.Errorf("artifact = %v", first)
	}
	code, _ = postCommand(t, srv, signedCommand(t, "b-3", "c-3", CmdFetchArtifact,
		map[string]any{"manifest": map[string]any{}}))
	if code != http.StatusBadRequest {
		t.Errorf("bad manifest = %d, want 400", code)
	}
}

func TestCommandRejects(t *testing.T) {
	srv, _ := testServer(t)
	postCommand(t, srv, signedCommand(t, "b-4", "c-1", CmdAssign,
		map[string]any{"prompt": "p", "workspace_id": "w-1"}))
	good := signedCommand(t, "b-4", "c-9", CmdStatus, map[string]any{})
	var tm map[string]any
	if err := json.Unmarshal(good, &tm); err != nil {
		t.Fatal(err)
	}
	tm["payload"] = map[string]any{"injected": true}
	tampered, err := json.Marshal(tm)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		body []byte
		want int
	}{
		"malformed":    {[]byte("{oops"), http.StatusBadRequest},
		"tampered":     {tampered, http.StatusUnauthorized},
		"unknown cmd":  {signedCommand(t, "b-4", "c-8", "frobnicate", map[string]any{}), http.StatusBadRequest},
		"unknown op":   {signedCommand(t, "b-4", "c-7", CmdStatus, map[string]any{}), http.StatusOK},
		"assign empty": {signedCommand(t, "b-5", "c-6", CmdAssign, map[string]any{}), http.StatusBadRequest},
		"assign no ws": {signedCommand(t, "b-5", "c-6b", CmdAssign,
			map[string]any{"prompt": "p"}), http.StatusBadRequest},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			code, _ := postCommand(t, srv, tc.body)
			if code != tc.want {
				t.Errorf("status = %d, want %d", code, tc.want)
			}
		})
	}
	// Unknown build: 404 without an oracle (valid signature, foreign id).
	code, _ := postCommand(t, srv, signedCommand(t, "nope", "c-5", CmdStatus, map[string]any{}))
	if code != http.StatusNotFound {
		t.Errorf("unknown build = %d, want 404", code)
	}
	// Wrong key: 401.
	var m map[string]any
	_ = json.Unmarshal(good, &m)
	m["signature"] = strings.Repeat("0", 64)
	raw, _ := json.Marshal(m)
	code, _ = postCommand(t, srv, raw)
	if code != http.StatusUnauthorized {
		t.Errorf("wrong key = %d, want 401", code)
	}
	// GET rejected (commands are POST-only).
	req := httptest.NewRequest(http.MethodGet, "/gw/v1/command", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET = %d, want 405", rec.Code)
	}
}

func TestDedupSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	_ = context.Background()
	d1, err := OpenDedupStore(dir + "/dedup.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := d1.Record("b", "c", DedupRecord{Status: 200, Body: json.RawMessage(`{}`)}); err != nil {
		t.Fatal(err)
	}
	d2, err := OpenDedupStore(dir + "/dedup.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := d2.Check("b", "c"); !ok {
		t.Error("record lost across reopen")
	}
	if _, ok := d2.Check("b", "other"); ok {
		t.Error("phantom record")
	}
}
