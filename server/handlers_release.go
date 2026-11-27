package server

import (
	"net/http"
)

// handleRelease renders the B5 staging/production area. Release promotion,
// rollback execution, and release history are Planned (Phase E) and every
// upcoming control is disabled with its reason. The candidate section reads
// real artifact ledger state for one chosen conversation; without a
// conversation selected it says so and links to the list.
func (s *Server) handleRelease(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	if convID := r.URL.Query().Get("conversation"); convID != "" {
		status, err := s.consoleArtifactStatus(r, convID)
		if err != nil {
			// Mirror the console's mapping: unknown conversation is
			// 404, malformed input stays 400. Same within-tenant
			// existence exposure as the console — accepted there.
			if isNotFound(err) {
				renderError(w, r, http.StatusNotFound, "Conversation not found.", s.logger, "")
				return
			}
			renderError(w, r, http.StatusBadRequest, "Release candidate unavailable for that conversation.", s.logger, err.Error())
			return
		}
		data["ConversationID"] = convID
		data["Artifacts"], _ = status["artifacts"].([]any)
	}
	s.renderFocusPage(w, r, "scene_release.html", "release", releaseSpine("pipeline"),
		"Staging & production", data)
}

// releaseSpine is the Release area lifecycle: the environment pipeline,
// the per-conversation candidate, and the planned history.
func releaseSpine(active string) []spinePill {
	pills := []spinePill{
		{Label: "Pipeline", Href: "/release#pipeline", Title: "Environment pipeline model"},
		{Label: "Candidate", Href: "/release#candidate", Title: "Release candidate per conversation"},
		{Label: "History", Href: "/release#history", Title: "Planned: promotion and rollback record"},
	}
	for i := range pills {
		state := "idle"
		if active == "pipeline" && i == 0 {
			state = "active"
		}
		pills[i].State = state
	}
	return pills
}
