package gateway

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FileKeys resolves per-build HMAC keys from a JSON object mapping build
// ids to hex-encoded keys. The file must be owner-only (0600 or stricter);
// anything group- or world-readable refuses to load, because these keys
// authenticate Studio commands and sign Studio-bound callbacks.
type FileKeys struct {
	keys map[string][]byte
}

// LoadKeys reads and validates a keys file.
func LoadKeys(path string) (*FileKeys, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("gateway: keys file: %w", err)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("gateway: keys file %s is group/world-readable (mode %o)", path, fi.Mode().Perm())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gateway: keys file: %w", err)
	}
	var doc map[string]string
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("gateway: keys file malformed")
	}
	keys := map[string][]byte{}
	for build, hexKey := range doc {
		key, err := hex.DecodeString(hexKey)
		if err != nil || len(key) < 16 {
			return nil, fmt.Errorf("gateway: bad key for build %q", build)
		}
		keys[build] = key
	}
	return &FileKeys{keys: keys}, nil
}

// KeyFor resolves one build's key, or refuses unknown builds without
// distinguishing them further (no oracle for scanners).
func (k *FileKeys) KeyFor(buildID string) ([]byte, error) {
	key, ok := k.keys[buildID]
	if !ok || len(key) == 0 {
		return nil, fmt.Errorf("gateway: unknown build")
	}
	return key, nil
}

// LoadKeysDir merges every *.json keys file in a directory (one file per
// build, as provisioning writes them). Later files win on collision, but
// collisions indicate a provisioning bug — file names embed build ids.
func LoadKeysDir(dir string) (*FileKeys, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("gateway: keys dir: %w", err)
	}
	merged := &FileKeys{keys: map[string][]byte{}}
	for _, e := range entries {
		if e.IsDir() || len(e.Name()) < 6 || e.Name()[len(e.Name())-5:] != ".json" {
			continue
		}
		one, err := LoadKeys(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		for build, key := range one.keys {
			merged.keys[build] = key
		}
	}
	return merged, nil
}

// ReloadingKeys resolves per-build keys by re-reading the keys dir on
// every lookup, so provisioned key files land without a gateway
// restart (F7 refresh path). Key files are tiny and commands are few,
// so the re-read cost is negligible; a malformed dir errors the lookup
// instead of serving stale keys.
type ReloadingKeys struct {
	Dir string
}

// KeyFor re-reads the keys dir and resolves one build's key.
func (k ReloadingKeys) KeyFor(buildID string) ([]byte, error) {
	keys, err := LoadKeysDir(k.Dir)
	if err != nil {
		return nil, err
	}
	return keys.KeyFor(buildID)
}

// DeriveKey derives a per-build HMAC key from the organization secret:
// HMAC-SHA256(orgSecret, "miniapp-gateway/v1:" + buildID). Studio derives
// on demand (never stored); provisioning delivers the derived key to the
// builder, which holds it in its keys dir. Domain separation keeps these
// keys disjoint from every other use of the org secret.
func DeriveKey(orgSecret []byte, buildID string) []byte {
	return hmacSum(orgSecret, []byte("miniapp-gateway/v1:"+buildID))
}

// KDFProvider resolves per-build keys by derivation. Production Studio
// configures this from the org secret file; empty secrets refuse.
type KDFProvider struct {
	Secret []byte
}

// KeyFor derives one build's key.
func (p KDFProvider) KeyFor(buildID string) ([]byte, error) {
	if len(p.Secret) == 0 || buildID == "" {
		return nil, fmt.Errorf("gateway: unknown build")
	}
	return DeriveKey(p.Secret, buildID), nil
}
