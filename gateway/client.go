package gateway

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

// Client sends signed commands to a Builder Gateway: one envelope built
// and signed once, then redelivered byte-identically on retry so gateway
// dedup collapses retries. 2xx decodes the JSON response map; 4xx fails
// terminally (malformed, unauthorized, unknown, or illegal-state —
// retrying cannot help); network errors, 429, and 5xx retry with
// exponential backoff, which also covers wake-on-command latency.
type Client struct {
	HTTP     *http.Client
	Timeout  time.Duration
	Retries  int
	BaseWait time.Duration
	MaxWait  time.Duration
}

// CallError is a gateway response the caller must classify: any non-2xx,
// or a 2xx whose body is not JSON. Error() preserves the historic message
// text; Status and Body (nil when the body is not a JSON object) let
// callers classify without string parsing.
type CallError struct {
	Status int
	Body   map[string]any
	msg    string
}

func (e *CallError) Error() string { return e.msg }

// callRefused wraps one non-2xx response, decoding the JSON body when it
// decodes so the machine-readable "code" survives to the caller.
func callRefused(status int, body []byte, format string) error {
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		decoded = nil
	}
	return &CallError{
		Status: status, Body: decoded,
		msg: fmt.Sprintf(format, status, truncateBody(body)),
	}
}

// Send delivers one command and returns the decoded response.
func (c *Client) Send(ctx context.Context, gatewayURL string, key []byte, buildID, name string, payload map[string]any) (map[string]any, error) {
	var id [16]byte
	if _, err := crand.Read(id[:]); err != nil {
		return nil, fmt.Errorf("gateway: message id: %w", err)
	}
	return c.SendWithID(ctx, gatewayURL, key, buildID, hex.EncodeToString(id[:]), name, payload)
}

// SendWithID delivers one command under a caller-supplied stable message
// id and returns the decoded response. Studio's sender passes the durable
// command-row id so every retry of every attempt — including after a
// Studio crash — collapses onto the gateway's recorded outcome for that
// id instead of executing twice.
func (c *Client) SendWithID(ctx context.Context, gatewayURL string, key []byte, buildID, messageID, name string, payload map[string]any) (map[string]any, error) {
	if buildID == "" || name == "" || messageID == "" {
		return nil, fmt.Errorf("gateway: build, message id, and command name required")
	}
	if err := RequireSecureTransport(gatewayURL); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	env := Envelope{
		Protocol: ProtocolV1, MessageID: messageID,
		BuildID: buildID, SentAt: time.Now().UTC().Format(time.RFC3339),
		Kind: "command", Name: name, Payload: payload,
	}
	body, err := canonicalBytes(env)
	if err != nil {
		return nil, fmt.Errorf("gateway: encode: %w", err)
	}
	mac := hmacSum(key, body)
	env.Signature = hex.EncodeToString(mac)
	raw, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("gateway: encode: %w", err)
	}
	url := strings.TrimSuffix(gatewayURL, "/") + "/gw/v1/command"
	retries := c.Retries
	if retries < 0 {
		retries = 0
	}
	var last error
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			if err := sleepContext(ctx, c.wait(attempt)); err != nil {
				return nil, err
			}
		}
		out, retry, err := c.once(ctx, url, raw)
		if err == nil {
			return out, nil
		}
		last = err
		if !retry {
			return nil, err
		}
	}
	return nil, last
}

func (c *Client) once(ctx context.Context, url string, raw []byte) (map[string]any, bool, error) {
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", MediaTypeV1)
	resp, err := client.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("gateway: command transport: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxCommandBytes))
	if err != nil {
		return nil, true, fmt.Errorf("gateway: command read: %w", err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var out map[string]any
		if err := json.Unmarshal(body, &out); err != nil {
			// Accepted but unreadable: the command executed (2xx), so
			// callers must reconcile rather than treat this as a
			// transport failure. Status + nil Body marks that.
			return nil, true, &CallError{Status: resp.StatusCode,
				msg: fmt.Sprintf("gateway: command decode: %v", err)}
		}
		return out, false, nil
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, callRefused(resp.StatusCode, body, "gateway: command status %d: %s")
	}
	return nil, false, callRefused(resp.StatusCode, body, "gateway: command refused %d: %s")
}

func (c *Client) wait(attempt int) time.Duration {
	base := c.BaseWait
	if base <= 0 {
		base = time.Second
	}
	max := c.MaxWait
	if max <= 0 {
		max = 30 * time.Second
	}
	wait := float64(base) * math.Pow(2, float64(attempt-1))
	if wait > float64(max) {
		wait = float64(max)
	}
	var jitter int64
	if base > 0 {
		jitter = rand.Int64N(int64(base))
	}
	return time.Duration(wait) + time.Duration(jitter)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func truncateBody(body []byte) string {
	const max = 300
	s := strings.TrimSpace(string(body))
	if len(s) > max {
		return s[:max] + "…"
	}
	if s == "" {
		return "empty"
	}
	return s
}
