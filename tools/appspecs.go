package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PithomLabs/miniapp/registry"
	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/apps"
	"github.com/PithomLabs/miniapp-studio/spec"
)

type validateIn struct {
	Spec json.RawMessage `json:"spec"`
}

type validateIssue struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type validateOut struct {
	Valid  bool            `json:"valid"`
	Errors []validateIssue `json:"errors,omitempty"`
	Hash   string          `json:"hash,omitempty"`
}

// specValidate validates a submitted spec and binds accepted versions to
// their canonical content hash. The spec owner must equal the calling
// tenant; anything else fails validation (never a usable result).
func (d Deps) specValidate(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in validateIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	parsed, errs := spec.Validate(in.Spec)
	if len(errs) != 0 {
		return marshalValidate(false, errs, "")
	}
	if parsed.App.Owner != tenant {
		return marshalValidate(false, []spec.Error{{
			Code: spec.EApp, Path: "app.owner",
			Message: "spec owner does not match calling tenant",
		}}, "")
	}
	hash, err := spec.ContentHash(parsed)
	if err != nil {
		return nil, fmt.Errorf("studio: hash: %w", err)
	}
	return marshalValidate(true, nil, hash)
}

func marshalValidate(valid bool, errs []spec.Error, hash string) (json.RawMessage, error) {
	out := validateOut{Valid: valid, Hash: hash}
	for _, e := range errs {
		out.Errors = append(out.Errors, validateIssue{
			Code: string(e.Code), Path: e.Path, Message: e.Message,
		})
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("studio: encode: %w", err)
	}
	return raw, nil
}

