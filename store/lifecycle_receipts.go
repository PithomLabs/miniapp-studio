package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// LifecycleReceipt is the durable form of one completed sprite lifecycle
// transition (migration 000025). Plain data with the fly contract's
// vocabulary; the mapping lives at the call site. Latency keeps
// millisecond precision — wake-latency notes operate at 100ms scale.
type LifecycleReceipt struct {
	Key        string
	Sprite     string
	Transition string
	From       string
	To         string
	At         time.Time
	Approver   string
	AuthRef    string
	Provider   string
	Latency    time.Duration
}

// ErrReceiptExists reports a duplicate idempotency key: the receipted
// transition already ran. Cross-restart replays read the stored row
// instead of re-executing.
var ErrReceiptExists = errors.New("store: lifecycle receipt already exists")

// ErrNoReceipt reports an unknown idempotency key or a sprite with no
// recorded transitions.
var ErrNoReceipt = errors.New("store: no lifecycle receipt")

// SaveLifecycleReceipt persists one completed-transition receipt. A key
// collision returns ErrReceiptExists; CHECK violations (unknown
// transition/state vocabulary) propagate as errors, never silent writes.
func (s *Store) SaveLifecycleReceipt(ctx context.Context, tenantID string, r LifecycleReceipt) error {
	return InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO studio.sprite_lifecycle_receipts
			 (id, tenant_id, sprite, transition, from_state, to_state, at, approver, auth_ref, provider, latency_ms)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			r.Key, tenantID, r.Sprite, r.Transition, r.From, r.To,
			r.At.UTC(), r.Approver, r.AuthRef, r.Provider, r.Latency.Milliseconds())
		if err == nil {
			return nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrReceiptExists
		}
		return fmt.Errorf("store: save lifecycle receipt: %w", err)
	})
}

// LifecycleReceiptByKey returns the receipt recorded under one key.
func (s *Store) LifecycleReceiptByKey(ctx context.Context, tenantID, key string) (LifecycleReceipt, error) {
	var out LifecycleReceipt
	err := InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var latencyMs int64
		err := tx.QueryRow(ctx,
			`SELECT id, sprite, transition, from_state, to_state, at, approver, auth_ref, provider, latency_ms
			 FROM studio.sprite_lifecycle_receipts WHERE tenant_id = $1 AND id = $2`,
			tenantID, key).Scan(&out.Key, &out.Sprite, &out.Transition, &out.From, &out.To,
			&out.At, &out.Approver, &out.AuthRef, &out.Provider, &latencyMs)
		if err == nil {
			// Canonicalize to UTC: the database preserves the
			// instant, not the zone label (pgx returns the
			// session zone, e.g. +0800 here), and the save path
			// already normalizes with .UTC(). Callers compare
			// instants, never zone spellings.
			out.At = out.At.UTC()
			out.Latency = time.Duration(latencyMs) * time.Millisecond
			return nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNoReceipt
		}
		return fmt.Errorf("store: lifecycle receipt by key: %w", err)
	})
	return out, err
}

// LatestLifecyclePosition returns the latest recorded To state for one
// sprite: the restart-recovery position. No rows means absent.
func (s *Store) LatestLifecyclePosition(ctx context.Context, tenantID, sprite string) (to string, found bool, err error) {
	err = InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		qerr := tx.QueryRow(ctx,
			`SELECT to_state FROM studio.sprite_lifecycle_receipts
			 WHERE tenant_id = $1 AND sprite = $2 ORDER BY at DESC LIMIT 1`,
			tenantID, sprite).Scan(&to)
		if qerr == nil {
			found = true
			return nil
		}
		if errors.Is(qerr, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("store: latest lifecycle position: %w", qerr)
	})
	return to, found, err
}
