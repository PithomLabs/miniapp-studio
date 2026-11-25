package fly

import (
	"context"
	"time"
)

// SpriteManager provisions and operates Sprites: the builder-agent host and
// (pending the topology decision) generated-app targets. Every method takes
// a context; URLs returned are Fly-issued values to persist verbatim.
type SpriteManager interface {
	CreateSprite(ctx context.Context, name string, opts SpriteOptions) (Sprite, error)
	GetSprite(ctx context.Context, name string) (Sprite, error)
	DeleteSprite(ctx context.Context, name string) error
	UpdateURLSettings(ctx context.Context, name string, settings URLSettings) error
	CreateService(ctx context.Context, sprite, service string, req ServiceRequest) error
	GetService(ctx context.Context, sprite, service string) (ServiceState, error)
	DeleteService(ctx context.Context, sprite, service string) error
	Exec(ctx context.Context, sprite string, cmd []string) (ExecResult, error)
	WriteFile(ctx context.Context, sprite, path string, data []byte) error
	CreateCheckpoint(ctx context.Context, sprite, comment string) (string, error)
	RestoreCheckpoint(ctx context.Context, sprite, checkpointID string) error
	GetNetworkPolicy(ctx context.Context, sprite string) (NetworkPolicy, error)
	UpdateNetworkPolicy(ctx context.Context, sprite string, policy NetworkPolicy) error
}

// SpriteOptions carries creation options. Zero values select Fly defaults.
type SpriteOptions struct {
	Region string
	Labels []string
}

// Sprite is the owned subset of Sprites API sprite state.
type Sprite struct {
	ID            string
	Name          string
	Status        string
	URL           string
	Auth          string
	PrivateAccess string
	Labels        []string
	CreatedAt     time.Time
}

// URLSettings controls Sprite URL authentication.
type URLSettings struct {
	Auth          string
	PrivateAccess string
}

// URL authentication modes (Sprites CLI reference, verified live 2026-11-24):
// "sprite" requires org membership (browser Fly.io login or Bearer org token)
// and is the default; "public" opens the URL to the internet. Values pass
// through to the API unvalidated so future modes keep working.
const (
	URLAuthSprite = "sprite"
	URLAuthPublic = "public"
)

// ServiceRequest defines a Sprite service. HTTPPort exposes the service over
// the Sprite URL when set.
type ServiceRequest struct {
	Cmd      string
	Args     []string
	HTTPPort *int
}

// ServiceState reports a service's runtime state.
type ServiceState struct {
	Name   string
	Status string
	PID    int
	Error  string
}

// ExecResult carries one remote command outcome.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// NetworkPolicyRule allows or denies one domain pattern.
type NetworkPolicyRule struct {
	Domain string
	Action string
}

// NetworkPolicy is the Sprite egress policy.
type NetworkPolicy struct {
	Rules []NetworkPolicyRule
}

// ConnectorManager manages Sprites connector connections: the organization's
// configured connections and Custom API connectors. Full-access operations
// execute only in the trusted Studio backend; the manager never returns,
// logs, or persists credential values.
type ConnectorManager interface {
	// ListConnections returns connections configured for the authenticated
	// organization, optionally filtered by provider. It never returns the
	// public marketing catalog.
	ListConnections(ctx context.Context, provider string) ([]Connection, error)
	// CreateCustomAPI creates (or, on duplicate name, updates in place) a
	// Custom API connection. Creation does not test the credential.
	CreateCustomAPI(ctx context.Context, req CustomAPIRequest) (Connection, error)
	// TestCustomAPI sends one request without storing anything.
	TestCustomAPI(ctx context.Context, req CustomAPITestRequest) (CustomAPITestResult, error)
	// UpdateCustomAPI updates one Custom API connection by ID.
	UpdateCustomAPI(ctx context.Context, id string, req CustomAPIUpdateRequest) error
}

// Connection is the owned subset of a configured connector connection.
// It is sanitized by the API: secrets are never present.
type Connection struct {
	ID                string
	Provider          string
	ConnectionType    string
	AccessPolicy      AccessPolicy
	ProviderAccountID string
	Transport         string
	GatewaySnippet    string
}

// AccessPolicy is the least-privilege contract for one connection.
type AccessPolicy struct {
	AllowAll         bool
	AllowedEndpoints []string
	BlockedEndpoints []string
	NamePrefix       string
	SpriteLabels     []string
}

// AuthMethod values for Custom API credential placement.
const (
	AuthHeader       = "header"
	AuthCustomHeader = "custom_header"
	AuthQueryParam   = "query_param"
	AuthURLPath      = "url_path"
)

// CustomAPIRequest creates one Custom API connection.
type CustomAPIRequest struct {
	Name                string
	Description         string
	BaseAPIURL          string
	TestURL             string
	AuthMethod          string
	AuthHeaderPrefix    string
	AuthCustomHeader    string
	AuthQueryParam      string
	AuthURLPathTemplate string
	AccessToken         string
	RefreshToken        string
	AccessPolicy        *AccessPolicy
}

// CustomAPITestRequest tests a stored connector, supplied credentials, or
// stored-with-overrides. Stored credentials are sent only to stored origins.
type CustomAPITestRequest struct {
	ConnectionID        string
	AccessToken         string
	BaseAPIURL          string
	TestURL             string
	AuthMethod          string
	AuthHeaderPrefix    string
	AuthCustomHeader    string
	AuthQueryParam      string
	AuthURLPathTemplate string
}

// CustomAPITestResult reports one test outcome. Credential failures arrive
// as Status "error" in a 200 body, not as an HTTP error.
type CustomAPITestResult struct {
	Status  string
	Message string
}

// CustomAPIUpdateRequest updates one Custom API connection. Only set fields
// change; empty slices and strings leave existing values in place.
type CustomAPIUpdateRequest struct {
	Description      string
	BaseAPIURL       string
	TestURL          string
	AccessToken      string
	AccessPolicy     *AccessPolicy
	ClearAccessToken bool
}
