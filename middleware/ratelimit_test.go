package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestKeyedRateLimiterUsesIndependentBuckets(t *testing.T) {
	limiter := NewKeyedRateLimiter(time.Hour, 1, time.Minute)

	require.True(t, limiter.Allow("user:1"))
	require.False(t, limiter.Allow("user:1"))
	require.True(t, limiter.Allow("user:2"))
}

func TestKeyedRateLimiterRecreatesExpiredBucket(t *testing.T) {
	limiter := NewKeyedRateLimiter(time.Hour, 1, time.Minute)
	require.True(t, limiter.Allow("user:1"))

	limiter.limiters["user:1"].lastSeen = time.Now().Add(-2 * time.Minute)
	require.True(t, limiter.Allow("user:1"))
}
