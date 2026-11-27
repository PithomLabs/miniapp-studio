package server

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The DB-backed tests below prove shell unification live; these DB-free
// tests prove the render contract repo-locally (template balance, shell
// keys, locked rail, NoPulse structure) on every run.
func TestFocusTemplatesParse(t *testing.T) {
	scenes := []string{
		"scene_gallery.html", "scene_spec.html", "scene_validate.html",
		"scene_connect.html", "scene_placeholder.html", "scene_run.html",
		"scene_evidence.html", "scene_settings.html", "scene_conversations.html",
		"scene_conversation.html", "scene_answer_conflict.html", "scene_home.html",
		"scene_apps.html", "scene_app_new.html", "scene_spec_result.html",
		"scene_preview_result.html", "scene_app_detail.html", "scene_deploy.html",
		"scene_deploy_result.html", "scene_deploy_interstitial.html", "scene_activity.html",
		"scene_connectors.html", "scene_connector_wizard.html", "scene_connector_result.html",
		"scene_previews.html", "scene_incidents.html", "scene_release.html",
	}
	for _, sc := range scenes {
		if _, err := template.New("focus").ParseFS(templateFiles,
			"templates/focus/layout.html", "templates/focus/"+sc); err != nil {
			t.Errorf("parse %s: %v", sc, err)
		}
	}
}

func renderLayout(t *testing.T, sceneFile string, data map[string]any) string {
	t.Helper()
	srv := &Server{}
	for k, v := range focusPageData("conversations", nil, "test") {
		if _, ok := data[k]; !ok {
			data[k] = v
		}
	}
	rec := httptest.NewRecorder()
	srv.renderFocus(rec, httptest.NewRequest(http.MethodGet, "/", nil), sceneFile, data)
	return rec.Body.String()
}

func TestFocusPageShellContract(t *testing.T) {
	body := renderLayout(t, "scene_conversations.html", map[string]any{
		"Conversations": []any{}, "OpKey": "k",
	})
	for _, want := range []string{
		`class="app nopulse"`, `<h1 class="title">Conversations</h1>`,
		"No conversations yet", `aria-label="Essentials"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	for _, banned := range []string{
		"<no value>", "id=\"pulseBtn\"", "id=\"killForm\"", "id=\"pulse\"",
		"id=\"pulseEdge\"", "id=\"spine\"", "id=\"lineForm\"",
	} {
		if strings.Contains(body, banned) {
			t.Errorf("B-area page leaks %q", banned)
		}
	}
	if got := strings.Count(body, `<a class="rbtn`); got != 4 {
		t.Errorf("rail anchors = %d, want exactly 4 locked destinations", got)
	}
}

