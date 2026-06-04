package authhandler

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/config"
	"github.com/muhammedfazall/Sendr/pkg/constants"
)

type mockAuthSvc struct {
	getAuthURLFn     func(state string) string
	handleCallbackFn func(ctx context.Context, code string) (string, string, error)
	refreshTokenFn   func(ctx context.Context, userID, refreshTokenID string) (string, string, error)
	logoutFn         func(ctx context.Context, userID, accessTokenID string, accessTokenTTL time.Duration) error
}

func (m *mockAuthSvc) GetAuthURL(state string) string                { return m.getAuthURLFn(state) }
func (m *mockAuthSvc) HandleCallback(ctx context.Context, code string) (string, string, error) {
	return m.handleCallbackFn(ctx, code)
}
func (m *mockAuthSvc) RefreshToken(ctx context.Context, userID, refreshTokenID string) (string, string, error) {
	return m.refreshTokenFn(ctx, userID, refreshTokenID)
}
func (m *mockAuthSvc) Logout(ctx context.Context, userID, accessTokenID string, accessTokenTTL time.Duration) error {
	return m.logoutFn(ctx, userID, accessTokenID, accessTokenTTL)
}

func authHandler(t *testing.T) *Handler {
	t.Helper()
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&pk.PublicKey)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	cfg := &config.Config{
		FrontendURL:      "http://localhost:5173",
		OAuthStateSecret: "test-state-secret",
		JWTPublicKeyPEM:  string(pubPEM),
		AppEnv:           "development",
	}
	return New(&mockAuthSvc{}, cfg)
}

func signTokenPK(t *testing.T, pk *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(pk)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

func TestLoginRedirectsToGoogle(t *testing.T) {
	h := authHandler(t)
	h.svc = &mockAuthSvc{
		getAuthURLFn: func(state string) string {
			if state == "" {
				t.Fatal("state should not be empty")
			}
			return "https://accounts.google.com/o/oauth2/auth?state=" + state
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	h.Login().ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d", w.Code)
	}

	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "oauth_state" {
			found = true
			if c.MaxAge != 300 {
				t.Fatalf("expected MaxAge 300, got %d", c.MaxAge)
			}
			if !c.HttpOnly {
				t.Fatal("expected HttpOnly")
			}
			break
		}
	}
	if !found {
		t.Fatal("expected oauth_state cookie")
	}
}

func TestLoginWithRedirectURI(t *testing.T) {
	h := authHandler(t)
	h.svc = &mockAuthSvc{
		getAuthURLFn: func(state string) string {
			return "https://accounts.google.com/o/oauth2/auth?state=" + state
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/google?redirect_uri=http://localhost:9876/callback", nil)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	h.Login().ServeHTTP(w, req)

	cookies := w.Result().Cookies()
	var foundRedirect bool
	for _, c := range cookies {
		if c.Name == "oauth_redirect_uri" {
			foundRedirect = true
			if c.Value != "http://localhost:9876/callback" {
				t.Fatalf("unexpected redirect_uri value: %s", c.Value)
			}
			break
		}
	}
	if !foundRedirect {
		t.Fatal("expected oauth_redirect_uri cookie when redirect_uri param present")
	}
}

func TestCallbackSuccess(t *testing.T) {
	h := authHandler(t)
	h.svc = &mockAuthSvc{
		handleCallbackFn: func(_ context.Context, code string) (string, string, error) {
			if code != "auth_code_xyz" {
				t.Fatalf("unexpected code: %s", code)
			}
			return "access-token-123", "refresh-token-456", nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=auth_code_xyz", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: buildState("test-state-secret", "test-agent")})
	// Add the state to the query
	req.URL.RawQuery = "code=auth_code_xyz&state=" + req.Cookies()[0].Value

	// Re-create with query containing the state
	req = httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=auth_code_xyz&state="+req.Cookies()[0].Value, nil)
	req.Header.Set("User-Agent", "test-agent")
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: req.URL.Query().Get("state")})

	w := httptest.NewRecorder()
	h.Callback().ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d", w.Code)
	}

	cookies := w.Result().Cookies()
	var hasAuth, hasRefresh bool
	for _, c := range cookies {
		if c.Name == "auth_token" {
			hasAuth = true
		}
		if c.Name == "refresh_token" {
			hasRefresh = true
		}
	}
	if !hasAuth {
		t.Fatal("expected auth_token cookie")
	}
	if !hasRefresh {
		t.Fatal("expected refresh_token cookie")
	}
}

func TestCallbackMissingStateCookie(t *testing.T) {
	h := authHandler(t)
	h.svc = &mockAuthSvc{}

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=xyz", nil)
	w := httptest.NewRecorder()

	h.Callback().ServeHTTP(w, req)

	// Should redirect with error
	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "missing_state") {
		t.Log("redirect includes error in query string")
	}
}

func TestCallbackInvalidState(t *testing.T) {
	h := authHandler(t)
	h.svc = &mockAuthSvc{}

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=xyz&state=invalid", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "invalid-state"})
	w := httptest.NewRecorder()

	h.Callback().ServeHTTP(w, req)

	// Should redirect with error
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "invalid_state") {
		t.Fatalf("expected redirect with invalid_state, got: %s", loc)
	}
}

