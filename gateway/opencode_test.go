package gateway

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testAdapter(t *testing.T, runner *ScriptedRunner) (*OpenCodeAdapter, *SessionManager, string) {
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
	return &OpenCodeAdapter{Runner: runner, Sessions: sessions, KillGrace: 50 * time.Millisecond}, sessions, work
}

func drain(t *testing.T, ch <-chan AgentEvent) []AgentEvent {
	t.Helper()
	var out []AgentEvent
	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, ev)
		case <-timeout:
			t.Fatal("event stream stalled")
		}
	}
}

func TestAdapterStartStreamsAndCompletes(t *testing.T) {
	runner := &ScriptedRunner{Scripts: []Script{{
		Stdout: `{"type":"text","timestamp":1,"sessionID":"ses-live-1","part":{"type":"text","text":"working"}}` + "\n" +
			`{"type":"tool_use","timestamp":2,"sessionID":"ses-live-1","part":{"tool":"bash","state":{"status":"completed"}}}` + "\n" +
			`{"type":"step_finish","timestamp":3,"sessionID":"ses-live-1"}` + "\n",
	}}}
	adapter, sessions, work := testAdapter(t, runner)
	ctx := context.Background()
	ref, ch, err := adapter.Start(ctx, BuildAssignment{BuildID: "b-oc1", WorkDir: work, Prompt: "do it", Turn: 1})
	if err != nil {
		t.Fatal(err)
	}
	if ref != "ses-live-1" {
		t.Fatalf("ref = %q", ref)
	}
	// The build→ref binding lives in the session record for later calls.
	events := drain(t, ch)
	kinds := []EventKind{}
	for _, ev := range events {
		kinds = append(kinds, ev.Kind)
	}
	want := []EventKind{EventMessage, EventProgress, EventProgress, EventDone}
	if len(kinds) != len(want) {
		t.Fatalf("kinds = %v", kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kinds = %v, want %v", kinds, want)
		}
	}
	if events[0].Text != "working" {
		t.Errorf("text = %q", events[0].Text)
	}
	rec, err := sessions.Get("b-oc1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.State != SessionDone {
		t.Errorf("record state = %q, want done", rec.State)
	}
	// Invocation shape: opencode run with format/session/dir flags.
	if len(runner.Calls) != 1 {
		t.Fatalf("calls = %d", len(runner.Calls))
	}
	call := runner.Calls[0]
	if call.Name != "opencode" || call.Dir != work {
		t.Errorf("call = %+v", call)
	}
	joined := strings.Join(call.Args, " ")
	for _, wantArg := range []string{"run", "--format", "json", "--dir", "do it", ".studio/inbox/question.json"} {
		if !strings.Contains(joined, wantArg) {
			t.Errorf("args lack %q: %q", wantArg, joined)
		}
	}
}

func TestAdapterQuestionDrop(t *testing.T) {
	runner := &ScriptedRunner{Scripts: []Script{{
		Stdout: `{"type":"text","timestamp":1,"sessionID":"ses-q-1","part":{"type":"text","text":"need input"}}` + "\n",
	}}}
	adapter, sessions, work := testAdapter(t, runner)
	ctx := context.Background()
	// The agent yields by dropping the question file as its last act.
	inbox := filepath.Join(work, inboxDir)
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	drop := `{"kind":"proposal","body":"Which plan?","options":["a","b"]}`
	// 0660 models the agent's pinned umask (group-readable via the
	// inbox group); the 0600 refusal is covered separately.
	if err := os.WriteFile(filepath.Join(inbox, questionFileName), []byte(drop), 0o660); err != nil {
		t.Fatal(err)
	}
	_, ch, err := adapter.Start(ctx, BuildAssignment{BuildID: "b-ocq", WorkDir: work, Prompt: "p", Turn: 2})
	if err != nil {
		t.Fatal(err)
	}
	events := drain(t, ch)
	last := events[len(events)-1]
	if last.Kind != EventQuestion || last.Question == nil {
		t.Fatalf("last = %+v, want question", last)
	}
	if last.Question.Kind != "proposal" || last.Question.Body != "Which plan?" {
		t.Errorf("drop = %+v", last.Question)
	}
	if len(last.Question.Options) != 2 {
		t.Errorf("options = %v", last.Question.Options)
	}
	// The drop is consumed exactly once.
	if _, err := os.Stat(filepath.Join(inbox, questionFileName)); !os.IsNotExist(err) {
		t.Error("question file not consumed")
	}
	rec, _ := sessions.Get("b-ocq")
	if rec.State != SessionYielded {
		t.Errorf("record state = %q, want yielded", rec.State)
	}
}

