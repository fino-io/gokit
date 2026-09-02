package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type GRPCServer struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inflight *prometheus.GaugeVec
}

func NewGRPCServer(registerer prometheus.Registerer, namespace string) *GRPCServer {
	m := &GRPCServer{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "grpc_server_requests_total", Help: "gRPC requests completed.",
		}, []string{"service", "method", "code"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Name: "grpc_server_request_duration_seconds",
			Help: "gRPC request duration.", Buckets: defaultBuckets,
		}, []string{"service", "method"}),
		inflight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "grpc_server_requests_in_flight", Help: "gRPC requests in flight.",
		}, []string{"service", "method"}),
	}
	m.requests = registerOrReuse(registerer, m.requests)
	m.duration = registerOrReuse(registerer, m.duration)
	m.inflight = registerOrReuse(registerer, m.inflight)
	return m
}

func (m *GRPCServer) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		service, method := splitMethod(info.FullMethod)
		started := time.Now()
		m.inflight.WithLabelValues(service, method).Inc()
		defer m.inflight.WithLabelValues(service, method).Dec()

		response, err := handler(ctx, req)
		m.requests.WithLabelValues(service, method, status.Code(err).String()).Inc()
		m.duration.WithLabelValues(service, method).Observe(time.Since(started).Seconds())
		return response, err
	}
}

func splitMethod(full string) (string, string) {
	parts := strings.Split(strings.TrimPrefix(full, "/"), "/")
	if len(parts) != 2 {
		return "unknown", "unknown"
	}
	return parts[0], parts[1]
}
