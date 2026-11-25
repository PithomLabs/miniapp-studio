package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// roundTripCase drives the real client against the real command server over
// HTTP: the pair must interoperate with no fakes between them.
func TestClientServerRoundTrip(t *testing.T) {
	srv, adapter := testServer(t)
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()
	client := &Client{Timeout: 5 * time.Second, Retries: 0}
	ctx := context.Background()
	out, err := client.Send(ctx, httpSrv.URL, StaticTestKey(), "b-rt", CmdAssign,
		map[string]any{"prompt": "hello", "workspace_id": "w-rt"})
	if err != nil {
		t.Fatalf("assign: %v", err)
	}
	if out["session_ref"] == nil {
		t.Fatalf("response = %v", out)
	}
	if len(adapter.Starts) != 1 {
		t.Fatalf("starts = %d", len(adapter.Starts))
	}
	out, err = client.Send(ctx, httpSrv.URL, StaticTestKey(), "b-rt", CmdStatus, nil)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if out["state"] != SessionRunning {
		t.Errorf("status = %v", out)
	}
	// Terminal client errors surface without retries.
	if _, err := client.Send(ctx, httpSrv.URL, StaticTestKey(), "nope", CmdStatus, nil); err == nil {
		t.Error("unknown build accepted, want refusal")
	} else if got := err.Error(); !strings.Contains(got, "404") {
		t.Errorf("error = %q, want 404 surfaced", got)
	}
}

func TestClientRetriesThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	client := &Client{Timeout: 5 * time.Second, Retries: 5,
		BaseWait: time.Millisecond, MaxWait: 5 * time.Millisecond}
	out, err := client.Send(context.Background(), srv.URL, StaticTestKey(),
		"b-1", CmdStatus, nil)
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if out["ok"] != true || calls.Load() != 3 {
		t.Errorf("out = %v after %d calls", out, calls.Load())
	}
}

func TestClientNoRetryOnTerminal(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	}))
	defer srv.Close()
	client := &Client{Timeout: 5 * time.Second, Retries: 5,
		BaseWait: time.Millisecond, MaxWait: 5 * time.Millisecond}
	if _, err := client.Send(context.Background(), srv.URL, StaticTestKey(),
		"b-1", CmdStatus, nil); err == nil {
		t.Error("400 accepted, want terminal refusal")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("calls = %d, want exactly 1 (no retry on 4xx)", got)
	}
}

func TestClientSameBytesOnRetry(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]any
		_ = json.NewDecoder(r.Body).Decode(&m)
		mid, _ := m["message_id"].(string)
		bodies = append(bodies, mid)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	client := &Client{Timeout: 5 * time.Second, Retries: 2,
		BaseWait: time.Millisecond, MaxWait: 2 * time.Millisecond}
	_, _ = client.Send(context.Background(), srv.URL, StaticTestKey(),
		"b-1", CmdStatus, nil)
	if len(bodies) != 3 {
		t.Fatalf("attempts = %d", len(bodies))
	}
	// Same message id across retries: gateway dedup collapses redelivery.
	if bodies[0] == "" || bodies[0] != bodies[1] || bodies[1] != bodies[2] {
		t.Errorf("message ids drift across retries: %v", bodies)
	}
}

// TestClientRefusesPlaintextRemote (M12): prompts (possibly secret
// bearing) must never cross plaintext HTTP to a non-loopback host.
func TestClientRefusesPlaintextRemote(t *testing.T) {
	client := &Client{Timeout: 5 * time.Second, Retries: 0}
	_, err := client.Send(context.Background(), "http://gateway.example.test",
		StaticTestKey(), "b-m12", CmdStatus, nil)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "https") {
		t.Errorf("err = %v, want https refusal (no request sent)", err)
	}
}

// TestClientAllowsLoopbackHTTP (M12): the loopback exception is pinned —
// tests and same-host wiring may use http, nothing else may.
func TestClientAllowsLoopbackHTTP(t *testing.T) {
	srv, _ := testServer(t)
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()
	client := &Client{Timeout: 5 * time.Second, Retries: 0}
	if _, err := client.Send(context.Background(), httpSrv.URL,
		StaticTestKey(), "b-m12l", CmdStatus, nil); err == nil {
		t.Fatal("expected 404-via-transport (unknown build), got success")
	} else if strings.Contains(strings.ToLower(err.Error()), "https") {
		t.Fatalf("loopback refused as insecure: %v", err)
	}
}
