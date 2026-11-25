package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/gateway"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/tools"
)

// senderWorld wires a full Studio server to a real Builder Gateway over
// loopback HTTP: the only doubles are the agent adapter (fake, scripted)
// and the tenant-scoped test database. Studio signs with the static test
// keys the gateway verifies — the same convention as the callback tests.
type senderWorld struct {
	srv     *Server
	snd     *Sender
	gw      *gateway.Server
	adapter *gateway.FakeAdapter
	gwURL   string
}

func testSenderCfg() SenderConfig {
	return SenderConfig{
		PollInterval:   time.Hour, // tests drive PollOnce directly
		WaitInterval:   30 * time.Second,
		RequestTimeout: 5 * time.Second,
		MaxAttempts:    5,
		BaseWait:       time.Second,
		MaxWait:        5 * time.Second,
	}
}

// newSenderWorld builds Studio plus a real gateway and a sender for
// testTenant only. The loop never starts; tests drive PollOnce.
func newSenderWorld(t *testing.T) *senderWorld {
	t.Helper()
	srv, _ := testServer(t)
	srv.gwKeys = gateway.StaticProvider{}
	dir := t.TempDir()
	sessions, err := gateway.NewSessionManager(filepath.Join(dir, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	outbox, err := gateway.OpenOutbox(filepath.Join(dir, "outbox.json"))
	if err != nil {
		t.Fatal(err)
	}
	dedup, err := gateway.OpenDedupStore(filepath.Join(dir, "dedup.json"))
	if err != nil {
		t.Fatal(err)
	}
	adapter := gateway.NewFakeAdapter()
	adapter.Sessions = sessions
	gw := &gateway.Server{
		Keys: gateway.StaticProvider{}, Adapter: adapter, Sessions: sessions,
		Outbox: outbox, Dedup: dedup, WorkRoot: filepath.Join(dir, "work"),
	}
	httpSrv := httptest.NewServer(gw)
	t.Cleanup(httpSrv.Close)
	cfg := testSenderCfg()
	snd := &Sender{
		pool: srv.pool, keys: srv.gwKeys,
		tenants: []senderBinding{{tenant: testTenant}},
		http:    &gateway.Client{Timeout: cfg.RequestTimeout, Retries: 0},
		cfg:     cfg, logger: srv.logger,
		done: make(chan struct{}),
	}
	return &senderWorld{srv: srv, snd: snd, gw: gw, adapter: adapter, gwURL: httpSrv.URL}
}

func (w *senderWorld) invoke(t *testing.T, tenant, tool string, in map[string]any, key string) map[string]any {
	t.Helper()
	out, err := w.srv.invokeToolTenant(context.Background(), tenant, tool, in, key)
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	return out
}

func (w *senderWorld) setGatewayURL(t *testing.T, tenant, build, url string) {
	t.Helper()
	ctx := context.Background()
	err := store.InTenantTx(ctx, w.srv.pool, tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE studio.builds SET gateway_url = $1 WHERE id = $2`, url, build)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

type commandRow struct {
	state       string
	attempts    int
	lastError   string
	nextAttempt time.Time
	payload     map[string]any
}

func (w *senderWorld) getCommand(t *testing.T, tenant, rowID string) commandRow {
	t.Helper()
	ctx := context.Background()
	var row commandRow
	var raw []byte
	err := store.InTenantTx(context.Background(), w.srv.pool, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT state, attempts, last_error, next_attempt_at, payload
			 FROM studio.commands WHERE tenant_id = $1 AND id = $2`,
			tenant, rowID).Scan(&row.state, &row.attempts, &row.lastError, &row.nextAttempt, &raw)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &row.payload); err != nil {
		t.Fatal(err)
	}
	return row
}

func (w *senderWorld) listCommands(t *testing.T, tenant, build string) []commandRow {
	t.Helper()
	ctx := context.Background()
	var rows []commandRow
	err := store.InTenantTx(context.Background(), w.srv.pool, tenant, func(tx pgx.Tx) error {
		rs, err := tx.Query(ctx,
			`SELECT state, attempts, last_error, next_attempt_at, payload
			 FROM studio.commands WHERE tenant_id = $1 AND build_id = $2 ORDER BY created_at`,
			tenant, build)
		if err != nil {
			return err
		}
		defer rs.Close()
		for rs.Next() {
			var row commandRow
			var raw []byte
			if err := rs.Scan(&row.state, &row.attempts, &row.lastError, &row.nextAttempt, &raw); err != nil {
				return err
			}
			if err := json.Unmarshal(raw, &row.payload); err != nil {
				return err
			}
			rows = append(rows, row)
		}
		return rs.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func (w *senderWorld) commandID(t *testing.T, tenant, build, kind string) string {
	t.Helper()
	ctx := context.Background()
	var id string
	err := store.InTenantTx(context.Background(), w.srv.pool, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT id FROM studio.commands WHERE tenant_id = $1 AND build_id = $2 AND kind = $3
			 ORDER BY created_at LIMIT 1`, tenant, build, kind).Scan(&id)
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func (w *senderWorld) insertCommand(t *testing.T, tenant, build, kind string, payload map[string]any, state string, attempts int) string {
	t.Helper()
	ctx := context.Background()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var id string
	err = store.InTenantTx(context.Background(), w.srv.pool, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO studio.commands (id, tenant_id, build_id, kind, payload, state, attempts, next_attempt_at)
			 VALUES ('cmd-' || md5(random()::text || clock_timestamp()::text), $1, $2, $3, $4, $5, $6, now())
			 RETURNING id`,
			tenant, build, kind, json.RawMessage(raw), state, attempts).Scan(&id)
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func (w *senderWorld) buildStatus(t *testing.T, tenant, build string) string {
	t.Helper()
	ctx := context.Background()
	var status string
	err := store.InTenantTx(ctx, w.srv.pool, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT status FROM studio.builds WHERE tenant_id = $1 AND id = $2`,
			tenant, build).Scan(&status)
	})
	if err != nil {
		t.Fatal(err)
	}
	return status
}

func (w *senderWorld) buildTracking(t *testing.T, tenant, build string) (gatewayTurn *int, sessionRef string) {
	t.Helper()
	ctx := context.Background()
	var turn *int
	err := store.InTenantTx(context.Background(), w.srv.pool, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT gateway_turn, agent_session_ref FROM studio.builds WHERE tenant_id = $1 AND id = $2`,
			tenant, build).Scan(&turn, &sessionRef)
	})
	if err != nil {
		t.Fatal(err)
	}
	return turn, sessionRef
}

func (w *senderWorld) openBuild(t *testing.T, tenant, key string) (conv, build string) {
	t.Helper()
	out := w.invoke(t, tenant, tools.ToolConversationCreate, map[string]any{"title": "s"}, "s-conv-"+key)
	conv, _ = out["conversation_id"].(string)
	out = w.invoke(t, tenant, tools.ToolBuildCreate,
		map[string]any{"conversation_id": conv, "title": "do the thing"}, "s-build-"+key)
	build, _ = out["build_id"].(string)
	w.setGatewayURL(t, tenant, build, w.gwURL)
	return conv, build
}

func (w *senderWorld) resetDue(t *testing.T, tenant, rowID string) {
	t.Helper()
	ctx := context.Background()
	err := store.InTenantTx(context.Background(), w.srv.pool, tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE studio.commands SET next_attempt_at = now() WHERE id = $1`, rowID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestSenderAssignChain is the C1 spine: a user message opens a build,
// the sender dispatches its assign exactly once, the row acks, and the
// tracked gateway turn lands on 1 with the gateway's session ref.
func TestSenderAssignChain(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	out := w.invoke(t, testTenant, tools.ToolConversationCreate, map[string]any{"title": "s"}, "s-c1")
	conv, _ := out["conversation_id"].(string)
	out = w.invoke(t, testTenant, tools.ToolMessageSend,
		map[string]any{"conversation_id": conv, "body": "Review invoice INV-1"}, "s-m1")
	build, _ := out["build_id"].(string)
	if build == "" {
		t.Fatalf("message.send opened no build: %v", out)
	}
	w.setGatewayURL(t, testTenant, build, w.gwURL)
	rowID := w.commandID(t, testTenant, build, gateway.CmdAssign)

	w.snd.PollOnce(ctx)
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "acked" || row.attempts != 1 {
		t.Fatalf("assign row = %s/%d (%q), want acked/1", row.state, row.attempts, row.lastError)
	}
	if !strings.Contains(row.lastError, "turn 1") {
		t.Errorf("assign detail = %q, want turn-1 evidence", row.lastError)
	}
	turn, ref := w.buildTracking(t, testTenant, build)
	if turn == nil || *turn != 1 || ref == "" {
		t.Fatalf("tracking = turn %v ref %q, want 1 + session ref", turn, ref)
	}
	if len(w.adapter.Starts) != 1 {
		t.Fatalf("gateway starts = %d, want 1", len(w.adapter.Starts))
	}
	got := w.adapter.Starts[0]
	if got.Prompt != "Review invoice INV-1" {
		t.Errorf("prompt = %q, want the user message", got.Prompt)
	}
	if got.Workspace != testTenant {
		t.Errorf("workspace = %q, want tenant %q", got.Workspace, testTenant)
	}
	if got.Turn != 1 {
		t.Errorf("gateway turn = %d, want 1", got.Turn)
	}
	// A second poll is a no-op: the acked row never redispatches.
	w.snd.PollOnce(ctx)
	if len(w.adapter.Starts) != 1 {
		t.Errorf("gateway starts after second poll = %d, want 1", len(w.adapter.Starts))
	}
}

// TestSenderAssignPassesModel pins verbatim model pass-through: the
// build's pinned model reaches the gateway untouched.
func TestSenderAssignPassesModel(t *testing.T) {
	w := newSenderWorld(t)
	out := w.invoke(t, testTenant, tools.ToolConversationCreate, map[string]any{"title": "s"}, "s-c2")
	conv, _ := out["conversation_id"].(string)
	out = w.invoke(t, testTenant, tools.ToolBuildCreate,
		map[string]any{"conversation_id": conv, "title": "Title prompt", "model": "openrouter/deepseek/deepseek-v4.1-flash"},
		"s-b2")
	build, _ := out["build_id"].(string)
	w.setGatewayURL(t, testTenant, build, w.gwURL)

	w.snd.PollOnce(context.Background())
	if len(w.adapter.Starts) != 1 {
		t.Fatalf("gateway starts = %d, want 1", len(w.adapter.Starts))
	}
	got := w.adapter.Starts[0]
	if got.Model != "openrouter/deepseek/deepseek-v4.1-flash" {
		t.Errorf("model = %q, want verbatim pass-through", got.Model)
	}
	if got.Prompt != "Title prompt" {
		t.Errorf("prompt = %q, want the title (no user message yet)", got.Prompt)
	}
}

// TestSenderContinueChain pins frozen-turn dispatch: a continue row with
// turn 2 lands exactly turn 2 at the gateway with its answer intact.
func TestSenderContinueChain(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "c3")
	w.snd.PollOnce(ctx) // assign lands turn 1
	rowID := w.insertCommand(t, testTenant, build, gateway.CmdContinue, map[string]any{
		"question_id": "q-1", "kind": "clarification", "answer": "Net 30", "turn": float64(2),
	}, "queued", 0)

	w.snd.PollOnce(ctx)
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "acked" {
		t.Fatalf("continue row = %s (%q), want acked", row.state, row.lastError)
	}
	turn, _ := w.buildTracking(t, testTenant, build)
	if turn == nil || *turn != 2 {
		t.Fatalf("tracked turn = %v, want 2", turn)
	}
	if len(w.adapter.Continues) != 1 {
		t.Fatalf("gateway continues = %d, want 1", len(w.adapter.Continues))
	}
	in := w.adapter.Continues[0].Input
	if in.Turn != 2 || in.Answer != "Net 30" || in.QuestionID != "q-1" {
		t.Errorf("continue input = %+v, want turn 2 with the frozen answer", in)
	}
}

// TestSenderContinueRequiresFrozenTurn pins the U2 seam: today's
// answer.submit shape (no frozen turn) fails Studio-side with no gateway
// traffic, instead of guessing a turn at dispatch.
func TestSenderContinueRequiresFrozenTurn(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "c4")
	// Retire the openBuild assign: this test pins the continue alone.
	assignID := w.commandID(t, testTenant, build, gateway.CmdAssign)
	err := store.InTenantTx(ctx, w.srv.pool, testTenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE studio.commands SET state = 'failed' WHERE id = $1`, assignID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	rowID := w.insertCommand(t, testTenant, build, gateway.CmdContinue, map[string]any{
		"question_id": "q-1", "kind": "clarification", "answer": "Net 30",
	}, "queued", 0)

	w.snd.PollOnce(ctx)
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "failed" || !strings.Contains(row.lastError, "frozen turn") {
		t.Fatalf("row = %s (%q), want failed/frozen-turn", row.state, row.lastError)
	}
	if row.attempts != 0 {
		t.Errorf("attempts = %d, want 0 (fail-fast burns nothing)", row.attempts)
	}
	if len(w.adapter.Starts)+len(w.adapter.Continues) != 0 {
		t.Error("gateway saw traffic for a turn-less row, want none")
	}
}

// TestSenderStaleReconcilesOnce pins crash-proof turn binding: Studio
// believes turn 1 while the gateway sits at 3 (out-of-band drives). The
// stale continue supersedes exactly once — new row at turn 4, tracked
// turn repaired from status — and the recovered row then acks with the
// original answer content.
func TestSenderStaleReconcilesOnce(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "c5")
	w.snd.PollOnce(ctx) // assign lands turn 1 on both sides
	// Two out-of-band continues move the gateway to 3 behind our back.
	direct := &gateway.Client{Timeout: 5 * time.Second}
	for i := 2; i <= 3; i++ {
		_, err := direct.Send(ctx, w.gwURL, gateway.StaticTestKey(), build, gateway.CmdContinue,
			map[string]any{"answer": "oob", "turn": float64(i)})
		if err != nil {
			t.Fatalf("oob continue: %v", err)
		}
	}
	rowID := w.insertCommand(t, testTenant, build, gateway.CmdContinue, map[string]any{
		"question_id": "q-9", "kind": "clarification", "answer": "The answer", "turn": float64(2),
	}, "queued", 0)

	w.snd.PollOnce(ctx)
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "failed" || !strings.Contains(row.lastError, "superseded at turn 4") {
		t.Fatalf("stale row = %s (%q), want failed/superseded-at-4", row.state, row.lastError)
	}
	rows := w.listCommands(t, testTenant, build)
	if len(rows) != 3 {
		t.Fatalf("command rows = %d, want 3 (assign + trigger + recovery)", len(rows))
	}
	rec := rows[2]
	if rec.state != "acked" {
		t.Fatalf("recovery row = %s (%q), want acked (same-pass dispatch)", rec.state, rec.lastError)
	}
	if rec.payload["turn"] != float64(4) {
		t.Errorf("recovery turn = %v, want 4", rec.payload["turn"])
	}
	if rec.payload["answer"] != "The answer" || rec.payload["question_id"] != "q-9" {
		t.Errorf("recovery payload = %v, want the original content", rec.payload)
	}
	if rec.payload[recoveredFromKey] == nil {
		t.Errorf("recovery payload lacks %q lineage", recoveredFromKey)
	}
	turn, _ := w.buildTracking(t, testTenant, build)
	if turn == nil || *turn != 4 {
		t.Fatalf("tracked turn = %v, want 4", turn)
	}
	last := w.adapter.Continues[len(w.adapter.Continues)-1].Input
	if last.Turn != 4 || last.Answer != "The answer" {
		t.Errorf("gateway saw %+v, want turn 4 with the original answer", last)
	}
}

// TestSenderStaleTwiceFails bounds the reconcile loop: a row that already
// descends from a recovery and goes stale again fails visibly instead of
// superseding forever.
func TestSenderStaleTwiceFails(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "c6")
	w.snd.PollOnce(ctx)
	direct := &gateway.Client{Timeout: 5 * time.Second}
	if _, err := direct.Send(ctx, w.gwURL, gateway.StaticTestKey(), build, gateway.CmdContinue,
		map[string]any{"answer": "oob", "turn": float64(2)}); err != nil {
		t.Fatal(err)
	}
	rowID := w.insertCommand(t, testTenant, build, gateway.CmdContinue, map[string]any{
		"question_id": "q", "kind": "k", "answer": "a", "turn": float64(2),
		recoveredFromKey: "older-row",
	}, "queued", 0)

	w.snd.PollOnce(ctx)
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "failed" || !strings.Contains(row.lastError, "stale twice") {
		t.Fatalf("row = %s (%q), want failed/stale-twice", row.state, row.lastError)
	}
	if rows := w.listCommands(t, testTenant, build); len(rows) != 2 {
		t.Fatalf("command rows = %d, want 2 (assign + trigger, no second recovery)", len(rows))
	}
}

// TestSenderIndeterminateContinueReissues pins ambiguous-dispatch
// recovery: the poisoned id retires, and the same content re-issues as a
// fresh id at the reconciled next turn.
func TestSenderIndeterminateContinueReissues(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "c7")
	w.snd.PollOnce(ctx) // turn 1 both sides
	w.adapter.Fail["Continue"] = gateway.ErrAgentStray
	rowID := w.insertCommand(t, testTenant, build, gateway.CmdContinue, map[string]any{
		"question_id": "q-2", "kind": "approval", "answer": "Approve", "turn": float64(2),
	}, "queued", 0)

	w.snd.PollOnce(ctx)
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "failed" || !strings.Contains(row.lastError, "ambiguous dispatch") {
		t.Fatalf("poisoned row = %s (%q), want failed/ambiguous", row.state, row.lastError)
	}
	rows := w.listCommands(t, testTenant, build)
	if len(rows) != 3 {
		t.Fatalf("command rows = %d, want 3 (assign + trigger + recovery)", len(rows))
	}
	// The recovery row dispatched in the same pass — and also hit the
	// injected fault, so it failed terminally (recovery fires once).
	rec := rows[2]
	if rec.state != "failed" || !strings.Contains(rec.lastError, "ambiguous twice") {
		t.Fatalf("recovery row = %s (%q), want failed/ambiguous-twice", rec.state, rec.lastError)
	}
	if len(w.adapter.Continues) != 0 {
		t.Fatalf("gateway continues = %d, want 0 (stray never ran)", len(w.adapter.Continues))
	}
	// Clearing the fault and re-asking (fresh content row) converges: the
	// gateway never saw turn 2, so it still wants exactly 2.
	w.adapter.Fail = map[string]error{}
	retryID := w.insertCommand(t, testTenant, build, gateway.CmdContinue, map[string]any{
		"question_id": "q-2", "kind": "approval", "answer": "Approve", "turn": float64(2),
	}, "queued", 0)
	w.snd.PollOnce(ctx)
	retry := w.getCommand(t, testTenant, retryID)
	if retry.state != "acked" {
		t.Fatalf("retry row = %s (%q), want acked", retry.state, retry.lastError)
	}
	turn, _ := w.buildTracking(t, testTenant, build)
	if turn == nil || *turn != 2 {
		t.Fatalf("tracked turn = %v, want 2", turn)
	}
}

// TestSenderIndeterminateAssignReissuesWhenSessionless pins the stray
// path: an ambiguous assign with no gateway session re-assigns under a
// fresh id, which then lands turn 1.
func TestSenderIndeterminateAssignReissuesWhenSessionless(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "c8")
	w.adapter.Fail["Start"] = gateway.ErrAgentStray
	rowID := w.commandID(t, testTenant, build, gateway.CmdAssign)

	w.snd.PollOnce(ctx)
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "failed" || !strings.Contains(row.lastError, "re-assigning") {
		t.Fatalf("poisoned assign = %s (%q), want failed/re-assigning", row.state, row.lastError)
	}
	rows := w.listCommands(t, testTenant, build)
	if len(rows) != 2 {
		t.Fatalf("command rows = %d, want 2", len(rows))
	}
	// Same-pass recovery also hit the fault: a marked assign that goes
	// ambiguous again fails terminally (recovery fires once).
	if rec := rows[1]; rec.state != "failed" || !strings.Contains(rec.lastError, "ambiguous twice") {
		t.Fatalf("recovery assign = %s (%q), want failed/ambiguous-twice", rec.state, rec.lastError)
	}
	w.adapter.Fail = map[string]error{}
	freshID := w.insertCommand(t, testTenant, build, gateway.CmdAssign, map[string]any{
		"conversation_id": "x", "title": "do the thing",
	}, "queued", 0)
	w.snd.PollOnce(ctx)
	if fresh := w.getCommand(t, testTenant, freshID); fresh.state != "acked" {
		t.Fatalf("fresh assign = %s (%q), want acked", fresh.state, fresh.lastError)
	}
	turn, _ := w.buildTracking(t, testTenant, build)
	if turn == nil || *turn != 1 {
		t.Fatalf("tracked turn = %v, want 1", turn)
	}
}

