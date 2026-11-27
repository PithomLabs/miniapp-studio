package dbenv

// File-based branch binding (security-ops handoff). No identifier
// ever crosses chat or clipboards: the owner-terminal `studio dev
// bind` command writes this file from `neon` CLI output, and every
// `studio dev` verb reads it back. The file carries identities only
// (project/branch IDs, hostnames) — never credentials. Anything
// malformed fails closed at load time; the endpoint policy
// re-validates at every use.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// BranchBinding is the JSON file form of an EndpointPolicy. Field
// names are fixed; unknown fields are rejected so a mistyped file
// can never silently bind the wrong branch.
type BranchBinding struct {
	ProjectID     string   `json:"project_id"`
	BranchID      string   `json:"branch_id"`
	ParentBranch  string   `json:"parent_branch,omitempty"`
	EndpointHosts []string `json:"endpoint_hosts"`
	// BoundAt records when the identity snapshot was taken (RFC3339).
	// Informational only: protection markers can change in the
	// dashboard after binding, so re-bind after toggling protection.
	// Never a guard input by itself.
	BoundAt string `json:"bound_at,omitempty"`
	// BranchName + Protected are the production-identity root: the
	// platform's own branch name and protection flag, recorded at bind
	// time. Both are required; bindings predating them fail closed
	// with a re-bind instruction (never grandfathered). Protected is a
	// pointer so an explicit JSON null fails closed exactly like a
	// missing key (a plain bool would silently read null as false).
	BranchName string `json:"branch_name"`
	Protected  *bool  `json:"protected"`
	// ProjectName is display-only (operator messages), never a guard input.
	ProjectName string `json:"project_name,omitempty"`
}

// PolicyFromFile loads and validates a branch binding file.
func PolicyFromFile(path string) (EndpointPolicy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return EndpointPolicy{}, fmt.Errorf("dbenv: read binding %q: %w", path, err)
	}
	var b BranchBinding
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil {
		return EndpointPolicy{}, fmt.Errorf("dbenv: parse binding %q: %w", path, err)
	}
	if b.ProjectID == "" {
		return EndpointPolicy{}, fmt.Errorf("dbenv: binding %q: project_id required", path)
	}
	if b.BranchID == "" {
		return EndpointPolicy{}, fmt.Errorf("dbenv: binding %q: branch_id required", path)
	}
	if strings.TrimSpace(b.BranchName) == "" {
		return EndpointPolicy{}, fmt.Errorf("dbenv: binding %q predates branch-identity metadata: re-run dev:bind to rebind", path)
	}
	// Pointer catches both a missing key and an explicit null: either
	// way the protection state is unknown and fails closed.
	if b.Protected == nil {
		return EndpointPolicy{}, fmt.Errorf("dbenv: binding %q predates branch-identity metadata: re-run dev:bind to rebind", path)
	}
	if len(b.EndpointHosts) == 0 {
		return EndpointPolicy{}, fmt.Errorf("dbenv: binding %q: endpoint_hosts required", path)
	}
	hosts := make([]string, 0, len(b.EndpointHosts))
	for _, h := range b.EndpointHosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" || strings.ContainsAny(h, " \t\n/@:") || strings.Contains(h, "://") {
			return EndpointPolicy{}, fmt.Errorf("dbenv: binding %q: bad endpoint host %q (want a bare hostname)", path, h)
		}
		if isLocalOrPrivate(h) {
			return EndpointPolicy{}, fmt.Errorf("dbenv: binding %q: endpoint host %q is not a remote branch endpoint", path, h)
		}
		hosts = append(hosts, h)
	}
	return EndpointPolicy{
		ProjectID:    b.ProjectID,
		BranchID:     b.BranchID,
		ParentBranch: b.ParentBranch,
		Hosts:        hosts,
		ProjectName:  strings.TrimSpace(b.ProjectName),
		BranchName:   strings.TrimSpace(b.BranchName),
		Protected:    *b.Protected,
	}, nil
}
