package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/PithomLabs/miniapp-studio/artifact"
	"github.com/PithomLabs/miniapp/contract"
)

// bundleParts assembles one fixture bundle. mutate alters members
// before digests are computed, so every negative pins the exact
// layer that must refuse it.
type bundleParts struct {
	tenant string
	build  string
	spec   []byte
	files  map[string][]byte
}

func validSpec(t *testing.T, tenant string) []byte {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "..", "spec", "testdata", "valid_invoice_review.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc["app"].(map[string]any)["owner"] = tenant
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func fixtureManifest(t *testing.T, substrate contract.Substrate, tools ...string) []byte {
	t.Helper()
	return fixtureManifestClass(t, contract.ClassA, substrate, tools...)
}

// fixtureManifestClass builds a manifest with an explicit class: the
// framework substrate matrix (ClassA→go, ClassB→river, …) means a
// non-go substrate needs a non-A class to reach Studio's own check.
func fixtureManifestClass(t *testing.T, class contract.Class, substrate contract.Substrate, tools ...string) []byte {
	t.Helper()
	if len(tools) == 0 {
		tools = []string{"documents.invoice.read", "purchasing.order.match"}
	}
	var tms []contract.ToolManifest
	deploy := map[string]contract.ToolDeployment{}
	for _, name := range tools {
		tms = append(tms, contract.ToolManifest{
			Name:         contract.ToolName(name),
			Description:  "fixture tool",
			Class:        class,
			Substrate:    substrate,
			InputSchema:  json.RawMessage(`{"type":"object"}`),
			OutputSchema: json.RawMessage(`{"type":"object"}`),
		})
		deploy[name] = contract.ToolDeployment{Placement: contract.PlacementMachine,
			Profile: contract.ProfileRef{Name: "default", Digest: contract.Digest("sha256:" + strings.Repeat("0", 64))}}
	}
	mf := contract.MiniappManifest{
		ID: contract.Name("invoice-review"), Version: contract.Version("1.0.0"),
		Description: "fixture", Tools: tms,
		Deployment: contract.DeploymentSection{Tools: deploy},
	}
	if err := mf.Validate(); err != nil && substrate == contract.SubstrateGo {
		t.Fatalf("fixture manifest invalid: %v", err)
	}
	raw, err := json.Marshal(mf)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func fixtureGoMod(tenant string) []byte {
	return []byte("module studio-artifact/" + tenant + "/invoice-review\n\ngo " + artifact.PinnedGoVersion + "\n\nrequire " +
		artifact.MiniappModulePath + " " + artifact.PinnedMiniapp + "\n")
}

func fixtureGoSum() []byte {
	return []byte(artifact.MiniappGoSumZip + "\n" + artifact.MiniappGoSumMod + "\n")
}

func validBundleParts(t *testing.T, tenant, build string) *bundleParts {
	t.Helper()
	return &bundleParts{
		tenant: tenant, build: build,
		spec: validSpec(t, tenant),
		files: map[string][]byte{
			artifact.SpecFileName:     nil, // set below
			artifact.ManifestFileName: fixtureManifest(t, contract.SubstrateGo),
			artifact.GoModFile:        fixtureGoMod(tenant),
			artifact.GoSumFile:        fixtureGoSum(),
			"app/main.go":             []byte("package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"ok\") }\n"),
			"app/main_test.go":        []byte("package main\n\nimport \"testing\"\n\nfunc TestOk(t *testing.T) {}\n"),
		},
	}
}

// seal computes member digests, builds the envelope, stores every
// blob, inserts the bundle row, and returns the artifact id and the
// computed root. Members listed in files (keyed by path) are hashed
// as-is; mutate before calling to craft adversarial bundles.
func (p *bundleParts) seal(t *testing.T, h harness) (artifactID, root string) {
	t.Helper()
	p.files[artifact.SpecFileName] = p.spec
	ctx := context.Background()
	type entry struct {
		path, digest string
		size         int64
	}
	var files []artifact.FileEntry
	for path, data := range p.files {
		d := blobDigest(data)
		files = append(files, artifact.FileEntry{Path: path, Digest: d, Size: int64(len(data))})
		if _, err := h.owner.Exec(ctx,
			`INSERT INTO studio.artifact_blobs (tenant_id, digest, bytes, size_bytes)
			 VALUES ($1, $2, $3, $4) ON CONFLICT (tenant_id, digest) DO NOTHING`,
			p.tenant, d, data, len(data)); err != nil {
			t.Fatalf("seed blob %s: %v", path, err)
		}
		_ = entry{}
	}
	envMap := map[string]any{
		"contract": artifact.ContractVersion, "tenant_id": p.tenant, "build_id": p.build,
		"conversation_id": "c-seal", "turn_id": "t-seal",
		"app_id": "invoice-review", "app_version": "1.0.0",
		"toolchain": map[string]any{"Go": artifact.PinnedGoVersion, "Miniapp": artifact.PinnedMiniapp},
		"spec":      map[string]any{"path": artifact.SpecFileName, "digest": blobDigest(p.files[artifact.SpecFileName])},
		"files":     files,
	}
	envRaw, err := json.Marshal(envMap)
	if err != nil {
		t.Fatal(err)
	}
	root, errs := artifact.ComputeRoot(envRaw, p.files)
	if errs != nil {
		t.Fatalf("fixture root: %v", errs)
	}
	// Store the envelope blob and claim manifest mirroring ingest.
	envDigest := sha256.Sum256(envRaw)
	envDigestStr := "sha256:" + hex.EncodeToString(envDigest[:])
	if _, err := h.owner.Exec(ctx,
		`INSERT INTO studio.artifact_blobs (tenant_id, digest, bytes, size_bytes)
		 VALUES ($1, $2, $3, $4) ON CONFLICT (tenant_id, digest) DO NOTHING`,
		p.tenant, envDigestStr, envRaw, len(envRaw)); err != nil {
		t.Fatalf("seed envelope blob: %v", err)
	}
	claimFiles := append([]artifact.FileEntry{}, files...)
	claimFiles = append(claimFiles, artifact.FileEntry{Path: artifact.EnvelopeFileName, Digest: envDigestStr, Size: int64(len(envRaw))})
	claimRaw, err := json.Marshal(map[string]any{"files": claimFiles})
	if err != nil {
		t.Fatal(err)
	}
	id, err := randomID()
	if err != nil {
		t.Fatal(err)
	}
	artifactID = "a-" + id
	if _, err := h.owner.Exec(ctx,
		`INSERT INTO studio.artifacts (id, tenant_id, build_id, kind, digest, manifest)
		 VALUES ($1, $2, $3, 'bundle', $4, $5)`,
		artifactID, p.tenant, p.build, root, string(claimRaw)); err != nil {
		t.Fatalf("seed artifact row: %v", err)
	}
	return artifactID, root
}

func blobDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// verifyResult invokes artifact.verify and returns the decoded result.
func verifyResult(t *testing.T, h harness, cred, artifactID, key string) map[string]any {
	t.Helper()
	return invoke(t, h, ToolArtifactVerify, map[string]any{"artifact_id": artifactID}, key, cred)
}

func checkCodes(t *testing.T, res map[string]any) []string {
	t.Helper()
	checks, _ := res["checks"].([]any)
	var out []string
	for _, c := range checks {
		m, _ := c.(map[string]any)
		if pass, _ := m["pass"].(bool); !pass {
			code, _ := m["code"].(string)
			out = append(out, code)
		}
	}
	return out
}

func TestArtifactVerifyValid(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "av", toolCredA)
	p := validBundleParts(t, toolTenantA, buildID)
	artifactID, _ := p.seal(t, h)
	res := verifyResult(t, h, toolCredA, artifactID, "av-k")
	if res["verified"] != true {
		t.Fatalf("valid bundle refused: %v", res["checks"])
	}
	var verifiedAt any
	err := h.owner.QueryRow(context.Background(),
		`SELECT verified_at FROM studio.artifacts WHERE id = $1`, artifactID).Scan(&verifiedAt)
	if err != nil || verifiedAt == nil {
		t.Fatalf("verified_at unset: %v", err)
	}
	var passed bool
	err = h.owner.QueryRow(context.Background(),
		`SELECT passed FROM studio.artifact_verifications WHERE artifact_id = $1`, artifactID).Scan(&passed)
	if err != nil || !passed {
		t.Fatalf("evidence passed=%v, err=%v", passed, err)
	}
	// Re-verify converges on the original evidence id.
	again := verifyResult(t, h, toolCredA, artifactID, "av-k2")
	if again["verified"] != true || again["evidence_id"] != res["evidence_id"] {
		t.Errorf("re-verify = %v, want same evidence %v", again["evidence_id"], res["evidence_id"])
	}
}

// verifySealedBundle seals parts under a fresh build and returns the
// verify result. Every negative pins the refusing code.
func verifySealedBundle(t *testing.T, h harness, prefix string, mutate func(*bundleParts)) map[string]any {
	t.Helper()
	_, buildID := runningBuild(t, h, prefix, toolCredA)
	p := validBundleParts(t, toolTenantA, buildID)
	if mutate != nil {
		mutate(p)
	}
	artifactID, _ := p.seal(t, h)
	return verifyResult(t, h, toolCredA, artifactID, prefix+"-k")
}

func requireRefused(t *testing.T, name, wantCode string, res map[string]any) {
	t.Helper()
	if res["verified"] == true {
		t.Fatalf("%s: accepted", name)
	}
	for _, code := range checkCodes(t, res) {
		if code == wantCode {
			return
		}
	}
	t.Errorf("%s: codes = %v, want %s present", name, checkCodes(t, res), wantCode)
}

func TestArtifactVerifyNegatives(t *testing.T) {
	h := setupFakes(t)
	// N1: non-Go implementation file.
	res := verifySealedBundle(t, h, "n1", func(p *bundleParts) {
		p.files["app/tool.py"] = []byte("print('hi')\n")
	})
	requireRefused(t, "N1 non-Go", "E_LANG", res)
	// N2: alternate substrate (ClassB+river passes the framework
	// matrix, so Studio's own Go-only check must refuse it).
	res = verifySealedBundle(t, h, "n2", func(p *bundleParts) {
		p.files[artifact.ManifestFileName] = fixtureManifestClass(t, contract.ClassB, contract.SubstrateRiver)
	})
	requireRefused(t, "N2 substrate", "E_SUBSTRATE", res)
	// N3: invalid spec (app.id removed).
	res = verifySealedBundle(t, h, "n3", func(p *bundleParts) {
		var doc map[string]any
		if err := json.Unmarshal(p.spec, &doc); err != nil {
			t.Fatal(err)
		}
		delete(doc["app"].(map[string]any), "id")
		raw, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		p.spec = raw
	})
	requireRefused(t, "N3 spec", "E_APP", res)
	// N5: traversal member path.
	res = verifySealedBundle(t, h, "n5", func(p *bundleParts) {
		p.files["../escape.go"] = []byte("package main\n")
	})
	requireRefused(t, "N5 traversal", "E_PATH", res)
	// N7: oversize member (6 MB stored bytes).
	res = verifySealedBundle(t, h, "n7", func(p *bundleParts) {
		p.files["app/big.go"] = bytes.Repeat([]byte("x"), artifact.MaxFileBytes+1)
	})
	requireRefused(t, "N7 oversize", "E_PATH", res)
	// N9: secret-bearing source.
	res = verifySealedBundle(t, h, "n9", func(p *bundleParts) {
		p.files["app/main.go"] = []byte("package main\n\nvar k = \"sk-live-abcdefghij1234567890\"\n")
	})
	requireRefused(t, "N9 secret", "E_SECRET", res)
	// N11a: go.mod replace directive.
	res = verifySealedBundle(t, h, "n11a", func(p *bundleParts) {
		p.files[artifact.GoModFile] = append(p.files[artifact.GoModFile],
			[]byte("\nreplace github.com/PithomLabs/miniapp => ../evil\n")...)
	})
	requireRefused(t, "N11a replace", "E_DEPS", res)
	// N11b: unapproved require.
	res = verifySealedBundle(t, h, "n11b", func(p *bundleParts) {
		p.files[artifact.GoModFile] = append(p.files[artifact.GoModFile],
			[]byte("\nrequire example.com/evil v1.0.0\n")...)
	})
	requireRefused(t, "N11b require", "E_DEPS", res)
	// N12a: banned import.
	res = verifySealedBundle(t, h, "n12a", func(p *bundleParts) {
		p.files["app/main.go"] = []byte("package main\n\nimport \"os/exec\"\n\nvar _ = exec.Command\n")
	})
	requireRefused(t, "N12a import", "E_DEPS", res)
	// N12b: cgo import.
	res = verifySealedBundle(t, h, "n12b", func(p *bundleParts) {
		p.files["app/main.go"] = []byte("package main\n\nimport \"C\"\n")
	})
	requireRefused(t, "N12b cgo", "E_DEPS", res)
	// N12c: unapproved third-party import.
	res = verifySealedBundle(t, h, "n12c", func(p *bundleParts) {
		p.files["app/main.go"] = []byte("package main\n\nimport _ \"example.com/evil\"\n")
	})
	requireRefused(t, "N12c third-party", "E_DEPS", res)
	// N13a: spec operation without a declared tool.
	res = verifySealedBundle(t, h, "n13a", func(p *bundleParts) {
		p.files[artifact.ManifestFileName] = fixtureManifest(t, contract.SubstrateGo, "documents.invoice.read")
	})
	requireRefused(t, "N13a binding", "E_BINDING", res)
}

func TestArtifactVerifyEnvelopeAppBinding(t *testing.T) {
	h := setupFakes(t)
	// N13c: envelope app id differs from the spec app id. The seal
	// helper binds the envelope to invoice-review; swapping the spec
	// document for a same-shape different-app spec breaks the tie.
	res := verifySealedBundle(t, h, "n13c", func(p *bundleParts) {
		var doc map[string]any
		if err := json.Unmarshal(p.spec, &doc); err != nil {
			t.Fatal(err)
		}
		doc["app"].(map[string]any)["id"] = "other-app"
		raw, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		p.spec = raw
	})
	requireRefused(t, "N13c envelope-spectie", "E_BINDING", res)
}

func TestArtifactVerifyMissingInputs(t *testing.T) {
	h := setupFakes(t)
	// Unknown artifact id: the invocation errors (no evidence can
	// persist without its parent row), it does not verify false.
	err := invokeErr(t, h, ToolArtifactVerify,
		mustJSON(t, map[string]any{"artifact_id": "a-nope"}), "m-k1", toolCredA)
	if err == nil || !strings.Contains(err.Error(), "unknown artifact") {
		t.Errorf("missing artifact err = %v, want unknown-artifact refusal", err)
	}
	// Non-bundle row.
	_, buildID := runningBuild(t, h, "m", toolCredA)
	if _, err := h.owner.Exec(context.Background(),
		`INSERT INTO studio.artifacts (id, tenant_id, build_id, kind, digest) VALUES ('a-spec', $1, $2, 'spec', 'sha256:x')`,
		toolTenantA, buildID); err != nil {
		t.Fatal(err)
	}
	res := verifyResult(t, h, toolCredA, "a-spec", "m-k2")
	requireRefused(t, "non-bundle kind", "E_CONTRACT", res)
}

// TestArtifactVerifyTamperedBlob pins N4 at the verify layer: bytes
// swapped after the claim fail the identity recomputation. The bundle
// carries a unique marker so its digests are unshared; the swap
// touches only this artifact's blobs, whatever order tests run in.
func TestArtifactVerifyTamperedBlob(t *testing.T) {
	h := setupFakes(t)
	_, buildID := runningBuild(t, h, "n4", toolCredA)
	p := validBundleParts(t, toolTenantA, buildID)
	p.files["app/main.go"] = []byte("package main\n\n// n4-marker\n\nfunc main() {}\n")
	artifactID, _ := p.seal(t, h)
	// Tamper one non-envelope member: the envelope still parses,
	// so the identity recomputation is what must fail (EDigest).
	if _, err := h.owner.Exec(context.Background(),
		`UPDATE studio.artifact_blobs AS b SET bytes = 'tampered' FROM studio.artifacts AS a,
		 jsonb_array_elements(a.manifest->'files') AS f
		 WHERE a.id = $1 AND b.tenant_id = a.tenant_id AND b.digest = f->>'digest'
		 AND f->>'path' = 'app/main.go'`,
		artifactID); err != nil {
		t.Fatal(err)
	}
	res := verifyResult(t, h, toolCredA, artifactID, "n4-k")
	requireRefused(t, "N4 tamper", "E_DIGEST", res)
}
