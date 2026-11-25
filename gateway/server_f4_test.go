package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// overlapAdapter is a Continue-focused test double that observes
// execution overlap: every Continue records entry, optionally blocks on
// a test gate, sleeps, and tracks max in-flight executions.
type overlapAdapter struct {
	mu        sync.Mutex
	continues int
	inFlight  atomic.Int32
	maxSeen   atomic.Int32
	delay     time.Duration
	entered   chan struct{}
	enterOnce sync.Once
	block     chan struct{}
	events    [][]AgentEvent
}

func (a *overlapAdapter) noteEnter() {
	a.enterOnce.Do(func() {
		if a.entered != nil {
			close(a.entered)
		}
	})
	cur := a.inFlight.Add(1)
	for {
		max := a.maxSeen.Load()
		if cur <= max || a.maxSeen.CompareAndSwap(max, cur) {
			break
		}
	}
}

func (a *overlapAdapter) Start(context.Context, BuildAssignment) (string, <-chan AgentEvent, error) {
	ch := make(chan AgentEvent)
	close(ch)
	return "ov-sess", ch, nil
}

func (a *overlapAdapter) Continue(_ context.Context, _ string, _ UserInput) (<-chan AgentEvent, error) {
	a.noteEnter()
	if a.block != nil {
		<-a.block
	}
	if a.delay > 0 {
		time.Sleep(a.delay)
	}
	a.inFlight.Add(-1)
	a.mu.Lock()
	a.continues++
	var script []AgentEvent
	if len(a.events) > 0 {
		script = a.events[0]
		a.events = a.events[1:]
	}
	a.mu.Unlock()
	ch := make(chan AgentEvent, len(script)+1)
	for _, ev := range script {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

func (a *overlapAdapter) Status(_ context.Context, ref string) (AgentStatus, error) {
	return AgentStatus{Alive: true, SessionRef: ref}, nil
}

func (a *overlapAdapter) Cancel(context.Context, string) error { return nil }

func (a *overlapAdapter) Collect(_ context.Context, _ string, _ Manifest) ([]Artifact, error) {
	return nil, nil
}

// f4Server builds a server on an explicit state dir so tests can reopen
// the same state (crash simulation).
func f4Server(t *testing.T, dir string, adapter AgentAdapter) *Server {
	t.Helper()
	sessions, err := NewSessionManager(filepath.Join(dir, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	outbox, err := OpenOutbox(filepath.Join(dir, "outbox.json"))
	if err != nil {
		t.Fatal(err)
	}
	dedup, err := OpenDedupStore(filepath.Join(dir, "dedup.json"))
	if err != nil {
		t.Fatal(err)
	}
	return &Server{
		Keys: StaticProvider{}, Adapter: adapter, Sessions: sessions,
		Outbox: outbox, Dedup: dedup, WorkRoot: filepath.Join(dir, "work"),
	}
}

func f4Record(t *testing.T, srv *Server, build string) {
	t.Helper()
	if err := srv.Sessions.Create(build, SessionRecord{
		SessionRef: "sess-" + build, BuildID: build,
		Workspace: "w-f4", WorkDir: t.TempDir(), State: SessionRunning,
	}); err != nil {
		t.Fatal(err)
	}
}

// TestServerConcurrentIdenticalContinueSingle (F4): N concurrent
// deliveries of one Continue execute exactly once; every delivery gets
// the shared outcome.
func TestServerConcurrentIdenticalContinueSingle(t *testing.T) {
	dir := t.TempDir()
	adapter := &overlapAdapter{delay: 20 * time.Millisecond}
	srv := f4Server(t, dir, adapter)
	f4Record(t, srv, "b-f4a")
	raw := signedCommand(t, "b-f4a", "same-id", CmdContinue,
		map[string]any{"question_id": "q", "kind": "clarification", "answer": "a", "turn": 1})
	const n = 8
	var wg sync.WaitGroup
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			code, _ := postCommand(t, srv, raw)
			codes[i] = code
		}(i)
	}
	wg.Wait()
	for i, code := range codes {
		if code != http.StatusOK {
			t.Errorf("delivery %d = %d, want 200", i, code)
		}
	}
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.continues != 1 {
		t.Errorf("adapter executions = %d, want exactly 1", adapter.continues)
	}
}

// TestServerDistinctContinuesSerialize (F4): N distinct concurrent
// Continues all execute, never overlapping.
func TestServerDistinctContinuesSerialize(t *testing.T) {
	dir := t.TempDir()
	adapter := &overlapAdapter{delay: 30 * time.Millisecond}
	srv := f4Server(t, dir, adapter)
	f4Record(t, srv, "b-f4b")
	const n = 4
	var wg sync.WaitGroup
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			raw := signedCommand(t, "b-f4b", "distinct-"+string(rune('a'+i)), CmdContinue,
				map[string]any{"question_id": "q", "kind": "clarification", "answer": "a", "turn": 1})
			code, _ := postCommand(t, srv, raw)
			codes[i] = code
		}(i)
	}
	wg.Wait()
	for i, code := range codes {
		if code != http.StatusOK {
			t.Errorf("delivery %d = %d, want 200", i, code)
		}
	}
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.continues != n {
		t.Errorf("adapter executions = %d, want %d", adapter.continues, n)
	}
	if max := adapter.maxSeen.Load(); max != 1 {
		t.Errorf("max in-flight = %d, want serialized 1", max)
	}
}

