package spec

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

// TestCanonicalJSONNumberNormalization locks the documented number model:
// alternate spellings of the same value canonicalize identically.
func TestCanonicalJSONNumberNormalization(t *testing.T) {
	a, err := CanonicalJSON([]byte(`{"n":1e3}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := CanonicalJSON([]byte(`{"n":1000}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Errorf("1e3 vs 1000 canonicalize differently: %s vs %s", a, b)
	}
}

// TestContentHashDoesNotMutate locks non-mutation of the hashed spec.
func TestContentHashDoesNotMutate(t *testing.T) {
	spec, errs := Validate(loadFixture(t, "valid_invoice_review.json"))
	if len(errs) != 0 || spec == nil {
		t.Fatalf("valid spec rejected: %v", errs)
	}
	before, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ContentHash(spec); err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	if err := json.Unmarshal(before, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(after, &b); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Error("ContentHash mutated its input")
	}
}

// TestOwnerPresenceVariants locks the owner contract: absent, null, and
// blank owners are rejected; any present label (including untrimmed or
// mixed-case spellings, whose canonical form is server-defined) is accepted.
func TestOwnerPresenceVariants(t *testing.T) {
	base := func() map[string]any {
		var doc map[string]any
		raw := loadFixture(t, "valid_invoice_review.json")
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		return doc
	}
	encode := func(doc map[string]any) []byte {
		raw, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	for name, mutate := range map[string]func(map[string]any){
		"absent": func(doc map[string]any) {
			delete(doc["app"].(map[string]any), "owner")
		},
		"null": func(doc map[string]any) {
			doc["app"].(map[string]any)["owner"] = nil
		},
		"blank": func(doc map[string]any) {
			doc["app"].(map[string]any)["owner"] = "   "
		},
	} {
		t.Run("reject/"+name, func(t *testing.T) {
			doc := base()
			mutate(doc)
			_, errs := Validate(encode(doc))
			if !hasCode(errs, EApp) {
				t.Errorf("owner %s accepted, want E_APP, got %v", name, errs)
			}
		})
	}
	for _, owner := range []string{"acme-workspace", " acme", "ACME"} {
		t.Run("accept/"+owner, func(t *testing.T) {
			doc := base()
			doc["app"].(map[string]any)["owner"] = owner
			if _, errs := Validate(encode(doc)); len(errs) != 0 {
				t.Errorf("owner %q rejected: %v", owner, errs)
			}
		})
	}
}

// TestWriteShapedInputsRejected sweeps method × effect × policy space: every
// non-GET method, every non-read effect, and every enabled policy value fails.
func TestWriteShapedInputsRejected(t *testing.T) {
	methods := []string{"POST", "PUT", "PATCH", "DELETE", "get", ""}
	effects := []string{"write", "exec", ""}
	policies := []string{"approval", "enabled", ""}
	t.Run("methods", func(t *testing.T) {
		for _, m := range methods {
			spec := validSkeleton()
			spec.Capabilities.Operations[0].Method = m
			errs := validateStruct(t, spec)
			if !hasCode(errs, ECapability) {
				t.Errorf("method %q accepted, want E_CAPABILITY", m)
			}
		}
	})
	t.Run("effects", func(t *testing.T) {
		for _, e := range effects {
			spec := validSkeleton()
			spec.Capabilities.Operations[0].Effect = e
			errs := validateStruct(t, spec)
			if !hasCode(errs, ECapability) {
				t.Errorf("effect %q accepted, want E_CAPABILITY", e)
			}
		}
	})
	t.Run("policies", func(t *testing.T) {
		for _, p := range policies {
			spec := validSkeleton()
			spec.Policies.Confirmation.ExternalWrites = p
			errs := validateStruct(t, spec)
			if !hasCode(errs, EPolicy) {
				t.Errorf("externalWrites %q accepted, want E_POLICY", p)
			}
		}
	})
}

// TestNullAndOmittedSections locks null/omitted equivalence per section: each
// section behaves identically whether null or absent.
func TestNullAndOmittedSections(t *testing.T) {
	sections := []string{"app", "data", "ui", "workflows", "capabilities", "policies", "fixtures", "presentation", "metadata"}
	codesOf := func(errs []Error) []string {
		var out []string
		for _, e := range errs {
			out = append(out, string(e.Code)+":"+e.Path)
		}
		sort.Strings(out)
		return out
	}
	var base map[string]any
	if err := json.Unmarshal(loadFixture(t, "valid_invoice_review.json"), &base); err != nil {
		t.Fatal(err)
	}
	for _, section := range sections {
		nulled := cloneDoc(t, base)
		nulled[section] = nil
		_, errsNull := Validate(mustJSON(t, nulled))
		omitted := cloneDoc(t, base)
		delete(omitted, section)
		_, errsOmit := Validate(mustJSON(t, omitted))
		a, b := codesOf(errsNull), codesOf(errsOmit)
		if !reflect.DeepEqual(a, b) {
			t.Errorf("section %s: null=%v omitted=%v", section, a, b)
		}
	}
}

func cloneDoc(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func mustJSON(t *testing.T, doc map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// TestDegenerateDocuments proves malformed top-level inputs fail closed
// without panics and never yield a usable spec.
func TestDegenerateDocuments(t *testing.T) {
	for _, doc := range []string{``, `null`, `[]`, `123`, `"str"`, `{}`, `[{}]`, "  \n\t "} {
		spec, errs := Validate([]byte(doc))
		if len(errs) == 0 {
			t.Errorf("input %q accepted, want rejection", doc)
		}
		if spec != nil {
			t.Errorf("input %q yielded a usable spec", doc)
		}
	}
}

// TestTrailingDocumentRejected locks the single-document contract.
func TestTrailingDocumentRejected(t *testing.T) {
	raw := loadFixture(t, "valid_invoice_review.json")
	for _, suffix := range []string{" {}", "\n{}", " garbage", "\n[1,2]"} {
		if _, errs := Validate(append(append([]byte(nil), raw...), suffix...)); !hasCode(errs, ESchema) {
			t.Errorf("trailing %q accepted, want E_SCHEMA", suffix)
		}
	}
}

// TestDuplicateKeysLastWin locks last-wins duplicate behavior: the surviving
// value is always itself validated.
func TestDuplicateKeysLastWin(t *testing.T) {
	// Surviving hostile value is still rejected on value.
	raw := `{"app": {"appVersion": "1.0.0", "id": "d", "name": "D", "owner": "o", "slug": "d"},
		"capabilities": {"operations": [{"connection": "c", "effect": "read", "effect": "write", "id": "a.b", "method": "GET", "paths": ["/x"]}]},
		"data": {"entities": []}, "fixtures": [{"name": "f1"}],
		"metadata": {"schemaVersion": "miniappspec/v1"},
		"policies": {"confirmation": {"externalWrites": "disabled"}},
		"ui": {"pages": [{"components": [], "id": "h", "path": "/", "title": "H"}]}}`
	if _, errs := Validate([]byte(raw)); !hasCode(errs, ECapability) {
		t.Errorf("duplicate key smuggling accepted: %v", errs)
	}
}

// TestCaseInsensitiveFieldNames locks Go decoding behavior: variant-case
// names decode, and hostile values under them are still value-checked.
func TestCaseInsensitiveFieldNames(t *testing.T) {
	raw := `{"app": {"appVersion": "1.0.0", "id": "d", "name": "D", "owner": "o", "slug": "d"},
		"capabilities": {"operations": [{"connection": "c", "Effect": "write", "id": "a.b", "Method": "POST", "paths": ["/x"]}]},
		"data": {"entities": []}, "fixtures": [{"name": "f1"}],
		"metadata": {"schemaVersion": "miniappspec/v1"},
		"policies": {"confirmation": {"externalWrites": "disabled"}},
		"ui": {"pages": [{"components": [], "id": "h", "path": "/", "title": "H"}]}}`
	_, errs := Validate([]byte(raw))
	if !hasCode(errs, ECapability) {
		t.Errorf("case-variant hostile values accepted: %v", errs)
	}
}
