package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/muhammedfazall/Sendr/pkg/config"
)

// testKeyPair generates an RSA key pair for JWT signing in tests.
func testKeyPair(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}
	return key
}

func newTestAuthService(m *mockDeps, cfg *config.Config, pk *rsa.PrivateKey) *authService {
	return &authService{
		users:      m.users,
		tokens:     m.tokens,
		cfg:        cfg,
		privateKey: pk,
	}
}

// Helper to generate a valid test JWT and extract the token string.
func signTestJWT(t *testing.T, pk *rsa.PrivateKey, userID, email string) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"jti":     "test-jti-123",
		"iss":     "sendr",
		"aud":     "sendr-api",
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	}).SignedString(pk)
	if err != nil {
		t.Fatalf("sign test JWT: %v", err)
	}
	return tok
}

func TestAuthServiceGetAuthURL(t *testing.T) {
	mock := newMockDeps()
	cfg := &config.Config{GoogleClientID: "test-id", GoogleClientSecret: "test-secret", BackendURL: "http://localhost:8080"}
	pk := testKeyPair(t)
	svc := newTestAuthService(mock, cfg, pk)

	url := svc.GetAuthURL("test-state")
	if url == "" {
		t.Fatal("expected non-empty auth URL")
	}
}

func TestAuthServiceIssueTokenPair(t *testing.T) {
	mock := newMockDeps()
	cfg := &config.Config{GoogleClientID: "test-id", GoogleClientSecret: "test-secret", BackendURL: "http://localhost:8080"}
	pk := testKeyPair(t)
	svc := newTestAuthService(mock, cfg, pk)

	accessToken, refreshToken, err := svc.issueTokenPair(context.Background(), "user-1", "test@example.com")
	if err != nil {
		t.Fatalf("issueTokenPair returned error: %v", err)
	}

	if accessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if refreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}

	// Verify the access token can be parsed
	parsed, err := jwt.Parse(accessToken, func(t *jwt.Token) (interface{}, error) {
		return &pk.PublicKey, nil
	}, jwt.WithAudience("sendr-api"), jwt.WithIssuer("sendr"))
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("expected valid token")
	}

	// Verify refresh token was stored
	valid, err := mock.tokens.Validate(context.Background(), "user-1", refreshToken)
	if err != nil {
		t.Fatalf("validate refresh token: %v", err)
	}
	if !valid {
		t.Fatal("expected refresh token to be valid")
	}
}

func TestAuthServiceIssueTokenPairStoresInRedis(t *testing.T) {
	mock := newMockDeps()
	cfg := &config.Config{GoogleClientID: "test-id", GoogleClientSecret: "test-secret", BackendURL: "http://localhost:8080"}
	pk := testKeyPair(t)
	svc := newTestAuthService(mock, cfg, pk)

	_, refreshToken, err := svc.issueTokenPair(context.Background(), "user-1", "test@example.com")
	if err != nil {
		t.Fatalf("issueTokenPair: %v", err)
	}

	// Verify stored in mock token store
	valid, err := mock.tokens.Validate(context.Background(), "user-1", refreshToken)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !valid {
		t.Fatal("expected token to be valid before logout")
	}
}

func TestAuthServiceLogout(t *testing.T) {
	mock := newMockDeps()
	cfg := &config.Config{GoogleClientID: "test-id", GoogleClientSecret: "test-secret", BackendURL: "http://localhost:8080"}
	pk := testKeyPair(t)
	svc := newTestAuthService(mock, cfg, pk)

	_, refreshToken, err := svc.issueTokenPair(context.Background(), "user-1", "test@example.com")
	if err != nil {
		t.Fatalf("issueTokenPair: %v", err)
	}

	// Verify token is valid before logout
	valid, _ := mock.tokens.Validate(context.Background(), "user-1", refreshToken)
	if !valid {
		t.Fatal("expected token to be valid before logout")
	}

	// Logout
	if err := svc.Logout(context.Background(), "user-1", "test-jti", 15*time.Minute); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}

	// Verify refresh token is gone
	valid, _ = mock.tokens.Validate(context.Background(), "user-1", refreshToken)
	if valid {
		t.Fatal("expected refresh token to be invalid after logout")
	}

	// Verify access token was blacklisted
	blacklisted, _ := mock.tokens.IsAccessTokenBlacklisted(context.Background(), "test-jti")
	if !blacklisted {
		t.Fatal("expected access token to be blacklisted after logout")
	}
}

