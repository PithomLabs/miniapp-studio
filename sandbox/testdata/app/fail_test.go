package app

import "testing"

// TestFailFixture always fails: the N8 honest-failure proof. It runs
// only under -run selection, never in the green-path fixture run.
func TestFailFixture(t *testing.T) {
	t.Error("fixture failure by design")
}
