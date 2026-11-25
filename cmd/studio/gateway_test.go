package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp-studio/gateway"
	"github.com/PithomLabs/miniapp-studio/store"
)

func TestGatewayUsage(t *testing.T) {
	if err := runGateway([]string{}); err == nil {
		t.Error("empty gateway args accepted")
	}
	if err := runGateway([]string{"frobnicate"}); err == nil {
		t.Error("unknown subcommand accepted")
	}
	if err := runGatewayProvision([]string{"--tenant", "t"}); err == nil {
		t.Error("partial provision flags accepted")
	}
	if err := runGatewayUpgrade([]string{"--tenant", "t", "--build", "b"}); err == nil {
		t.Error("partial upgrade flags accepted")
	}
	if err := runGatewayCommand([]string{"--tenant", "t"}); err == nil {
		t.Error("partial command flags accepted")
	}
	if err := runGatewayCommand([]string{"--tenant", "t", "--build", "b", "--payload", "{oops"}); err == nil {
		t.Error("malformed payload accepted")
	}
	// M13: typos fail client-side, before any config or network.
	if err := runGatewayCommand([]string{"--tenant", "t", "--build", "b", "--cmd", "frobnicate"}); err == nil ||
		!strings.Contains(err.Error(), "unknown command") {
		t.Errorf("err = %v, want client-side unknown-command refusal", err)
	}
	// The new outbox commands pass the allowlist (failing later on
	// config, which proves the name was accepted).
	for _, cmd := range []string{"outbox_inspect", "outbox_requeue"} {
		err := runGatewayCommand([]string{"--tenant", "t", "--build", "b", "--cmd", cmd})
		if err == nil || strings.Contains(err.Error(), "unknown command") {
			t.Errorf("cmd %s err = %v, want past-allowlist failure", cmd, err)
		}
	}
}

func TestFormatCommandOutput(t *testing.T) {
	big := strings.Repeat("A", 5000)
	out := map[string]any{"artifacts": []any{
		map[string]any{"path": "a.diff", "kind": "diff", "digest": "sha256:x", "size": 1, "data_b64": big},
		map[string]any{"path": "b.txt", "kind": "diff", "digest": "sha256:y", "size": 1, "data_b64": "tiny"},
	}}
	got := formatCommandOutput(out)
	arts, _ := got["artifacts"].([]any)
	if len(arts) != 2 {
		t.Fatalf("artifacts = %v", got["artifacts"])
	}
	first, _ := arts[0].(map[string]any)
	stub, _ := first["data_b64"].(string)
	if !strings.Contains(stub, "stubbed 5000 bytes") || !strings.Contains(stub, "--output") {
		t.Errorf("stub = %q, want size + --output notice", stub)
	}
	if first["digest"] != "sha256:x" {
		t.Errorf("stub lost metadata: %v", first)
	}
	second, _ := arts[1].(map[string]any)
	if second["data_b64"] != "tiny" {
		t.Errorf("small body altered: %v", second)
	}
	// The input map is not mutated (the --output file keeps full bytes).
	orig, _ := out["artifacts"].([]any)[0].(map[string]any)
	if orig["data_b64"] != big {
		t.Error("input map mutated by formatting")
	}
	plain := map[string]any{"state": "running"}
	if got := formatCommandOutput(plain); got["state"] != "running" {
		t.Errorf("non-fetch response altered: %v", got)
	}
}

// cliEnv wires the server environment the gateway commands load: operator
// file with one tenant, framework keys/grants, and database URLs.
func cliEnv(t *testing.T, dsn string) {
	t.Helper()
	sum := sha256.Sum256([]byte("cli-token"))
	ops := `{"operators":[{"tenant":"t-prov","binding":"prov-op",` +
		`"login_token_sha256":"` + hex.EncodeToString(sum[:]) + `","framework_credential":"prov-svc"}]}`
	t.Setenv("STUDIO_OPERATORS_FILE", writeSecret(t, "ops", ops))
	t.Setenv("MINIAPP_KEYS_FILE", writeSecret(t, "keys",
		`{"prov-svc":{"agent":"a","kind":"service","id":"s","tenant":"t-prov"}}`))
	t.Setenv("MINIAPP_GRANTS_FILE", writeSecret(t, "grants",
		`{"t-prov":{"studio":"autonomous","studio-runner":"autonomous"}}`))
	t.Setenv("STUDIO_DATABASE_URL", dsn)
	t.Setenv("STUDIO_OWNER_DATABASE_URL", dsn)
	t.Setenv("STUDIO_GATEWAY_ORG_SECRET", "cli-org-secret-0123456789abcdef")
}

