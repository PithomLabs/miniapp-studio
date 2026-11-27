package broker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/dbenv"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/testdb"
)

// fakeExchanger is a scripted provider double: it asserts the broker
// hands it the decrypted refresh token and returns canned access
// material. It never mirrors broker internals.
type fakeExchanger struct {
	wantRefresh string
	access      string
	ttl         time.Duration
	err         error
	calls       int
}

func (f *fakeExchanger) Exchange(_ context.Context, _ string, refresh string) (string, time.Duration, error) {
	f.calls++
	if refresh != f.wantRefresh {
		return "", 0, errors.New("fake: unexpected refresh token handed to provider")
	}
	return f.access, f.ttl, f.err
}

func testBroker(t *testing.T, ex Exchanger) (*Broker, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	dsn := testdb.RunDSN(t)
	owner, err := store.NewPool(ctx, dsn, 0)
	if err != nil {
		t.Fatalf("owner pool: %v", err)
	}
	t.Cleanup(owner.Close)
	if err := store.Migrate(ctx, dsn, testdb.Policy(t)); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := dbenv.GrantTestPrivileges(ctx, owner); err != nil {
		t.Fatalf("GrantTestPrivileges: %v", err)
	}
	app, err := store.NewPool(ctx, testdb.AppDSN(t, dsn), 0)
	if err != nil {
		t.Fatalf("app pool: %v", err)
	}
	t.Cleanup(app.Close)
	b, err := New(testKEK(t), app, ex)
	if err != nil {
		t.Fatal(err)
	}
	return b, owner
}

func intentFor(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func TestBrokerMintConsumeLifecycle(t *testing.T) {
	ex := &fakeExchanger{wantRefresh: "refresh-1", access: "access-1", ttl: time.Hour}
	b, _ := testBroker(t, ex)
	ctx := context.Background()
	if err := b.Enroll(ctx, "tenant-a", "prov", []byte("refresh-1")); err != nil {
		t.Fatalf("Enroll: %v", err)
	}
	intent := intentFor("tenant-a", "exec-1")
	leaseID, access, expires, err := b.Mint(ctx, "tenant-a", "prov", "invoice.read", intent, 10*time.Minute, true)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if access != "access-1" || ex.calls != 1 {
		t.Fatalf("mint handed access=%q calls=%d", access, ex.calls)
	}
	if time.Until(expires) > 10*time.Minute+time.Minute || time.Until(expires) <= 0 {
		t.Fatalf("expiry %s out of range", expires)
	}
	if err := b.Consume(ctx, "tenant-a", leaseID, "invoice.read", intent); err != nil {
		t.Fatalf("Consume: %v", err)
	}
	// Single-use: the second consume refuses.
	if err := b.Consume(ctx, "tenant-a", leaseID, "invoice.read", intent); !errors.Is(err, ErrLeaseConsumed) {
		t.Fatalf("second consume err = %v, want ErrLeaseConsumed", err)
	}
	leases, err := b.Leases(ctx, "tenant-a", "")
	if err != nil || len(leases) != 1 || leases[0].ID != leaseID {
		t.Fatalf("Leases = %+v, %v", leases, err)
	}
}

func TestBrokerConsumeRefusals(t *testing.T) {
	ex := &fakeExchanger{wantRefresh: "refresh-1", access: "access-1", ttl: time.Hour}
	b, owner := testBroker(t, ex)
	ctx := context.Background()
	if err := b.Enroll(ctx, "tenant-a", "prov", []byte("refresh-1")); err != nil {
		t.Fatalf("Enroll: %v", err)
	}
	intent := intentFor("tenant-a", "exec-1")
	leaseID, _, _, err := b.Mint(ctx, "tenant-a", "prov", "cap", intent, 10*time.Minute, false)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	// Cross-tenant: tenant-b cannot see or consume tenant-a's lease.
	if err := b.Consume(ctx, "tenant-b", leaseID, "cap", intent); !errors.Is(err, ErrLeaseNotFound) {
		t.Fatalf("cross-tenant consume err = %v, want ErrLeaseNotFound", err)
	}
	// Capability/intent binding: wrong scope refuses even live leases.
	if err := b.Consume(ctx, "tenant-a", leaseID, "other-cap", intent); !errors.Is(err, ErrLeaseMismatch) {
		t.Fatalf("capability mismatch err = %v, want ErrLeaseMismatch", err)
	}
	if err := b.Consume(ctx, "tenant-a", leaseID, "cap", intentFor("other")); !errors.Is(err, ErrLeaseMismatch) {
		t.Fatalf("intent mismatch err = %v, want ErrLeaseMismatch", err)
	}
	// Multi-use lease consumes repeatedly while live.
	if err := b.Consume(ctx, "tenant-a", leaseID, "cap", intent); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if err := b.Consume(ctx, "tenant-a", leaseID, "cap", intent); err != nil {
		t.Fatalf("second consume of multi-use: %v", err)
	}
	// Post-expiry: force the row into the past, then consume refuses.
	if _, err := owner.Exec(ctx, `UPDATE studio.broker_leases SET expires_at = now() - interval '1 second' WHERE id = $1`, leaseID); err != nil {
		t.Fatal(err)
	}
	if err := b.Consume(ctx, "tenant-a", leaseID, "cap", intent); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expired consume err = %v, want ErrLeaseExpired", err)
	}
	if err := b.Consume(ctx, "tenant-a", "no-such-lease", "cap", intent); !errors.Is(err, ErrLeaseNotFound) {
		t.Fatalf("missing lease err = %v, want ErrLeaseNotFound", err)
	}
}

