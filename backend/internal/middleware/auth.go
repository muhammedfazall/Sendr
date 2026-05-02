package middleware

import (
	"context"
	"crypto/subtle"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/pkg/helpers"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

// contextKey is a custom type to avoid key collisions in context
type contextKey string

const UserClaimsKey contextKey = "userClaims"

func JWTAuth(publicKeyPath string, tokens ports.TokenStore) func(http.Handler) http.Handler {
	keyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		log.Fatalf("cannot read JWT public key: %v", err)
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
	if err != nil {
		log.Fatalf("cannot parse JWT public key: %v", err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Step 1 — get the Authorization header
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				response.Error(w, http.StatusUnauthorized, "missing_token", "missing token")
				return
			}

			// Step 2 — extract the token string
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				// Make sure the signing method is RSA — reject anything else
				if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return publicKey, nil
			}, jwt.WithAudience("sendr-api"), jwt.WithIssuer("sendr"))

			if err != nil || !token.Valid {
				response.Error(w, http.StatusUnauthorized, "invalid_token", "invalid token")
				return
			}

			// Step 4 — inject claims into request context
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				response.Error(w, http.StatusUnauthorized, "invalid_token", "invalid token")
				return
			}
			jti, ok := claims["jti"].(string)
			if !ok || jti == "" {
				response.Error(w, http.StatusUnauthorized, "invalid_token", "invalid token")
				return
			}
			blacklisted, err := tokens.IsAccessTokenBlacklisted(r.Context(), jti)
			if err != nil {
				response.Error(w, http.StatusServiceUnavailable, "auth_unavailable", "authentication temporarily unavailable")
				return
			}
			if blacklisted {
				response.Error(w, http.StatusUnauthorized, "invalid_token", "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ValidateAPIKey middleware for routes that accept API keys instead of JWT
func ValidateAPIKey(db *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				response.Error(w, http.StatusUnauthorized, "missing_api_key", "missing api key")
				return
			}

			fullKey := strings.TrimPrefix(authHeader, "Bearer ")

			// Split on "." to get prefix and secret
			parts := strings.SplitN(fullKey, ".", 2)
			if len(parts) != 2 {
				response.Error(w, http.StatusUnauthorized, "invalid_api_key", "invalid api key")
				return
			}

			// Strip the "mk_live_" part to get the raw prefix
			prefix := strings.TrimPrefix(parts[0], "mk_live_")
			secret := parts[1]

			// Look up by prefix
			var storedHash string
			var userID string
			err := db.QueryRow(r.Context(),
				`SELECT hashed_key, user_id FROM api_keys
				 WHERE prefix = $1 AND revoked = false`,
				prefix,
			).Scan(&storedHash, &userID)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid_api_key", "invalid api key")
				return
			}

			// Hash the incoming secret and compare — constant time to prevent timing attacks
			incomingHash := helpers.HashSecret(secret)
			if subtle.ConstantTimeCompare([]byte(incomingHash), []byte(storedHash)) != 1 {
				response.Error(w, http.StatusUnauthorized, "invalid_api_key", "invalid api key")
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, jwt.MapClaims{
				"user_id": userID,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// handlers can easily get claims
func GetClaims(r *http.Request) (jwt.MapClaims, bool) {
	claims, ok := r.Context().Value(UserClaimsKey).(jwt.MapClaims)
	return claims, ok
}
