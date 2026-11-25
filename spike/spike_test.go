package spike

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/stack"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/testdb"
)

// runTestDB connects the framework stack to the isolated run database.
// Never testkit.NewDB(t): it follows TEST_DATABASE_URL to the
// bootstrap database, outside test isolation (invariant 7).
func runTestDB(t *testing.T) *testkit.DB {
	t.Helper()
	dsn := testdb.RunDSN(t)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return &testkit.DB{Pool: pool, DSN: dsn}
}

const (
	spikeTenant = "t-spike"
	spikeCred   = "cred-spike"
	spikeDecide = "cred-spike-decide"
	spikeApp    = "spike-probe"
	spikeTool   = "probe.echo"
	spikeGated  = "spike-gated"
	spikeGatedT = "probe.gated"
)

// assemble wires a production-shaped stack with a plain echo tool and an
// approval-gated echo tool, both counting handler executions.
func assemble(t *testing.T, calls *atomic.Int32) *stack.Assembled {
	t.Helper()
	db := runTestDB(t)
	db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events", "app.effect_records")
	binding := func(agent string, roles ...string) auth.Binding {
		return auth.Binding{AgentID: agent, Authority: contract.Authority{
			Kind: contract.AuthorityUser, ID: agent, TenantID: spikeTenant,
		}, Roles: roles}
	}
	policy := auth.NewPolicy(
		map[string]auth.Binding{
			spikeCred:   binding("agent-spike"),
			spikeDecide: binding("human-op", "tenant-ops"),
		},
		map[string]map[string]contract.TrustMode{
			spikeTenant: {
				spikeApp:   contract.TrustAutonomous,
				spikeGated: contract.TrustConfirm,
			},
		},
	)
	echo := func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
		calls.Add(1)
		return call.Input, nil
	}
	register := func(t *testing.T, reg *registry.Registry) {
		testkit.RegisterClassATool(t, reg, spikeApp, spikeTool, "", nil, echo)
		mf := testkit.ClassAManifest(spikeGated, spikeGatedT, "", nil)
		mf.Tools[0].Approval.Required = true
		if err := reg.Register(mf, testkit.ClassAImpl("spike", spikeGatedT, echo),
			testkit.Envelope(t, mf, "spike")); err != nil {
			t.Fatalf("register gated: %v", err)
		}
	}
	return stack.Assemble(t, stack.Config{DB: db, Register: register, Policy: policy})
}

func invoke(t *testing.T, svc *runtime.Service, miniapp, tool, key string, trust contract.TrustMode, input string) (runtime.InvokeResult, error) {
	t.Helper()
	return svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: miniapp, Version: "1.0.0", Tool: tool,
		Input: json.RawMessage(input), IdempotencyKey: key,
		Credential: spikeCred, RequestedTrust: trust,
	})
}

// TestImportClassAInvoke proves the pinned framework executes one Class A
// operation through the canonical path with echo fidelity.
func TestImportClassAInvoke(t *testing.T) {
	var calls atomic.Int32
	st := assemble(t, &calls)
	out, err := invoke(t, st.Svc, spikeApp, spikeTool, "spike-key-1", contract.TrustAutonomous, `{"ping":"pong"}`)
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if out.Status != contract.RunSucceeded {
		t.Fatalf("status = %s, want %s", out.Status, contract.RunSucceeded)
	}
	assertJSONEq(t, out.Result, `{"ping":"pong"}`)
	if calls.Load() != 1 {
		t.Errorf("handler calls = %d, want 1", calls.Load())
	}
}

