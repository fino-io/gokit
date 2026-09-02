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

type clientKey struct {
	name   string
	target string
}

type Instrumentation struct {
	enabled     bool
	namespace   string
	registry    *prometheus.Registry
	http        *httpServer
	grpc        *grpcServer
	mu          sync.Mutex
	httpClients map[clientKey]*httpClient
	grpcClients map[clientKey]*grpcClient
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

func metricNamespace(cfg config, service string) string {
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
	instrumentation := &Instrumentation{
		enabled:   enabled,
		namespace: normalizeNamespace(namespace),
		registry:  registry,
	}
	if enabled {
		instrumentation.httpClients = make(map[clientKey]*httpClient)
		instrumentation.grpcClients = make(map[clientKey]*grpcClient)
	}
	return instrumentation
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
		m.http = newHTTPServer(m.registry, m.namespace)
	}
	server := m.http
	m.mu.Unlock()
	return server.middleware(resolve)
}

func (m *Instrumentation) GRPCUnaryInterceptor() grpc.UnaryServerInterceptor {
	if m == nil || !m.enabled {
		return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
	}
	m.mu.Lock()
	if m.grpc == nil {
		m.grpc = newGRPCServer(m.registry, m.namespace)
	}
	server := m.grpc
	m.mu.Unlock()
	return server.unaryInterceptor()
}

func (m *Instrumentation) HTTPTransport(name, target string, next http.RoundTripper) http.RoundTripper {
	if m == nil || !m.enabled {
		if next == nil {
			return http.DefaultTransport
		}
		return next
	}
	m.mu.Lock()
	key := clientKey{name: name, target: target}
	client := m.httpClients[key]
	if client == nil {
		client = newHTTPClient(m.registry, m.namespace, name, target)
		m.httpClients[key] = client
	}
	m.mu.Unlock()
	return client.transport(next)
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
	key := clientKey{name: name, target: target}
	client := m.grpcClients[key]
	if client == nil {
		client = newGRPCClient(m.registry, m.namespace, name, target)
		m.grpcClients[key] = client
	}
	m.mu.Unlock()
	return client.unaryInterceptor()
}