func TestAuthServiceRefreshTokenSuccess(t *testing.T) {
	mock := newMockDeps()
	cfg := &config.Config{GoogleClientID: "test-id", GoogleClientSecret: "test-secret", BackendURL: "http://localhost:8080"}
	pk := testKeyPair(t)
	svc := newTestAuthService(mock, cfg, pk)

	// First, create a user
	user := mock.addUserWithPlan("free")

	// Issue initial token pair
	_, refreshToken, err := svc.issueTokenPair(context.Background(), user.ID, user.Email)
	if err != nil {
		t.Fatalf("issueTokenPair: %v", err)
	}

	// Refresh the token
	newAccess, newRefresh, err := svc.RefreshToken(context.Background(), user.ID, refreshToken)
	if err != nil {
		t.Fatalf("RefreshToken returned error: %v", err)
	}

	if newAccess == "" {
		t.Fatal("expected non-empty new access token")
	}
	if newRefresh == "" {
		t.Fatal("expected non-empty new refresh token")
	}

	// Old refresh token should be invalid (rotated)
	valid, _ := mock.tokens.Validate(context.Background(), user.ID, refreshToken)
	if valid {
		t.Fatal("expected old refresh token to be invalid after rotation")
	}

	// New refresh token should be valid
	valid, _ = mock.tokens.Validate(context.Background(), user.ID, newRefresh)
	if !valid {
		t.Fatal("expected new refresh token to be valid")
	}
}

func TestAuthServiceRefreshTokenInvalid(t *testing.T) {
	mock := newMockDeps()
	cfg := &config.Config{GoogleClientID: "test-id", GoogleClientSecret: "test-secret", BackendURL: "http://localhost:8080"}
	pk := testKeyPair(t)
	svc := newTestAuthService(mock, cfg, pk)

	_, _, err := svc.RefreshToken(context.Background(), "user-nonexistent", "bad-refresh-token")
	if err == nil {
		t.Fatal("expected error for invalid refresh token, got nil")
	}
}

func TestAuthServiceRefreshTokenUserNotFound(t *testing.T) {
	mock := newMockDeps()
	cfg := &config.Config{GoogleClientID: "test-id", GoogleClientSecret: "test-secret", BackendURL: "http://localhost:8080"}
	pk := testKeyPair(t)
	svc := newTestAuthService(mock, cfg, pk)

	// Store a refresh token for a user
	mock.tokens.Store(context.Background(), "user-no-db", "some-token", 7*24*time.Hour)

	// Try to refresh - user doesn't exist in user repo
	_, _, err := svc.RefreshToken(context.Background(), "user-no-db", "some-token")
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}
}

func TestAuthServiceSignJWTHasCorrectClaims(t *testing.T) {
	mock := newMockDeps()
	cfg := &config.Config{GoogleClientID: "test-id", GoogleClientSecret: "test-secret", BackendURL: "http://localhost:8080"}
	pk := testKeyPair(t)
	svc := newTestAuthService(mock, cfg, pk)

	tokenStr, err := svc.signJWT("user-42", "user@example.com")
	if err != nil {
		t.Fatalf("signJWT: %v", err)
	}

	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return &pk.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	if claims["user_id"] != "user-42" {
		t.Fatalf("expected user_id 'user-42', got %v", claims["user_id"])
	}
	if claims["email"] != "user@example.com" {
		t.Fatalf("expected email 'user@example.com', got %v", claims["email"])
	}
	if claims["iss"] != "sendr" {
		t.Fatalf("expected iss 'sendr', got %v", claims["iss"])
	}
	if claims["aud"] != "sendr-api" {
		t.Fatalf("expected aud 'sendr-api', got %v", claims["aud"])
	}
	// jti should be present
	if _, ok := claims["jti"].(string); !ok || claims["jti"] == "" {
		t.Fatal("expected non-empty jti claim")
	}
}

func TestAuthServiceSignJWTExpiry(t *testing.T) {
	mock := newMockDeps()
	cfg := &config.Config{GoogleClientID: "test-id", GoogleClientSecret: "test-secret", BackendURL: "http://localhost:8080"}
	pk := testKeyPair(t)
	svc := newTestAuthService(mock, cfg, pk)

	tokenStr, err := svc.signJWT("user-1", "test@test.com")
	if err != nil {
		t.Fatalf("signJWT: %v", err)
	}

	claims := jwt.MapClaims{}
	jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return &pk.PublicKey, nil
	})

	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("expected exp claim")
	}
	expTime := time.Unix(int64(exp), 0)
	if expTime.Before(time.Now()) {
		t.Fatal("token should not be expired yet")
	}
	if expTime.After(time.Now().Add(20 * time.Minute)) {
		t.Fatal("token expiry should be ~15 minutes, got longer")
	}
}
