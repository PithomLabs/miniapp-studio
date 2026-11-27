package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

// setupReceiptsEnv truncates the receipts table so lifecycle tests see a
// clean slate on the shared run database. Dedicated tenants make
// leftovers from other packages harmless either way.
func setupReceiptsEnv(t *testing.T) testEnv {
	t.Helper()
	env := setupTestEnv(t)
	if _, err := env.owner.Exec(context.Background(),
		`TRUNCATE studio.sprite_lifecycle_receipts`); err != nil {
		t.Fatalf("truncate receipts: %v", err)
	}
	return env
}

func testReceipt(key, sprite string) LifecycleReceipt {
	return LifecycleReceipt{
		Key: key, Sprite: sprite, Transition: "provision",
		From: "absent", To: "awake", At: time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC),
		Approver: "owner", AuthRef: "apr-1", Provider: "sprites-sdk", Latency: 250 * time.Millisecond,
	}
}

func TestSaveGetLifecycleReceipt(t *testing.T) {
	env := setupReceiptsEnv(t)
	ctx := context.Background()
	want := testReceipt("rk-1", "b1")
	if err := env.store.SaveLifecycleReceipt(ctx, "t-lc", want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := env.store.LifecycleReceiptByKey(ctx, "t-lc", "rk-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	want.At = want.At.UTC()
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestLifecycleReceiptDuplicateKey(t *testing.T) {
	env := setupReceiptsEnv(t)
	ctx := context.Background()
	if err := env.store.SaveLifecycleReceipt(ctx, "t-lc", testReceipt("rk-dup", "b1")); err != nil {
		t.Fatal(err)
	}
	if err := env.store.SaveLifecycleReceipt(ctx, "t-lc", testReceipt("rk-dup", "b1")); !errors.Is(err, ErrReceiptExists) {
		t.Errorf("duplicate err = %v, want ErrReceiptExists", err)
	}
	// Same key under another tenant is independent (UNIQUE is per-tenant).
	if err := env.store.SaveLifecycleReceipt(ctx, "t-lc-2", testReceipt("rk-dup", "b1")); err != nil {
		t.Errorf("cross-tenant same key: %v", err)
	}
}

func TestLifecycleReceiptMissing(t *testing.T) {
	env := setupReceiptsEnv(t)
	ctx := context.Background()
	if _, err := env.store.LifecycleReceiptByKey(ctx, "t-lc", "rk-ghost"); !errors.Is(err, ErrNoReceipt) {
		t.Errorf("err = %v, want ErrNoReceipt", err)
	}
	if _, found, err := env.store.LatestLifecyclePosition(ctx, "t-lc", "ghost"); err != nil || found {
		t.Errorf("position = %v, %v; want absent", found, err)
	}
}

func TestLatestLifecyclePosition(t *testing.T) {
	env := setupReceiptsEnv(t)
	ctx := context.Background()
	first := testReceipt("rk-p1", "b1")
	second := testReceipt("rk-p2", "b1")
	second.Transition, second.From, second.To = "connect", "awake", "connected"
	second.At = second.At.Add(time.Minute)
	if err := env.store.SaveLifecycleReceipt(ctx, "t-lc", first); err != nil {
		t.Fatal(err)
	}
	if err := env.store.SaveLifecycleReceipt(ctx, "t-lc", second); err != nil {
		t.Fatal(err)
	}
	to, found, err := env.store.LatestLifecyclePosition(ctx, "t-lc", "b1")
	if err != nil || !found || to != "connected" {
		t.Errorf("position = %q, %v, %v; want connected", to, found, err)
	}
}

func TestLifecycleReceiptTenantIsolation(t *testing.T) {
	env := setupReceiptsEnv(t)
	ctx := context.Background()
	if err := env.store.SaveLifecycleReceipt(ctx, "t-lc-a", testReceipt("rk-iso", "b1")); err != nil {
		t.Fatal(err)
	}
	if _, err := env.store.LifecycleReceiptByKey(ctx, "t-lc-b", "rk-iso"); !errors.Is(err, ErrNoReceipt) {
		t.Errorf("cross-tenant read err = %v, want ErrNoReceipt", err)
	}
	var unscoped int
	if err := env.app.QueryRow(ctx,
		`SELECT count(*) FROM studio.sprite_lifecycle_receipts`).Scan(&unscoped); err != nil {
		t.Fatalf("unscoped read: %v", err)
	}
	if unscoped != 0 {
		t.Errorf("unscoped read saw %d rows, want 0", unscoped)
	}
}

func TestLifecycleReceiptCheckRefusal(t *testing.T) {
	env := setupReceiptsEnv(t)
	ctx := context.Background()
	bad := testReceipt("rk-bad", "b1")
	bad.Transition = "teleport"
	if err := env.store.SaveLifecycleReceipt(ctx, "t-lc", bad); err == nil {
		t.Error("unknown transition accepted, want CHECK refusal")
	}
	bad = testReceipt("rk-bad2", "b1")
	bad.To = "everywhere"
	if err := env.store.SaveLifecycleReceipt(ctx, "t-lc", bad); err == nil {
		t.Error("unknown state accepted, want CHECK refusal")
	}
}
