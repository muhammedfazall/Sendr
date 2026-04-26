package helpers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// APIKey holds the three representations of a generated key.
type APIKey struct {
	Full   string // shown to user once: mk_live_<prefix>.<secret>
	Prefix string // stored in DB, used for lookup
	Hashed string // SHA-256 of the secret half — stored in DB, never exposed
}

// GenerateAPIKey creates a new API key with a random prefix and secret.
func GenerateAPIKey() (APIKey, error) {
	prefixBytes := make([]byte, 6) // 6 bytes → 12 char hex
	if _, err := rand.Read(prefixBytes); err != nil {
		return APIKey{}, fmt.Errorf("generate prefix: %w", err)
	}
	prefix := hex.EncodeToString(prefixBytes)

	secretBytes := make([]byte, 32) // 32 bytes → 64 char hex
	if _, err := rand.Read(secretBytes); err != nil {
		return APIKey{}, fmt.Errorf("generate secret: %w", err)
	}
	secret := hex.EncodeToString(secretBytes)

	h := sha256.Sum256([]byte(secret))
	hashed := hex.EncodeToString(h[:])

	return APIKey{
		Full:   fmt.Sprintf("mk_live_%s.%s", prefix, secret),
		Prefix: prefix,
		Hashed: hashed,
	}, nil
}

// HashSecret returns the SHA-256 hex digest of a secret.
// Used during validation to compare against the stored hash.
func HashSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}