package runner

import (
	"context"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp-studio/connector"
	"github.com/PithomLabs/miniapp-studio/spec"
)

// fakeReader serves canned bodies keyed by connection ID + path, enforcing
// the read contract a real client must honor: GET only, byte cap respected.
type fakeReader struct {
	t      *testing.T
	bodies map[string]string
	calls  []string
}

func (f *fakeReader) Read(ctx context.Context, conn connector.Connection, method, path string, opts ReadOpts) ([]byte, error) {
	f.t.Helper()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if method != spec.MethodGet {
		f.t.Fatalf("non-GET read %s", method)
	}
	f.calls = append(f.calls, conn.ID+" "+path)
	body, ok := f.bodies[conn.ID+" "+path]
	if !ok {
		f.t.Fatalf("unexpected read %s %s", conn.ID, path)
	}
	limit := opts.MaxResponseBytes
	if limit <= 0 {
		limit = spec.DefaultResponseBytes
	}
	if len(body) > limit {
		return nil, &limitError{have: len(body), max: limit}
	}
	return []byte(body), nil
}

type limitError struct{ have, max int }

func (e *limitError) Error() string { return "response exceeds byte cap" }

func testDeps(t *testing.T) (Deps, *fakeReader) {
	t.Helper()
	reg := connector.NewFakeRegistry(
		connector.Connection{ID: "conn_box_001", Provider: "box"},
		connector.Connection{ID: "conn_custom_po", Provider: "custom"},
	)
	reader := &fakeReader{t: t, bodies: map[string]string{
		"conn_box_001 /files":          `[{"id":"inv-1","vendor":"Acme","amount":100.0,"status":"pending"},{"id":"inv-2","vendor":"Globex","amount":50.0,"status":"paid"}]`,
		"conn_box_001 /files/weekly":   `[{"id":"inv-1","vendor":"Acme","amount":100.0,"status":"pending"}]`,
		"conn_custom_po /orders":       `[{"id":"po-1","total":100.0}]`,
		"conn_custom_po /orders/match": `[{"id":"po-1","total":100.0}]`,
	}}
	return Deps{Connections: reg, Reader: reader}, reader
}

func admitValid(t *testing.T) ValidatedSpec {
	t.Helper()
	got, err := Admit(loadValidSpec(t), "acme-workspace")
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	return got
}

func TestExecutePrepareReview(t *testing.T) {
	deps, reader := testDeps(t)
	res, err := Execute(context.Background(), admitValid(t), "prepare_review", deps)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.Completed {
		t.Error("Completed = false, want true")
	}
	if res.WorkflowID != "prepare_review" || res.SpecHash == "" {
		t.Errorf("result identity = %+v", res)
	}
	comp, ok := res.Outputs["comparison"].(map[string]any)
	if !ok {
		t.Fatalf("outputs = %v, want comparison object", res.Outputs)
	}
	if _, ok := comp["invoice"]; !ok {
		t.Errorf("comparison lacks invoice input: %v", comp)
	}
	if len(res.Messages) == 0 || len(res.Notifications) == 0 {
		t.Errorf("messages/notifications missing: %+v", res)
	}
	for _, c := range reader.calls {
		if strings.Contains(c, "..") {
			t.Errorf("unnormalized read issued: %s", c)
		}
	}
}

func TestExecuteBranchGuard(t *testing.T) {
	deps, _ := testDeps(t)
	// Escalate's guard references the non-empty pending query: truthy.
	res, err := Execute(context.Background(), admitValid(t), "escalate", deps)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.Completed {
		t.Error("truthy guard ended the workflow")
	}
	if _, ok := res.Outputs["touched"]; !ok {
		t.Errorf("loop output missing: %v", res.Outputs)
	}
}

func TestExecuteUnknownWorkflow(t *testing.T) {
	deps, _ := testDeps(t)
	if _, err := Execute(context.Background(), admitValid(t), "nope", deps); err == nil {
		t.Error("unknown workflow executed, want error")
	}
}

func TestExecuteRequiresDeps(t *testing.T) {
	if _, err := Execute(context.Background(), admitValid(t), "prepare_review", Deps{}); err == nil {
		t.Error("nil deps accepted, want error")
	}
}

