package fly

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Sprite lifecycle contract (B2-01A). Studio owns the lifecycle state
// machine, the per-transition authorization gate, idempotent replay, and
// the receipt shape. Providers execute effects: the deterministic fake
// implements every verb; the SDK adapter implements only what
// sprites-go v0.2.2 actually supports and refuses the rest with
// *UnsupportedError (B2-01B owns live discovery of the power and
// session mechanisms).
//
// Controller state is in-memory and authoritative only within one
// process: restart drops it, and Reconcile re-derives the coarsest safe
// position (absent vs present) from the provider. Durable lifecycle
// position is B2-01B's scope (durable receipts), not this contract's.

// Transition names one lifecycle state change.
type Transition string

// Lifecycle transitions. Provision, wake, sleep, connect, and disconnect
// are the B2-01A card's five; destroy is included because leaving the
// destructive transition outside the contract would punch an ungated hole
// through it, and B2-01B's recovery already requires destroy-scoped
// authorization.
const (
	TransitionProvision  Transition = "provision"
	TransitionWake       Transition = "wake"
	TransitionSleep      Transition = "sleep"
	TransitionConnect    Transition = "connect"
	TransitionDisconnect Transition = "disconnect"
	TransitionDestroy    Transition = "destroy"
)

// LifecycleState is Studio's tracked position of one sprite.
type LifecycleState string

// Lifecycle positions. Connected means awake with Studio's management
// session attached; detach before sleep, never silently.
const (
	StateAbsent    LifecycleState = "absent"
	StateAsleep    LifecycleState = "asleep"
	StateAwake     LifecycleState = "awake"
	StateConnected LifecycleState = "connected"
)

// Named refusal reasons. Every gate denial carries one; callers match on
// Reason, never on message text.
const (
	ReasonAuthMissing            = "auth-missing"
	ReasonAuthExpired            = "auth-expired"
	ReasonAuthNoExpiry           = "auth-missing-expiry"
	ReasonAuthTransitionMismatch = "auth-transition-mismatch"
	ReasonAuthSpriteMismatch     = "auth-sprite-mismatch"
	ReasonAuthApproverMissing    = "auth-approver-missing"
	ReasonAuthReferenceMissing   = "auth-reference-missing"
	ReasonIllegalTransition      = "illegal-transition"
	ReasonKeyMissing             = "idempotency-key-missing"
	ReasonSpriteMissing          = "sprite-missing"
	ReasonReplayConflict         = "replay-scope-conflict"
	ReasonReplayStale            = "replay-stale"
)

// RefusalError is a fail-closed gate denial with a named reason and a
// human-actionable detail naming the unblocking action.
type RefusalError struct {
	Reason string
	Detail string
}

// Error renders reason and detail without secrets (decisions carry no
// credential material by construction).
func (e *RefusalError) Error() string {
	return fmt.Sprintf("fly: lifecycle refused (%s): %s", e.Reason, e.Detail)
}

// UnsupportedError reports a verb the provider honestly cannot perform.
// It is distinct from RefusalError: the gate passed, but the backend has
// no mechanism. Callers match with errors.As.
type UnsupportedError struct {
	Op     string
	Detail string
}

// Error names the verb and the missing mechanism.
func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("fly: lifecycle %s unsupported: %s", e.Op, e.Detail)
}

// AuthDecision is the authorization grant for exactly one transition on
// exactly one sprite. It carries identity and scope, never credentials.
// Zero ExpiresAt is missing, not infinite: expiry denies by default.
type AuthDecision struct {
	Approver   string
	Reference  string
	Transition Transition
	Sprite     string
	ExpiresAt  time.Time
}

// AuthRequest is one gate evaluation.
type AuthRequest struct {
	Transition Transition
	Sprite     string
	Decision   *AuthDecision
	Now        time.Time
}

// Authorizer gates one transition. Implementations return nil or a
// *RefusalError; any other error is a gate malfunction and fails closed.
type Authorizer interface {
	Authorize(ctx context.Context, req AuthRequest) error
}

