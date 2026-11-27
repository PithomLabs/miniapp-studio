package fly

// Adapter/fake parity proof (B2-01A). A scripted local HTTPS double speaks
// the sprites-go v0.2.2 management dialect (POST/GET/DELETE /v1/sprites,
// verified against the SDK source); the shared contract suite runs against
// the real SDK adapter pointed at that double. This proves the adapter
// maps SDK semantics onto the identical contract the fake implements —
// it does not prove Fly's live behavior, which stays B2-01B's gate.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	sprites "github.com/superfly/sprites-go"
)

const parityToken = "parity-test-token"

// parityDouble is a scripted Sprites management API. It asserts the Bearer
// token on every request (compared in memory, never logged) and records
// the request log for wire-behavior assertions.
type parityDouble struct {
	t      *testing.T
	mu     sync.Mutex
	log    []string
	spr    map[string]*sprites.SpriteInfo
	labels map[string][]string
	// Scripts: nonzero status overrides the default; malformed corrupts
	// the next GET body.
	postStatus   int
	getStatus    int
	deleteStatus int
	malformedGet bool
}

func newParityDouble(t *testing.T) (*parityDouble, *httptest.Server) {
	t.Helper()
	d := &parityDouble{t: t, spr: map[string]*sprites.SpriteInfo{}, labels: map[string][]string{}}
	srv := httptest.NewServer(http.HandlerFunc(d.serve))
	t.Cleanup(srv.Close)
	return d, srv
}

func (d *parityDouble) requests() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.log...)
}

func (d *parityDouble) drop(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.spr, name)
}

func (d *parityDouble) serve(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+parityToken {
		d.t.Errorf("request without test bearer token: %s %s", r.Method, r.URL.Path)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	d.mu.Lock()
	d.log = append(d.log, r.Method+" "+r.URL.Path)
	d.mu.Unlock()
	if r.Method == http.MethodPost && r.URL.Path == "/v1/sprites" {
		d.serveCreate(w, r)
		return
	}
	name, ok := strings.CutPrefix(r.URL.Path, "/v1/sprites/")
	if !ok || name == "" || strings.Contains(name, "/") {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		d.serveGet(w, name)
	case http.MethodDelete:
		d.serveDelete(w, name)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (d *parityDouble) serveCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string   `json:"name"`
		Labels []string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	d.mu.Lock()
	if d.postStatus != 0 {
		st := d.postStatus
		d.mu.Unlock()
		http.Error(w, "scripted failure", st)
		return
	}
	now := time.Now().UTC()
	d.spr[req.Name] = &sprites.SpriteInfo{
		ID: req.Name, Name: req.Name, Status: "running",
		URL:    "https://sprites.test/" + req.Name,
		Labels: req.Labels, CreatedAt: now, UpdatedAt: now,
	}
	d.labels[req.Name] = req.Labels
	d.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"name": req.Name})
}

