package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/PithomLabs/miniapp-studio/artifact"
	"github.com/PithomLabs/miniapp-studio/spec"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/tools"
)

// handleConsoleArtifactDecide records one human decision on a bundle
// artifact. Deliberately NOT a registry tool (the kill-switch
// precedent in console_stop.go): operator session + Origin required,
// tenant and decider derived server-side, no agent surface reaches
// it. See DecideArtifact for the contract.
func (s *Server) handleConsoleArtifactDecide(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	if _, err := opKey(r); err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	tenant, binding, _, ok := TenantOf(r)
	if !ok || tenant == "" || binding == "" {
		renderError(w, r, http.StatusUnauthorized, "Authentication required.", s.logger, "")
		return
	}
	artifactID, decision, note := r.Form.Get("artifact_id"), r.Form.Get("decision"), r.Form.Get("note")
	convID := r.Form.Get("conversation_id")
	if artifactID == "" || (decision != "approved" && decision != "rejected") {
		renderError(w, r, http.StatusBadRequest, "Artifact and an approved/rejected decision are required.", s.logger, "")
		return
	}
	if len(note) > 2000 {
		renderError(w, r, http.StatusBadRequest, "Note is too long.", s.logger, "")
		return
	}
	out, derr := s.DecideArtifact(r.Context(), tenant, binding, artifactID, decision, note)
	if derr != nil {
		switch derr.Code {
		case http.StatusNotFound:
			renderError(w, r, http.StatusNotFound, derr.Message, s.logger, "")
		case http.StatusConflict:
			renderError(w, r, http.StatusConflict, derr.Message, s.logger, "")
		case http.StatusInternalServerError:
			renderError(w, r, http.StatusInternalServerError, "Decision failed.", s.logger, derr.Message)
		default:
			renderError(w, r, http.StatusBadRequest, derr.Message, s.logger, "")
		}
		return
	}
	notice := "Approved artifact " + shortDigest(out.Digest) + "."
	if decision == "rejected" {
		notice = "Rejected artifact " + shortDigest(out.Digest) + ": " + out.ReworkBrief
	}
	http.Redirect(w, r, "/console?scene=evidence&conversation="+url.QueryEscape(convID)+
		"&notice="+url.QueryEscape(notice), http.StatusFound)
}

// DecideError is a failed decision with the honest HTTP status.
type DecideError struct {
	Code    int
	Message string
}

func (e *DecideError) Error() string { return e.Message }

// DecideOutcome is the durable result of one decision.
type DecideOutcome struct {
	DecisionID string
	Digest     string
	Replay     bool
	// ReworkBrief carries rejection reasons for the next build.
	// Rejections never auto-continue: terminal builds refuse
	// continue rows (sticky terminality), so rework starts via a
	// new message carrying this brief — operator or agent driven.
	ReworkBrief string
	SpecVersion int
}

// DecideArtifact records a human decision on a verified bundle. One
// transaction: lock the artifact row (serializes concurrent
// decides), require bundle kind + verified_at, re-hash stored
// bytes against the pinned digest, require terminal test evidence
// (approve additionally requires passed), save-or-cite the spec
// version, insert the decision row. Same-outcome duplicates replay;
// anything else conflicts (409), never overwrites.
func (s *Server) DecideArtifact(ctx context.Context, tenant, binding, artifactID, decision, note string) (*DecideOutcome, *DecideError) {
	var out *DecideOutcome
	err := store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		var derr *DecideError
		out, derr = s.decideInTx(ctx, tx, tenant, binding, artifactID, decision, note)
		if derr != nil {
			return &decideAbort{derr: derr}
		}
		return nil
	})
	if err != nil {
		var abort *decideAbort
		if errors.As(err, &abort) {
			return nil, abort.derr
		}
		return nil, &DecideError{http.StatusInternalServerError, err.Error()}
	}
	return out, nil
}

// decideAbort carries a DecideError through the tx rollback.
type decideAbort struct {
	derr *DecideError
}

func (e *decideAbort) Error() string { return e.derr.Message }

