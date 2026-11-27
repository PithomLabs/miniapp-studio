// Package invite implements the tenant-invitation lifecycle on the
// existing flat identity model: operators are (tenant, binding)
// pairs with no role hierarchy, so any session-authenticated
// operator of a tenant may invite, revoke, and resend. Every entry
// point takes the caller-authorized tenant explicitly: server
// handlers must pass the session-derived tenant, never a
// client-supplied value; dev verbs satisfy this via owner-terminal
// trust with an explicit --tenant flag.
//
// Invitation tokens reuse the session-token machinery
// (store.MintToken/HashToken): 256-bit opaque, hash-only at rest,
// uniform failure responses so no accept outcome oracles
// invitation state across tenants.
package invite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/store"
)

// DefaultTTL bounds an invitation's life; MaxTTL caps explicit
// overrides below session_create's 30-day guard philosophy.
const (
	DefaultTTL = 7 * 24 * time.Hour
	MaxTTL     = 30 * 24 * time.Hour
)

// ErrInviteInvalid is the single uniform accept failure: bad,
// expired, revoked, consumed, or foreign tokens are
// indistinguishable to the caller.
var ErrInviteInvalid = errors.New("invite: invalid or expired invitation")

// Invitation is stored-invite metadata. The raw token never appears
// here — it exists only in Create's return value.
type Invitation struct {
	ID        string
	Tenant    string
	Binding   string
	InvitedBy string
	ExpiresAt time.Time
	CreatedAt time.Time
	Consumed  bool
	Revoked   bool
}

// Service issues invitation calls against an application-role pool.
type Service struct {
	db *pgxpool.Pool
}

// New wires the service to its pool.
func New(db *pgxpool.Pool) (*Service, error) {
	if db == nil {
		return nil, fmt.Errorf("invite: pool is required")
	}
	return &Service{db: db}, nil
}

