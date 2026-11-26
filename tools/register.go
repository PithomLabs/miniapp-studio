package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/registry"
)

// profileDoc is the deployment profile the manifest binds by digest.
const profileDoc = `{"name":"default","placement":"machine","substrate":"go"}`

func profileDigest() contract.Digest {
	sum := sha256.Sum256([]byte(profileDoc))
	return contract.Digest("sha256:" + hex.EncodeToString(sum[:]))
}

type toolDef struct {
	name        string
	description string
	entityKey   string
	input       string
}

var toolDefs = []toolDef{
	{ToolSpecValidate, "Validate a MiniappSpec and bind its content hash.", "",
		`{"type":"object","properties":{"spec":{"type":"object"}},"required":["spec"],"additionalProperties":false}`},
	{ToolAppCreate, "Register one application.", "slug",
		`{"type":"object","properties":{"slug":{"type":"string","maxLength":100},"name":{"type":"string","maxLength":256},"description":{"type":"string","maxLength":4096}},"required":["slug","name"],"additionalProperties":false}`},
	{ToolAppList, "List tenant applications with current versions.", "",
		`{"type":"object","properties":{"limit":{"type":"integer"},"offset":{"type":"integer"}},"additionalProperties":false}`},
	{ToolAppGet, "Read one application with its current version.", "app_id",
		`{"type":"object","properties":{"app_id":{"type":"string","maxLength":128}},"required":["app_id"],"additionalProperties":false}`},
	{ToolAppUpdate, "Update application metadata.", "app_id",
		`{"type":"object","properties":{"app_id":{"type":"string","maxLength":128},"name":{"type":"string","maxLength":256},"description":{"type":"string","maxLength":4096}},"required":["app_id"],"additionalProperties":false}`},
	{ToolSpecSave, "Validate and store one immutable spec version.", "app_id",
		`{"type":"object","properties":{"app_id":{"type":"string","maxLength":128},"spec":{"type":"object"}},"required":["app_id","spec"],"additionalProperties":false}`},
	{ToolSpecGet, "Read one spec version (latest by default).", "app_id",
		`{"type":"object","properties":{"app_id":{"type":"string","maxLength":128},"version":{"type":"integer"}},"required":["app_id"],"additionalProperties":false}`},
	{ToolDeployProvision, "Provision the Fly target and record its URL.", "app_id",
		`{"type":"object","properties":{"app_id":{"type":"string","maxLength":128}},"required":["app_id"],"additionalProperties":false}`},
	{ToolDeployStatus, "Read deployment status with live reconciliation.", "app_id",
		`{"type":"object","properties":{"app_id":{"type":"string","maxLength":128}},"required":["app_id"],"additionalProperties":false}`},
	{ToolDeployRetire, "Delete the Fly target and close the deployment.", "app_id",
		`{"type":"object","properties":{"app_id":{"type":"string","maxLength":128}},"required":["app_id"],"additionalProperties":false}`},
	{ToolConnectorsList, "List configured org connections or report unconfigured.", "",
		`{"type":"object","properties":{"provider":{"type":"string","maxLength":64}},"additionalProperties":false}`},
	{ToolConnectorsAttest, "Verify and attest a Custom API connection.", "",
		`{"type":"object","properties":{"id":{"type":"string","maxLength":128},"provider":{"type":"string","maxLength":64},"type":{"type":"string","maxLength":32}},"required":["id","provider","type"],"additionalProperties":false}`},
	{ToolActivityList, "List tenant audit events with redaction.", "",
		`{"type":"object","properties":{"limit":{"type":"integer"},"cursor":{"type":"string","maxLength":32},"exclude":{"type":"array","items":{"type":"string","maxLength":128},"maxItems":16}},"additionalProperties":false}`},
	{ToolConversationCreate, "Open one conversation.", "",
		`{"type":"object","properties":{"title":{"type":"string","maxLength":256},"application_id":{"type":"string","maxLength":128}},"additionalProperties":false}`},
	{ToolConversationGet, "Read one conversation with its builds and open questions.", "conversation_id",
		`{"type":"object","properties":{"conversation_id":{"type":"string","maxLength":128}},"required":["conversation_id"],"additionalProperties":false}`},
	{ToolConversationList, "List tenant conversations, newest first.", "",
		`{"type":"object","properties":{"limit":{"type":"integer"}},"additionalProperties":false}`},
	{ToolConversationStream, "Page the unified conversation stream after a cursor.", "conversation_id",
		`{"type":"object","properties":{"conversation_id":{"type":"string","maxLength":128},"after_seq":{"type":"integer"},"limit":{"type":"integer"}},"required":["conversation_id"],"additionalProperties":false}`},
	{ToolMessageSend, "Append one user message and ensure an open build, atomically.", "conversation_id",
		`{"type":"object","properties":{"conversation_id":{"type":"string","maxLength":128},"body":{"type":"string","maxLength":65536}},"required":["conversation_id","body"],"additionalProperties":false}`},
	{ToolBuildCreate, "Open one build on a conversation with turn 1.", "conversation_id",
		`{"type":"object","properties":{"conversation_id":{"type":"string","maxLength":128},"title":{"type":"string","maxLength":256},"application_id":{"type":"string","maxLength":128},"model":{"type":"string","maxLength":128}},"required":["conversation_id"],"additionalProperties":false}`},
	{ToolBuildProvision, "Provision one Builder Sprite and install the gateway.", "build_id",
		`{"type":"object","properties":{"build_id":{"type":"string","maxLength":128},"studio_url":{"type":"string","maxLength":512}},"required":["build_id","studio_url"],"additionalProperties":false}`},
	{ToolBuildUpgrade, "Replace the builder gateway binary after verifying all builds idle (force overrides).", "build_id",
		`{"type":"object","properties":{"build_id":{"type":"string","maxLength":128},"studio_url":{"type":"string","maxLength":512},"force":{"type":"boolean"}},"required":["build_id","studio_url"],"additionalProperties":false}`},
	{ToolBuildStatus, "Read one build with its current turn and open questions.", "build_id",
		`{"type":"object","properties":{"build_id":{"type":"string","maxLength":128}},"required":["build_id"],"additionalProperties":false}`},
	{ToolBuildCancel, "Cancel one build, closing turns and withdrawing questions.", "build_id",
		`{"type":"object","properties":{"build_id":{"type":"string","maxLength":128}},"required":["build_id"],"additionalProperties":false}`},
	{ToolBuildPacket, "Assemble the versioned B0 execution packet for one build.", "build_id",
		`{"type":"object","properties":{"build_id":{"type":"string","maxLength":128}},"required":["build_id"],"additionalProperties":false}`},
	{ToolBuildIngestEvent, "Ingest one gateway event with state-guarded transitions.", "build_id",
		`{"type":"object","properties":{"build_id":{"type":"string","maxLength":128},"event_id":{"type":"string","maxLength":128},"type":{"type":"string","maxLength":64},"seq":{"type":"integer"},"payload":{"type":"object"}},"required":["build_id","event_id","type","payload"],"additionalProperties":false}`},
	{ToolQuestionAnswer, "Answer one open question and resume its build.", "question_id",
		`{"type":"object","properties":{"question_id":{"type":"string","maxLength":128},"answer":{"type":"string","maxLength":65536}},"required":["question_id","answer"],"additionalProperties":false}`},
	{ToolArtifactVerify, "Deterministically verify one bundle artifact without executing code.", "artifact_id",
		`{"type":"object","properties":{"artifact_id":{"type":"string","maxLength":128}},"required":["artifact_id"],"additionalProperties":false}`},
	{ToolArtifactStatus, "Read artifact review state for one conversation.", "conversation_id",
		`{"type":"object","properties":{"conversation_id":{"type":"string","maxLength":128}},"required":["conversation_id"],"additionalProperties":false}`},
}

