package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/registry"
)

// Gateway event types (protocol §8, refined vocabulary).
const (
	evtStarted              = "agent.started"
	evtProgress             = "agent.progress"
	evtMessage              = "agent.message"
	evtQuestion             = "agent.question"
	evtProposal             = "agent.proposal"
	evtAuthorizationRequest = "agent.authorization_request"
	evtArtifactReady        = "agent.artifact_ready"
	evtCompleted            = "agent.completed"
	evtFailed               = "agent.failed"
	evtResumed              = "agent.resumed"
	evtCancelledAck         = "agent.cancelled_ack"
)

// TerminalBuild reports whether a build status is terminal. Exported so
// the sender's claim guard and ingest share one definition of death;
// the state table in tools/build.go remains the owner of legal edges.
func TerminalBuild(status string) bool {
	return status == buildSucceeded || status == buildFailed || status == buildCancelled
}

type ingestIn struct {
	BuildID string         `json:"build_id"`
	EventID string         `json:"event_id"`
	Type    string         `json:"type"`
	Seq     int            `json:"seq"`
	Payload map[string]any `json:"payload"`
}

// buildIngestEvent applies one gateway callback. Two rules govern it:
//
//  1. History is append-only: every well-formed event for a known build is
//     persisted with a stream row, in arrival order, even when no state
//     transition applies. Out-of-order and late events are evidence, not
//     errors.
//  2. Transitions are state-guarded: a transition applies only from its
//     legal states. Illegal combinations record the event and report
//     transitioned:false instead of failing — at-least-once delivery plus
//     user/gateway races (cancel vs in-flight completion) must never wedge
//     the gateway outbox on an error loop.
//
// Hard errors are reserved for unknown builds, cross-tenant references,
// and malformed envelopes: cases where recording history would itself be
// wrong. Signature verification happens in the HTTP handler before this
// tool runs.
func (d Deps) buildIngestEvent(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in ingestIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.BuildID == "" || in.EventID == "" || in.Type == "" || in.Payload == nil {
		return nil, errInvalidArguments()
	}
	// Write-path sanitization (closure item 1): the authenticated payload
	// is scrubbed before anything persists or derives from it, so
	// Studio-owned agent-message surfaces (events, questions, artifacts,
	// stream) never hold recognized secret spans. Signature verification
	// already ran in the HTTP handler on the original bytes; sanitizing
	// here cannot break it, and the framework's own input copy
	// (app.invocations) stays the explicitly disclosed residual.
	in.Payload = scrubValue(in.Payload).(map[string]any)
	b, err := requireBuild(ctx, call.Tx, tenant, in.BuildID, true)
	if err != nil {
		return nil, err
	}
	rawPayload, err := json.Marshal(in.Payload)
	if err != nil {
		return nil, fmt.Errorf("studio: encode event: %w", err)
	}
	var inserted bool
	err = call.Tx.QueryRow(ctx,
		`INSERT INTO studio.events (tenant_id, build_id, event_id, seq, type, payload)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (tenant_id, build_id, event_id) DO NOTHING
		 RETURNING true`,
		tenant, in.BuildID, in.EventID, in.Seq, in.Type, json.RawMessage(rawPayload)).Scan(&inserted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No row returned: the event is already recorded —
			// acknowledge the duplicate without reapplying any effect.
			return marshalMap(map[string]any{
				"recorded": true, "duplicate": true,
				"transitioned": false, "build_status": b.Status,
			})
		}
		return nil, dbErr(err, "")
	}
	seq, err := AppendStream(ctx, call.Tx, tenant, b.ConversationID, "event", in.EventID, in.BuildID)
	if err != nil {
		return nil, err
	}
	transitioned := false
	if !TerminalBuild(b.Status) {
		transitioned, err = applyEvent(ctx, call, tenant, b, in)
		if err != nil {
			return nil, err
		}
	}
	status := b.Status
	if transitioned {
		if err := call.Tx.QueryRow(ctx,
			`SELECT status FROM studio.builds WHERE id = $1 AND tenant_id = $2`,
			in.BuildID, tenant).Scan(&status); err != nil {
			return nil, dbErr(err, "")
		}
	}
	return marshalMap(map[string]any{
		"recorded": true, "duplicate": false,
		"transitioned": transitioned, "build_status": status, "seq": seq,
	})
}

