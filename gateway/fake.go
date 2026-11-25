package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// ScriptedRunner is a Runner double: each Start consumes the next script,
// recording its invocation. Scripts carry canned stdout plus the exit;
// an empty script list fails loudly so tests never silently run nothing.
type ScriptedRunner struct {
	mu      sync.Mutex
	Scripts []Script
	Calls   []RunCall
}

// Script is one canned process execution. Hang keeps the process alive
// until killed, modelling a stuck agent for cancellation tests.
type Script struct {
	Stdout   string
	ExitCode int
	WaitErr  error
	Hang     bool
}

// errScriptKilled reports a hung scripted process killed mid-wait.
var errScriptKilled = errors.New("gateway: scripted process killed")

// RunCall records one Start invocation for assertions.
type RunCall struct {
	ID   string
	Dir  string
	Name string
	Args []string
}

// scriptedProc replays one script.
type scriptedProc struct {
	out    io.Reader
	code   int
	err    error
	done   chan struct{}
	hang   bool
	killMu sync.Mutex
	killed bool
	once   sync.Once
}

func (s *ScriptedRunner) Start(_ context.Context, id, dir, name string, args []string, _ []string) (Proc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Scripts) == 0 {
		return nil, fmt.Errorf("gateway: script exhausted for %s", name)
	}
	script := s.Scripts[0]
	s.Scripts = s.Scripts[1:]
	s.Calls = append(s.Calls, RunCall{ID: id, Dir: dir, Name: name, Args: append([]string(nil), args...)})
	p := &scriptedProc{out: strings.NewReader(script.Stdout), code: script.ExitCode,
		err: script.WaitErr, done: make(chan struct{}), hang: script.Hang}
	if !script.Hang {
		close(p.done)
	}
	return p, nil
}

func (p *scriptedProc) Stdout() io.Reader { return p.out }

func (p *scriptedProc) Stderr() io.Reader { return strings.NewReader("") }

func (p *scriptedProc) Signal(os.Signal) error { return nil }

func (p *scriptedProc) Wait() (int, error) {
	<-p.done
	p.killMu.Lock()
	defer p.killMu.Unlock()
	if p.killed {
		return -1, errScriptKilled
	}
	return p.code, p.err
}

func (p *scriptedProc) Pid() int { return -1 }

func (p *scriptedProc) Kill() error {
	p.killMu.Lock()
	p.killed = true
	p.killMu.Unlock()
	if p.hang {
		p.once.Do(func() { close(p.done) })
	}
	return nil
}

// FakeAdapter is a scripted AgentAdapter for command-server tests. Calls
// are recorded; behaviors (refs, failures) are preloaded per method.
type FakeAdapter struct {
	mu        sync.Mutex
	Refs      []string
	Starts    []BuildAssignment
	Continues []ContinueCall
	Cancels   []string
	Statuses  []string
	Collects  []CollectCall
	Fail      map[string]error
	RunEvents [][]AgentEvent
	// Sessions, when set, receives the record Start would create.
	Sessions *SessionManager
}

// ContinueCall records one Continue with its session.
type ContinueCall struct {
	Ref   string
	Input UserInput
}

// CollectCall records one Collect with its session.
type CollectCall struct {
	Ref      string
	Manifest Manifest
}

// NewFakeAdapter returns an empty fake adapter.
func NewFakeAdapter() *FakeAdapter {
	return &FakeAdapter{Fail: map[string]error{}}
}

func (f *FakeAdapter) fail(op string) error {
	if err, ok := f.Fail[op]; ok {
		return err
	}
	return nil
}

func (f *FakeAdapter) Start(_ context.Context, build BuildAssignment) (string, <-chan AgentEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("Start"); err != nil {
		return "", nil, err
	}
	f.Starts = append(f.Starts, build)
	ref := "fake-sess-" + build.BuildID
	if len(f.Refs) > 0 {
		ref = f.Refs[0]
		f.Refs = f.Refs[1:]
	}
	// Like the real adapter, Start owns record creation when sessions are
	// wired; duplicate starts surface the exists sentinel for 409 mapping.
	if f.Sessions != nil {
		turn := build.Turn
		if turn <= 0 {
			turn = 1
		}
		if err := f.Sessions.Create(build.BuildID, SessionRecord{
			SessionRef: ref, BuildID: build.BuildID, Workspace: build.Workspace,
			State: SessionRunning, Turn: turn,
		}); err != nil {
			return "", nil, err
		}
	}
	return ref, f.channelLocked(), nil
}

func (f *FakeAdapter) Continue(_ context.Context, sessionRef string, input UserInput) (<-chan AgentEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("Continue"); err != nil {
		return nil, err
	}
	f.Continues = append(f.Continues, ContinueCall{Ref: sessionRef, Input: input})
	// Mirror the real adapter's record transition (running + new turn)
	// so fencing tests exercise the server, not the double. Unknown
	// refs stay lenient: the fake never invents failures.
	if f.Sessions != nil {
		if list, lerr := f.Sessions.List(); lerr == nil {
			for _, rec := range list {
				if rec.SessionRef == sessionRef {
					_ = f.Sessions.Update(rec.BuildID, func(r *SessionRecord) {
						r.State = SessionRunning
						r.Turn = input.Turn
					})
					break
				}
			}
		}
	}
	return f.channelLocked(), nil
}

// channelLocked builds the next scripted run channel. Scripts queue FIFO;
// an exhausted queue yields an immediately-closing channel (no events)
// rather than failing, so tests script only what they assert.
func (f *FakeAdapter) channelLocked() <-chan AgentEvent {
	var script []AgentEvent
	if len(f.RunEvents) > 0 {
		script = f.RunEvents[0]
		f.RunEvents = f.RunEvents[1:]
	}
	ch := make(chan AgentEvent, len(script)+1)
	for _, ev := range script {
		ch <- ev
	}
	close(ch)
	return ch
}

func (f *FakeAdapter) Status(_ context.Context, sessionRef string) (AgentStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("Status"); err != nil {
		return AgentStatus{}, err
	}
	f.Statuses = append(f.Statuses, sessionRef)
	return AgentStatus{Alive: true, SessionRef: sessionRef}, nil
}

func (f *FakeAdapter) Cancel(_ context.Context, sessionRef string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("Cancel"); err != nil {
		return err
	}
	f.Cancels = append(f.Cancels, sessionRef)
	return nil
}

func (f *FakeAdapter) Collect(_ context.Context, sessionRef string, manifest Manifest) ([]Artifact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("Collect"); err != nil {
		return nil, err
	}
	f.Collects = append(f.Collects, CollectCall{Ref: sessionRef, Manifest: manifest})
	var out []Artifact
	for _, e := range manifest.Entries {
		out = append(out, Artifact{Path: e.Path, Kind: e.Kind,
			Digest: "sha256:fake", Size: 3, Data: []byte("fake")})
	}
	return out, nil
}
