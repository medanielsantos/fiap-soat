package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	redislib "github.com/redis/go-redis/v9"
)

const rateLimitScript = `
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return count
`

var ErrUnavailable = errors.New("redis rate limiter is unavailable")

type RateLimiter struct {
	client redislib.UniversalClient
}

func NewRateLimiter(redisURL string) (*RateLimiter, error) {
	if strings.TrimSpace(redisURL) == "" {
		return &RateLimiter{}, nil
	}
	options, err := redislib.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	return &RateLimiter{client: redislib.NewClient(options)}, nil
}

func (limiter *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limiter == nil || limiter.client == nil {
		return false, ErrUnavailable
	}
	if strings.TrimSpace(key) == "" || limit < 1 || window <= 0 {
		return false, errors.New("rate limit key, positive limit, and positive window are required")
	}
	windowMilliseconds := window.Milliseconds()
	if windowMilliseconds < 1 {
		return false, errors.New("rate limit window must be at least one millisecond")
	}

	digest := sha256.Sum256([]byte(key))
	redisKey := "fiap-soat:rate-limit:" + hex.EncodeToString(digest[:])
	count, err := limiter.client.Eval(ctx, rateLimitScript, []string{redisKey}, windowMilliseconds).Int64()
	if err != nil {
		return false, fmt.Errorf("execute Redis rate limit: %w", err)
	}
	return count <= int64(limit), nil
}
