package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/tools"
)

// Checkpoint payload bounds. Context carries operator prose, never
// transcripts or credentials (scrubbed before storage regardless).
const (
	checkpointMaxNote    = 2000
	checkpointMaxContext = 4096
	checkpointMaxHistory = 50
)

// CheckpointVersion is the only writable checkpoint format. Unknown
// versions fail closed at write time.
const CheckpointVersion = "v1"

type checkpointError struct {
	Code    int
	Message string
}

func (e *checkpointError) Error() string { return e.Message }

// openAttempt resolves the currently open attempt: the latest started
// number with no recorded result. Zero means none open.
func openAttempt(ctx context.Context, tx pgx.Tx, tenant, buildID string) (int, error) {
	var started, recorded int
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(seq) FILTER (WHERE type = 'attempt_started'), 0),
		        COALESCE(MAX(seq) FILTER (WHERE type = 'attempt_result'), 0)
		 FROM studio.events WHERE tenant_id = $1 AND build_id = $2
		 AND type IN ('attempt_started', 'attempt_result')`,
		tenant, buildID).Scan(&started, &recorded); err != nil {
		return 0, err
	}
	if started > recorded {
		return started, nil
	}
	return 0, nil
}

// lockOpenBuild locks one owned build and refuses terminal rows.
func lockOpenBuild(ctx context.Context, tx pgx.Tx, tenant, buildID string) (status, convID string, err error) {
	if err := tx.QueryRow(ctx,
		`SELECT status, conversation_id FROM studio.builds
		 WHERE tenant_id = $1 AND id = $2 FOR UPDATE`,
		tenant, buildID).Scan(&status, &convID); err != nil {
		if err == pgx.ErrNoRows {
			return "", "", &checkpointError{http.StatusNotFound, "Unknown build."}
		}
		return "", "", &checkpointError{http.StatusInternalServerError, err.Error()}
	}
	if status != "assigned" && status != "running" {
		return "", "", &checkpointError{http.StatusBadRequest, refuseNonLocal(status)}
	}
	return status, convID, nil
}

// refuseNonLocal names the actual state: a gateway-owned waiting_user
// build is not terminal, and saying so would mislead the operator
// about what to do next. Local checkpoint operations never apply to
// gateway-driven execution; the gateway question must be answered or
// the build cancelled first.
func refuseNonLocal(status string) string {
	if status == "waiting_user" {
		return "Build is waiting on a gateway question (waiting_user); local operations apply to local execution only — answer or cancel the gateway question to proceed."
	}
	return "Build is terminal; open a new build to continue."
}

// insertProgressEvent appends one typed progress event plus its stream
// row. The caller owns idempotency through eventID.
func insertProgressEvent(ctx context.Context, tx pgx.Tx, tenant, buildID, convID, eventID string, seq int, typ string, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO studio.events (tenant_id, build_id, event_id, seq, type, payload)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (tenant_id, build_id, event_id) DO NOTHING`,
		tenant, buildID, eventID, seq, typ, json.RawMessage(raw)); err != nil {
		return err
	}
	_, err = store.AppendStream(ctx, tx, tenant, convID, "event", eventID, buildID)
	return err
}

// CheckpointView is one checkpoint row shaped for display.
type CheckpointView struct {
	ID         string
	Seq        int
	N          int
	Branch     string
	Head       string
	Clean      bool
	Digest     string
	CreatedAt  string
	Note       string
	EntryCount int
}

