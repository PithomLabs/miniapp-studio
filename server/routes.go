package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed templates/*.html templates/focus/*.html static/focus/console.js
var templateFiles embed.FS

func (s *Server) routes() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	// Gateway callbacks authenticate by HMAC envelope, not operator
	// session: no requireSession and no Origin check on this route.
	mux.HandleFunc("POST /gw/v1/callback", s.handleGatewayCallback)
	mux.HandleFunc("GET /conversations/{id}/events", s.requireSession(s.handleConversationEvents))
	mux.HandleFunc("GET /conversations", s.requireSession(s.handleConversationsList))
	mux.HandleFunc("POST /conversations", s.requireSession(requireOrigin(s.handleConversationsCreate)))
	mux.HandleFunc("GET /conversations/{id}", s.requireSession(s.handleConversationView))
	mux.HandleFunc("POST /conversations/{id}/messages", s.requireSession(requireOrigin(s.handleMessageSend)))
	mux.HandleFunc("POST /questions/{qid}/answer", s.requireSession(requireOrigin(s.handleQuestionAnswer)))
	mux.HandleFunc("GET /login", s.handleLogin)
	mux.HandleFunc("POST /login", requireOrigin(s.handleLoginPost))
	mux.HandleFunc("POST /logout", s.requireSession(requireOrigin(s.handleLogout)))
	mux.HandleFunc("GET /api/whoami", s.requireSession(s.handleWhoami))
	mux.HandleFunc("GET /{$}", s.requireSession(s.handleHome))
	mux.HandleFunc("GET /apps/new", s.requireSession(s.handleAppsNew))
	mux.HandleFunc("POST /apps/new/validate", s.requireSession(requireOrigin(s.handleAppsValidate)))
	mux.HandleFunc("POST /apps/new/preview", s.requireSession(requireOrigin(s.handleAppsPreview)))
	mux.HandleFunc("POST /apps", s.requireSession(requireOrigin(s.handleAppsCreate)))
	mux.HandleFunc("GET /apps", s.requireSession(s.handleAppsList))
	mux.HandleFunc("GET /apps/{id}", s.requireSession(s.handleAppDetail))
	mux.HandleFunc("POST /apps/{id}/specs", s.requireSession(requireOrigin(s.handleAppSaveSpec)))
	mux.HandleFunc("GET /apps/{id}/deploy", s.requireSession(s.handleDeployPage))
	mux.HandleFunc("POST /apps/{id}/deploy", s.requireSession(requireOrigin(s.handleDeployProvision)))
	mux.HandleFunc("POST /apps/{id}/retire", s.requireSession(requireOrigin(s.handleDeployRetire)))
	mux.HandleFunc("GET /connectors", s.requireSession(s.handleConnectors))
	mux.HandleFunc("GET /connectors/custom/new", s.requireSession(s.handleConnectorWizard))
	mux.HandleFunc("POST /connectors/custom/test", s.requireSession(requireOrigin(s.handleConnectorTest)))
	mux.HandleFunc("POST /connectors/custom/create", s.requireSession(requireOrigin(s.handleConnectorCreate)))
	mux.HandleFunc("GET /activity", s.requireSession(s.handleActivity))
	mux.HandleFunc("GET /console", s.requireSession(s.handleConsole))
	mux.HandleFunc("POST /console/line", s.requireSession(requireOrigin(s.handleConsoleLine)))
	mux.HandleFunc("POST /console/conversations", s.requireSession(requireOrigin(s.handleConsoleConversations)))
	mux.HandleFunc("POST /console/decide", s.requireSession(requireOrigin(s.handleConsoleDecide)))
	mux.HandleFunc("POST /console/kill", s.requireSession(requireOrigin(s.handleConsoleKill)))
	mux.HandleFunc("GET /static/focus/console.js", s.requireSession(s.handleFocusJS))
	mux.HandleFunc("/", s.handleNotFound)
	s.mux = mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok","service":"miniapp-studio"}`))
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := s.pool.Ping(r.Context()); err != nil {
		writeUnavailable(w)
		return
	}
	// F12: a migrated-back or partially migrated database reports unready.
	if err := checkSchemaVersion(r.Context(), s.pool); err != nil {
		s.logger.Warn("readyz: schema gate", "detail", err.Error())
		writeUnavailable(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ready"}`))
}

func writeUnavailable(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"status":"unavailable"}`))
}

// checkSchemaVersion enforces that the database matches this binary's
// embedded migrations exactly, and is not dirty from a failed run.
func checkSchemaVersion(ctx context.Context, pool *pgxpool.Pool) error {
	latest, err := store.LatestVersion()
	if err != nil {
		return fmt.Errorf("server: latest schema: %w", err)
	}
	version, dirty, err := store.CurrentVersion(ctx, pool)
	if err != nil {
		return fmt.Errorf("server: schema check: %w", err)
	}
	if dirty {
		return fmt.Errorf("server: schema is dirty at v%d (repair before serving)", version)
	}
	if version != latest {
		return fmt.Errorf("server: schema at v%d, binary expects v%d (run migrate)", version, latest)
	}
	return nil
}

func (s *Server) handleWhoami(w http.ResponseWriter, r *http.Request) {
	tenant, binding, sessionID, _ := TenantOf(r)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"tenant": tenant, "binding": binding, "session": sessionID,
	})
}

// handleNotFound answers unknown routes with the branded error page.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	notFound(w, r)
}
