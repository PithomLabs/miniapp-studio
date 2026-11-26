// Command genfixture builds one coherent rehearsal bundle: member
// files plus the claim entry a Builder would emit, bound to live
// tenant/conversation/build ids. Bytes go through the production
// identity code, so verification of the fetched bundle exercises
// the real path — only the Builder side is scripted.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PithomLabs/miniapp-studio/artifact"
)

func digestOf(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func main() {
	specPath := flag.String("spec", "", "MiniappSpec fixture JSON")
	out := flag.String("out", "", "output directory for member files")
	tenant := flag.String("tenant", "", "tenant id")
	conv := flag.String("conv", "", "conversation id")
	build := flag.String("build", "", "build id")
	turn := flag.String("turn", "", "turn id")
	appID := flag.String("app-id", "", "application id (from the created app row)")
	marker := flag.String("marker", "rehearsal", "Go comment marker unsharing the bytes")
	flag.Parse()
	if *specPath == "" || *out == "" || *tenant == "" || *conv == "" || *build == "" || *appID == "" {
		fmt.Fprintln(os.Stderr, "spec, out, tenant, conv, build, and app-id are required")
		os.Exit(2)
	}
	if err := run(*specPath, *out, *tenant, *conv, *build, *turn, *appID, *marker); err != nil {
		fmt.Fprintln(os.Stderr, "genfixture:", err)
		os.Exit(1)
	}
}

func run(specPath, out, tenant, conv, build, turn, appID, marker string) error {
	raw, err := os.ReadFile(specPath)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	doc["app"].(map[string]any)["owner"] = tenant
	doc["app"].(map[string]any)["id"] = appID
	specRaw, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	mf := map[string]any{
		"id": appID, "version": "1.0.0", "description": "d",
		"tools": []any{
			map[string]any{"name": "documents.invoice.read", "description": "d",
				"class": "A", "substrate": "go",
				"inputSchema": map[string]any{"type": "object"}, "outputSchema": map[string]any{"type": "object"}},
			map[string]any{"name": "purchasing.order.match", "description": "d",
				"class": "A", "substrate": "go",
				"inputSchema": map[string]any{"type": "object"}, "outputSchema": map[string]any{"type": "object"}},
		},
		"deployment": map[string]any{"tools": map[string]any{
			"documents.invoice.read": map[string]any{"placement": "machine",
				"profile": map[string]any{"name": "default", "digest": "sha256:" + strings.Repeat("0", 64)}},
			"purchasing.order.match": map[string]any{"placement": "machine",
				"profile": map[string]any{"name": "default", "digest": "sha256:" + strings.Repeat("0", 64)}},
		}},
	}
	mfRaw, err := json.Marshal(mf)
	if err != nil {
		return err
	}
	gomod := []byte("module studio-artifact/" + tenant + "/" + appID + "\n\ngo " + artifact.PinnedGoVersion +
		"\n\nrequire " + artifact.MiniappModulePath + " " + artifact.PinnedMiniapp + "\n")
	gosum := []byte(artifact.MiniappGoSumZip + "\n" + artifact.MiniappGoSumMod + "\n")
	members := map[string][]byte{
		artifact.SpecFileName:     specRaw,
		artifact.ManifestFileName: mfRaw,
		artifact.GoModFile:        gomod,
		artifact.GoSumFile:        gosum,
		"app/main.go":             []byte("package main\n\n// " + marker + "\n\nfunc main() {}\n"),
		"app/main_test.go":        []byte("package main\n\nimport \"testing\"\n\nfunc TestOk(t *testing.T) {}\n"),
	}
	var files []artifact.FileEntry
	for path, data := range members {
		files = append(files, artifact.FileEntry{Path: path, Digest: digestOf(data), Size: int64(len(data))})
	}
	envRaw, err := json.Marshal(map[string]any{
		"contract": artifact.ContractVersion, "tenant_id": tenant, "build_id": build,
		"conversation_id": conv, "turn_id": turn,
		"app_id": appID, "app_version": "1.0.0",
		"toolchain": map[string]any{"Go": artifact.PinnedGoVersion, "Miniapp": artifact.PinnedMiniapp},
		"spec":      map[string]any{"path": artifact.SpecFileName, "digest": digestOf(specRaw)},
		"files":     files,
	})
	if err != nil {
		return err
	}
	root, errs := artifact.ComputeRoot(envRaw, members)
	if errs != nil {
		return fmt.Errorf("fixture root: %v", errs)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	write := func(path string, data []byte) error {
		full := filepath.Join(out, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		return os.WriteFile(full, data, 0o644)
	}
	for path, data := range members {
		if err := write(path, data); err != nil {
			return err
		}
	}
	if err := write(artifact.EnvelopeFileName, envRaw); err != nil {
		return err
	}
	claimFiles := []any{}
	for _, e := range files {
		claimFiles = append(claimFiles, map[string]any{
			"path": e.Path, "digest": e.Digest, "size": float64(e.Size)})
	}
	claimFiles = append(claimFiles, map[string]any{
		"path":   artifact.EnvelopeFileName,
		"digest": digestOf(envRaw), "size": float64(len(envRaw))})
	claim := map[string]any{
		"kind": "bundle", "digest": root,
		"size_bytes": float64(100), "files": claimFiles,
	}
	claimRaw, err := json.MarshalIndent(claim, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "claim.json"), claimRaw, 0o644); err != nil {
		return err
	}
	fmt.Println(root)
	return nil
}
