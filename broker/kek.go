package broker

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/PithomLabs/miniapp-studio/secrets"
)

// LoadKEK reads the deployment KEK from path: the file must be
// owner-only and hold exactly 64 hex chars (32 bytes). Failures name
// the path problem, never key material.
func LoadKEK(path string) (kek []byte, kekID string, err error) {
	raw, _, err := secrets.ReadSecureFile(path, secrets.MaxValueBytes)
	if err != nil {
		return nil, "", fmt.Errorf("broker: KEK file: %w", err)
	}
	text := strings.TrimSpace(string(raw))
	kek, err = hex.DecodeString(text)
	if err != nil || len(kek) != kekBytes {
		return nil, "", fmt.Errorf("broker: KEK file must hold exactly 64 hex chars")
	}
	return kek, KeyID(kek), nil
}
