package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

func TestDisabledProviderReturnsNotFound(t *testing.T) {
	recorder := httptest.NewRecorder()
	newInstrumentation(false, "test").Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestEnabledProviderExposesMetrics(t *testing.T) {
	recorder := httptest.NewRecorder()
	newInstrumentation(true, "test").Handler().ServeHTTP(
		recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "go_goroutines") {
		t.Fatal("metrics output does not contain Go runtime metrics")
	}
}

func TestInstrumentationNormalizesNamespace(t *testing.T) {
	instrumentation := newInstrumentation(true, "9-mailgate.v1 MailgateService")
	instrumentation.HTTPMiddleware(func(*http.Request) string { return "/test" })(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/test", nil))

	recorder := httptest.NewRecorder()
	instrumentation.Handler().ServeHTTP(
		recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "_9_mailgate_v1_MailgateService_http_server_requests_in_flight") {
		t.Fatal("metrics output does not contain normalized namespace")
	}
}

func TestMetricNamespaceUsesConfiguredOverride(t *testing.T) {
	if got := metricNamespace(config{Namespace: "shared-api"}, "mailgate"); got != "shared-api" {
		t.Fatalf("namespace = %q, want shared-api", got)
	}
}

func TestMetricNamespaceFallsBackToServiceName(t *testing.T) {
	if got := metricNamespace(config{}, "mailgate"); got != "mailgate" {
		t.Fatalf("namespace = %q, want mailgate", got)
	}
}

func TestGRPCUnaryClientInterceptorReusesCollector(t *testing.T) {
	instrumentation := newInstrumentation(true, "test")
	_ = instrumentation.GRPCUnaryClientInterceptor("mailgate-client", "mailgate.v1.MailgateService")
	_ = instrumentation.GRPCUnaryClientInterceptor("mailgate-client", "mailgate.v1.MailgateService")
}

func TestHTTPTransportReusesCollector(t *testing.T) {
	instrumentation := newInstrumentation(true, "test")
	_ = instrumentation.HTTPTransport("registration", "provider", nil)
	_ = instrumentation.HTTPTransport("registration", "provider", nil)
}

func TestDisabledGRPCUnaryClientInterceptorIsTransparent(t *testing.T) {
	instrumentation := newInstrumentation(false, "test")
	called := false
	err := instrumentation.GRPCUnaryClientInterceptor("mailgate-client", "mailgate.v1.MailgateService")(
		context.Background(), "/mailgate.v1.MailgateService/CreateTask", nil, nil, nil,
		func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
			called = true
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("invoker was not called")
	}

	recorder := httptest.NewRecorder()
	instrumentation.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if strings.Contains(recorder.Body.String(), "grpc_client_") {
		t.Fatal("disabled instrumentation exposed client metrics")
	}
}

func TestDisabledHTTPTransportIsTransparent(t *testing.T) {
	instrumentation := newInstrumentation(false, "test")
	called := false
	transport := instrumentation.HTTPTransport("registration", "provider", roundTripperFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: http.StatusNoContent}, nil
	}))
	request := httptest.NewRequest(http.MethodGet, "https://provider.example/register", nil)
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("round trip was not called")
	}

	recorder := httptest.NewRecorder()
	instrumentation.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if strings.Contains(recorder.Body.String(), "http_client_") {
		t.Fatal("disabled instrumentation exposed client metrics")
	}
}

func TestInstrumentationUsesInjectedRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	instrumentation := newInstrumentationWithRegistry(true, "test", registry)
	interceptor := instrumentation.GRPCUnaryClientInterceptor("mailgate-client", "mailgate.v1.MailgateService")
	if err := interceptor(
		context.Background(), "/mailgate.v1.MailgateService/CreateTask", nil, nil, nil,
		func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error { return nil },
	); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	instrumentation.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(recorder.Body.String(), `test_grpc_client_requests_total`) {
		t.Fatal("injected registry does not expose client metrics")
	}
}

func TestHTTPTransportUsesInjectedRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	instrumentation := newInstrumentationWithRegistry(true, "test", registry)
	transport := instrumentation.HTTPTransport("registration", "provider", roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK}, nil
	}))
	if _, err := transport.RoundTrip(httptest.NewRequest(http.MethodGet, "https://provider.example/register", nil)); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	instrumentation.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(recorder.Body.String(), `test_http_client_requests_total`) {
		t.Fatal("injected registry does not expose HTTP client metrics")
	}
}

func TestInstrumentationReusesCollectorsInInjectedRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()
	first := newInstrumentationWithRegistry(true, "test", registry)
	second := newInstrumentationWithRegistry(true, "test", registry)

	_ = first.GRPCUnaryInterceptor()
	_ = second.GRPCUnaryInterceptor()
	_ = first.HTTPTransport("registration", "provider", nil)
	_ = second.HTTPTransport("registration", "provider", nil)
	_ = first.GRPCUnaryClientInterceptor("mailgate-client", "mailgate.v1.MailgateService")
	_ = second.GRPCUnaryClientInterceptor("mailgate-client", "mailgate.v1.MailgateService")
}

func TestInjectedRegistryCreatesServerCollectorsLazily(t *testing.T) {
	registry := prometheus.NewRegistry()
	instrumentation := newInstrumentationWithRegistry(true, "test", registry)
	_ = instrumentation.HTTPTransport("registration", "provider", nil)
	_ = instrumentation.GRPCUnaryClientInterceptor("mailgate-client", "mailgate.v1.MailgateService")

	recorder := httptest.NewRecorder()
	instrumentation.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := recorder.Body.String()
	if strings.Contains(body, "test_http_server_") || strings.Contains(body, "test_grpc_server_") {
		t.Fatal("client-only instrumentation registered server collectors")
	}
}
