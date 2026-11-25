package fly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// contractServer runs one scripted exchange: it asserts the request contract
// and serves the canned response. Every assertion failure fails the test.
func contractServer(t *testing.T, wantMethod, wantPath string, wantBody map[string]any, status int, response string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != wantMethod {
			t.Errorf("method = %s, want %s", r.Method, wantMethod)
		}
		if r.URL.Path != wantPath {
			t.Errorf("path = %s, want %s", r.URL.Path, wantPath)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("authorization header = %q, want bearer token", auth)
		}
		if wantBody != nil {
			if ct := r.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("content-type = %q", ct)
			}
			var got map[string]any
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Errorf("decode request body: %v", err)
			}
			for k, v := range wantBody {
				wantJSON, _ := json.Marshal(v)
				gotJSON, _ := json.Marshal(got[k])
				if string(wantJSON) != string(gotJSON) {
					t.Errorf("body[%s] = %s, want %s", k, gotJSON, wantJSON)
				}
			}
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
}

func TestListConnectionsContract(t *testing.T) {
	srv := contractServer(t, http.MethodGet, "/v1/oauth/connections", nil, 200,
		`{"connections":[{"id":"conn_1","provider":"box","connection_type":"oauth",`+
			`"access_policy":{"allow_all":false,"allowed_endpoints":["/files/*"],"name_prefix":"studio-"},`+
			`"provider_account_id":"A1","transport":"http","usage_snippet":"curl X"}]}`)
	defer srv.Close()
	client := NewConnectorClient("test-token", srv.URL)
	conns, err := client.ListConnections(context.Background(), "")
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(conns) != 1 || conns[0].ID != "conn_1" || conns[0].Provider != "box" {
		t.Fatalf("connections = %+v", conns)
	}
	if conns[0].AccessPolicy.NamePrefix != "studio-" {
		t.Errorf("policy = %+v", conns[0].AccessPolicy)
	}
	if conns[0].GatewaySnippet != "curl X" {
		t.Errorf("snippet = %q", conns[0].GatewaySnippet)
	}
}

func TestListConnectionsProviderFilter(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"connections":[]}`))
	}))
	defer srv.Close()
	client := NewConnectorClient("test-token", srv.URL)
	if _, err := client.ListConnections(context.Background(), "slack"); err != nil {
		t.Fatal(err)
	}
	if gotQuery != "provider=slack" {
		t.Errorf("query = %q", gotQuery)
	}
}

func TestCreateCustomAPIContract(t *testing.T) {
	srv := contractServer(t, http.MethodPost, "/v1/oauth/connections/custom_api",
		map[string]any{
			"name": "acme", "base_api_url": "https://api.acme.example",
			"auth_method": "header", "access_token": "sk-test",
			"access_policy": map[string]any{"allow_all": false, "name_prefix": "studio-"},
		}, 201, `{"connection":{"id":"conn_acme","provider":"custom_api","connection_type":"api_key"}}`)
	defer srv.Close()
	client := NewConnectorClient("test-token", srv.URL)
	conn, err := client.CreateCustomAPI(context.Background(), CustomAPIRequest{
		Name: "acme", BaseAPIURL: "https://api.acme.example",
		AuthMethod: AuthHeader, AccessToken: "sk-test",
		AccessPolicy: &AccessPolicy{NamePrefix: "studio-"},
	})
	if err != nil {
		t.Fatalf("CreateCustomAPI: %v", err)
	}
	if conn.ID != "conn_acme" || conn.Provider != "custom_api" {
		t.Errorf("connection = %+v", conn)
	}
}

func TestCustomAPIErrorStatuses(t *testing.T) {
	for _, status := range []int{400, 401, 403, 422} {
		srv := contractServer(t, http.MethodPost, "/v1/oauth/connections/custom_api", nil, status, `{"error":"nope"}`)
		client := NewConnectorClient("test-token", srv.URL)
		_, err := client.CreateCustomAPI(context.Background(), CustomAPIRequest{})
		srv.Close()
		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("status %d: error type %T, want *APIError", status, err)
		}
		if apiErr.Status != status {
			t.Errorf("status = %d, want %d", apiErr.Status, status)
		}
	}
}

func TestTestCustomAPIContract(t *testing.T) {
	srv := contractServer(t, http.MethodPost, "/v1/oauth/connections/custom_api/test",
		map[string]any{"connection_id": "conn_acme"}, 200,
		`{"status":"ok","message":"connected"}`)
	defer srv.Close()
	client := NewConnectorClient("test-token", srv.URL)
	res, err := client.TestCustomAPI(context.Background(), CustomAPITestRequest{ConnectionID: "conn_acme"})
	if err != nil {
		t.Fatalf("TestCustomAPI: %v", err)
	}
	if res.Status != "ok" || res.Message != "connected" {
		t.Errorf("result = %+v", res)
	}
}

func TestTestCustomAPIErrorStatus(t *testing.T) {
	srv := contractServer(t, http.MethodPost, "/v1/oauth/connections/custom_api/test", nil, 200,
		`{"status":"error","error":"bad credentials"}`)
	defer srv.Close()
	client := NewConnectorClient("test-token", srv.URL)
	res, err := client.TestCustomAPI(context.Background(), CustomAPITestRequest{ConnectionID: "c"})
	if err != nil {
		t.Fatalf("TestCustomAPI: %v", err)
	}
	if res.Status != "error" || !strings.Contains(res.Message, "bad credentials") {
		t.Errorf("result = %+v", res)
	}
}

func TestUpdateCustomAPIContract(t *testing.T) {
	srv := contractServer(t, http.MethodPut, "/v1/oauth/connections/custom_api/conn_acme",
		map[string]any{"description": "new"}, 200, `{}`)
	defer srv.Close()
	client := NewConnectorClient("test-token", srv.URL)
	if err := client.UpdateCustomAPI(context.Background(), "conn_acme",
		CustomAPIUpdateRequest{Description: "new"}); err != nil {
		t.Fatalf("UpdateCustomAPI: %v", err)
	}
	if err := client.UpdateCustomAPI(context.Background(), "", CustomAPIUpdateRequest{}); err == nil {
		t.Error("empty ID accepted, want error")
	}
}
