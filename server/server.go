package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/executor/classa"
	executor "github.com/PithomLabs/miniapp/executor/dbos"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/runtime"

	"github.com/PithomLabs/miniapp-studio/connector"
	"github.com/PithomLabs/miniapp-studio/fly"
	"github.com/PithomLabs/miniapp-studio/gateway"
	"github.com/PithomLabs/miniapp-studio/runner"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/tools"
)

// appPoolMaxConns budgets the shared application pool. Sessions, framework
// operations, and tools share this pool; the total must stay under the
// platform connection ceiling (MPG Developer: 20 across all processes).
const appPoolMaxConns = 8

// Server is the wired Studio control plane.
type Server struct {
	cfg      Config
	pool     *pgxpool.Pool
	svc      *runtime.Service
	sessions *store.Store
	tenants  map[string]Operator
	sprites  fly.SpriteManager
	conns    fly.ConnectorManager
	logger   *slog.Logger
	mux      *http.ServeMux
	gwKeys   gateway.KeyProvider
	// ssePoll overrides the SSE poll cadence in tests. Zero means default.
	ssePoll time.Duration
	// closeMu guards exactly-once shutdown with a retained error.
	closeMu  sync.Mutex
	closed   bool
	closeErr error
	// sender is the Studio-to-Gateway dispatch loop. Nil unless
	// StartSender ran; Close stops it before the pool it uses.
	sender *Sender
	// deployMu guards in-flight deploy workers keyed by tenant+app, so a
	// second provision POST while one runs gets an honest conflict.
	deployMu      sync.Mutex
	deployRunning map[string]bool
	executor      interface {
		Shutdown(time.Duration) error
	}
}

// Options injects seams for tests. Production passes zero Options and gets
// file-backed configuration with honest unconfigured states.
type Options struct {
	Sprites    fly.SpriteManager
	Connectors fly.ConnectorManager
	Logger     *slog.Logger
	// GatewayKeys authenticates gateway callbacks. Nil means unconfigured:
	// the callback route answers 503. Phase A tests inject the static
	// test provider; production keys arrive with Phase B provisioning.
	GatewayKeys gateway.KeyProvider
	// GatewayBinary feeds build.provision with gateway server bytes.
	// Tests inject fixture bytes; the operator CLI reads a file. Serve
	// leaves this empty until binary-path config lands (runbook item).
	GatewayBinary []byte
	// GatewayOrgSecret feeds build.provision with the org HMAC secret.
	// Serve takes it from configuration; tests inject fixtures.
	GatewayOrgSecret []byte
}

// New wires the server: pools, policy, registry with miniapps, executors,
// and services. It fails closed on any misconfiguration and leaves no
// half-built resources behind.
func New(ctx context.Context, cfg Config, opts Options) (*Server, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("server: parse database URL: %w", err)
	}
	poolCfg.MaxConns = appPoolMaxConns
	poolCfg.ConnConfig.ConnectTimeout = 30 * time.Second
	poolCfg.HealthCheckPeriod = time.Minute
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.AfterRelease = store.TenantResetHook()
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("server: database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("server: database unreachable: %w", err)
	}
	// F12: refuse to serve against a database that is behind this binary.
	if err := checkSchemaVersion(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	policy, err := loadFrameworkPolicy(cfg.FrameworkKeys, cfg.FrameworkGrants)
	if err != nil {
		pool.Close()
		return nil, err
	}
	reg := registry.New()
	if err := runner.Register(reg, runner.Deps{
		Connections: unconfiguredBackend{},
		Reader:      unconfiguredBackend{},
	}); err != nil {
		pool.Close()
		return nil, fmt.Errorf("server: register runner: %w", err)
	}
	orgSecret := opts.GatewayOrgSecret
	if orgSecret == nil && cfg.GatewayOrgSecret != "" {
		orgSecret = []byte(cfg.GatewayOrgSecret)
	}
	if err := tools.Register(reg, tools.Deps{
		Sprites: opts.Sprites, Connectors: opts.Connectors,
		GatewayBinary: opts.GatewayBinary, GatewayOrgSecret: orgSecret,
	}); err != nil {
		pool.Close()
		return nil, fmt.Errorf("server: register studio tools: %w", err)
	}
	ex, err := executor.New(executor.Config{Pool: pool, Registry: reg, Logger: logger})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("server: dbos executor: %w", err)
	}
	approver := approvals.New(pool)
	auditor := audit.New(pool, logger)
	eff := effects.New(effects.Config{Pool: pool,
		Channels:  map[string]effects.Channel{},
		Canceller: runtime.EffectCanceller(pool),
		Policy:    effects.StrictPolicy{Pool: pool, Registry: reg, Grants: policy},
		Registry:  reg, Grants: policy, Approver: approver, Auditor: auditor,
	})
	svc, err := runtime.New(runtime.Config{
		Registry: reg, Policy: policy, Pool: pool,
		Sync: classa.New(pool, reg), Async: ex,
		Approver: approver, Auditor: auditor, Effects: eff,
	})
	if err != nil {
		_ = ex.Shutdown(10 * time.Second)
		pool.Close()
		return nil, fmt.Errorf("server: runtime: %w", err)
	}
	gwKeys := opts.GatewayKeys
	if gwKeys == nil && len(orgSecret) > 0 {
		gwKeys = gateway.KDFProvider{Secret: orgSecret}
	}
	if gwKeys == nil {
		logger.Warn("server: gateway callbacks unconfigured (no org secret)")
	}
	s := &Server{
		cfg: cfg, pool: pool, svc: svc, sessions: store.New(pool),
		tenants: map[string]Operator{}, logger: logger,
		sprites: opts.Sprites, conns: opts.Connectors, executor: ex,
		gwKeys: gwKeys,
	}
	for _, op := range cfg.Operators {
		s.tenants[op.Tenant] = op
	}
	if cfg.SpritesTokenFile != "" {
		token, err := loadTokenFile(cfg.SpritesTokenFile)
		if err != nil {
			if cerr := s.Close(); cerr != nil {
				logger.Warn("server: cleanup after failed init", "detail", cerr.Error())
			}
			return nil, fmt.Errorf("server: read sprites token: %w", err)
		}
		if s.sprites == nil {
			s.sprites = fly.NewSpriteAdapter(token, "")
		}
		if s.conns == nil {
			s.conns = fly.NewConnectorClient(token, "")
		}
	}
	s.routes()
	return s, nil
}

