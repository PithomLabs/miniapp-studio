package secrets

import (
	"net/url"
	"sort"
	"strings"
)

// RedactDSNText replaces every credential-bearing span of dsn found
// in msg with a placeholder: the raw userinfo slice as written
// (covers percent-encoded forms verbatim, including mixed
// encoded/literal bytes), its user/password split, the parsed
// decoded user/password (covers messages embedding decoded values),
// and the full DSN. Longest-first replacement keeps a short span
// from fracturing a longer one. Non-credential context (hosts,
// ports, reasons) passes through.
func RedactDSNText(msg, dsn string) string {
	for _, span := range dsnSpans(dsn) {
		msg = strings.ReplaceAll(msg, span, "<redacted-dsn>")
	}
	return msg
}

// dsnSpans extracts every credential-bearing substring of a DSN.
// Empty spans are dropped; longest first.
func dsnSpans(dsn string) []string {
	if dsn == "" {
		return nil
	}
	seen := map[string]bool{}
	var spans []string
	add := func(s string) {
		if s != "" && !seen[s] {
			seen[s] = true
			spans = append(spans, s)
		}
	}
	// Raw userinfo slice: between "://" and the last "@" before the
	// first "/" — no parsing required, so malformed DSNs and exotic
	//-but-verbatim encodings are covered exactly as written.
	if i := strings.Index(dsn, "://"); i >= 0 {
		rest := dsn[i+3:]
		host := rest
		if j := strings.Index(rest, "/"); j >= 0 {
			host = rest[:j]
		}
		if k := strings.LastIndex(host, "@"); k >= 0 {
			info := host[:k]
			add(info)
			if user, pass, ok := strings.Cut(info, ":"); ok {
				add(user)
				add(pass)
			}
		}
	}
	if u, perr := url.Parse(dsn); perr == nil && u.User != nil {
		add(u.User.Username())
		if pw, ok := u.User.Password(); ok {
			add(pw)
		}
	}
	add(dsn)
	sort.Slice(spans, func(i, j int) bool { return len(spans[i]) > len(spans[j]) })
	return spans
}
