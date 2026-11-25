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

// Send delivers one command and returns the decoded response.
func (c *Client) Send(ctx context.Context, gatewayURL string, key []byte, buildID, name string, payload map[string]any) (map[string]any, error) {
	if buildID == "" || name == "" {
		return nil, fmt.Errorf("gateway: build and command name required")
	}
	if err := RequireSecureTransport(gatewayURL); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	var id [16]byte
	if _, err := crand.Read(id[:]); err != nil {
		return nil, fmt.Errorf("gateway: message id: %w", err)
	}
	env := Envelope{
		Protocol: ProtocolV1, MessageID: hex.EncodeToString(id[:]),
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
			return nil, true, fmt.Errorf("gateway: command decode: %w", err)
		}
		return out, false, nil
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("gateway: command status %d: %s", resp.StatusCode, truncateBody(body))
	}
	return nil, false, fmt.Errorf("gateway: command refused %d: %s", resp.StatusCode, truncateBody(body))
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
