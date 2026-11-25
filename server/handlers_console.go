package server

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/PithomLabs/miniapp-studio/tools"
)

// Focus Console handlers (C6). Server-rendered full pages per scene; the
// embedded vanilla JS only appends live stream items and flips display
// state. Every pill, card, and badge derives from tool output — nothing
// is inferred from browser state, timers, or dispatched requests.

var focusScenes = map[string]string{
	"gallery":  "scene_gallery.html",
	"spec":     "scene_placeholder.html",
	"validate": "scene_placeholder.html",
	"connect":  "scene_placeholder.html",
	"deploy":   "scene_placeholder.html",
	"run":      "scene_run.html",
	"evidence": "scene_evidence.html",
	"settings": "scene_settings.html",
}

var focusPlaceholders = map[string]map[string]string{
	"spec":     {"Title": "Spec", "Sub": "The canonical MiniappSpec for the focused application.", "Phase": "Phase D"},
	"validate": {"Title": "Validate", "Sub": "Real validation runs with fixture and refusal evidence.", "Phase": "Phase D"},
	"connect":  {"Title": "Connect", "Sub": "Connector catalog, organization configuration, scoped authorization.", "Phase": "Phase D"},
	"deploy":   {"Title": "Deploy", "Sub": "Approved artifact promotion into a separate Runtime Sprite.", "Phase": "Phase E"},
}

// spinePill is one lifecycle pill: navigation only, never permission to
// advance anything. State is one of idle/active/proven/refused.
type spinePill struct {
	Label, State, Title, Href string
}

// buildSpine maps persisted conversation/build state onto the seven
// lifecycle pills. Green means proven by persisted evidence; anything
// without a backend capability stays gray with an explicit note.
// Guardrail: the Run pill NEVER goes green from build.status — a
// working Builder is not a deployed product. Product Run stays
// explicitly unproven until Phase E wires Runtime evidence; builder
// activity shows amber (live) or red (failed/cancelled) with a note.
func buildSpine(convID string, builds []any) []spinePill {
	q := ""
	if convID != "" {
		q = "&conversation=" + convID
	}
	href := func(scene string) string { return "/console?scene=" + scene + q }
	pills := []spinePill{
		{Label: "Describe", State: "idle", Title: "No conversation in focus", Href: href("gallery")},
		{Label: "Spec", State: "idle", Title: "Lands in Phase D", Href: href("spec")},
		{Label: "Validate", State: "idle", Title: "Lands in Phase D", Href: href("validate")},
		{Label: "Connect", State: "idle", Title: "Lands in Phase D", Href: href("connect")},
		{Label: "Deploy", State: "idle", Title: "Lands in Phase E", Href: href("deploy")},
		{Label: "Run", State: "idle", Title: "No build in focus", Href: href("run")},
		{Label: "Evidence", State: "idle", Title: "Gate ledger lands in Phase D; receipts below are live", Href: href("evidence")},
	}
	if convID == "" {
		return pills
	}
	if len(builds) == 0 {
		pills[0].State, pills[0].Title = "active", "Intent captured; no build yet"
		return pills
	}
	pills[0].State, pills[0].Title = "proven", "Conversation and build exist"
	focus, _ := builds[len(builds)-1].(map[string]any)
	status, _ := focus["status"].(string)
	switch status {
	case "assigned", "running", "waiting_user":
		pills[5].State = "active"
		pills[5].Title = "Builder " + status + " — product Run unproven until Phase E"
	case "failed", "cancelled":
		pills[5].State = "refused"
		pills[5].Title = "Builder run ended (" + status + ") — product Run unproven until Phase E"
	default: // succeeded and anything future: builder done, product unproven
		pills[5].Title = "Builder finished — product Run unproven until Phase E"
	}
	return pills
}

// pulseItem is one server-rendered Pulse card. Time carries the stream
// cursor ("seq N"): the stream tool returns no timestamps, and the UI
// must not invent clock times.
type pulseItem struct {
	Seq, Class, Kind, Title, Detail, DevDetail, Time string
}