// loadTokenFile reads one secret file. F8: token files conventionally end in
// a newline, which must not become part of the credential.
func loadTokenFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return "", fmt.Errorf("token file %s is empty", path)
	}
	return token, nil
}

// unconfiguredBackend refuses every connector read with an honest error.
// Production connector reads arrive with the P5 gateway wiring; until then
// the runner admits specs and executes fixture-backed dry-runs, never live
// provider data.
type unconfiguredBackend struct{}

func (unconfiguredBackend) Resolve(_ context.Context, id string) (connector.Connection, error) {
	return connector.Connection{}, fmt.Errorf("server: connector %q unavailable (gateway wiring lands in P5)", id)
}

func (unconfiguredBackend) Read(_ context.Context, _ connector.Connection, _, _ string, _ runner.ReadOpts) ([]byte, error) {
	return nil, fmt.Errorf("server: connector reads unavailable (gateway wiring lands in P5)")
}

// Service exposes the canonical invocation service for handlers.
func (s *Server) Service() *runtime.Service { return s.svc }

// Sessions exposes the session store for handlers.
func (s *Server) Sessions() *store.Store { return s.sessions }

// FrameworkCredential returns the server-side Miniapp credential for one
// tenant. Handlers use it for runtime.Service invocations; it never leaves
// the server process.
func (s *Server) FrameworkCredential(tenantID string) (string, bool) {
	op, ok := s.tenants[tenantID]
	if !ok || op.FrameworkCredential == "" {
		return "", false
	}
	return op.FrameworkCredential, true
}

// Handler returns the route tree.
func (s *Server) Handler() http.Handler { return s.mux }

// Close releases server resources.
// Close shuts down the executor and pool exactly once and reports the
// first shutdown error, if any. Both halves are synchronous, so a nil
// return proves shutdown completed. Repeated and concurrent calls return
// the retained result without re-running shutdown.
func (s *Server) Close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	// The sender uses the pool: stop it before anything it touches.
	s.stopSenderLocked()
	if s.executor != nil {
		if err := s.executor.Shutdown(10 * time.Second); err != nil {
			s.closeErr = fmt.Errorf("server: executor shutdown: %w", err)
		}
	}
	if s.pool != nil {
		s.pool.Close()
	}
	return s.closeErr
}

// keyFileEntry mirrors the framework's keys-file record exactly. Studio must
// parse the identical format; inventing a parallel shape would silently
// misbind production credentials.
type keyFileEntry struct {
	Agent  string   `json:"agent"`
	Kind   string   `json:"kind"`
	ID     string   `json:"id"`
	Tenant string   `json:"tenant"`
	Roles  []string `json:"roles"`
}

func loadFrameworkPolicy(keysFile, grantsFile string) (*auth.Policy, error) {
	keysRaw, err := os.ReadFile(keysFile)
	if err != nil {
		return nil, fmt.Errorf("server: read keys file: %w", err)
	}
	grantsRaw, err := os.ReadFile(grantsFile)
	if err != nil {
		return nil, fmt.Errorf("server: read grants file: %w", err)
	}
	var keys map[string]keyFileEntry
	if err := json.Unmarshal(keysRaw, &keys); err != nil {
		return nil, fmt.Errorf("server: parse keys file: %w", err)
	}
	var grants map[string]map[string]contract.TrustMode
	if err := json.Unmarshal(grantsRaw, &grants); err != nil {
		return nil, fmt.Errorf("server: parse grants file: %w", err)
	}
	bindings := map[string]auth.Binding{}
	for cred, e := range keys {
		if cred == "" || e.Agent == "" || e.Tenant == "" {
			return nil, fmt.Errorf("server: keys file: credential, agent, and tenant are all required")
		}
		bindings[cred] = auth.Binding{AgentID: e.Agent, Authority: contract.Authority{
			Kind: contract.AuthorityKind(e.Kind), ID: e.ID, TenantID: e.Tenant,
		}, Roles: e.Roles}
	}
	policy := auth.NewPolicy(bindings, grants)
	// Fail closed on unusable bindings before serving anything.
	for cred := range keys {
		if _, err := policy.Authenticate(cred); err != nil {
			return nil, fmt.Errorf("server: keys file: %w", err)
		}
	}
	return policy, nil
}
