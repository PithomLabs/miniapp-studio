package app

import "testing"

// TestSecretOutput prints a synthetic credential-shaped span (see the
// N10 proof): the persisted report must carry [REDACTED], never the
// span. The value is fake (sequential runes) and matches only to
// exercise the redactor.
func TestSecretOutput(t *testing.T) {
	t.Log("leaked value sk-live-abcdefghij1234567890 in output")
}
