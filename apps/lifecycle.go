package apps

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var rxSlug = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// MaxSlugLen bounds user slugs; resource names derived from longer input are
// hashed, never truncated ambiguously.
const MaxSlugLen = 63

// MaxCandidates bounds automatic collision suffixes (slug, slug-2, ...).
const MaxCandidates = 100

// reservedSlugs can never identify an application: they collide with
// platform routes, infrastructure names, or phishing-sensitive hosts.
var reservedSlugs = map[string]bool{
	"api": true, "www": true, "admin": true, "studio": true,
	"fly": true, "sprite": true, "sprites": true, "miniapp": true,
	"mail": true, "app": true, "apps": true, "static": true,
	"auth": true, "login": true, "oauth": true, "health": true,
	"healthz": true, "ready": true, "metrics": true, "internal": true,
}

// NormalizeSlug validates a user-supplied slug: lowercase DNS-label form,
// length-bounded, and not reserved. It returns the canonical slug.
func NormalizeSlug(slug string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(slug))
	if s != slug {
		return "", fmt.Errorf("apps: slug must be lowercase without surrounding whitespace")
	}
	if len(s) == 0 || len(s) > MaxSlugLen || !rxSlug.MatchString(s) {
		return "", fmt.Errorf("apps: slug must be a DNS label (lowercase alphanumerics and '-', 1-%d chars)", MaxSlugLen)
	}
	if reservedSlugs[s] {
		return "", fmt.Errorf("apps: slug %q is reserved", s)
	}
	return s, nil
}

// ResourceKind names the Fly target family hosting one deployment.
type ResourceKind string

// Supported deployment targets.
const (
	ResourceSprite = "sprite"
	ResourceFlyApp = "fly-app"
)

// DeploymentStatus is one deployment lifecycle state.
type DeploymentStatus string

// Lifecycle states. Terminal: Ready (serving), Failed (recorded), Deleted.
const (
	StatusProvisioning DeploymentStatus = "provisioning"
	StatusReady        DeploymentStatus = "ready"
	StatusFailed       DeploymentStatus = "failed"
	StatusStopped      DeploymentStatus = "stopped"
	StatusDeleted      DeploymentStatus = "deleted"
)

// CanTransition reports whether a lifecycle move is legal. Deleted is
// terminal; failed deployments may return to provisioning for retry.
func CanTransition(from, to DeploymentStatus) bool {
	switch from {
	case StatusProvisioning:
		return to == StatusReady || to == StatusFailed || to == StatusDeleted
	case StatusReady:
		return to == StatusStopped || to == StatusFailed || to == StatusDeleted || to == StatusProvisioning
	case StatusStopped:
		return to == StatusProvisioning || to == StatusDeleted
	case StatusFailed:
		return to == StatusProvisioning || to == StatusDeleted
	default:
		return false
	}
}

// Deployment is the persisted identity of one deployed generated app.
type Deployment struct {
	AppID        string
	TenantID     string
	Slug         string
	ResourceKind ResourceKind
	ResourceName string
	ResourceID   string
	// URL is the Fly-issued address, stored verbatim from the authoritative
	// API response. It is never constructed by string concatenation.
	URL      string
	Status   DeploymentStatus
	SpecHash string
}

// Validate checks a deployment record: identity present, tenant-scoped,
// URL an https URL when serving, status known.
func (d Deployment) Validate() error {
	if d.AppID == "" || d.TenantID == "" {
		return fmt.Errorf("apps: app ID and tenant ID are required")
	}
	if _, err := NormalizeSlug(d.Slug); err != nil {
		return err
	}
	if d.ResourceKind != ResourceSprite && d.ResourceKind != ResourceFlyApp {
		return fmt.Errorf("apps: unknown resource kind %q", d.ResourceKind)
	}
	if d.ResourceName == "" || d.ResourceID == "" {
		return fmt.Errorf("apps: resource name and ID are required")
	}
	switch d.Status {
	case StatusProvisioning, StatusReady, StatusFailed, StatusStopped, StatusDeleted:
	default:
		return fmt.Errorf("apps: unknown status %q", d.Status)
	}
	if d.Status == StatusReady || d.Status == StatusStopped {
		u, err := url.Parse(d.URL)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return fmt.Errorf("apps: serving deployments need an https URL")
		}
	}
	if d.SpecHash == "" {
		return fmt.Errorf("apps: spec hash is required")
	}
	return nil
}

// CandidateNames yields deterministic resource names for one slug: the bare
// slug first, then slug-2, slug-3, ... up to MaxCandidates. The registrar
// tries them in order against live Fly state.
func CandidateNames(slug string) ([]string, error) {
	base, err := NormalizeSlug(slug)
	if err != nil {
		return nil, err
	}
	out := []string{base}
	for i := 2; i <= MaxCandidates; i++ {
		out = append(out, fmt.Sprintf("%s-%d", base, i))
	}
	return out, nil
}

// Registrar reserves resource names. Production implementations persist
// reservations; the in-memory registrar serves tests and documents the
// contract: reserve-then-verify against Fly, never assume.
type Registrar interface {
	// Reserve returns the first free candidate name for slug, or an error
	// when all candidates are taken.
	Reserve(tenantID, slug string) (string, error)
	// Release frees one reservation.
	Release(tenantID, name string)
}

// MemoryRegistrar is an in-memory Registrar for tests.
type MemoryRegistrar struct {
	taken map[string]bool
}

// NewMemoryRegistrar returns an empty registrar.
func NewMemoryRegistrar() *MemoryRegistrar {
	return &MemoryRegistrar{taken: map[string]bool{}}
}

// Reserve implements Registrar.
func (m *MemoryRegistrar) Reserve(tenantID, slug string) (string, error) {
	cands, err := CandidateNames(slug)
	if err != nil {
		return "", err
	}
	for _, name := range cands {
		key := tenantID + "\x00" + name
		if !m.taken[key] {
			m.taken[key] = true
			return name, nil
		}
	}
	return "", fmt.Errorf("apps: no free resource name for slug %q", slug)
}

// Release implements Registrar.
func (m *MemoryRegistrar) Release(tenantID, name string) {
	delete(m.taken, tenantID+"\x00"+name)
}
