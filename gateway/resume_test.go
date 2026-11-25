package gateway

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// e2eServer builds a server wired to the real OpenCode adapter over a
// scripted runner (server+adapter integration, scripted agent process).
func e2eServer(t *testing.T, dir string, runner *ScriptedRunner) (*Server, *SessionManager) {
	t.Helper()
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
	work := filepath.Join(dir, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := &OpenCodeAdapter{Runner: runner, Sessions: sessions, KillGrace: 50 * time.Millisecond}
	srv := &Server{
		Keys: StaticProvider{}, Adapter: adapter, Sessions: sessions,
		Outbox: outbox, Dedup: dedup, WorkRoot: work,
	}
	return srv, sessions
}

func pollOutboxNames(t *testing.T, srv *Server, want int) []string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		due := srv.Outbox.Due(time.Now())
		if len(due) >= want || time.Now().After(deadline) {
			var names []string
			for _, item := range due {
				env, err := Parse(item.Envelope)
				if err != nil {
					t.Fatal(err)
				}
				names = append(names, env.Name)
			}
			return names
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestContinueFromFailedE2E (F6): recover → authorized continue (new
// fenced turn) → supervised resume → agent.resumed → completed, with
// the preserved workdir and session lineage.
func TestContinueFromFailedE2E(t *testing.T) {
	dir := t.TempDir()
	runner := &ScriptedRunner{Scripts: []Script{{
		Stdout: `{"type":"text","timestamp":1,"sessionID":"sess-e2e","part":{"type":"text","text":"back"}}` + "\n",
	}}}
	srv, sessions := e2eServer(t, dir, runner)
	work := filepath.Join(dir, "work")
	if err := sessions.Create("b-e2e", SessionRecord{
		SessionRef: "sess-e2e", BuildID: "b-e2e", Workspace: "w-e2e",
		WorkDir: work, State: SessionRunning, Turn: 1,
	}); err != nil {
		t.Fatal(err)
	}
	srv.RecoverSessions(context.Background())
	if rec, _ := sessions.Get("b-e2e"); rec.State != SessionFailed {
		t.Fatalf("post-recovery state = %q, want failed", rec.State)
	}
	code, out := postCommand(t, srv, signedCommand(t, "b-e2e", "resume-1", CmdContinue,
		map[string]any{"question_id": "q1", "kind": "clarification", "answer": "go on", "turn": 2}))
	if code != http.StatusOK || out["resumed"] != true {
		t.Fatalf("continue-from-failed = %d %v, want 200 resumed", code, out)
	}
	names := pollOutboxNames(t, srv, 4)
	if len(names) != 4 || names[0] != "agent.failed" || names[len(names)-1] != "agent.completed" {
		t.Fatalf("callbacks = %v, want [failed ... completed]", names)
	}
	mid := strings.Join(names[1:3], ",")
	if !strings.Contains(mid, "agent.resumed") || !strings.Contains(mid, "agent.message") {
		t.Errorf("callbacks = %v, want resumed + message in the middle", names)
	}
	rec, _ := sessions.Get("b-e2e")
	if rec.State != SessionDone {
		t.Errorf("record state = %q, want done", rec.State)
	}
	if rec.SessionRef != "sess-e2e" || rec.WorkDir != work {
		t.Errorf("lineage = %q %q, want preserved ref + workdir", rec.SessionRef, rec.WorkDir)
	}
	if rec.Turn != 2 {
		t.Errorf("record turn = %d, want 2", rec.Turn)
	}
}

// TestContinueFromDoneStillConflicts (F6): only failed sessions resume;
// done and cancelled stay terminal.
func TestContinueFromDoneStillConflicts(t *testing.T) {
	srv, _ := testServer(t)
	for _, tc := range []struct {
		build string
		state string
	}{
		{"b-done2", SessionDone},
		{"b-canc2", SessionCancelled},
	} {
		if err := srv.Sessions.Create(tc.build, SessionRecord{
			SessionRef: "s-" + tc.build, BuildID: tc.build,
			WorkDir: t.TempDir(), State: tc.state, Turn: 1,
		}); err != nil {
			t.Fatal(err)
		}
		code, _ := postCommand(t, srv, signedCommand(t, tc.build, "c-"+tc.build, CmdContinue,
			map[string]any{"question_id": "q", "kind": "clarification", "answer": "a", "turn": 2}))
		if code != http.StatusConflict {
			t.Errorf("continue on %s = %d, want 409", tc.state, code)
		}
	}
}

// TestStaleTurnRefused (M11): a continue must carry exactly the next
// turn — repeats and skips conflict, never execute.
func TestStaleTurnRefused(t *testing.T) {
	srv, adapter := testServer(t)
	if err := srv.Sessions.Create("b-turn", SessionRecord{
		SessionRef: "s-turn", BuildID: "b-turn", Workspace: "w",
		WorkDir: t.TempDir(), State: SessionRunning, Turn: 2,
	}); err != nil {
		t.Fatal(err)
	}
	for _, turn := range []int{2, 4} {
		code, _ := postCommand(t, srv, signedCommand(t, "b-turn", "t-stale", CmdContinue,
			map[string]any{"question_id": "q", "kind": "clarification", "answer": "a", "turn": turn}))
		if code != http.StatusConflict {
			t.Errorf("turn %d on record-turn 2 = %d, want 409", turn, code)
		}
	}
	// The fenced next turn still works.
	code, _ := postCommand(t, srv, signedCommand(t, "b-turn", "t-next", CmdContinue,
		map[string]any{"question_id": "q", "kind": "clarification", "answer": "a", "turn": 3}))
	if code != http.StatusOK {
		t.Errorf("turn 3 on record-turn 2 = %d, want 200", code)
	}
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if len(adapter.Continues) != 1 {
		t.Errorf("executions = %d, want only the fenced turn", len(adapter.Continues))
	}
}

// TestContinueRequiresTurn (M11): a continue without turn identity is a
// malformed command, not an implicitly current turn.
func TestContinueRequiresTurn(t *testing.T) {
	srv, _ := testServer(t)
	if err := srv.Sessions.Create("b-noturn", SessionRecord{
		SessionRef: "s-noturn", BuildID: "b-noturn", Workspace: "w",
		WorkDir: t.TempDir(), State: SessionRunning, Turn: 1,
	}); err != nil {
		t.Fatal(err)
	}
	code, _ := postCommand(t, srv, signedCommand(t, "b-noturn", "c-no-turn", CmdContinue,
		map[string]any{"question_id": "q", "kind": "clarification", "answer": "a"}))
	if code != http.StatusBadRequest {
		t.Errorf("turn-less continue = %d, want 400", code)
	}
}

// TestRecoverySkipsCorruptRecord (M6): one corrupt session file must not
// abort recovery for every other build.
func TestRecoverySkipsCorruptRecord(t *testing.T) {
	dir := t.TempDir()
	srv := f4Server(t, dir, &overlapAdapter{})
	if err := os.WriteFile(filepath.Join(dir, "sessions", "zz-corrupt.json"), []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := srv.Sessions.Create("b-good", SessionRecord{
		SessionRef: "s-good", BuildID: "b-good", Workspace: "w",
		WorkDir: t.TempDir(), State: SessionRunning,
	}); err != nil {
		t.Fatal(err)
	}
	srv.RecoverSessions(context.Background())
	rec, err := srv.Sessions.Get("b-good")
	if err != nil {
		t.Fatal(err)
	}
	if rec.State != SessionFailed {
		t.Errorf("good record state = %q, want failed (recovery skipped it)", rec.State)
	}
	if q, _ := srv.Outbox.Depth(); q != 1 {
		t.Errorf("outbox depth = %d, want 1 failed callback", q)
	}
}

// TestRecoveryFailedCallbackIdempotent (M6): the recovery failed event
// carries a deterministic id — a crash between emit and mark re-emits
// the same id (Studio dedups) instead of minting a duplicate event.
func TestRecoveryFailedCallbackIdempotent(t *testing.T) {
	dir := t.TempDir()
	srv := f4Server(t, dir, &overlapAdapter{})
	if err := srv.Sessions.Create("b-idem", SessionRecord{
		SessionRef: "s-idem", BuildID: "b-idem", Workspace: "w",
		WorkDir: t.TempDir(), State: SessionRunning,
	}); err != nil {
		t.Fatal(err)
	}
	srv.RecoverSessions(context.Background())
	// Simulate a crash after emit but before mark: force the record
	// back to running and boot again.
	if err := srv.Sessions.Update("b-idem", func(r *SessionRecord) { r.State = SessionRunning }); err != nil {
		t.Fatal(err)
	}
	srv2 := f4Server(t, dir, &overlapAdapter{})
	srv2.RecoverSessions(context.Background())
	// The re-emit carries the same deterministic id, so the outbox
	// holds exactly one failed event (no duplicate) — and the
	// already-queued signal lets recovery proceed to marking.
	var ids []string
	for _, item := range srv2.Outbox.Due(time.Now()) {
		env, err := Parse(item.Envelope)
		if err != nil {
			t.Fatal(err)
		}
		if env.Name == "agent.failed" {
			ids = append(ids, env.MessageID)
		}
	}
	if len(ids) != 1 || ids[0] != "recovery-failed:b-idem" {
		t.Errorf("failed ids = %v, want exactly the deterministic id", ids)
	}
	rec, err := srv2.Sessions.Get("b-idem")
	if err != nil {
		t.Fatal(err)
	}
	if rec.State != SessionFailed {
		t.Errorf("record state = %q, want failed (mark proceeds past exists)", rec.State)
	}
}

// TestShouldReapDecoy (M7): a foreign process merely mentioning opencode
// in its arguments (an editor, a grep) must never be selected — only
// argv[0] identity counts.
func TestShouldReapDecoy(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "500")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte("vim\x00opencode-notes.txt\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if shouldReap(root, 500, 200) {
		t.Error("decoy cmdline selected for kill (substring match)")
	}
}

// TestAdapterContinueLineageMismatch (F6): when the resumed stream
// reports a different session than the one requested, the run is
// tainted — kill it and report ambiguity, never a clean resume.
func TestAdapterContinueLineageMismatch(t *testing.T) {
	scripts := []Script{{Stdout: okTranscript("ses-OTHER")}}
	adapter, sessions, work := filedropAdapter(t, scripts)
	if err := sessions.Create("b-lin", SessionRecord{
		SessionRef: "s-lin", BuildID: "b-lin", Workspace: "w",
		WorkDir: work, State: SessionYielded, Turn: 1,
	}); err != nil {
		t.Fatal(err)
	}
	_, err := adapter.Continue(context.Background(), "s-lin",
		UserInput{QuestionID: "q", Kind: "clarification", Answer: "a", Turn: 2})
	if !errors.Is(err, ErrAgentStray) {
		t.Errorf("lineage mismatch err = %v, want ErrAgentStray", err)
	}
	rec, _ := sessions.Get("b-lin")
	if rec.State != SessionYielded {
		t.Errorf("record state = %q, want yielded (untouched)", rec.State)
	}
}
