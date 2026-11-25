package render

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/PithomLabs/miniapp-studio/spec"
)

// component dispatches strictly on the validated discriminator. Unknown
// types fail the whole render: the catalog is closed and admission already
// guarantees membership, so an unknown type here means programmer error.
func (r *renderer) component(c *spec.Component) (string, error) {
	switch c.Type {
	case spec.ComponentDashboard:
		return r.dashboard(c)
	case spec.ComponentMetricCard:
		return r.metricCard(c)
	case spec.ComponentTable:
		return r.table(c)
	case spec.ComponentRecordDetail:
		return r.recordDetail(c)
	case spec.ComponentForm:
		return r.form(c)
	case spec.ComponentActionButton:
		return r.actionButton(c)
	case spec.ComponentChart:
		return r.chart(c)
	case spec.ComponentPermissionsCard:
		return r.permissionsCard(c)
	case spec.ComponentRunResult:
		return r.runResult(c)
	case spec.ComponentStatusPanel:
		return r.statusPanel(c)
	default:
		return "", fmt.Errorf("unknown component type %q", c.Type)
	}
}

const dashboardTpl = `<section class="st-ok" data-component="dashboard"><h2>{{.Title}}</h2><p>{{.Summary}}</p></section>`

func (r *renderer) dashboard(c *spec.Component) (string, error) {
	summary := fmt.Sprintf("%d pages", len(r.spec.UI.Pages))
	if c.Binding != "" {
		v, err := r.resolveBinding(c.Binding)
		if err != nil {
			return r.errorPanel(c, err)
		}
		if isEmpty(v) {
			return r.emptyPanel(c)
		}
		summary = summarize(v)
	}
	return execTemplate(dashboardTpl, map[string]any{"Title": c.Title, "Summary": summary})
}

const metricTpl = `<section class="st-ok" data-component="metric-card"><h2>{{.Title}}</h2><p class="st-metric">{{.Value}}</p></section>`

func (r *renderer) metricCard(c *spec.Component) (string, error) {
	v, err := r.resolveBinding(c.Binding)
	if err != nil {
		return r.errorPanel(c, err)
	}
	if isEmpty(v) {
		return r.emptyPanel(c)
	}
	return execTemplate(metricTpl, map[string]any{"Title": c.Title, "Value": summarize(v)})
}

// summarize renders one bound value as display text: scalars raw, objects
// with row arrays as counts, anything else as compact JSON.
func summarize(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", t), "0"), ".")
	case bool:
		return fmt.Sprintf("%v", t)
	case []any:
		return fmt.Sprintf("%d records", len(t))
	case map[string]any:
		if rows, ok := t["rows"].([]any); ok {
			return fmt.Sprintf("%d records", len(rows))
		}
		return fmt.Sprintf("%d fields", len(t))
	default:
		return fmt.Sprintf("%v", v)
	}
}

const tableTpl = `<section class="st-ok" data-component="table"><h2>{{.Title}}</h2>` +
	`<table><thead><tr>{{range .Columns}}<th>{{.}}</th>{{end}}</tr></thead>` +
	`<tbody>{{range .Rows}}<tr>{{range .}}<td>{{.}}</td>{{end}}</tr>{{end}}</tbody></table></section>`

func (r *renderer) table(c *spec.Component) (string, error) {
	v, err := r.resolveBinding(c.Binding)
	if err != nil {
		return r.errorPanel(c, err)
	}
	rows, err := rowsOf(v)
	if err != nil {
		return r.errorPanel(c, err)
	}
	if len(rows) == 0 {
		return r.emptyPanel(c)
	}
	table := make([][]any, 0, len(rows))
	for _, row := range rows {
		obj, ok := row.(map[string]any)
		if !ok {
			return r.errorPanel(c, fmt.Errorf("row is not an object"))
		}
		cells := make([]any, 0, len(c.Columns))
		for _, col := range c.Columns {
			val, ok := obj[col]
			if !ok {
				val = ""
			}
			cells = append(cells, cellText(val))
		}
		table = append(table, cells)
	}
	return execTemplate(tableTpl, map[string]any{
		"Title": c.Title, "Columns": c.Columns, "Rows": table,
	})
}

