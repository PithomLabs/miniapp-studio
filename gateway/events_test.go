package gateway

import (
	"strings"
	"testing"
)

func TestTranslateLine(t *testing.T) {
	const sess = "ses_abc123"
	cases := []struct {
		name     string
		line     string
		outcome  LineOutcome
		kind     EventKind
		contains string
	}{
		{"text", `{"type":"text","timestamp":1700000000000,"sessionID":"ses_abc123","part":{"type":"text","text":"hello there"}}`,
			LineEvent, EventMessage, "hello there"},
		{"empty text skips", `{"type":"text","timestamp":1,"sessionID":"ses_abc123","part":{"type":"text","text":"  "}}`,
			LineSkip, "", ""},
		{"tool completed", `{"type":"tool_use","timestamp":1,"sessionID":"ses_abc123","part":{"tool":"bash","state":{"status":"completed","output":"ok"}}}`,
			LineEvent, EventProgress, "tool bash: completed"},
		{"tool error", `{"type":"tool_use","timestamp":1,"sessionID":"ses_abc123","part":{"tool":"edit","state":{"status":"error","error":"no such file"}}}`,
			LineEvent, EventProgress, "no such file"},
		{"tool sparse part", `{"type":"tool_use","timestamp":1,"sessionID":"ses_abc123","part":{"tool":"read"}}`,
			LineEvent, EventProgress, "tool read: "},
		{"step start", `{"type":"step_start","timestamp":1,"sessionID":"ses_abc123"}`, LineEvent, EventProgress, "step started"},
		{"step finish", `{"type":"step_finish","timestamp":1,"sessionID":"ses_abc123"}`, LineEvent, EventProgress, "step finished"},
		{"reasoning", `{"type":"reasoning","timestamp":1,"sessionID":"ses_abc123","part":{"text":"hmm"}}`,
			LineEvent, EventProgress, "thinking: hmm"},
		{"error string", `{"type":"error","timestamp":1,"sessionID":"ses_abc123","error":"boom"}`,
			LineEvent, EventProgress, "boom"},
		{"error object", `{"type":"error","timestamp":1,"sessionID":"ses_abc123","error":{"name":"E","message":"bad"}}`,
			LineEvent, EventProgress, "bad"},
		{"unknown type", `{"type":"future_thing","timestamp":1,"sessionID":"ses_abc123"}`, LineSkip, "", ""},
		{"foreign session", `{"type":"text","timestamp":1,"sessionID":"ses_other","part":{"type":"text","text":"x"}}`,
			LineSkip, "", ""},
		{"foreign part session", `{"type":"text","timestamp":1,"sessionID":"ses_abc123","part":{"type":"text","text":"x","sessionID":"ses_sub"}}`,
			LineSkip, "", ""},
		{"blank", `   `, LineSkip, "", ""},
		{"corrupt", `{oops`, LineCorrupt, "", ""},
		{"truncated", `{"type":"text",`, LineCorrupt, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev, out := TranslateLine([]byte(tc.line), sess)
			if out != tc.outcome {
				t.Fatalf("outcome = %v, want %v", out, tc.outcome)
			}
			if out != LineEvent {
				return
			}
			if ev.Kind != tc.kind || !strings.Contains(ev.Text, tc.contains) {
				t.Errorf("event = %+v, want kind %s containing %q", ev, tc.kind, tc.contains)
			}
			if ev.RawSession != sess {
				t.Errorf("session = %q", ev.RawSession)
			}
		})
	}
}
