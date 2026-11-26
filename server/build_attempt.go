package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/tools"
)

// Attempt result bounds. Reports carry agent output (bounded high);
// digests and SHAs stay small and validated.
const (
	attemptMaxBranch   = 256
	attemptMaxDigest   = 256
	attemptMaxCommits  = 2000
	attemptMaxNote     = 2000
	attemptMaxSummary  = 2000
	attemptMaxReport   = 32768
	attemptMaxSHALines = 50
)

// AttemptInput is one manual-bridge execution result plus the human
// verdict on it. B0 records result and decision together: the report
// already contains the test outcomes the human evaluates.
type AttemptInput struct {
	Branch       string
	StatusDigest string
	Commits      []string
	DirtyNote    string
	TestsPassed  bool
	TestsSummary string
	Report       string
	Verdict      string
	VerdictNote  string
}

// AttemptOutcome is the durable result of one recording.
type AttemptOutcome struct {
	N           int
	ParentN     int
	Replay      bool
	BuildStatus string
}

// handleConsoleAttemptResult records one manual-bridge attempt.
// Deliberately NOT a registry tool (the kill-switch precedent):
// operator session + Origin required, tenant and recorder derived
// server-side. Same opkey replays the recorded attempt; a fresh key
// records a new one.
func (s *Server) handleConsoleAttemptResult(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	key, err := opKey(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	tenant, binding, _, ok := TenantOf(r)
	if !ok || tenant == "" || binding == "" {
		renderError(w, r, http.StatusUnauthorized, "Authentication required.", s.logger, "")
		return
	}
	buildID, convID := r.Form.Get("build_id"), r.Form.Get("conversation_id")
	in, verr := parseAttemptInput(r)
	if verr != "" {
		renderError(w, r, http.StatusBadRequest, verr, s.logger, "")
		return
	}
	out, code, err := s.RecordAttempt(r.Context(), tenant, binding, buildID, in, key)
	if err != nil {
		switch code {
		case http.StatusNotFound:
			renderError(w, r, http.StatusNotFound, err.Error(), s.logger, "")
		case http.StatusInternalServerError:
			renderError(w, r, http.StatusInternalServerError, "Attempt recording failed.", s.logger, err.Error())
		default:
			renderError(w, r, http.StatusBadRequest, err.Error(), s.logger, "")
		}
		return
	}
	notice := fmt.Sprintf("Attempt %d recorded (%s).", out.N, in.Verdict)
	if out.Replay {
		notice = fmt.Sprintf("Attempt %d already recorded — replayed.", out.N)
	}
	http.Redirect(w, r, "/console?scene=run&conversation="+url.QueryEscape(convID)+
		"&notice="+url.QueryEscape(notice), http.StatusFound)
}

// parseAttemptInput validates the result form. Secrets never persist:
// / reports, notes, and summaries scrub before storage (the ingest
// write-path precedent).
func parseAttemptInput(r *http.Request) (AttemptInput, string) {
	var in AttemptInput
	// Branch and digest render in the Console and packet: scrub
	// secret-shaped spans like every other operator field. Commits
	// need no scrubbing — hex validation admits nothing the span
	// table can match.
	in.Branch = tools.RedactSecretSpans(strings.TrimSpace(r.Form.Get("branch")))
	in.StatusDigest = tools.RedactSecretSpans(strings.TrimSpace(r.Form.Get("status_digest")))
	in.DirtyNote = tools.RedactSecretSpans(r.Form.Get("dirty_note"))
	in.TestsSummary = tools.RedactSecretSpans(r.Form.Get("tests_summary"))
	in.Report = tools.RedactSecretSpans(r.Form.Get("report"))
	in.VerdictNote = tools.RedactSecretSpans(r.Form.Get("verdict_note"))
	in.Verdict = r.Form.Get("verdict")
	switch r.Form.Get("tests_passed") {
	case "on", "true", "1":
		in.TestsPassed = true
	}
	for _, line := range strings.Split(r.Form.Get("commits"), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			in.Commits = append(in.Commits, trimmed)
		}
	}
	if in.Branch == "" {
		return in, "Branch is required."
	}
	if len(in.Branch) > attemptMaxBranch {
		return in, "Branch is too long."
	}
	if len(in.StatusDigest) > attemptMaxDigest {
		return in, "Status digest is too long."
	}
	if len(in.Commits) > attemptMaxSHALines {
		return in, "Too many commit lines."
	}
	for _, sha := range in.Commits {
		if !validSHALine(sha) {
			return in, "Commit lines must be 7–64 hex characters."
		}
	}
	if len(in.DirtyNote) > attemptMaxNote || len(in.VerdictNote) > attemptMaxNote {
		return in, "Note is too long."
	}
	if in.TestsSummary == "" {
		return in, "Test summary is required."
	}
	if len(in.TestsSummary) > attemptMaxSummary {
		return in, "Test summary is too long."
	}
	if in.Report == "" {
		return in, "Agent report is required."
	}
	if len(in.Report) > attemptMaxReport {
		return in, "Agent report is too long."
	}
	switch in.Verdict {
	case "accept", "reject", "rework":
	default:
		return in, "Verdict must be accept, reject, or rework."
	}
	return in, ""
}