// LatestCheckpoint returns the newest complete checkpoint for a build,
// or found=false when none exists. Incomplete rows never qualify.
func (s *Server) LatestCheckpoint(ctx context.Context, tenant, buildID string) (CheckpointView, bool, error) {
	var v CheckpointView
	var found bool
	err := store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		var manifest json.RawMessage
		var created time.Time
		err := tx.QueryRow(ctx,
			`SELECT id, seq, attempt_n, git_branch, git_head, git_clean,
			        manifest, created_at, context->>'note'
			 FROM studio.checkpoints
			 WHERE tenant_id = $1 AND build_id = $2 AND complete
			 ORDER BY seq DESC LIMIT 1`,
			tenant, buildID).Scan(&v.ID, &v.Seq, &v.N, &v.Branch, &v.Head,
			&v.Clean, &manifest, &created, &v.Note)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil
			}
			return err
		}
		found = true
		var m struct {
			Digest  string `json:"digest"`
			Entries []any  `json:"entries"`
		}
		if jerr := json.Unmarshal(manifest, &m); jerr != nil {
			return jerr
		}
		v.Digest = m.Digest
		v.EntryCount = len(m.Entries)
		v.CreatedAt = created.UTC().Format(time.RFC3339)
		return nil
	})
	if err != nil {
		return v, false, err
	}
	return v, found, nil
}

// CheckpointHistory returns complete checkpoints newest-first, bounded.
func (s *Server) CheckpointHistory(ctx context.Context, tenant, buildID string) ([]CheckpointView, error) {
	out := []CheckpointView{}
	err := store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, seq, attempt_n, git_branch, git_head, git_clean,
			        manifest->>'digest', created_at, context->>'note'
			 FROM studio.checkpoints
			 WHERE tenant_id = $1 AND build_id = $2 AND complete
			 ORDER BY seq DESC LIMIT $3`,
			tenant, buildID, checkpointMaxHistory)
		if err != nil {
			return err
		}
		for rows.Next() {
			var v CheckpointView
			var created time.Time
			if err := rows.Scan(&v.ID, &v.Seq, &v.N, &v.Branch, &v.Head,
				&v.Clean, &v.Digest, &created, &v.Note); err != nil {
				rows.Close()
				return err
			}
			v.CreatedAt = created.UTC().Format(time.RFC3339)
			out = append(out, v)
		}
		rows.Close()
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// OpenAttemptN reports the open attempt number, or 0 when none is open.
func (s *Server) OpenAttemptN(ctx context.Context, tenant, buildID string) (int, error) {
	var n int
	err := store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		var err error
		n, err = openAttempt(ctx, tx, tenant, buildID)
		return err
	})
	if err != nil {
		return 0, err
	}
	return n, nil
}

// priorProgress returns the seq of an already-recorded progress event
// for this opkey, so a retried POST replays instead of conflicting
// with its own prior success.
func priorProgress(ctx context.Context, tx pgx.Tx, tenant, buildID, eventID string) (int, json.RawMessage, bool) {
	var seq int
	var payload json.RawMessage
	if err := tx.QueryRow(ctx,
		`SELECT seq, payload FROM studio.events
		 WHERE tenant_id = $1 AND build_id = $2 AND event_id = $3`,
		tenant, buildID, eventID).Scan(&seq, &payload); err != nil {
		return 0, nil, false
	}
	return seq, payload, true
}

// opkeyOpFor reports which logical operation already consumed an opkey
// on this build: checkpoint, start, interrupt, resume, or record.
// Empty means unused. Same key, different op is a 409 mismatch —
// replay requires the same operation, never just the same key.
// Scoped per build: the same key on another build is an independent
// operation. (Checkpoint ids are tenant-global, so the checkpoint
// path additionally verifies row ownership on conflict.)
func opkeyOpFor(ctx context.Context, tx pgx.Tx, tenant, buildID, opkey string) (string, error) {
	var exists bool
	if err := tx.QueryRow(ctx,
		`SELECT true FROM studio.checkpoints WHERE tenant_id = $1 AND id = $2 AND build_id = $3`,
		tenant, "ckpt:"+opkey, buildID).Scan(&exists); err != nil && err != pgx.ErrNoRows {
		return "", err
	} else if err == nil {
		return "checkpoint", nil
	}
	var eventID string
	if err := tx.QueryRow(ctx,
		`SELECT event_id FROM studio.events
		 WHERE tenant_id = $1 AND build_id = $2 AND event_id IN ($3, $4, $5, $6, $7) LIMIT 1`,
		tenant, buildID, "start:"+opkey, "interrupt:"+opkey, "resume:"+opkey,
		"ckpt-ev:"+opkey, "attempt:"+opkey).Scan(&eventID); err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	switch {
	case strings.HasPrefix(eventID, "start:"):
		return "start", nil
	case strings.HasPrefix(eventID, "interrupt:"):
		return "interrupt", nil
	case strings.HasPrefix(eventID, "resume:"):
		return "resume", nil
	case strings.HasPrefix(eventID, "ckpt-ev:"):
		return "checkpoint", nil
	default:
		return "record", nil
	}
}

// rejectKeyMismatch fails a same-key different-operation submission.
func rejectKeyMismatch(used string) *checkpointError {
	return &checkpointError{http.StatusConflict,
		fmt.Sprintf("Operation key already used for %s; use a fresh form.", used)}
}

// StartAttempt opens attempt n = completed + 1. Refuses when one is
// already open (409): exactly one execution may be in flight.
func (s *Server) StartAttempt(ctx context.Context, tenant, buildID, opkey string) (int, int, error) {
	n := 0
	err := store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		_, convID, err := lockOpenBuild(ctx, tx, tenant, buildID)
		if err != nil {
			return err
		}
		if seq, _, found := priorProgress(ctx, tx, tenant, buildID, "start:"+opkey); found {
			n = seq
			return nil
		}
		if used, err := opkeyOpFor(ctx, tx, tenant, buildID, opkey); err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		} else if used != "" {
			return rejectKeyMismatch(used)
		}
		if open, err := openAttempt(ctx, tx, tenant, buildID); err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		} else if open > 0 {
			return &checkpointError{http.StatusConflict,
				fmt.Sprintf("Attempt %d is already open; record or interrupt it first.", open)}
		}
		var recorded int
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM studio.events
			 WHERE tenant_id = $1 AND build_id = $2 AND type = 'attempt_result'`,
			tenant, buildID).Scan(&recorded); err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		}
		n = recorded + 1
		return insertProgressEvent(ctx, tx, tenant, buildID, convID,
			"start:"+opkey, n, "attempt_started",
			map[string]any{"n": n, "text": fmt.Sprintf("Attempt %d started", n)})
	})
	if err != nil {
		if cerr, ok := err.(*checkpointError); ok {
			return 0, cerr.Code, cerr
		}
		return 0, http.StatusInternalServerError, err
	}
	return n, 0, nil
}

