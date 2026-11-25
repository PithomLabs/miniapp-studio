package gateway

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func kindsOf(events []AgentEvent) []string {
	var out []string
	for _, ev := range events {
		out = append(out, string(ev.Kind))
	}
	return out
}

// TestAdapterErrorThenCleanExit (F5): a recoverable mid-run error line
// followed by a clean exit is progress-then-success — never a terminal
// failure followed by a confusing success.
func TestAdapterErrorThenCleanExit(t *testing.T) {
	runner := &ScriptedRunner{Scripts: []Script{{
		Stdout: `{"type":"error","timestamp":1,"sessionID":"ses-t5","error":{"message":"flaky tool"}}` + "\n" +
			`{"type":"text","timestamp":2,"sessionID":"ses-t5","part":{"type":"text","text":"recovered"}}` + "\n",
		ExitCode: 0,
	}}}
	adapter, sessions, work := testAdapter(t, runner)
	_, ch, err := adapter.Start(context.Background(), BuildAssignment{BuildID: "b-t5", WorkDir: work, Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	events := drain(t, ch)
	if got := kindsOf(events); len(got) != 3 || got[0] != string(EventProgress) ||
		got[1] != string(EventMessage) || got[2] != string(EventDone) {
		t.Fatalf("kinds = %v, want [progress message done]", got)
	}
	if !strings.Contains(events[0].Text, "flaky tool") {
		t.Errorf("progress text = %q, want the diagnostic detail", events[0].Text)
	}
	if events[0].ToolOK {
		t.Error("error progress marked ok, want not-ok")
	}
	rec, _ := sessions.Get("b-t5")
	if rec.State != SessionDone {
		t.Errorf("record state = %q, want done", rec.State)
	}
}

// TestAdapterErrorThenFailure (F5): when the process genuinely fails,
// the terminal failure comes from the exit code — with the earlier
// error line preserved as diagnostic progress, not a premature
// terminal.
func TestAdapterErrorThenFailure(t *testing.T) {
	runner := &ScriptedRunner{Scripts: []Script{{
		Stdout:   `{"type":"error","timestamp":1,"sessionID":"ses-t5f","error":"bad"}` + "\n",
		ExitCode: 1,
	}}}
	adapter, sessions, work := testAdapter(t, runner)
	_, ch, err := adapter.Start(context.Background(), BuildAssignment{BuildID: "b-t5f", WorkDir: work, Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	events := drain(t, ch)
	if got := kindsOf(events); len(got) != 2 || got[0] != string(EventProgress) ||
		got[1] != string(EventFailed) {
		t.Fatalf("kinds = %v, want [progress failed]", got)
	}
	rec, _ := sessions.Get("b-t5f")
	if rec.State != SessionFailed {
		t.Errorf("record state = %q, want failed", rec.State)
	}
}

// TestAdapterQuestionYieldBeatsCleanExit (F5): a yielded question with a
// clean exit reports the yield — never misreported as completion.
func TestAdapterQuestionYieldBeatsCleanExit(t *testing.T) {
	runner := &ScriptedRunner{Scripts: []Script{{
		Stdout:   okTranscript("ses-t5q"),
		ExitCode: 0,
	}}}
	adapter, sessions, work := testAdapter(t, runner)
	if err := adapter.ensureInbox(work); err != nil {
		t.Fatal(err)
	}
	dropPath := work + "/" + inboxDir + "/" + questionFileName
	if err := os.WriteFile(dropPath, []byte(`{"kind":"clarification","body":"Which?"}`), 0o660); err != nil {
		t.Fatal(err)
	}
	_, ch, err := adapter.Start(context.Background(), BuildAssignment{BuildID: "b-t5q", WorkDir: work, Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	events := drain(t, ch)
	last := events[len(events)-1]
	if last.Kind != EventQuestion {
		t.Fatalf("last = %+v, want question (not done)", last)
	}
	rec, _ := sessions.Get("b-t5q")
	if rec.State != SessionYielded {
		t.Errorf("record state = %q, want yielded", rec.State)
	}
}

// TestAdapterMalformedStreamStillCompletes (F5): corrupt lines never
// terminalize a run by themselves — the exit code decides.
func TestAdapterMalformedStreamStillCompletes(t *testing.T) {
	runner := &ScriptedRunner{Scripts: []Script{{
		Stdout:   "not json at all\n{\"type\":\"text\",\n",
		ExitCode: 0,
	}}}
	adapter, sessions, work := testAdapter(t, runner)
	// No session identity in the stream: launch itself must refuse.
	if _, _, err := adapter.Start(context.Background(), BuildAssignment{BuildID: "b-t5m", WorkDir: work, Prompt: "p"}); err == nil {
		t.Fatal("identity-less stream accepted, want refusal")
	} else if _, err := sessions.Get("b-t5m"); err == nil {
		t.Error("failed launch left a session record")
	}
}

// TestAdapterKilledRunFails (F5): a signal-killed run (nonzero/negative
// outcome) stays an honest failure.
func TestAdapterKilledRunFails(t *testing.T) {
	runner := &ScriptedRunner{Scripts: []Script{{
		Stdout:   okTranscript("ses-t5k"),
		ExitCode: -1,
	}}}
	adapter, sessions, work := testAdapter(t, runner)
	_, ch, err := adapter.Start(context.Background(), BuildAssignment{BuildID: "b-t5k", WorkDir: work, Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	events := drain(t, ch)
	if last := events[len(events)-1]; last.Kind != EventFailed {
		t.Fatalf("last = %+v, want failed", last)
	}
	rec, _ := sessions.Get("b-t5k")
	if rec.State != SessionFailed {
		t.Errorf("record state = %q, want failed", rec.State)
	}
}

// TestAdapterOverlongLineSurfaced (M9): a line past the scanner limit
// must surface as observed truncation (corrupt tally + status detail),
// never silently end the event stream.
func TestAdapterOverlongLineSurfaced(t *testing.T) {
	huge := strings.Repeat("x", 5<<20)
	line1 := `{"type":"text","timestamp":1,"sessionID":"ses-t9","part":{"type":"text","text":"first"}}` + "\n"
	line2 := `{"type":"text","timestamp":2,"sessionID":"ses-t9","part":{"type":"text","text":"` + huge + `"}}` + "\n"
	runner := &ScriptedRunner{Scripts: []Script{{Stdout: line1 + line2, Hang: true}}}
	adapter, _, work := testAdapter(t, runner)
	ctx := context.Background()
	ref, ch, err := adapter.Start(ctx, BuildAssignment{BuildID: "b-t9", WorkDir: work, Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	// The pump scans line 2, hits the limit, then blocks in Wait on the
	// hung proc: Status taken now must show the truncation.
	deadline := time.Now().Add(5 * time.Second)
	var detail string
	for {
		st, serr := adapter.Status(ctx, ref)
		if serr != nil {
			t.Fatal(serr)
		}
		detail = st.Detail
		if strings.Contains(detail, "corrupt=1") || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(detail, "corrupt=1") {
		t.Errorf("status detail = %q, want corrupt=1", detail)
	}
	if !strings.Contains(detail, "truncated") {
		t.Errorf("status detail = %q, want truncation flagged", detail)
	}
	if err := adapter.Cancel(ctx, ref); err != nil {
		t.Fatal(err)
	}
	_ = drain(t, ch)
}
