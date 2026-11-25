package connector

import (
	"context"
	"fmt"
)

// Connection is a resolved connector reference. It carries identity and
// provider metadata only — never credentials, tokens, or secret-bearing
// URLs. Credentials stay in the server-side secret store the P3
// implementation integrates with.
type Connection struct {
	// ID is the opaque logical identifier from the spec.
	ID string
	// Provider names the connector provider for diagnostics and policy.
	Provider string
}

// Registry resolves connection IDs to server-side connections.
// Implementations must reject unknown IDs with an error wrapping
// ErrUnknownConnection. Implementations must never return credentials.
type Registry interface {
	Resolve(ctx context.Context, id string) (Connection, error)
}

// ErrUnknownConnection reports a connection ID with no server-side record.
type ErrUnknownConnection struct {
	ID string
}

// Error identifies the unknown connection without echoing secrets: IDs are
// opaque references by construction, so echoing the ID is safe.
func (e *ErrUnknownConnection) Error() string {
	return fmt.Sprintf("connector: unknown connection %q", e.ID)
}

// FakeRegistry is a fixed in-memory Registry for unit tests and dry-run.
// It resolves exactly the configured IDs and rejects everything else.
type FakeRegistry struct {
	conns map[string]Connection
}

// NewFakeRegistry returns a Registry resolving the given connections.
func NewFakeRegistry(conns ...Connection) *FakeRegistry {
	f := &FakeRegistry{conns: map[string]Connection{}}
	for _, c := range conns {
		f.conns[c.ID] = c
	}
	return f
}

// Resolve returns the recorded connection or an *ErrUnknownConnection.
func (f *FakeRegistry) Resolve(_ context.Context, id string) (Connection, error) {
	if c, ok := f.conns[id]; ok {
		return c, nil
	}
	return Connection{}, &ErrUnknownConnection{ID: id}
}
