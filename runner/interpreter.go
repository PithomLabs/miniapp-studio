package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/PithomLabs/miniapp-studio/connector"
	"github.com/PithomLabs/miniapp-studio/spec"
)

// DefaultMaxExecSeconds bounds one workflow execution when Deps leaves it unset.
const DefaultMaxExecSeconds = 60

// MaxResultBytes caps the marshalled workflow outputs.
const MaxResultBytes = 1048576

// ReadClient executes one connector read. Implementations speak to a fake,
// an httptest server (tests), or the P3 gateway adapter; the interpreter
// never sees URLs or credentials.
type ReadClient interface {
	Read(ctx context.Context, conn connector.Connection, method, path string, opts ReadOpts) ([]byte, error)
}

// ReadOpts carries per-operation bounds.
type ReadOpts struct {
	TimeoutSecs      int
	MaxResponseBytes int
}

// Deps wires the interpreter's only external seams.
type Deps struct {
	Connections connector.Registry
	Reader      ReadClient
	// MaxExecSeconds bounds Execute; <= 0 selects DefaultMaxExecSeconds.
	MaxExecSeconds int
}

// Result is one workflow execution outcome.
type Result struct {
	SpecHash      string         `json:"specHash"`
	WorkflowID    string         `json:"workflowId"`
	Outputs       map[string]any `json:"outputs"`
	Messages      []string       `json:"messages"`
	Notifications []string       `json:"notifications"`
	Completed     bool           `json:"completed"`
}

// Execute runs one workflow of an admitted spec to completion or to the first
// branch guard that ends it early. It revalidates admission, resolves tenant
// data lazily, and dispatches strictly on step type: unknown constructs fail
// closed and can never reach the read path.
func Execute(ctx context.Context, vspec ValidatedSpec, workflowID string, deps Deps) (Result, error) {
	if deps.Connections == nil || deps.Reader == nil {
		return Result{}, fmt.Errorf("runner: connections registry and read client are required")
	}
	if err := vspec.Revalidate(); err != nil {
		return Result{}, err
	}
	parsed, err := vspec.Spec()
	if err != nil {
		return Result{}, err
	}
	var flow *spec.Workflow
	for i := range parsed.Workflows {
		if parsed.Workflows[i].ID == workflowID {
			flow = &parsed.Workflows[i]
		}
	}
	if flow == nil {
		return Result{}, fmt.Errorf("runner: unknown workflow %q", workflowID)
	}
	maxExec := deps.MaxExecSeconds
	if maxExec <= 0 {
		maxExec = DefaultMaxExecSeconds
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(maxExec)*time.Second)
	defer cancel()
	r := &run{spec: parsed, deps: deps, outputs: map[string]any{}, entities: map[string][]any{}}
	res := Result{SpecHash: vspec.Hash(), WorkflowID: workflowID, Outputs: map[string]any{}, Completed: true}
	for i := range flow.Steps {
		step := &flow.Steps[i]
		outcome, err := r.execStep(ctx, step, &res)
		if err != nil {
			return Result{}, fmt.Errorf("runner: step %q (%s): %w", step.ID, step.Type, err)
		}
		if outcome == stepFinish {
			break
		}
		if outcome == stepBranchEnd {
			res.Completed = false
			break
		}
		if step.Output != "" {
			if v, ok := r.outputs[step.Output]; ok {
				res.Outputs[step.Output] = v
			}
		}
	}
	encoded, err := json.Marshal(res.Outputs)
	if err != nil {
		return Result{}, fmt.Errorf("runner: encode outputs: %w", err)
	}
	if len(encoded) > MaxResultBytes {
		return Result{}, fmt.Errorf("runner: outputs exceed %d bytes", MaxResultBytes)
	}
	return res, nil
}

// run carries per-execution interpreter state. When fixture is set, the run
// is a dry-run: connector and entity data resolve from the fixture and the
// live read path is unreachable (deps stay sealed).
type run struct {
	spec     *spec.MiniappSpec
	deps     Deps
	outputs  map[string]any
	entities map[string][]any
	fixture  *spec.Fixture
	fixName  string
}

