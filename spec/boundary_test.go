package spec

import (
	"encoding/json"
	"testing"
)

// TestInvalidSpecMustNotBeUsed proves the F1 validation boundary: a document
// that fails validation must never yield a usable spec value.
func TestInvalidSpecMustNotBeUsed(t *testing.T) {
	spec, errs := Validate(loadFixture(t, "invalid_unapproved_send.json"))
	if len(errs) == 0 {
		t.Fatal("unapproved send accepted, want rejection")
	}
	if spec != nil {
		t.Error("Validate returned a usable spec alongside errors")
	}
	for _, name := range []string{"invalid_capability.json", "invalid_secret.json"} {
		spec, errs := Validate(loadFixture(t, name))
		if len(errs) == 0 {
			t.Fatalf("%s accepted, want rejection", name)
		}
		if spec != nil {
			t.Errorf("%s: Validate returned a usable spec alongside errors", name)
		}
	}
}

// TestValidateSpecStructLevel proves constructed or transformed structs face
// the same validation as parsed documents.
func TestValidateSpecStructLevel(t *testing.T) {
	good, errs := Validate(loadFixture(t, "valid_invoice_review.json"))
	if len(errs) != 0 || good == nil {
		t.Fatalf("valid spec rejected: %v", errs)
	}
	if errs := ValidateSpec(good); len(errs) != 0 {
		t.Errorf("ValidateSpec rejected a valid struct: %v", errs)
	}
	raw, err := json.Marshal(good)
	if err != nil {
		t.Fatal(err)
	}
	var bad MiniappSpec
	if err := json.Unmarshal(raw, &bad); err != nil {
		t.Fatal(err)
	}
	bad.Capabilities.Operations[0].Method = "POST"
	bad.Capabilities.Operations[0].Effect = "write"
	errs = ValidateSpec(&bad)
	if len(errs) == 0 {
		t.Fatal("ValidateSpec accepted a programmatically built write-op struct")
	}
	found := false
	for _, e := range errs {
		if e.Code == ECapability {
			found = true
		}
	}
	if !found {
		t.Errorf("want E_CAPABILITY violations, got %v", errs)
	}
	if errs := ValidateSpec(nil); len(errs) == 0 {
		t.Error("ValidateSpec(nil) accepted, want rejection")
	}
}
