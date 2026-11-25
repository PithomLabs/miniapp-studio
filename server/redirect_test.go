package server

import (
	"os"
	"path/filepath"
	"testing"
)

// F6: safeNext is shared by login and logout; this table pins the rejection
// set, including control characters and encoded tricks.
// F8: a newline-terminated token file yields the bare credential.
func TestLoadTokenFileTrims(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("tok-abc-123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := loadTokenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "tok-abc-123" {
		t.Errorf("token = %q, want trimmed", got)
	}
	if _, err := loadTokenFile(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("missing file accepted, want error")
	}
}

func TestSafeNextTable(t *testing.T) {
	cases := []struct {
		name  string
		next  string
		want  string
		valid bool // want == next when valid
	}{
		{"empty", "", "/", false},
		{"root", "/", "/", true},
		{"plain path", "/apps", "/apps", true},
		{"nested path", "/apps/inv-rvw", "/apps/inv-rvw", true},
		{"query allowed", "/runs?cursor=12", "/runs?cursor=12", true},
		{"absolute URL", "https://evil.example/phish", "/", false},
		{"protocol-relative", "//evil.example/", "/", false},
		{"backslash lead", "/\\evil.example", "/", false},
		{"bare CRLF", "/a\r\nb", "/", false},
		{"bare LF", "/a\nb", "/", false},
		{"bare tab", "/a\tb", "/", false},
		{"bare NUL", "/a\x00b", "/", false},
		{"space", "/a b", "/", false},
		{"encoded CRLF", "/a%0db%0ac", "/", false},
		{"encoded NUL", "/a%00b", "/", false},
		{"encoded slash", "%2fapps", "/", false},
		{"double-encoded slash", "%252fapps", "/", false},
		{"double-encoded scheme", "%252f%252fevil.example", "/", false},
		{"encoded backslash", "/a%5cb", "/", false},
		{"encoded traversal", "/%2e%2e/secret", "/%2e%2e/secret", true},
		{"javascript scheme", "javascript:alert(1)", "/", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := safeNext(tc.next); got != tc.want {
				t.Errorf("safeNext(%q) = %q, want %q", tc.next, got, tc.want)
			}
		})
	}
}
