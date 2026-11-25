package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// failItem drives one item to terminally failed through the real
// deliverer against a refusing stub.
func failItem(t *testing.T, srv *Server, studioURL, build, id string) {
	t.Helper()
	env, err := Sign(StaticTestKey(), build, "w-f8", id, "agent.message", map[string]any{"text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Outbox.Enqueue(OutboxItem{ID: id, BuildID: build, MessageID: id, Envelope: env}); err != nil {
		t.Fatal(err)
	}
	d := &Deliverer{Outbox: srv.Outbox, Studio: studioURL, Keys: StaticProvider{}, Tick: time.Hour}
	d.pass(context.Background())
	if item, ok := srv.Outbox.Find(id); !ok || item.State != OutboxFailed {
		t.Fatalf("item = %+v, want failed", item)
	}
}

// TestOutboxInspectRequeue (F8): failed items are inspectable per build
// and requeue into a draining delivery — with cross-build ids and live
// items refused.
func TestOutboxInspectRequeue(t *testing.T) {
	dir := t.TempDir()
	srv := f4Server(t, dir, NewFakeAdapter())
	f4Record(t, srv, "b-f8a")
	f4Record(t, srv, "b-f8b")
	// Refuse terminally with a 400 stub (connection errors would
	// merely retry, never park the item as failed).
	refuse := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer refuse.Close()
	failItem(t, srv, refuse.URL, "b-f8a", "doomed-1")
	// Inspect shows the build's own items only.
	code, out := postCommand(t, srv, signedCommand(t, "b-f8a", "insp-1", CmdOutboxInspect, nil))
	if code != http.StatusOK {
		t.Fatalf("inspect = %d %v", code, out)
	}
	items, _ := out["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %v, want the one failed item", out["items"])
	}
	first, _ := items[0].(map[string]any)
	if first["id"] != "doomed-1" || first["state"] != OutboxFailed || first["name"] != "agent.message" {
		t.Errorf("item = %v", first)
	}
	code, out = postCommand(t, srv, signedCommand(t, "b-f8b", "insp-2", CmdOutboxInspect, nil))
	if code != http.StatusOK {
		t.Fatalf("inspect b = %d", code)
	}
	if items, _ := out["items"].([]any); len(items) != 0 {
		t.Errorf("cross-build items leaked: %v", items)
	}
	// Cross-build requeue 404s; live-item requeue 409s.
	code, _ = postCommand(t, srv, signedCommand(t, "b-f8b", "req-x", CmdOutboxRequeue,
		map[string]any{"item_id": "doomed-1"}))
	if code != http.StatusNotFound {
		t.Errorf("cross-build requeue = %d, want 404", code)
	}
	// Requeue drains: point a deliverer at an acking stub afterwards.
	url, acks := verifyStub(t, StaticProvider{})
	code, out = postCommand(t, srv, signedCommand(t, "b-f8a", "req-1", CmdOutboxRequeue,
		map[string]any{"item_id": "doomed-1"}))
	if code != http.StatusOK || out["requeued"] != "doomed-1" {
		t.Fatalf("requeue = %d %v", code, out)
	}
	if out["requeues"] != float64(1) {
		t.Errorf("requeues = %v, want 1 (operator action recorded)", out["requeues"])
	}
	d := &Deliverer{Outbox: srv.Outbox, Studio: url, Keys: StaticProvider{},
		Tick: time.Hour, BaseWait: time.Millisecond, MaxWait: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go d.Run(ctx)
	deadline := time.Now().Add(5 * time.Second)
	for {
		q, f := srv.Outbox.Depth()
		if (q == 0 && f == 0) || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if *acks != 1 {
		t.Errorf("acks = %d, want requeued delivery acked", *acks)
	}
	if q, f := srv.Outbox.Depth(); q+f != 0 {
		t.Errorf("depth = %d/%d, want drained", q, f)
	}
}