// InterruptAttempt marks the open attempt interrupted. A marker, not a
// state transition: history stays, and resume still validates.
func (s *Server) InterruptAttempt(ctx context.Context, tenant, buildID, note, opkey string) (int, int, error) {
	if len(note) > checkpointMaxNote {
		return 0, http.StatusBadRequest, &checkpointError{http.StatusBadRequest, "Note is too long."}
	}
	note = tools.RedactSecretSpans(note)
	n := 0
	err := store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		_, convID, err := lockOpenBuild(ctx, tx, tenant, buildID)
		if err != nil {
			return err
		}
		if seq, _, found := priorProgress(ctx, tx, tenant, buildID, "interrupt:"+opkey); found {
			n = seq
			return nil
		}
		if used, err := opkeyOpFor(ctx, tx, tenant, buildID, opkey); err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		} else if used != "" {
			return rejectKeyMismatch(used)
		}
		open, err := openAttempt(ctx, tx, tenant, buildID)
		if err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		}
		if open == 0 {
			return &checkpointError{http.StatusBadRequest, "No open attempt to interrupt."}
		}
		n = open
		text := fmt.Sprintf("Attempt %d interrupted", n)
		if note != "" {
			text += " — " + note
		}
		return insertProgressEvent(ctx, tx, tenant, buildID, convID,
			"interrupt:"+opkey, n, "attempt_interrupted",
			map[string]any{"n": n, "note": note, "text": text})
	})
	if err != nil {
		if cerr, ok := err.(*checkpointError); ok {
			return 0, cerr.Code, cerr
		}
		return 0, http.StatusInternalServerError, err
	}
	return n, 0, nil
}