// TestServerDigestMismatchConflicts (F4): the same message id with a
// different payload must conflict — never share the other payload's
// result and never execute twice.
func TestServerDigestMismatchConflicts(t *testing.T) {
	dir := t.TempDir()
	adapter := &overlapAdapter{entered: make(chan struct{}), block: make(chan struct{})}
	srv := f4Server(t, dir, adapter)
	f4Record(t, srv, "b-f4c")
	first := signedCommand(t, "b-f4c", "dup-id", CmdContinue,
		map[string]any{"question_id": "q", "kind": "clarification", "answer": "one", "turn": 1})
	second := signedCommand(t, "b-f4c", "dup-id", CmdContinue,
		map[string]any{"question_id": "q", "kind": "clarification", "answer": "two", "turn": 1})
	firstCode := make(chan int, 1)
	go func() {
		code, _ := postCommand(t, srv, first)
		firstCode <- code
	}()
	select {
	case <-adapter.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first dispatch never started")
	}
	// The mismatched redelivery must not join the in-flight execution;
	// run it aside so a blocking (pre-fix) server cannot wedge the test.
	secondCode := make(chan int, 1)
	go func() {
		code, _ := postCommand(t, srv, second)
		secondCode <- code
	}()
	// Give a correct server time to refuse without the gate; a pre-fix
	// server stays blocked until the gate below releases it.
	secondDone := false
	select {
	case code := <-secondCode:
		secondDone = true
		if code != http.StatusConflict {
			t.Errorf("mismatched redelivery = %d, want 409", code)
		}
	case <-time.After(500 * time.Millisecond):
		t.Errorf("mismatched redelivery joined the in-flight execution, want immediate 409")
	}
	close(adapter.block)
	if c := <-firstCode; c != http.StatusOK {
		t.Errorf("first dispatch = %d, want 200", c)
	}
	if !secondDone {
		<-secondCode // drain the pre-fix twin (already asserted above)
	}
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.continues != 1 {
		t.Errorf("adapter executions = %d, want 1", adapter.continues)
	}
}

// roundTripDigest digests a payload exactly as the server sees it
// (through envelope JSON).
func roundTripDigest(t *testing.T, payload map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	return payloadDigest(back)
}