func validSHALine(s string) bool {
	if len(s) < 7 || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// RecordAttempt persists one attempt as a build event plus a stream
// row, atomically. The event id derives from the opkey, so a retried
// POST replays the recorded attempt instead of duplicating it;
// numbering serializes on the locked build row. Accept closes the
// build (stepping through running, the ingest-passthrough
// precedent); reject and rework leave it open for the next attempt.
func (s *Server) RecordAttempt(ctx context.Context, tenant, binding, buildID string, in AttemptInput, opkey string) (*AttemptOutcome, int, error) {
	// Scrub at the persistence boundary so every caller — HTTP form or
	// direct — stores the same safe values. Redaction is idempotent, so
	// the handler's earlier pass is harmless, not double-applied.
	in.Branch = tools.RedactSecretSpans(in.Branch)
	in.StatusDigest = tools.RedactSecretSpans(in.StatusDigest)
	in.DirtyNote = tools.RedactSecretSpans(in.DirtyNote)
	in.TestsSummary = tools.RedactSecretSpans(in.TestsSummary)
	in.Report = tools.RedactSecretSpans(in.Report)
	in.VerdictNote = tools.RedactSecretSpans(in.VerdictNote)
	out := &AttemptOutcome{}
	err := store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		var status, convID string
		if err := tx.QueryRow(ctx,
			`SELECT status, conversation_id FROM studio.builds
			 WHERE tenant_id = $1 AND id = $2 FOR UPDATE`,
			tenant, buildID).Scan(&status, &convID); err != nil {
			if err == pgx.ErrNoRows {
				return &attemptError{http.StatusNotFound, "Unknown build."}
			}
			return &attemptError{http.StatusInternalServerError, err.Error()}
		}
		if status != "assigned" && status != "running" {
			return &attemptError{http.StatusBadRequest, refuseNonLocal(status)}
		}
		// Same key, different operation is a mismatch, not a replay.
		// ("record" itself falls through to the conflict path below.)
		if used, err := opkeyOpFor(ctx, tx, tenant, buildID, opkey); err != nil {
			return &attemptError{http.StatusInternalServerError, err.Error()}
		} else if used != "" && used != "record" {
			return &attemptError{http.StatusConflict,
				fmt.Sprintf("Operation key already used for %s; use a fresh form.", used)}
		}
		var n int
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM studio.events
			 WHERE tenant_id = $1 AND build_id = $2 AND type = 'attempt_result'`,
			tenant, buildID).Scan(&n); err != nil {
			return &attemptError{http.StatusInternalServerError, err.Error()}
		}
		n++
		parent := n - 1
		eventID := "attempt:" + opkey
		payloadMap := map[string]any{
			"n": n, "parent_n": parent,
			"verdict": in.Verdict, "verdict_note": in.VerdictNote,
			"git": map[string]any{
				"branch": in.Branch, "status_digest": in.StatusDigest,
				"commits": in.Commits, "dirty_note": in.DirtyNote,
			},
			"tests":       map[string]any{"passed": in.TestsPassed, "summary": in.TestsSummary},
			"report":      in.Report,
			"recorded_by": binding,
			"text":        attemptText(n, in),
		}
		// A resumed attempt carries its source checkpoint forward so
		// the packet and history show the full chain. Absent for B0
		// flows, which never start explicitly.
		var startedRaw json.RawMessage
		if err := tx.QueryRow(ctx,
			`SELECT payload FROM studio.events
			 WHERE tenant_id = $1 AND build_id = $2 AND type = 'attempt_started'
			 ORDER BY seq DESC LIMIT 1`,
			tenant, buildID).Scan(&startedRaw); err != nil && err != pgx.ErrNoRows {
			return &attemptError{http.StatusInternalServerError, err.Error()}
		} else if err == nil {
			var sp map[string]any
			if jerr := json.Unmarshal(startedRaw, &sp); jerr == nil {
				if sn, _ := sp["n"].(float64); int(sn) == n {
					if rf, _ := sp["resumed_from"].(string); rf != "" {
						payloadMap["source_checkpoint"] = rf
					}
					if rc, _ := sp["reconciled"].(bool); rc {
						payloadMap["reconciled"] = true
					}
				}
			}
		}
		payload, err := json.Marshal(payloadMap)
		if err != nil {
			return &attemptError{http.StatusInternalServerError, err.Error()}
		}
		var inserted bool
		if err := tx.QueryRow(ctx,
			`INSERT INTO studio.events (tenant_id, build_id, event_id, seq, type, payload)
			 VALUES ($1, $2, $3, $4, 'attempt_result', $5)
			 ON CONFLICT (tenant_id, build_id, event_id) DO NOTHING
			 RETURNING true`,
			tenant, buildID, eventID, n, json.RawMessage(payload)).Scan(&inserted); err != nil {
			if err == pgx.ErrNoRows {
				// Same opkey retried: replay the recorded attempt.
				var prior int
				if qerr := tx.QueryRow(ctx,
					`SELECT seq FROM studio.events
					 WHERE tenant_id = $1 AND build_id = $2 AND event_id = $3`,
					tenant, buildID, eventID).Scan(&prior); qerr != nil {
					return &attemptError{http.StatusInternalServerError, qerr.Error()}
				}
				out.N, out.ParentN, out.Replay, out.BuildStatus = prior, prior-1, true, status
				return nil
			}
			return &attemptError{http.StatusInternalServerError, err.Error()}
		}
		if _, err := store.AppendStream(ctx, tx, tenant, convID, "event", eventID, buildID); err != nil {
			return &attemptError{http.StatusInternalServerError, err.Error()}
		}
		final := status
		if in.Verdict == "accept" {
			// Manual builds have no gateway to transition them;
			// acceptance closes the build through the legal edges.
			if status == "assigned" {
				if _, err := tx.Exec(ctx,
					`UPDATE studio.builds SET status = 'running' WHERE id = $1 AND tenant_id = $2`,
					buildID, tenant); err != nil {
					return &attemptError{http.StatusInternalServerError, err.Error()}
				}
			}
			if _, err := tx.Exec(ctx,
				`UPDATE studio.builds SET status = 'succeeded' WHERE id = $1 AND tenant_id = $2`,
				buildID, tenant); err != nil {
				return &attemptError{http.StatusInternalServerError, err.Error()}
			}
			if _, err := tx.Exec(ctx,
				`UPDATE studio.turns SET status = 'closed'
				 WHERE tenant_id = $1 AND build_id = $2 AND status = 'open'`,
				tenant, buildID); err != nil {
				return &attemptError{http.StatusInternalServerError, err.Error()}
			}
			final = "succeeded"
		}
		out.N, out.ParentN, out.BuildStatus = n, parent, final
		return nil
	})
	if err != nil {
		if aerr, ok := err.(*attemptError); ok {
			return nil, aerr.Code, aerr
		}
		return nil, http.StatusInternalServerError, err
	}
	return out, 0, nil
}

type attemptError struct {
	Code    int
	Message string
}

func (e *attemptError) Error() string { return e.Message }

// attemptText is the one-line Pulse rendering: verdict plus the
// first line of evidence, bounded.
func attemptText(n int, in AttemptInput) string {
	detail := strings.TrimSpace(in.TestsSummary)
	if detail == "" {
		detail = strings.TrimSpace(in.VerdictNote)
	}
	if i := strings.Index(detail, "\n"); i >= 0 {
		detail = detail[:i]
	}
	runes := []rune(detail)
	if len(runes) > 120 {
		detail = string(runes[:120]) + "…"
	}
	if detail == "" {
		return fmt.Sprintf("Attempt %d %sed", n, in.Verdict)
	}
	return fmt.Sprintf("Attempt %d %sed — %s", n, in.Verdict, detail)
}
