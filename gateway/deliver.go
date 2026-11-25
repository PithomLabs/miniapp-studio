package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

// MaxDeliveryAttempts bounds redelivery of one callback. Past the cap the
// item parks as failed (kept, inspectable) instead of retrying forever.
// Failed items recover through the operator outbox_inspect/outbox_requeue
// commands; the Phase C Studio sender loop must additionally reconcile
// outbox-failed items (it is the standing backstop for anything the
// operator does not requeue by hand).
const MaxDeliveryAttempts = 25

// DefaultAttemptTimeout bounds one callback POST. A hung Studio must
// stall at most one attempt, never the single-threaded deliverer.
const DefaultAttemptTimeout = 30 * time.Second

// ErrInsecureStudioTransport reports a Studio URL violating the
// transport rule. It fails the item terminally (config errors never
// heal by retrying); the operator fixes the URL and requeues.
var ErrInsecureStudioTransport = errors.New("gateway: insecure studio transport")

// ErrPermanentDelivery marks attempt faults that cannot heal by
// retrying (corrupt stored envelope, missing signing key): terminal
// with an inspectable reason, never silent.
var ErrPermanentDelivery = errors.New("gateway: undeliverable callback")

// Deliverer POSTs outbox callbacks to Studio until acknowledged, with
// exponential backoff and jitter. 2xx acks; 429/5xx/network errors retry;
// other 4xx fail terminally (a malformed or unauthorized callback would
// never succeed on retry). Every attempt re-signs its envelope with a
// fresh timestamp under the stable message id, so items delayed past
// the freshness window still deliver.
type Deliverer struct {
	Outbox   *Outbox
	Studio   string
	Keys     KeyProvider
	Client   *http.Client
	Tick     time.Duration
	BaseWait time.Duration
	MaxWait  time.Duration
	// AttemptTimeout bounds one POST; zero means the default.
	AttemptTimeout time.Duration
	Now            func() time.Time
	Rand           func(int64) int64
}

// Run delivers due callbacks until ctx ends. One pass runs immediately so
// tests and restarts don't wait out the first tick.
func (d *Deliverer) Run(ctx context.Context) {
	d.pass(ctx)
	tick := d.Tick
	if tick <= 0 {
		tick = time.Second
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.pass(ctx)
		}
	}
}

func (d *Deliverer) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

func (d *Deliverer) attemptTimeout() time.Duration {
	if d.AttemptTimeout > 0 {
		return d.AttemptTimeout
	}
	return DefaultAttemptTimeout
}

// backoff schedules attempt n (1-based): base * 2^(n-1) plus jitter,
// capped. Pure and unit-tested.
func (d *Deliverer) backoff(n int) time.Duration {
	base := d.BaseWait
	if base <= 0 {
		base = time.Second
	}
	max := d.MaxWait
	if max <= 0 {
		max = 5 * time.Minute
	}
	wait := float64(base) * math.Pow(2, float64(n-1))
	if wait > float64(max) {
		wait = float64(max)
	}
	var jitter int64
	if d.Rand != nil {
		jitter = d.Rand(int64(base))
	} else if base > 0 {
		jitter = rand.Int64N(int64(base))
	}
	return time.Duration(wait) + time.Duration(jitter)
}

func (d *Deliverer) pass(ctx context.Context) {
	for _, item := range d.Outbox.Due(d.now()) {
		if err := d.Outbox.MarkSending(item.ID); err != nil {
			continue
		}
		attempts := item.Attempts + 1
		status, derr := d.deliver(ctx, item)
		switch {
		case derr == nil && status >= 200 && status < 300:
			_ = d.Outbox.MarkAcked(item.ID)
		case errors.Is(derr, ErrInsecureStudioTransport) || errors.Is(derr, ErrPermanentDelivery):
			_ = d.Outbox.MarkFailed(item.ID, derr.Error())
		case derr == nil && status != 429 && status >= 400 && status < 500:
			_ = d.Outbox.MarkFailed(item.ID, fmt.Sprintf("studio refused with %d", status))
		default:
			reason := ""
			if derr != nil {
				reason = derr.Error()
			} else {
				reason = fmt.Sprintf("studio status %d", status)
			}
			if attempts >= MaxDeliveryAttempts {
				_ = d.Outbox.MarkFailed(item.ID, "attempts exhausted: "+reason)
				continue
			}
			_ = d.Outbox.MarkRetry(item.ID, d.now().Add(d.backoff(attempts)), reason)
		}
	}
}

func (d *Deliverer) deliver(ctx context.Context, item OutboxItem) (int, error) {
	if err := RequireSecureTransport(d.Studio); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInsecureStudioTransport, err)
	}
	payload, err := d.attemptBody(item)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrPermanentDelivery, err)
	}
	timeout := d.attemptTimeout()
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	url := strings.TrimSuffix(d.Studio, "/") + "/gw/v1/callback"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", MediaTypeV1)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	return resp.StatusCode, nil
}

// attemptBody renders one attempt's bytes: the stored envelope parsed
// strictly and re-signed with a fresh timestamp under its stable id.
// The stored bytes stay immutable (forensics + stable identity);
// only the wire bytes refresh.
func (d *Deliverer) attemptBody(item OutboxItem) ([]byte, error) {
	if d.Keys == nil {
		return nil, fmt.Errorf("gateway: deliverer has no keys")
	}
	env, err := Parse(item.Envelope)
	if err != nil {
		return nil, fmt.Errorf("gateway: corrupt stored envelope: %w", err)
	}
	key, err := d.Keys.KeyFor(env.BuildID)
	if err != nil {
		return nil, err
	}
	return Resign(key, env, d.now())
}
