package tools

import (
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp-studio/fly"
)

func provisionedApp(t *testing.T, h harness, cred, tenant, slug, key string) string {
	t.Helper()
	res := invoke(t, h, ToolAppCreate, map[string]any{"slug": slug, "name": slug}, key+"-app", cred)
	id, _ := res["app_id"].(string)
	invoke(t, h, ToolSpecSave,
		map[string]any{"app_id": id, "spec": mustDecode(t, specDocForTenant(t, tenant))},
		key+"-spec", cred)
	return id
}

func TestDeployProvisionFlow(t *testing.T) {
	h := setupFakes(t)
	id := provisionedApp(t, h, toolCredA, toolTenantA, "shop", "p-1")
	res := invoke(t, h, ToolDeployProvision, map[string]any{"app_id": id}, "p-2", toolCredA)
	if res["status"] != "ready" {
		t.Fatalf("provision = %v", res)
	}
	if res["resource_name"] != "shop" {
		t.Errorf("resource = %v, want bare slug first", res)
	}
	url, _ := res["url"].(string)
	if url == "" {
		t.Error("no URL recorded")
	}
	// Idempotent attach: same result, still exactly one sprite.
	again := invoke(t, h, ToolDeployProvision, map[string]any{"app_id": id}, "p-3", toolCredA)
	if again["resource_name"] != "shop" || again["url"] != url {
		t.Errorf("re-provision = %v, want attach", again)
	}
	if n := len(h.sprites.Sprites); n != 1 {
		t.Errorf("sprites = %d, want 1", n)
	}
	if n := auditCount(t, h, toolTenantA, ToolDeployProvision); n == 0 {
		t.Error("no audit evidence for deploy.provision")
	}
}

// A record whose sprite vanished out of band must not attach: provision
// fails honestly naming the sprite, and status reconciles the record.
func TestStaleProvisioningRetry(t *testing.T) {
	h := setupFakes(t)
	id := provisionedApp(t, h, toolCredA, toolTenantA, "staleapp", "s-1")
	res := invoke(t, h, ToolDeployProvision, map[string]any{"app_id": id}, "s-2", toolCredA)
	name, _ := res["resource_name"].(string)
	if name == "" {
		t.Fatalf("provision = %v", res)
	}
	if err := h.sprites.DeleteSprite(ctxForTest(), name); err != nil {
		t.Fatal(err)
	}
	err := invokeErr(t, h, ToolDeployProvision,
		mustJSON(t, map[string]any{"app_id": id}), "s-3", toolCredA)
	if err == nil || !strings.Contains(err.Error(), name) {
		t.Fatalf("error = %v, want honest failure naming %q", err, name)
	}
	st := invoke(t, h, ToolDeployStatus, map[string]any{"app_id": id}, "s-4", toolCredA)
	if st["status"] != "failed" || st["error"] != "target missing" {
		t.Errorf("status = %v, want reconciled failed/target-missing", st)
	}
}

func TestDeployProvisionRequiresSpec(t *testing.T) {
	h := setupFakes(t)
	res := invoke(t, h, ToolAppCreate, map[string]any{"slug": "empty", "name": "e"}, "p-1", toolCredA)
	invokeErr(t, h, ToolDeployProvision,
		mustJSON(t, map[string]any{"app_id": res["app_id"]}), "p-2", toolCredA)
}

func TestDeployUnconfigured(t *testing.T) {
	h := setupTools(t, nil, nil)
	id := provisionedApp(t, h, toolCredA, toolTenantA, "shop", "u-1")
	err := invokeErr(t, h, ToolDeployProvision, mustJSON(t, map[string]any{"app_id": id}), "u-2", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Errorf("error = %v, want honest unconfigured refusal", err)
	}
	st := invoke(t, h, ToolDeployStatus, map[string]any{"app_id": id}, "u-3", toolCredA)
	if st["status"] != "none" {
		t.Errorf("status = %v, want none (nothing written)", st)
	}
}

