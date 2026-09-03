package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLocalKeyedRateLimiterUsesIndependentBuckets(t *testing.T) {
	limiter, err := NewLocalKeyedRateLimiter(time.Hour, 1, 10, time.Minute)
	require.NoError(t, err)

	require.True(t, limiter.Allow("user:1"))
	require.False(t, limiter.Allow("user:1"))
	require.True(t, limiter.Allow("user:2"))
}

func TestLocalKeyedRateLimiterRecreatesExpiredBucket(t *testing.T) {
	limiter, err := NewLocalKeyedRateLimiter(time.Hour, 1, 10, time.Minute)
	require.NoError(t, err)
	require.True(t, limiter.Allow("user:1"))

	entry, ok := limiter.limiters.Peek("user:1")
	require.True(t, ok)
	entry.lastSeen = time.Now().Add(-2 * time.Minute)
	require.True(t, limiter.Allow("user:1"))
}

func TestLocalKeyedRateLimiterEvictsLeastRecentlyUsedBucket(t *testing.T) {
	limiter, err := NewLocalKeyedRateLimiter(time.Hour, 1, 1, time.Minute)
	require.NoError(t, err)

	require.True(t, limiter.Allow("user:1"))
	require.True(t, limiter.Allow("user:2"))
	require.True(t, limiter.Allow("user:1"))
	require.Equal(t, 1, limiter.limiters.Len())
}
