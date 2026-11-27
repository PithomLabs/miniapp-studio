package hostexec

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ArmedState persists which capability class is live. Deployment ops
// disarm automatically after the first green suite run; re-arming is an
// owner-side file edit (the queue offers no re-arm operation — the
// fail-safe direction is one-way from the agent's side).
type ArmedState struct {
	path string
}

// OpenArmedState loads or initializes the state file (armed by default
// on first boot: the authorized deployment has not run yet).
func OpenArmedState(stateDir string) (*ArmedState, error) {
	p := filepath.Join(stateDir, "armed")
	a := &ArmedState{path: p}
	if _, err := os.Stat(p); err != nil {
		if err := os.WriteFile(p, []byte("deploy\n"), 0o644); err != nil {
			return nil, fmt.Errorf("init armed state: %w", err)
		}
		return a, nil
	}
	return a, nil
}

// DeployArmed reports whether deployment-only ops may run.
func (a *ArmedState) DeployArmed() bool {
	raw, err := os.ReadFile(a.path)
	if err != nil {
		return false
	}
	return string(raw) == "deploy\n"
}

// DisarmDeploy persists the steady-state posture. Irreversible via the
// queue; the owner re-arms by writing "deploy" to the state file.
func (a *ArmedState) DisarmDeploy() error {
	return os.WriteFile(a.path, []byte("steady\n"), 0o644)
}

// Broker is the serve loop over one queue.
type Broker struct {
	cfg   Config
	queue *Queue
	armed *ArmedState
	logf  func(string, ...any)
}

// Serve runs until ctx ends: claim, validate, policy-check, execute,
// publish. Single-threaded by design (single-flight needs nothing
// else), with an flock guard against a second broker on one queue.
func Serve(ctx context.Context, c Config, poll time.Duration, logf func(string, ...any)) error {
	q, err := OpenQueue(c.QueueDir, filepath.Join(c.StateDir, "results"))
	if err != nil {
		return err
	}
	armed, err := OpenArmedState(c.StateDir)
	if err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(c.StateDir, "broker.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("lock file: %w", err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("another broker holds this queue: %w", err)
	}
	if swept, err := q.SweepStartup(); err != nil {
		return err
	} else if len(swept) > 0 {
		logf("swept %d stale running jobs to UNKNOWN", len(swept))
	}
	// Preflight: refuse to serve when the daemon is unreachable, so
	// systemd restarts into a healthy state instead of failing jobs.
	if exit, _, errb, _, _ := Run(ctx, []string{c.DockerBin, "version", "--format", "{{.Server.Version}}"}, 30*time.Second); exit != 0 {
		return fmt.Errorf("docker preflight failed: %s", errb)
	}
	b := &Broker{cfg: c, queue: q, armed: armed, logf: logf}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		c, ok, err := q.Claim()
		if err != nil {
			logf("claim: %v", err)
		}
		if !ok {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(poll):
			}
			continue
		}
		b.execute(ctx, c)
	}
}

func (b *Broker) execute(ctx context.Context, c Claimed) {
	started := time.Now().UTC().Format(time.RFC3339)
	j, err := ParseJob(c.Raw)
	if err != nil {
		b.logf("reject %s: %v", c.ID, err)
		if rerr := b.queue.Reject(c, err.Error()); rerr != nil {
			b.logf("reject move %s: %v", c.ID, rerr)
		}
		return
	}
	if j.ID != c.ID {
		b.logf("reject %s: filename/id mismatch", c.ID)
		_ = b.queue.Reject(c, "filename/id mismatch")
		return
	}
	if IsDeployOp(j.Op) && !b.armed.DeployArmed() {
		b.logf("reject %s: deploy ops disarmed", c.ID)
		_ = b.queue.Reject(c, "deploy capabilities disarmed after workflow completion; owner re-arm required")
		return
	}
	res := Result{Op: j.Op, Started: started}
	b.runOp(ctx, j, &res)
	if err := b.queue.Finish(c, res); err != nil {
		b.logf("finish %s: %v", c.ID, err)
	}
	// Automatic disarm: the first green suite run completes the
	// authorized deployment.
	if j.Op == OpContainerRun && res.Status == StatusOK {
		if mode, _ := j.Params["mode"].(string); mode == ModeRun {
			b.recordDisarm(j, res)
			if err := b.armed.DisarmDeploy(); err != nil {
				b.logf("disarm: %v", err)
			} else {
				b.logf("deployment complete: deploy ops disarmed")
			}
		}
	}
}

