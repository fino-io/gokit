package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestHTTPServerRecordsStatusClasses(t *testing.T) {
	registry := prometheus.NewRegistry()
	server := newHTTPServer(registry, "test")
	handler := server.middleware(func(*http.Request) string { return "/tasks/{id}" })(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}),
	)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/tasks/42", nil))

	if got := counterValue(t, registry, "test_http_server_requests_total", map[string]string{
		"method": "GET", "route": "/tasks/{id}", "status_class": "4xx",
	}); got != 1 {
		t.Fatalf("requests = %v, want 1", got)
	}
}

func TestHTTPServerRecordsPanics(t *testing.T) {
	registry := prometheus.NewRegistry()
	server := newHTTPServer(registry, "test")
	handler := server.middleware(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	func() {
		defer func() { _ = recover() }()
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}()

	if got := counterValue(t, registry, "test_http_server_requests_total", map[string]string{
		"method": "GET", "route": "unknown", "status_class": "5xx",
	}); got != 1 {
		t.Fatalf("requests = %v, want 1", got)
	}
}
