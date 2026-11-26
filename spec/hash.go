package spec

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// CanonicalJSON decodes raw and re-encodes it in canonical form: object keys
// sorted recursively, no insignificant whitespace, no HTML escaping. Two
// documents with the same content hash identically regardless of key order or
// formatting in the input.
//
// Number model: JSON numbers decode to float64, so integers beyond 2^53 and
// alternate spellings (1e3 vs 1000) normalize to the same value. Key sorting
// relies on encoding/json's documented map-key ordering. Both properties are
// locked by tests; do not migrate number handling without a canonical-form
// migration plan.
func CanonicalJSON(raw []byte) ([]byte, error) {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(decoded); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// ContentHashOf hashes a raw spec document identically to ContentHash:
// strict decode first (unknown fields fail closed), then the same
// canonicalization. Callers pass validated documents; the strict
// decode only fires on corruption.
func ContentHashOf(raw []byte) (string, error) {
	var parsed MiniappSpec
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&parsed); err != nil {
		return "", err
	}
	return ContentHash(&parsed)
}

// ContentHash returns the hex SHA-256 of the spec's canonical JSON, excluding
// metadata.contentHash itself. The hash is computed over the decoded spec, so
// input formatting never affects it. Editing a spec must produce a new hash;
// runs bind the hash they started with.
func ContentHash(spec *MiniappSpec) (string, error) {
	raw, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", err
	}
	if meta, ok := decoded["metadata"].(map[string]any); ok {
		delete(meta, "contentHash")
	}
	canonical, err := CanonicalJSON(mustMarshal(decoded))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func mustMarshal(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		panic("spec: marshal decoded JSON: " + err.Error())
	}
	return raw
}