// stepOutcome tells Execute whether to continue, finish successfully, or end
// early on a falsy branch guard.
type stepOutcome int

const (
	stepContinue stepOutcome = iota
	stepFinish
	stepBranchEnd
)

// execStep executes one step. Dispatch is exhaustive on the validated
// discriminator; the default arm fails closed and is unreachable for admitted
// specs (defense in depth, covered by a white-box test).
func (r *run) execStep(ctx context.Context, s *spec.Step, res *Result) (stepOutcome, error) {
	switch s.Type {
	case spec.StepConnectorRead:
		return stepContinue, r.execRead(ctx, s)
	case spec.StepTransform:
		return stepContinue, r.execTransform(ctx, s)
	case spec.StepBranch:
		ended, err := r.execBranch(ctx, s)
		if err != nil {
			return stepContinue, err
		}
		if ended {
			return stepBranchEnd, nil
		}
		return stepContinue, nil
	case spec.StepBoundedLoop:
		return stepContinue, r.execLoop(ctx, s)
	case spec.StepNotify:
		msg, err := r.interpolate(ctx, s.Message)
		if err != nil {
			return stepContinue, err
		}
		res.Notifications = append(res.Notifications, msg)
		return stepContinue, nil
	case spec.StepFinish:
		msg, err := r.interpolate(ctx, s.Message)
		if err != nil {
			return stepContinue, err
		}
		res.Messages = append(res.Messages, msg)
		return stepFinish, nil
	case spec.StepConfirm:
		// v1 records the acknowledgement. True mid-flow approval parking
		// needs durable execution (roadmap); approval gating in the MVP
		// happens at tool-invocation level, not inside a workflow.
		msg, err := r.interpolate(ctx, s.Message)
		if err != nil {
			return stepContinue, err
		}
		res.Messages = append(res.Messages, msg)
		return stepContinue, nil
	default:
		return stepContinue, fmt.Errorf("unsupported step type %q", s.Type)
	}
}

// execRead performs one declared connector read. Method, effect, connection,
// path normalization, and allowlist membership are re-checked at execution
// time even though admission already verified them.
func (r *run) execRead(ctx context.Context, s *spec.Step) error {
	if r.fixture != nil {
		return r.execReadDry(s)
	}
	rows, err := r.readOp(ctx, s.Operation, s.Path)
	if err != nil {
		return err
	}
	if s.Output != "" {
		r.outputs[s.Output] = rows
	}
	return nil
}

// execReadDry satisfies a connector read from fixture inputs keyed by the
// step's output name. It never touches the registry or read client.
func (r *run) execReadDry(s *spec.Step) error {
	if s.Output == "" {
		return nil
	}
	v, ok := r.fixture.Input[s.Output]
	if !ok {
		return fmt.Errorf("dry-run: fixture %q has no input for step output %q", r.fixName, s.Output)
	}
	rows, ok := v.([]any)
	if !ok {
		return fmt.Errorf("dry-run: fixture %q input %q must be an array", r.fixName, s.Output)
	}
	r.outputs[s.Output] = rows
	return nil
}

// readOp is the single read path for steps and entity fetches: capability
// lookup, read-only re-check, registry resolution, normalized allowlist
// match, bounded read, array decode.
func (r *run) readOp(ctx context.Context, opID, path string) ([]any, error) {
	if r.fixture != nil {
		return nil, fmt.Errorf("dry-run: live read path is unreachable")
	}
	var op *spec.CapabilityOp
	for i := range r.spec.Capabilities.Operations {
		if r.spec.Capabilities.Operations[i].ID == opID {
			op = &r.spec.Capabilities.Operations[i]
		}
	}
	if op == nil {
		return nil, fmt.Errorf("undeclared capability %q", opID)
	}
	if op.Method != spec.MethodGet || op.Effect != spec.EffectRead {
		return nil, fmt.Errorf("v1 executes reads only (got %s/%s)", op.Method, op.Effect)
	}
	norm, err := spec.NormalizePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	if !spec.PathAllowed(op.Paths, norm) {
		return nil, fmt.Errorf("path %q is not allowlisted", path)
	}
	conn, err := r.deps.Connections.Resolve(ctx, op.Connection)
	if err != nil {
		return nil, err
	}
	body, err := r.deps.Reader.Read(ctx, conn, spec.MethodGet, norm,
		ReadOpts{TimeoutSecs: orDefault(op.TimeoutSecs, spec.DefaultTimeoutSecs),
			MaxResponseBytes: orDefault(op.MaxResponseBytes, spec.DefaultResponseBytes)})
	if err != nil {
		return nil, err
	}
	var rows []any
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("connector response must be a JSON array: %w", err)
	}
	return rows, nil
}

