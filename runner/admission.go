package runner

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/PithomLabs/miniapp-studio/spec"
)

// ValidatedSpec is the only value the interpreter, renderer, and execution
// paths accept. It can only be produced by [Admit], which validates the
// document, binds the calling tenant, and pins the content hash. The raw
// bytes are retained so execution-time revalidation replays the identical
// pipeline; [ValidatedSpec.Spec] decodes a fresh copy on every call, so
// no caller can mutate the admitted value.
type ValidatedSpec struct {
	raw    []byte
	hash   string
	tenant string
}

// Admit validates raw as a v1 MiniappSpec for execution under tenantID, which
// must be the server-derived tenant of the caller — never a client-supplied
// value. The spec's declared owner must equal that tenant; anything else,
// including any validation failure, refuses admission.
func Admit(raw []byte, tenantID string) (ValidatedSpec, error) {
	if tenantID == "" {
		return ValidatedSpec{}, fmt.Errorf("runner: tenant is required")
	}
	parsed, errs := spec.Validate(raw)
	if len(errs) != 0 {
		return ValidatedSpec{}, fmt.Errorf("runner: spec rejected (%d errors, first: %v)", len(errs), errs[0])
	}
	if parsed.App.Owner != tenantID {
		return ValidatedSpec{}, fmt.Errorf("runner: spec owner %q does not match calling tenant", parsed.App.Owner)
	}
	hash, err := spec.ContentHash(parsed)
	if err != nil {
		return ValidatedSpec{}, fmt.Errorf("runner: content hash: %w", err)
	}
	return ValidatedSpec{raw: append([]byte(nil), raw...), hash: hash, tenant: tenantID}, nil
}

// Spec decodes a fresh copy of the admitted document. Mutating the result
// cannot affect the admitted value or any other caller.
func (v ValidatedSpec) Spec() (*spec.MiniappSpec, error) {
	var parsed spec.MiniappSpec
	dec := json.NewDecoder(bytes.NewReader(v.raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("runner: admitted bytes no longer decode: %w", err)
	}
	return &parsed, nil
}

// Revalidate replays admission-time validation over the retained bytes and
// re-checks the pinned hash and tenant binding. Execution paths call it
// before running: admission-time validity is never trusted blindly later.
func (v ValidatedSpec) Revalidate() error {
	parsed, errs := spec.Validate(v.raw)
	if len(errs) != 0 {
		return fmt.Errorf("runner: admitted spec no longer validates (%d errors, first: %v)", len(errs), errs[0])
	}
	if parsed.App.Owner != v.tenant {
		return fmt.Errorf("runner: admitted owner no longer matches tenant")
	}
	hash, err := spec.ContentHash(parsed)
	if err != nil {
		return fmt.Errorf("runner: content hash: %w", err)
	}
	if hash != v.hash {
		return fmt.Errorf("runner: content hash changed since admission")
	}
	return nil
}

// Hash returns the content hash pinned at admission.
func (v ValidatedSpec) Hash() string { return v.hash }

// Tenant returns the tenant the spec was admitted for.
func (v ValidatedSpec) Tenant() string { return v.tenant }
