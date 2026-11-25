package fly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// SpritesAPIBase is the default Connectors API origin.
const SpritesAPIBase = "https://api.sprites.dev"

// ConnectorClient is the typed Connectors REST client. The sprites-go SDK
// exposes no connector management, so this client covers the documented
// endpoints directly. It never logs request or response bodies: requests
// carry credentials and responses must be treated as sensitive.
type ConnectorClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewConnectorClient returns a client authenticated with token. An empty
// baseURL selects SpritesAPIBase (overridable for contract tests).
func NewConnectorClient(token, baseURL string) *ConnectorClient {
	if baseURL == "" {
		baseURL = SpritesAPIBase
	}
	return &ConnectorClient{baseURL: baseURL, token: token,
		http: &http.Client{Timeout: 30 * time.Second}}
}

// APIError reports a non-2xx Connectors API response without echoing bodies.
type APIError struct {
	Status  int
	Op      string
	Message string
}

// Error renders status, operation, and the server message.
func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("fly: connectors %s failed with status %d", e.Op, e.Status)
	}
	return fmt.Sprintf("fly: connectors %s failed with status %d: %s", e.Op, e.Status, e.Message)
}

func (c *ConnectorClient) do(ctx context.Context, op, method, path string, query url.Values, body any, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("fly: connectors %s: encode: %w", op, err)
		}
		reader = bytes.NewReader(encoded)
	}
	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return fmt.Errorf("fly: connectors %s: %w", op, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("fly: connectors %s: %w", op, err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("fly: connectors %s: read: %w", op, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Op: op, Message: shortError(payload)}
	}
	if out != nil {
		if err := json.Unmarshal(payload, out); err != nil {
			return fmt.Errorf("fly: connectors %s: decode: %w", op, err)
		}
	}
	return nil
}

func shortError(payload []byte) string {
	text := string(payload)
	if len(text) > 300 {
		text = text[:300]
	}
	return text
}

type wireAccessPolicy struct {
	// AllowAll is always serialized, even when false: default-deny must be
	// explicit on the wire, never inferred from an absent field.
	AllowAll         bool     `json:"allow_all"`
	AllowedEndpoints []string `json:"allowed_endpoints,omitempty"`
	BlockedEndpoints []string `json:"blocked_endpoints,omitempty"`
	NamePrefix       string   `json:"name_prefix,omitempty"`
	SpriteLabels     []string `json:"sprite_labels,omitempty"`
}

func toWirePolicy(p *AccessPolicy) *wireAccessPolicy {
	if p == nil {
		return nil
	}
	return &wireAccessPolicy{
		AllowAll: p.AllowAll, AllowedEndpoints: p.AllowedEndpoints,
		BlockedEndpoints: p.BlockedEndpoints, NamePrefix: p.NamePrefix,
		SpriteLabels: p.SpriteLabels,
	}
}

func fromWirePolicy(p wireAccessPolicy) AccessPolicy {
	return AccessPolicy{
		AllowAll: p.AllowAll, AllowedEndpoints: p.AllowedEndpoints,
		BlockedEndpoints: p.BlockedEndpoints, NamePrefix: p.NamePrefix,
		SpriteLabels: p.SpriteLabels,
	}
}

type wireConnection struct {
	ID                string           `json:"id"`
	Provider          string           `json:"provider"`
	ConnectionType    string           `json:"connection_type"`
	AccessPolicy      wireAccessPolicy `json:"access_policy"`
	ProviderAccountID string           `json:"provider_account_id"`
	Transport         string           `json:"transport"`
	UsageSnippet      *string          `json:"usage_snippet"`
}

func fromWireConnection(c wireConnection) Connection {
	out := Connection{
		ID: c.ID, Provider: c.Provider, ConnectionType: c.ConnectionType,
		AccessPolicy:      fromWirePolicy(c.AccessPolicy),
		ProviderAccountID: c.ProviderAccountID, Transport: c.Transport,
	}
	if c.UsageSnippet != nil {
		out.GatewaySnippet = *c.UsageSnippet
	}
	return out
}

// ListConnections implements ConnectorManager.
func (c *ConnectorClient) ListConnections(ctx context.Context, provider string) ([]Connection, error) {
	var query url.Values
	if provider != "" {
		query = url.Values{"provider": {provider}}
	}
	var decoded struct {
		Connections []wireConnection `json:"connections"`
	}
	if err := c.do(ctx, "list", http.MethodGet, "/v1/oauth/connections", query, nil, &decoded); err != nil {
		return nil, err
	}
	out := make([]Connection, 0, len(decoded.Connections))
	for _, wire := range decoded.Connections {
		out = append(out, fromWireConnection(wire))
	}
	return out, nil
}

