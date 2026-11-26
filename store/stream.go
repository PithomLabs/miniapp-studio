package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// AppendStream adds one cursor row for a message, event, or dispatch
// outcome, allocating the
// per-conversation sequence transactionally with the source insert. The
// advisory lock serializes allocators per conversation; the sequence is
// monotonic (aborted transactions can leave gaps, which cursor replay
// tolerates — it needs "greater than", never contiguity).
//
// Lock order (deadlock fix): the conversation row is locked BEFORE the
// advisory lock, in here, for every caller. The stream INSERT's foreign
// key takes a KeyShare on the parent conversation row implicitly —
// without this, a build-first ingester holding the advisory lock
// deadlocks against a conversation-first sender holding the row lock
// (proven by TestSSEConcurrentTrafficNoGapsNoDupes going red). FOR KEY
// SHARE is FK-compatible and never blocks fellow appenders; it only waits
// on conversation FOR UPDATE holders (message sends), whose post-lock
// phase never waits, so the wait graph stays acyclic.
func AppendStream(ctx context.Context, tx pgx.Tx, tenantID, convID, kind, refID, buildID string) (int64, error) {
	var one int
	err := tx.QueryRow(ctx,
		`SELECT 1 FROM studio.conversations WHERE id = $1 AND tenant_id = $2 FOR KEY SHARE`,
		convID, tenantID).Scan(&one)
	if err != nil {
		return 0, fmt.Errorf("store: stream conversation: %w", err)
	}
	// Two-key lock: tenant and conversation hash independently, so no
	// separator games (NUL bytes are illegal in Postgres text).
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`, tenantID, convID); err != nil {
		return 0, fmt.Errorf("store: stream append: %w", err)
	}
	var seq int64
	err = tx.QueryRow(ctx,
		`INSERT INTO studio.conversation_stream (tenant_id, conversation_id, seq, kind, ref_id, build_id)
		 SELECT $1, $2, COALESCE(MAX(seq), 0) + 1, $3, $4, $5
		 FROM studio.conversation_stream WHERE tenant_id = $1 AND conversation_id = $2
		 RETURNING seq`,
		tenantID, convID, kind, refID, buildID).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("store: stream append: %w", err)
	}
	return seq, nil
}
