package tracing

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	traceSDK "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

type contextKey string

const traceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

func testTracer() (trace.Tracer, *tracetest.SpanRecorder) {
	recorder := tracetest.NewSpanRecorder()
	provider := traceSDK.NewTracerProvider(traceSDK.WithSpanProcessor(recorder))
	return provider.Tracer("tracing-test"), recorder
}

func TestTraceEndpointUsesRequestScopedOperationName(t *testing.T) {
	tracer, recorder := testTracer()
	middleware := TraceEndpoint(
		tracer,
		"default",
		WithOperationNameFunc(func(ctx context.Context, name string) string {
			if value, ok := ctx.Value(contextKey("operation")).(string); ok {
				return value
			}
			return name
		}),
	)

	runEndpoint(t, middleware, context.WithValue(context.Background(), contextKey("operation"), "first"))
	runEndpoint(t, middleware, context.WithValue(context.Background(), contextKey("operation"), "second"))

	spans := endedSpans(t, recorder, 2)
	if spans[0].Name() != "first" || spans[1].Name() != "second" {
		t.Fatalf("unexpected span names: %q, %q", spans[0].Name(), spans[1].Name())
	}
}

func TestTraceServerAndClientUseSpanKind(t *testing.T) {
	tests := []struct {
		name       string
		middleware func(trace.Tracer) endpoint.Middleware
		wantKind   trace.SpanKind
	}{
		{
			name: "server",
			middleware: func(tracer trace.Tracer) endpoint.Middleware {
				return TraceServer(tracer, "server")
			},
			wantKind: trace.SpanKindServer,
		},
		{
			name: "client",
			middleware: func(tracer trace.Tracer) endpoint.Middleware {
				return TraceClient(tracer, "client")
			},
			wantKind: trace.SpanKindClient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer, recorder := testTracer()
			runEndpoint(t, tt.middleware(tracer), context.Background())

			spans := endedSpans(t, recorder, 1)
			if spans[0].SpanKind() != tt.wantKind {
				t.Fatalf("expected %s span kind, got %s", tt.wantKind, spans[0].SpanKind())
			}
		})
	}
}

func TestTraceEndpointRecordsError(t *testing.T) {
	tracer, recorder := testTracer()
	wantErr := errors.New("boom")
	next := TraceEndpoint(tracer, "operation")(func(context.Context, interface{}) (interface{}, error) {
		return nil, wantErr
	})

	if _, err := next(context.Background(), nil); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}

	spans := endedSpans(t, recorder, 1)
	if spans[0].Status().Code != codes.Error {
		t.Fatalf("expected error status, got %s", spans[0].Status().Code)
	}
	if got := spans[0].Status().Description; got != wantErr.Error() {
		t.Fatalf("expected status description %q, got %q", wantErr.Error(), got)
	}
	if len(spans[0].Events()) == 0 {
		t.Fatal("expected recorded error event")
	}
}

