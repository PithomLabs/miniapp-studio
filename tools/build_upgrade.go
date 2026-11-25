package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/PithomLabs/miniapp-studio/gateway"
	"github.com/PithomLabs/miniapp/registry"
)

type buildUpgradeIn struct {
	BuildID   string `json:"build_id"`
	StudioURL string `json:"studio_url"`
	Force     bool   `json:"force"`
}

// buildUpgrade is the explicit, guarded gateway replacement operation
// (F7): provisioning never restarts a healthy service as a side effect,
// so binary replacement happens here, with eyes open. It verifies every
// build on the builder is idle (no live agent run) before replacing,
// unless force explicitly accepts the disruption. Replacement is
// stage + setup + delete + create + running-verify; the digest of the
// deployed bytes is echoed in the result (M14, print + record).
func (d Deps) buildUpgrade(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in buildUpgradeIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.BuildID == "" {
		return nil, errInvalidArguments()
	}
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
	if b.SpriteName == "" {
		return nil, fmt.Errorf("studio: build is not provisioned")
	}
	name := b.SpriteName
	if err := lockSprite(ctx, call.Tx, name); err != nil {
		return nil, err
	}
	checked, err := d.verifyBuilderIdle(ctx, call, tenant, name, in.Force)
	if err != nil {
		return nil, err
	}
	key := gateway.DeriveKey(d.GatewayOrgSecret, in.BuildID)
	if err := d.stageAndSetup(ctx, name, in.BuildID, key); err != nil {
		return nil, err
	}
	_ = d.Sprites.DeleteService(ctx, name, "studio-gateway")
	if err := d.Sprites.CreateService(ctx, name, "studio-gateway", gatewayServiceRequest(in.StudioURL)); err != nil {
		return nil, flyErr(err)
	}
	if err := d.verifyRunning(ctx, name); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(d.GatewayBinary)
	return marshalMap(map[string]any{
		"build_id": in.BuildID, "sprite_name": name, "upgraded": true,
		"gateway_sha256": hex.EncodeToString(sum[:]),
		"force":          in.Force, "idle_checked": checked,
	})
}

// verifyBuilderIdle confirms no build on the sprite has a live agent
// run, consulting each build's gateway status with its own derived key.
// Unreachable gateways and live runs refuse unless force accepts them;
// the count of verified-idle builds is returned for the result record.
func (d Deps) verifyBuilderIdle(ctx context.Context, call registry.ToolCall, tenant, name string, force bool) (int, error) {
	rows, err := call.Tx.Query(ctx,
		`SELECT id, gateway_url FROM studio.builds WHERE tenant_id = $1 AND sprite_name = $2`,
		tenant, name)
	if err != nil {
		return 0, dbErr(err, "")
	}
	type target struct{ id, url string }
	var targets []target
	for rows.Next() {
		var t target
		if err := rows.Scan(&t.id, &t.url); err != nil {
			rows.Close()
			return 0, dbErr(err, "")
		}
		targets = append(targets, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, dbErr(err, "")
	}
	client := &gateway.Client{}
	checked := 0
	for _, t := range targets {
		if t.url == "" {
			continue
		}
		key := gateway.DeriveKey(d.GatewayOrgSecret, t.id)
		out, err := client.Send(ctx, t.url, key, t.id, gateway.CmdStatus, map[string]any{})
		if err != nil {
			if !force {
				return 0, fmt.Errorf("studio: cannot verify build %s is idle: %s", t.id, capped(err.Error(), 200))
			}
			continue
		}
		if alive, _ := out["agent_alive"].(bool); alive {
			if !force {
				return 0, fmt.Errorf("studio: build %s has a live agent run; drain it or repeat with force", t.id)
			}
			continue
		}
		checked++
	}
	return checked, nil
}