func TestExecuteHonorsContext(t *testing.T) {
	deps, _ := testDeps(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Execute(ctx, admitValid(t), "prepare_review", deps); err == nil {
		t.Error("cancelled context executed, want error")
	}
}

func TestExecStepDispatchDefault(t *testing.T) {
	r := &run{outputs: map[string]any{}}
	var res Result
	if _, err := r.execStep(context.Background(), &spec.Step{ID: "x", Type: "bogus"}, &res); err == nil {
		t.Error("unknown step type dispatched, want closed failure")
	}
}

func TestTruthiness(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want bool
	}{
		{"nil", nil, false},
		{"false", false, false},
		{"empty string", "", false},
		{"zero", 0.0, false},
		{"empty array", []any{}, false},
		{"empty object", map[string]any{}, false},
		{"true", true, true},
		{"string", "x", true},
		{"number", 1.0, true},
		{"nonempty array", []any{nil}, true},
		{"nonempty object", map[string]any{"a": nil}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := truthy(tc.in); got != tc.want {
				t.Errorf("truthy(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestLoopBoundExceeded(t *testing.T) {
	deps, _ := testDeps(t)
	r := &run{spec: &spec.MiniappSpec{}, deps: deps,
		outputs: map[string]any{"big": []any{1, 2, 3}}}
	err := r.execLoop(context.Background(), &spec.Step{
		ID: "l", Type: spec.StepBoundedLoop,
		Input:  map[string]string{"items": "${big}"},
		Output: "t", MaxIterations: 2,
	})
	if err == nil {
		t.Error("over-bound loop accepted, want error")
	}
}

func TestLoopRequiresArray(t *testing.T) {
	deps, _ := testDeps(t)
	r := &run{spec: &spec.MiniappSpec{}, deps: deps,
		outputs: map[string]any{"s": "nope"}}
	err := r.execLoop(context.Background(), &spec.Step{
		ID: "l", Type: spec.StepBoundedLoop,
		Input:  map[string]string{"items": "${s}"},
		Output: "t", MaxIterations: 10,
	})
	if err == nil {
		t.Error("non-array loop input accepted, want error")
	}
}

func TestReadRejectsUnknownConnection(t *testing.T) {
	reg := connector.NewFakeRegistry()
	deps := Deps{Connections: reg, Reader: &fakeReader{t: t, bodies: map[string]string{}}}
	r := &run{spec: &spec.MiniappSpec{Capabilities: spec.CapabilitiesSection{
		Operations: []spec.CapabilityOp{{
			ID: "o.p", Connection: "conn_ghost",
			Method: "GET", Paths: []string{"/*"}, Effect: "read",
		}},
	}}, deps: deps, outputs: map[string]any{}}
	err := r.execRead(context.Background(), &spec.Step{
		ID: "s", Type: spec.StepConnectorRead,
		Operation: "o.p", Path: "/x",
	})
	if err == nil {
		t.Error("unknown connection read accepted, want error")
	}
}

func TestReadRejectsWriteOp(t *testing.T) {
	deps, _ := testDeps(t)
	r := &run{spec: &spec.MiniappSpec{Capabilities: spec.CapabilitiesSection{
		Operations: []spec.CapabilityOp{{
			ID: "o.p", Connection: "conn_box_001",
			Method: "POST", Paths: []string{"/*"}, Effect: "write",
		}},
	}}, deps: deps, outputs: map[string]any{}}
	err := r.execRead(context.Background(), &spec.Step{
		ID: "s", Type: spec.StepConnectorRead,
		Operation: "o.p", Path: "/x",
	})
	if err == nil {
		t.Error("write-shaped op executed, want refusal")
	}
}

func TestQueryFilterSortLimit(t *testing.T) {
	deps, _ := testDeps(t)
	parsed, err := admitValid(t).Spec()
	if err != nil {
		t.Fatal(err)
	}
	r := &run{spec: parsed, deps: deps, outputs: map[string]any{}, entities: map[string][]any{}}
	qv, err := r.queryValue(context.Background(), "pending")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	rows := qv["rows"].([]any)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1 pending invoice", len(rows))
	}
	if rows[0].(map[string]any)["id"] != "inv-1" {
		t.Errorf("row = %v", rows[0])
	}
}

func TestQueryAggregates(t *testing.T) {
	deps, _ := testDeps(t)
	parsed, err := admitValid(t).Spec()
	if err != nil {
		t.Fatal(err)
	}
	r := &run{spec: parsed, deps: deps, outputs: map[string]any{}, entities: map[string][]any{}}
	qv, err := r.queryValue(context.Background(), "order_totals")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if qv["total"] != 100.0 {
		t.Errorf("total = %v, want 100", qv["total"])
	}
}

func TestResultSizeCap(t *testing.T) {
	doc := `{"app":{"id":"big","slug":"big","name":"B","appVersion":"1.0.0","owner":"acme"},
		"data":{"entities":[]},
		"ui":{"pages":[{"id":"h","title":"H","path":"/","components":[]}]},
		"workflows":[{"id":"f","trigger":{"type":"manual"},"steps":[
			{"id":"s1","type":"connector-read","operation":"big.read","path":"/a","output":"o1"},
			{"id":"s2","type":"connector-read","operation":"big.read","path":"/b","output":"o2"},
			{"id":"s3","type":"connector-read","operation":"big.read","path":"/c","output":"o3"},
			{"id":"s4","type":"connector-read","operation":"big.read","path":"/d","output":"o4"}]}],
		"capabilities":{"operations":[{"id":"big.read","connection":"c","method":"GET","paths":["/*"],"effect":"read","maxResponseBytes":1048576}]},
		"policies":{"confirmation":{"externalWrites":"disabled"}},
		"fixtures":[{"name":"f1"}],
		"metadata":{"schemaVersion":"miniappspec/v1"}}`
	got, err := Admit([]byte(doc), "acme")
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	blob := `["` + strings.Repeat("x", 300000) + `"]`
	deps := Deps{
		Connections: connector.NewFakeRegistry(connector.Connection{ID: "c"}),
		Reader: &fakeReader{t: t, bodies: map[string]string{
			"c /a": blob, "c /b": blob, "c /c": blob, "c /d": blob,
		}},
	}
	if _, err := Execute(context.Background(), got, "f", deps); err == nil {
		t.Error("oversized outputs accepted, want cap refusal")
	}
}