// TestServerEmitThenCrashNeverRelaunches (F4/A1): the exact adversarial
// window — an event persisted to the outbox, then a crash before
// Complete. After reboot the command is indeterminate: the orphan is
// reaped, the event is preserved exactly once (never duplicated,
// never relaunched-for), and redelivery names the continue path.
func TestServerEmitThenCrashNeverRelaunches(t *testing.T) {
	dir := t.TempDir()
	setup := f4Server(t, dir, &overlapAdapter{})
	f4Record(t, setup, "b-f4e")
	// A live "agent" orphan, spawned the production way (supervisor:
	// own process group) and identifiable by executable name.
	sleepBin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("no sleep binary; cannot test orphan reap")
	}
	link := filepath.Join(dir, "opencode")
	if err := os.Symlink(sleepBin, link); err != nil {
		t.Fatal(err)
	}
	sup := testSupervisor(t)
	proc, err := sup.Start(context.Background(), "orphan", dir, link, []string{"300"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = proc.Kill() })
	reaped := make(chan struct{})
	go func() { _, _ = proc.Wait(); close(reaped) }()
	// Faithful crash state: won claim with launch identity + one
	// persisted outbox event, no completion.
	payload := map[string]any{"question_id": "q", "kind": "clarification", "answer": "a", "turn": 1}
	if res, _, _, err := setup.Dedup.Claim(ClaimRecord{
		MessageID: "emit-id", Name: CmdContinue, PayloadDigest: roundTripDigest(t, payload),
		BuildID: "b-f4e", SessionRef: "sess-b-f4e", Turn: 1,
		InvocationID: "inv-emit", PID: proc.Pid(),
	}); err != nil || res != ClaimWon {
		t.Fatalf("claim = %v %v", res, err)
	}
	env, err := Sign(StaticTestKey(), "b-f4e", "w-f4", "ev-1", "agent.message",
		map[string]any{"text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if err := setup.Outbox.Enqueue(OutboxItem{ID: "ev-1", BuildID: "b-f4e", MessageID: "ev-1", Envelope: env}); err != nil {
		t.Fatal(err)
	}
	// Reboot: reopen the same state and recover. SelfTicks far in the
	// future fixtures "gateway booted after the orphan started" (the
	// real comparison is covered by the M7 unit test).
	postAdapter := &overlapAdapter{}
	srv2 := f4Server(t, dir, postAdapter)
	srv2.SelfTicks = 1 << 62
	srv2.RecoverSessions(context.Background())
	select {
	case <-reaped:
	case <-time.After(5 * time.Second):
		t.Fatal("claim orphan not reaped by recovery")
	}
	// The persisted event survives exactly once (plus recovery's own
	// deterministic failed callback — never a duplicate of ev-1).
	var ids []string
	for _, item := range srv2.Outbox.Due(time.Now()) {
		ids = append(ids, item.MessageID)
	}
	seen := 0
	for _, id := range ids {
		if id == "ev-1" {
			seen++
		}
	}
	if seen != 1 {
		t.Errorf("ev-1 occurrences = %d, want exactly 1 (ids %v)", seen, ids)
	}
	// Redelivery neither relaunches nor replays success.
	raw := signedCommand(t, "b-f4e", "emit-id", CmdContinue, payload)
	code, out := postCommand(t, srv2, raw)
	if code != http.StatusConflict {
		t.Fatalf("redelivery = %d, want 409", code)
	}
	body, _ := out["error"].(string)
	if !strings.Contains(strings.ToLower(body), "indeterminate") {
		t.Errorf("redelivery error = %q, want indeterminate + continue path", body)
	}
	postAdapter.mu.Lock()
	defer postAdapter.mu.Unlock()
	if postAdapter.continues != 0 {
		t.Errorf("post-recovery executions = %d, want 0", postAdapter.continues)
	}
}

// TestServerPidlessClaimIsIndeterminate (F4/A1): crashes between claim
// persistence, process start, and pid recording all look identical on
// disk (a claim with no pid) — and all recover indeterminate, since no
// child can be proven absent.
func TestServerPidlessClaimIsIndeterminate(t *testing.T) {
	dir := t.TempDir()
	setup := f4Server(t, dir, &overlapAdapter{})
	f4Record(t, setup, "b-f4f")
	payload := map[string]any{"question_id": "q", "kind": "clarification", "answer": "a", "turn": 1}
	if res, _, _, err := setup.Dedup.Claim(ClaimRecord{
		MessageID: "nopid", Name: CmdContinue, PayloadDigest: roundTripDigest(t, payload),
		BuildID: "b-f4f", SessionRef: "sess-b-f4f", Turn: 1, InvocationID: "inv-nopid",
	}); err != nil || res != ClaimWon {
		t.Fatalf("claim = %v %v", res, err)
	}
	srv2 := f4Server(t, dir, &overlapAdapter{})
	srv2.RecoverSessions(context.Background())
	raw := signedCommand(t, "b-f4f", "nopid", CmdContinue, payload)
	code, out := postCommand(t, srv2, raw)
	if code != http.StatusConflict {
		t.Fatalf("redelivery = %d, want 409", code)
	}
	body, _ := out["error"].(string)
	if !strings.Contains(strings.ToLower(body), "indeterminate") {
		t.Errorf("redelivery error = %q, want indeterminate", body)
	}
}