// shapePulseItems renders stream page items as Pulse cards. Unknown
// future kinds (dispatch terminals in U3) fall through to event cards
// rather than dropping rows.
func shapePulseItems(items []any) []pulseItem {
	out := make([]pulseItem, 0, len(items))
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		seq := ""
		if f, ok := m["seq"].(float64); ok {
			seq = fmt.Sprintf("%.0f", f)
		}
		kind, _ := m["kind"].(string)
		if kind == "message" {
			role, _ := m["role"].(string)
			body, _ := m["body"].(string)
			ref, _ := m["ref_id"].(string)
			out = append(out, pulseItem{Seq: seq, Class: "event", Kind: "event",
				Title: role, Detail: body, DevDetail: "seq " + seq + " · ref " + ref,
				Time: "seq " + seq})
			continue
		}
		if kind == "dispatch" {
			// Terminal settlements only — the sender streams acks
			// and failures, never queued waits. Acks read as
			// receipts, failures as alerts; the filter key matches.
			pm, _ := m["payload"].(map[string]any)
			cmd, _ := pm["command_kind"].(string)
			state, _ := pm["state"].(string)
			reason, _ := pm["reason"].(string)
			class, filter := "receipt", "receipt"
			if state != "acked" {
				class, filter = "alert", "alert"
			}
			attempts := ""
			if f, ok := pm["attempts"].(float64); ok {
				attempts = fmt.Sprintf(" · attempts %.0f", f)
			}
			build, _ := m["build_id"].(string)
			out = append(out, pulseItem{Seq: seq, Class: class, Kind: filter,
				Title: "dispatch " + cmd + " " + state, Detail: reason,
				DevDetail: "seq " + seq + " · build " + build + attempts,
				Time:      "seq " + seq})
			continue
		}
		typ, _ := m["type"].(string)
		detail := ""
		if pm, ok := m["payload"].(map[string]any); ok {
			if t, _ := pm["text"].(string); t != "" {
				detail = t
			} else {
				detail, _ = pm["body"].(string)
			}
		}
		build, _ := m["build_id"].(string)
		out = append(out, pulseItem{Seq: seq, Class: "event", Kind: "event",
			Title: typ, Detail: detail, DevDetail: "seq " + seq + " · build " + build,
			Time: "seq " + seq})
	}
	return out
}

func (s *Server) renderFocus(w http.ResponseWriter, r *http.Request, sceneFile string, data map[string]any) {
	tpl, err := template.New("focus").ParseFS(templateFiles,
		"templates/focus/layout.html", "templates/focus/"+sceneFile)
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "focus-layout", data); err != nil {
		// Headers are already committed; the truncated page is the
		// honest signal, and the detail stays server-side.
		s.logger.Error("focus render failed", "scene", sceneFile, "detail", err.Error())
	}
}

// handleConsole renders the Focus Console shell for one scene. Without a
// focused conversation only the gallery renders; anything else redirects
// there — bookmarkable states stay meaningful, never half-focused.
func (s *Server) handleConsole(w http.ResponseWriter, r *http.Request) {
	scene := r.URL.Query().Get("scene")
	if scene == "" {
		scene = "gallery"
	}
	sceneFile, ok := focusScenes[scene]
	if !ok {
		renderError(w, r, http.StatusNotFound, "Unknown console scene.", s.logger, "")
		return
	}
	convID := r.URL.Query().Get("conversation")
	tenant, binding, _, _ := TenantOf(r)
	data := map[string]any{"Scene": scene, "Tenant": tenant, "Binding": binding}
	if convID == "" {
		// Gallery, evidence, and settings are tenant-scoped; the
		// rest need a focused conversation to mean anything.
		if scene != "gallery" && scene != "evidence" && scene != "settings" {
			http.Redirect(w, r, "/console?scene=gallery", http.StatusFound)
			return
		}
		s.renderConsoleUnfocused(w, r, data, scene)
		return
	}
	got, err := s.consoleFocus(r, convID)
	if err != nil {
		if isNotFound(err) {
			renderError(w, r, http.StatusNotFound, "Conversation not found.", s.logger, "")
			return
		}
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	data["Conversation"], _ = got["conversation"].(map[string]any)
	builds, _ := got["builds"].([]any)
	data["Builds"] = builds
	var focus map[string]any
	if len(builds) > 0 {
		focus, _ = builds[len(builds)-1].(map[string]any)
		data["FocusBuild"] = focus
	}
	data["Spine"] = buildSpine(convID, builds)
	if scene == "gallery" {
		s.renderConsoleGallery(w, r, data, convID)
		return
	}
	s.renderConsoleScene(w, r, data, scene, sceneFile, convID, focus)
}

// renderConsoleUnfocused renders a tenant-scoped scene with no focused
// conversation: empty Pulse and Line, live tenant data otherwise.
func (s *Server) renderConsoleUnfocused(w http.ResponseWriter, r *http.Request, data map[string]any, scene string) {
	data["Spine"] = buildSpine("", nil)
	data["MetaLeft"] = template.HTML("No conversation in focus")
	if t, _ := data["Tenant"].(string); t != "" {
		data["MetaRight"] = template.HTML(`<span class="tag mut">tenant ` + template.HTMLEscapeString(t) + `</span>`)
	} else {
		data["MetaRight"] = template.HTML("")
	}
	data["PulseItems"] = []pulseItem{}
	data["PendingApprovals"] = 0
	data["Approvals"] = []map[string]any{}
	if notice := r.URL.Query().Get("notice"); notice != "" {
		data["Notice"] = notice
	}
	if key, err := newOpKey(); err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	} else {
		data["KillOpKey"] = key
	}
	switch scene {
	case "gallery":
		key, err := newOpKey()
		if err != nil {
			renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
			return
		}
		out, err := s.invokeTool(r.Context(), r, tools.ToolConversationList,
			map[string]any{"limit": 20}, key)
		if err != nil {
			renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
			return
		}
		data["Conversations"], _ = out["conversations"].([]any)
		if key, err := newOpKey(); err != nil {
			renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
			return
		} else {
			data["GalleryOpKey"] = key
		}
		s.renderFocus(w, r, "scene_gallery.html", data)
	case "evidence":
		key, err := newOpKey()
		if err != nil {
			renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
			return
		}
		out, err := s.invokeTool(r.Context(), r, tools.ToolActivityList,
			map[string]any{"limit": 50, "exclude": []string{"conversation.stream"}}, key)
		if err != nil {
			renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
			return
		}
		data["Activity"], _ = out["events"].([]any)
		s.renderFocus(w, r, "scene_evidence.html", data)
	default: // settings
		s.renderFocus(w, r, "scene_settings.html", data)
	}
}

