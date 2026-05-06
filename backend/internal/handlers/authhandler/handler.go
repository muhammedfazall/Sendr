package authhandler

import (
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/constants"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

type Handler struct {
	svc              ports.AuthService
	frontendURL      string
	oauthStateSecret string
	jwtPublicKey     *rsa.PublicKey
}

func New(svc ports.AuthService, frontendURL, oauthStateSecret, publicKeyPath string) *Handler {
	keyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		log.Fatalf("cannot read JWT public key: %v", err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
	if err != nil {
		log.Fatalf("cannot parse JWT public key: %v", err)
	}

	return &Handler{
		svc:              svc,
		frontendURL:      frontendURL,
		oauthStateSecret: oauthStateSecret,
		jwtPublicKey:     publicKey,
	}
}

// GET /auth/google
func (h *Handler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := buildState(h.oauthStateSecret, r.UserAgent())

		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_state",
			Value:    state,
			MaxAge:   300,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})

		// Store CLI redirect_uri if present
		if redirectURI := r.URL.Query().Get("redirect_uri"); redirectURI != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "oauth_redirect_uri",
				Value:    redirectURI,
				MaxAge:   300,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Path:     "/",
			})
		}

		http.Redirect(w, r, h.svc.GetAuthURL(state), http.StatusTemporaryRedirect)
	}
}

// GET /auth/google/callback
// Sets short-lived HttpOnly cookies for both tokens, then redirects to frontend.
func (h *Handler) Callback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate state
		stateCookie, err := r.Cookie("oauth_state")
		if err != nil || stateCookie.Value == "" {
			http.Redirect(w, r,
				fmt.Sprintf("%s/callback?error=missing_state", h.frontendURL),
				http.StatusTemporaryRedirect)
			return
		}
		queryState := r.URL.Query().Get("state")
		if subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(queryState)) != 1 ||
			!validateState(h.oauthStateSecret, queryState, r.UserAgent()) {
			http.Redirect(w, r,
				fmt.Sprintf("%s/callback?error=invalid_state", h.frontendURL),
				http.StatusTemporaryRedirect)
			return
		}

		// Clear the state cookie
		http.SetCookie(w, &http.Cookie{
			Name: "oauth_state", Value: "", MaxAge: -1, Path: "/",
		})

		accessToken, refreshToken, err := h.svc.HandleCallback(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			http.Redirect(w, r,
				fmt.Sprintf("%s/callback?error=auth_failed", h.frontendURL),
				http.StatusTemporaryRedirect)
			return
		}

		// Short-lived cookie so the frontend can pick up the access token once
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    accessToken,
			MaxAge:   60,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
			Secure:   false, // must be false for CLI local HTTP callback to work
		})

		// Refresh token cookie — lasts 7 days, HttpOnly so JS can't touch it
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			MaxAge:   int(constants.RefreshTokenExpiry.Seconds()),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/auth", // only sent to /auth/* routes
			Secure:   false,
		})

		// Redirect to CLI local server if present, otherwise frontend
		redirectTo := fmt.Sprintf("%s/callback", h.frontendURL)
		if c, err := r.Cookie("oauth_redirect_uri"); err == nil && c.Value != "" {
			// Pass token directly in URL for CLI
			redirectTo = fmt.Sprintf("%s?token=%s", c.Value, accessToken)
			http.SetCookie(w, &http.Cookie{
				Name: "oauth_redirect_uri", Value: "", MaxAge: -1, Path: "/",
			})
		}

		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
	}
}

// GET /auth/token — exchanges the one-time HttpOnly auth_token cookie for a JSON response.
// Called by the frontend Callback page immediately after the OAuth redirect.
func (h *Handler) Token() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")
		if err != nil || cookie.Value == "" {
			response.Error(w, http.StatusUnauthorized, "no_token", "no auth token")
			return
		}

		// Clear the cookie immediately — one-time use
		http.SetCookie(w, &http.Cookie{
			Name: "auth_token", Value: "", MaxAge: -1, Path: "/",
		})

		response.JSON(w, http.StatusOK, map[string]string{
			"token": cookie.Value,
		})
	}
}

