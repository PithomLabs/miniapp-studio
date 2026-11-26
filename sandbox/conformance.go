package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/mod/modfile"
)

// Conformance convention (frozen D3, completing the v1 binding
// requirement): the artifact root package MUST export
// `func Implementation() *registry.Implementation`. The harness
// below registers the artifact's declared manifest against that
// implementation in a fresh registry and prints the registered tool
// names. Absence fails the harness build (conformance FAILED with
// clear evidence); no D2 change was needed because the build
// failure itself is the enforcement.
//
// Harness template: %s[0] is the artifact module path, %s[1] the
// manifest filename under /work.
const harnessMainTmpl = `package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/registry"

	artifact %[1]q
)

func fail(err error) {
	fmt.Fprintln(os.Stderr, "conformance:", err)
	os.Exit(3)
}

func main() {
	raw, err := os.ReadFile(%[2]q)
	if err != nil {
		fail(err)
	}
	var mf contract.MiniappManifest
	if err := json.Unmarshal(raw, &mf); err != nil {
		fail(err)
	}
	digest, err := contract.ProfileDigest(&mf)
	if err != nil {
		fail(err)
	}
	impl := artifact.Implementation()
	if impl == nil || impl.Build == "" {
		fail(fmt.Errorf("nil implementation or empty build identity"))
	}
	reg := registry.New()
	env := contract.ReleaseEnvelope{
		Manifest: digest, Implementation: impl.Build,
		Build: "conformance-probe", SourceRevision: "conformance-probe",
		Artifact: digest,
	}
	if err := reg.Register(&mf, impl, env); err != nil {
		fail(err)
	}
	mf2, _, err := reg.Resolve(string(mf.ID), string(mf.Version))
	if err != nil {
		fail(err)
	}
	var names []string
	for _, tool := range mf2.Tools {
		names = append(names, string(tool.Name))
	}
	out, _ := json.Marshal(map[string]any{"registered": names})
	fmt.Println(string(out))
}
`

// ConformanceResult is the N13b verdict: which declared operations
// the built code actually registers.
type ConformanceResult struct {
	Registered []string `json:"registered"`
	Missing    []string `json:"missing"`
	Pass       bool     `json:"pass"`
}

// RunConformance executes the registration-conformance probe: it
// builds a Studio-authored harness module in /scratch (the artifact
// bytes stay untouched in /work:ro), registers the artifact's
// minimapp.json against its exported Implementation, and requires
// every spec operation to be registered. modulePath is the artifact
// module path (from its verified go.mod); artifactGoSum is the
// artifact's own go.sum (the probe resolves the same closure —
// passed as bytes so the probe never reads untrusted paths
// directly); specOps are the spec's declared operation ids;
// manifestPath is the minimapp file under /work.
func RunConformance(ctx context.Context, r RunnerAPI, bundle map[string][]byte, modulePath string, artifactGoMod, artifactGoSum []byte, specOps []string, manifestPath string) (ConformanceResult, Result, error) {
	var zero ConformanceResult
	if modulePath == "" || manifestPath == "" || len(artifactGoSum) == 0 || len(bundle) == 0 || len(artifactGoMod) == 0 {
		return zero, Result{}, fmt.Errorf("sandbox: conformance needs bundle bytes, the module path, go.mod, go.sum, and manifest path")
	}
	// The probe inherits the artifact's go.mod verbatim (plus
	// itself): under -mod=readonly an incomplete require list fails
	// closed, so the probe builds in exactly the module universe
	// verification saw. The artifact's own successful test run
	// proves that universe resolves offline.
	if _, err := modfile.Parse("go.mod", artifactGoMod, nil); err != nil {
		return zero, Result{}, fmt.Errorf("sandbox: conformance go.mod: %w", err)
	}
	probeMod := "module probe\n\n" + strings.TrimPrefix(string(artifactGoMod),
		"module "+modulePath) + "\nrequire " + modulePath + " v0.0.0\n\nreplace " +
		modulePath + " => /work/app\n"
	req := Request{
		Files: bundle,
		Scratch: map[string][]byte{
			"probe/go.mod":  []byte(probeMod),
			"probe/go.sum":  artifactGoSum,
			"probe/main.go": []byte(fmt.Sprintf(harnessMainTmpl, modulePath, "/work/"+manifestPath)),
		},
		Workdir: "/scratch/probe",
		Command: []string{"go", "run", "."},
	}
	res, err := r.Run(ctx, req)
	if err != nil {
		return zero, res, err
	}
	if res.Blocked || res.Cause != "" || res.Truncated {
		return zero, res, nil
	}
	if res.ExitCode != 0 {
		return zero, res, nil
	}
	var decoded struct {
		Registered []string `json:"registered"`
	}
	// The harness prints exactly one JSON line on success; scan for
	// it so go build chatter on stdout cannot confuse the parse.
	for _, line := range strings.Split(string(res.Stdout), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{") {
			if err := json.Unmarshal([]byte(line), &decoded); err == nil && decoded.Registered != nil {
				break
			}
		}
	}
	if decoded.Registered == nil {
		return zero, res, nil
	}
	have := map[string]bool{}
	for _, name := range decoded.Registered {
		have[name] = true
	}
	var missing []string
	for _, want := range specOps {
		if !have[want] {
			missing = append(missing, want)
		}
	}
	return ConformanceResult{Registered: decoded.Registered, Missing: missing, Pass: len(missing) == 0}, res, nil
}
