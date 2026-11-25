package fly

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// FakeSpriteManager is an in-memory SpriteManager for unit tests. State is
// guarded for concurrent use. Fail maps a method name to an injected error.
type FakeSpriteManager struct {
	mu       sync.Mutex
	Sprites  map[string]Sprite
	Services map[string]ServiceState
	// ServiceReqs records the request of each CreateService call.
	ServiceReqs map[string]ServiceRequest
	// Deleted records sprite/service deletions in order.
	Deleted []string
	// Created records service creations in order.
	Created []string
	Files   map[string][]byte
	Execs   [][]string
	Fail    map[string]error
	// ExecOutputs scripts Exec results by sprite + command; missing keys
	// succeed empty (existing behavior preserved).
	ExecOutputs map[string]ExecResult
	// CheckpointIDs records created checkpoint IDs per sprite.
	CheckpointIDs map[string][]string
	Policies      map[string]NetworkPolicy
	// URLBase formats fake Fly-issued URLs deterministically.
	URLBase string
}

// NewFakeSpriteManager returns an empty fake.
func NewFakeSpriteManager() *FakeSpriteManager {
	return &FakeSpriteManager{
		Sprites: map[string]Sprite{}, Services: map[string]ServiceState{},
		ServiceReqs: map[string]ServiceRequest{},
		Files:       map[string][]byte{}, Fail: map[string]error{},
		CheckpointIDs: map[string][]string{}, Policies: map[string]NetworkPolicy{},
		ExecOutputs: map[string]ExecResult{},
		URLBase:     "https://sprites.test",
	}
}

func (f *FakeSpriteManager) fail(op string) error {
	if err, ok := f.Fail[op]; ok {
		return err
	}
	return nil
}

// CreateSprite records one sprite with a deterministic fake URL.
func (f *FakeSpriteManager) CreateSprite(_ context.Context, name string, opts SpriteOptions) (Sprite, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("CreateSprite"); err != nil {
		return Sprite{}, err
	}
	if _, dup := f.Sprites[name]; dup {
		return Sprite{}, fmt.Errorf("fly: fake sprite %q already exists", name)
	}
	s := Sprite{ID: "spr_" + name, Name: name, Status: "running",
		URL: f.URLBase + "/" + name, Labels: append([]string(nil), opts.Labels...),
		CreatedAt: time.Now().UTC()}
	f.Sprites[name] = s
	return s, nil
}

// GetSprite returns one recorded sprite.
func (f *FakeSpriteManager) GetSprite(_ context.Context, name string) (Sprite, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("GetSprite"); err != nil {
		return Sprite{}, err
	}
	s, ok := f.Sprites[name]
	if !ok {
		return Sprite{}, &NotFoundError{Resource: "sprite " + name}
	}
	return s, nil
}

// DeleteSprite removes one recorded sprite.
func (f *FakeSpriteManager) DeleteSprite(_ context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("DeleteSprite"); err != nil {
		return err
	}
	if _, ok := f.Sprites[name]; !ok {
		return &NotFoundError{Resource: "sprite " + name}
	}
	delete(f.Sprites, name)
	return nil
}

// UpdateURLSettings records URL settings on one sprite.
func (f *FakeSpriteManager) UpdateURLSettings(_ context.Context, name string, settings URLSettings) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("UpdateURLSettings"); err != nil {
		return err
	}
	s, ok := f.Sprites[name]
	if !ok {
		return &NotFoundError{Resource: "sprite " + name}
	}
	s.Auth, s.PrivateAccess = settings.Auth, settings.PrivateAccess
	f.Sprites[name] = s
	return nil
}

// CreateService records one running service.
func (f *FakeSpriteManager) CreateService(_ context.Context, sprite, service string, req ServiceRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("CreateService"); err != nil {
		return err
	}
	if _, ok := f.Sprites[sprite]; !ok {
		return fmt.Errorf("fly: fake sprite %q not found", sprite)
	}
	f.Services[sprite+"/"+service] = ServiceState{Name: service, Status: "running", PID: 100}
	f.ServiceReqs[sprite+"/"+service] = req
	f.Created = append(f.Created, sprite+"/"+service)
	return nil
}

// GetService returns one recorded service.
func (f *FakeSpriteManager) GetService(_ context.Context, sprite, service string) (ServiceState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("GetService"); err != nil {
		return ServiceState{}, err
	}
	s, ok := f.Services[sprite+"/"+service]
	if !ok {
		return ServiceState{}, &NotFoundError{Resource: "service " + service}
	}
	return s, nil
}

// DeleteService removes one recorded service.
func (f *FakeSpriteManager) DeleteService(_ context.Context, sprite, service string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("DeleteService"); err != nil {
		return err
	}
	f.Deleted = append(f.Deleted, sprite+"/"+service)
	delete(f.Services, sprite+"/"+service)
	return nil
}

// Exec records the command and returns the scripted output for it, or
// empty success when nothing is scripted. Scripts key on the joined
// command line; production code must never depend on script keys.
func (f *FakeSpriteManager) scriptedExec(sprite string, cmd []string) (ExecResult, bool) {
	res, ok := f.ExecOutputs[sprite+"\x00"+strings.Join(cmd, "\x00")]
	return res, ok
}

