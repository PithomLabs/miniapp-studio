package server

import (
	"net/http"
	"net/url"

	"github.com/PithomLabs/miniapp-studio/fly"
)

// handleConnectors renders the connector explorer: org-configured
// connections with their states, or an honest unconfigured panel when no
// Fly identity is configured.
func (s *Server) handleConnectors(w http.ResponseWriter, r *http.Request) {
	key, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	res, err := s.invokeTool(r.Context(), r, "connectors.list",
		map[string]any{"provider": r.URL.Query().Get("provider")}, key)
	if err != nil {
		renderError(w, r, http.StatusBadGateway, "Connector list unavailable.", s.logger, err.Error())
		return
	}
	opkey, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	res["OpKey"] = opkey
	renderTemplate(w, r, "connectors.html", res, s.logger)
}

// handleConnectorWizard renders the Custom API wizard form.
func (s *Server) handleConnectorWizard(w http.ResponseWriter, r *http.Request) {
	key, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	renderTemplate(w, r, "connector_wizard.html", map[string]any{"OpKey": key}, s.logger)
}

// customForm carries one wizard submission. Credential lives in memory only:
// it is never logged, persisted, rendered, or passed to any tool.
type customForm struct {
	Name         string
	BaseAPIURL   string
	TestURL      string
	AuthMethod   string
	AccessToken  string
	RefreshToken string
}

func parseCustomForm(r *http.Request) (customForm, error) {
	if err := r.ParseForm(); err != nil {
		return customForm{}, err
	}
	if _, err := opKey(r); err != nil {
		return customForm{}, err
	}
	f := customForm{
		Name:         r.Form.Get("name"),
		BaseAPIURL:   r.Form.Get("base_url"),
		TestURL:      r.Form.Get("test_url"),
		AuthMethod:   r.Form.Get("auth_method"),
		AccessToken:  r.Form.Get("credential"),
		RefreshToken: r.Form.Get("refresh_credential"),
	}
	if f.Name == "" || f.BaseAPIURL == "" || f.AccessToken == "" {
		return customForm{}, errBadForm("name, base URL, and credential are required")
	}
	u, err := url.Parse(f.BaseAPIURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return customForm{}, errBadForm("base URL must be an absolute http(s) URL")
	}
	switch f.AuthMethod {
	case fly.AuthHeader, fly.AuthCustomHeader, fly.AuthQueryParam, fly.AuthURLPath:
	default:
		return customForm{}, errBadForm("unknown auth method")
	}
	return f, nil
}

type formError struct{ msg string }

func (e formError) Error() string { return e.msg }

func errBadForm(msg string) error { return formError{msg} }

// handleConnectorTest runs one ephemeral credential test directly against
// the injected manager. The credential never leaves process memory: no
// persistence, no audit payload, no rendered echo.
func (s *Server) handleConnectorTest(w http.ResponseWriter, r *http.Request) {
	form, err := parseCustomForm(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, safeFormError(err), s.logger, "")
		return
	}
	if s.conns == nil {
		renderTemplate(w, r, "connector_result.html", map[string]any{
			"Name": form.Name, "Unconfigured": true,
		}, s.logger)
		return
	}
	res, err := s.conns.TestCustomAPI(r.Context(), fly.CustomAPITestRequest{
		AccessToken: form.AccessToken, BaseAPIURL: form.BaseAPIURL,
		TestURL: form.TestURL, AuthMethod: form.AuthMethod,
	})
	if err != nil {
		renderTemplate(w, r, "connector_result.html", map[string]any{
			"Name": form.Name, "Error": "Connector test failed.",
		}, s.logger)
		return
	}
	renderTemplate(w, r, "connector_result.html", map[string]any{
		"Name": form.Name, "Status": res.Status, "Message": res.Message,
	}, s.logger)
}

// handleConnectorCreate runs one Custom API creation directly against the
// injected manager with the memory-only credential, then attests the
// sanitized outcome through connectors.custom_attest so the security event
// leaves an audited, secret-free trail.
func (s *Server) handleConnectorCreate(w http.ResponseWriter, r *http.Request) {
	form, err := parseCustomForm(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, safeFormError(err), s.logger, "")
		return
	}
	if s.conns == nil {
		renderTemplate(w, r, "connector_result.html", map[string]any{
			"Name": form.Name, "Unconfigured": true,
		}, s.logger)
		return
	}
	conn, err := s.conns.CreateCustomAPI(r.Context(), fly.CustomAPIRequest{
		Name: form.Name, BaseAPIURL: form.BaseAPIURL, TestURL: form.TestURL,
		AuthMethod: form.AuthMethod, AccessToken: form.AccessToken,
		RefreshToken: form.RefreshToken,
	})
	if err != nil {
		renderTemplate(w, r, "connector_result.html", map[string]any{
			"Name": form.Name, "Error": "Connector creation failed.",
		}, s.logger)
		return
	}
	key, err := newOpKey()
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "", s.logger, err.Error())
		return
	}
	if _, err := s.invokeTool(r.Context(), r, "connectors.custom_attest", map[string]any{
		"id": conn.ID, "provider": conn.Provider, "type": conn.ConnectionType,
	}, key); err != nil {
		renderTemplate(w, r, "connector_result.html", map[string]any{
			"Name": form.Name, "Error": "Connector created but attestation failed.",
		}, s.logger)
		return
	}
	http.Redirect(w, r, "/connectors", http.StatusFound)
}

func safeFormError(err error) string {
	if _, ok := err.(formError); ok {
		return err.Error()
	}
	return "Bad request."
}
