package fositeadapter

import (
	"context"
	"sync"
	"time"

	"github.com/manuel/tinyidp/pkg/idp"
)

type AllowAllRateLimiter struct{}

var _ idp.RateLimiter = AllowAllRateLimiter{}

func (AllowAllRateLimiter) Allow(context.Context, string) bool { return true }

type FixedWindowRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	now     func() time.Time
	buckets map[string]rateBucket
}

var _ idp.RateLimiter = (*FixedWindowRateLimiter)(nil)

type rateBucket struct {
	start time.Time
	count int
}

func NewFixedWindowRateLimiter(limit int, window time.Duration) *FixedWindowRateLimiter {
	return &FixedWindowRateLimiter{limit: limit, window: window, now: time.Now, buckets: map[string]rateBucket{}}
}
func (l *FixedWindowRateLimiter) Allow(_ context.Context, key string) bool {
	if l == nil || l.limit <= 0 || l.window <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now().UTC()
	b := l.buckets[key]
	if b.start.IsZero() || now.Sub(b.start) >= l.window {
		b = rateBucket{start: now}
	}
	if b.count >= l.limit {
		l.buckets[key] = b
		return false
	}
	b.count++
	l.buckets[key] = b
	return true
}
