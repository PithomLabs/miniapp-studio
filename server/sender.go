package server

// The sender is Studio's durable dispatch loop to Builder Gateways. Tools
// record outbound commands as studio.commands rows inside their own
// transactions; the sender drives each row to exactly one gateway outcome:
//
//   - The row id is the stable gateway message id across every retry, so
//     redelivery after a crash collapses onto the gateway's recorded
//     response instead of executing twice.
//   - The single worker dispatches serially (per-build order falls out:
//     assign settles before its continues are claimed), and every
//     settlement — ack, backoff retry, prerequisite wait, terminal
//     failure, recovery row — commits atomically with the tracked
//     gateway_turn update.
//   - A 2xx means the gateway launched the turn; run outcomes arrive
//     later as callbacks through build.ingest_event. The sender never
//     infers execution from anything else.
//   - gateway_turn advances only from gateway-acknowledged accepts and
//     status reconciles, never from Studio's own turn counters.
//
// Retry budget: transport errors, 429/5xx, busy sessions, and failed
// reconciles burn one attempt with exponential backoff up to MaxAttempts
// (default 25). Prerequisite waits (no gateway URL, no prompt yet,
// unprovisioned gateway) touch the row without burning. Deterministic
// refusals (bad request, auth, terminal session, desync, twice-stale,
// twice-ambiguous) fail the row with operator-actionable detail.
//
// Recovery rows (stale-turn supersedes, indeterminate re-issues) carry a
// recovered_from marker; recovery fires once per content lineage, so a
// faulty or out-of-band-driven gateway degrades to a visible terminal
// failure instead of an unbounded row chain.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp-studio/gateway"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/tools"
)

const (
	defaultSenderPoll       = time.Second
	defaultSenderWait       = 30 * time.Second
	defaultSenderReqTimeout = 30 * time.Second
	defaultSenderAttempts   = 25
	defaultSenderBaseWait   = 2 * time.Second
	defaultSenderMaxWait    = 5 * time.Minute
	// senderBatchLimit caps the rows one poll pass dispatches so a deep
	// backlog cannot starve the loop's shutdown check.
	senderBatchLimit = 100
	// recoveredFromKey marks sender-created recovery rows with the id of
	// the row they supersede. Recovery fires once per lineage.
	recoveredFromKey = "recovered_from"
)

// SenderConfig tunes the dispatch loop. Zero values take the defaults
// via withDefaults; tests shrink the waits.
type SenderConfig struct {
	PollInterval   time.Duration
	WaitInterval   time.Duration
	RequestTimeout time.Duration
	MaxAttempts    int
	BaseWait       time.Duration
	MaxWait        time.Duration
}

// DefaultSenderConfig is the production tuning: 1s polls, 30s waits,
// 25 attempts over ~90 minutes of backoff.
func DefaultSenderConfig() SenderConfig {
	return SenderConfig{}.withDefaults()
}

func (c SenderConfig) withDefaults() SenderConfig {
	if c.PollInterval <= 0 {
		c.PollInterval = defaultSenderPoll
	}
	if c.WaitInterval <= 0 {
		c.WaitInterval = defaultSenderWait
	}
	if c.RequestTimeout <= 0 {
		c.RequestTimeout = defaultSenderReqTimeout
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = defaultSenderAttempts
	}
	if c.BaseWait <= 0 {
		c.BaseWait = defaultSenderBaseWait
	}
	if c.MaxWait <= 0 {
		c.MaxWait = defaultSenderMaxWait
	}
	return c
}

// senderBinding is one tenant the sender polls. Assign commands carry the
// tenant as workspace_id (gateway plan identifiers: workspace_id = tenant,
// server-derived): the gateway echoes it on callbacks and Studio routes
// them by tenant lookup — a binding here would 404 every callback.
type senderBinding struct {
	tenant string
}

// Sender drives studio.commands rows to their gateway. The zero value is
// useless; StartSender builds, sweeps, and runs one.
type Sender struct {
	pool    *pgxpool.Pool
	keys    gateway.KeyProvider
	tenants []senderBinding
	http    *gateway.Client
	cfg     SenderConfig
	logger  *slog.Logger

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	running bool
}

// errSenderShutdown marks a gateway call interrupted by sender shutdown:
// the attempt's outcome is unknown, so the row requeues without burning
// (gateway dedup makes the redelivery safe).
var errSenderShutdown = errors.New("sender: shutdown during gateway call")