// TestSenderBusyRetriesSameID pins transient-busy handling: the row
// burns one attempt with backoff, then the same id acks when the session
// goes idle — one gateway execution total.
func TestSenderBusyRetriesSameID(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "c9")
	w.snd.PollOnce(ctx)
	w.adapter.Fail["Continue"] = gateway.ErrSessionBusy
	rowID := w.insertCommand(t, testTenant, build, gateway.CmdContinue, map[string]any{
		"question_id": "q", "kind": "k", "answer": "a", "turn": float64(2),
	}, "queued", 0)

	w.snd.PollOnce(ctx)
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "queued" || row.attempts != 1 || !strings.Contains(row.lastError, "live run") {
		t.Fatalf("busy row = %s/%d (%q), want queued/1/live-run", row.state, row.attempts, row.lastError)
	}
	if !row.nextAttempt.After(time.Now()) {
		t.Errorf("next_attempt_at = %v, want future backoff", row.nextAttempt)
	}
	w.adapter.Fail = map[string]error{}
	w.resetDue(t, testTenant, rowID)
	w.snd.PollOnce(ctx)
	row = w.getCommand(t, testTenant, rowID)
	if row.state != "acked" || row.attempts != 2 {
		t.Fatalf("row after idle = %s/%d, want acked/2", row.state, row.attempts)
	}
	if len(w.adapter.Continues) != 1 {
		t.Fatalf("gateway continues = %d, want 1", len(w.adapter.Continues))
	}
}

