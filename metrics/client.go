package metrics

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HTTPClient struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inflight *prometheus.GaugeVec
}

func NewHTTPClient(registerer prometheus.Registerer, namespace, name, target string) *HTTPClient {
	constLabels := prometheus.Labels{
		"client": name, "target": target,
	}
	m := &HTTPClient{
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
		}, nil),
	}
	m.requests = registerOrReuse(registerer, m.requests)
	m.duration = registerOrReuse(registerer, m.duration)
	m.inflight = registerOrReuse(registerer, m.inflight)
	return m
}

func (m *HTTPClient) Transport(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		started := time.Now()
		m.inflight.WithLabelValues().Inc()
		defer m.inflight.WithLabelValues().Dec()

		response, err := next.RoundTrip(req)
		m.requests.WithLabelValues(
			req.Method, httpResult(req.Context(), response, err),
		).Inc()
		m.duration.WithLabelValues(
			req.Method,
		).Observe(time.Since(started).Seconds())
		return response, err
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type GRPCClient struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inflight *prometheus.GaugeVec
}

func NewGRPCClient(registerer prometheus.Registerer, namespace, name, target string) *GRPCClient {
	constLabels := prometheus.Labels{
		"client": name, "target": target,
	}
	m := &GRPCClient{
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

func (m *GRPCClient) UnaryInterceptor() grpc.UnaryClientInterceptor {
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
