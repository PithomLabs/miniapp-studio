package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/PithomLabs/miniapp/registry"
	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/apps"
	"github.com/PithomLabs/miniapp-studio/fly"
)

type deployRefIn struct {
	AppID string `json:"app_id"`
}

type deployView struct {
	AppID        string `json:"app_id"`
	Slug         string `json:"slug"`
	ResourceKind string `json:"resource_kind"`
	ResourceName string `json:"resource_name"`
	ResourceID   string `json:"resource_id"`
	URL          string `json:"url"`
	Status       string `json:"status"`
	SpecHash     string `json:"spec_hash"`
	Error        string `json:"error"`
	Live         string `json:"live,omitempty"`
	updatedAt    time.Time
}

// StuckThreshold bounds how long a provisioning row may sit untouched
// before a new provision attempt retries instead of attaching. No live
// worker takes this long; anything older belongs to a dead one.
const StuckThreshold = 10 * time.Minute

// Stale reports whether a provisioning record outlived any live worker.
func (v deployView) Stale() bool {
	return v.Status == string(apps.StatusProvisioning) &&
		!v.updatedAt.IsZero() && time.Since(v.updatedAt) > StuckThreshold
}

// deployProvision provisions the Fly target for one app and records the
// authoritative URL. It attaches idempotently to an active deployment,
// refuses when Fly is unconfigured (before writing anything), and records
// failures with evidence instead of failing silently.
func (d Deps) deployProvision(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in deployRefIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.AppID == "" {
		return nil, errInvalidArguments()
	}
	app, err := requireApp(ctx, call.Tx, tenant, in.AppID)
	if err != nil {
		return nil, err
	}
	_, hash, err := currentVersion(ctx, call.Tx, tenant, in.AppID)
	if err != nil {
		return nil, dbErr(err, "")
	}
	if hash == "" {
		return nil, fmt.Errorf("studio: no spec versions: save a spec first")
	}
	if existing, ok, err := loadDeployment(ctx, call.Tx, tenant, in.AppID); err != nil {
		return nil, dbErr(err, "")
	} else if ok && existing.Status == string(apps.StatusProvisioning) && existing.Stale() {
		// A provisioning row older than the stuck threshold belongs to a
		// dead worker: fall through and retry instead of attaching forever.
	} else if ok && (existing.Status == string(apps.StatusProvisioning) ||
		existing.Status == string(apps.StatusReady) ||
		existing.Status == string(apps.StatusStopped)) {
		// The attach path must not bless a dead record: when a manager is
		// configured, the sprite behind the record has to exist. A missing
		// sprite is an honest failure naming it — the operator retires the
		// deployment explicitly rather than the tool silently re-provisioning.
		if d.Sprites != nil {
			if _, err := d.Sprites.GetSprite(ctx, existing.ResourceName); err != nil {
				var missing *fly.NotFoundError
				if errors.As(err, &missing) {
					return nil, fmt.Errorf("studio: sprite %q no longer exists: retire the deployment before re-provisioning", existing.ResourceName)
				}
				return nil, fmt.Errorf("studio: verify sprite %q: %w", existing.ResourceName, err)
			}
		}
		return marshalDeployment(existing)
	}
	if d.Sprites == nil {
		return nil, fmt.Errorf("studio: deployment unavailable: Fly not configured")
	}
	name, err := reserveName(ctx, d.Sprites, app.Slug)
	if err != nil {
		// reserveName returns static errors only; still, persist a fixed
		// reason so future manager error text can never reach the record.
		return d.failProvision(ctx, call, tenant, in.AppID, app.Slug, hash, "provisioning failed")
	}
	created, err := d.Sprites.CreateSprite(ctx, name, fly.SpriteOptions{Labels: []string{"studio-app"}})
	if err != nil {
		// A lost create/get race surfaces here: re-check before failing.
		if _, rerr := d.Sprites.GetSprite(ctx, name); rerr == nil {
			return nil, fmt.Errorf("studio: name race, retry")
		}
		return d.failProvision(ctx, call, tenant, in.AppID, app.Slug, hash, "provisioning failed")
	}
	_ = created
	// The URL comes only from the authoritative read, never the create echo.
	live, err := d.Sprites.GetSprite(ctx, name)
	if err != nil {
		return d.failProvision(ctx, call, tenant, in.AppID, app.Slug, hash, "provisioning failed")
	}
	if live.URL == "" {
		tryDelete(ctx, d.Sprites, name)
		return d.failProvision(ctx, call, tenant, in.AppID, app.Slug, hash, "no URL issued")
	}
	rec := deployView{
		AppID: app.ID, Slug: app.Slug,
		ResourceKind: string(apps.ResourceSprite), ResourceName: name,
		ResourceID: live.ID, URL: live.URL,
		Status: string(apps.StatusReady), SpecHash: hash,
	}
	if err := upsertDeployment(ctx, call, tenant, rec, ""); err != nil {
		// Persisted nothing: remove the orphaned sprite best-effort.
		tryDelete(ctx, d.Sprites, name)
		return nil, dbErr(err, "")
	}
	return marshalDeployment(rec)
}

