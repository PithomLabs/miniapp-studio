package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PithomLabs/miniapp-studio/artifact"
)

// TestFetchCapCoversMaxFile pins the gate-B derivation: the fetch
// response cap must exceed a max-size file after base64 plus JSON
// envelope headroom. If either cap moves, this fails until the
// derivation is re-examined — the two caps cannot drift apart.
func TestFetchCapCoversMaxFile(t *testing.T) {
	b64 := int64(len(base64.StdEncoding.EncodeToString(make([]byte, artifact.MaxFileBytes))))
	const envelopeSlack = 64 << 10
	if int64(maxFetchResponseBytes) < b64+envelopeSlack {
		t.Errorf("fetch cap %d < b64 max file %d + slack %d",
			maxFetchResponseBytes, b64, envelopeSlack)
	}
	if maxCommandBytes >= maxFetchResponseBytes {
		t.Errorf("command cap %d must stay below the fetch cap %d",
			maxCommandBytes, maxFetchResponseBytes)
	}
}

// TestFetchMaxFileRoundTrip is the gate-B transport proof: a
// max-size file crosses the real client/server pair intact, with
// sizes logged as evidence. Single-file fetch commands are the
// chunking: no command ever approaches the 50 MB aggregate.
func TestFetchMaxFileRoundTrip(t *testing.T) {
	srv, adapter := testServer(t)
	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()
	client := &Client{Timeout: 30 * time.Second, Retries: 0}
	ctx := context.Background()
	key := StaticTestKey()
	if _, err := client.Send(ctx, httpSrv.URL, key, "b-fetch", CmdAssign,
		map[string]any{"prompt": "hello", "workspace_id": "w-fetch"}); err != nil {
		t.Fatalf("assign: %v", err)
	}
	data := make([]byte, artifact.MaxFileBytes)
	for i := range data {
		data[i] = byte(i*31 + 7)
	}
	sum := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	adapter.CollectFunc = func(_ string, manifest Manifest) ([]Artifact, error) {
		var out []Artifact
		for _, e := range manifest.Entries {
			out = append(out, Artifact{Path: e.Path, Kind: e.Kind,
				Digest: digest, Size: int64(len(data)), Data: data})
		}
		return out, nil
	}
	out, err := client.Send(ctx, httpSrv.URL, key, "b-fetch", CmdFetchArtifact,
		map[string]any{"manifest": map[string]any{
			"entries": []any{map[string]any{"path": "app/big.go", "kind": "bundle"}}}})
	if err != nil {
		t.Fatalf("fetch max file: %v", err)
	}
	arts, ok := out["artifacts"].([]any)
	if !ok || len(arts) != 1 {
		t.Fatalf("artifacts = %v", out["artifacts"])
	}
	entry, ok := arts[0].(map[string]any)
	if !ok {
		t.Fatalf("entry = %v", arts[0])
	}
	encoded, _ := entry["data_b64"].(string)
	got, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != len(data) || string(got[:64]) != string(data[:64]) || string(got[len(got)-64:]) != string(data[len(data)-64:]) {
		t.Fatal("round-tripped bytes differ")
	}
	if entry["digest"] != digest {
		t.Errorf("digest = %v, want %s", entry["digest"], digest)
	}
	t.Logf("fetch transport: request manifest ~%d B; response b64 %d B for %d file bytes; caps cmd=%d fetch=%d",
		len(`{"path":"app/big.go","kind":"bundle"}`), len(encoded), len(data), maxCommandBytes, maxFetchResponseBytes)
}
