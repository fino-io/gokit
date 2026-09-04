package tracing

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"unicode/utf8"

	traceSDK "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestRecordErrorSanitizesInvalidUTF8(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := traceSDK.NewTracerProvider(traceSDK.WithSpanProcessor(recorder))
	_, span := provider.Tracer("tracing-test").Start(context.Background(), "operation")

	rawMessage := string([]byte{'b', 0xff, 'd'})
	recordError(span, errors.New(rawMessage))
	span.End()

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected one span, got %d", len(ended))
	}
	if got := ended[0].Status().Description; got != "b\uFFFDd" {
		t.Fatalf("expected sanitized status description, got %q", got)
	}
	if !utf8.ValidString(ended[0].Status().Description) {
		t.Fatal("expected valid UTF-8 status description")
	}

	events := ended[0].Events()
	if len(events) != 1 {
		t.Fatalf("expected one error event, got %d", len(events))
	}
	for _, attr := range events[0].Attributes {
		if !utf8.ValidString(string(attr.Key)) || !utf8.ValidString(attr.Value.AsString()) {
			t.Fatalf("expected valid UTF-8 event attribute, got %v", attr)
		}
	}
}

func TestProviderExportsSanitizedError(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		_, _ = io.Copy(io.Discard, request.Body)
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider, err := newTraceProvider(context.Background(), "svc", &config{
		Enable:   true,
		Endpoint: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, span := provider.Tracer("tracing-test").Start(context.Background(), "operation")
	recordError(span, errors.New(string([]byte{'b', 0xff, 'd'})))
	span.End()

	if err := provider.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected sanitized span to export, got %v", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("expected one OTLP request, got %d", got)
	}
}
