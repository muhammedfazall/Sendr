package authhandler

import (
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/config"
	"github.com/muhammedfazall/Sendr/pkg/constants"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

type Handler struct {
	svc              ports.AuthService
	frontendURL      string
	oauthStateSecret string
	secureCookie     bool
	jwtPublicKey     *rsa.PublicKey
}

func New(svc ports.AuthService, cfg *config.Config) (*Handler, error) {
	keyBytes, err := cfg.JWTPublicKeyBytes()
	if err != nil {
		return nil, fmt.Errorf("load JWT public key: %w", err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse JWT public key: %w", err)
	}

	return &Handler{
		svc:              svc,
		frontendURL:      cfg.FrontendURL,
		oauthStateSecret: cfg.OAuthStateSecret,
		secureCookie:     cfg.AppEnv == "production",
		jwtPublicKey:     publicKey,
	}, nil
}

// GET /auth/google
func (h *Handler) Login(c *gin.Context) {
	state := buildState(h.oauthStateSecret, c.Request.UserAgent())

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		MaxAge:   300,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	// Store CLI redirect_uri if present
	if redirectURI := c.Query("redirect_uri"); redirectURI != "" {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "oauth_redirect_uri",
			Value:    redirectURI,
			MaxAge:   300,
			HttpOnly: true,
			Secure:   h.secureCookie,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})
	}

	c.Redirect(http.StatusTemporaryRedirect, h.svc.GetAuthURL(state))
}

// GET /auth/google/callback
// Sets short-lived cookies carrying the tokens, then redirects to the frontend/CLI.
func (h *Handler) Callback(c *gin.Context) {
	stateCookie, err := c.Cookie("oauth_state")
	if err != nil || stateCookie == "" {
		c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("%s/callback?error=missing_state", h.frontendURL))
		return
	}

	queryState := c.Query("state")
	if subtle.ConstantTimeCompare([]byte(stateCookie), []byte(queryState)) != 1 ||
		!validateState(h.oauthStateSecret, queryState, c.Request.UserAgent()) {
		c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("%s/callback?error=invalid_state", h.frontendURL))
		return
	}

	// Clear the state cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name: "oauth_state", Value: "", MaxAge: -1, Path: "/", Secure: h.secureCookie,
	})

	accessToken, refreshToken, err := h.svc.HandleCallback(c.Request.Context(), c.Query("code"))
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect,
			fmt.Sprintf("%s/callback?error=auth_failed", h.frontendURL))
		return
	}

	// Short-lived cookie so the frontend can pick up the access token once
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "auth_token",
		Value:    accessToken,
		MaxAge:   60,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Path:     "/",
		Secure:   h.secureCookie,
	})

	// Refresh token cookie — lasts 7 days, HttpOnly so JS can't touch it
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		MaxAge:   int(constants.RefreshTokenExpiry.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Path:     "/auth", // only sent to /auth/* routes
		Secure:   h.secureCookie,
	})

	// Redirect to CLI local server if present, otherwise frontend
	redirectTo := fmt.Sprintf("%s/callback?token=%s", h.frontendURL, accessToken)
	if cliRedirect, cerr := c.Cookie("oauth_redirect_uri"); cerr == nil && cliRedirect != "" {
		// Pass token directly in URL for CLI
		redirectTo = fmt.Sprintf("%s?token=%s", cliRedirect, accessToken)
		http.SetCookie(c.Writer, &http.Cookie{
			Name: "oauth_redirect_uri", Value: "", MaxAge: -1, Path: "/", Secure: h.secureCookie,
		})
	}

	c.Redirect(http.StatusTemporaryRedirect, redirectTo)
}

// GET /auth/token — exchanges the one-time HttpOnly auth_token cookie for a JSON response.
// Called by the frontend Callback page immediately after the OAuth redirect.
func (h *Handler) Token(c *gin.Context) {
	token, err := c.Cookie("auth_token")
	if err != nil || token == "" {
		response.Error(c, http.StatusUnauthorized, "no_token", "no auth token")
		return
	}

	// Clear the cookie immediately — one-time use
	http.SetCookie(c.Writer, &http.Cookie{
		Name: "auth_token", Value: "", MaxAge: -1, Path: "/", Secure: h.secureCookie,
	})

	response.JSON(c, http.StatusOK, map[string]string{
		"token": token,
	})
}

// POST /auth/refresh — issues a new access + refresh token pair.
// The JWT (even if expired) is sent via Authorization header to identify the user.
func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		response.Error(c, http.StatusUnauthorized, "no_refresh_token", "refresh token missing")
		return
	}

	// Verify the JWT signature and stable claims, while allowing expiry for refresh.
	claims, err := extractExpiredClaims(c.Request, h.jwtPublicKey)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "invalid_token", "cannot read claims from token")
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		response.Error(c, http.StatusUnauthorized, "invalid_claims", "user_id missing from token")
		return
	}

	// Validate + rotate the refresh token
	newAccess, newRefresh, err := h.svc.RefreshToken(c.Request.Context(), userID, refreshToken)
	if err != nil {
		// Invalid/expired refresh → clear the cookie and force re-login
		http.SetCookie(c.Writer, &http.Cookie{
			Name: "refresh_token", Value: "", MaxAge: -1, Path: "/auth", Secure: h.secureCookie,
		})
		response.Error(c, http.StatusUnauthorized, "refresh_failed", "refresh token is invalid or expired")
		return
	}

	// Set the new refresh token cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefresh,
		MaxAge:   int(constants.RefreshTokenExpiry.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/auth",
		Secure:   h.secureCookie,
	})

	response.JSON(c, http.StatusOK, map[string]string{
		"token": newAccess,
	})
}

// POST /auth/logout — blacklists the access token and clears the refresh cookie.
func (h *Handler) Logout(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "invalid token claims")
		return
	}

	jti, _ := claims["jti"].(string)
	if err := h.svc.Logout(c.Request.Context(), userID, jti, accessTokenTTL(claims)); err != nil {
		response.Error(c, http.StatusInternalServerError, "logout_failed", "could not log out")
		return
	}

	// Clear the refresh token cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name: "refresh_token", Value: "", MaxAge: -1, Path: "/auth", Secure: h.secureCookie,
	})

	c.Status(http.StatusNoContent)
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
