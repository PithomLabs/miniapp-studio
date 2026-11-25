package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/runtime"

	"github.com/PithomLabs/miniapp-studio/fly"
)

// testStudioURL is the fake Studio base for provision/upgrade inputs.
const testStudioURL = "https://studio.example.test"

// scriptDeployedDigest declares the fake's deployed gateway binary
// identity so the refresh path's digest verification can pass.
func scriptDeployedDigest(t *testing.T, sprites *fly.FakeSpriteManager, sprite string, binary []byte) {
	t.Helper()
	sum := sha256.Sum256(binary)
	sprites.ExecOutputs[sprite+"\x00sha256sum\x00/srv/studio-gateway/studio-gateway"] = fly.ExecResult{
		Stdout: hex.EncodeToString(sum[:]) + "  /srv/studio-gateway/studio-gateway\n",
	}
}

// TestBuildProvisionPreservesHealthyService (F7): provisioning a second
// build on a healthy shared builder refreshes keys without touching the
// running service — no delete, no recreate, first build undisrupted.
func TestBuildProvisionPreservesHealthyService(t *testing.T) {
	h, sprites := setupProvision(t)
	conv := createConversation(t, h, "ps-c", toolCredA)
	buildA, _ := createBuild(t, h, conv, "ps-a", toolCredA)
	resA := invoke(t, h, ToolBuildProvision, map[string]any{"build_id": buildA, "studio_url": testStudioURL}, "ps-pa", toolCredA)
	name, _ := resA["sprite_name"].(string)
	deletes, creates := len(sprites.Deleted), len(sprites.Created)
	scriptDeployedDigest(t, sprites, name, []byte("fake-gateway-binary"))
	conv2 := createConversation(t, h, "ps-c2", toolCredA)
	buildB, _ := createBuild(t, h, conv2, "ps-b", toolCredA)
	resB := invoke(t, h, ToolBuildProvision, map[string]any{"build_id": buildB, "studio_url": testStudioURL}, "ps-pb", toolCredA)
	if resB["sprite_name"] != name {
		t.Fatalf("second build sprite = %v, want shared %q", resB["sprite_name"], name)
	}
	if len(sprites.Deleted) != deletes {
		t.Errorf("healthy service deleted: %v", sprites.Deleted)
	}
	if len(sprites.Created) != creates {
		t.Error("healthy service recreated; want create-if-absent only")
	}
	svc, ok := sprites.Services[name+"/studio-gateway"]
	if !ok || svc.Status != "running" {
		t.Errorf("service = %+v, want preserved running", svc)
	}
	// B's key material still lands (refresh without restart).
	if _, ok := sprites.Files[name+":/srv/studio-staging/keys-"+buildB[:12]+".json"]; !ok {
		t.Error("second build key file not staged")
	}
}

// TestBuildProvisionUnhealthyFailsClosed (F7/A2): an existing but
// unhealthy service is a diagnosis case, not a delete-and-hope case.
func TestBuildProvisionUnhealthyFailsClosed(t *testing.T) {
	h, sprites := setupProvision(t)
	conv := createConversation(t, h, "pu-c", toolCredA)
	buildID, _ := createBuild(t, h, conv, "pu-b", toolCredA)
	name := builderSpriteName(toolTenantA, "")
	sprites.Sprites[name] = fly.Sprite{Name: name, URL: "https://sprites.test/" + name, Auth: fly.URLAuthSprite}
	sprites.Services[name+"/studio-gateway"] = fly.ServiceState{Name: "studio-gateway", Status: "failed", Error: "oom"}
	err := invokeErr(t, h, ToolBuildProvision, mustJSON(t, map[string]any{"build_id": buildID, "studio_url": testStudioURL}), "pu-p", toolCredA)
	if err == nil {
		t.Fatal("provision over unhealthy service succeeded, want fail-closed")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unhealthy") && !strings.Contains(err.Error(), "failed") {
		t.Errorf("error = %q, want unhealthy-service diagnosis", err)
	}
	if len(sprites.Deleted) != 0 {
		t.Errorf("unhealthy service deleted: %v", sprites.Deleted)
	}
}

// TestBuildProvisionInspectionErrorFailsClosed (F7/A2): when service
// inspection itself errors, provision reports and touches nothing.
func TestBuildProvisionInspectionErrorFailsClosed(t *testing.T) {
	h, sprites := setupProvision(t)
	conv := createConversation(t, h, "pi-c", toolCredA)
	buildID, _ := createBuild(t, h, conv, "pi-b", toolCredA)
	name := builderSpriteName(toolTenantA, "")
	sprites.Sprites[name] = fly.Sprite{Name: name, URL: "https://sprites.test/" + name, Auth: fly.URLAuthSprite}
	sprites.Fail["GetService"] = errors.New("control plane down")
	err := invokeErr(t, h, ToolBuildProvision, mustJSON(t, map[string]any{"build_id": buildID, "studio_url": testStudioURL}), "pi-p", toolCredA)
	if err == nil {
		t.Fatal("provision with inspection error succeeded, want fail-closed")
	}
	if len(sprites.Deleted) != 0 {
		t.Errorf("service deleted after inspection error: %v", sprites.Deleted)
	}
}

