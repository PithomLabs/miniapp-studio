package tools

import (
	"github.com/PithomLabs/miniapp-studio/artifact"
)

// Secret-span behavior delegates to the contract-level table in
// artifact/secrets.go: admission (tools), redaction (sandbox), and
// detection share one pattern set. Do not fork a second table here.
// RedactSecretSpans replaces credential-shaped spans with
// [REDACTED]. Exported for sandbox output capture: test logs are
// untrusted bytes, and persistence must never store credentials.
// Identical to scrubSpans (same patterns, same replacement).
func RedactSecretSpans(s string) string {
	return artifact.RedactSpans(s)
}

// ContainsSecretSpan reports whether s holds any credential-shaped
// span. Exported for artifact admission (refuse, don't redact);
// matching semantics are exactly scrubSpans'.
func ContainsSecretSpan(s string) bool {
	return artifact.MatchSecretSpan(s)
}

// scrubSpans replaces every credential-shaped span in s with [REDACTED],
// preserving boundary characters and all surrounding text.
func scrubSpans(s string) string {
	return artifact.RedactSpans(s)
}
