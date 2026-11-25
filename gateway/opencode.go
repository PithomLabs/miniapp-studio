package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Inbox file names (single-slot handshake under <workdir>/.studio/inbox).
// The agent writes question.json as its last act before yielding; the
// gateway writes answer.json before resuming. Overwrites are natural
// supersede semantics — no turn numbering to desynchronize.
const (
	inboxDir         = ".studio/inbox"
	questionFileName = "question.json"
	answerFileName   = "answer.json"
)

// OpenCodeAdapter drives `opencode run` through a Runner, translating its
// NDJSON stream to AgentEvents. Verified interface facts (run.ts @
// v1.19.6, see docs/opencode-run-v1.19.6.md): `--format json` emits
// {"type","timestamp","sessionID",...} lines; the run ends when the
// session idles; exit 0 means idle without session errors, exit 1 means
// errors; the `question` tool is denied in run mode, so yields arrive
// exclusively via the file-drop handshake.
type OpenCodeAdapter struct {
	Runner    Runner
	Sessions  *SessionManager
	BinPath   string
	KillGrace time.Duration
	// AgentUID is the OS uid that owns agent-written drops; the consume
	// path accepts drops from this uid only. Zero defaults to the
	// gateway's own euid (tests, probes); production sets the agent
	// user explicitly, where the default would fail closed.
	AgentUID int
	// IOGID is the inbox group for created handshake directories. Zero
	// defaults to the gateway's egid; production sets studio-io.
	IOGID int

	mu   sync.Mutex
	runs map[string]*ocRun
	seq  atomic.Int64
}

// agentUID resolves the expected drop-owner identity.
func (a *OpenCodeAdapter) agentUID() int {
	if a.AgentUID != 0 {
		return a.AgentUID
	}
	return os.Geteuid()
}

// ioGID resolves the handshake group identity.
func (a *OpenCodeAdapter) ioGID() int {
	if a.IOGID != 0 {
		return a.IOGID
	}
	return os.Getegid()
}

// ensureInbox creates the .studio/inbox chain with the B1 model:
// .studio traversable by the inbox group, inbox group-writable + sticky
// (2730) so the agent can drop questions but can never remove or
// replace gateway-owned answers.
func (a *OpenCodeAdapter) ensureInbox(workDir string) error {
	if err := SecureDir(filepath.Join(workDir, ".studio"), a.ioGID(), os.ModeSetgid|0o750); err != nil {
		return fmt.Errorf("gateway: studio dir: %w", err)
	}
	if err := SecureDir(filepath.Join(workDir, inboxDir), a.ioGID(), os.ModeSetgid|os.ModeSticky|0o730); err != nil {
		return fmt.Errorf("gateway: inbox dir: %w", err)
	}
	return nil
}

// ocRun tracks one agent run for event streaming and cancellation.
type ocRun struct {
	proc         Proc
	events       chan AgentEvent
	done         chan struct{}
	ref          string
	skipped      atomic.Int64
	corrupt      atomic.Int64
	truncated    atomic.Bool
	terminalText atomic.Value // string: last terminal-event summary, for Status
}

// OpenCodeBinary is the executable name, resolved on PATH in the sprite.
const OpenCodeBinary = "opencode"

// ErrSessionBusy reports a resume refused because a run is already live
// on the session. No child started: the command is safely retryable (the
// server drops its claim and answers 409).
var ErrSessionBusy = fmt.Errorf("gateway: session has a live run")

// ErrAgentStray reports a launch that started a child the adapter then
// failed to record. The stray is killed before this returns, but the
// command's outcome is ambiguous — the server marks it indeterminate,
// never silently retried.
var ErrAgentStray = fmt.Errorf("gateway: agent started but unrecorded")

func (a *OpenCodeAdapter) bin() string {
	if a.BinPath != "" {
		return a.BinPath
	}
	return OpenCodeBinary
}

func (a *OpenCodeAdapter) grace() time.Duration {
	if a.KillGrace > 0 {
		return a.KillGrace
	}
	return 10 * time.Second
}

// assignmentPrompt renders the build instructions with the yield
// convention. The agent must know exactly how to hand control back,
// because no native mechanism exists under run. It also restates the
// Miniapp application contract before the turn: instructions alone do
// not enforce it (admission gates arrive with their authorized phase),
// but the default must never be "whatever stack the agent prefers".
func assignmentPrompt(build BuildAssignment) string {
	return build.Prompt + "\n\n[Studio turn " + itoa(build.Turn) + "] " +
		"Work in this directory. Build a Miniapp exclusively: Go " +
		"implementation, Miniapp framework, MiniappSpec specification, " +
		"Miniapp runtime execution — no other stack. " +
		"If you need input from the business user, " +
		"write a JSON object {\"kind\": \"clarification\"|\"proposal\"|\"authorization_request\", " +
		"\"body\": \"...\", \"options\": [...]} to .studio/inbox/question.json " +
		"as your last act and then stop. Do not use any interactive question " +
		"feature; it is unavailable in this environment."
}