// DisarmRecord freezes what the green run attested: source revision,
// tree state, executed image digest, run manifest, and package counts
// as observed. It is tamper-evidence, not trust: a hollow green still
// records faithfully, and owner acceptance must review the frozen
// diff. First record wins; later greens never overwrite it.
type DisarmRecord struct {
	JobID      string              `json:"job_id"`
	At         string              `json:"at_utc"`
	GitHead    string              `json:"git_head"`
	GitStatus  string              `json:"git_status"`
	ImageID    string              `json:"image_id"`
	Manifest   string              `json:"manifest"`
	PkgPass    int                 `json:"pkg_pass"`
	PkgFail    int                 `json:"pkg_fail"`
	PkgSkipped bool                `json:"pkg_skipped"`
	PkgNote    string              `json:"pkg_note,omitempty"`
	Results    map[string]FileHash `json:"results_manifest"`
}

// FileHash freezes one evidence file's identity at capture time. The
// hashes establish integrity FROM capture (later edits are detectable
// by recomparison), never authenticity BEFORE capture (bytes the agent
// wrote pre-capture are agent bytes).
type FileHash struct {
	SHA256 string `json:"sha256,omitempty"`
	Size   int64  `json:"size"`
	Status string `json:"status"` // ok|missing|unreadable|oversized|irregular
	Note   string `json:"note,omitempty"`
}

// Required evidence: the entrypoint writes exactly these on every run.
var requiredResults = []string{"gotest.json", "summary.txt", "exitcode"}

// Per-file hash cap. Suite evidence is kilobytes; anything past the
// cap is recorded as oversized, never read unbounded.
const maxResultBytes = 64 << 20

func (b *Broker) recordDisarm(j Job, res Result) {
	path := filepath.Join(b.cfg.StateDir, "disarm.json")
	if _, err := os.Stat(path); err == nil {
		return // first disarm stands
	}
	// Provenance reads the EXECUTED snapshot, never the live tree: the
	// single-threaded broker runs nothing between this run's staging
	// and this record, so stage-run/ is provably this run's bytes.
	stage := filepath.Join(b.cfg.StateDir, "stage-run")
	head, status := gitProvenance(b.cfg.GitBin, stage)
	pass, fail, skipped, note := packageCounts(filepath.Join(b.cfg.RepoRoot, ResultsDir, "gotest.json"))
	manifest := hashResultsDir(filepath.Join(b.cfg.RepoRoot, ResultsDir))
	for name, fh := range manifest {
		if fh.Status != "ok" {
			b.logf("disarm evidence gap: %s is %s (%s)", name, fh.Status, fh.Note)
		}
	}
	rec := DisarmRecord{
		JobID: j.ID, At: time.Now().UTC().Format(time.RFC3339),
		GitHead: head, GitStatus: status,
		ImageID: res.ImageID, Manifest: res.Manifest,
		PkgPass: pass, PkgFail: fail, PkgSkipped: skipped, PkgNote: note,
		Results: manifest,
	}
	raw, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		b.logf("disarm record encode: %v", err)
		return
	}
	tmp, err := os.CreateTemp(b.cfg.StateDir, "disarm.*.tmp")
	if err != nil {
		b.logf("disarm record temp: %v", err)
		return
	}
	name := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		os.Remove(name)
		b.logf("disarm record write: %v", err)
		return
	}
	tmp.Close()
	// Best-effort first-wins: a concurrent record (second broker
	// despite the lock) loses the rename race harmlessly — either
	// record describes a green run.
	os.Rename(name, path)
}

