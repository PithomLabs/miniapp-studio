package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp-studio/artifact"
)

// BundleData is one artifact row with its member bytes loaded: the
// shared input to verification and approval. Both paths re-hash
// from these bytes; neither trusts stored digests.
type BundleData struct {
	ArtifactID string
	BuildID    string
	Kind       string
	Digest     string
	Verified   bool
	// Envelope is the manifest.json bytes; Members excludes it.
	Envelope []byte
	Members  map[string][]byte
	// ClaimFiles mirrors the claim manifest member list.
	ClaimFiles []artifact.FileEntry
}

// LoadBundle reads one artifact row plus every claimed member blob.
// Missing rows, malformed claim manifests, and unfetched members
// are errors the caller maps to its own failure shape.
func LoadBundle(ctx context.Context, tx pgx.Tx, tenant, artifactID string) (*BundleData, error) {
	var row struct {
		id       string
		build    string
		kind     string
		digest   string
		verified bool
		manifest json.RawMessage
	}
	err := tx.QueryRow(ctx,
		`SELECT id, build_id, kind, digest, verified_at IS NOT NULL, manifest
		 FROM studio.artifacts WHERE tenant_id = $1 AND id = $2`,
		tenant, artifactID).Scan(&row.id, &row.build, &row.kind, &row.digest, &row.verified, &row.manifest)
	if err != nil {
		return nil, err
	}
	var claim struct {
		Files []artifact.FileEntry `json:"files"`
	}
	if err := json.Unmarshal(row.manifest, &claim); err != nil || len(claim.Files) == 0 {
		return nil, fmt.Errorf("store: claim manifest unusable")
	}
	data := &BundleData{
		ArtifactID: row.id, BuildID: row.build, Kind: row.kind,
		Digest: row.digest, Verified: row.verified,
		Members:    make(map[string][]byte, len(claim.Files)),
		ClaimFiles: claim.Files,
	}
	for _, entry := range claim.Files {
		var blob []byte
		if err := tx.QueryRow(ctx,
			`SELECT bytes FROM studio.artifact_blobs WHERE tenant_id = $1 AND digest = $2`,
			tenant, entry.Digest).Scan(&blob); err != nil {
			return nil, fmt.Errorf("store: member %s bytes not fetched: %w", entry.Path, err)
		}
		data.Members[entry.Path] = blob
	}
	env, ok := data.Members[artifact.EnvelopeFileName]
	if !ok {
		return nil, fmt.Errorf("store: envelope bytes missing")
	}
	data.Envelope = env
	delete(data.Members, artifact.EnvelopeFileName)
	return data, nil
}
