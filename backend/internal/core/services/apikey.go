package services

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/pkg/constants"
	helper "github.com/muhammedfazall/Sendr/pkg/helpers"
)

type apiKeyService struct {
	keys ports.APIKeyRepository
}

func NewApiKeyServices(keys ports.APIKeyRepository) ports.APIKeyService {
	return &apiKeyService{keys: keys}
}

func (s *apiKeyService) Create(ctx context.Context, userID, name string) (string, *domain.APIKey, error) {
	k, err := helper.GenerateAPIKey()
	if err != nil {
		return "", nil, fmt.Errorf("generate Key: %w", err)
	}
	created, err := s.keys.Create(ctx, userID, name, k.Prefix, k.Hashed)
	if err != nil {
		return "", nil, fmt.Errorf("persist key: %w", err)
	}
	return k.Full, created, nil
}

func (s *apiKeyService) List(ctx context.Context, userID string) ([]domain.APIKey, error) {
	return s.keys.ListByUser(ctx, userID)
}

func (s *apiKeyService) Revoke(ctx context.Context, keyID, userID string) error {
	return s.keys.Revoke(ctx, keyID, userID)
}

// Validate extracts prefix + secret from the full key, looks up the record,
// then does a constant-time hash comparison to prevent timing attacks.
func (s *apiKeyService) Validate(ctx context.Context, fullKey string) (*domain.APIKey, error) {
	// Strip the "mk_live_" prefix, then split prefix.secret
	trimmed := strings.TrimPrefix(fullKey, "mk_live_")
	parts := strings.SplitN(trimmed, ".", 2)
	if len(parts) != 2 {
		return nil, constants.ErrAPIKeyInvalid
	}
	prefix, secret := parts[0], parts[1]

	rec, err := s.keys.FindByPrefix(ctx, prefix)
	if err != nil {
		return nil, constants.ErrAPIKeyNotFound
	}
	if rec.Revoked {
		return nil, constants.ErrAPIKeyRevoked
	}

	incoming := helper.HashSecret(secret)
	if subtle.ConstantTimeCompare([]byte(incoming), []byte(rec.Hashed)) != 1 {
		return nil, constants.ErrAPIKeyInvalid
	}
	return rec, nil
}
