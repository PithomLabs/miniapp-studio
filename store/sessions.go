package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Session is one authenticated workspace session, resolved from an opaque
// token. TenantID and BindingID come from the database, never the browser.
type Session struct {
	ID        string
	TenantID  string
	BindingID string
	ExpiresAt time.Time
}

// SessionTTL bounds one login; absolute, non-renewable in the MVP.
const SessionTTL = 12 * time.Hour

// ErrSessionRefused reports that session_create's guards declined: the
// operator row is missing or deactivated. Callers map it to 401, never 500,
// because it means "not provisioned", not "database broken".
var ErrSessionRefused = errors.New("store: session refused (operator not provisioned?)")

// MintToken generates one 256-bit opaque session token.
func MintToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("store: mint token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

// HashToken binds a token to its stored form. Only hashes cross the
// application boundary; plaintext tokens exist in memory and cookies only.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RandomHex returns n random bytes as hex, for IDs and test credentials.
func RandomHex(n int) (string, error) {
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("store: random: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

// Store issues narrow session calls against an application-role pool.
type Store struct {
	pool *pgxpool.Pool
}

// New returns a Store over an application-role pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// ErrNoSession reports an unknown, expired, or revoked token. Callers map it
// to unauthenticated without distinguishing the cause.
var ErrNoSession = errors.New("store: no such session")

// Lookup resolves one token hash through the SECURITY DEFINER function. It
// returns ErrNoSession for forged, expired, or revoked tokens.
func (s *Store) Lookup(ctx context.Context, tokenHash string) (Session, error) {
	var out Session
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, binding_id, expires_at FROM studio.session_lookup($1)`, tokenHash,
	).Scan(&out.ID, &out.TenantID, &out.BindingID, &out.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNoSession
	}
	if err != nil {
		return Session{}, fmt.Errorf("store: lookup session: %w", err)
	}
	return out, nil
}

// Create mints one session row for a server-authenticated (tenant, binding)
// pair and returns the plaintext token exactly once, for the login cookie.
// now anchors expiry math. It fails when the operator pair is not
// provisioned or active.
func (s *Store) Create(ctx context.Context, tenantID, bindingID string, now time.Time) (string, Session, error) {
	token, err := MintToken()
	if err != nil {
		return "", Session{}, err
	}
	id, err := RandomHex(16)
	if err != nil {
		return "", Session{}, err
	}
	var created sql.NullString
	err = s.pool.QueryRow(ctx,
		`SELECT studio.session_create($1, $2, $3, $4, $5)`,
		id, HashToken(token), tenantID, bindingID, now.Add(SessionTTL),
	).Scan(&created)
	// The function returns NULL (never zero rows) when its guards refuse.
	if err == nil && !created.Valid {
		return "", Session{}, ErrSessionRefused
	}
	if err != nil {
		return "", Session{}, fmt.Errorf("store: create session: %w", err)
	}
	return token, Session{ID: id, TenantID: tenantID, BindingID: bindingID, ExpiresAt: now.Add(SessionTTL)}, nil
}

// CreateWithToken is Create for a caller-supplied token. Tests use it to
// build deterministic fixtures; production code must prefer Create.
// CreateWithToken inserts one session with a caller-chosen token. Test-only:
// production issuance flows through Create with a random token. Callers must
// never persist a caller-chosen token hash outside tests.
func (s *Store) CreateWithToken(ctx context.Context, token, tenantID, bindingID string, expires time.Time) (Session, error) {
	id, err := RandomHex(16)
	if err != nil {
		return Session{}, err
	}
	var created sql.NullString
	err = s.pool.QueryRow(ctx,
		`SELECT studio.session_create($1, $2, $3, $4, $5)`,
		id, HashToken(token), tenantID, bindingID, expires,
	).Scan(&created)
	// The function returns NULL (never zero rows) when its guards refuse.
	if err == nil && !created.Valid {
		return Session{}, ErrSessionRefused
	}
	if err != nil {
		return Session{}, fmt.Errorf("store: create session: %w", err)
	}
	return Session{ID: id, TenantID: tenantID, BindingID: bindingID, ExpiresAt: expires}, nil
}

// Revoke marks one token revoked. Revoking an unknown token succeeds
// silently: revocation is idempotent by design.
func (s *Store) Revoke(ctx context.Context, tokenHash string) error {
	var revoked sql.NullBool
	err := s.pool.QueryRow(ctx,
		`SELECT studio.session_revoke($1)`, tokenHash,
	).Scan(&revoked)
	// NULL means no live row matched: revocation is idempotent by design.
	if err == nil && !revoked.Valid {
		return nil
	}
	if err != nil {
		return fmt.Errorf("store: revoke session: %w", err)
	}
	return nil
}
