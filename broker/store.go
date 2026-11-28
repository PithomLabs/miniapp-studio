package broker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/store"
)

// MaxLeaseTTL caps execution grants at one hour: leases are short
// by construction, and Mint refuses anything longer.
const MaxLeaseTTL = time.Hour

var rxHex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Lease describes a granted lease. It carries scope and lifecycle
// only — the access token is returned by Mint and never stored.
// CredentialVersion binds the lease to the issuing credential
// generation (F35-05): Consume requires a match.
type Lease struct {
	ID                string
	Tenant            string
	Provider          string
	Capability        string
	IntentHash        string
	SingleUse         bool
	IssuedAt          time.Time
	ExpiresAt         time.Time
	UsedAt            *time.Time
	CredentialVersion int64
}

// Broker mints execution-scoped access tokens. The KEK lives in
// memory only; refresh plaintext is decrypted per mint and never
// retained past the provider exchange.
type Broker struct {
	kek   []byte
	kekID string
	db    *pgxpool.Pool
	ex    Exchanger
}

// New wires the broker to its pool and provider exchanger. A nil KEK
// is allowed for verbs that never touch ciphertext (revoke/inspect);
// Enroll and Mint fail closed without one. A nil exchanger is
// allowed for verbs that never mint; Mint fails closed without one.
func New(kek []byte, db *pgxpool.Pool, ex Exchanger) (*Broker, error) {
	if kek != nil && len(kek) != kekBytes {
		return nil, fmt.Errorf("broker: KEK must be %d bytes", kekBytes)
	}
	if db == nil {
		return nil, fmt.Errorf("broker: pool is required")
	}
	var cp []byte
	var id string
	if kek != nil {
		cp = make([]byte, kekBytes)
		copy(cp, kek)
		id = KeyID(cp)
	}
	return &Broker{kek: cp, kekID: id, db: db, ex: ex}, nil
}