// resumePrompt renders the continuation instructions: read the answer file
// first, then continue the same task.
func resumePrompt(input UserInput) string {
	return "The business user has answered (see .studio/inbox/answer.json). " +
		"Read that file first, then continue exactly where you left off."
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

// Start launches one supervised `opencode run`, returning the agent-native
// session id plus the run's event channel. The session record is created
// before launch (ref filled in after capture) so the pump's end-of-run
// update can never race record creation.
func (a *OpenCodeAdapter) Start(ctx context.Context, build BuildAssignment) (string, <-chan AgentEvent, error) {
	if build.WorkDir == "" || build.Prompt == "" {
		return "", nil, fmt.Errorf("gateway: workdir and prompt required")
	}
	if err := a.ensureInbox(build.WorkDir); err != nil {
		return "", nil, err
	}
	turn := build.Turn
	if turn <= 0 {
		turn = 1
	}
	if err := a.Sessions.Create(build.BuildID, SessionRecord{
		BuildID: build.BuildID, Workspace: build.Workspace,
		WorkDir: build.WorkDir, State: SessionRunning, Turn: turn,
	}); err != nil {
		return "", nil, err
	}
	args := []string{"run", "--format", "json", "--dir", build.WorkDir}
	if build.Model != "" {
		args = append(args, "--model", build.Model)
	}
	if build.AgentName != "" {
		args = append(args, "--agent", build.AgentName)
	}
	args = append(args, assignmentPrompt(build))
	ref, ch, err := a.launch(ctx, build.BuildID, build.WorkDir, args, "")
	if err != nil {
		_ = a.Sessions.Remove(build.BuildID)
		return "", nil, err
	}
	pid := a.runPID(ref)
	if err := a.Sessions.Update(build.BuildID, func(r *SessionRecord) {
		r.SessionRef = ref
		r.PID = pid
	}); err != nil {
		// The child runs but the record is broken: kill the stray
		// (background context — the caller's may be gone) and report
		// ambiguity, never a clean failure.
		_ = a.Cancel(context.Background(), ref)
		_ = a.Sessions.Remove(build.BuildID)
		return "", nil, fmt.Errorf("%w: session record update failed", ErrAgentStray)
	}
	return ref, ch, nil
}

// runPID reports a live run's child pid, or -1 when the run is gone
// (already reaped: recovery treats pidless runs conservatively).
func (a *OpenCodeAdapter) runPID(ref string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	if run, ok := a.runs[ref]; ok && run.proc != nil {
		return run.proc.Pid()
	}
	return -1
}

// Continue writes the answer file and resumes the native session.
func (a *OpenCodeAdapter) Continue(ctx context.Context, sessionRef string, input UserInput) (<-chan AgentEvent, error) {
	a.mu.Lock()
	_, live := a.runs[sessionRef]
	a.mu.Unlock()
	if live {
		return nil, fmt.Errorf("%w: session %s", ErrSessionBusy, sessionRef)
	}
	rec, buildID, err := a.resolve(sessionRef)
	if err != nil {
		return nil, err
	}
	answer := map[string]any{
		"question_id": input.QuestionID, "kind": input.Kind, "answer": input.Answer,
	}
	raw, err := json.Marshal(answer)
	if err != nil {
		return nil, fmt.Errorf("gateway: encode answer: %w", err)
	}
	if err := a.ensureInbox(rec.WorkDir); err != nil {
		return nil, err
	}
	// The question slot belongs to the new turn: clear any leftover so
	// a stale drop can never surface as a late yield.
	a.clearQuestionDrop(rec.WorkDir)
	if err := a.writeAnswerDrop(rec.WorkDir, raw); err != nil {
		return nil, err
	}
	args := []string{"run", "--format", "json", "--dir", rec.WorkDir, "--session", sessionRef, resumePrompt(input)}
	newRef, ch, err := a.launch(ctx, buildID, rec.WorkDir, args, sessionRef)
	if err != nil {
		// Restore the pre-resume state (yielded normally, failed on
		// an F6 resume): a failed launch must not rewrite history.
		_ = a.Sessions.Update(buildID, func(r *SessionRecord) { r.State = rec.State })
		return nil, err
	}
	pid := a.runPID(newRef)
	if err := a.Sessions.Update(buildID, func(r *SessionRecord) {
		r.State = SessionRunning
		r.PID = pid
		r.Turn = input.Turn
	}); err != nil {
		_ = a.Cancel(context.Background(), newRef)
		_ = a.Sessions.Update(buildID, func(r *SessionRecord) { r.State = rec.State })
		return nil, fmt.Errorf("%w: session record update failed", ErrAgentStray)
	}
	return ch, nil
}

// Status reports liveness from the session record plus the live run table.
func (a *OpenCodeAdapter) Status(_ context.Context, sessionRef string) (AgentStatus, error) {
	rec, _, err := a.resolve(sessionRef)
	if err != nil {
		return AgentStatus{}, err
	}
	a.mu.Lock()
	run, live := a.runs[sessionRef]
	a.mu.Unlock()
	detail := rec.State
	if live {
		detail = fmt.Sprintf("%s skipped=%d corrupt=%d",
			rec.State, run.skipped.Load(), run.corrupt.Load())
		if run.truncated.Load() {
			detail += " truncated=1"
		}
	}
	return AgentStatus{Alive: live && !terminalSession(rec.State), SessionRef: sessionRef, Detail: detail}, nil
}

// Cancel stops one run gracefully, then forcefully. The session record
// transition is the caller's (server.cancelBuild marks cancelled).
func (a *OpenCodeAdapter) Cancel(ctx context.Context, sessionRef string) error {
	a.mu.Lock()
	run, ok := a.runs[sessionRef]
	a.mu.Unlock()
	if !ok {
		return nil // nothing live: idempotent success
	}
	_ = run.proc.Signal(terminateSignal())
	select {
	case <-run.done:
		return nil
	case <-time.After(a.grace()):
		return run.proc.Kill()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Shutdown cancels every live run for gateway shutdown: graceful SIGTERM
// first, best-effort kill after the grace, all in parallel. It never
// starts anything and never blocks past ctx.
func (a *OpenCodeAdapter) Shutdown(ctx context.Context) {
	a.mu.Lock()
	refs := make([]string, 0, len(a.runs))
	for ref := range a.runs {
		refs = append(refs, ref)
	}
	a.mu.Unlock()
	var wg sync.WaitGroup
	for _, ref := range refs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = a.Cancel(ctx, ref)
		}()
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// Collect gathers a manifest from the session working dir.
func (a *OpenCodeAdapter) Collect(_ context.Context, sessionRef string, manifest Manifest) ([]Artifact, error) {
	rec, _, err := a.resolve(sessionRef)
	if err != nil {
		return nil, err
	}
	return Collect(rec.WorkDir, manifest)
}

// resolve maps an agent session ref back to its record + build by scanning
// session records (few per sprite; no in-memory map to lose on restart).
func (a *OpenCodeAdapter) resolve(sessionRef string) (SessionRecord, string, error) {
	list, err := a.Sessions.List()
	if err != nil {
		return SessionRecord{}, "", err
	}
	for _, rec := range list {
		if rec.SessionRef == sessionRef {
			return rec, rec.BuildID, nil
		}
	}
	return SessionRecord{}, "", fmt.Errorf("gateway: unknown session")
}

// launch starts one supervised run and captures its native session id from
// the opening stream lines. Leading non-event output (at most a few lines)
// is tolerated and still fed to the translator, which classifies it —
// only a stream with no session identity at all fails the launch.
// wantRef, when set, pins the expected session lineage (resume): a
// stream reporting any other session is tainted — the child is killed
// before registration (no pump, no record write) and the launch reports
// ambiguity.
func (a *OpenCodeAdapter) launch(ctx context.Context, buildID, workDir string, args []string, wantRef string) (string, <-chan AgentEvent, error) {
	n := a.seq.Add(1)
	proc, err := a.Runner.Start(ctx, fmt.Sprintf("%s/run-%d", buildID, n), workDir, a.bin(), args, nil)
	if err != nil {
		return "", nil, fmt.Errorf("gateway: start agent: %w", err)
	}
	reader := bufio.NewReader(proc.Stdout())
	var head [][]byte
	sessionRef := ""
	for len(head) < 8 && sessionRef == "" {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			head = append(head, line)
			sessionRef = sessionOf(line)
		}
		if err != nil {
			break
		}
	}
	if sessionRef == "" {
		_, werr := proc.Wait()
		detail := firstLines(proc.Stderr(), 5)
		_ = proc.Kill()
		if werr != nil {
			return "", nil, fmt.Errorf("gateway: agent exited before streaming: %v: %s", werr, detail)
		}
		return "", nil, fmt.Errorf("gateway: agent stream lacks session identity: %s", detail)
	}
	if wantRef != "" && sessionRef != wantRef {
		// Tainted resume: kill first (the agent is streaming, so
		// waiting first would hang), then reap. Nothing is
		// registered and the record is untouched.
		_ = proc.Kill()
		_, _ = proc.Wait()
		return "", nil, fmt.Errorf("%w: resumed as %s, want %s", ErrAgentStray, sessionRef, wantRef)
	}
	run := &ocRun{proc: proc, events: make(chan AgentEvent, 256), done: make(chan struct{}), ref: sessionRef}
	a.mu.Lock()
	if a.runs == nil {
		a.runs = map[string]*ocRun{}
	}
	a.runs[sessionRef] = run
	a.mu.Unlock()
	go a.pump(buildID, workDir, sessionRef, run, reader, head)
	return sessionRef, run.events, nil
}

// sessionOf extracts a session id from one line, or "" when absent.
func sessionOf(line []byte) string {
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err != nil {
		return ""
	}
	id, _ := raw["sessionID"].(string)
	return id
}

// pump translates the NDJSON stream until EOF, then resolves the run end:
// a dropped question file becomes a question event, exit 0 becomes done,
// anything else becomes failed. The session record follows the outcome,
// and skip/corrupt tallies land on the run for Status reporting.
func (a *OpenCodeAdapter) pump(buildID, workDir, sessionRef string, run *ocRun, reader *bufio.Reader, head [][]byte) {
	defer close(run.events)
	defer close(run.done)
	defer func() {
		a.mu.Lock()
		delete(a.runs, sessionRef)
		a.mu.Unlock()
	}()
	emit := func(line []byte) {
		ev, outcome := TranslateLine(line, sessionRef)
		switch outcome {
		case LineEvent:
			run.events <- ev
		case LineSkip:
			run.skipped.Add(1)
		case LineCorrupt:
			run.corrupt.Add(1)
		}
	}
	for _, line := range head {
		emit(line)
	}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 4<<20)
	for scanner.Scan() {
		emit(append([]byte(nil), scanner.Bytes()...))
	}
	// A dead scanner (notably an overlong line past the buffer cap)
	// ends observation: surface it in the corrupt tally and the
	// status detail instead of silently truncating the stream.
	if err := scanner.Err(); err != nil {
		run.corrupt.Add(1)
		run.truncated.Store(true)
	}
	code, _ := run.proc.Wait()
	note := fmt.Sprintf("skipped=%d corrupt=%d exit=%d", run.skipped.Load(), run.corrupt.Load(), code)
	if run.truncated.Load() {
		note += " truncated=1"
	}
	run.terminalText.Store(note)
	if q, bad := a.consumeQuestionDrop(workDir); q != nil {
		run.events <- AgentEvent{Kind: EventQuestion, Question: q, RawSession: sessionRef}
		_ = a.Sessions.Update(buildID, func(r *SessionRecord) { r.State = SessionYielded })
		return
	} else if bad != "" {
		// Present-but-invalid drops fail the run even on exit 0: a
		// tampered or unreadable slot must never report success.
		text := fmt.Sprintf("invalid question drop: %s", bad)
		run.events <- AgentEvent{Kind: EventFailed, Text: text, RawSession: sessionRef}
		_ = a.Sessions.Update(buildID, func(r *SessionRecord) { r.State = SessionFailed })
		return
	}
	if code == 0 {
		run.events <- AgentEvent{Kind: EventDone, RawSession: sessionRef}
		_ = a.Sessions.Update(buildID, func(r *SessionRecord) { r.State = SessionDone })
		return
	}
	run.events <- AgentEvent{Kind: EventFailed, Text: fmt.Sprintf("agent exited %d", code), RawSession: sessionRef}
	_ = a.Sessions.Update(buildID, func(r *SessionRecord) { r.State = SessionFailed })
}

func terminateSignal() syscall.Signal {
	return syscall.SIGTERM
}

func firstLines(r io.Reader, n int) string {
	if r == nil {
		return ""
	}
	var out []string
	scanner := bufio.NewScanner(io.LimitReader(r, 4096))
	for len(out) < n && scanner.Scan() {
		out = append(out, scanner.Text())
	}
	return strings.Join(out, " | ")
}
