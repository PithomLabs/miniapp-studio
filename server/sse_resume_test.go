package server

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/gateway"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/tools"
)

// parseSSEIDs extracts the id: lines of an SSE body in arrival order.
func parseSSEIDs(t *testing.T, body string) []int64 {
	t.Helper()
	var ids []int64
	for _, line := range strings.Split(body, "\n") {
		n, ok := strings.CutPrefix(line, "id: ")
		if !ok {
			continue
		}
		seq, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err != nil {
			t.Fatalf("bad SSE id %q", line)
		}
		ids = append(ids, seq)
	}
	return ids
}

func assertIncreasingUnique(t *testing.T, ids []int64) {
	t.Helper()
	seen := map[int64]bool{}
	for i, id := range ids {
		if seen[id] {
			t.Fatalf("duplicate SSE id %d in %v", id, ids)
		}
		seen[id] = true
		if i > 0 && id <= ids[i-1] {
			t.Fatalf("SSE ids not increasing: %v", ids)
		}
	}
}

// TestSSEKillAndResume pins C4 restart recovery across a real process
// boundary: a stream is killed with the server, a fresh server boots over
// the same database, and a reconnect from the last cursor replays exactly
// the missed rows — no gaps, no duplicates.
func TestSSEKillAndResume(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset")
	}
	srv := callbackServer(t)
	srv.ssePoll = 10 * time.Millisecond
	conv, _ := setupBuild(t, srv, testTenant)
	seedMessages(t, srv, conv, "one", "two")

	ctx, cancel := context.WithCancel(context.Background())
	rec, req := streamRequest(t, srv, conv, "", ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.Handler().ServeHTTP(rec, req)
	}()
	time.Sleep(100 * time.Millisecond) // let the burst flush
	if err := srv.Close(); err != nil {
		t.Fatalf("kill: %v", err)
	}
	cancel()
	<-done
	before := parseSSEIDs(t, rec.Body.String())
	if len(before) == 0 {
		t.Fatalf("pre-kill stream empty:\n%s", rec.Body.String())
	}
	last := before[len(before)-1]

	// Fresh server, same database, no truncate: the restart.
	srv2, _ := bootServerOn(t, dsn, nil, nil, false, nil)
	srv2.ssePoll = 10 * time.Millisecond
	srv2.gwKeys = gateway.StaticProvider{}
	seedMessages(t, srv2, conv, "three")

	ctx2, cancel2 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel2()
	rec2, req2 := streamRequest(t, srv2, conv, strconv.FormatInt(last, 10), ctx2)
	srv2.Handler().ServeHTTP(rec2, req2)
	after := parseSSEIDs(t, rec2.Body.String())
	if len(after) == 0 {
		t.Fatalf("resumed stream empty:\n%s", rec2.Body.String())
	}
	assertIncreasingUnique(t, after)
	for _, id := range after {
		if id <= last {
			t.Fatalf("resumed id %d <= cursor %d: duplicate replay", id, last)
		}
	}
	if body := rec2.Body.String(); !strings.Contains(body, `"body":"three"`) {
		t.Errorf("resumed stream lacks the post-restart row:\n%s", body)
	}
	if body := rec2.Body.String(); strings.Contains(body, `"body":"two"`) {
		t.Errorf("resumed stream replays pre-cursor rows:\n%s", body)
	}
}

// TestSSEBacklogDrainsAcrossPages pins multi-page replay: with more rows
// than one poll page, the stream drains every row exactly once across
// successive polls.
func TestSSEBacklogDrainsAcrossPages(t *testing.T) {
	srv := callbackServer(t)
	srv.ssePoll = 10 * time.Millisecond
	conv, _ := setupBuild(t, srv, testTenant)
	bodies := make([]string, 0, 201)
	for i := 0; i < 201; i++ {
		bodies = append(bodies, "bulk-"+strconv.Itoa(i))
	}
	seedMessages(t, srv, conv, bodies...)
	// 1 system note + 201 messages = 202 rows > 200-row pages.

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rec, req := streamRequest(t, srv, conv, "", ctx)
	srv.Handler().ServeHTTP(rec, req)
	ids := parseSSEIDs(t, rec.Body.String())
	if len(ids) != 202 {
		t.Fatalf("streamed ids = %d, want 202", len(ids))
	}
	assertIncreasingUnique(t, ids)
	if !strings.Contains(rec.Body.String(), `"body":"bulk-200"`) {
		t.Errorf("stream lacks the last row")
	}
}

// TestSSEConcurrentTrafficNoGapsNoDupes hammers a live stream with
// concurrent message sends, agent callbacks, and sender dispatches: every
// persisted stream row arrives exactly once, in sequence order.
func TestSSEConcurrentTrafficNoGapsNoDupes(t *testing.T) {
	w := newSenderWorld(t)
	w.srv.ssePoll = 10 * time.Millisecond
	ctx := context.Background()
	out := w.invoke(t, testTenant, tools.ToolConversationCreate, map[string]any{"title": "s"}, "sx-c")
	conv, _ := out["conversation_id"].(string)
	out = w.invoke(t, testTenant, tools.ToolBuildCreate,
		map[string]any{"conversation_id": conv, "title": "sx"}, "sx-b")
	build, _ := out["build_id"].(string)
	w.setGatewayURL(t, testTenant, build, w.gwURL)

	streamCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rec, req := streamRequest(t, w.srv, conv, "", streamCtx)
	streamDone := make(chan struct{})
	go func() {
		defer close(streamDone)
		w.srv.Handler().ServeHTTP(rec, req)
	}()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			seedMessages(t, w.srv, conv, "race-m"+strconv.Itoa(i))
		}(i)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			code, _ := postCallback(t, w.srv, signedCallback(t, build, testTenant,
				"sx-p"+strconv.Itoa(i), "agent.progress",
				map[string]any{"text": "race-p" + strconv.Itoa(i)}))
			if code != 200 {
				t.Errorf("callback %d = %d", i, code)
			}
		}(i)
	}
	// Sender dispatches in the middle of it all: lock-interference probe
	// (acks write no stream rows, so the stream must stay exact).
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.snd.PollOnce(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.snd.PollOnce(ctx)
	}()
	wg.Wait()
	<-streamDone

	ids := parseSSEIDs(t, rec.Body.String())
	assertIncreasingUnique(t, ids)
	var want []int64
	err := store.InTenantTx(ctx, w.srv.pool, testTenant, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT seq FROM studio.conversation_stream
			 WHERE tenant_id = $1 AND conversation_id = $2 ORDER BY seq`,
			testTenant, conv)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var seq int64
			if err := rows.Scan(&seq); err != nil {
				return err
			}
			want = append(want, seq)
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != len(want) {
		t.Fatalf("streamed %d ids, database holds %d rows", len(ids), len(want))
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("streamed %v, database %v", ids, want)
		}
	}
	// The interfered dispatch still converged.
	w.snd.PollOnce(ctx)
	assignID := w.commandID(t, testTenant, build, gateway.CmdAssign)
	if row := w.getCommand(t, testTenant, assignID); row.state != "acked" {
		t.Fatalf("assign = %s (%q), want acked", row.state, row.lastError)
	}
}
