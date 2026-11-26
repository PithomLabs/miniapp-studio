package artifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/PithomLabs/miniapp-studio/spec"
	"github.com/PithomLabs/miniapp/contract"
)

// Envelope is the parsed Studio envelope (manifest.json). Correlation
// binds the bundle to its tenant/build/conversation/turn; Toolchain
// declares the build inputs; Files lists every member EXCEPT the
// envelope itself, whose digest is computed over its bytes and added
// to the root file list by ComputeRoot. This one-direction flow is
// what makes the root acyclic: envelope → member digests → root.
type Envelope struct {
	Contract       string      `json:"contract"`
	TenantID       string      `json:"tenant_id"`
	BuildID        string      `json:"build_id"`
	ConversationID string      `json:"conversation_id"`
	TurnID         string      `json:"turn_id"`
	AppID          string      `json:"app_id"`
	AppVersion     string      `json:"app_version"`
	Toolchain      Toolchain   `json:"toolchain"`
	Spec           FileRef     `json:"spec"`
	Files          []FileEntry `json:"files"`
}

// FileRef pins one file by path and digest.
type FileRef struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

// selfDigestKeys must never appear in an envelope: a self-digest
// would either be untrusted input confused with the computed root or
// a hashing cycle. Presence refuses with EContract.
var selfDigestKeys = []string{"digest", "artifactDigest", "artifact_digest", "root", "id"}

// ParseEnvelope decodes raw strictly: exactly one JSON document, no
// unknown fields, no trailing data, no self-digest keys, contract
// version exact. Deterministic: same input, same output or same
// errors.
func ParseEnvelope(raw []byte) (*Envelope, []Error) {
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, []Error{Errorf(EContract, "$", "invalid JSON: %v", err)}
	}
	for _, key := range selfDigestKeys {
		if _, ok := probe[key]; ok {
			return nil, []Error{Errorf(EContract, key, "envelope must not carry a self-digest")}
		}
	}
	var env Envelope
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&env); err != nil {
		return nil, []Error{Errorf(EContract, "$", "schema violation: %v", err)}
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, []Error{Errorf(EContract, "$", "trailing data after JSON document")}
	}
	if env.Contract != ContractVersion {
		return nil, []Error{Errorf(EContract, "contract",
			"want %q, got %q", ContractVersion, env.Contract)}
	}
	return &env, nil
}

// rootInput is the hashed shape. files is complete (members +
// envelope itself) and sorted by path before hashing.
type rootInput struct {
	Contract       string      `json:"contract"`
	TenantID       string      `json:"tenant_id"`
	BuildID        string      `json:"build_id"`
	ConversationID string      `json:"conversation_id"`
	TurnID         string      `json:"turn_id"`
	AppID          string      `json:"app_id"`
	AppVersion     string      `json:"app_version"`
	Toolchain      Toolchain   `json:"toolchain"`
	Spec           FileRef     `json:"spec"`
	Files          []FileEntry `json:"files"`
}

// ComputeRoot verifies member bytes against the envelope and returns
// the artifact identity: "sha256:" hex over the canonical root
// input. files maps member path → stored bytes and must contain
// exactly the envelope's member list (the envelope itself is
// envelopeRaw, never part of the map). Every digest and size is
// re-derived — declared values are cross-checked, never trusted.
func ComputeRoot(envelopeRaw []byte, files map[string][]byte) (string, []Error) {
	env, errs := ParseEnvelope(envelopeRaw)
	if errs != nil {
		return "", errs
	}
	if len(env.Files) == 0 {
		return "", []Error{Errorf(EContract, "files", "member list is empty")}
	}
	if len(files) != len(env.Files) {
		return "", []Error{Errorf(EContract, "files",
			"got %d member blobs, envelope lists %d", len(files), len(env.Files))}
	}
	type hashed struct {
		path   string
		digest string
		size   int64
	}
	var members []hashed
	for _, entry := range env.Files {
		data, ok := files[entry.Path]
		if !ok {
			return "", []Error{Errorf(EDigest, entry.Path, "member bytes missing")}
		}
		if err := contract.Digest(entry.Digest).Validate("digest"); err != nil {
			return "", []Error{Errorf(EDigest, entry.Path, "malformed declared digest")}
		}
		sum := sha256.Sum256(data)
		got := "sha256:" + hex.EncodeToString(sum[:])
		if got != entry.Digest {
			return "", []Error{Errorf(EDigest, entry.Path, "bytes do not match the declared digest")}
		}
		if int64(len(data)) != entry.Size {
			return "", []Error{Errorf(EDigest, entry.Path,
				"size %d does not match declared %d", len(data), entry.Size)}
		}
		members = append(members, hashed{path: entry.Path, digest: got, size: int64(len(data))})
	}
	envSum := sha256.Sum256(envelopeRaw)
	envDigest := "sha256:" + hex.EncodeToString(envSum[:])
	full := make([]FileEntry, 0, len(members)+1)
	for _, m := range members {
		full = append(full, FileEntry{Path: m.path, Digest: m.digest, Size: m.size})
	}
	full = append(full, FileEntry{Path: EnvelopeFileName, Digest: envDigest, Size: int64(len(envelopeRaw))})
	sort.Slice(full, func(i, j int) bool { return full[i].Path < full[j].Path })
	input := rootInput{
		Contract: env.Contract, TenantID: env.TenantID, BuildID: env.BuildID,
		ConversationID: env.ConversationID, TurnID: env.TurnID,
		AppID: env.AppID, AppVersion: env.AppVersion, Toolchain: env.Toolchain,
		Spec: env.Spec, Files: full,
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return "", []Error{Errorf(EContract, "$", "cannot encode root input: %v", err)}
	}
	canonical, err := spec.CanonicalJSON(raw)
	if err != nil {
		return "", []Error{Errorf(EContract, "$", "cannot canonicalize root input: %v", err)}
	}
	sum := sha256.Sum256(canonical)
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(sum[:])), nil
}
