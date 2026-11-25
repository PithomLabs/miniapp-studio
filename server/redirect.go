package server

import (
	"net/url"
	"strings"
)

// safeNext validates a post-login redirect target. Only same-origin absolute
// paths without control characters survive; anything else collapses to "/".
// The check runs on both the raw and once-decoded forms so encoded CRLF,
// NUL bytes, and double-encoded scheme/host tricks cannot pass.
func safeNext(next string) string {
	if !pathOK(next) {
		return "/"
	}
	if decoded, err := url.QueryUnescape(next); err == nil && decoded != next {
		if !pathOK(decoded) {
			return "/"
		}
	}
	return next
}

func pathOK(s string) bool {
	if !strings.HasPrefix(s, "/") || strings.HasPrefix(s, "//") {
		return false
	}
	for _, r := range s {
		if r <= 0x20 || r == 0x7f {
			return false
		}
		// Browsers parse a literal backslash as a path separator, so a
		// leading "/\" collapses into a protocol-relative open redirect.
		if r == '\\' {
			return false
		}
	}
	return true
}
