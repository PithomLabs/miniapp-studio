package spec

import (
	"encoding/json"
	"testing"
)

// validSkeleton returns a minimal spec that validates cleanly; tests mutate
// one field to prove the matrix rejects it.
func validSkeleton() MiniappSpec {
	return MiniappSpec{
		App: AppSection{
			ID: "demo", Slug: "demo", Name: "Demo",
			AppVersion: "1.0.0", Owner: "acme",
		},
		Data: DataSection{
			Entities: []Entity{{
				Name:   "things",
				Fields: []Field{{Name: "id", Type: "string"}},
				Source: DataSource{Capability: "docs.list", Entity: "t"},
			}},
			Queries: []Query{{Name: "all", Entity: "things"}},
		},
		UI: UISection{
			Pages: []Page{{
				ID: "home", Title: "Home", Path: "/",
				Components: []Component{
					{Type: "table", Title: "T", Binding: "all", Columns: []string{"id"}},
				},
			}},
		},
		Workflows: []Workflow{{
			ID:      "flow",
			Trigger: Trigger{Type: "manual"},
			Steps: []Step{{
				ID: "read", Type: "connector-read",
				Operation: "docs.list", Path: "/files", Output: "out",
			}},
		}},
		Capabilities: CapabilitiesSection{
			Operations: []CapabilityOp{{
				ID: "docs.list", Connection: "conn_1",
				Method: "GET", Paths: []string{"/files"}, Effect: "read",
			}},
		},
		Policies: PoliciesSection{
			Confirmation: ConfirmPolicy{ExternalWrites: "disabled"},
		},
		Fixtures: []Fixture{{Name: "f1"}},
		Metadata: MetadataSection{SchemaVersion: SchemaVersion},
	}
}

func validateStruct(t *testing.T, spec MiniappSpec) []Error {
	t.Helper()
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	_, errs := Validate(raw)
	return errs
}

func hasCode(errs []Error, code Code) bool {
	for _, e := range errs {
		if e.Code == code {
			return true
		}
	}
	return false
}

func TestSkeletonValid(t *testing.T) {
	if errs := validateStruct(t, validSkeleton()); len(errs) != 0 {
		t.Fatalf("skeleton rejected: %v", errs)
	}
}

// TestStepIrrelevantFieldsRejected proves the per-step-type field matrix:
// fields that do not belong to the declared step type are rejected.
func TestStepIrrelevantFieldsRejected(t *testing.T) {
	mutate := map[string]func(*Step){
		"transform+operation": func(s *Step) {
			s.Type, s.Operation = "transform", "docs.list"
			s.Input = map[string]string{"v": "${all}"}
			s.Output = "t"
		},
		"transform+path": func(s *Step) {
			s.Type, s.Path = "transform", "/files"
			s.Input = map[string]string{"v": "${all}"}
			s.Output = "t"
		},
		"transform+condition": func(s *Step) {
			s.Type, s.Condition = "transform", "${all}"
			s.Input = map[string]string{"v": "${all}"}
			s.Output = "t"
		},
		"transform+message": func(s *Step) {
			s.Type, s.Message = "transform", "hi"
			s.Input = map[string]string{"v": "${all}"}
			s.Output = "t"
		},
		"transform+maxIterations": func(s *Step) {
			s.Type, s.MaxIterations = "transform", 10
			s.Input = map[string]string{"v": "${all}"}
			s.Output = "t"
		},
		"branch+operation": func(s *Step) {
			s.Type, s.Operation, s.Condition = "branch", "docs.list", "${all}"
		},
		"bounded-loop+operation": func(s *Step) {
			s.Type, s.Operation = "bounded-loop", "docs.list"
			s.Input = map[string]string{"v": "${all}"}
			s.Output, s.MaxIterations = "t", 10
		},
		"notify+operation": func(s *Step) {
			s.Type, s.Operation, s.Message = "notify", "docs.list", "hi"
		},
		"connector-read+message": func(s *Step) {
			s.Message = "hi"
		},
		"connector-read+condition": func(s *Step) {
			s.Condition = "${all}"
		},
		"finish+maxIterations": func(s *Step) {
			s.Type, s.MaxIterations, s.Message = "finish", 3, "done"
		},
		"confirm+path": func(s *Step) {
			s.Type, s.Path, s.Message = "confirm", "/files", "ok?"
		},
	}
	for name, fn := range mutate {
		t.Run(name, func(t *testing.T) {
			spec := validSkeleton()
			fn(&spec.Workflows[0].Steps[0])
			errs := validateStruct(t, spec)
			if !hasCode(errs, EWorkflow) {
				t.Errorf("irrelevant field accepted, want E_WORKFLOW, got %v", errs)
			}
		})
	}
}