func TestCallbackHandleCallbackError(t *testing.T) {
	h := authHandler(t)
	h.svc = &mockAuthSvc{
		handleCallbackFn: func(_ context.Context, code string) (string, string, error) {
			return "", "", errors.New("oauth error")
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=bad", nil)
	req.Header.Set("User-Agent", "test-agent")
	state := buildState("test-state-secret", "test-agent")
	req.URL.RawQuery = "code=bad&state=" + state
	req.Header.Set("User-Agent", "test-agent")
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: state})
	w := httptest.NewRecorder()

	h.Callback().ServeHTTP(w, req)

	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "auth_failed") {
		t.Fatalf("expected redirect with auth_failed, got: %s", loc)
	}
}

func TestTokenSuccess(t *testing.T) {
	h := authHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/auth/token", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "my-access-token"})
	w := httptest.NewRecorder()

	h.Token().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTokenMissingCookie(t *testing.T) {
	h := authHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/auth/token", nil)
	w := httptest.NewRecorder()

	h.Token().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRefreshSuccess(t *testing.T) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pubDER, _ := x509.MarshalPKIXPublicKey(&pk.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	cfg := &config.Config{
		OAuthStateSecret: "secret",
		JWTPublicKeyPEM:  string(pubPEM),
		AppEnv:           "development",
	}

	svc := &mockAuthSvc{
		refreshTokenFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "new-access", "new-refresh", nil
		},
	}
	h := New(svc, cfg)

	// Create an expired signed token for the Authorization header
	claims := jwt.MapClaims{
		"user_id": "user-1",
		"jti":     "jti-1",
		"iss":     "sendr",
		"aud":     "sendr-api",
		"exp":     float64(time.Now().Add(-1 * time.Hour).Unix()),
	}
	tokenStr := signTokenPK(t, pk, claims)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "old-refresh"})
	w := httptest.NewRecorder()

	h.Refresh().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRefreshMissingRefreshCookie(t *testing.T) {
	h := authHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	w := httptest.NewRecorder()

	h.Refresh().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRefreshMissingAuthHeader(t *testing.T) {
	h := authHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "some-refresh"})
	w := httptest.NewRecorder()

	h.Refresh().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRefreshFailed(t *testing.T) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pubDER, _ := x509.MarshalPKIXPublicKey(&pk.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	cfg := &config.Config{
		OAuthStateSecret: "secret",
		JWTPublicKeyPEM:  string(pubPEM),
		AppEnv:           "development",
	}

	svc := &mockAuthSvc{
		refreshTokenFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "", "", errors.New("invalid refresh")
		},
	}
	h := New(svc, cfg)

	claims := jwt.MapClaims{
		"user_id": "user-1",
		"jti":     "jti-1",
		"iss":     "sendr",
		"aud":     "sendr-api",
		"exp":     float64(time.Now().Add(-1 * time.Hour).Unix()),
	}
	tokenStr := signTokenPK(t, pk, claims)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "old-refresh"})
	w := httptest.NewRecorder()

	h.Refresh().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLogoutSuccess(t *testing.T) {
	h := authHandler(t)
	h.svc = &mockAuthSvc{
		logoutFn: func(_ context.Context, _, _ string, _ time.Duration) error { return nil },
	}

	claims := jwt.MapClaims{
		"user_id": "user-1",
		"jti":     "jti-1",
		"iss":     "sendr",
		"aud":     "sendr-api",
		"exp":     float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	ctx := context.WithValue(req.Context(), middleware.UserClaimsKey, claims)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Logout().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestLogoutMissingClaims(t *testing.T) {
	h := authHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()

	h.Logout().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLogoutServiceError(t *testing.T) {
	h := authHandler(t)
	h.svc = &mockAuthSvc{
		logoutFn: func(_ context.Context, _, _ string, _ time.Duration) error {
			return errors.New("logout failed")
		},
	}

	claims := jwt.MapClaims{
		"user_id": "user-1",
		"jti":     "jti-1",
		"exp":     float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	ctx := context.WithValue(req.Context(), middleware.UserClaimsKey, claims)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Logout().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestExtractExpiredClaimsValid(t *testing.T) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	claims := jwt.MapClaims{"user_id": "u1", "iss": "sendr", "aud": "sendr-api", "exp": float64(time.Now().Add(-1).Unix())}
	tokenStr := signTokenPK(t, pk, claims)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	got, err := extractExpiredClaims(req, &pk.PublicKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["user_id"] != "u1" {
		t.Fatalf("expected user_id u1, got %v", got["user_id"])
	}
}

func TestExtractExpiredClaimsMissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	_, err := extractExpiredClaims(req, nil)
	if err == nil {
		t.Fatal("expected error for missing header")
	}
}

func TestExtractExpiredClaimsWrongSigningMethod(t *testing.T) {
	pk, _ := rsa.GenerateKey(rand.Reader, 2048)
	claims := jwt.MapClaims{"user_id": "u1", "iss": "sendr", "aud": "sendr-api"}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	_, err = extractExpiredClaims(req, &pk.PublicKey)
	if err == nil {
		t.Fatal("expected error for wrong signing method")
	}
}

func TestExtractExpiredClaimsWrongIssuer(t *testing.T) {
	pk, _ := rsa.GenerateKey(rand.Reader, 2048)
	claims := jwt.MapClaims{"user_id": "u1", "iss": "wrong", "aud": "sendr-api", "exp": float64(time.Now().Add(-1).Unix())}
	tokenStr := signTokenPK(t, pk, claims)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	_, err := extractExpiredClaims(req, &pk.PublicKey)
	if err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestExtractExpiredClaimsWrongAudience(t *testing.T) {
	pk, _ := rsa.GenerateKey(rand.Reader, 2048)
	claims := jwt.MapClaims{"user_id": "u1", "iss": "sendr", "aud": "wrong", "exp": float64(time.Now().Add(-1).Unix())}
	tokenStr := signTokenPK(t, pk, claims)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	_, err := extractExpiredClaims(req, &pk.PublicKey)
	if err == nil {
		t.Fatal("expected error for wrong audience")
	}
}

func TestBuildStateAndValidate(t *testing.T) {
	secret := "test-secret"
	ua := "Mozilla/5.0"

	state := buildState(secret, ua)
	if !strings.Contains(state, ".") {
		t.Fatal("state should contain a dot separator")
	}

	if !validateState(secret, state, ua) {
		t.Fatal("expected valid state")
	}

	if validateState(secret, state, "different-agent") {
		t.Fatal("state should be invalid for different user agent")
	}

	if validateState("wrong-secret", state, ua) {
		t.Fatal("state should be invalid for wrong secret")
	}

	if validateState(secret, "invalid", ua) {
		t.Fatal("expected invalid for malformed state")
	}
}

func TestAccessTokenTTL(t *testing.T) {
	tests := []struct {
		name   string
		claims jwt.MapClaims
		want   time.Duration
	}{
		{"in future", jwt.MapClaims{"exp": float64(time.Now().Add(1 * time.Hour).Unix())}, time.Hour - 1*time.Second},
		{"in past", jwt.MapClaims{"exp": float64(time.Now().Add(-1 * time.Hour).Unix())}, -1 * time.Hour},
		{"missing exp", jwt.MapClaims{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := accessTokenTTL(tt.claims)
			if tt.name == "in future" && got <= 0 {
				t.Fatalf("expected positive TTL, got %s", got)
			}
			if tt.name == "in past" && got >= 0 {
				t.Fatalf("expected negative TTL, got %s", got)
			}
			if tt.name == "missing exp" && got != 0 {
				t.Fatalf("expected 0, got %s", got)
			}
		})
	}
}

func TestHasIssuer(t *testing.T) {
	if !hasIssuer(jwt.MapClaims{"iss": "sendr"}, "sendr") {
		t.Fatal("expected issuer match")
	}
	if hasIssuer(jwt.MapClaims{"iss": "other"}, "sendr") {
		t.Fatal("expected issuer mismatch")
	}
	if hasIssuer(jwt.MapClaims{}, "sendr") {
		t.Fatal("expected false for missing issuer")
	}
}

func TestHasAudience(t *testing.T) {
	if !hasAudience(jwt.MapClaims{"aud": "sendr-api"}, "sendr-api") {
		t.Fatal("expected audience match (string)")
	}
	if hasAudience(jwt.MapClaims{"aud": "other"}, "sendr-api") {
		t.Fatal("expected audience mismatch")
	}
	if !hasAudience(jwt.MapClaims{"aud": []any{"sendr-api", "other"}}, "sendr-api") {
		t.Fatal("expected audience match (array)")
	}
	if hasAudience(jwt.MapClaims{}, "sendr-api") {
		t.Fatal("expected false for missing audience")
	}
}

func TestConstantsRefreshTokenExpiry(t *testing.T) {
	if constants.RefreshTokenExpiry <= 0 {
		t.Fatal("RefreshTokenExpiry must be positive")
	}
}
