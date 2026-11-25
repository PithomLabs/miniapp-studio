package gateway

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Command names (Studio → gateway).
const (
	CmdAssign        = "assign"
	CmdContinue      = "continue"
	CmdCancel        = "cancel"
	CmdStatus        = "status"
	CmdFetchArtifact = "fetch_artifact"
	// Outbox inspect/requeue are the operator recovery path for
	// terminally failed callbacks. Both are per-build signed and
	// scoped: a build sees and requeues only its own items.
	CmdOutboxInspect = "outbox_inspect"
	CmdOutboxRequeue = "outbox_requeue"
)

// maxCommandBytes caps inbound command bodies. Assignments carry prompts,
// not bulk data; artifacts move through fetch_artifact responses.
const maxCommandBytes = 1 << 20

// Server is the gateway command handler: it verifies signed Studio
// commands, drives one AgentAdapter, and enqueues §8 callbacks into the
// outbox for the deliverer. Sessions are keyed by build: one build owns
// one agent session line, and Studio correlates everything downstream.
//
// Effectful commands execute under a durable claim (persisted before
// dispatch) plus two in-process coordinators: a per-(build,message)
// call table sharing one execution among duplicate deliveries, and a
// per-build semaphore serializing distinct commands through run end.
// Lock order is call-join → build semaphore → store mutexes, never
// reversed; no blocking operation runs under a held mutex, so the order
// cannot deadlock. All coordinators lazy-initialize: a zero Server is
// usable.
type Server struct {
	Keys     KeyProvider
	Adapter  AgentAdapter
	Sessions *SessionManager
	Outbox   *Outbox
	Dedup    *DedupStore
	WorkRoot string
	Logger   *slog.Logger
	// ProcRoot overrides /proc for tests (orphan identity checks).
	ProcRoot string
	// IOGID is the inbox group for created build workdirs. Zero
	// defaults to the gateway's egid; production sets studio-io.
	IOGID int
	// SelfTicks is the gateway's own start time in kernel ticks (for
	// orphan start-time validation). serve() reads it from the real
	// /proc at boot; tests inject fixture values; <= 0 skips the time
	// gate (argv[0]-exact still applies).
	SelfTicks int64

	callsMu sync.Mutex
	calls   map[string]*callFlight
	buildMu sync.Mutex
	builds  map[string]chan struct{}
}

// callFlight shares one leader execution among duplicate in-flight
// deliveries of a message id. Waiters with a mismatched payload digest
// never join (they conflict immediately).
type callFlight struct {
	digest string
	done   chan struct{}
	status int
	body   []byte
}

// claimAction tells runClaimed how to settle the won claim after
// dispatch: record terminally, drop (dispatch never began), or mark
// indeterminate (a child may exist).
type claimAction int

const (
	claimComplete claimAction = iota
	claimDrop
	claimIndeterminate
)

// dispatchOut is one effectful handler's outcome: the response, the run
// channel when a run launched (nil otherwise), the pump-done channel
// closed when the run's events fully drain (nil without a run), and the
// claim settlement.
type dispatchOut struct {
	status  int
	body    []byte
	runCh   <-chan AgentEvent
	runDone chan struct{}
	action  claimAction
}

func (s *Server) log() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}

// ServeHTTP serves POST /gw/v1/command. Authentication is the HMAC
// envelope; unknown builds 404 without an oracle; effectful commands
// dedup on (build, message_id) with stored-response replay.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeCommand(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxCommandBytes))
	if err != nil {
		writeCommand(w, http.StatusBadRequest, map[string]any{"error": "unreadable body"})
		return
	}
	env, err := Parse(raw)
	if err != nil {
		writeCommand(w, http.StatusBadRequest, map[string]any{"error": "malformed envelope"})
		return
	}
	key, err := s.Keys.KeyFor(env.BuildID)
	if err != nil {
		writeCommand(w, http.StatusNotFound, map[string]any{"error": "unknown build"})
		return
	}
	if err := VerifyCommand(key, env); err != nil {
		s.log().Warn("gateway command rejected", "build", env.BuildID,
			"message", env.MessageID, "name", env.Name, "detail", err.Error())
		writeCommand(w, http.StatusUnauthorized, map[string]any{"error": "invalid command"})
		return
	}
	if dedupes(env.Name) {
		if rec, ok := s.Dedup.Check(env.BuildID, env.MessageID); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(rec.Status)
			_, _ = w.Write(rec.Body)
			return
		}
		status, body := s.runClaimed(r.Context(), env)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
		return
	}
	status, body := s.dispatch(r.Context(), env)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// runClaimed executes one effectful command under its durable claim:
