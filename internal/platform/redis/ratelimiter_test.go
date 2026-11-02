package redis

import (
	"context"
	"testing"
	"time"
)

func TestAllowPermitsUpToLimitWithinWindow(t *testing.T) {
	limiter := newTestLimiter(t)
	ctx, key := context.Background(), uniqueKey(t)

	for attempt := 1; attempt <= 3; attempt++ {
		allowed, err := limiter.Allow(ctx, key, 2, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		if want := attempt <= 2; allowed != want {
			t.Fatalf("attempt %d allowed = %v, want %v", attempt, allowed, want)
		}
	}
}

func TestAllowResetsAfterWindowExpires(t *testing.T) {
	limiter := newTestLimiter(t)
	ctx, key := context.Background(), uniqueKey(t)

	if allowed, err := limiter.Allow(ctx, key, 1, 50*time.Millisecond); err != nil || !allowed {
		t.Fatalf("first call = %v, %v", allowed, err)
	}
	if allowed, _ := limiter.Allow(ctx, key, 1, 50*time.Millisecond); allowed {
		t.Fatal("second call inside window must be rejected")
	}
	time.Sleep(120 * time.Millisecond)
	if allowed, err := limiter.Allow(ctx, key, 1, 50*time.Millisecond); err != nil || !allowed {
		t.Fatalf("call after window = %v, %v", allowed, err)
	}
}

func TestAllowIsolatesKeys(t *testing.T) {
	limiter := newTestLimiter(t)
	ctx, key := context.Background(), uniqueKey(t)

	if allowed, _ := limiter.Allow(ctx, key+"-a", 1, time.Minute); !allowed {
		t.Fatal("key a first call must be allowed")
	}
	if allowed, _ := limiter.Allow(ctx, key+"-b", 1, time.Minute); !allowed {
		t.Fatal("key b must not share key a counter")
	}
}