// Exec records the command and succeeds with empty output.
func (f *FakeSpriteManager) Exec(_ context.Context, sprite string, cmd []string) (ExecResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("Exec"); err != nil {
		return ExecResult{}, err
	}
	if _, ok := f.Sprites[sprite]; !ok {
		return ExecResult{}, fmt.Errorf("fly: fake sprite %q not found", sprite)
	}
	f.Execs = append(f.Execs, append([]string{sprite}, cmd...))
	if res, ok := f.scriptedExec(sprite, cmd); ok {
		return res, nil
	}
	return ExecResult{}, nil
}

// WriteFile records one file's content.
func (f *FakeSpriteManager) WriteFile(_ context.Context, sprite, path string, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("WriteFile"); err != nil {
		return err
	}
	if _, ok := f.Sprites[sprite]; !ok {
		return fmt.Errorf("fly: fake sprite %q not found", sprite)
	}
	f.Files[sprite+":"+path] = append([]byte(nil), data...)
	return nil
}

// CreateCheckpoint records one checkpoint ID.
func (f *FakeSpriteManager) CreateCheckpoint(_ context.Context, sprite, comment string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("CreateCheckpoint"); err != nil {
		return "", err
	}
	if _, ok := f.Sprites[sprite]; !ok {
		return "", fmt.Errorf("fly: fake sprite %q not found", sprite)
	}
	id := fmt.Sprintf("chk_%d", len(f.CheckpointIDs[sprite])+1)
	f.CheckpointIDs[sprite] = append(f.CheckpointIDs[sprite], id)
	return id, nil
}

// RestoreCheckpoint verifies one recorded checkpoint ID exists.
func (f *FakeSpriteManager) RestoreCheckpoint(_ context.Context, sprite, checkpointID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("RestoreCheckpoint"); err != nil {
		return err
	}
	for _, id := range f.CheckpointIDs[sprite] {
		if id == checkpointID {
			return nil
		}
	}
	return &NotFoundError{Resource: "checkpoint " + checkpointID}
}

// GetNetworkPolicy returns one recorded policy.
func (f *FakeSpriteManager) GetNetworkPolicy(_ context.Context, sprite string) (NetworkPolicy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("GetNetworkPolicy"); err != nil {
		return NetworkPolicy{}, err
	}
	return f.Policies[sprite], nil
}

// UpdateNetworkPolicy records one policy.
func (f *FakeSpriteManager) UpdateNetworkPolicy(_ context.Context, sprite string, policy NetworkPolicy) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("UpdateNetworkPolicy"); err != nil {
		return err
	}
	f.Policies[sprite] = policy
	return nil
}

// FakeConnectorManager is an in-memory ConnectorManager for unit tests.
type FakeConnectorManager struct {
	mu          sync.Mutex
	Connections []Connection
	Created     []CustomAPIRequest
	Tested      []CustomAPITestRequest
	Updated     map[string]CustomAPIUpdateRequest
	TestOutcome CustomAPITestResult
	Fail        map[string]error
}

// NewFakeConnectorManager returns an empty fake.
func NewFakeConnectorManager() *FakeConnectorManager {
	return &FakeConnectorManager{Updated: map[string]CustomAPIUpdateRequest{}, Fail: map[string]error{}}
}

func (f *FakeConnectorManager) fail(op string) error {
	if err, ok := f.Fail[op]; ok {
		return err
	}
	return nil
}

// ListConnections returns configured connections, filtered by provider.
func (f *FakeConnectorManager) ListConnections(_ context.Context, provider string) ([]Connection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("ListConnections"); err != nil {
		return nil, err
	}
	if provider == "" {
		return append([]Connection(nil), f.Connections...), nil
	}
	var out []Connection
	for _, c := range f.Connections {
		if c.Provider == provider {
			out = append(out, c)
		}
	}
	return out, nil
}

// CreateCustomAPI records the request and returns a sanitized connection:
// same-name creation updates in place, mirroring the real API.
func (f *FakeConnectorManager) CreateCustomAPI(_ context.Context, req CustomAPIRequest) (Connection, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("CreateCustomAPI"); err != nil {
		return Connection{}, err
	}
	f.Created = append(f.Created, req)
	conn := Connection{ID: "conn_" + req.Name, Provider: "custom_api", ConnectionType: "api_key"}
	if req.AccessPolicy != nil {
		conn.AccessPolicy = *req.AccessPolicy
	}
	for i, c := range f.Connections {
		if c.ID == conn.ID {
			f.Connections[i] = conn
			return conn, nil
		}
	}
	f.Connections = append(f.Connections, conn)
	return conn, nil
}

// TestCustomAPI records the request and returns the scripted outcome.
func (f *FakeConnectorManager) TestCustomAPI(_ context.Context, req CustomAPITestRequest) (CustomAPITestResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("TestCustomAPI"); err != nil {
		return CustomAPITestResult{}, err
	}
	f.Tested = append(f.Tested, req)
	out := f.TestOutcome
	if out.Status == "" {
		out.Status = "ok"
	}
	return out, nil
}

// UpdateCustomAPI records one update.
func (f *FakeConnectorManager) UpdateCustomAPI(_ context.Context, id string, req CustomAPIUpdateRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fail("UpdateCustomAPI"); err != nil {
		return err
	}
	f.Updated[id] = req
	return nil
}