// CheckpointOutcome is one captured recovery point.
type CheckpointOutcome struct {
	ID     string
	Seq    int
	N      int
	Replay bool
}

// CreateCheckpoint captures the worktree into a durable checkpoint on
// the open attempt. Manifest and digests come from server-side Git
// inspection only — never from caller input. Same opkey replays.
func (s *Server) CreateCheckpoint(ctx context.Context, tenant, buildID, note, agentContext, opkey string) (*CheckpointOutcome, int, error) {
	if len(note) > checkpointMaxNote || len(agentContext) > checkpointMaxContext {
		return nil, http.StatusBadRequest, &checkpointError{http.StatusBadRequest, "Checkpoint text is too long."}
	}
	if s.cfg.WorkspaceRoot == "" {
		return nil, http.StatusBadRequest, &checkpointError{http.StatusBadRequest, "Workspace is not configured."}
	}
	note = tools.RedactSecretSpans(note)
	agentContext = tools.RedactSecretSpans(agentContext)
	// Inspect before opening the transaction: a failing worktree must
	// not burn database state, and inspection takes no locks.
	state, err := InspectWorktree(ctx, s.cfg.WorkspaceRoot)
	if err != nil {
		return nil, http.StatusBadRequest, &checkpointError{http.StatusBadRequest, err.Error()}
	}
	out := &CheckpointOutcome{}
	err = store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		_, convID, err := lockOpenBuild(ctx, tx, tenant, buildID)
		if err != nil {
			return err
		}
		// Same-key replay must verify the row belongs to THIS build:
		// checkpoint ids are tenant-global, so a cross-build key reuse
		// would otherwise return another build's checkpoint as "replay".
		var priorBuild, priorID string
		var priorSeq, priorN int
		if err := tx.QueryRow(ctx,
			`SELECT build_id, id, seq, attempt_n FROM studio.checkpoints
			 WHERE tenant_id = $1 AND id = $2`,
			tenant, "ckpt:"+opkey).Scan(&priorBuild, &priorID, &priorSeq, &priorN); err != nil && err != pgx.ErrNoRows {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		} else if err == nil {
			if priorBuild != buildID {
				return &checkpointError{http.StatusConflict,
					"Operation key already used for a checkpoint on a different build."}
			}
			out.ID, out.Seq, out.N, out.Replay = priorID, priorSeq, priorN, true
			return nil
		}
		if used, err := opkeyOpFor(ctx, tx, tenant, buildID, opkey); err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		} else if used != "" {
			return rejectKeyMismatch(used)
		}
		n, err := openAttempt(ctx, tx, tenant, buildID)
		if err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		}
		if n == 0 {
			return &checkpointError{http.StatusBadRequest, "No open attempt; start one first."}
		}
		var turnID string
		if err := tx.QueryRow(ctx,
			`SELECT id FROM studio.turns WHERE tenant_id = $1 AND build_id = $2
			 ORDER BY n DESC LIMIT 1`,
			tenant, buildID).Scan(&turnID); err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		}
		var seq int
		if err := tx.QueryRow(ctx,
			`SELECT COALESCE(MAX(seq), 0) + 1 FROM studio.checkpoints
			 WHERE tenant_id = $1 AND build_id = $2`,
			tenant, buildID).Scan(&seq); err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		}
		manifestRaw, err := json.Marshal(map[string]any{
			"format": ManifestFormatV2, "digest": state.Digest, "entries": state.Entries,
		})
		if err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		}
		contextRaw, err := json.Marshal(map[string]any{
			"note": note, "agent_context": agentContext,
		})
		if err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		}
		id := "ckpt:" + opkey
		var inserted bool
		if err := tx.QueryRow(ctx,
			`INSERT INTO studio.checkpoints
			 (id, tenant_id, build_id, attempt_n, turn_id, version, complete, seq,
			  repo_path, git_branch, git_head, git_clean, manifest, context)
			 VALUES ($1, $2, $3, $4, $5, 'v1', true, $6, $7, $8, $9, $10, $11, $12)
			 ON CONFLICT (tenant_id, id) DO NOTHING
			 RETURNING true`,
			id, tenant, buildID, n, turnID, seq, s.cfg.WorkspaceRoot,
			state.Branch, state.Head, state.Clean,
			json.RawMessage(manifestRaw), json.RawMessage(contextRaw)).Scan(&inserted); err != nil {
			if err == pgx.ErrNoRows {
				// Lost a same-key race: re-read under the lock and
				// verify the winner belongs to this build.
				var winnerBuild, winnerID string
				var winnerSeq, winnerN int
				if qerr := tx.QueryRow(ctx,
					`SELECT build_id, id, seq, attempt_n FROM studio.checkpoints
					 WHERE tenant_id = $1 AND id = $2`,
					tenant, id).Scan(&winnerBuild, &winnerID, &winnerSeq, &winnerN); qerr != nil {
					return &checkpointError{http.StatusInternalServerError, qerr.Error()}
				}
				if winnerBuild != buildID {
					return &checkpointError{http.StatusConflict,
						"Operation key already used for a checkpoint on a different build."}
				}
				out.ID, out.Seq, out.N, out.Replay = winnerID, winnerSeq, winnerN, true
				return nil
			}
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		}
		if err := insertProgressEvent(ctx, tx, tenant, buildID, convID,
			"ckpt-ev:"+opkey, seq, "checkpoint_recorded", map[string]any{
				"checkpoint_id": id, "seq": seq, "n": n,
				"branch": state.Branch, "head": state.Head,
				"manifest_digest": state.Digest,
				"text":            fmt.Sprintf("Checkpoint #%d captured (attempt %d)", seq, n),
			}); err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		}
		out.ID, out.Seq, out.N = id, seq, n
		return nil
	})
	if err != nil {
		if cerr, ok := err.(*checkpointError); ok {
			return nil, cerr.Code, cerr
		}
		return nil, http.StatusInternalServerError, err
	}
	return out, 0, nil
}

