package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/PithomLabs/miniapp-studio/gateway"
	"github.com/PithomLabs/miniapp-studio/tools"
)

// maxCallbackBytes caps gateway callback bodies. Envelopes carry event
// metadata; artifact bytes move through fetch_artifact, never callbacks.
const maxCallbackBytes = 1 << 20

// handleGatewayCallback ingests one signed gateway event. Authentication is
// the HMAC envelope — this route takes no operator session and no Origin
// check (the gateway is not a browser and holds no cookies). Verified
// envelopes flow into build.ingest_event, which owns all transitions;
// this handler authenticates, routes, and maps outcomes to statuses.
func (s *Server) handleGatewayCallback(w http.ResponseWriter, r *http.Request) {
	if s.gwKeys == nil {
		writeCallback(w, http.StatusServiceUnavailable, "gateway callbacks not configured")
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxCallbackBytes))
	if err != nil {
		writeCallback(w, http.StatusBadRequest, "unreadable body")
		return
	}
	env, err := gateway.Parse(raw)
	if err != nil {
		writeCallback(w, http.StatusBadRequest, "malformed envelope")
		return
	}
	key, err := s.gwKeys.KeyFor(env.BuildID)
	if err != nil {
		// Unknown build: 404 without distinguishing it from an unknown
		// tenant or a bad key — no oracle for scanners.
		writeCallback(w, http.StatusNotFound, "unknown build")
		return
	}
	if err := gateway.Verify(key, env); err != nil {
		s.logger.Warn("gateway callback rejected", "build", env.BuildID,
			"message", env.MessageID, "name", env.Name, "detail", err.Error())
		writeCallback(w, http.StatusUnauthorized, "invalid callback")
		return
	}
	if _, ok := s.FrameworkCredential(env.Workspace); !ok {
		writeCallback(w, http.StatusNotFound, "unknown build")
		return
	}
	seq := 0
	if f, ok := env.Payload["seq"].(float64); ok && f >= 0 {
		seq = int(f)
	}
	// Fresh key per HTTP attempt, deliberately NOT derived from the message
	// id: the framework records failures against their key, so a keyed
	// redelivery after a transient failure would replay the recorded
	// failure instead of retrying. Exactly-once effects come from the
	// (tenant, build, event_id) dedup inside build.ingest_event; the
	// framework key only needs per-attempt uniqueness.
	attemptKey, err := newOpKey()
	if err != nil {
		writeCallback(w, http.StatusInternalServerError, "ingest failed")
		return
	}
	out, err := s.invokeToolTenant(r.Context(), env.Workspace, tools.ToolBuildIngestEvent,
		map[string]any{
			"build_id": env.BuildID, "event_id": env.MessageID,
			"type": env.Name, "seq": seq, "payload": env.Payload,
		}, attemptKey)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "not found"):
			writeCallback(w, http.StatusNotFound, "unknown build")
		case strings.Contains(msg, "unavailable"):
			// Retryable: the gateway redelivers with the same id.
			writeCallback(w, http.StatusServiceUnavailable, "temporarily unavailable")
		case strings.Contains(msg, "already recorded"):
			// Deterministic gateway-side bug (a question id reused
			// under a fresh event id): retrying cannot help, so fail
			// terminally instead of burning the outbox retry budget.
			writeCallback(w, http.StatusBadRequest, "bad gateway event")
		default:
			s.logger.Error("callback ingest failed", "build", env.BuildID,
				"message", env.MessageID, "detail", msg)
			writeCallback(w, http.StatusInternalServerError, "ingest failed")
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ack": true, "duplicate": out["duplicate"],
		"transitioned": out["transitioned"], "build_status": out["build_status"],
	})
}

func writeCallback(w http.ResponseWriter, code int, public string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": public})
}
