package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/pkg/config"
)

// mockTokenStore implements ports.TokenStore for middleware tests.
type mockTokenStore struct {
	blacklisted map[string]bool
}

func (m *mockTokenStore) Store(_ context.Context, _, _ string, _ time.Duration) error { return nil }
func (m *mockTokenStore) Validate(_ context.Context, _, _ string) (bool, error)       { return true, nil }
func (m *mockTokenStore) Delete(_ context.Context, _ string) error                    { return nil }
func (m *mockTokenStore) BlacklistAccessToken(_ context.Context, id string, _ time.Duration) error {
	m.blacklisted[id] = true
	return nil
}
func (m *mockTokenStore) IsAccessTokenBlacklisted(_ context.Context, id string) (bool, error) {
	return m.blacklisted[id], nil
}

func testJWTSetup(t *testing.T) (*rsa.PrivateKey, *config.Config) {
	t.Helper()
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	pubDER, err := x509.MarshalPKIXPublicKey(&pk.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	cfg := &config.Config{
		JWTPublicKeyPEM: string(pubPEM),
		FrontendURL:     "http://localhost:5173",
		BackendURL:      "http://localhost:8080",
		FromEmail:       "test@test.com",
	}

	return pk, cfg
}

func testHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			c.String(http.StatusInternalServerError, "no claims")
			return
		}
		c.String(http.StatusOK, claims["user_id"].(string))
	}
}

// runMiddleware executes a middleware + terminal handler through a real gin engine.
func runMiddleware(mw gin.HandlerFunc, handler gin.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", mw, handler)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func signTestToken(t *testing.T, pk *rsa.PrivateKey, userID string) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"user_id": userID,
		"email":   "test@test.com",
		"jti":     "test-jti",
		"iss":     "sendr",
		"aud":     "sendr-api",
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	}).SignedString(pk)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

func TestJWTAuthValidToken(t *testing.T) {
	pk, cfg := testJWTSetup(t)
	tokens := &mockTokenStore{blacklisted: make(map[string]bool)}

	mw, err := JWTAuth(cfg, tokens)
	if err != nil {
		t.Fatalf("JWTAuth: %v", err)
	}

	token := signTestToken(t, pk, "user-123")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := runMiddleware(mw, testHandler(), req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "user-123" {
		t.Fatalf("expected body 'user-123', got %q", w.Body.String())
	}
}

func TestJWTAuthMissingHeader(t *testing.T) {
	_, cfg := testJWTSetup(t)
	tokens := &mockTokenStore{blacklisted: make(map[string]bool)}

	mw, err := JWTAuth(cfg, tokens)
	if err != nil {
		t.Fatalf("JWTAuth: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := runMiddleware(mw, testHandler(), req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuthInvalidToken(t *testing.T) {
	_, cfg := testJWTSetup(t)
	tokens := &mockTokenStore{blacklisted: make(map[string]bool)}

	mw, err := JWTAuth(cfg, tokens)
	if err != nil {
		t.Fatalf("JWTAuth: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-here")
	w := runMiddleware(mw, testHandler(), req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWTAuthBlacklistedToken(t *testing.T) {
	pk, cfg := testJWTSetup(t)
	tokens := &mockTokenStore{blacklisted: make(map[string]bool)}

	mw, err := JWTAuth(cfg, tokens)
	if err != nil {
		t.Fatalf("JWTAuth: %v", err)
	}

	token := signTestToken(t, pk, "user-123")
	tokens.blacklisted["test-jti"] = true

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := runMiddleware(mw, testHandler(), req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for blacklisted token, got %d", w.Code)
	}
}

func TestJWTAuthWrongIssuer(t *testing.T) {
	pk, cfg := testJWTSetup(t)
	tokens := &mockTokenStore{blacklisted: make(map[string]bool)}

	mw, err := JWTAuth(cfg, tokens)
	if err != nil {
		t.Fatalf("JWTAuth: %v", err)
	}

	tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"user_id": "user-1",
		"email":   "test@test.com",
		"jti":     "test-jti",
		"iss":     "wrong-issuer",
		"aud":     "sendr-api",
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	}).SignedString(pk)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := runMiddleware(mw, testHandler(), req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong issuer, got %d", w.Code)
	}
}

func TestJWTAuthExpiredToken(t *testing.T) {
	pk, cfg := testJWTSetup(t)
	tokens := &mockTokenStore{blacklisted: make(map[string]bool)}

	mw, err := JWTAuth(cfg, tokens)
	if err != nil {
		t.Fatalf("JWTAuth: %v", err)
	}

	tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"user_id": "user-1",
		"email":   "test@test.com",
		"jti":     "test-jti",
		"iss":     "sendr",
		"aud":     "sendr-api",
		"exp":     time.Now().Add(-1 * time.Hour).Unix(),
	}).SignedString(pk)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := runMiddleware(mw, testHandler(), req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestGetClaimsFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Set(UserClaimsKey, jwt.MapClaims{
		"user_id": "user-42",
		"email":   "test@test.com",
	})

	got, ok := GetClaims(c)
	if !ok {
		t.Fatal("expected claims to be found")
	}
	if got["user_id"] != "user-42" {
		t.Fatalf("expected user_id 'user-42', got %v", got["user_id"])
	}
}

func TestGetClaimsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	_, ok := GetClaims(c)
	if ok {
		t.Fatal("expected no claims for context without value")
	}
}

func TestJWTAuthWrongSigningMethod(t *testing.T) {
	_, cfg := testJWTSetup(t)
	tokens := &mockTokenStore{blacklisted: make(map[string]bool)}

	mw, err := JWTAuth(cfg, tokens)
	if err != nil {
		t.Fatalf("JWTAuth: %v", err)
	}

	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user-1",
		"email":   "test@test.com",
		"jti":     "test-jti",
		"iss":     "sendr",
		"aud":     "sendr-api",
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	}).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := runMiddleware(mw, testHandler(), req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong signing method, got %d", w.Code)
	}
}

var _ ports.TokenStore = (*mockTokenStore)(nil)
