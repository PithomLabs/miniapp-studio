package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Exchanger trades a refresh token for a short-lived access token at
// the provider. Implementations speak provider wire protocols; the
// broker never sees anything but (access, expiresIn) or a typed
// failure. Refresh plaintext arrives as an argument and must not be
// retained or logged by implementations.
type Exchanger interface {
	Exchange(ctx context.Context, provider, refresh string) (access string, expiresIn time.Duration, err error)
}

// HTTPExchanger is the generic OAuth2 refresh-token grant: one POST
// of grant_type=refresh_token. Per-provider endpoints and client
// credentials arrive from the caller (connector increments own the
// provider registry); this type only owns the wire shape.
type HTTPExchanger struct {
	Endpoint     string
	ClientID     string
	ClientSecret string
	Client       *http.Client
}

func (h *HTTPExchanger) client() *http.Client {
	if h.Client != nil {
		return h.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// Exchange posts the refresh grant and parses the token response.
// invalid_grant maps to ErrInvalidGrant (the broker auto-revokes);
// any other failure is returned verbatim without secret bytes.
func (h *HTTPExchanger) Exchange(ctx context.Context, provider, refresh string) (string, time.Duration, error) {
	if h.Endpoint == "" {
		return "", 0, fmt.Errorf("broker: no token endpoint for provider %q", provider)
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
		"client_id":     {h.ClientID},
		"client_secret": {h.ClientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.Endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("broker: token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := h.client().Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("broker: token endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", 0, fmt.Errorf("broker: token response: %w", err)
	}
	var doc struct {
		Access    string `json:"access_token"`
		ExpiresIn int64  `json:"expires_in"`
		ErrCode   string `json:"error"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", 0, fmt.Errorf("broker: malformed token response (http %d)", resp.StatusCode)
	}
	if doc.ErrCode == "invalid_grant" {
		return "", 0, ErrInvalidGrant
	}
	if doc.ErrCode != "" {
		return "", 0, fmt.Errorf("broker: token endpoint error %q (http %d)", doc.ErrCode, resp.StatusCode)
	}
	if resp.StatusCode/100 != 2 || doc.Access == "" || doc.ExpiresIn <= 0 {
		return "", 0, fmt.Errorf("broker: token endpoint refused (http %d)", resp.StatusCode)
	}
	return doc.Access, time.Duration(doc.ExpiresIn) * time.Second, nil
}
