package gateway

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// The file-drop tests below drive the adapter against caller-owned
// workdirs so fixtures (symlinks, modes, sizes) are exact.
func filedropAdapter(t *testing.T, scripts []Script) (*OpenCodeAdapter, *SessionManager, string) {
	t.Helper()
	dir := t.TempDir()
	sessions, err := NewSessionManager(filepath.Join(dir, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(dir, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &ScriptedRunner{Scripts: scripts}
	return &OpenCodeAdapter{Runner: runner, Sessions: sessions}, sessions, work
}

func okTranscript(session string) string {
	return "{\"type\":\"text\",\"timestamp\":1,\"sessionID\":\"" + session + "\",\"part\":{\"type\":\"text\",\"text\":\"hi\"}}\n"
}

func TestQuestionSymlinkToKeyRefused(t *testing.T) {
	keyDir := t.TempDir()
	keyFile := filepath.Join(keyDir, "keys.json")
	if err := os.WriteFile(keyFile, []byte(`{"b":"HMAC-KEY-MATERIAL"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter, _, work := filedropAdapter(t, []Script{{Stdout: okTranscript("s-f2")}})
	inbox := filepath.Join(work, inboxDir)
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(keyFile, filepath.Join(inbox, questionFileName)); err != nil {
		t.Fatal(err)
	}
	_, ch, err := adapter.Start(context.Background(), BuildAssignment{BuildID: "b-f2", WorkDir: work, Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	for ev := range ch {
		if ev.Kind == EventQuestion && ev.Question != nil &&
			strings.Contains(ev.Question.Body, "HMAC-KEY-MATERIAL") {
			t.Fatal("question symlink exfiltrated key material")
		}
	}
	rec, _ := adapter.Sessions.Get("b-f2")
	if rec.State != SessionFailed {
		t.Errorf("record state = %q, want failed (invalid drop)", rec.State)
	}
}

// TestAnswerSymlinkRemoved (F3): a pre-created answer.json symlink to a
// victim file must be removed and replaced — the victim stays intact.
func TestAnswerSymlinkRemoved(t *testing.T) {
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim.json")
	if err := os.WriteFile(victim, []byte(`{"k":"ORIGINAL"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter, sessions, work := filedropAdapter(t, []Script{{Stdout: okTranscript("s-f3")}})
	inbox := filepath.Join(work, inboxDir)
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, filepath.Join(inbox, answerFileName)); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Create("b-f3", SessionRecord{SessionRef: "s-f3", BuildID: "b-f3", Workspace: "t", WorkDir: work, State: SessionYielded}); err != nil {
		t.Fatal(err)
	}
	ch, err := adapter.Continue(context.Background(), "s-f3", UserInput{QuestionID: "q", Kind: "clarification", Answer: "a", Turn: 1})
	if err != nil {
		t.Fatal(err)
	}
	_ = drain(t, ch)
	after, _ := os.ReadFile(victim)
	if !strings.Contains(string(after), "ORIGINAL") {
		t.Errorf("victim clobbered via answer symlink: %q", after)
	}
	fi, err := os.Lstat(filepath.Join(inbox, answerFileName))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Error("answer path still a symlink after Continue")
	}
	if fi.Mode().Perm() != 0o640 {
		t.Errorf("answer mode = %o, want 640", fi.Mode().Perm())
	}
}

// TestAnswerPrecreatedRegularSafe (F3): an agent-pre-created regular
// answer file cannot forge the answer — the gateway replaces it with
// its own gateway-owned content.
func TestAnswerPrecreatedRegularSafe(t *testing.T) {
	adapter, sessions, work := filedropAdapter(t, []Script{{Stdout: okTranscript("s-f3r")}})
	inbox := filepath.Join(work, inboxDir)
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inbox, answerFileName), []byte(`{"answer":"forged"}`), 0o660); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Create("b-f3r", SessionRecord{SessionRef: "s-f3r", BuildID: "b-f3r", Workspace: "t", WorkDir: work, State: SessionYielded}); err != nil {
		t.Fatal(err)
	}
	ch, err := adapter.Continue(context.Background(), "s-f3r", UserInput{QuestionID: "q", Kind: "clarification", Answer: "genuine", Turn: 1})
	if err != nil {
		t.Fatal(err)
	}
	_ = drain(t, ch)
	raw, err := os.ReadFile(filepath.Join(inbox, answerFileName))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["answer"] != "genuine" {
		t.Errorf("answer = %v, want gateway-authored genuine", doc)
	}
}

// TestQuestionOversizedRefused (M10): a giant question drop is refused
// instead of being read unbounded into memory.
func TestQuestionOversizedRefused(t *testing.T) {
	adapter, _, work := filedropAdapter(t, []Script{{Stdout: okTranscript("s-m10")}})
	inbox := filepath.Join(work, inboxDir)
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	big := `{"kind":"clarification","body":"` + strings.Repeat("x", 256<<10) + `"}`
	if err := os.WriteFile(filepath.Join(inbox, questionFileName), []byte(big), 0o660); err != nil {
		t.Fatal(err)
	}
	_, ch, err := adapter.Start(context.Background(), BuildAssignment{BuildID: "b-m10", WorkDir: work, Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	for ev := range ch {
		if ev.Kind == EventQuestion {
			t.Fatal("oversized drop yielded a question, want refusal")
		}
	}
	rec, _ := adapter.Sessions.Get("b-m10")
	if rec.State != SessionFailed {
		t.Errorf("record state = %q, want failed (oversized drop)", rec.State)
	}
}

// TestQuestionUnreadableModeRefused (F2): a drop the gateway cannot read
// through the inbox group (wrong mode) fails the run honestly instead of
// yielding or wedging.
func TestQuestionUnreadableModeRefused(t *testing.T) {
	adapter, _, work := filedropAdapter(t, []Script{{Stdout: okTranscript("s-f2m")}})
	inbox := filepath.Join(work, inboxDir)
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	drop := filepath.Join(inbox, questionFileName)
	if err := os.WriteFile(drop, []byte(`{"kind":"clarification","body":"hi"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(drop, 0o600); err != nil {
		t.Fatal(err)
	}
	_, ch, err := adapter.Start(context.Background(), BuildAssignment{BuildID: "b-f2m", WorkDir: work, Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	for ev := range ch {
		if ev.Kind == EventQuestion {
			t.Fatal("unreadable-mode drop yielded, want honest failure")
		}
	}
	rec, _ := adapter.Sessions.Get("b-f2m")
	if rec.State != SessionFailed {
		t.Errorf("record state = %q, want failed (unreadable drop)", rec.State)
	}
}

// TestQuestionMalformedYieldsLenient: a well-owned but malformed drop is
// still a yield attempt — it reaches the user as clarification with the
// raw body intact rather than terminalizing the build.
func TestQuestionMalformedYieldsLenient(t *testing.T) {
	adapter, _, work := filedropAdapter(t, []Script{{Stdout: okTranscript("s-f2j")}})
	inbox := filepath.Join(work, inboxDir)
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inbox, questionFileName), []byte(`not json {{{`), 0o660); err != nil {
		t.Fatal(err)
	}
	_, ch, err := adapter.Start(context.Background(), BuildAssignment{BuildID: "b-f2j", WorkDir: work, Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for ev := range ch {
		kinds = append(kinds, string(ev.Kind))
	}
	found := false
	for _, k := range kinds {
		if k == string(EventQuestion) {
			found = true
		}
	}
	if !found {
		t.Errorf("kinds = %v, want a lenient question yield", kinds)
	}
}

// TestAnswerTempSweep (F3): a crashed replace's orphaned temp is swept
// on the next write, while a fresh (concurrently live) temp is left
// alone.
func TestAnswerTempSweep(t *testing.T) {
	adapter, _, work := filedropAdapter(t, nil)
	inbox := filepath.Join(work, inboxDir)
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(inbox, ".answer.json.deadbeef.tmp")
	fresh := filepath.Join(inbox, ".answer.json.live1234.tmp")
	for _, p := range []string{stale, fresh} {
		if err := os.WriteFile(p, []byte("{}"), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"answer": "x"})
	if err := adapter.writeAnswerDrop(work, raw); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale temp not swept")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("fresh temp swept, want preserved: %v", err)
	}
}

// TestConcurrentAnswerReplace (F3): hammering the answer slot
// concurrently never crashes, never leaves a torn file, and always
// leaves gateway-authored content behind.
func TestConcurrentAnswerReplace(t *testing.T) {
	adapter, _, work := filedropAdapter(t, nil)
	inbox := filepath.Join(work, inboxDir)
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 8; j++ {
				raw, _ := json.Marshal(map[string]any{"answer": i*100 + j})
				if err := adapter.writeAnswerDrop(work, raw); err != nil {
					errs <- err
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent replace: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(inbox, answerFileName))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("torn answer file: %q", raw)
	}
	if _, ok := doc["answer"]; !ok {
		t.Errorf("answer file = %v, want an authored answer", doc)
	}
}
