package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestOutboxRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.json")
	o, err := OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := o.Enqueue(OutboxItem{ID: "m1", BuildID: "b", MessageID: "m1", Envelope: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if err := o.Enqueue(OutboxItem{ID: "m1", Envelope: []byte(`{}`)}); err == nil {
		t.Error("duplicate enqueue accepted, want refusal")
	}
	if err := o.Enqueue(OutboxItem{ID: "m2"}); err == nil {
		t.Error("envelope-less enqueue accepted, want refusal")
	}
	due := o.Due(time.Now())
	if len(due) != 1 || due[0].ID != "m1" {
		t.Fatalf("due = %v", due)
	}
	// Reopen: the item survives (restart recovery).
	o2, err := OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	if due := o2.Due(time.Now()); len(due) != 1 {
		t.Fatalf("reopened due = %d", len(due))
	}
	if err := o2.MarkSending("m1"); err != nil {
		t.Fatal(err)
	}
	// Reopen mid-send: sending resets to queued, never stuck, never lost.
	o3, err := OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	if due := o3.Due(time.Now()); len(due) != 1 {
		t.Fatalf("due after reopen = %d", len(due))
	}
	if err := o3.MarkAcked("m1"); err != nil {
		t.Fatal(err)
	}
	o4, err := OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	if due := o4.Due(time.Now().Add(time.Hour)); len(due) != 0 {
		t.Errorf("acked item redelivered")
	}
	if err := o4.MarkAcked("m1"); err == nil {
		t.Error("double ack accepted, want refusal")
	}
	if q, f := o4.Depth(); q+f != 0 {
		t.Errorf("depth = %d/%d", q, f)
	}
}

func TestOutboxRetryAndFail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.json")
	o, err := OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = o.Enqueue(OutboxItem{ID: "r1", BuildID: "b", Envelope: []byte(`{}`)})
	future := time.Now().Add(time.Hour)
	if err := o.MarkRetry("r1", future, "boom"); err != nil {
		t.Fatal(err)
	}
	if due := o.Due(time.Now()); len(due) != 0 {
		t.Errorf("premature redelivery")
	}
	if due := o.Due(future.Add(time.Second)); len(due) != 1 {
		t.Fatalf("due after backoff = %d", len(due))
	}
	if err := o.MarkFailed("r1", "given up"); err != nil {
		t.Fatal(err)
	}
	if due := o.Due(future.Add(100 * time.Hour)); len(due) != 0 {
		t.Errorf("failed item redelivered")
	}
	if q, f := o.Depth(); q != 0 || f != 1 {
		t.Errorf("depth = %d/%d, want 0/1", q, f)
	}
	// Failed records persist across reopen for inspection.
	o2, err := OpenOutbox(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, f := o2.Depth(); f != 1 {
		t.Errorf("reopened failed depth = %d", f)
	}
}

func TestDelivererFlow(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if r.URL.Path != "/gw/v1/callback" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != MediaTypeV1 {
			t.Errorf("content-type = %s", ct)
		}
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	o, err := OpenOutbox(filepath.Join(t.TempDir(), "outbox.json"))
	if err != nil {
		t.Fatal(err)
	}
	env, err := Sign(StaticTestKey(), "b", "w", "d1", "agent.message", map[string]any{"text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	_ = o.Enqueue(OutboxItem{ID: "d1", BuildID: "b", MessageID: "d1", Envelope: env})
	d := &Deliverer{Outbox: o, Studio: srv.URL, Keys: StaticProvider{}, Tick: 5 * time.Millisecond,
		BaseWait: time.Millisecond, MaxWait: 5 * time.Millisecond, Rand: func(int64) int64 { return 0 }}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	d.Run(ctx)
	if got := calls.Load(); got != 3 {
		t.Fatalf("deliveries = %d, want 3 (fail, fail, ack)", got)
	}
	if q, f := o.Depth(); q+f != 0 {
		t.Errorf("depth = %d/%d after ack", q, f)
	}
}

func TestDelivererTerminalStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	o, err := OpenOutbox(filepath.Join(t.TempDir(), "outbox.json"))
	if err != nil {
		t.Fatal(err)
	}
	env, err := Sign(StaticTestKey(), "b", "w", "d2", "agent.message", map[string]any{"text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	_ = o.Enqueue(OutboxItem{ID: "d2", BuildID: "b", MessageID: "d2", Envelope: env})
	d := &Deliverer{Outbox: o, Studio: srv.URL, Keys: StaticProvider{}, Tick: time.Hour}
	d.pass(context.Background())
	if q, f := o.Depth(); q != 0 || f != 1 {
		t.Errorf("depth = %d/%d, want 400 parked as failed", q, f)
	}
}

func TestBackoffMath(t *testing.T) {
	d := &Deliverer{BaseWait: time.Second, MaxWait: 5 * time.Second, Rand: func(int64) int64 { return 0 }}
	for attempt, want := range map[int]time.Duration{1: time.Second, 2: 2 * time.Second, 3: 4 * time.Second, 4: 5 * time.Second, 10: 5 * time.Second} {
		if got := d.backoff(attempt); got != want {
			t.Errorf("backoff(%d) = %v, want %v", attempt, got, want)
		}
	}
}
