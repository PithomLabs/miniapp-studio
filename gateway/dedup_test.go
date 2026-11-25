package gateway

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func testClaim(id string) ClaimRecord {
	return ClaimRecord{
		MessageID: id, Name: CmdContinue, PayloadDigest: "digest-" + id,
		BuildID: "b-claim", SessionRef: "s-claim", Turn: 2,
		InvocationID: "inv-" + id,
	}
}

// TestDedupClaimLifecycle pins the claim state machine: won claims
// dispatch, completions replay, drops free the id, and indeterminate
// terminalizes it.
func TestDedupClaimLifecycle(t *testing.T) {
	d, err := OpenDedupStore(filepath.Join(t.TempDir(), "dedup.json"))
	if err != nil {
		t.Fatal(err)
	}
	res, _, _, err := d.Claim(testClaim("m1"))
	if err != nil || res != ClaimWon {
		t.Fatalf("claim = %v %v, want won", res, err)
	}
	// Second claimant sees the live claim (no replay, no steal).
	res, _, live, err := d.Claim(testClaim("m1"))
	if err != nil || res != ClaimLive || live.InvocationID != "inv-m1" {
		t.Fatalf("reclaim = %v %+v %v, want live", res, live, err)
	}
	// In-progress claims never replay.
	if _, ok := d.Check("b-claim", "m1"); ok {
		t.Fatal("in-progress claim replays, want only completions")
	}
	// Drop frees the id for a proven-no-child retry.
	if err := d.Drop("b-claim", "m1"); err != nil {
		t.Fatal(err)
	}
	res, _, _, err = d.Claim(testClaim("m1"))
	if err != nil || res != ClaimWon {
		t.Fatalf("reclaim after drop = %v %v, want won", res, err)
	}
	// Complete records and clears the claim.
	rec := DedupRecord{Status: 200, Body: json.RawMessage(`{"ok":true}`)}
	if err := d.Complete("b-claim", "m1", rec); err != nil {
		t.Fatal(err)
	}
	got, ok := d.Check("b-claim", "m1")
	if !ok || got.Status != 200 {
		t.Fatalf("replay = %+v %v, want recorded 200", got, ok)
	}
	res, _, _, _ = d.Claim(testClaim("m1"))
	if res != ClaimCompleted {
		t.Errorf("claim after complete = %v, want completed", res)
	}
	// Indeterminate terminalizes without a completion.
	if _, _, _, err := d.Claim(testClaim("m2")); err != nil {
		t.Fatal(err)
	}
	if err := d.MarkIndeterminate("b-claim", "m2", "crashed"); err != nil {
		t.Fatal(err)
	}
	res, _, live, _ = d.Claim(testClaim("m2"))
	if res != ClaimLive || !live.Indeterminate || live.Detail != "crashed" {
		t.Errorf("indeterminate claim = %v %+v", res, live)
	}
	if _, ok := d.Check("b-claim", "m2"); ok {
		t.Error("indeterminate claim replays, want 409 path")
	}
	if lives := d.LiveClaims(); len(lives) != 0 {
		t.Errorf("live claims = %d, want 0 (m2 indeterminate, m1 complete)", len(lives))
	}
	// Launch identity attaches to the won claim.
	c3 := testClaim("m3")
	if _, _, _, err := d.Claim(c3); err != nil {
		t.Fatal(err)
	}
	if err := d.RecordLaunch("b-claim", "m3", "s-new", 4242); err != nil {
		t.Fatal(err)
	}
	_, _, live, _ = d.Claim(testClaim("m3"))
	if live.PID != 4242 || live.SessionRef != "s-new" {
		t.Errorf("launch identity = %+v", live)
	}
	if lives := d.LiveClaims(); len(lives) != 1 {
		t.Errorf("live claims = %d, want 1", len(lives))
	}
}

// TestDedupLegacyReopen proves pre-claim files still load: the bare seen
// map migrates in memory without loss.
func TestDedupLegacyReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dedup.json")
	legacy := map[string]map[string]DedupRecord{
		"b": {"c": {Status: 200, Body: json.RawMessage(`{"old":true}`)}},
	}
	raw, _ := json.Marshal(legacy)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := OpenDedupStore(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := d.Check("b", "c")
	if !ok || got.Status != 200 {
		t.Fatalf("legacy replay = %+v %v", got, ok)
	}
}

// TestDedupClaimPersistFailure rolls the memory claim back when the disk
// write fails, so a later attempt can win cleanly.
func TestDedupClaimPersistFailure(t *testing.T) {
	dir := t.TempDir()
	d, err := OpenDedupStore(filepath.Join(dir, "dedup.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Replace the file with a directory: temp+rename can never commit.
	// (The store creates the file lazily, so it may not exist yet.)
	_ = os.Remove(filepath.Join(dir, "dedup.json"))
	if err := os.Mkdir(filepath.Join(dir, "dedup.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := d.Claim(testClaim("mx")); err == nil {
		t.Fatal("claim on unwritable store accepted, want error")
	}
}
