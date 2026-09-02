package metrics

import (
	"context"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

type Instrumentation struct {
	enabled     bool
	namespace   string
	registry    *prometheus.Registry
	http        *HTTPServer
	grpc        *GRPCServer
	mu          sync.Mutex
	httpClients map[string]*HTTPClient
	grpcClients map[string]*GRPCClient
}

func Disabled() *Instrumentation {
	return newInstrumentation(false, "")
}

func New(service string) (*Instrumentation, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	return newInstrumentation(cfg.Enable, metricNamespace(*cfg, service)), nil
}

func metricNamespace(cfg Config, service string) string {
	if cfg.Namespace != "" {
		return cfg.Namespace
	}
	return service
}

func newInstrumentation(enabled bool, namespace string) *Instrumentation {
	registry := prometheus.NewRegistry()
	m := newInstrumentationWithRegistry(enabled, namespace, registry)
	if enabled {
		registry.MustRegister(
			collectors.NewGoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		)
	}
	return m
}

func NewWithRegistry(namespace string, registry *prometheus.Registry) *Instrumentation {
	if registry == nil {
		registry = prometheus.NewRegistry()
	}
	return newInstrumentationWithRegistry(true, namespace, registry)
}

func newInstrumentationWithRegistry(
	enabled bool,
	namespace string,
	registry *prometheus.Registry,
) *Instrumentation {
	return &Instrumentation{
		enabled:     enabled,
		namespace:   normalizeNamespace(namespace),
		registry:    registry,
		httpClients: make(map[string]*HTTPClient),
		grpcClients: make(map[string]*GRPCClient),
	}
}

func (m *Instrumentation) Handler() http.Handler {
	if m == nil || !m.enabled {
		return http.NotFoundHandler()
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Instrumentation) HTTPMiddleware(resolve RouteResolver) func(http.Handler) http.Handler {
	if m == nil || !m.enabled {
		return func(next http.Handler) http.Handler { return next }
	}
	m.mu.Lock()
	if m.http == nil {
		m.http = NewHTTPServer(m.registry, m.namespace)
	}
	server := m.http
	m.mu.Unlock()
	return server.Middleware(resolve)
}

func (m *Instrumentation) GRPCUnaryInterceptor() grpc.UnaryServerInterceptor {
	if m == nil || !m.enabled {
		return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
	}
	m.mu.Lock()
	if m.grpc == nil {
		m.grpc = NewGRPCServer(m.registry, m.namespace)
	}
	server := m.grpc
	m.mu.Unlock()
	return server.UnaryInterceptor()
}

func (m *Instrumentation) HTTPTransport(name, target string, next http.RoundTripper) http.RoundTripper {
	if m == nil || !m.enabled {
		if next == nil {
			return http.DefaultTransport
		}
		return next
	}
	m.mu.Lock()
	key := name + "\x00" + target
	client := m.httpClients[key]
	if client == nil {
		client = NewHTTPClient(m.registry, m.namespace, name, target)
		m.httpClients[key] = client
	}
	m.mu.Unlock()
	return client.Transport(next)
}

func (m *Instrumentation) GRPCUnaryClientInterceptor(name, target string) grpc.UnaryClientInterceptor {
	if m == nil || !m.enabled {
		return func(
			ctx context.Context,
			method string,
			req, reply any,
			cc *grpc.ClientConn,
			invoker grpc.UnaryInvoker,
			opts ...grpc.CallOption,
		) error {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
	}
	m.mu.Lock()
	key := name + "\x00" + target
	client := m.grpcClients[key]
	if client == nil {
		client = NewGRPCClient(m.registry, m.namespace, name, target)
		m.grpcClients[key] = client
	}
	m.mu.Unlock()
	return client.UnaryInterceptor()
}
