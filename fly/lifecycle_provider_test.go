package fly

// Backend-specific lifecycle unit tests (B2-01A): fake-provider guards,
// constructor misuse, and gate-malfunction behavior. Shared behavior lives
// in the contract suite.

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestFakeLifecycleGuards(t *testing.T) {
	ctx := context.Background()
	p := NewFakeLifecycleProvider()
	var nf *NotFoundError

	if err := p.SetPower(ctx, "ghost", true); !errors.As(err, &nf) {
		t.Errorf("SetPower absent = %v, want *NotFoundError", err)
	}
	if err := p.Attach(ctx, "ghost"); !errors.As(err, &nf) {
		t.Errorf("Attach absent = %v, want *NotFoundError", err)
	}
	if err := p.Detach(ctx, "ghost"); !errors.As(err, &nf) {
		t.Errorf("Detach absent = %v, want *NotFoundError", err)
	}
	if err := p.Destroy(ctx, "ghost"); !errors.As(err, &nf) {
		t.Errorf("Destroy absent = %v, want *NotFoundError", err)
	}
	if exists, err := p.Describe(ctx, "ghost"); err != nil || exists {
		t.Errorf("Describe absent = %v, %v; want false, nil", exists, err)
	}

	if err := p.Provision(ctx, "s", SpriteOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := p.Provision(ctx, "s", SpriteOptions{}); err == nil {
		t.Error("duplicate provision accepted, want error")
	}
	if err := p.Detach(ctx, "s"); err == nil {
		t.Error("detach-while-detached accepted, want error")
	}
	if err := p.Attach(ctx, "s"); err != nil {
		t.Fatal(err)
	}
	if err := p.Attach(ctx, "s"); err == nil {
		t.Error("second attach accepted, want error")
	}
	if err := p.SetPower(ctx, "s", false); err == nil {
		t.Error("sleep-while-attached accepted, want error")
	}
	if err := p.Detach(ctx, "s"); err != nil {
		t.Fatal(err)
	}
	if err := p.SetPower(ctx, "s", false); err != nil {
		t.Fatal(err)
	}
	if err := p.Attach(ctx, "s"); err == nil {
		t.Error("attach-while-asleep accepted, want error")
	}
	if err := p.Destroy(ctx, "s"); err != nil {
		t.Fatal(err)
	}
	if n := p.CallCount("Provision"); n != 2 {
		t.Errorf("Provision calls = %d, want 2 (incl. duplicate)", n)
	}
}

func TestFakeLifecycleFailInjection(t *testing.T) {
	ctx := context.Background()
	p := NewFakeLifecycleProvider()
	boom := errors.New("boom")
	p.Fail["Provision"] = boom
	if err := p.Provision(ctx, "s", SpriteOptions{}); err != boom {
		t.Errorf("injected = %v, want boom", err)
	}
	p.Fail["Describe"] = boom
	if _, err := p.Describe(ctx, "s"); err != boom {
		t.Errorf("injected = %v, want boom", err)
	}
}

func TestLifecycleNilConstructionPanics(t *testing.T) {
	rec := &MemoryRecorder{}
	for name, fn := range map[string]func(){
		"nil provider":   func() { NewLifecycle(nil, DecisionAuthorizer{}, rec, nil) },
		"nil authorizer": func() { NewLifecycle(NewFakeLifecycleProvider(), nil, rec, nil) },
		"nil recorder":   func() { NewLifecycle(NewFakeLifecycleProvider(), DecisionAuthorizer{}, nil, nil) },
		"nil adapter":    func() { NewAdapterLifecycleProvider(nil) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: no panic", name)
				}
			}()
			fn()
		}()
	}
	// A nil clock selects the system clock instead of panicking.
	lc := NewLifecycle(NewFakeLifecycleProvider(), DecisionAuthorizer{}, rec, nil)
	if lc.State("x") != StateAbsent {
		t.Error("nil-clock controller misbehaves")
	}
}

type brokenAuthorizer struct{}

func (brokenAuthorizer) Authorize(_ context.Context, _ AuthRequest) error {
	return errors.New("gate exploded")
}

func TestLifecycleGateMalfunctionFailsClosed(t *testing.T) {
	lc := NewLifecycle(NewFakeLifecycleProvider(), brokenAuthorizer{}, &MemoryRecorder{}, &contractClock{at: contractTime})
	_, err := lc.Provision(context.Background(), "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{})
	var rej *RefusalError
	if !errors.As(err, &rej) || rej.Reason != ReasonAuthMissing {
		t.Fatalf("err = %v, want fail-closed refusal", err)
	}
	if st := lc.State("b1"); st != StateAbsent {
		t.Errorf("state = %q, want absent", st)
	}
}

func TestTasksHoldArgs(t *testing.T) {
	want := []string{"curl", "-s", "--unix-socket", "/.sprite/api.sock",
		"-H", "Content-Type: application/json", "-f", "-X", "POST",
		"http://sprite/v1/tasks",
		"-d", `{"name":"studio-mgmt","expire":"30m"}`}
	if got := tasksHoldArgs(); !reflect.DeepEqual(got, want) {
		t.Errorf("hold argv = %q, want %q", got, want)
	}
	wantRelease := []string{"curl", "-s", "--unix-socket", "/.sprite/api.sock",
		"-f", "-X", "DELETE", "http://sprite/v1/tasks/studio-mgmt"}
	if got := tasksReleaseArgs(); !reflect.DeepEqual(got, wantRelease) {
		t.Errorf("release argv = %q, want %q", got, wantRelease)
	}
}

func TestBoundLifecycleOutput(t *testing.T) {
	if got := boundLifecycleOutput("short"); got != "short" {
		t.Errorf("short = %q", got)
	}
	b := make([]rune, 2000)
	for i := range b {
		b[i] = 'x'
	}
	got := boundLifecycleOutput(string(b))
	if n := len([]rune(got)); n != 1000+len([]rune("…[truncated]")) {
		t.Errorf("bounded length = %d runes", n)
	}
}

func TestFakeLifecycleDesync(t *testing.T) {
	ctx := context.Background()
	p := NewFakeLifecycleProvider()
	lc := NewLifecycle(p, DecisionAuthorizer{}, &MemoryRecorder{}, &contractClock{at: contractTime})
	if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
		t.Fatal(err)
	}
	// Provider loses the sprite behind the controller's back.
	if err := p.Destroy(ctx, "b1"); err != nil {
		t.Fatal(err)
	}
	var nf *NotFoundError
	if _, err := lc.Destroy(ctx, "b1", "k2", contractDecision(TransitionDestroy, "b1")); !errors.As(err, &nf) {
		t.Fatalf("destroy err = %v, want *NotFoundError", err)
	}
	if st := lc.State("b1"); st != StateAwake {
		t.Errorf("state = %q, want kept awake", st)
	}
	if st, err := lc.Reconcile(ctx, "b1"); err != nil || st != StateAbsent {
		t.Errorf("reconciled = %q, %v; want absent, nil", st, err)
	}
}