// gitProvenance captures revision and tree state, or records exactly
// why it could not. Failures degrade the record, never block disarm:
// disarm reduces capability, so blocking it would be the unsafe
// direction.
func gitProvenance(gitBin, repo string) (head, status string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	head = gitOut(ctx, gitBin, repo, "rev-parse", "HEAD")
	status = gitOut(ctx, gitBin, repo, "status", "--porcelain")
	if len(status) > 8192 {
		status = status[:8192] + "\n…[truncated]"
	}
	return head, status
}

// hashResultsDir freezes the required evidence files' identities.
// Every required name gets exactly one entry: ok with hash+size, or
// an explicit gap status. Reads are streaming and capped; symlinks,
// dirs, and special files are never followed.
func hashResultsDir(dir string) map[string]FileHash {
	out := map[string]FileHash{}
	for _, name := range requiredResults {
		p := filepath.Join(dir, name)
		fi, err := os.Lstat(p)
		if err != nil {
			out[name] = FileHash{Status: "missing", Note: err.Error()}
			continue
		}
		if !fi.Mode().IsRegular() {
			out[name] = FileHash{Status: "irregular", Note: fmt.Sprintf("mode %s", fi.Mode().String())}
			continue
		}
		if fi.Size() > maxResultBytes {
			out[name] = FileHash{Size: fi.Size(), Status: "oversized", Note: fmt.Sprintf("over %d-byte cap", maxResultBytes)}
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			out[name] = FileHash{Status: "unreadable", Note: err.Error()}
			continue
		}
		h := sha256.New()
		n, err := io.Copy(h, io.LimitReader(f, maxResultBytes+1))
		f.Close()
		if err != nil {
			out[name] = FileHash{Status: "unreadable", Note: err.Error()}
			continue
		}
		if n > maxResultBytes {
			out[name] = FileHash{Size: n, Status: "oversized", Note: "grew past cap during read"}
			continue
		}
		out[name] = FileHash{SHA256: hex.EncodeToString(h.Sum(nil)), Size: n, Status: "ok"}
	}
	return out
}

func gitOut(ctx context.Context, gitBin, repo string, args ...string) string {
	full := append([]string{"-C", repo}, args...)
	cmd := exec.CommandContext(ctx, gitBin, full...)
	raw, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("unavailable: %v", err)
	}
	out := strings.TrimRight(string(raw), "\n")
	if out == "" {
		return "(clean)"
	}
	return out
}

// packageCounts parses package-level pass/fail lines from a go test
// -json stream, mirroring the entrypoint's summary filter. The file is
// agent-writable, so counts are frozen observations for owner review,
// never independent proof. Reads are size-capped.
func packageCounts(path string) (pass, fail int, skipped bool, note string) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, true, "gotest.json absent: " + err.Error()
	}
	defer f.Close()
	sc := bufio.NewScanner(io.LimitReader(f, 16<<20))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	capped := false
	for sc.Scan() {
		line := sc.Bytes()
		if !bytes.Contains(line, []byte(`"Package":`)) || bytes.Contains(line, []byte(`"Test":`)) {
			continue
		}
		switch {
		case bytes.Contains(line, []byte(`"Action":"pass"`)):
			pass++
		case bytes.Contains(line, []byte(`"Action":"fail"`)):
			fail++
		case bytes.Contains(line, []byte(`"Action":"skip"`)):
			skipped = true
		}
	}
	if err := sc.Err(); err != nil {
		capped = true
		note = "stream capped mid-parse: " + err.Error()
	}
	if f2, err := os.Stat(path); err == nil && f2.Size() > 16<<20 {
		capped = true
		note = "gotest.json exceeds 16MiB cap; counts partial"
	}
	_ = capped
	return pass, fail, skipped, note
}