type wireCustomAPICreate struct {
	Name                string            `json:"name"`
	Description         string            `json:"description,omitempty"`
	BaseAPIURL          string            `json:"base_api_url"`
	TestURL             string            `json:"test_url,omitempty"`
	AuthMethod          string            `json:"auth_method,omitempty"`
	AuthHeaderPrefix    string            `json:"auth_header_prefix,omitempty"`
	AuthCustomHeader    string            `json:"auth_custom_header_name,omitempty"`
	AuthQueryParam      string            `json:"auth_query_param_name,omitempty"`
	AuthURLPathTemplate string            `json:"auth_url_path_template,omitempty"`
	AccessToken         string            `json:"access_token"`
	RefreshToken        string            `json:"refresh_token,omitempty"`
	AccessPolicy        *wireAccessPolicy `json:"access_policy,omitempty"`
}

// CreateCustomAPI implements ConnectorManager.
func (c *ConnectorClient) CreateCustomAPI(ctx context.Context, req CustomAPIRequest) (Connection, error) {
	var decoded struct {
		Connection wireConnection `json:"connection"`
	}
	err := c.do(ctx, "create-custom-api", http.MethodPost, "/v1/oauth/connections/custom_api", nil,
		wireCustomAPICreate{
			Name: req.Name, Description: req.Description,
			BaseAPIURL: req.BaseAPIURL, TestURL: req.TestURL,
			AuthMethod: req.AuthMethod, AuthHeaderPrefix: req.AuthHeaderPrefix,
			AuthCustomHeader: req.AuthCustomHeader, AuthQueryParam: req.AuthQueryParam,
			AuthURLPathTemplate: req.AuthURLPathTemplate,
			AccessToken:         req.AccessToken, RefreshToken: req.RefreshToken,
			AccessPolicy: toWirePolicy(req.AccessPolicy),
		}, &decoded)
	if err != nil {
		return Connection{}, err
	}
	return fromWireConnection(decoded.Connection), nil
}

type wireCustomAPITest struct {
	ConnectionID        string `json:"connection_id,omitempty"`
	AccessToken         string `json:"access_token,omitempty"`
	BaseAPIURL          string `json:"base_api_url,omitempty"`
	TestURL             string `json:"test_url,omitempty"`
	AuthMethod          string `json:"auth_method,omitempty"`
	AuthHeaderPrefix    string `json:"auth_header_prefix,omitempty"`
	AuthCustomHeader    string `json:"auth_custom_header_name,omitempty"`
	AuthQueryParam      string `json:"auth_query_param_name,omitempty"`
	AuthURLPathTemplate string `json:"auth_url_path_template,omitempty"`
}

// TestCustomAPI implements ConnectorManager. Response parsing is tolerant:
// status is required, the message is read from the first present text field.
func (c *ConnectorClient) TestCustomAPI(ctx context.Context, req CustomAPITestRequest) (CustomAPITestResult, error) {
	var decoded map[string]any
	err := c.do(ctx, "test-custom-api", http.MethodPost, "/v1/oauth/connections/custom_api/test", nil,
		wireCustomAPITest{
			ConnectionID: req.ConnectionID, AccessToken: req.AccessToken,
			BaseAPIURL: req.BaseAPIURL, TestURL: req.TestURL,
			AuthMethod: req.AuthMethod, AuthHeaderPrefix: req.AuthHeaderPrefix,
			AuthCustomHeader: req.AuthCustomHeader, AuthQueryParam: req.AuthQueryParam,
			AuthURLPathTemplate: req.AuthURLPathTemplate,
		}, &decoded)
	if err != nil {
		return CustomAPITestResult{}, err
	}
	out := CustomAPITestResult{}
	if s, _ := decoded["status"].(string); s != "" {
		out.Status = s
	} else {
		return CustomAPITestResult{}, fmt.Errorf("fly: connectors test-custom-api: missing status")
	}
	for _, key := range []string{"message", "error", "detail"} {
		if s, _ := decoded[key].(string); s != "" {
			out.Message = s
			break
		}
	}
	return out, nil
}

// UpdateCustomAPI implements ConnectorManager against the documented update
// shape. The endpoint contract comes from official docs; live verification
// against a real connection is still pending (spike procedure, not yet run).
func (c *ConnectorClient) UpdateCustomAPI(ctx context.Context, id string, req CustomAPIUpdateRequest) error {
	if id == "" {
		return fmt.Errorf("fly: connectors update-custom-api: id is required")
	}
	body := map[string]any{}
	if req.Description != "" {
		body["description"] = req.Description
	}
	if req.BaseAPIURL != "" {
		body["base_api_url"] = req.BaseAPIURL
	}
	if req.TestURL != "" {
		body["test_url"] = req.TestURL
	}
	if req.AccessToken != "" {
		body["access_token"] = req.AccessToken
	}
	if req.AccessPolicy != nil {
		body["access_policy"] = toWirePolicy(req.AccessPolicy)
	}
	return c.do(ctx, "update-custom-api", http.MethodPut,
		"/v1/oauth/connections/custom_api/"+url.PathEscape(id), nil, body, nil)
}
