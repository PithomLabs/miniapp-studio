package server

import "testing"

// Pure unit test: the failure set is shared by handler and test, so the
// incidents page can never silently narrow it again (P1-1 regression).
func TestIsFailureOutcome(t *testing.T) {
	failures := []string{
		"failed", "invalid-input", "invalid-entity", "forbidden",
		"key-reuse", "denied", "expired",
		"verification failed", "conformance failed",
		"approval-denied", "approval-expired",
		"key conflict rejected", "rate blocked", "attachment error",
	}
	for _, o := range failures {
		if !isFailureOutcome(o) {
			t.Errorf("isFailureOutcome(%q) = false, want true", o)
		}
	}
	nonFailures := []string{
		"", "succeeded", "pending", "cancelled", "fetched", "verified",
		"passed 1/1", "acknowledged", "assigned", "running",
	}
	for _, o := range nonFailures {
		if isFailureOutcome(o) {
			t.Errorf("isFailureOutcome(%q) = true, want false", o)
		}
	}
}
