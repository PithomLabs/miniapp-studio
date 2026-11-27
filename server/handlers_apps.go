package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// appSlug reads the slug out of an app.get result for display metadata.
func appSlug(app map[string]any) string {
	slug, _ := app["slug"].(string)
	if slug == "" {
		return "app"
	}
	return slug
}

// handleHome renders the tenant dashboard: recent apps plus recent activity.
// Each panel degrades independently: a backend failure shows an error panel,
// never a fabricated list.
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	tenant, _, _, _ := TenantOf(r)
	data := map[string]any{"Tenant": tenant}
	if key, err := newOpKey(); err == nil {
		apps, err := s.invokeTool(r.Context(), r, "app.list",
			map[string]any{"limit": 5, "offset": 0}, key)
		if err != nil {
			data["AppsError"] = "Application list unavailable."
		} else {
			data["Apps"] = apps["apps"]
		}
	} else {
		data["AppsError"] = "Application list unavailable."
	}
	if key, err := newOpKey(); err == nil {
		events, err := s.invokeTool(r.Context(), r, "activity.list",
			map[string]any{"limit": 5}, key)
		if err != nil {
			data["ActivityError"] = "Activity unavailable."
		} else {
			data["Events"] = events["events"]
		}
	} else {
		data["ActivityError"] = "Activity unavailable."
	}
	s.renderFocusPage(w, r, "scene_home.html", "home", nil, "Studio overview", data)
}

// handleAppsNew renders the Create App editor: a spec textarea, validation
// and preview actions, and an explicit P5 reservation for the conversational
// builder (never presented as implemented).
func (s *Server) handleAppsNew(w http.ResponseWriter, r *http.Request) {
	key, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	s.renderFocusPage(w, r, "scene_app_new.html", "apps", appsSpine("new"),
		"Register · Validate · Preview", map[string]any{"OpKey": key})
}

// handleAppsValidate runs spec.validate and renders actionable results.
func (s *Server) handleAppsValidate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	key, err := opKey(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	text := r.Form.Get("spec")
	var doc any
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		s.renderFocusPage(w, r, "scene_spec_result.html", "apps", appsSpine("new"),
			"Validation failed", map[string]any{
				"Spec": text, "valid": false,
				"errors": []map[string]any{{"code": "E_SCHEMA", "path": "$", "message": "invalid JSON"}},
			})
		return
	}
	res, err := s.invokeTool(r.Context(), r, "spec.validate", map[string]any{"spec": doc}, key)
	if err != nil {
		renderError(w, r, http.StatusBadGateway, "Validation unavailable.", s.logger, err.Error())
		return
	}
	res["Spec"] = text
	meta := "Validation failed"
	if valid, _ := res["valid"].(bool); valid {
		meta = "Validation passed"
	}
	s.renderFocusPage(w, r, "scene_spec_result.html", "apps", appsSpine("new"), meta, res)
}

// handleAppsPreview validates and dry-runs a spec draft against one of its
// own fixtures. Nothing persists and no provider is contacted.
func (s *Server) handleAppsPreview(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	key, err := opKey(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	text := r.Form.Get("spec")
	workflow, fixture := r.Form.Get("workflow"), r.Form.Get("fixture")
	if workflow == "" || fixture == "" {
		renderError(w, r, http.StatusBadRequest, "Workflow and fixture are required.", s.logger, "")
		return
	}
	var doc any
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		renderError(w, r, http.StatusBadRequest, "Spec is not valid JSON.", s.logger, "")
		return
	}
	validated, err := s.invokeTool(r.Context(), r, "spec.validate", map[string]any{"spec": doc}, key+"-validate")
	if err != nil {
		renderError(w, r, http.StatusBadGateway, "Validation unavailable.", s.logger, err.Error())
		return
	}
	if valid, _ := validated["valid"].(bool); !valid {
		validated["Spec"] = text
		s.renderFocusPage(w, r, "scene_spec_result.html", "apps", appsSpine("new"),
			"Validation failed", validated)
		return
	}
	tenant, _, _, _ := TenantOf(r)
	admitted, err := admitForTenant([]byte(text), tenant)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Spec rejected.", s.logger, "")
		return
	}
	res, err := dryRunForTenant(admitted, fixture, workflow)
	if err != nil {
		s.renderFocusPage(w, r, "scene_preview_result.html", "apps", appsSpine("new"),
			"Dry-run failed", map[string]any{
				"Spec": text, "DryError": err.Error(),
			})
		return
	}
	res["Spec"] = text
	// Marshal outputs to JSON: the raw map would render Go syntax.
	if out, ok := res["outputs"]; ok {
		if raw, err := json.MarshalIndent(out, "", "  "); err == nil {
			res["outputs"] = string(raw)
		}
	}
	s.renderFocusPage(w, r, "scene_preview_result.html", "apps", appsSpine("new"),
		"Dry-run preview", res)
}