func (b *Broker) runOp(ctx context.Context, j Job, res *Result) {
	switch j.Op {
	case OpImageBuild:
		b.runBuild(ctx, j, res)
	case OpContainerRm:
		b.runSweepOp(ctx, res)
	case OpContainerRun:
		b.runStaged(ctx, j, res)
	default:
		argv, err := Argv(b.cfg, j, "")
		if err != nil {
			res.Status, res.Reason = StatusError, err.Error()
			return
		}
		exit, out, errb, trunc, timedOut := Run(ctx, argv, opTimeout(j.Op))
		res.Exit, res.Stdout, res.Stderr, res.Trunc = exit, out, errb, trunc
		if timedOut {
			res.Status, res.Reason = StatusUnknown, "docker timeout; reconcile via inspect.readonly"
		} else if exit != 0 {
			res.Status = StatusError
		} else {
			res.Status = StatusOK
		}
	}
}

// runBuild stages the verified context, builds the image from the stage
// only, and records the resulting image ID for run binding.
func (b *Broker) runBuild(ctx context.Context, j Job, res *Result) {
	_ = j
	stage := filepath.Join(b.cfg.StateDir, "stage-image")
	if err := StageContext(ctx, b.cfg.RepoRoot, stage, b.cfg.Pins); err != nil {
		res.Status, res.Reason = StatusError, err.Error()
		return
	}
	argv, err := Argv(b.cfg, Job{Op: OpImageBuild}, stage)
	if err != nil {
		res.Status, res.Reason = StatusError, err.Error()
		return
	}
	exit, out, errb, trunc, timedOut := Run(ctx, argv, opTimeout(OpImageBuild))
	res.Exit, res.Stdout, res.Stderr, res.Trunc = exit, out, errb, trunc
	if timedOut {
		res.Status, res.Reason = StatusUnknown, "build timed out; reconcile via inspect.readonly"
		return
	}
	if exit != 0 {
		res.Status = StatusError
		return
	}
	ixit, idOut, _, _, _ := Run(ctx, []string{b.cfg.DockerBin, "inspect", "--format", "{{.Id}}", ImageTag}, time.Minute)
	if ixit != 0 {
		res.Status, res.Reason = StatusUnknown, "build ok but image inspect failed; digest unrecorded"
		return
	}
	id := trimNL(idOut)
	if err := writeVerifiedImage(b.cfg.StateDir, id); err != nil {
		res.Status, res.Reason = StatusError, err.Error()
		return
	}
	res.Status, res.ImageID = StatusOK, id
}

// runStaged snapshots the repo, binds the run to the verified digest,
// and executes. The container mounts the snapshot, never the live tree.
func (b *Broker) runStaged(ctx context.Context, j Job, res *Result) {
	stage := filepath.Join(b.cfg.StateDir, "stage-run")
	manifest, err := StageRepo(ctx, b.cfg.RepoRoot, stage)
	if err != nil {
		res.Status, res.Reason = StatusError, err.Error()
		return
	}
	res.Manifest = manifest
	want, err := readVerifiedImage(b.cfg.StateDir)
	if err != nil {
		res.Status, res.Reason = StatusError, "no verified build on record; run image.build first"
		return
	}
	ixit, idOut, _, _, _ := Run(ctx, []string{b.cfg.DockerBin, "inspect", "--format", "{{.Id}}", ImageTag}, time.Minute)
	if ixit != 0 {
		res.Status, res.Reason = StatusError, "cannot resolve image tag; reconcile via inspect.readonly"
		return
	}
	if got := trimNL(idOut); got != want {
		res.Status, res.Reason = StatusError,
			fmt.Sprintf("image tag resolves to %s, verified build is %s; refusing stale image", got, want)
		return
	}
	res.ImageID = want
	argv, err := Argv(b.cfg, j, stage)
	if err != nil {
		res.Status, res.Reason = StatusError, err.Error()
		return
	}
	b.runContainer(ctx, argv, res)
}

