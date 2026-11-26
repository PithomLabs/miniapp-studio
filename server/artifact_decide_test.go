package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/artifact"
	"github.com/PithomLabs/miniapp-studio/store"
)

// decideBundle seeds a coherent verified bundle: app row, artifact
// row, member blobs, a passed verification row, and a terminal test
// run. Bytes are built with the production identity code so the
// seed is coherent by construction; tamper tests mutate after.
type decideBundle struct {
	conv, build, artifactID, digest string
	members                         map[string][]byte
	spec                            []byte
}

func seedDecidedBundle(t *testing.T, srv *Server, tenant, tag, runStatus string) *decideBundle {
	return seedDecidedBundleMarked(t, srv, tenant, tag, runStatus, "")
}

// coherentBundle is a bundle's bytes without any rows: the digest,
// members, envelope, and member list, coherent by construction
// through the production identity code. Seeds persist it; the
// sender link test fetches it through the real pipeline.
type coherentBundle struct {
	digest  string
	members map[string][]byte
	envRaw  []byte
	files   []artifact.FileEntry
	spec    []byte
}

func digestHex(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// buildCoherentBundle assembles bundle bytes bound to one
// tenant/conversation/build/turn. marker unshares the Go bytes.
func buildCoherentBundle(t *testing.T, tenant, conv, build, turn, marker string) *coherentBundle {
	t.Helper()
	raw, err := os.ReadFile("../spec/testdata/valid_invoice_review.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc["app"].(map[string]any)["owner"] = tenant
	specRaw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	mf := map[string]any{
		"id": "invoice-review", "version": "1.0.0", "description": "d",
		"tools": []any{
			map[string]any{"name": "documents.invoice.read", "description": "d",
				"class": "A", "substrate": "go",
				"inputSchema": map[string]any{"type": "object"}, "outputSchema": map[string]any{"type": "object"}},
			map[string]any{"name": "purchasing.order.match", "description": "d",
				"class": "A", "substrate": "go",
				"inputSchema": map[string]any{"type": "object"}, "outputSchema": map[string]any{"type": "object"}},
		},
		"deployment": map[string]any{"tools": map[string]any{
			"documents.invoice.read": map[string]any{"placement": "machine",
				"profile": map[string]any{"name": "default", "digest": "sha256:" + strings.Repeat("0", 64)}},
			"purchasing.order.match": map[string]any{"placement": "machine",
				"profile": map[string]any{"name": "default", "digest": "sha256:" + strings.Repeat("0", 64)}},
		}},
	}
	mfRaw, err := json.Marshal(mf)
	if err != nil {
		t.Fatal(err)
	}
	gomod := []byte("module studio-artifact/" + tenant + "/invoice-review\n\ngo " + artifact.PinnedGoVersion +
		"\n\nrequire " + artifact.MiniappModulePath + " " + artifact.PinnedMiniapp + "\n")
	gosum := []byte(artifact.MiniappGoSumZip + "\n" + artifact.MiniappGoSumMod + "\n")
	mainGo := "package main\n\nfunc main() {}\n"
	if marker != "" {
		mainGo = "package main\n\n// " + marker + "\n\nfunc main() {}\n"
	}
	members := map[string][]byte{
		artifact.SpecFileName:     specRaw,
		artifact.ManifestFileName: mfRaw,
		artifact.GoModFile:        gomod,
		artifact.GoSumFile:        gosum,
		"app/main.go":             []byte(mainGo),
		"app/main_test.go":        []byte("package main\n\nimport \"testing\"\n\nfunc TestOk(t *testing.T) {}\n"),
	}
	var files []artifact.FileEntry
	for path, data := range members {
		files = append(files, artifact.FileEntry{Path: path, Digest: digestHex(data), Size: int64(len(data))})
	}
	envMap := map[string]any{
		"contract": artifact.ContractVersion, "tenant_id": tenant, "build_id": build,
		"conversation_id": conv, "turn_id": turn,
		"app_id": "invoice-review", "app_version": "1.0.0",
		"toolchain": map[string]any{"Go": artifact.PinnedGoVersion, "Miniapp": artifact.PinnedMiniapp},
		"spec":      map[string]any{"path": artifact.SpecFileName, "digest": digestHex(specRaw)},
		"files":     files,
	}
	envRaw, err := json.Marshal(envMap)
	if err != nil {
		t.Fatal(err)
	}
	root, errs := artifact.ComputeRoot(envRaw, members)
	if errs != nil {
		t.Fatalf("fixture root: %v", errs)
	}
	return &coherentBundle{digest: root, members: members, envRaw: envRaw, files: files, spec: specRaw}
}

// seedDecidedBundleMarked adds a unique marker comment so the bundle's
// bytes (and hence its blob rows) are unshared: tamper tests must not
// corrupt fixtures other tests rely on.
func seedDecidedBundleMarked(t *testing.T, srv *Server, tenant, tag, runStatus, marker string) *decideBundle {
	t.Helper()
	ctx := context.Background()
	owner := ownerPool(t)
	cb := buildCoherentBundle(t, tenant, "c-"+tag, "b-"+tag, "t-"+tag, marker)
	b := &decideBundle{conv: "c-" + tag, build: "b-" + tag,
		artifactID: "a-" + tag, digest: cb.digest, members: cb.members, spec: cb.spec}
	err := store.InTenantTx(ctx, owner, tenant, func(tx pgx.Tx) error {
		batch := []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO studio.conversations (id, tenant_id) VALUES ($1, $2)`, []any{b.conv, tenant}},
			{`INSERT INTO studio.builds (id, tenant_id, conversation_id, status) VALUES ($1, $2, $3, 'succeeded')`,
				[]any{b.build, tenant, b.conv}},
			// Production births every build with turn 1
			// (createBuildTx); conversation.get requires it.
			{`INSERT INTO studio.turns (id, tenant_id, build_id, n, status) VALUES ($1, $2, $3, 1, 'closed')`,
				[]any{"t-" + tag, tenant, b.build}},
			{`INSERT INTO studio.apps (id, tenant_id, slug, name, owner_identity) VALUES ('invoice-review', $1, 'invoice-review', 'd', 'op')
			 ON CONFLICT (id) DO NOTHING`,
				[]any{tenant}},
			{`INSERT INTO studio.artifacts (id, tenant_id, build_id, kind, digest, verified_at, manifest)
			 VALUES ($1, $2, $3, 'bundle', $4, now(), $5)`,
				[]any{b.artifactID, tenant, b.build, cb.digest, claimJSON(t, cb.files, cb.envRaw)}},
			{`INSERT INTO studio.artifact_verifications (id, tenant_id, artifact_id, check_set_version, passed, checks)
			 VALUES ($1, $2, $3, 'v1', true, '[]')`, []any{"v-" + tag, tenant, b.artifactID}},
			{`INSERT INTO studio.artifact_test_runs (id, tenant_id, artifact_id, status, input_digest, finished_at)
			 VALUES ($1, $2, $3, $4, $5, now())`,
				[]any{"tr-" + tag, tenant, b.artifactID, runStatus, cb.digest}},
		}
		for _, s := range batch {
			if _, err := tx.Exec(ctx, s.sql, s.args...); err != nil {
				return err
			}
		}
		storeBlob := func(path string, data []byte) error {
			_, err := tx.Exec(ctx,
				`INSERT INTO studio.artifact_blobs (tenant_id, digest, bytes, size_bytes)
				 VALUES ($1, $2, $3, $4) ON CONFLICT (tenant_id, digest) DO NOTHING`,
				tenant, digestHex(data), data, len(data))
			return err
		}
		for path, data := range cb.members {
			if err := storeBlob(path, data); err != nil {
				return err
			}
		}
		return storeBlob(artifact.EnvelopeFileName, cb.envRaw)
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// claimJSON builds the claim manifest: member files plus the envelope
// itself, mirroring ingest.
func claimJSON(t *testing.T, files []artifact.FileEntry, envRaw []byte) string {
	t.Helper()
	sum := sha256.Sum256(envRaw)
	all := append(append([]artifact.FileEntry{}, files...),
		artifact.FileEntry{Path: artifact.EnvelopeFileName,
			Digest: "sha256:" + hex.EncodeToString(sum[:]), Size: int64(len(envRaw))})
	raw, err := json.Marshal(map[string]any{"files": all})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func decideForm(t *testing.T, artifactID, decision, note, conv string) string {
	t.Helper()
	return (&url.Values{"opkey": {opkeyForTest(t)}, "artifact_id": {artifactID},
		"decision": {decision}, "note": {note}, "conversation_id": {conv}}).Encode()
}

func decisionCount(t *testing.T, tenant, artifactID string) int {
	t.Helper()
	owner := ownerPool(t)
	var n int
	err := store.InTenantTx(context.Background(), owner, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(),
			`SELECT count(*) FROM studio.approvals WHERE tenant_id = $1 AND artifact_id = $2`,
			tenant, artifactID).Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// TestDecideApproveRoundTrip pins the D4 happy path: verified +
// passed tests approve with a cited spec version.
func TestDecideApproveRoundTrip(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	b := seedDecidedBundle(t, srv, testTenant, "appr", "passed")
	rec := doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
		decideForm(t, b.artifactID, "approved", "looks good", b.conv), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("decide = %d: %s", rec.Code, rec.Body.String())
	}
	if n := decisionCount(t, testTenant, b.artifactID); n != 1 {
		t.Fatalf("decisions = %d, want 1", n)
	}
	owner := ownerPool(t)
	var decidedBy, digest, note string
	var version int
	err := store.InTenantTx(context.Background(), owner, testTenant, func(tx pgx.Tx) error {
		if err := tx.QueryRow(context.Background(),
			`SELECT decided_by, artifact_digest, note FROM studio.approvals
			 WHERE tenant_id = $1 AND artifact_id = $2`,
			testTenant, b.artifactID).Scan(&decidedBy, &digest, &note); err != nil {
			return err
		}
		return tx.QueryRow(context.Background(),
			`SELECT max(version) FROM studio.app_specs WHERE tenant_id = $1 AND app_id = 'invoice-review'`,
			testTenant).Scan(&version)
	})
	if err != nil {
		t.Fatal(err)
	}
	if decidedBy != testBinding || digest != b.digest || note != "looks good" {
		t.Errorf("decision row = %q %q %q", decidedBy, digest, note)
	}
	if version != 1 {
		t.Errorf("spec version = %d, want 1 (saved at approval)", version)
	}
	// Replay: same outcome converges without a second row.
	rec = doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
		decideForm(t, b.artifactID, "approved", "again", b.conv), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("replay = %d, want 302", rec.Code)
	}
	if n := decisionCount(t, testTenant, b.artifactID); n != 1 {
		t.Errorf("decisions after replay = %d, want 1", n)
	}
}

// TestDecideConflict pins 409 on a changed outcome.
func TestDecideConflict(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	b := seedDecidedBundle(t, srv, testTenant, "conf", "passed")
	rec := doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
		decideForm(t, b.artifactID, "approved", "", b.conv), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("approve = %d", rec.Code)
	}
	rec = doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
		decideForm(t, b.artifactID, "rejected", "changed mind", b.conv), cookie)
	if rec.Code != http.StatusConflict {
		t.Fatalf("changed outcome = %d, want 409", rec.Code)
	}
}

// TestDecideGating pins every precondition: unknown, unverified,
// untested, failed-tests approve/refuse matrix.
func TestDecideGating(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	post := func(artifactID, decision, note, conv string) int {
		return doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
			decideForm(t, artifactID, decision, note, conv), cookie).Code
	}
	if code := post("a-nope", "approved", "", "c-x"); code != http.StatusNotFound {
		t.Errorf("unknown artifact = %d, want 404", code)
	}
	unverified := seedDecidedBundle(t, srv, testTenant, "unv", "passed")
	owner := ownerPool(t)
	err := store.InTenantTx(context.Background(), owner, testTenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(),
			`UPDATE studio.artifacts SET verified_at = NULL WHERE id = $1`, unverified.artifactID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if code := post(unverified.artifactID, "approved", "", unverified.conv); code != http.StatusBadRequest {
		t.Errorf("unverified approve = %d, want 400", code)
	}
	failed := seedDecidedBundle(t, srv, testTenant, "fail", "failed")
	if code := post(failed.artifactID, "approved", "", failed.conv); code != http.StatusBadRequest {
		t.Errorf("failed-tests approve = %d, want 400", code)
	}
	if code := post(failed.artifactID, "rejected", "bad tests", failed.conv); code != http.StatusFound {
		t.Errorf("failed-tests reject = %d, want 302", code)
	}
	if loc := doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
		decideForm(t, failed.artifactID, "rejected", "bad tests again", failed.conv),
		cookie).Header().Get("Location"); !strings.Contains(loc, "Rejected") {
		t.Errorf("reject redirect lacks the rework notice: %q", loc)
	}
	norun := seedDecidedBundle(t, srv, testTenant, "norun", "passed")
	err = store.InTenantTx(context.Background(), owner, testTenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(),
			`DELETE FROM studio.artifact_test_runs WHERE tenant_id = $1 AND artifact_id = $2`,
			testTenant, norun.artifactID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if code := post(norun.artifactID, "approved", "", norun.conv); code != http.StatusBadRequest {
		t.Errorf("no-run approve = %d, want 400", code)
	}
}

// TestDecideConcurrentConverges pins the race: parallel same-outcome
// submits yield one row and all redirect; mixed outcomes yield one
// row with exactly one 409.
func TestDecideConcurrentConverges(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	race := func(tag, runStatus string, decisions []string) (codes []int) {
		b := seedDecidedBundle(t, srv, testTenant, tag, runStatus)
		var mu sync.Mutex
		var wg sync.WaitGroup
		for i, d := range decisions {
			wg.Add(1)
			go func(i int, decision string) {
				defer wg.Done()
				rec := doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
					decideForm(t, b.artifactID, decision, "", b.conv), cookie)
				mu.Lock()
				codes = append(codes, rec.Code)
				mu.Unlock()
				_ = i
			}(i, d)
		}
		wg.Wait()
		if n := decisionCount(t, testTenant, b.artifactID); n != 1 {
			t.Errorf("%s: decisions = %d, want 1", tag, n)
		}
		return codes
	}
	same := race("race-same", "passed", []string{"approved", "approved", "approved"})
	for _, c := range same {
		if c != http.StatusFound {
			t.Errorf("same-outcome race: code %d, want all 302", c)
		}
	}
	// Mixed race on passed tests: one winner redirects, the loser
	// 409s (order varies; the multiset is deterministic).
	mixed := race("race-mixed", "passed", []string{"approved", "rejected"})
	var found, conflicted int
	for _, c := range mixed {
		switch c {
		case http.StatusFound:
			found++
		case http.StatusConflict:
			conflicted++
		default:
			t.Errorf("mixed race: unexpected code %d", c)
		}
	}
	if found != 1 || conflicted != 1 {
		t.Errorf("mixed race: %d redirects + %d conflicts, want 1+1", found, conflicted)
	}
}

// TestDecideTamperedBytes pins corruption fail-closed: bytes swapped
// after verification refuse with 500 and decide nothing.
func TestDecideTamperedBytes(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	b := seedDecidedBundleMarked(t, srv, testTenant, "tamp", "passed", "tamper-marker")
	owner := ownerPool(t)
	err := store.InTenantTx(context.Background(), owner, testTenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(),
			`UPDATE studio.artifact_blobs AS b SET bytes = 'tampered' FROM studio.artifacts AS a,
			 jsonb_array_elements(a.manifest->'files') AS f
			 WHERE a.id = $1 AND b.tenant_id = a.tenant_id AND b.digest = f->>'digest'
			 AND f->>'path' = 'app/main.go'`, b.artifactID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
		decideForm(t, b.artifactID, "approved", "", b.conv), cookie)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("tampered decide = %d, want 500", rec.Code)
	}
	if n := decisionCount(t, testTenant, b.artifactID); n != 0 {
		t.Errorf("decisions = %d, want 0", n)
	}
}

// TestDecideCitesExistingSpec pins single-canonical-spec: approving a
// second artifact with identical spec bytes cites the saved version
// instead of saving a duplicate.
func TestDecideCitesExistingSpec(t *testing.T) {
	srv := callbackServer(t)
	cookie := login(t, srv, testToken)
	first := seedDecidedBundle(t, srv, testTenant, "cite1", "passed")
	rec := doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
		decideForm(t, first.artifactID, "approved", "", first.conv), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("first approve = %d", rec.Code)
	}
	second := seedDecidedBundle(t, srv, testTenant, "cite2", "passed")
	rec = doRequest(t, srv, http.MethodPost, "/console/artifact-decide",
		decideForm(t, second.artifactID, "approved", "", second.conv), cookie)
	if rec.Code != http.StatusFound {
		t.Fatalf("second approve = %d", rec.Code)
	}
	owner := ownerPool(t)
	var n int
	err := store.InTenantTx(context.Background(), owner, testTenant, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(),
			`SELECT count(*) FROM studio.app_specs WHERE tenant_id = $1 AND app_id = 'invoice-review'`,
			testTenant).Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("spec versions = %d, want 1 (cite, not duplicate)", n)
	}
}