// join-or-lead the in-flight call, serialize on the build, claim
// durably, dispatch once, settle the claim. Duplicate deliveries share
// the leader's outcome; mismatched payloads conflict without joining.
func (s *Server) runClaimed(ctx context.Context, env Envelope) (status int, body []byte) {
	payload := env.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	digest := payloadDigest(payload)
	flight, shared := s.joinCall(env.BuildID, env.MessageID, digest)
	if flight == nil {
		return http.StatusConflict, commandError("payload_mismatch", "message id reused with different payload")
	}
	if shared {
		select {
		case <-flight.done:
			return flight.status, flight.body
		case <-ctx.Done():
			return http.StatusServiceUnavailable, commandBody(map[string]any{"error": "command in progress; retry"})
		}
	}
	defer func() { s.finishCall(env.BuildID, env.MessageID, flight, status, body) }()
	sem := s.buildSem(env.BuildID)
	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	case <-ctx.Done():
		return http.StatusServiceUnavailable, commandBody(map[string]any{"error": "build busy; retry"})
	}
	invocation, err := newInvocationID()
	if err != nil {
		s.log().Error("gateway invocation id", "build", env.BuildID, "detail", err.Error())
		return http.StatusInternalServerError, commandBody(map[string]any{"error": "command setup failed"})
	}
	res, rec, live, err := s.Dedup.Claim(ClaimRecord{
		MessageID: env.MessageID, Name: env.Name, PayloadDigest: digest,
		BuildID: env.BuildID, Turn: claimTurn(env.Name, payload),
		InvocationID: invocation,
	})
	if err != nil {
		s.log().Error("gateway claim", "build", env.BuildID, "detail", err.Error())
		return http.StatusInternalServerError, commandBody(map[string]any{"error": "command setup failed"})
	}
	switch res {
	case ClaimCompleted:
		return rec.Status, rec.Body
	case ClaimLive:
		if live.PayloadDigest != digest {
			return http.StatusConflict, commandError("payload_mismatch", "message id reused with different payload")
		}
		if live.Indeterminate {
			return http.StatusConflict, commandError("indeterminate", "command indeterminate (gateway restarted before completion); recover with continue as a new turn")
		}
		// Post-recovery residue without a flight means a second
		// gateway shares this state dir (unsupported deployment):
		// refuse honestly rather than dispatching blindly.
		return http.StatusServiceUnavailable, commandBody(map[string]any{"error": "command claimed by another gateway process"})
	default:
		return s.dispatchClaimed(ctx, env, payload)
	}
}

