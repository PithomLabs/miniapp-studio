package render

import (
	"os"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp-studio/runner"
	"github.com/PithomLabs/miniapp-studio/spec"
)

func admitFixture(t *testing.T) runner.ValidatedSpec {
	t.Helper()
	raw, err := os.ReadFile("../spec/testdata/valid_invoice_review.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := runner.Admit(raw, "acme-workspace")
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	return got
}

func invoiceData() MapData {
	return MapData{
		"pending": map[string]any{"rows": []any{
			map[string]any{"id": "inv-1", "vendor": "Acme", "amount": 100.0, "status": "pending"},
		}},
		"invoices": map[string]any{"rows": []any{
			map[string]any{"id": "inv-1", "vendor": "Acme", "amount": 100.0, "status": "pending"},
		}},
		"order_totals": map[string]any{"rows": []any{}, "total": 0.0},
	}
}

func TestRenderOverviewSuccess(t *testing.T) {
	html, err := Render(admitFixture(t), "acme-workspace", "overview",
		Options{Data: invoiceData(), InvokeEndpoint: "/invoke"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, want := range []string{"Awaiting review", "inv-1", "Acme", "<table>", "<svg", "1 records"} {
		if !strings.Contains(html, want) {
			t.Errorf("output lacks %q", want)
		}
	}
}

func TestRenderDetailPage(t *testing.T) {
	html, err := Render(admitFixture(t), "acme-workspace", "invoice_detail",
		Options{Data: invoiceData(), InvokeEndpoint: "/invoke"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, want := range []string{"record-detail", "Acme", `name="workflow" value="prepare_review"`, `action="/invoke"`} {
		if !strings.Contains(html, want) {
			t.Errorf("output lacks %q", want)
		}
	}
}

func TestRenderUnknownPage(t *testing.T) {
	if _, err := Render(admitFixture(t), "acme-workspace", "nope",
		Options{Data: invoiceData()}); err == nil {
		t.Error("unknown page rendered, want error")
	}
}

func TestRenderTenantEnforcement(t *testing.T) {
	vspec := admitFixture(t)
	if _, err := Render(vspec, "other-tenant", "overview", Options{Data: invoiceData()}); err == nil {
		t.Error("cross-tenant render accepted, want denial")
	}
	if _, err := Render(vspec, "", "overview", Options{Data: invoiceData()}); err == nil {
		t.Error("empty tenant accepted, want denial")
	}
}

func TestRenderEscapesContent(t *testing.T) {
	data := invoiceData()
	data["pending"] = map[string]any{"rows": []any{
		map[string]any{"id": `<script>alert(1)</script>`, "vendor": "x", "amount": 1.0, "status": "y"},
	}}
	html, err := Render(admitFixture(t), "acme-workspace", "overview", Options{Data: data})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Error("unescaped script in output")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("escaped script missing from output")
	}
}

func TestRenderEmptyAndErrorStates(t *testing.T) {
	data := invoiceData()
	data["pending"] = map[string]any{"rows": []any{}}
	html, err := Render(admitFixture(t), "acme-workspace", "overview", Options{Data: data})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(html, "st-empty") {
		t.Error("empty data did not render the empty state")
	}
	broken := MapData{}
	html, err = Render(admitFixture(t), "acme-workspace", "overview", Options{Data: broken})
	if err != nil {
		t.Fatalf("Render with failing data: %v", err)
	}
	if !strings.Contains(html, "st-error") {
		t.Error("resolution failure did not render the error state")
	}
}

func TestRenderActionsDisabledWithoutEndpoint(t *testing.T) {
	html, err := Render(admitFixture(t), "acme-workspace", "invoice_detail",
		Options{Data: invoiceData()})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(html, "<form") {
		t.Error("live form rendered without an invoke endpoint")
	}
	if !strings.Contains(html, "Action wiring pending") {
		t.Error("wiring note missing")
	}
}

func TestRenderRunResult(t *testing.T) {
	// Nil run renders the empty state.
	html, err := Render(admitFixture(t), "acme-workspace", "overview",
		Options{Data: invoiceData()})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	_ = html
	r := &renderer{spec: mustSpec(t), opts: Options{Run: &runner.Result{
		SpecHash: "abc123", WorkflowID: "prepare_review",
		Messages: []string{"done <ok>"}, Completed: true,
	}}}
	out, err := r.runResult(mustComponent(t, "run-result"))
	if err != nil {
		t.Fatalf("runResult: %v", err)
	}
	for _, want := range []string{"prepare_review", "completed", "abc123", "done &lt;ok&gt;"} {
		if !strings.Contains(out, want) {
			t.Errorf("run output lacks %q", want)
		}
	}
	empty, err := (&renderer{opts: Options{}}).runResult(mustComponent(t, "run-result"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(empty, "st-empty") {
		t.Error("nil run did not render the empty state")
	}
}

func TestRenderPermissionsHonest(t *testing.T) {
	r := &renderer{spec: mustSpec(t), opts: Options{Data: invoiceData()}}
	out, err := r.permissionsCard(mustComponent(t, "permissions-card"))
	if err != nil {
		t.Fatalf("permissionsCard: %v", err)
	}
	for _, want := range []string{"documents.invoice.read", "GET", "/files/*", "Declared by this spec"} {
		if !strings.Contains(out, want) {
			t.Errorf("permissions output lacks %q", want)
		}
	}
}

func TestRenderUnknownComponentFailsClosed(t *testing.T) {
	r := &renderer{spec: mustSpec(t), opts: Options{Data: invoiceData()}}
	c := mustComponent(t, "quantum-portal")
	if _, err := r.component(c); err == nil {
		t.Error("unknown component rendered, want error")
	}
}

func TestSkeleton(t *testing.T) {
	html, err := Skeleton(admitFixture(t), "acme-workspace", "overview")
	if err != nil {
		t.Fatalf("Skeleton: %v", err)
	}
	if !strings.Contains(html, "st-loading") || !strings.Contains(html, "aria-busy") {
		t.Error("skeleton lacks loading markup")
	}
	if _, err := Skeleton(admitFixture(t), "other-tenant", "overview"); err == nil {
		t.Error("cross-tenant skeleton accepted, want denial")
	}
}

func mustSpec(t *testing.T) *spec.MiniappSpec {
	t.Helper()
	vspec := admitFixture(t)
	s, err := vspec.Spec()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func mustComponent(t *testing.T, typ string) *spec.Component {
	t.Helper()
	return &spec.Component{Type: typ, Title: "T", Binding: "pending"}
}
