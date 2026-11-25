package spec

import (
	"bytes"
	"encoding/json"
	"regexp"
	"testing"
)

var rxHex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

func TestContentHashStable(t *testing.T) {
	spec, errs := Validate(loadFixture(t, "valid_invoice_review.json"))
	if len(errs) != 0 {
		t.Fatalf("valid spec rejected: %v", errs)
	}
	first, err := ContentHash(spec)
	if err != nil {
		t.Fatalf("ContentHash: %v", err)
	}
	if !rxHex64.MatchString(first) {
		t.Errorf("hash %q is not 64 hex chars", first)
	}
	second, err := ContentHash(spec)
	if err != nil || second != first {
		t.Errorf("hash unstable: %q vs %q (%v)", first, second, err)
	}
}

func TestContentHashFormattingInvariant(t *testing.T) {
	raw := loadFixture(t, "valid_invoice_review.json")
	base, errs := Validate(raw)
	if len(errs) != 0 {
		t.Fatalf("valid spec rejected: %v", errs)
	}
	want, err := ContentHash(base)
	if err != nil {
		t.Fatalf("ContentHash: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	pretty, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	reformatted, errs := Validate(pretty)
	if len(errs) != 0 {
		t.Fatalf("reformatted spec rejected: %v", errs)
	}
	got, err := ContentHash(reformatted)
	if err != nil {
		t.Fatalf("ContentHash: %v", err)
	}
	if got != want {
		t.Errorf("formatting changed hash: %s vs %s", got, want)
	}
}

func TestContentHashSensitive(t *testing.T) {
	raw := loadFixture(t, "valid_invoice_review.json")
	base, errs := Validate(raw)
	if len(errs) != 0 {
		t.Fatalf("valid spec rejected: %v", errs)
	}
	want, err := ContentHash(base)
	if err != nil {
		t.Fatalf("ContentHash: %v", err)
	}
	base.App.Name += "!"
	got, err := ContentHash(base)
	if err != nil {
		t.Fatalf("ContentHash: %v", err)
	}
	if got == want {
		t.Error("content change did not change the hash")
	}
}

func TestContentHashRoundTrip(t *testing.T) {
	raw := loadFixture(t, "valid_invoice_review.json")
	spec, errs := Validate(raw)
	if len(errs) != 0 {
		t.Fatalf("valid spec rejected: %v", errs)
	}
	hash, err := ContentHash(spec)
	if err != nil {
		t.Fatalf("ContentHash: %v", err)
	}
	spec.Metadata.ContentHash = hash
	stamped, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, errs := Validate(stamped); len(errs) != 0 {
		t.Errorf("correctly stamped spec rejected: %v", errs)
	}
	spec.Metadata.ContentHash = "ffff" + hash[4:]
	tampered, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	_, errs = Validate(tampered)
	found := false
	for _, e := range errs {
		if e.Code == EHash {
			found = true
		}
	}
	if !found {
		t.Errorf("tampered hash accepted, got %v", errs)
	}
}

func TestCanonicalJSONSorted(t *testing.T) {
	out, err := CanonicalJSON([]byte(`{"b":1,"a":{"d":4,"c":3}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, []byte(`{"a":{"c":3,"d":4},"b":1}`)) {
		t.Errorf("not canonical: %s", out)
	}
}
