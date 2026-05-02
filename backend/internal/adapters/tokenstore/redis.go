package tokenstore

import (
	"context"
	"encoding/json"
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

const refreshIdleTimeout = 30 * time.Minute

type refreshSession struct {
	TokenID       string    `json:"token_id"`
	IdleExpiresAt time.Time `json:"idle_expires_at"`
}

func New(rdb *redis.Client) *RedisTokenStore {
	return &RedisTokenStore{rdb: rdb}
}

func key(userID string) string {
	return fmt.Sprintf("refresh:%s", userID)
}

func accessBlacklistKey(tokenID string) string {
	return fmt.Sprintf("blacklist:access:%s", tokenID)
}

// Store persists a refresh token for the given user with a TTL.
func (s *RedisTokenStore) Store(ctx context.Context, userID, tokenID string, ttl time.Duration) error {
	session := refreshSession{
		TokenID:       tokenID,
		IdleExpiresAt: time.Now().Add(refreshIdleTimeout),
	}
	raw, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal refresh session: %w", err)
	}
	return s.rdb.Set(ctx, key(userID), raw, ttl).Err()
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
	var session refreshSession
	if err := json.Unmarshal([]byte(stored), &session); err != nil {
		return stored == tokenID, nil
	}
	if time.Now().After(session.IdleExpiresAt) {
		_ = s.Delete(ctx, userID)
		return false, nil
	}
	return session.TokenID == tokenID, nil
}

// Delete removes the refresh token for the given user (logout).
func (s *RedisTokenStore) Delete(ctx context.Context, userID string) error {
	return s.rdb.Del(ctx, key(userID)).Err()
}

func (s *RedisTokenStore) BlacklistAccessToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	if tokenID == "" || ttl <= 0 {
		return nil
	}
	return s.rdb.Set(ctx, accessBlacklistKey(tokenID), "1", ttl).Err()
}

func (s *RedisTokenStore) IsAccessTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	if tokenID == "" {
		return false, nil
	}
	exists, err := s.rdb.Exists(ctx, accessBlacklistKey(tokenID)).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return exists > 0, nil
}