// TestSenderAlreadyAssignedAdopts is the crash-recovery proof: our assign
// attempt landed (recorded 200) before Studio died. After restart the row
// is still queued; redelivery replays already_assigned, and the sender
// adopts the recorded turn instead of executing twice.
func TestSenderAlreadyAssignedAdopts(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "c10")
	// The pre-crash attempt, under a different id now lost to us.
	direct := &gateway.Client{Timeout: 5 * time.Second}
	if _, err := direct.Send(ctx, w.gwURL, gateway.StaticTestKey(), build, gateway.CmdAssign,
		map[string]any{"prompt": "p", "workspace_id": testTenant}); err != nil {
		t.Fatal(err)
	}
	rowID := w.commandID(t, testTenant, build, gateway.CmdAssign)

	w.snd.PollOnce(ctx)
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "acked" || !strings.Contains(row.lastError, "adopted turn 1") {
		t.Fatalf("row = %s (%q), want acked/adopted-turn-1", row.state, row.lastError)
	}
	turn, ref := w.buildTracking(t, testTenant, build)
	if turn == nil || *turn != 1 || ref == "" {
		t.Fatalf("tracking = turn %v ref %q, want 1 + adopted ref", turn, ref)
	}
	if len(w.adapter.Starts) != 1 {
		t.Fatalf("gateway starts = %d, want 1 (no double execution)", len(w.adapter.Starts))
	}
}

