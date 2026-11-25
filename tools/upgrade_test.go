package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp-studio/fly"
	"github.com/PithomLabs/miniapp-studio/gateway"
)

// upgradeHarness provisions one build on the fake sprite and stands up a
// test gateway server (KDF keys from the same org secret) that the
// upgrade tool consults for liveness.
func upgradeHarness(t *testing.T) (harness, *fly.FakeSpriteManager, string, string, *gateway.SessionManager) {
	t.Helper()
	h, sprites := setupProvision(t)
	conv := createConversation(t, h, "ug-c", toolCredA)
	buildID, _ := createBuild(t, h, conv, "ug-b", toolCredA)
	res := invoke(t, h, ToolBuildProvision, map[string]any{"build_id": buildID, "studio_url": testStudioURL}, "ug-p", toolCredA)
	name, _ := res["sprite_name"].(string)
	sessions, err := gateway.NewSessionManager(t.TempDir() + "/sessions")
	if err != nil {
		t.Fatal(err)
	}
	outbox, err := gateway.OpenOutbox(t.TempDir() + "/outbox.json")
	if err != nil {
		t.Fatal(err)
	}
	dedup, err := gateway.OpenDedupStore(t.TempDir() + "/dedup.json")
	if err != nil {
		t.Fatal(err)
	}
	adapter := gateway.NewFakeAdapter()
	gw := &gateway.Server{
		Keys:    gateway.KDFProvider{Secret: []byte("test-org-secret-0123456789")},
		Adapter: adapter, Sessions: sessions,
		Outbox: outbox, Dedup: dedup, WorkRoot: t.TempDir(),
	}
	httpSrv := httptest.NewServer(gw)
	t.Cleanup(httpSrv.Close)
	if _, err := h.owner.Exec(context.Background(),
		`UPDATE studio.builds SET gateway_url = $1 WHERE id = $2 AND tenant_id = $3`,
		httpSrv.URL, buildID, toolTenantA); err != nil {
		t.Fatal(err)
	}
	return h, sprites, name, buildID, sessions
}

func idleRecord(t *testing.T, sessions *gateway.SessionManager, build string) {
	t.Helper()
	if err := sessions.Create(build, gateway.SessionRecord{
		SessionRef: "sess-" + build, BuildID: build,
		State: gateway.SessionDone, Turn: 1,
	}); err != nil {
		t.Fatal(err)
	}
}

func liveRecord(t *testing.T, sessions *gateway.SessionManager, build string) {
	t.Helper()
	if err := sessions.Create(build, gateway.SessionRecord{
		SessionRef: "sess-" + build, BuildID: build,
		State: gateway.SessionRunning, Turn: 1,
	}); err != nil {
		t.Fatal(err)
	}
}

// TestBuildUpgradeIdleReplaces (F7): with every build idle, upgrade
// replaces the service and echoes the new digest.
func TestBuildUpgradeIdleReplaces(t *testing.T) {
	h, sprites, name, buildID, sessions := upgradeHarness(t)
	idleRecord(t, sessions, buildID)
	deletesBefore := len(sprites.Deleted)
	out := invoke(t, h, ToolBuildUpgrade, map[string]any{"build_id": buildID, "studio_url": testStudioURL}, "ug-u1", toolCredA)
	if out["upgraded"] != true {
		t.Fatalf("upgrade = %v, want upgraded", out)
	}
	sum := sha256.Sum256([]byte("fake-gateway-binary"))
	if out["gateway_sha256"] != hex.EncodeToString(sum[:]) {
		t.Errorf("gateway_sha256 = %v, want digest of deployed bytes", out["gateway_sha256"])
	}
	if out["idle_checked"] != float64(1) {
		t.Errorf("idle_checked = %v, want 1", out["idle_checked"])
	}
	if len(sprites.Deleted) != deletesBefore+1 {
		t.Errorf("deletes = %v, want exactly the replaced service", sprites.Deleted)
	}
	svc, ok := sprites.Services[name+"/studio-gateway"]
	if !ok || svc.Status != "running" {
		t.Errorf("service = %+v, want running replacement", svc)
	}
}

// TestBuildUpgradeLiveRefuses (F7): a live agent run blocks replacement
// unless force explicitly accepts the disruption.
func TestBuildUpgradeLiveRefuses(t *testing.T) {
	h, sprites, _, buildID, sessions := upgradeHarness(t)
	liveRecord(t, sessions, buildID)
	deletes := len(sprites.Deleted)
	err := invokeErr(t, h, ToolBuildUpgrade,
		mustJSON(t, map[string]any{"build_id": buildID, "studio_url": testStudioURL}), "ug-u2", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "live agent run") {
		t.Fatalf("err = %v, want live-run refusal", err)
	}
	if len(sprites.Deleted) != deletes {
		t.Errorf("service deleted despite live run: %v", sprites.Deleted)
	}
	out := invoke(t, h, ToolBuildUpgrade,
		map[string]any{"build_id": buildID, "force": true, "studio_url": testStudioURL}, "ug-u3", toolCredA)
	if out["upgraded"] != true || out["force"] != true {
		t.Fatalf("forced upgrade = %v", out)
	}
	if len(sprites.Deleted) != deletes+1 {
		t.Errorf("deletes = %v, want the forced replacement", sprites.Deleted)
	}
}

// TestBuildUpgradeRefusals (F7): unprovisioned builds and unconfigured
// environments refuse honestly.
func TestBuildUpgradeRefusals(t *testing.T) {
	h, _ := setupProvision(t)
	conv := createConversation(t, h, "ug-c2", toolCredA)
	buildID, _ := createBuild(t, h, conv, "ug-b2", toolCredA)
	err := invokeErr(t, h, ToolBuildUpgrade,
		mustJSON(t, map[string]any{"build_id": buildID, "studio_url": testStudioURL}), "ug-u4", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "not provisioned") {
		t.Fatalf("err = %v, want not-provisioned refusal", err)
	}
	bare := setupToolsDeps(t, Deps{})
	err = invokeErr(t, bare, ToolBuildUpgrade,
		mustJSON(t, map[string]any{"build_id": buildID, "studio_url": testStudioURL}), "ug-u5", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("err = %v, want unconfigured refusal", err)
	}
}