func TestAdapterFailureHonest(t *testing.T) {
	runner := &ScriptedRunner{Scripts: []Script{{
		Stdout:   `{"type":"error","timestamp":1,"sessionID":"ses-f-1","error":{"message":"model blew up"}}` + "\n",
		ExitCode: 1,
	}}}
	adapter, sessions, work := testAdapter(t, runner)
	ctx := context.Background()
	_, ch, err := adapter.Start(ctx, BuildAssignment{BuildID: "b-ocf", WorkDir: work, Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	events := drain(t, ch)
	if len(events) != 2 || events[0].Kind != EventProgress || events[1].Kind != EventFailed {
		t.Fatalf("events = %+v, want error-progress + terminal failed", events)
	}
	if !strings.Contains(events[0].Text, "model blew up") {
		t.Errorf("error text = %q", events[0].Text)
	}
	if events[0].ToolOK {
		t.Error("error progress marked ok, want not-ok")
	}
	rec, _ := sessions.Get("b-ocf")
	if rec.State != SessionFailed {
		t.Errorf("record state = %q, want failed", rec.State)
	}
}

func TestAdapterStartFailure(t *testing.T) {
	runner := &ScriptedRunner{}
	adapter, _, work := testAdapter(t, runner)
	_, _, err := adapter.Start(context.Background(), BuildAssignment{BuildID: "b-x", WorkDir: work, Prompt: "p"})
	if err == nil {
		t.Fatal("start with no script accepted, want error")
	}
	runner2 := &ScriptedRunner{Scripts: []Script{{Stdout: "not json\nstill not json\n"}}}
	adapter2, _, work2 := testAdapter(t, runner2)
	if _, _, err := adapter2.Start(context.Background(), BuildAssignment{BuildID: "b-y", WorkDir: work2, Prompt: "p"}); err == nil {
		t.Error("identity-less stream accepted, want refusal")
	}
}

func TestAdapterContinueAndCancel(t *testing.T) {
	first := `{"type":"text","timestamp":1,"sessionID":"ses-c-1","part":{"type":"text","text":"q?"}}` + "\n"
	second := `{"type":"text","timestamp":2,"sessionID":"ses-c-1","part":{"type":"text","text":"thanks"}}` + "\n"
	runner := &ScriptedRunner{Scripts: []Script{{Stdout: first}, {Stdout: second}}}
	adapter, _, work := testAdapter(t, runner)
	ctx := context.Background()
	ref, ch, err := adapter.Start(ctx, BuildAssignment{BuildID: "b-occ", WorkDir: work, Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	_ = drain(t, ch)
	ch2, err := adapter.Continue(ctx, ref, UserInput{QuestionID: "q", Kind: "clarification", Answer: "a", Turn: 1})
	if err != nil {
		t.Fatal(err)
	}
	// The answer file lands for the agent; the resume reuses the session.
	raw, err := os.ReadFile(filepath.Join(work, inboxDir, answerFileName))
	if err != nil {
		t.Fatalf("answer file: %v", err)
	}
	if !strings.Contains(string(raw), `"answer":"a"`) {
		t.Errorf("answer file = %s", raw)
	}
	if len(runner.Calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(runner.Calls))
	}
	resume := strings.Join(runner.Calls[1].Args, " ")
	if !strings.Contains(resume, "--session ses-c-1") || !strings.Contains(resume, "answer.json") {
		t.Errorf("resume args = %q", resume)
	}
	events := drain(t, ch2)
	if last := events[len(events)-1]; last.Kind != EventDone {
		t.Fatalf("last = %+v, want done", last)
	}
	// Unknown sessions fail every operation cleanly.
	for name, fn := range map[string]func() error{
		"continue": func() error { _, err := adapter.Continue(ctx, "nope", UserInput{}); return err },
		"status":   func() error { _, err := adapter.Status(ctx, "nope"); return err },
		"collect":  func() error { _, err := adapter.Collect(ctx, "nope", Manifest{}); return err },
	} {
		if err := fn(); err == nil {
			t.Errorf("%s on unknown session accepted", name)
		}
	}
	// Cancel on an unknown session is an idempotent no-op.
	if err := adapter.Cancel(ctx, "nope"); err != nil {
		t.Errorf("cancel unknown: %v", err)
	}
}

func TestAdapterCancelHungRun(t *testing.T) {
	runner := &ScriptedRunner{Scripts: []Script{{
		Stdout: `{"type":"text","timestamp":1,"sessionID":"ses-h-1","part":{"type":"text","text":"stuck"}}` + "\n",
		Hang:   true,
	}}}
	adapter, sessions, work := testAdapter(t, runner)
	ctx := context.Background()
	ref, ch, err := adapter.Start(ctx, BuildAssignment{BuildID: "b-och", WorkDir: work, Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.Cancel(ctx, ref); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	events := drain(t, ch)
	if last := events[len(events)-1]; last.Kind != EventFailed {
		t.Fatalf("last = %+v, want failed after kill", last)
	}
	_ = sessions
	_ = work
}

func TestAdapterCollect(t *testing.T) {
	runner := &ScriptedRunner{}
	adapter, sessions, work := testAdapter(t, runner)
	if err := os.WriteFile(work+"/out.diff", []byte("D"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Create("b-ocl", SessionRecord{SessionRef: "ses-l-1", BuildID: "b-ocl", WorkDir: work, State: SessionDone}); err != nil {
		t.Fatal(err)
	}
	arts, err := adapter.Collect(context.Background(), "ses-l-1",
		Manifest{Entries: []ManifestEntry{{Path: "out.diff", Kind: "diff"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 1 || arts[0].Digest == "" || string(arts[0].Data) != "D" {
		t.Fatalf("artifacts = %+v", arts)
	}
}

func TestAdapterStatus(t *testing.T) {
	runner := &ScriptedRunner{Scripts: []Script{{
		Stdout: `{"type":"wut","timestamp":1,"sessionID":"ses-st-1"}` + "\n{bad json\n",
		Hang:   true,
	}}}
	adapter, _, work := testAdapter(t, runner)
	ctx := context.Background()
	ref, ch, err := adapter.Start(ctx, BuildAssignment{BuildID: "b-ocs", WorkDir: work, Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	// Drain one event tick so the pump classifies the scripted lines, then
	// check status while the run is still live.
	_ = ch
	deadline := time.Now().Add(2 * time.Second)
	for {
		st, err := adapter.Status(ctx, ref)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(st.Detail, "skipped=") {
			if !st.Alive {
				t.Fatalf("status = %+v, want alive", st)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("status never observed the run: %+v", st)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := adapter.Cancel(ctx, ref); err != nil {
		t.Fatal(err)
	}
}

// TestAssignmentPromptStatesContract (§7): every assignment restates
// the Miniapp application contract — Go, Miniapp framework, MiniappSpec,
// Miniapp runtime — so the agent default is never another stack.
func TestAssignmentPromptStatesContract(t *testing.T) {
	got := assignmentPrompt(BuildAssignment{BuildID: "b", Prompt: "do it", Turn: 1})
	for _, want := range []string{"Miniapp exclusively", "Go", "Miniapp framework", "MiniappSpec", "Miniapp runtime", "no other stack"} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt lacks %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "[Studio turn 1]") || !strings.Contains(got, "do it") {
		t.Errorf("prompt lost turn/prompt identity:\n%s", got)
	}
}