// TestSenderWaitsWithoutBurning pins prerequisite waits: no gateway URL
// and no prompt hold the row with attempts untouched and a future due
// time. A later user message self-unblocks the promptless assign.
func TestSenderWaitsWithoutBurning(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	out := w.invoke(t, testTenant, tools.ToolConversationCreate, map[string]any{"title": "s"}, "s-c11")
	conv, _ := out["conversation_id"].(string)
	out = w.invoke(t, testTenant, tools.ToolBuildCreate,
		map[string]any{"conversation_id": conv}, "s-b11")
	build, _ := out["build_id"].(string)
	rowID := w.commandID(t, testTenant, build, gateway.CmdAssign)

	w.snd.PollOnce(ctx) // no gateway URL yet
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "queued" || row.attempts != 0 || !strings.Contains(row.lastError, "no gateway URL") {
		t.Fatalf("row = %s/%d (%q), want queued/0/no-URL-wait", row.state, row.attempts, row.lastError)
	}
	if !row.nextAttempt.After(time.Now()) {
		t.Errorf("next_attempt_at = %v, want future wait", row.nextAttempt)
	}
	w.setGatewayURL(t, testTenant, build, w.gwURL)
	w.resetDue(t, testTenant, rowID)
	w.snd.PollOnce(ctx) // URL set, still no prompt (no title, no user message)
	row = w.getCommand(t, testTenant, rowID)
	if row.state != "queued" || row.attempts != 0 || !strings.Contains(row.lastError, "no prompt yet") {
		t.Fatalf("row = %s/%d (%q), want queued/0/no-prompt-wait", row.state, row.attempts, row.lastError)
	}
	// A user message lands in the conversation: the same row dispatches.
	err := store.InTenantTx(ctx, w.srv.pool, testTenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO studio.messages (id, tenant_id, conversation_id, role, body)
			 VALUES ('msg-unblock', $1, $2, 'user', 'Now do it')`, testTenant, conv)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	w.resetDue(t, testTenant, rowID)
	w.snd.PollOnce(ctx)
	row = w.getCommand(t, testTenant, rowID)
	if row.state != "acked" {
		t.Fatalf("row = %s (%q), want acked after message", row.state, row.lastError)
	}
	if len(w.adapter.Starts) != 1 || w.adapter.Starts[0].Prompt != "Now do it" {
		t.Fatalf("gateway starts = %+v, want the unblocking message", w.adapter.Starts)
	}
}

// denyKeys is a gateway key provider that knows nothing: every command
// 404s as unprovisioned.
type denyKeys struct{}

func (denyKeys) KeyFor(string) ([]byte, error) { return nil, errors.New("unknown build") }

// TestSenderUnprovisionedGatewayWaits pins the 404 rule: a gateway with
// no key for the build holds the row without burning — provisioning is
// an owner step the sender waits for, never a failure.
func TestSenderUnprovisionedGatewayWaits(t *testing.T) {
	w := newSenderWorld(t)
	w.gw.Keys = denyKeys{}
	_, build := w.openBuild(t, testTenant, "c12")
	rowID := w.commandID(t, testTenant, build, gateway.CmdAssign)

	w.snd.PollOnce(context.Background())
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "queued" || row.attempts != 0 || !strings.Contains(row.lastError, "no key for build") {
		t.Fatalf("row = %s/%d (%q), want queued/0/unprovisioned-wait", row.state, row.attempts, row.lastError)
	}
}

// TestSenderExhaustsAttempts pins the bounded budget: a persistently
// failing gateway burns one attempt per due poll, then the row fails with
// the exhaustion evidence.
func TestSenderExhaustsAttempts(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "c13")
	w.snd.PollOnce(ctx)
	w.adapter.Fail["Continue"] = errors.New("boom")
	rowID := w.insertCommand(t, testTenant, build, gateway.CmdContinue, map[string]any{
		"question_id": "q", "kind": "k", "answer": "a", "turn": float64(2),
	}, "queued", 0)

	for i := 0; i < testSenderCfg().MaxAttempts; i++ {
		w.resetDue(t, testTenant, rowID)
		w.snd.PollOnce(ctx)
	}
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "failed" || !strings.Contains(row.lastError, "exhausted 5 attempts") {
		t.Fatalf("row = %s (%q), want failed/exhausted-5", row.state, row.lastError)
	}
	if row.attempts != 5 {
		t.Errorf("attempts = %d, want 5", row.attempts)
	}
}

// TestSenderCancelChain pins cancel delivery through the real tool: an
// acked assign followed by build.cancel ends with the gateway session
// cancelled and both rows acked.
func TestSenderCancelChain(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "c14")
	w.snd.PollOnce(ctx)
	w.invoke(t, testTenant, tools.ToolBuildCancel, map[string]any{"build_id": build}, "s-x14")

	w.snd.PollOnce(ctx)
	cancelID := w.commandID(t, testTenant, build, gateway.CmdCancel)
	row := w.getCommand(t, testTenant, cancelID)
	if row.state != "acked" {
		t.Fatalf("cancel row = %s (%q), want acked", row.state, row.lastError)
	}
	direct := &gateway.Client{Timeout: 5 * time.Second}
	st, err := direct.Send(ctx, w.gwURL, gateway.StaticTestKey(), build, gateway.CmdStatus, nil)
	if err != nil {
		t.Fatal(err)
	}
	if st["state"] != gateway.SessionCancelled {
		t.Fatalf("gateway state = %v, want cancelled", st)
	}
}

// TestSenderCancelUnknownBuildAcks pins idempotent cancel: when the
// gateway never saw the build, the cancel goal already holds and the row
// acks without a session ever existing.
func TestSenderCancelUnknownBuildAcks(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "c15")
	w.invoke(t, testTenant, tools.ToolBuildCancel, map[string]any{"build_id": build}, "s-x15")
	// The assign never dispatched: retire it so only the cancel goes out.
	assignID := w.commandID(t, testTenant, build, gateway.CmdAssign)
	err := store.InTenantTx(ctx, w.srv.pool, testTenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE studio.commands SET state = 'failed' WHERE id = $1`, assignID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	w.snd.PollOnce(ctx)
	cancelID := w.commandID(t, testTenant, build, gateway.CmdCancel)
	row := w.getCommand(t, testTenant, cancelID)
	if row.state != "acked" || !strings.Contains(row.lastError, "goal holds") {
		t.Fatalf("cancel row = %s (%q), want acked/goal-holds", row.state, row.lastError)
	}
	if len(w.adapter.Starts) != 0 {
		t.Fatalf("gateway starts = %d, want 0", len(w.adapter.Starts))
	}
}

// TestSenderTenantIsolation pins per-tenant polling: a sender bound to
// one tenant never touches another tenant's rows, and a tenant-B sender
// dispatches them.
func TestSenderTenantIsolation(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	out := w.invoke(t, testTenantB, tools.ToolConversationCreate, map[string]any{"title": "s"}, "s-c16")
	conv, _ := out["conversation_id"].(string)
	out = w.invoke(t, testTenantB, tools.ToolBuildCreate,
		map[string]any{"conversation_id": conv, "title": "B work"}, "s-b16")
	build, _ := out["build_id"].(string)
	w.setGatewayURL(t, testTenantB, build, w.gwURL)
	rowID := w.commandID(t, testTenantB, build, gateway.CmdAssign)

	w.snd.PollOnce(ctx) // tenant-A sender
	row := w.getCommand(t, testTenantB, rowID)
	if row.state != "queued" || row.attempts != 0 {
		t.Fatalf("B row after A poll = %s/%d, want queued/0", row.state, row.attempts)
	}
	if len(w.adapter.Starts) != 0 {
		t.Fatalf("gateway starts = %d, want 0", len(w.adapter.Starts))
	}
	cfg := w.snd.cfg
	sndB := &Sender{
		pool: w.srv.pool, keys: w.srv.gwKeys,
		tenants: []senderBinding{{tenant: testTenantB}},
		http:    &gateway.Client{Timeout: cfg.RequestTimeout, Retries: 0},
		cfg:     cfg, logger: w.srv.logger,
		done: make(chan struct{}),
	}
	sndB.PollOnce(ctx)
	row = w.getCommand(t, testTenantB, rowID)
	if row.state != "acked" {
		t.Fatalf("B row after B poll = %s (%q), want acked", row.state, row.lastError)
	}
	if got := w.adapter.Starts[0].Workspace; got != testTenantB {
		t.Errorf("workspace = %q, want tenant %q", got, testTenantB)
	}
}

// TestSenderLifecycle pins Start/Stop semantics: keys required, double
// start refused, stop idempotent.
func TestSenderLifecycle(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	srv, _ := testServer(t)
	if err := srv.StartSender(testSenderCfg()); err == nil {
		t.Fatal("StartSender without keys succeeded, want refusal")
	}
	srv.gwKeys = gateway.StaticProvider{}
	if err := srv.StartSender(testSenderCfg()); err != nil {
		t.Fatalf("StartSender: %v", err)
	}
	if err := srv.StartSender(testSenderCfg()); err == nil {
		t.Fatal("second StartSender succeeded, want refusal")
	}
	srv.StopSender()
	srv.StopSender()
}

// TestSenderSweepRecoversOrphans pins crash recovery across a restart: a
// row orphaned in sending returns to queued at StartSender and dispatches
// on the loop's first pass without burning the orphaned attempt.
func TestSenderSweepRecoversOrphans(t *testing.T) {
	ctx := context.Background()
	w := newSenderWorld(t)
	_, build := w.openBuild(t, testTenant, "c17")
	rowID := w.commandID(t, testTenant, build, gateway.CmdAssign)
	err := store.InTenantTx(context.Background(), w.srv.pool, testTenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE studio.commands SET state = 'sending' WHERE id = $1`, rowID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg := testSenderCfg()
	cfg.PollInterval = time.Hour // loop's immediate first pass does the work
	if err := w.srv.StartSender(cfg); err != nil {
		t.Fatalf("StartSender: %v", err)
	}
	defer w.srv.StopSender()
	deadline := time.Now().Add(10 * time.Second)
	for {
		row := w.getCommand(t, testTenant, rowID)
		if row.state == "acked" {
			if row.attempts != 1 {
				t.Fatalf("attempts = %d, want 1 (sweep burned nothing)", row.attempts)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("orphan still %s (%q) after 10s", row.state, row.lastError)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestSenderShutdownRequeuesWithoutBurn pins the two shutdown paths: a
// cancelled poll claims nothing, and a dispatch interrupted mid-flight
// requeues without burning.
func TestSenderShutdownRequeuesWithoutBurn(t *testing.T) {
	w := newSenderWorld(t)
	_, build := w.openBuild(t, testTenant, "c18")
	rowID := w.commandID(t, testTenant, build, gateway.CmdAssign)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	w.snd.PollOnce(cancelled)
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "queued" || row.attempts != 0 {
		t.Fatalf("row after cancelled poll = %s/%d, want queued/0", row.state, row.attempts)
	}
	st := w.snd.dispatch(cancelled, &sendPlan{
		rowID: rowID, buildID: build, kind: gateway.CmdAssign,
		url: w.gwURL, wire: map[string]any{"prompt": "p"},
	})
	if st.state != "queued" || st.burn {
		t.Fatalf("interrupted dispatch = %+v, want queued/no-burn", st)
	}
}

// TestSenderHelpers pins the pure contracts: turn parsing, recovery
// payload shaping, and backoff bounds. No database needed.
func TestSenderHelpers(t *testing.T) {
	if _, ok := frozenTurn(float64(2)); !ok {
		t.Error("frozenTurn(2) rejected")
	}
	for _, bad := range []any{nil, "2", float64(0), float64(-1), float64(2.5)} {
		if _, ok := frozenTurn(bad); ok {
			t.Errorf("frozenTurn(%v) accepted", bad)
		}
	}
	if _, ok := statusTurn(float64(0)); !ok {
		t.Error("statusTurn(0) rejected (0 is structurally valid)")
	}
	if _, ok := statusTurn(float64(1.5)); ok {
		t.Error("statusTurn(1.5) accepted")
	}
	if recoveredFrom(map[string]any{}) {
		t.Error("fresh payload reads as recovered")
	}
	if !recoveredFrom(map[string]any{recoveredFromKey: "x"}) {
		t.Error("marked payload reads as fresh")
	}
	cp := continuePayload(map[string]any{
		"question_id": "q", "kind": "k", "answer": "a", "turn": float64(2), "wire": "drop",
	}, 3, "old")
	if cp["turn"] != float64(3) || cp["answer"] != "a" || cp[recoveredFromKey] != "old" {
		t.Errorf("continuePayload = %v", cp)
	}
	if _, ok := cp["wire"]; ok {
		t.Errorf("continuePayload carried wire-only field: %v", cp)
	}
	snd := &Sender{cfg: SenderConfig{BaseWait: 2 * time.Second, MaxWait: 5 * time.Minute}}
	if got := snd.backoffFor(1); got != 2*time.Second {
		t.Errorf("backoff(1) = %v", got)
	}
	if got := snd.backoffFor(3); got != 8*time.Second {
		t.Errorf("backoff(3) = %v", got)
	}
	if got := snd.backoffFor(25); got != 5*time.Minute {
		t.Errorf("backoff(25) = %v, want capped", got)
	}
}

// TestSenderDefensiveBranches pins classification the real gateway cannot
// produce (contradictory 404s, undecodable 2xx, uncoded conflicts,
// unknown statuses) against a scripted stub, so every branch stays total.
func TestSenderDefensiveBranches(t *testing.T) {
	cases := []struct {
		name       string
		kind       string
		command    string // response for the command call
		status     string // response for the status call
		wantState  string
		wantDetail string
		wantTurn   *int
	}{
		{"desync fails",
			gateway.CmdContinue, `{"error":"unknown build"}`,
			`{"session_ref":"s","state":"running","turn":1}`,
			"failed", "desync", nil},
		{"assign raced retries",
			gateway.CmdAssign, `{"error":"unknown build"}`,
			`{"session_ref":"s","state":"running","turn":1}`,
			"queued", "raced", nil},
		{"undecodable accept adopts",
			gateway.CmdContinue, `not-json`,
			`{"session_ref":"s","state":"running","turn":2}`,
			"acked", "adopted turn 2", intPtr(2)},
		{"uncoded conflict fails",
			gateway.CmdContinue, `{"error":"weird"}`,
			``, "failed", "unclassified 409", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			w := newSenderWorld(t)
			_, build := w.openBuild(t, testTenant, "def-"+tc.name)
			stub := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				var env struct {
					Name string `json:"name"`
				}
				_ = json.NewDecoder(r.Body).Decode(&env)
				body := tc.command
				code := http.StatusOK
				if env.Name == gateway.CmdStatus {
					body = tc.status
				} else {
					switch {
					case body == "not-json":
						// 200 with garbage: accepted but unreadable.
					case strings.Contains(body, "unknown build"):
						code = http.StatusNotFound
					case strings.Contains(body, "weird"):
						code = http.StatusConflict
					}
				}
				rw.WriteHeader(code)
				_, _ = rw.Write([]byte(body))
			}))
			t.Cleanup(stub.Close)
			w.setGatewayURL(t, testTenant, build, stub.URL)
			var rowID string
			if tc.kind == gateway.CmdContinue {
				// Retire the openBuild assign; only the continue goes out.
				assignID := w.commandID(t, testTenant, build, gateway.CmdAssign)
				err := store.InTenantTx(context.Background(), w.srv.pool, testTenant, func(tx pgx.Tx) error {
					_, err := tx.Exec(ctx, `UPDATE studio.commands SET state = 'failed' WHERE id = $1`, assignID)
					return err
				})
				if err != nil {
					t.Fatal(err)
				}
				rowID = w.insertCommand(t, testTenant, build, gateway.CmdContinue, map[string]any{
					"question_id": "q", "kind": "k", "answer": "a", "turn": float64(2),
				}, "queued", 0)
			} else {
				rowID = w.commandID(t, testTenant, build, gateway.CmdAssign)
			}

			w.snd.PollOnce(context.Background())
			row := w.getCommand(t, testTenant, rowID)
			if row.state != tc.wantState || !strings.Contains(row.lastError, tc.wantDetail) {
				t.Fatalf("row = %s (%q), want %s/%q", row.state, row.lastError, tc.wantState, tc.wantDetail)
			}
			if tc.wantTurn != nil {
				turn, _ := w.buildTracking(t, testTenant, build)
				if turn == nil || *turn != *tc.wantTurn {
					t.Fatalf("tracked turn = %v, want %d", turn, *tc.wantTurn)
				}
			}
		})
	}
}

func intPtr(n int) *int { return &n }

// TestSenderRestartResumesLoop pins in-process sender restart: stop and
// start again, and fresh rows dispatch on the new loop's first pass.
func TestSenderRestartResumesLoop(t *testing.T) {
	w := newSenderWorld(t)
	cfg := testSenderCfg()
	cfg.PollInterval = time.Hour
	if err := w.srv.StartSender(cfg); err != nil {
		t.Fatalf("start 1: %v", err)
	}
	w.srv.StopSender()
	_, build := w.openBuild(t, testTenant, "rs")
	if err := w.srv.StartSender(cfg); err != nil {
		t.Fatalf("start 2: %v", err)
	}
	defer w.srv.StopSender()
	rowID := w.commandID(t, testTenant, build, gateway.CmdAssign)
	deadline := time.Now().Add(10 * time.Second)
	for {
		if row := w.getCommand(t, testTenant, rowID); row.state == "acked" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("row never dispatched on the restarted loop")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestSenderQuestionAnswerLoop is the U2 chain: a full Q&A round through
// every layer — real tools, the sender, the real gateway over HTTP, and
// signed callbacks. It pins the handoff order: assign-ack flips the build
// running (so the later started absorbs instead of moving), the yield
// parks, the answer freezes turn 2, the continue lands exactly turn 2,
// and completion closes the build.
func TestSenderQuestionAnswerLoop(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "loop")
	if got := w.buildStatus(t, testTenant, build); got != "assigned" {
		t.Fatalf("build = %s, want assigned", got)
	}

	w.snd.PollOnce(ctx) // assign dispatched + acked
	if got := w.buildStatus(t, testTenant, build); got != "running" {
		t.Fatalf("build after assign-ack = %s, want running (sender handoff)", got)
	}
	if turn, _ := w.buildTracking(t, testTenant, build); turn == nil || *turn != 1 {
		t.Fatalf("tracked turn = %v, want 1", turn)
	}
	// The started callback arrives after the flip: absorbed, not moved.
	// Callbacks route by the workspace the gateway recorded from our
	// assign — not a hardcoded tenant — pinning sender→gateway→callback
	// workspace continuity end to end.
	ws := w.adapter.Starts[0].Workspace
	code, ack := postCallback(t, w.srv, signedCallback(t, build, ws,
		"loop-started", "agent.started", map[string]any{"agent_session_id": "sess-loop"}))
	if code != http.StatusOK || ack["transitioned"] != false {
		t.Fatalf("started = %d %v, want 200/absorbed", code, ack)
	}
	code, ack = postCallback(t, w.srv, signedCallback(t, build, ws,
		"loop-q", "agent.question", map[string]any{"question_id": "loop-1", "body": "Which region?"}))
	if code != http.StatusOK || ack["transitioned"] != true || ack["build_status"] != "waiting_user" {
		t.Fatalf("question = %d %v, want parked/waiting_user", code, ack)
	}
	out := w.invoke(t, testTenant, tools.ToolQuestionAnswer,
		map[string]any{"question_id": "loop-1", "answer": "eu-west"}, "loop-a")
	if out["build_status"] != "running" {
		t.Fatalf("answer = %v, want running", out)
	}
	contID := w.commandID(t, testTenant, build, gateway.CmdContinue)
	if frozen := w.getCommand(t, testTenant, contID).payload["turn"]; frozen != float64(2) {
		t.Fatalf("frozen turn = %v, want 2", frozen)
	}

	w.snd.PollOnce(ctx) // continue dispatched + acked
	if row := w.getCommand(t, testTenant, contID); row.state != "acked" {
		t.Fatalf("continue row = %s (%q), want acked", row.state, row.lastError)
	}
	if len(w.adapter.Continues) != 1 {
		t.Fatalf("gateway continues = %d, want 1", len(w.adapter.Continues))
	}
	in := w.adapter.Continues[0].Input
	if in.Turn != 2 || in.Answer != "eu-west" || in.QuestionID != "loop-1" {
		t.Fatalf("gateway saw %+v, want turn 2 with the frozen triple", in)
	}
	if turn, _ := w.buildTracking(t, testTenant, build); turn == nil || *turn != 2 {
		t.Fatalf("tracked turn = %v, want 2", turn)
	}
	code, ack = postCallback(t, w.srv, signedCallback(t, build, ws,
		"loop-done", "agent.completed", map[string]any{}))
	if code != http.StatusOK || ack["build_status"] != "succeeded" {
		t.Fatalf("completed = %d %v, want succeeded", code, ack)
	}
	if got := w.buildStatus(t, testTenant, build); got != "succeeded" {
		t.Fatalf("build = %s, want succeeded", got)
	}
}
