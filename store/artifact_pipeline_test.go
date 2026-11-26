package store

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// seedArtifactLineage inserts conversation → build → bundle artifact
// rows under a per-test tag. Tests share one run database, so every
// id derives from the tag.
func seedArtifactLineage(t *testing.T, env testEnv, tag string) (tenant, build, artifact string) {
	t.Helper()
	tenant, conv := "t-"+tag, "c-"+tag
	build, artifact = "b-"+tag, "a-"+tag
	ctx := context.Background()
	stmts := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO studio.conversations (id, tenant_id) VALUES ($1, $2)`, []any{conv, tenant}},
		{`INSERT INTO studio.builds (id, tenant_id, conversation_id) VALUES ($1, $2, $3)`, []any{build, tenant, conv}},
		{`INSERT INTO studio.artifacts (id, tenant_id, build_id, kind, digest) VALUES ($1, $2, $3, 'bundle', 'sha256:` + strings.Repeat("e", 64) + `')`, []any{artifact, tenant, build}},
	}
	for _, s := range stmts {
		if _, err := env.owner.Exec(ctx, s.sql, s.args...); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return tenant, build, artifact
}

// TestArtifactBlobs pins the 000021 content store: idempotent
// digest-keyed writes, size CHECK, and the storage_ref FK (NULL
// skips; bogus references fail).
func TestArtifactBlobs(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	tenant, _, artifact := seedArtifactLineage(t, env, "d1b")
	digest := "sha256:" + strings.Repeat("f", 64)
	for i := 0; i < 2; i++ {
		if _, err := env.owner.Exec(ctx,
			`INSERT INTO studio.artifact_blobs (tenant_id, digest, bytes, size_bytes)
			 VALUES ($1, $2, $3, $4) ON CONFLICT (tenant_id, digest) DO NOTHING`,
			tenant, digest, []byte("bundle-bytes"), 12); err != nil {
			t.Fatalf("blob write %d: %v", i, err)
		}
	}
	var n int
	if err := env.owner.QueryRow(ctx,
		`SELECT count(*) FROM studio.artifact_blobs WHERE tenant_id = $1`, tenant).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("blobs = %d, want 1 (idempotent write)", n)
	}
	if _, err := env.owner.Exec(ctx,
		`UPDATE studio.artifacts SET storage_ref = $1 WHERE id = $2 AND tenant_id = $3`,
		digest, artifact, tenant); err != nil {
		t.Fatalf("storage_ref link: %v", err)
	}
	if _, err := env.owner.Exec(ctx,
		`UPDATE studio.artifacts SET storage_ref = $1 WHERE id = $2 AND tenant_id = $3`,
		"sha256:"+strings.Repeat("0", 64), artifact, tenant); err == nil {
		t.Error("bogus storage_ref accepted, want FK violation")
	}
	if _, err := env.owner.Exec(ctx,
		`INSERT INTO studio.artifact_blobs (tenant_id, digest, bytes, size_bytes)
		 VALUES ($1, $2, 'x', 52428801)`,
		tenant, "sha256:"+strings.Repeat("1", 64)); err == nil {
		t.Error("oversize blob accepted, want CHECK violation")
	}
}

// TestApprovalPartialUnique pins the two-shape approvals rule: any
// number of question-grant rows (artifact_id = ”) coexist, but one
// artifact takes exactly one decision row.
func TestApprovalPartialUnique(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	tenant, build, artifact := seedArtifactLineage(t, env, "d1p")
	for _, id := range []string{"g-1", "g-2"} {
		if _, err := env.owner.Exec(ctx,
			`INSERT INTO studio.approvals (id, tenant_id, build_id, question_id, decision, decided_by)
			 VALUES ($1, $2, $3, 'q-1', 'approved', 'op')`, id, tenant, build); err != nil {
			t.Fatalf("grant %s: %v", id, err)
		}
	}
	digest := "sha256:" + strings.Repeat("e", 64)
	if _, err := env.owner.Exec(ctx,
		`INSERT INTO studio.approvals (id, tenant_id, build_id, artifact_id, artifact_digest, decision, decided_by)
		 VALUES ('d-1', $1, $2, $3, $4, 'approved', 'op')`, tenant, build, artifact, digest); err != nil {
		t.Fatalf("decision: %v", err)
	}
	if _, err := env.owner.Exec(ctx,
		`INSERT INTO studio.approvals (id, tenant_id, build_id, artifact_id, artifact_digest, decision, decided_by)
		 VALUES ('d-2', $1, $2, $3, $4, 'rejected', 'op')`, tenant, build, artifact, digest); err == nil {
		t.Error("second decision for one artifact accepted, want unique violation")
	}
}

// TestArtifactEvidenceTablesReachAppRole proves the D1 tables are
// usable through the test-role mirror (RLS-filtered, never 42501),
// the same F2 shape as the framework tables.
func TestArtifactEvidenceTablesReachAppRole(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	if err := InTenantTx(ctx, env.app, "t-d1r", func(tx pgx.Tx) error {
		for _, table := range []string{"artifact_blobs", "artifact_verifications", "artifact_test_runs"} {
			var n int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM studio.`+table).Scan(&n); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("app-role read: %v", err)
	}
}
