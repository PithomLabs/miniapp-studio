// Command gateway runs the Builder Sprite's Agent Gateway: the command
// handler Studio drives, the supervised coding agent behind it, and the
// durable callback outbox back to Studio. Configuration arrives via flags;
// secrets arrive via files only, never flags or environment values.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/PithomLabs/miniapp-studio/gateway"
)

type config struct {
	listen    string
	stateDir  string
	workRoot  string
	studio    string
	keysDir   string
	binPath   string
	agentUser string
	ioGroup   string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "gateway: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var cfg config
	fs := flag.NewFlagSet("gateway", flag.ContinueOnError)
	fs.StringVar(&cfg.listen, "listen", "127.0.0.1:8080", "address to serve")
	fs.StringVar(&cfg.stateDir, "state-dir", "", "state directory (required)")
	fs.StringVar(&cfg.workRoot, "work-root", "", "agent workdir root (default <state>/work)")
	fs.StringVar(&cfg.studio, "studio-url", "", "Studio base URL for callbacks (required)")
	fs.StringVar(&cfg.keysDir, "keys-dir", "", "build HMAC keys dir, one 0600 JSON per build (required)")
	fs.StringVar(&cfg.binPath, "opencode-bin", gateway.OpenCodeBinary, "agent binary")
	fs.StringVar(&cfg.agentUser, "agent-user", "studio-agent", "OS user the agent runs as")
	fs.StringVar(&cfg.ioGroup, "io-group", "studio-io", "inbox group shared by gateway and agent")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if cfg.stateDir == "" || cfg.studio == "" || cfg.keysDir == "" {
		return fmt.Errorf("usage: gateway --state-dir DIR --studio-url URL --keys-dir DIR [--listen ADDR]")
	}
	if cfg.workRoot == "" {
		cfg.workRoot = filepath.Join(cfg.stateDir, "work")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return serve(ctx, cfg)
}

// resolveAgent maps the configured agent user and inbox group to the
// numeric credential the supervisor drops to. Any lookup failure refuses
// the boot: an agent that cannot be confined must never start.
func resolveAgent(agentUser, ioGroup string) (gateway.AgentCredential, error) {
	u, err := user.Lookup(agentUser)
	if err != nil {
		return gateway.AgentCredential{}, fmt.Errorf("gateway: agent user %q: %w", agentUser, err)
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil || uid < 0 {
		return gateway.AgentCredential{}, fmt.Errorf("gateway: agent user %q has bad uid", agentUser)
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil || gid < 0 {
		return gateway.AgentCredential{}, fmt.Errorf("gateway: agent user %q has bad gid", agentUser)
	}
	g, err := user.LookupGroup(ioGroup)
	if err != nil {
		return gateway.AgentCredential{}, fmt.Errorf("gateway: io group %q: %w", ioGroup, err)
	}
	ioGID, err := strconv.Atoi(g.Gid)
	if err != nil || ioGID < 0 {
		return gateway.AgentCredential{}, fmt.Errorf("gateway: io group %q has bad gid", ioGroup)
	}
	groupIDs, err := u.GroupIds()
	if err != nil {
		return gateway.AgentCredential{}, fmt.Errorf("gateway: agent groups: %w", err)
	}
	groups := make([]uint32, 0, len(groupIDs))
	for _, gs := range groupIDs {
		n, err := strconv.Atoi(strings.TrimSpace(gs))
		if err != nil || n < 0 {
			return gateway.AgentCredential{}, fmt.Errorf("gateway: agent group %q invalid", gs)
		}
		groups = append(groups, uint32(n))
	}
	return gateway.AgentCredential{
		UID: uint32(uid), GID: uint32(gid), Groups: groups,
		IOGID: uint32(ioGID), Set: true,
	}, nil
}

// checkAgentDistinct refuses boot when the configured agent identity
// equals the service's own. Production has no same-user fallback:
// the supervisor independently refuses such spawns (serve never sets
// AllowSameUID), and this check fails first with a boot diagnostic.
func checkAgentDistinct(runner *gateway.Supervisor) error {
	if runner.Agent.UID == uint32(os.Geteuid()) {
		return fmt.Errorf("gateway: boot refused: agent uid %d equals service uid (no same-user fallback; run the service privileged with a distinct agent user)", runner.Agent.UID)
	}
	return nil
}

// checkAgentSpawn is the boot-time credential-drop self-test: it
// launches a harmless probe through the production supervisor using
// the exact credential configuration real agent processes use, and
// refuses boot unless the transition succeeds. The probe binary is a
// parameter so tests can observe the child's identity; production
// always passes "true". No keys are accessed or exposed.
func checkAgentSpawn(ctx context.Context, runner *gateway.Supervisor, probe string, args ...string) error {
	p, err := runner.Start(ctx, "boot-selftest", os.TempDir(), probe, args, nil)
	if err != nil {
		return fmt.Errorf("gateway: boot refused: agent spawn failed (%v); the service lacks privilege to drop to the agent identity", err)
	}
	code, err := p.Wait()
	if err != nil || code != 0 {
		return fmt.Errorf("gateway: boot refused: self-test probe exited %d (%v)", code, err)
	}
	return nil
}

// serve wires the gateway and runs until ctx ends: recover pre-restart
// sessions first (so no detached agent keeps working silently), then
// serve commands while the deliverer drains callbacks.
func serve(ctx context.Context, cfg config) error {
	// Validate once at boot (fail closed on a broken keys dir), then
	// resolve per lookup so provisioned keys land without a restart.
	if _, err := gateway.LoadKeysDir(cfg.keysDir); err != nil {
		return err
	}
	keys := gateway.ReloadingKeys{Dir: cfg.keysDir}
	sessions, err := gateway.NewSessionManager(filepath.Join(cfg.stateDir, "sessions"))
	if err != nil {
		return err
	}
	outbox, err := gateway.OpenOutbox(filepath.Join(cfg.stateDir, "outbox.json"))
	if err != nil {
		return err
	}
	dedup, err := gateway.OpenDedupStore(filepath.Join(cfg.stateDir, "dedup.json"))
	if err != nil {
		return err
	}
	cred, err := resolveAgent(cfg.agentUser, cfg.ioGroup)
	if err != nil {
		return err
	}
	runner := gateway.NewSupervisor()
	runner.Agent = cred
	// Boot-time privilege self-test (N1): prove the service can
	// actually drop to the agent identity before doing any recovery
	// work or reporting ready. No readiness path bypasses this —
	// serve() is the only route to listen().
	if err := checkAgentDistinct(runner); err != nil {
		return err
	}
	if err := checkAgentSpawn(ctx, runner, "true"); err != nil {
		return err
	}
	adapter := &gateway.OpenCodeAdapter{
		Runner: runner, Sessions: sessions, BinPath: cfg.binPath,
		AgentUID: int(cred.UID), IOGID: int(cred.IOGID),
	}
	srv := &gateway.Server{
		Keys: keys, Adapter: adapter, Sessions: sessions,
		Outbox: outbox, Dedup: dedup, WorkRoot: cfg.workRoot,
		IOGID: int(cred.IOGID),
	}
	if ticks, err := gateway.StartTicks("/proc", os.Getpid()); err != nil {
		fmt.Fprintf(os.Stderr, "gateway: WARNING: cannot read own start time (%v); orphan reaping loses start-time validation\n", err)
	} else {
		srv.SelfTicks = ticks
	}
	srv.RecoverSessions(ctx)
	if err := gateway.RequireSecureTransport(cfg.studio); err != nil {
		return err
	}
	deliverer := &gateway.Deliverer{Outbox: outbox, Studio: cfg.studio, Keys: keys}
	go deliverer.Run(ctx)
	ln, err := net.Listen("tcp", cfg.listen)
	if err != nil {
		return fmt.Errorf("gateway: listen: %w", err)
	}
	err = serveOn(ctx, ln, srv)
	shutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adapter.Shutdown(shutdown)
	return err
}

// serveOn serves one gateway on an explicit listener until ctx ends.
// Tests pass their own listener to learn its port.
func serveOn(ctx context.Context, ln net.Listener, srv *gateway.Server) error {
	mux := http.NewServeMux()
	mux.Handle("/gw/v1/command", srv)
	httpSrv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdown)
	}()
	if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("gateway: serve: %w", err)
	}
	return nil
}
