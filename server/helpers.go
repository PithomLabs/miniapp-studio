package server

import (
	"encoding/json"
	"strings"

	"github.com/PithomLabs/miniapp-studio/runner"
)

// admitForTenant admits a pasted spec for dry-run preview. It enforces the
// identical boundary as execution: validated, tenant-bound, hash-pinned.
func admitForTenant(raw []byte, tenant string) (runner.ValidatedSpec, error) {
	return runner.Admit(raw, tenant)
}

// dryRunForTenant executes a fixture-backed dry-run and shapes the result
// for templates.
func dryRunForTenant(vspec runner.ValidatedSpec, fixture, workflow string) (map[string]any, error) {
	res, err := runner.DryRun(vspec, fixture, workflow)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// friendlyToolError maps tool failures to user-safe messages. Tool errors
// are safe by construction (static strings, never database or provider
// text); anything else becomes a generic failure so framework internals,
// SQL states, and transport details never reach the browser.
func friendlyToolError(err error) string {
	msg := err.Error()
	if idx := strings.Index(msg, "studio: "); idx >= 0 {
		return msg[idx:]
	}
	return "Operation failed. Retry, or contact the operator if it persists."
}
