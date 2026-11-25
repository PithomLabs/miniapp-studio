package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/registry"
)

// Build lifecycle states (gateway plan §6). Transitions run through
// transitionBuild so every edge is checked in one place.
const (
	buildAssigned    = "assigned"
	buildRunning     = "running"
	buildWaitingUser = "waiting_user"
	buildSucceeded   = "succeeded"
	buildFailed      = "failed"
	buildCancelled   = "cancelled"
)

const (
	turnOpen        = "open"
	turnWaitingUser = "waiting_user"
	turnAnswered    = "answered"
	turnExpired     = "expired"
	turnClosed      = "closed"
)

const (
	questionOpen      = "open"
	questionAnswered  = "answered"
	questionExpired   = "expired"
	questionWithdrawn = "withdrawn"
)

// buildAllowed reports whether the build state machine permits from → to.
func buildAllowed(from, to string) bool {
	switch from {
	case buildAssigned:
		return to == buildRunning || to == buildCancelled
	case buildRunning:
		return to == buildWaitingUser || to == buildSucceeded ||
			to == buildFailed || to == buildCancelled
	case buildWaitingUser:
		return to == buildRunning || to == buildCancelled
	default:
		return false
	}
}

// turnAllowed reports whether the turn state machine permits from → to.
func turnAllowed(from, to string) bool {
	switch from {
	case turnOpen:
		return to == turnWaitingUser || to == turnClosed || to == turnExpired
	case turnWaitingUser:
		return to == turnAnswered || to == turnClosed || to == turnExpired
	case turnAnswered:
		return to == turnClosed
	default:
		return false
	}
}

// questionAllowed reports whether the question state machine permits from → to.
func questionAllowed(from, to string) bool {
	return from == questionOpen &&
		(to == questionAnswered || to == questionExpired || to == questionWithdrawn)
}

type buildRow struct {
	ID              string
	TenantID        string
	ConversationID  string
	ApplicationID   string
	SpriteName      string
	GatewayURL      string
	Status          string
	AgentKind       string
	AgentSessionRef string
}

// requireBuild loads one tenant-owned build, locking its row for transitions.
// Foreign or missing builds produce the identical not-found error.
func requireBuild(ctx context.Context, tx pgx.Tx, tenantID, buildID string, forUpdate bool) (buildRow, error) {
	var b buildRow
	query := `SELECT id, tenant_id, conversation_id, application_id, sprite_name,
		 gateway_url, status, agent_kind, agent_session_ref
		 FROM studio.builds WHERE id = $1 AND tenant_id = $2`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	err := tx.QueryRow(ctx, query, buildID, tenantID).Scan(
		&b.ID, &b.TenantID, &b.ConversationID, &b.ApplicationID, &b.SpriteName,
		&b.GatewayURL, &b.Status, &b.AgentKind, &b.AgentSessionRef)
	if err != nil {
		return buildRow{}, notFoundOrDbErr(err)
	}
	return b, nil
}

// transitionBuild moves one build along a legal edge. The row lock makes
// concurrent transitions serialize on the database; the loser re-reads
// committed state, so no transition ever applies to a stale status.
func transitionBuild(ctx context.Context, tx pgx.Tx, tenantID, buildID, to string) error {
	b, err := requireBuild(ctx, tx, tenantID, buildID, true)
	if err != nil {
		return err
	}
	if !buildAllowed(b.Status, to) {
		return fmt.Errorf("studio: build cannot move from %s to %s", b.Status, to)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE studio.builds SET status = $1, updated_at = now()
		 WHERE id = $2 AND tenant_id = $3`, to, buildID, tenantID); err != nil {
		return dbErr(err, "")
	}
	return nil
}

type turnRow struct {
	ID      string
	Tenant  string
	BuildID string
	N       int
	Status  string
}

// currentTurn returns the latest turn of one build, locking it for
// transitions. Every build owns at least turn 1 from creation.
func currentTurn(ctx context.Context, tx pgx.Tx, tenantID, buildID string) (turnRow, error) {
	var t turnRow
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, build_id, n, status FROM studio.turns
		 WHERE tenant_id = $1 AND build_id = $2 ORDER BY n DESC LIMIT 1 FOR UPDATE`,
		tenantID, buildID).Scan(&t.ID, &t.Tenant, &t.BuildID, &t.N, &t.Status)
	if err != nil {
		return turnRow{}, notFoundOrDbErr(err)
	}
	return t, nil
}

// lockTurn takes the turn row lock without reading state, for callers that
// must acquire locks in build → turn → question order before deciding.
func lockTurn(ctx context.Context, tx pgx.Tx, tenantID, turnID string) error {
	var one int
	err := tx.QueryRow(ctx,
		`SELECT 1 FROM studio.turns WHERE id = $1 AND tenant_id = $2 FOR UPDATE`,
		turnID, tenantID).Scan(&one)
	if err != nil {
		return notFoundOrDbErr(err)
	}
	return nil
}