// POST /auth/refresh — issues a new access + refresh token pair.
// The JWT (even if expired) is sent via Authorization header to identify the user.
func (h *Handler) Refresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read the refresh token from the HttpOnly cookie
		refreshCookie, err := r.Cookie("refresh_token")
		if err != nil || refreshCookie.Value == "" {
			response.Error(w, http.StatusUnauthorized, "no_refresh_token", "refresh token missing")
			return
		}

		// Verify the JWT signature and stable claims, while allowing expiry for refresh.
		claims, err := extractExpiredClaims(r, h.jwtPublicKey)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "invalid_token", "cannot read claims from token")
			return
		}
		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			response.Error(w, http.StatusUnauthorized, "invalid_claims", "user_id missing from token")
			return
		}

		// Validate + rotate the refresh token
		newAccess, newRefresh, err := h.svc.RefreshToken(r.Context(), userID, refreshCookie.Value)
		if err != nil {
			// Invalid/expired refresh → clear the cookie and force re-login
			http.SetCookie(w, &http.Cookie{
				Name: "refresh_token", Value: "", MaxAge: -1, Path: "/auth",
			})
			response.Error(w, http.StatusUnauthorized, "refresh_failed", "refresh token is invalid or expired")
			return
		}

		// Set the new refresh token cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    newRefresh,
			MaxAge:   int(constants.RefreshTokenExpiry.Seconds()),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/auth",
			Secure:   true,
		})

		response.JSON(w, http.StatusOK, map[string]string{
			"token": newAccess,
		})
	}
}

// POST /auth/logout — deletes the refresh token and clears cookies.
// Protected by JWT middleware so we know who's logging out.
func (h *Handler) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token claims")
			return
		}

		jti, _ := claims["jti"].(string)
		if err := h.svc.Logout(r.Context(), userID, jti, accessTokenTTL(claims)); err != nil {
			response.Error(w, http.StatusInternalServerError, "logout_failed", "could not log out")
			return
		}

		// Clear the refresh token cookie
		http.SetCookie(w, &http.Cookie{
			Name: "refresh_token", Value: "", MaxAge: -1, Path: "/auth",
		})

		w.WriteHeader(http.StatusNoContent)
	}
}

// extractExpiredClaims reads JWT claims without verifying expiry.
// Used by the refresh endpoint where the access token may already be expired.
func extractExpiredClaims(r *http.Request, publicKey *rsa.PublicKey) (jwt.MapClaims, error) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("missing authorization header")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return publicKey, nil
	}, jwt.WithoutClaimsValidation())
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if !hasIssuer(claims, "sendr") || !hasAudience(claims, "sendr-api") {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func buildState(secret, userAgent string) string {
	nonce := uuid.NewString()
	return nonce + "." + signState(secret, nonce, userAgent)
}

func validateState(secret, state, userAgent string) bool {
	parts := strings.SplitN(state, ".", 2)
	if len(parts) != 2 {
		return false
	}
	expected := signState(secret, parts[0], userAgent)
	return subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expected)) == 1
}

func signState(secret, nonce, userAgent string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(nonce))
	mac.Write([]byte("|"))
	mac.Write([]byte(userAgent))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func accessTokenTTL(claims jwt.MapClaims) time.Duration {
	exp, ok := claims["exp"].(float64)
	if !ok {
		return 0
	}
	return time.Until(time.Unix(int64(exp), 0))
}

func hasIssuer(claims jwt.MapClaims, issuer string) bool {
	v, ok := claims["iss"].(string)
	return ok && v == issuer
}

func hasAudience(claims jwt.MapClaims, audience string) bool {
	switch aud := claims["aud"].(type) {
	case string:
		return aud == audience
	case []any:
		for _, item := range aud {
			if s, ok := item.(string); ok && s == audience {
				return true
			}
		}
	}
	return false
}