// ResumeOutcome links a fresh attempt to its source checkpoint.
type ResumeOutcome struct {
	N            int
	CheckpointID string
	Reconciled   bool
}

// attemptForm carries the shared Console form fields.
type attemptForm struct {
	opkey        string
	tenant       string
	binding      string
	convID       string
	buildID      string
	note         string
	agentContext string
}

// parseAttemptForm enforces the operator-only contract shared by the
// B1 progress actions: session, Origin (at the route), opkey, and
// server-side tenant/binding. POSTed fields after it are per-action.
func parseAttemptForm(w http.ResponseWriter, r *http.Request, s *Server) (attemptForm, bool) {
	var f attemptForm
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return f, false
	}
	key, err := opKey(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return f, false
	}
	tenant, binding, _, ok := TenantOf(r)
	if !ok || tenant == "" || binding == "" {
		renderError(w, r, http.StatusUnauthorized, "Authentication required.", s.logger, "")
		return f, false
	}
	f.opkey, f.tenant, f.binding = key, tenant, binding
	f.convID, f.buildID = r.Form.Get("conversation_id"), r.Form.Get("build_id")
	f.note, f.agentContext = r.Form.Get("note"), r.Form.Get("agent_context")
	if f.buildID == "" {
		renderError(w, r, http.StatusBadRequest, "Build is required.", s.logger, "")
		return f, false
	}
	return f, true
}