// dispatchClaimed dispatches a won claim and settles it per the
// handler's outcome. Launch successes record before the run is awaited:
// the 200 means launch-accepted (callbacks carry the outcome), and the
// build semaphore stays held through run end so distinct commands
// serialize into ordered runs.
func (s *Server) dispatchClaimed(ctx context.Context, env Envelope, payload map[string]any) (int, []byte) {
	var out dispatchOut
	switch env.Name {
	case CmdAssign:
		out = s.assign(ctx, env.BuildID, payload)
	case CmdContinue:
		out = s.continueBuild(ctx, env.BuildID, payload)
	case CmdCancel:
		out = s.cancelBuild(ctx, env.BuildID)
	case CmdOutboxRequeue:
		out = s.outboxRequeue(env.BuildID, payload)
	default:
		// Unreachable (dedupes gates names); drop defensively.
		_ = s.Dedup.Drop(env.BuildID, env.MessageID)
		return http.StatusBadRequest, commandBody(map[string]any{"error": "unknown command"})
	}
	switch out.action {
	case claimDrop:
		_ = s.Dedup.Drop(env.BuildID, env.MessageID)
		return out.status, out.body
	case claimIndeterminate:
		_ = s.Dedup.MarkIndeterminate(env.BuildID, env.MessageID, "dispatch may have started a child")
		return out.status, out.body
	}
	if out.runCh != nil {
		// Best-effort launch identity for recovery; a missing pid
		// only makes a later crash more conservative.
		if rec, gerr := s.Sessions.Get(env.BuildID); gerr == nil {
			_ = s.Dedup.RecordLaunch(env.BuildID, env.MessageID, rec.SessionRef, rec.PID)
		}
	}
	if err := s.Dedup.Complete(env.BuildID, env.MessageID, DedupRecord{Status: out.status, Body: out.body}); err != nil {
		// Launched but unrecorded: the run's result could never be
		// honestly attributed, so abort it and free the id. Memory
		// drops even when the disk write fails (same-process retry
		// works); the on-disk claim still forces indeterminate on
		// reboot, which is the conservative-correct outcome.
		s.log().Error("gateway complete", "build", env.BuildID, "detail", err.Error())
		s.abortRun(env.BuildID)
		_ = s.Dedup.Drop(env.BuildID, env.MessageID)
		return http.StatusInternalServerError, commandBody(map[string]any{"error": "command aborted while recording; safe to retry"})
	}
	if out.runDone != nil {
		select {
		case <-out.runDone:
		case <-ctx.Done():
		}
	}
	return out.status, out.body
}

// abortRun best-effort cancels a run whose command outcome could not be
// recorded, and marks its session failed AND degraded: callbacks were
// provably lost (that is why the run aborts), so the degraded flag must
// reflect it even if the pump never runs again. The write commutes with
// the pump's own degraded marking (same flag), so no lost update is
// possible between them.
func (s *Server) abortRun(buildID string) {
	rec, err := s.Sessions.Get(buildID)
	if err != nil || rec.SessionRef == "" || terminalSession(rec.State) {
		return
	}
	_ = s.Adapter.Cancel(context.Background(), rec.SessionRef)
	_ = s.Sessions.Update(buildID, func(r *SessionRecord) {
		r.State = SessionFailed
		r.Degraded = true
		r.DegradedDetail = "command aborted while recording; callbacks lost"
	})
}

// joinCall joins the in-flight call for an id (shared=true) or becomes
// its leader. A digest mismatch returns nil (conflict, never join).
func (s *Server) joinCall(buildID, messageID, digest string) (*callFlight, bool) {
	s.callsMu.Lock()
	defer s.callsMu.Unlock()
	if s.calls == nil {
		s.calls = map[string]*callFlight{}
	}
	key := buildID + "\x00" + messageID
	if f, ok := s.calls[key]; ok {
		if f.digest != digest {
			return nil, false
		}
		return f, true
	}
	f := &callFlight{digest: digest, done: make(chan struct{})}
	s.calls[key] = f
	return f, false
}

// finishCall publishes the leader's outcome to waiters and releases the id.
func (s *Server) finishCall(buildID, messageID string, flight *callFlight, status int, body []byte) {
	s.callsMu.Lock()
	defer s.callsMu.Unlock()
	flight.status, flight.body = status, body
	close(flight.done)
	if s.calls[buildID+"\x00"+messageID] == flight {
		delete(s.calls, buildID+"\x00"+messageID)
	}
}

// buildSem returns the build's execution semaphore (capacity 1),
// creating it on first use.
func (s *Server) buildSem(buildID string) chan struct{} {
	s.buildMu.Lock()
	defer s.buildMu.Unlock()
	if s.builds == nil {
		s.builds = map[string]chan struct{}{}
	}
	sem, ok := s.builds[buildID]
	if !ok {
		sem = make(chan struct{}, 1)
		s.builds[buildID] = sem
	}
	return sem
}

// payloadDigest binds a claim to its exact payload (JSON object keys
// marshal sorted, so the digest is deterministic): same id + different
// payload is a conflict, never a shared execution.
func payloadDigest(payload map[string]any) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte("{}")
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// claimTurn extracts the turn identity a claim covers. Assign opens turn
// 1; cancel is not turn-scoped.
func claimTurn(name string, payload map[string]any) int {
	if name == CmdAssign {
		return 1
	}
	if f, ok := payload["turn"].(float64); ok {
		return int(f)
	}
	return 0
}

