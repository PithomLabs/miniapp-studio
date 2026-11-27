package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/stack"

	"github.com/PithomLabs/miniapp-studio/fly"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/testdb"
)

const (
	toolTenantA = "t-tools-a"
	toolTenantB = "t-tools-b"
	toolTenantC = "t-tools-c"
	toolCredA   = "cred-tools-a"
	toolCredB   = "cred-tools-b"
	toolCredC   = "cred-tools-c"
)

type harness struct {
	svc     *runtime.Service
	owner   *pgxpool.Pool
	sprites *fly.FakeSpriteManager
	conns   *fly.FakeConnectorManager
}

// setupTools assembles a real Service with Studio tools registered. Pass nil
// managers to exercise unconfigured production paths.
func setupTools(t *testing.T, sprites fly.SpriteManager, conns fly.ConnectorManager) harness {
	t.Helper()
	return setupToolsDeps(t, Deps{Sprites: sprites, Connectors: conns})
}

// setupToolsDeps assembles like setupTools with full dependency control
// (gateway binary injection for provision tests).
func setupToolsDeps(t *testing.T, deps Deps) harness {
	t.Helper()
	ctx := context.Background()
	// U5.12: isolated run database; the live role is never touched.
	dsn := testdb.RunDSN(t)
	if err := store.Migrate(ctx, dsn, testdb.Policy(t)); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	owner, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(owner.Close)
	for _, stmt := range []string{
		`TRUNCATE app.invocations, app.audit_events`,
		`TRUNCATE studio.sessions, studio.operators, studio.apps, studio.app_specs, studio.deployments, studio.runs`,
		`TRUNCATE studio.conversations, studio.messages, studio.builds, studio.turns, studio.questions, studio.commands, studio.events, studio.conversation_stream, studio.artifacts, studio.approvals, studio.artifact_blobs, studio.artifact_verifications, studio.artifact_test_runs, studio.checkpoints`,
	} {
		if _, err := owner.Exec(ctx, stmt); err != nil {
			t.Fatalf("truncate: %v", err)
		}
	}
	binding := func(agent, tenant string) auth.Binding {
		return auth.Binding{AgentID: agent, Authority: contract.Authority{
			Kind: contract.AuthorityUser, ID: agent, TenantID: tenant,
		}}
	}
	policy := auth.NewPolicy(
		map[string]auth.Binding{
			toolCredA: binding("agent-a", toolTenantA),
			toolCredB: binding("agent-b", toolTenantB),
			toolCredC: binding("agent-c", toolTenantC),
		},
		map[string]map[string]contract.TrustMode{
			toolTenantA: {MiniappID: contract.TrustAutonomous},
			toolTenantB: {MiniappID: contract.TrustAutonomous},
		},
	)
	st := stack.Assemble(t, stack.Config{
		// The Service shares the run database: tool writes and the
		// harness owner pool observe the same rows. Never
		// testkit.NewDB(t) here — it follows the environment to the
		// bootstrap database, outside test isolation.
		DB:     &testkit.DB{Pool: owner, DSN: dsn},
		Policy: policy,
		Register: func(t *testing.T, reg *registry.Registry) {
			if err := Register(reg, deps); err != nil {
				t.Fatalf("register tools: %v", err)
			}
		},
	})
	var fakeSprites *fly.FakeSpriteManager
	var fakeConns *fly.FakeConnectorManager
	if s, ok := deps.Sprites.(*fly.FakeSpriteManager); ok {
		fakeSprites = s
	}
	if c, ok := deps.Connectors.(*fly.FakeConnectorManager); ok {
		fakeConns = c
	}
	return harness{svc: st.Svc, owner: owner, sprites: fakeSprites, conns: fakeConns}
}

func setupFakes(t *testing.T) harness {
	t.Helper()
	return setupTools(t, fly.NewFakeSpriteManager(), fly.NewFakeConnectorManager())
}

// invoke calls one tool and decodes a successful result.
func invoke(t *testing.T, h harness, tool string, input any, key, cred string) map[string]any {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	out, err := h.svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: MiniappID, Version: Version, Tool: tool,
		Input: raw, IdempotencyKey: key,
		Credential: cred, RequestedTrust: contract.TrustAutonomous,
	})
	if err != nil {
		t.Fatalf("invoke %s: %v", tool, err)
	}
	if out.Status != contract.RunSucceeded {
		t.Fatalf("invoke %s: status %s", tool, out.Status)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out.Result, &decoded); err != nil {
		t.Fatalf("decode %s result: %v", tool, err)
	}
	return decoded
}

