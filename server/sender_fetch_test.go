package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/artifact"
	"github.com/PithomLabs/miniapp-studio/gateway"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/tools"
)

func fetchRowPayload() map[string]any {
	return map[string]any{
		"artifact_id": "a-1", "path": "app/main.go",
		"digest": "sha256:" + strings.Repeat("a", 64), "size": float64(100),
	}
}

func TestFetchWireShape(t *testing.T) {
	wire, want, err := fetchWire(fetchRowPayload())
	if err != nil {
		t.Fatalf("valid row refused: %v", err)
	}
	man, ok := wire["manifest"].(map[string]any)
	if !ok {
		t.Fatalf("wire manifest = %v", wire["manifest"])
	}
	entries, ok := man["entries"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("wire entries = %v", man["entries"])
	}
	entry, ok := entries[0].(map[string]any)
	if !ok || entry["path"] != "app/main.go" || entry["kind"] != "bundle" {
		t.Fatalf("wire entry = %v", entries[0])
	}
	if want.path != "app/main.go" || want.size != 100 || want.artifactID != "a-1" {
		t.Fatalf("claim expectation = %+v", want)
	}
}

func TestFetchWireMalformed(t *testing.T) {
	base := fetchRowPayload()
	mutate := func(k string, v any) map[string]any {
		out := make(map[string]any, len(base))
		for key, val := range base {
			out[key] = val
		}
		if v == nil {
			delete(out, k)
		} else {
			out[k] = v
		}
		return out
	}
	cases := map[string]map[string]any{
		"missing artifact": mutate("artifact_id", nil),
		"missing path":     mutate("path", nil),
		"missing digest":   mutate("digest", nil),
		"missing size":     mutate("size", nil),
		"traversal path":   mutate("path", "../escape.go"),
		"absolute path":    mutate("path", "/etc/passwd"),
		"oversize":         mutate("size", float64(artifact.MaxFileBytes+1)),
		"negative size":    mutate("size", float64(-1)),
		"non-Go file":      mutate("path", "app/tool.py"),
	}
	for name, payload := range cases {
		if _, _, err := fetchWire(payload); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func fetchResponse(t *testing.T, path string, data []byte) map[string]any {
	t.Helper()
	return map[string]any{"artifacts": []any{map[string]any{
		"path": path, "kind": "bundle",
		"digest":   digestOf(data),
		"size":     float64(len(data)),
		"data_b64": base64.StdEncoding.EncodeToString(data),
	}}}
}

func digestOf(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func TestParseFetchOutTable(t *testing.T) {
	data := []byte("package main\n")
	want := &fetchedFile{artifactID: "a-1", path: "app/main.go", digest: digestOf(data), size: int64(len(data))}
	got, err := parseFetchOut(fetchResponse(t, "app/main.go", data), want)
	if err != nil {
		t.Fatalf("valid response refused: %v", err)
	}
	if string(got.data) != string(data) || got.digest != want.digest {
		t.Fatalf("parsed = %+v", got)
	}
	big := make([]byte, artifact.MaxFileBytes+1)
	cases := map[string]map[string]any{
		"no artifacts key": {},
		"empty list":       {"artifacts": []any{}},
		"two entries": {"artifacts": []any{
			map[string]any{"path": "app/main.go"}, map[string]any{"path": "app/other.go"}}},
		"path mismatch": fetchResponse(t, "app/other.go", data),
		"bad base64": {"artifacts": []any{map[string]any{
			"path": "app/main.go", "data_b64": "!!!not-b64"}}},
		"oversize": fetchResponse(t, "app/main.go", big),
	}
	for name, out := range cases {
		if _, err := parseFetchOut(out, want); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	// Size mismatch with correct digest shape: claim says 100, bytes are 14.
	sizeOff := &fetchedFile{artifactID: "a-1", path: "app/main.go", digest: digestOf(data), size: 100}
	if _, err := parseFetchOut(fetchResponse(t, "app/main.go", data), sizeOff); err == nil {
		t.Error("size mismatch accepted")
	}
	// Digest mismatch with correct size.
	digestOff := &fetchedFile{artifactID: "a-1", path: "app/main.go",
		digest: "sha256:" + strings.Repeat("0", 64), size: int64(len(data))}
	if _, err := parseFetchOut(fetchResponse(t, "app/main.go", data), digestOff); err == nil {
		t.Error("digest mismatch accepted")
	}
}

// seedFetchArtifact inserts a bundle artifact row for fetch tests.
func seedFetchArtifact(t *testing.T, w *senderWorld, tenant, build, artifact string) {
	t.Helper()
	ctx := context.Background()
	err := store.InTenantTx(ctx, w.srv.pool, tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO studio.artifacts (id, tenant_id, build_id, kind, digest) VALUES ($1, $2, $3, 'bundle', $4)`,
			artifact, tenant, build, "sha256:"+strings.Repeat("e", 64))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func fetchBlob(t *testing.T, w *senderWorld, tenant, digest string) []byte {
	t.Helper()
	var data []byte
	err := store.InTenantTx(context.Background(), w.srv.pool, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(),
			`SELECT bytes FROM studio.artifact_blobs WHERE tenant_id = $1 AND digest = $2`,
			tenant, digest).Scan(&data)
	})
	if err != nil {
		t.Fatalf("blob read: %v", err)
	}
	return data
}

func fetchBlobCount(t *testing.T, w *senderWorld, tenant string) int {
	t.Helper()
	var n int
	err := store.InTenantTx(context.Background(), w.srv.pool, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(),
			`SELECT count(*) FROM studio.artifact_blobs WHERE tenant_id = $1`, tenant).Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// TestSenderFetchRoundTrip is the D1 spine: a fetch row dispatches,
// the gateway serves bytes, the row acks, and the blob lands
// byte-identical in the content store.
func TestSenderFetchRoundTrip(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "fetch")
	seedFetchArtifact(t, w, testTenant, build, "a-fetch")
	data := []byte("package main\n\n// fetched\n")
	digest := digestOf(data)
	w.adapter.CollectFunc = func(_ string, manifest gateway.Manifest) ([]gateway.Artifact, error) {
		if len(manifest.Entries) != 1 || manifest.Entries[0].Path != "app/main.go" {
			t.Errorf("collect manifest = %+v", manifest.Entries)
		}
		return []gateway.Artifact{{Path: "app/main.go", Kind: "bundle",
			Digest: digest, Size: int64(len(data)), Data: data}}, nil
	}
	rowID := w.insertCommand(t, testTenant, build, gateway.CmdFetchArtifact, map[string]any{
		"artifact_id": "a-fetch", "path": "app/main.go", "digest": digest, "size": float64(len(data)),
	}, "queued", 0)
	w.snd.PollOnce(ctx)
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "acked" {
		t.Fatalf("row = %s (%s), want acked", row.state, row.lastError)
	}
	if got := fetchBlob(t, w, testTenant, digest); string(got) != string(data) {
		t.Fatal("stored blob differs from served bytes")
	}
}

// TestSenderFetchTamper proves digest enforcement end to end: wrong
// bytes settle terminally and store nothing.
func TestSenderFetchTamper(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "tamper")
	seedFetchArtifact(t, w, testTenant, build, "a-tamper")
	digest := digestOf([]byte("real bytes"))
	w.adapter.CollectFunc = func(_ string, manifest gateway.Manifest) ([]gateway.Artifact, error) {
		return []gateway.Artifact{{Path: "app/main.go", Kind: "bundle",
			Digest: digest, Size: 10, Data: []byte("WRONG!!!!!")}}, nil
	}
	rowID := w.insertCommand(t, testTenant, build, gateway.CmdFetchArtifact, map[string]any{
		"artifact_id": "a-tamper", "path": "app/main.go", "digest": digest, "size": float64(10),
	}, "queued", 0)
	w.snd.PollOnce(ctx)
	row := w.getCommand(t, testTenant, rowID)
	if row.state != "failed" {
		t.Fatalf("row = %s, want failed", row.state)
	}
	if !strings.Contains(row.lastError, "digest mismatch") {
		t.Errorf("error = %q, want digest mismatch", row.lastError)
	}
	if n := fetchBlobCount(t, w, testTenant); n != 0 {
		t.Errorf("blobs = %d, want 0 (nothing stored on mismatch)", n)
	}
}

// TestSenderFetchFailFast proves malformed rows and vanished
// artifacts settle terminally without burning an attempt or
// touching the network.
func TestSenderFetchFailFast(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	_, build := w.openBuild(t, testTenant, "failfast")
	bad := w.insertCommand(t, testTenant, build, gateway.CmdFetchArtifact, map[string]any{
		"artifact_id": "a-missing",
	}, "queued", 0)
	w.snd.PollOnce(ctx)
	row := w.getCommand(t, testTenant, bad)
	if row.state != "failed" || row.attempts != 0 {
		t.Errorf("malformed row = %s attempts=%d, want failed/0", row.state, row.attempts)
	}
	seedFetchArtifact(t, w, testTenant, build, "a-gone")
	gone := w.insertCommand(t, testTenant, build, gateway.CmdFetchArtifact, map[string]any{
		"artifact_id": "a-never", "path": "app/main.go",
		"digest": "sha256:" + strings.Repeat("a", 64), "size": float64(10),
	}, "queued", 0)
	w.snd.PollOnce(ctx)
	row = w.getCommand(t, testTenant, gone)
	if row.state != "failed" || row.attempts != 0 {
		t.Errorf("vanished artifact row = %s attempts=%d, want failed/0", row.state, row.attempts)
	}
	if len(w.adapter.Collects) != 0 {
		t.Errorf("gateway collects = %d, want 0 (fail fast precedes dispatch)", len(w.adapter.Collects))
	}
}

// TestSenderFetchCompletionVerifies closes the Fetch→Verify link: a
// coherent claim ingested through build.ingest_event, fetched file by
// file through the sender, verifies automatically on completion —
// verified_at set, passed evidence persisted, both stream receipts
// present. Before the hook, fetched bundles sat unverified forever:
// test and decide both refuse them, and nothing else sets the bit.
func TestSenderFetchCompletionVerifies(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	conv, build := w.openBuild(t, testTenant, "vlink")
	cb := buildCoherentBundle(t, testTenant, conv, build, "t-vlink", "vlink-marker")
	files := []any{}
	for _, e := range cb.files {
		files = append(files, map[string]any{
			"path": e.Path, "digest": e.Digest, "size": float64(e.Size)})
	}
	files = append(files, map[string]any{
		"path":   artifact.EnvelopeFileName,
		"digest": digestHex(cb.envRaw), "size": float64(len(cb.envRaw))})
	w.invoke(t, testTenant, tools.ToolBuildIngestEvent, map[string]any{
		"build_id": build, "event_id": "e-vlink", "type": "agent.artifact_ready",
		"payload": map[string]any{"artifacts": []any{map[string]any{
			"kind": "bundle", "digest": cb.digest,
			"size_bytes": float64(100), "files": files}}},
	}, "s-vlink-claim")
	byPath := map[string][]byte{artifact.EnvelopeFileName: cb.envRaw}
	for p, d := range cb.members {
		byPath[p] = d
	}
	w.adapter.CollectFunc = func(_ string, manifest gateway.Manifest) ([]gateway.Artifact, error) {
		if len(manifest.Entries) != 1 {
			return nil, fmt.Errorf("collect manifest has %d entries, want 1", len(manifest.Entries))
		}
		data, ok := byPath[manifest.Entries[0].Path]
		if !ok {
			return nil, fmt.Errorf("unknown path %q", manifest.Entries[0].Path)
		}
		return []gateway.Artifact{{Path: manifest.Entries[0].Path, Kind: "bundle",
			Digest: digestHex(data), Size: int64(len(data)), Data: data}}, nil
	}
	for i := 0; i < len(byPath)+2; i++ {
		w.snd.PollOnce(ctx)
	}
	var artifactID string
	var verified bool
	err := store.InTenantTx(ctx, w.srv.pool, testTenant, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT id, verified_at IS NOT NULL FROM studio.artifacts
			 WHERE tenant_id = $1 AND build_id = $2 AND digest = $3`,
			testTenant, build, cb.digest).Scan(&artifactID, &verified); err != nil {
			return err
		}
		var passed bool
		if err := tx.QueryRow(ctx,
			`SELECT passed FROM studio.artifact_verifications
			 WHERE tenant_id = $1 AND artifact_id = $2`,
			testTenant, artifactID).Scan(&passed); err != nil {
			return fmt.Errorf("verification evidence: %w", err)
		}
		if !passed {
			return fmt.Errorf("verification passed = false")
		}
		for _, kind := range []string{"artifact_fetched", "artifact_verified"} {
			var n int
			if err := tx.QueryRow(ctx,
				`SELECT count(*) FROM studio.conversation_stream
				 WHERE tenant_id = $1 AND conversation_id = $2 AND kind = $3 AND ref_id = $4`,
				testTenant, conv, kind, artifactID).Scan(&n); err != nil {
				return err
			}
			if n != 1 {
				return fmt.Errorf("stream %s rows = %d, want 1", kind, n)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Error("verified_at still NULL after fetch completion")
	}
}

// TestSenderFetchCompletionFailureKeepsBytes proves the failure side
// of the hook: a bundle whose bytes fail verification keeps its
// stored bytes and its failure evidence — Passed=false is an
// outcome, never a settlement error that would unwind the fetch.
func TestSenderFetchCompletionFailureKeepsBytes(t *testing.T) {
	w := newSenderWorld(t)
	ctx := context.Background()
	conv, build := w.openBuild(t, testTenant, "vfail")
	cb := buildCoherentBundle(t, testTenant, conv, build, "t-vfail", "vfail-marker")
	// Corrupt one member AFTER the claim commits to digests: fetch
	// still verifies per-file digests, so serve the corrupt bytes
	// under a matching claim by rebuilding the claim around them —
	// then identity fails against the pinned root instead.
	bad := map[string][]byte{}
	for p, d := range cb.members {
		bad[p] = d
	}
	bad["app/main.go"] = []byte("package main\n\n// tampered\n\nfunc main() {}\n")
	files := []any{}
	for p, d := range bad {
		files = append(files, map[string]any{
			"path": p, "digest": digestHex(d), "size": float64(len(d))})
	}
	files = append(files, map[string]any{
		"path":   artifact.EnvelopeFileName,
		"digest": digestHex(cb.envRaw), "size": float64(len(cb.envRaw))})
	w.invoke(t, testTenant, tools.ToolBuildIngestEvent, map[string]any{
		"build_id": build, "event_id": "e-vfail", "type": "agent.artifact_ready",
		"payload": map[string]any{"artifacts": []any{map[string]any{
			"kind": "bundle", "digest": cb.digest,
			"size_bytes": float64(100), "files": files}}},
	}, "s-vfail-claim")
	byPath := map[string][]byte{artifact.EnvelopeFileName: cb.envRaw}
	for p, d := range bad {
		byPath[p] = d
	}
	w.adapter.CollectFunc = func(_ string, manifest gateway.Manifest) ([]gateway.Artifact, error) {
		data, ok := byPath[manifest.Entries[0].Path]
		if !ok {
			return nil, fmt.Errorf("unknown path %q", manifest.Entries[0].Path)
		}
		return []gateway.Artifact{{Path: manifest.Entries[0].Path, Kind: "bundle",
			Digest: digestHex(data), Size: int64(len(data)), Data: data}}, nil
	}
	for i := 0; i < len(byPath)+2; i++ {
		w.snd.PollOnce(ctx)
	}
	err := store.InTenantTx(ctx, w.srv.pool, testTenant, func(tx pgx.Tx) error {
		var artifactID string
		var verified bool
		if err := tx.QueryRow(ctx,
			`SELECT id, verified_at IS NOT NULL FROM studio.artifacts
			 WHERE tenant_id = $1 AND build_id = $2 AND digest = $3`,
			testTenant, build, cb.digest).Scan(&artifactID, &verified); err != nil {
			return err
		}
		if verified {
			return fmt.Errorf("tampered bundle verified")
		}
		var passed bool
		if err := tx.QueryRow(ctx,
			`SELECT passed FROM studio.artifact_verifications
			 WHERE tenant_id = $1 AND artifact_id = $2`,
			testTenant, artifactID).Scan(&passed); err != nil {
			return fmt.Errorf("verification evidence: %w", err)
		}
		if passed {
			return fmt.Errorf("tampered bundle evidence passed")
		}
		var blobs int
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM studio.artifact_blobs WHERE tenant_id = $1`,
			testTenant).Scan(&blobs); err != nil {
			return err
		}
		if blobs != len(byPath) {
			return fmt.Errorf("blobs = %d, want %d (bytes kept with failure evidence)", blobs, len(byPath))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
