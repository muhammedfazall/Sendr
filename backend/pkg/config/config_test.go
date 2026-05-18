package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJWTPrivateKeyBytesPrefersPEM(t *testing.T) {
	cfg := &Config{
		JWTPrivateKeyPEM:  `line-one\nline-two`,
		JWTPrivateKeyPath: filepath.Join(t.TempDir(), "missing.pem"),
	}

	got, err := cfg.JWTPrivateKeyBytes()
	if err != nil {
		t.Fatalf("JWTPrivateKeyBytes returned error: %v", err)
	}

	if string(got) != "line-one\nline-two" {
		t.Fatalf("JWTPrivateKeyBytes = %q, want normalized PEM", got)
	}
}

func TestJWTPublicKeyBytesFallsBackToPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(path, []byte("public-key"), 0o600); err != nil {
		t.Fatalf("write test key: %v", err)
	}

	cfg := &Config{JWTPublicKeyPath: path}

	got, err := cfg.JWTPublicKeyBytes()
	if err != nil {
		t.Fatalf("JWTPublicKeyBytes returned error: %v", err)
	}

	if string(got) != "public-key" {
		t.Fatalf("JWTPublicKeyBytes = %q, want file contents", got)
	}
}
