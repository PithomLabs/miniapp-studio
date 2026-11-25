package tools

import (
	"strings"
	"testing"
)

// slackShape is assembled from parts so the source file never contains a
// contiguous Slack-token literal (GitHub push protection scans blobs).
// At runtime it is the same shape-exact fake canary as before.
var slackShape = "xoxb-" + "123456789012" + "-ABCDEFGHIJKLMN"

// Detection: every accepted shape redacts mid-string with surrounding text
// preserved. Canary cores are fake (TESTCANARY*) but shape-exact, which is
// what the patterns match on.
func TestScrubSpansDetects(t *testing.T) {
	cases := map[string]string{
		"stripe live":   "paste sk-live-TESTCANARY000000000001 to proceed",
		"stripe test":   "key=sk-test-TESTCANARY000000000002!",
		"stripe live _": "sk_live_TESTCANARY000000000003 here",
		"stripe test _": "use sk_test_TESTCANARY000000000004.",
		"aws":           "id AKIAIOSFODNN7EXAMPLE ok",
		"slack bot":     "token " + slackShape + " here",
		"github pat":    "ghp_ABCDEFGHIJKLMNOPQRSTUVWX end",
		"github oauth":  "gho_ABCDEFGHIJKLMNOPQRSTUVWX end",
		"github fine":   "github_pat_ABCDEFGHIJKLMNOPQRSTUVWX end",
		"groq":          "gsk_ABCDEFGHIJKLMNOPQRSTUVWX end",
		"at start":      "sk-live-TESTCANARY000000000005",
		"at end":        "ends with sk-live-TESTCANARY000000000006",
		"two secrets":   "a sk-live-TESTCANARY000000000007 b sk-test-TESTCANARY000000000008 c",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			got := scrubSpans(in)
			if !strings.Contains(got, "[REDACTED]") {
				t.Errorf("no redaction: %q", got)
			}
			for _, leak := range []string{"TESTCANARY", "AKIAIOSFODNN7EXAMPLE", "xoxb-12345", "ghp_ABC", "gho_ABC", "github_pat_ABC", "gsk_ABC"} {
				if strings.Contains(got, leak) {
					t.Errorf("leaked %q in %q", leak, got)
				}
			}
		})
	}
	// Surrounding prose survives verbatim around the redaction.
	got := scrubSpans("paste sk-live-TESTCANARY000000000009 to proceed")
	if got != "paste [REDACTED] to proceed" {
		t.Errorf("got %q", got)
	}
}

// False positives: ordinary prose, identifiers, and short fragments pass
// through byte-identical. Each case guards a boundary or minimum rule.
func TestScrubSpansPreserves(t *testing.T) {
	cases := map[string]string{
		"plain":            "nothing secret here",
		"empty":            "",
		"short sk":         "sk-abc",
		"short live":       "sk-live-short",
		"task id":          "task-123456789012345678901234567890",
		"mid-word sk":      "flask-session handling",
		"risk word":        "risk-assessment for mask-12 and desk-99",
		"short akia":       "AKIA7",
		"lowercase akia":   "akiaiosfodnn7example",
		"short slack":      "xoxb-123",
		"wrong slack kind": "xoxz-12345678901234567890",
		"short ghp":        "ghp_abc",
		"uuid":             "ses-550e8400-e29b-41d4-a716-446655440000",
		"code":             `if err != nil { return err } // ask-123`,
		"url":              "https://example.com/tasks/42",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if got := scrubSpans(in); got != in {
				t.Errorf("mangled %q -> %q", in, got)
			}
		})
	}
}

// Denylisted map keys still redact whole values regardless of shape.
func TestScrubValueKeys(t *testing.T) {
	got, _ := scrubValue(map[string]any{
		"api_key": "whatever", "text": "hi",
		"nested": map[string]any{"password": "hunter2"},
	}).(map[string]any)
	if got["api_key"] != "[REDACTED]" || got["text"] != "hi" {
		t.Errorf("map scrub = %v", got)
	}
	if nested, _ := got["nested"].(map[string]any); nested["password"] != "[REDACTED]" {
		t.Errorf("nested scrub = %v", got)
	}
}
