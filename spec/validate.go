package spec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

// Error codes. Each code names one failure class; Path locates the value in
// JSON-path form (for example "ui.pages[0].components[2].type").
type Code string

// Error taxonomy for v1 validation.
const (
	ESchema        Code = "E_SCHEMA"
	ESchemaVersion Code = "E_SCHEMA_VERSION"
	EApp           Code = "E_APP"
	EData          Code = "E_DATA"
	EComponent     Code = "E_COMPONENT"
	EWorkflow      Code = "E_WORKFLOW"
	ECapability    Code = "E_CAPABILITY"
	EPolicy        Code = "E_POLICY"
	EFixture       Code = "E_FIXTURE"
	EPresentation  Code = "E_PRESENTATION"
	EBinding       Code = "E_BINDING"
	ESecret        Code = "E_SECRET"
	EHash          Code = "E_HASH"
)

// Error is one deterministic validation failure.
type Error struct {
	Code    Code
	Path    string
	Message string
}

// Error renders code, path, and message.
func (e Error) Error() string {
	return string(e.Code) + ": " + e.Path + ": " + e.Message
}

var (
	rxSlug       = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,98}[a-z0-9])?$`)
	rxAppID      = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	rxSemver     = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	rxIdent      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	rxCapability = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
	rxRef        = regexp.MustCompile(`^\$\{[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*\}$`)
	rxBinding    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)
)

// secretFields are JSON field names that must never appear in a spec (matched
// case-insensitively). Secrets live in the connector gateway, never in specs,
// prompts, logs, or artifacts.
var secretFields = map[string]bool{
	"password": true, "secret": true, "api_key": true, "apikey": true,
	"access_token": true, "refresh_token": true, "private_key": true,
	"client_secret": true, "auth_token": true, "bearer_token": true,
}

// credentialPrefixes is a tripwire, not a security boundary: connection
// values starting with a known credential prefix are refused at admission.
// The actual boundary is server-side registry resolution — unknown IDs are
// rejected at runtime whether or not they match this list.
var credentialPrefixes = []string{"sk-", "AKIA", "xoxo-", "ghp-", "gsk-"}

var (
	componentTypes = map[string]bool{
		ComponentDashboard: true, ComponentMetricCard: true, ComponentTable: true,
		ComponentRecordDetail: true, ComponentForm: true, ComponentActionButton: true,
		ComponentChart: true, ComponentPermissionsCard: true, ComponentRunResult: true,
		ComponentStatusPanel: true,
	}
	stepTypes = map[string]bool{
		StepConnectorRead: true, StepTransform: true, StepBranch: true,
		StepBoundedLoop: true, StepNotify: true, StepFinish: true, StepConfirm: true,
	}
	fieldTypes = map[string]bool{
		FieldString: true, FieldInteger: true, FieldNumber: true, FieldBoolean: true,
		FieldDate: true, FieldObject: true, FieldArray: true,
	}
	filterOps = map[string]bool{
		OpEq: true, OpNeq: true, OpGt: true, OpGte: true, OpLt: true,
		OpLte: true, OpContains: true, OpIn: true,
	}
	aggregateFuncs = map[string]bool{"count": true, "sum": true, "avg": true}
)

// validator accumulates errors in deterministic section order.
type validator struct {
	errs    []Error
	capIdx  map[string]*CapabilityOp
	entIdx  map[string]*Entity
	queryIx map[string]*Query
	wfIx    map[string]*Workflow
	pending []pendingRef
}

func (v *validator) add(code Code, path, format string, args ...any) {
	v.errs = append(v.errs, Error{Code: code, Path: path, Message: fmt.Sprintf(format, args...)})
}

// Validate decodes raw strictly (unknown fields rejected) and checks every
// section in fixed order, collecting all failures. A nil error slice means
// the spec is valid; a non-empty slice forbids ANY use of the spec, and
// Validate returns nil in that case — callers must never ignore errs.
// Validation is deterministic: same input, same output.
//
// Decoding contract (locked by tests): field names match case-insensitively
// per encoding/json; duplicate keys resolve last-wins and the surviving value
// is always itself validated; exactly one JSON document is accepted and any
// trailing data is rejected.
func Validate(raw []byte) (*MiniappSpec, []Error) {
	v := &validator{
		capIdx:  map[string]*CapabilityOp{},
		entIdx:  map[string]*Entity{},
		queryIx: map[string]*Query{},
		wfIx:    map[string]*Workflow{},
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, []Error{{Code: ESchema, Path: "$", Message: "invalid JSON: " + err.Error()}}
	}
	v.scanSecrets("$", decoded)
	var spec MiniappSpec
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&spec); err != nil {
		v.add(ESchema, "$", "schema violation: %v", err)
		return nil, v.errs
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		v.add(ESchema, "$", "trailing data after JSON document")
		return nil, v.errs
	}
	if spec.Metadata.SchemaVersion != SchemaVersion {
		v.add(ESchemaVersion, "metadata.schemaVersion",
			"unsupported schema version %q, want %q", spec.Metadata.SchemaVersion, SchemaVersion)
	}
	v.checkApp(&spec.App)
	v.checkCapabilities(&spec.Capabilities)
	v.checkData(&spec.Data)
	v.checkUI(&spec.UI)
	v.checkWorkflows(spec.Workflows)
	v.checkPolicies(&spec.Policies)
	v.checkFixtures(spec.Fixtures)
	v.checkPresentation(&spec.Presentation)
	v.checkHash(&spec)
	if len(v.errs) != 0 {
		return nil, v.errs
	}
	return &spec, nil
}

// ValidateSpec validates an already-constructed spec — programmatic builds,
// transformations, or structs received out of band — with exactly the same
// rules as Validate: the struct is marshalled and run through the document
// pipeline, so no second rule implementation can drift. It reports only
// errors; use ContentHash for identity after success.
func ValidateSpec(spec *MiniappSpec) []Error {
	if spec == nil {
		return []Error{{Code: ESchema, Path: "$", Message: "nil spec"}}
	}
	raw, err := json.Marshal(spec)
	if err != nil {
		return []Error{{Code: ESchema, Path: "$", Message: "cannot encode spec: " + err.Error()}}
	}
	_, errs := Validate(raw)
	return errs
}

// ScanSecretsJSON applies the secret-field scan to a raw JSON document
// (specs, envelopes, manifests). Exported for artifact admission so
// the rule list stays single-sourced; behavior matches Validate's.
func ScanSecretsJSON(raw []byte) []Error {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return []Error{{Code: ESchema, Path: "$", Message: "invalid JSON: " + err.Error()}}
	}
	v := &validator{}
	v.scanSecrets("$", decoded)
	return v.errs
}

// scanSecrets rejects secret-bearing field names anywhere in the raw document.
func (v *validator) scanSecrets(path string, node any) {
	switch n := node.(type) {
	case map[string]any:
		keys := make([]string, 0, len(n))
		for k := range n {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p := path + "." + k
			if secretFields[strings.ToLower(k)] {
				v.add(ESecret, p, "secret-bearing field %q must never appear in a spec", k)
			}
			v.scanSecrets(p, n[k])
		}
	case []any:
		for i, e := range n {
			v.scanSecrets(fmt.Sprintf("%s[%d]", path, i), e)
		}
	}
}

func (v *validator) checkApp(a *AppSection) {
	if len(a.ID) == 0 || len(a.ID) > 63 || !rxAppID.MatchString(a.ID) {
		v.add(EApp, "app.id", "must be a DNS label (lowercase alphanumerics and '-', 1-63 chars)")
	}
	if len(a.Slug) == 0 || len(a.Slug) > 100 || !rxSlug.MatchString(a.Slug) {
		v.add(EApp, "app.slug", "must be a URL-safe slug (lowercase alphanumerics and '-', 1-100 chars)")
	}
	if strings.TrimSpace(a.Name) == "" || len(a.Name) > 256 {
		v.add(EApp, "app.name", "must be non-empty, at most 256 chars")
	}
	if !rxSemver.MatchString(a.AppVersion) {
		v.add(EApp, "app.appVersion", "must be MAJOR.MINOR.PATCH, got %q", a.AppVersion)
	}
	if strings.TrimSpace(a.Owner) == "" || len(a.Owner) > 256 {
		v.add(EApp, "app.owner", "must name the owning workspace (tenant anchor)")
	}
}

func (v *validator) checkCapabilities(c *CapabilitiesSection) {
	for i := range c.Operations {
		op := &c.Operations[i]
		p := fmt.Sprintf("capabilities.operations[%d]", i)
		if !rxCapability.MatchString(op.ID) {
			v.add(ECapability, p+".id", "must be a dotted logical ID such as %q", "invoices.list")
		} else if _, dup := v.capIdx[op.ID]; dup {
			v.add(ECapability, p+".id", "duplicate capability %q", op.ID)
		} else {
			v.capIdx[op.ID] = op
		}
		if strings.TrimSpace(op.Connection) == "" || len(op.Connection) > 128 ||
			strings.ContainsAny(op.Connection, " \t\n") || strings.Contains(op.Connection, "://") {
			v.add(ECapability, p+".connection", "must be a connection ID reference, never a secret or URL")
		}
		for _, pre := range credentialPrefixes {
			if strings.HasPrefix(op.Connection, pre) {
				v.add(ECapability, p+".connection",
					"value looks like a credential; connections must be opaque registry IDs")
				break
			}
		}
		if op.Method != MethodGet {
			v.add(ECapability, p+".method", "v1 admits %q only, got %q", MethodGet, op.Method)
		}
		if op.Effect != EffectRead {
			v.add(ECapability, p+".effect", "v1 admits %q only, got %q", EffectRead, op.Effect)
		}
		if len(op.Paths) == 0 {
			v.add(ECapability, p+".paths", "must declare at least one allowed path")
		}
		for j, pat := range op.Paths {
			if !validPathPattern(pat) {
				v.add(ECapability, fmt.Sprintf("%s.paths[%d]", p, j),
					"must be an absolute path without '..' or whitespace, optional trailing '/*'")
			}
		}
		if op.TimeoutSecs < 0 || op.TimeoutSecs > MaxTimeoutSecs {
			v.add(ECapability, p+".timeoutSecs", "must be 0 (default) or 1-%d", MaxTimeoutSecs)
		}
		if op.MaxResponseBytes < 0 || op.MaxResponseBytes > MaxResponseBytes {
			v.add(ECapability, p+".maxResponseBytes", "must be 0 (default) or 1-%d", MaxResponseBytes)
		}
	}
}

// validPathPattern allows exact absolute paths and trailing-/* wildcards.
// The literal base must satisfy the shared path predicate.
func validPathPattern(pat string) bool {
	base := pat
	if strings.HasSuffix(pat, "/*") {
		base = pat[:len(pat)-1]
	}
	if _, err := NormalizePath(base); err != nil {
		return false
	}
	return !strings.Contains(base, "*")
}

func (v *validator) checkData(d *DataSection) {
	for i := range d.Entities {
		e := &d.Entities[i]
		p := fmt.Sprintf("data.entities[%d]", i)
		if !rxIdent.MatchString(e.Name) {
			v.add(EData, p+".name", "must be an identifier")
		} else if _, dup := v.entIdx[e.Name]; dup {
			v.add(EData, p+".name", "duplicate entity %q", e.Name)
		} else {
			v.entIdx[e.Name] = e
		}
		seen := map[string]bool{}
		for j, f := range e.Fields {
			fp := fmt.Sprintf("%s.fields[%d]", p, j)
			if !rxIdent.MatchString(f.Name) {
				v.add(EData, fp+".name", "must be an identifier")
			} else if seen[f.Name] {
				v.add(EData, fp+".name", "duplicate field %q", f.Name)
			}
			seen[f.Name] = true
			if !fieldTypes[f.Type] {
				v.add(EData, fp+".type", "unknown field type %q", f.Type)
			}
		}
		if _, ok := v.capIdx[e.Source.Capability]; !ok {
			v.add(ECapability, p+".source.capability", "undeclared capability %q", e.Source.Capability)
		}
		if strings.TrimSpace(e.Source.Entity) == "" {
			v.add(EData, p+".source.entity", "must name the remote entity")
		}
	}
	for i := range d.Queries {
		q := &d.Queries[i]
		p := fmt.Sprintf("data.queries[%d]", i)
		if !rxIdent.MatchString(q.Name) {
			v.add(EData, p+".name", "must be an identifier")
		} else if _, dup := v.queryIx[q.Name]; dup {
			v.add(EData, p+".name", "duplicate query %q", q.Name)
		} else {
			v.queryIx[q.Name] = q
		}
		ent, ok := v.entIdx[q.Entity]
		if !ok {
			v.add(EBinding, p+".entity", "unknown entity %q", q.Entity)
			continue
		}
		fields := map[string]bool{}
		for _, f := range ent.Fields {
			fields[f.Name] = true
		}
		for j, f := range q.Filter {
			fp := fmt.Sprintf("%s.filter[%d]", p, j)
			if !fields[f.Field] {
				v.add(EBinding, fp+".field", "unknown field %q on entity %q", f.Field, q.Entity)
			}
			if !filterOps[f.Op] {
				v.add(EData, fp+".op", "unknown filter op %q", f.Op)
			}
			if f.Value == nil {
				v.add(EData, fp+".value", "filter value is required")
			}
		}
		for j, s := range q.Sort {
			if !fields[s.Field] {
				v.add(EBinding, fmt.Sprintf("%s.sort[%d].field", p, j),
					"unknown field %q on entity %q", s.Field, q.Entity)
			}
			if s.Dir != "" && s.Dir != "asc" && s.Dir != "desc" {
				v.add(EData, fmt.Sprintf("%s.sort[%d].dir", p, j), "must be \"asc\" or \"desc\"")
			}
		}
		if q.Limit < 0 || q.Limit > MaxQueryLimit {
			v.add(EData, p+".limit", "must be 0 (unset) or 1-%d", MaxQueryLimit)
		}
		for j, a := range q.Aggregates {
			ap := fmt.Sprintf("%s.aggregates[%d]", p, j)
			if !aggregateFuncs[a.Func] {
				v.add(EData, ap+".func", "must be count, sum, or avg")
			}
			if !fields[a.Field] {
				v.add(EBinding, ap+".field", "unknown field %q on entity %q", a.Field, q.Entity)
			}
			if !rxIdent.MatchString(a.As) {
				v.add(EData, ap+".as", "must be an identifier")
			}
		}
	}
}

func (v *validator) checkUI(u *UISection) {
	if len(u.Pages) == 0 {
		v.add(EComponent, "ui.pages", "must declare at least one page")
	}
	if len(u.Pages) > MaxPagesPerApp {
		v.add(EComponent, "ui.pages", "at most %d pages", MaxPagesPerApp)
	}
	pageIDs := map[string]bool{}
	for i := range u.Pages {
		pg := &u.Pages[i]
		p := fmt.Sprintf("ui.pages[%d]", i)
		if !rxIdent.MatchString(pg.ID) {
			v.add(EComponent, p+".id", "must be an identifier")
		} else if pageIDs[pg.ID] {
			v.add(EComponent, p+".id", "duplicate page %q", pg.ID)
		}
		pageIDs[pg.ID] = true
		if strings.TrimSpace(pg.Title) == "" {
			v.add(EComponent, p+".title", "must be non-empty")
		}
		if _, err := NormalizePath(pg.Path); err != nil {
			v.add(EComponent, p+".path", "invalid page path: %v", err)
		}
		if len(pg.Components) > MaxComponentsPerPg {
			v.add(EComponent, p+".components", "at most %d components", MaxComponentsPerPg)
		}
		for j := range pg.Components {
			v.checkComponent(fmt.Sprintf("%s.components[%d]", p, j), &pg.Components[j])
		}
	}
}

func (v *validator) checkComponent(p string, c *Component) {
	if !componentTypes[c.Type] {
		v.add(EComponent, p+".type", "unknown component type %q", c.Type)
		return
	}
	switch c.Type {
	case ComponentTable:
		if c.Binding == "" {
			v.add(EComponent, p+".binding", "table requires a binding")
		} else {
			v.checkBinding(p+".binding", c.Binding)
		}
		if len(c.Columns) == 0 {
			v.add(EComponent, p+".columns", "table requires at least one column")
		}
		v.rejectComponentFields(p, c, true, true, false, false)
		v.checkBindingFields(p, c.Binding, c.Columns, "columns")
	case ComponentForm, ComponentActionButton:
		// Workflows validate after UI; record names now, resolve in checkWorkflows.
		v.checkWorkflowRef(p+".action", c.Action)
		if c.Type == ComponentForm {
			for j, f := range c.Fields {
				if !rxIdent.MatchString(f) {
					v.add(EComponent, fmt.Sprintf("%s.fields[%d]", p, j), "must be an identifier")
				}
			}
			v.rejectComponentFields(p, c, false, false, true, true)
		} else {
			v.rejectComponentFields(p, c, false, false, false, true)
		}
	case ComponentRecordDetail:
		if c.Binding == "" {
			v.add(EComponent, p+".binding", "%s requires a binding", c.Type)
		} else {
			v.checkBinding(p+".binding", c.Binding)
		}
		v.rejectComponentFields(p, c, true, false, true, false)
		v.checkBindingFields(p, c.Binding, c.Fields, "fields")
	case ComponentMetricCard, ComponentChart, ComponentPermissionsCard:
		if c.Binding == "" {
			v.add(EComponent, p+".binding", "%s requires a binding", c.Type)
		} else {
			v.checkBinding(p+".binding", c.Binding)
		}
		v.rejectComponentFields(p, c, true, false, false, false)
	case ComponentDashboard, ComponentRunResult, ComponentStatusPanel:
		if c.Binding != "" {
			v.checkBinding(p+".binding", c.Binding)
		}
		v.rejectComponentFields(p, c, true, false, false, false)
	}
	if c.Target != "" {
		if _, err := NormalizePath(c.Target); err != nil {
			v.add(EComponent, p+".target", "invalid target path: %v", err)
		}
	}
}

// rejectComponentFields enforces the per-component field matrix. Each flag
// permits one optional field; title and target are always allowed. Anything
// else present is rejected so ignored fields can never smuggle references
// past validation.
func (v *validator) rejectComponentFields(p string, c *Component, allowBinding, allowColumns, allowFields, allowAction bool) {
	if !allowBinding && c.Binding != "" {
		v.add(EComponent, p+".binding", "binding does not apply to %s", c.Type)
	}
	if !allowColumns && len(c.Columns) != 0 {
		v.add(EComponent, p+".columns", "columns do not apply to %s", c.Type)
	}
	if !allowFields && len(c.Fields) != 0 {
		v.add(EComponent, p+".fields", "fields do not apply to %s", c.Type)
	}
	if !allowAction && c.Action != "" {
		v.add(EComponent, p+".action", "action does not apply to %s", c.Type)
	}
}

// checkBinding validates a query/entity data path. Component bindings may only
// reference declared queries and entities, never workflow step outputs. When
// the binding carries a sub-path, the first segment must be a declared field
// of the resolved entity; deeper navigation is runtime-resolved.
func (v *validator) checkBinding(path, binding string) {
	if !rxBinding.MatchString(binding) {
		v.add(EBinding, path, "malformed binding %q", binding)
		return
	}
	segs := strings.Split(binding, ".")
	ent := v.bindingEntity(segs[0])
	if ent == nil {
		v.add(EBinding, path, "unknown query or entity %q", segs[0])
		return
	}
	if len(segs) > 1 && !entityHasField(ent, segs[1]) {
		v.add(EBinding, path, "unknown field %q on entity %q", segs[1], ent.Name)
	}
}

// bindingEntity resolves a binding head to its entity definition via a query
// or directly, or returns nil when undeclared.
func (v *validator) bindingEntity(head string) *Entity {
	if q, ok := v.queryIx[head]; ok {
		if ent, ok := v.entIdx[q.Entity]; ok {
			return ent
		}
		return nil
	}
	if ent, ok := v.entIdx[head]; ok {
		return ent
	}
	return nil
}

func entityHasField(ent *Entity, field string) bool {
	for _, f := range ent.Fields {
		if f.Name == field {
			return true
		}
	}
	return false
}

// checkBindingFields resolves every name against the entity behind binding.
// It stays silent when the binding itself is unresolvable; the binding error
// already covers that case.
func (v *validator) checkBindingFields(p, binding string, names []string, kind string) {
	if binding == "" || !rxBinding.MatchString(binding) {
		return
	}
	ent := v.bindingEntity(strings.Split(binding, ".")[0])
	if ent == nil {
		return
	}
	for j, name := range names {
		if !entityHasField(ent, name) {
			v.add(EBinding, fmt.Sprintf("%s.%s[%d]", p, kind, j),
				"unknown field %q on entity %q", name, ent.Name)
		}
	}
}

// checkWorkflowRef validates a workflow reference. UI validates before
// workflows are indexed, so resolution is deferred: names are recorded here
// and resolved in checkWorkflows.
func (v *validator) checkWorkflowRef(path, id string) {
	v.pending = append(v.pending, pendingRef{path: path, id: id})
}

// pendingRef records one forward reference for deferred resolution.
type pendingRef struct {
	path string
	id   string
}

func (v *validator) checkWorkflows(wfs []Workflow) {
	if len(wfs) > MaxWorkflowsPerApp {
		v.add(EWorkflow, "workflows", "at most %d workflows", MaxWorkflowsPerApp)
	}
	for i := range wfs {
		w := &wfs[i]
		p := fmt.Sprintf("workflows[%d]", i)
		if !rxIdent.MatchString(w.ID) {
			v.add(EWorkflow, p+".id", "must be an identifier")
		} else if _, dup := v.wfIx[w.ID]; dup {
			v.add(EWorkflow, p+".id", "duplicate workflow %q", w.ID)
		} else {
			v.wfIx[w.ID] = w
		}
		if w.Trigger.Type != TriggerManual {
			v.add(EWorkflow, p+".trigger.type", "v1 admits %q only", TriggerManual)
		}
		if len(w.Steps) == 0 || len(w.Steps) > MaxStepsPerWorkflow {
			v.add(EWorkflow, p+".steps", "must declare 1-%d steps", MaxStepsPerWorkflow)
		}
		outputs := map[string]bool{}
		stepIDs := map[string]bool{}
		for j := range w.Steps {
			v.checkStep(fmt.Sprintf("%s.steps[%d]", p, j), &w.Steps[j], outputs)
			s := &w.Steps[j]
			if stepIDs[s.ID] {
				v.add(EWorkflow, fmt.Sprintf("%s.steps[%d].id", p, j), "duplicate step %q", s.ID)
			}
			stepIDs[s.ID] = true
			if s.Output != "" {
				if outputs[s.Output] {
					v.add(EWorkflow, fmt.Sprintf("%s.steps[%d].output", p, j),
						"duplicate output %q", s.Output)
				}
				outputs[s.Output] = true
			}
		}
	}
	for _, r := range v.pending {
		if r.id == "" {
			v.add(EComponent, r.path, "action must reference a workflow")
		} else if _, ok := v.wfIx[r.id]; !ok {
			v.add(EBinding, r.path, "unknown workflow %q", r.id)
		}
	}
}

func (v *validator) checkStep(p string, s *Step, outputs map[string]bool) {
	if !rxIdent.MatchString(s.ID) {
		v.add(EWorkflow, p+".id", "must be an identifier")
	}
	if !stepTypes[s.Type] {
		v.add(EWorkflow, p+".type", "unknown step type %q", s.Type)
		return
	}
	if s.Output != "" && !rxIdent.MatchString(s.Output) {
		v.add(EWorkflow, p+".output", "must be an identifier")
	}
	keys := make([]string, 0, len(s.Input))
	for k := range s.Input {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v.checkRef(p+".input."+k, s.Input[k], outputs)
	}
	switch s.Type {
	case StepConnectorRead:
		op, ok := v.capIdx[s.Operation]
		if !ok {
			v.add(ECapability, p+".operation", "undeclared capability %q", s.Operation)
			break
		}
		norm, err := NormalizePath(s.Path)
		if err != nil {
			v.add(ECapability, p+".path", "invalid request path: %v", err)
		} else if !PathAllowed(op.Paths, norm) {
			v.add(ECapability, p+".path", "path %q is not in the declared allowlist", s.Path)
		}
	case StepTransform:
		if len(s.Input) == 0 {
			v.add(EWorkflow, p+".input", "transform requires at least one input")
		}
		if s.Output == "" {
			v.add(EWorkflow, p+".output", "transform requires an output name")
		}
	case StepBranch:
		if s.Condition == "" {
			v.add(EWorkflow, p+".condition", "branch requires a condition reference")
		} else {
			v.checkRef(p+".condition", s.Condition, outputs)
		}
	case StepBoundedLoop:
		if s.MaxIterations < 1 || s.MaxIterations > MaxLoopIterations {
			v.add(EWorkflow, p+".maxIterations", "must be 1-%d", MaxLoopIterations)
		}
		if len(s.Input) == 0 {
			v.add(EWorkflow, p+".input", "bounded loop requires a collection input")
		}
		if s.Output == "" {
			v.add(EWorkflow, p+".output", "bounded loop requires an output name")
		}
	case StepNotify, StepFinish, StepConfirm:
		if strings.TrimSpace(s.Message) == "" {
			v.add(EWorkflow, p+".message", "%s requires a message", s.Type)
		}
		for _, ref := range extractRefs(s.Message) {
			v.checkRef(p+".message", ref, outputs)
		}
	}
	v.rejectStepFields(p, s)
}

// rejectStepFields enforces the per-step-type field matrix. Step ID, type,
// input (validated where present), and output are allowed on every step;
// anything else present must belong to the declared type, otherwise ignored
// fields could smuggle undeclared operations or references past validation.
func (v *validator) rejectStepFields(p string, s *Step) {
	if s.Type != StepConnectorRead {
		if s.Operation != "" {
			v.add(EWorkflow, p+".operation", "operation does not apply to %s", s.Type)
		}
		if s.Path != "" {
			v.add(EWorkflow, p+".path", "path does not apply to %s", s.Type)
		}
	}
	if s.Type != StepBranch && s.Condition != "" {
		v.add(EWorkflow, p+".condition", "condition does not apply to %s", s.Type)
	}
	if s.Type != StepBoundedLoop && s.MaxIterations != 0 {
		v.add(EWorkflow, p+".maxIterations", "maxIterations does not apply to %s", s.Type)
	}
	switch s.Type {
	case StepNotify, StepFinish, StepConfirm:
	default:
		if s.Message != "" {
			v.add(EWorkflow, p+".message", "message does not apply to %s", s.Type)
		}
	}
}

// checkRef validates one ${...} reference against queries, entities, and prior
// step outputs. Forward and self references are rejected because outputs only
// contains already-validated steps.
func (v *validator) checkRef(path, ref string, outputs map[string]bool) {
	if !rxRef.MatchString(ref) {
		v.add(EBinding, path, "malformed reference %q, want ${name} or ${name.field...}", ref)
		return
	}
	inner := ref[2 : len(ref)-1]
	head := inner
	if i := strings.Index(head, "."); i >= 0 {
		head = head[:i]
	}
	if outputs[head] {
		return
	}
	if _, ok := v.queryIx[head]; ok {
		return
	}
	if _, ok := v.entIdx[head]; ok {
		return
	}
	v.add(EBinding, path, "unknown reference source %q", head)
}

// extractRefs returns the ${...} references embedded in free text.
func extractRefs(text string) []string {
	var out []string
	for i := 0; i < len(text); i++ {
		if text[i] == '$' && i+1 < len(text) && text[i+1] == '{' {
			if j := strings.Index(text[i+2:], "}"); j >= 0 {
				out = append(out, text[i:i+2+j+1])
				i += 2 + j
			}
		}
	}
	return out
}

func (v *validator) checkPolicies(p *PoliciesSection) {
	if p.Confirmation.ExternalWrites != ExternalWritesOff {
		v.add(EPolicy, "policies.confirmation.externalWrites",
			"v1 requires %q (writes are rejected by the validator until G-01)", ExternalWritesOff)
	}
	seen := map[string]bool{}
	for i, r := range p.Roles {
		rp := fmt.Sprintf("policies.roles[%d]", i)
		if strings.TrimSpace(r.Name) == "" {
			v.add(EPolicy, rp+".name", "must be non-empty")
		} else if seen[r.Name] {
			v.add(EPolicy, rp+".name", "duplicate role %q", r.Name)
		}
		seen[r.Name] = true
		for j, a := range r.Actions {
			if _, ok := v.wfIx[a]; !ok {
				v.add(EBinding, fmt.Sprintf("%s.actions[%d]", rp, j), "unknown workflow %q", a)
			}
		}
	}
	if p.Limits.MaxRunsPerDay < 0 || p.Limits.MaxRunsPerDay > MaxRunsPerDay {
		v.add(EPolicy, "policies.limits.maxRunsPerDay", "must be 0 (unset) or 1-%d", MaxRunsPerDay)
	}
	if p.Limits.MaxActionsPerRun < 0 || p.Limits.MaxActionsPerRun > MaxActionsPerRun {
		v.add(EPolicy, "policies.limits.maxActionsPerRun", "must be 0 (unset) or 1-%d", MaxActionsPerRun)
	}
}

func (v *validator) checkFixtures(fx []Fixture) {
	if len(fx) == 0 {
		v.add(EFixture, "fixtures", "must declare at least one dry-run fixture")
		return
	}
	seen := map[string]bool{}
	for i, f := range fx {
		p := fmt.Sprintf("fixtures[%d]", i)
		if strings.TrimSpace(f.Name) == "" {
			v.add(EFixture, p+".name", "must be non-empty")
		} else if seen[f.Name] {
			v.add(EFixture, p+".name", "duplicate fixture %q", f.Name)
		}
		seen[f.Name] = true
	}
}

func (v *validator) checkPresentation(p *PresentationSection) {
	if p.Theme != "" && p.Theme != "light" && p.Theme != "dark" {
		v.add(EPresentation, "presentation.theme", "must be \"light\" or \"dark\"")
	}
}

func (v *validator) checkHash(spec *MiniappSpec) {
	if spec.Metadata.ContentHash == "" {
		return
	}
	want, err := ContentHash(spec)
	if err != nil {
		v.add(EHash, "metadata.contentHash", "cannot compute content hash: %v", err)
		return
	}
	if spec.Metadata.ContentHash != want {
		v.add(EHash, "metadata.contentHash", "mismatch: spec content does not match the declared hash")
	}
}
