package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp-studio/fly"
)

func setupProvision(t *testing.T) (harness, *fly.FakeSpriteManager) {
	t.Helper()
	sprites := fly.NewFakeSpriteManager()
	h := setupToolsDeps(t, Deps{
		Sprites: sprites, Connectors: fly.NewFakeConnectorManager(),
		GatewayBinary:    []byte("fake-gateway-binary"),
		GatewayOrgSecret: []byte("test-org-secret-0123456789"),
	})
	return h, sprites
}

func TestBuildProvisionFlow(t *testing.T) {
	h, sprites := setupProvision(t)
	conv := createConversation(t, h, "pv-c", toolCredA)
	buildID, _ := createBuild(t, h, conv, "pv-b", toolCredA)
	res := invoke(t, h, ToolBuildProvision,
		map[string]any{"build_id": buildID, "studio_url": testStudioURL}, "pv-p", toolCredA)
	name, _ := res["sprite_name"].(string)
	url, _ := res["url"].(string)
	wantName := builderSpriteName(toolTenantA, "")
	if name != wantName || url == "" {
		t.Fatalf("provision = %v, want sprite %q", res, wantName)
	}
	// Sprite created with builder labels and token-only URL auth.
	sprite, ok := sprites.Sprites[name]
	if !ok {
		t.Fatalf("sprite %q not created", name)
	}
	joined := strings.Join(sprite.Labels, ",")
	tHash, aHash := builderScope(toolTenantA, "")
	for _, want := range []string{"studio-builder", "tenant-" + tHash, "app-" + aHash} {
		if !strings.Contains(joined, want) {
			t.Errorf("labels = %v", sprite.Labels)
		}
	}
	if sprite.Auth != fly.URLAuthSprite {
		t.Errorf("url auth = %q, want sprite-only", sprite.Auth)
	}
	// Install bundle staged byte-identical, including this build's key file.
	for _, file := range []string{"studio-gateway", "opencode.json", "builder.md", "setup.sh",
		"keys-" + buildID[:12] + ".json"} {
		if _, ok := sprites.Files[name+":/srv/studio-staging/"+file]; !ok {
			t.Errorf("staged file %s missing", file)
		}
	}
	if string(sprites.Files[name+":/srv/studio-staging/studio-gateway"]) != "fake-gateway-binary" {
		t.Error("staged binary mismatch")
	}
	var keysDoc map[string]string
	if err := json.Unmarshal(sprites.Files[name+":/srv/studio-staging/keys-"+buildID[:12]+".json"], &keysDoc); err != nil {
		t.Fatal(err)
	}
	if len(keysDoc[buildID]) != 64 {
		t.Errorf("staged key = %q, want 64 hex chars", keysDoc[buildID])
	}
	// A second build for the same tenant shares the builder (per-app).
	// The deployed binary matches the staged one, so the refresh path
	// (no restart) applies.
	scriptDeployedDigest(t, sprites, name, []byte("fake-gateway-binary"))
	conv2 := createConversation(t, h, "pv-c2", toolCredA)
	build2, _ := createBuild(t, h, conv2, "pv-b2", toolCredA)
	res2 := invoke(t, h, ToolBuildProvision,
		map[string]any{"build_id": build2, "studio_url": testStudioURL}, "pv-p2", toolCredA)
	if res2["sprite_name"] != name {
		t.Errorf("second build sprite = %v, want shared %q", res2["sprite_name"], name)
	}
	if len(sprites.Sprites) != 1 {
		t.Errorf("sprites = %d, want 1 shared builder", len(sprites.Sprites))
	}
	// Setup executed and gateway service running.
	foundSetup := false
	for _, exec := range sprites.Execs {
		if len(exec) == 3 && exec[1] == "sh" && strings.HasSuffix(exec[2], "setup.sh") {
			foundSetup = true
		}
	}
	if !foundSetup {
		t.Errorf("setup exec missing: %v", sprites.Execs)
	}
	svc, ok := sprites.Services[name+"/studio-gateway"]
	if !ok || svc.Status != "running" {
		t.Fatalf("service = %+v, %v", svc, ok)
	}
	// Build row records the sprite and verbatim URL.
	var recorded, recordedURL string
	err := h.owner.QueryRow(context.Background(),
		`SELECT sprite_name, gateway_url FROM studio.builds WHERE id = $1 AND tenant_id = $2`,
		buildID, toolTenantA).Scan(&recorded, &recordedURL)
	if err != nil {
		t.Fatal(err)
	}
	if recorded != name || recordedURL != url || recordedURL != sprite.URL {
		t.Errorf("row = %q %q, want %q %q", recorded, recordedURL, name, sprite.URL)
	}
}