// decideInTx runs the decision inside an armed tenant transaction.
func (s *Server) decideInTx(ctx context.Context, tx pgx.Tx, tenant, binding, artifactID, decision, note string) (*DecideOutcome, *DecideError) {
	var locked bool
	if err := tx.QueryRow(ctx,
		`SELECT TRUE FROM studio.artifacts WHERE tenant_id = $1 AND id = $2 FOR UPDATE`,
		tenant, artifactID).Scan(&locked); err != nil {
		if err == pgx.ErrNoRows {
			return nil, &DecideError{http.StatusNotFound, "Unknown artifact."}
		}
		return nil, &DecideError{http.StatusInternalServerError, err.Error()}
	}
	data, err := store.LoadBundle(ctx, tx, tenant, artifactID)
	if err != nil {
		return nil, &DecideError{http.StatusInternalServerError, err.Error()}
	}
	if data.Kind != "bundle" {
		return nil, &DecideError{http.StatusBadRequest, "Only bundle artifacts can be decided."}
	}
	if !data.Verified {
		return nil, &DecideError{http.StatusBadRequest, "Artifact is not verified."}
	}
	env, errs := artifact.ParseEnvelope(data.Envelope)
	if errs != nil {
		return nil, &DecideError{http.StatusInternalServerError, "stored envelope no longer parses"}
	}
	root, errs := artifact.ComputeRoot(data.Envelope, data.Members)
	if errs != nil || root != data.Digest {
		// Verified bytes changed underneath: corruption or storage
		// fault. Never decide on bytes that fail re-proof.
		return nil, &DecideError{http.StatusInternalServerError, "artifact bytes changed since verification"}
	}
	// Existing decision: same outcome replays, anything else 409s.
	var prevDecision, prevDigest string
	err = tx.QueryRow(ctx,
		`SELECT decision, artifact_digest FROM studio.approvals WHERE tenant_id = $1 AND artifact_id = $2`,
		tenant, artifactID).Scan(&prevDecision, &prevDigest)
	if err == nil {
		if prevDecision == decision && prevDigest == data.Digest {
			return &DecideOutcome{Digest: data.Digest, Replay: true}, nil
		}
		return nil, &DecideError{http.StatusConflict, "Artifact already decided differently."}
	}
	if err != pgx.ErrNoRows {
		return nil, &DecideError{http.StatusInternalServerError, err.Error()}
	}
	// Test gate: a terminal run on these exact bytes must exist;
	// approval additionally requires it passed.
	var runID, runStatus string
	err = tx.QueryRow(ctx,
		`SELECT id, status FROM studio.artifact_test_runs
		 WHERE tenant_id = $1 AND artifact_id = $2 AND input_digest = $3
		 AND status IN ('passed', 'failed', 'blocked')
		 ORDER BY finished_at DESC NULLS LAST, created_at DESC LIMIT 1`,
		tenant, artifactID, data.Digest).Scan(&runID, &runStatus)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, &DecideError{http.StatusBadRequest, "No test evidence: run tests first."}
		}
		return nil, &DecideError{http.StatusInternalServerError, err.Error()}
	}
	if decision == "approved" && runStatus != "passed" {
		return nil, &DecideError{http.StatusBadRequest, "Approval requires passing tests."}
	}
	// Spec binding: envelope app must equal the embedded spec's app,
	// then save-or-cite the spec version atomically with the decision.
	specRaw, ok := data.Members[artifact.SpecFileName]
	if !ok {
		return nil, &DecideError{http.StatusInternalServerError, "spec bytes missing"}
	}
	specHash, err := spec.ContentHashOf(specRaw)
	if err != nil {
		return nil, &DecideError{http.StatusInternalServerError, "spec bytes no longer parse"}
	}
	var doc struct {
		App struct {
			ID string `json:"id"`
		} `json:"app"`
	}
	if err := json.Unmarshal(specRaw, &doc); err != nil || doc.App.ID == "" {
		return nil, &DecideError{http.StatusInternalServerError, "spec app id unreadable"}
	}
	if env.AppID != doc.App.ID {
		return nil, &DecideError{http.StatusBadRequest, "Envelope app does not match the spec."}
	}
	var appExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM studio.apps WHERE tenant_id = $1 AND id = $2)`,
		tenant, doc.App.ID).Scan(&appExists); err != nil {
		return nil, &DecideError{http.StatusInternalServerError, err.Error()}
	}
	if !appExists {
		return nil, &DecideError{http.StatusBadRequest, "Unknown app."}
	}
	version, found, err := tools.FindSpecVersion(ctx, tx, tenant, doc.App.ID, specHash)
	if err != nil {
		return nil, &DecideError{http.StatusInternalServerError, err.Error()}
	}
	if !found {
		if version, _, err = tools.SaveSpecVersion(ctx, tx, tenant, doc.App.ID, json.RawMessage(specRaw)); err != nil {
			return nil, &DecideError{http.StatusInternalServerError, err.Error()}
		}
	}
	var verificationID string
	if err := tx.QueryRow(ctx,
		`SELECT id FROM studio.artifact_verifications WHERE tenant_id = $1 AND artifact_id = $2
		 ORDER BY created_at DESC LIMIT 1`, tenant, artifactID).Scan(&verificationID); err != nil {
		return nil, &DecideError{http.StatusInternalServerError, "verification evidence missing"}
	}
	decisionID, err := store.RandomHex(16)
	if err != nil {
		return nil, &DecideError{http.StatusInternalServerError, err.Error()}
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO studio.approvals (id, tenant_id, build_id, artifact_id, artifact_digest,
		 verification_id, test_run_id, decision, decided_by, note)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		decisionID, tenant, data.BuildID, artifactID, data.Digest,
		verificationID, runID, decision, binding, note); err != nil {
		// The artifact lock serializes Studio writers, so a unique
		// violation here means a concurrent decide won first.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, &DecideError{http.StatusConflict, "Artifact already decided."}
		}
		return nil, &DecideError{http.StatusInternalServerError, err.Error()}
	}
	out := &DecideOutcome{DecisionID: decisionID, Digest: data.Digest, SpecVersion: version}
	if decision == "rejected" {
		out.ReworkBrief = "rework " + data.Digest + ": " + note
	}
	var conv string
	if err := tx.QueryRow(ctx,
		`SELECT conversation_id FROM studio.builds WHERE tenant_id = $1 AND id = $2`,
		tenant, data.BuildID).Scan(&conv); err != nil {
		return nil, &DecideError{http.StatusInternalServerError, err.Error()}
	}
	if _, err := store.AppendStream(ctx, tx, tenant, conv, "artifact_decided", artifactID, data.BuildID); err != nil {
		return nil, &DecideError{http.StatusInternalServerError, err.Error()}
	}
	return out, nil
}

func shortDigest(digest string) string {
	if len(digest) > 15 {
		return digest[:15] + "…"
	}
	return digest
}