// Enroll seals a tenant/provider refresh token into storage.
// Re-enrollment overwrites, clears revocation, and bumps the
// credential generation (the operator re-consent path); leases
// bound to older generations stay retired. Lease history itself
// is untouched.
func (b *Broker) Enroll(ctx context.Context, tenant, provider string, refresh []byte) error {
	if tenant == "" || provider == "" {
		return fmt.Errorf("broker: enroll requires tenant and provider")
	}
	ct, err := Seal(b.kek, tenant, provider, refresh)
	if err != nil {
		return err
	}
	return store.InTenantTx(ctx, b.db, tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
INSERT INTO studio.broker_credentials (tenant_id, provider, refresh_ct, kek_id, updated_at, revoked_at)
VALUES ($1, $2, $3, $4, now(), NULL)
ON CONFLICT (tenant_id, provider) DO UPDATE
SET refresh_ct = EXCLUDED.refresh_ct, kek_id = EXCLUDED.kek_id,
    updated_at = now(), revoked_at = NULL,
    version = studio.broker_credentials.version + 1`,
			tenant, provider, ct, b.kekID)
		if err != nil {
			return fmt.Errorf("broker: enroll %s/%s: %w", tenant, provider, err)
		}
		return nil
	})
}

// Mint trades the stored refresh token for a short-lived access
// token bound to (tenant, provider, capability, intent). The lease
// row carries scope only; the access token returns in memory and is
// never persisted. Finalization re-checks the credential generation
// under a row lock AFTER the network exchange (F35-05): a revoke or
// re-enrollment that landed mid-exchange fails the mint instead of
// issuing a lease for a dead generation. A provider invalid_grant
// auto-revokes the dead credential (conditional on it being
// unchanged, F35-07) and fails the mint.
func (b *Broker) Mint(ctx context.Context, tenant, provider, capability, intentHash string, ttl time.Duration, singleUse bool) (leaseID, access string, expires time.Time, err error) {
	if tenant == "" || provider == "" || capability == "" {
		return "", "", time.Time{}, fmt.Errorf("broker: mint requires tenant, provider, and capability")
	}
	if !rxHex64.MatchString(intentHash) {
		return "", "", time.Time{}, fmt.Errorf("broker: intent must be a sha256 hex fingerprint")
	}
	if ttl <= 0 || ttl > MaxLeaseTTL {
		return "", "", time.Time{}, fmt.Errorf("broker: ttl must be within (0, %s]", MaxLeaseTTL)
	}
	if b.ex == nil {
		return "", "", time.Time{}, fmt.Errorf("broker: mint requires a provider exchanger")
	}
	cred, err := b.refreshFor(ctx, tenant, provider)
	if err != nil {
		return "", "", time.Time{}, err
	}
	access, providerTTL, err := b.ex.Exchange(ctx, provider, string(cred.plain))
	clear(cred.plain)
	if errors.Is(err, ErrInvalidGrant) {
		revoked, rerr := b.markRevoked(ctx, tenant, provider, cred.ct, cred.version)
		if rerr != nil {
			return "", "", time.Time{}, fmt.Errorf("broker: mint %s/%s: %v (revoke also failed: %v)", tenant, provider, err, rerr)
		}
		if !revoked {
			return "", "", time.Time{}, fmt.Errorf("broker: mint %s/%s: %w (credential changed mid-exchange; not auto-revoked)", tenant, provider, err)
		}
		return "", "", time.Time{}, fmt.Errorf("broker: mint %s/%s: %w (credential auto-revoked)", tenant, provider, err)
	}
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("broker: mint %s/%s: %w", tenant, provider, err)
	}
	if providerTTL < ttl {
		ttl = providerTTL
	}
	leaseID, err = newLeaseID()
	if err != nil {
		return "", "", time.Time{}, err
	}
	err = store.InTenantTx(ctx, b.db, tenant, func(tx pgx.Tx) error {
		var version int64
		var revoked bool
		if err := tx.QueryRow(ctx, `
SELECT version, revoked_at IS NOT NULL FROM studio.broker_credentials
WHERE tenant_id = $1 AND provider = $2 FOR UPDATE`,
			tenant, provider).Scan(&version, &revoked); err != nil {
			return err
		}
		if revoked || version != cred.version {
			return ErrCredentialSuperseded
		}
		return tx.QueryRow(ctx, `
INSERT INTO studio.broker_leases (id, tenant_id, provider, capability, intent_hash, single_use, expires_at, credential_version)
VALUES ($1, $2, $3, $4, $5, $6, now() + make_interval(secs => $7), $8)
RETURNING expires_at`,
			leaseID, tenant, provider, capability, intentHash, singleUse, ttl.Seconds(), cred.version).Scan(&expires)
	})
	if err != nil {
		if errors.Is(err, ErrCredentialSuperseded) {
			return "", "", time.Time{}, fmt.Errorf("broker: mint %s/%s: %w (retry binds the current generation)", tenant, provider, err)
		}
		return "", "", time.Time{}, fmt.Errorf("broker: mint %s/%s: %w", tenant, provider, err)
	}
	return leaseID, access, expires, nil
}

// Consume validates a lease for one use: tenant, capability, and
// intent must match the grant; expiry, single-use, and credential
// generation are enforced atomically. A lease from a revoked or
// re-enrolled generation refuses with ErrLeaseSuperseded even when
// otherwise live. Failures are typed so adapters can react without
// parsing strings.
func (b *Broker) Consume(ctx context.Context, tenant, leaseID, capability, intentHash string) error {
	if tenant == "" || leaseID == "" {
		return fmt.Errorf("broker: consume requires tenant and lease id")
	}
	var consumed bool
	err := store.InTenantTx(ctx, b.db, tenant, func(tx pgx.Tx) error {
		cmd, err := tx.Exec(ctx, `
UPDATE studio.broker_leases AS l
SET used_at = now()
WHERE l.tenant_id = $1 AND l.id = $2 AND l.capability = $3 AND l.intent_hash = $4
  AND l.expires_at > now() AND (NOT l.single_use OR l.used_at IS NULL)
  AND NOT EXISTS (
    SELECT 1 FROM studio.broker_credentials AS c
    WHERE c.tenant_id = l.tenant_id AND c.provider = l.provider
      AND (c.revoked_at IS NOT NULL OR c.version <> l.credential_version))`,
			tenant, leaseID, capability, intentHash)
		if err != nil {
			return err
		}
		consumed = cmd.RowsAffected() == 1
		return nil
	})
	if err != nil {
		return fmt.Errorf("broker: consume lease %s: %w", leaseID, err)
	}
	if consumed {
		return nil
	}
	return b.classifyRefusal(ctx, tenant, leaseID, capability, intentHash)
}

// Revoke marks a credential dead and retires its generation: future
// mints fail, outstanding leases refuse. The credential row is
// locked first so revoke serializes against Mint finalization
// (linearizable revocation, F35-05). Missing credentials are a
// typed NotFound (a typo'd provider must not silently "succeed").
func (b *Broker) Revoke(ctx context.Context, tenant, provider string) error {
	if tenant == "" || provider == "" {
		return fmt.Errorf("broker: revoke requires tenant and provider")
	}
	var found bool
	err := store.InTenantTx(ctx, b.db, tenant, func(tx pgx.Tx) error {
		var revoked bool
		err := tx.QueryRow(ctx, `
SELECT revoked_at IS NOT NULL FROM studio.broker_credentials
WHERE tenant_id = $1 AND provider = $2 FOR UPDATE`,
			tenant, provider).Scan(&revoked)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		found = true
		if revoked {
			return nil // idempotent: already dead
		}
		_, err = tx.Exec(ctx, `
UPDATE studio.broker_credentials
SET revoked_at = now(), version = version + 1
WHERE tenant_id = $1 AND provider = $2`, tenant, provider)
		return err
	})
	if err != nil {
		return fmt.Errorf("broker: revoke %s/%s: %w", tenant, provider, err)
	}
	if !found {
		return fmt.Errorf("broker: revoke %s/%s: %w", tenant, provider, ErrCredentialNotFound)
	}
	return nil
}

// Leases lists lease metadata for a tenant (optionally one
// provider). Rows hold no token material, so listing is safe.
func (b *Broker) Leases(ctx context.Context, tenant, provider string) ([]Lease, error) {
	if tenant == "" {
		return nil, fmt.Errorf("broker: leases requires tenant")
	}
	var out []Lease
	err := store.InTenantTx(ctx, b.db, tenant, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT id, tenant_id, provider, capability, intent_hash, single_use,
       issued_at, expires_at, used_at, credential_version
FROM studio.broker_leases
WHERE tenant_id = $1 AND ($2 = '' OR provider = $2)
ORDER BY issued_at DESC`, tenant, provider)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var l Lease
			if err := rows.Scan(&l.ID, &l.Tenant, &l.Provider, &l.Capability,
				&l.IntentHash, &l.SingleUse, &l.IssuedAt, &l.ExpiresAt, &l.UsedAt,
				&l.CredentialVersion); err != nil {
				return err
			}
			out = append(out, l)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("broker: leases %s: %w", tenant, err)
	}
	return out, nil
}

