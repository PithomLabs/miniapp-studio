// Package hostexec is the host execution broker: the narrow, audited
// bridge through which the sandboxed agent drives exactly the approved
// Docker operations for the PostgreSQL test-runner deployment. The agent
// NEVER receives Docker access; it writes strictly validated job files
// into a workspace queue, and the owner-side broker daemon executes
// allowlisted operations with fixed arguments it constructs itself.
//
// Threat model: the queue is UNTRUSTED input. Jobs are size-capped,
// charset-pinned, schema-strict (unknown fields rejected), and every
// path/argument the broker touches is a constant or install-time
// config value — never job-supplied. No shell is ever invoked.
package hostexec

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Operations. The deployment sequence uses exactly these; nothing else
// is expressible. Deploy-only ops disarm automatically after the first
// green suite run (see ArmedState); steady-state ops stay armed for
// repeatable hands-off test runs.
const (
	OpVersion         = "version"
	OpNetCreate       = "net.create"
	OpNetConnect      = "net.connect"
	OpImageBuild      = "image.build"
	OpContainerRun    = "container.run"
	OpNetDisconnect   = "net.disconnect"
	OpNetRemove       = "net.remove"
	OpContainerRm     = "container.rm"
	OpInspectReadonly = "inspect.readonly"
)

// Run modes for container.run: the ONLY varying job parameter in the
// entire schema (besides the job id).
const (
	ModeNetcheck = "netcheck"
	ModeRun      = "run"
)

// Read-only inspectors.
const (
	InspectNetLs      = "net.ls"
	InspectNetInspect = "net.inspect"
	InspectPsTest     = "ps.test"
	InspectPgNets     = "pg.networks"
	InspectPgPorts    = "pg.ports"
)

// Fixed deployment constants. Changing any of these is a code change
// under owner review — never a job parameter.
const (
	NetName       = "studio-test-internal"
	NetLabel      = "studio.testnet=1"
	PgContainer   = "miniapp-p0-pg"
	PgAlias       = "pg-test"
	ImageTag      = "studio-test-runner:1"
	ContainerBase = "studio-test-"
	ResultsDir    = "test-results"
	RelayPort     = "5434"
	PgPort        = "5432"
)

// Absolute limits.
const (
	MaxJobBytes    = 64 * 1024
	MaxResultBytes = 32 * 1024 // per stream tail kept in the result
)

var (
	rxJobID    = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)
	rxFileName = regexp.MustCompile(`^[a-z0-9-]{1,64}\.json$`)
	// rxSweepName matches only broker-generated container names for the
	// orphan sweep. Nothing job-supplied ever reaches docker rm.
	rxSweepName = regexp.MustCompile(`^studio-test-(netcheck|run-[0-9]+)$`)
	// Banned substrings are a backstop, not the primary control: jobs
	// cannot supply arguments at all, so these must never appear. If
	// one does, the broker refuses rather than executing.
	bannedSubstrings = []string{
		"--privileged", "--network=host", ".sock", "/var/run",
		"postgres://", "postgresql://", ";", "|", "$", "`",
	}
)

// Job is one validated queue entry.
type Job struct {
	ID     string         `json:"id"`
	Op     string         `json:"op"`
	Params map[string]any `json:"params,omitempty"`
}

// validOps enumerates the schema; deployOnly marks the capabilities
// that disarm after the first green suite run.
var validOps = map[string]bool{
	OpVersion:         false,
	OpNetCreate:       true,
	OpNetConnect:      true,
	OpImageBuild:      true,
	OpContainerRun:    false, // steady-state: repeatable test runs
	OpNetDisconnect:   true,
	OpNetRemove:       true,
	OpContainerRm:     false, // steady-state: per-run orphan hygiene
	OpInspectReadonly: false,
}

