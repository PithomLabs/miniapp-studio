package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/PithomLabs/miniapp/registry"
)

type connectorsListIn struct {
	Provider string `json:"provider"`
}

// connectorsList reports configured org connections through the injected
// manager. A nil manager yields an explicit unconfigured state — never an
// error, never fabricated connections — so the UI can explain itself.
func (d Deps) connectorsList(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	if _, err := tenantOf(call.TenantID); err != nil {
		return nil, err
	}
	var in connectorsListIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if d.Connectors == nil {
		return marshalMap(map[string]any{"status": "unconfigured", "connections": []any{}})
	}
	conns, err := d.Connectors.ListConnections(call.Context, in.Provider)
	if err != nil {
		return nil, fmt.Errorf("studio: connector list unavailable")
	}
	out := make([]any, 0, len(conns))
	for _, c := range conns {
		out = append(out, map[string]any{
			"id": c.ID, "provider": c.Provider, "type": c.ConnectionType,
			"policy": map[string]any{
				"allow_all": c.AccessPolicy.AllowAll,
				"allowed":   c.AccessPolicy.AllowedEndpoints,
				"blocked":   c.AccessPolicy.BlockedEndpoints,
				"prefix":    c.AccessPolicy.NamePrefix,
				"labels":    c.AccessPolicy.SpriteLabels,
			},
		})
	}
	return marshalMap(map[string]any{"status": "ok", "connections": out})
}

type attestIn struct {
	ID             string `json:"id"`
	Provider       string `json:"provider"`
	ConnectionType string `json:"type"`
}

// connectorsCustomAttest records that an out-of-band Custom API creation
// completed and verifies the result against the live connection list. The
// credential-bearing create/test calls themselves run in server handlers
// with memory-only secrets — never as tool inputs, which the framework
// persists — and this tool attests only their sanitized outcome.
func (d Deps) connectorsCustomAttest(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	if _, err := tenantOf(call.TenantID); err != nil {
		return nil, err
	}
	var in attestIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.ID == "" || len(in.ID) > 128 || in.Provider == "" || len(in.Provider) > 64 {
		return nil, errInvalidArguments()
	}
	switch in.ConnectionType {
	case "oauth", "api_key", "provisioned", "internal", "database":
	default:
		return nil, errInvalidArguments()
	}
	if d.Connectors == nil {
		return nil, fmt.Errorf("studio: connector verification unavailable")
	}
	conns, err := d.Connectors.ListConnections(call.Context, "")
	if err != nil {
		return nil, fmt.Errorf("studio: connector verification unavailable")
	}
	for _, c := range conns {
		if c.ID == in.ID {
			if c.Provider != in.Provider {
				return nil, fmt.Errorf("studio: connection mismatch")
			}
			return marshalMap(map[string]any{
				"recorded": true, "id": c.ID, "provider": c.Provider,
			})
		}
	}
	return nil, fmt.Errorf("studio: unknown connection")
}
