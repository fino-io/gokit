package metrics

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestHTTPClientRecordsCalls(t *testing.T) {
	tests := []struct {
		name     string
		response *http.Response
		err      error
		result   string
	}{
		{name: "success", response: &http.Response{StatusCode: http.StatusOK}, result: "success"},
		{name: "client error", response: &http.Response{StatusCode: http.StatusBadRequest}, result: "client_error"},
		{name: "server error", response: &http.Response{StatusCode: http.StatusServiceUnavailable}, result: "server_error"},
		{name: "transport error", err: errors.New("failed"), result: "failure"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			client := newHTTPClient(registry, "test", "registration", "provider")
			transport := client.transport(roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return tt.response, tt.err
			}))
			request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://provider.example/register", nil)
			if err != nil {
				t.Fatal(err)
			}

			response, gotErr := transport.RoundTrip(request)
			if !errors.Is(gotErr, tt.err) {
				t.Fatalf("error = %v, want %v", gotErr, tt.err)
			}
			if response != tt.response {
				t.Fatalf("response = %v, want %v", response, tt.response)
			}

			labels := map[string]string{
				"client": "registration", "target": "provider",
				"method": "POST", "result": tt.result,
			}
			if got := counterValue(t, registry, "test_http_client_requests_total", labels); got != 1 {
				t.Fatalf("requests = %v, want 1", got)
			}
			if got := gaugeValue(t, registry, "test_http_client_requests_in_flight", withoutLabel(labels, "result")); got != 0 {
				t.Fatalf("in flight = %v, want 0", got)
			}
			if got := histogramCount(t, registry, "test_http_client_request_duration_seconds", withoutLabel(labels, "result")); got != 1 {
				t.Fatalf("duration count = %v, want 1", got)
			}
		})
	}
}

func TestHTTPClientTracksInFlightCall(t *testing.T) {
	registry := prometheus.NewRegistry()
	client := newHTTPClient(registry, "test", "registration", "provider")
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	transport := client.transport(roundTripperFunc(func(*http.Request) (*http.Response, error) {
		close(started)
		<-release
		return &http.Response{StatusCode: http.StatusOK}, nil
	}))
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://provider.example/register", nil)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		_, err := transport.RoundTrip(request)
		done <- err
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("round trip did not start")
	}
	labels := map[string]string{
		"client": "registration", "target": "provider", "method": "GET",
	}
	if got := gaugeValue(t, registry, "test_http_client_requests_in_flight", labels); got != 1 {
		t.Fatalf("in flight = %v, want 1", got)
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := gaugeValue(t, registry, "test_http_client_requests_in_flight", labels); got != 0 {
		t.Fatalf("in flight after call = %v, want 0", got)
	}
}

func TestHTTPResultClassifiesContext(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		if got := httpResult(ctx, nil, errors.New("failed")); got != "timeout" {
			t.Fatalf("result = %q, want timeout", got)
		}
	})
	t.Run("canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if got := httpResult(ctx, nil, errors.New("failed")); got != "canceled" {
			t.Fatalf("result = %q, want canceled", got)
		}
	})
}
