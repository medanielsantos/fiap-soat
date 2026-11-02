package redis

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// newTestLimiter connects to the Redis server in TEST_REDIS_URL, skipping when unset.
func newTestLimiter(t *testing.T) *RateLimiter {
	t.Helper()
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("set TEST_REDIS_URL to run Redis rate limiter tests")
	}
	limiter, err := NewRateLimiter(redisURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = limiter.client.Close() })
	return limiter
}

func uniqueKey(t *testing.T) string {
	return fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano())
}
