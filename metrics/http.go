package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type RouteResolver func(*http.Request) string

type HTTPMiddleware func(RouteResolver) func(http.Handler) http.Handler

type HTTPServer struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inflight *prometheus.GaugeVec
}

func NewHTTPServer(registerer prometheus.Registerer, namespace string) *HTTPServer {
	m := &HTTPServer{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "http_server_requests_total", Help: "HTTP requests completed.",
		}, []string{"method", "route", "status_class"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Name: "http_server_request_duration_seconds",
			Help: "HTTP request duration.", Buckets: defaultBuckets,
		}, []string{"method", "route"}),
		inflight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "http_server_requests_in_flight", Help: "HTTP requests in flight.",
		}, []string{"method"}),
	}
	m.requests = registerOrReuse(registerer, m.requests)
	m.duration = registerOrReuse(registerer, m.duration)
	m.inflight = registerOrReuse(registerer, m.inflight)
	return m
}

func (m *HTTPServer) Middleware(resolve RouteResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			writer := newResponseWriter(w)
			m.inflight.WithLabelValues(r.Method).Inc()
			defer func() {
				m.inflight.WithLabelValues(r.Method).Dec()
				route := "unknown"
				if resolve != nil {
					if value := resolve(r); value != "" {
						route = value
					}
				}
				status := writer.StatusCode()
				if recovered := recover(); recovered != nil {
					status = http.StatusInternalServerError
					m.requests.WithLabelValues(r.Method, route, "5xx").Inc()
					m.duration.WithLabelValues(r.Method, route).Observe(time.Since(started).Seconds())
					panic(recovered)
				}
				m.requests.WithLabelValues(r.Method, route, strconv.Itoa(status/100)+"xx").Inc()
				m.duration.WithLabelValues(r.Method, route).Observe(time.Since(started).Seconds())
			}()
			next.ServeHTTP(writer, r)
		})
	}
}
