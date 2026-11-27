package hostexec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// Config is the owner-side broker configuration. It lives OUTSIDE the
// workspace (the agent must never be able to rewrite pins or paths).
type Config struct {
	// RepoRoot is the checkout the broker operates on.
	RepoRoot string `json:"repo_root"`
	// QueueDir holds pending/running/done/rejected (inside the repo).
	QueueDir string `json:"queue_dir"`
	// StateDir holds armed state, lock, and full logs (owner-side).
	StateDir string `json:"state_dir"`
	// EnvFile is the 0600 DSN file passed to test containers by path.
	EnvFile string `json:"env_file"`
	// ModCache is the read-only module cache mount source.
	ModCache string `json:"mod_cache"`
	// Toolchain is the read-only Go toolchain mount source.
	Toolchain string `json:"toolchain"`
	// Pins maps pin inputs (see PinInputs) to sha256 hex digests,
	// recorded by the owner at install from the reviewed tree.
	Pins map[string]string `json:"pins"`
	// DockerBin is the docker executable (validated once at startup).
	DockerBin string `json:"docker_bin"`
	// GitBin is the git executable for disarm provenance capture
	// (validated once at startup; read-only invocations only).
	GitBin string `json:"git_bin"`
	// ExtraAgentRoots names additional agent-writable roots the
	// install declares (sandbox temp dirs beyond /tmp). Owner-side
	// state is refused under any of them.
	ExtraAgentRoots []string `json:"extra_agent_roots,omitempty"`
}

// LoadConfig reads and validates owner-side configuration. Every path
// must exist with the expected shape; the env file must be
// owner-readable-only. System agent roots (/tmp, /dev/shm) always
// deny owner-side state; installs add their sandbox's roots via
// ExtraAgentRoots.
func LoadConfig(path string) (Config, error) {
	return loadConfig(path, systemAgentRoots)
}

func loadConfig(path string, sysDeny []string) (Config, error) {
	var c Config
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	for _, dir := range []string{c.RepoRoot, c.StateDir, c.ModCache, c.Toolchain} {
		fi, err := os.Stat(dir)
		if err != nil || !fi.IsDir() {
			return Config{}, fmt.Errorf("config dir %q missing", dir)
		}
	}
	fi, err := os.Stat(c.EnvFile)
	if err != nil || !fi.Mode().IsRegular() {
		return Config{}, fmt.Errorf("env file %q missing", c.EnvFile)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return Config{}, fmt.Errorf("env file %q must not be group/world accessible", c.EnvFile)
	}
	if err := checkOwner(c.EnvFile, fi); err != nil {
		return Config{}, err
	}
	if err := checkPlacement(c, sysDeny); err != nil {
		return Config{}, err
	}
	if _, err := exec.LookPath(c.DockerBin); err != nil {
		return Config{}, fmt.Errorf("docker bin %q: %w", c.DockerBin, err)
	}
	if _, err := exec.LookPath(c.GitBin); err != nil {
		return Config{}, fmt.Errorf("git bin %q: %w", c.GitBin, err)
	}
	walked, err := WalkContext(c.RepoRoot)
	if err != nil {
		return Config{}, fmt.Errorf("pins: %w", err)
	}
	if len(c.Pins) != len(walked) {
		return Config{}, fmt.Errorf("pins cover %d inputs, context holds %d (re-pin under owner review)", len(c.Pins), len(walked))
	}
	for _, rel := range walked {
		if len(c.Pins[rel]) != 64 {
			return Config{}, fmt.Errorf("pin for %s malformed", rel)
		}
	}
	return c, nil
}

// System-known agent-accessible roots. The workspace itself (RepoRoot)
// is added by checkPlacement; installs declare anything else via
// ExtraAgentRoots.
var systemAgentRoots = []string{"/tmp", "/dev/shm"}