func transitionTurn(ctx context.Context, tx pgx.Tx, tenantID, turnID, to string) error {
	var from string
	err := tx.QueryRow(ctx,
		`SELECT status FROM studio.turns WHERE id = $1 AND tenant_id = $2 FOR UPDATE`,
		turnID, tenantID).Scan(&from)
	if err != nil {
		return notFoundOrDbErr(err)
	}
	if !turnAllowed(from, to) {
		return fmt.Errorf("studio: turn cannot move from %s to %s", from, to)
	}
	_, err = tx.Exec(ctx,
		`UPDATE studio.turns SET status = $1, ended_at = CASE WHEN $1 IN ('closed','expired')
		 THEN now() ELSE ended_at END WHERE id = $2 AND tenant_id = $3`,
		to, turnID, tenantID)
	if err != nil {
		return dbErr(err, "")
	}
	return nil
}

// appendStream adds one cursor row for a message or event, allocating the
// per-conversation sequence transactionally with the source insert. The
// advisory lock serializes allocators per conversation; the sequence is
// monotonic (aborted transactions can leave gaps, which cursor replay
// tolerates — it needs "greater than", never contiguity).
func appendStream(ctx context.Context, tx pgx.Tx, tenantID, convID, kind, refID, buildID string) (int64, error) {
	// Two-key lock: tenant and conversation hash independently, so no
	// separator games (NUL bytes are illegal in Postgres text).
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`, tenantID, convID); err != nil {
		return 0, dbErr(err, "")
	}
	var seq int64
	err := tx.QueryRow(ctx,
		`INSERT INTO studio.conversation_stream (tenant_id, conversation_id, seq, kind, ref_id, build_id)
		 SELECT $1, $2, COALESCE(MAX(seq), 0) + 1, $3, $4, $5
		 FROM studio.conversation_stream WHERE tenant_id = $1 AND conversation_id = $2
		 RETURNING seq`,
		tenantID, convID, kind, refID, buildID).Scan(&seq)
	if err != nil {
		return 0, dbErr(err, "")
	}
	return seq, nil
}

// appendMessage inserts one message plus its stream row in one transaction.
func appendMessage(ctx context.Context, tx pgx.Tx, tenantID, convID, role, body string) (string, int64, error) {
	id, err := randomID()
	if err != nil {
		return "", 0, err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO studio.messages (id, tenant_id, conversation_id, role, body)
		 VALUES ($1, $2, $3, $4, $5)`, id, tenantID, convID, role, body); err != nil {
		return "", 0, dbErr(err, "")
	}
	seq, err := appendStream(ctx, tx, tenantID, convID, "message", id, "")
	if err != nil {
		return "", 0, err
	}
	return id, seq, nil
}

type buildCreateIn struct {
	ConversationID string `json:"conversation_id"`
	Title          string `json:"title"`
	ApplicationID  string `json:"application_id"`
}

// buildCreate opens one build on a conversation: the build row (assigned),
// turn 1 (open), a queued assign command for the gateway, and a system
// message so every viewer sees the lifecycle change in the stream.
func (d Deps) buildCreate(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in buildCreateIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.ConversationID == "" {
		return nil, errInvalidArguments()
	}
	if _, err := requireConversation(ctx, call.Tx, tenant, in.ConversationID, false); err != nil {
		return nil, err
	}
	buildID, turnID, err := d.createBuildTx(ctx, call.Tx, tenant, in.ConversationID, in.ApplicationID)
	if err != nil {
		return nil, err
	}
	if err := logCommand(ctx, call.Tx, tenant, buildID, "assign",
		map[string]any{"conversation_id": in.ConversationID, "title": in.Title}); err != nil {
		return nil, err
	}
	return marshalMap(map[string]any{
		"build_id": buildID, "conversation_id": in.ConversationID, "turn_id": turnID,
		"status": buildAssigned,
	})
}

// createBuildTx opens the build row, turn 1, and the system stream note.
// Shared by build.create and the message.send ensure step so both produce
// identical builds.
func (d Deps) createBuildTx(ctx context.Context, tx pgx.Tx, tenantID, convID, appID string) (string, string, error) {
	buildID, err := randomID()
	if err != nil {
		return "", "", err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO studio.builds (id, tenant_id, conversation_id, application_id, status)
		 VALUES ($1, $2, $3, $4, 'assigned')`,
		buildID, tenantID, convID, appID); err != nil {
		return "", "", dbErr(err, "")
	}
	turnID, err := randomID()
	if err != nil {
		return "", "", err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO studio.turns (id, tenant_id, build_id, n, status)
		 VALUES ($1, $2, $3, 1, 'open')`, turnID, tenantID, buildID); err != nil {
		return "", "", dbErr(err, "")
	}
	if _, _, err := appendMessage(ctx, tx, tenantID, convID, "system", "Build opened."); err != nil {
		return "", "", err
	}
	return buildID, turnID, nil
}

