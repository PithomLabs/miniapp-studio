package broker

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestHTTPExchangerWireShape asserts the actual provider wire
// protocol against a scripted endpoint: grant fields posted, token
// parsed, invalid_grant typed. The double asserts behavior, never
// mirrors broker internals.
func TestHTTPExchangerWireShape(t *testing.T) {
	var gotGrant, gotRefresh, gotID, gotSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		gotGrant, gotRefresh = r.Form.Get("grant_type"), r.Form.Get("refresh_token")
		gotID, gotSecret = r.Form.Get("client_id"), r.Form.Get("client_secret")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"tok-abc","expires_in":300,"token_type":"Bearer"}`))
	}))
	defer srv.Close()
	ex := &HTTPExchanger{Endpoint: srv.URL, ClientID: "cid", ClientSecret: "csec"}
	access, ttl, err := ex.Exchange(context.Background(), "prov", "refresh-123")
	if err != nil {
		t.Fatal(err)
	}
	if access != "tok-abc" || ttl != 5*time.Minute {
		t.Fatalf("parsed = (%q, %s)", access, ttl)
	}
	if gotGrant != "refresh_token" || gotRefresh != "refresh-123" || gotID != "cid" || gotSecret != "csec" {
		t.Fatalf("wire fields = %q %q %q %q", gotGrant, gotRefresh, gotID, gotSecret)
	}
}

func TestHTTPExchangerMapsInvalidGrant(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant","error_description":"revoked"}`))
	}))
	defer srv.Close()
	ex := &HTTPExchanger{Endpoint: srv.URL}
	if _, _, err := ex.Exchange(context.Background(), "prov", "dead"); !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("err = %v, want ErrInvalidGrant", err)
	}
}

// TestHTTPExchangerRefusesRedirects proves credentials never leave
// for a redirected endpoint: every 3xx shape fails closed and the
// catcher receives nothing credential-bearing.
func TestHTTPExchangerRefusesRedirects(t *testing.T) {
	var mu sync.Mutex
	var hits int
	var bodies strings.Builder
	catcher := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		mu.Lock()
		hits++
		bodies.Write(body)
		mu.Unlock()
		w.Write([]byte(`{"access_token":"x","expires_in":1}`))
	}))
	defer catcher.Close()
	for _, code := range []int{301, 302, 303, 307, 308} {
		redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", catcher.URL)
			w.WriteHeader(code)
		}))
		ex := &HTTPExchanger{Endpoint: redir.URL, ClientID: "cid", ClientSecret: "SECRET-CANARY"}
		_, _, err := ex.Exchange(context.Background(), "prov", "REFRESH-CANARY")
		redir.Close()
		if err == nil || !strings.Contains(err.Error(), "redirected (refused") {
			t.Fatalf("status %d err = %v, want redirect refusal", code, err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if hits != 0 {
		t.Fatalf("catcher hit %d times", hits)
	}
	if s := bodies.String(); strings.Contains(s, "CANARY") {
		t.Fatalf("catcher received credential bytes: %q", s)
	}
}

// TestHTTPExchangerCallerClientKeepsRefusal proves a caller-supplied
// client (even one with a permissive CheckRedirect) cannot re-enable
// following, and the caller's client is left unmutated.
func TestHTTPExchangerCallerClientKeepsRefusal(t *testing.T) {
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "http://example.invalid/")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer redir.Close()
	caller := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return nil }}
	ex := &HTTPExchanger{Endpoint: redir.URL, Client: caller}
	if _, _, err := ex.Exchange(context.Background(), "prov", "r"); err == nil || !strings.Contains(err.Error(), "redirected (refused") {
		t.Fatalf("err = %v, want redirect refusal", err)
	}
	if caller.CheckRedirect == nil {
		t.Fatal("caller client mutated")
	}
	// Permissive check still present and permissive.
	if err := caller.CheckRedirect(nil, nil); err != nil {
		t.Fatalf("caller CheckRedirect mutated: %v", err)
	}
}

// TestHTTPExchangerBucketsUnknownErrorCodes (F35-13): allowlisted
// OAuth codes repeat verbatim; provider-controlled novelties stay
// out of errors.
func TestHTTPExchangerBucketsUnknownErrorCodes(t *testing.T) {
	mk := func(code string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"` + code + `"}`))
		}))
	}
	known := mk("invalid_client")
	defer known.Close()
	ex := &HTTPExchanger{Endpoint: known.URL}
	if _, _, err := ex.Exchange(context.Background(), "prov", "r"); err == nil || !strings.Contains(err.Error(), "invalid_client") {
		t.Fatalf("known code err = %v", err)
	}
	marker := "marker-exfil-probe-zz9"
	rogue := mk(marker)
	defer rogue.Close()
	ex2 := &HTTPExchanger{Endpoint: rogue.URL}
	_, _, err := ex2.Exchange(context.Background(), "prov", "r")
	if err == nil || strings.Contains(err.Error(), marker) {
		t.Fatalf("unknown code echoed: %v", err)
	}
	if !strings.Contains(err.Error(), "unrecognized code") {
		t.Fatalf("unknown code not bucketed: %v", err)
	}
}

func TestHTTPExchangerRefusals(t *testing.T) {
	ex := &HTTPExchanger{}
	if _, _, err := ex.Exchange(context.Background(), "prov", "r"); err == nil {
		t.Fatal("empty endpoint accepted")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer srv.Close()
	ex2 := &HTTPExchanger{Endpoint: srv.URL}
	if _, _, err := ex2.Exchange(context.Background(), "prov", "r"); err == nil {
		t.Fatal("malformed response accepted")
	}
}