// gateAdapter holds run 1's event channel open until the test releases
// it; resumes attempted while it is live fail busy.
type gateAdapter struct {
	mu       sync.Mutex
	attempts int
	launches int
	runCh    chan AgentEvent
}

func (a *gateAdapter) Start(context.Context, BuildAssignment) (string, <-chan AgentEvent, error) {
	ch := make(chan AgentEvent)
	close(ch)
	return "gate-sess", ch, nil
}

func (a *gateAdapter) Continue(context.Context, string, UserInput) (<-chan AgentEvent, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.attempts++
	if a.attempts == 1 {
		a.launches++
		return a.runCh, nil
	}
	select {
	case <-a.runCh:
		a.launches++
		ch := make(chan AgentEvent)
		close(ch)
		return ch, nil
	default:
		return nil, ErrSessionBusy
	}
}

func (a *gateAdapter) Status(_ context.Context, ref string) (AgentStatus, error) {
	return AgentStatus{Alive: true, SessionRef: ref}, nil
}

func (a *gateAdapter) Cancel(context.Context, string) error { return nil }

func (a *gateAdapter) Collect(context.Context, string, Manifest) ([]Artifact, error) {
	return nil, nil
}

func f4postCommand(t *testing.T, srv *Server, ctx context.Context, body []byte) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/gw/v1/command", strings.NewReader(string(body)))
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	var decoded map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return rec.Code, decoded
}

// TestServerBusyDropRetryable (F4): a resume refused busy drops its
// claim (no child started) and stays retryable — the same id succeeds
// once the run ends.
func TestServerBusyDropRetryable(t *testing.T) {
	dir := t.TempDir()
	adapter := &gateAdapter{runCh: make(chan AgentEvent)}
	srv := f4Server(t, dir, adapter)
	f4Record(t, srv, "b-f4g")
	first := signedCommand(t, "b-f4g", "run-1", CmdContinue,
		map[string]any{"question_id": "q", "kind": "clarification", "answer": "a", "turn": 1})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	firstCode := make(chan int, 1)
	go func() {
		code, _ := f4postCommand(t, srv, ctx, first)
		firstCode <- code
	}()
	// Wait for the leader to complete (it then holds the build
	// semaphore through run end).
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, ok := srv.Dedup.Check("b-f4g", "run-1"); ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("leader never completed")
		}
		time.Sleep(10 * time.Millisecond)
	}
	second := signedCommand(t, "b-f4g", "run-2", CmdContinue,
		map[string]any{"question_id": "q", "kind": "clarification", "answer": "b", "turn": 1})
	secondCode := make(chan int, 1)
	var secondBody map[string]any
	go func() {
		code, out := postCommand(t, srv, second)
		secondBody = out
		secondCode <- code
	}()
	time.Sleep(100 * time.Millisecond) // let the second queue on the build
	cancel()                           // leader releases early; the run stays live
	if c := <-firstCode; c != http.StatusOK {
		t.Errorf("leader = %d, want recorded 200", c)
	}
	if c := <-secondCode; c != http.StatusConflict {
		t.Errorf("busy resume = %d, want 409", c)
	} else if body, _ := secondBody["error"].(string); !strings.Contains(body, "live run") {
		t.Errorf("busy error = %q, want live-run detail", body)
	}
	if _, ok := srv.Dedup.Check("b-f4g", "run-2"); ok {
		t.Error("busy 409 was recorded; want unrecorded (retryable)")
	}
	close(adapter.runCh) // run 1 ends; the id retries cleanly
	code, _ := postCommand(t, srv, second)
	if code != http.StatusOK {
		t.Errorf("retry after run end = %d, want 200", code)
	}
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.launches != 2 || adapter.attempts != 3 {
		t.Errorf("launches/attempts = %d/%d, want 2/3", adapter.launches, adapter.attempts)
	}
}