// TestBuildProvisionServicePinned (B1): the gateway service launches
// through env with PATH pinned to trusted root-owned locations —
// agent-binary resolution and every agent child inherit it.
func TestBuildProvisionServicePinned(t *testing.T) {
	h, sprites := setupProvision(t)
	conv := createConversation(t, h, "pp-c", toolCredA)
	buildID, _ := createBuild(t, h, conv, "pp-b", toolCredA)
	res := invoke(t, h, ToolBuildProvision,
		map[string]any{"build_id": buildID, "studio_url": testStudioURL}, "pp-p", toolCredA)
	name, _ := res["sprite_name"].(string)
	req, ok := sprites.ServiceReqs[name+"/studio-gateway"]
	if !ok {
		t.Fatalf("service request for %q not recorded", name)
	}
	if req.Cmd != "/usr/bin/env" {
		t.Errorf("service cmd = %q, want /usr/bin/env wrapper", req.Cmd)
	}
	// Pinned PATH + binary + the required boot flags: a service
	// without them exits on usage error instead of serving.
	want := []string{"PATH=/usr/local/bin:/usr/bin:/bin",
		"/srv/studio-gateway/studio-gateway",
		"--state-dir", "/srv/studio-gateway/state",
		"--work-root", "/srv/studio-agent/builds",
		"--studio-url", testStudioURL,
		"--keys-dir", "/srv/studio-gateway/keys"}
	if len(req.Args) != len(want) {
		t.Fatalf("service args = %q, want %q", req.Args, want)
	}
	for i := range want {
		if req.Args[i] != want[i] {
			t.Fatalf("service args = %q, want %q", req.Args, want)
		}
	}
	if req.HTTPPort == nil || *req.HTTPPort != 8080 {
		t.Errorf("service port = %v, want 8080", req.HTTPPort)
	}
}

func TestBuildProvisionRefusals(t *testing.T) {
	h, _ := setupProvision(t)
	conv := createConversation(t, h, "pr-c", toolCredA)
	buildID, _ := createBuild(t, h, conv, "pr-b", toolCredA)
	// studio_url is required and transport-guarded: the service boots
	// with it as its callback target.
	err := invokeErr(t, h, ToolBuildProvision,
		mustJSON(t, map[string]any{"build_id": buildID}), "pr-p0", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "studio_url") {
		t.Errorf("missing studio_url = %v, want schema refusal naming studio_url", err)
	}
	err = invokeErr(t, h, ToolBuildProvision,
		mustJSON(t, map[string]any{"build_id": buildID, "studio_url": "http://studio.example.test:1"}), "pr-p0b", toolCredA)
	if !strings.Contains(err.Error(), "https") {
		t.Errorf("plaintext studio_url = %v, want transport refusal", err)
	}
	// Second provision refuses (already provisioned after the first).
	invoke(t, h, ToolBuildProvision,
		map[string]any{"build_id": buildID, "studio_url": testStudioURL}, "pr-p1", toolCredA)
	err = invokeErr(t, h, ToolBuildProvision,
		mustJSON(t, map[string]any{"build_id": buildID, "studio_url": testStudioURL}), "pr-p2", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "already provisioned") {
		t.Errorf("reprovision = %v, want already-provisioned", err)
	}
	// Non-assigned builds refuse.
	invoke(t, h, ToolBuildCancel,
		map[string]any{"build_id": buildID}, "pr-cx", toolCredA)
	conv2 := createConversation(t, h, "pr-c2", toolCredA)
	other, _ := createBuild(t, h, conv2, "pr-b2", toolCredA)
	invoke(t, h, ToolBuildCancel,
		map[string]any{"build_id": other}, "pr-cx2", toolCredA)
	err = invokeErr(t, h, ToolBuildProvision,
		mustJSON(t, map[string]any{"build_id": other, "studio_url": testStudioURL}), "pr-p3", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "only assigned builds provision") {
		t.Errorf("cancelled provision = %v", err)
	}
	// Cross-tenant and unknown builds: no oracle.
	err = invokeErr(t, h, ToolBuildProvision,
		mustJSON(t, map[string]any{"build_id": buildID, "studio_url": testStudioURL}), "pr-p4", toolCredB)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("cross-tenant = %v, want not found", err)
	}
}

func TestBuildProvisionUnconfigured(t *testing.T) {
	h := setupFakes(t)
	conv := createConversation(t, h, "pu-c", toolCredA)
	buildID, _ := createBuild(t, h, conv, "pu-b", toolCredA)
	// setupFakes injects no gateway binary: honest refusal, no sprite.
	err := invokeErr(t, h, ToolBuildProvision,
		mustJSON(t, map[string]any{"build_id": buildID, "studio_url": testStudioURL}), "pu-p", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("unconfigured = %v, want refusal", err)
	}
	if len(h.sprites.Sprites) != 0 {
		t.Errorf("sprites created while unconfigured: %v", h.sprites.Sprites)
	}
	hu := setupTools(t, nil, nil)
	convU := createConversation(t, hu, "pu-c2", toolCredA)
	buildU, _ := createBuild(t, hu, convU, "pu-b2", toolCredA)
	err = invokeErr(t, hu, ToolBuildProvision,
		mustJSON(t, map[string]any{"build_id": buildU, "studio_url": testStudioURL}), "pu-p2", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Errorf("nil-sprites = %v, want refusal", err)
	}
}