// TestBuildProvisionDigestDriftRefuses (F7/M14): a running service whose
// binary differs from the staged one is an explicit upgrade, never a
// silent side effect of provisioning another build.
func TestBuildProvisionDigestDriftRefuses(t *testing.T) {
	h, sprites := setupProvision(t)
	conv := createConversation(t, h, "pd-c", toolCredA)
	buildID, _ := createBuild(t, h, conv, "pd-b", toolCredA)
	name := builderSpriteName(toolTenantA, "")
	sprites.Sprites[name] = fly.Sprite{Name: name, URL: "https://sprites.test/" + name, Auth: fly.URLAuthSprite}
	sprites.Services[name+"/studio-gateway"] = fly.ServiceState{Name: "studio-gateway", Status: "running", PID: 100}
	sprites.ExecOutputs[name+"\x00sha256sum\x00/srv/studio-gateway/studio-gateway"] = fly.ExecResult{
		Stdout: "0000000000000000000000000000000000000000000000000000000000000000  /srv/studio-gateway/studio-gateway\n",
	}
	err := invokeErr(t, h, ToolBuildProvision, mustJSON(t, map[string]any{"build_id": buildID, "studio_url": testStudioURL}), "pd-p", toolCredA)
	if err == nil {
		t.Fatal("provision over drifted binary succeeded, want explicit-upgrade refusal")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "upgrade") {
		t.Errorf("error = %q, want explicit-upgrade direction", err)
	}
	if len(sprites.Deleted) != 0 {
		t.Errorf("service deleted on drift: %v", sprites.Deleted)
	}
}

// TestBuildProvisionEchoesDigest (M14): the provision result records the
// deployed binary's SHA-256 (print + record, no migration).
func TestBuildProvisionEchoesDigest(t *testing.T) {
	h, _ := setupProvision(t)
	conv := createConversation(t, h, "pe-c", toolCredA)
	buildID, _ := createBuild(t, h, conv, "pe-b", toolCredA)
	res := invoke(t, h, ToolBuildProvision, map[string]any{"build_id": buildID, "studio_url": testStudioURL}, "pe-p", toolCredA)
	sum := sha256.Sum256([]byte("fake-gateway-binary"))
	if res["gateway_sha256"] != hex.EncodeToString(sum[:]) {
		t.Errorf("gateway_sha256 = %v, want digest of deployed bytes", res["gateway_sha256"])
	}
}

// countingSprites wraps a SpriteManager observing concurrent Exec
// overlap (the install long pole) for the advisory-lock proof.
type countingSprites struct {
	fly.SpriteManager
	delay    time.Duration
	inFlight atomic.Int32
	maxSeen  atomic.Int32
}

func (c *countingSprites) Exec(ctx context.Context, sprite string, cmd []string) (fly.ExecResult, error) {
	cur := c.inFlight.Add(1)
	for {
		max := c.maxSeen.Load()
		if cur <= max || c.maxSeen.CompareAndSwap(max, cur) {
			break
		}
	}
	defer c.inFlight.Add(-1)
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	return c.SpriteManager.Exec(ctx, sprite, cmd)
}

// TestBuildProvisionSerializesAcrossSessions (F7/A2): two genuinely
// concurrent provisions (independent DB sessions, as serve + operator
// CLI would hold) on one sprite serialize — the per-sprite advisory
// lock coordinates across processes, not just goroutines.
func TestBuildProvisionSerializesAcrossSessions(t *testing.T) {
	sprites := fly.NewFakeSpriteManager()
	counting := &countingSprites{SpriteManager: sprites, delay: 150 * time.Millisecond}
	mkDeps := func() Deps {
		return Deps{
			Sprites: counting, Connectors: fly.NewFakeConnectorManager(),
			GatewayBinary: []byte("fake-gateway-binary"), GatewayOrgSecret: []byte("test-org-secret-0123456789"),
		}
	}
	// NOTE: each setup truncates; set both harnesses up first, then
	// create rows, so both see all rows on their own pools.
	h1 := setupToolsDeps(t, mkDeps())
	h2 := setupToolsDeps(t, mkDeps())
	conv1 := createConversation(t, h1, "pl-c1", toolCredA)
	build1, _ := createBuild(t, h1, conv1, "pl-b1", toolCredA)
	conv2 := createConversation(t, h2, "pl-c2", toolCredA)
	build2, _ := createBuild(t, h2, conv2, "pl-b2", toolCredA)
	// Whoever attaches second refreshes against the matching digest.
	scriptDeployedDigest(t, sprites, builderSpriteName(toolTenantA, ""), []byte("fake-gateway-binary"))
	// Invoke directly (not via the Fatal-ing helper): goroutines must
	// report errors through the channel.
	provision := func(h harness, build, key string) error {
		raw, _ := json.Marshal(map[string]any{"build_id": build, "studio_url": testStudioURL})
		out, err := h.svc.Invoke(context.Background(), runtime.InvokeRequest{
			MiniappID: MiniappID, Version: Version, Tool: ToolBuildProvision,
			Input: raw, IdempotencyKey: key,
			Credential: toolCredA, RequestedTrust: contract.TrustAutonomous,
		})
		if err != nil {
			return err
		}
		if out.Status != contract.RunSucceeded {
			return errors.New("provision status " + string(out.Status))
		}
		return nil
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, tc := range []struct {
		h     harness
		build string
		key   string
	}{{h1, build1, "pl-p1"}, {h2, build2, "pl-p2"}} {
		wg.Add(1)
		go func(h harness, build, key string) {
			defer wg.Done()
			if err := provision(h, build, key); err != nil {
				errs <- err
			}
		}(tc.h, tc.build, tc.key)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent provision: %v", err)
	}
	if max := counting.maxSeen.Load(); max != 1 {
		t.Errorf("max concurrent installs = %d, want serialized 1", max)
	}
	if len(sprites.Sprites) != 1 {
		t.Errorf("sprites = %d, want 1 shared builder", len(sprites.Sprites))
	}
}
