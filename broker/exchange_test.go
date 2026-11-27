package broker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
