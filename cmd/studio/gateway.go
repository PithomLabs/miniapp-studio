package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/PithomLabs/miniapp-studio/gateway"
	"github.com/PithomLabs/miniapp-studio/server"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/tools"
)

// runGateway dispatches operator gateway commands: provision a builder for
// a build, send one signed command to a provisioned builder's gateway, or
// explicitly upgrade a builder's gateway binary. All exist so the live
// runbook verifies real behavior without guessing; none executes
// anything without explicit operator invocation.
func runGateway(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: studio gateway <provision|command|upgrade> [flags]")
	}
	switch args[0] {
	case "provision":
		return runGatewayProvision(args[1:])
	case "command":
		return runGatewayCommand(args[1:])
	case "upgrade":
		return runGatewayUpgrade(args[1:])
	default:
		return fmt.Errorf("unknown gateway command %q (want provision|command|upgrade)", args[0])
	}
}

func runGatewayUpgrade(args []string) error {
	var tenant, build, binaryPath, studioURL string
	var force bool
	fs := flag.NewFlagSet("gateway upgrade", flag.ContinueOnError)
	fs.StringVar(&tenant, "tenant", "", "operator tenant id (required)")
	fs.StringVar(&build, "build", "", "provisioned build id resolving the builder (required)")
	fs.StringVar(&binaryPath, "gateway-binary", "", "replacement gateway server binary (required)")
	fs.StringVar(&studioURL, "studio-url", "", "Studio base URL for gateway callbacks (required: runbook D3)")
	fs.BoolVar(&force, "force", false, "replace even with live agent runs or unreachable gateways")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if tenant == "" || build == "" || binaryPath == "" || studioURL == "" {
		return fmt.Errorf("gateway upgrade needs --tenant, --build, --gateway-binary, and --studio-url")
	}
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("gateway upgrade: read binary: %w", err)
	}
	cfg, err := server.Load()
	if err != nil {
		return err
	}
	if cfg.GatewayOrgSecret == "" {
		return fmt.Errorf("gateway upgrade: STUDIO_GATEWAY_ORG_SECRET (or _FILE) is required")
	}
	ctx := context.Background()
	srv, err := server.New(ctx, cfg, server.Options{GatewayBinary: binary})
	if err != nil {
		return err
	}
	defer func() {
		if cerr := srv.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "studio: shutdown: %v\n", cerr)
		}
	}()
	key, err := freshOpKey()
	if err != nil {
		return err
	}
	out, err := srv.RunTool(ctx, tenant, tools.ToolBuildUpgrade,
		map[string]any{"build_id": build, "studio_url": studioURL, "force": force}, key)
	if err != nil {
		return fmt.Errorf("gateway upgrade: %w", err)
	}
	pretty, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(pretty))
	return nil
}

func runGatewayProvision(args []string) error {
	var tenant, build, binaryPath, studioURL string
	fs := flag.NewFlagSet("gateway provision", flag.ContinueOnError)
	fs.StringVar(&tenant, "tenant", "", "operator tenant id (required)")
	fs.StringVar(&build, "build", "", "build id to provision for (required)")
	fs.StringVar(&binaryPath, "gateway-binary", "", "gateway server binary to deploy (required)")
	fs.StringVar(&studioURL, "studio-url", "", "Studio base URL for gateway callbacks (required: runbook D3)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if tenant == "" || build == "" || binaryPath == "" || studioURL == "" {
		return fmt.Errorf("gateway provision needs --tenant, --build, --gateway-binary, and --studio-url")
	}
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("gateway provision: read binary: %w", err)
	}
	cfg, err := server.Load()
	if err != nil {
		return err
	}
	if cfg.GatewayOrgSecret == "" {
		return fmt.Errorf("gateway provision: STUDIO_GATEWAY_ORG_SECRET (or _FILE) is required")
	}
	ctx := context.Background()
	srv, err := server.New(ctx, cfg, server.Options{GatewayBinary: binary})
	if err != nil {
		return err
	}
	defer func() {
		if cerr := srv.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "studio: shutdown: %v\n", cerr)
		}
	}()
	key, err := freshOpKey()
	if err != nil {
		return err
	}
	out, err := srv.RunTool(ctx, tenant, tools.ToolBuildProvision,
		map[string]any{"build_id": build, "studio_url": studioURL}, key)
	if err != nil {
		return fmt.Errorf("gateway provision: %w", err)
	}
	pretty, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(pretty))
	return nil
}

