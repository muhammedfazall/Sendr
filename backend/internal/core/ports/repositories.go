package ports

import (
	"context"
	"time"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	Upsert(ctx context.Context, googleID, email, name string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindWithPlan(ctx context.Context, id string) (*domain.User, *domain.Plan, error)
}

// APIKeyRepository defines persistence operations for API keys.
type APIKeyRepository interface {
	Create(ctx context.Context, userID, name, prefix, hashedKey string) (*domain.APIKey, error)
	ListByUser(ctx context.Context, userID string) ([]domain.APIKey, error)
	FindByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error)
	Revoke(ctx context.Context, keyID, userID string) error
}

// TokenStore defines persistence for refresh tokens (backed by Redis).
type TokenStore interface {
	Store(ctx context.Context, userID, tokenID string, ttl time.Duration) error
	Validate(ctx context.Context, userID, tokenID string) (bool, error)
	Delete(ctx context.Context, userID string) error
}

// JobRepository defines persistence operations for the job queue.
type JobRepository interface {
	Enqueue(ctx context.Context, userID, apiKeyID string, payload domain.EmailPayload) (*domain.Job, error)
	ClaimBatch(ctx context.Context, batchSize int) ([]domain.Job, error)
	MarkDone(ctx context.Context, jobID string) error
	// MarkFailed resets the job to pending with a delayed run_at for retry backoff.
	MarkFailed(ctx context.Context, jobID string, backoff time.Duration) error
	MoveToDLQ(ctx context.Context, job domain.Job, errMsg string) error
	ReclaimZombies(ctx context.Context) (int64, error)
	GetByID(ctx context.Context, jobID string) (*domain.Job, error)
	ListByUser(ctx context.Context, userID, status string, limit, offset int) ([]domain.Job, error)
}