func invokeErr(t *testing.T, h harness, tool string, raw json.RawMessage, key, cred string) error {
	t.Helper()
	_, err := h.svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: MiniappID, Version: Version, Tool: tool,
		Input: raw, IdempotencyKey: key,
		Credential: cred, RequestedTrust: contract.TrustAutonomous,
	})
	if err == nil {
		t.Fatalf("invoke %s: expected error, got success", tool)
	}
	return err
}

func auditCount(t *testing.T, h harness, tenant, tool string) int {
	t.Helper()
	var n int
	err := h.owner.QueryRow(context.Background(),
		`SELECT count(*) FROM app.audit_events WHERE tenant_id = $1 AND tool = $2`,
		tenant, tool).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

var allTools = []string{
	ToolSpecValidate, ToolAppCreate, ToolAppList, ToolAppGet, ToolAppUpdate,
	ToolSpecSave, ToolSpecGet, ToolDeployProvision, ToolDeployStatus,
	ToolDeployRetire, ToolConnectorsList, ToolConnectorsAttest, ToolActivityList,
	ToolConversationCreate, ToolConversationGet, ToolConversationList, ToolConversationStream,
	ToolMessageSend, ToolBuildCreate, ToolBuildProvision, ToolBuildUpgrade, ToolBuildStatus, ToolBuildCancel,
	ToolBuildPacket, ToolBuildIngestEvent, ToolQuestionAnswer, ToolArtifactVerify, ToolArtifactStatus,
}

// TestArtifactDecideNotRegistered pins N17: artifact approval is a
// human-only control-plane operation (server/artifact_decide.go). It
// must never appear in the agent-callable tool surface, and invoking
// the name must fail closed as an unknown tool.
func TestArtifactDecideNotRegistered(t *testing.T) {
	impl := NewImplementation(Deps{})
	for name := range impl.ClassATools {
		if name == "artifact.decide" {
			t.Fatal("artifact.decide is registered: human-only boundary violated")
		}
	}
	h := setupFakes(t)
	if err := invokeErr(t, h, "artifact.decide",
		mustJSON(t, map[string]any{"artifact_id": "a-x"}), "n17-k", toolCredA); err == nil {
		t.Error("artifact.decide invocation accepted, want unknown-tool refusal")
	}
}

// TestAllHandlersRequireTenant proves every handler refuses an empty tenant
// before touching storage. Handlers are invoked directly (unit level);
// routes can never produce this state because middleware guarantees identity.
func TestAllHandlersRequireTenant(t *testing.T) {
	impl := NewImplementation(Deps{})
	if len(impl.ClassATools) != len(allTools) {
		t.Fatalf("registered %d tools, want %d", len(impl.ClassATools), len(allTools))
	}
	for _, name := range allTools {
		fn, ok := impl.ClassATools[name]
		if !ok {
			t.Errorf("tool %s not registered", name)
			continue
		}
		if _, err := fn(context.Background(), registry.ToolCall{TenantID: ""}); err == nil {
			t.Errorf("tool %s accepted empty tenant, want refusal", name)
		}
	}
}

// TestAllToolsDenyUnauthorized proves credentials without grants are denied
// on every tool.
func TestAllToolsDenyUnauthorized(t *testing.T) {
	h := setupFakes(t)
	for _, name := range allTools {
		invokeErr(t, h, name, json.RawMessage(`{}`), "unauth-"+name, toolCredC)
	}
}

// TestAllToolsRejectMalformed proves malformed inputs fail closed on every
// tool, whether the framework or the handler reports it.
func TestAllToolsRejectMalformed(t *testing.T) {
	h := setupFakes(t)
	for _, name := range allTools {
		invokeErr(t, h, name, json.RawMessage(`"just-a-string"`), "malformed-"+name, toolCredA)
	}
}

func ctxForTest() context.Context {
	return context.Background()
}

func errFakeBoom() error {
	return errors.New("fake boom")
}