// newInvocationID mints the correlator binding one claim to its single
// dispatch attempt (logged and stored; never reused).
func newInvocationID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

// dedupes reports whether a command replays stored responses. Reads are
// safe to re-execute and must stay fresh, so only effectful commands dedup.
func dedupes(name string) bool {
	switch name {
	case CmdAssign, CmdContinue, CmdCancel, CmdOutboxRequeue:
		return true
	default:
		return false
	}
}

func writeCommand(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) dispatch(ctx context.Context, env Envelope) (int, []byte) {
	payload := env.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	switch env.Name {
	case CmdAssign:
		out := s.assign(ctx, env.BuildID, payload)
		return out.status, out.body
	case CmdContinue:
		out := s.continueBuild(ctx, env.BuildID, payload)
		return out.status, out.body
	case CmdCancel:
		out := s.cancelBuild(ctx, env.BuildID)
		return out.status, out.body
	case CmdStatus:
		return s.statusBuild(env.BuildID)
	case CmdFetchArtifact:
		return s.fetchArtifact(env.BuildID, payload)
	case CmdOutboxInspect:
		return s.outboxInspect(env.BuildID)
	default:
		return http.StatusBadRequest, commandBody(map[string]any{"error": "unknown command"})
	}
}

func commandBody(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"error":"encode"}`)
	}
	return raw
}

// commandError encodes a machine-readable failure: the human "error"
// string stays byte-stable while "code" gives Studio's sender a taxonomy
// it can classify without string parsing. Codes: already_assigned,
// session_busy, stale_turn, session_terminal, payload_mismatch,
// indeterminate. Responses without a code keep their standing meaning:
// 4xx terminal for the sender's purpose, 5xx same-id retryable.
func commandError(code, msg string) []byte {
	return commandBody(map[string]any{"error": msg, "code": code})
}

func strField(payload map[string]any, key string) string {
	s, _ := payload[key].(string)
	return s
}

func (s *Server) assign(ctx context.Context, buildID string, payload map[string]any) dispatchOut {
	complete := func(status int, body []byte) dispatchOut {
		return dispatchOut{status: status, body: body, action: claimComplete}
	}
	drop := func(status int, body []byte) dispatchOut {
		return dispatchOut{status: status, body: body, action: claimDrop}
	}
	if _, err := s.Sessions.Get(buildID); err == nil {
		return complete(http.StatusConflict, commandError("already_assigned", "build already assigned"))
	}
	prompt := strField(payload, "prompt")
	if prompt == "" {
		return complete(http.StatusBadRequest, commandBody(map[string]any{"error": "prompt required"}))
	}
	// The workspace arrives authenticated inside the signed assign; the
	// gateway echoes it on callbacks so Studio can route them. Studio
	// re-validates by tenant-scoped build lookup — never trusted alone.
	workspace := strField(payload, "workspace_id")
	if workspace == "" {
		return complete(http.StatusBadRequest, commandBody(map[string]any{"error": "workspace_id required"}))
	}
	workDir := filepath.Join(s.WorkRoot, "build-"+sanitizeSegment(buildID))
	ioGID := s.IOGID
	if ioGID == 0 {
		ioGID = os.Getegid()
	}
	if err := SecureDir(workDir, ioGID, os.ModeSetgid|0o770); err != nil {
		s.log().Error("gateway workdir", "build", buildID, "detail", err.Error())
		return drop(http.StatusInternalServerError, commandBody(map[string]any{"error": "workdir failed"}))
	}
	ref, ch, err := s.Adapter.Start(ctx, BuildAssignment{
		BuildID: buildID, WorkDir: workDir, Prompt: prompt, Workspace: workspace,
		Model: strField(payload, "model"), AgentName: strField(payload, "agent"), Turn: 1,
	})
	if err != nil {
		// Lost a concurrent-assign race after the pre-check: the other
		// assign owns the build, so this one conflicts. No stray agent
		// exists — the adapter creates the record before launching.
		if errors.Is(err, ErrSessionExists) {
			return complete(http.StatusConflict, commandError("already_assigned", "build already assigned"))
		}
		// A stray (launched but unrecorded) is ambiguous; every other
		// Start failure means no child exists (Runner contract).
		if errors.Is(err, ErrAgentStray) {
			s.log().Error("gateway start stray", "build", buildID, "detail", err.Error())
			return dispatchOut{status: http.StatusInternalServerError,
				body:   commandError("indeterminate", "agent start ambiguous; recover with continue as a new turn"),
				action: claimIndeterminate}
		}
		s.log().Error("gateway start", "build", buildID, "detail", err.Error())
		return drop(http.StatusInternalServerError, commandBody(map[string]any{"error": "agent start failed"}))
	}
	runDone := make(chan struct{})
	go s.pumpBuild(buildID, ch, runDone)
	if err := s.emit(buildID, "agent.started", map[string]any{"agent_session_id": ref}); err != nil {
		// The run is live but its first callback is lost: abort it so
		// no unattributed run continues, and terminalize the claim —
		// the pump may already have persisted events, so this id must
		// never relaunch.
		s.log().Error("gateway outbox", "build", buildID, "detail", err.Error())
		s.abortRun(buildID)
		return dispatchOut{status: http.StatusInternalServerError,
			body:   commandError("indeterminate", "callback queue failed; run aborted, recover with continue as a new turn"),
			action: claimIndeterminate}
	}
	return dispatchOut{status: http.StatusOK, body: commandBody(map[string]any{"session_ref": ref}),
		runCh: ch, runDone: runDone, action: claimComplete}
}

func (s *Server) continueBuild(ctx context.Context, buildID string, payload map[string]any) dispatchOut {
	complete := func(status int, body []byte) dispatchOut {
		return dispatchOut{status: status, body: body, action: claimComplete}
	}
	drop := func(status int, body []byte) dispatchOut {
		return dispatchOut{status: status, body: body, action: claimDrop}
	}
	rec, err := s.Sessions.Get(buildID)
	if err != nil {
		return complete(http.StatusNotFound, commandBody(map[string]any{"error": "unknown build"}))
	}
	// Failed sessions resume (F6: explicit authorized continue as a new
	// fenced turn); done and cancelled are terminal forever.
	if rec.State == SessionDone || rec.State == SessionCancelled {
		return complete(http.StatusConflict, commandError("session_terminal", "session is terminal"))
	}
	// Turn fencing (M11): the gateway enforces Studio's turn order as
	// defense in depth — every continue carries exactly the next turn.
	traw, present := payload["turn"]
	if !present {
		return complete(http.StatusBadRequest, commandBody(map[string]any{"error": "turn required"}))
	}
	turnFloat, ok := traw.(float64)
	if !ok || turnFloat <= 0 || turnFloat != float64(int(turnFloat)) {
		return complete(http.StatusBadRequest, commandBody(map[string]any{"error": "turn must be a positive integer"}))
	}
	turn := int(turnFloat)
	if turn != rec.Turn+1 {
		return complete(http.StatusConflict, commandError("stale_turn", "stale turn: session is at turn "+itoa(rec.Turn)))
	}
	ch, err := s.Adapter.Continue(ctx, rec.SessionRef, UserInput{
		QuestionID: strField(payload, "question_id"),
		Kind:       strField(payload, "kind"),
		Answer:     strField(payload, "answer"),
		Turn:       turn,
	})
	if err != nil {
		// Busy means the session is legitimately occupied: no child
		// started, so the id stays retryable (unrecorded 409).
		if errors.Is(err, ErrSessionBusy) {
			return drop(http.StatusConflict, commandError("session_busy", "session has a live run; retry when idle"))
		}
		if errors.Is(err, ErrAgentStray) {
			s.log().Error("gateway resume stray", "build", buildID, "detail", err.Error())
			return dispatchOut{status: http.StatusInternalServerError,
				body:   commandError("indeterminate", "resume ambiguous; recover with continue as a new turn"),
				action: claimIndeterminate}
		}
		s.log().Error("gateway continue", "build", buildID, "detail", err.Error())
		return drop(http.StatusInternalServerError, commandBody(map[string]any{"error": "resume failed"}))
	}
	runDone := make(chan struct{})
	go s.pumpBuild(buildID, ch, runDone)
	if err := s.emit(buildID, "agent.resumed", map[string]any{"agent_session_id": rec.SessionRef}); err != nil {
		s.log().Error("gateway outbox", "build", buildID, "detail", err.Error())
		s.abortRun(buildID)
		return dispatchOut{status: http.StatusInternalServerError,
			body:   commandError("indeterminate", "callback queue failed; run aborted, recover with continue as a new turn"),
			action: claimIndeterminate}
	}
	return dispatchOut{status: http.StatusOK, body: commandBody(map[string]any{"resumed": true}),
		runCh: ch, runDone: runDone, action: claimComplete}
}

func (s *Server) cancelBuild(ctx context.Context, buildID string) dispatchOut {
	complete := func(status int, body []byte) dispatchOut {
		return dispatchOut{status: status, body: body, action: claimComplete}
	}
	indeterminate := func(body []byte) dispatchOut {
		return dispatchOut{status: http.StatusInternalServerError, body: body, action: claimIndeterminate}
	}
	rec, err := s.Sessions.Get(buildID)
	if err != nil {
		return complete(http.StatusNotFound, commandBody(map[string]any{"error": "unknown build"}))
	}
	if terminalSession(rec.State) {
		return complete(http.StatusOK, commandBody(map[string]any{"cancelled": true, "already": true}))
	}
	// Cancel errors are ambiguous (the kill may have landed) and so is
	// any failure after it: the agent is dead-or-dying while the record
	// or ack is uncertain. A fresh id re-cancel converges (already-true).
	if err := s.Adapter.Cancel(ctx, rec.SessionRef); err != nil {
		s.log().Error("gateway cancel", "build", buildID, "detail", err.Error())
		return indeterminate(commandError("indeterminate", "cancel ambiguous; retry with a new id"))
	}
	if err := s.Sessions.Update(buildID, func(r *SessionRecord) { r.State = SessionCancelled }); err != nil {
		s.log().Error("gateway cancel record", "build", buildID, "detail", err.Error())
		return indeterminate(commandError("indeterminate", "cancel ambiguous; retry with a new id"))
	}
	if err := s.emit(buildID, "agent.cancelled_ack", map[string]any{}); err != nil {
		s.log().Error("gateway outbox", "build", buildID, "detail", err.Error())
		// The cancel stands (record is cancelled) but its
		// acknowledgement was lost: flag it for reconciliation.
		s.markDegraded(buildID, "cancelled acknowledgement lost")
		return indeterminate(commandError("indeterminate", "cancel ambiguous; retry with a new id"))
	}
	return complete(http.StatusOK, commandBody(map[string]any{"cancelled": true}))
}

func (s *Server) statusBuild(buildID string) (int, []byte) {
	rec, err := s.Sessions.Get(buildID)
	if err != nil {
		return http.StatusNotFound, commandBody(map[string]any{"error": "unknown build"})
	}
	alive := false
	detail := ""
	if !terminalSession(rec.State) {
		if st, serr := s.Adapter.Status(context.Background(), rec.SessionRef); serr == nil {
			alive, detail = st.Alive, st.Detail
		}
	}
	queued, failed := s.Outbox.Depth()
	out := map[string]any{
		"session_ref": rec.SessionRef, "state": rec.State,
		"agent_alive": alive, "detail": detail,
		"outbox_queued": queued, "outbox_failed": failed,
		// turn is the last turn the gateway accepted (assign lands
		// turn 1). Studio reconciles its tracked gateway_turn from
		// this field after stale-turn and indeterminate outcomes —
		// never from its own turn counters.
		"turn": rec.Turn,
	}
	if rec.Degraded {
		out["degraded"] = true
		out["degraded_detail"] = rec.DegradedDetail
	}
	return http.StatusOK, commandBody(out)
}

// outboxInspect lists one build's outbox items for operator triage:
// identity, event name, state, attempts, and last error. Payload bytes
// stay out (bounded responses); the event name suffices to aim a
// requeue or a Studio-side check.
func (s *Server) outboxInspect(buildID string) (int, []byte) {
	if _, err := s.Sessions.Get(buildID); err != nil {
		return http.StatusNotFound, commandBody(map[string]any{"error": "unknown build"})
	}
	items := s.Outbox.ItemsForBuild(buildID)
	out := make([]any, 0, len(items))
	for _, item := range items {
		name := ""
		if env, err := Parse(item.Envelope); err == nil {
			name = env.Name
		}
		out = append(out, map[string]any{
			"id": item.ID, "message_id": item.MessageID, "name": name,
			"state": item.State, "attempts": item.Attempts,
			"requeues": item.Requeues, "last_error": item.LastError,
		})
	}
	return http.StatusOK, commandBody(map[string]any{"items": out})
}

// outboxRequeue returns one terminally failed item of this build to the
// deliverable queue. Cross-build ids 404 without an oracle; live items
// conflict (only failed items requeue).
func (s *Server) outboxRequeue(buildID string, payload map[string]any) dispatchOut {
	complete := func(status int, body []byte) dispatchOut {
		return dispatchOut{status: status, body: body, action: claimComplete}
	}
	if _, err := s.Sessions.Get(buildID); err != nil {
		return complete(http.StatusNotFound, commandBody(map[string]any{"error": "unknown build"}))
	}
	id := strField(payload, "item_id")
	if id == "" {
		return complete(http.StatusBadRequest, commandBody(map[string]any{"error": "item_id required"}))
	}
	item, ok := s.Outbox.Find(id)
	if !ok || item.BuildID != buildID {
		return complete(http.StatusNotFound, commandBody(map[string]any{"error": "unknown item"}))
	}
	if item.State != OutboxFailed {
		return complete(http.StatusConflict, commandBody(map[string]any{"error": "only failed items requeue"}))
	}
	if err := s.Outbox.Requeue(id); err != nil {
		s.log().Error("gateway requeue", "build", buildID, "detail", err.Error())
		return complete(http.StatusInternalServerError, commandBody(map[string]any{"error": "requeue failed"}))
	}
	return complete(http.StatusOK, commandBody(map[string]any{"requeued": id, "requeues": item.Requeues + 1}))
}

func (s *Server) fetchArtifact(buildID string, payload map[string]any) (int, []byte) {
	rec, err := s.Sessions.Get(buildID)
	if err != nil {
		return http.StatusNotFound, commandBody(map[string]any{"error": "unknown build"})
	}
	entries, err := decodeManifest(payload["manifest"])
	if err != nil {
		return http.StatusBadRequest, commandBody(map[string]any{"error": "bad manifest"})
	}
	arts, err := s.Adapter.Collect(context.Background(), rec.SessionRef, Manifest{Entries: entries})
	if err != nil {
		s.log().Error("gateway collect", "build", buildID, "detail", err.Error())
		return http.StatusInternalServerError, commandBody(map[string]any{"error": "collect failed"})
	}
	out := make([]any, 0, len(arts))
	for _, a := range arts {
		out = append(out, map[string]any{
			"path": a.Path, "kind": a.Kind, "digest": a.Digest,
			"size": a.Size, "data_b64": encodeB64(a.Data),
		})
	}
	return http.StatusOK, commandBody(map[string]any{"artifacts": out})
}

// pumpBuild forwards one run's translated events into the outbox as signed
// §8 callbacks until the run channel closes, then closes done so the
// commanding handler's build-serialization wait ends. The first emit
// failure marks the session degraded (durable, status-surfaced) and the
// pump keeps draining undispatched: stalling would wedge the build
// behind a sick disk, while silent loss would hide the gap. Reconciliation
// of degraded sessions belongs to the Phase C Studio sender loop, which
// must treat degraded as backlogged work, not completion.
func (s *Server) pumpBuild(buildID string, ch <-chan AgentEvent, done chan struct{}) {
	defer close(done)
	degraded := false
	for ev := range ch {
		name, payload := s.translate(ev)
		if name == "" {
			continue
		}
		if degraded {
			s.log().Error("gateway pump dropping while degraded", "build", buildID, "event", name)
			continue
		}
		if err := s.emit(buildID, name, payload); err != nil {
			s.log().Error("gateway pump", "build", buildID, "detail", err.Error())
			s.markDegraded(buildID, "callback queue failed: "+truncate(err.Error(), 200))
			degraded = true
		}
	}
}

// markDegraded durably flags callback loss on a session. Best-effort:
// when the session write itself fails the disk is catastrophically
// sick and only the log survives (documented residual).
func (s *Server) markDegraded(buildID, detail string) {
	if err := s.Sessions.Update(buildID, func(r *SessionRecord) {
		r.Degraded = true
		r.DegradedDetail = detail
	}); err != nil {
		s.log().Error("gateway degraded mark", "build", buildID, "detail", err.Error())
	}
}

// translate maps one adapter event to its §8 name + payload. Question ids
// are minted per drop; Studio dedups delivery on the envelope id, so a
// retried callback never double-records.
func (s *Server) translate(ev AgentEvent) (string, map[string]any) {
	switch ev.Kind {
	case EventMessage:
		return "agent.message", map[string]any{"text": ev.Text}
	case EventProgress:
		return "agent.progress", map[string]any{
			"text": ev.Text, "tool": ev.Tool, "ok": ev.ToolOK,
		}
	case EventQuestion:
		q := ev.Question
		if q == nil {
			return "", nil
		}
		var id [8]byte
		if _, err := rand.Read(id[:]); err != nil {
			s.log().Error("gateway question id", "detail", err.Error())
			return "", nil
		}
		options := make([]any, 0, len(q.Options))
		for _, o := range q.Options {
			options = append(options, o)
		}
		return "agent.question", map[string]any{
			"question_id": "q-" + hex.EncodeToString(id[:]),
			"kind":        q.Kind, "body": q.Body, "options": options,
		}
	case EventDone:
		return "agent.completed", map[string]any{}
	case EventFailed:
		return "agent.failed", map[string]any{"reason": ev.Text}
	default:
		return "", nil
	}
}

func terminalSession(state string) bool {
	switch state {
	case SessionDone, SessionFailed, SessionCancelled:
		return true
	default:
		return false
	}
}

// emit signs one gateway→Studio event and enqueues it for delivery. The
// workspace comes from the session record written at assign time. Event
// ids are random per emission.
func (s *Server) emit(buildID, name string, payload map[string]any) error {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return err
	}
	return s.emitWithID(buildID, name, payload, hex.EncodeToString(id[:]))
}

// emitWithID signs and enqueues one event under a caller-chosen id.
// Recovery uses deterministic ids so a crash between emit and mark
// re-emits the same id (Studio dedups) instead of minting duplicates.
func (s *Server) emitWithID(buildID, name string, payload map[string]any, messageID string) error {
	key, err := s.Keys.KeyFor(buildID)
	if err != nil {
		return err
	}
	rec, err := s.Sessions.Get(buildID)
	if err != nil {
		return err
	}
	// Turn identity rides the envelope for Studio-side correlation
	// (M16: populate, don't leave zero), inside the signed bytes.
	// Callbacks never move turns — ingest owns transitions — so this
	// is observability, not control.
	raw, err := signEvent(key, buildID, rec.Workspace, messageID, rec.Turn, name, payload)
	if err != nil {
		return err
	}
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	return s.Outbox.Enqueue(OutboxItem{
		ID: env.MessageID, BuildID: buildID,
		MessageID: env.MessageID, Envelope: raw,
	})
}

// decodeManifest parses a fetch_artifact manifest: entries of {path, kind}.
func decodeManifest(v any) ([]ManifestEntry, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("gateway: manifest must be an object")
	}
	raw, ok := m["entries"].([]any)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("gateway: manifest needs entries")
	}
	var out []ManifestEntry
	for _, e := range raw {
		em, ok := e.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("gateway: bad manifest entry")
		}
		path, _ := em["path"].(string)
		kind, _ := em["kind"].(string)
		if path == "" || kind == "" {
			return nil, fmt.Errorf("gateway: manifest entries need path and kind")
		}
		out = append(out, ManifestEntry{Path: path, Kind: kind})
	}
	return out, nil
}

func encodeB64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func sanitizeSegment(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "build"
	}
	return out
}
