package server

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"
)

// TestUIScreenshots renders representative scenes with fixture data to
// /tmp/uishots for headless-browser capture. Fixtures only — the shots
// prove layout/responsive structure, never backend behavior.
func TestUIScreenshots(t *testing.T) {
	dir := "/tmp/uishots"
	if err := os.MkdirAll(dir+"/static/focus", 0o755); err != nil {
		t.Fatal(err)
	}
	js, err := templateFiles.ReadFile("static/focus/console.js")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/static/focus/console.js", js, 0o644); err != nil {
		t.Fatal(err)
	}
	spine := []spinePill{
		{Label: "Describe", State: "proven", Title: "t", Href: "#"},
		{Label: "Spec", State: "active", Title: "t", Href: "#"},
		{Label: "Validate", State: "idle", Title: "t", Href: "#"},
		{Label: "Connect", State: "idle", Title: "t", Href: "#"},
		{Label: "Deploy", State: "idle", Title: "t", Href: "#"},
		{Label: "Run", State: "idle", Title: "t", Href: "#"},
		{Label: "Evidence", State: "idle", Title: "t", Href: "#"},
	}
	conv := map[string]any{"id": "c1", "title": "Invoice Review Desk"}
	convs := []any{
		map[string]any{"id": "c1", "title": "Invoice Review Desk", "status": "open"},
		map[string]any{"id": "c2", "title": "Box Digest", "status": "waiting_user"},
	}
	consoleBase := func() map[string]any {
		return map[string]any{
			"Scene": "gallery", "Spine": spine, "MetaLeft": "meta", "MetaRight": "",
			"Conversation": conv, "Conversations": convs, "GalleryOpKey": "k",
			"Areas": focusAreas, "KillOpKey": "kk", "LineOpKey": "lk",
			"LineHint": "Ask, answer, or say what's next",
			"PulseItems": []pulseItem{
				{Seq: "12", Class: "receipt", Kind: "receipt", Title: "Build opened",
					Detail: "turn 1 assigned", Time: "just now"},
				{Seq: "13", Class: "alert", Kind: "alert", Title: "Probe refused",
					Detail: "policy gov.noop", Time: "just now"},
			},
			"PendingApprovals": 1,
			"Approvals": []map[string]any{{
				"body": "Deploy v3 to staging?", "pending": "expires soon",
				"id": "q9", "build_id": "b1",
				"approve_opkey": "a", "reject_opkey": "r",
			}},
			"ContextBar": map[string]any{"project": "studio", "live_ok": true,
				"live_branch": "main", "live_head": "abc1234", "live_dirty": false,
				"ckpt": "#7", "stale": false, "agent": "opencode", "workspace": "w1", "env": "dev"},
		}
	}
	areaBase := func(scene, meta string) map[string]any {
		return focusPageData(scene, nil, meta)
	}
	build := map[string]any{"build_id": "b1", "status": "running", "conversation_id": "c1",
		"turn":              map[string]any{"n": 2, "status": "open", "id": "t2"},
		"agent_session_ref": "sess-1",
		"open_questions": []any{map[string]any{
			"id": "q1", "kind": "clarification", "body": "Which city?",
			"turn_id": "t2",
		}},
		"commands": []any{map[string]any{
			"command_kind": "assign", "state": "acked", "attempts": 1, "reason": "ok",
		}},
		"decisions": []any{map[string]any{
			"decision": "approved", "question_id": "q0",
			"decided_by": "op", "decided_at": "10:50",
		}},
	}
	app := map[string]any{"id": "a1", "name": "Invoicer", "slug": "invoicer",
		"version": "3", "hash": "abc123"}
	cases := map[string]struct {
		scene string
		data  map[string]any
	}{
		"gallery":       {"scene_gallery.html", consoleBase()},
		"run":           {"scene_run.html", merge(consoleBase(), map[string]any{"Scene": "run", "FocusBuild": build, "BuildTag": "warn", "AttemptOpKey": "k", "StartOpKey": "k", "CaptureOpKey": "k", "InterruptOpKey": "k", "ResumeOpKey": "k"})},
		"conversations": {"scene_conversations.html", merge(areaBase("conversations", "Development tasks"), map[string]any{"Conversations": convs, "OpKey": "k"})},
		"conversation":  {"scene_conversation.html", merge(areaBase("conversations", "1 build"), map[string]any{"Thread": conv, "Builds": []any{build}, "Items": []pageItem{{Seq: 1, Kind: "message", Role: "user", Body: "Book it"}}, "LatestSeq": int64(1), "MsgOpKey": "k"})},
		"apps":          {"scene_apps.html", merge(areaBase("apps", "1 application"), map[string]any{"Apps": []any{map[string]any{"id": "a1", "name": "Invoicer", "slug": "invoicer", "version": "3", "deploy_status": "live"}}, "OpKey": "k"})},
		"app-detail":    {"scene_app_detail.html", merge(areaBase("apps", "detail"), map[string]any{"App": app, "OpKey": "k", "Versions": []any{map[string]any{"version": "3", "hash": "abc123"}}, "Deploy": map[string]any{"status": "live", "url": "https://x"}})},
		"connectors":    {"scene_connectors.html", merge(areaBase("connectors", "Managed connections"), map[string]any{"status": "ok", "connections": []any{map[string]any{"id": "c1", "provider": "GitHub", "type": "oauth"}}, "OpKey": "k"})},
		"previews":      {"scene_previews.html", merge(areaBase("previews", "1 application"), map[string]any{"Apps": []any{map[string]any{"id": "a1", "name": "Invoicer", "version": "3", "deploy_status": "live", "deploy_url": "https://x"}}})},
		"incidents":     {"scene_incidents.html", merge(areaBase("incidents", "Troubleshooting"), map[string]any{"Total": 3, "ToolFilter": "", "Failed": []any{map[string]any{"id": 7, "at": "t", "tool": "app", "action": "create", "outcome": "failed", "miniapp": "m1", "version": "2"}}})},
		"release":       {"scene_release.html", merge(areaBase("release", "Staging & production"), map[string]any{"ConversationID": "c1", "Artifacts": []any{map[string]any{"digest": "d1", "verified": true, "latest_run_status": "passed", "decision": "approved", "decided_by": "op"}}})},
	}
	for name, tc := range cases {
		tpl, err := template.New("focus").ParseFS(templateFiles,
			"templates/focus/layout.html", "templates/focus/"+tc.scene)
		if err != nil {
			t.Fatalf("%s: parse: %v", name, err)
		}
		f, err := os.Create(filepath.Join(dir, name+".html"))
		if err != nil {
			t.Fatal(err)
		}
		if err := tpl.ExecuteTemplate(f, "focus-layout", tc.data); err != nil {
			f.Close()
			t.Fatalf("%s: execute: %v", name, err)
		}
		f.Close()
	}
}

func merge(base, over map[string]any) map[string]any {
	for k, v := range over {
		base[k] = v
	}
	return base
}
