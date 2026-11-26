package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/artifact"
	"github.com/PithomLabs/miniapp-studio/store"
	"github.com/PithomLabs/miniapp-studio/tools"
)

// fetchedFile is one verified member file carried from dispatch to
// settlement. settleTx persists it to the blob store in the same
// transaction that acks the command row: bytes and ack commit
// atomically, so a crash can neither lose fetched bytes nor ack an
// unstored fetch.
type fetchedFile struct {
	artifactID string
	path       string
	digest     string
	size       int64
	data       []byte
}

// fetchWire builds the exact signed payload for a fetch_artifact row:
// one manifest entry (single-file commands per the D0 transport
// freeze). Malformed rows fail fast without burning — retry cannot
// fix a corrupt row.
func fetchWire(payload map[string]any) (map[string]any, *fetchedFile, error) {
	artifactID, _ := payload["artifact_id"].(string)
	path, _ := payload["path"].(string)
	digest, _ := payload["digest"].(string)
	sizeRaw, present := payload["size"]
	size, isNum := sizeRaw.(float64)
	if artifactID == "" || path == "" || digest == "" || !present || !isNum || size < 0 {
		return nil, nil, fmt.Errorf("fetch row lacks artifact_id/path/digest/size")
	}
	if err := artifact.ValidatePath(path); err != nil {
		return nil, nil, fmt.Errorf("fetch row path refused: %v", err)
	}
	if size > float64(artifact.MaxFileBytes) {
		return nil, nil, fmt.Errorf("fetch row size %v exceeds the per-file cap", size)
	}
	// Kind labels the collection; the path identifies the member.
	// Bundle members ride kind "bundle" — the gateway kind set has
	// no per-file source kinds, and adding any would widen the
	// Builder-side contract for no Studio gain. Shape matches
	// gateway decodeManifest: {"entries": [{path, kind}]}.
	wire := map[string]any{"manifest": map[string]any{
		"entries": []any{map[string]any{"path": path, "kind": "bundle"}}}}
	want := &fetchedFile{artifactID: artifactID, path: path, digest: digest, size: int64(size)}
	return wire, want, nil
}

// parseFetchOut validates a fetch_artifact response against the claim:
// exactly one entry, path match, base64 decode, size within cap and
// equal to claimed, sha256 equal to claimed. Any failure is terminal
// for the row — a mismatched source is not retried into agreement;
// the operator re-drives with a fresh claim.
func parseFetchOut(out map[string]any, want *fetchedFile) (*fetchedFile, error) {
	raw, ok := out["artifacts"].([]any)
	if !ok || len(raw) != 1 {
		return nil, fmt.Errorf("fetch response carries %d artifacts, want exactly 1", len(raw))
	}
	entry, ok := raw[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("fetch response entry is not an object")
	}
	path, _ := entry["path"].(string)
	if path != want.path {
		return nil, fmt.Errorf("fetch response path %q does not match claimed %q", path, want.path)
	}
	encoded, _ := entry["data_b64"].(string)
	if encoded == "" {
		return nil, fmt.Errorf("fetch response for %q carries no data", path)
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("fetch response for %q is not valid base64", path)
	}
	if int64(len(data)) > artifact.MaxFileBytes {
		return nil, fmt.Errorf("fetched %q is %d bytes, over the per-file cap", path, len(data))
	}
	if int64(len(data)) != want.size {
		return nil, fmt.Errorf("fetched %q is %d bytes, claimed %d", path, len(data), want.size)
	}
	sum := sha256.Sum256(data)
	if got := "sha256:" + hex.EncodeToString(sum[:]); got != want.digest {
		return nil, fmt.Errorf("fetched %q digest mismatch (bytes differ from claim)", path)
	}
	got := *want
	got.data = data
	return &got, nil
}

// maybeAppendFetched streams artifact_fetched once the artifact's
// blobs cover every claimed member. Called from settleTx after the
// blob write, in the same transaction: bytes and receipt commit
// atomically. Already-fetched bundles (re-fetch rows) append
// nothing — the receipt marks first completion only.
func maybeAppendFetched(ctx context.Context, tx pgx.Tx, tenant, conv string, fetched *fetchedFile) error {
	var manifest json.RawMessage
	var buildID string
	if err := tx.QueryRow(ctx,
		`SELECT manifest, build_id FROM studio.artifacts WHERE tenant_id = $1 AND id = $2`,
		tenant, fetched.artifactID).Scan(&manifest, &buildID); err != nil {
		return err
	}
	var claim struct {
		Files []artifact.FileEntry `json:"files"`
	}
	if err := json.Unmarshal(manifest, &claim); err != nil || len(claim.Files) == 0 {
		return nil
	}
	for _, entry := range claim.Files {
		var present bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM studio.artifact_blobs WHERE tenant_id = $1 AND digest = $2)`,
			tenant, entry.Digest).Scan(&present); err != nil {
			return err
		}
		if !present {
			return nil
		}
	}
	var streamed bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM studio.conversation_stream
		 WHERE tenant_id = $1 AND conversation_id = $2 AND kind = 'artifact_fetched' AND ref_id = $3)`,
		tenant, conv, fetched.artifactID).Scan(&streamed); err != nil {
		return err
	}
	if streamed {
		return nil
	}
	if _, err := store.AppendStream(ctx, tx, tenant, conv, "artifact_fetched", fetched.artifactID, buildID); err != nil {
		return err
	}
	// First completion runs verification in the same transaction:
	// bytes, verified_at, evidence, and both receipts commit
	// atomically — a crash can neither strand fetched-but-
	// unverified bytes nor verify unstored ones. A bundle that
	// fails verification is a Passed=false outcome, not an error:
	// the bytes stay stored with their failure evidence.
	_, err := tools.VerifyBundleTx(ctx, tx, tenant, fetched.artifactID)
	return err
}