func TestDeployFailureRecorded(t *testing.T) {
	h := setupFakes(t)
	h.sprites.Fail["CreateSprite"] = errFakeBoom()
	id := provisionedApp(t, h, toolCredA, toolTenantA, "shop", "f-1")
	// Business failure returns a result (persisting evidence), not a
	// terminal error (which would roll the record back).
	res := invoke(t, h, ToolDeployProvision, map[string]any{"app_id": id}, "f-2", toolCredA)
	if res["status"] != "failed" {
		t.Errorf("provision = %v, want failed outcome", res)
	}
	st := invoke(t, h, ToolDeployStatus, map[string]any{"app_id": id}, "f-3", toolCredA)
	if st["status"] != "failed" {
		t.Errorf("status = %v, want failed with evidence", st)
	}
	// Recovery: clearing the fault lets the retry succeed on the same app.
	delete(h.sprites.Fail, "CreateSprite")
	rec := invoke(t, h, ToolDeployProvision, map[string]any{"app_id": id}, "f-4", toolCredA)
	if rec["status"] != "ready" {
		t.Errorf("retry = %v, want ready", rec)
	}
}

func TestDeployCollisionTriesCandidates(t *testing.T) {
	h := setupFakes(t)
	// A foreign sprite already owns the bare slug.
	if _, err := h.sprites.CreateSprite(ctxForTest(), "shop", fly.SpriteOptions{}); err != nil {
		t.Fatal(err)
	}
	id := provisionedApp(t, h, toolCredA, toolTenantA, "shop", "k-1")
	res := invoke(t, h, ToolDeployProvision, map[string]any{"app_id": id}, "k-2", toolCredA)
	if res["resource_name"] != "shop-2" {
		t.Errorf("resource = %v, want shop-2", res)
	}
}

func TestDeployStatusReconciles(t *testing.T) {
	h := setupFakes(t)
	id := provisionedApp(t, h, toolCredA, toolTenantA, "shop", "r-1")
	invoke(t, h, ToolDeployProvision, map[string]any{"app_id": id}, "r-2", toolCredA)
	// Vanishing target reconciles to failed with evidence.
	if err := h.sprites.DeleteSprite(ctxForTest(), "shop"); err != nil {
		t.Fatal(err)
	}
	st := invoke(t, h, ToolDeployStatus, map[string]any{"app_id": id}, "r-3", toolCredA)
	if st["status"] != "failed" {
		t.Errorf("status = %v, want reconciled failed", st)
	}
}

func TestDeployStatusTransientKeepsRow(t *testing.T) {
	h := setupFakes(t)
	id := provisionedApp(t, h, toolCredA, toolTenantA, "shop", "t-1")
	invoke(t, h, ToolDeployProvision, map[string]any{"app_id": id}, "t-2", toolCredA)
	h.sprites.Fail["GetSprite"] = errFakeBoom()
	invokeErr(t, h, ToolDeployStatus, mustJSON(t, map[string]any{"app_id": id}), "t-3", toolCredA)
	delete(h.sprites.Fail, "GetSprite")
	st := invoke(t, h, ToolDeployStatus, map[string]any{"app_id": id}, "t-4", toolCredA)
	if st["status"] != "ready" {
		t.Errorf("status = %v, want untouched ready", st)
	}
}

func TestDeployRetire(t *testing.T) {
	h := setupFakes(t)
	id := provisionedApp(t, h, toolCredA, toolTenantA, "shop", "d-1")
	invoke(t, h, ToolDeployProvision, map[string]any{"app_id": id}, "d-2", toolCredA)
	ret := invoke(t, h, ToolDeployRetire, map[string]any{"app_id": id}, "d-3", toolCredA)
	if ret["status"] != "deleted" {
		t.Errorf("retire = %v", ret)
	}
	if _, err := h.sprites.GetSprite(ctxForTest(), "shop"); err == nil {
		t.Error("sprite survives retire")
	}
	// Idempotent: retiring again still reports deleted.
	again := invoke(t, h, ToolDeployRetire, map[string]any{"app_id": id}, "d-4", toolCredA)
	if again["status"] != "deleted" {
		t.Errorf("re-retire = %v", again)
	}
}

func TestDeployCrossTenant(t *testing.T) {
	h := setupFakes(t)
	id := provisionedApp(t, h, toolCredA, toolTenantA, "shop", "x-1")
	invokeErr(t, h, ToolDeployProvision, mustJSON(t, map[string]any{"app_id": id}), "x-2", toolCredB)
	invokeErr(t, h, ToolDeployStatus, mustJSON(t, map[string]any{"app_id": id}), "x-3", toolCredB)
	invokeErr(t, h, ToolDeployRetire, mustJSON(t, map[string]any{"app_id": id}), "x-4", toolCredB)
	if n := len(h.sprites.Sprites); n != 0 {
		t.Errorf("cross-tenant provision created %d sprites", n)
	}
}
