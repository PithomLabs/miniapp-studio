package tools

import (
	"encoding/json"
	"os"
	"testing"
)

// specDocForTenant loads the valid P1 fixture and rebinds its owner.
func specDocForTenant(t *testing.T, owner string) json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile("../spec/testdata/valid_invoice_review.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc["app"].(map[string]any)["owner"] = owner
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func createApp(t *testing.T, h harness, cred, tenant, slug, key string) string {
	t.Helper()
	res := invoke(t, h, ToolAppCreate, map[string]any{
		"slug": slug, "name": slug, "description": "d",
	}, key, cred)
	id, _ := res["app_id"].(string)
	if id == "" {
		t.Fatalf("app.create returned no id: %v", res)
	}
	return id
}

func saveSpec(t *testing.T, h harness, cred, appID string, doc json.RawMessage, key string) (int, string) {
	t.Helper()
	var raw any
	if err := json.Unmarshal(doc, &raw); err != nil {
		t.Fatal(err)
	}
	res := invoke(t, h, ToolSpecSave, map[string]any{"app_id": appID, "spec": raw}, key, cred)
	version, _ := res["version"].(float64)
	hash, _ := res["hash"].(string)
	if version == 0 || hash == "" {
		t.Fatalf("spec.save returned %v", res)
	}
	return int(version), hash
}

func TestSpecValidateTool(t *testing.T) {
	h := setupFakes(t)
	var doc any
	if err := json.Unmarshal(specDocForTenant(t, toolTenantA), &doc); err != nil {
		t.Fatal(err)
	}
	res := invoke(t, h, ToolSpecValidate, map[string]any{"spec": doc}, "v-1", toolCredA)
	if res["valid"] != true || res["hash"] == "" {
		t.Errorf("valid spec result = %v", res)
	}
	var bad any
	if err := json.Unmarshal(specDocForTenant(t, toolTenantA), &bad); err != nil {
		t.Fatal(err)
	}
	bad.(map[string]any)["capabilities"].(map[string]any)["operations"].([]any)[0].(map[string]any)["method"] = "POST"
	res = invoke(t, h, ToolSpecValidate, map[string]any{"spec": bad}, "v-2", toolCredA)
	if res["valid"] != false {
		t.Errorf("invalid spec accepted: %v", res)
	}
	if errs, _ := res["errors"].([]any); len(errs) == 0 {
		t.Error("no actionable errors returned")
	}
	if _, hasHash := res["hash"]; hasHash {
		t.Error("hash present alongside validation errors")
	}
	var foreign any
	if err := json.Unmarshal(specDocForTenant(t, toolTenantB), &foreign); err != nil {
		t.Fatal(err)
	}
	res = invoke(t, h, ToolSpecValidate, map[string]any{"spec": foreign}, "v-3", toolCredA)
	if res["valid"] != false {
		t.Errorf("foreign-owned spec validated: %v", res)
	}
}

func TestAppCRUD(t *testing.T) {
	h := setupFakes(t)
	id := createApp(t, h, toolCredA, toolTenantA, "shop", "c-1")
	got := invoke(t, h, ToolAppGet, map[string]any{"app_id": id}, "g-1", toolCredA)
	if got["slug"] != "shop" {
		t.Errorf("app.get = %v", got)
	}
	upd := invoke(t, h, ToolAppUpdate, map[string]any{"app_id": id, "name": "Shop 2"}, "u-1", toolCredA)
	if upd["name"] != "Shop 2" {
		t.Errorf("app.update = %v", upd)
	}
	createApp(t, h, toolCredA, toolTenantA, "blog", "c-2")
	createApp(t, h, toolCredA, toolTenantA, "wiki", "c-3")
	page := invoke(t, h, ToolAppList, map[string]any{"limit": 2, "offset": 0}, "l-1", toolCredA)
	if apps, _ := page["apps"].([]any); len(apps) != 2 {
		t.Errorf("page 1 = %d apps", len(apps))
	}
	if total, _ := page["total"].(float64); total != 3 {
		t.Errorf("total = %v, want 3", page["total"])
	}
	page2 := invoke(t, h, ToolAppList, map[string]any{"limit": 2, "offset": 2}, "l-2", toolCredA)
	if apps, _ := page2["apps"].([]any); len(apps) != 1 {
		t.Errorf("page 2 = %d apps", len(apps))
	}
	// Duplicate slug and idempotent retry.
	invokeErr(t, h, ToolAppCreate, mustJSON(t, map[string]any{"slug": "shop", "name": "x"}), "c-4", toolCredA)
	dup1 := invoke(t, h, ToolAppCreate, map[string]any{"slug": "retry", "name": "r"}, "c-dup", toolCredA)
	dup2 := invoke(t, h, ToolAppCreate, map[string]any{"slug": "retry", "name": "r"}, "c-dup", toolCredA)
	if dup1["app_id"] != dup2["app_id"] {
		t.Error("idempotent retry returned a different app")
	}
	// Cross-tenant and wrong-owner reads are uniform not-found.
	invokeErr(t, h, ToolAppGet, mustJSON(t, map[string]any{"app_id": id}), "g-x", toolCredB)
	invokeErr(t, h, ToolAppGet, mustJSON(t, map[string]any{"app_id": "nope"}), "g-y", toolCredA)
	invokeErr(t, h, ToolAppUpdate, mustJSON(t, map[string]any{"app_id": id, "name": "evil"}), "u-x", toolCredB)
	if got := invoke(t, h, ToolAppGet, map[string]any{"app_id": id}, "g-2", toolCredA); got["name"] != "Shop 2" {
		t.Errorf("cross-tenant write landed: %v", got)
	}
	if n := auditCount(t, h, toolTenantA, ToolAppCreate); n == 0 {
		t.Error("no audit evidence for app.create")
	}
}

func TestSpecSaveGet(t *testing.T) {
	h := setupFakes(t)
	id := createApp(t, h, toolCredA, toolTenantA, "docs", "s-1")
	doc := specDocForTenant(t, toolTenantA)
	v1, h1 := saveSpec(t, h, toolCredA, id, doc, "s-2")
	if v1 != 1 || h1 == "" {
		t.Errorf("save v1 = %d %q", v1, h1)
	}
	var doc2 any
	if err := json.Unmarshal(doc, &doc2); err != nil {
		t.Fatal(err)
	}
	doc2.(map[string]any)["app"].(map[string]any)["name"] = "Changed"
	raw2, _ := json.Marshal(doc2)
	v2, h2 := saveSpec(t, h, toolCredA, id, raw2, "s-3")
	if v2 != 2 || h2 == h1 {
		t.Errorf("save v2 = %d (hash changed: %v)", v2, h2 != h1)
	}
	latest := invoke(t, h, ToolSpecGet, map[string]any{"app_id": id}, "g-1", toolCredA)
	if latest["version"] != float64(2) {
		t.Errorf("latest = %v", latest)
	}
	first := invoke(t, h, ToolSpecGet, map[string]any{"app_id": id, "version": 1}, "g-2", toolCredA)
	if first["hash"] != h1 {
		t.Errorf("v1 hash = %v", first)
	}
	// Idempotent duplicate: same key replays version 2 without version 3.
	again := invoke(t, h, ToolSpecSave, map[string]any{"app_id": id, "spec": mustDecode(t, raw2)}, "s-3", toolCredA)
	if again["version"] != float64(2) {
		t.Errorf("duplicate save = %v, want replay of v2", again)
	}
	// Foreign and invalid specs refused.
	invokeErr(t, h, ToolSpecSave, mustJSON(t, map[string]any{"app_id": id, "spec": mustDecode(t, specDocForTenant(t, toolTenantB))}), "s-4", toolCredA)
	invokeErr(t, h, ToolSpecGet, mustJSON(t, map[string]any{"app_id": id}), "g-x", toolCredB)
	if n := auditCount(t, h, toolTenantA, ToolSpecSave); n == 0 {
		t.Error("no audit evidence for spec.save")
	}
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func mustDecode(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}
