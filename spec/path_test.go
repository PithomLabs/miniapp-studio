package spec

import (
	"strings"
	"testing"
)

// TestStepPathTraversalRejected proves step request paths cannot escape the
// declared allowlist via dot-segments or encoded equivalents.
func TestStepPathTraversalRejected(t *testing.T) {
	evil := []string{
		"/x/../admin",
		"/files/../../admin/purge",
		"/%2e%2e/admin",
		"/%2E%2E/admin",
		"/files%2f..%2fadmin",
		`/files\..\admin`,
		"/files?x=1",
		"/files#frag",
		"/files /x",
	}
	for _, path := range evil {
		t.Run(path, func(t *testing.T) {
			spec := validSkeleton()
			spec.Capabilities.Operations[0].Paths = []string{"/*"}
			spec.Workflows[0].Steps[0].Path = path
			errs := validateStruct(t, spec)
			if !hasCode(errs, ECapability) {
				t.Errorf("path %q accepted, want E_CAPABILITY, got %v", path, errs)
			}
		})
	}
}

// TestTargetTraversalRejected proves component targets reject traversal.
func TestTargetTraversalRejected(t *testing.T) {
	for _, target := range []string{"/../evil", "/%2e%2e/evil", `/\evil`} {
		spec := validSkeleton()
		spec.UI.Pages[0].Components[0].Target = target
		errs := validateStruct(t, spec)
		if !hasCode(errs, EComponent) {
			t.Errorf("target %q accepted, want E_COMPONENT, got %v", target, errs)
		}
	}
}

// TestPathNormalization proves benign redundancies normalize to the same
// authorized representation used for allowlist matching.
func TestPathNormalization(t *testing.T) {
	got, err := NormalizePath("/files//weekly/")
	if err != nil {
		t.Fatalf("NormalizePath: %v", err)
	}
	if got != "/files/weekly" {
		t.Errorf("normalized = %q, want %q", got, "/files/weekly")
	}
	if _, err := NormalizePath("/"); err != nil {
		t.Errorf("root rejected: %v", err)
	}
}

// TestAllowlistMatchesNormalizedForm proves authorization compares the
// normalized path, never the raw input.
func TestAllowlistMatchesNormalizedForm(t *testing.T) {
	spec := validSkeleton()
	spec.Capabilities.Operations[0].Paths = []string{"/files/weekly"}
	spec.Workflows[0].Steps[0].Path = "/files//weekly/"
	errs := validateStruct(t, spec)
	for _, e := range errs {
		if e.Code == ECapability && strings.Contains(e.Path, ".path") {
			t.Fatalf("normalized in-allowlist path rejected: %v", errs)
		}
	}
}
