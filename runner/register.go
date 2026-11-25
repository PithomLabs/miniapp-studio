package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/registry"
)

// Identity of the generic runner miniapp: one static registration
// interprets every user-created MiniappSpec as data.
const (
	ID      = "studio-runner"
	Version = "1.0.0"
	// Build identifies this development implementation. Production
	// releases stamp the OCI artifact digest here instead.
	Build = "studio-dev"
	// ToolInvoke runs one workflow of an admitted spec.
	ToolInvoke = "runner.invoke"
)

// profileDoc is the deployment profile the manifest binds by digest.
// Phase 2 serves the runner in-process on the host machine.
const profileDoc = `{"name":"default","placement":"machine","substrate":"go"}`

func profileDigest() contract.Digest {
	sum := sha256.Sum256([]byte(profileDoc))
	return contract.Digest("sha256:" + hex.EncodeToString(sum[:]))
}

var inputSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"spec": {"type": "object", "description": "MiniappSpec document; validated at admission."},
		"workflow": {"type": "string", "description": "Workflow ID to execute."}
	},
	"required": ["spec", "workflow"]
}`)

var outputSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"specHash": {"type": "string"},
		"workflowId": {"type": "string"},
		"outputs": {"type": "object"},
		"messages": {"type": "array"},
		"notifications": {"type": "array"},
		"completed": {"type": "boolean"}
	},
	"required": ["specHash", "workflowId", "outputs", "completed"]
}`)

// Manifest returns the versioned artifact of record.
func Manifest() *contract.MiniappManifest {
	return &contract.MiniappManifest{
		ID:          contract.Name(ID),
		Version:     contract.Version(Version),
		Description: "Generic MiniappSpec interpreter: runs one workflow of an admitted spec. Read-only in v1.",
		Tools: []contract.ToolManifest{
			{
				Name:         contract.ToolName(ToolInvoke),
				Description:  "Execute one workflow of a validated MiniappSpec.",
				Class:        contract.ClassA,
				Substrate:    contract.SubstrateGo,
				InputSchema:  inputSchema,
				OutputSchema: outputSchema,
				TrustModes:   []contract.TrustMode{contract.TrustAutonomous},
				NoEntityWhy:  "Tenant scoping is enforced by admission, not entity locks.",
			},
		},
		Deployment: contract.DeploymentSection{Tools: map[string]contract.ToolDeployment{
			ToolInvoke: {Placement: contract.PlacementMachine,
				Profile: contract.ProfileRef{Name: "default", Digest: profileDigest()}},
		}},
	}
}

type invokeInput struct {
	Spec     json.RawMessage `json:"spec"`
	Workflow string          `json:"workflow"`
}

// Envelope binds this manifest digest to this implementation.
func Envelope() (contract.ReleaseEnvelope, error) {
	digest, err := contract.ProfileDigest(Manifest())
	if err != nil {
		return contract.ReleaseEnvelope{}, err
	}
	artifact, err := contract.ProfileDigest(Build)
	if err != nil {
		return contract.ReleaseEnvelope{}, err
	}
	return contract.ReleaseEnvelope{
		Manifest:       digest,
		Implementation: Build,
		Build:          Build,
		SourceRevision: "working-tree",
		Artifact:       artifact,
	}, nil
}

// Register admits the runner to reg with deps. Production callers pass
// server-assembled dependencies; tests pass fakes.
func Register(reg *registry.Registry, deps Deps) error {
	env, err := Envelope()
	if err != nil {
		return err
	}
	return reg.Register(Manifest(), NewImplementation(deps), env)
}

// NewImplementation binds the interpreter handler with its dependencies.
// The returned implementation registers once; every invocation revalidates.
func NewImplementation(deps Deps) *registry.Implementation {
	return &registry.Implementation{Build: Build,
		ClassATools: map[string]registry.ToolFunc{ToolInvoke: func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
			var in invokeInput
			if err := json.Unmarshal(call.Input, &in); err != nil {
				return nil, fmt.Errorf("runner: invalid input: %w", err)
			}
			if in.Workflow == "" || len(in.Spec) == 0 {
				return nil, fmt.Errorf("runner: input requires spec and workflow")
			}
			// Tenant comes from the framework-resolved call, never the payload.
			admitted, err := Admit(in.Spec, call.TenantID)
			if err != nil {
				return nil, err
			}
			res, err := Execute(ctx, admitted, in.Workflow, deps)
			if err != nil {
				return nil, err
			}
			out, err := json.Marshal(res)
			if err != nil {
				return nil, fmt.Errorf("runner: encode result: %w", err)
			}
			return out, nil
		}}}
}
