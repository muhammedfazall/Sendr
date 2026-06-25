package ports

import (
	"context"
	"time"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

// AuthService handles Google OAuth, JWT issuance, and refresh tokens.
type AuthService interface {
	GetAuthURL(state string) string
	// HandleCallback exchanges the OAuth code for an access token + refresh token.
	HandleCallback(ctx context.Context, code string) (accessToken, refreshToken string, err error)
	// RefreshToken validates the refresh token and issues a new access + refresh token pair.
	RefreshToken(ctx context.Context, userID, refreshTokenID string) (newAccess, newRefresh string, err error)
	// Logout deletes the user's refresh token and blacklists the current access token.
	Logout(ctx context.Context, userID, accessTokenID string, accessTokenTTL time.Duration) error
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
	// GetCount returns today's usage count for a user without incrementing it.
	GetCount(ctx context.Context, userID string) (int, error)
}

// TemplateService renders user-defined email templates with provided data.
type TemplateService interface {
	Render(ctx context.Context, templateID, userID string, data map[string]any) (subject, html, text string, err error)
}

// PaymentService handles Razorpay order creation, verification, and plan upgrades.
type PaymentService interface {
	CreateOrder(ctx context.Context, userID, planName string) (map[string]any, error)
	VerifyPayment(ctx context.Context, userID, orderID, paymentID, signature string) error
	GetPlans(ctx context.Context) ([]domain.Plan, error)
}