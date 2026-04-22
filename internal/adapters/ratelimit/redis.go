package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements ports.RateLimiter using a Redis fixed window.
// Key format: rate_limit:<userID>:<YYYY-MM-DD>
// The window resets at UTC midnight because the date is baked into the key.
type RedisRateLimiter struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{rdb: rdb}
}

// Check atomically increments the counter for today and reports whether
// the caller is within their plan limit.
// Returns: allowed, remaining, error.
func (r *RedisRateLimiter) Check(ctx context.Context, userID string, limit int) (bool, int, error) {
	key := fmt.Sprintf("rate_limit:%s:%s", userID, time.Now().UTC().Format("2006-01-02"))

	// Pipeline: INCR + EXPIRE in one round-trip.
	// TTL is 25h so a key created at 23:59 is never evicted before midnight.
	pipe := r.rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 25*time.Hour)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, 0, fmt.Errorf("ratelimit pipeline: %w", err)
	}

	current := int(incr.Val())
	remaining := limit - current
	if remaining < 0 {
		remaining = 0
	}

	return current <= limit, remaining, nil
}