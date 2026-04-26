package services

import (
	"context"
	"fmt"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/pkg/constants"
)

type emailService struct {
	apiKeys     ports.APIKeyService
	jobs        ports.JobRepository
	users       ports.UserRepository
	rateLimiter ports.RateLimiter
}

// NewEmailService wires up the email send pipeline.
func NewEmailService(
	apiKeys ports.APIKeyService,
	jobs ports.JobRepository,
	users ports.UserRepository,
	rateLimiter ports.RateLimiter,
) ports.EmailService {
	return &emailService{
		apiKeys:     apiKeys,
		jobs:        jobs,
		users:       users,
		rateLimiter: rateLimiter,
	}
}

// Send is the core pipeline:
//  1. Validate API key
//  2. Load user + plan to get daily limit
//  3. Check + increment Redis rate limit
//  4. Enqueue job in Postgres
func (s *emailService) Send(ctx context.Context, fullKey string, payload domain.EmailPayload) (*domain.Job, error) {
// Step 1 — validate key, get owning user
key, err := s.apiKeys.Validate(ctx, fullKey)
if err != nil {
	// err is already a sentinel (ErrAPIKeyInvalid / ErrAPIKeyRevoked / ErrAPIKeyNotFound)
	return nil, err
}

// Step 2 — load user + plan to know their daily limit
_, plan, err := s.users.FindWithPlan(ctx, key.UserID)
if err != nil {
	return nil, fmt.Errorf("%w: %s", constants.ErrUserNotFound, err)
}

// Step 3 — Redis rate limit check
allowed, remaining, err := s.rateLimiter.Check(ctx, key.UserID, plan.DailyLimit)
if err != nil {
	return nil, fmt.Errorf("%w: redis check failed: %s", constants.ErrInternalServer, err)
}
if !allowed {
	_ = remaining // could include in error context / response header
	return nil, constants.ErrRateLimitExceeded
}

job, err := s.jobs.Enqueue(ctx, key.UserID, key.ID, payload)
if err != nil {
	return nil, fmt.Errorf("%w: %s", constants.ErrJobQueueFull, err)
}

return job, nil
}
