package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jackc/pgx/v5"

	"golang.org/x/mod/modfile"

	"github.com/PithomLabs/miniapp-studio/artifact"
	"github.com/PithomLabs/miniapp-studio/sandbox"
	"github.com/PithomLabs/miniapp-studio/spec"
	"github.com/PithomLabs/miniapp-studio/store"
)

// newTestRunner builds the production runner. Overridden in tests;
// production always uses the default configuration.
var newTestRunner = func() sandbox.RunnerAPI {
	return sandbox.NewRunner(sandbox.Config{})
}

// handleConsoleArtifactTest triggers one artifact test execution.
// Operator session + Origin required. The outcome (passed, failed,
// or honestly blocked) lands in the notice; detail lives on the
// Validate scene.
func (s *Server) handleConsoleArtifactTest(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderError(w, r, http.StatusBadRequest, "Bad request.", s.logger, err.Error())
		return
	}
	if _, err := opKey(r); err != nil {
		renderError(w, r, http.StatusBadRequest, "Missing operation key.", s.logger, "")
		return
	}
	tenant, _, _, ok := TenantOf(r)
	if !ok || tenant == "" {
		renderError(w, r, http.StatusUnauthorized, "Authentication required.", s.logger, "")
		return
	}
	artifactID, convID := r.Form.Get("artifact_id"), r.Form.Get("conversation_id")
	if artifactID == "" {
		renderError(w, r, http.StatusBadRequest, "Artifact is required.", s.logger, "")
		return
	}
	outcome, err := s.runArtifactTests(r.Context(), tenant, artifactID)
	notice := outcome
	if err != nil {
		notice = "Test run failed to start: " + err.Error()
	}
	http.Redirect(w, r, "/console?scene=validate&conversation="+url.QueryEscape(convID)+
		"&notice="+url.QueryEscape(notice), http.StatusFound)
}

// runArtifactTests executes the full test phase for one verified
// bundle: claim + start a run row, re-hash inputs, run the suite in
// a disposable container, run registration conformance on green,
// record the merged report. It returns a one-line outcome for the
// operator notice.
func (s *Server) runArtifactTests(ctx context.Context, tenant, artifactID string) (string, error) {
	data, err := s.loadBundleForTest(ctx, tenant, artifactID)
	if err != nil {
		return "", err
	}
	st := sandbox.NewStore(s.pool)
	runID, status, err := st.ClaimRun(ctx, tenant, artifactID, data.digest)
	if err != nil {
		return "", err
	}
	if status == sandbox.RunRunning {
		return "Test run " + runID + " already running.", nil
	}
	started, err := st.StartRun(ctx, tenant, runID)
	if err != nil || !started {
		return "", fmt.Errorf("test run %s already handled", runID)
	}
	runner := newTestRunner()
	res, err := runner.Run(ctx, sandbox.Request{
		Files:   data.testFiles(),
		Workdir: "/work/app",
		Command: []string{"go", "test", "-json", "./..."},
	})
	if err != nil {
		return "", s.finishTestRun(ctx, tenant, runID, sandbox.Report{}, err)
	}
	if res.Blocked {
		report := sandbox.Report{}
		if _, ferr := st.FinishRun(ctx, tenant, runID, sandbox.RunBlocked, report); ferr != nil {
			return "", ferr
		}
		return "Test run blocked: " + res.Cause, nil
	}
	report := sandbox.ParseTestJSON(res.Stdout, res.ExitCode, res.Truncated)
	outcome := "failed"
	final := sandbox.RunFailed
	if report.Pass {
		final = sandbox.RunPassed
		outcome = fmt.Sprintf("passed %d/%d", report.Passed, report.Total)
		// Conformance runs only on green tests: a failing suite
		// refuses the artifact without further execution.
		conf, cres, cerr := s.runConformance(ctx, runner, data)
		if cerr != nil {
			return "", s.finishTestRun(ctx, tenant, runID, report, cerr)
		}
		report.Conformance = &conf
		if cres.Blocked || !conf.Pass {
			final = sandbox.RunFailed
			outcome = "conformance failed"
		}
	}
	if _, err := st.FinishRun(ctx, tenant, runID, final, report); err != nil {
		return "", err
	}
	return "Tests " + outcome + ".", nil
}

// finishTestRun records a runner-level failure and returns the cause.
func (s *Server) finishTestRun(ctx context.Context, tenant, runID string, report sandbox.Report, cause error) error {
	st := sandbox.NewStore(s.pool)
	if _, err := st.FinishRun(ctx, tenant, runID, sandbox.RunFailed, report); err != nil {
		return err
	}
	return cause
}

// testBundle is the verified bytes under test.
type testBundle struct {
	digest  string
	members map[string][]byte
	specOps []string
	goMod   []byte
	goSum   []byte
}

// loadBundleForTest loads verified bytes and re-proves identity:
// recomputed root must equal the pinned digest or nothing executes.
func (s *Server) loadBundleForTest(ctx context.Context, tenant, artifactID string) (*testBundle, error) {
	var data *store.BundleData
	err := store.InTenantTx(ctx, s.pool, tenant, func(tx pgx.Tx) error {
		var lerr error
		data, lerr = store.LoadBundle(ctx, tx, tenant, artifactID)
		return lerr
	})
	if err != nil {
		return nil, fmt.Errorf("unknown artifact")
	}
	if !data.Verified {
		return nil, fmt.Errorf("artifact is not verified")
	}
	root, errs := artifact.ComputeRoot(data.Envelope, data.Members)
	if errs != nil || root != data.Digest {
		return nil, fmt.Errorf("artifact bytes changed since verification")
	}
	files := make(map[string][]byte, len(data.Members)+1)
	for path, blob := range data.Members {
		files[path] = blob
	}
	var doc spec.MiniappSpec
	if err := json.Unmarshal(data.Members[artifact.SpecFileName], &doc); err != nil {
		return nil, fmt.Errorf("spec bytes unreadable")
	}
	var ops []string
	for _, op := range doc.Capabilities.Operations {
		ops = append(ops, op.ID)
	}
	return &testBundle{
		digest:  data.Digest,
		members: files,
		specOps: ops,
		goMod:   data.Members[artifact.GoModFile],
		goSum:   data.Members[artifact.GoSumFile],
	}, nil
}

func (b *testBundle) testFiles() map[string][]byte { return b.members }

// runConformance registers the artifact against its declared
// manifest in-container and requires every spec operation.
func (s *Server) runConformance(ctx context.Context, runner sandbox.RunnerAPI, b *testBundle) (sandbox.ConformanceResult, sandbox.Result, error) {
	f, err := modfile.Parse("go.mod", b.goMod, nil)
	if err != nil || f.Module == nil {
		return sandbox.ConformanceResult{}, sandbox.Result{}, fmt.Errorf("go.mod unreadable")
	}
	return sandbox.RunConformance(ctx, runner, b.members, f.Module.Mod.Path, b.goMod, b.goSum, b.specOps, artifact.ManifestFileName)
}
