package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/artifact"
	"github.com/PithomLabs/miniapp/registry"
)

type artifactStatusIn struct {
	ConversationID string `json:"conversation_id"`
}

// artifactSummary is one artifact's review state.
type artifactSummary struct {
	ID           string         `json:"id"`
	BuildID      string         `json:"build_id"`
	Kind         string         `json:"kind"`
	Digest       string         `json:"digest"`
	CreatedAt    string         `json:"created_at"`
	Verified     bool           `json:"verified"`
	PassedVerifs int            `json:"passed_verifications"`
	PassedRuns   int            `json:"passed_runs"`
	Decision     string         `json:"decision"`
	DecidedBy    string         `json:"decided_by"`
	DecidedAt    string         `json:"decided_at"`
	LatestRun    string         `json:"latest_run_status"`
	Checks       []any          `json:"checks"`
	Report       map[string]any `json:"report"`
	AppID        string         `json:"app_id"`
	SpecHash     string         `json:"spec_hash"`
}

// artifactStatus is the read-only review bundle for one
// conversation: every artifact with verification, test runs, and
// decision state, plus the canonical saved spec for the newest
// bundle's app. The Console scenes and spine derive from this;
// agents may read it (nothing here confers authority).
func (d Deps) artifactStatus(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in artifactStatusIn
	if err := decode(call.Input, &in); err != nil || in.ConversationID == "" {
		return nil, errInvalidArguments()
	}
	if _, err := requireConversation(call.Context, call.Tx, tenant, in.ConversationID, false); err != nil {
		return nil, err
	}
	rows, err := call.Tx.Query(call.Context,
		`SELECT a.id, a.build_id, a.kind, a.digest, a.created_at,
			a.verified_at IS NOT NULL,
			(SELECT count(*) FROM studio.artifact_verifications AS v
			 WHERE v.tenant_id = a.tenant_id AND v.artifact_id = a.id AND v.passed),
			(SELECT count(*) FROM studio.artifact_test_runs AS r
			 WHERE r.tenant_id = a.tenant_id AND r.artifact_id = a.id AND r.status = 'passed'),
			a.manifest
		 FROM studio.artifacts AS a
		 JOIN studio.builds AS b ON b.tenant_id = a.tenant_id AND b.id = a.build_id
		 WHERE a.tenant_id = $1 AND b.conversation_id = $2
		 ORDER BY a.created_at ASC`,
		tenant, in.ConversationID)
	if err != nil {
		return nil, dbErr(err, "")
	}
	type rawRow struct {
		s        artifactSummary
		manifest []byte
	}
	var raws []rawRow
	for rows.Next() {
		var r rawRow
		var createdAt time.Time
		var manifest []byte
		if err := rows.Scan(&r.s.ID, &r.s.BuildID, &r.s.Kind, &r.s.Digest, &createdAt,
			&r.s.Verified, &r.s.PassedVerifs, &r.s.PassedRuns, &manifest); err != nil {
			rows.Close()
			return nil, dbErr(err, "")
		}
		r.s.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		r.manifest = manifest
		raws = append(raws, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, dbErr(err, "")
	}
	// Enrichment queries run after the main rows drain: pgx forbids
	// interleaved queries on one connection ("conn busy").
	out := make([]artifactSummary, 0, len(raws))
	for _, r := range raws {
		s := r.s
		if err := enrichStatus(call, tenant, &s, r.manifest); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return marshalMap(map[string]any{
		"artifacts":      out,
		"canonical_spec": canonicalSpec(call, tenant, out),
	})
}

// enrichStatus fills decision, test, check, and app/spec identity
// detail for one artifact. Missing rows stay zero-valued (honest
// absence); only real failures error.
func enrichStatus(call registry.ToolCall, tenant string, s *artifactSummary, manifest []byte) error {
	ctx := call.Context
	var decidedAt time.Time
	if err := call.Tx.QueryRow(ctx,
		`SELECT decision, decided_by, decided_at FROM studio.approvals
		 WHERE tenant_id = $1 AND artifact_id = $2`,
		tenant, s.ID).Scan(&s.Decision, &s.DecidedBy, &decidedAt); err != nil {
		if err != pgx.ErrNoRows {
			return dbErr(err, "")
		}
	} else {
		s.DecidedAt = decidedAt.UTC().Format(time.RFC3339)
	}
	var reportRaw []byte
	if err := call.Tx.QueryRow(ctx,
		`SELECT status, report FROM studio.artifact_test_runs
		 WHERE tenant_id = $1 AND artifact_id = $2
		 ORDER BY created_at DESC LIMIT 1`,
		tenant, s.ID).Scan(&s.LatestRun, &reportRaw); err != nil && err != pgx.ErrNoRows {
		return dbErr(err, "")
	} else if err == nil {
		var report map[string]any
		if jerr := json.Unmarshal(reportRaw, &report); jerr == nil {
			s.Report = report
		}
	}
	var checksRaw []byte
	if err := call.Tx.QueryRow(ctx,
		`SELECT checks FROM studio.artifact_verifications
		 WHERE tenant_id = $1 AND artifact_id = $2
		 ORDER BY created_at DESC LIMIT 1`,
		tenant, s.ID).Scan(&checksRaw); err != nil && err != pgx.ErrNoRows {
		return dbErr(err, "")
	} else if err == nil {
		var checks []any
		if jerr := json.Unmarshal(checksRaw, &checks); jerr == nil {
			s.Checks = checks
		}
	}
	// App/spec identity comes from the envelope correlation stored in
	// the claim manifest: find the envelope entry, load its bytes,
	// and read app_id plus the spec digest.
	var claim struct {
		Files []struct {
			Path   string `json:"path"`
			Digest string `json:"digest"`
		} `json:"files"`
	}
	if err := json.Unmarshal(manifest, &claim); err != nil {
		return nil
	}
	for _, entry := range claim.Files {
		if entry.Path != artifact.EnvelopeFileName || entry.Digest == "" {
			continue
		}
		var envRaw []byte
		if err := call.Tx.QueryRow(ctx,
			`SELECT bytes FROM studio.artifact_blobs WHERE tenant_id = $1 AND digest = $2`,
			tenant, entry.Digest).Scan(&envRaw); err != nil {
			return nil
		}
		var env struct {
			AppID string `json:"app_id"`
			Spec  struct {
				Digest string `json:"digest"`
			} `json:"spec"`
		}
		if err := json.Unmarshal(envRaw, &env); err != nil {
			return nil
		}
		s.AppID, s.SpecHash = env.AppID, env.Spec.Digest
	}
	return nil
}

// canonicalSpec returns the latest saved spec for the newest
// bundle's app (version, hash, document) or nil when no app or no
// saved version exists. The Spec scene renders from this alone.
func canonicalSpec(call registry.ToolCall, tenant string, out []artifactSummary) map[string]any {
	appID := ""
	for i := len(out) - 1; i >= 0; i-- {
		if out[i].Kind == "bundle" && out[i].AppID != "" {
			appID = out[i].AppID
			break
		}
	}
	if appID == "" {
		return nil
	}
	var version int
	var hash string
	var doc json.RawMessage
	if err := call.Tx.QueryRow(call.Context,
		`SELECT version, spec_hash, document FROM studio.app_specs
		 WHERE tenant_id = $1 AND app_id = $2 ORDER BY version DESC LIMIT 1`,
		tenant, appID).Scan(&version, &hash, &doc); err != nil {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(doc, &decoded); err != nil {
		return nil
	}
	return map[string]any{"app_id": appID, "version": version, "hash": hash, "document": decoded}
}