// StartSender sweeps rows orphaned mid-dispatch by a crash, then starts
// the dispatch loop. It fails loud when gateway keys are unconfigured —
// silently wedging every command row is not an option — and refuses a
// second loop. Tenants snapshot at start; operator changes need a
// restart, like every other server wiring.
func (s *Server) StartSender(cfg SenderConfig) error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.closed {
		return fmt.Errorf("server: closed")
	}
	if s.sender != nil {
		return fmt.Errorf("server: sender already running")
	}
	if s.gwKeys == nil {
		return fmt.Errorf("server: sender requires gateway keys (STUDIO_GATEWAY_ORG_SECRET)")
	}
	if s.pool == nil {
		return fmt.Errorf("server: sender requires a database pool")
	}
	cfg = cfg.withDefaults()
	var tenants []senderBinding
	for tenant := range s.tenants {
		tenants = append(tenants, senderBinding{tenant: tenant})
	}
	sort.Slice(tenants, func(i, j int) bool { return tenants[i].tenant < tenants[j].tenant })
	snd := &Sender{
		pool: s.pool, keys: s.gwKeys, tenants: tenants,
		http: &gateway.Client{Timeout: cfg.RequestTimeout, Retries: 0},
		cfg:  cfg, logger: s.logger,
		done: make(chan struct{}),
	}
	// Retries stay 0: every retry is durable at row level with backoff in
	// next_attempt_at. Client-side redelivery would burn attempts the
	// database never sees.
	ctx := context.Background()
	for _, b := range tenants {
		if err := snd.sweepTenant(ctx, b.tenant); err != nil {
			return fmt.Errorf("server: sender sweep %s: %w", b.tenant, err)
		}
	}
	runCtx, cancel := context.WithCancel(context.Background())
	snd.cancel = cancel
	snd.running = true
	s.sender = snd
	go snd.loop(runCtx)
	s.logger.Info("sender started", "tenants", len(tenants))
	return nil
}

// StopSender stops the dispatch loop idempotently. In-flight HTTP calls
// abort and their rows requeue without burning; the loop never outlives
// the call.
func (s *Server) StopSender() {
	s.closeMu.Lock()
	snd := s.sender
	s.sender = nil
	s.closeMu.Unlock()
	if snd != nil {
		snd.stop()
	}
}

func (s *Server) stopSenderLocked() {
	snd := s.sender
	s.sender = nil
	if snd != nil {
		snd.stop()
	}
}

func (snd *Sender) stop() {
	snd.mu.Lock()
	if !snd.running {
		snd.mu.Unlock()
		return
	}
	snd.running = false
	cancel := snd.cancel
	done := snd.done
	snd.mu.Unlock()
	cancel()
	<-done
}

// sweepTenant returns rows orphaned in sending (Studio died between claim
// and settle) to queued without burning: the attempt may have landed,
// and its redelivery replays the recorded outcome.
func (snd *Sender) sweepTenant(ctx context.Context, tenant string) error {
	return store.InTenantTx(ctx, snd.pool, tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`UPDATE studio.commands SET state = 'queued', updated_at = now()
			 WHERE tenant_id = $1 AND state = 'sending'`, tenant)
		return err
	})
}

func (snd *Sender) loop(ctx context.Context) {
	defer close(snd.done)
	ticker := time.NewTicker(snd.cfg.PollInterval)
	defer ticker.Stop()
	snd.poll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snd.poll(ctx)
		}
	}
}

// PollOnce runs one dispatch pass over every configured tenant. The loop
// calls it on every tick; tests drive it directly for determinism.
func (snd *Sender) PollOnce(ctx context.Context) {
	snd.poll(ctx)
}

func (snd *Sender) poll(ctx context.Context) {
	for _, b := range snd.tenants {
		if ctx.Err() != nil {
			return
		}
		if err := snd.pollTenant(ctx, b); err != nil {
			snd.logger.Error("sender poll", "tenant", b.tenant, "detail", err.Error())
		}
	}
}

func (snd *Sender) pollTenant(ctx context.Context, b senderBinding) error {
	for i := 0; i < senderBatchLimit; i++ {
		if ctx.Err() != nil {
			return nil
		}
		plan, claimed, err := snd.claimAndPlan(ctx, b)
		if err != nil {
			return err
		}
		if !claimed {
			return nil
		}
		if plan == nil {
			// Claim-time skip or fail-fast settled inline; keep draining.
			continue
		}
		st := snd.dispatch(ctx, plan)
		if err := snd.settle(ctx, b.tenant, plan, st); err != nil {
			return err
		}
	}
	return nil
}

