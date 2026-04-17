package hash

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type APIKey struct {
	Full   string
	Prefix string
	Hashed string
}

func GenerateAPIKey() (APIKey, error) {
	// Generate 6 random bytes → 12 char hex prefix
	prefixBytes := make([]byte, 6)
	if _, err := rand.Read(prefixBytes); err != nil {
		return APIKey{}, err 
	}
	prefix := hex.EncodeToString(prefixBytes)
	
	// Generate 6 random bytes → 12 char hex prefix
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return APIKey{}, err 
	}
	secret := hex.EncodeToString(secretBytes)

	// Hash the secret with SHA-256 — what we store
	h:= sha256.Sum256([]byte(secret))
	hashed := hex.EncodeToString(h[:])

	return APIKey{
		Full: fmt.Sprintf("mk_live_%s.%s",prefix,secret),
		Prefix: prefix,
		Hashed: hashed,
	}, nil
}

// HashSecret hashes a secret for comparison during validation
func HashSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}