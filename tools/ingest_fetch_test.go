package tools

import (
	"context"
	"strings"
	"testing"
)

func bundleClaim(digest string, files []any) map[string]any {
	return map[string]any{
		"artifacts": []any{
			map[string]any{
				"kind": "bundle", "digest": digest, "size_bytes": float64(100),
				"files": files,
			},
		},
	}
}

func member(path, digest string, size float64) map[string]any {
	return map[string]any{"path": path, "digest": digest, "size": size}
}

func countFetchRows(t *testing.T, h harness, buildID string) int {
	t.Helper()
	var n int
	err := h.owner.QueryRow(context.Background(),
		`SELECT count(*) FROM studio.commands WHERE build_id = $1 AND kind = 'fetch_artifact'`,
		buildID).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// TestBundleClaimLogsFetchRows pins D1 ingestion: one fetch row per
// member file of a bundle claim, with the artifact linkage.
func TestBundleClaimLogsFetchRows(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "fr", toolCredA)
	ingest(t, h, buildID, "fr-done", evtCompleted, bundleClaim("sha256:"+strings.Repeat("a", 64), []any{
		member("manifest.json", "sha256:"+strings.Repeat("b", 64), 50),
		member("app/main.go", "sha256:"+strings.Repeat("c", 64), 50),
	}), "fr-k", toolCredA)
	if n := countFetchRows(t, h, buildID); n != 2 {
		t.Fatalf("fetch rows = %d, want 2", n)
	}
	var artifactID, path string
	err := h.owner.QueryRow(context.Background(),
		`SELECT payload->>'artifact_id', payload->>'path' FROM studio.commands
		 WHERE build_id = $1 AND kind = 'fetch_artifact' ORDER BY payload->>'path' LIMIT 1`,
		buildID).Scan(&artifactID, &path)
	if err != nil {
		t.Fatal(err)
	}
	if path != "app/main.go" {
		t.Errorf("fetch path = %q", path)
	}
	var n int
	err = h.owner.QueryRow(context.Background(),
		`SELECT count(*) FROM studio.artifacts WHERE id = $1 AND kind = 'bundle'`,
		artifactID).Scan(&n)
	if err != nil || n != 1 {
		t.Errorf("fetch artifact linkage: rows = %d, err = %v", n, err)
	}
}

// TestBundleClaimCoherentOrNothing pins the refusal side: malformed
// member lists and non-bundle kinds log zero fetch rows, while the
// claim row itself persists as evidence.
func TestBundleClaimCoherentOrNothing(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "fn", toolCredA)
	ingest(t, h, buildID, "fn-d1", evtCompleted, map[string]any{
		"artifacts": []any{
			map[string]any{"kind": "diff", "digest": "d1"},
			map[string]any{"kind": "bundle", "digest": "sha256:" + strings.Repeat("d", 64)},
			map[string]any{"kind": "bundle", "digest": "sha256:" + strings.Repeat("e", 64),
				"files": []any{map[string]any{"path": "app/main.go"}}},
			map[string]any{"kind": "bundle", "digest": "sha256:" + strings.Repeat("f", 64),
				"files": []any{member("../escape.go", "sha256:"+strings.Repeat("0", 64), 10)}},
		},
	}, "fn-k", toolCredA)
	if n := countFetchRows(t, h, buildID); n != 0 {
		t.Fatalf("fetch rows = %d, want 0 (diff kinds, missing/malformed/traversal members)", n)
	}
	var n int
	err := h.owner.QueryRow(context.Background(),
		`SELECT count(*) FROM studio.artifacts WHERE build_id = $1`, buildID).Scan(&n)
	if err != nil || n != 4 {
		t.Errorf("claim rows = %d, want 4 persisted as evidence", n)
	}
}

// TestBundleReclaimRedrivesFetch pins duplicate-claim behavior: the
// artifact row collapses by digest, but fetch rows log again so a
// terminally failed fetch can recover on re-drive.
func TestBundleReclaimRedrivesFetch(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "fd", toolCredA)
	claim := bundleClaim("sha256:"+strings.Repeat("a", 64), []any{
		member("app/main.go", "sha256:"+strings.Repeat("c", 64), 50),
	})
	// artifact_ready persists without transitioning, so both claims
	// flow while the build runs (a repeat completed would absorb on
	// the terminal build instead — existing sticky-terminality).
	ingest(t, h, buildID, "fd-d1", evtArtifactReady, claim, "fd-k1", toolCredA)
	ingest(t, h, buildID, "fd-d2", evtArtifactReady, claim, "fd-k2", toolCredA)
	var n int
	err := h.owner.QueryRow(context.Background(),
		`SELECT count(*) FROM studio.artifacts WHERE build_id = $1`, buildID).Scan(&n)
	if err != nil || n != 1 {
		t.Fatalf("artifact rows = %d, want 1 (digest collapse)", n)
	}
	if n := countFetchRows(t, h, buildID); n != 2 {
		t.Errorf("fetch rows = %d, want 2 (re-drive)", n)
	}
}