func (s *Server) redirectRun(w http.ResponseWriter, r *http.Request, convID, notice string) {
	http.Redirect(w, r, "/console?scene=run&conversation="+url.QueryEscape(convID)+
		"&notice="+url.QueryEscape(notice), http.StatusFound)
}

func (s *Server) renderOpError(w http.ResponseWriter, r *http.Request, code int, err error) {
	switch code {
	case http.StatusNotFound:
		renderError(w, r, http.StatusNotFound, err.Error(), s.logger, "")
	case http.StatusConflict:
		renderError(w, r, http.StatusConflict, err.Error(), s.logger, "")
	case http.StatusInternalServerError:
		renderError(w, r, http.StatusInternalServerError, "Operation failed.", s.logger, err.Error())
	default:
		renderError(w, r, http.StatusBadRequest, err.Error(), s.logger, "")
	}
}

func (s *Server) handleConsoleAttemptStart(w http.ResponseWriter, r *http.Request) {
	f, ok := parseAttemptForm(w, r, s)
	if !ok {
		return
	}
	n, code, err := s.StartAttempt(r.Context(), f.tenant, f.buildID, f.opkey)
	if err != nil {
		s.renderOpError(w, r, code, err)
		return
	}
	s.redirectRun(w, r, f.convID, fmt.Sprintf("Attempt %d started.", n))
}

func (s *Server) handleConsoleCheckpointCapture(w http.ResponseWriter, r *http.Request) {
	f, ok := parseAttemptForm(w, r, s)
	if !ok {
		return
	}
	out, code, err := s.CreateCheckpoint(r.Context(), f.tenant, f.buildID, f.note, f.agentContext, f.opkey)
	if err != nil {
		s.renderOpError(w, r, code, err)
		return
	}
	notice := fmt.Sprintf("Checkpoint #%d captured (attempt %d).", out.Seq, out.N)
	if out.Replay {
		notice = fmt.Sprintf("Checkpoint #%d already recorded — replayed.", out.Seq)
	}
	s.redirectRun(w, r, f.convID, notice)
}

func (s *Server) handleConsoleAttemptInterrupt(w http.ResponseWriter, r *http.Request) {
	f, ok := parseAttemptForm(w, r, s)
	if !ok {
		return
	}
	n, code, err := s.InterruptAttempt(r.Context(), f.tenant, f.buildID, f.note, f.opkey)
	if err != nil {
		s.renderOpError(w, r, code, err)
		return
	}
	s.redirectRun(w, r, f.convID, fmt.Sprintf("Attempt %d marked interrupted.", n))
}

func (s *Server) handleConsoleAttemptResume(w http.ResponseWriter, r *http.Request) {
	f, ok := parseAttemptForm(w, r, s)
	if !ok {
		return
	}
	reconcile := r.Form.Get("reconcile") == "on"
	out, code, err := s.ResumeAttempt(r.Context(), f.tenant, f.buildID,
		r.Form.Get("checkpoint_id"), reconcile, f.note, f.opkey)
	if err != nil {
		s.renderOpError(w, r, code, err)
		return
	}
	notice := fmt.Sprintf("Attempt %d resumed from checkpoint.", out.N)
	if out.Reconciled {
		notice = fmt.Sprintf("Attempt %d resumed with reconciliation.", out.N)
	}
	s.redirectRun(w, r, f.convID, notice)
}