// rowsOf unwraps query/entity values to row arrays.
func rowsOf(v any) ([]any, error) {
	switch t := v.(type) {
	case []any:
		return t, nil
	case map[string]any:
		rows, ok := t["rows"].([]any)
		if !ok {
			return nil, fmt.Errorf("value has no rows array")
		}
		return rows, nil
	default:
		return nil, fmt.Errorf("value is not tabular")
	}
}

func cellText(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", t), "0"), ".")
	case bool:
		return fmt.Sprintf("%v", t)
	default:
		return fmt.Sprintf("%v", v)
	}
}

const detailTpl = `<section class="st-ok" data-component="record-detail"><h2>{{.Title}}</h2><dl>` +
	`{{range .Fields}}<dt>{{.Name}}</dt><dd>{{.Value}}</dd>{{end}}</dl></section>`

type detailField struct {
	Name  string
	Value string
}

func (r *renderer) recordDetail(c *spec.Component) (string, error) {
	v, err := r.resolveBinding(c.Binding)
	if err != nil {
		return r.errorPanel(c, err)
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return r.errorPanel(c, fmt.Errorf("value is not an object"))
	}
	if len(obj) == 0 {
		return r.emptyPanel(c)
	}
	names := c.Fields
	if len(names) == 0 {
		names = sortedKeys(obj)
	}
	fields := make([]detailField, 0, len(names))
	for _, name := range names {
		val, ok := obj[name]
		if !ok {
			val = ""
		}
		fields = append(fields, detailField{Name: name, Value: cellText(val)})
	}
	return execTemplate(detailTpl, map[string]any{"Title": c.Title, "Fields": fields})
}

const formTpl = `<section class="st-ok" data-component="form"><h2>{{.Title}}</h2>` +
	`<form method="post" action="{{.Endpoint}}">` +
	`{{range .Fields}}<label>{{.}}<input name="{{.}}" type="text"></label>{{end}}` +
	`<input type="hidden" name="workflow" value="{{.Action}}">` +
	`<button type="submit">{{.Submit}}</button></form></section>`

const disabledFormTpl = `<section class="st-empty" data-component="{{.Type}}"><h2>{{.Title}}</h2>` +
	`<p>Action wiring pending: routes bind in a later phase.</p></section>`

func (r *renderer) form(c *spec.Component) (string, error) {
	if r.opts.InvokeEndpoint == "" {
		return execTemplate(disabledFormTpl, map[string]any{"Type": c.Type, "Title": c.Title})
	}
	return execTemplate(formTpl, map[string]any{
		"Title": c.Title, "Endpoint": r.opts.InvokeEndpoint,
		"Fields": c.Fields, "Action": c.Action, "Submit": c.Title,
	})
}

const buttonTpl = `<section class="st-ok" data-component="action-button"><h2>{{.Title}}</h2>` +
	`<form method="post" action="{{.Endpoint}}">` +
	`<input type="hidden" name="workflow" value="{{.Action}}">` +
	`<button type="submit">{{.Title}}</button></form></section>`

func (r *renderer) actionButton(c *spec.Component) (string, error) {
	if r.opts.InvokeEndpoint == "" {
		return execTemplate(disabledFormTpl, map[string]any{"Type": c.Type, "Title": c.Title})
	}
	return execTemplate(buttonTpl, map[string]any{
		"Title": c.Title, "Endpoint": r.opts.InvokeEndpoint, "Action": c.Action,
	})
}

const chartTpl = `<section class="st-ok" data-component="chart"><h2>{{.Title}}</h2>{{.SVG}}</section>`

