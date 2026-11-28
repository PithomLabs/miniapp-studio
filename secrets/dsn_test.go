package secrets

import (
	"strings"
	"testing"
)

func TestRedactDSNText(t *testing.T) {
	dsn := "postgres://us%40er:p%40ss%3Aword%2Fx@db.example.com:5432/neondb?sslmode=require"
	for _, msg := range []string{
		"dial " + dsn + ": refused",
		"auth failed for us@er",
		`parse "postgres://u:pw@h/db": bad`,
	} {
		got := RedactDSNText(msg, dsn)
		for _, leak := range []string{"us%40er", "p%40ss%3Aword%2Fx", "us@er", "p@ss", dsn} {
			if strings.Contains(got, leak) {
				t.Errorf("msg %q leaked %q: %q", msg, leak, got)
			}
		}
	}
	if got := RedactDSNText("db.example.com:5432 refused", dsn); !strings.Contains(got, "db.example.com") {
		t.Errorf("host context lost: %q", got)
	}
	if got := RedactDSNText("clean message", ""); got != "clean message" {
		t.Errorf("empty DSN rewrote: %q", got)
	}
}