// checkPlacement enforces the trust geography on RESOLVED paths:
// QueueDir inside the repo; StateDir and EnvFile outside the repo AND
// outside every agent root; EnvFile outside any checkout. Symlinks are
// resolved first, so link-based placement lies fail.
func checkPlacement(c Config, sysDeny []string) error {
	repo, err := filepath.EvalSymlinks(c.RepoRoot)
	if err != nil {
		return fmt.Errorf("resolve repo: %w", err)
	}
	queue, err := filepath.EvalSymlinks(c.QueueDir)
	if err != nil {
		// QueueDir may not exist yet (OpenQueue creates it):
		// resolve its parent and re-attach the leaf.
		parent, perr := filepath.EvalSymlinks(filepath.Dir(c.QueueDir))
		if perr != nil {
			return fmt.Errorf("resolve queue dir: %w", perr)
		}
		queue = filepath.Join(parent, filepath.Base(c.QueueDir))
	}
	if !underDir(queue, repo) {
		return fmt.Errorf("queue dir %q outside repo", c.QueueDir)
	}
	state, err := filepath.EvalSymlinks(c.StateDir)
	if err != nil {
		return fmt.Errorf("resolve state dir: %w", err)
	}
	env, err := filepath.EvalSymlinks(c.EnvFile)
	if err != nil {
		return fmt.Errorf("resolve env file: %w", err)
	}
	denies := append([]string{repo}, sysDeny...)
	for _, r := range c.ExtraAgentRoots {
		if !filepath.IsAbs(r) {
			return fmt.Errorf("agent root %q not absolute", r)
		}
		denies = append(denies, filepath.Clean(r))
	}
	for _, target := range []struct {
		what, path string
	}{{"state dir", state}, {"env file", env}} {
		for _, d := range denies {
			if target.path == d || underDir(target.path, d) || underDir(d, target.path) {
				return fmt.Errorf("%s %q overlaps agent-accessible %q", target.what, target.path, d)
			}
		}
	}
	if hasGitAncestor(env) {
		return fmt.Errorf("env file %q sits inside version control", c.EnvFile)
	}
	if fi, err := os.Stat(c.StateDir); err != nil || fi.Mode().Perm()&0o002 != 0 {
		return fmt.Errorf("state dir %q must not be world-writable", c.StateDir)
	}
	return nil
}

// checkOwner requires path to be owned by the broker's own uid. (Against
// the same-uid agent the path placement is the real boundary; ownership
// and modes defend against other local users.)
func checkOwner(path string, fi os.FileInfo) error {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("owner of %q unknowable", path)
	}
	if !uidMatches(st.Uid, uint32(os.Geteuid())) {
		return fmt.Errorf("%q owned by uid %d, want %d", path, st.Uid, os.Geteuid())
	}
	return nil
}

func uidMatches(fileUID, euid uint32) bool { return fileUID == euid }