func TestTraceEndpointAppliesAttributes(t *testing.T) {
	tracer, recorder := testTracer()
	middleware := TraceEndpoint(
		tracer,
		"operation",
		WithAttributes(attribute.String("static", "value")),
		WithAttributesFunc(func(context.Context) []attribute.KeyValue {
			return []attribute.KeyValue{attribute.String("dynamic", "value")}
		}),
	)

	if _, err := middleware(func(context.Context, interface{}) (interface{}, error) {
		return "ok", nil
	})(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	attrs := endedSpans(t, recorder, 1)[0].Attributes()
	assertAttribute(t, attrs, "static", "value")
	assertAttribute(t, attrs, "dynamic", "value")
}

func TestHTTPToContextExtractsTraceParent(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.test/users?id=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("traceparent", traceparent)

	assertRemoteTraceContext(t, HTTPToContext(context.Background(), req))
}

func TestHTTPAttributes(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.test/users?id=1", nil)
	if err != nil {
		t.Fatal(err)
	}

	attrs := HTTPAttributes(req)
	assertAttribute(t, attrs, "http.method", http.MethodPost)
	assertAttribute(t, attrs, "http.url", "https://example.test/users?id=1")
	assertAttribute(t, attrs, "http.scheme", "https")
	assertAttribute(t, attrs, "http.path", "/users")
	assertAttribute(t, attrs, "http.query", "id=1")
}

func TestGRPCToContextExtractsTraceParent(t *testing.T) {
	md := metadata.Pairs("traceparent", traceparent)

	assertRemoteTraceContext(t, GRPCToContext(context.Background(), md))
}

func TestGRPCUnaryServerInterceptorExtractsTraceParent(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("traceparent", traceparent))
	_, err := GRPCUnaryServerInterceptor()(ctx, nil, nil, func(ctx context.Context, _ any) (any, error) {
		assertRemoteTraceContext(t, ctx)
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewWithNilConfigReturnsError(t *testing.T) {
	tracer, shutdown, err := NewWith(context.Background(), "svc", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if tracer == nil {
		t.Fatal("expected noop tracer")
	}
	if shutdown == nil {
		t.Fatal("expected noop shutdown")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNewWithDisabledConfigReturnsNoop(t *testing.T) {
	tracer, shutdown, err := NewWith(context.Background(), "svc", &Config{})
	if err != nil {
		t.Fatal(err)
	}
	if tracer == nil {
		t.Fatal("expected tracer")
	}
	if shutdown == nil {
		t.Fatal("expected shutdown")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSamplerFromRatioUsesParentBasedTraceIDRatio(t *testing.T) {
	dropRoot := samplerFromRatio(0).ShouldSample(traceSDK.SamplingParameters{
		TraceID: trace.TraceID{0x01},
	})
	if dropRoot.Decision != traceSDK.Drop {
		t.Fatalf("expected root span to drop, got %v", dropRoot.Decision)
	}

	sampleRoot := samplerFromRatio(1).ShouldSample(traceSDK.SamplingParameters{
		TraceID: trace.TraceID{0xff},
	})
	if sampleRoot.Decision != traceSDK.RecordAndSample {
		t.Fatalf("expected root span to sample, got %v", sampleRoot.Decision)
	}

	parentContext := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x01},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	}))
	sampledParent := samplerFromRatio(0).ShouldSample(traceSDK.SamplingParameters{
		ParentContext: parentContext,
		TraceID:       trace.TraceID{0xff},
	})
	if sampledParent.Decision != traceSDK.RecordAndSample {
		t.Fatalf("expected sampled parent to sample child, got %v", sampledParent.Decision)
	}
}

func TestSampleRatioDefaultsToOneForOldConfigs(t *testing.T) {
	if got := sampleRatio(&Config{}); got != 1 {
		t.Fatalf("expected default sample ratio 1, got %v", got)
	}
}

func TestNewResourceIncludesDeploymentAttributes(t *testing.T) {
	res, err := newResource(context.Background(), "svc", &Config{
		Environment:       "prod",
		ServiceVersion:    "v1.2.3",
		ServiceInstanceID: "pod-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	attrs := res.Attributes()
	assertAttribute(t, attrs, "service.name", "svc")
	assertAttribute(t, attrs, "deployment.environment", "prod")
	assertAttribute(t, attrs, "service.version", "v1.2.3")
	assertAttribute(t, attrs, "service.instance.id", "pod-1")
}

func runEndpoint(
	t *testing.T,
	middleware endpoint.Middleware,
	ctx context.Context,
) {
	t.Helper()

	next := middleware(func(context.Context, interface{}) (interface{}, error) {
		return "ok", nil
	})
	if _, err := next(ctx, nil); err != nil {
		t.Fatal(err)
	}
}

func endedSpans(t *testing.T, recorder *tracetest.SpanRecorder, count int) []traceSDK.ReadOnlySpan {
	t.Helper()

	spans := recorder.Ended()
	if len(spans) != count {
		t.Fatalf("expected %d spans, got %d", count, len(spans))
	}
	return spans
}

func assertRemoteTraceContext(t *testing.T, ctx context.Context) {
	t.Helper()

	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		t.Fatal("expected valid span context")
	}
	if got := spanContext.TraceID().String(); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("unexpected trace id: %s", got)
	}
	if !spanContext.IsRemote() {
		t.Fatal("expected remote span context")
	}
}

func assertAttribute(t *testing.T, attrs []attribute.KeyValue, key, value string) {
	t.Helper()
	for _, attr := range attrs {
		if string(attr.Key) == key && attr.Value.AsString() == value {
			return
		}
	}
	t.Fatalf("expected attribute %s=%s in %v", key, value, attrs)
}