var outputSchema = json.RawMessage(`{"type":"object"}`)

// Manifest returns the versioned artifact of record.
func Manifest() *contract.MiniappManifest {
	tools := make([]contract.ToolManifest, 0, len(toolDefs))
	deploy := map[string]contract.ToolDeployment{}
	for _, def := range toolDefs {
		tm := contract.ToolManifest{
			Name:         contract.ToolName(def.name),
			Description:  def.description,
			Class:        contract.ClassA,
			Substrate:    contract.SubstrateGo,
			InputSchema:  json.RawMessage(def.input),
			OutputSchema: outputSchema,
			TrustModes:   []contract.TrustMode{contract.TrustAutonomous},
		}
		if def.entityKey != "" {
			tm.EntityKey = contract.EntityKey(def.entityKey)
		} else {
			tm.NoEntityWhy = "Read-only or tenant-scoped without per-resource serialization needs."
		}
		tools = append(tools, tm)
		deploy[def.name] = contract.ToolDeployment{Placement: contract.PlacementMachine,
			Profile: contract.ProfileRef{Name: "default", Digest: profileDigest()}}
	}
	return &contract.MiniappManifest{
		ID:          contract.Name(MiniappID),
		Version:     contract.Version(Version),
		Description: "Studio control-plane tools: specs, registry, deployments, connectors, activity, conversations, builds.",
		Tools:       tools,
		Deployment:  contract.DeploymentSection{Tools: deploy},
	}
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

// NewImplementation binds tool handlers with dependencies.
func NewImplementation(deps Deps) *registry.Implementation {
	return &registry.Implementation{Build: Build, ClassATools: map[string]registry.ToolFunc{
		ToolSpecValidate:       deps.specValidate,
		ToolAppCreate:          deps.appCreate,
		ToolAppList:            deps.appList,
		ToolAppGet:             deps.appGet,
		ToolAppUpdate:          deps.appUpdate,
		ToolSpecSave:           deps.specSave,
		ToolSpecGet:            deps.specGet,
		ToolDeployProvision:    deps.deployProvision,
		ToolDeployStatus:       deps.deployStatus,
		ToolDeployRetire:       deps.deployRetire,
		ToolConnectorsList:     deps.connectorsList,
		ToolConnectorsAttest:   deps.connectorsCustomAttest,
		ToolActivityList:       deps.activityList,
		ToolConversationCreate: deps.conversationCreate,
		ToolConversationGet:    deps.conversationGet,
		ToolConversationList:   deps.conversationList,
		ToolConversationStream: deps.conversationStream,
		ToolMessageSend:        deps.messageSend,
		ToolBuildCreate:        deps.buildCreate,
		ToolBuildProvision:     deps.buildProvision,
		ToolBuildUpgrade:       deps.buildUpgrade,
		ToolBuildStatus:        deps.buildStatus,
		ToolBuildCancel:        deps.buildCancel,
		ToolBuildPacket:        deps.buildPacket,
		ToolBuildIngestEvent:   deps.buildIngestEvent,
		ToolQuestionAnswer:     deps.questionAnswer,
		ToolArtifactVerify:     deps.artifactVerify,
		ToolArtifactStatus:     deps.artifactStatus,
	}}
}

// Register admits the Studio tools to reg.
func Register(reg *registry.Registry, deps Deps) error {
	env, err := Envelope()
	if err != nil {
		return err
	}
	return reg.Register(Manifest(), NewImplementation(deps), env)
}
