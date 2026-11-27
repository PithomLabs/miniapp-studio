package fly

// Shared lifecycle contract suite (B2-01A). Every LifecycleProvider — the
// deterministic fake and the SDK adapter — must satisfy the same behavioral
// contract: authorized transitions execute exactly once per idempotency key
// and emit receipts; everything else fails closed with a named reason.
// Backends declare their verb capabilities; unsupported verbs must refuse
// with *UnsupportedError, never silently succeed.

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

var contractTime = time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)

type contractClock struct{ at time.Time }

func (c *contractClock) Now() time.Time { return c.at }

// contractCaps declares which lifecycle verbs a backend proves locally.
// Fully capable backends run the happy-path cycles. sessionLiveOnly
// covers verbs that are wired but need the live control plane (adapter
// Tasks-API exec): the suite pins the wiring attempt instead of the
// outcome. Anything else outside the set must refuse with
// *UnsupportedError, never silently succeed.
type contractCaps struct {
	power           bool // wake/sleep via SetPower
	session         bool // connect/disconnect via Attach/Detach
	sessionLiveOnly bool // session verbs wired; harness lacks a sprite
}

func contractDecision(tr Transition, sprite string) *AuthDecision {
	return &AuthDecision{
		Approver: "owner", Reference: "apr-1",
		Transition: tr, Sprite: sprite,
		ExpiresAt: contractTime.Add(time.Hour),
	}
}

