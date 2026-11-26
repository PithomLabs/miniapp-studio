package sandbox

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PithomLabs/miniapp-studio/artifact"
)

// ToolchainImage is the Phase D test image, built from
// testdata/toolchain/Dockerfile. Runs resolve it to an image id per
// execution and run the id (immutable); tags are never trusted
// across the resolve/run gap.
const ToolchainImage = "miniapp-phase-d-toolchain:local"

// Container user: non-root, matching the image's baked tester.
const containerUser = "65532:65532"

// Resource and output bounds (review gate C).
const (
	defaultPidsLimit = "256"
	defaultMemory    = "1g"
	defaultCPUs      = "1.0"
	defaultTimeout   = 5 * time.Minute
	// MaxStreamBytes caps each captured stream; MaxTotalBytes caps
	// the pair. Anything beyond records truncation (never silent).
	MaxStreamBytes = 1 << 20
	MaxTotalBytes  = 2 << 20
)

// runEnv is the complete container environment allowlist. docker run
// never inherits host env; exactly these keys are passed with -e.
// GOMODCACHE pins the pre-seeded module cache; CGO_ENABLED=0 fails
// cgo closed even if admission missed it; -mod=readonly refuses to
// repair go.mod/go.sum (verified artifacts arrive complete).
var runEnv = map[string]string{
	"PATH":        "/usr/local/go/bin:/usr/bin:/bin",
	"GOPROXY":     "off",
	"GOSUMDB":     "off",
	"GOFLAGS":     "-mod=readonly",
	"GOTOOLCHAIN": "local",
	"GOCACHE":     "/tmp/gocache",
	"GOMODCACHE":  "/go/pkg/mod",
	"CGO_ENABLED": "0",
	"HOME":        "/home/tester",
	"TMPDIR":      "/tmp",
}

// BreachMarker is printed by the egress canary when a dial succeeds.
// The runner scans every byte of container output for it: a match
// fails the run and quarantines the runner until re-proven.
const BreachMarker = "SANDBOX-BREACH: egress succeeded"

// Config tunes the runner. Zero values take the documented defaults.
type Config struct {
	// DockerBin is the docker client path (default "docker").
	DockerBin string
	// Timeout bounds one container execution (default 5m).
	Timeout time.Duration
}

// Request is one container execution.
type Request struct {
	// Files maps bundle-relative path → bytes, materialized
	// read-only at /work. Paths are re-validated defensively.
	Files map[string][]byte
	// Scratch maps scratch-relative path → Studio-authored bytes
	// (harness modules, never artifact content), materialized
	// writable at /scratch. Paths are re-validated defensively.
	Scratch map[string][]byte
	// Workdir is the container working directory: /work/... for
	// artifact commands, /scratch/... for harness commands.
	Workdir string
	// Command is the argv to run (no shell).
	Command []string
}

// Result is the observed outcome. Blocked means no container ran
// (Cause explains); otherwise ExitCode/Output describe the run and
// Truncated marks capped output.
type Result struct {
	Blocked   bool
	Cause     string
	ExitCode  int
	Stdout    []byte
	Stderr    []byte
	Truncated bool
}

// RunnerAPI is the execution seam: *Runner is the production
// implementation; tests substitute fakes.
type RunnerAPI interface {
	Run(ctx context.Context, req Request) (Result, error)
}

// Runner executes requests in disposable containers. The zero value
// is not usable; use NewRunner. Runner is safe for concurrent use;
// quarantine state is process-local (a restart re-proves by
// re-running the canary).
type Runner struct {
	cfg         Config
	dockerBin   string
	timeout     time.Duration
	mu          sync.Mutex
	quarantined bool
	// runCmd executes one docker invocation; stubbed in unit tests.
	runCmd func(ctx context.Context, args []string) cmdResult
}

// cmdResult is one docker invocation outcome. Truncated reports
// capped-writer overflow (measured, never inferred from size).
type cmdResult struct {
	stdout    []byte
	stderr    []byte
	exit      int
	truncated bool
	err       error
}

// NewRunner validates configuration without touching Docker: even
// construction performs no I/O, so blocked-state tests need no daemon.
func NewRunner(cfg Config) *Runner {
	bin := cfg.DockerBin
	if bin == "" {
		bin = "docker"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	r := &Runner{dockerBin: bin, timeout: timeout, cfg: cfg}
	r.runCmd = r.execDocker
	return r
}

// Quarantined reports whether an egress breach quarantined the runner.
func (r *Runner) Quarantined() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.quarantined
}

// Reprove clears quarantine after the canary passes again. Callers
// must actually re-run the canary first; this only clears the flag.
func (r *Runner) Reprove() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.quarantined = false
}

func (r *Runner) quarantine() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.quarantined = true
}