// handleAppsCreate registers one application, then redirects to its page.
func (s *Server) handleAppsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	key, err := opKey(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	res, err := s.invokeTool(r.Context(), r, "app.create", map[string]any{
		"slug": r.Form.Get("slug"), "name": r.Form.Get("name"),
		"description": r.Form.Get("description"),
	}, key)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, friendlyToolError(err), s.logger, err.Error())
		return
	}
	id, _ := res["app_id"].(string)
	http.Redirect(w, r, "/apps/"+id, http.StatusFound)
}

// handleAppsList renders tenant applications with deployment status.
func (s *Server) handleAppsList(w http.ResponseWriter, r *http.Request) {
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
		if id != "" {
			if dkey, err := newOpKey(); err == nil {
				if st, err := s.invokeTool(r.Context(), r, "deploy.status",
					map[string]any{"app_id": id}, dkey); err == nil {
					status, _ = st["status"].(string)
				} else {
					status = "unknown"
				}
			}
		}
		app["deploy_status"] = status
		rows = append(rows, app)
	}
	fresh, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	s.renderFocusPage(w, r, "scene_apps.html", "apps", appsSpine("apps"),
		plural(len(rows), "application"),
		map[string]any{"Apps": rows, "OpKey": fresh})
}

// handleAppDetail renders one application: metadata, spec versions, and the
// deployment record with management forms.
func (s *Server) handleAppDetail(w http.ResponseWriter, r *http.Request) {
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
	versions := []any{}
	if current, _ := app["version"].(float64); current > 0 {
		for v := 1; v <= int(current); v++ {
			vkey, err := newOpKey()
			if err != nil {
				break
			}
			ver, err := s.invokeTool(r.Context(), r, "spec.get",
				map[string]any{"app_id": id, "version": v}, vkey)
			if err != nil {
				break
			}
			versions = append(versions, ver)
		}
	}
	deploy := map[string]any{"status": "none"}
	if dkey, err := newOpKey(); err == nil {
		if st, err := s.invokeTool(r.Context(), r, "deploy.status",
			map[string]any{"app_id": id}, dkey); err == nil {
			deploy = st
		}
	}
	opkey, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	s.renderFocusPage(w, r, "scene_app_detail.html", "apps", appsSpine("apps"),
		fmt.Sprintf("%s · %s", appSlug(app), plural(len(versions), "version")), map[string]any{
			"App": app, "Versions": versions, "Deploy": deploy, "OpKey": opkey,
		})
}

// handleAppSaveSpec stores one immutable spec version from pasted JSON.
func (s *Server) handleAppSaveSpec(w http.ResponseWriter, r *http.Request) {
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
	var doc any
	if err := json.Unmarshal([]byte(r.Form.Get("spec")), &doc); err != nil {
		renderError(w, r, http.StatusBadRequest, "Spec is not valid JSON.", s.logger, "")
		return
	}
	if _, err := s.invokeTool(r.Context(), r, "spec.save",
		map[string]any{"app_id": id, "spec": doc}, key); err != nil {
		renderError(w, r, http.StatusBadRequest, friendlyToolError(err), s.logger, err.Error())
		return
	}
	http.Redirect(w, r, "/apps/"+id, http.StatusFound)
}