// TestServerClaimFailureDispatchesNothing (F4): when the claim itself
// cannot persist, nothing dispatches — setup failure precedes any side
// effect by construction.
func TestServerClaimFailureDispatchesNothing(t *testing.T) {
	dir := t.TempDir()
	adapter := &overlapAdapter{}
	srv := f4Server(t, dir, adapter)
	f4Record(t, srv, "b-f4h")
	_ = os.Remove(filepath.Join(dir, "dedup.json"))
	if err := os.Mkdir(filepath.Join(dir, "dedup.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := signedCommand(t, "b-f4h", "nlaim", CmdContinue,
		map[string]any{"question_id": "q", "kind": "clarification", "answer": "a", "turn": 1})
	code, _ := postCommand(t, srv, raw)
	if code != http.StatusInternalServerError {
		t.Errorf("code = %d, want 500 setup failure", code)
	}
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.continues != 0 {
		t.Errorf("executions = %d, want 0 (claim failed)", adapter.continues)
	}
}

// TestServerAbortRun (F4): aborting cancels the live run and fails its
// record; terminal records are untouched.
func TestServerAbortRun(t *testing.T) {
	srv, adapter := testServer(t)
	if err := srv.Sessions.Create("b-ab", SessionRecord{
		SessionRef: "sess-ab", BuildID: "b-ab", Workspace: "w",
		WorkDir: t.TempDir(), State: SessionRunning,
	}); err != nil {
		t.Fatal(err)
	}
	srv.abortRun("b-ab")
	adapter.mu.Lock()
	cancels := len(adapter.Cancels)
	adapter.mu.Unlock()
	if cancels != 1 {
		t.Errorf("cancels = %d, want 1", cancels)
	}
	rec, _ := srv.Sessions.Get("b-ab")
	if rec.State != SessionFailed {
		t.Errorf("record state = %q, want failed", rec.State)
	}
	if !rec.Degraded {
		t.Error("aborted record not marked degraded (callbacks were lost)")
	}
	srv.abortRun("b-ab") // terminal: no-op
	adapter.mu.Lock()
	cancels = len(adapter.Cancels)
	adapter.mu.Unlock()
	if cancels != 1 {
		t.Errorf("cancels after terminal abort = %d, want still 1", cancels)
	}
}

// TestServerInFlightCrashIsIndeterminate (F4/A1): a crash between claim
// and dispatch completion leaves the command indeterminate — redelivery
// must name recovery via continue, never relaunch.
func TestServerInFlightCrashIsIndeterminate(t *testing.T) {
	dir := t.TempDir()
	adapter := &overlapAdapter{entered: make(chan struct{}), block: make(chan struct{})}
	srv := f4Server(t, dir, adapter)
	f4Record(t, srv, "b-f4d")
	raw := signedCommand(t, "b-f4d", "crash-id", CmdContinue,
		map[string]any{"question_id": "q", "kind": "clarification", "answer": "a", "turn": 1})
	abandoned := make(chan struct{})
	go func() { _, _ = postCommand(t, srv, raw); close(abandoned) }()
	select {
	case <-adapter.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("dispatch never started")
	}
	// Crash: abandon the old server objects, reopen the same state, run
	// boot recovery, then redeliver.
	postAdapter := &overlapAdapter{}
	srv2 := f4Server(t, dir, postAdapter)
	srv2.RecoverSessions(context.Background())
	code, out := postCommand(t, srv2, raw)
	if code != http.StatusConflict {
		t.Fatalf("redelivery = %d, want 409", code)
	}
	body, _ := out["error"].(string)
	if !strings.Contains(strings.ToLower(body), "indeterminate") {
		t.Errorf("redelivery error = %q, want indeterminate + continue path", body)
	}
	postAdapter.mu.Lock()
	if postAdapter.continues != 0 {
		t.Errorf("post-recovery executions = %d, want 0 (no relaunch)", postAdapter.continues)
	}
	postAdapter.mu.Unlock()
	// Release the abandoned dispatch and wait it out: its post-crash
	// writes must finish before TempDir cleanup.
	close(adapter.block)
	<-abandoned
}
