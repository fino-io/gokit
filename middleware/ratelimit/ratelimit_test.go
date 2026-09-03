package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewLocalKeyedRateLimiterRejectsInvalidConfig(t *testing.T) {
	for _, test := range []struct {
		name    string
		every   time.Duration
		burst   int
		maxKeys int
		err     string
	}{
		{name: "every", every: 0, burst: 1, maxKeys: 1, err: "every must be positive"},
		{name: "burst", every: time.Second, burst: 0, maxKeys: 1, err: "burst must be positive"},
		{name: "max keys", every: time.Second, burst: 1, maxKeys: 0, err: "maxKeys must be positive"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewLocalKeyedRateLimiter(test.every, test.burst, test.maxKeys, time.Minute)
			require.EqualError(t, err, test.err)
		})
	}
}

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
