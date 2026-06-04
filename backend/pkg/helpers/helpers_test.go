package helpers

import (
	"strings"
	"testing"
)

func TestGenerateAPIKeyFormat(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey returned error: %v", err)
	}

	if !strings.HasPrefix(key.Full, "mk_live_") {
		t.Fatalf("expected mk_live_ prefix, got %q", key.Full)
	}

	parts := strings.SplitN(key.Full, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("expected full key in format prefix.secret, got %q", key.Full)
	}

	gotPrefix := strings.TrimPrefix(parts[0], "mk_live_")
	if gotPrefix != key.Prefix {
		t.Fatalf("prefix mismatch: mk_live_%s vs prefix %q", gotPrefix, key.Prefix)
	}

	if len(key.Prefix) != 12 {
		t.Fatalf("expected prefix length 12 hex chars, got %d: %q", len(key.Prefix), key.Prefix)
	}

	if len(key.Hashed) != 64 {
		t.Fatalf("expected sha256 hash length 64, got %d: %q", len(key.Hashed), key.Hashed)
	}
}

func TestGenerateAPIKeyUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 10; i++ {
		key, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey returned error: %v", err)
		}
		if seen[key.Full] {
			t.Fatalf("duplicate key generated: %s", key.Full)
		}
		seen[key.Full] = true
	}
}

func TestHashSecretDeterministic(t *testing.T) {
	h1 := HashSecret("my-secret-key")
	h2 := HashSecret("my-secret-key")
	if h1 != h2 {
		t.Fatalf("HashSecret not deterministic: %q vs %q", h1, h2)
	}
}

func TestHashSecretLength(t *testing.T) {
	h := HashSecret("anything")
	if len(h) != 64 {
		t.Fatalf("expected 64 hex chars, got %d: %q", len(h), h)
	}
}

func TestHashSecretDifferentInputs(t *testing.T) {
	h1 := HashSecret("key-a")
	h2 := HashSecret("key-b")
	if h1 == h2 {
		t.Fatal("HashSecret should produce different outputs for different inputs")
	}
}

func TestHashSecretEmpty(t *testing.T) {
	h := HashSecret("")
	if len(h) != 64 {
		t.Fatalf("empty input should still produce a 64-char hash, got %d", len(h))
	}
}

func TestGenerateAPIKeySecretMatchesHash(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey returned error: %v", err)
	}

	parts := strings.SplitN(strings.TrimPrefix(key.Full, "mk_live_"), ".", 2)
	secret := parts[1]

	expectedHash := HashSecret(secret)
	if key.Hashed != expectedHash {
		t.Fatalf("HashSecret(%q) = %q, but key.Hashed = %q", secret, expectedHash, key.Hashed)
	}
}
