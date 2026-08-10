package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/kit/endpoint"
	"golang.org/x/time/rate"

	"github.com/fino-io/core/go/fino/core"
)

func NewTokenBucketLimitMW(bkt *rate.Limiter) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if !bkt.Allow() {
				return nil, core.NewResourceExhaustedError("Rate limit exceed!")
			}
			return next(ctx, request)
		}
	}
}

func EveryRateLimiter(interval time.Duration, b int) endpoint.Middleware {
	limiter := rate.NewLimiter(rate.Every(interval), b)
	return NewTokenBucketLimitMW(limiter)
}

type keyedRateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// KeyedRateLimiter maintains an independent token bucket for each key.
type KeyedRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*keyedRateLimiterEntry
	every    time.Duration
	burst    int
	ttl      time.Duration
}

// NewKeyedRateLimiter creates a limiter with one token bucket per key.
func NewKeyedRateLimiter(every time.Duration, burst int, ttl time.Duration) *KeyedRateLimiter {
	return &KeyedRateLimiter{
		limiters: make(map[string]*keyedRateLimiterEntry),
		every:    every,
		burst:    burst,
		ttl:      ttl,
	}
}

// Allow reports whether the key has a token available.
func (l *KeyedRateLimiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupExpired(now)
	entry := l.limiters[key]
	if entry == nil {
		entry = &keyedRateLimiterEntry{
			limiter: rate.NewLimiter(rate.Every(l.every), l.burst),
		}
		l.limiters[key] = entry
	}
	entry.lastSeen = now
	return entry.limiter.Allow()
}

func (l *KeyedRateLimiter) cleanupExpired(now time.Time) {
	if l.ttl <= 0 {
		return
	}
	for key, entry := range l.limiters {
		if now.Sub(entry.lastSeen) > l.ttl {
			delete(l.limiters, key)
		}
	}
}