func TestBuildProvisionAttachAndRetry(t *testing.T) {
	h, sprites := setupProvision(t)
	conv := createConversation(t, h, "pa-c", toolCredA)
	buildID, _ := createBuild(t, h, conv, "pa-b", toolCredA)
	// A sprite left behind by a failed attempt is re-attached, not
	// duplicated: the deterministic name converges retries.
	name := builderSpriteName(toolTenantA, "")
	if _, err := sprites.CreateSprite(context.Background(), name, fly.SpriteOptions{}); err != nil {
		t.Fatal(err)
	}
	res := invoke(t, h, ToolBuildProvision,
		map[string]any{"build_id": buildID, "studio_url": testStudioURL}, "pa-p", toolCredA)
	if res["sprite_name"] != name {
		t.Fatalf("attach = %v", res)
	}
	if len(sprites.Sprites) != 1 {
		t.Errorf("sprites = %d, want exactly the pre-existing one", len(sprites.Sprites))
	}
	// Distinct tenants never share a builder, even for the same app key.
	otherTenantSprite := builderSpriteName(toolTenantB, "")
	if otherTenantSprite == name {
		t.Errorf("tenant collision: %q", name)
	}
	// A failed setup run leaves the build assigned and unrecorded, so a
	// retry with a fresh key converges.
	sprites.Fail["Exec"] = errFakeBoom()
	conv2 := createConversation(t, h, "pa-c2", toolCredA)
	build2, _ := createBuild(t, h, conv2, "pa-b2", toolCredA)
	err := invokeErr(t, h, ToolBuildProvision,
		mustJSON(t, map[string]any{"build_id": build2, "studio_url": testStudioURL}), "pa-p2", toolCredA)
	if err == nil {
		t.Fatal("failed setup accepted")
	}
	var spriteName string
	if err := h.owner.QueryRow(context.Background(),
		`SELECT sprite_name FROM studio.builds WHERE id = $1`, build2).Scan(&spriteName); err != nil {
		t.Fatal(err)
	}
	if spriteName != "" {
		t.Errorf("failed provision recorded sprite %q", spriteName)
	}
	delete(sprites.Fail, "Exec")
	// The retry finds the first build's running service and refreshes
	// it (digest matches: same staged binary).
	scriptDeployedDigest(t, sprites, name, []byte("fake-gateway-binary"))
	res = invoke(t, h, ToolBuildProvision,
		map[string]any{"build_id": build2, "studio_url": testStudioURL}, "pa-p3", toolCredA)
	if res["sprite_name"] != name {
		t.Errorf("retry = %v, want re-attached %q", res, name)
	}
}