type appCreateIn struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (d Deps) appCreate(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in appCreateIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	slug, err := apps.NormalizeSlug(in.Slug)
	if err != nil {
		return nil, fmt.Errorf("studio: invalid slug")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || len(in.Name) > 256 {
		return nil, fmt.Errorf("studio: invalid name")
	}
	if len(in.Description) > 4096 {
		return nil, fmt.Errorf("studio: description too long")
	}
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	owner := call.Principal.AgentID
	if owner == "" {
		owner = tenant
	}
	if _, err := call.Tx.Exec(call.Context,
		`INSERT INTO studio.apps (id, tenant_id, slug, name, description, owner_identity)
		 VALUES ($1, $2, $3, $4, $5, $6)`, id, tenant, slug, name, in.Description, owner); err != nil {
		return nil, dbErr(err, "slug taken")
	}
	return marshalMap(map[string]any{"app_id": id})
}

type appListIn struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func boundedLimit(limit int) (int, error) {
	if limit == 0 {
		return 50, nil
	}
	if limit < 1 || limit > 100 {
		return 0, fmt.Errorf("studio: invalid limit")
	}
	return limit, nil
}

func (d Deps) appList(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in appListIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	limit, err := boundedLimit(in.Limit)
	if err != nil {
		return nil, err
	}
	if in.Offset < 0 {
		return nil, fmt.Errorf("studio: invalid offset")
	}
	rows, err := call.Tx.Query(call.Context,
		`SELECT a.id, a.slug, a.name, a.description, a.created_at,
			s.version, s.spec_hash, COUNT(*) OVER()
		 FROM studio.apps AS a
		 LEFT JOIN LATERAL (
		   SELECT version, spec_hash FROM studio.app_specs AS s
		   WHERE s.tenant_id = a.tenant_id AND s.app_id = a.id
		   ORDER BY version DESC LIMIT 1
		 ) AS s ON true
		 WHERE a.tenant_id = $1
		 ORDER BY a.created_at DESC, a.id DESC LIMIT $2 OFFSET $3`,
		tenant, limit, in.Offset)
	if err != nil {
		return nil, dbErr(err, "")
	}
	defer rows.Close()
	type appView struct {
		ID          string `json:"id"`
		Slug        string `json:"slug"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Version     int    `json:"version"`
		Hash        string `json:"hash"`
		CreatedAt   string `json:"created_at"`
	}
	apps := []appView{}
	total := 0
	for rows.Next() {
		var v appView
		var version *int
		var hash *string
		var created time.Time
		if err := rows.Scan(&v.ID, &v.Slug, &v.Name, &v.Description, &created,
			&version, &hash, &total); err != nil {
			return nil, dbErr(err, "")
		}
		if version != nil {
			v.Version = *version
		}
		if hash != nil {
			v.Hash = *hash
		}
		v.CreatedAt = created.UTC().Format(time.RFC3339)
		apps = append(apps, v)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErr(err, "")
	}
	return marshalMap(map[string]any{"apps": apps, "total": total})
}

type appRefIn struct {
	AppID string `json:"app_id"`
}

func (d Deps) appGet(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in appRefIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.AppID == "" {
		return nil, errInvalidArguments()
	}
	app, err := requireApp(call.Context, call.Tx, tenant, in.AppID)
	if err != nil {
		return nil, err
	}
	version, hash, err := currentVersion(call.Context, call.Tx, tenant, in.AppID)
	if err != nil {
		return nil, dbErr(err, "")
	}
	return marshalMap(map[string]any{
		"id": app.ID, "slug": app.Slug, "name": app.Name,
		"description": app.Description, "owner": app.OwnerIdentity,
		"version": version, "hash": hash,
	})
}

type appUpdateIn struct {
	AppID       string  `json:"app_id"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (d Deps) appUpdate(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in appUpdateIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.AppID == "" || (in.Name == nil && in.Description == nil) {
		return nil, errInvalidArguments()
	}
	if _, err := requireApp(call.Context, call.Tx, tenant, in.AppID); err != nil {
		return nil, err
	}
	type field struct {
		column string
		value  string
	}
	var fields []field
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" || len(*in.Name) > 256 {
			return nil, fmt.Errorf("studio: invalid name")
		}
		fields = append(fields, field{"name", name})
	}
	if in.Description != nil {
		if len(*in.Description) > 4096 {
			return nil, fmt.Errorf("studio: description too long")
		}
		fields = append(fields, field{"description", *in.Description})
	}
	setClause := "updated_at = now()"
	args := []any{in.AppID, tenant}
	for _, f := range fields {
		args = append(args, f.value)
		setClause += ", " + f.column + " = $" + strconv.Itoa(len(args))
	}
	if _, err := call.Tx.Exec(call.Context,
		`UPDATE studio.apps SET `+setClause+` WHERE id = $1 AND tenant_id = $2`, args...); err != nil {
		return nil, dbErr(err, "")
	}
	app, err := requireApp(call.Context, call.Tx, tenant, in.AppID)
	if err != nil {
		return nil, err
	}
	return marshalMap(map[string]any{
		"id": app.ID, "slug": app.Slug, "name": app.Name, "description": app.Description,
	})
}

type specSaveIn struct {
	AppID string          `json:"app_id"`
	Spec  json.RawMessage `json:"spec"`
}

func (d Deps) specSave(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in specSaveIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.AppID == "" || len(in.Spec) == 0 {
		return nil, errInvalidArguments()
	}
	if _, err := requireApp(call.Context, call.Tx, tenant, in.AppID); err != nil {
		return nil, err
	}
	parsed, errs := spec.Validate(in.Spec)
	if len(errs) != 0 {
		return nil, fmt.Errorf("studio: spec invalid")
	}
	if parsed.App.Owner != tenant {
		return nil, fmt.Errorf("studio: spec owner does not match calling tenant")
	}
	version, hash, err := SaveSpecVersion(call.Context, call.Tx, tenant, in.AppID, in.Spec)
	if err != nil {
		return nil, err
	}
	return marshalMap(map[string]any{"version": version, "hash": hash})
}

// SaveSpecVersion persists one validated spec document as the next
// version for its app and returns (version, hash). Shared by the
// spec.save tool and artifact approval (single-canonical-spec rule:
// approval cites the version it saves, atomically, in one tx).
func SaveSpecVersion(ctx context.Context, tx pgx.Tx, tenant, appID string, doc json.RawMessage) (int, string, error) {
	hash, err := spec.ContentHashOf(doc)
	if err != nil {
		return 0, "", fmt.Errorf("studio: hash: %w", err)
	}
	// Serialize versions per app: lock the parent row, then max+1. The
	// unique constraint remains the backstop against races.
	if _, err := tx.Exec(ctx,
		`SELECT id FROM studio.apps WHERE id = $1 AND tenant_id = $2 FOR UPDATE`,
		appID, tenant); err != nil {
		return 0, "", dbErr(err, "")
	}
	version, _, err := currentVersion(ctx, tx, tenant, appID)
	if err != nil {
		return 0, "", dbErr(err, "")
	}
	version++
	id, err := randomID()
	if err != nil {
		return 0, "", err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO studio.app_specs (id, tenant_id, app_id, version, spec_hash, document)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, tenant, appID, version, hash, doc); err != nil {
		return 0, "", dbErr(err, "version conflict, retry")
	}
	return version, hash, nil
}

// FindSpecVersion locates a saved spec version by app and content
// hash. Approval cites the hit instead of saving a duplicate.
func FindSpecVersion(ctx context.Context, tx pgx.Tx, tenant, appID, hash string) (int, bool, error) {
	var version int
	err := tx.QueryRow(ctx,
		`SELECT version FROM studio.app_specs WHERE tenant_id = $1 AND app_id = $2 AND spec_hash = $3`,
		tenant, appID, hash).Scan(&version)
	if err == pgx.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, dbErr(err, "")
	}
	return version, true, nil
}

type specGetIn struct {
	AppID   string `json:"app_id"`
	Version int    `json:"version"`
}

func (d Deps) specGet(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in specGetIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.AppID == "" || in.Version < 0 {
		return nil, errInvalidArguments()
	}
	if _, err := requireApp(call.Context, call.Tx, tenant, in.AppID); err != nil {
		return nil, err
	}
	var version int
	var hash string
	var document json.RawMessage
	if in.Version == 0 {
		err = call.Tx.QueryRow(call.Context,
			`SELECT version, spec_hash, document FROM studio.app_specs
			 WHERE tenant_id = $1 AND app_id = $2 ORDER BY version DESC LIMIT 1`,
			tenant, in.AppID).Scan(&version, &hash, &document)
	} else {
		err = call.Tx.QueryRow(call.Context,
			`SELECT version, spec_hash, document FROM studio.app_specs
			 WHERE tenant_id = $1 AND app_id = $2 AND version = $3`,
			tenant, in.AppID, in.Version).Scan(&version, &hash, &document)
	}
	if err != nil {
		return nil, notFoundOrDbErr(err)
	}
	return marshalMap(map[string]any{
		"version": version, "hash": hash, "document": document,
	})
}
