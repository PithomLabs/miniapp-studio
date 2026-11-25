package spec

// SchemaVersion is the only MiniappSpec schema version the MVP validator
// accepts. It is compared for exact equality; unknown versions are rejected.
const SchemaVersion = "miniappspec/v1"

// Validator bounds. These are part of the v1 contract: out-of-range values
// are rejected, never silently clamped.
const (
	MaxPagesPerApp       = 64
	MaxComponentsPerPg   = 64
	MaxWorkflowsPerApp   = 32
	MaxStepsPerWorkflow  = 32
	MaxLoopIterations    = 1000
	MaxQueryLimit        = 1000
	MaxTimeoutSecs       = 120
	DefaultTimeoutSecs   = 30
	MaxResponseBytes     = 1048576 // 1 MiB
	DefaultResponseBytes = 262144  // 256 KiB
	MaxActionsPerRun     = 10000
	MaxRunsPerDay        = 100000
)

// Closed UI component vocabulary for v1.
const (
	ComponentDashboard       = "dashboard"
	ComponentMetricCard      = "metric-card"
	ComponentTable           = "table"
	ComponentRecordDetail    = "record-detail"
	ComponentForm            = "form"
	ComponentActionButton    = "action-button"
	ComponentChart           = "chart"
	ComponentPermissionsCard = "permissions-card"
	ComponentRunResult       = "run-result"
	ComponentStatusPanel     = "status-panel"
)

// Closed workflow vocabulary for v1. There is deliberately no write step:
// external writes are rejected by the validator until the G-01 lifecycle is
// fixed and verified. Rejection is enforced at validation and execution
// boundaries; the Go types alone cannot make a string field unconstructible.
const (
	TriggerManual = "manual"

	StepConnectorRead = "connector-read"
	StepTransform     = "transform"
	StepBranch        = "branch"
	StepBoundedLoop   = "bounded-loop"
	StepNotify        = "notify"
	StepFinish        = "finish"
	StepConfirm       = "confirm"
)

// Closed data vocabulary for v1.
const (
	FieldString  = "string"
	FieldInteger = "integer"
	FieldNumber  = "number"
	FieldBoolean = "boolean"
	FieldDate    = "date"
	FieldObject  = "object"
	FieldArray   = "array"
)

// Closed filter operators for v1.
const (
	OpEq       = "eq"
	OpNeq      = "neq"
	OpGt       = "gt"
	OpGte      = "gte"
	OpLt       = "lt"
	OpLte      = "lte"
	OpContains = "contains"
	OpIn       = "in"
)

// Capability and policy constants. v1 admits reads only.
const (
	MethodGet         = "GET"
	EffectRead        = "read"
	ExternalWritesOff = "disabled"
)

// MiniappSpec is one versioned application definition. A workflow definition
// is a section of the spec, not a separate language.
type MiniappSpec struct {
	App          AppSection          `json:"app"`
	Data         DataSection         `json:"data"`
	UI           UISection           `json:"ui"`
	Workflows    []Workflow          `json:"workflows,omitempty"`
	Capabilities CapabilitiesSection `json:"capabilities"`
	Policies     PoliciesSection     `json:"policies"`
	Fixtures     []Fixture           `json:"fixtures"`
	Presentation PresentationSection `json:"presentation,omitempty"`
	Metadata     MetadataSection     `json:"metadata"`
}

// AppSection identifies the application. Owner names the owning workspace and
// is the tenant anchor: app IDs are unique per tenant, and persistence and
// resource ownership scope every record by (tenant, app ID). Owner is an
// opaque tenant label — required present, with no canonical form defined
// here; the server derives the tenant from authentication context and
// compares against this value. Never treat it as proof of authorization.
type AppSection struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	AppVersion  string `json:"appVersion"`
	Owner       string `json:"owner"`
}

// DataSection declares logical entities and named queries over them.
type DataSection struct {
	Entities []Entity `json:"entities"`
	Queries  []Query  `json:"queries,omitempty"`
}

// Entity is a logical collection backed by one capability operation.
type Entity struct {
	Name   string     `json:"name"`
	Fields []Field    `json:"fields"`
	Source DataSource `json:"source"`
}

// Field is one typed attribute of an entity.
type Field struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// DataSource binds an entity to the connector operation that serves it.
// The interpreter fetches entity rows with path "/" plus the remote entity
// name; that path must appear in the operation's allowlist like any other.
type DataSource struct {
	Capability string `json:"capability"`
	Entity     string `json:"entity"`
}

