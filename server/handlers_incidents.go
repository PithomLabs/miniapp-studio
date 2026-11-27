package server

import (
	"net/http"
	"strings"
)

// failureOutcomes is the audit-outcome failure set: exact framework
// rejection literals (miniapp runtime/service audit call sites:
// failed, invalid-input, invalid-entity, forbidden, key-reuse, denied,
// expired) plus substring fallback for studio-composed strings
// ("verification failed", "conformance failed", decisions). Deliberately
// excludes cancelled/pending/succeeded: not failures. Pure, so the
// incidents page and its unit test share one definition.
func isFailureOutcome(outcome string) bool {
	switch outcome {
	case "failed", "invalid-input", "invalid-entity", "forbidden",
		"key-reuse", "denied", "expired":
		return true
	}
	o := strings.ToLower(outcome)
	return strings.Contains(o, "fail") || strings.Contains(o, "reject") ||
		strings.Contains(o, "block") || strings.Contains(o, "error") ||
		strings.Contains(o, "denied") || strings.Contains(o, "expir")
}

// handleIncidents renders the B4 troubleshooting area. The activity trail
// is real audit data; incident management itself (incident list, RCA
// workflow, recovery execution) is Planned and every upcoming control is
// disabled with its reason. The recent-failures section derives from
// audit outcomes in-handler — visibly audit-derived, never presented as
// an incident system. Transport polls are excluded in-query (same as
// console evidence) so the window counts kept rows, not noise.
func (s *Server) handleIncidents(w http.ResponseWriter, r *http.Request) {
	key, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	input := map[string]any{"limit": 50, "exclude": []string{"conversation.stream"}}
	if tool := strings.TrimSpace(r.URL.Query().Get("tool")); tool != "" {
		input["tool"] = tool
	}
	res, err := s.invokeTool(r.Context(), r, "activity.list", input, key)
	if err != nil {
		renderError(w, r, http.StatusBadGateway, "Activity unavailable.", s.logger, err.Error())
		return
	}
	events, _ := res["events"].([]any)
	failed := []any{}
	for _, e := range events {
		if m, ok := e.(map[string]any); ok {
			if outcome, _ := m["outcome"].(string); isFailureOutcome(outcome) {
				failed = append(failed, m)
			}
		}
	}
	s.renderFocusPage(w, r, "scene_incidents.html", "incidents", incidentsSpine("failures"),
		"Troubleshooting", map[string]any{
			"Failed": failed, "Total": len(events),
			"ToolFilter": strings.TrimSpace(r.URL.Query().Get("tool")),
		})
}

// incidentsSpine is the Incidents area lifecycle: audit-derived failures
// today, managed incidents when the capability lands.
func incidentsSpine(active string) []spinePill {
	pills := []spinePill{
		{Label: "Failures", Href: "/incidents#failures", Title: "Recent failed audit outcomes"},
		{Label: "Managed", Href: "/incidents#managed", Title: "Planned: incident list, RCA, recovery"},
	}
	for i := range pills {
		state := "idle"
		if active == "failures" && i == 0 {
			state = "active"
		}
		pills[i].State = state
	}
	return pills
}
