package tokenstore

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisTokenStore implements ports.TokenStore using Redis.
// Each refresh token is stored as key  "refresh:<userID>"  →  value "<tokenID>".
// One active refresh token per user — issuing a new one invalidates the old.
type RedisTokenStore struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *RedisTokenStore {
	return &RedisTokenStore{rdb: rdb}
}

func key(userID string) string {
	return fmt.Sprintf("refresh:%s", userID)
}

// Store persists a refresh token for the given user with a TTL.
func (s *RedisTokenStore) Store(ctx context.Context, userID, tokenID string, ttl time.Duration) error {
	return s.rdb.Set(ctx, key(userID), tokenID, ttl).Err()
}

// Validate checks if the given tokenID matches the one stored for the user.
// Returns true only if the token exists AND matches.
func (s *RedisTokenStore) Validate(ctx context.Context, userID, tokenID string) (bool, error) {
	stored, err := s.rdb.Get(ctx, key(userID)).Result()
	if err == redis.Nil {
		return false, nil // no token — expired or logged out
	}
	if err != nil {
		return false, fmt.Errorf("redis get: %w", err)
	}
	return stored == tokenID, nil
}

// Delete removes the refresh token for the given user (logout).
func (s *RedisTokenStore) Delete(ctx context.Context, userID string) error {
	return s.rdb.Del(ctx, key(userID)).Err()
}