// TestProvisionBundlePinned asserts the embedded install bundle's pinned
// facts: every permission key exists in the verified MCP/local tool
// vocabulary (typos would silently neuter rules), destructive tools deny,
// guidance stays off, instructions point at the shipped file, and the
// setup script pins versions with fail-closed checks.
func TestProvisionBundlePinned(t *testing.T) {
	raw, err := provisionFiles.ReadFile("provisiondata/opencode.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Plugin       []any             `json:"plugin"`
		Permission   map[string]string `json:"permission"`
		Instructions []string          `json:"instructions"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Plugin) != 1 {
		t.Fatalf("plugin entries = %d", len(doc.Plugin))
	}
	entry, _ := doc.Plugin[0].([]any)
	if len(entry) != 2 {
		t.Fatalf("plugin tuple = %v", doc.Plugin[0])
	}
	path, _ := entry[0].(string)
	opts, _ := entry[1].(map[string]any)
	if path != "file:///srv/studio-agent/plugin" || opts["guidance"] != false {
		t.Errorf("plugin = %q %v, want file checkout with guidance:false", path, opts)
	}
	verified := map[string]bool{
		"bash": true, "read": true, "edit": true, "write": true, "grep": true, "glob": true,
		"question": true, "external_directory": true,
		"sprites_list_sprites": true, "sprites_get_sprite_info": true, "sprites_get_fly_context": true,
		"sprites_create_sprite": true, "sprites_destroy_sprite": true,
		"sprites_checkpoint_list": true, "sprites_checkpoint_create": true, "sprites_checkpoint_restore": true,
		"sprites_policy_network_get": true, "sprites_policy_network_update": true,
		"sprites_exec": true, "sprites_exec_list": true, "sprites_exec_kill": true,
		"sprites_file_read": true, "sprites_file_list": true, "sprites_file_write": true,
		"sprites_file_delete": true, "sprites_file_rename": true, "sprites_file_copy": true,
		"sprites_service_create": true, "sprites_service_get": true, "sprites_service_list": true,
		"sprites_service_logs": true, "sprites_service_start": true, "sprites_service_stop": true,
	}
	for key, action := range doc.Permission {
		if !verified[key] {
			t.Errorf("permission key %q not in the verified tool vocabulary (typo?)", key)
		}
		if action != "allow" && action != "deny" {
			t.Errorf("permission %s = %q", key, action)
		}
	}
	for _, deny := range []string{"question", "external_directory", "sprites_create_sprite",
		"sprites_destroy_sprite", "sprites_checkpoint_restore", "sprites_policy_network_update",
		"sprites_exec", "sprites_file_write"} {
		if doc.Permission[deny] != "deny" {
			t.Errorf("permission %s = %q, want deny", deny, doc.Permission[deny])
		}
	}
	if doc.Permission["sprites_checkpoint_create"] != "allow" {
		t.Error("checkpoint_create must stay allowed (evidence checkpoints)")
	}
	if len(doc.Instructions) != 1 || doc.Instructions[0] != "/srv/studio-gateway/builder.md" {
		t.Errorf("instructions = %v", doc.Instructions)
	}
	builder, err := provisionFiles.ReadFile("provisiondata/builder.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{".studio/inbox/question.json", "No interactive question feature exists", "Never claim success without evidence"} {
		if !strings.Contains(string(builder), want) {
			t.Errorf("builder.md lacks %q", want)
		}
	}
	// §7 application contract: Go + Miniapp + MiniappSpec + runtime,
	// with other stacks explicitly excluded.
	for _, want := range []string{"Application contract", "Miniapp, exclusively", "Implement it in Go",
		"Miniapp framework", "MiniappSpec", "Miniapp runtime", "never become the implementation"} {
		if !strings.Contains(string(builder), want) {
			t.Errorf("builder.md lacks contract %q", want)
		}
	}
	setup, err := provisionFiles.ReadFile("provisiondata/setup.sh")
	if err != nil {
		t.Fatal(err)
	}
	script := string(setup)
	for _, want := range []string{
		"set -eu", "v1.19.6", "opencode-linux-x64.tar.gz",
		"47734e08968b30cc22a0bffb6d9feeed76ad4aa743213a12122040392d09a057",
		"f63a85a7da9fa2f42b21877ab58b35d18ccaa682",
		"studio-gw", "studio-agent", "studio-io", "sha256sum -c",
		// B1 staging lockdown + key cleanup (fail-closed assertions).
		`chmod 0700 "$STAGING"`, `chmod 0600 "$keys"`,
		`rm -f "$STAGING"/keys-*.json`, "staging key cleanup failed",
		"staging dir not 0700",
		// F7 atomic key install (live gateway re-reads per lookup).
		`rm -f "$GW_HOME/keys"/.keys-*.tmp`, `.tmp" || fail "key stage failed`,
		`mv "$GW_HOME/keys/.$base.tmp"`, `|| fail "key install failed`,
		// Service boot layout: the provisioned service args expect
		// a gateway-owned state dir.
		`mkdir -p "$GW_HOME/keys" "$GW_HOME/state"`,
		`chown "$GW_USER:$GW_USER" "$GW_HOME/keys" "$GW_HOME/state"`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("setup.sh lacks %q", want)
		}
	}
	if strings.Contains(string(builder), "unreadable\n  to you by design") ||
		strings.Contains(string(builder), "unreadable to you by design") {
		t.Error("builder.md still carries the unenforced unreadability promise")
	}
	if !strings.Contains(string(builder), "different OS user than the") {
		t.Error("builder.md lacks the enforced-identity wording")
	}
}

func TestShortIDSafe(t *testing.T) {
	if got := shortID("abc123DEF4567890abcd"); got != "abc123DEF456" {
		t.Errorf("hex id = %q, want truncated passthrough", got)
	}
	if got := shortID("ab12"); got != "ab12" {
		t.Errorf("short hex id = %q, want passthrough", got)
	}
	a, b := shortID("../../evil"), shortID("../../evil")
	if a != b || len(a) != 12 {
		t.Fatalf("non-hex id = %q, want deterministic 12-hex", a)
	}
	for _, r := range a {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			t.Fatalf("fragment %q not hex-safe", a)
		}
	}
	if shortID("") == "" {
		t.Error("empty id maps to empty fragment")
	}
}
