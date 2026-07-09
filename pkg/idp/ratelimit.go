package idp

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// FixedWindowRateLimiter is a bounded in-process limiter for single-node
// deployments. A distributed deployment must inject a shared implementation.
type FixedWindowRateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	now      func() time.Time
	buckets  map[string]rateBucket
	accepted atomic.Uint64
	rejected atomic.Uint64
}

type rateBucket struct {
	start time.Time
	count int
}

var _ RateLimiter = (*FixedWindowRateLimiter)(nil)
var _ ProductionReadyReporter = (*FixedWindowRateLimiter)(nil)

func NewFixedWindowRateLimiter(limit int, window time.Duration) *FixedWindowRateLimiter {
	return &FixedWindowRateLimiter{limit: limit, window: window, now: time.Now, buckets: map[string]rateBucket{}}
}

func (l *FixedWindowRateLimiter) Allow(ctx context.Context, key string) bool {
	if ctx.Err() != nil || l == nil || l.limit <= 0 || l.window <= 0 || key == "" {
		if l != nil {
			l.rejected.Add(1)
		}
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now().UTC()
	bucket := l.buckets[key]
	if bucket.start.IsZero() || now.Sub(bucket.start) >= l.window {
		bucket = rateBucket{start: now}
	}
	if bucket.count >= l.limit {
		l.buckets[key] = bucket
		l.rejected.Add(1)
		return false
	}
	bucket.count++
	l.buckets[key] = bucket
	l.accepted.Add(1)
	return true
}

func (l *FixedWindowRateLimiter) ProductionReady() bool {
	return l != nil && l.limit > 0 && l.window > 0
}

type RateLimitStats struct {
	Accepted uint64
	Rejected uint64
	Buckets  int
}

func (l *FixedWindowRateLimiter) Stats() RateLimitStats {
	if l == nil {
		return RateLimitStats{}
	}
	l.mu.Lock()
	buckets := len(l.buckets)
	l.mu.Unlock()
	return RateLimitStats{Accepted: l.accepted.Load(), Rejected: l.rejected.Load(), Buckets: buckets}
}
