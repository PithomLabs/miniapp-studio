package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PithomLabs/miniapp-studio/tools"
)

// ssePollDefault is the live-update poll cadence. Each open stream polls the
// persisted conversation stream; there is no in-memory broadcaster, so a
// restarted Studio resumes every stream from its cursor with no lost or
// render-only state.
const ssePollDefault = time.Second

// ssePageSize bounds one poll page. Backlogs drain over successive polls.
const ssePageSize = 200

// handleConversationEvents streams one conversation as Server-Sent Events.
// The Last-Event-ID header (or ?after=) resumes from a persisted sequence;
// every emitted item comes from conversation.stream, so replay after
// browser reconnects and Studio restarts renders exactly the stored rows.
func (s *Server) handleConversationEvents(w http.ResponseWriter, r *http.Request) {
	convID := r.PathValue("id")
	cursor, err := sseCursor(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad event cursor.", s.logger, err.Error())
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, "streaming unsupported")
		return
	}
	// Ownership and the initial page come from one tool call: unknown or
	// foreign conversations fail here, before any bytes stream.
	first, err := s.streamPage(r.Context(), r, convID, cursor)
	if err != nil {
		if isNotFound(err) {
			renderError(w, r, http.StatusNotFound, "Conversation not found.", s.logger, "")
			return
		}
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	last := cursor
	if last, err = writeStreamItems(w, flusher, first); err != nil {
		return
	}
	poll := s.ssePoll
	if poll <= 0 {
		poll = ssePollDefault
	}
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	idle := 0
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			page, err := s.streamPage(r.Context(), r, convID, last)
			if err != nil {
				// Log and close: EventSource reconnects with its cursor
				// and replays from persisted rows.
				s.logger.Warn("sse poll failed", "conversation", convID, "detail", err.Error())
				return
			}
			items, _ := page["items"].([]any)
			if len(items) == 0 {
				idle++
				if idle >= 15 {
					_, _ = w.Write([]byte(": ping\n\n"))
					flusher.Flush()
					idle = 0
				}
				continue
			}
			idle = 0
			if last, err = writeStreamItems(w, flusher, page); err != nil {
				return
			}
		}
	}
}

// sseCursor reads the resume cursor: Last-Event-ID wins, then ?after=,
// defaulting to the stream head. Malformed cursors fail closed.
func sseCursor(r *http.Request) (int64, error) {
	raw := r.Header.Get("Last-Event-ID")
	if raw == "" {
		raw = r.URL.Query().Get("after")
	}
	if raw == "" {
		return 0, nil
	}
	seq, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || seq < 0 {
		return 0, fmt.Errorf("invalid cursor %q", raw)
	}
	return seq, nil
}

// streamPage fetches one page after a cursor through the conversation tool,
// so handlers never query stream tables directly.
func (s *Server) streamPage(ctx context.Context, r *http.Request, convID string, after int64) (map[string]any, error) {
	key, err := newOpKey()
	if err != nil {
		return nil, err
	}
	return s.invokeTool(ctx, r, tools.ToolConversationStream,
		map[string]any{"conversation_id": convID, "after_seq": after, "limit": ssePageSize}, key)
}

// writeStreamItems renders one page as SSE events, returning the last seq
// written. Item payloads are shaped for the browser; shaping never drops
// rows — an unshapable row would be a data bug, surfaced as an error.
func writeStreamItems(w http.ResponseWriter, flusher http.Flusher, page map[string]any) (int64, error) {
	items, _ := page["items"].([]any)
	latest, _ := page["latest_seq"].(float64)
	last := int64(latest)
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		data, seq, err := shapeStreamItem(m)
		if err != nil {
			return last, err
		}
		if _, err := fmt.Fprintf(w, "id: %d\nevent: update\ndata: %s\n\n", seq, data); err != nil {
			return last, err
		}
		last = seq
	}
	flusher.Flush()
	return last, nil
}

// shapeStreamItem normalizes one stream row for SSE/JSON: JSONB payloads
// arrive as bytes and are embedded parsed, never base64; NULL join misses
// stay null. It returns the item bytes and its sequence.
func shapeStreamItem(m map[string]any) ([]byte, int64, error) {
	seq, _ := m["seq"].(float64)
	out := map[string]any{
		"seq": seq, "kind": m["kind"], "ref_id": m["ref_id"], "build_id": m["build_id"],
		"role": m["role"], "body": m["body"], "type": m["type"],
	}
	switch payload := m["payload"].(type) {
	case []byte:
		var decoded any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			return nil, 0, fmt.Errorf("server: decode stream payload: %w", err)
		}
		out["payload"] = decoded
	case string:
		var decoded any
		if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
			return nil, 0, fmt.Errorf("server: decode stream payload: %w", err)
		}
		out["payload"] = decoded
	default:
		out["payload"] = m["payload"]
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, 0, fmt.Errorf("server: encode stream item: %w", err)
	}
	return data, int64(seq), nil
}

func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}
