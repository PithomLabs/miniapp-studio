package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/registry"
)

type questionRow struct {
	ID        string
	TenantID  string
	BuildID   string
	TurnID    string
	Kind      string
	Body      string
	Status    string
	Options   json.RawMessage
	SourceEvt string
}

func requireQuestion(ctx context.Context, tx pgx.Tx, tenantID, questionID string, forUpdate bool) (questionRow, error) {
	var q questionRow
	query := `SELECT id, tenant_id, build_id, turn_id, kind, body, status, options, source_event_id
		 FROM studio.questions WHERE id = $1 AND tenant_id = $2`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	err := tx.QueryRow(ctx, query, questionID, tenantID).Scan(
		&q.ID, &q.TenantID, &q.BuildID, &q.TurnID, &q.Kind, &q.Body, &q.Status,
		&q.Options, &q.SourceEvt)
	if err != nil {
		return questionRow{}, notFoundOrDbErr(err)
	}
	return q, nil
}

type questionAnswerIn struct {
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
}

// questionAnswer records one user answer: the question row, its turn
// (waiting_user → answered), its build (waiting_user → running), a
// continue command for the gateway, and the answer as a user message so
// the transcript stays complete — all in one transaction.
//
// Authorization requests decide explicitly: the answer must be exactly
// "approved" or "rejected", and the decision lands in a separate approvals
// row. Anything else is invalid — an answer never silently grants.
//
// Answering a non-open question fails. The HTTP layer maps the failure to
// 409 with the current open question for the turn (via build.status).
func (d Deps) questionAnswer(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in questionAnswerIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.QuestionID == "" || in.Answer == "" {
		return nil, errInvalidArguments()
	}
	// Lock order is build → turn → question everywhere a transaction
	// touches more than one (M1): ingest locks in that order, so answer
	// must too, or concurrent answer-vs-yield deadlocks. The question is
	// read unlocked first for its ids; openness is re-validated after the
	// question lock lands (a concurrent yield may have expired it).
	// The preliminary check is sound either way: questions only leave
	// open, so a non-open read can never become open again.
	q0, err := requireQuestion(ctx, call.Tx, tenant, in.QuestionID, false)
	if err != nil {
		return nil, err
	}
	if q0.Status != questionOpen {
		return nil, fmt.Errorf("studio: question is no longer open")
	}
	b, err := requireBuild(ctx, call.Tx, tenant, q0.BuildID, true)
	if err != nil {
		return nil, err
	}
	if err := lockTurn(ctx, call.Tx, tenant, q0.TurnID); err != nil {
		return nil, err
	}
	q, err := requireQuestion(ctx, call.Tx, tenant, in.QuestionID, true)
	if err != nil {
		return nil, err
	}
	if q.Status != questionOpen {
		return nil, fmt.Errorf("studio: question is no longer open")
	}
	if q.Kind == "authorization_request" && in.Answer != "approved" && in.Answer != "rejected" {
		return nil, fmt.Errorf("studio: authorization answers must decide approved or rejected")
	}
	by := call.Principal.AgentID
	if _, err := call.Tx.Exec(ctx,
		`UPDATE studio.questions SET status = 'answered', answer = $1,
		 answered_by = $2, answered_at = now()
		 WHERE id = $3 AND tenant_id = $4`,
		in.Answer, by, in.QuestionID, tenant); err != nil {
		return nil, dbErr(err, "")
	}
	if q.Kind == "authorization_request" {
		approvalID, err := randomID()
		if err != nil {
			return nil, err
		}
		if _, err := call.Tx.Exec(ctx,
			`INSERT INTO studio.approvals (id, tenant_id, build_id, question_id, decision, decided_by, note)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			approvalID, tenant, q.BuildID, q.ID, in.Answer, by, in.Answer); err != nil {
			return nil, dbErr(err, "")
		}
	}
	if err := transitionTurn(ctx, call.Tx, tenant, q.TurnID, turnAnswered); err != nil {
		return nil, err
	}
	if err := transitionBuild(ctx, call.Tx, tenant, q.BuildID, buildRunning); err != nil {
		return nil, err
	}
	if err := logCommand(ctx, call.Tx, tenant, q.BuildID, "continue",
		map[string]any{"question_id": q.ID, "kind": q.Kind, "answer": in.Answer}); err != nil {
		return nil, err
	}
	msgID, seq, err := appendMessage(ctx, call.Tx, tenant, b.ConversationID, "user", in.Answer)
	if err != nil {
		return nil, err
	}
	return marshalMap(map[string]any{
		"question_id": q.ID, "status": questionAnswered,
		"build_id": q.BuildID, "build_status": buildRunning,
		"message_id": msgID, "seq": seq,
	})
}