// renderConsoleGallery renders the gallery with the tenant's
// conversations around a focused conversation.
func (s *Server) renderConsoleGallery(w http.ResponseWriter, r *http.Request, data map[string]any, convID string) {
	key, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	out, err := s.invokeTool(r.Context(), r, tools.ToolConversationList,
		map[string]any{"limit": 20}, key)
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	data["Conversations"], _ = out["conversations"].([]any)
	key, err = newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	data["GalleryOpKey"] = key
	s.renderConsoleScene(w, r, data, "gallery", "scene_gallery.html", convID, nil)
}

// consoleFocus loads one conversation with its builds for the console.
// Callers map not-found to 404 without distinguishing foreign ids.
func (s *Server) consoleFocus(r *http.Request, convID string) (map[string]any, error) {
	key, err := newOpKey()
	if err != nil {
		return nil, err
	}
	return s.invokeTool(r.Context(), r, tools.ToolConversationGet,
		map[string]any{"conversation_id": convID}, key)
}

// renderConsoleScene fills Pulse, Line, meta, and scene extras, then renders.
func (s *Server) renderConsoleScene(w http.ResponseWriter, r *http.Request, data map[string]any, scene, sceneFile, convID string, focus map[string]any) {
	s.renderConsoleSceneCode(w, r, data, scene, sceneFile, convID, focus, http.StatusOK)
}

func (s *Server) renderConsoleSceneCode(w http.ResponseWriter, r *http.Request, data map[string]any, scene, sceneFile, convID string, focus map[string]any, code int) {
	if ph, ok := focusPlaceholders[scene]; ok {
		data["Placeholder"] = ph
	}
	if scene == "evidence" {
		key, err := newOpKey()
		if err != nil {
			renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
			return
		}
		// Per-second SSE stream polls audit as invocations (F7
		// residual) and would bury real receipts under the limit, so
		// the query excludes them; the trail itself stays complete.
		out, err := s.invokeTool(r.Context(), r, tools.ToolActivityList,
			map[string]any{"limit": 50, "exclude": []string{"conversation.stream"}}, key)
		if err != nil {
			renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
			return
		}
		data["Activity"], _ = out["events"].([]any)
	}
	if focus != nil {
		if st, _ := focus["status"].(string); st == "failed" || st == "cancelled" {
			data["BuildTag"] = "bad"
		} else if st == "running" || st == "waiting_user" {
			data["BuildTag"] = "warn"
		} else {
			data["BuildTag"] = "mut"
		}
		if st, _ := focus["status"].(string); st == "running" || st == "waiting_user" {
			data["LineHint"] = "Saved to history — the agent only receives answers submitted to its open question."
		}
	}
	page, err := s.streamPage(r.Context(), r, convID, 0)
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	items, _ := page["items"].([]any)
	data["PulseItems"] = shapePulseItems(items)
	if latest, ok := page["latest_seq"].(float64); ok {
		data["LatestSeq"] = fmt.Sprintf("%.0f", latest)
	} else {
		data["LatestSeq"] = "0"
	}
	data["EventsURL"] = "/conversations/" + convID + "/events?after=" + data["LatestSeq"].(string)
	pending := 0
	if focus != nil {
		if qs, ok := focus["open_questions"].([]any); ok {
			for _, q := range qs {
				if m, ok := q.(map[string]any); ok && m["kind"] == "authorization_request" {
					pending++
				}
			}
		}
	}
	data["PendingApprovals"] = pending
	approvals, err := shapeApprovals(focus)
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	data["Approvals"] = approvals
	key, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	data["LineOpKey"] = key
	if key, err := newOpKey(); err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	} else {
		data["KillOpKey"] = key
	}
	if notice := r.URL.Query().Get("notice"); notice != "" {
		data["Notice"] = notice
	}
	left := "No conversation in focus"
	if c, ok := data["Conversation"].(map[string]any); ok {
		if title, _ := c["title"].(string); title != "" {
			left = title
		} else {
			left = "Conversation"
		}
	}
	data["MetaLeft"] = template.HTML(template.HTMLEscapeString(left))
	if t, _ := data["Tenant"].(string); t != "" {
		data["MetaRight"] = template.HTML(`<span class="tag mut">tenant ` + template.HTMLEscapeString(t) + `</span>`)
	} else {
		data["MetaRight"] = template.HTML("")
	}
	if code != http.StatusOK {
		w.WriteHeader(code)
	}
	s.renderFocus(w, r, sceneFile, data)
}

