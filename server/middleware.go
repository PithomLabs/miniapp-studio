package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/PithomLabs/miniapp-studio/store"
)

type ctxKey int

const (
	ctxTenant ctxKey = iota
	ctxBinding
	ctxSession
)

// SessionCookie is the session cookie name.
const SessionCookie = "studio_session"

// TenantOf returns the request's server-derived tenant, binding, and session
// ID. The second result reports whether identity is present at all.
func TenantOf(r *http.Request) (tenant, binding, sessionID string, ok bool) {
	tenant, _ = r.Context().Value(ctxTenant).(string)
	binding, _ = r.Context().Value(ctxBinding).(string)
	sessionID, _ = r.Context().Value(ctxSession).(string)
	return tenant, binding, sessionID, tenant != ""
}

// requireSession authenticates via the session cookie and arms tenant
// context. Unauthenticated browsers (GET) go to /login with a validated
// relative next path; anything else gets 401 JSON. No identity ever comes
// from query parameters, headers, or request bodies.
func (s *Server) requireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookie)
		if err != nil || cookie.Value == "" {
			s.deny(w, r)
			return
		}
		sess, err := s.sessions.Lookup(r.Context(), store.HashToken(cookie.Value))
		if err != nil {
			s.deny(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), ctxTenant, sess.TenantID)
		ctx = context.WithValue(ctx, ctxBinding, sess.BindingID)
		ctx = context.WithValue(ctx, ctxSession, sess.ID)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) deny(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && !strings.HasPrefix(r.URL.Path, "/api/") {
		http.Redirect(w, r, "/login?next="+url.QueryEscape(safeNext(r.URL.RequestURI())), http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"authentication required"}`))
}
