package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestHTTPServerRecordsStatusClasses(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewHTTPServer(registry, "test")
	handler := metrics.Middleware(func(*http.Request) string { return "/tasks/{id}" })(
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
	metrics := NewHTTPServer(registry, "test")
	handler := metrics.Middleware(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
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

func counterValue(t *testing.T, gatherer prometheus.Gatherer, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := gatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.Metric {
			if metricLabels(metric, labels) {
				return metric.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func gaugeValue(t *testing.T, gatherer prometheus.Gatherer, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := gatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.Metric {
			if metricLabels(metric, labels) {
				return metric.GetGauge().GetValue()
			}
		}
	}
	return 0
}

func histogramCount(t *testing.T, gatherer prometheus.Gatherer, name string, labels map[string]string) uint64 {
	t.Helper()
	families, err := gatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.Metric {
			if metricLabels(metric, labels) {
				return metric.GetHistogram().GetSampleCount()
			}
		}
	}
	return 0
}

func metricLabels(metric *dto.Metric, labels map[string]string) bool {
	for key, value := range labels {
		found := false
		for _, pair := range metric.Label {
			if pair.GetName() == key && pair.GetValue() == value {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
