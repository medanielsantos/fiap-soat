package ratelimit

import (
	"context"
	"time"
)

// Limiter atomically counts requests for a key during a fixed time window.
type Limiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}
