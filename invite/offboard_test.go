package invite

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/PithomLabs/miniapp-studio/store"
)

// Offboarding regression suite for F35-01. Each test maps to a
// prompt36 §2 case; all run against the live run database and skip
// honestly without Neon.

// Case 1: an active operator accepts a valid invitation.
func TestOffboardActiveAcceptsValidInvite(t *testing.T) {
	svc, owner, _ := testService(t)
	ctx := context.Background()
	if err := store.UpsertOperator(ctx, owner, "tenant-ob-1", "op-a", true); err != nil {
		t.Fatal(err)
	}
	_, token, err := svc.Create(ctx, "tenant-ob-1", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	sess, _, err := svc.Accept(ctx, token, time.Now())
	if err != nil {
		t.Fatalf("accept for active operator: %v", err)
	}
	if sess.TenantID != "tenant-ob-1" || sess.BindingID != "op-a" {
		t.Fatalf("session %+v", sess)
	}
}

// Case 2: an inactive operator cannot accept an outstanding invite.
func TestOffboardInactiveCannotAccept(t *testing.T) {
	svc, owner, _ := testService(t)
	ctx := context.Background()
	if err := store.UpsertOperator(ctx, owner, "tenant-ob-2", "op-a", false); err != nil {
		t.Fatal(err)
	}
	_, token, err := svc.Create(ctx, "tenant-ob-2", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Accept(ctx, token, time.Now()); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("accept for inactive operator err = %v, want ErrInviteInvalid", err)
	}
}

// Case 7: every failed acceptance rolls back all relevant state
// (inactive-gate path: no consume, no session, operator unchanged).
func TestOffboardFailedAcceptRollsBack(t *testing.T) {
	svc, owner, _ := testService(t)
	ctx := context.Background()
	if err := store.UpsertOperator(ctx, owner, "tenant-ob-7", "op-a", false); err != nil {
		t.Fatal(err)
	}
	_, token, err := svc.Create(ctx, "tenant-ob-7", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Accept(ctx, token, time.Now()); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("accept err = %v", err)
	}
	var consumed, revoked bool
	if err := owner.QueryRow(ctx, `SELECT consumed_at IS NOT NULL, revoked_at IS NOT NULL FROM studio.invitations WHERE tenant_id = 'tenant-ob-7'`).Scan(&consumed, &revoked); err != nil {
		t.Fatal(err)
	}
	if consumed || revoked {
		t.Fatalf("refused accept mutated invite: consumed=%v revoked=%v", consumed, revoked)
	}
	if exists, active := operatorActive(t, owner, "tenant-ob-7", "op-a"); !exists || active {
		t.Fatalf("operator mutated: exists=%v active=%v", exists, active)
	}
	var sessions int
	if err := owner.QueryRow(ctx, `SELECT COUNT(*) FROM studio.sessions WHERE tenant_id = 'tenant-ob-7'`).Scan(&sessions); err != nil || sessions != 0 {
		t.Fatalf("sessions = %d (err=%v)", sessions, err)
	}
}

// Case 3: offboarding wins when serialized before acceptance.
func TestOffboardWinsSerialized(t *testing.T) {
	svc, owner, _ := testService(t)
	ctx := context.Background()
	_, token, err := svc.Create(ctx, "tenant-ob-3", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.OffboardOperator(ctx, owner, "tenant-ob-3", "op-a"); err != nil {
		t.Fatalf("offboard: %v", err)
	}
	if _, _, err := svc.Accept(ctx, token, time.Now()); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("accept after offboard err = %v", err)
	}
	var revoked bool
	if err := owner.QueryRow(ctx, `SELECT revoked_at IS NOT NULL FROM studio.invitations WHERE tenant_id = 'tenant-ob-3'`).Scan(&revoked); err != nil || !revoked {
		t.Fatalf("outstanding invite not revoked (revoked=%v err=%v)", revoked, err)
	}
}

// Case 4: acceptance wins only when validly serialized first.
func TestOffboardAcceptWinsSerializedFirst(t *testing.T) {
	svc, owner, _ := testService(t)
	ctx := context.Background()
	_, token, err := svc.Create(ctx, "tenant-ob-4", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Accept(ctx, token, time.Now()); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if err := store.OffboardOperator(ctx, owner, "tenant-ob-4", "op-a"); err != nil {
		t.Fatalf("offboard: %v", err)
	}
	if exists, active := operatorActive(t, owner, "tenant-ob-4", "op-a"); !exists || active {
		t.Fatalf("operator exists=%v active=%v, want inactive", exists, active)
	}
	var sessions int
	if err := owner.QueryRow(ctx, `SELECT COUNT(*) FROM studio.sessions WHERE tenant_id = 'tenant-ob-4'`).Scan(&sessions); err != nil || sessions != 1 {
		t.Fatalf("sessions = %d (err=%v), want the won session", sessions, err)
	}
}

// Case 5: reactivation does not revive an old invitation.
func TestOffboardReactivationDoesNotRevive(t *testing.T) {
	svc, owner, _ := testService(t)
	ctx := context.Background()
	_, token, err := svc.Create(ctx, "tenant-ob-5", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.OffboardOperator(ctx, owner, "tenant-ob-5", "op-a"); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertOperator(ctx, owner, "tenant-ob-5", "op-a", true); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if _, _, err := svc.Accept(ctx, token, time.Now()); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("old invite after reactivation err = %v", err)
	}
}

// Case 6: owner reactivation plus a fresh invitation succeeds.
func TestOffboardReactivationFreshInviteSucceeds(t *testing.T) {
	svc, owner, _ := testService(t)
	ctx := context.Background()
	if err := store.UpsertOperator(ctx, owner, "tenant-ob-6", "op-a", true); err != nil {
		t.Fatal(err)
	}
	if err := store.OffboardOperator(ctx, owner, "tenant-ob-6", "op-a"); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertOperator(ctx, owner, "tenant-ob-6", "op-a", true); err != nil {
		t.Fatal(err)
	}
	_, token, err := svc.Create(ctx, "tenant-ob-6", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Accept(ctx, token, time.Now()); err != nil {
		t.Fatalf("fresh invite after reactivation: %v", err)
	}
}

// Case 8 (owner amendment): raw-SQL deactivate → raw-SQL reactivate
// still kills the old invitation (trigger path, no function used).
func TestOffboardRawSQLReactivationKillsInvite(t *testing.T) {
	svc, owner, _ := testService(t)
	ctx := context.Background()
	if err := store.UpsertOperator(ctx, owner, "tenant-ob-8", "op-a", true); err != nil {
		t.Fatal(err)
	}
	_, token, err := svc.Create(ctx, "tenant-ob-8", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `UPDATE studio.operators SET active = false WHERE tenant_id = 'tenant-ob-8' AND binding_id = 'op-a'`); err != nil {
		t.Fatal(err)
	}
	// While inactive the invite refuses (gate path, no state change).
	if _, _, err := svc.Accept(ctx, token, time.Now()); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("accept while raw-inactive err = %v", err)
	}
	if _, err := owner.Exec(ctx, `UPDATE studio.operators SET active = true WHERE tenant_id = 'tenant-ob-8' AND binding_id = 'op-a'`); err != nil {
		t.Fatal(err)
	}
	var revoked bool
	if err := owner.QueryRow(ctx, `SELECT revoked_at IS NOT NULL FROM studio.invitations WHERE tenant_id = 'tenant-ob-8'`).Scan(&revoked); err != nil || !revoked {
		t.Fatalf("trigger did not revoke (revoked=%v err=%v)", revoked, err)
	}
	if _, _, err := svc.Accept(ctx, token, time.Now()); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("old invite after raw reactivation err = %v", err)
	}
}

// Dynamic counterpart to cases 3+4: concurrent offboard vs accept.
// Exactly one order wins; end states are deterministic either way:
// the operator ends inactive, and consumed ⟺ exactly one session.
func TestOffboardAcceptRace(t *testing.T) {
	svc, owner, _ := testService(t)
	ctx := context.Background()
	_, token, err := svc.Create(ctx, "tenant-ob-race", "op-a", "owner-op", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	acceptErr := make(chan error, 1)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _, err := svc.Accept(ctx, token, time.Now())
		acceptErr <- err
	}()
	go func() {
		defer wg.Done()
		if err := store.OffboardOperator(ctx, owner, "tenant-ob-race", "op-a"); err != nil {
			t.Errorf("offboard: %v", err)
		}
	}()
	wg.Wait()
	close(acceptErr)
	for err := range acceptErr {
		if err != nil && !errors.Is(err, ErrInviteInvalid) {
			t.Fatalf("accept error quality: %v", err)
		}
	}
	if exists, active := operatorActive(t, owner, "tenant-ob-race", "op-a"); !exists || active {
		t.Fatalf("operator exists=%v active=%v, want inactive", exists, active)
	}
	var consumed bool
	if err := owner.QueryRow(ctx, `SELECT consumed_at IS NOT NULL FROM studio.invitations WHERE tenant_id = 'tenant-ob-race'`).Scan(&consumed); err != nil {
		t.Fatal(err)
	}
	var sessions int
	if err := owner.QueryRow(ctx, `SELECT COUNT(*) FROM studio.sessions WHERE tenant_id = 'tenant-ob-race'`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if consumed != (sessions == 1) {
		t.Fatalf("consumed=%v sessions=%d: want consumed ⟺ one session", consumed, sessions)
	}
}