// TestApprovalParkDecideResume proves the verified resumption contract:
// park → decide → same-key re-Invoke executes via pollApproval; Status never
// advances the approval; terminal replay and duplicates never re-execute.
func TestApprovalParkDecideResume(t *testing.T) {
	var calls atomic.Int32
	st := assemble(t, &calls)
	ctx := context.Background()
	const key, input = "spike-key-gated-1", `{"op":"read"}`

	parked, err := invoke(t, st.Svc, spikeGated, spikeGatedT, key, contract.TrustConfirm, input)
	if err != nil {
		t.Fatalf("park Invoke: %v", err)
	}
	if parked.Status != contract.RunWaitingApproval || parked.ApprovalID == "" {
		t.Fatalf("park = %+v, want waiting_approval with approval ID", parked)
	}
	if calls.Load() != 0 {
		t.Fatalf("handler executed before approval (calls=%d)", calls.Load())
	}

	// Status observes but never advances.
	if err := observe(t, st.Svc, parked.InvocationID, contract.RunWaitingApproval); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatalf("Status advanced execution (calls=%d)", calls.Load())
	}

	// Decide, then prove Status still does not advance.
	hash := runtime.IntentHash(spikeGated, "1.0.0", spikeGatedT, json.RawMessage(input))
	if err := st.Svc.DecideApproval(ctx, spikeDecide, parked.ApprovalID, contract.DecisionApprove, hash); err != nil {
		t.Fatalf("DecideApproval: %v", err)
	}
	if err := observe(t, st.Svc, parked.InvocationID, contract.RunWaitingApproval); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatalf("execution ran before resubmission (calls=%d)", calls.Load())
	}

	// Same-key resubmission resumes and executes exactly once.
	resumed, err := invoke(t, st.Svc, spikeGated, spikeGatedT, key, contract.TrustConfirm, input)
	if err != nil {
		t.Fatalf("resubmission Invoke: %v", err)
	}
	if resumed.Status != contract.RunSucceeded {
		t.Fatalf("resumed = %+v, want succeeded echo", resumed)
	}
	assertJSONEq(t, resumed.Result, input)
	if calls.Load() != 1 {
		t.Fatalf("handler calls = %d, want exactly 1", calls.Load())
	}

	// Terminal replay and duplicate submission never re-execute.
	again, err := invoke(t, st.Svc, spikeGated, spikeGatedT, key, contract.TrustConfirm, input)
	if err != nil {
		t.Fatalf("replay Invoke: %v", err)
	}
	if again.Status != contract.RunSucceeded {
		t.Fatalf("replay = %+v, want succeeded echo", again)
	}
	assertJSONEq(t, again.Result, input)
	if calls.Load() != 1 {
		t.Fatalf("replay re-executed handler (calls=%d)", calls.Load())
	}

	// Same key with different input is a terminal conflict, not a second run.
	if _, err := invoke(t, st.Svc, spikeGated, spikeGatedT, key, contract.TrustConfirm, `{"op":"other"}`); err == nil {
		t.Fatal("key reuse with different input accepted, want terminal conflict")
	} else if !strings.Contains(err.Error(), "idempotency-key-reuse") {
		t.Fatalf("key reuse error = %v, want idempotency-key-reuse", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("conflicting resubmission executed (calls=%d)", calls.Load())
	}
}

// TestApprovalDenySettlesTerminal proves denial ends the invocation with no
// execution and no later resumption.
func TestApprovalDenySettlesTerminal(t *testing.T) {
	var calls atomic.Int32
	st := assemble(t, &calls)
	ctx := context.Background()
	const key, input = "spike-key-gated-2", `{"op":"read"}`

	parked, err := invoke(t, st.Svc, spikeGated, spikeGatedT, key, contract.TrustConfirm, input)
	if err != nil {
		t.Fatalf("park Invoke: %v", err)
	}
	hash := runtime.IntentHash(spikeGated, "1.0.0", spikeGatedT, json.RawMessage(input))
	if err := st.Svc.DecideApproval(ctx, spikeDecide, parked.ApprovalID, contract.DecisionReject, hash); err != nil {
		t.Fatalf("DecideApproval: %v", err)
	}
	denied, err := invoke(t, st.Svc, spikeGated, spikeGatedT, key, contract.TrustConfirm, input)
	_ = denied
	if err == nil {
		t.Fatal("resubmission after deny accepted, want terminal denial")
	}
	if !strings.Contains(err.Error(), "approval-denied") {
		t.Fatalf("deny error = %v, want approval-denied", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("denied invocation executed (calls=%d)", calls.Load())
	}
}

// assertJSONEq compares payloads semantically: replayed results come back in
// the database's jsonb text form, which need not match input whitespace.
func assertJSONEq(t *testing.T, got json.RawMessage, want string) {
	t.Helper()
	var a, b any
	if err := json.Unmarshal(got, &a); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &b); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("result = %s, want %s", got, want)
	}
}

func observe(t *testing.T, svc *runtime.Service, invocationID string, want contract.RunStatus) error {
	t.Helper()
	out, err := svc.Status(context.Background(), spikeCred, invocationID)
	if err != nil {
		return err
	}
	if out.Status != want {
		t.Fatalf("Status = %s, want %s", out.Status, want)
	}
	return nil
}