// Create records an invitation for binding in tenant, invited by the
// caller-authorized inviter binding. The tenant is stamped from the
// authorized parameter — never from client input — and RLS confines
// the row. Returns the raw token exactly once; callers must deliver
// it out-of-band and never log it.
func (s *Service) Create(ctx context.Context, tenant, binding, invitedBy string, ttl time.Duration) (inviteID, rawToken string, err error) {
	if tenant == "" || binding == "" || invitedBy == "" {
		return "", "", fmt.Errorf("invite: create requires tenant, binding, and inviter")
	}
	if strings.ContainsAny(tenant+binding+invitedBy, "\x00") {
		return "", "", fmt.Errorf("invite: identifiers must not contain NUL")
	}
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if ttl > MaxTTL {
		return "", "", fmt.Errorf("invite: ttl exceeds %s", MaxTTL)
	}
	token, err := store.MintToken()
	if err != nil {
		return "", "", err
	}
	id, err := store.RandomHex(16)
	if err != nil {
		return "", "", err
	}
	err = store.InTenantTx(ctx, s.db, tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
INSERT INTO studio.invitations (tenant_id, id, token_hash, binding_id, invited_by, expires_at)
VALUES ($1, $2, $3, $4, $5, now() + make_interval(secs => $6))`,
			tenant, id, store.HashToken(token), binding, invitedBy, ttl.Seconds())
		return err
	})
	if err != nil {
		return "", "", fmt.Errorf("invite: create for tenant %s: %w", tenant, err)
	}
	return id, token, nil
}

// Accept consumes a raw invitation token and mints the invitee's
// session in ONE transaction: invitation_accept (guard → operator
// upsert → consume) then session_create for the returned tenant and
// binding. Any failure rolls everything back, so a failed accept
// leaves the invitation consumable and the caller can retry safely —
// acceptance can never strand a consumed invite without a session.
// All failures collapse to ErrInviteInvalid (uniform response).
// The session token returns alongside the session: the invitee
// presents it exactly like a login-issued token.
func (s *Service) Accept(ctx context.Context, rawToken string, now time.Time) (store.Session, string, error) {
	return s.acceptWithExpiry(ctx, rawToken, now.Add(store.SessionTTL))
}

func (s *Service) acceptWithExpiry(ctx context.Context, rawToken string, sessionExpiry time.Time) (store.Session, string, error) {
	fail := func() (store.Session, string, error) { return store.Session{}, "", ErrInviteInvalid }
	if strings.TrimSpace(rawToken) == "" {
		return fail()
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fail()
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// No tenant context: the token is the capability, and the
	// SECURITY DEFINER function resolves (tenant, binding) from it.
	var tenant, binding string
	err = tx.QueryRow(ctx, `SELECT o_tenant_id, o_binding_id FROM studio.invitation_accept($1)`,
		store.HashToken(rawToken)).Scan(&tenant, &binding)
	if err != nil {
		return fail()
	}
	token, err := store.MintToken()
	if err != nil {
		return fail()
	}
	id, err := store.RandomHex(16)
	if err != nil {
		return fail()
	}
	var created sql.NullString
	err = tx.QueryRow(ctx, `SELECT studio.session_create($1, $2, $3, $4, $5)`,
		id, store.HashToken(token), tenant, binding, sessionExpiry).Scan(&created)
	if err != nil || !created.Valid {
		return fail()
	}
	if err := tx.Commit(ctx); err != nil {
		return fail()
	}
	return store.Session{ID: id, TenantID: tenant, BindingID: binding, ExpiresAt: sessionExpiry}, token, nil
}

// Revoke marks an invitation revoked: the token dies even if
// unexpired and unconsumed. Revoking an unknown, consumed, or
// foreign-tenant id collapses to ErrInviteInvalid without
// distinguishing them (no enumeration oracle for tenant members).
func (s *Service) Revoke(ctx context.Context, tenant, inviteID string) error {
	if tenant == "" || inviteID == "" {
		return fmt.Errorf("invite: revoke requires tenant and id")
	}
	var revoked bool
	err := store.InTenantTx(ctx, s.db, tenant, func(tx pgx.Tx) error {
		cmd, err := tx.Exec(ctx, `
UPDATE studio.invitations SET revoked_at = now()
WHERE tenant_id = $1 AND id = $2 AND revoked_at IS NULL AND consumed_at IS NULL`,
			tenant, inviteID)
		if err != nil {
			return err
		}
		revoked = cmd.RowsAffected() == 1
		return nil
	})
	if err != nil {
		return fmt.Errorf("invite: revoke: %w", err)
	}
	if !revoked {
		return fmt.Errorf("invite: revoke: %w", ErrInviteInvalid)
	}
	return nil
}

// Resend rotates an invitation's token in ONE transaction: the old
// hash is revoked and a fresh invite issued for the same (tenant,
// binding). The old token dies immediately, even mid-delivery —
// rotation, not duplication — and a crash mid-resend leaves the
// original invite untouched rather than revoked-without-replacement.
func (s *Service) Resend(ctx context.Context, tenant, inviteID string, ttl time.Duration) (newID, rawToken string, err error) {
	if tenant == "" || inviteID == "" {
		return "", "", fmt.Errorf("invite: resend requires tenant and id")
	}
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if ttl > MaxTTL {
		return "", "", fmt.Errorf("invite: ttl exceeds %s", MaxTTL)
	}
	token, err := store.MintToken()
	if err != nil {
		return "", "", err
	}
	id, err := store.RandomHex(16)
	if err != nil {
		return "", "", err
	}
	err = store.InTenantTx(ctx, s.db, tenant, func(tx pgx.Tx) error {
		var binding, invitedBy string
		cmd, err := tx.Exec(ctx, `
UPDATE studio.invitations SET revoked_at = now()
WHERE tenant_id = $1 AND id = $2 AND revoked_at IS NULL AND consumed_at IS NULL`,
			tenant, inviteID)
		if err != nil {
			return err
		}
		if cmd.RowsAffected() != 1 {
			return ErrInviteInvalid
		}
		if err := tx.QueryRow(ctx, `
SELECT binding_id, invited_by FROM studio.invitations
WHERE tenant_id = $1 AND id = $2`, tenant, inviteID).Scan(&binding, &invitedBy); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
INSERT INTO studio.invitations (tenant_id, id, token_hash, binding_id, invited_by, expires_at)
VALUES ($1, $2, $3, $4, $5, now() + make_interval(secs => $6))`,
			tenant, id, store.HashToken(token), binding, invitedBy, ttl.Seconds())
		return err
	})
	if err != nil {
		return "", "", err
	}
	return id, token, nil
}

// List returns invitation metadata for a tenant, newest first. Rows
// hold hashes only, so listing is safe; expired/consumed/revoked
// rows are included with their terminal state (lifecycle honesty).
func (s *Service) List(ctx context.Context, tenant string) ([]Invitation, error) {
	if tenant == "" {
		return nil, fmt.Errorf("invite: list requires tenant")
	}
	var out []Invitation
	err := store.InTenantTx(ctx, s.db, tenant, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT id, tenant_id, binding_id, invited_by, expires_at, created_at,
       consumed_at IS NOT NULL, revoked_at IS NOT NULL
FROM studio.invitations WHERE tenant_id = $1 ORDER BY created_at DESC`, tenant)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var in Invitation
			if err := rows.Scan(&in.ID, &in.Tenant, &in.Binding, &in.InvitedBy,
				&in.ExpiresAt, &in.CreatedAt, &in.Consumed, &in.Revoked); err != nil {
				return err
			}
			out = append(out, in)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("invite: list: %w", err)
	}
	return out, nil
}
