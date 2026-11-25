package runner

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/stack"

	"github.com/PithomLabs/miniapp-studio/connector"
	"github.com/PithomLabs/miniapp-studio/spec"
)

const (
	regTenantA = "acme-workspace"
	regTenantB = "other-tenant"
	regCredA   = "cred-runner-a"
	regCredB   = "cred-runner-b"
)

func assembleRunner(t *testing.T) (*stack.Assembled, *fakeReader) {
	t.Helper()
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.audit_events")
	binding := func(agent, tenant string) auth.Binding {
		return auth.Binding{AgentID: agent, Authority: contract.Authority{
			Kind: contract.AuthorityUser, ID: agent, TenantID: tenant,
		}}
	}
	policy := auth.NewPolicy(
		map[string]auth.Binding{
			regCredA: binding("agent-a", regTenantA),
			regCredB: binding("agent-b", regTenantB),
		},
		map[string]map[string]contract.TrustMode{
			regTenantA: {ID: contract.TrustAutonomous},
			regTenantB: {ID: contract.TrustAutonomous},
		},
	)
	reg := connector.NewFakeRegistry(
		connector.Connection{ID: "conn_box_001", Provider: "box"},
		connector.Connection{ID: "conn_custom_po", Provider: "custom"},
	)
	reader := &fakeReader{t: t, bodies: map[string]string{
		"conn_box_001 /files":          `[{"id":"inv-1","vendor":"Acme","amount":100.0,"status":"pending"}]`,
		"conn_box_001 /files/weekly":   `[{"id":"inv-1","vendor":"Acme","amount":100.0,"status":"pending"}]`,
		"conn_custom_po /orders":       `[{"id":"po-1","total":100.0}]`,
		"conn_custom_po /orders/match": `[{"id":"po-1","total":100.0}]`,
	}}
	deps := Deps{Connections: reg, Reader: reader}
	st := stack.Assemble(t, stack.Config{DB: db,
		Register: func(t *testing.T, r *registry.Registry) {
			mf := Manifest()
			if err := r.Register(mf, NewImplementation(deps), testkit.Envelope(t, mf, Build)); err != nil {
				t.Fatalf("register runner: %v", err)
			}
		},
		Policy: policy,
	})
	return st, reader
}

func runnerInput(t *testing.T, specDoc json.RawMessage, workflow string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"spec": json.RawMessage(specDoc), "workflow": workflow})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestRunnerInvokeEndToEnd(t *testing.T) {
	st, _ := assembleRunner(t)
	raw, err := os.ReadFile("../spec/testdata/valid_invoice_review.json")
	if err != nil {
		t.Fatal(err)
	}
	out, err := st.Svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: ID, Version: Version, Tool: ToolInvoke,
		Input:          runnerInput(t, raw, "prepare_review"),
		IdempotencyKey: "runner-e2e-1", Credential: regCredA,
		RequestedTrust: contract.TrustAutonomous,
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if out.Status != contract.RunSucceeded {
		t.Fatalf("status = %s, want succeeded", out.Status)
	}
	var res Result
	if err := json.Unmarshal(out.Result, &res); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if res.WorkflowID != "prepare_review" || !res.Completed {
		t.Errorf("result = %+v", res)
	}
	parsed, errs := spec.Validate(raw)
	if len(errs) != 0 {
		t.Fatalf("fixture invalid: %v", errs)
	}
	want, err := spec.ContentHash(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if res.SpecHash != want {
		t.Error("run not bound to the spec content hash")
	}
	if _, ok := res.Outputs["comparison"]; !ok {
		t.Errorf("outputs lack comparison: %v", res.Outputs)
	}
}

func TestRunnerRejectsInvalidSpec(t *testing.T) {
	st, _ := assembleRunner(t)
	raw, err := os.ReadFile("../spec/testdata/valid_invoice_review.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	ops := doc["capabilities"].(map[string]any)["operations"].([]any)
	ops[0].(map[string]any)["method"] = "POST"
	tampered, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.Svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: ID, Version: Version, Tool: ToolInvoke,
		Input:          runnerInput(t, tampered, "prepare_review"),
		IdempotencyKey: "runner-e2e-2", Credential: regCredA,
		RequestedTrust: contract.TrustAutonomous,
	})
	if err == nil {
		t.Fatal("tampered spec executed, want refusal")
	}
}

func TestRunnerRejectsTenantMismatch(t *testing.T) {
	st, _ := assembleRunner(t)
	raw, err := os.ReadFile("../spec/testdata/valid_invoice_review.json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.Svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: ID, Version: Version, Tool: ToolInvoke,
		Input:          runnerInput(t, raw, "prepare_review"),
		IdempotencyKey: "runner-e2e-3", Credential: regCredB,
		RequestedTrust: contract.TrustAutonomous,
	})
	if err == nil {
		t.Fatal("cross-tenant invocation executed, want refusal")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Errorf("error = %v, want tenant-mismatch refusal", err)
	}
}
