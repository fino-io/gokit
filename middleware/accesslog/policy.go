package accesslog

import (
	"sync/atomic"
	"time"
)

type policy struct {
	slowThreshold time.Duration
	sampleEvery   uint64
	skip          map[string]struct{}
	counter       atomic.Uint64
}

func newPolicy(cfg config, skip []string) *policy {
	paths := make(map[string]struct{}, len(skip))
	for _, path := range skip {
		if path != "" {
			paths[path] = struct{}{}
		}
	}
	return &policy{
		slowThreshold: cfg.SlowThreshold,
		sampleEvery:   cfg.SampleEvery,
		skip:          paths,
	}
}

func (p *policy) shouldLog(operation, requestID string, duration time.Duration, important bool) bool {
	if important {
		return true
	}
	if _, skipped := p.skip[operation]; skipped {
		return false
	}
	if p.slowThreshold > 0 && duration >= p.slowThreshold {
		return true
	}
	switch p.sampleEvery {
	case 0:
		return false
	case 1:
		return true
	}
	if requestID != "" {
		return fnv64a(requestID)%p.sampleEvery == 0
	}
	return p.counter.Add(1)%p.sampleEvery == 0
}

func fnv64a(value string) uint64 {
	const (
		offset = 14695981039346656037
		prime  = 1099511628211
	)
	hash := uint64(offset)
	for i := 0; i < len(value); i++ {
		hash ^= uint64(value[i])
		hash *= prime
	}
	return hash
}
