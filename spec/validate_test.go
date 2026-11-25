package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return raw
}

func codesOf(errs []Error) map[Code]int {
	out := map[Code]int{}
	for _, e := range errs {
		out[e.Code]++
	}
	return out
}

func TestValidInvoiceReview(t *testing.T) {
	spec, errs := Validate(loadFixture(t, "valid_invoice_review.json"))
	if len(errs) != 0 {
		t.Fatalf("valid spec rejected: %v", errs)
	}
	if spec.App.ID != "invoice-review" {
		t.Errorf("app ID = %q", spec.App.ID)
	}
	if len(spec.Workflows) != 2 {
		t.Errorf("workflows = %d, want 2", len(spec.Workflows))
	}
	if len(spec.UI.Pages) != 2 {
		t.Errorf("pages = %d, want 2", len(spec.UI.Pages))
	}
}

func TestInvalidFixtures(t *testing.T) {
	cases := []struct {
		file string
		want []Code
	}{
		{"invalid_version.json", []Code{ESchemaVersion}},
		{"invalid_unknown_section.json", []Code{ESchema}},
		{"invalid_app.json", []Code{EApp}},
		{"invalid_secret.json", []Code{ESecret, ESchema}},
		{"invalid_component.json", []Code{EComponent, ECapability}},
		{"invalid_capability.json", []Code{ECapability}},
		{"invalid_binding.json", []Code{EBinding}},
		{"invalid_workflow.json", []Code{EWorkflow, EBinding}},
		{"invalid_data.json", []Code{EData, EBinding, ECapability}},
		{"invalid_policy.json", []Code{EPolicy, EBinding}},
		{"invalid_fixtures_theme.json", []Code{EFixture, EPresentation}},
		{"invalid_hash.json", []Code{EHash}},
		{"invalid_unapproved_send.json", []Code{ECapability}},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			_, errs := Validate(loadFixture(t, tc.file))
			if len(errs) == 0 {
				t.Fatalf("%s accepted, want rejection", tc.file)
			}
			got := codesOf(errs)
			for _, want := range tc.want {
				if got[want] == 0 {
					t.Errorf("%s: missing code %s, got %v", tc.file, want, errs)
				}
			}
		})
	}
}

// TestDeliberateRejection is the demo safety beat: an unapproved mass-send
// must be refused with explicit method/effect violations.
func TestDeliberateRejection(t *testing.T) {
	_, errs := Validate(loadFixture(t, "invalid_unapproved_send.json"))
	if len(errs) == 0 {
		t.Fatal("unapproved send accepted, want rejection")
	}
	var sawMethod, sawEffect bool
	for _, e := range errs {
		if e.Code != ECapability {
			t.Errorf("unexpected code %s (%s)", e.Code, e.Path)
		}
		if strings.HasSuffix(e.Path, ".method") {
			sawMethod = true
		}
		if strings.HasSuffix(e.Path, ".effect") {
			sawEffect = true
		}
		if !strings.Contains(e.Error(), string(e.Code)) {
			t.Errorf("Error() %q missing its code", e.Error())
		}
	}
	if !sawMethod || !sawEffect {
		t.Errorf("want method+effect violations, got %v", errs)
	}
}

func TestDeterminism(t *testing.T) {
	for _, name := range []string{"valid_invoice_review.json", "invalid_capability.json", "invalid_binding.json"} {
		raw := loadFixture(t, name)
		_, first := Validate(raw)
		_, second := Validate(raw)
		if len(first) != len(second) {
			t.Fatalf("%s: %d vs %d errors", name, len(first), len(second))
		}
		for i := range first {
			if first[i] != second[i] {
				t.Fatalf("%s: error %d differs: %v vs %v", name, i, first[i], second[i])
			}
		}
	}
}

func TestMalformedJSON(t *testing.T) {
	spec, errs := Validate([]byte("{bad json"))
	if spec != nil {
		t.Error("spec non-nil for malformed JSON")
	}
	if len(errs) != 1 || errs[0].Code != ESchema {
		t.Errorf("want single E_SCHEMA, got %v", errs)
	}
}

func TestSecretScanCaseInsensitive(t *testing.T) {
	raw := `{"app": {"appVersion": "1.0.0", "id": "d", "name": "D", "owner": "o", "slug": "d"},
		"capabilities": {"operations": []}, "data": {"entities": []},
		"fixtures": [{"name": "f1", "Password": "hunter2"}],
		"metadata": {"schemaVersion": "miniappspec/v1"},
		"policies": {"confirmation": {"externalWrites": "disabled"}},
		"ui": {"pages": [{"components": [], "id": "h", "path": "/", "title": "H"}]}}`
	_, errs := Validate([]byte(raw))
	found := false
	for _, e := range errs {
		if e.Code == ESecret && strings.Contains(e.Path, "Password") {
			found = true
		}
	}
	if !found {
		t.Errorf("Password field not flagged, got %v", errs)
	}
}