// Query is a named, bounded read over one entity.
type Query struct {
	Name       string      `json:"name"`
	Entity     string      `json:"entity"`
	Filter     []Filter    `json:"filter,omitempty"`
	Sort       []SortKey   `json:"sort,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Aggregates []Aggregate `json:"aggregates,omitempty"`
}

// Filter constrains one field. Value carries the comparison operand.
type Filter struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}

// SortKey orders a query by one field.
type SortKey struct {
	Field string `json:"field"`
	Dir   string `json:"dir,omitempty"` // "asc" (default) or "desc"
}

// Aggregate computes one grouped metric.
type Aggregate struct {
	Func  string `json:"func"` // "count", "sum", or "avg"
	Field string `json:"field"`
	As    string `json:"as"`
}

// UISection declares pages of reusable components.
type UISection struct {
	Pages []Page `json:"pages"`
}

// Page is one routable screen.
type Page struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Path       string      `json:"path"`
	Components []Component `json:"components"`
}

// Component is one reusable UI primitive instance.
//
// Field matrix (enforced by the validator; anything else present is rejected):
//   - table: binding (required), columns (required, resolved against the entity)
//   - form: action (required), fields (optional free-form input names)
//   - action-button: action (required)
//   - record-detail: binding (required), fields (optional, resolved against the entity)
//   - metric-card, chart, permissions-card: binding (required)
//   - dashboard, run-result, status-panel: binding (optional)
//
// Type, title, and target are allowed on every component.
type Component struct {
	Type    string   `json:"type"`
	Title   string   `json:"title,omitempty"`
	Binding string   `json:"binding,omitempty"`
	Action  string   `json:"action,omitempty"`
	Target  string   `json:"target,omitempty"`
	Columns []string `json:"columns,omitempty"`
	Fields  []string `json:"fields,omitempty"`
}

// Workflow is a trigger plus a constrained, ordered step list. Steps execute
// in order; a branch whose condition is falsy ends the workflow (guard
// semantics). v1 has no other control-flow edges.
type Workflow struct {
	ID      string  `json:"id"`
	Trigger Trigger `json:"trigger"`
	Steps   []Step  `json:"steps"`
}

// Trigger starts a workflow. v1 admits manual triggers only.
type Trigger struct {
	Type string `json:"type"`
}

// Step is one typed unit of work. Input maps argument names to `${...}`
// references; Output names the value later steps may reference.
//
// Field matrix (enforced by the validator; anything else present is rejected):
//   - connector-read: operation, path
//   - transform: input (required), output (required)
//   - branch: condition
//   - bounded-loop: input (required), maxIterations, output (required)
//   - notify, finish, confirm: message
//
// ID, type, input, and output are allowed on every step.
type Step struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Operation     string            `json:"operation,omitempty"`
	Path          string            `json:"path,omitempty"`
	Input         map[string]string `json:"input,omitempty"`
	Output        string            `json:"output,omitempty"`
	Condition     string            `json:"condition,omitempty"`
	MaxIterations int               `json:"maxIterations,omitempty"`
	Message       string            `json:"message,omitempty"`
}

// CapabilitiesSection declares every logical operation the app may invoke.
type CapabilitiesSection struct {
	Operations []CapabilityOp `json:"operations,omitempty"`
}

// CapabilityOp binds one logical operation to a connector connection. The
// connection field references a connection ID resolved against a server-side
// registry; it must never carry a secret, URL, or credential. The secret
// scanner matches field names only and does not inspect values.
type CapabilityOp struct {
	ID               string   `json:"id"`
	Connection       string   `json:"connection"`
	Method           string   `json:"method"`
	Paths            []string `json:"paths"`
	Effect           string   `json:"effect"`
	TimeoutSecs      int      `json:"timeoutSecs,omitempty"`
	MaxResponseBytes int      `json:"maxResponseBytes,omitempty"`
}

// PoliciesSection declares governance. v1 requires external writes disabled.
type PoliciesSection struct {
	Roles        []Role        `json:"roles,omitempty"`
	Confirmation ConfirmPolicy `json:"confirmation"`
	Limits       Limits        `json:"limits,omitempty"`
}

// Role maps one named role to allowed workflow actions.
type Role struct {
	Name    string   `json:"name"`
	Actions []string `json:"actions"`
}

// ConfirmPolicy gates consequential behavior.
type ConfirmPolicy struct {
	ExternalWrites string `json:"externalWrites"`
}

// Limits bounds runaway execution. Zero means unset.
type Limits struct {
	MaxRunsPerDay    int `json:"maxRunsPerDay,omitempty"`
	MaxActionsPerRun int `json:"maxActionsPerRun,omitempty"`
}

// Fixture is one deterministic dry-run case.
type Fixture struct {
	Name   string         `json:"name"`
	Input  map[string]any `json:"input,omitempty"`
	Expect map[string]any `json:"expect,omitempty"`
}

// PresentationSection carries layout tokens. Theming beyond light/dark is roadmap.
type PresentationSection struct {
	Theme  string            `json:"theme,omitempty"`
	Tokens map[string]string `json:"tokens,omitempty"`
}

// MetadataSection versions the spec. ContentHash, when present, must equal
// [ContentHash] of the spec; it is excluded from the hashed content itself.
// Provenance must never carry provider credentials; provider selection lives
// in generation/session configuration, never in the spec.
type MetadataSection struct {
	SchemaVersion string     `json:"schemaVersion"`
	ContentHash   string     `json:"contentHash,omitempty"`
	Provenance    Provenance `json:"provenance,omitempty"`
}

// Provenance records safe generator metadata.
type Provenance struct {
	Generator string `json:"generator,omitempty"`
	Model     string `json:"model,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}