// TestMigratedScenesRender proves every migrated B-area scene renders
// representative data with no "<no value>" leaks — template
// self-consistency plus the shell contract, repo-locally. Honest scope:
// fixtures mirror the templates by hand, so a handler passing a wrongly
// shaped map still slips through here; the live test below
// (TestAreaPagesServeClean) executes the real handlers and closes that
// gap, as do the per-route content tests.
func TestMigratedScenesRender(t *testing.T) {
	conv := map[string]any{"id": "c1", "title": "Desk chat"}
	build := map[string]any{"build_id": "b1", "status": "waiting_user",
		"turn": map[string]any{"n": 2, "status": "open"},
		"open_questions": []any{map[string]any{
			"id": "q1", "kind": "clarification", "body": "Which city?",
			"options": []any{"Leeds", "York"}, "opkey": "k",
		}}}
	app := map[string]any{"id": "a1", "name": "Invoicer", "slug": "invoicer",
		"version": "3", "hash": "abc123"}
	cases := []struct {
		name   string
		scene  string
		data   map[string]any
		want   []string
		banned []string
	}{
		{"conversations", "scene_conversations.html",
			map[string]any{"Conversations": []any{
				map[string]any{"id": "c1", "title": "Desk chat", "status": "open"},
			}, "OpKey": "k"},
			[]string{"Desk chat", "/conversations/c1", "New conversation"}, nil},
		{"conversation", "scene_conversation.html",
			map[string]any{"Thread": conv, "Builds": []any{build},
				"Items":     []pageItem{{Seq: 1, Kind: "message", Role: "user", Body: "Book it"}},
				"LatestSeq": int64(1), "MsgOpKey": "k"},
			[]string{"Desk chat", "b1", "Book it", "Which city?", "Leeds", "id=\"thread\""},
			[]string{"id=\"lineForm\"", "Select or start a conversation first"}},
		{"answer-conflict", "scene_answer_conflict.html",
			map[string]any{"BuildID": "b1", "ConversationID": "c1",
				"OpenQuestions": []any{map[string]any{"kind": "clarification", "body": "Which city?"}}},
			[]string{"already answered", "Which city?", "/conversations/c1"}, nil},
		{"home", "scene_home.html",
			map[string]any{"Tenant": "t1",
				"Apps":   []any{map[string]any{"id": "a1", "name": "Invoicer", "slug": "invoicer"}},
				"Events": []any{map[string]any{"tool": "app", "action": "create", "outcome": "ok"}}},
			[]string{"t1", "Invoicer", "app create", "/console?scene=gallery"}, nil},
		{"apps", "scene_apps.html",
			map[string]any{"Apps": []any{map[string]any{
				"id": "a1", "name": "Invoicer", "slug": "invoicer", "version": "3", "deploy_status": "live",
			}}, "OpKey": "k"},
			[]string{"Invoicer", "invoicer", "/apps/a1"}, nil},
		{"app-new", "scene_app_new.html",
			map[string]any{"OpKey": "k"},
			[]string{"Register the app", "/apps/new/validate", "/apps/new/preview", "ships in P5"}, nil},
		{"spec-valid", "scene_spec_result.html",
			map[string]any{"valid": true, "hash": "h1", "Spec": "{}"},
			[]string{"Valid", "h1"}, nil},
		{"spec-invalid", "scene_spec_result.html",
			map[string]any{"valid": false, "Spec": "{bad",
				"errors": []any{map[string]any{"code": "E_SCHEMA", "path": "$", "message": "invalid JSON"}}},
			[]string{"Invalid", "E_SCHEMA", "invalid JSON"}, nil},
		{"preview-ok", "scene_preview_result.html",
			map[string]any{"fixture": "f1", "workflowId": "w1", "completed": true,
				"outputs": "{}", "messages": []any{"m1"}},
			[]string{"f1", "w1", "m1"}, nil},
		{"preview-failed", "scene_preview_result.html",
			map[string]any{"DryError": "boom"},
			[]string{"Preview failed", "boom"}, nil},
		{"app-detail", "scene_app_detail.html",
			map[string]any{"App": app, "OpKey": "k",
				"Versions": []any{map[string]any{"version": "3", "hash": "abc123"}},
				"Deploy":   map[string]any{"status": "live", "url": "https://x"}},
			[]string{"Invoicer", "abc123", "live", "https://x", "/apps/a1/deploy"}, nil},
		{"deploy", "scene_deploy.html",
			map[string]any{"App": app, "OpKey": "k", "FlyConfigured": true,
				"Deploy": map[string]any{"status": "live", "url": "https://x"}},
			[]string{"Validate &amp; Deploy", "configured", "Provision", "Retire"}, nil},
		{"deploy-provisioning", "scene_deploy.html",
			map[string]any{"App": app, "OpKey": "k", "FlyConfigured": false,
				"Deploy": map[string]any{"status": "provisioning"}},
			[]string{"http-equiv=\"refresh\"", "not configured"}, nil},
		{"deploy-result", "scene_deploy_result.html",
			map[string]any{"AppID": "a1", "Deploy": map[string]any{"status": "live", "url": "https://x"}},
			[]string{"live", "https://x", "/apps/a1/deploy"}, nil},
		{"deploy-interstitial", "scene_deploy_interstitial.html",
			map[string]any{"AppID": "a1"},
			[]string{"Provisioning", "refresh", "/apps/a1/deploy"}, nil},
		{"activity", "scene_activity.html",
			map[string]any{"events": []any{map[string]any{
				"at": "t", "tool": "app", "action": "create", "outcome": "ok",
			}}, "next_cursor": "c9"},
			[]string{"app", "/activity?cursor=c9"}, nil},
		{"activity-empty", "scene_activity.html",
			map[string]any{},
			[]string{"No activity yet"}, nil},
		{"connectors", "scene_connectors.html",
			map[string]any{"status": "ok", "connections": []any{
				map[string]any{"id": "c1", "provider": "GitHub", "type": "oauth"},
			}},
			[]string{"GitHub", "oauth", "/connectors/custom/new", "/console?scene=connect"}, nil},
		{"connectors-unconfigured", "scene_connectors.html",
			map[string]any{"status": "unconfigured"},
			[]string{"not configured", "nothing here is live data"}, nil},
		{"connector-wizard", "scene_connector_wizard.html",
			map[string]any{"OpKey": "k"},
			[]string{"Custom API", "Test without storing", "create scoped connection", "never displayed back"}, nil},
		{"connector-result-ok", "scene_connector_result.html",
			map[string]any{"Name": "Acme", "Status": "tested", "Message": "reachable"},
			[]string{"Acme", "tested", "reachable"}, nil},
		{"connector-result-failed", "scene_connector_result.html",
			map[string]any{"Name": "Acme", "Error": "Connector test failed."},
			[]string{"Failed", "Connector test failed."}, nil},
		{"connector-result-unconfigured", "scene_connector_result.html",
			map[string]any{"Name": "Acme", "Unconfigured": true},
			[]string{"Not configured", "nothing was tested or stored"}, nil},
		{"previews", "scene_previews.html",
			map[string]any{"Apps": []any{map[string]any{
				"id": "a1", "name": "Invoicer", "version": "3",
				"deploy_status": "live", "deploy_url": "https://x",
			}}},
			[]string{"Workspace Preview", "Application Preview", "Planned", "T3-gated",
				"Invoicer", "never merged here", "disabled"}, nil},
		{"previews-empty", "scene_previews.html",
			map[string]any{"Apps": []any{}},
			[]string{"No applications yet", "Planned"}, nil},
		{"incidents", "scene_incidents.html",
			map[string]any{"Total": 3, "ToolFilter": "", "Failed": []any{
				map[string]any{"id": 7, "at": "t", "tool": "app", "action": "create",
					"outcome": "failed", "miniapp": "m1", "version": "2"},
			}},
			[]string{"Recent failures", "failed", "Managed incidents", "Planned",
				"/activity", "Filter by exact tool name", "m1", "Per-second stream polls"}, nil},
		{"incidents-clean", "scene_incidents.html",
			map[string]any{"Total": 3, "ToolFilter": "app"},
			[]string{"No failed outcomes", "Planned", "Clear"}, nil},
		{"release-no-conv", "scene_release.html",
			map[string]any{},
			[]string{"Release", "Staging", "Production", "Planned", "Phase E",
				"Choose conversation", "disabled"}, nil},
		{"release-candidate", "scene_release.html",
			map[string]any{"ConversationID": "c1", "Artifacts": []any{
				map[string]any{"digest": "d1", "verified": true,
					"latest_run_status": "passed", "decision": "approved", "decided_by": "op"},
			}},
			[]string{"c1", "d1", "verified", "approved", "Promote to staging", "disabled"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := renderLayout(t, tc.scene, tc.data)
			for _, want := range tc.want {
				if !strings.Contains(body, want) {
					t.Errorf("scene lacks %q", want)
				}
			}
			for _, banned := range tc.banned {
				if strings.Contains(body, banned) {
					t.Errorf("scene leaks %q", banned)
				}
			}
			if strings.Contains(body, "<no value>") {
				t.Error("scene leaks <no value>")
			}
		})
	}
}