// ResumeAttempt validates the latest complete checkpoint against the
// live worktree and opens a resumed attempt from it. A mismatch
// without explicit reconcile refuses with the discrepancy; nothing
// is reset, cleaned, or overwritten, ever.
func (s *Server) ResumeAttempt(ctx context.Context, tenant, buildID, checkpointID string, reconcile bool, note, opkey string) (*ResumeOutcome, int, error) {
	if s.cfg.WorkspaceRoot == "" {
		return nil, http.StatusBadRequest, &checkpointError{http.StatusBadRequest, "Workspace is not configured."}
	}
	note = tools.RedactSecretSpans(note)
	if reconcile && len(note) == 0 {
		return nil, http.StatusBadRequest, &checkpointError{http.StatusBadRequest, "Reconciliation requires a note."}
	}
	if len(note) > checkpointMaxNote {
		return nil, http.StatusBadRequest, &checkpointError{http.StatusBadRequest, "Note is too long."}
	}
	state, err := InspectWorktree(ctx, s.cfg.WorkspaceRoot)
	if err != nil {
		return nil, http.StatusBadRequest, &checkpointError{http.StatusBadRequest, err.Error()}
	}
	out := &ResumeOutcome{}
	err = store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		_, convID, err := lockOpenBuild(ctx, tx, tenant, buildID)
		if err != nil {
			return err
		}
		if seq, payload, found := priorProgress(ctx, tx, tenant, buildID, "resume:"+opkey); found {
			var p map[string]any
			_ = json.Unmarshal(payload, &p)
			rf, _ := p["resumed_from"].(string)
			rc, _ := p["reconciled"].(bool)
			out.N, out.CheckpointID, out.Reconciled = seq, rf, rc
			return nil
		}
		if used, err := opkeyOpFor(ctx, tx, tenant, buildID, opkey); err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		} else if used != "" {
			return rejectKeyMismatch(used)
		}
		var latest, branch, head, manifestDigest string
		var latestSeq int
		var format *string
		if err := tx.QueryRow(ctx,
			`SELECT c.id, c.seq, c.git_branch, c.git_head, c.manifest->>'digest', c.manifest->>'format'
			 FROM studio.checkpoints AS c
			 WHERE c.tenant_id = $1 AND c.build_id = $2 AND c.complete
			 ORDER BY c.seq DESC LIMIT 1`,
			tenant, buildID).Scan(&latest, &latestSeq, &branch, &head, &manifestDigest, &format); err != nil {
			if err == pgx.ErrNoRows {
				return &checkpointError{http.StatusBadRequest, "No complete checkpoint to resume from."}
			}
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		}
		if latest != checkpointID {
			return &checkpointError{http.StatusConflict, "Checkpoint superseded; refresh and resume from the latest."}
		}
		rowFormat := ""
		if format != nil {
			rowFormat = *format
		}
		if diff := state.DiffAgainst(rowFormat, branch, head, manifestDigest); diff != "" && !reconcile {
			return &checkpointError{http.StatusConflict, "Worktree differs from checkpoint: " + diff}
		}
		if open, err := openAttempt(ctx, tx, tenant, buildID); err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		} else if open > 0 {
			return &checkpointError{http.StatusConflict,
				fmt.Sprintf("Attempt %d is already open; record or interrupt it first.", open)}
		}
		var recorded int
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM studio.events
			 WHERE tenant_id = $1 AND build_id = $2 AND type = 'attempt_result'`,
			tenant, buildID).Scan(&recorded); err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		}
		n := recorded + 1
		text := fmt.Sprintf("Attempt %d resumed from checkpoint #%d (reconstructed)", n, latestSeq)
		if reconcile {
			text += " — reconciled: " + note
		}
		if err := insertProgressEvent(ctx, tx, tenant, buildID, convID,
			"resume:"+opkey, n, "attempt_started", map[string]any{
				"n": n, "resumed_from": latest, "reconciled": reconcile,
				"note": note, "text": text,
			}); err != nil {
			return &checkpointError{http.StatusInternalServerError, err.Error()}
		}
		out.N, out.CheckpointID, out.Reconciled = n, latest, reconcile
		return nil
	})
	if err != nil {
		if cerr, ok := err.(*checkpointError); ok {
			return nil, cerr.Code, cerr
		}
		return nil, http.StatusInternalServerError, err
	}
	return out, 0, nil
}