// execTransform merges resolved inputs into one named object. v1 defines no
// expression language beyond reference resolution; richer transforms are an
// explicit future schema decision, not a silent extension.
func (r *run) execTransform(ctx context.Context, s *spec.Step) error {
	out := map[string]any{}
	keys := make([]string, 0, len(s.Input))
	for k := range s.Input {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v, err := r.resolve(ctx, s.Input[k])
		if err != nil {
			return err
		}
		out[k] = v
	}
	r.outputs[s.Output] = out
	return nil
}

// execBranch evaluates the guard: falsy ends the workflow (done=true).
// Truthiness: false, null, "", 0, [], and {} are falsy; all else is truthy.
func (r *run) execBranch(ctx context.Context, s *spec.Step) (bool, error) {
	v, err := r.resolve(ctx, s.Condition)
	if err != nil {
		return false, err
	}
	return !truthy(v), nil
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	case []any:
		return len(t) != 0
	case map[string]any:
		return len(t) != 0
	default:
		return true
	}
}

// execLoop binds a bounded collection. v1 steps have no sub-steps, so the
// documented semantics are bounded-collect: the single input must resolve to
// an array of at most maxIterations items, bound as the output. Longer inputs
// fail closed rather than silently dropping rows.
func (r *run) execLoop(ctx context.Context, s *spec.Step) error {
	if len(s.Input) != 1 {
		return fmt.Errorf("bounded loop takes exactly one collection input")
	}
	var coll any
	for _, ref := range s.Input {
		var err error
		coll, err = r.resolve(ctx, ref)
		if err != nil {
			return err
		}
	}
	rows, ok := coll.([]any)
	if !ok {
		return fmt.Errorf("bounded loop input must resolve to an array")
	}
	if len(rows) > s.MaxIterations {
		return fmt.Errorf("collection holds %d items, exceeding maxIterations %d", len(rows), s.MaxIterations)
	}
	r.outputs[s.Output] = rows
	return nil
}

// resolve evaluates one ${...} reference against step outputs, queries, and
// entities, fetching connector data lazily with per-run caching.
func (r *run) resolve(ctx context.Context, ref string) (any, error) {
	inner := ref
	if len(inner) < 4 || inner[0] != '$' || inner[1] != '{' || inner[len(inner)-1] != '}' {
		return nil, fmt.Errorf("malformed reference %q", ref)
	}
	segs := strings.Split(inner[2:len(inner)-1], ".")
	head := segs[0]
	var base any
	if v, ok := r.outputs[head]; ok {
		base = v
	} else if r.hasQuery(head) {
		qv, err := r.queryValue(ctx, head)
		if err != nil {
			return nil, err
		}
		base = qv
	} else if r.hasEntity(head) {
		rows, err := r.entity(ctx, head)
		if err != nil {
			return nil, err
		}
		base = map[string]any{"rows": rows}
	} else {
		return nil, fmt.Errorf("unknown reference source %q", head)
	}
	cur := base
	for _, seg := range segs[1:] {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("reference %q does not resolve through non-object", ref)
		}
		v, ok := obj[seg]
		if !ok {
			return nil, fmt.Errorf("reference %q has no field %q", ref, seg)
		}
		cur = v
	}
	return cur, nil
}

