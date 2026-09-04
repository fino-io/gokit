package accesslog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

func TestHTTPMiddlewareLogsStructuredServerError(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	middleware := httpMiddleware(config{SampleEvery: 0}, logger.Log)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	}))

	r := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	r.RemoteAddr = "203.0.113.10:4321"
	r.Header.Set("X-Request-ID", "req-http")
	r = r.WithContext(contextWithTrace(r.Context()))
	handler.ServeHTTP(httptest.NewRecorder(), r)

	entry := logger.single(t)
	if entry.Level != levelError {
		t.Fatalf("level = %v, want %v", entry.Level, levelError)
	}
	assertField(t, entry.Fields, "protocol", "http")
	assertField(t, entry.Fields, "method", http.MethodGet)
	assertField(t, entry.Fields, "path", "/users/42")
	assertField(t, entry.Fields, "status", http.StatusServiceUnavailable)
	assertField(t, entry.Fields, "bytes", int64(4))
	assertField(t, entry.Fields, "remote_ip", "203.0.113.10")
	assertField(t, entry.Fields, "request_id", "req-http")
	span := trace.SpanContextFromContext(logger.logContext(t))
	if span.TraceID() != testTraceID || span.SpanID() != testSpanID {
		t.Fatalf("trace context = %s/%s, want %s/%s", span.TraceID(), span.SpanID(), testTraceID, testSpanID)
	}
}

func TestHTTPMiddlewareGeneratesRequestIDWhenMissing(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	var handlerRequestID string
	handler := httpMiddleware(config{SampleEvery: 1}, logger.Log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerRequestID = r.Header.Get(requestIDHeader)
		if _, err := uuid.Parse(handlerRequestID); err != nil {
			t.Fatalf("request ID = %q, want UUID: %v", handlerRequestID, err)
		}
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/users", nil)
	handler.ServeHTTP(recorder, request)

	responseRequestID := recorder.Header().Get(requestIDHeader)
	if _, err := uuid.Parse(responseRequestID); err != nil {
		t.Fatalf("response request ID = %q, want UUID: %v", responseRequestID, err)
	}
	if responseRequestID != handlerRequestID {
		t.Fatalf("response request ID = %q, handler request ID = %q", responseRequestID, handlerRequestID)
	}
	if entry := logger.single(t); entry.Fields["request_id"] != responseRequestID {
		t.Fatalf("logged request ID = %#v, response request ID = %q", entry.Fields["request_id"], responseRequestID)
	}
}

func TestHTTPMiddlewareSkipsSuccessfulConfiguredPath(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	handler := httpMiddleware(config{
		SampleEvery: 1,
		HTTP:        httpConfig{SkipPaths: []string{"/healthz"}},
	}, logger.Log)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if got := logger.count(); got != 0 {
		t.Fatalf("log count = %d, want 0", got)
	}
}

func TestHTTPMiddlewareLogsSlowSuccessfulRequest(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	handler := httpMiddleware(config{
		SlowThreshold: time.Nanosecond,
		SampleEvery:   0,
	}, logger.Log)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/users", nil))
	if entry := logger.single(t); entry.Level != levelInfo {
		t.Fatalf("level = %v, want %v", entry.Level, levelInfo)
	}
}

func TestHTTPMiddlewarePreservesFlusher(t *testing.T) {
	t.Parallel()

	writer := &flushingRecorder{ResponseRecorder: httptest.NewRecorder()}
	handler := HTTPMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, ok := w.(http.Flusher); !ok {
			t.Fatal("wrapped response writer does not preserve http.Flusher")
		}
	}))
	handler.ServeHTTP(writer, httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestHTTPMiddlewareLogsAndPropagatesPanic(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	handler := httpMiddleware(config{SampleEvery: 0}, logger.Log)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	defer func() {
		if recovered := recover(); recovered != "boom" {
			t.Fatalf("recovered = %v, want boom", recovered)
		}
		if entry := logger.single(t); entry.Level != levelError {
			t.Fatalf("level = %v, want %v", entry.Level, levelError)
		}
	}()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/panic", nil))
}

type flushingRecorder struct{ *httptest.ResponseRecorder }

func (*flushingRecorder) Flush() {}

var (
	testTraceID = trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	testSpanID  = trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8}
)

func contextWithTrace(ctx context.Context) context.Context {
	return trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: testTraceID,
		SpanID:  testSpanID,
	}))
}