// DecisionAuthorizer validates decision structure: presence, attribution,
// reference, scope, and expiry. Revocation state lives at the server
// boundary (B2-08); this authorizer owns the shape, not the registry.
type DecisionAuthorizer struct{}

// Authorize implements Authorizer.
func (DecisionAuthorizer) Authorize(_ context.Context, req AuthRequest) error {
	d := req.Decision
	if d == nil {
		return &RefusalError{Reason: ReasonAuthMissing,
			Detail: fmt.Sprintf("transition %q on %q needs an authorization decision", req.Transition, req.Sprite)}
	}
	if d.Approver == "" {
		return &RefusalError{Reason: ReasonAuthApproverMissing,
			Detail: "decision names no approver; obtain an attributable approval"}
	}
	if d.Reference == "" {
		return &RefusalError{Reason: ReasonAuthReferenceMissing,
			Detail: "decision carries no approval reference; receipts must bind an approval"}
	}
	if d.Transition != req.Transition {
		return &RefusalError{Reason: ReasonAuthTransitionMismatch,
			Detail: fmt.Sprintf("decision scopes %q, request is %q; one approval covers one action", d.Transition, req.Transition)}
	}
	if d.Sprite != req.Sprite {
		return &RefusalError{Reason: ReasonAuthSpriteMismatch,
			Detail: fmt.Sprintf("decision scopes sprite %q, request targets %q", d.Sprite, req.Sprite)}
	}
	if d.ExpiresAt.IsZero() {
		return &RefusalError{Reason: ReasonAuthNoExpiry,
			Detail: "decision carries no expiry; re-authorize with a bounded grant"}
	}
	if !req.Now.Before(d.ExpiresAt) {
		return &RefusalError{Reason: ReasonAuthExpired,
			Detail: fmt.Sprintf("decision expired at %s; re-authorize", d.ExpiresAt.UTC().Format(time.RFC3339))}
	}
	return nil
}

// LifecycleReceipt is the durable shape of one completed transition. Key is
// its stable identity: one intention, one receipt. Latency measures the
// provider effect so B2-01B can report wake-latency notes from receipts.
type LifecycleReceipt struct {
	Key        string
	Transition Transition
	Sprite     string
	From       LifecycleState
	To         LifecycleState
	At         time.Time
	Approver   string
	AuthRef    string
	Provider   string
	Latency    time.Duration
}

// ReceiptRecorder sinks completed-transition receipts.
type ReceiptRecorder interface {
	Record(LifecycleReceipt)
}

// MemoryRecorder keeps receipts in memory for tests and local runs.
type MemoryRecorder struct {
	mu sync.Mutex
	rs []LifecycleReceipt
}

// Record appends one receipt.
func (m *MemoryRecorder) Record(r LifecycleReceipt) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rs = append(m.rs, r)
}

// All returns recorded receipts in order.
func (m *MemoryRecorder) All() []LifecycleReceipt {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]LifecycleReceipt(nil), m.rs...)
}

// LifecycleProvider executes lifecycle effects. Name identifies the
// backend on receipts ("fake", "sprites-sdk"). Describe reports bare
// existence for reconciliation; power and session position are
// Studio-side state, not provider queries.
type LifecycleProvider interface {
	Name() string
	Provision(ctx context.Context, name string, opts SpriteOptions) error
	Destroy(ctx context.Context, name string) error
	Describe(ctx context.Context, name string) (exists bool, err error)
	SetPower(ctx context.Context, name string, awake bool) error
	Attach(ctx context.Context, name string) error
	Detach(ctx context.Context, name string) error
}

// Clock supplies the controller's timestamps. Tests inject a fixed clock
// for byte-identical receipts.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production Clock.
type SystemClock struct{}

// Now returns the current UTC time.
func (SystemClock) Now() time.Time { return time.Now().UTC() }