// interpolate substitutes ${...} references in free text. Strings interpolate
// raw; other values interpolate as canonical JSON.
func (r *run) interpolate(ctx context.Context, text string) (string, error) {
	var out strings.Builder
	for i := 0; i < len(text); {
		if text[i] == '$' && i+1 < len(text) && text[i+1] == '{' {
			j := strings.Index(text[i+2:], "}")
			if j < 0 {
				return "", fmt.Errorf("unterminated reference in %q", text)
			}
			ref := text[i : i+2+j+1]
			v, err := r.resolve(ctx, ref)
			if err != nil {
				return "", err
			}
			if s, ok := v.(string); ok {
				out.WriteString(s)
			} else {
				enc, err := json.Marshal(v)
				if err != nil {
					return "", fmt.Errorf("interpolate: %w", err)
				}
				out.Write(enc)
			}
			i += 2 + j + 1
		} else {
			out.WriteByte(text[i])
			i++
		}
	}
	return out.String(), nil
}

// hasQuery reports whether name is a declared query.
func (r *run) hasQuery(name string) bool {
	for i := range r.spec.Data.Queries {
		if r.spec.Data.Queries[i].Name == name {
			return true
		}
	}
	return false
}

// hasEntity reports whether name is a declared entity.
func (r *run) hasEntity(name string) bool {
	for i := range r.spec.Data.Entities {
		if r.spec.Data.Entities[i].Name == name {
			return true
		}
	}
	return false
}

// entity fetches (and caches) all rows of one entity via its declared
// capability. The fetch path is "/" plus the remote entity name and must be
// allowlisted like any other request path.
func (r *run) entity(ctx context.Context, name string) ([]any, error) {
	if rows, ok := r.entities[name]; ok {
		return rows, nil
	}
	var ent *spec.Entity
	for i := range r.spec.Data.Entities {
		if r.spec.Data.Entities[i].Name == name {
			ent = &r.spec.Data.Entities[i]
		}
	}
	if ent == nil {
		return nil, fmt.Errorf("unknown entity %q", name)
	}
	if r.fixture != nil {
		v, ok := r.fixture.Input[name]
		if !ok {
			return nil, fmt.Errorf("dry-run: fixture %q has no input for entity %q", r.fixName, name)
		}
		rows, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("dry-run: fixture %q input %q must be an array", r.fixName, name)
		}
		r.entities[name] = rows
		return rows, nil
	}
	rows, err := r.readOp(ctx, ent.Source.Capability, "/"+ent.Source.Entity)
	if err != nil {
		return nil, err
	}
	r.entities[name] = rows
	return rows, nil
}

// queryValue executes one named query and wraps rows with computed aggregates.
func (r *run) queryValue(ctx context.Context, name string) (map[string]any, error) {
	var q *spec.Query
	for i := range r.spec.Data.Queries {
		if r.spec.Data.Queries[i].Name == name {
			q = &r.spec.Data.Queries[i]
		}
	}
	if q == nil {
		return nil, fmt.Errorf("unknown query %q", name)
	}
	rows, err := r.entity(ctx, q.Entity)
	if err != nil {
		return nil, err
	}
	rows, err = applyFilter(rows, q.Filter)
	if err != nil {
		return nil, err
	}
	rows, err = applySort(rows, q.Sort)
	if err != nil {
		return nil, err
	}
	if q.Limit > 0 && len(rows) > q.Limit {
		rows = rows[:q.Limit]
	}
	out := map[string]any{"rows": rows}
	for _, a := range q.Aggregates {
		out[a.As] = aggregate(a.Func, rows, a.Field)
	}
	return out, nil
}

func orDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func rowField(row any, field string) (any, bool) {
	obj, ok := row.(map[string]any)
	if !ok {
		return nil, false
	}
	v, ok := obj[field]
	return v, ok
}

func asNumber(v any) (float64, bool) {
	n, ok := v.(float64)
	return n, ok
}

func jsonEq(a, b any) bool {
	ac, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bc, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(ac) == string(bc)
}

