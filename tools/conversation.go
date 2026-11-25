package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/registry"
)

// shapeDispatchPayload reduces one joined command row to the stable
// subset the Pulse renders: kind, terminal state, attempts, and a
// capped first-line reason. Raw transport text never reaches the
// browser; full detail stays queryable in studio.commands.
func shapeDispatchPayload(item map[string]any) map[string]any {
	kind, _ := item["cmd_kind"].(string)
	state, _ := item["cmd_state"].(string)
	if kind == "" {
		// Join miss: the command row is gone (cascade) but the
		// stream row survived. Say unknown, never blank.
		return map[string]any{
			"command_kind": "", "state": "unknown",
			"attempts": int64(0), "reason": "command row gone",
			"exhausted": false,
		}
	}
	var attempts int64
	switch n := item["cmd_attempts"].(type) {
	case int32:
		attempts = int64(n)
	case int64:
		attempts = n
	}
	reason, _ := item["cmd_error"].(string)
	if i := strings.IndexByte(reason, '\n'); i >= 0 {
		reason = reason[:i]
	}
	if len(reason) > 300 {
		reason = reason[:300] + "…"
	}
	return map[string]any{
		"command_kind": kind, "state": state,
		"attempts": attempts, "reason": reason,
		"exhausted": state == "failed" && strings.HasPrefix(reason, "exhausted "),
	}
}

type conversationRow struct {
	ID            string
	TenantID      string
	ApplicationID string
	Title         string
	Status        string
}

func requireConversation(ctx context.Context, tx pgx.Tx, tenantID, convID string, forUpdate bool) (conversationRow, error) {
	var c conversationRow
	query := `SELECT id, tenant_id, application_id, title, status
		 FROM studio.conversations WHERE id = $1 AND tenant_id = $2`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	err := tx.QueryRow(ctx, query, convID, tenantID).Scan(
		&c.ID, &c.TenantID, &c.ApplicationID, &c.Title, &c.Status)
	if err != nil {
		return conversationRow{}, notFoundOrDbErr(err)
	}
	return c, nil
}

type conversationCreateIn struct {
	Title         string `json:"title"`
	ApplicationID string `json:"application_id"`
}

// conversationCreate opens one conversation. The first user message and the
// build arrive through their own tools so each mutation keeps its own
// idempotency key and audit record.
func (d Deps) conversationCreate(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in conversationCreateIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	if _, err := call.Tx.Exec(ctx,
		`INSERT INTO studio.conversations (id, tenant_id, application_id, title, status)
		 VALUES ($1, $2, $3, $4, 'open')`, id, tenant, in.ApplicationID, in.Title); err != nil {
		return nil, dbErr(err, "")
	}
	return marshalMap(map[string]any{"conversation_id": id})
}

type conversationGetIn struct {
	ConversationID string `json:"conversation_id"`
}

// conversationGet reads one conversation with all its builds (each shaped
// like build.status). The conversation page renders from this plus the
// stream; one call keeps build snapshots mutually consistent.
func (d Deps) conversationGet(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in conversationGetIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.ConversationID == "" {
		return nil, errInvalidArguments()
	}
	c, err := requireConversation(ctx, call.Tx, tenant, in.ConversationID, false)
	if err != nil {
		return nil, err
	}
	rows, err := call.Tx.Query(ctx,
		`SELECT id FROM studio.builds WHERE tenant_id = $1 AND conversation_id = $2 ORDER BY created_at`,
		tenant, in.ConversationID)
	if err != nil {
		return nil, dbErr(err, "")
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, dbErr(err, "")
	}
	builds := []map[string]any{}
	for _, id := range ids {
		b, err := requireBuild(ctx, call.Tx, tenant, id, false)
		if err != nil {
			return nil, err
		}
		view, err := buildStatusView(ctx, call.Tx, tenant, b)
		if err != nil {
			return nil, err
		}
		builds = append(builds, view)
	}
	return marshalMap(map[string]any{
		"conversation": map[string]any{
			"id": c.ID, "application_id": c.ApplicationID,
			"title": c.Title, "status": c.Status,
		},
		"builds": builds,
	})
}

type conversationListIn struct {
	Limit int `json:"limit"`
}

// conversationList returns the tenant's recent conversations, newest first.
func (d Deps) conversationList(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in conversationListIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := call.Tx.Query(ctx,
		`SELECT id, application_id, title, status, updated_at FROM studio.conversations
		 WHERE tenant_id = $1 ORDER BY updated_at DESC LIMIT $2`, tenant, limit)
	if err != nil {
		return nil, dbErr(err, "")
	}
	convs, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, dbErr(err, "")
	}
	if convs == nil {
		convs = []map[string]any{}
	}
	return marshalMap(map[string]any{"conversations": convs})
}

