package server

import (
	"net/http"
	"time"
)

// handleDeployPage renders validation state, eligibility, the deployment
// record, and provision/retire actions. While a deployment reports
// provisioning, the page meta-refreshes: genuine polling of async work.
func (s *Server) handleDeployPage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	key, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	app, err := s.invokeTool(r.Context(), r, "app.get", map[string]any{"app_id": id}, key)
	if err != nil {
		renderError(w, r, http.StatusNotFound, "Application not found.", s.logger, "")
		return
	}
	status := map[string]any{"status": "none"}
	if skey, err := newOpKey(); err == nil {
		if st, err := s.invokeTool(r.Context(), r, "deploy.status",
			map[string]any{"app_id": id}, skey); err == nil {
			status = st
		} else {
			status = map[string]any{"status": "unknown"}
		}
	}
	opkey, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	refresh := status["status"] == "provisioning"
	meta := "Deployment record"
	if st, _ := status["status"].(string); st != "" {
		meta = "Status: " + st
	}
	s.renderFocusPage(w, r, "scene_deploy.html", "apps", appsSpine("apps"), meta, map[string]any{
		"App": app, "Deploy": status, "OpKey": opkey,
		"FlyConfigured": s.sprites != nil, "Refresh": refresh,
	})
}

// handleDeployProvision runs one provision with a race: fast completions
// render inline, slow ones continue in the background behind a status
// interstitial. A second provision while one runs gets an honest conflict.
func (s *Server) handleDeployProvision(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	key, err := opKey(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	if r.Form.Get("confirm") != "yes" {
		renderError(w, r, http.StatusBadRequest, "Confirmation required.", s.logger, "")
		return
	}
	tenant, _, _, _ := TenantOf(r)
	if !s.markDeploy(tenant, id) {
		renderError(w, r, http.StatusConflict, "Provisioning already in progress.", s.logger, "")
		return
	}
	outcome := s.provisionAsync(tenant, id, key)
	select {
	case res := <-outcome:
		if res.err != nil {
			renderError(w, r, http.StatusBadGateway, friendlyToolError(res.err), s.logger, res.err.Error())
			return
		}
		s.renderFocusPage(w, r, "scene_deploy_result.html", "apps", appsSpine("apps"),
			"Provision outcome", map[string]any{
				"AppID": id, "Deploy": res.result,
			})
	case <-time.After(slowThreshold):
		w.WriteHeader(http.StatusAccepted)
		s.renderFocusPage(w, r, "scene_deploy_interstitial.html", "apps", appsSpine("apps"),
			"Provisioning", map[string]any{"AppID": id})
	}
}

// handleDeployRetire closes one deployment after explicit confirmation.
func (s *Server) handleDeployRetire(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	key, err := opKey(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	if r.Form.Get("confirm") != "yes" {
		renderError(w, r, http.StatusBadRequest, "Confirmation required.", s.logger, "")
		return
	}
	if _, err := s.invokeTool(r.Context(), r, "deploy.retire",
		map[string]any{"app_id": id}, key); err != nil {
		renderError(w, r, http.StatusBadRequest, friendlyToolError(err), s.logger, err.Error())
		return
	}
	http.Redirect(w, r, "/apps/"+id+"/deploy", http.StatusFound)
}
