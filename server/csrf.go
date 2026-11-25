package server

import (
	"net/http"
	"net/url"
	"strings"
)

// requireOrigin rejects cookie-authenticated mutations whose Origin header
// is missing or does not match the request host. Same-origin browser forms
// always send Origin on POST; cross-site forgery cannot forge it. Combined
// with SameSite=Lax cookies, this is the documented CSRF strategy; no
// anti-forgery token dance is needed while all mutations are same-origin
// form posts. If cross-origin or non-browser clients ever arrive, revisit.
func requireOrigin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"origin check failed"}`))
			return
		}
		next(w, r)
	}
}

func sameOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return strings.EqualFold(parsed.Host, r.Host)
}
