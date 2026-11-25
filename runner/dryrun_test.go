package runner

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// drySpec is a minimal spec exercising dry-run edges: a branch over query
// rows plus a bounded loop over the same collection.
const drySpec = `{"app":{"id":"dryt","slug":"dryt","name":"D","appVersion":"1.0.0","owner":"acme"},
	"data":{"entities":[{"fields":[{"name":"id","type":"string"}],"name":"e","source":{"capability":"o.p","entity":"es"}}],
	"queries":[{"entity":"e","name":"q"}]},
	"ui":{"pages":[{"components":[],"id":"h","path":"/","title":"H"}]},
	"workflows":[{"id":"f","trigger":{"type":"manual"},"steps":[
		{"id":"g","type":"branch","condition":"${q.rows}"},
		{"id":"l","type":"bounded-loop","input":{"items":"${q.rows}"},"output":"touched","maxIterations":10},
		{"id":"d","type":"finish","message":"done ${touched}"}]}],
	"capabilities":{"operations":[{"connection":"c","effect":"read","id":"o.p","method":"GET","paths":["/es"]}]},
	"policies":{"confirmation":{"externalWrites":"disabled"}},
	"fixtures":[{"name":"empty","input":{"e":[]}},{"name":"full","input":{"e":[{"id":"a"},{"id":"b"}]}}],
	"metadata":{"schemaVersion":"miniappspec/v1"}}`

func TestDryRunSuccess(t *testing.T) {
	vspec := admitValid(t)
	res, err := DryRun(vspec, "one-pending-invoice", "prepare_review")
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if !res.DryRun || res.Fixture != "one-pending-invoice" {
		t.Errorf("dry-run provenance missing: %+v", res)
	}
	if res.SpecHash != vspec.Hash() || res.WorkflowID != "prepare_review" {
		t.Errorf("identity wrong: %+v", res)
	}
	if !res.Completed {
		t.Error("Completed = false, want true")
	}
	if _, ok := res.Outputs["comparison"]; !ok {
		t.Errorf("outputs lack comparison: %v", res.Outputs)
	}
	if len(res.Messages) == 0 || len(res.Notifications) == 0 {
		t.Errorf("messages/notifications missing: %+v", res)
	}
}

func TestDryRunUnknownNames(t *testing.T) {
	vspec := admitValid(t)
	if _, err := DryRun(vspec, "nope", "prepare_review"); err == nil {
		t.Error("unknown fixture accepted, want error")
	}
	if _, err := DryRun(vspec, "one-pending-invoice", "nope"); err == nil {
		t.Error("unknown workflow accepted, want error")
	}
}

func TestDryRunMissingInputActionable(t *testing.T) {
	got, err := Admit([]byte(drySpec), "acme")
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	// Remove the entity input by running against a fixture-less name variant:
	// craft a spec whose fixture omits the entity rows.
	if _, err := DryRun(got, "empty", "f"); err != nil {
		t.Fatalf("empty fixture should run (empty rows): %v", err)
	}
	sparse := `{"app":{"id":"d","slug":"d","name":"D","appVersion":"1.0.0","owner":"acme"},
		"data":{"entities":[{"fields":[{"name":"id","type":"string"}],"name":"e","source":{"capability":"o.p","entity":"es"}}],"queries":[{"entity":"e","name":"q"}]},
		"ui":{"pages":[{"components":[],"id":"h","path":"/","title":"H"}]},
		"workflows":[{"id":"f","trigger":{"type":"manual"},"steps":[{"id":"g","type":"branch","condition":"${q.rows}"},{"id":"d","type":"finish","message":"done"}]}],
		"capabilities":{"operations":[{"connection":"c","effect":"read","id":"o.p","method":"GET","paths":["/es"]}]},
		"policies":{"confirmation":{"externalWrites":"disabled"}},
		"fixtures":[{"name":"sparse","input":{}}],
		"metadata":{"schemaVersion":"miniappspec/v1"}}`
	sp, err := Admit([]byte(sparse), "acme")
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	_, err = DryRun(sp, "sparse", "f")
	if err == nil {
		t.Fatal("missing fixture input accepted, want actionable error")
	}
	if got := err.Error(); !strings.Contains(got, "sparse") || !strings.Contains(got, `"e"`) {
		t.Errorf("error %q does not name fixture and key", got)
	}
}

func TestDryRunBranchEarlyEnd(t *testing.T) {
	got, err := Admit([]byte(drySpec), "acme")
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	res, err := DryRun(got, "empty", "f")
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if res.Completed {
		t.Error("empty guard completed, want early end")
	}
	if _, ok := res.Outputs["touched"]; ok {
		t.Error("post-guard step executed after falsy guard")
	}
	res, err = DryRun(got, "full", "f")
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if !res.Completed {
		t.Error("truthy guard ended early, want completion")
	}
	if touched, ok := res.Outputs["touched"].([]any); !ok || len(touched) != 2 {
		t.Errorf("touched = %v, want 2 rows", res.Outputs["touched"])
	}
}

func TestDryRunDeterministic(t *testing.T) {
	vspec := admitValid(t)
	first, err := DryRun(vspec, "one-pending-invoice", "prepare_review")
	if err != nil {
		t.Fatal(err)
	}
	second, err := DryRun(vspec, "one-pending-invoice", "prepare_review")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Error("dry-run results differ across runs")
	}
}

// TestDryRunSealed proves the live read path is unreachable during dry-run:
// the guard refuses even before touching dependencies.
func TestDryRunSealed(t *testing.T) {
	vspec := admitValid(t)
	parsed, err := vspec.Spec()
	if err != nil {
		t.Fatal(err)
	}
	r := &run{spec: parsed, fixture: &parsed.Fixtures[0], fixName: parsed.Fixtures[0].Name}
	if _, err := r.readOp(context.Background(), "documents.invoice.read", "/files"); err == nil {
		t.Error("live read path reachable during dry-run, want refusal")
	}
}
