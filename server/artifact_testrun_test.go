package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/sandbox"
	"github.com/PithomLabs/miniapp-studio/store"
)

// fakeRunner scripts container results by command shape.
type fakeRunner struct {
	testJSON []byte
	confRegs []string
	blocked  string
}

func (f *fakeRunner) Run(_ context.Context, req sandbox.Request) (sandbox.Result, error) {
	if f.blocked != "" {
		return sandbox.Result{Blocked: true, Cause: f.blocked}, nil
	}
	if len(req.Command) >= 2 && req.Command[0] == "go" && req.Command[1] == "run" {
		body := `{"registered": [`
		for i, name := range f.confRegs {
			if i > 0 {
				body += ", "
			}
			body += `"` + name + `"`
		}
		return sandbox.Result{ExitCode: 0, Stdout: []byte(body + "]}\n")}, nil
	}
	return sandbox.Result{ExitCode: 0, Stdout: f.testJSON}, nil
}

const fakePassJSON = `{"Action":"run","Package":"p","Test":"TestA"}
{"Action":"pass","Package":"p","Test":"TestA"}
{"Action":"pass","Package":"p"}
`

func withFakeRunner(t *testing.T, fake *fakeRunner) {
	t.Helper()
	prev := newTestRunner
	newTestRunner = func() sandbox.RunnerAPI { return fake }
	t.Cleanup(func() { newTestRunner = prev })
}

// TestTriggerRecordsPass pins the happy path: claim → run → parse →
// conformance → finish(passed), with the spec ops covered.
func TestTriggerRecordsPass(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	b := seedTriggerBundle(t, srv, testTenant, "trig-pass")
	withFakeRunner(t, &fakeRunner{testJSON: []byte(fakePassJSON),
		confRegs: []string{"documents.invoice.read", "purchasing.order.match"}})
	rec := doRequest(t, srv, http.MethodPost, "/console/artifact-test",
		(&url.Values{"opkey": {opkeyForTest(t)}, "artifact_id": {b.artifactID},
			"conversation_id": {b.conv}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("trigger = %d: %s", rec.Code, rec.Body.String())
	}
	owner := ownerPool(t)
	var status string
	err := store.InTenantTx(context.Background(), owner, testTenant, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(),
			`SELECT status FROM studio.artifact_test_runs WHERE tenant_id = $1 AND artifact_id = $2`,
			testTenant, b.artifactID).Scan(&status)
	})
	if err != nil || status != "passed" {
		t.Fatalf("run status = %q, err = %v", status, err)
	}
}

// TestTriggerRefusesUnverified pins the gate: unverified bundles
// never execute, and no run row is recorded.
func TestTriggerRefusesUnverified(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	b := seedTriggerBundle(t, srv, testTenant, "trig-unv")
	owner := ownerPool(t)
	err := store.InTenantTx(context.Background(), owner, testTenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(),
			`UPDATE studio.artifacts SET verified_at = NULL WHERE id = $1`, b.artifactID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	withFakeRunner(t, &fakeRunner{testJSON: []byte(fakePassJSON)})
	rec := doRequest(t, srv, http.MethodPost, "/console/artifact-test",
		(&url.Values{"opkey": {opkeyForTest(t)}, "artifact_id": {b.artifactID},
			"conversation_id": {b.conv}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("trigger = %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "not+verified") && !strings.Contains(loc, "not%20verified") {
		t.Errorf("notice lacks the refusal: %q", loc)
	}
	var n int
	err = store.InTenantTx(context.Background(), owner, testTenant, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(),
			`SELECT count(*) FROM studio.artifact_test_runs WHERE tenant_id = $1 AND artifact_id = $2`,
			testTenant, b.artifactID).Scan(&n)
	})
	if err != nil || n != 0 {
		t.Errorf("runs recorded = %d, want 0", n)
	}
}

// TestTriggerRecordsBlocked pins the honest blocked path end to end:
// the run row records blocked with the cause.
func TestTriggerRecordsBlocked(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	b := seedTriggerBundle(t, srv, testTenant, "trig-blk")
	withFakeRunner(t, &fakeRunner{blocked: "docker unavailable in test"})
	rec := doRequest(t, srv, http.MethodPost, "/console/artifact-test",
		(&url.Values{"opkey": {opkeyForTest(t)}, "artifact_id": {b.artifactID},
			"conversation_id": {b.conv}}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("trigger = %d", rec.Code)
	}
	owner := ownerPool(t)
	var status string
	err := store.InTenantTx(context.Background(), owner, testTenant, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(),
			`SELECT status FROM studio.artifact_test_runs WHERE tenant_id = $1 AND artifact_id = $2`,
			testTenant, b.artifactID).Scan(&status)
	})
	if err != nil || status != "blocked" {
		t.Fatalf("run status = %q, err = %v", status, err)
	}
}

// seedTriggerBundle reuses the decide seed with a passed-run placeholder
// removed: tests seed their own runs through the trigger.
func seedTriggerBundle(t *testing.T, srv *Server, tenant, tag string) *decideBundle {
	t.Helper()
	b := seedDecidedBundle(t, srv, tenant, tag, "passed")
	owner := ownerPool(t)
	err := store.InTenantTx(context.Background(), owner, tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(),
			`DELETE FROM studio.artifact_test_runs WHERE tenant_id = $1 AND artifact_id = $2`,
			tenant, b.artifactID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}