// Lifecycle is the Studio-owned lifecycle controller: authorize, replay,
// transition, record. One mutex serializes every transition including the
// provider call: lifecycle operations are human-paced, and holding the
// lock across the effect is what collapses concurrent same-key callers
// into exactly one execution. Nil provider, authorizer, or recorder is a
// programming error and panics; nil clock selects SystemClock.
type Lifecycle struct {
	mu       sync.Mutex
	provider LifecycleProvider
	auth     Authorizer
	rec      ReceiptRecorder
	clock    Clock
	states   map[string]LifecycleState
	replayed map[string]LifecycleReceipt
}

// NewLifecycle returns a controller over one provider.
func NewLifecycle(p LifecycleProvider, a Authorizer, r ReceiptRecorder, c Clock) *Lifecycle {
	if p == nil || a == nil || r == nil {
		panic("fly: lifecycle needs provider, authorizer, and recorder")
	}
	if c == nil {
		c = SystemClock{}
	}
	return &Lifecycle{provider: p, auth: a, rec: r, clock: c,
		states: map[string]LifecycleState{}, replayed: map[string]LifecycleReceipt{}}
}

// State returns the tracked position of one sprite, absent when unseen.
func (l *Lifecycle) State(sprite string) LifecycleState {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stateLocked(sprite)
}

func (l *Lifecycle) stateLocked(sprite string) LifecycleState {
	if st, ok := l.states[sprite]; ok {
		return st
	}
	return StateAbsent
}

// Provision creates one sprite: absent → awake.
func (l *Lifecycle) Provision(ctx context.Context, sprite, key string, d *AuthDecision, opts SpriteOptions) (LifecycleReceipt, error) {
	return l.do(ctx, TransitionProvision, sprite, key, d, StateAbsent, StateAwake,
		func(ctx context.Context) error { return l.provider.Provision(ctx, sprite, opts) })
}

// Wake powers one sprite: asleep → awake.
func (l *Lifecycle) Wake(ctx context.Context, sprite, key string, d *AuthDecision) (LifecycleReceipt, error) {
	return l.do(ctx, TransitionWake, sprite, key, d, StateAsleep, StateAwake,
		func(ctx context.Context) error { return l.provider.SetPower(ctx, sprite, true) })
}

// Sleep powers one sprite down: awake → asleep. Connected sessions must
// detach first; sleep never drops an attached session silently.
func (l *Lifecycle) Sleep(ctx context.Context, sprite, key string, d *AuthDecision) (LifecycleReceipt, error) {
	return l.do(ctx, TransitionSleep, sprite, key, d, StateAwake, StateAsleep,
		func(ctx context.Context) error { return l.provider.SetPower(ctx, sprite, false) })
}

// Connect attaches Studio's management session: awake → connected.
func (l *Lifecycle) Connect(ctx context.Context, sprite, key string, d *AuthDecision) (LifecycleReceipt, error) {
	return l.do(ctx, TransitionConnect, sprite, key, d, StateAwake, StateConnected,
		func(ctx context.Context) error { return l.provider.Attach(ctx, sprite) })
}

// Disconnect detaches the management session: connected → awake.
func (l *Lifecycle) Disconnect(ctx context.Context, sprite, key string, d *AuthDecision) (LifecycleReceipt, error) {
	return l.do(ctx, TransitionDisconnect, sprite, key, d, StateConnected, StateAwake,
		func(ctx context.Context) error { return l.provider.Detach(ctx, sprite) })
}

// Destroy deletes one sprite from any present state: * → absent.
// Destruction needs its own destroy-scoped decision; no other approval
// covers it.
func (l *Lifecycle) Destroy(ctx context.Context, sprite, key string, d *AuthDecision) (LifecycleReceipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	from := l.stateLocked(sprite)
	if from == StateAbsent {
		return LifecycleReceipt{}, &RefusalError{Reason: ReasonIllegalTransition,
			Detail: fmt.Sprintf("destroy needs a present sprite, %q is absent", sprite)}
	}
	return l.doLocked(ctx, TransitionDestroy, sprite, key, d, from, StateAbsent,
		func(ctx context.Context) error { return l.provider.Destroy(ctx, sprite) })
}

