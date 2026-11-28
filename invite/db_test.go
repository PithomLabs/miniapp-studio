package invite

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/dbenv"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/testdb"
)

func testService(t *testing.T) (*Service, *pgxpool.Pool, *store.Store) {
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
	svc, err := New(app)
	if err != nil {
		t.Fatal(err)
	}
	return svc, owner, store.New(app)
}

func operatorActive(t *testing.T, owner *pgxpool.Pool, tenant, binding string) (exists, active bool) {
	t.Helper()
	err := owner.QueryRow(context.Background(),
		`SELECT true, active FROM studio.operators WHERE tenant_id = $1 AND binding_id = $2`,
		tenant, binding).Scan(&exists, &active)
	if err != nil {
		return false, false
	}
	return exists, active
}

func TestInviteLifecycle(t *testing.T) {
	svc, owner, sessions := testService(t)
	ctx := context.Background()
	id, token, err := svc.Create(ctx, "tenant-inv-1", "new-op", "owner-op", 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == "" || token == "" {
		t.Fatal("empty id or token")
	}
	sess, sessToken, err := svc.Accept(ctx, token, time.Now())
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if sess.TenantID != "tenant-inv-1" || sess.BindingID != "new-op" {
		t.Fatalf("session bound to %+v", sess)
	}
	// The issued session token authenticates through the real path.
	looked, err := sessions.Lookup(ctx, store.HashToken(sessToken))
	if err != nil {
		t.Fatalf("session lookup: %v", err)
	}
	if looked.TenantID != "tenant-inv-1" || looked.BindingID != "new-op" {
		t.Fatalf("looked up %+v", looked)
	}
	// Operator row provisioned and active.
	if exists, active := operatorActive(t, owner, "tenant-inv-1", "new-op"); !exists || !active {
		t.Fatalf("operator exists=%v active=%v", exists, active)
	}
	// Replay: no second session, uniform failure, no state change.
	if _, _, err := svc.Accept(ctx, token, time.Now()); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("replay err = %v, want ErrInviteInvalid", err)
	}
	list, err := svc.List(ctx, "tenant-inv-1")
	if err != nil || len(list) != 1 || !list[0].Consumed {
		t.Fatalf("list = %+v, %v", list, err)
	}
}

func TestInviteRefusals(t *testing.T) {
	svc, owner, _ := testService(t)
	ctx := context.Background()
	_, token, err := svc.Create(ctx, "tenant-inv-2", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	// Unknown token: uniform failure.
	if _, _, err := svc.Accept(ctx, "definitely-not-a-token", time.Now()); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("bad token err = %v", err)
	}
	// Expired: force the row into the past.
	if _, err := owner.Exec(ctx, `UPDATE studio.invitations SET expires_at = now() - interval '1 second' WHERE tenant_id = 'tenant-inv-2'`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Accept(ctx, token, time.Now()); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("expired err = %v", err)
	}
	// Revoked: fresh invite, revoked before accept.
	id2, token2, err := svc.Create(ctx, "tenant-inv-2", "op-b", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Revoke(ctx, "tenant-inv-2", id2); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, _, err := svc.Accept(ctx, token2, time.Now()); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("revoked err = %v", err)
	}
	// Cross-tenant revoke by id: uniform failure, victim row intact.
	id3, _, err := svc.Create(ctx, "tenant-inv-2", "op-c", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Revoke(ctx, "tenant-other", id3); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("cross-tenant revoke err = %v", err)
	}
	var revoked bool
	if err := owner.QueryRow(ctx, `SELECT revoked_at IS NOT NULL FROM studio.invitations WHERE id = $1`, id3).Scan(&revoked); err != nil || revoked {
		t.Fatalf("victim row revoked=%v err=%v", revoked, err)
	}
	// Cross-tenant list sees nothing.
	if list, err := svc.List(ctx, "tenant-other"); err != nil || len(list) != 0 {
		t.Fatalf("foreign list = %+v, %v", list, err)
	}
}

func TestInviteResendRotates(t *testing.T) {
	svc, _, _ := testService(t)
	ctx := context.Background()
	id, oldToken, err := svc.Create(ctx, "tenant-inv-3", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	newID, newToken, err := svc.Resend(ctx, "tenant-inv-3", id, time.Hour)
	if err != nil {
		t.Fatalf("Resend: %v", err)
	}
	if newID == id || newToken == oldToken {
		t.Fatal("resend did not rotate")
	}
	// Old token dead, new token works.
	if _, _, err := svc.Accept(ctx, oldToken, time.Now()); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("old token err = %v", err)
	}
	sess, _, err := svc.Accept(ctx, newToken, time.Now())
	if err != nil {
		t.Fatalf("new token: %v", err)
	}
	if sess.BindingID != "op-a" {
		t.Fatalf("resent session %+v", sess)
	}
	// Resending a consumed invite fails uniformly.
	if _, _, err := svc.Resend(ctx, "tenant-inv-3", newID, time.Hour); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("resend consumed err = %v", err)
	}
}

