package runner

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/PithomLabs/miniapp-studio/spec"
)

func loadValidSpec(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("../spec/testdata/valid_invoice_review.json")
	if err != nil {
		t.Fatalf("read valid fixture: %v", err)
	}
	return raw
}

func TestAdmitValid(t *testing.T) {
	raw := loadValidSpec(t)
	got, err := Admit(raw, "acme-workspace")
	if err != nil {
		t.Fatalf("Admit valid: %v", err)
	}
	if got.Hash() == "" || got.Tenant() != "acme-workspace" {
		t.Errorf("admission = %+v", got)
	}
	parsed, err := got.Spec()
	if err != nil {
		t.Fatalf("Spec(): %v", err)
	}
	if parsed.App.ID != "invoice-review" {
		t.Errorf("app ID = %q", parsed.App.ID)
	}
	want, err := spec.ContentHash(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hash() != want {
		t.Error("admitted hash does not match content hash")
	}
}

func TestAdmitRejectsInvalid(t *testing.T) {
	raw, err := os.ReadFile("../spec/testdata/invalid_unapproved_send.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Admit(raw, "acme"); err == nil {
		t.Error("Admit accepted an invalid spec")
	}
}

func TestAdmitRejectsTenantMismatch(t *testing.T) {
	if _, err := Admit(loadValidSpec(t), "other-tenant"); err == nil {
		t.Error("Admit accepted a foreign-tenant spec")
	}
	if _, err := Admit(loadValidSpec(t), ""); err == nil {
		t.Error("Admit accepted an empty tenant")
	}
}

func TestAdmittedSpecMutationIndependent(t *testing.T) {
	got, err := Admit(loadValidSpec(t), "acme-workspace")
	if err != nil {
		t.Fatal(err)
	}
	mut, err := got.Spec()
	if err != nil {
		t.Fatal(err)
	}
	mut.App.Name = "tampered"
	mut.Capabilities.Operations[0].Method = "POST"
	if err := got.Revalidate(); err != nil {
		t.Errorf("mutating a decoded copy affected admission: %v", err)
	}
	fresh, err := got.Spec()
	if err != nil {
		t.Fatal(err)
	}
	if fresh.App.Name == "tampered" {
		t.Error("admitted value mutated through decoded copy")
	}
}

func TestRevalidateDetectsTampering(t *testing.T) {
	got, err := Admit(loadValidSpec(t), "acme-workspace")
	if err != nil {
		t.Fatal(err)
	}
	got.raw[100] ^= 0xff
	if err := got.Revalidate(); err == nil {
		t.Error("Revalidate accepted tampered bytes")
	}
}

func TestAdmitHashBinding(t *testing.T) {
	raw := loadValidSpec(t)
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	// A stamped hash must verify; admission binds the run to it.
	parsed, errs := spec.Validate(raw)
	if len(errs) != 0 {
		t.Fatalf("fixture invalid: %v", errs)
	}
	hash, err := spec.ContentHash(parsed)
	if err != nil {
		t.Fatal(err)
	}
	doc["metadata"].(map[string]any)["contentHash"] = hash
	stamped, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Admit(stamped, "acme-workspace")
	if err != nil {
		t.Fatalf("Admit stamped: %v", err)
	}
	if got.Hash() != hash {
		t.Error("admitted hash differs from stamped hash")
	}
}