// underDir reports whether target is strictly under dir (separator
// boundary: /tmpfoo is NOT under /tmp).
func underDir(target, dir string) bool {
	if target == dir {
		return false
	}
	rel, err := filepath.Rel(dir, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

// hasGitAncestor reports whether any ancestor directory of path holds
// a .git entry (file or dir).
func hasGitAncestor(path string) bool {
	dir := filepath.Dir(path)
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

// Op timeouts. Suite runs are the long pole; everything else is fast.
func opTimeout(op string) time.Duration {
	switch op {
	case OpContainerRun:
		return 30 * time.Minute
	case OpImageBuild:
		return 10 * time.Minute
	default:
		return 60 * time.Second
	}
}

// Argv builds the exact docker invocation for a validated job. Every
// element is a constant, a validated config value, or broker-generated
// (container names, timestamps, stage paths). NOTHING job-supplied
// reaches argv except the run mode, which is enum-pinned by ParseJob.
// stage is the owner-side snapshot the operation consumes: the image
// context for builds, the repo mount source for runs. Image builds and
// container runs never touch the live tree.
func Argv(c Config, j Job, stage string) ([]string, error) {
	d := c.DockerBin
	results := filepath.Join(c.RepoRoot, ResultsDir)
	switch j.Op {
	case OpVersion:
		return []string{d, "version", "--format", "{{.Server.Version}}"}, nil
	case OpNetCreate:
		return []string{d, "network", "create", "--internal",
			"--label", NetLabel, NetName}, nil
	case OpNetConnect:
		return []string{d, "network", "connect", "--alias", PgAlias, NetName, PgContainer}, nil
	case OpNetDisconnect:
		return []string{d, "network", "disconnect", NetName, PgContainer}, nil
	case OpNetRemove:
		return []string{d, "network", "rm", NetName}, nil
	case OpImageBuild:
		if stage == "" {
			return nil, fmt.Errorf("build needs a staged context")
		}
		// Explicit -f with an absolute broker-composed path: docker
		// defaults to PATH/Dockerfile, which does not exist (ours is
		// Containerfile), and the broker's working directory is
		// undefined under systemd. The target is guaranteed present
		// by the pin-set match in StageContext.
		return []string{d, "build", "-f", filepath.Join(stage, "Containerfile"), "-t", ImageTag, stage}, nil
	case OpContainerRm:
		// Sweep reaps broker-generated names only (see SweepNames);
		// this argv lists candidates, and rm runs per verified name.
		return []string{d, "ps", "-aq", "--filter", "name=" + ContainerBase, "--format", "{{.Names}}"}, nil
	case OpContainerRun:
		mode, _ := j.Params["mode"].(string)
		if stage == "" {
			return nil, fmt.Errorf("run needs a staged snapshot")
		}
		name := ContainerBase + "netcheck"
		if mode == ModeRun {
			name = fmt.Sprintf("%srun-%d", ContainerBase, time.Now().UnixNano())
		}
		return []string{d, "run", "--rm",
			"--network", NetName,
			"--name", name,
			"-v", stage + ":/repo:ro",
			"-v", c.Toolchain + ":" + c.Toolchain + ":ro",
			"-v", c.ModCache + ":" + c.ModCache + ":ro",
			"-v", results + ":/results:rw",
			"--env-file", c.EnvFile,
			"-e", "GOROOT=" + c.Toolchain,
			"-e", "GOMODCACHE=" + c.ModCache,
			ImageTag, mode,
		}, nil
	case OpInspectReadonly:
		what, _ := j.Params["what"].(string)
		switch what {
		case InspectNetLs:
			return []string{d, "network", "ls"}, nil
		case InspectNetInspect:
			return []string{d, "network", "inspect", NetName}, nil
		case InspectPsTest:
			return []string{d, "ps", "--filter", "name=" + ContainerBase, "--format", "{{.Names}}\t{{.Status}}"}, nil
		case InspectPgNets:
			return []string{d, "inspect", "-f", "{{range $k,$v := .NetworkSettings.Networks}}{{$k}} {{end}}", PgContainer}, nil
		case InspectPgPorts:
			return []string{d, "port", PgContainer}, nil
		}
	}
	return nil, fmt.Errorf("no argv for op %q", j.Op)
}

// SweepNames filters candidate container names to broker-generated
// test names. Anything else is never removed, whatever docker lists.
func SweepNames(out string) []string {
	var names []string
	for _, line := range strings.Split(out, "\n") {
		if rxSweepName.MatchString(strings.TrimSpace(line)) {
			names = append(names, strings.TrimSpace(line))
		}
	}
	return names
}

var rxDSN = regexp.MustCompile(`postgres(ql)?://[^\s"']+`)

// Scrub redacts DSN-shaped secrets from captured output (defense in
// depth — jobs never carry values and argv never embeds them).
func Scrub(s string) string { return rxDSN.ReplaceAllString(s, "[REDACTED-DSN]") }

// Tail bounds captured output, reporting whether it truncated.
func Tail(s string) (string, bool) {
	if len(s) <= MaxResultBytes {
		return s, false
	}
	return s[len(s)-MaxResultBytes:], true
}

// Run executes argv with a timeout, returning exit code and scrubbed,
// bounded streams. No shell is involved: argv runs directly.
func Run(ctx context.Context, argv []string, timeout time.Duration) (exit int, stdout, stderr string, trunc, timedOut bool) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	stdout, t1 := Tail(Scrub(out.String()))
	stderr, t2 := Tail(Scrub(errb.String()))
	trunc = t1 || t2
	if err == nil {
		return 0, stdout, stderr, trunc, false
	}
	if ctx.Err() == context.DeadlineExceeded {
		return -1, stdout, stderr, trunc, true
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), stdout, stderr, trunc, false
	}
	return -1, stdout, stderr + "\nlaunch: " + err.Error(), trunc, false
}
