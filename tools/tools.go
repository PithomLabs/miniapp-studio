package tools

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/PithomLabs/miniapp-studio/fly"
)

// Miniapp identity for the Studio control plane.
const (
	MiniappID = "studio"
	Version   = "1.0.0"
	// Build identifies this development implementation. Production
	// releases stamp the OCI artifact digest here instead.
	Build = "studio-dev"
)

// Deps wires external managers into tools that need them. Nil managers mean
// production-unconfigured: affected tools refuse honestly instead of
// fabricating results. Tests inject fakes; no live mutations occur in P4.
type Deps struct {
	Sprites    fly.SpriteManager
	Connectors fly.ConnectorManager
	// GatewayBinary holds the gateway server bytes deployed to Builder
	// Sprites by build.provision. Empty means unconfigured (honest
	// refusal); production wiring (binary path config) arrives with the
	// Phase B live segment per the runbook.
	GatewayBinary []byte
	// GatewayOrgSecret derives per-build gateway HMAC keys. Empty means
	// unconfigured (honest refusal).
	GatewayOrgSecret []byte
}

// Tool names.
const (
	ToolSpecValidate       = "spec.validate"
	ToolAppCreate          = "app.create"
	ToolAppList            = "app.list"
	ToolAppGet             = "app.get"
	ToolAppUpdate          = "app.update"
	ToolSpecSave           = "spec.save"
	ToolSpecGet            = "spec.get"
	ToolDeployProvision    = "deploy.provision"
	ToolDeployStatus       = "deploy.status"
	ToolDeployRetire       = "deploy.retire"
	ToolConnectorsList     = "connectors.list"
	ToolConnectorsAttest   = "connectors.custom_attest"
	ToolActivityList       = "activity.list"
	ToolConversationCreate = "conversation.create"
	ToolConversationGet    = "conversation.get"
	ToolConversationList   = "conversation.list"
	ToolConversationStream = "conversation.stream"
	ToolMessageSend        = "message.send"
	ToolBuildCreate        = "build.create"
	ToolBuildProvision     = "build.provision"
	ToolBuildUpgrade       = "build.upgrade"
	ToolBuildStatus        = "build.status"
	ToolBuildCancel        = "build.cancel"
	ToolBuildIngestEvent   = "build.ingest_event"
	ToolQuestionAnswer     = "question.answer"
)

// decode parses tool input strictly: unknown fields and trailing data are
// rejected, and failures carry no input excerpts (inputs may be large, and
// error text persists in invocation records).
func decode(input json.RawMessage, out any) error {
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return errInvalidArguments()
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return errInvalidArguments()
	}
	return nil
}

func errInvalidArguments() error {
	return fmt.Errorf("studio: invalid arguments")
}

func errTenantRequired() error {
	return fmt.Errorf("studio: tenant is required")
}

func errNotFound() error {
	return fmt.Errorf("studio: not found")
}

// notFoundOrDbErr maps a query error honestly: no rows means not found,
// while anything else (deadlocks, connection faults, permission errors)
// is a database failure. Collapsing every error to not-found would mask
// exactly the faults operators need to see — including the 40P01 this
// package's concurrency tests exercise.
func notFoundOrDbErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errNotFound()
	}
	return dbErr(err, "")
}

func errDenied() error {
	return fmt.Errorf("studio: denied")
}

// dbErr maps database failures to safe terminal messages by SQLSTATE. Raw
// database errors are never returned: they can echo values, and error text
// persists in invocation and audit records.
func dbErr(err error, conflictMsg string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			if conflictMsg != "" {
				return fmt.Errorf("studio: %s", conflictMsg)
			}
			return fmt.Errorf("studio: already exists")
		case "23503":
			return fmt.Errorf("studio: invalid reference")
		case "42501":
			return fmt.Errorf("studio: denied")
		}
	}
	return fmt.Errorf("studio: unavailable")
}

// tenantOf extracts the server-derived tenant or refuses.
func tenantOf(tenantID string) (string, error) {
	if tenantID == "" {
		return "", errTenantRequired()
	}
	return tenantID, nil
}

// Registry access shared by registry tools.
type appRow struct {
	ID            string
	TenantID      string
	Slug          string
	Name          string
	Description   string
	OwnerIdentity string
	Version       int
	SpecHash      string
}

// requireApp loads one app owned by this tenant. Foreign or missing apps
// produce the identical not-found error: no existence oracle.
func requireApp(ctx context.Context, tx pgx.Tx, tenantID, appID string) (appRow, error) {
	var a appRow
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, slug, name, description, owner_identity
		 FROM studio.apps WHERE id = $1 AND tenant_id = $2`, appID, tenantID,
	).Scan(&a.ID, &a.TenantID, &a.Slug, &a.Name, &a.Description, &a.OwnerIdentity)
	if err != nil {
		return appRow{}, notFoundOrDbErr(err)
	}
	return a, nil
}

// currentVersion returns the latest spec version and hash for one app,
// or zeroes when the app has no versions yet.
func currentVersion(ctx context.Context, tx pgx.Tx, tenantID, appID string) (int, string, error) {
	var version int
	var hash string
	err := tx.QueryRow(ctx,
		`SELECT version, spec_hash FROM studio.app_specs
		 WHERE tenant_id = $1 AND app_id = $2 ORDER BY version DESC LIMIT 1`,
		tenantID, appID).Scan(&version, &hash)
	if err != nil {
		return 0, "", nil
	}
	return version, hash, nil
}

// randomID mints one 128-bit hex identifier.
func randomID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("studio: random: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}

// marshalMap encodes one result object. Values are pre-shaped structs and
// maps; encoding cannot leak secrets because handlers never place any.
func marshalMap(v map[string]any) (json.RawMessage, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("studio: encode: %w", err)
	}
	return raw, nil
}

// scrubbedJSON decodes a JSONB value, scrubs secret-shaped content, and
// returns the scrubbed value for display. Undecodable input passes through
// untouched — scrubbing must never destroy evidence it cannot parse.
func scrubbedJSON(v any) any {
	var raw []byte
	switch t := v.(type) {
	case []byte:
		raw = t
	case string:
		raw = []byte(t)
	default:
		return scrubValue(v)
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return v
	}
	return scrubValue(decoded)
}