// chart renders a deterministic SVG bar chart over the first numeric field
// found across the bound rows. No client script is involved.
func (r *renderer) chart(c *spec.Component) (string, error) {
	v, err := r.resolveBinding(c.Binding)
	if err != nil {
		return r.errorPanel(c, err)
	}
	rows, err := rowsOf(v)
	if err != nil || len(rows) == 0 {
		if err != nil {
			return r.errorPanel(c, err)
		}
		return r.emptyPanel(c)
	}
	field := ""
scan:
	for _, row := range rows {
		obj, ok := row.(map[string]any)
		if !ok {
			continue
		}
		for _, k := range sortedKeys(obj) {
			if _, ok := obj[k].(float64); ok {
				field = k
				break scan
			}
		}
	}
	if field == "" {
		return r.emptyPanel(c)
	}
	var max float64
	vals := make([]float64, 0, len(rows))
	for _, row := range rows {
		n, _ := row.(map[string]any)[field].(float64)
		vals = append(vals, n)
		if n > max {
			max = n
		}
	}
	if max == 0 {
		max = 1
	}
	var bars strings.Builder
	const w, h = 320, 120
	for i, n := range vals {
		bw := w / len(vals)
		bh := int(n / max * float64(h-20))
		fmt.Fprintf(&bars, `<rect x="%d" y="%d" width="%d" height="%d"/>`, i*bw, h-bh, bw-2, bh)
	}
	// Safe to mark: every dynamic fragment is a formatted number or was
	// escaped above; no raw spec or provider text reaches the markup.
	svg := fmt.Sprintf(`<svg viewBox="0 0 %d %d" role="img" aria-label="%s">%s</svg>`,
		w, h, template.HTMLEscapeString(c.Title+" by "+field), bars.String())
	return execTemplate(chartTpl, map[string]any{"Title": c.Title, "SVG": template.HTML(svg)})
}

const permsTpl = `<section class="st-ok" data-component="permissions-card"><h2>{{.Title}}</h2>` +
	`<p>Declared by this spec; live connection status appears in Connectors.</p><ul>` +
	`{{range .Ops}}<li>{{.}}</li>{{end}}</ul></section>`

// permissionsCard renders the app's declared operations honestly labeled as
// declared — never presented as live gateway state.
func (r *renderer) permissionsCard(c *spec.Component) (string, error) {
	if _, err := r.resolveBinding(c.Binding); err != nil {
		return r.errorPanel(c, err)
	}
	ops := make([]string, 0, len(r.spec.Capabilities.Operations))
	for _, op := range r.spec.Capabilities.Operations {
		ops = append(ops, op.ID+" "+op.Method+" "+strings.Join(op.Paths, ", "))
	}
	if len(ops) == 0 {
		return r.emptyPanel(c)
	}
	return execTemplate(permsTpl, map[string]any{"Title": c.Title, "Ops": ops})
}

const runTpl = `<section class="st-ok" data-component="run-result"><h2>{{.Title}}</h2>` +
	`<p>Workflow {{.Workflow}}: {{.State}} (spec {{.Hash}})</p>` +
	`{{range .Messages}}<p>{{.}}</p>{{end}}</section>`

func (r *renderer) runResult(c *spec.Component) (string, error) {
	if r.opts.Run == nil {
		return r.emptyPanel(c)
	}
	state := "finished early"
	if r.opts.Run.Completed {
		state = "completed"
	}
	return execTemplate(runTpl, map[string]any{
		"Title": c.Title, "Workflow": r.opts.Run.WorkflowID,
		"State": state, "Hash": r.opts.Run.SpecHash, "Messages": r.opts.Run.Messages,
	})
}

const statusTpl = `<section class="st-ok" data-component="status-panel"><h2>{{.Title}}</h2><p>{{.Value}}</p></section>`

func (r *renderer) statusPanel(c *spec.Component) (string, error) {
	if c.Binding == "" {
		return execTemplate(statusTpl, map[string]any{"Title": c.Title, "Value": "ok"})
	}
	v, err := r.resolveBinding(c.Binding)
	if err != nil {
		return r.errorPanel(c, err)
	}
	if isEmpty(v) {
		return r.emptyPanel(c)
	}
	return execTemplate(statusTpl, map[string]any{"Title": c.Title, "Value": summarize(v)})
}
