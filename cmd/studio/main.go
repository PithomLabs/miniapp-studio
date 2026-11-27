// Command studio runs the Miniapp Studio server and its operational
// commands. Secrets always arrive via files or environment, never flags,
// so credentials never appear in process listings.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/PithomLabs/miniapp-studio/server"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "studio: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: studio <serve|migrate|provision|gateway|dev> [flags]")
	}
	switch args[0] {
	case "serve":
		return runServe(args[1:])
	case "migrate":
		return runMigrate(args[1:])
	case "provision":
		return runProvision(args[1:])
	case "gateway":
		return runGateway(args[1:])
	case "dev":
		return runDev(args[1:])
	default:
		return fmt.Errorf("unknown command %q (want serve|migrate|provision|gateway|dev)", args[0])
	}
}

// runServe loads the environment config and serves until interrupted.
func runServe(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("serve takes no flags (configure via environment)")
	}
	cfg, err := server.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return serve(ctx, cfg)
}

// serve runs the HTTP server until ctx ends.
func serve(ctx context.Context, cfg server.Config) error {
	ln, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("studio: listen: %w", err)
	}
	return serveOn(ctx, cfg, ln)
}

// serveOn serves on an explicit listener so tests can launch the real path
// against an ephemeral port and shut it down.
func serveOn(ctx context.Context, cfg server.Config, ln net.Listener) error {
	srv, err := server.New(ctx, cfg, server.Options{})
	if err != nil {
		_ = ln.Close()
		return err
	}
	defer func() {
		if err := srv.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "studio: shutdown: %v\n", err)
		}
	}()
	// Explicit opt-in: without STUDIO_SENDER the server serves but never
	// dispatches, so command rows cannot move unexpectedly. A failing
	// sender start is fatal — silently serving without dispatch would
	// wedge every build.
	if cfg.SenderEnabled {
		if err := srv.StartSender(server.DefaultSenderConfig()); err != nil {
			_ = ln.Close()
			_ = srv.Close()
			return fmt.Errorf("studio: sender: %w", err)
		}
	}
	httpSrv := &http.Server{
		Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdown)
	}()
	if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("studio: serve: %w", err)
	}
	return nil
}
