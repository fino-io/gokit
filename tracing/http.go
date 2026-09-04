package tracing

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var httpPropagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

// extractHTTPContext extracts OpenTelemetry propagation headers from req.
func extractHTTPContext(ctx context.Context, req *http.Request) context.Context {
	if req == nil || trace.SpanContextFromContext(ctx).IsValid() {
		return ctx
	}
	return httpPropagator.Extract(ctx, propagation.HeaderCarrier(req.Header))
}

// httpAttributes returns low-cardinality HTTP attributes for a request span.
func httpAttributes(req *http.Request) []attribute.KeyValue {
	if req == nil {
		return nil
	}

	attrs := []attribute.KeyValue{attribute.String("http.method", req.Method)}
	if req.URL == nil {
		return attrs
	}

	return append(attrs, attribute.String("http.scheme", httpScheme(req)))
}

func httpScheme(req *http.Request) string {
	if req.URL != nil && req.URL.Scheme != "" {
		return req.URL.Scheme
	}
	if req.TLS != nil {
		return "https"
	}
	return "http"
}
