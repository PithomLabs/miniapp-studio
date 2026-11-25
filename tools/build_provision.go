package tools

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/fly"
	"github.com/PithomLabs/miniapp-studio/gateway"
	"github.com/PithomLabs/miniapp/registry"
)

// provisionFiles is the pinned install bundle deployed to every Builder
// Sprite: the agent config, its instructions, and the idempotent setup
// script. Embedded so the deployed files always match this code — no
// drift between repo and sprite, no fetch at provision time.
//
//go:embed provisiondata/opencode.json provisiondata/builder.md provisiondata/setup.sh
var provisionFiles embed.FS

// stagingDir is where install files land inside the builder (mirrored by
// setup.sh, which moves them into place with ownership and modes).
const stagingDir = "/srv/studio-staging"

// builderSpriteName derives the deterministic builder name for one tenant
// application (gateway plan §10: builders are per-application and
// long-lived across builds). Tenant and application hash to fixed
// collision-resistant prefixes, so arbitrary workspace names can never
// collide or smuggle path characters; empty applications share the
// tenant's default builder. Stable across retries: a failed provision
// re-attaches instead of orphaning a sprite per attempt.
func builderSpriteName(tenantID, appID string) string {
	t, a := builderScope(tenantID, appID)
	return "studio-bld-" + t + "-" + a
}

