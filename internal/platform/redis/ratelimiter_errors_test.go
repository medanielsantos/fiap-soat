package redis

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewRateLimiterWithoutURLIsUnavailable(t *testing.T) {
	limiter, err := NewRateLimiter("   ")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := limiter.Allow(context.Background(), "key", 1, time.Second); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
	var nilLimiter *RateLimiter
	if _, err := nilLimiter.Allow(context.Background(), "key", 1, time.Second); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil limiter err = %v, want ErrUnavailable", err)
	}
}

func TestNewRateLimiterRejectsInvalidURL(t *testing.T) {
	if _, err := NewRateLimiter("http://not-redis"); err == nil || !strings.HasPrefix(err.Error(), "parse REDIS_URL:") {
		t.Fatalf("err = %v, want parse REDIS_URL error", err)
	}
}

func TestAllowValidatesArguments(t *testing.T) {
	limiter, err := NewRateLimiter("redis://127.0.0.1:1/0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = limiter.client.Close() })
	cases := []struct {
		name   string
		key    string
		limit  int
		window time.Duration
	}{
		{"blank key", " ", 1, time.Second},
		{"zero limit", "key", 0, time.Second},
		{"zero window", "key", 1, 0},
		{"sub millisecond window", "key", 1, time.Microsecond},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := limiter.Allow(context.Background(), tc.key, tc.limit, tc.window); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestAllowWrapsRedisFailure(t *testing.T) {
	limiter, err := NewRateLimiter("redis://127.0.0.1:1/0?dial_timeout=200ms&max_retries=-1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = limiter.client.Close() })
	_, err = limiter.Allow(context.Background(), "key", 1, time.Second)
	if err == nil || !strings.HasPrefix(err.Error(), "execute Redis rate limit:") {
		t.Fatalf("err = %v, want wrapped Redis error", err)
	}
}
