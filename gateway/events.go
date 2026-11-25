package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LineOutcome classifies one NDJSON line. Unknown types and foreign
// sessions skip silently (forward robustness); corrupt lines skip loudly
// (the adapter counts them into status detail so a systematic shape
// change cannot blind the adapter unnoticed).
type LineOutcome int

const (
	LineEvent LineOutcome = iota
	LineSkip
	LineCorrupt
)

// TranslateLine maps one `opencode run --format json` output line to an
// AgentEvent. Shapes follow run.ts @ v1.19.6 (vendored audit record in
// docs/opencode-run-v1.19.6.md): every line is {"type", "timestamp",
// "sessionID", ...data}; known types are text, tool_use, step_start,
// step_finish, reasoning, and error. Part internals are read
// defensively — absent fields degrade the event, never fail it — and
// exact part shapes stay a P-G0 live-capture item.
func TranslateLine(line []byte, sessionID string) (AgentEvent, LineOutcome) {
	if len(bytes.TrimSpace(line)) == 0 {
		return AgentEvent{}, LineSkip
	}
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err != nil {
		return AgentEvent{}, LineCorrupt
	}
	if id, _ := raw["sessionID"].(string); id != "" && id != sessionID {
		return AgentEvent{}, LineSkip
	}
	typ, _ := raw["type"].(string)
	var at time.Time
	if ms, ok := raw["timestamp"].(float64); ok {
		at = time.UnixMilli(int64(ms))
	}
	part, _ := raw["part"].(map[string]any)
	if part != nil {
		if ps, _ := part["sessionID"].(string); ps != "" && ps != sessionID {
			return AgentEvent{}, LineSkip
		}
	}
	switch typ {
	case "text":
		text, _ := partText(part)
		if text == "" {
			return AgentEvent{}, LineSkip
		}
		return AgentEvent{Kind: EventMessage, Text: text, At: at, RawSession: sessionID}, LineEvent
	case "tool_use":
		name, _ := part["tool"].(string)
		state, _ := part["state"].(map[string]any)
		status, _ := state["status"].(string)
		ok := status == "completed"
		text := fmt.Sprintf("tool %s: %s", name, status)
		if detail := partDetail(part, state); detail != "" {
			text += ": " + detail
		}
		return AgentEvent{Kind: EventProgress, Text: text, Tool: name, ToolOK: ok, At: at, RawSession: sessionID}, LineEvent
	case "step_start":
		return AgentEvent{Kind: EventProgress, Text: "step started", At: at, RawSession: sessionID}, LineEvent
	case "step_finish":
		return AgentEvent{Kind: EventProgress, Text: "step finished", At: at, RawSession: sessionID}, LineEvent
	case "reasoning":
		text, _ := partText(part)
		if text == "" {
			return AgentEvent{}, LineSkip
		}
		return AgentEvent{Kind: EventProgress, Text: "thinking: " + text, At: at, RawSession: sessionID}, LineEvent
	case "error":
		// Mid-run error lines are diagnostic progress, not terminal
		// failure: a recoverable tool error must never terminal-fail
		// a build that subsequently exits cleanly. Terminality comes
		// from the process outcome (and question yield) only.
		return AgentEvent{Kind: EventProgress, Text: "agent error: " + errorText(raw["error"]),
			ToolOK: false, At: at, RawSession: sessionID}, LineEvent
	default:
		return AgentEvent{}, LineSkip
	}
}

func partText(part map[string]any) (string, bool) {
	if part == nil {
		return "", false
	}
	s, ok := part["text"].(string)
	if !ok {
		return "", false
	}
	// run.ts trims before emitting; whitespace-only parts carry nothing.
	s = strings.TrimSpace(s)
	return s, s != ""
}

// partDetail extracts a one-line tool summary from candidate fields. Part
// internals are unpinned (P-G0); every candidate is optional.
func partDetail(part, state map[string]any) string {
	if state != nil {
		if s, ok := state["error"].(string); ok && s != "" {
			return s
		}
		if s, ok := state["output"].(string); ok && s != "" {
			return truncate(s, 300)
		}
	}
	if part != nil {
		if s, ok := part["title"].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func errorText(v any) string {
	switch t := v.(type) {
	case string:
		if t != "" {
			return t
		}
	case map[string]any:
		if m, ok := t["message"].(string); ok && m != "" {
			return m
		}
		if n, ok := t["name"].(string); ok && n != "" {
			return n
		}
	}
	return "agent error"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
