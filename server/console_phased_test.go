package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/store"
)

// Phase D console scenes render the persisted review state: the Spec
// scene shows the canonical saved version, Validate shows checks,
// test runs, and the decide forms, Connect reports connector state,
// and Evidence carries the artifact ledger. These tests seed coherent
// bundles and drive the real routes.

func TestConsoleValidateSceneShowsBundle(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	b := seedDecidedBundle(t, srv, testTenant, "d5val", "passed")
	// Give the verification row real check evidence.
	owner := ownerPool(t)
	err := store.InTenantTx(context.Background(), owner, testTenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(),
			`UPDATE studio.artifact_verifications SET checks = $1
			 WHERE tenant_id = $2 AND artifact_id = $3`,
			`[{"check":"identity.root","pass":true},{"check":"admission.app","pass":true}]`,
			testTenant, b.artifactID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := doRequest(t, srv, http.MethodGet,
		"/console?scene=validate&conversation="+b.conv, "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("validate = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		b.digest, "verified", "identity.root", "admission.app",
		"/console/artifact-test", "/console/artifact-decide",
		`name="artifact_id" value="` + b.artifactID + `"`,
		"Run tests", "Approve", "Reject",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("validate lacks %q", want)
		}
	}
}

func TestConsoleValidateSceneEmpty(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	conv := consoleConversation(t, srv, cookie, "Fresh")
	rec := doRequest(t, srv, http.MethodGet,
		"/console?scene=validate&conversation="+conv, "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("validate = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No artifacts") {
		t.Errorf("empty validate lacks the honest empty state")
	}
}

func TestConsoleDecideFormApproves(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	b := seedDecidedBundle(t, srv, testTenant, "d5appr", "passed")
	rec := doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
		(&url.Values{
			"opkey":           {opkeyForTest(t)},
			"artifact_id":     {b.artifactID},
			"conversation_id": {b.conv},
			"decision":        {"approved"},
		}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("decide = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "scene=evidence") {
		t.Errorf("decide redirects to %q, want the evidence scene", loc)
	}
	rec = doRequest(t, srv, http.MethodGet,
		"/console?scene=evidence&conversation="+b.conv, "", cookie)
	body := rec.Body.String()
	for _, want := range []string{"Artifact ledger (1)", b.digest, "approved", testBinding} {
		if !strings.Contains(body, want) {
			t.Errorf("evidence lacks %q", want)
		}
	}
}

func TestConsoleDecideFormRejects(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	b := seedDecidedBundle(t, srv, testTenant, "d5rej", "failed")
	rec := doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
		(&url.Values{
			"opkey":           {opkeyForTest(t)},
			"artifact_id":     {b.artifactID},
			"conversation_id": {b.conv},
			"decision":        {"rejected"},
			"note":            {"tests fail on retry logic"},
		}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("decide = %d, want 302", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet,
		"/console?scene=evidence&conversation="+b.conv, "", cookie)
	body := rec.Body.String()
	for _, want := range []string{"Artifact ledger (1)", b.digest, "rejected"} {
		if !strings.Contains(body, want) {
			t.Errorf("evidence lacks %q", want)
		}
	}
}

func TestConsoleSpecSceneShowsCanonical(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	b := seedDecidedBundle(t, srv, testTenant, "d5spec", "passed")
	// Before any decision there is no saved version: honest absence.
	rec := doRequest(t, srv, http.MethodGet,
		"/console?scene=spec&conversation="+b.conv, "", cookie)
	if !strings.Contains(rec.Body.String(), "No saved spec") {
		t.Errorf("spec shows a version before any decision")
	}
	rec = doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
		(&url.Values{
			"opkey":           {opkeyForTest(t)},
			"artifact_id":     {b.artifactID},
			"conversation_id": {b.conv},
			"decision":        {"approved"},
		}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("decide = %d, want 302", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet,
		"/console?scene=spec&conversation="+b.conv, "", cookie)
	body := rec.Body.String()
	for _, want := range []string{"invoice-review", "version 1", "Spec hash", "Invoice Review Desk"} {
		if !strings.Contains(body, want) {
			t.Errorf("spec lacks %q", want)
		}
	}
}

func TestConsoleConnectSceneUnconfigured(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	conv := consoleConversation(t, srv, cookie, "Shop")
	rec := doRequest(t, srv, http.MethodGet,
		"/console?scene=connect&conversation="+conv, "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("connect = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Connector manager unconfigured") {
		t.Errorf("connect lacks the honest unconfigured state")
	}
}

func TestConsolePulseShowsArtifactDecision(t *testing.T) {
	srv, _ := testServer(t)
	cookie := login(t, srv, testToken)
	b := seedDecidedBundle(t, srv, testTenant, "d5pulse", "passed")
	rec := doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
		(&url.Values{
			"opkey":           {opkeyForTest(t)},
			"artifact_id":     {b.artifactID},
			"conversation_id": {b.conv},
			"decision":        {"approved"},
		}).Encode(), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("decide = %d, want 302", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodGet,
		"/console?scene=run&conversation="+b.conv, "", cookie)
	body := rec.Body.String()
	for _, want := range []string{"artifact_decided", "approved by " + testBinding} {
		if !strings.Contains(body, want) {
			t.Errorf("pulse lacks %q", want)
		}
	}
}

// The status tool must carry the full review bundle the scenes
// render from: decision detail, checks, the latest report, and the
// canonical spec only after a decision saves one.
func TestArtifactStatusCarriesReviewBundle(t *testing.T) {
	srv, _ := testServer(t)
	b := seedDecidedBundle(t, srv, testTenant, "d5status", "passed")
	key, err := newOpKey()
	if err != nil {
		t.Fatal(err)
	}
	out, err := srv.invokeToolTenant(context.Background(), testTenant,
		"artifact.status", map[string]any{"conversation_id": b.conv}, key)
	if err != nil {
		t.Fatal(err)
	}
	arts, _ := out["artifacts"].([]any)
	if len(arts) != 1 {
		t.Fatalf("artifacts = %d, want 1", len(arts))
	}
	a, _ := arts[0].(map[string]any)
	if a["digest"] != b.digest || a["verified"] != true {
		t.Errorf("summary = %v/%v, want digest+verified", a["digest"], a["verified"])
	}
	if a["app_id"] != "invoice-review" || a["spec_hash"] == "" {
		t.Errorf("app/spec identity = %v/%v", a["app_id"], a["spec_hash"])
	}
	if _, ok := out["canonical_spec"]; ok && out["canonical_spec"] != nil {
		t.Errorf("canonical spec present before any decision")
	}
	if _, err := srv.DecideArtifact(context.Background(), testTenant, testBinding,
		b.artifactID, "approved", ""); err != nil {
		t.Fatal(err)
	}
	key, err = newOpKey()
	if err != nil {
		t.Fatal(err)
	}
	out, err = srv.invokeToolTenant(context.Background(), testTenant,
		"artifact.status", map[string]any{"conversation_id": b.conv}, key)
	if err != nil {
		t.Fatal(err)
	}
	arts, _ = out["artifacts"].([]any)
	a, _ = arts[0].(map[string]any)
	if a["decision"] != "approved" || a["decided_by"] != testBinding || a["decided_at"] == "" {
		t.Errorf("decision detail = %v/%v/%v", a["decision"], a["decided_by"], a["decided_at"])
	}
	canon, _ := out["canonical_spec"].(map[string]any)
	if canon == nil || canon["app_id"] != "invoice-review" || canon["version"] != 1.0 {
		t.Errorf("canonical spec = %v", out["canonical_spec"])
	}
}