// applyEvent runs the type-specific transitions for a freshly recorded
// event. It returns whether the build moved. Unknown types record history
// only — forward compatibility without crashes or silent drops.
func applyEvent(ctx context.Context, call registry.ToolCall, tenant string, b buildRow, in ingestIn) (bool, error) {
	switch in.Type {
	case evtStarted:
		if b.Status != buildAssigned {
			return false, nil
		}
		if ref, _ := in.Payload["agent_session_id"].(string); ref != "" {
			if _, err := call.Tx.Exec(ctx,
				`UPDATE studio.builds SET agent_session_ref = $1 WHERE id = $2 AND tenant_id = $3`,
				ref, b.ID, tenant); err != nil {
				return false, dbErr(err, "")
			}
		}
		return true, transitionBuild(ctx, call.Tx, tenant, b.ID, buildRunning)
	case evtQuestion, evtProposal, evtAuthorizationRequest:
		return applyYield(ctx, call, tenant, b, in)
	case evtCompleted, evtFailed:
		persistArtifacts(ctx, call, tenant, b.ID, in.Payload)
		status := b.Status
		if status == buildAssigned {
			// The run finished before its started arrived (lost or
			// reordered): step through running so the terminal state
			// applies instead of stranding the build assigned forever.
			// Trust assumption (ratified Boundary A §C): an
			// HMAC-authenticated terminal proves the run it names.
			// Out-of-band drives can move builds Studio didn't
			// dispatch — convergent via sender reconcile, never
			// silent — and absorbing instead would strand.
			if err := transitionBuild(ctx, call.Tx, tenant, b.ID, buildRunning); err != nil {
				return false, err
			}
			status = buildRunning
		}
		if status == buildWaitingUser {
			// Terminal arrived while an answer was outstanding: the
			// run is over, so the open question can never be
			// answered. Expire it with a durable cause (a later
			// answer conflicts honestly) instead of absorbing and
			// stranding the build waiting forever. The build steps
			// through running — same txn, invisible — because the
			// edge table has no waiting-to-terminal edge.
			to := buildSucceeded
			if in.Type == evtFailed {
				to = buildFailed
			}
			cause := fmt.Sprintf("terminal %s arrived while awaiting answer (event %s)", in.Type, in.EventID)
			if _, err := call.Tx.Exec(ctx,
				`UPDATE studio.questions SET status = 'expired', expired_reason = $1
				 WHERE tenant_id = $2 AND build_id = $3 AND status = 'open'`,
				cause, tenant, b.ID); err != nil {
				return false, dbErr(err, "")
			}
			cur, err := currentTurn(ctx, call.Tx, tenant, b.ID)
			if err != nil {
				return false, err
			}
			if cur.Status == turnWaitingUser {
				if err := transitionTurn(ctx, call.Tx, tenant, cur.ID, turnExpired); err != nil {
					return false, err
				}
			} else if cur.Status != turnClosed && cur.Status != turnExpired {
				if err := transitionTurn(ctx, call.Tx, tenant, cur.ID, turnClosed); err != nil {
					return false, err
				}
			}
			if err := transitionBuild(ctx, call.Tx, tenant, b.ID, buildRunning); err != nil {
				return false, err
			}
			return true, transitionBuild(ctx, call.Tx, tenant, b.ID, to)
		}
		if status != buildRunning {
			return false, nil
		}
		to := buildSucceeded
		if in.Type == evtFailed {
			to = buildFailed
		}
		cur, err := currentTurn(ctx, call.Tx, tenant, b.ID)
		if err != nil {
			return false, err
		}
		if cur.Status != turnClosed && cur.Status != turnExpired {
			if err := transitionTurn(ctx, call.Tx, tenant, cur.ID, turnClosed); err != nil {
				return false, err
			}
		}
		return true, transitionBuild(ctx, call.Tx, tenant, b.ID, to)
	case evtArtifactReady:
		persistArtifacts(ctx, call, tenant, b.ID, in.Payload)
		return false, nil
	case evtResumed:
		if ref, _ := in.Payload["agent_session_id"].(string); ref != "" {
			if _, err := call.Tx.Exec(ctx,
				`UPDATE studio.builds SET agent_session_ref = $1 WHERE id = $2 AND tenant_id = $3`,
				ref, b.ID, tenant); err != nil {
				return false, dbErr(err, "")
			}
			return true, nil
		}
		return false, nil
	default:
		// agent.message, agent.progress, agent.cancelled_ack, and any
		// future type: history only.
		return false, nil
	}
}