func seedBuildRow(t *testing.T, dsn, tenant, conv, build, sprite, url string) (string, string) {
	t.Helper()
	ctx := context.Background()
	owner, err := store.NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	// Suffix ids per-run: this package's setup predates Phase A tables, so
	// fixed ids would collide with prior runs' rows.
	tag, err := store.RandomHex(4)
	if err != nil {
		t.Fatal(err)
	}
	conv, build = conv+"-"+tag, build+"-"+tag
	if _, err := owner.Exec(ctx,
		`INSERT INTO studio.conversations (id, tenant_id, status) VALUES ($1, $2, 'open')`,
		conv, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx,
		`INSERT INTO studio.builds (id, tenant_id, conversation_id, status, sprite_name, gateway_url)
		 VALUES ($1, $2, $3, 'assigned', $4, $5)`, build, tenant, conv, sprite, url); err != nil {
		t.Fatal(err)
	}
	return conv, build
}

func TestGatewayProvisionUnconfigured(t *testing.T) {
	dsn := provisionDB(t)
	cliEnv(t, dsn)
	seedBuildRow(t, dsn, "t-prov", "cli-conv", "cli-build", "", "")
	bin := writeSecret(t, "gw.bin", "fake-bytes")
	// No sprite manager without STUDIO_SPRITES_TOKEN_FILE: honest refusal,
	// proving the CLI wires config → server → tool without live calls.
	err := runGatewayProvision([]string{
		"--tenant", "t-prov", "--build", "cli-build", "--gateway-binary", bin,
		"--studio-url", "https://studio.example.test",
	})
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("err = %v, want honest unconfigured refusal", err)
	}
}

// derivedKeys serves KeyFor from the test org secret, mirroring what
// provisioning would have delivered to a real builder.
type derivedKeys struct{ secret string }

func (k derivedKeys) KeyFor(buildID string) ([]byte, error) {
	return gateway.DeriveKey([]byte(k.secret), buildID), nil
}

func TestGatewayCommandRoundTrip(t *testing.T) {
	dsn := provisionDB(t)
	cliEnv(t, dsn)
	secret := "cli-org-secret-0123456789abcdef"
	sessions, err := gateway.NewSessionManager(t.TempDir() + "/sessions")
	if err != nil {
		t.Fatal(err)
	}
	outbox, err := gateway.OpenOutbox(t.TempDir() + "/outbox.json")
	if err != nil {
		t.Fatal(err)
	}
	dedup, err := gateway.OpenDedupStore(t.TempDir() + "/dedup.json")
	if err != nil {
		t.Fatal(err)
	}
	adapter := gateway.NewFakeAdapter()
	adapter.Sessions = sessions
	gw := &gateway.Server{Keys: derivedKeys{secret}, Adapter: adapter,
		Sessions: sessions, Outbox: outbox, Dedup: dedup, WorkRoot: t.TempDir()}
	httpSrv := httptest.NewServer(gw)
	defer httpSrv.Close()
	// Assign through the client so a session exists, then drive status
	// through the CLI exactly as the runbook does.
	_, build := seedBuildRow(t, dsn, "t-prov", "cli-conv-2", "cli-cmd-build", "spr-x", httpSrv.URL)
	client := &gateway.Client{}
	if _, err := client.Send(context.Background(), httpSrv.URL,
		gateway.DeriveKey([]byte(secret), build), build, gateway.CmdAssign,
		map[string]any{"prompt": "p", "workspace_id": "t-prov"}); err != nil {
		t.Fatal(err)
	}
	if err := runGatewayCommand([]string{
		"--tenant", "t-prov", "--build", build, "--cmd", "status",
	}); err != nil {
		t.Fatalf("gateway command: %v", err)
	}
}