// do runs one fixed-edge transition: key check, authorization, replay,
// state check, provider effect, receipt.
func (l *Lifecycle) do(ctx context.Context, tr Transition, sprite, key string, d *AuthDecision, from, to LifecycleState, effect func(context.Context) error) (LifecycleReceipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.doLocked(ctx, tr, sprite, key, d, from, to, effect)
}

func (l *Lifecycle) doLocked(ctx context.Context, tr Transition, sprite, key string, d *AuthDecision, from, to LifecycleState, effect func(context.Context) error) (LifecycleReceipt, error) {
	if key == "" {
		return LifecycleReceipt{}, &RefusalError{Reason: ReasonKeyMissing,
			Detail: "every lifecycle transition needs a per-intention idempotency key"}
	}
	if sprite == "" {
		return LifecycleReceipt{}, &RefusalError{Reason: ReasonSpriteMissing,
			Detail: "transition targets no sprite"}
	}
	now := l.clock.Now().UTC()
	if err := l.auth.Authorize(ctx, AuthRequest{Transition: tr, Sprite: sprite, Decision: d, Now: now}); err != nil {
		if _, ok := err.(*RefusalError); ok {
			return LifecycleReceipt{}, err
		}
		// A malfunctioning gate fails closed, never open.
		return LifecycleReceipt{}, &RefusalError{Reason: ReasonAuthMissing,
			Detail: fmt.Sprintf("authorization gate malfunction: %v", err)}
	}
	if prev, ok := l.replayed[key]; ok {
		if prev.Transition != tr || prev.Sprite != sprite {
			return LifecycleReceipt{}, &RefusalError{Reason: ReasonReplayConflict,
				Detail: fmt.Sprintf("key already used for %s on %q; mint a fresh key per intention", prev.Transition, prev.Sprite)}
		}
		if l.stateLocked(sprite) != prev.To {
			return LifecycleReceipt{}, &RefusalError{Reason: ReasonReplayStale,
				Detail: fmt.Sprintf("key recorded %s → %s but %q is now %s; mint a fresh key", prev.From, prev.To, sprite, l.stateLocked(sprite))}
		}
		return prev, nil
	}
	// Destroy passes its pre-state through; fixed-edge transitions pin it.
	if cur := l.stateLocked(sprite); cur != from {
		return LifecycleReceipt{}, &RefusalError{Reason: ReasonIllegalTransition,
			Detail: fmt.Sprintf("%s needs %s, %q is %s", tr, from, sprite, cur)}
	}
	start := l.clock.Now()
	if err := effect(ctx); err != nil {
		return LifecycleReceipt{}, fmt.Errorf("fly: lifecycle %s %q: %w", tr, sprite, err)
	}
	r := LifecycleReceipt{
		Key: key, Transition: tr, Sprite: sprite, From: from, To: to,
		At: now, Approver: d.Approver, AuthRef: d.Reference,
		Provider: l.provider.Name(), Latency: l.clock.Now().Sub(start),
	}
	l.states[sprite] = to
	l.replayed[key] = r
	l.rec.Record(r)
	return r, nil
}

// Reconcile re-derives the coarsest safe position from the provider: a
// sprite the provider cannot see is absent; a sprite the controller has
// never seen but the provider holds is adopted as awake. Adoption is a
// documented heuristic — power and session position are unknowable after
// a restart without the durable store (B2-01B) — and a wrong adoption
// fails loudly at the provider, never silently: subsequent transitions
// still require fresh decisions and the provider enforces its own
// consistency. Reconcile emits no receipt; it only repairs the cache.
func (l *Lifecycle) Reconcile(ctx context.Context, sprite string) (LifecycleState, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	exists, err := l.provider.Describe(ctx, sprite)
	if err != nil {
		return "", fmt.Errorf("fly: lifecycle reconcile %q: %w", sprite, err)
	}
	if !exists {
		l.states[sprite] = StateAbsent
		return StateAbsent, nil
	}
	if _, seen := l.states[sprite]; !seen {
		l.states[sprite] = StateAwake
		return StateAwake, nil
	}
	return l.states[sprite], nil
}