func TestFocusPageConsoleUnaffected(t *testing.T) {
	data := map[string]any{
		"Scene": "gallery", "Conversations": []any{}, "GalleryOpKey": "k",
		"Areas": focusAreas, "KillOpKey": "kk",
		"Spine":    []spinePill{{Label: "Describe", State: "idle", Title: "t", Href: "/console?scene=gallery"}},
		"MetaLeft": "m", "MetaRight": "", "NoPulse": false,
		"PulseItems":       []pulseItem{{Title: "t", Detail: "d"}},
		"PendingApprovals": 0,
	}
	rec := httptest.NewRecorder()
	(&Server{}).renderFocus(rec, httptest.NewRequest(http.MethodGet, "/", nil), "scene_gallery.html", data)
	body := rec.Body.String()
	for _, want := range []string{
		`class="app"`, "id=\"pulseBtn\"", "id=\"killForm\"", "id=\"pulse\"",
		"id=\"spine\"", "Studio areas", `href="/apps"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("console page lacks %q", want)
		}
	}
	if strings.Contains(body, "<no value>") {
		t.Error("console page leaks <no value>")
	}
}

// W1 pattern tests: B-area pages render inside the locked Focus shell.
// The rail keeps exactly its five locked destinations (owner decision —
// no area buttons may creep in); areas are entered from gallery cards.
func TestConversationsPageInFocusShell(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	rec := doRequest(t, srv, http.MethodGet, "/conversations", "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /conversations = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`class="app nopulse"`,
		`aria-label="Essentials"`,
		"<h1 class=\"title\">Conversations</h1>",
		"New conversation",
		`action="/conversations"`,
		"Development tasks",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	for _, banned := range []string{"<no value>", "id=\"pulseBtn\"", "id=\"killForm\"", "id=\"pulse\"", "id=\"lineForm\""} {
		if strings.Contains(body, banned) {
			t.Errorf("B-area page leaks %q (pulse/kill must be structurally absent)", banned)
		}
	}
}

func TestNewAreaPagesLive(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	for target, want := range map[string]string{
		"/previews":  "Which preview is which",
		"/incidents": "Recent failures",
		"/release":   "Environment pipeline",
	} {
		rec := doRequest(t, srv, http.MethodGet, target, "", cookie)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d", target, rec.Code)
			continue
		}
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("GET %s lacks %q", target, want)
		}
	}
	rec := doRequest(t, srv, http.MethodGet, "/release?conversation=nope", "", cookie)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /release?conversation=nope = %d, want 404", rec.Code)
	}
}

// TestAreaPagesServeClean executes every no-param B-area GET live and
// asserts the handler/template key contract end to end: 200, shell
// present, and no "<no value>" leak. This is the test that would have
// caught a Valid/valid-class miss on any migrated page.
func TestAreaPagesServeClean(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	targets := []string{
		"/", "/conversations", "/apps", "/apps/new", "/connectors",
		"/connectors/custom/new", "/activity", "/previews", "/incidents",
		"/release",
	}
	for _, target := range targets {
		rec := doRequest(t, srv, http.MethodGet, target, "", cookie)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", target, rec.Code)
			continue
		}
		body := rec.Body.String()
		if strings.Contains(body, "<no value>") {
			t.Errorf("GET %s leaks <no value>", target)
		}
		if !strings.Contains(body, `aria-label="Essentials"`) {
			t.Errorf("GET %s lacks the focus shell", target)
		}
	}
}

func TestGalleryAreaCards(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	rec := doRequest(t, srv, http.MethodGet, "/console?scene=gallery", "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET gallery = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Studio areas") {
		t.Fatal("gallery lacks the areas card")
	}
	for _, href := range []string{`href="/"`, `href="/conversations"`, `href="/apps"`, `href="/connectors"`, `href="/activity"`, `href="/previews"`, `href="/incidents"`, `href="/release"`} {
		if !strings.Contains(body, href) {
			t.Errorf("gallery lacks area link %s", href)
		}
	}
}

func TestLockedRailUnchanged(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	rec := doRequest(t, srv, http.MethodGet, "/console?scene=gallery", "", cookie)
	body := rec.Body.String()
	if got := strings.Count(body, `<a class="rbtn`); got != 4 {
		t.Errorf("rail anchors = %d, want exactly 4 (gallery/connect/evidence/settings)", got)
	}
	for _, want := range []string{
		`href="/console?scene=gallery"`,
		`href="/console?scene=connect"`,
		`href="/console?scene=evidence"`,
		`href="/console?scene=settings"`,
		`id="pulseBtn"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rail lacks locked destination %q", want)
		}
	}
}