func TestBrokerRevocationAndInvalidGrant(t *testing.T) {
	ex := &fakeExchanger{wantRefresh: "refresh-1", access: "access-1", ttl: time.Hour}
	b, _ := testBroker(t, ex)
	ctx := context.Background()
	if err := b.Enroll(ctx, "tenant-a", "prov", []byte("refresh-1")); err != nil {
		t.Fatal(err)
	}
	if err := b.Revoke(ctx, "tenant-a", "prov"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, _, _, err := b.Mint(ctx, "tenant-a", "prov", "cap", intentFor("x"), time.Minute, true); !errors.Is(err, ErrCredentialRevoked) {
		t.Fatalf("mint after revoke err = %v, want ErrCredentialRevoked", err)
	}
	if err := b.Revoke(ctx, "tenant-a", "nope"); !errors.Is(err, ErrCredentialNotFound) {
		t.Fatalf("revoke missing err = %v, want ErrCredentialNotFound", err)
	}
	// Re-enrollment (operator re-consent) clears revocation.
	if err := b.Enroll(ctx, "tenant-a", "prov", []byte("refresh-2")); err != nil {
		t.Fatal(err)
	}
	ex.wantRefresh = "refresh-2"
	if _, _, _, err := b.Mint(ctx, "tenant-a", "prov", "cap", intentFor("x"), time.Minute, true); err != nil {
		t.Fatalf("mint after re-enroll: %v", err)
	}
	// Provider-dead refresh auto-revokes on invalid_grant.
	ex.err = ErrInvalidGrant
	if _, _, _, err := b.Mint(ctx, "tenant-a", "prov", "cap", intentFor("y"), time.Minute, true); !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("mint with dead refresh err = %v, want ErrInvalidGrant", err)
	}
	ex.err = nil
	if _, _, _, err := b.Mint(ctx, "tenant-a", "prov", "cap", intentFor("z"), time.Minute, true); !errors.Is(err, ErrCredentialRevoked) {
		t.Fatalf("mint after auto-revoke err = %v, want ErrCredentialRevoked", err)
	}
}

func TestBrokerMintValidation(t *testing.T) {
	ex := &fakeExchanger{wantRefresh: "r", access: "a", ttl: time.Hour}
	b, _ := testBroker(t, ex)
	ctx := context.Background()
	if err := b.Enroll(ctx, "tenant-a", "prov", []byte("r")); err != nil {
		t.Fatal(err)
	}
	for name, fn := range map[string]func() error{
		"empty-capability": func() error {
			_, _, _, err := b.Mint(ctx, "tenant-a", "prov", "", intentFor("x"), time.Minute, true)
			return err
		},
		"bad-intent": func() error {
			_, _, _, err := b.Mint(ctx, "tenant-a", "prov", "cap", "not-a-hash", time.Minute, true)
			return err
		},
		"zero-ttl": func() error {
			_, _, _, err := b.Mint(ctx, "tenant-a", "prov", "cap", intentFor("x"), 0, true)
			return err
		},
		"over-ttl": func() error {
			_, _, _, err := b.Mint(ctx, "tenant-a", "prov", "cap", intentFor("x"), 2*time.Hour, true)
			return err
		},
		"missing-credential": func() error {
			_, _, _, err := b.Mint(ctx, "tenant-a", "other", "cap", intentFor("x"), time.Minute, true)
			return err
		},
	} {
		if err := fn(); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	if ex.calls != 0 {
		t.Fatalf("validation failures reached the provider (%d calls)", ex.calls)
	}
}

// TestBrokerRowsHoldNoSecrets is the canary proof: refresh and access
// material flow through enroll/mint, then every broker row is dumped
// and asserted free of both canaries. Ciphertext must not contain
// the plaintext either (AES-GCM diffuses, but the test pins it).
func TestBrokerRowsHoldNoSecrets(t *testing.T) {
	refreshCanary, accessCanary := "canary-refresh-9f31", "canary-access-77ae"
	ex := &fakeExchanger{wantRefresh: refreshCanary, access: accessCanary, ttl: time.Hour}
	b, owner := testBroker(t, ex)
	ctx := context.Background()
	if err := b.Enroll(ctx, "tenant-a", "prov", []byte(refreshCanary)); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := b.Mint(ctx, "tenant-a", "prov", "cap", intentFor("x"), time.Minute, true); err != nil {
		t.Fatal(err)
	}
	var dump strings.Builder
	for _, q := range []string{
		`SELECT tenant_id, provider, encode(refresh_ct, 'escape'), kek_id FROM studio.broker_credentials`,
		`SELECT id, tenant_id, provider, capability, intent_hash FROM studio.broker_leases`,
	} {
		rows, err := owner.Query(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			vals, err := rows.Values()
			if err != nil {
				t.Fatal(err)
			}
			for _, v := range vals {
				dump.WriteString(safeString(v))
				dump.WriteString("|")
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
	}
	for _, leak := range []string{refreshCanary, accessCanary} {
		if strings.Contains(dump.String(), leak) {
			t.Fatalf("broker rows leaked %q", leak)
		}
	}
}

func safeString(v any) string {
	if v == nil {
		return "NULL"
	}
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	if t, ok := v.(time.Time); ok {
		return t.Format(time.RFC3339)
	}
	return "?"
}