// runSweepOp lists candidate containers and removes only verified
// broker-generated names.
func (b *Broker) runSweepOp(ctx context.Context, res *Result) {
	argv, err := Argv(b.cfg, Job{Op: OpContainerRm}, "")
	if err != nil {
		res.Status, res.Reason = StatusError, err.Error()
		return
	}
	exit, out, errb, trunc, _ := Run(ctx, argv, opTimeout(OpContainerRm))
	if exit != 0 {
		res.Exit, res.Stdout, res.Stderr, res.Trunc = exit, out, errb, trunc
		res.Status = StatusError
		return
	}
	removed := []string{}
	for _, name := range SweepNames(out) {
		rexit, _, rerr, _, _ := Run(ctx, []string{b.cfg.DockerBin, "rm", "-f", name}, time.Minute)
		if rexit != 0 {
			res.Status, res.Reason = StatusUnknown, fmt.Sprintf("partial sweep, removed %s then rm %s failed: %s; reconcile via inspect.readonly", joinComma(removed), name, rerr)
			res.Stdout = "removed: " + joinComma(removed)
			return
		}
		removed = append(removed, name)
	}
	res.Status, res.Stdout = StatusOK, "removed: "+joinComma(removed)
}

// runContainer executes a test run and reaps orphans on timeout.
func (b *Broker) runContainer(ctx context.Context, argv []string, res *Result) {
	exit, out, errb, trunc, timedOut := Run(ctx, argv, opTimeout(OpContainerRun))
	res.Exit, res.Stdout, res.Stderr, res.Trunc = exit, out, errb, trunc
	if timedOut {
		b.bestEffortSweep(ctx)
		res.Status, res.Reason = StatusUnknown, "container timed out; orphans reaped; reconcile via inspect.readonly"
	} else if exit != 0 {
		res.Status = StatusError
	} else {
		res.Status = StatusOK
	}
}

func (b *Broker) bestEffortSweep(ctx context.Context) {
	exit, out, _, _, _ := Run(ctx, []string{b.cfg.DockerBin, "ps", "-aq", "--filter", "name=" + ContainerBase, "--format", "{{.Names}}"}, time.Minute)
	if exit != 0 {
		return
	}
	for _, name := range SweepNames(out) {
		_, _, _, _, _ = Run(ctx, []string{b.cfg.DockerBin, "rm", "-f", name}, time.Minute)
	}
}

// verifiedImagePath is the owner-side record of the image ID the last
// verified build produced. Runs bind to it (see runStaged).
func verifiedImagePath(stateDir string) string {
	return filepath.Join(stateDir, "verified-image")
}

func writeVerifiedImage(stateDir, id string) error {
	tmp, err := os.CreateTemp(stateDir, "verified-image.*.tmp")
	if err != nil {
		return fmt.Errorf("verified record temp: %w", err)
	}
	name := tmp.Name()
	if _, err := fmt.Fprintf(tmp, "%s\n%s\n", id, time.Now().UTC().Format(time.RFC3339)); err != nil {
		tmp.Close()
		os.Remove(name)
		return fmt.Errorf("verified record write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return fmt.Errorf("verified record close: %w", err)
	}
	if err := os.Rename(name, verifiedImagePath(stateDir)); err != nil {
		os.Remove(name)
		return fmt.Errorf("verified record publish: %w", err)
	}
	return nil
}

func readVerifiedImage(stateDir string) (string, error) {
	raw, err := os.ReadFile(verifiedImagePath(stateDir))
	if err != nil {
		return "", err
	}
	line, _, _ := strings.Cut(string(raw), "\n")
	if line == "" {
		return "", fmt.Errorf("verified record empty")
	}
	return line, nil
}

func trimNL(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		return s[:len(s)-1]
	}
	return s
}

func joinComma(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ","
		}
		out += s
	}
	if out == "" {
		return "(none)"
	}
	return out
}
