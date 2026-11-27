package broker

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Blob layout: 0x01 || nonce(12) || sealed(plaintext+tag). The AAD
// binds every ciphertext to its (tenant, provider) owner, so a row
// copied or confused across tenants fails decryption instead of
// minting for the wrong tenant.
const (
	blobVersion  = 0x01
	nonceSize    = 12
	kekBytes     = 32
	maxPlaintext = 16 * 1024
)

// KeyID fingerprints a KEK for the kek_id column: rotation
// generations stay distinguishable without exposing key material.
func KeyID(kek []byte) string {
	sum := sha256.Sum256(kek)
	return hex.EncodeToString(sum[:4])
}

// Seal encrypts plaintext under kek for one (tenant, provider).
func Seal(kek []byte, tenant, provider string, plaintext []byte) ([]byte, error) {
	if len(kek) != kekBytes {
		return nil, fmt.Errorf("broker: KEK must be %d bytes", kekBytes)
	}
	if tenant == "" || provider == "" {
		return nil, fmt.Errorf("broker: seal requires tenant and provider")
	}
	if len(plaintext) == 0 || len(plaintext) > maxPlaintext {
		return nil, fmt.Errorf("broker: plaintext must be 1..%d bytes", maxPlaintext)
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("broker: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("broker: gcm: %w", err)
	}
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("broker: nonce: %w", err)
	}
	aad := aadFor(tenant, provider)
	out := make([]byte, 0, 1+nonceSize+len(plaintext)+gcm.Overhead())
	out = append(out, blobVersion)
	out = append(out, nonce...)
	out = gcm.Seal(out, nonce, plaintext, aad)
	return out, nil
}

// Open decrypts a Seal blob, verifying version, length, and AAD.
func Open(kek []byte, tenant, provider string, blob []byte) ([]byte, error) {
	if len(kek) != kekBytes {
		return nil, fmt.Errorf("broker: KEK must be %d bytes", kekBytes)
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("broker: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("broker: gcm: %w", err)
	}
	if len(blob) < 1+nonceSize+gcm.Overhead() || blob[0] != blobVersion {
		return nil, fmt.Errorf("broker: malformed credential blob")
	}
	plain, err := gcm.Open(nil, blob[1:1+nonceSize], blob[1+nonceSize:], aadFor(tenant, provider))
	if err != nil {
		return nil, fmt.Errorf("broker: credential blob failed authentication")
	}
	return plain, nil
}

func aadFor(tenant, provider string) []byte {
	return []byte(tenant + "\x00" + provider)
}
