// Package app is the D3 fixture artifact: a minimal Miniapp tool
// implementation exporting the conformance convention
// (Implementation). Studio-authored, reviewed, pinned in-tree.
package app

import (
	"context"
	"encoding/json"

	"github.com/PithomLabs/miniapp/registry"
)

// Implementation exposes the fixture tools for registration
// conformance. Every Phase D artifact must export this function.
func Implementation() *registry.Implementation {
	return &registry.Implementation{
		Build: "fixture",
		ClassATools: map[string]registry.ToolFunc{
			"fixture.echo":  echo,
			"fixture.probe": echo,
		},
	}
}

func echo(_ context.Context, _ registry.ToolCall) (json.RawMessage, error) {
	return json.RawMessage(`{"ok":true}`), nil
}
