package tools

import (
	"context"
	"testing"
)

// TestArtifactStatus pins the review-bundle read: one entry per
// artifact with verification, test-run, and decision state,
// tracking the lifecycle as it advances.
func TestArtifactStatus(t *testing.T) {
	h := setupFakes(t)
	conv, buildID := runningBuild(t, h, "st", toolCredA)
	p := validBundleParts(t, toolTenantA, buildID)
	artifactID, _ := p.seal(t, h)
	seq := 0
	status := func() map[string]any {
		seq++
		// Fresh key per call: reusing a key would replay the first
		// response through Miniapp idempotency.
		return invoke(t, h, ToolArtifactStatus,
			map[string]any{"conversation_id": conv}, "st-k-"+artifactID+"-"+string(rune('0'+seq)), toolCredA)
	}
	arts := func(res map[string]any) []any {
		list, _ := res["artifacts"].([]any)
		return list
	}
	// Claimed only: present, unverified, undecided.
	res := status()
	if list := arts(res); len(list) != 1 {
		t.Fatalf("artifacts = %d, want 1", len(list))
	} else if m := list[0].(map[string]any); m["verified"] == true || m["decision"] != "" {
		t.Fatalf("claimed entry = %v, want unverified/undecided", m)
	}
	// Verify through the real tool: status flips to verified.
	verifyResult(t, h, toolCredA, artifactID, "st-verify")
	res = status()
	if m := arts(res)[0].(map[string]any); m["verified"] != true {
		t.Fatalf("after verify: %v, want verified", m)
	}
	// A decision row surfaces with outcome and decider.
	ctx := context.Background()
	if _, err := h.owner.Exec(ctx,
		`INSERT INTO studio.approvals (id, tenant_id, build_id, artifact_id, artifact_digest, decision, decided_by)
		 VALUES ('st-d1', $1, $2, $3, 'sha256:x', 'approved', 'op-x')`,
		toolTenantA, buildID, artifactID); err != nil {
		t.Fatal(err)
	}
	res = status()
	if m := arts(res)[0].(map[string]any); m["decision"] != "approved" || m["decided_by"] != "op-x" {
		t.Fatalf("after decision: %v", m)
	}
}
