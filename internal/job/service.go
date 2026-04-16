package job

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/muhammedfazall/Sendr/internal/types"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Enqueue(ctx context.Context, userID, apiKeyID string, payload types.EmailPayload) (*Job, error) {
	//validation

	if payload.To == "" || payload.Subject == "" {
		return nil, fmt.Errorf("invalid payload")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	job := &Job{
		UserID:     userID,
		APIKeyID:   apiKeyID,
		Payload:    data,
		Status:     StatusPending,
		MaxRetries: 3,
	}

	if err := s.repo.Insert(ctx, job); err != nil {
		return nil, err
	}

	return job, nil
}