func runGatewayCommand(args []string) error {
	var tenant, build, name, payloadJSON, outputPath string
	var turn int
	fs := flag.NewFlagSet("gateway command", flag.ContinueOnError)
	fs.StringVar(&tenant, "tenant", "", "operator tenant id (required)")
	fs.StringVar(&build, "build", "", "build id (required)")
	fs.StringVar(&name, "cmd", "status", "command: assign|continue|cancel|status|fetch_artifact|outbox_inspect|outbox_requeue")
	fs.StringVar(&payloadJSON, "payload", "{}", "command payload as JSON object")
	fs.IntVar(&turn, "turn", 0, "turn number for continue (required: exactly the session's next turn)")
	fs.StringVar(&outputPath, "output", "", "write the full response JSON here (fetch_artifact bodies stay out of the terminal)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if tenant == "" || build == "" {
		return fmt.Errorf("gateway command needs --tenant and --build")
	}
	// Client-side command allowlist (M13): typos fail here, before any
	// config load or network — never only at the gateway.
	switch name {
	case "assign", "continue", "cancel", "status", "fetch_artifact",
		"outbox_inspect", "outbox_requeue":
	default:
		return fmt.Errorf("unknown command %q (want assign|continue|cancel|status|fetch_artifact|outbox_inspect|outbox_requeue)", name)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return fmt.Errorf("gateway command: bad payload JSON: %w", err)
	}
	// Turn fencing (M11): continue carries its turn explicitly — the
	// gateway refuses absent or stale turns. A --turn flag and a
	// payload turn must agree when both are given.
	if pt, ok := payload["turn"]; ok && turn != 0 {
		if f, ok := pt.(float64); !ok || int(f) != turn {
			return fmt.Errorf("gateway command: --turn %d disagrees with payload turn %v", turn, pt)
		}
	}
	if turn != 0 {
		payload["turn"] = turn
	}
	if name == "continue" {
		if _, ok := payload["turn"]; !ok {
			return fmt.Errorf("gateway command: continue needs --turn (the session's next turn)")
		}
	}
	cfg, err := server.Load()
	if err != nil {
		return err
	}
	if cfg.GatewayOrgSecret == "" {
		return fmt.Errorf("gateway command: STUDIO_GATEWAY_ORG_SECRET (or _FILE) is required")
	}
	if cfg.OwnerDatabaseURL == "" {
		return fmt.Errorf("gateway command: STUDIO_OWNER_DATABASE_URL is required")
	}
	ctx := context.Background()
	owner, err := store.NewPool(ctx, cfg.OwnerDatabaseURL, 0)
	if err != nil {
		return fmt.Errorf("gateway command: owner pool: %w", err)
	}
	defer owner.Close()
	var url, sprite string
	if err := owner.QueryRow(ctx,
		`SELECT gateway_url, sprite_name FROM studio.builds WHERE tenant_id = $1 AND id = $2`,
		tenant, build).Scan(&url, &sprite); err != nil {
		return fmt.Errorf("gateway command: build has no provisioned gateway (provision first)")
	}
	key := gateway.DeriveKey([]byte(cfg.GatewayOrgSecret), build)
	client := &gateway.Client{}
	out, err := client.Send(ctx, url, key, build, name, payload)
	if err != nil {
		return fmt.Errorf("gateway command: %w", err)
	}
	if outputPath != "" {
		raw, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return fmt.Errorf("gateway command: encode response: %w", err)
		}
		if err := os.WriteFile(outputPath, raw, 0o600); err != nil {
			return fmt.Errorf("gateway command: write output: %w", err)
		}
		fmt.Printf("wrote full response to %s\n", outputPath)
	}
	pretty, _ := json.MarshalIndent(formatCommandOutput(out), "", "  ")
	fmt.Println(string(pretty))
	return nil
}

// maxInlineB64 bounds artifact bytes echoed to the terminal (M13): a
// fetch_artifact response can carry 50MB of base64. Bodies past the cap
// print as a stub naming size + digest; --output captures the whole
// response to a file.
const maxInlineB64 = 1024

// formatCommandOutput renders a command response for terminal display,
// stubbing oversized embedded artifact bodies with an explicit notice.
func formatCommandOutput(out map[string]any) map[string]any {
	rawArts, ok := out["artifacts"].([]any)
	if !ok {
		return out
	}
	shown := make([]any, 0, len(rawArts))
	for _, entry := range rawArts {
		m, ok := entry.(map[string]any)
		if !ok {
			shown = append(shown, entry)
			continue
		}
		body, _ := m["data_b64"].(string)
		if len(body) <= maxInlineB64 {
			shown = append(shown, entry)
			continue
		}
		stub := map[string]any{}
		for k, v := range m {
			if k != "data_b64" {
				stub[k] = v
			}
		}
		stub["data_b64"] = fmt.Sprintf("<stubbed %d bytes; see --output>", len(body))
		shown = append(shown, stub)
	}
	dup := map[string]any{}
	for k, v := range out {
		dup[k] = v
	}
	dup["artifacts"] = shown
	return dup
}

// freshOpKey mints one unpredictable operation key for operator commands.
func freshOpKey() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("studio: random key: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}