// ParseJob decodes and strictly validates one job's bytes. Unknown
// fields, oversize, bad charset, and unknown ops/modes are all refused.
func ParseJob(raw []byte) (Job, error) {
	var j Job
	if len(raw) > MaxJobBytes {
		return Job{}, fmt.Errorf("job oversized: %d bytes", len(raw))
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&j); err != nil {
		return Job{}, fmt.Errorf("malformed job: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return Job{}, fmt.Errorf("trailing data after job")
	}
	if err := rejectDupKeys(raw); err != nil {
		return Job{}, err
	}
	if !rxJobID.MatchString(j.ID) {
		return Job{}, fmt.Errorf("bad job id %q", j.ID)
	}
	deployOnly, ok := validOps[j.Op]
	if !ok {
		return Job{}, fmt.Errorf("unknown op %q", j.Op)
	}
	_ = deployOnly
	if err := checkParams(j); err != nil {
		return Job{}, err
	}
	if err := checkBanned(raw); err != nil {
		return Job{}, err
	}
	return j, nil
}

func checkParams(j Job) error {
	need := func(keys ...string) error {
		if len(j.Params) != len(keys) {
			return fmt.Errorf("op %s takes params %v", j.Op, keys)
		}
		for _, k := range keys {
			if _, ok := j.Params[k]; !ok {
				return fmt.Errorf("op %s takes params %v", j.Op, keys)
			}
		}
		return nil
	}
	switch j.Op {
	case OpContainerRun:
		if err := need("mode"); err != nil {
			return err
		}
		m, _ := j.Params["mode"].(string)
		if m != ModeNetcheck && m != ModeRun {
			return fmt.Errorf("bad run mode %q", j.Params["mode"])
		}
		return nil
	case OpInspectReadonly:
		if err := need("what"); err != nil {
			return err
		}
		w, _ := j.Params["what"].(string)
		switch w {
		case InspectNetLs, InspectNetInspect, InspectPsTest, InspectPgNets, InspectPgPorts:
			return nil
		}
		return fmt.Errorf("bad inspector %q", j.Params["what"])
	default:
		if len(j.Params) != 0 {
			return fmt.Errorf("op %s takes no params", j.Op)
		}
		return nil
	}
}

// rejectDupKeys refuses any object with a repeated key, at any depth.
// encoding/json silently last-wins duplicates, which would let the
// executed value differ from a naive first-key audit reading.
func rejectDupKeys(raw []byte) error {
	type frame struct {
		obj     bool
		keys    map[string]bool
		wantKey bool
	}
	var stack []frame
	pushValue := func() {
		if n := len(stack); n > 0 && stack[n-1].obj {
			stack[n-1].wantKey = true
		}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("malformed job: %w", err)
		}
		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{', '[':
				if len(stack) >= 32 {
					return fmt.Errorf("job nesting too deep")
				}
				stack = append(stack, frame{obj: d == '{', keys: map[string]bool{}, wantKey: true})
			case '}', ']':
				if len(stack) == 0 {
					return fmt.Errorf("malformed job: unbalanced close")
				}
				stack = stack[:len(stack)-1]
				pushValue()
			}
			continue
		}
		if n := len(stack); n > 0 && stack[n-1].obj && stack[n-1].wantKey {
			s, ok := tok.(string)
			if !ok {
				return fmt.Errorf("malformed job: non-string key")
			}
			if stack[n-1].keys[s] {
				return fmt.Errorf("duplicate key %q", s)
			}
			stack[n-1].keys[s] = true
			stack[n-1].wantKey = false
			continue
		}
		pushValue()
	}
	return nil
}

func checkBanned(raw []byte) error {
	lower := strings.ToLower(string(raw))
	for _, b := range bannedSubstrings {
		if strings.Contains(lower, strings.ToLower(b)) {
			return fmt.Errorf("job contains banned token %q", b)
		}
	}
	return nil
}

// IsDeployOp reports whether op disarms after workflow completion.
func IsDeployOp(op string) bool { return validOps[op] }

// MatchJobFile reports whether a directory entry name is a candidate
// job file. The broker reads ONLY matching regular files, so path
// traversal via crafted names is impossible by construction (names
// never leave the scanned directory).
func MatchJobFile(name string) bool { return rxFileName.MatchString(name) }

// RunnerContext is the image build context, relative to the repo root.
const RunnerContext = "testrunner"

// WalkContext enumerates every build input: the context must be flat
// (no subdirectories) and contain regular files only. Symlinks,
// directories, and anything else fail the walk — there is no legacy
// shape to accommodate, so strictness costs nothing.
func WalkContext(root string) ([]string, error) {
	dir := filepath.Join(root, RunnerContext)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("walk context: %w", err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			return nil, fmt.Errorf("context subdir %q refused (flat only)", e.Name())
		}
		fi, err := os.Lstat(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("context stat %q: %w", e.Name(), err)
		}
		if !fi.Mode().IsRegular() {
			return nil, fmt.Errorf("context entry %q is not a regular file", e.Name())
		}
		out = append(out, RunnerContext+"/"+e.Name())
	}
	sort.Strings(out)
	return out, nil
}

// HashFiles returns sha256 hex digests of rel paths under root.
func HashFiles(root string, rels []string) (map[string]string, error) {
	out := map[string]string{}
	for _, rel := range rels {
		clean := filepath.Clean(rel)
		if clean != rel || strings.HasPrefix(clean, "..") {
			return nil, fmt.Errorf("bad hash path %q", rel)
		}
		f, err := os.Open(filepath.Join(root, clean))
		if err != nil {
			return nil, fmt.Errorf("hash input %s: %w", rel, err)
		}
		h := sha256.New()
		_, cerr := io.Copy(h, f)
		ferr := f.Close()
		if cerr != nil {
			return nil, fmt.Errorf("hash input %s: %w", rel, cerr)
		}
		if ferr != nil {
			return nil, fmt.Errorf("hash input %s: %w", rel, ferr)
		}
		out[rel] = hex.EncodeToString(h.Sum(nil))
	}
	return out, nil
}

// HashPinInputs returns sha256 hex digests of the full build context.
func HashPinInputs(root string) (map[string]string, error) {
	rels, err := WalkContext(root)
	if err != nil {
		return nil, err
	}
	return HashFiles(root, rels)
}
