package authhandler

import (
	"crypto/subtle"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/constants"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

type Handler struct {
	svc         ports.AuthService
	frontendURL string
}

func New(svc ports.AuthService, frontendURL string) *Handler {
	return &Handler{svc: svc, frontendURL: frontendURL}
}

// GET /auth/google
func (h *Handler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := uuid.NewString()
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_state",
			Value:    state,
			MaxAge:   300,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})
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
		if subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(queryState)) != 1 {
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
			Secure:   true,
		})

		// Refresh token cookie — lasts 7 days, HttpOnly so JS can't touch it
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			MaxAge:   int(constants.RefreshTokenExpiry.Seconds()),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/auth", // only sent to /auth/* routes
			Secure:   true,
		})

		http.Redirect(w, r,
			fmt.Sprintf("%s/callback", h.frontendURL),
			http.StatusTemporaryRedirect)
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
// The refresh token comes from an HttpOnly cookie, NOT from the request body.
// The JWT (even if expired) is sent via Authorization header to identify the user.
func (h *Handler) Refresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read the refresh token from the HttpOnly cookie
		refreshCookie, err := r.Cookie("refresh_token")
		if err != nil || refreshCookie.Value == "" {
			response.Error(w, http.StatusUnauthorized, "no_refresh_token", "refresh token missing")
			return
		}

		// Parse the JWT from Authorization header to get user_id.
		// We use ParseUnverified here because the access token may be expired —
		// that's the whole point of refresh. We still verify the refresh token against Redis.
		claims, err := extractExpiredClaims(r)
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

		_ = h.svc.Logout(r.Context(), userID)

		// Clear the refresh token cookie
		http.SetCookie(w, &http.Cookie{
			Name: "refresh_token", Value: "", MaxAge: -1, Path: "/auth",
		})

		w.WriteHeader(http.StatusNoContent)
	}
}

// extractExpiredClaims reads JWT claims without verifying expiry.
// Used by the refresh endpoint where the access token may already be expired.
func extractExpiredClaims(r *http.Request) (jwt.MapClaims, error) {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 {
		return nil, fmt.Errorf("missing authorization header")
	}
	tokenStr := authHeader[7:] // strip "Bearer "

	// ParseUnverified skips signature and expiry checks.
	// This is safe here because the refresh token (validated against Redis)
	// is what actually proves the user's session is valid.
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}
	return claims, nil
}
