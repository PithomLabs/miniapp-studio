package store

import (
	"strings"
	"testing"
)

// TestMigrationRepairSQL pins the canonical SET-capability repair in
// the EMBEDDED migration bytes (what ships, not just what's on disk).
// 000001 owns the single explicit repair + postcondition; 000002
// asserts the postcondition and must not implement a second repair.
func TestMigrationRepairSQL(t *testing.T) {
	_, _, v1 := migrationDigest(1)
	if len(v1) == 0 {
		t.Fatal("no embedded bytes for migration 1")
	}
	s1 := string(v1)
	for _, want := range []string{
		"GRANT studio_lookup TO SESSION_USER",
		"WITH ADMIN FALSE, INHERIT FALSE, SET TRUE",
		"pg_has_role(session_user, 'studio_lookup', 'SET')",
		"postcondition failed",
	} {
		if !strings.Contains(s1, want) {
			t.Errorf("000001 lacks %q", want)
		}
	}
	// The existence-gated grant that caused 42501 must be gone.
	if strings.Contains(s1, "pg_auth_members") {
		t.Error("000001 still checks membership existence instead of SET capability")
	}
	_, _, v2 := migrationDigest(2)
	if len(v2) == 0 {
		t.Fatal("no embedded bytes for migration 2")
	}
	s2 := string(v2)
	for _, want := range []string{
		"GRANT CREATE ON SCHEMA studio TO studio_lookup",
		"pg_has_role(session_user, 'studio_lookup', 'SET')",
		"has_schema_privilege('studio_lookup', 'studio', 'CREATE')",
		"precondition failed",
	} {
		if !strings.Contains(s2, want) {
			t.Errorf("000002 lacks %q", want)
		}
	}
	// No divergent second repair in 000002.
	if strings.Contains(s2, "GRANT studio_lookup TO") {
		t.Error("000002 implements its own repair; the canonical repair lives in 000001")
	}
}
