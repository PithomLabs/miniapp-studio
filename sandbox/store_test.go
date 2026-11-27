package sandbox

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/testdb"
)

func testStore(t *testing.T) (*Store, string) {
	t.Helper()
	dsn := testdb.RunDSN(t)
	if err := store.Migrate(context.Background(), dsn, testdb.Policy(t)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return NewStore(pool), dsn
}

func seedBundleRow(t *testing.T, pool *pgxpool.Pool, tenant, build, artifact string) {
	t.Helper()
	ctx := context.Background()
	err := store.InTenantTx(ctx, pool, tenant, func(tx pgx.Tx) error {
		conv := "c-" + build
		if _, err := tx.Exec(ctx,
			`INSERT INTO studio.conversations (id, tenant_id) VALUES ($1, $2)`, conv, tenant); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO studio.builds (id, tenant_id, conversation_id) VALUES ($1, $2, $3)`,
			build, tenant, conv); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO studio.artifacts (id, tenant_id, build_id, kind, digest) VALUES ($1, $2, $3, 'bundle', 'sha256:x')`,
			artifact, tenant, build)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestClaimRunLifecycle pins claim → start → finish with
// serialization: a second claim joins the live run.
func TestClaimRunLifecycle(t *testing.T) {
	st, _ := testStore(t)
	ctx := context.Background()
	seedBundleRow(t, st.pool, "t-sb1", "b-sb1", "a-sb1")
	id1, status, err := st.ClaimRun(ctx, "t-sb1", "a-sb1", "sha256:abc")
	if err != nil || status != RunQueued {
		t.Fatalf("claim = %s %s, err = %v", id1, status, err)
	}
	ok, err := st.StartRun(ctx, "t-sb1", id1)
	if err != nil || !ok {
		t.Fatalf("start = %v, err = %v", ok, err)
	}
	id2, status, err := st.ClaimRun(ctx, "t-sb1", "a-sb1", "sha256:abc")
	if err != nil || id2 != id1 || status != RunRunning {
		t.Fatalf("second claim = %s %s, err = %v", id2, status, err)
	}
	done, err := st.FinishRun(ctx, "t-sb1", id1, RunPassed, Report{Pass: true, Total: 1, Passed: 1})
	if err != nil || !done {
		t.Fatalf("finish = %v, err = %v", done, err)
	}
	done, err = st.FinishRun(ctx, "t-sb1", id1, RunFailed, Report{})
	if err != nil || done {
		t.Errorf("second finish = %v, err = %v, want refusal", done, err)
	}
	if _, err := st.FinishRun(ctx, "t-sb1", id1, "bogus", Report{}); err == nil {
		t.Error("bogus terminal status accepted")
	}
}

// TestClaimRunReconcilesStale pins orphan recovery: a running row
// older than the lease fails with cause on next touch, and the
// claim proceeds to a fresh run.
func TestClaimRunReconcilesStale(t *testing.T) {
	st, _ := testStore(t)
	ctx := context.Background()
	seedBundleRow(t, st.pool, "t-sb2", "b-sb2", "a-sb2")
	id1, _, err := st.ClaimRun(ctx, "t-sb2", "a-sb2", "sha256:abc")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.StartRun(ctx, "t-sb2", id1); err != nil {
		t.Fatal(err)
	}
	err = store.InTenantTx(ctx, st.pool, "t-sb2", func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`UPDATE studio.artifact_test_runs SET created_at = now() - make_interval(hours => 2) WHERE id = $1`, id1)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	id2, status, err := st.ClaimRun(ctx, "t-sb2", "a-sb2", "sha256:abc")
	if err != nil {
		t.Fatal(err)
	}
	if id2 == id1 || status != RunQueued {
		t.Fatalf("claim after stale = %s %s, want a fresh queued run", id2, status)
	}
	var oldStatus string
	err = store.InTenantTx(ctx, st.pool, "t-sb2", func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT status FROM studio.artifact_test_runs WHERE id = $1`, id1).Scan(&oldStatus)
	})
	if err != nil {
		t.Fatal(err)
	}
	if oldStatus != RunFailed {
		t.Errorf("orphan status = %s, want failed-with-cause", oldStatus)
	}
}
