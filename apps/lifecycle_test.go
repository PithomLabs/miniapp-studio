package apps

import (
	"strings"
	"testing"
)

func TestNormalizeSlug(t *testing.T) {
	for _, good := range []string{"invoice-review", "a", "app-01", strings.Repeat("x", 63)} {
		if _, err := NormalizeSlug(good); err != nil {
			t.Errorf("slug %q rejected: %v", good, err)
		}
	}
	for _, bad := range []string{"", "UPPER", " has-space", "under_score", "-lead", "trail-",
		"api", "www", "admin", "studio", strings.Repeat("x", 64), "a/b", "a.b"} {
		if _, err := NormalizeSlug(bad); err == nil {
			t.Errorf("slug %q accepted, want rejection", bad)
		}
	}
}

func TestDeploymentValidate(t *testing.T) {
	good := Deployment{
		AppID: "app-1", TenantID: "t1", Slug: "invoices",
		ResourceKind: ResourceSprite, ResourceName: "invoices",
		ResourceID: "spr_1", URL: "https://sprites.test/invoices",
		Status: StatusReady, SpecHash: "abc",
	}
	if err := good.Validate(); err != nil {
		t.Errorf("valid deployment rejected: %v", err)
	}
	mutate := map[string]func(*Deployment){
		"no tenant":       func(d *Deployment) { d.TenantID = "" },
		"bad slug":        func(d *Deployment) { d.Slug = "BAD" },
		"bad kind":        func(d *Deployment) { d.ResourceKind = "vm" },
		"no resource":     func(d *Deployment) { d.ResourceID = "" },
		"bad status":      func(d *Deployment) { d.Status = "floating" },
		"http url":        func(d *Deployment) { d.URL = "http://x.test/a" },
		"empty url ready": func(d *Deployment) { d.URL = "" },
		"no hash":         func(d *Deployment) { d.SpecHash = "" },
	}
	for name, fn := range mutate {
		t.Run(name, func(t *testing.T) {
			d := good
			fn(&d)
			if err := d.Validate(); err == nil {
				t.Error("invalid deployment accepted, want rejection")
			}
		})
	}
	// Provisioning needs no URL yet.
	pending := good
	pending.Status, pending.URL = StatusProvisioning, ""
	if err := pending.Validate(); err != nil {
		t.Errorf("provisioning without URL rejected: %v", err)
	}
}

func TestCanTransition(t *testing.T) {
	allowed := [][2]DeploymentStatus{
		{StatusProvisioning, StatusReady}, {StatusProvisioning, StatusFailed},
		{StatusReady, StatusStopped}, {StatusStopped, StatusProvisioning},
		{StatusFailed, StatusProvisioning}, {StatusReady, StatusDeleted},
	}
	for _, tr := range allowed {
		if !CanTransition(tr[0], tr[1]) {
			t.Errorf("transition %s->%s refused, want allowed", tr[0], tr[1])
		}
	}
	denied := [][2]DeploymentStatus{
		{StatusDeleted, StatusReady}, {StatusDeleted, StatusProvisioning},
		{StatusReady, StatusReady}, {StatusStopped, StatusReady},
		{StatusFailed, StatusReady}, {"bogus", StatusReady},
	}
	for _, tr := range denied {
		if CanTransition(tr[0], tr[1]) {
			t.Errorf("transition %s->%s allowed, want refusal", tr[0], tr[1])
		}
	}
}

func TestCandidateNames(t *testing.T) {
	cands, err := CandidateNames("invoices")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != MaxCandidates || cands[0] != "invoices" || cands[1] != "invoices-2" {
		t.Errorf("candidates = %v...", cands[:3])
	}
	if _, err := CandidateNames("BAD"); err == nil {
		t.Error("invalid slug produced candidates, want error")
	}
}

func TestMemoryRegistrar(t *testing.T) {
	r := NewMemoryRegistrar()
	first, err := r.Reserve("t1", "invoices")
	if err != nil || first != "invoices" {
		t.Fatalf("reserve = %q, %v", first, err)
	}
	second, err := r.Reserve("t1", "invoices")
	if err != nil || second != "invoices-2" {
		t.Fatalf("reserve = %q, %v", second, err)
	}
	// Tenants have independent namespaces.
	other, err := r.Reserve("t2", "invoices")
	if err != nil || other != "invoices" {
		t.Fatalf("reserve = %q, %v", other, err)
	}
	r.Release("t1", "invoices")
	again, err := r.Reserve("t1", "invoices")
	if err != nil || again != "invoices" {
		t.Fatalf("reserve after release = %q, %v", again, err)
	}
	if _, err := r.Reserve("t1", "BAD"); err == nil {
		t.Error("invalid slug reserved, want error")
	}
}
