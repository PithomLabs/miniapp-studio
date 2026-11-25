package runner

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/PithomLabs/miniapp-studio/spec"
)

// DryResult is one fixture-backed dry-run outcome. It mirrors Result with
// added provenance proving which fixture and spec hash produced it. DryRun
// is always true: dry-run output is never mistakable for a real invocation.
type DryResult struct {
	DryRun        bool           `json:"dryRun"`
	Fixture       string         `json:"fixture"`
	SpecHash      string         `json:"specHash"`
	WorkflowID    string         `json:"workflowId"`
	Outputs       map[string]any `json:"outputs"`
	Messages      []string       `json:"messages"`
	Notifications []string       `json:"notifications"`
	Completed     bool           `json:"completed"`
}

// DryRun executes one workflow against declared fixture inputs only. Entity
// rows resolve from fixture inputs keyed by entity name; connector-read step
// outputs resolve from fixture inputs keyed by output name; transforms,
// branches, loops, notifications, and finishes execute identically to live
// runs. No registry, read client, provider, gateway, or network path is
// reachable: the live read function refuses whenever a fixture is set, and
// DryRun supplies no dependencies at all.
func DryRun(vspec ValidatedSpec, fixture, workflowID string) (DryResult, error) {
	if err := vspec.Revalidate(); err != nil {
		return DryResult{}, err
	}
	parsed, err := vspec.Spec()
	if err != nil {
		return DryResult{}, err
	}
	var flow *spec.Workflow
	for i := range parsed.Workflows {
		if parsed.Workflows[i].ID == workflowID {
			flow = &parsed.Workflows[i]
		}
	}
	if flow == nil {
		return DryResult{}, fmt.Errorf("dry-run: unknown workflow %q", workflowID)
	}
	var fix *spec.Fixture
	for i := range parsed.Fixtures {
		if parsed.Fixtures[i].Name == fixture {
			fix = &parsed.Fixtures[i]
		}
	}
	if fix == nil {
		return DryResult{}, fmt.Errorf("dry-run: unknown fixture %q", fixture)
	}
	r := &run{spec: parsed, outputs: map[string]any{}, entities: map[string][]any{},
		fixture: fix, fixName: fixture}
	res := DryResult{DryRun: true, Fixture: fixture, SpecHash: vspec.Hash(),
		WorkflowID: workflowID, Outputs: map[string]any{}, Completed: true}
	inner := Result{Outputs: map[string]any{}}
	for i := range flow.Steps {
		step := &flow.Steps[i]
		outcome, err := r.execStep(context.Background(), step, &inner)
		if err != nil {
			return DryResult{}, fmt.Errorf("dry-run: step %q (%s): %w", step.ID, step.Type, err)
		}
		if outcome == stepFinish {
			break
		}
		if outcome == stepBranchEnd {
			res.Completed = false
			break
		}
		if step.Output != "" {
			if v, ok := r.outputs[step.Output]; ok {
				res.Outputs[step.Output] = v
			}
		}
	}
	res.Messages, res.Notifications = inner.Messages, inner.Notifications
	encoded, err := json.Marshal(res.Outputs)
	if err != nil {
		return DryResult{}, fmt.Errorf("dry-run: encode outputs: %w", err)
	}
	if len(encoded) > MaxResultBytes {
		return DryResult{}, fmt.Errorf("dry-run: outputs exceed %d bytes", MaxResultBytes)
	}
	return res, nil
}
