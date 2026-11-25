package server

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/executor/classa"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit"

	"github.com/PithomLabs/miniapp-studio/runner"
	"github.com/PithomLabs/miniapp-studio/store"
)

// noReadSpec is a valid spec whose workflow touches no external data: one
// static finish step. It exercises Service choreography without providers.
func noReadSpec(t *testing.T, owner string) json.RawMessage {
	t.Helper()
	doc := map[string]any{
		"app":  map[string]any{"id": "t", "slug": "t", "name": "T", "appVersion": "1.0.0", "owner": owner},
		"data": map[string]any{"entities": []any{}},
		"ui":   map[string]any{"pages": []any{map[string]any{"id": "h", "title": "H", "path": "/", "components": []any{}}}},
		"workflows": []any{map[string]any{"id": "f", "trigger": map[string]any{"type": "manual"},
			"steps": []any{map[string]any{"id": "d", "type": "finish", "message": "done"}}}},
		"capabilities": map[string]any{"operations": []any{}},
		"policies":     map[string]any{"confirmation": map[string]any{"externalWrites": "disabled"}},
		"fixtures":     []any{map[string]any{"name": "f1"}},
		"metadata":     map[string]any{"schemaVersion": "miniappspec/v1"},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func truncateFramework(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	owner, err := store.NewPool(ctx, os.Getenv("TEST_DATABASE_URL"), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	if _, err := owner.Exec(ctx, `TRUNCATE app.invocations, app.audit_events`); err != nil {
		t.Fatalf("truncate framework: %v", err)
	}
}

func countRows(t *testing.T, table, where string, args ...any) int {
	t.Helper()
	owner, err := store.NewPool(context.Background(), os.Getenv("TEST_DATABASE_URL"), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	var n int
	q := `SELECT count(*) FROM app.` + table
	if where != "" {
		q += " WHERE " + where
	}
	if err := owner.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// TestServiceInvocationTenantScoped proves F2 end to end: a Class A Service
// invocation through the application pool lands tenant-scoped rows, writes
// audit evidence (the audit-INSERT branch), and attaches duplicates without
// re-executing. Fails before 000005 with 42501 on schema app.
func TestServiceInvocationTenantScoped(t *testing.T) {
	srv, _ := testServer(t)
	truncateFramework(t)
	ctx := context.Background()
	input, err := json.Marshal(map[string]any{
		"spec":     json.RawMessage(noReadSpec(t, testTenant)),
		"workflow": "f",
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := srv.Service().Invoke(ctx, runtime.InvokeRequest{
		MiniappID: runner.ID, Version: runner.Version, Tool: runner.ToolInvoke,
		Input: input, IdempotencyKey: "f2-key-1",
		Credential: testSvcCred, RequestedTrust: contract.TrustAutonomous,
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if out.Status != contract.RunSucceeded {
		t.Fatalf("status = %s", out.Status)
	}
	if n := countRows(t, "invocations", "tenant_id = $1", testTenant); n != 1 {
		t.Errorf("tenant invocations = %d, want 1", n)
	}
	if n := countRows(t, "audit_events", "tenant_id = $1", testTenant); n == 0 {
		t.Error("no audit rows for tenant: audit-INSERT branch broken")
	}
	again, err := srv.Service().Invoke(ctx, runtime.InvokeRequest{
		MiniappID: runner.ID, Version: runner.Version, Tool: runner.ToolInvoke,
		Input: input, IdempotencyKey: "f2-key-1",
		Credential: testSvcCred, RequestedTrust: contract.TrustAutonomous,
	})
	if err != nil {
		t.Fatalf("duplicate Invoke: %v", err)
	}
	if again.InvocationID != out.InvocationID {
		t.Error("duplicate submission created a second invocation")
	}
	if n := countRows(t, "invocations", "idempotency_key = $1", "f2-key-1"); n != 1 {
		t.Errorf("rows for key = %d, want 1", n)
	}
}

// TestBareInvocationFailsClosed proves tenant context is mandatory: the
// Class A executor refuses an empty tenant before touching the database
// (framework store.WithTenant guard, through which every executor path
// funnels), and no rows are written. The request is otherwise fully valid
// (registered tool) so the tenant guard is the only possible refusal.
func TestBareInvocationFailsClosed(t *testing.T) {
	truncateFramework(t)
	pool, err := store.NewPool(context.Background(), os.Getenv("TEST_DATABASE_URL"), 0)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	reg := registry.New()
	testkit.RegisterClassATool(t, reg, "bare-app", "bare.tool", "", nil,
		func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
			return json.RawMessage(`{}`), nil
		})
	ex := classa.New(pool, reg)
	hooks := &executor.ClaimHooks{
		Recheck: func(tx pgx.Tx) (bool, error) { return true, nil },
		Complete: func(tx pgx.Tx, status contract.RunStatus, result json.RawMessage, errMsg string) (bool, error) {
			return true, nil
		},
	}
	// Control: the identical request WITH a tenant succeeds, proving the
	// harness and tool wiring work and isolating tenant as the variable.
	if _, err := ex.Execute(context.Background(), executor.Request{
		TenantID: "t-bare", MiniappID: "bare-app", Version: "1.0.0",
		Tool: "bare.tool", Input: json.RawMessage(`{}`),
		InvocationID: "00000000-0000-0000-0000-000000000001", IdempotencyKey: "bare-key-1",
		Claim: hooks,
	}); err != nil {
		t.Fatalf("control execution failed: %v", err)
	}
	// Bare case: identical request with the tenant removed must refuse
	// before touching the database.
	_, err = ex.Execute(context.Background(), executor.Request{
		MiniappID: "bare-app", Version: "1.0.0",
		Tool: "bare.tool", Input: json.RawMessage(`{}`),
		InvocationID: "00000000-0000-0000-0000-000000000002", IdempotencyKey: "bare-key-2",
		Claim: hooks,
	})
	if err == nil {
		t.Fatal("empty-tenant execution accepted, want refusal")
	}
	if !strings.Contains(err.Error(), "tenant") {
		t.Errorf("error = %v, want tenant refusal", err)
	}
	if n := countRows(t, "invocations", "id = $1", "00000000-0000-0000-0000-000000000002"); n != 0 {
		t.Errorf("bare invocation wrote %d rows", n)
	}
}

// TestCrossTenantStatusDenied proves invocation reads stay tenant-scoped: a
// credential from another tenant cannot observe the invocation.
func TestCrossTenantStatusDenied(t *testing.T) {
	srv, _ := testServer(t)
	truncateFramework(t)
	ctx := context.Background()
	input, err := json.Marshal(map[string]any{
		"spec":     json.RawMessage(noReadSpec(t, testTenant)),
		"workflow": "f",
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := srv.Service().Invoke(ctx, runtime.InvokeRequest{
		MiniappID: runner.ID, Version: runner.Version, Tool: runner.ToolInvoke,
		Input: input, IdempotencyKey: "f2-key-x",
		Credential: testSvcCred, RequestedTrust: contract.TrustAutonomous,
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if _, err := srv.Service().Status(ctx, testSvcCredB, out.InvocationID); err == nil {
		t.Fatal("cross-tenant Status returned a record, want denial")
	} else if !strings.Contains(err.Error(), "unknown-invocation") {
		t.Errorf("error = %v, want unknown-invocation", err)
	}
}
