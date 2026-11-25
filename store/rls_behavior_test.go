package store

import (
	"context"
	"testing"
)

// TestPhaseARLSEnforced proves the 000016 policies actually filter: as the
// application role, an unscoped read sees zero rows, and an armed read sees
// exactly the armed tenant's rows. Uses dedicated tenants so shared-DB
// leftovers cannot affect the counts.
func TestPhaseARLSEnforced(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	for _, tenant := range []string{"t-rls-a", "t-rls-b"} {
		if _, err := env.owner.Exec(ctx,
			`INSERT INTO studio.conversations (id, tenant_id, title, status)
			 VALUES ($1, $2, 't', 'open')`, "rls-conv-"+tenant, tenant); err != nil {
			t.Fatalf("seed conv: %v", err)
		}
		if _, err := env.owner.Exec(ctx,
			`INSERT INTO studio.messages (id, tenant_id, conversation_id, role, body)
			 VALUES ($1, $2, $3, 'user', 'hi')`,
			"rls-msg-"+tenant, tenant, "rls-conv-"+tenant); err != nil {
			t.Fatalf("seed message: %v", err)
		}
	}
	// Unarmed: the policy compares against NULL, so nothing is visible.
	var unscoped int
	if err := env.app.QueryRow(ctx,
		`SELECT count(*) FROM studio.messages`).Scan(&unscoped); err != nil {
		t.Fatalf("unscoped read: %v", err)
	}
	if unscoped != 0 {
		t.Fatalf("unscoped app-role read saw %d rows, want 0 (policy inert?)", unscoped)
	}
	// Armed: exactly the armed tenant's rows.
	tx, err := env.app.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SET LOCAL app.tenant_id = 't-rls-a'`); err != nil {
		t.Fatal(err)
	}
	var scoped int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM studio.messages WHERE tenant_id = 't-rls-a'`).Scan(&scoped); err != nil {
		t.Fatal(err)
	}
	if scoped != 1 {
		t.Errorf("armed read saw %d t-rls-a rows, want 1", scoped)
	}
	var foreign int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM studio.messages WHERE tenant_id = 't-rls-b'`).Scan(&foreign); err != nil {
		t.Fatal(err)
	}
	if foreign != 0 {
		t.Errorf("armed read saw %d t-rls-b rows, want 0", foreign)
	}
	// WITH CHECK: armed inserts land in the armed tenant only.
	if _, err := tx.Exec(ctx,
		`INSERT INTO studio.messages (id, tenant_id, conversation_id, role, body)
		 VALUES ('rls-msg-a2', 't-rls-a', 'rls-conv-t-rls-a', 'user', 'ok')`); err != nil {
		t.Errorf("armed own-tenant insert: %v", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO studio.messages (id, tenant_id, conversation_id, role, body)
		 VALUES ('rls-msg-b2', 't-rls-b', 'rls-conv-t-rls-b', 'user', 'no')`); err == nil {
		t.Error("armed cross-tenant insert accepted, want refusal")
	}
}