// applyFilter keeps rows matching every predicate. A missing field never
// matches; type errors fail the query deterministically.
func applyFilter(rows []any, filters []spec.Filter) ([]any, error) {
	var out []any
	for _, row := range rows {
		keep := true
		for _, f := range filters {
			v, ok := rowField(row, f.Field)
			if !ok {
				keep = false
				break
			}
			match, err := applyOp(f.Op, v, f.Value)
			if err != nil {
				return nil, err
			}
			if !match {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, row)
		}
	}
	return out, nil
}

func applyOp(op string, field, operand any) (bool, error) {
	switch op {
	case spec.OpEq:
		return jsonEq(field, operand), nil
	case spec.OpNeq:
		return !jsonEq(field, operand), nil
	case spec.OpGt, spec.OpGte, spec.OpLt, spec.OpLte:
		a, ok1 := asNumber(field)
		b, ok2 := asNumber(operand)
		if !ok1 || !ok2 {
			return false, fmt.Errorf("operator %q requires numbers", op)
		}
		switch op {
		case spec.OpGt:
			return a > b, nil
		case spec.OpGte:
			return a >= b, nil
		case spec.OpLt:
			return a < b, nil
		default:
			return a <= b, nil
		}
	case spec.OpContains:
		s, ok1 := field.(string)
		sub, ok2 := operand.(string)
		if !ok1 || !ok2 {
			return false, fmt.Errorf("operator %q requires strings", op)
		}
		return strings.Contains(s, sub), nil
	case spec.OpIn:
		list, ok := operand.([]any)
		if !ok {
			return false, fmt.Errorf("operator %q requires an array operand", op)
		}
		for _, e := range list {
			if jsonEq(field, e) {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("unknown filter op %q", op)
	}
}

// applySort orders rows by each key in order. Missing fields fail
// deterministically; mixed-type values compare by a fixed type order so the
// result is total and stable.
func applySort(rows []any, keys []spec.SortKey) ([]any, error) {
	out := append([]any(nil), rows...)
	for _, k := range keys {
		for _, row := range out {
			if _, ok := rowField(row, k.Field); !ok {
				return nil, fmt.Errorf("sort field %q missing from a row", k.Field)
			}
		}
		desc := k.Dir == "desc"
		sort.SliceStable(out, func(i, j int) bool {
			a, _ := rowField(out[i], k.Field)
			b, _ := rowField(out[j], k.Field)
			c := compareTotal(a, b)
			if desc {
				return c > 0
			}
			return c < 0
		})
	}
	return out, nil
}

// compareTotal orders null < bool < number < string < other deterministically.
func compareTotal(a, b any) int {
	rank := func(v any) int {
		switch v.(type) {
		case nil:
			return 0
		case bool:
			return 1
		case float64:
			return 2
		case string:
			return 3
		default:
			return 4
		}
	}
	ra, rb := rank(a), rank(b)
	if ra != rb {
		if ra < rb {
			return -1
		}
		return 1
	}
	switch av := a.(type) {
	case bool:
		bv := b.(bool)
		switch {
		case av == bv:
			return 0
		case !av:
			return -1
		default:
			return 1
		}
	case float64:
		bv := b.(float64)
		switch {
		case av < bv:
			return -1
		case av > bv:
			return 1
		default:
			return 0
		}
	case string:
		return strings.Compare(av, b.(string))
	default:
		if reflect.DeepEqual(a, b) {
			return 0
		}
		return strings.Compare(jsonString(a), jsonString(b))
	}
}

func jsonString(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(raw)
}

func aggregate(fn string, rows []any, field string) any {
	switch fn {
	case "count":
		n := 0
		for _, row := range rows {
			if _, ok := rowField(row, field); ok {
				n++
			}
		}
		return n
	case "sum", "avg":
		var total float64
		n := 0
		for _, row := range rows {
			if v, ok := rowField(row, field); ok {
				if num, ok := asNumber(v); ok {
					total += num
					n++
				}
			}
		}
		if fn == "sum" {
			return total
		}
		if n == 0 {
			return nil
		}
		return total / float64(n)
	default:
		return nil
	}
}
