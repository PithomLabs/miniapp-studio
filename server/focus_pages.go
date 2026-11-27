package server

import (
	"fmt"
	"html/template"
	"net/http"
)

// Focus area pages (W1): every authenticated B-area renders inside the
// Focus shell as a scene, with an area-lifecycle spine and no pulse
// column. Routes and handler logic are unchanged — only the shell is
// unified, so the Studio reads as one control plane. The 5-icon rail
// stays locked per the binding UX spec; areas are entered from gallery
// cards and contextual links, never from new rail buttons.
//
// Areas link only to routes that exist. Previews/Release arrive with
// their routes in W4/W5; nothing links a 404.
type focusArea struct {
	// Scene is the .Scene highlight key. B-area keys live in a separate
	// namespace from console scenes and are never valid ?scene= values.
	Scene string
	Label string
	Href  string
	Blurb string
}

var focusAreas = []focusArea{
	{Scene: "home", Label: "Overview", Href: "/", Blurb: "Apps + activity summary"},
	{Scene: "conversations", Label: "Conversations", Href: "/conversations", Blurb: "Development tasks"},
	{Scene: "apps", Label: "Apps", Href: "/apps", Blurb: "Applications"},
	{Scene: "connectors", Label: "Connectors", Href: "/connectors", Blurb: "Connector manager"},
	{Scene: "activity", Label: "Activity", Href: "/activity", Blurb: "Evidence trail"},
	{Scene: "previews", Label: "Previews", Href: "/previews", Blurb: "Environments + planned tracks"},
	{Scene: "incidents", Label: "Incidents", Href: "/incidents", Blurb: "Failures + planned tracking"},
	{Scene: "release", Label: "Release", Href: "/release", Blurb: "Pipeline + candidates"},
}

// appsSpine is the Apps area lifecycle: browse the tenant's apps or start
// a new one. Validation and preview live inside the New flow, so result
// pages mark New active. Pills are navigation, never permission.
func appsSpine(active string) []spinePill {
	pills := []spinePill{
		{Label: "Apps", Href: "/apps", Title: "Tenant applications"},
		{Label: "New", Href: "/apps/new", Title: "Register, validate, preview"},
	}
	for i := range pills {
		state := "idle"
		if (active == "apps" && i == 0) || (active == "new" && i == 1) {
			state = "active"
		}
		pills[i].State = state
	}
	return pills
}

// connectorsSpine is the Connectors area lifecycle: browse managed
// connections or open the Custom API wizard. Pills are navigation,
// never permission.
func connectorsSpine(active string) []spinePill {
	pills := []spinePill{
		{Label: "Manager", Href: "/connectors", Title: "Managed connections"},
		{Label: "Wizard", Href: "/connectors/custom/new", Title: "New Custom API connection"},
	}
	for i := range pills {
		state := "idle"
		if (active == "manager" && i == 0) || (active == "wizard" && i == 1) {
			state = "active"
		}
		pills[i].State = state
	}
	return pills
}

// plural renders a counted noun for stagemeta lines ("1 build", "3 builds").
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// focusPageData builds shell defaults for a B-area page. Handler domain
// data is merged on top by renderFocusPage, so area content always wins.
// B-area pages ship without pulse (conversation-scoped stream) and without
// the tenant-kill control (console-only surface); both omissions are
// structural (keys absent), never disabled controls that imply function.
func focusPageData(scene string, spine []spinePill, metaLeft string) map[string]any {
	return map[string]any{
		"Scene":            scene,
		"Spine":            spine,
		"MetaLeft":         template.HTML(metaLeft),
		"MetaRight":        template.HTML(""),
		"PulseItems":       []pulseItem{},
		"PendingApprovals": 0,
		"Approvals":        []map[string]any{},
		"NoPulse":          true,
	}
}

// renderFocusPage renders a B-area scene inside the Focus shell. data
// carries the handler's domain values; shell defaults fill every key the
// layout requires so no "<no value>" ever leaks into the page.
func (s *Server) renderFocusPage(w http.ResponseWriter, r *http.Request, sceneFile, scene string, spine []spinePill, metaLeft string, data map[string]any) {
	for k, v := range focusPageData(scene, spine, metaLeft) {
		if _, ok := data[k]; !ok {
			data[k] = v
		}
	}
	s.renderFocus(w, r, sceneFile, data)
}
