package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type httpClient struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inflight *prometheus.GaugeVec
}

func newHTTPClient(registerer prometheus.Registerer, namespace, name, target string) *httpClient {
	constLabels := prometheus.Labels{
		"client": name, "target": target,
	}
	m := &httpClient{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "http_client_requests_total", Help: "Outbound HTTP requests completed.",
			ConstLabels: constLabels,
		}, []string{"method", "result"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Name: "http_client_request_duration_seconds",
			Help: "Outbound HTTP request duration.", Buckets: defaultBuckets, ConstLabels: constLabels,
		}, []string{"method"}),
		inflight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "http_client_requests_in_flight", Help: "Outbound HTTP requests in flight.",
			ConstLabels: constLabels,
		}, []string{"method"}),
	}
	m.requests = registerOrReuse(registerer, m.requests)
	m.duration = registerOrReuse(registerer, m.duration)
	m.inflight = registerOrReuse(registerer, m.inflight)
	return m
}

func (m *httpClient) transport(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		started := time.Now()
		inflight := m.inflight.WithLabelValues(req.Method)
		inflight.Inc()
		defer inflight.Dec()

		response, err := next.RoundTrip(req)
		m.requests.WithLabelValues(req.Method, httpResult(req.Context(), response, err)).Inc()
		m.duration.WithLabelValues(req.Method).Observe(time.Since(started).Seconds())
		return response, err
	})
}

func httpResult(ctx context.Context, response *http.Response, err error) string {
	if ctx != nil {
		switch ctx.Err() {
		case context.DeadlineExceeded:
			return "timeout"
		case context.Canceled:
			return "canceled"
		}
	}
	if err != nil || response == nil {
		return "failure"
	}
	switch {
	case response.StatusCode >= http.StatusInternalServerError:
		return "server_error"
	case response.StatusCode >= http.StatusBadRequest:
		return "client_error"
	default:
		return "success"
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