// applyYield parks the build on an agent question. Yields apply to running
// builds (park) and waiting builds (supersede the open question). An
// assigned build joins too: the authenticated yield proves a live run
// whose started event was lost or reordered, so the build steps through
// running to waiting instead of stranding the agent on a question Studio
// never parks. Anything else records history without touching turns.
//
// Trust assumption (ratified Boundary A §C): an HMAC-authenticated yield
// implies a live run. Out-of-band drives can park questions for runs
// Studio didn't dispatch; those converge via the sender's
// status-reconcile supersede rather than failing silently.
func applyYield(ctx context.Context, call registry.ToolCall, tenant string, b buildRow, in ingestIn) (bool, error) {
	status := b.Status
	if status == buildAssigned {
		if err := transitionBuild(ctx, call.Tx, tenant, b.ID, buildRunning); err != nil {
			return false, err
		}
		status = buildRunning
	}
	if status != buildRunning && status != buildWaitingUser {
		return false, nil
	}
	var kind string
	switch in.Type {
	case evtQuestion:
		kind = "clarification"
	case evtProposal:
		kind = "proposal"
	default:
		kind = "authorization_request"
	}
	body, _ := in.Payload["body"].(string)
	questionID, _ := in.Payload["question_id"].(string)
	if questionID == "" {
		// The gateway omitted its correlation id; mint one so Studio-side
		// continue commands still correlate. Retries dedup on event_id, so
		// the mint happens exactly once.
		var err error
		questionID, err = randomID()
		if err != nil {
			return false, err
		}
	}
	var options json.RawMessage = json.RawMessage(`[]`)
	if opts, ok := in.Payload["options"]; ok {
		if raw, err := json.Marshal(opts); err == nil {
			options = raw
		}
	}
	cur, err := currentTurn(ctx, call.Tx, tenant, b.ID)
	if err != nil {
		return false, err
	}
	turnID := cur.ID
	switch cur.Status {
	case turnOpen:
		if err := transitionTurn(ctx, call.Tx, tenant, cur.ID, turnWaitingUser); err != nil {
			return false, err
		}
	case turnWaitingUser:
		// A second yield supersedes the first: expire the open question
		// and its turn, and park a fresh turn. The expired rows stay as
		// the audit trail of what the user never saw answered.
		if _, err := call.Tx.Exec(ctx,
			`UPDATE studio.questions SET status = 'expired'
			 WHERE tenant_id = $1 AND turn_id = $2 AND status = 'open'`,
			tenant, cur.ID); err != nil {
			return false, dbErr(err, "")
		}
		if err := transitionTurn(ctx, call.Tx, tenant, cur.ID, turnExpired); err != nil {
			return false, err
		}
		turnID, err = openTurn(ctx, call, tenant, b.ID, cur.N+1, turnWaitingUser)
		if err != nil {
			return false, err
		}
	case turnAnswered:
		if err := transitionTurn(ctx, call.Tx, tenant, cur.ID, turnClosed); err != nil {
			return false, err
		}
		turnID, err = openTurn(ctx, call, tenant, b.ID, cur.N+1, turnWaitingUser)
		if err != nil {
			return false, err
		}
	default:
		// Closed or expired latest turn with a live build is unreachable
		// through the tools (every close opens a successor or ends the
		// build). Park a fresh turn rather than failing a legal event.
		turnID, err = openTurn(ctx, call, tenant, b.ID, cur.N+1, turnWaitingUser)
		if err != nil {
			return false, err
		}
	}
	if _, err := call.Tx.Exec(ctx,
		`INSERT INTO studio.questions
		 (id, tenant_id, build_id, turn_id, kind, body, status, source_event_id, options)
		 VALUES ($1, $2, $3, $4, $5, $6, 'open', $7, $8)`,
		questionID, tenant, b.ID, turnID, kind, body, in.EventID, options); err != nil {
		return false, dbErr(err, "question already recorded")
	}
	if status == buildRunning {
		return true, transitionBuild(ctx, call.Tx, tenant, b.ID, buildWaitingUser)
	}
	// Supersede path: the build already waits; turns and questions moved.
	return true, nil
}

func openTurn(ctx context.Context, call registry.ToolCall, tenantID, buildID string, n int, status string) (string, error) {
	id, err := randomID()
	if err != nil {
		return "", err
	}
	if _, err := call.Tx.Exec(ctx,
		`INSERT INTO studio.turns (id, tenant_id, build_id, n, status)
		 VALUES ($1, $2, $3, $4, $5)`, id, tenantID, buildID, n, status); err != nil {
		return "", dbErr(err, "")
	}
	return id, nil
}

// persistArtifacts records artifact metadata from completed/artifact_ready
// payloads. Entries are evidence: malformed entries are skipped (the full
// manifest persists in the event row for forensics), digest duplicates
// collapse. Phase A never fetches bytes, so verified_at stays NULL —
// unverified until proven, and the UI must say so.
func persistArtifacts(ctx context.Context, call registry.ToolCall, tenantID, buildID string, payload map[string]any) int {
	raw, ok := payload["artifacts"].([]any)
	if !ok {
		return 0
	}
	skipped := 0
	for _, entry := range raw {
		m, ok := entry.(map[string]any)
		if !ok {
			skipped++
			continue
		}
		kind, _ := m["kind"].(string)
		digest, _ := m["digest"].(string)
		if !validArtifactKind(kind) || digest == "" {
			skipped++
			continue
		}
		var size int64
		if f, ok := m["size_bytes"].(float64); ok && f >= 0 {
			size = int64(f)
		}
		manifest, err := json.Marshal(m)
		if err != nil {
			skipped++
			continue
		}
		id, err := randomID()
		if err != nil {
			return skipped
		}
		if _, err := call.Tx.Exec(ctx,
			`INSERT INTO studio.artifacts (id, tenant_id, build_id, kind, digest, size_bytes, manifest)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (tenant_id, build_id, digest) DO NOTHING`,
			id, tenantID, buildID, kind, digest, size, json.RawMessage(manifest)); err != nil {
			skipped++
			continue
		}
	}
	return skipped
}

func validArtifactKind(kind string) bool {
	switch kind {
	case "diff", "spec", "test_report", "bundle":
		return true
	default:
		return false
	}
}
