// Package ratelimit provides keyed rate limiters.
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/time/rate"
)

// KeyedLimiter accepts or rejects a request for a key.
//
// Implementations may use local memory or a shared backend. Errors indicate
// that the limit could not be evaluated and must be handled by the caller.
type KeyedLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

type localRateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// LocalKeyedRateLimiter maintains an independent token bucket for each key in
// the current process. Use a distributed limiter or gateway for cluster-wide quotas.
type LocalKeyedRateLimiter struct {
	mu       sync.Mutex
	limiters *lru.Cache[string, *localRateLimiterEntry]
	every    time.Duration
	burst    int
	ttl      time.Duration
}

var _ KeyedLimiter = (*LocalKeyedRateLimiter)(nil)

// NewLocalKeyedRateLimiter creates a bounded, process-local limiter.
func NewLocalKeyedRateLimiter(every time.Duration, burst, maxKeys int, ttl time.Duration) (*LocalKeyedRateLimiter, error) {
	if every <= 0 {
		return nil, fmt.Errorf("every must be positive")
	}
	if burst <= 0 {
		return nil, fmt.Errorf("burst must be positive")
	}
	if maxKeys <= 0 {
		return nil, fmt.Errorf("maxKeys must be positive")
	}
	limiters, err := lru.New[string, *localRateLimiterEntry](maxKeys)
	if err != nil {
		return nil, err
	}
	return &LocalKeyedRateLimiter{
		limiters: limiters,
		every:    every,
		burst:    burst,
		ttl:      ttl,
	}, nil
}

// Allow reports whether the key has a token available.
func (l *LocalKeyedRateLimiter) Allow(_ context.Context, key string) (bool, error) {
	now := time.Now()
	l.mu.Lock()
	l.cleanupExpired(now)
	entry, ok := l.limiters.Get(key)
	if !ok {
		entry = &localRateLimiterEntry{
			limiter: rate.NewLimiter(rate.Every(l.every), l.burst),
		}
		l.limiters.Add(key, entry)
	}
	entry.lastSeen = now
	l.mu.Unlock()

	return entry.limiter.Allow(), nil
}

func (l *LocalKeyedRateLimiter) cleanupExpired(now time.Time) {
	if l.ttl <= 0 {
		return
	}
	for {
		_, entry, ok := l.limiters.GetOldest()
		if !ok || now.Sub(entry.lastSeen) <= l.ttl {
			return
		}
		l.limiters.RemoveOldest()
	}
}