// reserveName returns the first candidate name with no live sprite behind
// it. Existence checks distinguish not-found from genuine failures: any
// ambiguous error fails closed instead of guessing.
func reserveName(ctx context.Context, mgr fly.SpriteManager, slug string) (string, error) {
	cands, err := apps.CandidateNames(slug)
	if err != nil {
		return "", fmt.Errorf("studio: invalid slug")
	}
	for _, name := range cands {
		_, err := mgr.GetSprite(ctx, name)
		if err == nil {
			continue
		}
		var missing *fly.NotFoundError
		if errors.As(err, &missing) {
			return name, nil
		}
		return "", fmt.Errorf("studio: provisioning failed")
	}
	return "", fmt.Errorf("studio: no free resource name")
}

func tryDelete(ctx context.Context, mgr fly.SpriteManager, name string) {
	_ = mgr.DeleteSprite(ctx, name)
}

// failProvision records a failed deployment and returns it as a successful
// result. Business failure is evidence, not a handler error: returning an
// error would roll the record back, while a result persists the outcome.
// Terminal errors stay reserved for invalid input, denial, and transient
// unavailability (all retryable under a fresh key).
func (d Deps) failProvision(ctx context.Context, call registry.ToolCall, tenant, appID, slug, hash, reason string) (json.RawMessage, error) {
	id, err := randomID()
	if err != nil {
		return nil, fmt.Errorf("studio: unavailable")
	}
	if _, err := call.Tx.Exec(ctx,
		`INSERT INTO studio.deployments (id, tenant_id, app_id, slug, resource_kind, resource_name, status, spec_hash, error)
		 VALUES ($1, $2, $3, $4, 'sprite', '', 'failed', $5, $6)
		 ON CONFLICT (tenant_id, app_id) DO UPDATE SET
		   status = 'failed', error = $6, spec_hash = $5, updated_at = now()`,
		id, tenant, appID, slug, hash, reason); err != nil {
		return nil, dbErr(err, "")
	}
	return marshalDeployment(deployView{
		AppID: appID, Slug: slug, ResourceKind: string(apps.ResourceSprite),
		Status: string(apps.StatusFailed), SpecHash: hash, Error: reason,
	})
}

func upsertDeployment(ctx context.Context, call registry.ToolCall, tenant string, rec deployView, failure string) error {
	id, err := randomID()
	if err != nil {
		return err
	}
	_, err = call.Tx.Exec(ctx,
		`INSERT INTO studio.deployments (id, tenant_id, app_id, slug, resource_kind, resource_name, resource_id, url, status, spec_hash, error)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (tenant_id, app_id) DO UPDATE SET
		   resource_kind = EXCLUDED.resource_kind, resource_name = EXCLUDED.resource_name,
		   resource_id = EXCLUDED.resource_id, url = EXCLUDED.url, status = EXCLUDED.status,
		   spec_hash = EXCLUDED.spec_hash, error = EXCLUDED.error, updated_at = now()`,
		id, tenant, rec.AppID, rec.Slug, rec.ResourceKind, rec.ResourceName,
		rec.ResourceID, rec.URL, rec.Status, rec.SpecHash, failure)
	return err
}

