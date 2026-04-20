package ports

import (
	"context"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

// UserRepository defines persistence operations for users.
type UserRepository interface {
	Upsert(ctc context.Context, googleID, email, name string ) (*domain.User, error)
	FindByID(ctc context.Context, id string ) (*domain.User, error)
	FindWithPlan(ctc context.Context, id string ) (*domain.User, *domain.Plan, error)
}

// APIKeyRepository defines persistence operations for API keys.
type APIKeyRepository interface {
	Create(ctc context.Context, userID, name, prefix, hashedKey string) (*domain.APIKey, error)
	ListByUser(ctx context.Context, userID string) ([]domain.APIKey, error)
	FindByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error)
	Revoke(ctx context.Context, keyID, userID string) error
}

// JobRepository defines persistence operations for the job queue.
type JobRepository interface {
	Enqueue(ctx context.Context, userID, apiKeyID string, payload domain.EmailPayload) (*domain.Job, error)
	ClaimBatch(ctx context.Context, batchSize int) ([]domain.Job, error)
	MarkDone(ctx context.Context, jobID string) error
	MarkFailed(ctx context.Context, jobID, errMsg string) error
	MoveToDLQ(ctx context.Context, job domain.Job, errMsg string) error
}