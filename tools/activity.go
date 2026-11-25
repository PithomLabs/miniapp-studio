package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PithomLabs/miniapp/registry"
)

type activityListIn struct {
	Limit   int      `json:"limit"`
	Cursor  string   `json:"cursor"`
	Exclude []string `json:"exclude"`
}

// activityList exposes tenant-scoped framework audit events with keyset
// pagination and recursive secret scrubbing of detail payloads.
func (d Deps) activityList(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in activityListIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	limit, err := boundedLimit(in.Limit)
	if err != nil {
		return nil, err
	}
	var cursor int64
	if in.Cursor != "" {
		cursor, err = strconv.ParseInt(in.Cursor, 10, 64)
		if err != nil || cursor <= 0 {
			return nil, errInvalidArguments()
		}
	}
	if len(in.Exclude) > 16 {
		return nil, errInvalidArguments()
	}
	for _, tool := range in.Exclude {
		if tool == "" || len(tool) > 128 {
			return nil, errInvalidArguments()
		}
	}
	// The listing excludes its own dispatch record: the runtime audits every
	// invocation including this read, and showing the observer would make
	// the empty state unreachable. The record itself still exists.
	query := `SELECT id, at, agent_id, miniapp_id, miniapp_version, tool, action, outcome, details
		FROM app.audit_events WHERE tenant_id = $1
		AND (invocation_id IS NULL OR invocation_id::TEXT <> $2)`
	args := []any{tenant, call.InvocationID}
	next := 3
	if cursor > 0 {
		query += ` AND id < $3`
		args = append(args, cursor)
		next = 4
	}
	// Exclusion applies inside the query so the limit counts kept
	// rows: filtering after the limit would starve real receipts
	// behind high-frequency transport records (SSE polls).
	if len(in.Exclude) > 0 {
		query += ` AND NOT (tool = ANY($` + strconv.Itoa(next) + `::text[]))`
		args = append(args, in.Exclude)
	}
	query += ` ORDER BY id DESC LIMIT ` + strconv.Itoa(limit)
	rows, err := call.Tx.Query(call.Context, query, args...)
	if err != nil {
		return nil, dbErr(err, "")
	}
	defer rows.Close()
	type event struct {
		ID      int64  `json:"id"`
		At      string `json:"at"`
		Agent   string `json:"agent"`
		Miniapp string `json:"miniapp"`
		Version string `json:"version"`
		Tool    string `json:"tool"`
		Action  string `json:"action"`
		Outcome string `json:"outcome"`
		Details any    `json:"details"`
	}
	events := []event{}
	var last int64
	for rows.Next() {
		var e event
		var details []byte
		var at time.Time
		if err := rows.Scan(&e.ID, &at, &e.Agent, &e.Miniapp, &e.Version,
			&e.Tool, &e.Action, &e.Outcome, &details); err != nil {
			return nil, dbErr(err, "")
		}
		e.At = at.UTC().Format(time.RFC3339)
		if len(details) > 0 {
			var decoded any
			if err := json.Unmarshal(details, &decoded); err == nil {
				e.Details = scrubValue(decoded)
			}
		}
		last = e.ID
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErr(err, "")
	}
	out := map[string]any{"events": events}
	if len(events) == limit {
		out["next_cursor"] = strconv.FormatInt(last, 10)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("studio: encode: %w", err)
	}
	return raw, nil
}

var scrubKeys = map[string]bool{
	"password": true, "secret": true, "token": true, "credential": true,
	"authorization": true, "cookie": true, "api_key": true, "apikey": true,
	"access_token": true, "refresh_token": true, "private_key": true,
	"client_secret": true, "auth_token": true, "bearer_token": true,
	"set-cookie": true, "set_cookie": true,
}

// scrubValue redacts secret-shaped content recursively: denylisted keys are
// dropped whole, and credential-shaped spans inside strings are replaced
// (surrounding text preserved). Structure is otherwise preserved so
// evidence stays useful.
func scrubValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			if scrubKeys[strings.ToLower(k)] {
				out[k] = "[REDACTED]"
				continue
			}
			out[k] = scrubValue(e)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = scrubValue(e)
		}
		return out
	case string:
		return scrubSpans(t)
	default:
		return v
	}
}
