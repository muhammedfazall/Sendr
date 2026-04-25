package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/pkg/helpers"
)

// contextKey is a custom type to avoid key collisions in context
type contextKey string

const UserClaimsKey contextKey = "userClaims"

func JWTAuth(publicKeyPath string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Step 1 — get the Authorization header
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}

			// Step 2 — extract the token string
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// Step 3 — load public key and verify the token
			keyBytes, err := os.ReadFile(publicKeyPath)
			if err != nil {
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}

			publicKey, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
			if err != nil {
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				// Make sure the signing method is RSA — reject anything else
				if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return publicKey, nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Step 4 — inject claims into request context
			ctx := context.WithValue(r.Context(), UserClaimsKey, token.Claims)
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
				http.Error(w, "missing api key", http.StatusUnauthorized)
				return
			}

			fullKey := strings.TrimPrefix(authHeader, "Bearer ")

			// Split on "." to get prefix and secret
			parts := strings.SplitN(fullKey, ".", 2)
			if len(parts) != 2 {
				http.Error(w, "invalid api key format", http.StatusUnauthorized)
				return
			}

			// Strip the "mk_live_" part to get the raw prefix
			prefix := strings.TrimPrefix(parts[0], "mk_live_")
			secret := parts[1]

			// Look up by prefix
			var storedHash string
			var userID string
			err := db.QueryRow(context.Background(),
				`SELECT hashed_key, user_id FROM api_keys
				 WHERE prefix = $1 AND revoked = false`,
				prefix,
			).Scan(&storedHash, &userID)
			if err != nil {
				http.Error(w, "invalid api key", http.StatusUnauthorized)
				return
			}

			// Hash the incoming secret and compare — constant time to prevent timing attacks
			incomingHash := helpers.HashSecret(secret)
			if subtle.ConstantTimeCompare([]byte(incomingHash), []byte(storedHash)) != 1 {
				http.Error(w, "invalid api key", http.StatusUnauthorized)
				return
			}

			// Inject user_id into context
			ctx := context.WithValue(r.Context(), UserClaimsKey, map[string]interface{}{
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
