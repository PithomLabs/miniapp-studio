package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// maxStderrTail bounds the retained stderr per run. Diagnostic tails aid
// launch-failure reports; the rest is discarded as it streams.
const maxStderrTail = 32 * 1024

// AgentCredential is the OS identity the agent runs as. Every spawn drops
// to it via SysProcAttr.Credential — there is no code path that launches
// the agent as the gateway user. The zero value is invalid: spawning
// without explicit identity refuses. IOGID is the inbox group; it must be
// among the agent's memberships or the spawn refuses (an agent that
// cannot use the inbox is a misconfiguration, not a runtime surprise).
//
// Same-uid spawns are forbidden unless AllowSameUID is set (tests and
// single-user development only; serve() never sets it): a production
// gateway spawning its own uid would silently recreate the B1
// vulnerability. Same-uid spawns keep the caller's groups (setgroups is
// meaningless without a user change and EPERMs unprivileged); every
// different-uid spawn pins explicit non-empty groups so the child can
// never inherit the gateway's memberships.
type AgentCredential struct {
	UID          uint32
	GID          uint32
	Groups       []uint32
	IOGID        uint32
	Set          bool
	AllowSameUID bool
}

// Supervisor spawns and reaps agent child processes. It never restarts
// anything automatically: a dead agent is an event for Studio, not a
// reboot. Safe for concurrent use.
var (
	_ Runner = (*Supervisor)(nil)
	_ Proc   = (*Process)(nil)
)

type Supervisor struct {
	mu    sync.Mutex
	procs map[string]*Process
	// Agent is the mandatory drop identity; Start refuses when unset.
	Agent AgentCredential
}

// Process is one supervised child. Stdout streams the child's stdout for
// event translation; Wait reaps it exactly once.
type Process struct {
	ID     string
	cmd    *exec.Cmd
	stdout io.ReadCloser
	ring   *tailBuffer
	drain  chan struct{}
	done   chan struct{}
	once   sync.Once
	code   int
	err    error
}

// NewSupervisor returns an empty supervisor. Identity must be assigned
// before Start or every spawn refuses.
func NewSupervisor() *Supervisor {
	return &Supervisor{procs: map[string]*Process{}}
}

// tailBuffer keeps the last N bytes written, for bounded stderr tails.
type tailBuffer struct {
	mu  sync.Mutex
	buf []byte
	max int
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.max {
		b.buf = append([]byte(nil), b.buf[len(b.buf)-b.max:]...)
	}
	return len(p), nil
}

func (b *tailBuffer) Snapshot() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf...)
}

// Start launches name with args in dir, capturing both output streams.
// Env augments the gateway's own environment. Duplicate ids are refused.
//
// The child runs through a umask-pinning wrapper (agent-created files
// land group-writable for the inbox group), in its own process group, as
// the configured agent identity. The caller's context gates admission
// only: cancelling it after spawn never kills the agent (M3) — only the
// explicit Cancel path stops a run. A missing binary fails fast via
// PATH lookup before any process starts.
func (s *Supervisor) Start(ctx context.Context, id, dir, name string, args []string, env []string) (Proc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.procs[id]; dup {
		return nil, fmt.Errorf("gateway: process %q already supervised", id)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !s.Agent.Set {
		return nil, fmt.Errorf("gateway: agent identity unconfigured (refusing same-user spawn)")
	}
	if !memberOf(s.Agent.IOGID, s.Agent.GID, s.Agent.Groups) {
		return nil, fmt.Errorf("gateway: agent identity lacks inbox-group membership")
	}
	if s.Agent.UID == uint32(os.Geteuid()) && !s.Agent.AllowSameUID {
		return nil, fmt.Errorf("gateway: same-user agent spawn forbidden (no fallback)")
	}
	if s.Agent.UID != uint32(os.Geteuid()) && len(s.Agent.Groups) == 0 {
		return nil, fmt.Errorf("gateway: different-user spawn needs explicit groups")
	}
	if _, err := exec.LookPath(name); err != nil {
		return nil, fmt.Errorf("gateway: resolve %s: %w", name, err)
	}
	cmd := exec.CommandContext(context.WithoutCancel(ctx), "/bin/sh", "-c",
		"umask 007; exec \"$@\"", "studio-agent-sh", name)
	cmd.Args = append(cmd.Args, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	if err := applyChildIdentity(cmd, s.Agent); err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("gateway: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdout.Close()
		return nil, fmt.Errorf("gateway: stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		_ = stderr.Close()
		return nil, fmt.Errorf("gateway: start %s: %w", name, err)
	}
	p := &Process{
		ID: id, cmd: cmd, stdout: stdout,
		ring: &tailBuffer{max: maxStderrTail}, drain: make(chan struct{}),
		done: make(chan struct{}), code: -1,
	}
	s.procs[id] = p
	go func() {
		_, _ = io.Copy(p.ring, stderr)
		_ = stderr.Close()
		close(p.drain)
	}()
	go p.reap()
	return p, nil
}

func memberOf(gid, primary uint32, groups []uint32) bool {
	if gid == primary {
		return true
	}
	for _, g := range groups {
		if g == gid {
			return true
		}
	}
	return false
}

// Lookup returns a supervised process by id.
func (s *Supervisor) Lookup(id string) (Proc, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.procs[id]
	return p, ok
}

// Stdout streams the child's standard output. The adapter scans NDJSON
// events from it; EOF arrives when the child exits.
func (p *Process) Stdout() io.Reader {
	return p.stdout
}

// Stderr replays the bounded retained tail of the child's standard
// error. The supervisor drains the live pipe continuously (an undrained
// pipe would wedge the agent past 64KB); diagnostics read the snapshot.
func (p *Process) Stderr() io.Reader {
	return bytes.NewReader(p.ring.Snapshot())
}

// Pid reports the child PID, or -1 when unknown.
func (p *Process) Pid() int {
	if p.cmd.Process == nil {
		return -1
	}
	return p.cmd.Process.Pid
}

// Signal delivers one signal to the child's whole process group (for
// graceful cancellation reaching grandchildren too).
func (p *Process) Signal(sig os.Signal) error {
	if p.cmd.Process == nil {
		return fmt.Errorf("gateway: process %s has no pid", p.ID)
	}
	return killProcessGroup(p.cmd.Process.Pid, sig)
}

// Kill force-terminates the child's whole process group.
func (p *Process) Kill() error {
	if p.cmd.Process == nil {
		return fmt.Errorf("gateway: process %s has no pid", p.ID)
	}
	return killProcessGroup(p.cmd.Process.Pid, os.Kill)
}

// Wait blocks until the child exits and returns its exit code. A
// signal-killed child reports code -1 with the wait error. Repeated calls
// return the same result. It also waits out the stderr drain so Stderr
// snapshots are complete.
func (p *Process) Wait() (int, error) {
	<-p.done
	return p.code, p.err
}

func (p *Process) reap() {
	defer close(p.done)
	defer func() { _ = p.stdout.Close() }()
	err := p.cmd.Wait()
	<-p.drain
	if err == nil {
		p.code = 0
		return
	}
	p.err = err
	if exit, ok := err.(*exec.ExitError); ok {
		p.code = exit.ExitCode()
	}
}