// runLifecycleContract executes the shared suite against fresh backends from
// mk. It is the adapter/fake parity proof: identical expectations, both
// providers, modulo declared capabilities.
func runLifecycleContract(t *testing.T, mk func() LifecycleProvider, caps contractCaps) {
	t.Helper()
	fresh := func() (*Lifecycle, *MemoryRecorder) {
		rec := &MemoryRecorder{}
		lc := NewLifecycle(mk(), DecisionAuthorizer{}, rec, &contractClock{at: contractTime})
		return lc, rec
	}
	mustRefuse := func(t *testing.T, err error, reason string) {
		t.Helper()
		var rej *RefusalError
		if !errors.As(err, &rej) {
			t.Fatalf("err = %v (%T), want *RefusalError %q", err, err, reason)
		}
		if rej.Reason != reason {
			t.Fatalf("reason = %q, want %q (detail %q)", rej.Reason, reason, rej.Detail)
		}
	}

	t.Run("ProvisionHappy", func(t *testing.T) {
		lc, rec := fresh()
		r, err := lc.Provision(context.Background(), "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{})
		if err != nil {
			t.Fatalf("provision: %v", err)
		}
		if r.From != StateAbsent || r.To != StateAwake || r.Transition != TransitionProvision {
			t.Errorf("receipt = %+v", r)
		}
		if r.Key != "k1" || r.Sprite != "b1" || r.Approver != "owner" || r.AuthRef != "apr-1" {
			t.Errorf("receipt identity/scope = %+v", r)
		}
		if !r.At.Equal(contractTime) {
			t.Errorf("At = %v, want contract time", r.At)
		}
		if got := rec.All(); len(got) != 1 || !reflect.DeepEqual(got[0], r) {
			t.Errorf("recorded = %+v, want the returned receipt", got)
		}
		if st := lc.State("b1"); st != StateAwake {
			t.Errorf("state = %q, want awake", st)
		}
	})

	t.Run("ProvisionDuplicateIllegal", func(t *testing.T) {
		lc, _ := fresh()
		ctx := context.Background()
		if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
			t.Fatalf("first provision: %v", err)
		}
		_, err := lc.Provision(ctx, "b1", "k2", contractDecision(TransitionProvision, "b1"), SpriteOptions{})
		mustRefuse(t, err, ReasonIllegalTransition)
	})

	t.Run("DestroyHappy", func(t *testing.T) {
		lc, _ := fresh()
		ctx := context.Background()
		if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
			t.Fatal(err)
		}
		r, err := lc.Destroy(ctx, "b1", "k2", contractDecision(TransitionDestroy, "b1"))
		if err != nil {
			t.Fatalf("destroy: %v", err)
		}
		if r.From != StateAwake || r.To != StateAbsent {
			t.Errorf("receipt = %+v", r)
		}
		if st := lc.State("b1"); st != StateAbsent {
			t.Errorf("state = %q, want absent", st)
		}
	})

	t.Run("DestroyAbsentIllegal", func(t *testing.T) {
		lc, _ := fresh()
		_, err := lc.Destroy(context.Background(), "ghost", "k1", contractDecision(TransitionDestroy, "ghost"))
		mustRefuse(t, err, ReasonIllegalTransition)
	})

	t.Run("DestroyRequiresDestroyScope", func(t *testing.T) {
		lc, _ := fresh()
		ctx := context.Background()
		if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
			t.Fatal(err)
		}
		// A provision-scoped approval must not authorize destruction.
		_, err := lc.Destroy(ctx, "b1", "k2", contractDecision(TransitionProvision, "b1"))
		mustRefuse(t, err, ReasonAuthTransitionMismatch)
	})

	t.Run("AuthMissing", func(t *testing.T) {
		lc, _ := fresh()
		_, err := lc.Provision(context.Background(), "b1", "k1", nil, SpriteOptions{})
		mustRefuse(t, err, ReasonAuthMissing)
	})

	t.Run("AuthExpired", func(t *testing.T) {
		lc, _ := fresh()
		d := contractDecision(TransitionProvision, "b1")
		d.ExpiresAt = contractTime.Add(-time.Minute)
		_, err := lc.Provision(context.Background(), "b1", "k1", d, SpriteOptions{})
		mustRefuse(t, err, ReasonAuthExpired)
	})

	t.Run("AuthNoExpiry", func(t *testing.T) {
		lc, _ := fresh()
		d := contractDecision(TransitionProvision, "b1")
		d.ExpiresAt = time.Time{}
		_, err := lc.Provision(context.Background(), "b1", "k1", d, SpriteOptions{})
		mustRefuse(t, err, ReasonAuthNoExpiry)
	})

	t.Run("AuthSpriteMismatch", func(t *testing.T) {
		lc, _ := fresh()
		_, err := lc.Provision(context.Background(), "b1", "k1", contractDecision(TransitionProvision, "other"), SpriteOptions{})
		mustRefuse(t, err, ReasonAuthSpriteMismatch)
	})

	t.Run("AuthApproverMissing", func(t *testing.T) {
		lc, _ := fresh()
		d := contractDecision(TransitionProvision, "b1")
		d.Approver = ""
		_, err := lc.Provision(context.Background(), "b1", "k1", d, SpriteOptions{})
		mustRefuse(t, err, ReasonAuthApproverMissing)
	})

	t.Run("AuthReferenceMissing", func(t *testing.T) {
		lc, _ := fresh()
		d := contractDecision(TransitionProvision, "b1")
		d.Reference = ""
		_, err := lc.Provision(context.Background(), "b1", "k1", d, SpriteOptions{})
		mustRefuse(t, err, ReasonAuthReferenceMissing)
	})

	t.Run("KeyMissing", func(t *testing.T) {
		lc, _ := fresh()
		_, err := lc.Provision(context.Background(), "b1", "", contractDecision(TransitionProvision, "b1"), SpriteOptions{})
		mustRefuse(t, err, ReasonKeyMissing)
	})

	t.Run("SpriteMissing", func(t *testing.T) {
		lc, _ := fresh()
		_, err := lc.Provision(context.Background(), "", "k1", contractDecision(TransitionProvision, ""), SpriteOptions{})
		mustRefuse(t, err, ReasonSpriteMissing)
	})

	t.Run("ReplaySameScope", func(t *testing.T) {
		p := mk()
		rec := &MemoryRecorder{}
		lc := NewLifecycle(p, DecisionAuthorizer{}, rec, &contractClock{at: contractTime})
		ctx := context.Background()
		first, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{})
		if err != nil {
			t.Fatal(err)
		}
		second, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{})
		if err != nil {
			t.Fatalf("replay: %v", err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Errorf("replay receipt differs:\nfirst  %+v\nsecond %+v", first, second)
		}
		if n := countCalls(p, "Provision"); n != 1 {
			t.Errorf("provider Provision calls = %d, want 1", n)
		}
		if got := rec.All(); len(got) != 1 {
			t.Errorf("recorded receipts = %d, want 1", len(got))
		}
	})

	t.Run("ReplayConflict", func(t *testing.T) {
		lc, _ := fresh()
		ctx := context.Background()
		if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
			t.Fatal(err)
		}
		// Same key, different sprite: a new intention needs a new key.
		_, err := lc.Provision(ctx, "b2", "k1", contractDecision(TransitionProvision, "b2"), SpriteOptions{})
		mustRefuse(t, err, ReasonReplayConflict)
	})

	t.Run("ReplayStale", func(t *testing.T) {
		lc, _ := fresh()
		ctx := context.Background()
		if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
			t.Fatal(err)
		}
		if _, err := lc.Destroy(ctx, "b1", "k2", contractDecision(TransitionDestroy, "b1")); err != nil {
			t.Fatal(err)
		}
		// Same key and scope, but the recorded outcome no longer holds:
		// returning the old receipt would be a lie.
		_, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{})
		mustRefuse(t, err, ReasonReplayStale)
	})

	t.Run("ConcurrentSameKey", func(t *testing.T) {
		p := mk()
		lc := NewLifecycle(p, DecisionAuthorizer{}, &MemoryRecorder{}, &contractClock{at: contractTime})
		const n = 8
		var wg sync.WaitGroup
		errs := make([]error, n)
		out := make([]LifecycleReceipt, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				out[i], errs[i] = lc.Provision(context.Background(), "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{})
			}(i)
		}
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Fatalf("worker %d: %v", i, err)
			}
			if !reflect.DeepEqual(out[i], out[0]) {
				t.Fatalf("worker %d receipt differs: %+v vs %+v", i, out[i], out[0])
			}
		}
		if n := countCalls(p, "Provision"); n != 1 {
			t.Errorf("provider Provision calls = %d, want exactly 1", n)
		}
	})

	t.Run("ConcurrentDistinct", func(t *testing.T) {
		lc, _ := fresh()
		const n = 8
		var wg sync.WaitGroup
		errs := make([]error, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				name := string(rune('a' + i))
				_, errs[i] = lc.Provision(context.Background(), name, "k-"+name, contractDecision(TransitionProvision, name), SpriteOptions{})
			}(i)
		}
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Errorf("worker %d: %v", i, err)
			}
		}
	})

	t.Run("DeterministicReceipts", func(t *testing.T) {
		script := func() []LifecycleReceipt {
			lc := NewLifecycle(mk(), DecisionAuthorizer{}, &MemoryRecorder{}, &contractClock{at: contractTime})
			ctx := context.Background()
			var out []LifecycleReceipt
			r1, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{})
			if err != nil {
				t.Fatal(err)
			}
			r2, err := lc.Destroy(ctx, "b1", "k2", contractDecision(TransitionDestroy, "b1"))
			if err != nil {
				t.Fatal(err)
			}
			return append(out, r1, r2)
		}
		if a, b := script(), script(); !reflect.DeepEqual(a, b) {
			t.Errorf("runs differ:\n%+v\n%+v", a, b)
		}
	})

	t.Run("ReconcileAbsent", func(t *testing.T) {
		lc, _ := fresh()
		st, err := lc.Reconcile(context.Background(), "ghost")
		if err != nil {
			t.Fatal(err)
		}
		if st != StateAbsent || lc.State("ghost") != StateAbsent {
			t.Errorf("state = %q", st)
		}
	})

	t.Run("ReconcileAdopt", func(t *testing.T) {
		p := mk()
		lc := NewLifecycle(p, DecisionAuthorizer{}, &MemoryRecorder{}, &contractClock{at: contractTime})
		ctx := context.Background()
		// Simulate a restart: the provider holds a sprite the fresh
		// controller has never seen.
		if err := p.Provision(ctx, "orphan", SpriteOptions{}); err != nil {
			t.Fatal(err)
		}
		st, err := lc.Reconcile(ctx, "orphan")
		if err != nil {
			t.Fatal(err)
		}
		if st != StateAwake {
			t.Errorf("adopted = %q, want awake", st)
		}
	})

	t.Run("ReconcileKeep", func(t *testing.T) {
		lc, _ := fresh()
		ctx := context.Background()
		if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
			t.Fatal(err)
		}
		st, err := lc.Reconcile(ctx, "b1")
		if err != nil {
			t.Fatal(err)
		}
		if st != StateAwake {
			t.Errorf("state = %q, want kept awake", st)
		}
	})

	if caps.power {
		t.Run("PowerCycle", func(t *testing.T) {
			lc, _ := fresh()
			ctx := context.Background()
			if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
				t.Fatal(err)
			}
			r, err := lc.Sleep(ctx, "b1", "k2", contractDecision(TransitionSleep, "b1"))
			if err != nil {
				t.Fatalf("sleep: %v", err)
			}
			if r.From != StateAwake || r.To != StateAsleep {
				t.Errorf("sleep receipt = %+v", r)
			}
			if _, err := lc.Sleep(ctx, "b1", "k3", contractDecision(TransitionSleep, "b1")); err == nil {
				t.Error("second sleep accepted, want illegal-transition")
			} else {
				mustRefuse(t, err, ReasonIllegalTransition)
			}
			r, err = lc.Wake(ctx, "b1", "k4", contractDecision(TransitionWake, "b1"))
			if err != nil {
				t.Fatalf("wake: %v", err)
			}
			if r.From != StateAsleep || r.To != StateAwake {
				t.Errorf("wake receipt = %+v", r)
			}
			if _, err := lc.Wake(ctx, "b1", "k5", contractDecision(TransitionWake, "b1")); err == nil {
				t.Error("wake-from-awake accepted, want illegal-transition")
			} else {
				mustRefuse(t, err, ReasonIllegalTransition)
			}
		})

		t.Run("PowerAbsentIllegal", func(t *testing.T) {
			lc, _ := fresh()
			ctx := context.Background()
			_, err := lc.Wake(ctx, "ghost", "k1", contractDecision(TransitionWake, "ghost"))
			mustRefuse(t, err, ReasonIllegalTransition)
			_, err = lc.Sleep(ctx, "ghost", "k2", contractDecision(TransitionSleep, "ghost"))
			mustRefuse(t, err, ReasonIllegalTransition)
		})
	} else {
		t.Run("PowerUnsupported", func(t *testing.T) {
			lc, _ := fresh()
			ctx := context.Background()
			if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
				t.Fatal(err)
			}
			var unsup *UnsupportedError
			_, err := lc.Sleep(ctx, "b1", "k2", contractDecision(TransitionSleep, "b1"))
			if !errors.As(err, &unsup) || unsup.Op != "SetPower" {
				t.Errorf("sleep err = %v, want *UnsupportedError for SetPower", err)
			}
		})
	}

	if caps.session {
		t.Run("SessionCycle", func(t *testing.T) {
			lc, _ := fresh()
			ctx := context.Background()
			if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
				t.Fatal(err)
			}
			r, err := lc.Connect(ctx, "b1", "k2", contractDecision(TransitionConnect, "b1"))
			if err != nil {
				t.Fatalf("connect: %v", err)
			}
			if r.From != StateAwake || r.To != StateConnected {
				t.Errorf("connect receipt = %+v", r)
			}
			if _, err := lc.Connect(ctx, "b1", "k3", contractDecision(TransitionConnect, "b1")); err == nil {
				t.Error("second connect accepted, want illegal-transition")
			} else {
				mustRefuse(t, err, ReasonIllegalTransition)
			}
			// Sleep must not silently drop an attached session.
			if _, err := lc.Sleep(ctx, "b1", "k4", contractDecision(TransitionSleep, "b1")); err == nil {
				t.Error("sleep-from-connected accepted, want illegal-transition")
			} else {
				mustRefuse(t, err, ReasonIllegalTransition)
			}
			r, err = lc.Disconnect(ctx, "b1", "k5", contractDecision(TransitionDisconnect, "b1"))
			if err != nil {
				t.Fatalf("disconnect: %v", err)
			}
			if r.From != StateConnected || r.To != StateAwake {
				t.Errorf("disconnect receipt = %+v", r)
			}
			// Destroy wins over an attached session.
			if _, err := lc.Connect(ctx, "b1", "k6", contractDecision(TransitionConnect, "b1")); err != nil {
				t.Fatal(err)
			}
			r, err = lc.Destroy(ctx, "b1", "k7", contractDecision(TransitionDestroy, "b1"))
			if err != nil {
				t.Fatalf("destroy-from-connected: %v", err)
			}
			if r.From != StateConnected || r.To != StateAbsent {
				t.Errorf("destroy receipt = %+v", r)
			}
		})

		t.Run("SessionAbsentIllegal", func(t *testing.T) {
			lc, _ := fresh()
			ctx := context.Background()
			_, err := lc.Connect(ctx, "ghost", "k1", contractDecision(TransitionConnect, "ghost"))
			mustRefuse(t, err, ReasonIllegalTransition)
			_, err = lc.Disconnect(ctx, "ghost", "k2", contractDecision(TransitionDisconnect, "ghost"))
			mustRefuse(t, err, ReasonIllegalTransition)
		})
	} else if caps.sessionLiveOnly {
		t.Run("SessionLiveOnly", func(t *testing.T) {
			lc, rec := fresh()
			ctx := context.Background()
			if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
				t.Fatal(err)
			}
			// The verb is wired to exec, so against the harness it
			// must fail on transport — not on refusal, and not as
			// unsupported. The argv it would execute is pinned by
			// TestTasksHoldArgs; the live proof runs the outcome.
			_, err := lc.Connect(ctx, "b1", "k2", contractDecision(TransitionConnect, "b1"))
			if err == nil {
				t.Fatal("connect succeeded without a sprite, want transport failure")
			}
			var rej *RefusalError
			var unsup *UnsupportedError
			if errors.As(err, &rej) {
				t.Errorf("connect refused (%s): the gate must pass; only transport may fail", rej.Reason)
			}
			if errors.As(err, &unsup) {
				t.Errorf("connect unsupported: the verb is wired, want a transport error, got %v", err)
			}
			if st := lc.State("b1"); st != StateAwake {
				t.Errorf("state = %q, want kept awake", st)
			}
			if got := rec.All(); len(got) != 1 {
				t.Errorf("recorded %d receipts, want only the provision", len(got))
			}
		})
	} else {
		t.Run("SessionUnsupported", func(t *testing.T) {
			lc, _ := fresh()
			ctx := context.Background()
			if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
				t.Fatal(err)
			}
			var unsup *UnsupportedError
			_, err := lc.Connect(ctx, "b1", "k2", contractDecision(TransitionConnect, "b1"))
			if !errors.As(err, &unsup) || unsup.Op != "Attach" {
				t.Errorf("connect err = %v, want *UnsupportedError for Attach", err)
			}
		})
	}
}

// countCalls reports how many times op was invoked on p. Backends expose
// call counts for exactly-once assertions: the fake via CallCounts, the
// parity double via its request log (see lifecycle_parity_test.go).
func countCalls(p LifecycleProvider, op string) int {
	if c, ok := p.(CallCounter); ok {
		return c.CallCount(op)
	}
	return -1
}

// CallCounter is implemented by test backends that record invocations.
type CallCounter interface {
	CallCount(op string) int
}

// TestLifecycleContractFake runs the full shared suite against the
// deterministic fake: the behavior half of the parity proof.
func TestLifecycleContractFake(t *testing.T) {
	runLifecycleContract(t, func() LifecycleProvider {
		return NewFakeLifecycleProvider()
	}, contractCaps{power: true, session: true})
}
