package metrics

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestHTTPClientUsesConfiguredTargetLabel(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewHTTPClient(registry, "test", "registration", "provider")
	transport := metrics.Transport(roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusServiceUnavailable}, nil
	}))
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://user-input.example/path", nil)

	_, _ = transport.RoundTrip(request)

	if got := counterValue(t, registry, "test_http_client_requests_total", map[string]string{
		"client": "registration", "target": "provider", "method": "POST", "result": "server_error",
	}); got != 1 {
		t.Fatalf("requests = %v, want 1", got)
	}
}

func TestHTTPClientClassifiesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := result(ctx, errors.New("failed")); got != "canceled" {
		t.Fatalf("result = %q, want canceled", got)
	}
}
