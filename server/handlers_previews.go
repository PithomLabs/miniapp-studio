package server

import (
	"fmt"
	"net/http"
)

// handlePreviews renders the preview environments area: the three preview
// kinds with honest maturity, plus the live deployment records that exist
// today. Dry-run fixture previews are functional; Workspace Preview and
// per-PR Application Preview are Planned (preview controller is T3-gated)
// and every upcoming control is disabled with its reason. Engineering PR
// Preview and customer Application Preview are never conflated: they are
// separate sections with separate lifecycles.
func (s *Server) handlePreviews(w http.ResponseWriter, r *http.Request) {
	key, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	res, err := s.invokeTool(r.Context(), r, "app.list", map[string]any{"limit": 50}, key)
	if err != nil {
		renderError(w, r, http.StatusBadGateway, "Application list unavailable.", s.logger, err.Error())
		return
	}
	apps, _ := res["apps"].([]any)
	rows := make([]map[string]any, 0, len(apps))
	for _, item := range apps {
		app, _ := item.(map[string]any)
		id, _ := app["id"].(string)
		status := "none"
		var url string
		if id != "" {
			if dkey, err := newOpKey(); err == nil {
				if st, err := s.invokeTool(r.Context(), r, "deploy.status",
					map[string]any{"app_id": id}, dkey); err == nil {
					status, _ = st["status"].(string)
					url, _ = st["url"].(string)
				} else {
					status = "unknown"
				}
			}
		}
		rows = append(rows, map[string]any{
			"id": id, "name": app["name"], "slug": app["slug"],
			"version": app["version"], "deploy_status": status, "deploy_url": url,
		})
	}
	s.renderFocusPage(w, r, "scene_previews.html", "previews", previewsSpine("environments"),
		fmt.Sprintf("%d applications", len(rows)), map[string]any{"Apps": rows})
}

// previewsSpine is the Previews area lifecycle: current environments plus
// the two planned preview tracks. Planned pills link to the same page's
// sections — navigation to an honest statement, never to a fake control.
func previewsSpine(active string) []spinePill {
	pills := []spinePill{
		{Label: "Environments", Href: "/previews#environments", Title: "Current deployment records"},
		{Label: "Workspace", Href: "/previews#workspace", Title: "Planned: Builder workspace preview"},
		{Label: "Application", Href: "/previews#application", Title: "Planned: per-PR application preview (T3-gated)"},
	}
	for i := range pills {
		state := "idle"
		if active == "environments" && i == 0 {
			state = "active"
		}
		pills[i].State = state
	}
	return pills
}