// sendPlan is one claimed row with its wire payload resolved. The gateway
// fences id-to-payload stability (payload_mismatch), so the plan freezes
// everything up front: the same wire bytes go out on every attempt.
type sendPlan struct {
	rowID    string
	buildID  string
	kind     string
	attempts int
	payload  map[string]any
	url      string
	// wire is the exact signed payload. Continue rows pass their frozen
	// row payload through verbatim; assigns resolve the prompt below.
	wire map[string]any
	// turn is the frozen continue turn (0 for assign/cancel).
	turn int
	// promptSrc records where an assign prompt came from for evidence.
	promptSrc string
}

// claimAndPlan claims the oldest due row and resolves its wire payload in
// one transaction — no network inside. Prerequisite waits and fail-fast
// validations settle inline (no sending orbit, no burn); otherwise the
// row moves to sending and the plan dispatches outside the transaction.
func (snd *Sender) claimAndPlan(ctx context.Context, b senderBinding) (*sendPlan, bool, error) {
	var plan *sendPlan
	var inline settle
	inlined := false
	err := store.InTenantTx(ctx, snd.pool, b.tenant, func(tx pgx.Tx) error {
		var rowID, buildID, kind string
		var rawPayload []byte
		var attempts int
		err := tx.QueryRow(ctx,
			`UPDATE studio.commands SET state = 'sending', updated_at = now()
			 WHERE id = (
			   SELECT id FROM studio.commands
			   WHERE tenant_id = $1 AND state = 'queued' AND next_attempt_at <= now()
			   ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED
			 )
			 RETURNING id, build_id, kind, payload, attempts`,
			b.tenant).Scan(&rowID, &buildID, &kind, &rawPayload, &attempts)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return fmt.Errorf("sender: claim: %w", err)
		}
		p := &sendPlan{rowID: rowID, buildID: buildID, kind: kind, attempts: attempts}
		if err := json.Unmarshal(rawPayload, &p.payload); err != nil {
			inline = terminalSettle(fmt.Sprintf("command payload is not JSON: %v", err))
			inlined = true
			return snd.settleTx(ctx, tx, b.tenant, p, inline)
		}
		if p.payload == nil {
			p.payload = map[string]any{}
		}
		var build struct {
			url, model, agent, conv string
		}
		err = tx.QueryRow(ctx,
			`SELECT gateway_url, model, agent_kind, conversation_id
			 FROM studio.builds WHERE tenant_id = $1 AND id = $2`,
			b.tenant, buildID).Scan(&build.url, &build.model, &build.agent, &build.conv)
		if err == pgx.ErrNoRows {
			inline = terminalSettle("build row vanished (foreign key should prevent this)")
			inlined = true
			return snd.settleTx(ctx, tx, b.tenant, p, inline)
		}
		if err != nil {
			return fmt.Errorf("sender: load build: %w", err)
		}
		p.url = build.url
		if build.url == "" {
			inline = snd.holdSettle("build has no gateway URL; waiting for provisioning")
			inlined = true
			return snd.settleTx(ctx, tx, b.tenant, p, inline)
		}
		switch kind {
		case gateway.CmdAssign:
			prompt, src, skip, fail := snd.resolvePrompt(ctx, tx, b.tenant, p.payload, build.conv)
			if fail != "" {
				inline = terminalSettle(fail)
				inlined = true
				return snd.settleTx(ctx, tx, b.tenant, p, inline)
			}
			if skip != "" {
				inline = snd.holdSettle(skip)
				inlined = true
				return snd.settleTx(ctx, tx, b.tenant, p, inline)
			}
			p.promptSrc = src
			// model and agent pass through verbatim (possibly empty for
			// the gateway default); Studio never substitutes its own.
			p.wire = map[string]any{
				"workspace_id": b.tenant, "prompt": prompt,
				"model": build.model, "agent": build.agent,
			}
		case gateway.CmdContinue:
			turn, ok := frozenTurn(p.payload["turn"])
			if !ok {
				inline = terminalSettle("continue row lacks a frozen turn (answer.submit must freeze it)")
				inlined = true
				return snd.settleTx(ctx, tx, b.tenant, p, inline)
			}
			p.turn = turn
			wire := make(map[string]any, len(p.payload))
			for k, v := range p.payload {
				wire[k] = v
			}
			p.wire = wire
		case gateway.CmdCancel:
			p.wire = map[string]any{}
		default:
			inline = terminalSettle(fmt.Sprintf("sender does not drive kind %q", kind))
			inlined = true
			return snd.settleTx(ctx, tx, b.tenant, p, inline)
		}
		plan = p
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if inlined {
		return nil, true, nil
	}
	if plan == nil {
		return nil, false, nil
	}
	return plan, true, nil
}

// resolvePrompt finds an assign's turn-1 prompt: the triggering message
// when the row names one, else the conversation's latest user message at
// dispatch time (a build.create title-only assign unblocks when the user
// writes), else the row's title. Empty-handed returns a wait reason —
// never an empty prompt, which the gateway would terminally refuse.
func (snd *Sender) resolvePrompt(ctx context.Context, tx pgx.Tx, tenant string, payload map[string]any, convID string) (prompt, src, skip, fail string) {
	if msgID, _ := payload["message_id"].(string); msgID != "" {
		var body string
		err := tx.QueryRow(ctx,
			`SELECT body FROM studio.messages WHERE tenant_id = $1 AND id = $2`,
			tenant, msgID).Scan(&body)
		if err == pgx.ErrNoRows {
			return "", "", "", fmt.Sprintf("triggering message %s is gone", msgID)
		}
		if err != nil {
			return "", "", "", fmt.Sprintf("load triggering message: %v", err)
		}
		if body == "" {
			return "", "", "", fmt.Sprintf("triggering message %s is empty", msgID)
		}
		return body, "message " + msgID, "", ""
	}
	var body string
	err := tx.QueryRow(ctx,
		`SELECT body FROM studio.messages
		 WHERE tenant_id = $1 AND conversation_id = $2 AND role = 'user'
		 ORDER BY created_at DESC LIMIT 1`,
		tenant, convID).Scan(&body)
	if err == nil && body != "" {
		return body, "latest user message", "", ""
	}
	if err != nil && err != pgx.ErrNoRows {
		return "", "", "", fmt.Sprintf("load latest message: %v", err)
	}
	if title, _ := payload["title"].(string); title != "" {
		return title, "title", "", ""
	}
	return "", "", "assign has no prompt yet; waiting for a user message", ""
}

// frozenTurn parses a continue row's frozen turn: an integral JSON number
// >= 1. Anything else is a producer bug the gateway would refuse; failing
// here keeps the refusal Studio-side and explicit.
func frozenTurn(v any) (int, bool) {
	f, ok := v.(float64)
	if !ok || f < 1 || f != float64(int(f)) {
		return 0, false
	}
	return int(f), true
}

// statusTurn parses a status response's turn the same way, except turn 0
// is structurally valid (next = turn+1 holds for any integral turn).
func statusTurn(v any) (int, bool) {
	f, ok := v.(float64)
	if !ok || f < 0 || f != float64(int(f)) {
		return 0, false
	}
	return int(f), true
}

// settle is one row's settlement: the new state, whether the dispatch
// burned an attempt, when a queued row is next due, the tracked-turn and
// session updates, and at most one recovery row to enqueue atomically.
// attempts counts gateway dispatches performed, so claim-time skips and
// fail-fasts settle with burn=false.
type settle struct {
	state   string
	burn    bool
	wait    time.Duration
	backoff bool
	turn    *int
	session *string
	// flipRunning steps an assigned build to running: the gateway
	// accepted, so a run exists. Set only on assign accepts (fresh or
	// adopted); every other status move belongs to the tools/ingest.
	flipRunning bool
	newRow      *newRowSpec
	detail      string
}

// newRowSpec is one recovery row enqueued in the settlement transaction.
type newRowSpec struct {
	kind    string
	payload map[string]any
}

func ackSettle(turn int, session, detail string) settle {
	st := settle{state: "acked", burn: true, detail: detail}
	if turn >= 0 {
		st.turn = &turn
	}
	if session != "" {
		st.session = &session
	}
	return st
}

func burnSettle(detail string) settle {
	return settle{state: "queued", burn: true, backoff: true, detail: detail}
}

func (snd *Sender) holdSettle(detail string) settle {
	return settle{state: "queued", wait: snd.cfg.WaitInterval, detail: detail}
}

func terminalSettle(detail string) settle {
	return settle{state: "failed", detail: detail}
}

// shutdownSettle requeues without burning: the attempt may or may not
// have landed, and gateway dedup makes the redelivery safe either way.
func shutdownSettle() settle {
	return settle{state: "queued", detail: "shutdown during dispatch; redelivery is safe"}
}

// dispatch sends one claimed plan and classifies the outcome. No database
// work happens here; settle commits the returned settlement atomically.
func (snd *Sender) dispatch(ctx context.Context, plan *sendPlan) settle {
	key, err := snd.keys.KeyFor(plan.buildID)
	if err != nil {
		return terminalSettle(fmt.Sprintf("gateway key resolution failed: %v", err))
	}
	out, err := snd.http.SendWithID(ctx, plan.url, key, plan.buildID, plan.rowID, plan.kind, plan.wire)
	if err != nil && ctx.Err() != nil {
		return shutdownSettle()
	}
	if err == nil {
		return snd.ackPlan(plan, out)
	}
	var cerr *gateway.CallError
	if !errors.As(err, &cerr) {
		return burnSettle(fmt.Sprintf("transport: %v", cappedErr(err)))
	}
	return snd.classify(ctx, plan, cerr)
}

// ackPlan settles a 2xx: the gateway launched the turn. Assign lands turn
// 1 with its session ref; continue lands its frozen turn; cancel needs no
// tracked state. (Build status transitions belong to the U2 handoff.)
func (snd *Sender) ackPlan(plan *sendPlan, out map[string]any) settle {
	switch plan.kind {
	case gateway.CmdAssign:
		ref, _ := out["session_ref"].(string)
		detail := "assign accepted turn 1"
		if ref == "" {
			detail += " (gateway omitted session_ref)"
		}
		if plan.promptSrc != "" {
			detail += " from " + plan.promptSrc
		}
		st := ackSettle(1, ref, detail)
		st.flipRunning = true
		return st
	case gateway.CmdContinue:
		return ackSettle(plan.turn, "", fmt.Sprintf("continue accepted turn %d", plan.turn))
	case gateway.CmdCancel:
		st := settle{state: "acked", burn: true, detail: "cancel accepted"}
		return st
	default:
		return terminalSettle(fmt.Sprintf("acked unexpected kind %q", plan.kind))
	}
}

// statusRec is one parsed status reconcile.
type statusRec struct {
	turn       int
	state      string
	sessionRef string
}

// reconcile asks the gateway where a build stands. found=false means the
// gateway holds no key for the build (unprovisioned or withdrawn), which
// callers treat as a wait — never a failure.
func (snd *Sender) reconcile(ctx context.Context, plan *sendPlan) (statusRec, bool, error) {
	key, err := snd.keys.KeyFor(plan.buildID)
	if err != nil {
		return statusRec{}, false, fmt.Errorf("gateway key resolution failed: %v", err)
	}
	out, err := snd.http.Send(ctx, plan.url, key, plan.buildID, gateway.CmdStatus, map[string]any{})
	if err != nil && ctx.Err() != nil {
		return statusRec{}, false, errSenderShutdown
	}
	if err != nil {
		var cerr *gateway.CallError
		if errors.As(err, &cerr) && cerr.Status == http.StatusNotFound {
			return statusRec{}, false, nil
		}
		return statusRec{}, false, err
	}
	turn, ok := statusTurn(out["turn"])
	if !ok {
		return statusRec{}, false, fmt.Errorf("status lacks a turn: %v", out["turn"])
	}
	state, _ := out["state"].(string)
	ref, _ := out["session_ref"].(string)
	return statusRec{turn: turn, state: state, sessionRef: ref}, true, nil
}

// reconcileOrBurn runs status for a row whose command outcome needs it.
// Status failures requeue the row with backoff: the trigger stays queued
// and the reconcile retried, so a transient status outage never loses or
// duplicates a command.
func (snd *Sender) reconcileOrBurn(ctx context.Context, plan *sendPlan) (statusRec, bool, *settle) {
	rec, found, err := snd.reconcile(ctx, plan)
	if err != nil {
		if errors.Is(err, errSenderShutdown) {
			st := shutdownSettle()
			return rec, false, &st
		}
		st := burnSettle(fmt.Sprintf("status reconcile failed: %v", cappedErr(err)))
		return rec, false, &st
	}
	return rec, found, nil
}

// cappedErr keeps transport details bounded for the last_error column.
func cappedErr(err error) string {
	msg := err.Error()
	if len(msg) > 300 {
		return msg[:300] + "…"
	}
	return msg
}

// classify settles one classified gateway refusal. Every branch is total:
// unknown statuses and uncoded conflicts degrade to visible terminal or
// capped-retry outcomes, never to a silent wedge or an unbounded loop.
func (snd *Sender) classify(ctx context.Context, plan *sendPlan, cerr *gateway.CallError) settle {
	code := ""
	gwErr := ""
	if cerr.Body != nil {
		code, _ = cerr.Body["code"].(string)
		gwErr, _ = cerr.Body["error"].(string)
	}
	// Accepted but unreadable: the command executed (2xx), so adopt the
	// reconciled turn rather than failing a launch that happened.
	if cerr.Status >= 200 && cerr.Status < 300 {
		rec, found, retry := snd.reconcileOrBurn(ctx, plan)
		if retry != nil {
			return *retry
		}
		if !found {
			return burnSettle("accepted but unreadable, and gateway now unknown; retrying")
		}
		return ackSettle(rec.turn, rec.sessionRef,
			fmt.Sprintf("accepted but response undecodable; adopted turn %d", rec.turn))
	}
	switch cerr.Status {
	case http.StatusNotFound:
		// Cancel's goal is no live session; unknown-build holds it.
		if plan.kind == gateway.CmdCancel {
			st := settle{state: "acked", burn: true, detail: "no session at gateway; cancel goal holds"}
			return st
		}
		_, found, retry := snd.reconcileOrBurn(ctx, plan)
		if retry != nil {
			return *retry
		}
		if !found {
			return snd.holdSettle("gateway has no key for build; waiting for provisioning")
		}
		if plan.kind == gateway.CmdAssign {
			return burnSettle("gateway provisioned; assign raced, retrying")
		}
		return terminalSettle("gateway provisioned but session missing (state desync); operator triage")
	case http.StatusConflict:
		return snd.classifyConflict(ctx, plan, code, gwErr)
	case http.StatusUnauthorized, http.StatusForbidden:
		return terminalSettle("gateway refused authentication (provisioning/key mismatch); operator triage")
	}
	if cerr.Status == http.StatusTooManyRequests || cerr.Status >= 500 {
		if code == "indeterminate" {
			return snd.classifyPoison(ctx, plan)
		}
		return burnSettle(fmt.Sprintf("gateway %d: %s; retrying", cerr.Status, gwErr))
	}
	if cerr.Status >= 400 && cerr.Status < 500 {
		return terminalSettle(fmt.Sprintf("gateway refused %d: %s", cerr.Status, gwErr))
	}
	return burnSettle(fmt.Sprintf("unexpected gateway status %d; retrying", cerr.Status))
}

// classifyConflict settles a 409 by its machine-readable code. Busy
// retries under the same id (the gateway recorded nothing); stale,
// already-assigned, and indeterminate reconcile through status exactly
// once per content lineage, then supersede or adopt.
func (snd *Sender) classifyConflict(ctx context.Context, plan *sendPlan, code, gwErr string) settle {
	switch code {
	case "session_busy":
		return burnSettle("session has a live run; retrying when idle")
	case "stale_turn":
		if plan.kind != gateway.CmdContinue {
			return terminalSettle(fmt.Sprintf("unexpected stale_turn for %s", plan.kind))
		}
		if recoveredFrom(plan.payload) {
			return terminalSettle("turn went stale twice; out-of-band driver suspected; operator triage")
		}
		rec, found, retry := snd.reconcileOrBurn(ctx, plan)
		if retry != nil {
			return *retry
		}
		if !found {
			return snd.holdSettle("session vanished mid-flight; waiting")
		}
		return supersedeSettle(gateway.CmdContinue, continuePayload(plan.payload, rec.turn+1, plan.rowID),
			rec.turn, fmt.Sprintf("superseded at turn %d after stale turn", rec.turn+1))
	case "already_assigned":
		if plan.kind != gateway.CmdAssign {
			return terminalSettle(fmt.Sprintf("unexpected already_assigned for %s", plan.kind))
		}
		rec, found, retry := snd.reconcileOrBurn(ctx, plan)
		if retry != nil {
			return *retry
		}
		if !found {
			return supersedeSettle(gateway.CmdAssign, copyPayload(plan.payload, plan.rowID), -1,
				"gateway lost session after recording assign; re-assigning")
		}
		// Our own earlier attempt landed before Studio recorded it
		// (crash between gateway accept and settle): adopt it.
		st := ackSettle(rec.turn, rec.sessionRef,
			fmt.Sprintf("earlier attempt landed before Studio recorded it; adopted turn %d", rec.turn))
		st.flipRunning = true
		return st
	case "session_terminal":
		return terminalSettle(fmt.Sprintf("gateway session is terminal; %s cannot be delivered", plan.kind))
	case "indeterminate":
		return snd.classifyPoison(ctx, plan)
	case "payload_mismatch":
		return terminalSettle("id collided with a different payload (sender bug); operator triage")
	default:
		return terminalSettle(fmt.Sprintf("unclassified 409: %s", gwErr))
	}
}

// classifyPoison settles an ambiguous dispatch (500 or 409 with code
// indeterminate): the id is retired — never resent — and recovery
// proceeds by content. Continues re-issue as the next turn with the same
// content; cancels re-issue fresh; assigns fail visibly (the turn-1 run
// is gone and its prompt must be consciously re-sent, never silently
// replayed into a turn the agent never saw).
func (snd *Sender) classifyPoison(ctx context.Context, plan *sendPlan) settle {
	if recoveredFrom(plan.payload) {
		return terminalSettle("gateway ambiguous twice for this content; operator triage")
	}
	rec, found, retry := snd.reconcileOrBurn(ctx, plan)
	if retry != nil {
		return *retry
	}
	switch plan.kind {
	case gateway.CmdAssign:
		if !found {
			return supersedeSettle(gateway.CmdAssign, copyPayload(plan.payload, plan.rowID), -1,
				"gateway has no session; re-assigning")
		}
		return terminalSettle(fmt.Sprintf(
			"gateway aborted the turn-1 run ambiguously at turn %d; send the message again to resume as turn %d",
			rec.turn, rec.turn+1))
	case gateway.CmdContinue:
		if !found {
			return terminalSettle("gateway lost the session; re-assign required (operator)")
		}
		return supersedeSettle(gateway.CmdContinue, continuePayload(plan.payload, rec.turn+1, plan.rowID),
			rec.turn, fmt.Sprintf("superseded at turn %d after ambiguous dispatch", rec.turn+1))
	case gateway.CmdCancel:
		if !found {
			st := settle{state: "acked", burn: true, detail: "no session at gateway; cancel goal holds"}
			return st
		}
		return supersedeSettle(gateway.CmdCancel, copyPayload(plan.payload, plan.rowID), -1,
			"cancel ambiguous; retrying with a new id")
	default:
		return terminalSettle(fmt.Sprintf("ambiguous dispatch of unexpected kind %q", plan.kind))
	}
}

// recoveredFrom reports whether a row payload already descends from a
// recovery: recovery fires once per content lineage.
func recoveredFrom(payload map[string]any) bool {
	s, _ := payload[recoveredFromKey].(string)
	return s != ""
}

// copyPayload clones a row payload for a recovery row, marked with the
// superseded id.
func copyPayload(payload map[string]any, from string) map[string]any {
	out := make(map[string]any, len(payload)+1)
	for k, v := range payload {
		out[k] = v
	}
	out[recoveredFromKey] = from
	return out
}

// continuePayload builds a superseding continue: same question content,
// fresh turn, marked lineage. Only content keys carry over — never the
// stale turn or any wire-only field.
func continuePayload(payload map[string]any, turn int, from string) map[string]any {
	out := map[string]any{
		"turn":           float64(turn),
		recoveredFromKey: from,
	}
	for _, k := range []string{"question_id", "kind", "answer"} {
		if v, ok := payload[k]; ok {
			out[k] = v
		}
	}
	return out
}

// supersedeSettle fails the trigger row and enqueues its recovery row
// atomically; settle appends the fresh row's id to the detail.
func supersedeSettle(kind string, payload map[string]any, turn int, detail string) settle {
	st := settle{state: "failed", burn: true, detail: detail,
		newRow: &newRowSpec{kind: kind, payload: payload}}
	if turn >= 0 {
		st.turn = &turn
	}
	return st
}

// settle commits one settlement atomically: the recovery row first (so
// its id lands in the trigger's detail), then the trigger update guarded
// on still being ours, then the tracked-turn/session updates.
func (snd *Sender) settle(ctx context.Context, tenant string, plan *sendPlan, st settle) error {
	return store.InTenantTx(ctx, snd.pool, tenant, func(tx pgx.Tx) error {
		return snd.settleTx(ctx, tx, tenant, plan, st)
	})
}

func (snd *Sender) settleTx(ctx context.Context, tx pgx.Tx, tenant string, plan *sendPlan, st settle) error {
	detail := st.detail
	if st.newRow != nil {
		newID, err := tools.LogCommandTx(ctx, tx, tenant, plan.buildID, st.newRow.kind, st.newRow.payload)
		if err != nil {
			return fmt.Errorf("sender: recovery row: %w", err)
		}
		detail += "; see " + newID
	}
	burn := 0
	if st.burn {
		burn = 1
	}
	next := time.Now()
	switch {
	case st.backoff:
		next = next.Add(snd.backoffFor(plan.attempts + 1))
	case st.wait > 0:
		next = next.Add(st.wait)
	}
	cmd, err := tx.Exec(ctx,
		`UPDATE studio.commands SET state = $1, attempts = attempts + $2,
		 last_error = $3, next_attempt_at = $4, updated_at = now()
		 WHERE tenant_id = $5 AND id = $6 AND state = 'sending'`,
		st.state, burn, detail, next, tenant, plan.rowID)
	if err != nil {
		return fmt.Errorf("sender: settle row: %w", err)
	}
	if cmd.RowsAffected() != 1 {
		return fmt.Errorf("sender: row %s lost mid-flight", plan.rowID)
	}
	// Burned retries exhaust the budget terminally.
	if st.burn && st.state == "queued" && plan.attempts+1 >= snd.cfg.MaxAttempts {
		if _, err := tx.Exec(ctx,
			`UPDATE studio.commands SET state = 'failed',
			 last_error = $1, updated_at = now()
			 WHERE tenant_id = $2 AND id = $3`,
			fmt.Sprintf("exhausted %d attempts; last: %s", snd.cfg.MaxAttempts, detail),
			tenant, plan.rowID); err != nil {
			return fmt.Errorf("sender: exhaust row: %w", err)
		}
	}
	if st.flipRunning {
		// The accept proves a run; events racing ahead may already
		// have moved the build, so the flip is conditional and
		// silent when something else got there first.
		if _, err := tx.Exec(ctx,
			`UPDATE studio.builds SET status = 'running', updated_at = now()
			 WHERE tenant_id = $1 AND id = $2 AND status = 'assigned'`,
			tenant, plan.buildID); err != nil {
			return fmt.Errorf("sender: flip running: %w", err)
		}
	}
	if st.turn != nil || st.session != nil {
		switch {
		case st.turn != nil && st.session != nil:
			_, err = tx.Exec(ctx,
				`UPDATE studio.builds SET gateway_turn = $1, agent_session_ref = $2, updated_at = now()
				 WHERE tenant_id = $3 AND id = $4`, *st.turn, *st.session, tenant, plan.buildID)
		case st.turn != nil:
			_, err = tx.Exec(ctx,
				`UPDATE studio.builds SET gateway_turn = $1, updated_at = now()
				 WHERE tenant_id = $2 AND id = $3`, *st.turn, tenant, plan.buildID)
		default:
			_, err = tx.Exec(ctx,
				`UPDATE studio.builds SET agent_session_ref = $1, updated_at = now()
				 WHERE tenant_id = $2 AND id = $3`, *st.session, tenant, plan.buildID)
		}
		if err != nil {
			return fmt.Errorf("sender: track turn: %w", err)
		}
	}
	return nil
}

// backoffFor is attempt n's wait: base*2^(n-1) capped at max. No jitter —
// one worker needs none, and determinism keeps tests honest.
func (snd *Sender) backoffFor(n int) time.Duration {
	wait := float64(snd.cfg.BaseWait)
	for i := 1; i < n; i++ {
		wait *= 2
		if wait >= float64(snd.cfg.MaxWait) {
			return snd.cfg.MaxWait
		}
	}
	if wait > float64(snd.cfg.MaxWait) {
		return snd.cfg.MaxWait
	}
	return time.Duration(wait)
}