type messageSendIn struct {
	ConversationID string `json:"conversation_id"`
	Body           string `json:"body"`
}

// messageSend is the UI's submit operation, atomic: it persists one user
// message with its stream row and ensures an open build exists (creating
// one, with turn 1 and a queued assign command, when none is
// non-terminal). Atomicity matters because the framework records failures
// against their idempotency key: a two-call append-then-create could strand
// a retry on a recorded partial failure, while one transaction retries
// cleanly. The conversation row lock serializes concurrent sends so at
// most one build opens.
func (d Deps) messageSend(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in messageSendIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.ConversationID == "" || in.Body == "" {
		return nil, errInvalidArguments()
	}
	if _, err := requireConversation(ctx, call.Tx, tenant, in.ConversationID, true); err != nil {
		return nil, err
	}
	id, seq, err := appendMessage(ctx, call.Tx, tenant, in.ConversationID, "user", in.Body)
	if err != nil {
		return nil, err
	}
	var buildID string
	err = call.Tx.QueryRow(ctx,
		`SELECT id FROM studio.builds WHERE tenant_id = $1 AND conversation_id = $2
		 AND status IN ('assigned','running','waiting_user')
		 ORDER BY created_at LIMIT 1`,
		tenant, in.ConversationID).Scan(&buildID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, dbErr(err, "")
		}
		created, _, err := d.createBuildTx(ctx, call.Tx, tenant, in.ConversationID, "", "")
		if err != nil {
			return nil, err
		}
		if err := logCommand(ctx, call.Tx, tenant, created, "assign",
			map[string]any{"conversation_id": in.ConversationID, "message_id": id}); err != nil {
			return nil, err
		}
		buildID = created
	}
	return marshalMap(map[string]any{"message_id": id, "seq": seq, "build_id": buildID})
}

type conversationStreamIn struct {
	ConversationID string `json:"conversation_id"`
	AfterSeq       int64  `json:"after_seq"`
	Limit          int    `json:"limit"`
}

// conversationStream pages the unified stream: messages and agent events in
// one monotonic sequence. after_seq is the SSE Last-Event-ID cursor; the
// response carries the latest seq for the next resume. Payloads come from
// their source tables — the stream stores references only (decision D1).
func (d Deps) conversationStream(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in conversationStreamIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.ConversationID == "" || in.AfterSeq < 0 {
		return nil, errInvalidArguments()
	}
	if _, err := requireConversation(ctx, call.Tx, tenant, in.ConversationID, false); err != nil {
		return nil, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := call.Tx.Query(ctx,
		`SELECT s.seq, s.kind, s.ref_id, s.build_id,
			m.role, m.body,
			e.type, e.payload,
			c.kind AS cmd_kind, c.state AS cmd_state,
			c.attempts AS cmd_attempts, c.last_error AS cmd_error
		 FROM studio.conversation_stream AS s
		 LEFT JOIN studio.messages AS m
		   ON m.tenant_id = s.tenant_id AND m.id = s.ref_id AND s.kind = 'message'
		 LEFT JOIN studio.events AS e
		   ON e.tenant_id = s.tenant_id AND e.build_id = s.build_id
		   AND e.event_id = s.ref_id AND s.kind = 'event'
		 LEFT JOIN studio.commands AS c
		   ON c.tenant_id = s.tenant_id AND c.id = s.ref_id AND s.kind = 'dispatch'
		 WHERE s.tenant_id = $1 AND s.conversation_id = $2 AND s.seq > $3
		 ORDER BY s.seq ASC LIMIT $4`,
		tenant, in.ConversationID, in.AfterSeq, limit)
	if err != nil {
		return nil, dbErr(err, "")
	}
	items, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, dbErr(err, "")
	}
	if items == nil {
		items = []map[string]any{}
	}
	// Agent-supplied payloads scrub on read (persisted verbatim for
	// evidence, following the activity.list pattern). User message bodies
	// stay verbatim: they are the operator's own transcript.
	for _, item := range items {
		if item["kind"] == "event" {
			item["payload"] = scrubbedJSON(item["payload"])
		}
		if item["kind"] == "dispatch" {
			item["payload"] = shapeDispatchPayload(item)
		}
	}
	var latest int64 = in.AfterSeq
	if err := call.Tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(seq), 0) FROM studio.conversation_stream
		 WHERE tenant_id = $1 AND conversation_id = $2`,
		tenant, in.ConversationID).Scan(&latest); err != nil {
		return nil, dbErr(err, "")
	}
	return marshalMap(map[string]any{"items": items, "latest_seq": latest})
}
