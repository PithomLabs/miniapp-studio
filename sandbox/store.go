package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/store"
)

// Run statuses (mirror the CHECK constraint).
const (
	RunQueued  = "queued"
	RunRunning = "running"
	RunPassed  = "passed"
	RunFailed  = "failed"
	RunBlocked = "blocked"
)

// StaleRunAge bounds a running row's lease: older running rows are
// reconciled to failed-with-cause on next touch (the runner that
// owned them is gone). Display layers must render over-age running
// rows as stale, never as live progress.
const StaleRunAge = 30 * time.Minute

// Store persists test-run rows. Each method is one tenant-scoped
// transaction. ClaimRun serializes per artifact with an advisory
// lock (same mechanism as the conversation-stream sequencer): at
// most one live run per artifact, no partial unique index needed.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wires a test-run store over an owner pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// ClaimRun reconciles stale running runs for the artifact, joins a
// live one if present, else queues a fresh run. It returns the run
// id and its status ("running" for a joined run, "queued" fresh).
func (s *Store) ClaimRun(ctx context.Context, tenant, artifactID, inputDigest string) (string, string, error) {
	var runID, status string
	err := store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, artifactID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`UPDATE studio.artifact_test_runs SET status = 'failed', finished_at = now(),
			 report = '{"stale":true,"error":"orphaned: runner lost before finishing"}'
			 WHERE tenant_id = $1 AND artifact_id = $2 AND status = 'running'
			 AND created_at < now() - make_interval(mins => $3)`,
			tenant, artifactID, int(StaleRunAge.Minutes())); err != nil {
			return err
		}
		err := tx.QueryRow(ctx,
			`SELECT id FROM studio.artifact_test_runs
			 WHERE tenant_id = $1 AND artifact_id = $2 AND status IN ('queued', 'running')
			 ORDER BY created_at LIMIT 1`, tenant, artifactID).Scan(&runID)
		if err == nil {
			err = tx.QueryRow(ctx, `SELECT status FROM studio.artifact_test_runs WHERE id = $1`, runID).Scan(&status)
			return err
		}
		if err != pgx.ErrNoRows {
			return err
		}
		runID = newRunID()
		status = RunQueued
		_, err = tx.Exec(ctx,
			`INSERT INTO studio.artifact_test_runs (id, tenant_id, artifact_id, status, input_digest)
			 VALUES ($1, $2, $3, 'queued', $4)`, runID, tenant, artifactID, inputDigest)
		return err
	})
	if err != nil {
		return "", "", fmt.Errorf("sandbox: claim run: %w", err)
	}
	return runID, status, nil
}

// StartRun moves a queued run to running. It reports false when the
// row is no longer queued (lost race, already handled).
func (s *Store) StartRun(ctx context.Context, tenant, runID string) (bool, error) {
	var started bool
	err := store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE studio.artifact_test_runs SET status = 'running'
			 WHERE tenant_id = $1 AND id = $2 AND status = 'queued'`, tenant, runID)
		if err != nil {
			return err
		}
		started = tag.RowsAffected() == 1
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("sandbox: start run: %w", err)
	}
	return started, nil
}

// FinishRun records a terminal outcome. Only running rows finish;
// anything else reports false (never silently rewrites history).
func (s *Store) FinishRun(ctx context.Context, tenant, runID, status string, report Report) (bool, error) {
	switch status {
	case RunPassed, RunFailed, RunBlocked:
	default:
		return false, fmt.Errorf("sandbox: invalid terminal status %q", status)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		return false, err
	}
	var done bool
	err = store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE studio.artifact_test_runs SET status = $1, report = $2, finished_at = now()
			 WHERE tenant_id = $3 AND id = $4 AND status = 'running'`,
			status, json.RawMessage(raw), tenant, runID)
		if err != nil {
			return err
		}
		done = tag.RowsAffected() == 1
		if !done {
			return nil
		}
		var conv, build, artifactID string
		if err := tx.QueryRow(ctx,
			`SELECT b.conversation_id, b.id, r.artifact_id FROM studio.artifact_test_runs AS r
			 JOIN studio.builds AS b ON b.tenant_id = r.tenant_id AND b.id = (
			   SELECT build_id FROM studio.artifacts WHERE tenant_id = r.tenant_id AND id = r.artifact_id)
			 WHERE r.tenant_id = $1 AND r.id = $2`,
			tenant, runID).Scan(&conv, &build, &artifactID); err != nil {
			return err
		}
		_, err = store.AppendStream(ctx, tx, tenant, conv, "artifact_tested", artifactID, build)
		return err
	})
	if err != nil {
		return false, fmt.Errorf("sandbox: finish run: %w", err)
	}
	return done, nil
}

func newRunID() string {
	// Tenant-scoped hex id; crypto not required (uniqueness only).
	return fmt.Sprintf("tr-%d", time.Now().UnixNano())
}
