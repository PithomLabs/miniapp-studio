package spec

import (
	"testing"
)

// TestConnectionKnownSecretPrefixesRejected proves the credential tripwire:
// values shaped like known secrets are refused at admission. The tripwire is
// not the security boundary — server-side registry resolution is.
func TestConnectionKnownSecretPrefixesRejected(t *testing.T) {
	for _, conn := range []string{
		"sk-live-1234567890",
		"AKIAIOSFODNN7EXAMPLE",
		"xoxo-token-value",
		"ghp-abcdefghij1234567890",
		"gsk-abcdefghij1234567890",
	} {
		spec := validSkeleton()
		spec.Capabilities.Operations[0].Connection = conn
		errs := validateStruct(t, spec)
		if !hasCode(errs, ECapability) {
			t.Errorf("connection %q accepted, want E_CAPABILITY, got %v", conn, errs)
		}
	}
}

// TestConnectionOpaqueIDAccepted locks the documented contract: opaque,
// well-formed identifiers validate.
func TestConnectionOpaqueIDAccepted(t *testing.T) {
	for _, conn := range []string{"conn_abc123", "box-prod-01", "custom_po"} {
		spec := validSkeleton()
		spec.Capabilities.Operations[0].Connection = conn
		errs := validateStruct(t, spec)
		for _, e := range errs {
			if e.Code == ECapability {
				t.Errorf("connection %q rejected: %v", conn, errs)
			}
		}
	}
}