// credRead is one credential generation: plaintext for the exchange
// plus the ciphertext and version the finalize re-check binds to.
type credRead struct {
	plain   []byte
	ct      []byte
	version int64
}

func (b *Broker) refreshFor(ctx context.Context, tenant, provider string) (credRead, error) {
	var out credRead
	var revoked bool
	err := store.InTenantTx(ctx, b.db, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
SELECT refresh_ct, version, revoked_at IS NOT NULL
FROM studio.broker_credentials
WHERE tenant_id = $1 AND provider = $2`, tenant, provider).Scan(&out.ct, &out.version, &revoked)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return credRead{}, fmt.Errorf("broker: %s/%s: %w", tenant, provider, ErrCredentialNotFound)
		}
		return credRead{}, fmt.Errorf("broker: load %s/%s: %w", tenant, provider, err)
	}
	if revoked {
		return credRead{}, fmt.Errorf("broker: %s/%s: %w", tenant, provider, ErrCredentialRevoked)
	}
	plain, err := Open(b.kek, tenant, provider, out.ct)
	if err != nil {
		return credRead{}, fmt.Errorf("broker: %s/%s: stored credential failed authentication (wrong KEK or tampered row)", tenant, provider)
	}
	out.plain = plain
	return out, nil
}

// markRevoked auto-revokes after a provider invalid_grant, but only
// when the exact generation Mint exchanged is still current (F35-07):
// a concurrent re-enrollment must not be killed by a stale grant
// failure. Reports whether it revoked.
func (b *Broker) markRevoked(ctx context.Context, tenant, provider string, ct []byte, version int64) (bool, error) {
	var revoked bool
	err := store.InTenantTx(ctx, b.db, tenant, func(tx pgx.Tx) error {
		cmd, err := tx.Exec(ctx, `
UPDATE studio.broker_credentials
SET revoked_at = now(), version = version + 1
WHERE tenant_id = $1 AND provider = $2 AND revoked_at IS NULL
  AND version = $3 AND refresh_ct = $4`, tenant, provider, version, ct)
		if err != nil {
			return err
		}
		revoked = cmd.RowsAffected() == 1
		return nil
	})
	return revoked, err
}

// classifyRefusal names WHY Consume's atomic gate refused, so
// adapters get typed errors. The UPDATE already decided; this only
// reads rows to label the outcome.
func (b *Broker) classifyRefusal(ctx context.Context, tenant, leaseID, capability, intentHash string) error {
	var l Lease
	err := store.InTenantTx(ctx, b.db, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
SELECT provider, capability, intent_hash, single_use, expires_at, used_at, credential_version
FROM studio.broker_leases WHERE tenant_id = $1 AND id = $2`,
			tenant, leaseID).Scan(&l.Provider, &l.Capability, &l.IntentHash, &l.SingleUse, &l.ExpiresAt, &l.UsedAt, &l.CredentialVersion)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("broker: consume lease %s: %w", leaseID, ErrLeaseNotFound)
		}
		return fmt.Errorf("broker: consume lease %s: %w", leaseID, err)
	}
	switch {
	case l.Capability != capability || l.IntentHash != intentHash:
		return fmt.Errorf("broker: consume lease %s: %w", leaseID, ErrLeaseMismatch)
	case time.Now().After(l.ExpiresAt):
		return fmt.Errorf("broker: consume lease %s: %w", leaseID, ErrLeaseExpired)
	case l.SingleUse && l.UsedAt != nil:
		return fmt.Errorf("broker: consume lease %s: %w", leaseID, ErrLeaseConsumed)
	}
	// Lease-side checks pass, so the gate refused on credential
	// generation: revoked or re-enrolled under this lease.
	var superseded bool
	err = store.InTenantTx(ctx, b.db, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
SELECT revoked_at IS NOT NULL OR version <> $3
FROM studio.broker_credentials WHERE tenant_id = $1 AND provider = $2`,
			tenant, l.Provider, l.CredentialVersion).Scan(&superseded)
	})
	if err == nil && superseded {
		return fmt.Errorf("broker: consume lease %s: %w", leaseID, ErrLeaseSuperseded)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("broker: consume lease %s: %w", leaseID, err)
	}
	// Credential row gone, or lost a race with a concurrent
	// consumer: report the conservative terminal.
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("broker: consume lease %s: %w", leaseID, ErrLeaseSuperseded)
	}
	return fmt.Errorf("broker: consume lease %s: %w", leaseID, ErrLeaseConsumed)
}

func newLeaseID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("broker: lease id: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}