// execDocker runs one docker CLI invocation with capped output.
func (r *Runner) execDocker(ctx context.Context, args []string) cmdResult {
	cmd := exec.CommandContext(ctx, r.dockerBin, args...)
	var outBuf, errBuf bytes.Buffer
	outW := &cappedWriter{buf: &outBuf, cap: MaxStreamBytes}
	errW := &cappedWriter{buf: &errBuf, cap: MaxStreamBytes}
	cmd.Stdout, cmd.Stderr = outW, errW
	runErr := cmd.Run()
	truncated := outW.truncated || errW.truncated ||
		outBuf.Len()+errBuf.Len() >= MaxTotalBytes
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return cmdResult{stdout: outBuf.Bytes(), stderr: errBuf.Bytes(),
				exit: exitErr.ExitCode(), truncated: truncated}
		}
		return cmdResult{stdout: outBuf.Bytes(), stderr: errBuf.Bytes(),
			exit: -1, truncated: truncated, err: runErr}
	}
	return cmdResult{stdout: outBuf.Bytes(), stderr: errBuf.Bytes(),
		exit: 0, truncated: truncated}
}

// cappedWriter keeps at most cap bytes; overflow is discarded and
// flagged so callers can record truncation honestly.
type cappedWriter struct {
	buf       *bytes.Buffer
	cap       int
	truncated bool
}

func (w *cappedWriter) Write(p []byte) (int, error) {
	room := w.cap - w.buf.Len()
	if room <= 0 {
		w.truncated = true
		return len(p), nil
	}
	if len(p) > room {
		w.truncated = true
		p = p[:room]
	}
	return w.buf.Write(p)
}

// checkPrereqs verifies docker presence, daemon security options,
// and image identity/labels. Any failure returns a blocked cause;
// nothing has executed at this point.
func (r *Runner) checkPrereqs(ctx context.Context) (imageID string, cause string) {
	if _, err := exec.LookPath(r.dockerBin); err != nil {
		return "", "docker client unavailable: " + err.Error()
	}
	info := r.runCmd(ctx, []string{"info", "--format", "{{json .SecurityOptions}}"})
	if info.err != nil || info.exit != 0 {
		return "", "docker daemon unreachable"
	}
	var opts []string
	if err := json.Unmarshal(info.stdout, &opts); err != nil {
		return "", "docker info unreadable"
	}
	joined := strings.Join(opts, " ")
	if !strings.Contains(joined, "seccomp") {
		return "", "daemon lacks seccomp enforcement"
	}
	if !strings.Contains(joined, "apparmor") && !strings.Contains(joined, "selinux") {
		return "", "daemon lacks apparmor/selinux confinement"
	}
	inspect := r.runCmd(ctx, []string{"image", "inspect",
		"--format", "{{.Id}}|{{json .Config.Labels}}", ToolchainImage})
	if inspect.err != nil || inspect.exit != 0 {
		return "", "toolchain image unverifiable"
	}
	parts := strings.SplitN(string(inspect.stdout), "|", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", "toolchain image inspect unreadable"
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(parts[1]), &labels); err != nil {
		return "", "toolchain image labels unreadable"
	}
	if labels["studio.contract"] != artifact.ContractVersion ||
		labels["studio.toolchain.go"] != artifact.PinnedGoVersion ||
		labels["studio.toolchain.miniapp"] != artifact.PinnedMiniapp {
		return "", "toolchain image labels do not match the pinned toolchain"
	}
	return strings.TrimSpace(parts[0]), ""
}