// TestComponentIrrelevantFieldsRejected proves the per-component-type field
// matrix: fields that do not belong to the declared component type are rejected.
func TestComponentIrrelevantFieldsRejected(t *testing.T) {
	mutate := map[string]func(*Component){
		"form+binding": func(c *Component) {
			c.Type, c.Binding, c.Action = "form", "ghost_query", "flow"
		},
		"form+columns": func(c *Component) {
			c.Type, c.Columns, c.Action = "form", []string{"id"}, "flow"
		},
		"action-button+binding": func(c *Component) {
			c.Type, c.Binding, c.Action = "action-button", "all", "flow"
		},
		"table+action": func(c *Component) {
			c.Action = "flow"
		},
		"metric-card+action": func(c *Component) {
			c.Type, c.Binding, c.Action = "metric-card", "all", "flow"
		},
		"metric-card+columns": func(c *Component) {
			c.Type, c.Binding, c.Columns = "metric-card", "all", []string{"id"}
		},
		"record-detail+action": func(c *Component) {
			c.Type, c.Binding, c.Action = "record-detail", "all", "flow"
		},
		"record-detail+columns": func(c *Component) {
			c.Type, c.Binding, c.Columns = "record-detail", "all", []string{"id"}
		},
	}
	for name, fn := range mutate {
		t.Run(name, func(t *testing.T) {
			spec := validSkeleton()
			fn(&spec.UI.Pages[0].Components[0])
			errs := validateStruct(t, spec)
			if !hasCode(errs, EComponent) {
				t.Errorf("irrelevant field accepted, want E_COMPONENT, got %v", errs)
			}
		})
	}
}

// TestComponentFieldsResolved proves columns and detail fields resolve
// against the bound entity, and unknown names fail with E_BINDING.
func TestComponentFieldsResolved(t *testing.T) {
	good := validSkeleton()
	good.UI.Pages[0].Components[0].Type = "record-detail"
	good.UI.Pages[0].Components[0].Binding = "all"
	good.UI.Pages[0].Components[0].Columns = nil
	good.UI.Pages[0].Components[0].Fields = []string{"id"}
	if errs := validateStruct(t, good); len(errs) != 0 {
		t.Fatalf("resolved fields rejected: %v", errs)
	}
	bad := validSkeleton()
	bad.UI.Pages[0].Components[0].Columns = []string{"nope_not_a_field"}
	if errs := validateStruct(t, bad); !hasCode(errs, EBinding) {
		t.Errorf("unknown column accepted, want E_BINDING, got %v", errs)
	}
}

// TestBindingSubPathResolved proves binding sub-paths resolve their first
// segment against the bound entity.
func TestBindingSubPathResolved(t *testing.T) {
	good := validSkeleton()
	good.UI.Pages[0].Components[0].Binding = "all.id"
	if errs := validateStruct(t, good); len(errs) != 0 {
		t.Fatalf("resolved sub-path rejected: %v", errs)
	}
	bad := validSkeleton()
	bad.UI.Pages[0].Components[0].Binding = "all.nope_not_a_field"
	if errs := validateStruct(t, bad); !hasCode(errs, EBinding) {
		t.Errorf("unknown sub-path accepted, want E_BINDING, got %v", errs)
	}
}

// TestDuplicateStepOutputRejected proves step outputs are unique per workflow.
func TestDuplicateStepOutputRejected(t *testing.T) {
	spec := validSkeleton()
	spec.Workflows[0].Steps = append(spec.Workflows[0].Steps, Step{
		ID: "dup", Type: "transform",
		Input: map[string]string{"v": "${all}"}, Output: "out",
	})
	if errs := validateStruct(t, spec); !hasCode(errs, EWorkflow) {
		t.Errorf("duplicate output accepted, want E_WORKFLOW, got %v", errs)
	}
}
