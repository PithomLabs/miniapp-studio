package render

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"strings"

	"github.com/PithomLabs/miniapp-studio/runner"
	"github.com/PithomLabs/miniapp-studio/spec"
)

// DataSource resolves declared bindings to values. Implementations are
// caller-scoped: dry-run supplies fixture data, live preview supplies
// execution outputs. Resolution failures render as component error states,
// never as panics or silent blanks.
type DataSource interface {
	Resolve(binding string) (any, error)
}

// Options configures one render call.
type Options struct {
	// Data supplies bound values. Required.
	Data DataSource
	// Run supplies execution results for run-result components. Nil renders
	// the empty state ("no run yet").
	Run *runner.Result
	// InvokeEndpoint receives form/action POSTs. Empty renders actions
	// disabled with a wiring note; P4 binds real routes.
	InvokeEndpoint string
}

// MapData is a DataSource backed by a fixed map, for tests and dry-run.
type MapData map[string]any

// Resolve returns the bound value or an error for unknown bindings.
func (m MapData) Resolve(binding string) (any, error) {
	v, ok := m[binding]
	if !ok {
		return nil, fmt.Errorf("unbound data %q", binding)
	}
	return v, nil
}

// Render renders one page of an admitted spec for the calling tenant. The
// tenant must equal the admission tenant or rendering is denied; the spec is
// revalidated first, so admission-time validity is never trusted blindly.
func Render(vspec runner.ValidatedSpec, tenantID, pageID string, opts Options) (string, error) {
	if tenantID == "" {
		return "", fmt.Errorf("render: tenant is required")
	}
	if vspec.Tenant() != tenantID {
		return "", fmt.Errorf("render: cross-tenant rendering denied")
	}
	if err := vspec.Revalidate(); err != nil {
		return "", err
	}
	if opts.Data == nil {
		return "", fmt.Errorf("render: data source is required")
	}
	parsed, err := vspec.Spec()
	if err != nil {
		return "", err
	}
	var page *spec.Page
	for i := range parsed.UI.Pages {
		if parsed.UI.Pages[i].ID == pageID {
			page = &parsed.UI.Pages[i]
		}
	}
	if page == nil {
		return "", fmt.Errorf("render: unknown page %q", pageID)
	}
	r := &renderer{spec: parsed, opts: opts}
	var body strings.Builder
	for i := range page.Components {
		frag, err := r.component(&page.Components[i])
		if err != nil {
			return "", fmt.Errorf("render: component %d: %w", i, err)
		}
		body.WriteString(string(frag))
	}
	return execTemplate(pageShell, map[string]any{
		"Title": page.Title,
		"Body":  template.HTML(body.String()),
	})
}

// Skeleton renders loading placeholders for one page. It performs the same
// admission, tenant, and page checks as Render; only the body differs.
func Skeleton(vspec runner.ValidatedSpec, tenantID, pageID string) (string, error) {
	if tenantID == "" {
		return "", fmt.Errorf("render: tenant is required")
	}
	if vspec.Tenant() != tenantID {
		return "", fmt.Errorf("render: cross-tenant rendering denied")
	}
	if err := vspec.Revalidate(); err != nil {
		return "", err
	}
	parsed, err := vspec.Spec()
	if err != nil {
		return "", err
	}
	var page *spec.Page
	for i := range parsed.UI.Pages {
		if parsed.UI.Pages[i].ID == pageID {
			page = &parsed.UI.Pages[i]
		}
	}
	if page == nil {
		return "", fmt.Errorf("render: unknown page %q", pageID)
	}
	var body strings.Builder
	for i := range page.Components {
		frag, err := execTemplate(skeletonFrag, map[string]any{
			"Type": page.Components[i].Type,
		})
		if err != nil {
			return "", err
		}
		body.WriteString(frag)
	}
	return execTemplate(pageShell, map[string]any{
		"Title": page.Title,
		"Body":  template.HTML(body.String()),
	})
}

type renderer struct {
	spec *spec.MiniappSpec
	opts Options
}

const pageShell = `<!DOCTYPE html><html><head><meta charset="utf-8"><title>{{.Title}}</title></head>` +
	`<body class="st-app"><main>{{.Body}}</main></body></html>`

const skeletonFrag = `<section class="st-loading" data-component="{{.Type}}"><span aria-busy="true">Loading…</span></section>`

const errorFrag = `<section class="st-error" data-component="{{.Type}}"><h2>{{.Title}}</h2><p>{{.Message}}</p></section>`

const emptyFrag = `<section class="st-empty" data-component="{{.Type}}"><h2>{{.Title}}</h2><p>No data to display.</p></section>`

func execTemplate(text string, data any) (string, error) {
	tpl, err := template.New("frag").Parse(text)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (r *renderer) errorPanel(c *spec.Component, err error) (string, error) {
	return execTemplate(errorFrag, map[string]any{
		"Type": c.Type, "Title": c.Title, "Message": err.Error(),
	})
}

func (r *renderer) emptyPanel(c *spec.Component) (string, error) {
	return execTemplate(emptyFrag, map[string]any{
		"Type": c.Type, "Title": c.Title,
	})
}

// resolveBinding re-checks that a binding head is declared, then resolves it
// through the caller-supplied data source.
func (r *renderer) resolveBinding(binding string) (any, error) {
	head := binding
	if i := strings.Index(head, "."); i >= 0 {
		head = head[:i]
	}
	declared := false
	for _, q := range r.spec.Data.Queries {
		if q.Name == head {
			declared = true
		}
	}
	for _, e := range r.spec.Data.Entities {
		if e.Name == head {
			declared = true
		}
	}
	if !declared {
		return nil, fmt.Errorf("binding %q is not declared", binding)
	}
	return r.opts.Data.Resolve(binding)
}

func isEmpty(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return t == ""
	case []any:
		return len(t) == 0
	case map[string]any:
		return len(t) == 0
	default:
		return false
	}
}

func sortedKeys(obj map[string]any) []string {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
