package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/PithomLabs/miniapp-studio/store"
)

// renderTemplate executes one embedded template with autoescaping. Template
// failures are 500s with server-side detail only.
func renderTemplate(w http.ResponseWriter, r *http.Request, name string, data any, log *slog.Logger) {
	tpl, err := template.New("page").ParseFS(templateFiles, "templates/layout.html", "templates/"+name)
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", log, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Error("render failed", "template", name, "detail", err.Error())
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "login.html", map[string]any{
		"Next": safeNext(r.URL.Query().Get("next")),
	}, s.logger)
}

func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	s.login(w, r)
}

// login verifies the operator token against every provisioned operator hash
// (always comparing all entries, so timing reveals nothing about which
// tenant exists), then mints a session bound to the matched tenant.
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad login request.", s.logger, err.Error())
		return
	}
	submitted := r.Form.Get("token")
	sum := sha256.Sum256([]byte(submitted))
	presented := hex.EncodeToString(sum[:])
	var matched *Operator
	for i := range s.cfg.Operators {
		op := &s.cfg.Operators[i]
		if subtle.ConstantTimeCompare([]byte(presented), []byte(op.LoginTokenSHA256)) == 1 {
			matched = op
		}
	}
	if matched == nil {
		w.WriteHeader(http.StatusUnauthorized)
		renderTemplate(w, r, "login.html", map[string]string{
			"Next": safeNext(r.Form.Get("next")), "Error": "Invalid credentials.",
		}, s.logger)
		return
	}
	token, _, err := s.sessions.Create(r.Context(), matched.Tenant, matched.Binding, time.Now())
	if err != nil {
		// F5: the token matched the operators file but the operator row is
		// missing or deactivated — the operator is not provisioned. That is
		// 401 with the identical page, never a 500.
		if errors.Is(err, store.ErrSessionRefused) {
			w.WriteHeader(http.StatusUnauthorized)
			renderTemplate(w, r, "login.html", map[string]string{
				"Next": safeNext(r.Form.Get("next")), "Error": "Invalid credentials.",
			}, s.logger)
			return
		}
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookie, Value: token, Path: "/",
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
		MaxAge: int((12 * time.Hour).Seconds()),
	})
	next := safeNext(r.Form.Get("next"))
	// next arrives form-encoded from the login page; decode defensively.
	if decoded, err := url.QueryUnescape(next); err == nil {
		next = safeNext(decoded)
	}
	http.Redirect(w, r, next, http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(SessionCookie); err == nil && cookie.Value != "" {
		sum := sha256.Sum256([]byte(cookie.Value))
		_ = s.sessions.Revoke(r.Context(), hex.EncodeToString(sum[:]))
	}
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookie, Value: "", Path: "/",
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}