func (d *parityDouble) serveGet(w http.ResponseWriter, name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.malformedGet {
		d.malformedGet = false
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not json"))
		return
	}
	if d.getStatus != 0 {
		http.Error(w, "scripted failure", d.getStatus)
		return
	}
	info, ok := d.spr[name]
	if !ok {
		http.Error(w, "sprite not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(info)
}

func (d *parityDouble) serveDelete(w http.ResponseWriter, name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.deleteStatus != 0 {
		http.Error(w, "scripted failure", d.deleteStatus)
		return
	}
	if _, ok := d.spr[name]; !ok {
		http.Error(w, "sprite not found", http.StatusNotFound)
		return
	}
	delete(d.spr, name)
	w.WriteHeader(http.StatusNoContent)
}

// countingProvider decorates a LifecycleProvider with invocation counts.
type countingProvider struct {
	LifecycleProvider
	mu    sync.Mutex
	calls map[string]int
}

func countProvider(p LifecycleProvider) *countingProvider {
	return &countingProvider{LifecycleProvider: p, calls: map[string]int{}}
}

func (c *countingProvider) inc(op string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls[op]++
}

// CallCount implements CallCounter.
func (c *countingProvider) CallCount(op string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[op]
}

func (c *countingProvider) Provision(ctx context.Context, name string, opts SpriteOptions) error {
	c.inc("Provision")
	return c.LifecycleProvider.Provision(ctx, name, opts)
}

func (c *countingProvider) Destroy(ctx context.Context, name string) error {
	c.inc("Destroy")
	return c.LifecycleProvider.Destroy(ctx, name)
}

func (c *countingProvider) Describe(ctx context.Context, name string) (bool, error) {
	c.inc("Describe")
	return c.LifecycleProvider.Describe(ctx, name)
}

// TestLifecycleContractAdapter runs the shared suite against the real SDK
// adapter over the scripted double: the parity half of the proof. Power
// and session verbs are declared unsupported and must refuse as such.
func TestLifecycleContractAdapter(t *testing.T) {
	runLifecycleContract(t, func() LifecycleProvider {
		_, srv := newParityDouble(t)
		return countProvider(NewAdapterLifecycleProvider(NewSpriteAdapter(parityToken, srv.URL)))
	}, contractCaps{power: false, sessionLiveOnly: true})
}

func parityLifecycle(t *testing.T) (*Lifecycle, *MemoryRecorder, *parityDouble) {
	t.Helper()
	d, srv := newParityDouble(t)
	rec := &MemoryRecorder{}
	lc := NewLifecycle(
		NewAdapterLifecycleProvider(NewSpriteAdapter(parityToken, srv.URL)),
		DecisionAuthorizer{}, rec, &contractClock{at: contractTime})
	return lc, rec, d
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParityWireLog(t *testing.T) {
	lc, _, d := parityLifecycle(t)
	ctx := context.Background()
	if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{Labels: []string{"studio"}}); err != nil {
		t.Fatal(err)
	}
	// Adapter create is POST followed by a describing GET; the controller
	// adds no verification round-trip of its own.
	if got, want := d.requests(), []string{"POST /v1/sprites", "GET /v1/sprites/b1"}; !sameStrings(got, want) {
		t.Errorf("provision wire log = %v, want %v", got, want)
	}
	d.mu.Lock()
	labels := d.labels["b1"]
	d.mu.Unlock()
	if !sameStrings(labels, []string{"studio"}) {
		t.Errorf("labels sent = %v", labels)
	}
	if _, err := lc.Destroy(ctx, "b1", "k2", contractDecision(TransitionDestroy, "b1")); err != nil {
		t.Fatal(err)
	}
	if got, want := d.requests(), []string{"POST /v1/sprites", "GET /v1/sprites/b1", "DELETE /v1/sprites/b1"}; !sameStrings(got, want) {
		t.Errorf("destroy wire log = %v, want %v", got, want)
	}
}

func TestParityErrorMapping(t *testing.T) {
	ctx := context.Background()

	t.Run("Create500", func(t *testing.T) {
		lc, rec, d := parityLifecycle(t)
		d.mu.Lock()
		d.postStatus = http.StatusInternalServerError
		d.mu.Unlock()
		if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err == nil {
			t.Fatal("provision accepted, want provider error")
		}
		if st := lc.State("b1"); st != StateAbsent {
			t.Errorf("state = %q, want absent", st)
		}
		if got := rec.All(); len(got) != 0 {
			t.Errorf("recorded %d receipts, want none", len(got))
		}
	})

	t.Run("GetMalformed", func(t *testing.T) {
		lc, _, d := parityLifecycle(t)
		d.mu.Lock()
		d.malformedGet = true
		d.mu.Unlock()
		// The adapter's create-then-describe GET hits the corrupt body.
		if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err == nil {
			t.Fatal("provision accepted, want parse error")
		}
		if st := lc.State("b1"); st != StateAbsent {
			t.Errorf("state = %q, want absent", st)
		}
	})

	t.Run("Delete500", func(t *testing.T) {
		lc, _, d := parityLifecycle(t)
		if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
			t.Fatal(err)
		}
		d.mu.Lock()
		d.deleteStatus = http.StatusInternalServerError
		d.mu.Unlock()
		if _, err := lc.Destroy(ctx, "b1", "k2", contractDecision(TransitionDestroy, "b1")); err == nil {
			t.Fatal("destroy accepted, want provider error")
		}
		if st := lc.State("b1"); st != StateAwake {
			t.Errorf("state = %q, want kept awake", st)
		}
	})

	t.Run("DescribeOutageIsNotAbsence", func(t *testing.T) {
		lc, _, d := parityLifecycle(t)
		if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
			t.Fatal(err)
		}
		d.mu.Lock()
		d.getStatus = http.StatusBadGateway
		d.mu.Unlock()
		if _, err := lc.Reconcile(ctx, "b1"); err == nil {
			t.Fatal("reconcile accepted, want outage error")
		}
		if st := lc.State("b1"); st != StateAwake {
			t.Errorf("state = %q, want kept awake (outage is not absence)", st)
		}
	})
}

func TestParityDesync(t *testing.T) {
	lc, rec, d := parityLifecycle(t)
	ctx := context.Background()
	if _, err := lc.Provision(ctx, "b1", "k1", contractDecision(TransitionProvision, "b1"), SpriteOptions{}); err != nil {
		t.Fatal(err)
	}
	// The provider loses the sprite behind the controller's back.
	d.drop("b1")
	var nf *NotFoundError
	if _, err := lc.Destroy(ctx, "b1", "k2", contractDecision(TransitionDestroy, "b1")); !errors.As(err, &nf) {
		t.Fatalf("destroy err = %v, want *NotFoundError", err)
	}
	if st := lc.State("b1"); st != StateAwake {
		t.Errorf("state = %q, want kept awake", st)
	}
	if got := rec.All(); len(got) != 1 {
		t.Errorf("recorded %d receipts, want only the provision", len(got))
	}
	st, err := lc.Reconcile(ctx, "b1")
	if err != nil {
		t.Fatal(err)
	}
	if st != StateAbsent {
		t.Errorf("reconciled = %q, want absent", st)
	}
}