// lockSprite takes the per-sprite advisory lock (two-key form, per the
// established codebase pattern), held for the whole provision/upgrade
// operation. Serve and operator CLI transact against the same
// framework DB, so the lock coordinates across processes, not just
// goroutines; xact scope releases it at commit/rollback.
func lockSprite(ctx context.Context, tx pgx.Tx, name string) error {
	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtext('gateway-sprite'), hashtext($1))`, name); err != nil {
		return dbErr(err, "")
	}
	return nil
}

// builderScope hashes tenant and application to fixed prefixes for names
// and labels. Empty applications share the tenant's default builder.
func builderScope(tenantID, appID string) (string, string) {
	if appID == "" {
		appID = "default"
	}
	sum := func(s string) string {
		digest := sha256.Sum256([]byte(s))
		return hex.EncodeToString(digest[:])[:8]
	}
	return sum(tenantID), sum(appID)
}

type buildProvisionIn struct {
	BuildID   string `json:"build_id"`
	StudioURL string `json:"studio_url"`
}

// buildProvision provisions one Builder Sprite and installs the gateway:
// ensure sprite (attach if the deterministic name exists), enforce
// token-only URL auth, stage install files, run setup, refresh the
// gateway service, verify running, record sprite + URL on the build.
//
// Failure leaves the build assigned with no sprite recorded: every step
// is idempotent (attach, overwrite, delete-before-create), so retrying
// with a fresh key converges instead of duplicating. Requires a
// configured SpriteManager and gateway binary — both absent in
// unconfigured environments, where the tool refuses honestly.
func (d Deps) buildProvision(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in buildProvisionIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.BuildID == "" {
		return nil, errInvalidArguments()
	}
	// The Studio base URL (callback target) is an explicit per-provision
	// operator decision (runbook D3: tunnel vs Fly-hosted Studio), not
	// ambient config: the provisioned service cannot boot without it.
	if in.StudioURL == "" {
		return nil, fmt.Errorf("studio: studio_url is required")
	}
	if err := gateway.RequireSecureTransport(in.StudioURL); err != nil {
		return nil, err
	}
	if d.Sprites == nil {
		return nil, fmt.Errorf("studio: sprite provisioning not configured")
	}
	if len(d.GatewayBinary) == 0 {
		return nil, fmt.Errorf("studio: gateway binary not configured")
	}
	if len(d.GatewayOrgSecret) == 0 {
		return nil, fmt.Errorf("studio: gateway keys not configured")
	}
	b, err := requireBuild(ctx, call.Tx, tenant, in.BuildID, true)
	if err != nil {
		return nil, err
	}
	if b.SpriteName != "" {
		return nil, fmt.Errorf("studio: build already provisioned")
	}
	if b.Status != buildAssigned {
		return nil, fmt.Errorf("studio: only assigned builds provision, build is %s", b.Status)
	}
	name := builderSpriteName(tenant, b.ApplicationID)
	if err := lockSprite(ctx, call.Tx, name); err != nil {
		return nil, err
	}
	key := gateway.DeriveKey(d.GatewayOrgSecret, in.BuildID)
	sprite, err := d.ensureSprite(ctx, tenant, b.ApplicationID, name)
	if err != nil {
		return nil, err
	}
	if sprite.URL == "" {
		return nil, fmt.Errorf("studio: sprite %q has no URL", name)
	}
	if err := d.installGateway(ctx, name, in.BuildID, key, in.StudioURL); err != nil {
		return nil, err
	}
	if _, err := call.Tx.Exec(ctx,
		`UPDATE studio.builds SET sprite_name = $1, gateway_url = $2, updated_at = now()
		 WHERE id = $3 AND tenant_id = $4`, name, sprite.URL, in.BuildID, tenant); err != nil {
		return nil, dbErr(err, "")
	}
	sum := sha256.Sum256(d.GatewayBinary)
	return marshalMap(map[string]any{
		"build_id": in.BuildID, "sprite_name": name, "url": sprite.URL,
		"gateway_sha256": hex.EncodeToString(sum[:]),
	})
}

// ensureSprite attaches to the deterministic sprite or creates it,
// enforcing token-only URL auth either way.
func (d Deps) ensureSprite(ctx context.Context, tenant, appID, name string) (fly.Sprite, error) {
	sprite, err := d.Sprites.GetSprite(ctx, name)
	if err == nil {
		return d.lockURL(ctx, name, sprite)
	}
	var missing *fly.NotFoundError
	if !errors.As(err, &missing) {
		return fly.Sprite{}, flyErr(err)
	}
	t, a := builderScope(tenant, appID)
	sprite, err = d.Sprites.CreateSprite(ctx, name, fly.SpriteOptions{
		Labels: []string{"studio-builder", "tenant-" + t, "app-" + a},
	})
	if err != nil {
		return fly.Sprite{}, flyErr(err)
	}
	return d.lockURL(ctx, name, sprite)
}

func (d Deps) lockURL(ctx context.Context, name string, sprite fly.Sprite) (fly.Sprite, error) {
	if err := d.Sprites.UpdateURLSettings(ctx, name, fly.URLSettings{Auth: fly.URLAuthSprite}); err != nil {
		return fly.Sprite{}, flyErr(err)
	}
	sprite.Auth = fly.URLAuthSprite
	return sprite, nil
}

// installGateway inspects the gateway service BEFORE staging anything
// and acts non-destructively (F7): absent → fresh install; healthy →
// digest-verified key refresh with no restart; unhealthy or
// uninspectable → fail closed without touching the service. Keys stage
// per build so shared builders accumulate one file per build, never a
// read-modify-write.
func (d Deps) installGateway(ctx context.Context, name, buildID string, key []byte, studioURL string) error {
	state, err := d.Sprites.GetService(ctx, name, "studio-gateway")
	if err == nil {
		if state.Status != "running" {
			return fmt.Errorf("studio: gateway service is %q, want running (diagnose the builder; replacement is an explicit gateway upgrade)", state.Status)
		}
		return d.refreshGateway(ctx, name, buildID, key)
	}
	var missing *fly.NotFoundError
	if !errors.As(err, &missing) {
		return fmt.Errorf("studio: inspect gateway service: %s", capped(err.Error(), 300))
	}
	if err := d.stageAndSetup(ctx, name, buildID, key); err != nil {
		return err
	}
	// Clear any half-created residue from a prior failed install, then
	// create (fresh only — refresh never reaches here).
	_ = d.Sprites.DeleteService(ctx, name, "studio-gateway")
	if err := d.Sprites.CreateService(ctx, name, "studio-gateway", gatewayServiceRequest(studioURL)); err != nil {
		return flyErr(err)
	}
	return d.verifyRunning(ctx, name)
}

// refreshGateway refreshes a healthy running service without restarting
// it: the staged binary must digest-match the deployed one (drift is an
// explicit upgrade, never a silent side effect), then setup installs the
// new build's key file, which the live gateway picks up without a
// restart (ReloadingKeys).
func (d Deps) refreshGateway(ctx context.Context, name, buildID string, key []byte) error {
	want := sha256.Sum256(d.GatewayBinary)
	res, err := d.Sprites.Exec(ctx, name, []string{"sha256sum", "/srv/studio-gateway/studio-gateway"})
	if err != nil {
		return fmt.Errorf("studio: cannot verify deployed gateway binary: %s", capped(err.Error(), 300))
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("studio: cannot verify deployed gateway binary (exit %d): %s", res.ExitCode, capped(res.Stderr, 300))
	}
	fields := strings.Fields(res.Stdout)
	if len(fields) == 0 {
		return fmt.Errorf("studio: cannot verify deployed gateway binary: empty checksum output")
	}
	if !strings.EqualFold(fields[0], hex.EncodeToString(want[:])) {
		return fmt.Errorf("studio: deployed gateway binary differs from staged (deployed %.12s…); run gateway upgrade to replace it explicitly", fields[0])
	}
	if err := d.stageAndSetup(ctx, name, buildID, key); err != nil {
		return err
	}
	return d.verifyRunning(ctx, name)
}

// stageAndSetup stages the install bundle (binary, config,
// instructions, setup, and this build's HMAC key file) and runs setup.
func (d Deps) stageAndSetup(ctx context.Context, name, buildID string, key []byte) error {
	keysDoc, err := json.Marshal(map[string]string{buildID: hex.EncodeToString(key)})
	if err != nil {
		return fmt.Errorf("studio: encode keys: %w", err)
	}
	staged := map[string][]byte{
		"studio-gateway":                     d.GatewayBinary,
		"keys-" + shortID(buildID) + ".json": keysDoc,
	}
	for _, file := range []string{"opencode.json", "builder.md", "setup.sh"} {
		raw, err := provisionFiles.ReadFile("provisiondata/" + file)
		if err != nil {
			return fmt.Errorf("studio: install bundle: %w", err)
		}
		staged[file] = raw
	}
	for file, data := range staged {
		if err := d.Sprites.WriteFile(ctx, name, stagingDir+"/"+file, data); err != nil {
			return flyErr(err)
		}
	}
	res, err := d.Sprites.Exec(ctx, name, []string{"sh", stagingDir + "/setup.sh"})
	if err != nil {
		return flyErr(err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("studio: setup failed (exit %d): %s", res.ExitCode, capped(res.Stderr, 300))
	}
	return nil
}

// gatewayServiceRequest is the pinned service shape: env-wrapped so the
// service environment pins PATH to trusted root-owned executable
// locations (B1): the gateway resolves the agent binary through this
// PATH, and every agent child inherits it. The binary's required flags
// ride along — a service without them exits on usage error, which is
// why studioURL is a required provision input rather than ambient
// config. Agent identity flags use their built-in studio-agent /
// studio-io defaults (setup.sh guarantees those names).
func gatewayServiceRequest(studioURL string) fly.ServiceRequest {
	return fly.ServiceRequest{
		Cmd: "/usr/bin/env",
		Args: []string{
			"PATH=/usr/local/bin:/usr/bin:/bin",
			"/srv/studio-gateway/studio-gateway",
			"--state-dir", "/srv/studio-gateway/state",
			"--work-root", "/srv/studio-agent/builds",
			"--studio-url", studioURL,
			"--keys-dir", "/srv/studio-gateway/keys",
		},
		HTTPPort: intPtr(8080),
	}
}

// verifyRunning confirms the gateway service reports running (behavioral
// proof that a refresh preserved it or an install started it).
func (d Deps) verifyRunning(ctx context.Context, name string) error {
	state, err := d.Sprites.GetService(ctx, name, "studio-gateway")
	if err != nil {
		return flyErr(err)
	}
	if state.Status != "running" {
		return fmt.Errorf("studio: gateway service is %q, want running", state.Status)
	}
	return nil
}

// shortID derives the key-file name fragment for a build. Studio mints
// hex ids today, but filenames must never trust that: hex ids pass
// through truncated, anything else hashes to a fixed hex fragment
// (deterministic, filename-safe, no traversal by construction).
func shortID(id string) string {
	hexOnly := len(id) > 0
	for _, r := range id {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			hexOnly = false
			break
		}
	}
	if hexOnly {
		if len(id) > 12 {
			return id[:12]
		}
		return id
	}
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])[:12]
}

func intPtr(n int) *int { return &n }

// flyErr maps control-plane failures to a retryable, capped error.
// Bodies are capped for readable invocation records; credential
// non-echo is UNVERIFIED (staged WriteFile payloads are key bytes and
// sprites-go error shapes were not audited) — inspect one staged
// failure output live before claiming secret-freedom.
func flyErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("studio: fly: %s", capped(err.Error(), 300))
}

func capped(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
