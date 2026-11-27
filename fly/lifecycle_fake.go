package fly

import (
	"context"
	"fmt"
	"sync"
)

// FakeLifecycleProvider is a deterministic in-memory LifecycleProvider.
// It enforces its own consistency (power and session position) so that a
// controller/provider divergence fails loudly here instead of corrupting
// state silently. Fail maps a verb name to an injected error; CallCount
// exposes invocation counts for exactly-once assertions.
type FakeLifecycleProvider struct {
	mu    sync.Mutex
	state map[string]*fakeLifecycleSprite
	Fail  map[string]error
	Calls map[string]int
	Opts  map[string]SpriteOptions
}

type fakeLifecycleSprite struct {
	awake    bool
	attached bool
}

// NewFakeLifecycleProvider returns an empty fake.
func NewFakeLifecycleProvider() *FakeLifecycleProvider {
	return &FakeLifecycleProvider{Fail: map[string]error{}, Calls: map[string]int{}, Opts: map[string]SpriteOptions{}}
}

// Name identifies the backend on receipts.
func (f *FakeLifecycleProvider) Name() string { return "fake" }

// CallCount reports invocations of one verb.
func (f *FakeLifecycleProvider) CallCount(op string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Calls[op]
}

func (f *FakeLifecycleProvider) fail(op string) error {
	if err, ok := f.Fail[op]; ok {
		return err
	}
	return nil
}

func (f *FakeLifecycleProvider) sprites() map[string]*fakeLifecycleSprite {
	if f.state == nil {
		f.state = map[string]*fakeLifecycleSprite{}
	}
	return f.state
}

// Provision records one awake sprite.
func (f *FakeLifecycleProvider) Provision(_ context.Context, name string, opts SpriteOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls["Provision"]++
	if err := f.fail("Provision"); err != nil {
		return err
	}
	ss := f.sprites()
	if _, dup := ss[name]; dup {
		return fmt.Errorf("fly: fake sprite %q already exists", name)
	}
	ss[name] = &fakeLifecycleSprite{awake: true}
	f.Opts[name] = opts
	return nil
}

// Destroy removes one sprite, attached or not.
func (f *FakeLifecycleProvider) Destroy(_ context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls["Destroy"]++
	if err := f.fail("Destroy"); err != nil {
		return err
	}
	ss := f.sprites()
	if _, ok := ss[name]; !ok {
		return &NotFoundError{Resource: "sprite " + name}
	}
	delete(ss, name)
	delete(f.Opts, name)
	return nil
}

// Describe reports bare existence; unknown sprites are absent, not errors.
func (f *FakeLifecycleProvider) Describe(_ context.Context, name string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls["Describe"]++
	if err := f.fail("Describe"); err != nil {
		return false, err
	}
	_, ok := f.sprites()[name]
	return ok, nil
}

// SetPower flips one sprite's power position.
func (f *FakeLifecycleProvider) SetPower(_ context.Context, name string, awake bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls["SetPower"]++
	if err := f.fail("SetPower"); err != nil {
		return err
	}
	s, ok := f.sprites()[name]
	if !ok {
		return &NotFoundError{Resource: "sprite " + name}
	}
	if !awake && s.attached {
		return fmt.Errorf("fly: fake sprite %q still attached", name)
	}
	s.awake = awake
	return nil
}

// Attach connects one awake sprite's management session.
func (f *FakeLifecycleProvider) Attach(_ context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls["Attach"]++
	if err := f.fail("Attach"); err != nil {
		return err
	}
	s, ok := f.sprites()[name]
	if !ok {
		return &NotFoundError{Resource: "sprite " + name}
	}
	if !s.awake {
		return fmt.Errorf("fly: fake sprite %q asleep", name)
	}
	if s.attached {
		return fmt.Errorf("fly: fake sprite %q already attached", name)
	}
	s.attached = true
	return nil
}

// Detach disconnects one sprite's management session.
func (f *FakeLifecycleProvider) Detach(_ context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls["Detach"]++
	if err := f.fail("Detach"); err != nil {
		return err
	}
	s, ok := f.sprites()[name]
	if !ok {
		return &NotFoundError{Resource: "sprite " + name}
	}
	if !s.attached {
		return fmt.Errorf("fly: fake sprite %q not attached", name)
	}
	s.attached = false
	return nil
}
