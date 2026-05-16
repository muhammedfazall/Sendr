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

var checkScript = redis.NewScript(`
local current = tonumber(redis.call("GET", KEYS[1]) or "0")
local limit = tonumber(ARGV[1])
if current >= limit then
  return {0, 0}
end
current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[2])
end
local remaining = limit - current
return {1, remaining}
`)

func New(rdb *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{rdb: rdb}
}

// Check atomically reserves one send for today if the caller is within their
// plan limit. Rejected requests do not increment usage.
// Returns: allowed, remaining, error.
func (r *RedisRateLimiter) Check(ctx context.Context, userID string, limit int) (bool, int, error) {
	key := fmt.Sprintf("rate_limit:%s:%s", userID, time.Now().UTC().Format("2006-01-02"))

	res, err := checkScript.Run(ctx, r.rdb, []string{key}, limit, int((25 * time.Hour).Seconds())).Result()
	if err != nil {
		return false, 0, fmt.Errorf("ratelimit script: %w", err)
	}
	values, ok := res.([]any)
	if !ok || len(values) != 2 {
		return false, 0, fmt.Errorf("ratelimit script returned unexpected result")
	}

	allowed, ok := values[0].(int64)
	if !ok {
		return false, 0, fmt.Errorf("ratelimit script returned invalid allowed value")
	}
	remaining, ok := values[1].(int64)
	if !ok {
		return false, 0, fmt.Errorf("ratelimit script returned invalid remaining value")
	}

	return allowed == 1, int(remaining), nil
}

// GetCount returns today's usage count for a user without incrementing it.
func (r *RedisRateLimiter) GetCount(ctx context.Context, userID string) (int, error) {
	key := fmt.Sprintf("rate_limit:%s:%s", userID, time.Now().UTC().Format("2006-01-02"))

	val, err := r.rdb.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("ratelimit get count: %w", err)
	}
	return val, nil
}
