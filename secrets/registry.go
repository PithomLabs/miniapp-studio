// Package secrets is the single configuration contract for Miniapp
// Studio: a canonical variable inventory, one validating parser, one
// loader with explicit precedence, and redaction. Secrets may come
// from process env, _FILE siblings, or dotenv files — never flags,
// never defaults, never code. Every failure names the key, never
// the value.
package secrets

import (
	"fmt"
	"regexp"
	"sort"
)

// Scope separates who a variable belongs to. Deployment values are
// ambient to the process; tenant values must arrive brokered (never
// ambient) once the broker exists — the registry already enforces
// the distinction.
type Scope string

const (
	ScopeDeployment Scope = "deployment"
	ScopeTenant     Scope = "tenant"
)

// Sensitivity separates values that must never appear in output.
type Sensitivity string

const (
	SensitivitySecret Sensitivity = "secret"
	SensitivityConfig Sensitivity = "config"
)

// Variable is one registry entry: metadata only, values never.
type Variable struct {
	Name         string
	Scope        Scope
	Sensitivity  Sensitivity
	Rotation     string // static | rotating | session
	Consumers    []string
	Environments []string
	Default      string // non-secrets only; enforced at registration
}

// Registry is the closed set of variables the platform reads.
type Registry struct {
	vars map[string]Variable
}

var rxVarName = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// NewRegistry validates and freezes a variable inventory.
func NewRegistry(vars []Variable) (*Registry, error) {
	r := &Registry{vars: map[string]Variable{}}
	for _, v := range vars {
		if !rxVarName.MatchString(v.Name) {
			return nil, fmt.Errorf("secrets: invalid variable name %q", v.Name)
		}
		if _, dup := r.vars[v.Name]; dup {
			return nil, fmt.Errorf("secrets: duplicate variable %q", v.Name)
		}
		switch v.Scope {
		case ScopeDeployment, ScopeTenant:
		default:
			return nil, fmt.Errorf("secrets: %s: scope must be deployment or tenant", v.Name)
		}
		switch v.Sensitivity {
		case SensitivitySecret, SensitivityConfig:
		default:
			return nil, fmt.Errorf("secrets: %s: sensitivity must be secret or config", v.Name)
		}
		switch v.Rotation {
		case "static", "rotating", "session":
		default:
			return nil, fmt.Errorf("secrets: %s: rotation must be static, rotating, or session", v.Name)
		}
		if v.Sensitivity == SensitivitySecret && v.Default != "" {
			return nil, fmt.Errorf("secrets: %s: secrets must never carry defaults", v.Name)
		}
		r.vars[v.Name] = v
	}
	return r, nil
}

// Lookup resolves one variable or reports it unknown.
func (r *Registry) Lookup(name string) (Variable, bool) {
	v, ok := r.vars[name]
	return v, ok
}

// Names returns registered names in sorted order.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.vars))
	for name := range r.vars {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// StudioRegistry is the canonical inventory for the Studio server
// surface. Tenant-scoped entries arrive with the credential broker;
// the scope exists now so nothing ambient slips in later.
func StudioRegistry() *Registry {
	r, err := NewRegistry([]Variable{
		{Name: "STUDIO_DATABASE_URL", Scope: ScopeDeployment, Sensitivity: SensitivitySecret, Rotation: "static", Consumers: []string{"server"}, Environments: []string{"dev", "preview", "staging", "production"}},
		{Name: "STUDIO_OWNER_DATABASE_URL", Scope: ScopeDeployment, Sensitivity: SensitivitySecret, Rotation: "static", Consumers: []string{"server", "migrate"}, Environments: []string{"dev", "preview", "staging", "production"}},
		{Name: "STUDIO_GATEWAY_ORG_SECRET", Scope: ScopeDeployment, Sensitivity: SensitivitySecret, Rotation: "rotating", Consumers: []string{"server"}, Environments: []string{"dev", "preview", "staging", "production"}},
		{Name: "STUDIO_OPERATORS_FILE", Scope: ScopeDeployment, Sensitivity: SensitivityConfig, Rotation: "static", Consumers: []string{"server"}, Environments: []string{"dev", "preview", "staging", "production"}},
		{Name: "MINIAPP_KEYS_FILE", Scope: ScopeDeployment, Sensitivity: SensitivityConfig, Rotation: "static", Consumers: []string{"server"}, Environments: []string{"dev", "preview", "staging", "production"}},
		{Name: "MINIAPP_GRANTS_FILE", Scope: ScopeDeployment, Sensitivity: SensitivityConfig, Rotation: "static", Consumers: []string{"server"}, Environments: []string{"dev", "preview", "staging", "production"}},
		{Name: "STUDIO_SPRITES_TOKEN_FILE", Scope: ScopeDeployment, Sensitivity: SensitivityConfig, Rotation: "rotating", Consumers: []string{"server"}, Environments: []string{"dev", "preview", "staging", "production"}},
		{Name: "STUDIO_FLY_ORG", Scope: ScopeDeployment, Sensitivity: SensitivityConfig, Rotation: "static", Consumers: []string{"server"}, Environments: []string{"dev", "preview", "staging", "production"}},
		{Name: "STUDIO_LISTEN", Scope: ScopeDeployment, Sensitivity: SensitivityConfig, Rotation: "static", Consumers: []string{"server"}, Environments: []string{"dev", "preview", "staging", "production"}, Default: "127.0.0.1:8080"},
		{Name: "STUDIO_SENDER", Scope: ScopeDeployment, Sensitivity: SensitivityConfig, Rotation: "static", Consumers: []string{"server"}, Environments: []string{"dev", "preview", "staging", "production"}},
		{Name: "STUDIO_WORKSPACE_ROOT", Scope: ScopeDeployment, Sensitivity: SensitivityConfig, Rotation: "static", Consumers: []string{"server"}, Environments: []string{"dev", "preview", "staging", "production"}},
	})
	if err != nil {
		panic("secrets: studio registry invalid: " + err.Error())
	}
	return r
}