func loadDeployment(ctx context.Context, tx pgx.Tx, tenant, appID string) (deployView, bool, error) {
	var rec deployView
	var slug, kind, name, id, url, status, hash, failure string
	var updated time.Time
	err := tx.QueryRow(ctx,
		`SELECT slug, resource_kind, resource_name, resource_id, url, status, spec_hash, error, updated_at
		 FROM studio.deployments WHERE tenant_id = $1 AND app_id = $2`,
		tenant, appID).Scan(&slug, &kind, &name, &id, &url, &status, &hash, &failure, &updated)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return deployView{}, false, nil
		}
		return deployView{}, false, err
	}
	rec.AppID, rec.Slug, rec.ResourceKind, rec.ResourceName = appID, slug, kind, name
	rec.ResourceID, rec.URL, rec.Status, rec.SpecHash, rec.Error = id, url, status, hash, failure
	rec.updatedAt = updated
	return rec, true, nil
}

func marshalDeployment(rec deployView) (json.RawMessage, error) {
	raw, err := json.Marshal(rec)
	if err != nil {
		return nil, fmt.Errorf("studio: encode: %w", err)
	}
	return raw, nil
}

func (d Deps) deployStatus(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in deployRefIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.AppID == "" {
		return nil, errInvalidArguments()
	}
	if _, err := requireApp(call.Context, call.Tx, tenant, in.AppID); err != nil {
		return nil, err
	}
	rec, ok, err := loadDeployment(call.Context, call.Tx, tenant, in.AppID)
	if err != nil {
		return nil, dbErr(err, "")
	}
	if !ok {
		return marshalMap(map[string]any{"status": "none"})
	}
	if d.Sprites == nil {
		rec.Live = "unconfigured"
		return marshalDeployment(rec)
	}
	live, err := d.Sprites.GetSprite(call.Context, rec.ResourceName)
	if err != nil {
		var missing *fly.NotFoundError
		if errors.As(err, &missing) {
			// Definitive absence reconciles the record; anything else
			// leaves it untouched and fails the call instead.
			if _, uerr := call.Tx.Exec(call.Context,
				`UPDATE studio.deployments SET status = 'failed', error = 'target missing', updated_at = now()
				 WHERE tenant_id = $1 AND app_id = $2`, tenant, in.AppID); uerr != nil {
				return nil, dbErr(uerr, "")
			}
			rec.Status, rec.Error = "failed", "target missing"
			return marshalDeployment(rec)
		}
		return nil, fmt.Errorf("studio: status unavailable")
	}
	rec.Live = live.Status
	return marshalDeployment(rec)
}

func (d Deps) deployRetire(_ context.Context, call registry.ToolCall) (json.RawMessage, error) {
	tenant, err := tenantOf(call.TenantID)
	if err != nil {
		return nil, err
	}
	var in deployRefIn
	if err := decode(call.Input, &in); err != nil {
		return nil, err
	}
	if in.AppID == "" {
		return nil, errInvalidArguments()
	}
	if _, err := requireApp(call.Context, call.Tx, tenant, in.AppID); err != nil {
		return nil, err
	}
	rec, ok, err := loadDeployment(call.Context, call.Tx, tenant, in.AppID)
	if err != nil {
		return nil, dbErr(err, "")
	}
	if !ok {
		return nil, errNotFound()
	}
	if d.Sprites == nil {
		return nil, fmt.Errorf("studio: deployment unavailable: Fly not configured")
	}
	if rec.ResourceName != "" {
		if err := d.Sprites.DeleteSprite(call.Context, rec.ResourceName); err != nil {
			var missing *fly.NotFoundError
			if !errors.As(err, &missing) {
				return nil, fmt.Errorf("studio: retire failed")
			}
		}
	}
	if _, err := call.Tx.Exec(call.Context,
		`UPDATE studio.deployments SET status = 'deleted', updated_at = now()
		 WHERE tenant_id = $1 AND app_id = $2`, tenant, in.AppID); err != nil {
		return nil, dbErr(err, "")
	}
	rec.Status = "deleted"
	return marshalDeployment(rec)
}
