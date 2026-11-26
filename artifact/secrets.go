package artifact

import "regexp"

// Secret-span patterns live at contract level: admission (refuse),
// redaction (replace), and detection (report) share one table.
// Originally bounded closure item 1 for agent-supplied content.
//
// Each pattern matches an optional one-character left boundary plus the
// credential span; the boundary keeps mid-word collisions (task-ids,
// "flask-session") from matching while letting genuine embedded secrets
// match anywhere in a string. Suffix minima keep short fragments
// ("sk-abc", "AKIA7") intact.
//
// This collection is deliberately NOT exhaustive: it covers the credential
// shapes Studio is likely to meet in agent output. Unknown shapes pass
// through — a documented limitation, not a claim of total detection.
var secretSpanPatterns = []*regexp.Regexp{
	// Stripe-style publishable/secret keys with live/test qualifier.
	regexp.MustCompile(`(^|[^A-Za-z0-9_-])(sk-(?:live|test)-[A-Za-z0-9_-]{20,})`),
	// Underscore variants of the same families.
	regexp.MustCompile(`(^|[^A-Za-z0-9_])(sk_(?:live|test)_[A-Za-z0-9_]{20,})`),
	// AWS access key IDs: AKIA + 16 uppercase alphanumerics. Overlong
	// runs redact from the head (fail-closed: the visible tail alone is
	// not a usable credential).
	regexp.MustCompile(`(^|[^A-Za-z0-9])(AKIA[0-9A-Z]{16})`),
	// Slack tokens: xox{bearer,app,bot,page,rsa?} + body.
	regexp.MustCompile(`(^|[^A-Za-z0-9-])(xox[baprs]-[A-Za-z0-9-]{10,})`),
	// GitHub personal / OAuth / fine-grained tokens.
	regexp.MustCompile(`(^|[^A-Za-z0-9_])(ghp_[A-Za-z0-9]{20,})`),
	regexp.MustCompile(`(^|[^A-Za-z0-9_])(gho_[A-Za-z0-9]{20,})`),
	regexp.MustCompile(`(^|[^A-Za-z0-9_])(github_pat_[A-Za-z0-9_]{20,})`),
	// Groq API keys.
	regexp.MustCompile(`(^|[^A-Za-z0-9_])(gsk_[A-Za-z0-9]{20,})`),
}

// MatchSecretSpan reports whether s holds any credential-shaped span.
func MatchSecretSpan(s string) bool {
	for _, re := range secretSpanPatterns {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// RedactSpans replaces every credential-shaped span in s with
// [REDACTED], preserving boundary characters and surrounding text.
func RedactSpans(s string) string {
	for _, re := range secretSpanPatterns {
		s = re.ReplaceAllString(s, "${1}[REDACTED]")
	}
	return s
}
