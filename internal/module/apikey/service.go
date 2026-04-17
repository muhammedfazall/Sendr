package apikey

import (
	"context"
	"fmt"

	"github.com/muhammedfazall/Sendr/pkg/helpers/hash"
)

// Service contains the business logic for API key management.
type Service struct {
	repo *Repository
}

// NewService creates a new apikey Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateKey generates a new API key, stores the hash, and returns the full key (shown once).
func (s *Service) CreateKey(ctx context.Context, userID, name string) (*CreateKeyResponse, error) {
	key, err := hash.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	id, err := s.repo.Create(ctx, userID, name, key.Prefix, key.Hashed)
	if err != nil {
		return nil, fmt.Errorf("failed to save key: %w", err)
	}

	return &CreateKeyResponse{
		ID:     id,
		Name:   name,
		Prefix: key.Prefix,
		APIKey: key.Full,
	}, nil
}

// ListKeys returns all non-revoked API keys for the user.
func (s *Service) ListKeys(ctx context.Context, userID string) ([]APIKey, error) {
	return s.repo.ListByUser(ctx, userID)
}

// RevokeKey soft-deletes the specified API key.
func (s *Service) RevokeKey(ctx context.Context, keyID, userID string) error {
	return s.repo.Revoke(ctx, keyID, userID)
}
