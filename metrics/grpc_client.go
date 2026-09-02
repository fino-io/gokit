package metrics

import (
	"context"
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcClient struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inflight *prometheus.GaugeVec
}

func newGRPCClient(registerer prometheus.Registerer, namespace, name, target string) *grpcClient {
	constLabels := prometheus.Labels{
		"client": name, "target": target,
	}
	m := &grpcClient{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Name: "grpc_client_requests_total", Help: "Outbound gRPC requests completed.",
			ConstLabels: constLabels,
		}, []string{"service", "method", "code"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace, Name: "grpc_client_request_duration_seconds",
			Help: "Outbound gRPC request duration.", Buckets: defaultBuckets, ConstLabels: constLabels,
		}, []string{"service", "method"}),
		inflight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace, Name: "grpc_client_requests_in_flight", Help: "Outbound gRPC requests in flight.",
			ConstLabels: constLabels,
		}, []string{"service", "method"}),
	}
	m.requests = registerOrReuse(registerer, m.requests)
	m.duration = registerOrReuse(registerer, m.duration)
	m.inflight = registerOrReuse(registerer, m.inflight)
	return m
}

func (m *grpcClient) unaryInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		service, methodName := splitMethod(method)
		started := time.Now()
		m.inflight.WithLabelValues(service, methodName).Inc()
		defer m.inflight.WithLabelValues(service, methodName).Dec()

		err := invoker(ctx, method, req, reply, cc, opts...)
		m.requests.WithLabelValues(service, methodName, grpcCode(err).String()).Inc()
		m.duration.WithLabelValues(service, methodName).Observe(time.Since(started).Seconds())
		return err
	}
}

func grpcCode(err error) codes.Code {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return status.FromContextError(err).Code()
	}
	return status.Code(err)
}
