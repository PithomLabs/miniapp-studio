package fly

import (
	"fmt"
	"testing"
	"time"

	sprites "github.com/superfly/sprites-go"
)

func TestParentDir(t *testing.T) {
	cases := map[string]string{
		"/app/main.go": "/app",
		"/a/b/c":       "/a/b",
		"/f":           "",
		"/":            "",
		"relative":     "",
	}
	for in, want := range cases {
		if got := parentDir(in); got != want {
			t.Errorf("parentDir(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestToOwnedSprite(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	got := toOwnedSprite(&sprites.Sprite{
		ID: "spr_1", Status: "running", URL: "https://sprites.test/app-1",
		URLSettings: &sprites.URLSettings{Auth: "private", PrivateAccess: "strict"},
		Labels:      []string{"studio"}, CreatedAt: now,
	})
	if got.ID != "spr_1" || got.Status != "running" {
		t.Errorf("sprite = %+v", got)
	}
	// The Fly-issued URL passes through verbatim; it is never constructed.
	if got.URL != "https://sprites.test/app-1" {
		t.Errorf("URL = %q", got.URL)
	}
	if got.Auth != "private" || got.PrivateAccess != "strict" {
		t.Errorf("URL settings = %+v", got)
	}
	if len(got.Labels) != 1 || got.Labels[0] != "studio" {
		t.Errorf("labels = %v", got.Labels)
	}
	if !got.CreatedAt.Equal(now) {
		t.Errorf("created = %v", got.CreatedAt)
	}
	plain := toOwnedSprite(&sprites.Sprite{})
	if plain.Auth != "" || plain.PrivateAccess != "" {
		t.Errorf("nil URL settings leaked: %+v", plain)
	}
}

func TestAsNotFound(t *testing.T) {
	// Pinned to the sprites-go v0.2.2 message shapes observed in
	// management.go and services.go; re-verify on SDK upgrade.
	for _, msg := range []string{"sprite not found: x", "service not found: y", "SERVICE NOT FOUND: z"} {
		err := asNotFound("r", fmt.Errorf("%s", msg))
		if msg == "SERVICE NOT FOUND: z" {
			if _, ok := err.(*NotFoundError); ok {
				t.Errorf("%q classified as not-found, want passthrough (case-sensitive by design)", msg)
			}
			continue
		}
		if _, ok := err.(*NotFoundError); !ok {
			t.Errorf("%q not classified, want *NotFoundError (got %T)", msg, err)
		}
	}
	if err := asNotFound("r", nil); err != nil {
		t.Errorf("nil became %v", err)
	}
	sentinel := fmt.Errorf("boom")
	if err := asNotFound("r", sentinel); err != sentinel {
		t.Errorf("genuine error rewritten: %v", err)
	}
}

func TestToOwnedPolicy(t *testing.T) {
	got := toOwnedPolicy(&sprites.NetworkPolicy{
		Rules: []sprites.NetworkPolicyRule{{Domain: "a.example", Action: "allow"}},
	})
	if len(got.Rules) != 1 || got.Rules[0].Domain != "a.example" || got.Rules[0].Action != "allow" {
		t.Errorf("policy = %+v", got)
	}
}
