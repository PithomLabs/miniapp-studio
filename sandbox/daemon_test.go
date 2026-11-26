package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// requireImage skips when the toolchain image is absent (fresh
// checkout, CI without the image). The skip names its precondition;
// D3 evidence records executed-vs-skipped honestly.
func requireImage(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "image", "inspect", ToolchainImage).Run(); err != nil {
		t.Skipf("toolchain image %s absent: build sandbox/testdata/toolchain", ToolchainImage)
	}
}

func loadFixtureDir(t *testing.T, dir, prefix string) map[string][]byte {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	root := filepath.Join(filepath.Dir(thisFile), "testdata", dir)
	out := map[string][]byte{}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[prefix+e.Name()] = raw
	}
	return out
}

func daemonRunner() *Runner {
	return NewRunner(Config{Timeout: 4 * time.Minute})
}

// TestToolchainImageContent behaviorally pins the image: labels,
// toolchain version, non-root user, and the pre-seeded miniapp
// module. Rebuilds change the image id (recorded in evidence),
// never these properties.
func TestToolchainImageContent(t *testing.T) {
	requireImage(t)
	r := daemonRunner()
	ctx := context.Background()
	res, err := r.Run(ctx, Request{
		Workdir: "/work",
		Command: []string{"sh", "-c", "go version && whoami && ls /go/pkg/mod/github.com/ && go env GOMODCACHE"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Blocked || res.ExitCode != 0 {
		t.Fatalf("probe: %+v", res)
	}
	for _, want := range []string{"go1.27.3", "tester", "!pithom!labs", "/go/pkg/mod"} {
		if !strings.Contains(string(res.Stdout), want) {
			t.Errorf("image content lacks %q:\n%s", want, res.Stdout)
		}
	}
}

// TestDaemonValidRun is the green path: the fixture suite passes
// in-container with a parsed machine-readable report.
func TestDaemonValidRun(t *testing.T) {
	requireImage(t)
	r := daemonRunner()
	files := loadFixtureDir(t, "app", "app/")
	res, err := r.Run(context.Background(), Request{
		Files: files, Workdir: "/work/app",
		Command: []string{"go", "test", "-json", "-run", "TestOk|TestSecretOutput", "./..."},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Blocked {
		t.Fatalf("blocked: %s", res.Cause)
	}
	rep := ParseTestJSON(res.Stdout, res.ExitCode, res.Truncated)
	if !rep.Pass || rep.Total != 2 || rep.Passed != 2 {
		t.Fatalf("report = %+v", rep)
	}
	t.Logf("valid run: %d tests passed", rep.Passed)
}

// TestDaemonFailingRun pins N8: failing tests record FAILED with the
// failures listed — never green.
func TestDaemonFailingRun(t *testing.T) {
	requireImage(t)
	r := daemonRunner()
	files := loadFixtureDir(t, "app", "app/")
	res, err := r.Run(context.Background(), Request{
		Files: files, Workdir: "/work/app",
		Command: []string{"go", "test", "-json", "-run", "TestFailFixture", "./..."},
	})
	if err != nil {
		t.Fatal(err)
	}
	rep := ParseTestJSON(res.Stdout, res.ExitCode, res.Truncated)
	if rep.Pass || rep.Failed != 1 {
		t.Fatalf("report = %+v, want 1 failure", rep)
	}
	if len(rep.Failures) != 1 || rep.Failures[0].Test != "TestFailFixture" {
		t.Fatalf("failures = %+v", rep.Failures)
	}
}

// TestDaemonCanary pins N18: the egress canary passes (dial refused)
// inside the container. A breaching sandbox would fail this test
// AND trip the runner quarantine via the breach marker.
func TestDaemonCanary(t *testing.T) {
	requireImage(t)
	r := daemonRunner()
	files := loadFixtureDir(t, "app", "app/")
	res, err := r.Run(context.Background(), Request{
		Files: files, Workdir: "/work/app",
		Command: []string{"go", "test", "-json", "-run", "TestCanaryEgress", "./..."},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep := ParseTestJSON(res.Stdout, res.ExitCode, res.Truncated); !rep.Pass {
		t.Fatalf("canary failed: %+v (quarantined=%v)", rep, r.Quarantined())
	}
	if r.Quarantined() {
		t.Error("runner quarantined on a green canary")
	}
}

// TestDaemonEnvScrub proves host environment never crosses: the
// suite sets a sentinel the fixture must not see.
func TestDaemonEnvScrub(t *testing.T) {
	requireImage(t)
	t.Setenv("STUDIO_SENTINEL", "sentinel-value-12345")
	r := daemonRunner()
	files := loadFixtureDir(t, "app", "app/")
	res, err := r.Run(context.Background(), Request{
		Files: files, Workdir: "/work/app",
		Command: []string{"go", "test", "-json", "-run", "TestEnvScrub", "./..."},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep := ParseTestJSON(res.Stdout, res.ExitCode, res.Truncated); !rep.Pass {
		t.Fatalf("env scrub failed: %+v", rep)
	}
}

// TestDaemonFSConfinement proves mount policy: /work read-only,
// /scratch writable, expected content present.
func TestDaemonFSConfinement(t *testing.T) {
	requireImage(t)
	r := daemonRunner()
	files := loadFixtureDir(t, "app", "app/")
	res, err := r.Run(context.Background(), Request{
		Files: files, Workdir: "/work/app",
		Command: []string{"go", "test", "-json", "-run", "TestFSConfinement", "./..."},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep := ParseTestJSON(res.Stdout, res.ExitCode, res.Truncated); !rep.Pass {
		t.Fatalf("fs confinement failed: %+v", rep)
	}
}

// TestDaemonPidsCap proves pids-limit enforcement with a bounded
// bomb (300 sleepers; safe even uncapped, red if the cap is gone).
// It re-asserts the flag first: without it, the test refuses to run
// the bomb at all.
func TestDaemonPidsCap(t *testing.T) {
	requireImage(t)
	r := daemonRunner()
	args := strings.Join(r.containerArgs("n", "img", "/w", "/s",
		Request{Workdir: "/work/app", Command: []string{"x"}}), " ")
	if !strings.Contains(args, "--pids-limit "+defaultPidsLimit) {
		t.Fatal("pids-limit flag absent: refusing to run the bomb")
	}
	files := loadFixtureDir(t, "app", "app/")
	res, err := r.Run(context.Background(), Request{
		Files: files, Workdir: "/work/app",
		Command: []string{"go", "test", "-json", "-timeout", "90s", "-run", "TestPidsCap", "./..."},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep := ParseTestJSON(res.Stdout, res.ExitCode, res.Truncated); !rep.Pass {
		t.Fatalf("pids cap proof failed: %+v", rep)
	}
}

// TestDaemonTimeoutCleanup proves wall-clock kill plus cleanup: a
// sleeper dies on timeout and leaves no container behind.
func TestDaemonTimeoutCleanup(t *testing.T) {
	requireImage(t)
	r := NewRunner(Config{Timeout: 5 * time.Second})
	res, err := r.Run(context.Background(), Request{
		Workdir: "/work", Command: []string{"sleep", "60"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != -1 || !strings.Contains(res.Cause, "timeout") {
		t.Fatalf("result = %+v, want timeout kill", res)
	}
	out, err := exec.Command("docker", "ps", "-aq", "--filter", "name=phase-d-run-").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Errorf("leftover containers: %s", out)
	}
}

// TestDaemonSecretRedaction pins N10: credential-shaped spans in
// container output persist redacted, never raw.
func TestDaemonSecretRedaction(t *testing.T) {
	requireImage(t)
	r := daemonRunner()
	files := loadFixtureDir(t, "app", "app/")
	res, err := r.Run(context.Background(), Request{
		Files: files, Workdir: "/work/app",
		Command: []string{"go", "test", "-json", "-run", "TestSecretOutput", "-v", "./..."},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(res.Stdout), "sk-live-abcdefghij1234567890") {
		t.Error("raw credential span survived in runner output")
	}
	if !strings.Contains(string(res.Stdout), "[REDACTED]") {
		t.Error("redaction marker missing from runner output")
	}
}

// conformanceBundle merges one app dir (under app/) with the shared
// minimapp.json at the bundle root.
func conformanceBundle(t *testing.T, appDir string) map[string][]byte {
	t.Helper()
	files := loadFixtureDir(t, appDir, "app/")
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "testdata", "minimapp.json"))
	if err != nil {
		t.Fatal(err)
	}
	files["minimapp.json"] = raw
	return files
}

// TestDaemonConformance pins N13b green: the fixture registers its
// declared tools through the real registry in-container.
func TestDaemonConformance(t *testing.T) {
	requireImage(t)
	r := daemonRunner()
	files := conformanceBundle(t, "app")
	conf, res, err := RunConformance(context.Background(), r, files,
		"studio-artifact/t-sandbox/fixture", files["app/go.mod"], files["app/go.sum"],
		[]string{"fixture.echo", "fixture.probe"}, "minimapp.json")
	if err != nil {
		t.Fatal(err)
	}
	if res.Blocked {
		t.Fatalf("blocked: %s", res.Cause)
	}
	if !conf.Pass || len(conf.Missing) != 0 {
		t.Fatalf("conformance = %+v (exit %d):\n%s\n%s",
			conf, res.ExitCode, res.Stdout, res.Stderr)
	}
}

// TestDaemonConformanceMissingExport pins the N13b negative: a
// module without the Implementation export fails conformance with
// build evidence, never a silent pass.
func TestDaemonConformanceMissingExport(t *testing.T) {
	requireImage(t)
	r := daemonRunner()
	files := conformanceBundle(t, "noimpl")
	conf, res, err := RunConformance(context.Background(), r, files,
		"studio-artifact/t-sandbox/noimpl", files["app/go.mod"], files["app/go.sum"],
		[]string{"fixture.echo"}, "minimapp.json")
	if err != nil {
		t.Fatal(err)
	}
	if res.Blocked {
		t.Fatalf("blocked: %s", res.Cause)
	}
	if conf.Pass {
		t.Fatal("conformance passed without the export")
	}
	// The failure must name the missing export — a build that fails
	// for any other reason (bad probe, missing files) is a broken
	// test, not a passing negative.
	combined := string(res.Stdout) + string(res.Stderr)
	if !strings.Contains(combined, "Implementation") {
		t.Errorf("failure evidence does not name the missing export:\n%s", combined)
	}
}
