package spec

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizePath validates and normalizes one request path, page path, target,
// or allowlist pattern base. It is the single shared predicate: admission and
// (later) runtime authorization must use this same normalized representation,
// never match-then-normalize.
//
// Policy: absolute paths only; no dot-segments, backslashes, whitespace,
// control characters, query, or fragment. Percent-encoded input is decoded
// exactly once and the decoded form must satisfy the same rules; nested
// encoding (`%25` sequences that would decode to sensitive characters) is
// rejected. Redundant slashes collapse and one trailing slash is stripped.
func NormalizePath(raw string) (string, error) {
	if raw == "" || !strings.HasPrefix(raw, "/") {
		return "", fmt.Errorf("must be absolute")
	}
	if strings.Contains(raw, "\\") {
		return "", fmt.Errorf("backslashes are not allowed")
	}
	decoded := raw
	if strings.Contains(raw, "%") {
		var err error
		decoded, err = url.PathUnescape(raw)
		if err != nil {
			return "", fmt.Errorf("malformed percent-encoding: %v", err)
		}
		lower := strings.ToLower(decoded)
		if strings.Contains(lower, "%2e") || strings.Contains(lower, "%2f") ||
			strings.Contains(lower, "%5c") || strings.Contains(decoded, "%25") {
			return "", fmt.Errorf("nested encoding is not allowed")
		}
	}
	for _, seg := range strings.Split(decoded, "/") {
		if seg == "." || seg == ".." {
			return "", fmt.Errorf("dot-segments are not allowed")
		}
	}
	if strings.ContainsAny(decoded, " \t\n\r") || strings.ContainsAny(decoded, "?#") ||
		strings.IndexFunc(decoded, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return "", fmt.Errorf("whitespace, controls, query, and fragment are not allowed")
	}
	parts := strings.FieldsFunc(decoded, func(r rune) bool { return r == '/' })
	return "/" + strings.Join(parts, "/"), nil
}

// PathAllowed reports whether a normalized path matches one declared allowlist
// pattern: exact equality, or prefix match for trailing-/* wildcards.
// Callers must pass NormalizePath output, never raw input, so admission and
// execution authorize the identical representation.
func PathAllowed(patterns []string, normalized string) bool {
	for _, pat := range patterns {
		if strings.HasSuffix(pat, "/*") {
			if strings.HasPrefix(normalized, pat[:len(pat)-1]) {
				return true
			}
		} else if normalized == pat {
			return true
		}
	}
	return false
}
