package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// renderError answers with a safe message: 4xx carries specifics, 5xx carries
// a generic line while details stay server-side. API paths get JSON, pages
// get minimal HTML. Nothing secret-adjacent is ever rendered: callers pass
// only pre-approved public strings.
func renderError(w http.ResponseWriter, r *http.Request, code int, public string, log *slog.Logger, detail string) {
	if code >= 500 {
		log.Error("request failed", "method", r.Method, "path", r.URL.Path, "code", code, "detail", detail)
		public = "Something went wrong. The operator has been notified."
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": public})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Error %d</title></head>`+
		`<body><h1>Error %d</h1><p>%s</p><p><a href="/">Home</a></p></body></html>`,
		code, code, public)
}

// notFound answers 404 without leaking route existence details.
func notFound(w http.ResponseWriter, r *http.Request) {
	renderError(w, r, http.StatusNotFound, "Not found.", slog.Default(), "")
}
