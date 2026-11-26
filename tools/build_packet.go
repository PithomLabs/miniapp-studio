package tools

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/registry"
)

// PacketVersion versions the B0 task-packet format. Any field added,
// removed, or redefined bumps this — consumers must refuse unknown
// versions rather than guess.
const PacketVersion = "v0"

// packetBounds keeps packets bounded: full message bodies would let
// one 64k message dominate the packet.
const (
	packetMaxMessages    = 8
	packetMaxBodyRunes   = 4000
	packetMaxReportRunes = 2000
	packetMaxArtifacts   = 20
	packetMaxDecisions   = 20
)

type buildPacketIn struct {
	BuildID string `json:"build_id"`
}

// buildPacket assembles the versioned B0 execution packet for one
// build: objective, task messages, workspace identity, prior
// attempts, and evidence references. Read-only; the manual bridge
// (docs/local-agent-bridge.md) hands it to the local agent.
func (d Deps) buildPacket(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in buildPacketIn
	if err := decode(call.Input, &in); err != nil || in.BuildID == "" {
		return nil, errInvalidArguments()
	}
	b, err := requireBuild(ctx, call.Tx, tenant, in.BuildID, false)
	if err != nil {
		return nil, err
	}
	var location string
	if err := call.Tx.QueryRow(ctx,
		`SELECT execution_location FROM studio.builds WHERE id = $1 AND tenant_id = $2`,
		in.BuildID, tenant).Scan(&location); err != nil {
		return nil, dbErr(err, "")
	}
	c, err := requireConversation(ctx, call.Tx, tenant, b.ConversationID, false)
	if err != nil {
		return nil, err
	}
	messages, err := packetMessages(ctx, call.Tx, tenant, b.ConversationID)
	if err != nil {
		return nil, err
	}
	attempts, err := packetAttempts(ctx, call.Tx, tenant, in.BuildID)
	if err != nil {
		return nil, err
	}
	artifacts, err := packetArtifacts(ctx, call.Tx, tenant, in.BuildID)
	if err != nil {
		return nil, err
	}
	decisions, err := packetDecisions(ctx, call.Tx, tenant, in.BuildID)
	if err != nil {
		return nil, err
	}
	var branchHint any
	if len(attempts) > 0 {
		if git, _ := attempts[len(attempts)-1]["git"].(map[string]any); git != nil {
			if br, _ := git["branch"].(string); br != "" {
				branchHint = br
			}
		}
	}
	workspaceID := "local-desktop"
	if location != "local_desktop" {
		workspaceID = location
	}
	return marshalMap(map[string]any{
		"packet_version":  PacketVersion,
		"build_id":        b.ID,
		"conversation_id": b.ConversationID,
		"status":          b.Status,
		"workspace":       map[string]any{"id": workspaceID, "location": location},
		"objective":       c.Title,
		"task_messages":   messages,
		"branch_hint":     branchHint,
		"prior_attempts":  attempts,
		"evidence_refs": map[string]any{
			"artifacts": artifacts,
			"decisions": decisions,
		},
		"bridge": map[string]any{"mode": "manual", "runbook": "docs/local-agent-bridge.md"},
	})
}

// packetMessages returns user-role messages oldest-first, bounded
// with explicit truncation flags (never silent cuts).
func packetMessages(ctx context.Context, tx pgx.Tx, tenant, convID string) ([]any, error) {
	rows, err := tx.Query(ctx,
		`SELECT role, body FROM studio.messages
		 WHERE tenant_id = $1 AND conversation_id = $2 AND role = 'user'
		 ORDER BY created_at ASC LIMIT $3`,
		tenant, convID, packetMaxMessages+1)
	if err != nil {
		return nil, dbErr(err, "")
	}
	type msg struct{ role, body string }
	var all []msg
	for rows.Next() {
		var m msg
		if err := rows.Scan(&m.role, &m.body); err != nil {
			rows.Close()
			return nil, dbErr(err, "")
		}
		all = append(all, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, dbErr(err, "")
	}
	out := []any{}
	for i, m := range all {
		if i >= packetMaxMessages {
			break
		}
		body, truncated := truncateRunes(m.body, packetMaxBodyRunes)
		out = append(out, map[string]any{
			"seq": i + 1, "role": m.role, "body": body, "truncated": truncated,
		})
	}
	if len(all) > packetMaxMessages {
		out = append(out, map[string]any{
			"truncated": true,
			"note":      "message list truncated; see conversation for the rest",
		})
	}
	return out, nil
}

// packetAttempts replays recorded attempt_result events oldest-first.
// Reports travel as excerpts with event refs to the full text.
func packetAttempts(ctx context.Context, tx pgx.Tx, tenant, buildID string) ([]map[string]any, error) {
	rows, err := tx.Query(ctx,
		`SELECT event_id, payload FROM studio.events
		 WHERE tenant_id = $1 AND build_id = $2 AND type = 'attempt_result'
		 ORDER BY seq ASC`,
		tenant, buildID)
	if err != nil {
		return nil, dbErr(err, "")
	}
	out := []map[string]any{}
	for rows.Next() {
		var eventID string
		var payload json.RawMessage
		if err := rows.Scan(&eventID, &payload); err != nil {
			rows.Close()
			return nil, dbErr(err, "")
		}
		var p map[string]any
		if err := json.Unmarshal(payload, &p); err != nil {
			rows.Close()
			return nil, dbErr(err, "")
		}
		if report, _ := p["report"].(string); report != "" {
			excerpt, truncated := truncateRunes(report, packetMaxReportRunes)
			p["report_excerpt"] = excerpt
			p["report_truncated"] = truncated
			delete(p, "report")
		}
		delete(p, "text")
		p["event_id"] = eventID
		out = append(out, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, dbErr(err, "")
	}
	return out, nil
}

func packetArtifacts(ctx context.Context, tx pgx.Tx, tenant, buildID string) ([]any, error) {
	rows, err := tx.Query(ctx,
		`SELECT digest FROM studio.artifacts
		 WHERE tenant_id = $1 AND build_id = $2 ORDER BY created_at ASC LIMIT $3`,
		tenant, buildID, packetMaxArtifacts)
	if err != nil {
		return nil, dbErr(err, "")
	}
	out := []any{}
	for rows.Next() {
		var digest string
		if err := rows.Scan(&digest); err != nil {
			rows.Close()
			return nil, dbErr(err, "")
		}
		out = append(out, digest)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, dbErr(err, "")
	}
	return out, nil
}

func packetDecisions(ctx context.Context, tx pgx.Tx, tenant, buildID string) ([]any, error) {
	rows, err := tx.Query(ctx,
		`SELECT decision, decided_by FROM studio.approvals
		 WHERE tenant_id = $1 AND build_id = $2 ORDER BY decided_at ASC LIMIT $3`,
		tenant, buildID, packetMaxDecisions)
	if err != nil {
		return nil, dbErr(err, "")
	}
	out := []any{}
	for rows.Next() {
		var decision, by string
		if err := rows.Scan(&decision, &by); err != nil {
			rows.Close()
			return nil, dbErr(err, "")
		}
		out = append(out, map[string]any{"decision": decision, "decided_by": by})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, dbErr(err, "")
	}
	return out, nil
}

func truncateRunes(s string, max int) (string, bool) {
	if len([]rune(s)) <= max {
		return s, false
	}
	return string([]rune(s)[:max]) + "…(truncated)", true
}