// Run executes req in a fresh disposable container, enforcing every
// gate-C control. It returns a blocked Result (never an error) when
// prerequisites fail or quarantine holds; hard runner failures
// (workdir I/O) return errors.
func (r *Runner) Run(ctx context.Context, req Request) (Result, error) {
	if r.Quarantined() {
		return Result{Blocked: true, Cause: "runner quarantined after egress breach; re-prove required"}, nil
	}
	if len(req.Command) == 0 {
		return Result{}, fmt.Errorf("sandbox: empty command")
	}
	// Clean before prefix-matching: "/work/../etc" must not pass.
	// The cleaned path is what ships to docker, so equivalent
	// spellings ("/scratch/../work" → "/work") resolve safely.
	req.Workdir = path.Clean("/" + req.Workdir)
	if req.Workdir != "/work" && !strings.HasPrefix(req.Workdir, "/work/") &&
		req.Workdir != "/scratch" && !strings.HasPrefix(req.Workdir, "/scratch/") {
		return Result{}, fmt.Errorf("sandbox: workdir outside /work or /scratch")
	}
	for path := range req.Files {
		if err := artifact.ValidatePath(path); err != nil {
			return Result{}, fmt.Errorf("sandbox: refusing unsafe member path: %v", err)
		}
	}
	for path := range req.Scratch {
		if err := artifact.ValidatePath(path); err != nil {
			return Result{}, fmt.Errorf("sandbox: refusing unsafe scratch path: %v", err)
		}
	}
	workdir, err := os.MkdirTemp("", "phase-d-run-")
	if err != nil {
		return Result{}, fmt.Errorf("sandbox: workdir: %w", err)
	}
	defer func() { _ = os.RemoveAll(workdir) }()
	scratchdir, err := os.MkdirTemp("", "phase-d-scratch-")
	if err != nil {
		return Result{}, fmt.Errorf("sandbox: scratchdir: %w", err)
	}
	defer func() { _ = os.RemoveAll(scratchdir) }()
	// Staging is world-accessible by necessity: the container runs
	// as unprivileged uid 65532, and host-0700 dirs would deny it.
	// Random names under sticky /tmp plus non-secret contents
	// (artifact bytes live in Postgres too) make this safe; the
	// alternative — running as the Studio uid — would hand a
	// container escape the keys to /tmp/boundary-b and friends.
	if err := os.Chmod(workdir, 0o755); err != nil {
		return Result{}, fmt.Errorf("sandbox: dir mode: %w", err)
	}
	if err := os.Chmod(scratchdir, 0o777); err != nil {
		return Result{}, fmt.Errorf("sandbox: scratch mode: %w", err)
	}
	for path, data := range req.Files {
		if err := writeMember(workdir, path, data); err != nil {
			return Result{}, err
		}
	}
	for path, data := range req.Scratch {
		if err := writeMember(scratchdir, path, data); err != nil {
			return Result{}, err
		}
	}
	imageID, cause := r.checkPrereqs(ctx)
	if cause != "" {
		return Result{Blocked: true, Cause: cause}, nil
	}
	raw := make([]byte, 4)
	if _, err := rand.Read(raw); err != nil {
		return Result{}, fmt.Errorf("sandbox: run id: %w", err)
	}
	name := "phase-d-run-" + hex.EncodeToString(raw)
	args := r.containerArgs(name, imageID, workdir, scratchdir, req)
	runCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	defer func() { _ = r.runCmd(context.Background(), []string{"rm", "-f", name}) }()
	got := r.runCmd(runCtx, args)
	// Single redaction choke point: container output is untrusted
	// bytes, and nothing raw ever leaves the runner — timeout,
	// breach, and success paths alike.
	got.stdout = []byte(artifact.RedactSpans(string(got.stdout)))
	got.stderr = []byte(artifact.RedactSpans(string(got.stderr)))
	// Deadline expiry surfaces as a killed docker CLI (signal exit),
	// not a Go error — so check the context first, before err.
	if runCtx.Err() != nil {
		_ = r.runCmd(context.Background(), []string{"kill", name})
		if ctx.Err() != nil {
			return Result{}, fmt.Errorf("sandbox: parent context ended: %w", ctx.Err())
		}
		return Result{ExitCode: -1,
			Cause:  "timeout: container killed after " + r.timeout.String(),
			Stdout: got.stdout,
			Stderr: got.stderr,
		}, nil
	}
	if got.err != nil {
		return Result{}, fmt.Errorf("sandbox: container exec: %w", got.err)
	}
	if bytes.Contains(got.stdout, []byte(BreachMarker)) || bytes.Contains(got.stderr, []byte(BreachMarker)) {
		r.quarantine()
		return Result{ExitCode: got.exit, Stdout: got.stdout, Stderr: got.stderr, Truncated: got.truncated,
			Cause: "egress breach observed; runner quarantined"}, nil
	}
	return Result{ExitCode: got.exit, Stdout: got.stdout, Stderr: got.stderr, Truncated: got.truncated}, nil
}

// writeMember writes one validated member file into a staging dir.
// World-readable (see the staging comment): the container uid must
// traverse and read what Studio stages.
func writeMember(dir, path string, data []byte) error {
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("sandbox: staging layout: %w", err)
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return fmt.Errorf("sandbox: staging write: %w", err)
	}
	return nil
}

// containerArgs builds the docker run invocation. Every gate-C
// control maps to a flag here; TestContainerArgs pins them all.
func (r *Runner) containerArgs(name, imageID, workdir, scratchdir string, req Request) []string {
	args := []string{"run", "--rm", "--name", name,
		"--network=none",
		"--user", containerUser,
		"--cap-drop=ALL",
		"--security-opt", "no-new-privileges",
		"--pids-limit", defaultPidsLimit,
		"--memory", defaultMemory,
		"--cpus", defaultCPUs,
		"--read-only",
		"--tmpfs", "/tmp:rw,exec,size=1g",
		"-v", workdir + ":/work:ro",
		"-v", scratchdir + ":/scratch:rw",
		"-w", req.Workdir,
	}
	// /tmp stays executable: Go test binaries execute from the
	// build cache under /tmp, so noexec would refuse every run.
	// Containment comes from network/fs/user/capability controls.
	for _, key := range []string{"CGO_ENABLED", "GOCACHE", "GOFLAGS", "GOMODCACHE",
		"GOPROXY", "GOSUMDB", "GOTOOLCHAIN", "HOME", "PATH", "TMPDIR"} {
		args = append(args, "-e", key+"="+runEnv[key])
	}
	args = append(args, imageID)
	return append(args, req.Command...)
}