// logCommand records one outbound gateway command for retry and evidence.
// Phase A has no sender; the rows are the contract Phase B will drive.
func logCommand(ctx context.Context, tx pgx.Tx, tenantID, buildID, kind string, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("studio: encode command: %w", err)
	}
	id, err := randomID()
	if err != nil {
		return err
	}
	// json.RawMessage (not []byte) so pgx binds the JSON codec, not bytea.
	if _, err := tx.Exec(ctx,
		`INSERT INTO studio.commands (id, tenant_id, build_id, kind, payload, state)
		 VALUES ($1, $2, $3, $4, $5, 'queued')`, id, tenantID, buildID, kind, json.RawMessage(raw)); err != nil {
		return dbErr(err, "")
	}
	return nil
}

type buildRefIn struct {
	BuildID string `json:"build_id"`
}

// buildCancel ends one build from any non-terminal state: open turns close,
// open questions withdraw, a cancel command is logged, and a system message
// keeps every stream viewer consistent.
func (d Deps) buildCancel(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in buildRefIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.BuildID == "" {
		return nil, errInvalidArguments()
	}
	b, err := requireBuild(ctx, call.Tx, tenant, in.BuildID, true)
	if err != nil {
		return nil, err
	}
	if err := transitionBuild(ctx, call.Tx, tenant, in.BuildID, buildCancelled); err != nil {
		return nil, err
	}
	if _, err := call.Tx.Exec(ctx,
		`UPDATE studio.turns SET status = 'closed', ended_at = now()
		 WHERE tenant_id = $1 AND build_id = $2 AND status IN ('open','waiting_user','answered')`,
		tenant, in.BuildID); err != nil {
		return nil, dbErr(err, "")
	}
	if _, err := call.Tx.Exec(ctx,
		`UPDATE studio.questions SET status = 'withdrawn'
		 WHERE tenant_id = $1 AND build_id = $2 AND status = 'open'`,
		tenant, in.BuildID); err != nil {
		return nil, dbErr(err, "")
	}
	if err := logCommand(ctx, call.Tx, tenant, in.BuildID, "cancel", map[string]any{}); err != nil {
		return nil, err
	}
	if _, _, err := appendMessage(ctx, call.Tx, tenant, b.ConversationID,
		"system", "Build cancelled."); err != nil {
		return nil, err
	}
	return marshalMap(map[string]any{"build_id": in.BuildID, "status": buildCancelled})
}

// buildStatus reads one build with its current turn and open questions.
// Reads take no row locks; callers needing transition safety use the
// mutating tools, which lock.
func (d Deps) buildStatus(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in buildRefIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.BuildID == "" {
		return nil, errInvalidArguments()
	}
	b, err := requireBuild(ctx, call.Tx, tenant, in.BuildID, false)
	if err != nil {
		return nil, err
	}
	view, err := buildStatusView(ctx, call.Tx, tenant, b)
	if err != nil {
		return nil, err
	}
	return marshalMap(view)
}

// buildStatusView shapes one build with its current turn and open questions.
// Shared by build.status and conversation.get so both render identically.
func buildStatusView(ctx context.Context, tx pgx.Tx, tenant string, b buildRow) (map[string]any, error) {
	var turnID, turnStatus string
	var turnN int
	err := tx.QueryRow(ctx,
		`SELECT id, n, status FROM studio.turns
		 WHERE tenant_id = $1 AND build_id = $2 ORDER BY n DESC LIMIT 1`,
		tenant, b.ID).Scan(&turnID, &turnN, &turnStatus)
	if err != nil {
		return nil, dbErr(err, "")
	}
	rows, err := tx.Query(ctx,
		`SELECT id, turn_id, kind, body, status, options, created_at FROM studio.questions
		 WHERE tenant_id = $1 AND build_id = $2 AND status = 'open' ORDER BY created_at`,
		tenant, b.ID)
	if err != nil {
		return nil, dbErr(err, "")
	}
	questions, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, dbErr(err, "")
	}
	if questions == nil {
		questions = []map[string]any{}
	}
	for _, q := range questions {
		q["options"] = scrubbedJSON(q["options"])
	}
	return map[string]any{
		"build_id": b.ID, "conversation_id": b.ConversationID, "status": b.Status,
		"agent_kind": b.AgentKind, "agent_session_ref": b.AgentSessionRef,
		"turn":           map[string]any{"id": turnID, "n": turnN, "status": turnStatus},
		"open_questions": questions,
	}, nil
}
