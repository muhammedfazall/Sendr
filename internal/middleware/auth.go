package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
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