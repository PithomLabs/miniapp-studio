package fly

import (
	"context"
	"errors"
	"fmt"
)

// AdapterLifecycleProvider implements the SDK-supported LifecycleProvider
// verbs over SpriteAdapter: provision, destroy, and existence checks are
// real API calls, and session attach/detach run the documented in-sprite
// Tasks API over exec (create/delete a management hold). Only SetPower
// refuses with *UnsupportedError: sprites-go v0.2.2 exposes no power
// primitive, the sprite CLI has no power verb, and the vendor docs state
// pause/wake are automatic — sleep is observed, wake is any request.
// The adapter invents no endpoints.
type AdapterLifecycleProvider struct {
	inner *SpriteAdapter
}

// NewAdapterLifecycleProvider returns a provider over one adapter.
func NewAdapterLifecycleProvider(inner *SpriteAdapter) *AdapterLifecycleProvider {
	if inner == nil {
		panic("fly: lifecycle adapter needs a SpriteAdapter")
	}
	return &AdapterLifecycleProvider{inner: inner}
}

// Name identifies the backend on receipts.
func (p *AdapterLifecycleProvider) Name() string { return "sprites-sdk" }

// Provision creates one sprite through the API.
func (p *AdapterLifecycleProvider) Provision(ctx context.Context, name string, opts SpriteOptions) error {
	_, err := p.inner.CreateSprite(ctx, name, opts)
	return err
}

// Destroy deletes one sprite through the API.
func (p *AdapterLifecycleProvider) Destroy(ctx context.Context, name string) error {
	return p.inner.DeleteSprite(ctx, name)
}

// Describe reports existence: a missing sprite is absent, not an error.
// Any other failure propagates so reconciliation never mistakes an
// outage for absence.
func (p *AdapterLifecycleProvider) Describe(ctx context.Context, name string) (bool, error) {
	_, err := p.inner.GetSprite(ctx, name)
	if err == nil {
		return true, nil
	}
	var nf *NotFoundError
	if errors.As(err, &nf) {
		return false, nil
	}
	return false, err
}

// SetPower refuses: pausing a sprite is not an API operation. The vendor
// docs state sprites pause automatically on idle; Studio observes the
// transition (B2-01B proof step) instead of commanding it.
func (p *AdapterLifecycleProvider) SetPower(_ context.Context, _ string, _ bool) error {
	return &UnsupportedError{Op: "SetPower",
		Detail: "sprites pause automatically on idle; no power API exists to call"}
}

// Management-hold identity for the Tasks API. One fixed hold per sprite
// covers the single-sprite B2-01B proof; per-tenant/per-purpose naming
// arrives with fleets (out of scope). The 30-minute expiry spans the
// connect → observe → disconnect proof with room to spare; expiry is the
// backstop, and hold renewal for long agent runs is B2-05 scope.
const (
	lifecycleHoldName   = "studio-mgmt"
	lifecycleHoldExpiry = "30m"
)

// tasksHoldArgs builds the exact exec argv that creates the management
// hold: the vendor-documented explicit curl form over the in-sprite
// management socket. -f fails loudly on HTTP errors — an unexpected
// Tasks API answer must surface, never pass as success.
func tasksHoldArgs() []string {
	return []string{"curl", "-s", "--unix-socket", "/.sprite/api.sock",
		"-H", "Content-Type: application/json", "-f", "-X", "POST",
		"http://sprite/v1/tasks",
		"-d", `{"name":"` + lifecycleHoldName + `","expire":"` + lifecycleHoldExpiry + `"}`}
}

// tasksReleaseArgs builds the exact exec argv that deletes the hold,
// freeing the sprite to pause.
func tasksReleaseArgs() []string {
	return []string{"curl", "-s", "--unix-socket", "/.sprite/api.sock",
		"-f", "-X", "DELETE", "http://sprite/v1/tasks/" + lifecycleHoldName}
}

// Attach creates the Studio management hold inside the sprite. Exec wakes
// a paused sprite first (documented wake-on-request), so attach also
// re-establishes reachability; the receipt's latency captures the cost.
func (p *AdapterLifecycleProvider) Attach(ctx context.Context, name string) error {
	res, err := p.inner.Exec(ctx, name, tasksHoldArgs())
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("fly: tasks hold create failed (exit %d): %s",
			res.ExitCode, boundLifecycleOutput(res.Stdout+res.Stderr))
	}
	return nil
}

// Detach deletes the Studio management hold, freeing the sprite to pause.
// An already-released hold (expiry raced the disconnect) surfaces as an
// error for the live proof to record; renewal/disconnect races are B2-05
// scope.
func (p *AdapterLifecycleProvider) Detach(ctx context.Context, name string) error {
	res, err := p.inner.Exec(ctx, name, tasksReleaseArgs())
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("fly: tasks hold delete failed (exit %d): %s",
			res.ExitCode, boundLifecycleOutput(res.Stdout+res.Stderr))
	}
	return nil
}

// boundLifecycleOutput caps failure text. Tasks API responses carry no
// credentials, but output is bounded on principle.
func boundLifecycleOutput(s string) string {
	const max = 1000
	r := []rune(s)
	if len(r) > max {
		return string(r[:max]) + "…[truncated]"
	}
	return s
}
