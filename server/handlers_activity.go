package server

import (
	"net/http"
)

// handleActivity renders tenant-scoped audit events with cursor pagination.
func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
	key, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	input := map[string]any{"limit": 50}
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		input["cursor"] = cursor
	}
	res, err := s.invokeTool(r.Context(), r, "activity.list", input, key)
	if err != nil {
		renderError(w, r, http.StatusBadGateway, "Activity unavailable.", s.logger, err.Error())
		return
	}
	renderTemplate(w, r, "activity.html", res, s.logger)
}
