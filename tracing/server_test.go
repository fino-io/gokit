package tracing

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestHTTPServerMiddlewareCreatesServerSpan(t *testing.T) {
	tracer, recorder := testTracer()
	var got trace.SpanContext

	handler := HTTPServerMiddleware(tracer)(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		got = trace.SpanContextFromContext(request.Context())
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodPost, "http://example.test/users?id=1", nil)
	recorderHTTP := httptest.NewRecorder()
	handler.ServeHTTP(recorderHTTP, request)

	if recorderHTTP.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorderHTTP.Code)
	}
	if !got.IsValid() {
		t.Fatal("expected handler to receive a valid span context")
	}
	if got.IsRemote() {
		t.Fatal("expected handler to receive a local server span")
	}

	span := endedSpans(t, recorder, 1)[0]
	if span.Name() != "HTTP POST" {
		t.Fatalf("unexpected span name %q", span.Name())
	}
	if span.SpanKind() != trace.SpanKindServer {
		t.Fatalf("expected server span, got %s", span.SpanKind())
	}
	assertAttribute(t, span.Attributes(), "http.method", http.MethodPost)
	assertAttribute(t, span.Attributes(), "http.path", "/users")
}

func TestHTTPServerMiddlewarePreservesIncomingTraceID(t *testing.T) {
	tracer, recorder := testTracer()
	var got trace.SpanContext

	handler := HTTPServerMiddleware(tracer)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		got = trace.SpanContextFromContext(request.Context())
	}))
	request := httptest.NewRequest(http.MethodGet, "http://example.test/users", nil)
	request.Header.Set("traceparent", traceparent)
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if got.TraceID().String() != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("unexpected trace id %s", got.TraceID())
	}
	if got.IsRemote() {
		t.Fatal("expected server span to be local")
	}
	endedSpans(t, recorder, 1)
}

func TestGRPCUnaryServerTracingInterceptorCreatesServerSpan(t *testing.T) {
	tracer, recorder := testTracer()
	wantErr := errors.New("unavailable")
	var got trace.SpanContext

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("traceparent", traceparent))
	interceptor := GRPCUnaryServerTracingInterceptor(tracer)
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/identity.v1.UserService/GetUser"}, func(ctx context.Context, _ any) (any, error) {
		got = trace.SpanContextFromContext(ctx)
		return nil, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if got.TraceID().String() != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("unexpected trace id %s", got.TraceID())
	}
	if got.IsRemote() {
		t.Fatal("expected server span to be local")
	}

	span := endedSpans(t, recorder, 1)[0]
	if span.Name() != "/identity.v1.UserService/GetUser" {
		t.Fatalf("unexpected span name %q", span.Name())
	}
	if span.SpanKind() != trace.SpanKindServer {
		t.Fatalf("expected server span, got %s", span.SpanKind())
	}
	if span.Status().Code != codes.Error {
		t.Fatalf("expected error status, got %s", span.Status().Code)
	}
}

func TestServerTracingMiddlewareWithNilTracerOnlyPropagates(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://example.test/users", nil)
	request.Header.Set("traceparent", traceparent)
	var got trace.SpanContext

	handler := HTTPServerMiddleware(nil)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		got = trace.SpanContextFromContext(request.Context())
	}))
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if !got.IsValid() || !got.IsRemote() {
		t.Fatalf("expected incoming remote span context, got %+v", got)
	}
}

func TestHTTPToContextPreservesExistingSpanContext(t *testing.T) {
	parent := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x02},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), parent)
	request := httptest.NewRequest(http.MethodGet, "http://example.test/users", nil)
	request.Header.Set("traceparent", traceparent)

	got := trace.SpanContextFromContext(HTTPToContext(ctx, request))
	if got.TraceID() != parent.TraceID() || got.SpanID() != parent.SpanID() || got.TraceFlags() != parent.TraceFlags() || got.IsRemote() != parent.IsRemote() {
		t.Fatalf("expected existing span context to be preserved, got %+v", got)
	}
}

func TestGRPCToContextPreservesExistingSpanContext(t *testing.T) {
	parent := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x02},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), parent)

	got := trace.SpanContextFromContext(GRPCToContext(ctx, metadata.Pairs("traceparent", traceparent)))
	if got.TraceID() != parent.TraceID() || got.SpanID() != parent.SpanID() || got.TraceFlags() != parent.TraceFlags() || got.IsRemote() != parent.IsRemote() {
		t.Fatalf("expected existing span context to be preserved, got %+v", got)
	}
}
