package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/runtime"

	"github.com/PithomLabs/miniapp-studio/tools"
)

// invokeTool is the only path from HTTP handlers to consequential work: it
// resolves the tenant's server-side framework credential, invokes the named
// Studio tool with the caller's idempotency key, and decodes a successful
// result. Tenant identity comes from the session context, never arguments.
func (s *Server) invokeTool(ctx context.Context, r *http.Request, tool string, input any, key string) (map[string]any, error) {
	tenant, _, _, ok := TenantOf(r)
	if !ok || tenant == "" {
		return nil, fmt.Errorf("server: missing tenant context")
	}
	return s.invokeToolTenant(ctx, tenant, tool, input, key)
}

// invokeToolTenant is invokeTool for callers holding an already-verified
// tenant, such as background workers provisioned from an authenticated
// request. The tenant must still originate server-side, never from input.
// RunTool invokes one Studio tool for an explicit tenant outside any HTTP
// request. It exists for operator commands (the gateway runbook drives
// provisioning through it); HTTP handlers use the request-scoped variant.
func (s *Server) RunTool(ctx context.Context, tenant, tool string, input any, key string) (map[string]any, error) {
	return s.invokeToolTenant(ctx, tenant, tool, input, key)
}

func (s *Server) invokeToolTenant(ctx context.Context, tenant, tool string, input any, key string) (map[string]any, error) {
	if tenant == "" {
		return nil, fmt.Errorf("server: missing tenant context")
	}
	cred, ok := s.FrameworkCredential(tenant)
	if !ok {
		return nil, fmt.Errorf("server: no framework credential for tenant")
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("server: encode input: %w", err)
	}
	out, err := s.svc.Invoke(ctx, runtime.InvokeRequest{
		MiniappID: tools.MiniappID, Version: tools.Version, Tool: tool,
		Input: raw, IdempotencyKey: key,
		Credential: cred, RequestedTrust: contract.TrustAutonomous,
	})
	if err != nil {
		return nil, err
	}
	if out.Status != contract.RunSucceeded {
		return nil, fmt.Errorf("server: tool %s ended %s", tool, out.Status)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out.Result, &decoded); err != nil {
		return nil, fmt.Errorf("server: decode result: %w", err)
	}
	return decoded, nil
}

// newOpKey mints one fresh, unpredictable idempotency key per intentional
// operation. Forms render it into a hidden field; resubmissions reuse it
// and attach to the original outcome, while fresh renders stay independent.
// The key identifies the intention; the framework binds it to the payload
// fingerprint and rejects same-key-different-payload as a conflict.
func newOpKey() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("server: random: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}

// opKey extracts the required idempotency key from a mutating form. Missing
// keys fail closed: every intentional write must name its operation.
func opKey(r *http.Request) (string, error) {
	key := r.Form.Get("opkey")
	if len(key) < 32 || len(key) > 128 {
		return "", fmt.Errorf("missing operation key")
	}
	for _, c := range key {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return "", fmt.Errorf("malformed operation key")
		}
	}
	return key, nil
}
