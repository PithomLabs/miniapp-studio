package broker

import (
	"context"
	"encoding/json"
	"errors"
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

// errNoRedirect refuses every redirect: refresh tokens and client
// secrets go to the configured endpoint only (F35-04). Go's default
// client would forward the credential-bearing body on 307/308.
var errNoRedirect = errors.New("broker: token endpoint redirected (refused)")

func (h *HTTPExchanger) client() *http.Client {
	base := h.Client
	if base == nil {
		base = &http.Client{Timeout: 10 * time.Second}
	}
	// Clone before arming: the caller's client must not be mutated,
	// and a caller-supplied CheckRedirect must not re-enable
	// following (the refusal is this type's invariant).
	c := *base
	c.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return errNoRedirect
	}
	return &c
}

// oauthErrorCodes are the RFC 6749 §5.2 error codes plus the
// registered refresh-flow extensions the broker repeats verbatim.
// Anything else is provider-controlled text and stays out of
// errors (F35-13).
var oauthErrorCodes = map[string]bool{
	"invalid_request": true, "invalid_client": true,
	"invalid_grant": true, "unauthorized_client": true,
	"unsupported_grant_type": true, "invalid_scope": true,
	"temporarily_unavailable": true,
}

// Exchange posts the refresh grant and parses the token response.
// invalid_grant maps to ErrInvalidGrant (the broker auto-revokes);
// redirects fail closed without forwarding credentials; unknown
// provider error codes are bucketed, never echoed.
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
		// errors.Is unwraps the *url.Error the client wraps
		// CheckRedirect failures in.
		if errors.Is(err, errNoRedirect) {
			return "", 0, fmt.Errorf("broker: token endpoint redirected (refused, credentials not forwarded)")
		}
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
		if !oauthErrorCodes[doc.ErrCode] {
			return "", 0, fmt.Errorf("broker: token endpoint error (unrecognized code, http %d)", resp.StatusCode)
		}
		return "", 0, fmt.Errorf("broker: token endpoint error %q (http %d)", doc.ErrCode, resp.StatusCode)
	}
	if resp.StatusCode/100 != 2 || doc.Access == "" || doc.ExpiresIn <= 0 {
		return "", 0, fmt.Errorf("broker: token endpoint refused (http %d)", resp.StatusCode)
	}
	return doc.Access, time.Duration(doc.ExpiresIn) * time.Second, nil
}
