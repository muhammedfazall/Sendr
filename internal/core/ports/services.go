package ports

import (
	"context"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

// AuthService handles Google OAuth and JWT issuance.
type AuthService interface {
	GetAuthURL(state string) string
	HandleCallback(ctx context.Context, code string) (token string, err error)
}

// APIKeyService handles key generation, listing, and validation.
type APIKeyService interface {
	Create(ctx context.Context, userID, name string) (fullKey string, key *domain.APIKey, err error)
	List(ctx context.Context, userID string) ([]domain.APIKey, error)
	Revoke(ctx context.Context, keyID, userID string) error
	Validate(ctx context.Context, fullKey string) (*domain.APIKey, error)
}

// EmailService coordinates key validation, rate limiting, and job queuing.
type EmailService interface {
	Send(ctx context.Context, fullKey string, payload domain.EmailPayload) (*domain.Job, error)
}

// RateLimiter defines the Redis-backed rate-check contract.
type RateLimiter interface {
	Check(ctx context.Context, userID string, limit int) (allowed bool, remaining int, err error)
}