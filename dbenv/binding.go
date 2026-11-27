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
	}, nil
}