// TestInviteAcceptFailureRollsBack is the failure-injection proof for
// the atomic-acceptance safeguard: session_create is forced to refuse
// (expiry beyond its 30-day guard), so the whole transaction must
// roll back — invite unconsumed, no operator row, no session — and a
// normal retry then succeeds.
func TestInviteAcceptFailureRollsBack(t *testing.T) {
	svc, owner, _ := testService(t)
	ctx := context.Background()
	_, token, err := svc.Create(ctx, "tenant-inv-4", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.acceptWithExpiry(ctx, token, time.Now().Add(31*24*time.Hour)); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("forced failure err = %v, want ErrInviteInvalid", err)
	}
	var consumed, revoked bool
	if err := owner.QueryRow(ctx, `SELECT consumed_at IS NOT NULL, revoked_at IS NOT NULL FROM studio.invitations WHERE tenant_id = 'tenant-inv-4'`).Scan(&consumed, &revoked); err != nil {
		t.Fatal(err)
	}
	if consumed || revoked {
		t.Fatalf("failed accept mutated invite: consumed=%v revoked=%v", consumed, revoked)
	}
	if exists, _ := operatorActive(t, owner, "tenant-inv-4", "op-a"); exists {
		t.Fatal("failed accept left an operator row")
	}
	var sessions int
	if err := owner.QueryRow(ctx, `SELECT COUNT(*) FROM studio.sessions WHERE tenant_id = 'tenant-inv-4'`).Scan(&sessions); err != nil || sessions != 0 {
		t.Fatalf("failed accept left %d sessions (err=%v)", sessions, err)
	}
	// Retry with a valid expiry succeeds: the invite survived.
	if _, _, err := svc.Accept(ctx, token, time.Now()); err != nil {
		t.Fatalf("retry after rollback: %v", err)
	}
}

// TestInviteConcurrentAccept proves exactly-once acceptance under a
// race: one winner, one operator row, one session, losers uniform.
func TestInviteConcurrentAccept(t *testing.T) {
	svc, owner, _ := testService(t)
	ctx := context.Background()
	_, token, err := svc.Create(ctx, "tenant-inv-5", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	const racers = 8
	var wg sync.WaitGroup
	results := make(chan error, racers)
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := svc.Accept(ctx, token, time.Now())
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	var succeeded int
	for err := range results {
		if err == nil {
			succeeded++
		} else if !errors.Is(err, ErrInviteInvalid) {
			t.Fatalf("loser error quality: %v (want ErrInviteInvalid)", err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("winners = %d, want exactly 1", succeeded)
	}
	var sessions int
	if err := owner.QueryRow(ctx, `SELECT COUNT(*) FROM studio.sessions WHERE tenant_id = 'tenant-inv-5'`).Scan(&sessions); err != nil || sessions != 1 {
		t.Fatalf("sessions = %d (err=%v), want 1", sessions, err)
	}
}

// TestInviteRowsHoldNoTokens is the canary proof: the raw token from
// Create must appear nowhere in stored rows, and failure paths must
// not echo presented tokens.
func TestInviteRowsHoldNoTokens(t *testing.T) {
	svc, owner, _ := testService(t)
	ctx := context.Background()
	_, token, err := svc.Create(ctx, "tenant-inv-6", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	var dump strings.Builder
	rows, err := owner.Query(ctx, `SELECT id, tenant_id, token_hash, binding_id, invited_by FROM studio.invitations WHERE tenant_id = 'tenant-inv-6'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id, tenant, hash, binding, by string
		if err := rows.Scan(&id, &tenant, &hash, &binding, &by); err != nil {
			t.Fatal(err)
		}
		dump.WriteString(id + "|" + tenant + "|" + hash + "|" + binding + "|" + by + "\n")
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(dump.String(), token) {
		t.Fatal("invitation rows leaked the raw token")
	}
	if _, _, err := svc.Accept(ctx, "canary-presented-token-zz", time.Now()); err == nil || strings.Contains(err.Error(), "canary-presented-token-zz") {
		t.Fatalf("failure echoed the token: %v", err)
	}
}
