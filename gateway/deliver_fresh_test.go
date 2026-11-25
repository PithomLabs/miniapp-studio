package gateway

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// verifyStub is a Studio-callback stand-in enforcing the real contract:
// strict parse + per-build HMAC + five-minute freshness, 200 ack or 401.
func verifyStub(t *testing.T, keys KeyProvider) (url string, acks *int) {
	t.Helper()
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		env, err := Parse(raw)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		key, err := keys.KeyFor(env.BuildID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := Verify(key, env); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		n++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &n
}

// signAged crafts a validly-signed envelope stamped backdated ago
// (simulating an outbox item delayed past the skew window).
func signAged(t *testing.T, key []byte, backdated time.Duration) []byte {
	t.Helper()
	// Build through the public signer, then re-stamp: the signer is the
	// only constructor, so backdating re-signs explicitly (test-only
	// dating control).
	raw, err := Sign(key, "b-f9", "w-f9", "aged-1", "agent.message", map[string]any{"text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	env.SentAt = time.Now().Add(-backdated).UTC().Format(time.RFC3339)
	body, err := canonicalBytes(env)
	if err != nil {
		t.Fatal(err)
	}
	sig := hmacSum(key, body)
	env.Signature = hex.EncodeToString(sig)
	out, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestDelivererResignsAgedEnvelope (F9): a callback delayed past the
// freshness window still delivers — each attempt re-signs with a fresh
// timestamp under the stable message id.
func TestDelivererResignsAgedEnvelope(t *testing.T) {
	key := StaticTestKey()
	url, acks, ids := verifyStubIDs(t, StaticProvider{})
	outbox, err := OpenOutbox(t.TempDir() + "/outbox.json")
	if err != nil {
		t.Fatal(err)
	}
	stale := signAged(t, key, 10*time.Minute)
	if err := outbox.Enqueue(OutboxItem{ID: "aged-1", BuildID: "b-f9", MessageID: "aged-1", Envelope: stale}); err != nil {
		t.Fatal(err)
	}
	// The stored bytes stay stale (immutable record); only wire attempts
	// refresh.
	if item, ok := outbox.Find("aged-1"); !ok {
		t.Fatal("enqueued item missing")
	} else if env, err := Parse(item.Envelope); err != nil {
		t.Fatal(err)
	} else if Verify(key, env) == nil {
		t.Fatal("stored envelope verifies fresh; want stale-at-rest")
	}
	d := &Deliverer{Outbox: outbox, Studio: url, Keys: StaticProvider{},
		Tick: time.Hour, BaseWait: time.Millisecond, MaxWait: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go d.Run(ctx)
	deadline := time.Now().Add(5 * time.Second)
	for {
		q, f := outbox.Depth()
		if (q == 0 && f == 0) || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if *acks != 1 {
		t.Errorf("acks = %d, want 1 (fresh re-sign must deliver)", *acks)
	}
	if q, f := outbox.Depth(); q+f != 0 {
		t.Errorf("depth = %d/%d, want drained", q, f)
	}
	if len(*ids) != 1 || (*ids)[0] != "aged-1" {
		t.Errorf("delivered ids = %v, want stable [aged-1]", *ids)
	}
}

// verifyStubIDs is verifyStub that also captures delivered message ids.
func verifyStubIDs(t *testing.T, keys KeyProvider) (string, *int, *[]string) {
	t.Helper()
	var n int
	var ids []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		env, err := Parse(raw)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		key, err := keys.KeyFor(env.BuildID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := Verify(key, env); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		n++
		ids = append(ids, env.MessageID)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &n, &ids
}

// TestResignPreservesIdentity (F9): re-signing refreshes only the
// timestamp and signature — id, build, name, and payload are untouched,
// and the result verifies.
func TestResignPreservesIdentity(t *testing.T) {
	key := StaticTestKey()
	stale := signAged(t, key, 10*time.Minute)
	var env Envelope
	if err := json.Unmarshal(stale, &env); err != nil {
		t.Fatal(err)
	}
	fresh, err := Resign(key, env, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var out Envelope
	if err := json.Unmarshal(fresh, &out); err != nil {
		t.Fatal(err)
	}
	if out.MessageID != "aged-1" || out.BuildID != "b-f9" || out.Name != "agent.message" {
		t.Errorf("identity changed: %+v", out)
	}
	if out.Payload["text"] != "hi" {
		t.Errorf("payload changed: %v", out.Payload)
	}
	if err := Verify(key, out); err != nil {
		t.Errorf("re-signed envelope rejected: %v", err)
	}
}

// TestDelivererAttemptTimeout (M2): a hung Studio stalls at most one
// attempt — the pass returns and the item retries on schedule.
func TestDelivererAttemptTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // hang until the test ends
	}))
	defer srv.Close()
	defer close(release) // LIFO: frees the handler before Close waits
	outbox, err := OpenOutbox(t.TempDir() + "/outbox.json")
	if err != nil {
		t.Fatal(err)
	}
	env, err := Sign(StaticTestKey(), "b", "w", "d-hang", "agent.message", map[string]any{"text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if err := outbox.Enqueue(OutboxItem{ID: "d-hang", BuildID: "b", MessageID: "d-hang", Envelope: env}); err != nil {
		t.Fatal(err)
	}
	d := &Deliverer{Outbox: outbox, Studio: srv.URL, Keys: StaticProvider{},
		Tick: time.Hour, AttemptTimeout: 50 * time.Millisecond}
	done := make(chan struct{})
	go func() { d.pass(context.Background()); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("pass wedged on hung Studio")
	}
	if q, f := outbox.Depth(); q != 1 || f != 0 {
		t.Errorf("depth = %d/%d, want requeued 1/0", q, f)
	}
}

// TestDelivererInsecureTransport (M12): a plaintext remote Studio URL
// fails the item terminally without sending anything.
func TestDelivererInsecureTransport(t *testing.T) {
	outbox, err := OpenOutbox(t.TempDir() + "/outbox.json")
	if err != nil {
		t.Fatal(err)
	}
	env, err := Sign(StaticTestKey(), "b", "w", "d-plain", "agent.message", map[string]any{"text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if err := outbox.Enqueue(OutboxItem{ID: "d-plain", BuildID: "b", MessageID: "d-plain", Envelope: env}); err != nil {
		t.Fatal(err)
	}
	d := &Deliverer{Outbox: outbox, Studio: "http://studio.example.test", Keys: StaticProvider{}, Tick: time.Hour}
	d.pass(context.Background())
	item, ok := outbox.Find("d-plain")
	if !ok || item.State != OutboxFailed {
		t.Fatalf("item = %+v, want terminally failed", item)
	}
	if !strings.Contains(item.LastError, "insecure") {
		t.Errorf("reason = %q, want insecure-transport diagnosis", item.LastError)
	}
}

// TestStaleForgedRejected pins the other half of F9: the original stale
// bytes, posted verbatim, stay 401 — re-signing must never weaken the
// receiver's freshness check.
func TestStaleForgedRejected(t *testing.T) {
	key := StaticTestKey()
	url, acks := verifyStub(t, StaticProvider{})
	stale := signAged(t, key, 10*time.Minute)
	resp, err := http.Post(url+"/gw/v1/callback", MediaTypeV1, strings.NewReader(string(stale)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("stale verbatim post = %d, want 401", resp.StatusCode)
	}
	if *acks != 0 {
		t.Errorf("acks = %d, want 0", *acks)
	}
}
