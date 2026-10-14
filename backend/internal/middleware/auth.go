package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/pkg/config"
	"github.com/muhammedfazall/Sendr/pkg/helpers"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

// UserClaimsKey is the gin context key holding the authenticated user's claims.
const UserClaimsKey = "user_claims"

// JWTAuth validates the bearer JWT on protected routes.
func JWTAuth(cfg *config.Config, tokens ports.TokenStore) (gin.HandlerFunc, error) {
	keyBytes, err := cfg.JWTPublicKeyBytes()
	if err != nil {
		return nil, err
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(keyBytes)
	if err != nil {
		return nil, err
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			response.Error(c, 401, "missing_token", "missing token")
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return publicKey, nil
		}, jwt.WithAudience("sendr-api"), jwt.WithIssuer("sendr"))

		if err != nil || !token.Valid {
			response.Error(c, 401, "invalid_token", "invalid token")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(c, 401, "invalid_token", "invalid token")
			c.Abort()
			return
		}
		jti, ok := claims["jti"].(string)
		if !ok || jti == "" {
			response.Error(c, 401, "invalid_token", "invalid token")
			c.Abort()
			return
		}

		blacklisted, err := tokens.IsAccessTokenBlacklisted(c.Request.Context(), jti)
		if err != nil {
			response.Error(c, 503, "auth_unavailable", "authentication temporarily unavailable")
			c.Abort()
			return
		}
		if blacklisted {
			response.Error(c, 401, "invalid_token", "invalid token")
			c.Abort()
			return
		}

		c.Set(UserClaimsKey, claims)
		c.Next()
	}, nil
}

// ValidateAPIKey authenticates requests using an mk_live_ API key.
func ValidateAPIKey(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			response.Error(c, 401, "missing_api_key", "missing api key")
			c.Abort()
			return
		}

		fullKey := strings.TrimPrefix(authHeader, "Bearer ")
		parts := strings.SplitN(fullKey, ".", 2)
		if len(parts) != 2 {
			response.Error(c, 401, "invalid_api_key", "invalid api key")
			c.Abort()
			return
		}

		prefix := strings.TrimPrefix(parts[0], "mk_live_")
		secret := parts[1]

		var storedHash string
		var userID string
		err := db.QueryRow(c.Request.Context(),
			`SELECT hashed_key, user_id FROM api_keys
			 WHERE prefix = $1 AND revoked = false`,
			prefix,
		).Scan(&storedHash, &userID)
		if err != nil {
			response.Error(c, 401, "invalid_api_key", "invalid api key")
			c.Abort()
			return
		}

		incomingHash := helpers.HashSecret(secret)
		if subtle.ConstantTimeCompare([]byte(incomingHash), []byte(storedHash)) != 1 {
			response.Error(c, 401, "invalid_api_key", "invalid api key")
			c.Abort()
			return
		}

		c.Set(UserClaimsKey, jwt.MapClaims{"user_id": userID})
		c.Next()
	}
}

// GetClaims returns the claims injected by JWTAuth or ValidateAPIKey.
func GetClaims(c *gin.Context) (jwt.MapClaims, bool) {
	v, ok := c.Get(UserClaimsKey)
	if !ok {
		return nil, false
	}
	claims, ok := v.(jwt.MapClaims)
	return claims, ok
}