// shapeApprovals renders the focus build's open authorization requests
// as decide-ready cards, each with its own opkey. Clarifications and
// proposals never appear here: answering them must not authorize.
func shapeApprovals(focus map[string]any) ([]map[string]any, error) {
	out := []map[string]any{}
	if focus == nil {
		return out, nil
	}
	qs, _ := focus["open_questions"].([]any)
	for _, q := range qs {
		m, ok := q.(map[string]any)
		if !ok || m["kind"] != "authorization_request" {
			continue
		}
		// Approve and reject are distinct intentions: each form
		// gets its own key, or the framework's same-key payload
		// fingerprint would conflict the second decision.
		approveKey, err := newOpKey()
		if err != nil {
			return nil, err
		}
		rejectKey, err := newOpKey()
		if err != nil {
			return nil, err
		}
		pending := ""
		if ts, ok := m["created_at"].(string); ok {
			pending = "pending since " + ts
		}
		out = append(out, map[string]any{
			"id": m["id"], "build_id": focus["build_id"], "kind": m["kind"],
			"body": m["body"], "pending": pending,
			"approve_opkey": approveKey, "reject_opkey": rejectKey,
		})
	}
	return out, nil
}

// renderConsoleConflict answers a stale Console POST with the run scene
// at 409: the asked question went stale, and the banner names it. The
// expired-questions card below carries the durable cause.
func (s *Server) renderConsoleConflict(w http.ResponseWriter, r *http.Request, convID, askedQID string) {
	got, err := s.consoleFocus(r, convID)
	if err != nil {
		if isNotFound(err) {
			renderError(w, r, http.StatusNotFound, "Conversation not found.", s.logger, "")
			return
		}
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	tenant, binding, _, _ := TenantOf(r)
	data := map[string]any{"Scene": "run", "Tenant": tenant, "Binding": binding}
	data["Conversation"], _ = got["conversation"].(map[string]any)
	builds, _ := got["builds"].([]any)
	data["Builds"] = builds
	var focus map[string]any
	if len(builds) > 0 {
		focus, _ = builds[len(builds)-1].(map[string]any)
		data["FocusBuild"] = focus
	}
	data["Spine"] = buildSpine(convID, builds)
	data["Conflict"] = map[string]any{"asked": askedQID}
	s.renderConsoleSceneCode(w, r, data, "run", "scene_run.html", convID, focus, http.StatusConflict)
}

// handleConsoleLine appends one Line message and returns to the console.
// It mirrors handleMessageSend's contract with a console redirect.
func (s *Server) handleConsoleLine(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	key, err := opKey(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	convID := r.Form.Get("conversation_id")
	if convID == "" {
		renderError(w, r, http.StatusBadRequest, "Conversation required.", s.logger, "")
		return
	}
	if r.Form.Get("body") == "" {
		renderError(w, r, http.StatusBadRequest, "Message body required.", s.logger, "")
		return
	}
	// Reply mode: the Line carries a question_id from a Pulse reply
	// action and answers it instead of appending a bare message. A
	// stale question conflicts honestly instead of applying silently.
	if qid := r.Form.Get("question_id"); qid != "" {
		_, err = s.invokeTool(r.Context(), r, tools.ToolQuestionAnswer,
			map[string]any{"question_id": qid, "answer": r.Form.Get("body")}, key)
		if err != nil {
			if isStaleQuestion(err) {
				s.renderConsoleConflict(w, r, convID, qid)
				return
			}
			if isNotFound(err) {
				renderError(w, r, http.StatusNotFound, "Question not found.", s.logger, "")
				return
			}
			renderError(w, r, http.StatusBadRequest, friendlyToolError(err), s.logger, err.Error())
			return
		}
		http.Redirect(w, r, "/console?scene=run&conversation="+convID, http.StatusFound)
		return
	}
	_, err = s.invokeTool(r.Context(), r, tools.ToolMessageSend,
		map[string]any{"conversation_id": convID, "body": r.Form.Get("body")}, key)
	if err != nil {
		if isNotFound(err) {
			renderError(w, r, http.StatusNotFound, "Conversation not found.", s.logger, "")
			return
		}
		renderError(w, r, http.StatusBadRequest, friendlyToolError(err), s.logger, err.Error())
		return
	}
	http.Redirect(w, r, "/console?scene=run&conversation="+convID, http.StatusFound)
}

// handleConsoleDecide records one human approval decision. Only open
// authorization_request questions decide here — the kind is re-checked
// server-side, so a clarification can never authorize through this
// endpoint (and the tool writes approval rows for authz kinds only).
// Operator session + Origin required; no agent surface reaches this.
func (s *Server) handleConsoleDecide(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	key, err := opKey(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	qid, decision, convID := r.Form.Get("question_id"), r.Form.Get("decision"), r.Form.Get("conversation_id")
	if qid == "" || convID == "" || (decision != "approved" && decision != "rejected") {
		renderError(w, r, http.StatusBadRequest, "Question, conversation, and an approved/rejected decision are required.", s.logger, "")
		return
	}
	stKey, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	st, err := s.invokeTool(r.Context(), r, tools.ToolBuildStatus,
		map[string]any{"build_id": r.Form.Get("build_id")}, stKey)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, friendlyToolError(err), s.logger, err.Error())
		return
	}
	kind := ""
	if qs, ok := st["open_questions"].([]any); ok {
		for _, q := range qs {
			if m, ok := q.(map[string]any); ok && m["id"] == qid {
				kind, _ = m["kind"].(string)
			}
		}
	}
	if kind != "authorization_request" {
		renderError(w, r, http.StatusBadRequest, "Only authorization requests can be decided here.", s.logger, "")
		return
	}
	_, err = s.invokeTool(r.Context(), r, tools.ToolQuestionAnswer,
		map[string]any{"question_id": qid, "answer": decision}, key)
	if err != nil {
		if isStaleQuestion(err) {
			s.renderConsoleConflict(w, r, convID, qid)
			return
		}
		renderError(w, r, http.StatusBadRequest, friendlyToolError(err), s.logger, err.Error())
		return
	}
	http.Redirect(w, r, "/console?scene=run&conversation="+convID, http.StatusFound)
}

// handleConsoleKill stops every non-terminal build of the session
// tenant. Tenant comes from the session, never from input; the
// redirect reports the honest count, including zero.
func (s *Server) handleConsoleKill(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	if _, err := opKey(r); err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	tenant, _, _, ok := TenantOf(r)
	if !ok || tenant == "" {
		renderError(w, r, http.StatusUnauthorized, "Authentication required.", s.logger, "")
		return
	}
	ids, err := s.cancelAllBuilds(r.Context(), tenant)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, friendlyToolError(err), s.logger, err.Error())
		return
	}
	notice := "Stopped 0 builds — nothing was running."
	if len(ids) == 1 {
		notice = "Stopped 1 build."
	} else if len(ids) > 1 {
		notice = fmt.Sprintf("Stopped %d builds.", len(ids))
	}
	http.Redirect(w, r, "/console?scene=gallery&notice="+url.QueryEscape(notice), http.StatusFound)
}

// handleConsoleConversations opens one conversation and focuses it.
func (s *Server) handleConsoleConversations(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	key, err := opKey(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	out, err := s.invokeTool(r.Context(), r, tools.ToolConversationCreate,
		map[string]any{"title": r.Form.Get("title")}, key)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, friendlyToolError(err), s.logger, err.Error())
		return
	}
	convID, _ := out["conversation_id"].(string)
	http.Redirect(w, r, "/console?scene=run&conversation="+convID, http.StatusFound)
}

// handleFocusJS serves the embedded Console client. Session-gated like
// every app surface; no-cache so deploys never strand old behavior.
func (s *Server) handleFocusJS(w http.ResponseWriter, r *http.Request) {
	raw, err := templateFiles.ReadFile("static/focus/console.js")
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(raw)
}
