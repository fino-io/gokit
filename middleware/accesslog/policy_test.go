package accesslog

import (
	"fmt"
	"testing"
	"time"
)

func TestPolicyAlwaysLogsImportantAndSlowEvents(t *testing.T) {
	t.Parallel()

	cfg := config{
		SlowThreshold: time.Second,
		SampleEvery:   0,
	}
	p := newPolicy(cfg, []string{"/healthz"})

	tests := []struct {
		name      string
		operation string
		duration  time.Duration
		important bool
		want      bool
	}{
		{name: "ordinary request", operation: "/users"},
		{name: "successful skipped request", operation: "/healthz"},
		{name: "important skipped request", operation: "/healthz", important: true, want: true},
		{name: "slow request", operation: "/users", duration: time.Second, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.shouldLog(tt.operation, "", tt.duration, tt.important); got != tt.want {
				t.Fatalf("shouldLog() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPolicyUsesStableRequestIDSampling(t *testing.T) {
	t.Parallel()

	p := newPolicy(config{SampleEvery: 8}, nil)
	foundSampled := false
	for i := 0; i < 100; i++ {
		requestID := fmt.Sprintf("request-%d", i)
		first := p.shouldLog("", requestID, 0, false)
		if second := p.shouldLog("", requestID, 0, false); second != first {
			t.Fatalf("sampling changed for the same request ID: first=%v second=%v", first, second)
		}
		foundSampled = foundSampled || first
	}
	if !foundSampled {
		t.Fatal("expected at least one request ID to be sampled")
	}
}

func TestPolicyFallsBackToAtomicCounter(t *testing.T) {
	t.Parallel()

	p := newPolicy(config{SampleEvery: 2}, nil)
	if p.shouldLog("", "", 0, false) {
		t.Fatal("first event should not be sampled")
	}
	if !p.shouldLog("", "", 0, false) {
		t.Fatal("second event should be sampled")
	}
}
